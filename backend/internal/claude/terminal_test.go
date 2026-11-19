package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
)

func TestShellCdPersistsAndStaysInRoots(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "proj")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()

	browser := fsbrowse.New([]string{root})
	sh := NewShell(browser, nil, nil)

	cwd, err := sh.Enter("")
	if err != nil {
		t.Fatal(err)
	}
	wantRoot, ok := browser.ResolveDir(root)
	if !ok || cwd != wantRoot {
		t.Fatalf("enter root: cwd=%q want=%q", cwd, wantRoot)
	}

	res, err := sh.Run(context.Background(), "cd proj && pwd")
	if err != nil {
		t.Fatal(err)
	}
	wantSub, _ := browser.ResolveDir(sub)
	if res.Cwd != wantSub {
		t.Fatalf("setelah cd: cwd=%q want=%q out=%q", res.Cwd, wantSub, res.Stdout)
	}

	res, err = sh.Run(context.Background(), "pwd")
	if err != nil || res.Cwd != wantSub {
		t.Fatalf("cwd bertahan: %+v err=%v", res, err)
	}

	// Coba keluar roots — cwd harus tetap di wantSub.
	res, err = sh.Run(context.Background(), "cd "+outside)
	if err != nil {
		t.Fatal(err)
	}
	if res.Cwd != wantSub {
		t.Fatalf("cwd setelah cd luar: %q want %q stderr=%q", res.Cwd, wantSub, res.Stderr)
	}
}

func TestShellEnterUsesSessionCwd(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	chatDir := filepath.Join(root, "chat-a")
	if err := os.Mkdir(chatDir, 0o755); err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(home, "projects", "-demo")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := filepath.Join(proj, sid+".jsonl")
	if err := os.WriteFile(jsonl, []byte(`{"type":"user","cwd":"`+chatDir+`","message":{"role":"user","content":"x"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := store.New(t.TempDir())
	st.Register(sid, chatDir)
	svc := &Service{Home: home, DesktopDir: t.TempDir(), Store: st}
	browser := fsbrowse.New([]string{root})
	sh := NewShell(browser, svc, nil)

	cwd, err := sh.Enter(sid)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := browser.ResolveDir(chatDir)
	if cwd != want {
		t.Fatalf("enter session: %q want %q", cwd, want)
	}
}
