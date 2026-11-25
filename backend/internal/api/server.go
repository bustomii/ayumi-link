// Package api: HTTP API (Go Fiber) Ayumi Link.
package api

import (
	"errors"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"ayumi-link/internal/auth"
	"ayumi-link/internal/browser"
	"ayumi-link/internal/claude"
	"ayumi-link/internal/config"
	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
	"ayumi-link/internal/termpty"
	"ayumi-link/internal/uploads"

	"github.com/gofiber/contrib/websocket"
)

const cookieName = "ayumi_session"

// Deps adalah semua layanan yang dibutuhkan handler.
type Deps struct {
	Config  *config.Config
	Auth    *auth.Manager
	Svc     *claude.Service
	Runner  *claude.Runner
	Browser *fsbrowse.Browser
	Store   *store.Store
	Shell   *claude.Shell    // shell command-box legacy (opsional)
	Term    *termpty.Manager // terminal PTY penuh multi-tab
	Web     *browser.Manager // Chrome host CDP + multi-tab Browser (nil = dimatikan)
	CliAuth *claude.CliAuth  // login Claude CLI dari settings (nil = tidak tersedia)
	Uploads *uploads.Manager // lampiran chat (nil = fitur dimatikan)
}

type Server struct {
	Deps

	versionOnce   sync.Once
	claudeVersion string

	diskSyncMu sync.Mutex
	diskSyncAt time.Time
}

