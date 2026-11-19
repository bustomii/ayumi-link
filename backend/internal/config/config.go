// Package config membaca konfigurasi dari environment.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Version adalah versi aplikasi yang ditampilkan di halaman profil.
const Version = "0.3.0"

type Config struct {
	Host string
	Port string

	ClaudeHome string // ~/.claude
	DesktopDir string // folder data Claude Desktop (group sidebar)
	DataDir    string // data Ayumi Link (auth, registry, state)

	Roots          []string // folder yang boleh dipilih untuk session baru
	AllowedOrigins []string // origin tambahan selain same-origin dan loopback
	ClaudeBin      string   // override path CLI claude
	PermissionMode string   // mode izin default bila klien tidak memilih

	AuthRateLimit int // maks request/menit/IP ke endpoint auth

	CookieSecure bool // paksa atribut Secure pada cookie (pakai di balik HTTPS)
	TrustProxy   bool // percaya header IP/protokol klien dari reverse proxy/tunnel

	// AllowedIPs membatasi alamat klien yang boleh mengakses apa pun (kosong = semua).
	AllowedIPs *IPList
	// AllowedHosts membatasi nama host (domain) yang boleh dipakai (kosong = semua).
	// Melindungi dari DNS rebinding dan akses lewat domain yang tidak dikenal.
	AllowedHosts *HostList
	// ProxyHeader: header berisi IP klien sungguhan saat TrustProxy aktif
	// (X-Forwarded-For, X-Real-IP, atau CF-Connecting-IP untuk Cloudflare Tunnel).
	ProxyHeader string
	// TrustedProxies: hanya proxy dengan alamat ini yang header IP-nya dipercaya.
	TrustedProxies []string

	// Lampiran chat (gambar, PDF, Excel, Word).
	MaxUploadMB      int // ukuran maksimal per file
	UploadMaxTotalMB int // total semua lampiran tersimpan
	UploadTTLDays    int // lampiran lebih tua dari ini dihapus otomatis

	// StaticDir: folder hasil build frontend yang disajikan backend (kosong = tidak disajikan).
	StaticDir string
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = filepath.Join(home, ".config")
	}

	c := &Config{
		// Loopback saja secara bawaan; set HOST=0.0.0.0 hanya di balik HTTPS + login.
		Host:           env("HOST", "127.0.0.1"),
		Port:           strings.TrimPrefix(env("PORT", "8081"), ":"),
		ClaudeHome:     env("CLAUDE_HOME", filepath.Join(home, ".claude")),
		DesktopDir:     env("CLAUDE_DESKTOP_DIR", filepath.Join(home, "Library", "Application Support", "Claude")),
		DataDir:        env("AYUMI_DATA_DIR", filepath.Join(cfgDir, "ayumi-link")),
		ClaudeBin:      os.Getenv("CLAUDE_BIN"),
		PermissionMode: env("AYUMI_PERMISSION_MODE", "auto"),
		AuthRateLimit:  envInt("AYUMI_AUTH_RATE_LIMIT", 10),
		CookieSecure:   truthy(os.Getenv("AYUMI_COOKIE_SECURE")),
		TrustProxy:     truthy(os.Getenv("AYUMI_TRUST_PROXY")),
		ProxyHeader:    env("AYUMI_PROXY_HEADER", "X-Forwarded-For"),
		StaticDir:      os.Getenv("AYUMI_STATIC_DIR"),

		MaxUploadMB:      envInt("AYUMI_MAX_UPLOAD_MB", 20),
		UploadMaxTotalMB: envInt("AYUMI_UPLOAD_MAX_TOTAL_MB", 2048),
		UploadTTLDays:    envInt("AYUMI_UPLOAD_TTL_DAYS", 30),
	}
	if v := os.Getenv("AYUMI_ROOTS"); v != "" {
		c.Roots = splitList(v, string(os.PathListSeparator))
	} else {
		c.Roots = []string{home}
	}
	c.AllowedOrigins = splitList(os.Getenv("ALLOWED_ORIGINS"), ",")
	c.TrustedProxies = splitList(os.Getenv("AYUMI_TRUSTED_PROXIES"), ",")

	if c.AllowedIPs, err = ParseIPList(splitList(os.Getenv("AYUMI_ALLOWED_IPS"), ",")); err != nil {
		return nil, fmt.Errorf("AYUMI_ALLOWED_IPS: %w", err)
	}
	if c.AllowedHosts, err = ParseHostList(splitList(os.Getenv("AYUMI_ALLOWED_HOSTS"), ",")); err != nil {
		return nil, fmt.Errorf("AYUMI_ALLOWED_HOSTS: %w", err)
	}
	if _, err = ParseIPList(c.TrustedProxies); err != nil {
		return nil, fmt.Errorf("AYUMI_TRUSTED_PROXIES: %w", err)
	}
	return c, nil
}

func (c *Config) Addr() string { return c.Host + ":" + c.Port }

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil && n > 0 {
		return n
	}
	return def
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func splitList(v, sep string) []string {
	var out []string
	for _, p := range strings.Split(v, sep) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
