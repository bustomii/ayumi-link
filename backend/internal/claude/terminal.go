package claude

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"ayumi-link/internal/fsbrowse"
)

const (
	defaultTermTimeout = 30 * time.Second
	maxTermOutput      = 256 * 1024
	pwdMarker          = "___AYUMI_PWD___"
)

// TermResult hasil satu perintah di shell interaktif Ayumi (bukan Bash agent chat).
type TermResult struct {
	Command   string `json:"command"`
	Cwd       string `json:"cwd"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exitCode"`
	TimedOut  bool   `json:"timedOut"`
	Truncated bool   `json:"truncated"`
}

// Shell adalah satu sesi terminal per proses Ayumi Link (akun tunggal).
// Cwd bertahan antar perintah (cd seperti terminal biasa), tetap di dalam AYUMI_ROOTS.
type Shell struct {
	mu      sync.Mutex
	browser *fsbrowse.Browser
	svc     *Service
	auth    *CliAuth
	cwd     string
}

func NewShell(browser *fsbrowse.Browser, svc *Service, auth *CliAuth) *Shell {
	return &Shell{browser: browser, svc: svc, auth: auth}
}

func (sh *Shell) defaultRoot() (string, error) {
	roots := sh.browser.Roots()
	if len(roots) == 0 {
		return "", &RequestError{Status: 500, Message: "AYUMI_ROOTS kosong"}
	}
	return roots[0], nil
}

func (sh *Shell) resolveCwd(p string) (string, bool) {
	return sh.browser.ResolveDir(p)
}

// Enter menyiapkan cwd: folder session chat bila valid di roots, selain itu root AYUMI_ROOTS.
func (sh *Shell) Enter(sessionID string) (string, error) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	root, err := sh.defaultRoot()
	if err != nil {
		return "", err
	}
	target := root
	if sessionID != "" && sh.svc != nil {
		if sc, e := sh.svc.SessionCwd(sessionID); e == nil {
			if real, ok := sh.resolveCwd(sc); ok {
				target = real
			}
		}
	}
	if real, ok := sh.resolveCwd(target); ok {
		sh.cwd = real
		return sh.cwd, nil
	}
	sh.cwd = root
	return sh.cwd, nil
}

// Status mengembalikan cwd saat ini (inisialisasi ke root bila belum ada).
func (sh *Shell) Status() (string, error) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if sh.cwd != "" {
		if real, ok := sh.resolveCwd(sh.cwd); ok {
			sh.cwd = real
			return sh.cwd, nil
		}
	}
	root, err := sh.defaultRoot()
	if err != nil {
		return "", err
	}
	sh.cwd = root
	return sh.cwd, nil
}

// Run menjalankan perintah di cwd shell; cd (termasuk dalam perintah) memperbarui cwd.
func (sh *Shell) Run(ctx context.Context, command string) (*TermResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, &RequestError{Status: 400, Message: "perintah kosong"}
	}
	if len(command) > 8000 {
		return nil, &RequestError{Status: 400, Message: "perintah terlalu panjang"}
	}

	sh.mu.Lock()
	defer sh.mu.Unlock()

	if sh.cwd == "" {
		root, err := sh.defaultRoot()
		if err != nil {
			return nil, err
		}
		sh.cwd = root
	}
	startCwd, ok := sh.resolveCwd(sh.cwd)
	if !ok {
		root, err := sh.defaultRoot()
		if err != nil {
			return nil, err
		}
		startCwd = root
		sh.cwd = root
	}

	timeout := defaultTermTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if d := time.Until(deadline); d > 0 && d < timeout {
			timeout = d
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	script := "cd " + shellQuote(startCwd) + " || exit 126\n" +
		command + "\n" +
		"__ayumi_ec=$?\n" +
		"printf '\\n%s%s\\n' '" + pwdMarker + "' \"$(pwd -P)\"\n" +
		"exit $__ayumi_ec\n"

	cmd := exec.CommandContext(runCtx, "sh", "-c", script)
	cmd.Dir = startCwd
	cmd.Env = append(sh.auth.ApplyEnv(os.Environ()), "TERM=xterm-256color")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitWriter{buf: &stdout, max: maxTermOutput}
	cmd.Stderr = &limitWriter{buf: &stderr, max: maxTermOutput}
	err := cmd.Run()

	outStr, newCwd := splitPwdMarker(stdout.String())
	res := &TermResult{
		Command:   command,
		Cwd:       startCwd,
		Stdout:    outStr,
		Stderr:    stderr.String(),
		TimedOut:  errors.Is(runCtx.Err(), context.DeadlineExceeded),
		Truncated: stdout.Len() >= maxTermOutput || stderr.Len() >= maxTermOutput,
	}
	if newCwd != "" {
		if real, ok := sh.resolveCwd(newCwd); ok {
			sh.cwd = real
			res.Cwd = real
		} else {
			res.Stderr = strings.TrimSpace(res.Stderr + "\nayumi: folder di luar AYUMI_ROOTS diabaikan; tetap di " + startCwd)
			res.Cwd = startCwd
			sh.cwd = startCwd
		}
	} else {
		res.Cwd = sh.cwd
	}

	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
		} else if res.TimedOut {
			res.ExitCode = -1
		} else {
			return nil, &RequestError{Status: 500, Message: "gagal menjalankan perintah: " + err.Error()}
		}
	}
	return res, nil
}

func splitPwdMarker(stdout string) (clean, pwd string) {
	idx := strings.LastIndex(stdout, pwdMarker)
	if idx < 0 {
		return stdout, ""
	}
	clean = strings.TrimSuffix(stdout[:idx], "\n")
	rest := strings.TrimSpace(stdout[idx+len(pwdMarker):])
	pwd = rest
	if i := strings.IndexByte(pwd, '\n'); i >= 0 {
		pwd = strings.TrimSpace(pwd[:i])
	}
	return clean, pwd
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// SessionCwd mengembalikan cwd session dari daftar (Desktop/registry).
func (s *Service) SessionCwd(id string) (string, error) {
	if !IsUUID(id) {
		return "", ErrBadID
	}
	if FindJSONL(s.Home, id) == "" {
		return "", ErrNotFound
	}
	for _, x := range s.List() {
		if x.ID == id && x.Cwd != "" {
			return x.Cwd, nil
		}
	}
	if p := FindJSONL(s.Home, id); p != "" {
		if cwd := cwdFromJSONL(p); cwd != "" {
			return cwd, nil
		}
	}
	return "", errors.New("cwd session tidak diketahui")
}

type limitWriter struct {
	buf *bytes.Buffer
	max int
}

func (w *limitWriter) Write(p []byte) (int, error) {
	remain := w.max - w.buf.Len()
	if remain <= 0 {
		return len(p), nil
	}
	if len(p) > remain {
		_, _ = w.buf.Write(p[:remain])
		return len(p), nil
	}
	return w.buf.Write(p)
}
