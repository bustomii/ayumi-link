package auth

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrUnauthenticated    = errors.New("belum login")
	ErrSamePassword       = errors.New("password baru harus berbeda dari yang lama")
	ErrSessionNotFound    = errors.New("sesi tidak ditemukan")
	ErrCannotRevokeActive = errors.New("sesi ini sedang dipakai; gunakan Keluar")
)

// publicIDLen: panjang awalan hash token yang dipakai sebagai id sesi di UI.
// Hash tidak bisa dipakai untuk memalsukan token, jadi aman ditampilkan.
const publicIDLen = 16

type SessionInfo struct {
	ID       string    `json:"id"`
	Created  time.Time `json:"createdAt"`
	LastSeen time.Time `json:"lastSeen"`
	UA       string    `json:"ua"`
	IP       string    `json:"ip"`
	Current  bool      `json:"current"`
}

type Profile struct {
	Username          string    `json:"username"`
	CreatedAt         time.Time `json:"createdAt"`
	RecoveryRemaining int       `json:"recoveryRemaining"`
	// TwoFactorSince: sejak kapan autentikator saat ini dipakai (setup atau penggantian terakhir).
	TwoFactorSince time.Time     `json:"twoFactorSince"`
	Sessions       []SessionInfo `json:"sessions"`
}

// activeKeyLocked memvalidasi token sesi tanpa memperpanjangnya; mengembalikan kunci hash-nya.
func (m *Manager) activeKeyLocked(token string) (string, bool) {
	if token == "" || !m.configuredLocked() {
		return "", false
	}
	key := hashToken(token)
	rec, ok := m.data.Sessions[key]
	if !ok || m.now().After(rec.Expires) || m.now().After(rec.Created.Add(sessionMaxTTL)) {
		return "", false
	}
	return key, true
}

// Profile mengembalikan data akun dan daftar sesi aktif (sesi pemanggil ditandai Current).
func (m *Manager) Profile(token string) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.activeKeyLocked(token)
	if !ok {
		return nil, ErrUnauthenticated
	}
	u := m.data.User
	since := u.TOTPChangedAt
	if since.IsZero() {
		since = u.CreatedAt
	}
	p := &Profile{Username: u.Username, CreatedAt: u.CreatedAt, RecoveryRemaining: len(u.Recovery), TwoFactorSince: since, Sessions: []SessionInfo{}}
	now := m.now()
	for key, rec := range m.data.Sessions {
		if now.After(rec.Expires) {
			continue
		}
		p.Sessions = append(p.Sessions, SessionInfo{
			ID: key[:publicIDLen], Created: rec.Created, LastSeen: rec.LastSeen,
			UA: rec.UA, IP: rec.IP, Current: key == current,
		})
	}
	sort.Slice(p.Sessions, func(i, j int) bool { return p.Sessions[i].LastSeen.After(p.Sessions[j].LastSeen) })
	return p, nil
}

// stepUpLocked: verifikasi ulang (password + kode 2FA) sebelum aksi sensitif.
// Kegagalan dihitung ke penguncian akun yang sama dengan login.
func (m *Manager) stepUpLocked(password, code string) error {
	u := m.data.User
	if m.now().Before(u.LockedUntil) {
		return &LockedError{Until: u.LockedUntil}
	}
	if !verifyPassword(password, u.PasswordHash) {
		m.failLocked(u)
		if m.now().Before(u.LockedUntil) {
			return &LockedError{Until: u.LockedUntil}
		}
		return ErrInvalidCredentials
	}
	if !m.checkTOTPLocked(u, code) && !m.useRecoveryLocked(u, code) {
		m.failLocked(u)
		if m.now().Before(u.LockedUntil) {
			return &LockedError{Until: u.LockedUntil}
		}
		return ErrInvalidCode
	}
	u.Failed = 0
	return nil
}

// ChangePassword mengganti password lalu mencabut semua sesi lain.
func (m *Manager) ChangePassword(token, current, next, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.activeKeyLocked(token)
	if !ok {
		return ErrUnauthenticated
	}
	if len([]rune(next)) < minPasswordLen {
		return ErrWeakPassword
	}
	if current == next {
		return ErrSamePassword
	}
	if err := m.stepUpLocked(current, code); err != nil {
		return err
	}
	hash, err := hashPassword(next)
	if err != nil {
		return err
	}
	m.data.User.PasswordHash = hash
	for k := range m.data.Sessions {
		if k != key {
			delete(m.data.Sessions, k)
		}
	}
	return m.saveLocked()
}

// RegenerateRecovery membuat 8 kode pemulihan baru (kode lama hangus) dan mengembalikannya sekali.
func (m *Manager) RegenerateRecovery(token, password, code string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.activeKeyLocked(token); !ok {
		return nil, ErrUnauthenticated
	}
	if err := m.stepUpLocked(password, code); err != nil {
		return nil, err
	}
	plain, hashed := newRecoveryCodes()
	m.data.User.Recovery = hashed
	return plain, m.saveLocked()
}

// RevokeSession mencabut satu sesi lain berdasarkan id publiknya.
func (m *Manager) RevokeSession(token, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.activeKeyLocked(token)
	if !ok {
		return ErrUnauthenticated
	}
	if len(id) != publicIDLen {
		return ErrSessionNotFound
	}
	for key := range m.data.Sessions {
		if strings.HasPrefix(key, id) {
			if key == current {
				return ErrCannotRevokeActive
			}
			delete(m.data.Sessions, key)
			return m.saveLocked()
		}
	}
	return ErrSessionNotFound
}

// RevokeOthers mencabut semua sesi kecuali yang sedang dipakai; mengembalikan jumlahnya.
func (m *Manager) RevokeOthers(token string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.activeKeyLocked(token)
	if !ok {
		return 0, ErrUnauthenticated
	}
	n := 0
	for key := range m.data.Sessions {
		if key != current {
			delete(m.data.Sessions, key)
			n++
		}
	}
	return n, m.saveLocked()
}
