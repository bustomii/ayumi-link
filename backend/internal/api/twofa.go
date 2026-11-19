package api

import "github.com/gofiber/fiber/v2"

// twoFactorBegin: langkah 1 mengganti autentikator (verifikasi ulang password + kode saat ini).
func (s *Server) twoFactorBegin(c *fiber.Ctx) error {
	var b struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	res, err := s.Auth.BeginTwoFactorChange(c.Cookies(cookieName), b.Password, b.Code)
	if err != nil {
		return authError(c, err)
	}
	return c.JSON(res)
}

// twoFactorConfirm: langkah 2 (kode dari perangkat baru). Mengembalikan kode pemulihan baru sekali.
func (s *Server) twoFactorConfirm(c *fiber.Ctx) error {
	var b struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	codes, err := s.Auth.ConfirmTwoFactorChange(c.Cookies(cookieName), b.Code)
	if err != nil {
		return authError(c, err)
	}
	return c.JSON(fiber.Map{"ok": true, "recoveryCodes": codes})
}
