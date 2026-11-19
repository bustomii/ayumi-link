package claude

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveBinOverrideWins(t *testing.T) {
	if got := ResolveBin("/opt/claude", t.TempDir()); got != "/opt/claude" {
		t.Fatalf("override: %q", got)
	}
}

func TestResolveBinSkipsMacBundleOffDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("hanya relevan di Linux/Docker")
	}
	desk := t.TempDir()
	bundle := filepath.Join(desk, "claude-code", "9.9.9", "hash", "claude.app", "Contents", "MacOS")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(bundle, "claude")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ResolveBin("", desk); got != "claude" {
		t.Fatalf("di Linux harus PATH claude, bukan bundle macOS: %q", got)
	}
}
