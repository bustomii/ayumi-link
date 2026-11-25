package claude

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
	"ayumi-link/internal/uploads"
)

// RequestError adalah kesalahan input dari klien (dipetakan ke HTTP status oleh lapisan API).
type RequestError struct {
	Status  int
	Message string
}

func (e *RequestError) Error() string { return e.Message }

func badRequest(msg string) *RequestError { return &RequestError{Status: 400, Message: msg} }

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID memeriksa format id session.
func IsUUID(s string) bool { return uuidRe.MatchString(s) }

// Setiap chat dari Ayumi Link berjalan "bersih": tanpa atribusi AI di commit, PR, kode, atau komentar.
// Berlaku hanya untuk yang ditulis pada giliran itu; kode lama tidak disentuh.
const cleanOutputPrompt = `Aturan penulisan untuk sesi ini:
- Jangan menambahkan atribusi atau watermark AI pada hal yang kamu tulis sekarang: tidak ada "Co-Authored-By" untuk Claude/Anthropic, "Generated with Claude Code", noreply@anthropic.com, atau komentar/header "dibuat oleh AI" di kode, dokumen, pesan commit, maupun deskripsi PR.
- Commit memakai identitas git milik pengguna saja. Tulis pesan commit seperti commit manual biasa.
- Komentar kode hanya menjelaskan alasan, bukan siapa penulisnya. Ikuti gaya kode yang sudah ada.
- Aturan ini hanya untuk kode dan teks baru yang kamu tulis. Jangan mengubah atau membersihkan kode lama demi aturan ini.
- Jangan menulis ulang riwayat git atau force push kecuali diminta jelas.`

// cleanOutputSettings menonaktifkan atribusi bawaan Claude Code pada commit dan PR.
const cleanOutputSettings = `{"attribution":{"commit":"","pr":""},"includeCoAuthoredBy":false}`

// ---- opsi yang diizinkan ----

// curatedModels = daftar UI / Max aliases. ValidModel juga menerima ID API lain
// (mis. claude-sonnet-4-20250514) agar bisa dipakai dengan ANTHROPIC_API_KEY.
var curatedModels = map[string]bool{
	"claude-fable-5-2": true, "claude-opus-5-5": true, "claude-sonnet-5-5": true, "claude-haiku-5-5": true,
	// model lainnya
	"claude-fable-5-1": true, "claude-fable-5": true, "claude-opus-5": true, "claude-opus-4-8": true,
	"claude-opus-4-7": true, "claude-opus-4-6": true, "claude-sonnet-5": true, "claude-sonnet-4-6": true,
	"claude-haiku-4-5": true,
}

// modelIDRe: alias Claude Code atau ID API Anthropic (huruf kecil, angka, . _ -).
var modelIDRe = regexp.MustCompile(`^claude-[a-z0-9][a-z0-9._-]{0,63}$`)

var allowedEfforts = map[string]bool{"low": true, "medium": true, "high": true, "xhigh": true, "max": true}

// bypassPermissions sengaja tidak boleh dipilih dari UI.
var allowedPermissionModes = map[string]bool{"manual": true, "acceptEdits": true, "auto": true, "plan": true}

// ValidModel, ValidEffort, ValidPermissionMode dipakai API untuk memvalidasi preferensi.
func ValidModel(v string) bool {
	if curatedModels[v] {
		return true
	}
	return modelIDRe.MatchString(v)
}
func ValidEffort(v string) bool         { return allowedEfforts[v] }
func ValidPermissionMode(v string) bool { return allowedPermissionModes[v] }

// effortFor menyesuaikan effort dengan kemampuan model: Haiku 4.5 tidak mendukung
// effort, Opus/Sonnet 4.6 belum punya xhigh.
func effortFor(model, effort string) string {
	if !allowedEfforts[effort] {
		return ""
	}
	switch model {
	case "claude-haiku-4-5":
		return ""
	case "claude-opus-4-6", "claude-sonnet-4-6":
		if effort == "xhigh" {
			return "high"
		}
	}
	return effort
}

// ---- memilih CLI claude ----