// New membangun aplikasi Fiber lengkap dengan middleware dan rute.
func New(d Deps) *fiber.App {
	s := &Server{Deps: d}

	cfg := fiber.Config{
		AppName:               "ayumi-link",
		DisableStartupMessage: true,
		BodyLimit:             bodyLimit(d),
		IdleTimeout:           2 * time.Minute,
		ErrorHandler:          errorHandler,
	}
	if d.Config.TrustProxy {
		cfg.ProxyHeader = d.Config.ProxyHeader
		if cfg.ProxyHeader == "" {
			cfg.ProxyHeader = fiber.HeaderXForwardedFor
		}
		if len(d.Config.TrustedProxies) > 0 {
			cfg.EnableTrustedProxyCheck = true
			cfg.TrustedProxies = d.Config.TrustedProxies
		}
	}
	app := fiber.New(cfg)

	app.Use(recover.New())
	app.Use(s.ipGuard)
	app.Use(s.hostGuard)
	app.Use(logger.New(logger.Config{
		Format:     "${time} ${status} ${method} ${path} ${latency}\n",
		TimeFormat: "15:04:05",
	}))
	app.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy: csp,
		XFrameOptions:         "DENY",
	}))
	app.Use(noStore)
	app.Use(s.originGuard)
	app.Use(requireJSON)

	api := app.Group("/api")
	api.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) })

	// ---- auth (publik, dibatasi per IP) ----
	maxAuth := d.Config.AuthRateLimit
	if maxAuth <= 0 {
		maxAuth = 10
	}
	authLimit := limiter.New(limiter.Config{
		Max:        maxAuth,
		Expiration: time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			c.Set(fiber.HeaderRetryAfter, "60")
			return fiber.NewError(fiber.StatusTooManyRequests, "terlalu banyak permintaan, coba lagi sebentar")
		},
	})
	a := api.Group("/auth")
	a.Get("/status", s.authStatus)
	a.Post("/setup", authLimit, s.authSetup)
	a.Post("/setup/confirm", authLimit, s.authSetupConfirm)
	a.Post("/login", authLimit, s.authLogin)
	a.Post("/login/verify", authLimit, s.authLoginVerify)
	a.Post("/logout", s.authLogout)

	// ---- terproteksi ----
	p := api.Group("", s.requireAuth)
	p.Get("/sessions", s.listSessions)
	p.Get("/sessions/:id", s.getSession)
	p.Get("/sessions/:id/changes", s.listSessionChanges)
	p.Get("/sessions/:id/changes/diff", s.getSessionChangeDiff)
	p.Get("/sessions/:id/git", s.listSessionRepos)
	p.Get("/sessions/:id/git/diff", s.getSessionRepoDiff)
	p.Post("/sessions/:id/git/repos", s.addSessionRepo)
	p.Delete("/sessions/:id/git/repos", s.removeSessionRepo)
	p.Get("/terminal", s.enterTerminal)
	p.Post("/terminal", s.runTerminal)
	p.Get("/sessions/:id/tools/tabs", s.listToolsTabs)
	p.Post("/sessions/:id/terminal/tabs", s.createTerminalTab)
	p.Delete("/sessions/:id/terminal/tabs/:tabId", s.deleteTerminalTab)
	p.Post("/sessions/:id/browser/tabs", s.createBrowserTab)
	p.Delete("/sessions/:id/browser/tabs/:tabId", s.deleteBrowserTab)
	p.Post("/sessions/:id/browser/tabs/:tabId/activate", s.activateBrowserTab)
	p.Get("/browser", s.webStatus)
	p.Post("/browser/open", s.webOpen)
	p.Get("/browser/frame", s.webFrame)
	p.Post("/browser/input", s.webInput)
	p.Post("/browser/close", s.webClose)
	p.Put("/sessions/:id/group", s.moveSession)
	p.Post("/sessions/:id/fork", s.forkSession)
	p.Put("/sessions/:id/title", s.renameSession)
	p.Put("/groups/rename", s.renameGroup)
	p.Delete("/sessions/:id", s.deleteSession)
	p.Post("/chat", s.chat)
	p.Get("/agents", s.listAgents)
	p.Post("/agents", s.createAgent)
	p.Get("/agents/:id", s.getAgent)
	p.Put("/agents/:id", s.updateAgent)
	p.Delete("/agents/:id", s.deleteAgent)
	p.Get("/skills", s.listSkills)
	p.Post("/skills", s.createSkill)
	p.Get("/skills/:id", s.getSkill)
	p.Put("/skills/:id", s.updateSkill)
	p.Delete("/skills/:id", s.deleteSkill)
	p.Post("/agents-skills/generate-prompt", s.generateAgentsSkillsPrompt)
	p.Get("/fs", s.browse)
	if d.Uploads != nil {
		p.Get("/uploads/limits", s.uploadLimits)
		p.Post("/uploads", s.uploadFile)
		p.Get("/uploads/:id/file", s.uploadDownload)
		p.Delete("/uploads/:id", s.uploadDelete)
	}

	// ---- profil & konfigurasi ----
	p.Get("/profile", s.getProfile)
	p.Get("/profile/prefs", s.getPrefs)
	p.Put("/profile/prefs", s.putPrefs)
	p.Get("/profile/branding", s.getBranding)
	p.Put("/profile/branding", s.putBranding)
	p.Delete("/profile/branding", s.resetBranding)
	p.Post("/profile/password", authLimit, s.changePassword)
	p.Post("/profile/recovery", authLimit, s.regenerateRecovery)
	p.Post("/profile/2fa/begin", authLimit, s.twoFactorBegin)
	p.Post("/profile/2fa/confirm", authLimit, s.twoFactorConfirm)
	p.Post("/profile/sessions/revoke-others", s.revokeOthers)
	p.Delete("/profile/sessions/:id", s.revokeSession)
	p.Get("/claude/auth", s.getClaudeAuth)
	p.Put("/claude/auth", authLimit, s.putClaudeAuth)
	p.Delete("/claude/auth", authLimit, s.deleteClaudeAuth)

	// WebSocket terminal (auth cookie dicek di middleware grup p).
	p.Get("/terminal/ws", websocket.New(s.handleTerminalWS, websocket.Config{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}))

	api.Use(func(c *fiber.Ctx) error { return fiber.NewError(fiber.StatusNotFound, "tidak ditemukan") })

	if dir := d.Config.StaticDir; dir != "" {
		if !mountStatic(app, dir, s) {
			log.Printf("PERINGATAN: AYUMI_STATIC_DIR=%s tidak berisi index.html; frontend tidak disajikan", dir)
		}
	} else {
		// Tanpa static dir (tes / API-only): tetap sajikan branding publik.
		s.mountBrandingAssets(app, "")
	}
	return app
}

// ---- middleware ----

func errorHandler(c *fiber.Ctx, err error) error {
	status, msg := fiber.StatusInternalServerError, "terjadi kesalahan di server"
	var fe *fiber.Error
	if errors.As(err, &fe) {
		status, msg = fe.Code, fe.Message
	} else {
		log.Printf("error: %s %s: %v", c.Method(), c.Path(), err)
	}
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// noStore: respons API tidak boleh di-cache (berisi data sesi). File statis diatur sendiri.
func noStore(c *fiber.Ctx) error {
	if strings.HasPrefix(c.Path(), "/api") {
		c.Set(fiber.HeaderCacheControl, "no-store")
	}
	return c.Next()
}

// requireJSON: POST/PUT wajib application/json (memaksa preflight CORS bagi situs lain).
func requireJSON(c *fiber.Ctx) error {
	m := c.Method()
	if m != fiber.MethodPost && m != fiber.MethodPut {
		return c.Next()
	}
	ct := c.Get(fiber.HeaderContentType)
	if c.Path() == uploadsPath && m == fiber.MethodPost {
		// Satu-satunya rute yang menerima multipart (unggahan lampiran).
		if !strings.HasPrefix(ct, fiber.MIMEMultipartForm) {
			return fiber.NewError(fiber.StatusUnsupportedMediaType, "unggahan harus multipart/form-data")
		}
		return c.Next()
	}
	if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
		return fiber.NewError(fiber.StatusUnsupportedMediaType, "content-type harus application/json")
	}
	if len(c.Body()) > maxJSONBody {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "permintaan terlalu besar")
	}
	return c.Next()
}

