package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// loginSession membuat sesi baru lewat login penuh dan memajukan jam 31 dtk agar kode TOTP berikutnya baru.
func loginSession(t *testing.T, m *Manager, clock *time.Time, secret string, ci ClientInfo) string {
	t.Helper()
	mfa, err := m.Login(testUser, testPass)
	if err != nil {
		t.Fatal(err)
	}
	s, err := m.VerifyMFA(mfa, code(t, secret, *clock), ci)
	if err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(31 * time.Second)
	return s
}

func TestProfileListsSessionsWithClientInfo(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)
	a := loginSession(t, m, clock, secret, ClientInfo{UA: "Chrome macOS", IP: "10.0.0.1"})
	loginSession(t, m, clock, secret, ClientInfo{UA: "Safari iOS", IP: "10.0.0.2"})

	p, err := m.Profile(a)
	if err != nil {
		t.Fatal(err)
	}
	if p.Username != testUser || p.RecoveryRemaining != recoveryCount {
		t.Fatalf("profil: %+v", p)
	}
	current := 0
	for _, s := range p.Sessions {
		if s.Current {
			current++
			if s.UA != "Chrome macOS" || s.IP != "10.0.0.1" {
				t.Fatalf("sesi saat ini salah: %+v", s)
			}
		}
		if len(s.ID) != publicIDLen {
			t.Fatalf("id publik: %q", s.ID)
		}
	}
	if current != 1 || len(p.Sessions) != 3 { // 1 dari setup + 2 dari login
		t.Fatalf("jumlah sesi=%d current=%d", len(p.Sessions), current)
	}
	if _, err := m.Profile("token-palsu"); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("token palsu: %v", err)
	}
}

func TestChangePasswordRequiresStepUpAndRevokesOthers(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)
	keep := loginSession(t, m, clock, secret, ClientInfo{UA: "A"})
	other := loginSession(t, m, clock, secret, ClientInfo{UA: "B"})
	const next = "password-baru-yang-panjang"

	if err := m.ChangePassword(keep, testPass, "pendek", code(t, secret, *clock)); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("password lemah: %v", err)
	}
	if err := m.ChangePassword(keep, testPass, testPass, code(t, secret, *clock)); !errors.Is(err, ErrSamePassword) {
		t.Fatalf("password sama: %v", err)
	}
	if err := m.ChangePassword(keep, "salah-salah-salah", next, code(t, secret, *clock)); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("password lama salah: %v", err)
	}
	if err := m.ChangePassword(keep, testPass, next, "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode 2FA salah: %v", err)
	}
	if err := m.ChangePassword("token-palsu", testPass, next, code(t, secret, *clock)); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("tanpa sesi: %v", err)
	}

	if err := m.ChangePassword(keep, testPass, next, code(t, secret, *clock)); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Authenticate(keep); !ok {
		t.Fatal("sesi yang dipakai harus tetap valid")
	}
	if _, ok := m.Authenticate(other); ok {
		t.Fatal("sesi lain harus dicabut setelah ganti password")
	}
	*clock = clock.Add(31 * time.Second)
	if _, err := m.Login(testUser, testPass); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("password lama tidak boleh dipakai lagi: %v", err)
	}
	if _, err := m.Login(testUser, next); err != nil {
		t.Fatalf("password baru harus bisa: %v", err)
	}
}

func TestRegenerateRecoveryInvalidatesOldCodes(t *testing.T) {
	m, clock := newManager(t)
	secret, old := setup(t, m, clock)
	s := loginSession(t, m, clock, secret, ClientInfo{})

	if _, err := m.RegenerateRecovery(s, "salah-salah-salah", code(t, secret, *clock)); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("password salah: %v", err)
	}
	fresh, err := m.RegenerateRecovery(s, testPass, code(t, secret, *clock))
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh) != recoveryCount || fresh[0] == old[0] {
		t.Fatalf("kode baru: %v", fresh)
	}
	*clock = clock.Add(31 * time.Second)
	mfa, _ := m.Login(testUser, testPass)
	if _, err := m.VerifyMFA(mfa, old[0], ClientInfo{}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode lama harus hangus: %v", err)
	}
	if _, err := m.VerifyMFA(mfa, strings.ToLower(fresh[0]), ClientInfo{}); err != nil {
		t.Fatalf("kode baru harus berlaku: %v", err)
	}
}

func TestRevokeSessions(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)
	me := loginSession(t, m, clock, secret, ClientInfo{UA: "me"})
	other := loginSession(t, m, clock, secret, ClientInfo{UA: "other"})
	third := loginSession(t, m, clock, secret, ClientInfo{UA: "third"})

	p, _ := m.Profile(me)
	var myID, otherID string
	for _, s := range p.Sessions {
		switch s.UA {
		case "me":
			myID = s.ID
		case "other":
			otherID = s.ID
		}
	}
	if err := m.RevokeSession(me, myID); !errors.Is(err, ErrCannotRevokeActive) {
		t.Fatalf("sesi sendiri: %v", err)
	}
	if err := m.RevokeSession(me, "0000000000000000"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("id tak ada: %v", err)
	}
	if err := m.RevokeSession(me, "x"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("id terlalu pendek (cegah pencocokan awalan): %v", err)
	}
	if err := m.RevokeSession(me, otherID); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Authenticate(other); ok {
		t.Fatal("sesi yang dicabut harus tidak valid")
	}
	n, err := m.RevokeOthers(me)
	if err != nil || n < 1 {
		t.Fatalf("revoke others: %d %v", n, err)
	}
	if _, ok := m.Authenticate(third); ok {
		t.Fatal("sesi lain harus dicabut")
	}
	if _, ok := m.Authenticate(me); !ok {
		t.Fatal("sesi sendiri harus tetap valid")
	}
}

func TestStepUpFailuresLockAccount(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)
	s := loginSession(t, m, clock, secret, ClientInfo{})
	var locked *LockedError
	var err error
	for i := 0; i < maxFailures; i++ {
		err = m.ChangePassword(s, "salah-salah-salah", "password-baru-yang-panjang", "000000")
	}
	if !errors.As(err, &locked) {
		t.Fatalf("percobaan berulang di profil harus mengunci akun: %v", err)
	}
}
