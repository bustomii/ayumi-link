package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/branding"
	"ayumi-link/internal/store"
)

type brandingDTO struct {
	Icon      string   `json:"icon"`
	Bg        string   `json:"bg"`
	Fg        string   `json:"fg"`
	UpdatedAt int64    `json:"updatedAt"`
	Icons     []string `json:"icons,omitempty"`
	Custom    bool     `json:"custom"`
}

func (s *Server) brandingOrDefault() store.Branding {
	if b, ok := s.Store.Branding(); ok && branding.ValidIcon(b.Icon) {
		icon, bg, fg := branding.Normalize(b.Icon, b.Bg, b.Fg)
		return store.Branding{Icon: icon, Bg: bg, Fg: fg, UpdatedAt: b.UpdatedAt}
	}
	return store.Branding{
		Icon: branding.DefaultIcon,
		Bg:   branding.DefaultBg,
		Fg:   branding.DefaultFg,
	}
}

func (s *Server) getBranding(c *fiber.Ctx) error {
	b := s.brandingOrDefault()
	_, custom := s.Store.Branding()
	return c.JSON(brandingDTO{
		Icon: b.Icon, Bg: b.Bg, Fg: b.Fg, UpdatedAt: b.UpdatedAt,
		Icons: branding.Icons, Custom: custom,
	})
}

