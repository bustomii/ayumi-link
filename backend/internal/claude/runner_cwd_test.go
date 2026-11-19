package claude

import (
	"os"
	"path/filepath"
	"testing"

	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
)

func TestResolveNewCwdRequiresRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "proj")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	r := &Runner{Browser: fsbrowse.New([]string{root})}

	wantInside, ok := r.Browser.ResolveDir(inside)
	if !ok {
		t.Fatal("fixture inside harus resolve")
	}
	got, err := r.resolveNewCwd(inside)
	if err != nil || got != wantInside {
		t.Fatalf("inside: got=%q want=%q err=%v", got, wantInside, err)
	}
	if _, err := r.resolveNewCwd(outside); err == nil {
		t.Fatal("outside root harus ditolak")
	}
	wantRoot := r.Browser.Roots()[0]
	got, err = r.resolveNewCwd("")
	if err != nil || got != wantRoot {
		t.Fatalf("default root: got=%q want=%q err=%v", got, wantRoot, err)
	}
}

func TestResolveResumeCwdIgnoresClientAndEnforcesRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "app")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	st := store.New(t.TempDir())
	st.Register(sid, inside)

	r := &Runner{
		Browser: fsbrowse.New([]string{root}),
		Store:   st,
		Svc:     &Service{Home: t.TempDir(), DesktopDir: t.TempDir(), Store: st},
	}

	wantInside, ok := r.Browser.ResolveDir(inside)
	if !ok {
		t.Fatal("fixture inside harus resolve")
	}
	got, err := r.resolveResumeCwd(sid)
	if err != nil || got != wantInside {
		t.Fatalf("registry cwd: got=%q want=%q err=%v", got, wantInside, err)
	}

	// Registry diarahkan ke luar roots → ditolak (meski folder ada).
	st.Register(sid, outside)
	if _, err := r.resolveResumeCwd(sid); err == nil {
		t.Fatal("cwd di luar roots harus ditolak")
	}
}
