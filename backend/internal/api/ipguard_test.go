package api

import (
	"net/http/httptest"
	"testing"

	"ayumi-link/internal/auth"
	"ayumi-link/internal/claude"
	"ayumi-link/internal/config"
	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
)

// newIPEnv: server di balik "proxy tepercaya" (0.0.0.0/0 = alamat soket app.Test) dengan allowlist.
func newIPEnv(t *testing.T, allowed ...string) *env {
	t.Helper()
	dir := t.TempDir()
	list, err := config.ParseIPList(allowed)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		DataDir: dir, ClaudeHome: dir, DesktopDir: dir, PermissionMode: "auto", AuthRateLimit: 1000,
		TrustProxy: true, ProxyHeader: "X-Forwarded-For", TrustedProxies: []string{"0.0.0.0/0"},
		AllowedIPs: list,
	}
	st := store.New(dir)
	am, err := auth.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	br := fsbrowse.New([]string{dir})
	svc := &claude.Service{Home: dir, DesktopDir: dir, Store: st}
	run := &claude.Runner{Svc: svc, Browser: br, Store: st, Bin: "false", DefaultPerm: "auto"}
	return &env{app: New(Deps{Config: cfg, Auth: am, Svc: svc, Runner: run, Browser: br, Store: st}), auth: am}
}

func statusFor(t *testing.T, e *env, path, xff string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	res, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	return res.StatusCode, nil
}

func TestIPAllowlistBlocksUnlistedClients(t *testing.T) {
	e := newIPEnv(t, "203.0.113.7", "192.168.1.0/24")

	for _, ip := range []string{"203.0.113.7", "192.168.1.99"} {
		if code, _ := statusFor(t, e, "/api/health", ip); code != 200 {
			t.Errorf("%s harus diizinkan, dapat %d", ip, code)
		}
	}
	for _, ip := range []string{"198.51.100.9", "192.168.2.1", "127.0.0.1"} {
		if code, _ := statusFor(t, e, "/api/health", ip); code != 403 {
			t.Errorf("%s harus ditolak (403), dapat %d", ip, code)
		}
	}
	// berlaku juga untuk rute auth/API terproteksi, sebelum pemeriksaan login
	if code, _ := statusFor(t, e, "/api/auth/status", "198.51.100.9"); code != 403 {
		t.Errorf("rute auth harus ikut dibatasi: %d", code)
	}
	if code, _ := statusFor(t, e, "/api/sessions", "198.51.100.9"); code != 403 {
		t.Errorf("rute terproteksi dari IP asing harus 403 (bukan 401): %d", code)
	}
}

func TestIPAllowlistDisabledAllowsEveryone(t *testing.T) {
	e := newIPEnv(t) // kosong
	if code, _ := statusFor(t, e, "/api/health", "198.51.100.9"); code != 200 {
		t.Fatalf("tanpa allowlist semua diizinkan: %d", code)
	}
}

func TestIPAllowlistIgnoresHeaderFromUntrustedPeer(t *testing.T) {
	// Proxy tepercaya hanya 10.0.0.1; peer app.Test (0.0.0.0) bukan proxy tepercaya,
	// jadi header X-Forwarded-For harus DIABAIKAN dan alamat soket yang dipakai.
	dir := t.TempDir()
	list, _ := config.ParseIPList([]string{"203.0.113.7"})
	cfg := &config.Config{
		DataDir: dir, ClaudeHome: dir, DesktopDir: dir, PermissionMode: "auto", AuthRateLimit: 1000,
		TrustProxy: true, ProxyHeader: "X-Forwarded-For", TrustedProxies: []string{"10.0.0.1"}, AllowedIPs: list,
	}
	st := store.New(dir)
	am, _ := auth.Open(dir)
	br := fsbrowse.New([]string{dir})
	svc := &claude.Service{Home: dir, DesktopDir: dir, Store: st}
	run := &claude.Runner{Svc: svc, Browser: br, Store: st, Bin: "false", DefaultPerm: "auto"}
	e := &env{app: New(Deps{Config: cfg, Auth: am, Svc: svc, Runner: run, Browser: br, Store: st}), auth: am}

	if code, _ := statusFor(t, e, "/api/health", "203.0.113.7"); code != 403 {
		t.Fatalf("header dari peer tak tepercaya tidak boleh memalsukan IP: %d", code)
	}
}
