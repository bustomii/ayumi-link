// Package browser: satu Chromium headless yang dikendalikan lewat DevTools Protocol, supaya
// tab "Browser" di UI bisa dipakai dari Docker maupun run native. Profil (cookie, localStorage)
// disimpan di data dir sehingga login situs bertahan antar restart.
package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"ayumi-link/internal/store"
)

var (
	ErrUnavailable = errors.New("Chromium/Chrome tidak ditemukan; pasang atau set AYUMI_BROWSER_BIN")
	ErrNotRunning  = errors.New("browser belum dibuka")
)

const (
	defaultW, defaultH = 390, 720
	minDim, maxDim     = 200, 1600
	callTimeout        = 20 * time.Second
	cookieSaveEvery    = 20 * time.Second
)

// Info: keadaan halaman saat ini.
type Info struct {
	Running bool   `json:"running"`
	URL     string `json:"url"`
	Title   string `json:"title"`
}

// Input: satu aksi pengguna.
type Input struct {
	Type   string  `json:"type"` // click | type | key | scroll | back | forward | reload
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Text   string  `json:"text"`
	Key    string  `json:"key"`
	DeltaY float64 `json:"deltaY"`
}

type Manager struct {
	bin     string
	dir     string // <data>/browser (cookies); profile di browser-profile atau browser/profile
	idle    time.Duration
	store   *store.Store
	mu      sync.Mutex
	cmd     *exec.Cmd
	exited  chan struct{}
	port    string
	target  string
	cdp     *cdpClient
	pages   map[string]*pageTab // tabId -> page
	activeTab string
	w, h    int
	last    time.Time
	stopSav chan struct{}
}

// New menyiapkan manager. bin kosong = cari otomatis. idle = lama tanpa dipakai sebelum Chromium dimatikan.
// Profil Chrome khusus Ayumi: <dataDir>/browser-profile (cookie bagus, tidak kunci profil harian).
func New(dataDir, bin string, idle time.Duration) *Manager {
	if bin == "" {
		bin = findBin()
	}
	if idle <= 0 {
		idle = 10 * time.Minute
	}
	m := &Manager{
		bin: bin, dir: filepath.Join(dataDir, "browser"), idle: idle,
		w: defaultW, h: defaultH, pages: map[string]*pageTab{},
	}
	go m.idleLoop()
	return m
}

// SetStore menghubungkan persistensi tab browser per chat.
func (m *Manager) SetStore(st *store.Store) { m.store = st }

// Bin: path Chromium yang dipakai ("" bila tidak ada).
func (m *Manager) Bin() string { return m.bin }

func (m *Manager) Available() bool { return m.bin != "" }

func findBin() string {
	names := []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "chrome"}
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	if runtime.GOOS == "darwin" {
		for _, p := range []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

func inContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return os.Getuid() == 0
}

// ---- siklus hidup ----

// ensure memastikan Chromium + koneksi CDP hidup. Dipanggil dengan m.mu terkunci.
func (m *Manager) ensure(ctx context.Context) error {
	if m.bin == "" {
		return ErrUnavailable
	}
	if m.cdp != nil && m.cdp.alive() && m.cmd != nil {
		select {
		case <-m.exited:
		default:
			return nil
		}
	}
	m.stopLocked() // sisa proses lama

	// Profil khusus Ayumi (bukan Chrome harian) — cookie bertahan, tanpa profile lock.
	profile := filepath.Join(filepath.Dir(m.dir), "browser-profile")
	if _, err := os.Stat(filepath.Join(m.dir, "profile")); err == nil {
		profile = filepath.Join(m.dir, "profile") // migrasi: pakai profil lama bila ada
	}
	if err := os.MkdirAll(profile, 0o700); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(profile, "DevToolsActivePort"))

	// Native default: headed (seperti Claude Desktop / Cursor) — Cloudflare jarang lolos headless.
	// Docker/container tetap headless kecuali AYUMI_BROWSER_HEADLESS=false + display tersedia.
	headless := wantHeadless()
	args := []string{
		"--remote-debugging-port=0",
		"--user-data-dir=" + profile,
		"--no-first-run", "--no-default-browser-check",
		"--disable-blink-features=AutomationControlled",
		"--disable-dev-shm-usage",
		"--mute-audio",
		fmt.Sprintf("--window-size=%d,%d", max(m.w, 1024), max(m.h, 768)),
	}
	if headless {
		args = append(args,
			"--headless=new",
			"--disable-gpu",
			"--hide-scrollbars",
			"--renderer-process-limit=2",
		)
	} else {
		// Jendela di luar layar supaya tidak mengganggu, tapi tetap “browser sungguhan”.
		args = append(args, "--window-position=-2400,-2400")
	}
	if inContainer() || truthy(os.Getenv("AYUMI_BROWSER_NO_SANDBOX")) {
		args = append(args, "--no-sandbox")
	}
	args = append(args, "about:blank")
	log.Printf("browser start headless=%v bin=%s", headless, m.bin)

	cmd := exec.Command(m.bin, args...)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("gagal menjalankan browser: %w", err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	m.cmd, m.exited = cmd, exited

	port, err := waitPort(ctx, filepath.Join(profile, "DevToolsActivePort"), exited)
	if err != nil {
		m.stopLocked()
		return err
	}
	m.port = port

	target, wsPath, err := m.pageTarget(ctx)
	if err != nil {
		m.stopLocked()
		return err
	}
	ws, err := wsDial(net.JoinHostPort("127.0.0.1", port), wsPath)
	if err != nil {
		m.stopLocked()
		return err
	}
	m.cdp, m.target = newCDP(ws), target

	cctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	if _, err := m.cdp.call(cctx, "Page.enable", nil); err != nil {
		m.stopLocked()
		return err
	}
	_, _ = m.cdp.call(cctx, "Network.enable", nil)
	m.hardenPage(cctx, m.cdp)
	m.restoreCookies(cctx)
	m.applyViewport(cctx)

	m.stopSav = make(chan struct{})
	go m.cookieSaver(m.cdp, m.stopSav)
	return nil
}

