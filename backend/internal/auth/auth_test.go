package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	testUser = "owner"
	testPass = "correct-horse-battery"
)

func newManager(t *testing.T) (*Manager, *time.Time) {
	t.Helper()
	m, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return clock }
	return m, &clock
}

func code(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	c, err := totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// setup menyelesaikan setup dan mengembalikan secret + kode pemulihan.
func setup(t *testing.T, m *Manager, clock *time.Time) (secret string, recovery []string) {
	t.Helper()
	res, err := m.BeginSetup(m.SetupToken(), testUser, testPass)
	if err != nil {
		t.Fatal(err)
	}
	recovery, session, err := m.ConfirmSetup(m.setupToken, code(t, res.Secret, *clock), ClientInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Authenticate(session); !ok {
		t.Fatal("sesi dari setup harus valid")
	}
	*clock = clock.Add(31 * time.Second) // kode setup tidak boleh dipakai ulang untuk login
	return res.Secret, recovery
}

func TestSetupRules(t *testing.T) {
	m, _ := newManager(t)
	if st := m.Status(""); st.State != "setup_required" {
		t.Fatalf("state = %q", st.State)
	}
	if _, err := m.BeginSetup("salah", testUser, testPass); !errors.Is(err, ErrBadSetupToken) {
		t.Fatalf("token salah harus ditolak: %v", err)
	}
	if _, err := m.BeginSetup(m.SetupToken(), "A", testPass); !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("username buruk: %v", err)
	}
	if _, err := m.BeginSetup(m.SetupToken(), testUser, "pendek"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("password lemah: %v", err)
	}
}

func TestSetupConfirmRequiresValidCode(t *testing.T) {
	m, clock := newManager(t)
	res, err := m.BeginSetup(m.SetupToken(), testUser, testPass)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.ConfirmSetup(m.SetupToken(), "000000", ClientInfo{}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode salah harus ditolak: %v", err)
	}
	if m.Status("").State != "setup_required" {
		t.Fatal("belum boleh dianggap terkonfigurasi")
	}
	if _, _, err := m.ConfirmSetup(m.SetupToken(), code(t, res.Secret, *clock), ClientInfo{}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.BeginSetup("apa-saja", testUser, testPass); !errors.Is(err, ErrSetupDone) {
		t.Fatalf("setup ulang harus ditolak: %v", err)
	}
}

func TestLoginFlow(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)

	if _, err := m.Login(testUser, "salah-salah-salah"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("password salah: %v", err)
	}
	if _, err := m.Login("orang-lain", testPass); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("username salah: %v", err)
	}

	mfa, err := m.Login(strings.ToUpper(testUser), testPass) // username tidak peka huruf besar/kecil
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.VerifyMFA(mfa, "123456", ClientInfo{}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode salah: %v", err)
	}
	session, err := m.VerifyMFA(mfa, code(t, secret, *clock), ClientInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if st := m.Status(session); st.State != "authenticated" || st.Username != testUser {
		t.Fatalf("status = %+v", st)
	}
	m.Logout(session)
	if m.Status(session).State != "logged_out" {
		t.Fatal("setelah logout harus logged_out")
	}
}

func TestTOTPReplayBlocked(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)

	c := code(t, secret, *clock)
	mfa1, _ := m.Login(testUser, testPass)
	if _, err := m.VerifyMFA(mfa1, c, ClientInfo{}); err != nil {
		t.Fatal(err)
	}
	mfa2, _ := m.Login(testUser, testPass)
	if _, err := m.VerifyMFA(mfa2, c, ClientInfo{}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode TOTP yang sama tidak boleh dipakai dua kali: %v", err)
	}
}

func TestRecoveryCodeSingleUse(t *testing.T) {
	m, clock := newManager(t)
	_, recovery := setup(t, m, clock)
	if len(recovery) != recoveryCount {
		t.Fatalf("jumlah kode pemulihan = %d", len(recovery))
	}

	mfa1, _ := m.Login(testUser, testPass)
	if _, err := m.VerifyMFA(mfa1, strings.ToLower(recovery[0]), ClientInfo{}); err != nil { // peka-huruf tidak berlaku
		t.Fatalf("kode pemulihan valid: %v", err)
	}
	mfa2, _ := m.Login(testUser, testPass)
	if _, err := m.VerifyMFA(mfa2, recovery[0], ClientInfo{}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode pemulihan hanya sekali pakai: %v", err)
	}
	if got := m.RecoveryRemaining(); got != recoveryCount-1 {
		t.Fatalf("sisa = %d", got)
	}
}

func TestLockoutAfterRepeatedFailures(t *testing.T) {
	m, clock := newManager(t)
	setup(t, m, clock)

	for i := 0; i < maxFailures; i++ {
		_, _ = m.Login(testUser, "password-salah-banget")
	}
	var locked *LockedError
	if _, err := m.Login(testUser, testPass); !errors.As(err, &locked) {
		t.Fatalf("akun harus terkunci walau password benar: %v", err)
	}
	*clock = clock.Add(lockDuration + time.Second)
	if _, err := m.Login(testUser, testPass); err != nil {
		t.Fatalf("setelah masa kunci harus bisa login: %v", err)
	}
}

func TestMFAChallengeLimits(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)

	mfa, _ := m.Login(testUser, testPass)
	*clock = clock.Add(mfaTTL + time.Second)
	if _, err := m.VerifyMFA(mfa, code(t, secret, *clock), ClientInfo{}); !errors.Is(err, ErrChallengeExpired) {
		t.Fatalf("tantangan harus kedaluwarsa: %v", err)
	}
}

func TestSessionExpiry(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)
	mfa, _ := m.Login(testUser, testPass)
	session, err := m.VerifyMFA(mfa, code(t, secret, *clock), ClientInfo{})
	if err != nil {
		t.Fatal(err)
	}

	*clock = clock.Add(sessionIdleTTL - time.Hour)
	if _, ok := m.Authenticate(session); !ok {
		t.Fatal("masih dalam masa berlaku")
	}
	*clock = clock.Add(sessionIdleTTL + time.Hour) // diam melewati batas idle
	if _, ok := m.Authenticate(session); ok {
		t.Fatal("sesi idle harus kedaluwarsa")
	}
}

func TestPasswordHashing(t *testing.T) {
	h, err := hashPassword("rahasia-banget-123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$") {
		t.Fatalf("format hash: %s", h)
	}
	if !verifyPassword("rahasia-banget-123", h) || verifyPassword("rahasia-banget-124", h) {
		t.Fatal("verifikasi password salah")
	}
	if verifyPassword("x", "bukan-hash") {
		t.Fatal("hash rusak harus ditolak")
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	m, _ := Open(dir)
	clock := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return clock }
	setup(t, m, &clock)

	m2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m2.Status("").State != "logged_out" {
		t.Fatal("setelah restart akun harus tetap ada")
	}
	if m2.SetupToken() != "" {
		t.Fatal("token setup tidak boleh dibuat bila akun sudah ada")
	}
}
