package api

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// csp: aplikasi hanya memuat sumber dari origin-nya sendiri (font dan skrip di-bundle).
// 'unsafe-inline' pada style dibutuhkan atribut style yang dipasang React.
const csp = "default-src 'self'; " +
	"img-src 'self' data: blob:; " +
	"style-src 'self' 'unsafe-inline'; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

// mountStatic menyajikan hasil build frontend dari dir: file statis apa adanya, dan
// `index.html` untuk rute klien (mis. /s/<id>) supaya reload dan tautan langsung berfungsi.
// Rute /api tidak pernah jatuh ke index.html.
func mountStatic(app *fiber.App, dir string) bool {
	index := filepath.Join(dir, "index.html")
	if st, err := os.Stat(index); err != nil || st.IsDir() {
		return false
	}

	// Aset ber-hash dari Vite aman di-cache selamanya.
	app.Static("/assets", filepath.Join(dir, "assets"), fiber.Static{
		Compress: true, MaxAge: 365 * 24 * 3600,
	})
	// Manifest PWA butuh MIME yang benar; jangan andalkan deteksi ekstensi Fiber/Static.
	app.Get("/manifest.webmanifest", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, "no-cache")
		c.Set(fiber.HeaderContentType, "application/manifest+json; charset=utf-8")
		return c.SendFile(filepath.Join(dir, "manifest.webmanifest"))
	})

	// index.html + service worker: selalu revalidasi (jangan pakai SW usang).
	app.Use(func(c *fiber.Ctx) error {
		p := c.Path()
		switch {
		case p == "/" || p == "/index.html":
			c.Set(fiber.HeaderCacheControl, "no-cache")
		case p == "/sw.js" || p == "/registerSW.js" || strings.HasPrefix(p, "/workbox-"):
			c.Set(fiber.HeaderCacheControl, "no-cache")
		}
		return c.Next()
	})
	app.Static("/", dir, fiber.Static{Compress: true, Index: "index.html"})

	app.Use(func(c *fiber.Ctx) error {
		if (c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead) || strings.HasPrefix(c.Path(), "/api") {
			return c.Next()
		}
		c.Set(fiber.HeaderCacheControl, "no-cache") // selalu cek versi terbaru shell aplikasi
		return c.SendFile(index)
	})
	return true
}