// bodyLimit: batas badan global mengikuti ukuran unggahan terbesar; rute JSON dibatasi lagi
// oleh requireJSON.
func bodyLimit(d Deps) int {
	if d.Uploads == nil {
		return maxJSONBody
	}
	return int(d.Uploads.MaxBytes()) + (1 << 20) // ruang untuk overhead multipart
}

// originGuard menolak request lintas-origin: Origin harus sama dengan Host,
// loopback, atau terdaftar di ALLOWED_ORIGINS. Request tanpa Origin (curl, proxy) lolos.
func (s *Server) originGuard(c *fiber.Ctx) error {
	origin := c.Get(fiber.HeaderOrigin)
	if origin == "" {
		return c.Next()
	}
	for _, o := range s.Config.AllowedOrigins {
		if o == origin {
			return c.Next()
		}
	}
	if u, err := url.Parse(origin); err == nil {
		if strings.EqualFold(u.Host, s.authority(c)) {
			return c.Next()
		}
		switch u.Hostname() {
		case "localhost", "127.0.0.1", "::1":
			return c.Next()
		}
	}
	return fiber.NewError(fiber.StatusForbidden, "origin tidak diizinkan")
}

func (s *Server) requireAuth(c *fiber.Ctx) error {
	st := s.Auth.Status(c.Cookies(cookieName))
	switch st.State {
	case "authenticated":
		c.Locals("user", st.Username)
		return c.Next()
	case "setup_required":
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "setup diperlukan", "code": "setup_required"})
	default:
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "belum login", "code": "unauthenticated"})
	}
}

// ipGuard menolak klien yang alamatnya tidak ada di AYUMI_ALLOWED_IPS (bila diisi).
// Berlaku untuk semua rute, termasuk halaman login, sebelum pemrosesan lain.
func (s *Server) ipGuard(c *fiber.Ctx) error {
	if !s.Config.AllowedIPs.Enabled() || s.Config.AllowedIPs.Allows(c.IP()) {
		return c.Next()
	}
	log.Printf("akses ditolak: IP %s tidak ada di allowlist", c.IP())
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"error": "akses dari alamat ini tidak diizinkan",
		"code":  "ip_blocked",
	})
}

// authority: host[:port] permintaan seperti dikirim klien. Header Host dibaca mentah dari fasthttp
// karena c.Get("Host") di Fiber selalu kosong. X-Forwarded-Host hanya dihormati bila AYUMI_TRUST_PROXY
// aktif dan pengirimnya proxy tepercaya (Hostname() milik Fiber memercayai header itu dari siapa pun,
// sehingga tidak dipakai).
func (s *Server) authority(c *fiber.Ctx) string {
	host := string(c.Context().Request.Host())
	if s.Config.TrustProxy && c.IsProxyTrusted() {
		if xf := strings.TrimSpace(strings.Split(c.Get(fiber.HeaderXForwardedHost), ",")[0]); xf != "" {
			host = xf
		}
	}
	return host
}

// requestHost: nama host permintaan tanpa port.
func (s *Server) requestHost(c *fiber.Ctx) string {
	host := s.authority(c)
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h // "host:port" atau "[::1]:port"
	}
	return host
}

// hostGuard menolak permintaan yang nama host-nya tidak ada di AYUMI_ALLOWED_HOSTS (bila diisi).
func (s *Server) hostGuard(c *fiber.Ctx) error {
	if !s.Config.AllowedHosts.Enabled() {
		return c.Next()
	}
	host := s.requestHost(c)
	if s.Config.AllowedHosts.Allows(host) {
		return c.Next()
	}
	log.Printf("akses ditolak: host %q tidak ada di AYUMI_ALLOWED_HOSTS", host)
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"error": "alamat situs ini tidak diizinkan",
		"code":  "host_blocked",
	})
}