func versionParts(v string) []int {
	var out []int
	for _, p := range strings.Split(v, ".") {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

func versionLess(a, b string) bool {
	x, y := versionParts(a), versionParts(b)
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return len(x) < len(y)
}

// ResolveBin memilih CLI claude: override eksplisit, lalu (hanya di macOS) versi terbaru
// yang dibundel Claude Desktop, lalu "claude" di PATH.
//
// Di Linux/Docker jangan pernah memakai binary .app macOS dari folder Desktop yang
// di-mount — itu menyebabkan "exec format error".
func ResolveBin(override, desktopDir string) string {
	if override != "" {
		return override
	}
	bin := "claude"
	if runtime.GOOS != "darwin" {
		return bin
	}
	root := filepath.Join(desktopDir, "claude-code")
	entries, err := os.ReadDir(root)
	if err != nil {
		return bin
	}
	best := ""
	for _, e := range entries {
		if !e.IsDir() || (best != "" && !versionLess(best, e.Name())) {
			continue
		}
		m, _ := filepath.Glob(filepath.Join(root, e.Name(), "*", "claude.app", "Contents", "MacOS", "claude"))
		if len(m) > 0 {
			best, bin = e.Name(), m[0]
		}
	}
	return bin
}

// ---- runner ----

type Runner struct {
	Svc         *Service
	Browser     *fsbrowse.Browser
	Store       *store.Store
	Bin         string
	DefaultPerm string
	Uploads     *uploads.Manager // lampiran chat (boleh nil)
	Auth        *CliAuth         // kredensial CLI dari settings (boleh nil)
}

type ChatOptions struct {
	SessionID      string
	Cwd            string
	Message        string
	PermissionMode string
	Model          string
	Effort         string
	Attachments    []string // id lampiran yang sudah diunggah
}

// Event dikirim ke klien lewat SSE. Kind: session | delta | tool | done | error.
type Event struct {
	Kind string
	Data map[string]string
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (r *Runner) permission(requested string) string {
	if allowedPermissionModes[requested] {
		return requested
	}
	if r.DefaultPerm != "" {
		return r.DefaultPerm
	}
	return "auto"
}

// resolveNewCwd memastikan folder session baru ada di dalam AYUMI_ROOTS.
func (r *Runner) resolveNewCwd(cwd string) (string, error) {
	if cwd == "" && len(r.Browser.Roots()) > 0 {
		cwd = r.Browser.Roots()[0]
	}
	real, ok := r.Browser.ResolveDir(cwd)
	if !ok {
		return "", badRequest("folder tidak diizinkan atau tidak ada")
	}
	return real, nil
}

// resolveResumeCwd mengambil cwd dari metadata server (list session / registry),
// mengabaikan input klien, lalu memaksa path tetap di dalam AYUMI_ROOTS.
func (r *Runner) resolveResumeCwd(sessionID string) (string, error) {
	cwd := ""
	if r.Svc != nil {
		for _, s := range r.Svc.List() {
			if s.ID == sessionID {
				cwd = s.Cwd
				break
			}
		}
	}
	if cwd == "" && r.Store != nil {
		cwd = r.Store.Registry()[sessionID]
	}
	real, ok := r.Browser.ResolveDir(cwd)
	if !ok {
		return "", badRequest("folder session tidak diizinkan atau tidak ada")
	}
	return real, nil
}

// Start memvalidasi permintaan lalu menjalankan claude di background.
// Event dibaca dari channel yang ditutup saat proses selesai; membatalkan ctx menghentikan proses.
func (r *Runner) Start(ctx context.Context, o ChatOptions) (<-chan Event, error) {
	o.Message = strings.TrimSpace(o.Message)
	atts, err := r.resolveAttachments(o.Attachments)
	if err != nil {
		return nil, err
	}
	if o.Message == "" {
		if len(atts) == 0 {
			return nil, badRequest("message kosong")
		}
		o.Message = defaultAttachPrompt
	}
	prompt := o.Message + attachmentFooter(atts, r.Uploads)

	args := []string{"-p", "--output-format", "stream-json", "--verbose", "--include-partial-messages",
		"--permission-mode", r.permission(o.PermissionMode),
		"--append-system-prompt", cleanOutputPrompt, "--settings", cleanOutputSettings}
	if ValidModel(o.Model) {
		args = append(args, "--model", o.Model)
	}
	if eff := effortFor(o.Model, o.Effort); eff != "" {
		args = append(args, "--effort", eff)
	}
	if len(atts) > 0 {
		// Izinkan Claude membaca folder lampiran (di luar folder kerja) dan hanya itu.
		args = append(args, "--add-dir")
		for _, in := range atts {
			args = append(args, r.Uploads.Dir(in))
		}
	}

	sessionID := o.SessionID
	var cwd string
	if sessionID == "" {
		// Session baru: folder harus berada di dalam root yang diizinkan.
		real, err := r.resolveNewCwd(o.Cwd)
		if err != nil {
			return nil, err
		}
		cwd = real
		sessionID = newUUID()
		args = append(args, "--session-id", sessionID)
	} else {
		if !IsUUID(sessionID) {
			return nil, badRequest("sessionId tidak valid")
		}
		if FindJSONL(r.Svc.Home, sessionID) == "" {
			return nil, &RequestError{Status: 404, Message: "session tidak ditemukan"}
		}
		args = append(args, "--resume", sessionID)
		// Resume: abaikan cwd dari klien; hanya metadata server + allowlist roots.
		real, err := r.resolveResumeCwd(sessionID)
		if err != nil {
			return nil, err
		}
		cwd = real
	}

	// Prompt lewat stdin supaya aman dari parsing flag (mis. diawali "-").
	cmd := exec.CommandContext(ctx, r.Bin, args...)
	cmd.Dir = cwd
	cmd.Env = r.Auth.ApplyEnv(os.Environ())
	cmd.Stdin = strings.NewReader(prompt)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, &RequestError{Status: 500, Message: "gagal menjalankan claude: " + err.Error()}
	}

	r.Store.Register(sessionID, cwd)

	events := make(chan Event, 64)
	go func() {
		defer close(events)
		send := func(kind string, kv ...string) bool {
			d := map[string]string{}
			for i := 0; i+1 < len(kv); i += 2 {
				d[kv[i]] = kv[i+1]
			}
			select {
			case events <- Event{Kind: kind, Data: d}:
				return true
			case <-ctx.Done():
				return false
			}
		}
		send("session", "id", sessionID, "cwd", cwd)

		gotResult, streamed := false, false
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
		for sc.Scan() {
			ev := parseStreamLine(sc.Bytes())
			switch ev.kind {
			case "delta":
				streamed = true
				send("delta", "text", ev.text)
			case "tool":
				send("tool", "name", ev.text)
			case "assistant":
				// fallback bila partial streaming tidak aktif (mis. pesan error sintetis)
				if !streamed && ev.text != "" {
					send("delta", "text", ev.text)
				}
			case "result":
				gotResult = true
				if ev.isErr {
					send("error", "message", ev.text)
				} else {
					send("done", "id", sessionID)
				}
			}
		}
		werr := cmd.Wait()
		if !gotResult {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" && werr != nil {
				msg = werr.Error()
			}
			if msg == "" {
				msg = "claude berhenti tanpa hasil"
			}
			send("error", "message", truncate(msg, 500))
		}
	}()
	return events, nil
}
