package fsbrowse

import (
	"os"
	"path/filepath"
	"testing"
)

func setup(t *testing.T) (root, outside string) {
	t.Helper()
	base, _ := filepath.EvalSymlinks(t.TempDir())
	root = filepath.Join(base, "home")
	outside = filepath.Join(base, "outside")
	for _, d := range []string{"home/projects/app", "home/.ssh", "home/Library", "home/node_modules", "outside/secret"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o600)
	return root, outside
}

func TestListHidesSensitiveAndFiles(t *testing.T) {
	root, _ := setup(t)
	b := New([]string{root})
	l, ok := b.List("")
	if !ok {
		t.Fatal("root harus bisa dibuka")
	}
	got := map[string]bool{}
	for _, d := range l.Dirs {
		got[d.Name] = true
	}
	if !got["projects"] {
		t.Fatal("projects harus tampil")
	}
	for _, hidden := range []string{".ssh", "Library", "node_modules", "file.txt"} {
		if got[hidden] {
			t.Fatalf("%s tidak boleh tampil", hidden)
		}
	}
	if l.Parent != "" {
		t.Fatalf("root tidak boleh punya parent, dapat %q", l.Parent)
	}
}

func TestCannotEscapeRoot(t *testing.T) {
	root, outside := setup(t)
	b := New([]string{root})

	cases := map[string]string{
		"luar root":      outside,
		"traversal":      filepath.Join(root, "..", "outside"),
		"relatif":        "../outside",
		"file biasa":     filepath.Join(root, "file.txt"),
		"tidak ada":      filepath.Join(root, "nope"),
		"kosong":         "",
		"akar sistem":    "/",
		"saudara serupa": root + "-evil",
	}
	for name, p := range cases {
		if _, ok := b.ResolveDir(p); ok {
			t.Errorf("%s (%q) seharusnya ditolak", name, p)
		}
	}
	if _, ok := b.ResolveDir(filepath.Join(root, "projects", "app")); !ok {
		t.Error("subfolder sah harus diterima")
	}
}

func TestSymlinkOutOfRootRejected(t *testing.T) {
	root, outside := setup(t)
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "projects"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	b := New([]string{root})

	if _, ok := b.ResolveDir(filepath.Join(root, "escape")); ok {
		t.Fatal("symlink keluar root harus ditolak")
	}
	if _, ok := b.ResolveDir(filepath.Join(root, "escape", "secret")); ok {
		t.Fatal("masuk lewat symlink keluar root harus ditolak")
	}
	l, _ := b.List("")
	for _, d := range l.Dirs {
		if d.Name == "escape" {
			t.Fatal("symlink keluar root tidak boleh tampil di daftar")
		}
	}
	if _, ok := b.ResolveDir(filepath.Join(root, "alias")); !ok {
		t.Fatal("symlink ke dalam root boleh")
	}
}
