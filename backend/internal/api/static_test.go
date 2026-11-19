package api

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ayumi-link/internal/auth"
	"ayumi-link/internal/claude"
	"ayumi-link/internal/config"
	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
)

func newStaticEnv(t *testing.T) *env {
	t.Helper()
	web := t.TempDir()
	if err := os.MkdirAll(filepath.Join(web, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	must := func(p, body string) {
		if err := os.WriteFile(filepath.Join(web, p), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("index.html", "<!doctype html><title>Ayumi Link</title><div id=root></div>")
	must("favicon.svg", "<svg/>")
	must("assets/app-abc123.js", "console.log('app')")

	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir, ClaudeHome: dir, DesktopDir: dir, PermissionMode: "auto", AuthRateLimit: 1000, StaticDir: web}
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

func get(t *testing.T, e *env, method, path string) (int, string, map[string]string) {
	t.Helper()
	res, err := e.app.Test(httptest.NewRequest(method, path, nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	h := map[string]string{}
	for k := range res.Header {
		h[k] = res.Header.Get(k)
	}
	return res.StatusCode, string(b), h
}

func TestFrontendServedWithSPAFallback(t *testing.T) {
	e := newStaticEnv(t)

	for _, path := range []string{"/", "/s/08f379ef-64ab-4504-8dbc-fd6f564c33e1", "/halaman/apa-saja"} {
		code, body, h := get(t, e, "GET", path)
		if code != 200 || !strings.Contains(body, "Ayumi Link") {
			t.Errorf("%s harus menyajikan index.html: %d %q", path, code, body)
		}
		if !strings.Contains(h["Content-Type"], "text/html") || h["Cache-Control"] != "no-cache" {
			t.Errorf("%s header: %v", path, h)
		}
	}
	if code, body, _ := get(t, e, "GET", "/favicon.svg"); code != 200 || body != "<svg/>" {
		t.Errorf("file statis: %d %q", code, body)
	}
	code, body, h := get(t, e, "GET", "/assets/app-abc123.js")
	if code != 200 || !strings.Contains(body, "console.log") || !strings.Contains(h["Cache-Control"], "max-age=31536000") {
		t.Errorf("aset ber-hash harus di-cache lama: %d %v", code, h)
	}
}

func TestAPINeverFallsBackToIndex(t *testing.T) {
	e := newStaticEnv(t)
	code, body, h := get(t, e, "GET", "/api/health")
	if code != 200 || !strings.Contains(body, `"ok":true`) || h["Cache-Control"] != "no-store" {
		t.Errorf("health: %d %q %v", code, body, h)
	}
	if code, body, _ := get(t, e, "GET", "/api/tidak-ada"); code != 401 || strings.Contains(body, "<!doctype") {
		t.Errorf("rute API tak dikenal harus JSON 401, bukan index.html: %d %q", code, body)
	}
	if code, _, _ := get(t, e, "POST", "/halaman/apa-saja"); code == 200 {
		t.Errorf("POST ke rute klien tidak boleh dijawab index.html: %d", code)
	}
}

func TestSecurityHeadersOnFrontend(t *testing.T) {
	e := newStaticEnv(t)
	_, _, h := get(t, e, "GET", "/")
	policy := h["Content-Security-Policy"]
	for _, want := range []string{"default-src 'self'", "frame-ancestors 'none'", "object-src 'none'", "img-src 'self' data:"} {
		if !strings.Contains(policy, want) {
			t.Errorf("CSP kurang %q: %s", want, policy)
		}
	}
	if strings.Contains(policy, "http") {
		t.Errorf("CSP tidak boleh mengizinkan sumber eksternal: %s", policy)
	}
	if h["X-Frame-Options"] != "DENY" || h["X-Content-Type-Options"] != "nosniff" {
		t.Errorf("header keamanan: %v", h)
	}
}

func TestStaticPathTraversalBlocked(t *testing.T) {
	e := newStaticEnv(t)
	for _, p := range []string{"/../../etc/passwd", "/..%2f..%2fetc%2fpasswd", "/assets/../../etc/passwd"} {
		code, body, _ := get(t, e, "GET", p)
		if strings.Contains(body, "root:") {
			t.Fatalf("traversal bocor di %s (%d)", p, code)
		}
	}
}

func TestNoFrontendWhenStaticDirMissing(t *testing.T) {
	e := newEnv(t) // tanpa StaticDir
	if code, _, _ := get(t, e, "GET", "/"); code != 404 {
		t.Errorf("tanpa frontend, / harus 404: %d", code)
	}
}
