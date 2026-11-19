package auth

import (
	"bytes"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"image/png"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	pendingTTL         = 10 * time.Minute
	pendingMaxAttempts = 5
)

var ErrNoPendingChange = errors.New("penggantian autentikator belum dimulai atau sudah kedaluwarsa")

// pendingTOTP: rahasia TOTP baru yang menunggu konfirmasi kode dari perangkat baru.
// Hanya di memori dan terikat pada satu sesi login.
type pendingTOTP struct {
	secret     string
	sessionKey string
	expires    time.Time
	attempts   int
}

// matchTOTP memeriksa kode 6 digit terhadap secret (toleransi ±1 langkah 30 dtk) dan
// mengembalikan nomor langkah waktu yang cocok.
func matchTOTP(secret, code string, now time.Time) (step int64, ok bool) {
	code = strings.Join(strings.Fields(code), "")
	if len(code) != 6 {
		return 0, false
	}
	for _, off := range []time.Duration{0, -30 * time.Second, 30 * time.Second} {
		t := now.Add(off)
		want, err := totp.GenerateCodeCustom(secret, t, totp.ValidateOpts{
			Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
		})
		if err == nil && subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 && !ok {
			step, ok = t.Unix()/30, true
		}
	}
	return step, ok
}

func setupResultFor(key *otp.Key) (*SetupResult, error) {
	img, err := key.Image(220, 220)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return &SetupResult{
		Secret: key.Secret(),
		URL:    key.URL(),
		QR:     "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

// BeginTwoFactorChange memulai penggantian perangkat autentikator. Wajib verifikasi ulang
// (password + kode saat ini). Rahasia lama tetap berlaku sampai kode baru dikonfirmasi.
func (m *Manager) BeginTwoFactorChange(token, password, code string) (*SetupResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sessionKey, ok := m.activeKeyLocked(token)
	if !ok {
		return nil, ErrUnauthenticated
	}
	if err := m.stepUpLocked(password, code); err != nil {
		return nil, err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: Issuer, AccountName: m.data.User.Username, Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, err
	}
	res, err := setupResultFor(key)
	if err != nil {
		return nil, err
	}
	m.pending = &pendingTOTP{secret: key.Secret(), sessionKey: sessionKey, expires: m.now().Add(pendingTTL)}
	if err := m.saveLocked(); err != nil { // stepUp memperbarui langkah TOTP terakhir
		return nil, err
	}
	return res, nil
}

// ConfirmTwoFactorChange memverifikasi kode dari perangkat baru lalu mengganti rahasia TOTP.
// Efeknya: kode pemulihan lama hangus (diganti 8 kode baru, dikembalikan sekali) dan semua sesi
// login lain dicabut, supaya perangkat/akses lama tidak tersisa.
func (m *Manager) ConfirmTwoFactorChange(token, code string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sessionKey, ok := m.activeKeyLocked(token)
	if !ok {
		return nil, ErrUnauthenticated
	}
	p := m.pending
	if p == nil || p.sessionKey != sessionKey || m.now().After(p.expires) {
		m.pending = nil
		return nil, ErrNoPendingChange
	}
	step, matched := matchTOTP(p.secret, code, m.now())
	if !matched {
		p.attempts++
		if p.attempts >= pendingMaxAttempts {
			m.pending = nil
		}
		return nil, ErrInvalidCode
	}

	u := m.data.User
	u.TOTPSecret = p.secret
	u.LastTOTPStep = step
	u.TOTPChangedAt = m.now()
	var plain []string
	plain, u.Recovery = newRecoveryCodes()
	m.pending = nil
	for k := range m.data.Sessions {
		if k != sessionKey {
			delete(m.data.Sessions, k)
		}
	}
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return plain, nil
}
