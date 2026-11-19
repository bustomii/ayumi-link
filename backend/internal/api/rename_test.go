package api

import (
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

func TestCleanName(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
		ok   bool
	}{
		{"  halo   dunia  ", 20, "halo dunia", true},
		{"baris\nbaru\tdan\x00kontrol", 40, "baris baru dan kontrol", true},
		{"", 10, "", true},
		{"   ", 10, "", true},
		{strings.Repeat("a", 11), 10, strings.Repeat("a", 11), false},
		{strings.Repeat("é", 10), 10, strings.Repeat("é", 10), true}, // dihitung per karakter, bukan byte
	}
	for _, c := range cases {
		got, ok := cleanName(c.in, c.max)
		if got != c.want || ok != c.ok {
			t.Errorf("cleanName(%q,%d) = %q,%v; want %q,%v", c.in, c.max, got, ok, c.want, c.ok)
		}
	}
}

const renameSID = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"

// newRenameEnv: lingkungan dengan satu session lokal ("demo") yang sudah punya transcript.
func newRenameEnv(t *testing.T) (*env, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir, ClaudeHome: dir, DesktopDir: dir, PermissionMode: "auto", AuthRateLimit: 1000}
	st := store.New(dir)
	am, err := auth.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	br := fsbrowse.New([]string{dir})
	svc := &claude.Service{Home: dir, DesktopDir: dir, Store: st}
	run := &claude.Runner{Svc: svc, Browser: br, Store: st, Bin: "false", DefaultPerm: "auto"}

	pdir := filepath.Join(dir, "projects", "-tmp-demo")
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","uuid":"u1","timestamp":"2026-11-18T10:00:00Z","cwd":"/tmp/demo","message":{"role":"user","content":"pertanyaan awal"}}`
	if err := os.WriteFile(filepath.Join(pdir, renameSID+".jsonl"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	st.Register(renameSID, "/tmp/demo")

	e := &env{app: New(Deps{Config: cfg, Auth: am, Svc: svc, Runner: run, Browser: br, Store: st}), auth: am}
	return e, dir
}

func firstSession(t *testing.T, e *env, cookie string) map[string]any {
	t.Helper()
	_, body := e.do(t, "GET", "/api/sessions", "", cookie, nil)
	list, _ := body["sessions"].([]any)
	if len(list) != 1 {
		t.Fatalf("harus ada 1 session, dapat %d", len(list))
	}
	return list[0].(map[string]any)
}

func TestRenameEndpoints(t *testing.T) {
	e, _ := newRenameEnv(t)

	// tanpa login ditolak
	if res, _ := e.do(t, "PUT", "/api/sessions/"+renameSID+"/title", `{"title":"x"}`, "", nil); res.StatusCode != 401 {
		t.Fatalf("rename tanpa login harus 401: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "PUT", "/api/groups/rename", `{"from":"demo","to":"x"}`, "", nil); res.StatusCode != 401 {
		t.Fatalf("rename folder tanpa login harus 401: %d", res.StatusCode)
	}

	cookie, _, _ := setupAccount(t, e)
	if s := firstSession(t, e, cookie); s["title"] != "pertanyaan awal" || s["group"] != "demo" {
		t.Fatalf("awal: %v", s)
	}

	// ubah nama session (spasi/baris baru dirapikan)
	res, body := e.do(t, "PUT", "/api/sessions/"+renameSID+"/title", `{"title":"  Judul\nBaru  "}`, cookie, nil)
	if res.StatusCode != 200 || body["title"] != "Judul Baru" {
		t.Fatalf("rename session: %d %v", res.StatusCode, body)
	}
	if s := firstSession(t, e, cookie); s["title"] != "Judul Baru" {
		t.Fatalf("daftar harus memakai nama baru: %v", s)
	}
	_, detail := e.do(t, "GET", "/api/sessions/"+renameSID, "", cookie, nil)
	if got := detail["session"].(map[string]any)["title"]; got != "Judul Baru" {
		t.Fatalf("detail harus memakai nama baru: %v", got)
	}

	// kosong = kembali ke nama asli
	e.do(t, "PUT", "/api/sessions/"+renameSID+"/title", `{"title":""}`, cookie, nil)
	if s := firstSession(t, e, cookie); s["title"] != "pertanyaan awal" {
		t.Fatalf("reset: %v", s)
	}

	// ubah nama folder
	res, body = e.do(t, "PUT", "/api/groups/rename", `{"from":"demo","to":"  Proyek   Baru "}`, cookie, nil)
	if res.StatusCode != 200 || body["group"] != "Proyek Baru" {
		t.Fatalf("rename folder: %d %v", res.StatusCode, body)
	}
	if s := firstSession(t, e, cookie); s["group"] != "Proyek Baru" {
		t.Fatalf("folder baru: %v", s)
	}

	// validasi
	bad := []struct{ method, path, body string }{
		{"PUT", "/api/sessions/bukan-uuid/title", `{"title":"x"}`},
		{"PUT", "/api/sessions/" + renameSID + "/title", `{"title":"` + strings.Repeat("a", 121) + `"}`},
		{"PUT", "/api/groups/rename", `{"from":"demo","to":""}`},
		{"PUT", "/api/groups/rename", `{"from":"","to":"x"}`},
		{"PUT", "/api/groups/rename", `{"from":"demo","to":"` + strings.Repeat("a", 61) + `"}`},
		{"PUT", "/api/sessions/" + renameSID + "/title", `bukan json`},
	}
	for _, b := range bad {
		if res, _ := e.do(t, b.method, b.path, b.body, cookie, nil); res.StatusCode != 400 {
			t.Errorf("%s %s %.40q harus 400, dapat %d", b.method, b.path, b.body, res.StatusCode)
		}
	}
}
