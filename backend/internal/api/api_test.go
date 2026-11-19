package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"ayumi-link/internal/auth"
	"ayumi-link/internal/claude"
	"ayumi-link/internal/config"
	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
)

type env struct {
	app  *fiber.App
	auth *auth.Manager
}

func newEnv(t *testing.T) *env { return newEnvLimit(t, 1000) }

// newEnvLimit membuat lingkungan uji dengan batas request auth per menit tertentu.
func newEnvLimit(t *testing.T, authLimit int) *env {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir, ClaudeHome: dir, DesktopDir: dir, PermissionMode: "auto", AuthRateLimit: authLimit}
	st := store.New(dir)
	am, err := auth.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	br := fsbrowse.New([]string{dir})
	svc := &claude.Service{Home: dir, DesktopDir: dir, Store: st}
	cliAuth, err := claude.OpenCliAuth(dir, "false", dir)
	if err != nil {
		t.Fatal(err)
	}
	run := &claude.Runner{Svc: svc, Browser: br, Store: st, Bin: "false", DefaultPerm: "auto", Auth: cliAuth}
	return &env{app: New(Deps{Config: cfg, Auth: am, Svc: svc, Runner: run, Browser: br, Store: st, CliAuth: cliAuth}), auth: am}
}

func (e *env) do(t *testing.T, method, path, body, cookie string, hdr map[string]string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return res, out
}

func sessionCookie(res *http.Response) string {
	for _, c := range res.Cookies() {
		if c.Name == cookieName {
			return c.Name + "=" + c.Value
		}
	}
	return ""
}

