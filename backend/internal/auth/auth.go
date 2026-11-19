// Package auth: login satu pengguna dengan password (argon2id) + 2FA TOTP,
// kode pemulihan sekali pakai, sesi berbasis cookie, dan pembatasan percobaan.
package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	Issuer          = "Ayumi Link"
	sessionIdleTTL  = 7 * 24 * time.Hour  // diperpanjang setiap dipakai
	sessionMaxTTL   = 30 * 24 * time.Hour // batas mutlak
	mfaTTL          = 5 * time.Minute
	maxFailures     = 5
	lockDuration    = 15 * time.Minute
	mfaMaxAttempts  = 5
	recoveryCount   = 8
	minPasswordLen  = 12
	persistInterval = 5 * time.Minute
)

// Kesalahan yang dipetakan ke respons HTTP oleh lapisan API.
var (
	ErrSetupDone          = errors.New("setup sudah selesai")
	ErrBadSetupToken      = errors.New("token setup salah")
	ErrInvalidUsername    = errors.New("username 3-32 karakter: huruf kecil, angka, titik, strip, garis bawah")
	ErrWeakPassword       = errors.New("password minimal 12 karakter")
	ErrNoPendingSetup     = errors.New("setup belum dimulai")
	ErrInvalidCredentials = errors.New("username atau password salah")
	ErrInvalidCode        = errors.New("kode 2FA salah")
	ErrChallengeExpired   = errors.New("sesi login kedaluwarsa, ulangi dari awal")
	ErrNotConfigured      = errors.New("belum ada akun")
)

// LockedError: akun dikunci sementara karena terlalu banyak percobaan gagal.
type LockedError struct{ Until time.Time }

func (e *LockedError) Error() string { return "terlalu banyak percobaan, coba lagi nanti" }

var usernameRe = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)

type user struct {
	Username      string    `json:"username"`
	PasswordHash  string    `json:"passwordHash"`
	TOTPSecret    string    `json:"totpSecret"`
	TOTPConfirmed bool      `json:"totpConfirmed"`
	LastTOTPStep  int64     `json:"lastTotpStep"`
	Recovery      []string  `json:"recovery"` // sha256 hex kode pemulihan yang belum dipakai
	Failed        int       `json:"failed"`
	LockedUntil   time.Time `json:"lockedUntil"`
	CreatedAt     time.Time `json:"createdAt"`
	// TOTPChangedAt: kapan autentikator terakhir diganti (kosong = sejak setup).
	TOTPChangedAt time.Time `json:"totpChangedAt,omitempty"`
}

type sessionRec struct {
	Created  time.Time `json:"created"`
	LastSeen time.Time `json:"lastSeen"`
	Expires  time.Time `json:"expires"`
	UA       string    `json:"ua,omitempty"`
	IP       string    `json:"ip,omitempty"`
}

// ClientInfo: asal sebuah login, ditampilkan di daftar sesi aktif.
type ClientInfo struct{ UA, IP string }

type fileData struct {
	User     *user                  `json:"user,omitempty"`
	Sessions map[string]*sessionRec `json:"sessions"` // sha256 hex token -> rekaman
}

type challenge struct {
	expires  time.Time
	attempts int
}

type Manager struct {
	mu         sync.Mutex
	path       string
	data       fileData
	setupToken string
	mfa        map[string]*challenge // sha256 hex mfaToken -> tantangan
	now        func() time.Time
	lastSave   time.Time
	pending    *pendingTOTP // penggantian autentikator yang menunggu konfirmasi
}

