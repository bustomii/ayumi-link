package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/claude"
)

// forkSession: salin session ke folder kerja baru (untuk session yang foldernya hilang/pindah).
func (s *Server) forkSession(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	var b struct {
		Cwd string `json:"cwd"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	real, ok := s.Browser.ResolveDir(b.Cwd)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "folder tidak diizinkan atau tidak ada")
	}
	newID, err := s.Svc.Fork(id, real)
	if errors.Is(err, claude.ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, "session tidak ditemukan")
	}
	if errors.Is(err, claude.ErrBadID) {
		return fiber.NewError(fiber.StatusBadRequest, "sessionId tidak valid")
	}
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"id": newID, "cwd": real})
}
