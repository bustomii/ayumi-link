package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDataDir(t *testing.T) {
	fresh := filepath.Join(t.TempDir(), "a", "data")
	if err := EnsureDataDir(fresh); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(fresh, ".gitignore")); err != nil || string(b) != "*\n" {
		t.Fatalf("gitignore: %q %v", b, err)
	}
	existing := t.TempDir()
	if err := EnsureDataDir(existing); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(existing, ".gitignore")); !os.IsNotExist(err) {
		t.Fatal("folder yang sudah ada tidak boleh diberi .gitignore")
	}
}
