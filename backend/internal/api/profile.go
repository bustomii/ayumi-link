package api

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/auth"
	"ayumi-link/internal/claude"
	"ayumi-link/internal/config"
	"ayumi-link/internal/store"
)

const (
	defaultModel  = "claude-sonnet-5-5"
	defaultEffort = "medium"
)

// configInfo: konfigurasi server yang ditampilkan (hanya baca) di halaman profil.
type configInfo struct {
	Version        string   `json:"version"`
	Address        string   `json:"address"`
	Roots          []string `json:"roots"`
	ClaudeBin      string   `json:"claudeBin"`
	ClaudeVersion  string   `json:"claudeVersion"`
	DataDir        string   `json:"dataDir"`
	CookieSecure   bool     `json:"cookieSecure"`
	TrustProxy     bool     `json:"trustProxy"`
	PermissionMode string   `json:"permissionMode"`
}

type profileResponse struct {
	*auth.Profile
	Prefs  store.Prefs `json:"prefs"`
	Config configInfo  `json:"config"`
}

// prefs mengembalikan preferensi tersimpan; nilai tak valid/kosong diganti default.
func (s *Server) prefs() store.Prefs {
	def := store.Prefs{PermissionMode: s.Config.PermissionMode, Model: defaultModel, Effort: defaultEffort}
	if !claude.ValidPermissionMode(def.PermissionMode) {
		def.PermissionMode = "auto"
	}
	saved, ok := s.Store.Prefs()
	if !ok {
		return def
	}
	if claude.ValidPermissionMode(saved.PermissionMode) {
		def.PermissionMode = saved.PermissionMode
	}
	if claude.ValidModel(saved.Model) {
		def.Model = saved.Model
	}
	if claude.ValidEffort(saved.Effort) {
		def.Effort = saved.Effort
	}
	return def
}

// claudeVer menjalankan `claude --version` sekali (di-cache); kosong bila gagal.
func (s *Server) claudeVer() string {
	s.versionOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, s.Runner.Bin, "--version").Output(); err == nil {
			s.claudeVersion = strings.TrimSpace(string(out))
		}
	})
	return s.claudeVersion
}

func (s *Server) configInfo() configInfo {
	roots := s.Browser.Roots()
	if roots == nil {
		roots = []string{}
	}
	return configInfo{
		Version: config.Version, Address: s.Config.Addr(), Roots: roots,
		ClaudeBin: s.Runner.Bin, ClaudeVersion: s.claudeVer(), DataDir: s.Config.DataDir,
		CookieSecure: s.Config.CookieSecure, TrustProxy: s.Config.TrustProxy,
		PermissionMode: s.Config.PermissionMode,
	}
}

func (s *Server) getProfile(c *fiber.Ctx) error {
	p, err := s.Auth.Profile(c.Cookies(cookieName))
	if err != nil {
		return authError(c, err)
	}
	return c.JSON(profileResponse{Profile: p, Prefs: s.prefs(), Config: s.configInfo()})
}

func (s *Server) getPrefs(c *fiber.Ctx) error { return c.JSON(s.prefs()) }

func (s *Server) putPrefs(c *fiber.Ctx) error {
	var b store.Prefs
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	switch {
	case !claude.ValidPermissionMode(b.PermissionMode):
		return fiber.NewError(fiber.StatusBadRequest, "mode izin tidak valid")
	case !claude.ValidModel(b.Model):
		return fiber.NewError(fiber.StatusBadRequest, "model tidak valid")
	case !claude.ValidEffort(b.Effort):
		return fiber.NewError(fiber.StatusBadRequest, "effort tidak valid")
	}
	if err := s.Store.SetPrefs(b); err != nil {
		return err
	}
	return c.JSON(b)
}

func (s *Server) changePassword(c *fiber.Ctx) error {
	var b struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
		Code            string `json:"code"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	if err := s.Auth.ChangePassword(c.Cookies(cookieName), b.CurrentPassword, b.NewPassword, b.Code); err != nil {
		return authError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) regenerateRecovery(c *fiber.Ctx) error {
	var b struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	codes, err := s.Auth.RegenerateRecovery(c.Cookies(cookieName), b.Password, b.Code)
	if err != nil {
		return authError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true, "recoveryCodes": codes})
}

func (s *Server) revokeSession(c *fiber.Ctx) error {
	if err := s.Auth.RevokeSession(c.Cookies(cookieName), c.Params("id")); err != nil {
		return authError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) revokeOthers(c *fiber.Ctx) error {
	n, err := s.Auth.RevokeOthers(c.Cookies(cookieName))
	if err != nil {
		return authError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true, "revoked": n})
}
