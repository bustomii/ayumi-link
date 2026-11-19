package auth

import (
	"errors"
	"testing"
	"time"
)

// login menyelesaikan login penuh dan mengembalikan token sesi.
func login(t *testing.T, m *Manager, clock *time.Time, secret string) string {
	t.Helper()
	mfa, err := m.Login(testUser, testPass)
	if err != nil {
		t.Fatal(err)
	}
	s, err := m.VerifyMFA(mfa, code(t, secret, *clock), ClientInfo{})
	if err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(31 * time.Second) // langkah TOTP berikutnya (anti-replay)
	return s
}

func TestTwoFactorChangeRequiresStepUp(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)
	s := login(t, m, clock, secret)

	if _, err := m.BeginTwoFactorChange(s, "password-salah-banget", code(t, secret, *clock)); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("password salah harus ditolak: %v", err)
	}
	if _, err := m.BeginTwoFactorChange(s, testPass, "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode salah harus ditolak: %v", err)
	}
	if _, err := m.BeginTwoFactorChange("token-ngawur", testPass, code(t, secret, *clock)); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("tanpa sesi harus ditolak: %v", err)
	}
	res, err := m.BeginTwoFactorChange(s, testPass, code(t, secret, *clock))
	if err != nil {
		t.Fatal(err)
	}
	if res.Secret == "" || res.Secret == secret || res.QR == "" {
		t.Fatalf("harus menghasilkan rahasia baru + QR: %+v", res)
	}
	// Sebelum dikonfirmasi, rahasia lama masih berlaku.
	if _, err := m.Login(testUser, testPass); err != nil {
		t.Fatalf("login masih harus bisa: %v", err)
	}
}

func TestTwoFactorChangeFullFlow(t *testing.T) {
	m, clock := newManager(t)
	oldSecret, oldRecovery := setup(t, m, clock)
	s1 := login(t, m, clock, oldSecret)
	otherSession := login(t, m, clock, oldSecret) // perangkat lain

	res, err := m.BeginTwoFactorChange(s1, testPass, code(t, oldSecret, *clock))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.ConfirmTwoFactorChange(s1, "123456"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode salah: %v", err)
	}
	newRecovery, err := m.ConfirmTwoFactorChange(s1, code(t, res.Secret, *clock))
	if err != nil {
		t.Fatal(err)
	}
	if len(newRecovery) != recoveryCount {
		t.Fatalf("kode pemulihan baru = %d", len(newRecovery))
	}

	// sesi pemanggil tetap, sesi lain dicabut
	if _, ok := m.Authenticate(s1); !ok {
		t.Fatal("sesi pemanggil harus tetap sah")
	}
	if _, ok := m.Authenticate(otherSession); ok {
		t.Fatal("sesi lain harus dicabut")
	}

	// kode lama tidak lagi berlaku, kode baru berlaku
	*clock = clock.Add(31 * time.Second)
	mfa, _ := m.Login(testUser, testPass)
	if _, err := m.VerifyMFA(mfa, code(t, oldSecret, *clock), ClientInfo{}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("autentikator lama harus ditolak: %v", err)
	}
	if _, err := m.VerifyMFA(mfa, code(t, res.Secret, *clock), ClientInfo{}); err != nil {
		t.Fatalf("autentikator baru harus diterima: %v", err)
	}

	// kode pemulihan lama hangus, yang baru berlaku
	mfa, _ = m.Login(testUser, testPass)
	if _, err := m.VerifyMFA(mfa, oldRecovery[0], ClientInfo{}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode pemulihan lama harus hangus: %v", err)
	}
	mfa, _ = m.Login(testUser, testPass)
	if _, err := m.VerifyMFA(mfa, newRecovery[0], ClientInfo{}); err != nil {
		t.Fatalf("kode pemulihan baru harus berlaku: %v", err)
	}

	p, err := m.Profile(s1)
	if err != nil {
		t.Fatal(err)
	}
	if !p.TwoFactorSince.After(p.CreatedAt) {
		t.Fatalf("twoFactorSince harus diperbarui: %v vs %v", p.TwoFactorSince, p.CreatedAt)
	}
}

func TestTwoFactorConfirmReplayAndBinding(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)
	s1 := login(t, m, clock, secret)
	s2 := login(t, m, clock, secret)

	res, err := m.BeginTwoFactorChange(s1, testPass, code(t, secret, *clock))
	if err != nil {
		t.Fatal(err)
	}
	// sesi lain tidak boleh menyelesaikan perubahan milik s1
	if _, err := m.ConfirmTwoFactorChange(s2, code(t, res.Secret, *clock)); !errors.Is(err, ErrNoPendingChange) {
		t.Fatalf("pending terikat pada sesi pemula: %v", err)
	}
	// konfirmasi tanpa Begin
	if _, err := m.ConfirmTwoFactorChange(s1, code(t, res.Secret, *clock)); !errors.Is(err, ErrNoPendingChange) {
		// pending dibuang saat dicoba sesi lain; konfirmasi berikutnya harus gagal juga
		t.Fatalf("pending harus sudah dibuang: %v", err)
	}

	// kode konfirmasi yang sama tidak boleh dipakai ulang untuk login pada langkah waktu itu
	*clock = clock.Add(31 * time.Second) // kode saat ini pada langkah sebelumnya sudah terpakai
	res, err = m.BeginTwoFactorChange(s1, testPass, code(t, secret, *clock))
	if err != nil {
		t.Fatal(err)
	}
	c := code(t, res.Secret, *clock)
	if _, err := m.ConfirmTwoFactorChange(s1, c); err != nil {
		t.Fatal(err)
	}
	mfa, _ := m.Login(testUser, testPass)
	if _, err := m.VerifyMFA(mfa, c, ClientInfo{}); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode konfirmasi tidak boleh dipakai ulang: %v", err)
	}
}

func TestTwoFactorPendingLimits(t *testing.T) {
	m, clock := newManager(t)
	secret, _ := setup(t, m, clock)
	s := login(t, m, clock, secret)

	res, _ := m.BeginTwoFactorChange(s, testPass, code(t, secret, *clock))
	for i := 0; i < pendingMaxAttempts; i++ {
		if _, err := m.ConfirmTwoFactorChange(s, "000000"); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("percobaan %d: %v", i, err)
		}
	}
	if _, err := m.ConfirmTwoFactorChange(s, code(t, res.Secret, *clock)); !errors.Is(err, ErrNoPendingChange) {
		t.Fatalf("setelah terlalu banyak percobaan pending harus dibuang: %v", err)
	}

	*clock = clock.Add(31 * time.Second)
	res, _ = m.BeginTwoFactorChange(s, testPass, code(t, secret, *clock))
	*clock = clock.Add(pendingTTL + time.Second)
	if _, err := m.ConfirmTwoFactorChange(s, code(t, res.Secret, *clock)); !errors.Is(err, ErrNoPendingChange) {
		t.Fatalf("pending harus kedaluwarsa: %v", err)
	}
}