func (s *Server) putBranding(c *fiber.Ctx) error {
	var body struct {
		Icon string `json:"icon"`
		Bg   string `json:"bg"`
		Fg   string `json:"fg"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	if !branding.ValidIcon(body.Icon) {
		return fiber.NewError(fiber.StatusBadRequest, "ikon tidak valid")
	}
	if !branding.ValidColor(body.Bg) {
		return fiber.NewError(fiber.StatusBadRequest, "warna latar tidak valid")
	}
	if body.Fg == "" {
		body.Fg = branding.DefaultFg
	}
	if !branding.ValidColor(body.Fg) {
		return fiber.NewError(fiber.StatusBadRequest, "warna ikon tidak valid")
	}
	icon, bg, fg := branding.Normalize(body.Icon, body.Bg, body.Fg)
	now := time.Now().Unix()
	for _, item := range []struct {
		name string
		size int
	}{
		{"branding-192.png", 192},
		{"branding-512.png", 512},
		{"branding-apple.png", 180},
	} {
		png, err := branding.PNG(icon, bg, fg, item.size)
		if err != nil {
			return err
		}
		if err := s.Store.WriteBrandingPNG(item.name, png); err != nil {
			return err
		}
	}
	b := store.Branding{Icon: icon, Bg: bg, Fg: fg, UpdatedAt: now}
	if err := s.Store.SetBranding(b); err != nil {
		return err
	}
	return c.JSON(brandingDTO{Icon: icon, Bg: bg, Fg: fg, UpdatedAt: now, Icons: branding.Icons, Custom: true})
}

func (s *Server) resetBranding(c *fiber.Ctx) error {
	s.Store.ClearBranding()
	b := s.brandingOrDefault()
	return c.JSON(brandingDTO{
		Icon: b.Icon, Bg: b.Bg, Fg: b.Fg, UpdatedAt: 0,
		Icons: branding.Icons, Custom: false,
	})
}

// ---- aset publik (tanpa login; dibutuhkan PWA / favicon) ----

func (s *Server) publicBrandingMeta(c *fiber.Ctx) error {
	b := s.brandingOrDefault()
	_, custom := s.Store.Branding()
	c.Set(fiber.HeaderCacheControl, "no-cache")
	return c.JSON(fiber.Map{
		"icon": b.Icon, "bg": b.Bg, "fg": b.Fg,
		"updatedAt": b.UpdatedAt, "custom": custom,
	})
}

func (s *Server) publicBrandingSVG(c *fiber.Ctx) error {
	b := s.brandingOrDefault()
	c.Set(fiber.HeaderContentType, "image/svg+xml; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "public, max-age=3600")
	return c.SendString(branding.SVG(b.Icon, b.Bg, b.Fg, 512))
}

func (s *Server) serveBrandingPNG(c *fiber.Ctx, storeName string, size int) error {
	b := s.brandingOrDefault()
	if data, ok := s.Store.BrandingPNG(storeName); ok {
		c.Set(fiber.HeaderContentType, "image/png")
		c.Set(fiber.HeaderCacheControl, "public, max-age=3600")
		return c.Send(data)
	}
	// Belum ada file tersimpan → generate on the fly.
	png, err := branding.PNG(b.Icon, b.Bg, b.Fg, size)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, "image/png")
	c.Set(fiber.HeaderCacheControl, "public, max-age=300")
	return c.Send(png)
}

func (s *Server) mountBrandingAssets(app *fiber.App, staticDir string) {
	app.Get("/branding/meta.json", s.publicBrandingMeta)
	app.Get("/branding/icon.svg", s.publicBrandingSVG)
	app.Get("/branding/pwa-192.png", func(c *fiber.Ctx) error { return s.serveBrandingPNG(c, "branding-192.png", 192) })
	app.Get("/branding/pwa-512.png", func(c *fiber.Ctx) error { return s.serveBrandingPNG(c, "branding-512.png", 512) })
	app.Get("/branding/apple-touch-icon.png", func(c *fiber.Ctx) error {
		return s.serveBrandingPNG(c, "branding-apple.png", 180)
	})

	// Override favicon / PWA icons bawaan bila branding custom aktif.
	app.Get("/favicon.svg", func(c *fiber.Ctx) error {
		if _, ok := s.Store.Branding(); ok {
			return s.publicBrandingSVG(c)
		}
		if staticDir == "" {
			return s.publicBrandingSVG(c)
		}
		return c.SendFile(filepath.Join(staticDir, "favicon.svg"))
	})
	app.Get("/apple-touch-icon.png", func(c *fiber.Ctx) error {
		if _, ok := s.Store.Branding(); ok {
			return s.serveBrandingPNG(c, "branding-apple.png", 180)
		}
		if staticDir == "" {
			return s.serveBrandingPNG(c, "branding-apple.png", 180)
		}
		return c.SendFile(filepath.Join(staticDir, "apple-touch-icon.png"))
	})
	for _, name := range []struct{ path, file string; size int }{
		{"/pwa-192.png", "branding-192.png", 192},
		{"/pwa-512.png", "branding-512.png", 512},
	} {
		p, f, sz := name.path, name.file, name.size
		app.Get(p, func(c *fiber.Ctx) error {
			if _, ok := s.Store.Branding(); ok {
				return s.serveBrandingPNG(c, f, sz)
			}
			if staticDir == "" {
				return s.serveBrandingPNG(c, f, sz)
			}
			return c.SendFile(filepath.Join(staticDir, filepath.Base(p)))
		})
	}
}

func (s *Server) dynamicManifest(c *fiber.Ctx, staticDir string) error {
	path := filepath.Join(staticDir, "manifest.webmanifest")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "manifest tidak ada")
	}
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderContentType, "application/manifest+json; charset=utf-8")

	b, custom := s.Store.Branding()
	if !custom {
		return c.Send(raw)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return c.Send(raw)
	}
	_, bg, _ := branding.Normalize(b.Icon, b.Bg, b.Fg)
	v := b.UpdatedAt
	if v == 0 {
		v = 1
	}
	m["background_color"] = bg
	m["theme_color"] = bg
	qv := strconv.FormatInt(v, 10)
	m["icons"] = []map[string]any{
		{"src": "/branding/pwa-192.png?v=" + qv, "sizes": "192x192", "type": "image/png"},
		{"src": "/branding/pwa-512.png?v=" + qv, "sizes": "512x512", "type": "image/png"},
		{"src": "/branding/pwa-512.png?v=" + qv, "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
	}
	out, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return c.Send(out)
}
