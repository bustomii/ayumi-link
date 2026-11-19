package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCliAuthSaveClearApplyEnv(t *testing.T) {
	dir := t.TempDir()
	a, err := OpenCliAuth(dir, "false", dir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := a.Save(context.Background(), CliCredOAuth, "sk-ant-oat-test-token-1234567890")
	if err != nil {
		t.Fatal(err)
	}
	if !st.LoggedIn || !st.Stored || st.StoredMethod != CliCredOAuth {
		t.Fatalf("status setelah save: %+v", st)
	}
	if _, err := os.Stat(filepath.Join(dir, "claude-cli-auth.json")); err != nil {
		t.Fatal(err)
	}

	env := a.ApplyEnv([]string{"PATH=/bin", "CLAUDE_CODE_OAUTH_TOKEN=old", "FOO=1"})
	found, old := false, false
	for _, e := range env {
		if e == "CLAUDE_CODE_OAUTH_TOKEN=sk-ant-oat-test-token-1234567890" {
			found = true
		}
		if e == "CLAUDE_CODE_OAUTH_TOKEN=old" {
			old = true
		}
	}
	if !found || old {
		t.Fatalf("ApplyEnv salah: %v", env)
	}

	if err := a.Clear(); err != nil {
		t.Fatal(err)
	}
	if a.Stored() != nil {
		t.Fatal("masih tersimpan setelah Clear")
	}
	if _, err := os.Stat(filepath.Join(dir, "claude-cli-auth.json")); !os.IsNotExist(err) {
		t.Fatalf("file harus hilang: %v", err)
	}
}

func TestCliAuthRejectsShortToken(t *testing.T) {
	a, err := OpenCliAuth(t.TempDir(), "false", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Save(context.Background(), CliCredOAuth, "short")
	if err == nil {
		t.Fatal("harus tolak token pendek")
	}
}

func TestSanitizeTokenStripsWhitespace(t *testing.T) {
	got := sanitizeToken("sk-ant-oat 01\nabc\t")
	if got != "sk-ant-oat01abc" {
		t.Fatalf("got %q", got)
	}
}

func TestCliAuthLoadStripsWhitespace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-cli-auth.json")
	raw := []byte(`{"method":"oauth_token","token":"sk-ant-oat01-test-token-with spaces XX","updatedAt":"2026-11-18T00:00:00Z"}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := OpenCliAuth(dir, "false", dir)
	if err != nil {
		t.Fatal(err)
	}
	st := a.Stored()
	if st == nil || strings.Contains(st.Token, " ") {
		t.Fatalf("token harus tanpa spasi: %#v", st)
	}
	if st.Token != "sk-ant-oat01-test-token-withspacesXX" {
		t.Fatalf("got %q", st.Token)
	}
}
