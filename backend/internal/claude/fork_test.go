package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ayumi-link/internal/store"
)

func TestEncodeProjectDir(t *testing.T) {
	if got := encodeProjectDir("/Users/me/.claude/My App"); got != "-Users-me--claude-My-App" {
		t.Fatalf("got %q", got)
	}
}

func TestForkCopiesTranscriptToNewFolder(t *testing.T) {
	home := t.TempDir()
	st := store.New(t.TempDir())
	svc := &Service{Home: home, DesktopDir: t.TempDir(), Store: st}

	oldDir := filepath.Join(home, "projects", encodeProjectDir("/gone/old"))
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lines := []string{
		`{"type":"user","sessionId":"` + sid + `","cwd":"/gone/old","message":{"role":"user","content":"halo <b>"}}`,
		`{"type":"summary","summary":"tanpa sessionId"}`,
		`bukan json`,
		`{"type":"assistant","sessionId":"` + sid + `","cwd":"/gone/old","message":{"role":"assistant","content":[{"type":"text","text":"hai"}]}}`,
	}
	if err := os.WriteFile(filepath.Join(oldDir, sid+".jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	newCwd := t.TempDir()
	newID, err := svc.Fork(sid, newCwd)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if newID == sid || !IsUUID(newID) {
		t.Fatalf("id baru tidak valid: %q", newID)
	}
	dst := filepath.Join(home, "projects", encodeProjectDir(newCwd), newID+".jsonl")
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("file hasil fork tidak ada: %v", err)
	}
	got := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(got) != len(lines) {
		t.Fatalf("jumlah baris %d, want %d", len(got), len(lines))
	}
	if strings.Contains(string(b), sid) || strings.Contains(string(b), "/gone/old") {
		t.Fatalf("id/cwd lama masih ada:\n%s", b)
	}
	if !strings.Contains(got[0], newID) || !strings.Contains(got[0], "<b>") {
		t.Fatalf("baris 1 salah: %s", got[0])
	}
	if got[1] != lines[1] || got[2] != lines[2] {
		t.Fatalf("baris tanpa sessionId / non-JSON harus tetap: %q %q", got[1], got[2])
	}
	if _, err := os.Stat(filepath.Join(oldDir, sid+".jsonl")); err != nil {
		t.Fatal("session asal tidak boleh berubah/hilang")
	}
	if st.Registry()[newID] != newCwd {
		t.Fatal("session baru harus terdaftar dengan folder baru")
	}

	if _, err := svc.Fork("bukan-uuid", newCwd); err != ErrBadID {
		t.Fatalf("want ErrBadID, got %v", err)
	}
	if _, err := svc.Fork("bbbbbbbb-bbbb-4ccc-8ddd-eeeeeeeeeeee", newCwd); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