func wantHeadless() bool {
	if v := strings.TrimSpace(os.Getenv("AYUMI_BROWSER_HEADLESS")); v != "" {
		return truthy(v)
	}
	// Di container biasanya tidak ada display → headless.
	if inContainer() {
		return true
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// hardenPage mengurangi sinyal otomasi (webdriver) agar situs seperti Cloudflare lebih mirip browser biasa.
func (m *Manager) hardenPage(ctx context.Context, cdp *cdpClient) {
	if cdp == nil {
		return
	}
	_, _ = cdp.call(ctx, "Page.addScriptToEvaluateOnNewDocument", map[string]any{
		"source": stealthJS,
	})
	// Juga jalankan di dokumen yang sudah terbuka (about:blank).
	_, _ = cdp.call(ctx, "Runtime.evaluate", map[string]any{
		"expression":    stealthJS,
		"returnByValue": false,
	})
}

// Skrip ringan anti-deteksi bot (bukan bypass captcha). Mirip perilaku browser desktop biasa.
const stealthJS = `(function(){
  try {
    Object.defineProperty(navigator, 'webdriver', { get: function(){ return undefined; }, configurable: true });
  } catch (e) {}
  try {
    if (!window.chrome) { window.chrome = { runtime: {} }; }
  } catch (e) {}
  try {
    const p = navigator.permissions;
    if (p && p.query) {
      const orig = p.query.bind(p);
      p.query = function(params) {
        if (params && params.name === 'notifications') {
          return Promise.resolve({ state: Notification.permission, onchange: null });
        }
        return orig(params);
      };
    }
  } catch (e) {}
  try {
    Object.defineProperty(navigator, 'plugins', {
      get: function(){ return [1, 2, 3, 4, 5]; },
      configurable: true
    });
  } catch (e) {}
  try {
    Object.defineProperty(navigator, 'languages', {
      get: function(){ return ['id-ID', 'id', 'en-US', 'en']; },
      configurable: true
    });
  } catch (e) {}
})();`

func waitPort(ctx context.Context, file string, exited <-chan struct{}) (string, error) {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(file); err == nil {
			if line := strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0]); line != "" {
				if _, err := strconv.Atoi(line); err == nil {
					return line, nil
				}
			}
		}
		select {
		case <-exited:
			return "", errors.New("browser berhenti saat dijalankan (cek apakah Chromium dapat berjalan di sini)")
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	return "", errors.New("browser tidak siap tepat waktu")
}

type listEntry struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title"`
	WSURL string `json:"webSocketDebuggerUrl"`
}