func totpNow(t *testing.T, secret string, off time.Duration) string {
	t.Helper()
	c, err := totp.GenerateCodeCustom(secret, time.Now().Add(off), totp.ValidateOpts{Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	e := newEnv(t)
	routes := [][2]string{{"GET", "/api/sessions"}, {"GET", "/api/fs"}, {"GET", "/api/sessions/11111111-2222-4333-8444-555555555555"}}
	for _, r := range routes {
		res, body := e.do(t, r[0], r[1], "", "", nil)
		if res.StatusCode != 401 || body["code"] != "setup_required" {
			t.Errorf("%s %s sebelum setup: %d %v", r[0], r[1], res.StatusCode, body)
		}
	}
	res, body := e.do(t, "POST", "/api/chat", `{"message":"hai"}`, "", nil)
	if res.StatusCode != 401 {
		t.Errorf("chat tanpa login: %d %v", res.StatusCode, body)
	}
	if res, _ := e.do(t, "GET", "/api/health", "", "", nil); res.StatusCode != 200 {
		t.Errorf("health harus publik: %d", res.StatusCode)
	}
}

func TestFullAuthFlowOverHTTP(t *testing.T) {
	e := newEnv(t)
	tok := e.auth.SetupToken()

	_, st := e.do(t, "GET", "/api/auth/status", "", "", nil)
	if st["state"] != "setup_required" {
		t.Fatalf("status: %v", st)
	}
	if res, _ := e.do(t, "POST", "/api/auth/setup", `{"setupToken":"salah","username":"owner","password":"correct-horse-battery"}`, "", nil); res.StatusCode != 403 {
		t.Fatalf("token setup salah harus 403, dapat %d", res.StatusCode)
	}
	res, setup := e.do(t, "POST", "/api/auth/setup", `{"setupToken":"`+tok+`","username":"owner","password":"correct-horse-battery"}`, "", nil)
	if res.StatusCode != 200 || setup["secret"] == "" || !strings.HasPrefix(setup["qr"].(string), "data:image/png;base64,") {
		t.Fatalf("setup: %d %v", res.StatusCode, setup)
	}
	secret := setup["secret"].(string)

	res, done := e.do(t, "POST", "/api/auth/setup/confirm", `{"setupToken":"`+tok+`","code":"`+totpNow(t, secret, 0)+`"}`, "", nil)
	if res.StatusCode != 200 {
		t.Fatalf("confirm: %d %v", res.StatusCode, done)
	}
	if n := len(done["recoveryCodes"].([]any)); n != 8 {
		t.Fatalf("kode pemulihan: %d", n)
	}
	cookie := sessionCookie(res)
	if cookie == "" {
		t.Fatal("cookie sesi harus di-set")
	}
	for _, c := range res.Cookies() {
		if c.Name == cookieName && (!c.HttpOnly || c.SameSite != http.SameSiteStrictMode) {
			t.Fatalf("cookie harus HttpOnly + SameSite=Strict: %+v", c)
		}
	}
	if res, _ := e.do(t, "GET", "/api/sessions", "", cookie, nil); res.StatusCode != 200 {
		t.Fatalf("sesi sah harus bisa akses: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "POST", "/api/auth/setup", `{"setupToken":"`+tok+`","username":"x","password":"correct-horse-battery"}`, "", nil); res.StatusCode != 403 {
		t.Fatalf("setup ulang harus 403: %d", res.StatusCode)
	}

	// logout lalu login penuh dengan 2FA
	e.do(t, "POST", "/api/auth/logout", `{}`, cookie, nil)
	if res, _ := e.do(t, "GET", "/api/sessions", "", cookie, nil); res.StatusCode != 401 {
		t.Fatalf("setelah logout harus 401: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "POST", "/api/auth/login", `{"username":"owner","password":"salah-salah-salah"}`, "", nil); res.StatusCode != 401 {
		t.Fatalf("password salah harus 401: %d", res.StatusCode)
	}
	res, login := e.do(t, "POST", "/api/auth/login", `{"username":"owner","password":"correct-horse-battery"}`, "", nil)
	if res.StatusCode != 200 || login["mfaRequired"] != true {
		t.Fatalf("login: %d %v", res.StatusCode, login)
	}
	if sessionCookie(res) != "" {
		t.Fatal("sebelum 2FA tidak boleh ada cookie sesi")
	}
	mfa := login["mfaToken"].(string)
	res, _ = e.do(t, "POST", "/api/auth/login/verify", `{"mfaToken":"`+mfa+`","code":"000000"}`, "", nil)
	if res.StatusCode != 401 {
		t.Fatalf("kode salah harus 401: %d", res.StatusCode)
	}
	res, _ = e.do(t, "POST", "/api/auth/login/verify", `{"mfaToken":"`+mfa+`","code":"`+totpNow(t, secret, 30*time.Second)+`"}`, "", nil)
	if res.StatusCode != 200 || sessionCookie(res) == "" {
		t.Fatalf("verify: %d", res.StatusCode)
	}
}

func TestRequestGuards(t *testing.T) {
	e := newEnv(t)
	if res, _ := e.do(t, "POST", "/api/auth/login", `{}`, "", map[string]string{"Content-Type": "text/plain"}); res.StatusCode != 415 {
		t.Errorf("content-type bukan json harus 415: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "GET", "/api/health", "", "", map[string]string{"Origin": "http://evil.example"}); res.StatusCode != 403 {
		t.Errorf("origin asing harus 403: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "GET", "/api/health", "", "", map[string]string{"Origin": "http://localhost:5173"}); res.StatusCode != 200 {
		t.Errorf("origin loopback harus lolos: %d", res.StatusCode)
	}
	// Rute tak dikenal ikut diproteksi: tanpa login hasilnya 401, bukan 404 (tidak bisa dipakai memetakan rute).
	if res, _ := e.do(t, "GET", "/api/tidak-ada", "", "", nil); res.StatusCode != 401 {
		t.Errorf("rute tak dikenal tanpa login harus 401: %d", res.StatusCode)
	}
	res, _ := e.do(t, "GET", "/api/health", "", "", nil)
	if res.Header.Get("X-Content-Type-Options") != "nosniff" || res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("header keamanan kurang: %v", res.Header)
	}
}

func TestAuthRateLimit(t *testing.T) {
	e := newEnvLimit(t, 10)
	var last int
	for i := 0; i < 12; i++ {
		res, _ := e.do(t, "POST", "/api/auth/login", `{"username":"a","password":"b"}`, "", nil)
		last = res.StatusCode
	}
	if last != 429 {
		t.Fatalf("request ke-12 dalam semenit harus 429, dapat %d", last)
	}
}

// ---- profil ----

const (
	pwOld = "correct-horse-battery"
	pwNew = "password-baru-yang-panjang"
)

// setupAccount menyelesaikan setup dan mengembalikan cookie, secret TOTP, dan kode pemulihan.
func setupAccount(t *testing.T, e *env) (cookie, secret string, recovery []string) {
	t.Helper()
	tok := e.auth.SetupToken()
	_, setup := e.do(t, "POST", "/api/auth/setup", `{"setupToken":"`+tok+`","username":"owner","password":"`+pwOld+`"}`, "", nil)
	secret = setup["secret"].(string)
	res, done := e.do(t, "POST", "/api/auth/setup/confirm", `{"setupToken":"`+tok+`","code":"`+totpNow(t, secret, 0)+`"}`, "", nil)
	if res.StatusCode != 200 {
		t.Fatalf("setup: %d %v", res.StatusCode, done)
	}
	for _, c := range done["recoveryCodes"].([]any) {
		recovery = append(recovery, c.(string))
	}
	return sessionCookie(res), secret, recovery
}

func TestProfileRequiresAuth(t *testing.T) {
	e := newEnv(t)
	for _, r := range [][2]string{
		{"GET", "/api/profile"}, {"GET", "/api/profile/prefs"},
		{"PUT", "/api/profile/prefs"}, {"POST", "/api/profile/password"}, {"POST", "/api/profile/recovery"},
		{"POST", "/api/profile/sessions/revoke-others"}, {"DELETE", "/api/profile/sessions/abc"},
	} {
		body := ""
		if r[0] == "POST" || r[0] == "PUT" {
			body = `{}`
		}
		if res, _ := e.do(t, r[0], r[1], body, "", nil); res.StatusCode != 401 {
			t.Errorf("%s %s tanpa login harus 401: %d", r[0], r[1], res.StatusCode)
		}
	}
}

func TestProfileOverview(t *testing.T) {
	e := newEnv(t)
	cookie, _, _ := setupAccount(t, e)

	res, p := e.do(t, "GET", "/api/profile", "", cookie, nil)
	if res.StatusCode != 200 || p["username"] != "owner" || p["recoveryRemaining"] != float64(8) {
		t.Fatalf("profil: %d %v", res.StatusCode, p)
	}
	sessions := p["sessions"].([]any)
	if len(sessions) != 1 || sessions[0].(map[string]any)["current"] != true {
		t.Fatalf("sesi: %v", sessions)
	}
	cfg := p["config"].(map[string]any)
	if cfg["version"] == "" || cfg["dataDir"] == "" {
		t.Fatalf("config: %v", cfg)
	}
	for _, secret := range []string{"passwordHash", "totpSecret", "recovery"} {
		if _, leaked := p[secret]; leaked {
			t.Fatalf("profil tidak boleh membocorkan %s", secret)
		}
	}
	prefs := p["prefs"].(map[string]any)
	if prefs["model"] != "claude-sonnet-5-5" || prefs["effort"] != "medium" || prefs["permissionMode"] != "auto" {
		t.Fatalf("default prefs: %v", prefs)
	}
}

func TestPrefsValidationAndPersistence(t *testing.T) {
	e := newEnv(t)
	cookie, _, _ := setupAccount(t, e)

	for name, body := range map[string]string{
		"model ngawur":    `{"model":"gpt-9","effort":"low","permissionMode":"auto"}`,
		"effort ngawur":   `{"model":"claude-opus-5-5","effort":"turbo","permissionMode":"auto"}`,
		"bypass dilarang": `{"model":"claude-opus-5-5","effort":"low","permissionMode":"bypassPermissions"}`,
		"semua kosong":    `{}`,
	} {
		if res, _ := e.do(t, "PUT", "/api/profile/prefs", body, cookie, nil); res.StatusCode != 400 {
			t.Errorf("%s harus 400: %d", name, res.StatusCode)
		}
	}
	good := `{"model":"claude-opus-5-5","effort":"high","permissionMode":"acceptEdits"}`
	if res, _ := e.do(t, "PUT", "/api/profile/prefs", good, cookie, nil); res.StatusCode != 200 {
		t.Fatalf("prefs valid: %d", res.StatusCode)
	}
	_, got := e.do(t, "GET", "/api/profile/prefs", "", cookie, nil)
	if got["model"] != "claude-opus-5-5" || got["effort"] != "high" || got["permissionMode"] != "acceptEdits" {
		t.Fatalf("prefs tersimpan: %v", got)
	}
}

func TestProfileSessionsRecoveryAndPassword(t *testing.T) {
	e := newEnv(t)
	cookie, secret, recovery := setupAccount(t, e)

	// sesi kedua dari "perangkat lain" (kode TOTP langkah berikutnya, karena yang sekarang sudah terpakai saat setup)
	_, l := e.do(t, "POST", "/api/auth/login", `{"username":"owner","password":"`+pwOld+`"}`, "", nil)
	res, _ := e.do(t, "POST", "/api/auth/login/verify",
		`{"mfaToken":"`+l["mfaToken"].(string)+`","code":"`+totpNow(t, secret, 30*time.Second)+`"}`, "", map[string]string{"User-Agent": "TestPhone/1.0"})
	other := sessionCookie(res)
	if other == "" {
		t.Fatal("sesi kedua harus terbentuk")
	}

	_, p := e.do(t, "GET", "/api/profile", "", cookie, nil)
	sessions := p["sessions"].([]any)
	if len(sessions) != 2 {
		t.Fatalf("harus ada 2 sesi: %v", sessions)
	}
	var otherID string
	for _, s := range sessions {
		m := s.(map[string]any)
		if m["ua"] == "TestPhone/1.0" {
			otherID = m["id"].(string)
		}
	}
	if otherID == "" {
		t.Fatal("UA perangkat lain harus tercatat")
	}

	// cabut sesi lain
	if res, _ := e.do(t, "DELETE", "/api/profile/sessions/"+otherID, "", cookie, nil); res.StatusCode != 200 {
		t.Fatalf("cabut sesi: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "GET", "/api/sessions", "", other, nil); res.StatusCode != 401 {
		t.Fatalf("sesi yang dicabut harus 401: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "DELETE", "/api/profile/sessions/"+otherID, "", cookie, nil); res.StatusCode != 404 {
		t.Fatalf("cabut dua kali harus 404: %d", res.StatusCode)
	}

	// buat ulang kode pemulihan: perlu password + kode 2FA (di sini kode pemulihan)
	if res, _ := e.do(t, "POST", "/api/profile/recovery", `{"password":"salah-salah-salah","code":"`+recovery[0]+`"}`, cookie, nil); res.StatusCode != 401 {
		t.Fatalf("password salah harus 401: %d", res.StatusCode)
	}
	res, fresh := e.do(t, "POST", "/api/profile/recovery", `{"password":"`+pwOld+`","code":"`+recovery[0]+`"}`, cookie, nil)
	if res.StatusCode != 200 || len(fresh["recoveryCodes"].([]any)) != 8 {
		t.Fatalf("regenerate: %d %v", res.StatusCode, fresh)
	}
	newCodes := fresh["recoveryCodes"].([]any)

	// ganti password: validasi lalu berhasil
	body := func(cur, next, code string) string {
		return `{"currentPassword":"` + cur + `","newPassword":"` + next + `","code":"` + code + `"}`
	}
	if res, _ := e.do(t, "POST", "/api/profile/password", body(pwOld, "pendek", newCodes[0].(string)), cookie, nil); res.StatusCode != 400 {
		t.Fatalf("password lemah harus 400: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "POST", "/api/profile/password", body(pwOld, pwOld, newCodes[0].(string)), cookie, nil); res.StatusCode != 400 {
		t.Fatalf("password sama harus 400: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "POST", "/api/profile/password", body(pwOld, pwNew, recovery[1]), cookie, nil); res.StatusCode != 401 {
		t.Fatalf("kode pemulihan lama (sudah hangus) harus 401: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "POST", "/api/profile/password", body(pwOld, pwNew, newCodes[0].(string)), cookie, nil); res.StatusCode != 200 {
		t.Fatalf("ganti password: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "POST", "/api/auth/login", `{"username":"owner","password":"`+pwOld+`"}`, "", nil); res.StatusCode != 401 {
		t.Fatalf("password lama harus ditolak: %d", res.StatusCode)
	}
	if res, body := e.do(t, "POST", "/api/auth/login", `{"username":"owner","password":"`+pwNew+`"}`, "", nil); res.StatusCode != 200 || body["mfaRequired"] != true {
		t.Fatalf("password baru harus diterima: %d %v", res.StatusCode, body)
	}
}