// Open memuat data auth. Bila belum ada akun terkonfirmasi, token setup dibuat.
func Open(dir string) (*Manager, error) {
	m := &Manager{
		path: filepath.Join(dir, "auth.json"),
		mfa:  map[string]*challenge{},
		now:  time.Now,
	}
	m.data.Sessions = map[string]*sessionRec{}
	if b, err := os.ReadFile(m.path); err == nil {
		if err := json.Unmarshal(b, &m.data); err != nil {
			return nil, err
		}
		if m.data.Sessions == nil {
			m.data.Sessions = map[string]*sessionRec{}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if !m.configuredLocked() {
		m.setupToken = randomString(20)
	}
	return m, nil
}

// Reset menghapus seluruh data auth (dipakai perintah CLI `reset-auth`).
func Reset(dir string) error {
	err := os.Remove(filepath.Join(dir, "auth.json"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (m *Manager) configuredLocked() bool {
	return m.data.User != nil && m.data.User.TOTPConfirmed
}

// SetupToken: token sekali pakai yang dicetak di log saat belum ada akun.
func (m *Manager) SetupToken() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.setupToken
}

type Status struct {
	State    string `json:"state"` // setup_required | logged_out | authenticated
	Username string `json:"username,omitempty"`
}

func (m *Manager) Status(sessionToken string) Status {
	m.mu.Lock()
	configured := m.configuredLocked()
	m.mu.Unlock()
	if !configured {
		return Status{State: "setup_required"}
	}
	if name, ok := m.Authenticate(sessionToken); ok {
		return Status{State: "authenticated", Username: name}
	}
	return Status{State: "logged_out"}
}

// ---- setup ----

type SetupResult struct {
	Secret string `json:"secret"`
	URL    string `json:"otpauthUrl"`
	QR     string `json:"qr"` // data URL PNG
}

func (m *Manager) BeginSetup(setupToken, username, password string) (*SetupResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.configuredLocked() {
		return nil, ErrSetupDone
	}
	if !tokenEqual(setupToken, m.setupToken) {
		return nil, ErrBadSetupToken
	}
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernameRe.MatchString(username) {
		return nil, ErrInvalidUsername
	}
	if len([]rune(password)) < minPasswordLen {
		return nil, ErrWeakPassword
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: Issuer, AccountName: username, Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, err
	}
	img, err := key.Image(220, 220)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	m.data.User = &user{
		Username: username, PasswordHash: hash, TOTPSecret: key.Secret(), CreatedAt: m.now(),
	}
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return &SetupResult{
		Secret: key.Secret(),
		URL:    key.URL(),
		QR:     "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

// ConfirmSetup memverifikasi kode pertama, mengaktifkan 2FA, membuat kode pemulihan
// (ditampilkan sekali) dan langsung masuk.
func (m *Manager) ConfirmSetup(setupToken, code string, ci ClientInfo) (recovery []string, session string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.configuredLocked() {
		return nil, "", ErrSetupDone
	}
	if !tokenEqual(setupToken, m.setupToken) {
		return nil, "", ErrBadSetupToken
	}
	u := m.data.User
	if u == nil {
		return nil, "", ErrNoPendingSetup
	}
	if !m.checkTOTPLocked(u, code) {
		return nil, "", ErrInvalidCode
	}
	u.TOTPConfirmed = true
	recovery, u.Recovery = newRecoveryCodes()
	m.setupToken = ""
	session = m.newSessionLocked(ci)
	if err := m.saveLocked(); err != nil {
		return nil, "", err
	}
	return recovery, session, nil
}

// ---- login ----

// Login memeriksa password; bila benar mengembalikan token tantangan 2FA.
func (m *Manager) Login(username, password string) (mfaToken string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.configuredLocked() {
		return "", ErrNotConfigured
	}
	u := m.data.User
	now := m.now()
	if now.Before(u.LockedUntil) {
		return "", &LockedError{Until: u.LockedUntil}
	}

	// Selalu hitung hash agar waktu tidak membocorkan keberadaan username.
	hash := dummyHash
	nameOK := subtle.ConstantTimeCompare([]byte(strings.ToLower(strings.TrimSpace(username))), []byte(u.Username)) == 1
	if nameOK {
		hash = u.PasswordHash
	}
	passOK := verifyPassword(password, hash)
	if !nameOK || !passOK {
		m.failLocked(u)
		return "", ErrInvalidCredentials
	}

	token := randomString(32)
	m.purgeMFALocked()
	m.mfa[hashToken(token)] = &challenge{expires: now.Add(mfaTTL)}
	return token, nil
}

// VerifyMFA menerima kode TOTP (6 digit) atau kode pemulihan, lalu membuat sesi.
func (m *Manager) VerifyMFA(mfaToken, code string, ci ClientInfo) (session string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.configuredLocked() {
		return "", ErrNotConfigured
	}
	u := m.data.User
	now := m.now()
	if now.Before(u.LockedUntil) {
		return "", &LockedError{Until: u.LockedUntil}
	}
	key := hashToken(mfaToken)
	ch, ok := m.mfa[key]
	if !ok || now.After(ch.expires) {
		delete(m.mfa, key)
		return "", ErrChallengeExpired
	}

	if m.checkTOTPLocked(u, code) || m.useRecoveryLocked(u, code) {
		delete(m.mfa, key)
		u.Failed = 0
		session = m.newSessionLocked(ci)
		if err := m.saveLocked(); err != nil {
			return "", err
		}
		return session, nil
	}

	ch.attempts++
	if ch.attempts >= mfaMaxAttempts {
		delete(m.mfa, key)
	}
	m.failLocked(u)
	if now.Before(u.LockedUntil) {
		return "", &LockedError{Until: u.LockedUntil}
	}
	return "", ErrInvalidCode
}

func (m *Manager) failLocked(u *user) {
	u.Failed++
	if u.Failed >= maxFailures {
		u.Failed = 0
		u.LockedUntil = m.now().Add(lockDuration)
		m.mfa = map[string]*challenge{} // batalkan semua tantangan berjalan
	}
	_ = m.saveLocked()
}

// ---- sesi ----

func (m *Manager) newSessionLocked(ci ClientInfo) string {
	token := randomString(32)
	now := m.now()
	m.data.Sessions[hashToken(token)] = &sessionRec{
		Created: now, LastSeen: now, Expires: now.Add(sessionIdleTTL),
		UA: truncate(ci.UA, 200), IP: truncate(ci.IP, 64),
	}
	return token
}

// Authenticate memvalidasi token sesi dan memperpanjang masa berlakunya.
func (m *Manager) Authenticate(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.configuredLocked() {
		return "", false
	}
	key := hashToken(token)
	rec, ok := m.data.Sessions[key]
	if !ok {
		return "", false
	}
	now := m.now()
	if now.After(rec.Expires) || now.After(rec.Created.Add(sessionMaxTTL)) {
		delete(m.data.Sessions, key)
		_ = m.saveLocked()
		return "", false
	}
	rec.LastSeen = now
	rec.Expires = now.Add(sessionIdleTTL)
	if now.Sub(m.lastSave) > persistInterval {
		_ = m.saveLocked()
	}
	return m.data.User.Username, true
}

func (m *Manager) Logout(token string) {
	if token == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data.Sessions, hashToken(token))
	_ = m.saveLocked()
}

// SessionMaxAge: umur cookie sesi (detik).
func SessionMaxAge() int { return int(sessionIdleTTL.Seconds()) }

// ---- TOTP & kode pemulihan ----

func (m *Manager) checkTOTPLocked(u *user, code string) bool {
	step, ok := matchTOTP(u.TOTPSecret, code, m.now())
	// Cegah pemakaian ulang kode yang sama (replay).
	if !ok || step <= u.LastTOTPStep {
		return false
	}
	u.LastTOTPStep = step
	return true
}

var recoveryAlphabet = base32.StdEncoding.WithPadding(base32.NoPadding)

func normalizeRecovery(s string) string {
	s = strings.ToUpper(s)
	return strings.NewReplacer("-", "", " ", "").Replace(s)
}

func newRecoveryCodes() (plain, hashed []string) {
	for i := 0; i < recoveryCount; i++ {
		b := make([]byte, 7)
		_, _ = rand.Read(b)
		raw := recoveryAlphabet.EncodeToString(b)[:10] // 10 karakter base32 = 50 bit
		plain = append(plain, raw[:5]+"-"+raw[5:])
		hashed = append(hashed, hashToken(raw))
	}
	return plain, hashed
}

func (m *Manager) useRecoveryLocked(u *user, code string) bool {
	norm := normalizeRecovery(code)
	if len(norm) != 10 {
		return false
	}
	h := hashToken(norm)
	for i, existing := range u.Recovery {
		if subtle.ConstantTimeCompare([]byte(existing), []byte(h)) == 1 {
			u.Recovery = append(u.Recovery[:i:i], u.Recovery[i+1:]...)
			return true
		}
	}
	return false
}

// RecoveryRemaining: jumlah kode pemulihan yang tersisa.
func (m *Manager) RecoveryRemaining() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data.User == nil {
		return 0
	}
	return len(m.data.User.Recovery)
}

// ---- util ----

func (m *Manager) purgeMFALocked() {
	now := m.now()
	for k, c := range m.mfa {
		if now.After(c.expires) {
			delete(m.mfa, k)
		}
	}
}

func (m *Manager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return err
	}
	now := m.now()
	for k, s := range m.data.Sessions { // buang sesi kedaluwarsa
		if now.After(s.Expires) {
			delete(m.data.Sessions, k)
		}
	}
	b, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(m.path), "auth.*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), m.path); err != nil {
		return err
	}
	m.lastSave = now
	return nil
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func tokenEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func randomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}
