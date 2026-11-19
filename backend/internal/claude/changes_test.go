package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ayumi-link/internal/store"
)

func TestListChangesAndDiff(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	backend := filepath.Join(root, "backend")
	if err := os.MkdirAll(backend, 0o755); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(backend, "app.go")
	if err := os.WriteFile(abs, []byte("package main\nfunc Ok() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	proj := filepath.Join(home, "projects", "-demo")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	hist := filepath.Join(home, "file-history", sid)
	if err := os.MkdirAll(hist, 0o755); err != nil {
		t.Fatal(err)
	}
	backup := "abc123@v2"
	if err := os.WriteFile(filepath.Join(hist, backup), []byte("package main\nfunc Old() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	jsonl := filepath.Join(proj, sid+".jsonl")
	line := `{"type":"file-history-snapshot","snapshot":{"trackedFileBackups":{"backend/app.go":{"backupFileName":"` + backup + `","version":2,"backupTime":"2026-11-18T10:00:00Z","realParentDir":"` + backend + `"}}}}` + "\n"
	if err := os.WriteFile(jsonl, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &Service{Home: home, DesktopDir: t.TempDir(), Store: store.New(t.TempDir())}
	list, err := svc.ListChanges(sid)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	if list[0].Path != "backend/app.go" || list[0].Status != "modified" {
		t.Fatalf("entry: %+v", list[0])
	}
	diff, err := svc.GetChangeDiff(sid, "backend/app.go")
	if err != nil {
		t.Fatal(err)
	}
	if diff.Before == "" || diff.After == "" || diff.Diff == "" {
		t.Fatalf("diff kosong: %+v", diff)
	}
	if !strings.Contains(diff.Diff, "-func Old()") || !strings.Contains(diff.Diff, "+func Ok()") {
		t.Fatalf("diff: %s", diff.Diff)
	}
}
