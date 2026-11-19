// Ayumi Link: jembatan web ke session Claude Code lokal.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ayumi-link/internal/api"
	"ayumi-link/internal/auth"
	webbrowser "ayumi-link/internal/browser"
	"ayumi-link/internal/claude"
	"ayumi-link/internal/config"
	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
	"ayumi-link/internal/termpty"
	"ayumi-link/internal/uploads"
)

func main() {
	envFile, err := config.LoadDotEnv()
	if err != nil {
		log.Fatalf("gagal membaca file env: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("konfigurasi tidak valid: %v", err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "reset-auth":
			resetAuth(cfg)
			return
		case "healthcheck":
			os.Exit(healthcheck(cfg))
		case "version":
			fmt.Println("ayumi-link", config.Version)
			return
		default:
			fmt.Fprintf(os.Stderr, "perintah tidak dikenal: %s\n\nPerintah:\n  reset-auth    hapus akun + 2FA (untuk pengaturan ulang)\n  healthcheck   cek apakah server berjalan (untuk Docker)\n  version       tampilkan versi\n", os.Args[1])
			os.Exit(2)
		}
	}

	st := store.New(cfg.DataDir)
	authMgr, err := auth.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("gagal membuka data auth: %v", err)
	}
	browser := fsbrowse.New(cfg.Roots)
	svc := &claude.Service{Home: cfg.ClaudeHome, DesktopDir: cfg.DesktopDir, Store: st}
	svc.CwdOK = func(p string) bool { _, ok := browser.ResolveDir(p); return ok }
	bin := claude.ResolveBin(cfg.ClaudeBin, cfg.DesktopDir)
	up := uploads.New(filepath.Join(cfg.DataDir, "uploads"), uploads.Options{
		MaxBytes: int64(cfg.MaxUploadMB) << 20,
		MaxTotal: int64(cfg.UploadMaxTotalMB) << 20,
		TTL:      time.Duration(cfg.UploadTTLDays) * 24 * time.Hour,
	})
	go func() { // lampiran lama dibersihkan saat start lalu tiap 6 jam
		for {
			if n := up.Cleanup(); n > 0 {
				log.Printf("lampiran kedaluwarsa dihapus: %d", n)
			}
			time.Sleep(6 * time.Hour)
		}
	}()
	cliAuth, err := claude.OpenCliAuth(cfg.DataDir, bin, cfg.ClaudeHome)
	if err != nil {
		log.Fatalf("gagal membuka auth Claude CLI: %v", err)
	}
	runner := &claude.Runner{Svc: svc, Browser: browser, Store: st, Bin: bin, DefaultPerm: cfg.PermissionMode, Uploads: up, Auth: cliAuth}
	shell := claude.NewShell(browser, svc, cliAuth)
	idle := time.Duration(envMinutes("AYUMI_BROWSER_IDLE_MIN", 10)) * time.Minute
	term := termpty.New(browser, st, func(id string) (string, error) {
		return svc.SessionCwd(id)
	}, idle)
	var web *webbrowser.Manager
	if !truthy(os.Getenv("AYUMI_BROWSER_DISABLED")) {
		web = webbrowser.New(cfg.DataDir, os.Getenv("AYUMI_BROWSER_BIN"), idle)
		web.SetStore(st)
		if web.Available() {
			log.Printf("browser: %s (profil khusus Ayumi)", web.Bin())
		} else {
			log.Printf("browser: Chrome/Chromium tidak ditemukan; tab Browser dimatikan")
		}
	}

	app := api.New(api.Deps{Config: cfg, Auth: authMgr, Svc: svc, Runner: runner, Browser: browser, Store: st, Shell: shell, Term: term, Web: web, CliAuth: cliAuth, Uploads: up})
	defer term.CloseAll()
	if web != nil {
		defer web.Close()
	}

	log.Printf("ayumi-link %s listening on %s (claude=%s)", config.Version, cfg.Addr(), bin)
	if envFile != "" {
		log.Printf("konfigurasi dibaca dari %s", envFile)
	}
	if cfg.AllowedIPs.Enabled() {
		log.Printf("allowlist IP aktif: %s", strings.Join(cfg.AllowedIPs.Strings(), ", "))
	}
	if cfg.AllowedHosts.Enabled() {
		log.Printf("allowlist host aktif: %s", strings.Join(cfg.AllowedHosts.Strings(), ", "))
	}
	if cfg.TrustProxy && len(cfg.TrustedProxies) == 0 {
		log.Printf("PERINGATAN: AYUMI_TRUST_PROXY aktif tanpa AYUMI_TRUSTED_PROXIES; header %s dipercaya dari siapa pun yang bisa menjangkau server", cfg.ProxyHeader)
	}
	if tok := authMgr.SetupToken(); tok != "" {
		log.Printf("=====================================================================")
		log.Printf(" SETUP DIPERLUKAN: buka UI lalu masukkan token setup berikut")
		log.Printf("   %s", tok)
		log.Printf("=====================================================================")
	}

	go func() {
		if err := app.Listen(cfg.Addr()); err != nil {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)
	if web != nil {
		web.Close()
	}
}

// healthcheck memeriksa server lokal; keluar 0 bila hidup. Jawaban 403 ip_blocked/host_blocked juga
// berarti server hidup (allowlist menolak permintaan loopback), jadi dianggap sehat.
func healthcheck(cfg *config.Config) int {
	host := cfg.Host
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get("http://" + net.JoinHostPort(host, cfg.Port) + "/api/health")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck gagal:", err)
		return 1
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<10))
	if res.StatusCode == http.StatusOK || (res.StatusCode == http.StatusForbidden && (strings.Contains(string(body), "ip_blocked") || strings.Contains(string(body), "host_blocked"))) {
		return 0
	}
	fmt.Fprintln(os.Stderr, "healthcheck gagal: status", res.StatusCode)
	return 1
}

func resetAuth(cfg *config.Config) {
	fmt.Printf("Ini menghapus akun, 2FA, dan semua sesi login Ayumi Link di:\n  %s\n", cfg.DataDir)
	fmt.Print("Ketik YA untuk melanjutkan: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.TrimSpace(line) != "YA" {
		fmt.Println("Dibatalkan.")
		return
	}
	if err := auth.Reset(cfg.DataDir); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Selesai. Jalankan ulang server untuk membuat akun baru.")
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func envMinutes(key string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key))); err == nil && n > 0 {
		return n
	}
	return def
}
