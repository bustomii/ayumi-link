package api

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/auth"
)

// clientInfo: asal request, disimpan bersama sesi untuk ditampilkan di profil.
func clientInfo(c *fiber.Ctx) auth.ClientInfo {
	return auth.ClientInfo{UA: c.Get(fiber.HeaderUserAgent), IP: c.IP()}
}

func (s *Server) secureCookie(c *fiber.Ctx) bool {
	if s.Config.CookieSecure || c.Secure() {
		return true
	}
	return s.Config.TrustProxy && c.Get(fiber.HeaderXForwardedProto) == "https"
}

func (s *Server) setSession(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name: cookieName, Value: token, Path: "/",
		MaxAge: auth.SessionMaxAge(), HTTPOnly: true, SameSite: fiber.CookieSameSiteStrictMode,
		Secure: s.secureCookie(c),
	})
}

func (s *Server) clearSession(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0),
		HTTPOnly: true, SameSite: fiber.CookieSameSiteStrictMode, Secure: s.secureCookie(c),
	})
}

// authError memetakan kesalahan paket auth ke respons HTTP.
func authError(c *fiber.Ctx, err error) error {
	var locked *auth.LockedError
	switch {
	case errors.As(err, &locked):
		secs := int(time.Until(locked.Until).Seconds()) + 1
		c.Set(fiber.HeaderRetryAfter, strconv.Itoa(secs))
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": locked.Error(), "code": "locked", "retryAfter": secs})
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrInvalidCode):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error(), "code": "invalid"})
	case errors.Is(err, auth.ErrChallengeExpired):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error(), "code": "challenge_expired"})
	case errors.Is(err, auth.ErrSetupDone), errors.Is(err, auth.ErrNotConfigured):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, auth.ErrBadSetupToken):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, auth.ErrInvalidUsername), errors.Is(err, auth.ErrWeakPassword), errors.Is(err, auth.ErrNoPendingSetup),
		errors.Is(err, auth.ErrSamePassword), errors.Is(err, auth.ErrCannotRevokeActive), errors.Is(err, auth.ErrNoPendingChange):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrSessionNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, auth.ErrUnauthenticated):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error(), "code": "unauthenticated"})
	}
	return err
}

func (s *Server) authStatus(c *fiber.Ctx) error {
	return c.JSON(s.Auth.Status(c.Cookies(cookieName)))
}

func (s *Server) authSetup(c *fiber.Ctx) error {
	var b struct {
		SetupToken string `json:"setupToken"`
		Username   string `json:"username"`
		Password   string `json:"password"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	res, err := s.Auth.BeginSetup(b.SetupToken, b.Username, b.Password)
	if err != nil {
		return authError(c, err)
	}
	return c.JSON(res)
}

func (s *Server) authSetupConfirm(c *fiber.Ctx) error {
	var b struct {
		SetupToken string `json:"setupToken"`
		Code       string `json:"code"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	recovery, session, err := s.Auth.ConfirmSetup(b.SetupToken, b.Code, clientInfo(c))
	if err != nil {
		return authError(c, err)
	}
	s.setSession(c, session)
	return c.JSON(fiber.Map{"ok": true, "recoveryCodes": recovery})
}

func (s *Server) authLogin(c *fiber.Ctx) error {
	var b struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	token, err := s.Auth.Login(b.Username, b.Password)
	if err != nil {
		return authError(c, err)
	}
	return c.JSON(fiber.Map{"mfaRequired": true, "mfaToken": token})
}

func (s *Server) authLoginVerify(c *fiber.Ctx) error {
	var b struct {
		MFAToken string `json:"mfaToken"`
		Code     string `json:"code"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	session, err := s.Auth.VerifyMFA(b.MFAToken, b.Code, clientInfo(c))
	if err != nil {
		return authError(c, err)
	}
	s.setSession(c, session)
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) authLogout(c *fiber.Ctx) error {
	s.Auth.Logout(c.Cookies(cookieName))
	s.clearSession(c)
	return c.JSON(fiber.Map{"ok": true})
}
