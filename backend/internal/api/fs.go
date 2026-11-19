package api

import "github.com/gofiber/fiber/v2"

func (s *Server) browse(c *fiber.Ctx) error {
	l, ok := s.Browser.List(c.Query("path"))
	if !ok {
		return fiber.NewError(fiber.StatusForbidden, "folder tidak tersedia")
	}
	return c.JSON(l)
}