func (m *Manager) list(ctx context.Context) ([]listEntry, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+m.port+"/json/list", nil)
	res, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var out []listEntry
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *Manager) pageTarget(ctx context.Context) (id, wsPath string, err error) {
	for i := 0; i < 20; i++ {
		entries, e := m.list(ctx)
		if e == nil {
			for _, t := range entries {
				if t.Type == "page" && t.WSURL != "" {
					u, perr := url.Parse(t.WSURL)
					if perr != nil {
						continue
					}
					return t.ID, u.Path, nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	return "", "", errors.New("halaman browser tidak ditemukan")
}

func (m *Manager) stopLocked() {
	if m.stopSav != nil {
		close(m.stopSav)
		m.stopSav = nil
	}
	if m.cdp != nil && m.cdp.alive() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		m.saveCookies(ctx)
		cancel()
	}
	for id, p := range m.pages {
		if p.cdp != nil {
			p.cdp.close()
		}
		delete(m.pages, id)
	}
	if m.cdp != nil {
		m.cdp.close()
		m.cdp = nil
	}
	if m.cmd != nil && m.cmd.Process != nil {
		select {
		case <-m.exited:
		default:
			_ = m.cmd.Process.Signal(os.Interrupt)
			select {
			case <-m.exited:
			case <-time.After(3 * time.Second):
				_ = m.cmd.Process.Kill()
				<-m.exited
			}
		}
	}
	m.cmd, m.port, m.target, m.activeTab = nil, "", "", ""
}

// Close mematikan browser (cookie disimpan lebih dulu).
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
}

func (m *Manager) idleLoop() {
	for range time.Tick(30 * time.Second) {
		m.mu.Lock()
		if m.cmd != nil && time.Since(m.last) > m.idle {
			log.Printf("browser dimatikan karena tidak dipakai %s", m.idle)
			m.stopLocked()
		}
		m.mu.Unlock()
	}
}

// ---- cookie: Chromium headless membuang cookie sesi saat berhenti, jadi disalin sendiri ----

func (m *Manager) cookieFile() string { return filepath.Join(m.dir, "cookies.json") }

func (m *Manager) saveCookies(ctx context.Context) {
	cdp := m.activePageLocked()
	if cdp == nil {
		return
	}
	raw, err := cdp.call(ctx, "Network.getAllCookies", nil)
	if err != nil {
		return
	}
	var r struct {
		Cookies []map[string]any `json:"cookies"`
	}
	if json.Unmarshal(raw, &r) != nil || r.Cookies == nil {
		return
	}
	b, err := json.Marshal(r.Cookies)
	if err != nil {
		return
	}
	tmp := m.cookieFile() + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, m.cookieFile())
	}
}

func (m *Manager) restoreCookies(ctx context.Context) {
	b, err := os.ReadFile(m.cookieFile())
	if err != nil {
		return
	}
	var all []map[string]any
	if json.Unmarshal(b, &all) != nil {
		return
	}
	keep := []string{"name", "value", "domain", "path", "httpOnly", "secure", "sameSite"}
	params := make([]map[string]any, 0, len(all))
	for _, c := range all {
		p := map[string]any{}
		for _, k := range keep {
			if v, ok := c[k]; ok {
				p[k] = v
			}
		}
		if session, _ := c["session"].(bool); !session {
			if exp, ok := c["expires"].(float64); ok && exp > 0 {
				p["expires"] = exp
			}
		}
		if p["name"] != nil && p["domain"] != nil {
			params = append(params, p)
		}
	}
	if len(params) > 0 {
		_, _ = m.cdp.call(ctx, "Network.setCookies", map[string]any{"cookies": params})
	}
}

func (m *Manager) cookieSaver(c *cdpClient, stop <-chan struct{}) {
	t := time.NewTicker(cookieSaveEvery)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			m.mu.Lock()
			if m.cdp == c && c.alive() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				m.saveCookies(ctx)
				cancel()
			}
			m.mu.Unlock()
		}
	}
}

// ---- operasi ----

func clampDim(v, def int) int {
	if v <= 0 {
		return def
	}
	if v < minDim {
		return minDim
	}
	if v > maxDim {
		return maxDim
	}
	return v
}

func (m *Manager) applyViewport(ctx context.Context) {
	cdp := m.activePageLocked()
	if cdp == nil {
		return
	}
	_, _ = cdp.call(ctx, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": m.w, "height": m.h, "deviceScaleFactor": 1, "mobile": false,
	})
}

// SetViewport mengubah ukuran layar virtual (dipanggil saat panel berubah ukuran).
func (m *Manager) setSizeLocked(ctx context.Context, w, h int) {
	w, h = clampDim(w, m.w), clampDim(h, m.h)
	if w != m.w || h != m.h {
		m.w, m.h = w, h
		m.applyViewport(ctx)
	}
}

// Open menyalakan browser bila perlu lalu membuka url pada tab aktif (legacy / tanpa chat).
func (m *Manager) Open(ctx context.Context, rawURL string, w, h int) (Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensure(ctx); err != nil {
		return Info{}, err
	}
	m.last = time.Now()
	cctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	m.setSizeLocked(cctx, w, h)
	cdp := m.activePageLocked()
	if cdp == nil {
		return Info{}, ErrNotRunning
	}
	if rawURL != "" {
		res, err := cdp.call(cctx, "Page.navigate", map[string]any{"url": rawURL})
		if err != nil {
			return Info{}, err
		}
		var nav struct {
			ErrorText string `json:"errorText"`
		}
		if json.Unmarshal(res, &nav) == nil && nav.ErrorText != "" {
			return m.infoLocked(ctx), fmt.Errorf("gagal membuka halaman: %s", nav.ErrorText)
		}
	}
	return m.infoLocked(ctx), nil
}

