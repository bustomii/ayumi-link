package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Metode kredensial CLI yang disimpan Ayumi Link (diinjeksikan ke proses claude).
const (
	CliCredOAuth  = "oauth_token" // CLAUDE_CODE_OAUTH_TOKEN (claude setup-token)
	CliCredAPIKey = "api_key"     // ANTHROPIC_API_KEY
)

// CliCreds adalah token yang disimpan di data dir (0600), bukan di .claude.json host.
type CliCreds struct {
	Method    string    `json:"method"`
	Token     string    `json:"token"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CliAuthStatus hasil GET status (tidak pernah memuat token penuh).
type CliAuthStatus struct {
	LoggedIn       bool   `json:"loggedIn"`
	AuthMethod     string `json:"authMethod"`
	Stored         bool   `json:"stored"`
	StoredMethod   string `json:"storedMethod,omitempty"`
	TokenHint      string `json:"tokenHint,omitempty"`
	LoginExpiresAt string `json:"loginExpiresAt,omitempty"`
	Source         string `json:"source,omitempty"` // stored | env | none
	Message        string `json:"message,omitempty"`
}

type cliAuthFile struct {
	Method    string `json:"method"`
	Token     string `json:"token"`
	UpdatedAt string `json:"updatedAt"`
}

// CliAuth menyimpan + menerapkan kredensial Claude CLI untuk proses anak.
type CliAuth struct {
	mu   sync.RWMutex
	path string
	bin  string
	home string
	cur  *CliCreds
}

// OpenCliAuth memuat kredensial tersimpan dari dataDir (boleh kosong).
func OpenCliAuth(dataDir, bin, claudeHome string) (*CliAuth, error) {
	a := &CliAuth{
		path: filepath.Join(dataDir, "claude-cli-auth.json"),
		bin:  bin,
		home: claudeHome,
	}
	if err := a.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return a, nil
}

func (a *CliAuth) load() error {
	raw, err := os.ReadFile(a.path)
	if err != nil {
		return err
	}
	var f cliAuthFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	f.Method = strings.TrimSpace(f.Method)
	clean := sanitizeToken(f.Token)
	if clean == "" || (f.Method != CliCredOAuth && f.Method != CliCredAPIKey) {
		return nil
	}
	t, _ := time.Parse(time.RFC3339, f.UpdatedAt)
	a.mu.Lock()
	a.cur = &CliCreds{Method: f.Method, Token: clean, UpdatedAt: t}
	if clean != f.Token {
		_ = a.persistLocked() // hapus whitespace dari file lama
	}
	a.mu.Unlock()
	return nil
}

// sanitizeToken buang semua whitespace (copy-paste setup-token sering bawa spasi/newline).
func sanitizeToken(token string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, token)
}

func (a *CliAuth) persistLocked() error {
	if a.cur == nil {
		_ = os.Remove(a.path)
		return nil
	}
	f := cliAuthFile{
		Method:    a.cur.Method,
		Token:     a.cur.Token,
		UpdatedAt: a.cur.UpdatedAt.UTC().Format(time.RFC3339),
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(a.path), 0o700); err != nil {
		return err
	}
	tmp := a.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, a.path)
}

// Stored mengembalikan salinan kredensial tersimpan (untuk tes); nil bila tidak ada.
func (a *CliAuth) Stored() *CliCreds {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.cur == nil {
		return nil
	}
	cp := *a.cur
	return &cp
}

// ApplyEnv menempelkan token tersimpan ke env proses anak (mengganti key yang sama).
func (a *CliAuth) ApplyEnv(env []string) []string {
	if a == nil {
		return env
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.cur == nil || a.cur.Token == "" {
		return env
	}
	out := filterEnvKeys(env, "CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_API_KEY")
	switch a.cur.Method {
	case CliCredAPIKey:
		out = append(out, "ANTHROPIC_API_KEY="+a.cur.Token)
	default:
		out = append(out, "CLAUDE_CODE_OAUTH_TOKEN="+a.cur.Token)
	}
	return out
}

func filterEnvKeys(env []string, keys ...string) []string {
	drop := map[string]bool{}
	for _, k := range keys {
		drop[k] = true
	}
	out := make([]string, 0, len(env))
	for _, e := range env {
		k, _, _ := strings.Cut(e, "=")
		if drop[k] {
			continue
		}
		out = append(out, e)
	}
	return out
}

func tokenHint(tok string) string {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return ""
	}
	if len(tok) <= 4 {
		return "••••"
	}
	return "••••" + tok[len(tok)-4:]
}

// Clear menghapus kredensial tersimpan di Ayumi (tidak mengubah env proses induk).
func (a *CliAuth) Clear() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cur = nil
	return a.persistLocked()
}

// Save memverifikasi token lewat `claude auth status` lalu menyimpannya.
func (a *CliAuth) Save(ctx context.Context, method, token string) (*CliAuthStatus, error) {
	if a == nil {
		return nil, &RequestError{Status: 500, Message: "auth CLI tidak tersedia"}
	}
	method = strings.TrimSpace(method)
	token = sanitizeToken(token)
	if method != CliCredOAuth && method != CliCredAPIKey {
		return nil, &RequestError{Status: 400, Message: "method harus oauth_token atau api_key"}
	}
	if len(token) < 20 {
		return nil, &RequestError{Status: 400, Message: "token terlalu pendek"}
	}

	tmp := &CliCreds{Method: method, Token: token, UpdatedAt: time.Now().UTC()}
	// auth status saja tidak cukup: CLI menganggap oauth_token env apa pun "loggedIn".
	// Verifikasi dengan panggilan -p singkat.
	if err := a.verifyChat(ctx, tmp); err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.cur = tmp
	if err := a.persistLocked(); err != nil {
		a.mu.Unlock()
		return nil, err
	}
	a.mu.Unlock()
	return a.Status(ctx)
}

// Status menjalankan `claude auth status` dengan env tersimpan (jika ada).
func (a *CliAuth) Status(ctx context.Context) (*CliAuthStatus, error) {
	if a == nil {
		return &CliAuthStatus{Message: "auth CLI tidak tersedia"}, nil
	}
	a.mu.RLock()
	cur := a.cur
	a.mu.RUnlock()
	return a.probe(ctx, cur)
}

// verifyChat memastikan token benar-benar dipakai chat (-p), bukan hanya status palsu.
func (a *CliAuth) verifyChat(ctx context.Context, creds *CliCreds) error {
	if a.bin == "" || a.bin == "false" {
		return nil // uji unit
	}
	cmd := exec.CommandContext(ctx, a.bin, "-p", "jawab hanya: OK", "--output-format", "text", "--model", "claude-haiku-4-5")
	env := filterEnvKeys(os.Environ(), "CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_API_KEY")
	if creds.Method == CliCredAPIKey {
		env = append(env, "ANTHROPIC_API_KEY="+creds.Token)
	} else {
		env = append(env, "CLAUDE_CODE_OAUTH_TOKEN="+creds.Token)
	}
	// HOME sementara agar .claude.json mount yang rusak tidak menghalangi.
	tmpHome, err := os.MkdirTemp("", "ayumi-cli-auth-*")
	if err == nil {
		defer os.RemoveAll(tmpHome)
		_ = os.WriteFile(filepath.Join(tmpHome, ".claude.json"), []byte("{}"), 0o600)
		env = filterEnvKeys(env, "HOME")
		env = append(env, "HOME="+tmpHome)
		cmd.Dir = tmpHome
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if strings.Contains(strings.ToLower(msg), "not logged in") {
			return &RequestError{Status: 400, Message: "token tidak dipakai Claude untuk chat. Salin ulang dari `claude setup-token` (tanpa spasi) atau coba API key."}
		}
		return &RequestError{Status: 400, Message: truncateMsg(msg, 240)}
	}
	return nil
}

func (a *CliAuth) probe(ctx context.Context, creds *CliCreds) (*CliAuthStatus, error) {
	out := &CliAuthStatus{Source: "none"}
	if creds != nil && creds.Token != "" {
		out.Stored = true
		out.StoredMethod = creds.Method
		out.TokenHint = tokenHint(creds.Token)
		out.Source = "stored"
	} else if os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") != "" || os.Getenv("ANTHROPIC_API_KEY") != "" {
		out.Source = "env"
	}

	if a.bin == "" || a.bin == "false" {
		// Uji unit tanpa CLI sungguhan.
		if creds != nil && creds.Token != "" {
			out.LoggedIn = true
			out.AuthMethod = creds.Method
		}
		return out, nil
	}

	cmd := exec.CommandContext(ctx, a.bin, "auth", "status")
	if a.home != "" {
		cmd.Dir = a.home
	}
	env := os.Environ()
	if creds != nil && creds.Token != "" {
		env = filterEnvKeys(env, "CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_API_KEY")
		if creds.Method == CliCredAPIKey {
			env = append(env, "ANTHROPIC_API_KEY="+creds.Token)
		} else {
			env = append(env, "CLAUDE_CODE_OAUTH_TOKEN="+creds.Token)
		}
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // status non-zero tetap bisa berisi JSON

	raw := stdout.Bytes()
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = stderr.Bytes()
	}
	parsed, ok := extractJSONObject(raw)
	if !ok {
		out.Message = "gagal membaca status Claude CLI"
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			out.Message = truncateMsg(msg, 240)
		}
		return out, nil
	}
	if v, _ := parsed["loggedIn"].(bool); v {
		out.LoggedIn = true
	}
	if s, _ := parsed["authMethod"].(string); s != "" {
		out.AuthMethod = s
	}
	switch exp := parsed["loginExpiresAt"].(type) {
	case string:
		out.LoginExpiresAt = exp
	case float64:
		if exp > 0 {
			out.LoginExpiresAt = time.UnixMilli(int64(exp)).UTC().Format(time.RFC3339)
		}
	}
	return out, nil
}

func extractJSONObject(b []byte) (map[string]any, bool) {
	s := string(b)
	i := strings.LastIndex(s, "{")
	if i < 0 {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s[i:]), &m); err != nil {
		return nil, false
	}
	return m, true
}

func truncateMsg(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
