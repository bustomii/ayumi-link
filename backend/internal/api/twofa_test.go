package api

import (
	"strings"
	"testing"
	"time"
)

func TestTwoFactorChangeOverHTTP(t *testing.T) {
	e := newEnv(t)
	cookie, secret, recovery := setupAccount(t, e)

	// tanpa login ditolak
	if res, _ := e.do(t, "POST", "/api/profile/2fa/begin", `{"password":"x","code":"000000"}`, "", nil); res.StatusCode != 401 {
		t.Fatalf("begin tanpa login: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "POST", "/api/profile/2fa/confirm", `{"code":"000000"}`, "", nil); res.StatusCode != 401 {
		t.Fatalf("confirm tanpa login: %d", res.StatusCode)
	}

	// confirm sebelum begin
	if res, _ := e.do(t, "POST", "/api/profile/2fa/confirm", `{"code":"000000"}`, cookie, nil); res.StatusCode != 400 {
		t.Fatalf("confirm tanpa begin harus 400: %d", res.StatusCode)
	}
	// verifikasi ulang salah
	if res, _ := e.do(t, "POST", "/api/profile/2fa/begin", `{"password":"`+pwOld+`","code":"000000"}`, cookie, nil); res.StatusCode != 401 {
		t.Fatalf("kode salah harus 401: %d", res.StatusCode)
	}

	// kode TOTP saat setup sudah terpakai; gunakan langkah waktu berikutnya
	res, begin := e.do(t, "POST", "/api/profile/2fa/begin", `{"password":"`+pwOld+`","code":"`+totpNow(t, secret, 30*time.Second)+`"}`, cookie, nil)
	if res.StatusCode != 200 || begin["secret"] == "" || !strings.HasPrefix(begin["qr"].(string), "data:image/png;base64,") {
		t.Fatalf("begin: %d %v", res.StatusCode, begin)
	}
	newSecret := begin["secret"].(string)
	if newSecret == secret {
		t.Fatal("rahasia baru harus berbeda")
	}

	if res, _ := e.do(t, "POST", "/api/profile/2fa/confirm", `{"code":"000000"}`, cookie, nil); res.StatusCode != 401 {
		t.Fatalf("kode salah di confirm harus 401: %d", res.StatusCode)
	}
	res, done := e.do(t, "POST", "/api/profile/2fa/confirm", `{"code":"`+totpNow(t, newSecret, 0)+`"}`, cookie, nil)
	if res.StatusCode != 200 {
		t.Fatalf("confirm: %d %v", res.StatusCode, done)
	}
	codes := done["recoveryCodes"].([]any)
	if len(codes) != 8 || codes[0].(string) == recovery[0] {
		t.Fatalf("kode pemulihan baru: %v", codes)
	}

	// profil: sesi tetap sah, twoFactorSince tersedia
	res, prof := e.do(t, "GET", "/api/profile", "", cookie, nil)
	if res.StatusCode != 200 || prof["twoFactorSince"] == nil || prof["recoveryRemaining"].(float64) != 8 {
		t.Fatalf("profil: %d %v", res.StatusCode, prof)
	}
}