func (m *Manager) infoLocked(ctx context.Context) Info {
	info := Info{Running: m.cmd != nil}
	if m.port == "" {
		return info
	}
	entries, err := m.list(ctx)
	if err != nil {
		return info
	}
	for _, t := range entries {
		if t.ID == m.target {
			info.URL, info.Title = t.URL, t.Title
			if info.URL == "about:blank" {
				info.URL = ""
			}
		}
	}
	return info
}

// Status tidak menyalakan browser.
func (m *Manager) Status(ctx context.Context) Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || m.cdp == nil || !m.cdp.alive() {
		return Info{}
	}
	return m.infoLocked(ctx)
}

// Frame mengambil screenshot JPEG halaman aktif (tanpa menyalakan browser).
func (m *Manager) Frame(ctx context.Context, w, h int) ([]byte, Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cdp := m.activePageLocked()
	if m.cmd == nil || cdp == nil || !cdp.alive() {
		return nil, Info{}, ErrNotRunning
	}
	m.last = time.Now()
	if p := m.pages[m.activeTab]; p != nil {
		p.last = m.last
	}
	cctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	m.setSizeLocked(cctx, w, h)
	res, err := cdp.call(cctx, "Page.captureScreenshot", map[string]any{"format": "jpeg", "quality": 60})
	if err != nil {
		return nil, Info{}, err
	}
	var s struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(res, &s); err != nil {
		return nil, Info{}, err
	}
	img, err := base64.StdEncoding.DecodeString(s.Data)
	if err != nil {
		return nil, Info{}, err
	}
	return img, m.infoLocked(ctx), nil
}

var keyCodes = map[string]struct {
	code string
	vk   int
	text string
}{
	"Enter":      {"Enter", 13, "\r"},
	"Backspace":  {"Backspace", 8, ""},
	"Tab":        {"Tab", 9, ""},
	"Escape":     {"Escape", 27, ""},
	"Delete":     {"Delete", 46, ""},
	"ArrowUp":    {"ArrowUp", 38, ""},
	"ArrowDown":  {"ArrowDown", 40, ""},
	"ArrowLeft":  {"ArrowLeft", 37, ""},
	"ArrowRight": {"ArrowRight", 39, ""},
	"Home":       {"Home", 36, ""},
	"End":        {"End", 35, ""},
	"PageUp":     {"PageUp", 33, ""},
	"PageDown":   {"PageDown", 34, ""},
}

// Do menjalankan satu aksi pengguna pada halaman aktif.
func (m *Manager) Do(ctx context.Context, in Input) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cdp := m.activePageLocked()
	if m.cmd == nil || cdp == nil || !cdp.alive() {
		return ErrNotRunning
	}
	m.last = time.Now()
	if p := m.pages[m.activeTab]; p != nil {
		p.last = m.last
	}
	cctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	call := func(method string, p any) error {
		_, err := cdp.call(cctx, method, p)
		return err
	}
	switch in.Type {
	case "click":
		x, y := in.X, in.Y
		if err := call("Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": x, "y": y}); err != nil {
			return err
		}
		if err := call("Input.dispatchMouseEvent", map[string]any{"type": "mousePressed", "x": x, "y": y, "button": "left", "clickCount": 1}); err != nil {
			return err
		}
		return call("Input.dispatchMouseEvent", map[string]any{"type": "mouseReleased", "x": x, "y": y, "button": "left", "clickCount": 1})
	case "type":
		if in.Text == "" || len(in.Text) > 4096 {
			return errors.New("teks kosong atau terlalu panjang")
		}
		return call("Input.insertText", map[string]any{"text": in.Text})
	case "key":
		k, ok := keyCodes[in.Key]
		if !ok {
			return errors.New("tombol tidak didukung")
		}
		down := map[string]any{"type": "rawKeyDown", "key": in.Key, "code": k.code, "windowsVirtualKeyCode": k.vk}
		if k.text != "" {
			down["type"], down["text"] = "keyDown", k.text
		}
		if err := call("Input.dispatchKeyEvent", down); err != nil {
			return err
		}
		return call("Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": in.Key, "code": k.code, "windowsVirtualKeyCode": k.vk})
	case "scroll":
		return call("Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": in.X, "y": in.Y, "deltaX": 0, "deltaY": in.DeltaY})
	case "back":
		return call("Runtime.evaluate", map[string]any{"expression": "history.back()"})
	case "forward":
		return call("Runtime.evaluate", map[string]any{"expression": "history.forward()"})
	case "reload":
		return call("Page.reload", nil)
	}
	return errors.New("aksi tidak dikenal")
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
