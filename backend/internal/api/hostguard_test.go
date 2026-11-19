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

func newHostEnv(t *testing.T, hosts []string, trustProxy bool) *env {
	t.Helper()
	dir := t.TempDir()
	list, err := config.ParseHostList(hosts)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{DataDir: dir, ClaudeHome: dir, DesktopDir: dir, PermissionMode: "auto", AuthRateLimit: 1000,
		AllowedHosts: list, TrustProxy: trustProxy, ProxyHeader: "X-Forwarded-For", TrustedProxies: []string{"0.0.0.0/0"}}
	st := store.New(dir)
	am, _ := auth.Open(dir)
	br := fsbrowse.New([]string{dir})
	svc := &claude.Service{Home: dir, DesktopDir: dir, Store: st}
	run := &claude.Runner{Svc: svc, Browser: br, Store: st, Bin: "false", DefaultPerm: "auto"}
	return &env{app: New(Deps{Config: cfg, Auth: am, Svc: svc, Runner: run, Browser: br, Store: st}), auth: am}
}

func hostStatus(t *testing.T, e *env, target string, hdr map[string]string) (int, string) {
	t.Helper()
	req := httptest.NewRequest("GET", target, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body := make([]byte, 200)
	n, _ := res.Body.Read(body)
	return res.StatusCode, string(body[:n])
}

func TestHostAllowlistBlocksOtherHosts(t *testing.T) {
	e := newHostEnv(t, []string{"link.neonn.my.id", "localhost"}, false)

	for _, u := range []string{"http://link.neonn.my.id/api/health", "http://LINK.NEONN.MY.ID:8081/api/health", "http://localhost:8081/api/health"} {
		if code, _ := hostStatus(t, e, u, nil); code != 200 {
			t.Errorf("%s harus diizinkan: %d", u, code)
		}
	}
	for _, u := range []string{"http://evil.com/api/health", "http://link.neonn.my.id.evil.com/api/health", "http://127.0.0.1:8081/api/health", "http://neonn.my.id/"} {
		code, body := hostStatus(t, e, u, nil)
		if code != 403 || !contains(body, "host_blocked") {
			t.Errorf("%s harus ditolak (403 host_blocked): %d %s", u, code, body)
		}
	}
	// berlaku juga untuk rute terproteksi sebelum pemeriksaan login
	if code, _ := hostStatus(t, e, "http://evil.com/api/sessions", nil); code != 403 {
		t.Errorf("rute terproteksi dari host asing harus 403 (bukan 401): %d", code)
	}
}

func TestHostAllowlistIgnoresForwardedHostFromUntrustedClients(t *testing.T) {
	// Tanpa TrustProxy, X-Forwarded-Host tidak boleh dipakai untuk melewati pemeriksaan.
	e := newHostEnv(t, []string{"link.neonn.my.id"}, false)
	if code, _ := hostStatus(t, e, "http://evil.com/api/health", map[string]string{"X-Forwarded-Host": "link.neonn.my.id"}); code != 403 {
		t.Fatalf("X-Forwarded-Host dari klien tak tepercaya harus diabaikan: %d", code)
	}
}

func TestHostAllowlistHonorsForwardedHostFromTrustedProxy(t *testing.T) {
	// Di balik proxy tepercaya yang menulis ulang Host, X-Forwarded-Host menentukan domain asli.
	e := newHostEnv(t, []string{"link.neonn.my.id"}, true)
	if code, _ := hostStatus(t, e, "http://127.0.0.1:8081/api/health", map[string]string{"X-Forwarded-Host": "link.neonn.my.id"}); code != 200 {
		t.Errorf("X-Forwarded-Host dari proxy tepercaya harus dihormati: %d", code)
	}
	if code, _ := hostStatus(t, e, "http://127.0.0.1:8081/api/health", map[string]string{"X-Forwarded-Host": "evil.com"}); code != 403 {
		t.Errorf("X-Forwarded-Host asing harus ditolak: %d", code)
	}
	if code, _ := hostStatus(t, e, "http://127.0.0.1:8081/api/health", nil); code != 403 {
		t.Errorf("tanpa header forward, host mentah dipakai dan harus ditolak: %d", code)
	}
}

func TestHostAllowlistDisabledAllowsAll(t *testing.T) {
	e := newHostEnv(t, nil, false)
	if code, _ := hostStatus(t, e, "http://apa.saja/api/health", nil); code != 200 {
		t.Fatalf("tanpa allowlist semua host diizinkan: %d", code)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func originStatus(t *testing.T, e *env, target, origin string, hdr map[string]string) int {
	t.Helper()
	req := httptest.NewRequest("GET", target, nil)
	req.Header.Set("Origin", origin)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	return res.StatusCode
}

// Regresi: pemeriksaan same-origin harus bekerja untuk domain non-loopback (mis. lewat tunnel).
func TestOriginGuardSameOriginOnRealDomain(t *testing.T) {
	e := newHostEnv(t, nil, false)
	if c := originStatus(t, e, "http://link.neonn.my.id/api/health", "http://link.neonn.my.id", nil); c != 200 {
		t.Errorf("origin sama dengan Host harus lolos: %d", c)
	}
	if c := originStatus(t, e, "http://link.neonn.my.id:8081/api/health", "http://link.neonn.my.id:8081", nil); c != 200 {
		t.Errorf("origin+port sama dengan Host harus lolos: %d", c)
	}
	if c := originStatus(t, e, "http://link.neonn.my.id/api/health", "https://evil.com", nil); c != 403 {
		t.Errorf("origin asing harus ditolak: %d", c)
	}
	if c := originStatus(t, e, "http://link.neonn.my.id/api/health", "http://link.neonn.my.id:9999", nil); c != 403 {
		t.Errorf("port berbeda = origin berbeda, harus ditolak: %d", c)
	}
}

func TestOriginGuardBehindTrustedProxyUsesForwardedHost(t *testing.T) {
	e := newHostEnv(t, nil, true) // proxy tepercaya menulis ulang Host ke alamat internal
	hdr := map[string]string{"X-Forwarded-Host": "link.neonn.my.id"}
	if c := originStatus(t, e, "http://127.0.0.1:8081/api/health", "https://link.neonn.my.id", hdr); c != 200 {
		t.Errorf("origin cocok dengan X-Forwarded-Host dari proxy tepercaya harus lolos: %d", c)
	}
	if c := originStatus(t, e, "http://127.0.0.1:8081/api/health", "https://evil.com", hdr); c != 403 {
		t.Errorf("origin asing harus ditolak: %d", c)
	}
}
