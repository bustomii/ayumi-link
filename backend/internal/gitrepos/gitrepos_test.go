package gitrepos

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func TestStatusDiffAndConfig(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("satu\n"), 0o644)
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "awal")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("dua\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "baru.txt"), []byte("isi\n"), 0o644)

	if err := Save(dir, []string{dir}); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); len(got) != 1 || got[0] != dir {
		t.Fatalf("load: %v", got)
	}
	files, err := Status(dir)
	if err != nil {
		t.Fatal(err)
	}
	// .ayumi tidak boleh ikut muncul.
	if len(files) != 2 || files[0].Path != "a.txt" || files[0].Status != "modified" || files[1].Status != "untracked" {
		t.Fatalf("status: %+v", files)
	}
	d, err := Diff(dir, files[0])
	if err != nil || !strings.Contains(d, "+dua") || !strings.Contains(d, "-satu") {
		t.Fatalf("diff: %q %v", d, err)
	}
	d, _ = Diff(dir, files[1])
	if !strings.Contains(d, "+isi") {
		t.Fatalf("diff baru: %q", d)
	}
	sub := filepath.Join(dir, "x", "y")
	os.MkdirAll(sub, 0o755)
	if Root(sub) != dir {
		t.Fatalf("root: %q", Root(sub))
	}
}
