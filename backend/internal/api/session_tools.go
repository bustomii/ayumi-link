package api

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/claude"
)

func (s *Server) listSessionChanges(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	list, err := s.Svc.ListChanges(id)
	if errors.Is(err, claude.ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, "session tidak ditemukan")
	}
	if errors.Is(err, claude.ErrBadID) {
		return fiber.NewError(fiber.StatusBadRequest, "sessionId tidak valid")
	}
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"changes": list})
}

func (s *Server) getSessionChangeDiff(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	path := strings.TrimSpace(c.Query("path"))
	if path == "" {
		return fiber.NewError(fiber.StatusBadRequest, "query path wajib")
	}
	diff, err := s.Svc.GetChangeDiff(id, path)
	if errors.Is(err, claude.ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, "file/session tidak ditemukan")
	}
	if errors.Is(err, claude.ErrBadID) {
		return fiber.NewError(fiber.StatusBadRequest, "sessionId tidak valid")
	}
	if err != nil {
		return err
	}
	return c.JSON(diff)
}

// enterTerminal: satu shell global; bila sessionId ada, cwd = folder chat (atau root AYUMI_ROOTS).
func (s *Server) enterTerminal(c *fiber.Ctx) error {
	if s.Shell == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "terminal tidak tersedia")
	}
	sessionID := strings.TrimSpace(c.Query("sessionId"))
	if sessionID != "" && !claude.IsUUID(sessionID) {
		return fiber.NewError(fiber.StatusBadRequest, "sessionId tidak valid")
	}
	cwd, err := s.Shell.Enter(sessionID)
	if err != nil {
		var re *claude.RequestError
		if errors.As(err, &re) {
			return c.Status(re.Status).JSON(fiber.Map{"error": re.Message})
		}
		return err
	}
	return c.JSON(fiber.Map{"cwd": cwd})
}

func (s *Server) runTerminal(c *fiber.Ctx) error {
	if s.Shell == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "terminal tidak tersedia")
	}
	var b struct {
		Command string `json:"command"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	res, err := s.Shell.Run(c.UserContext(), b.Command)
	if err != nil {
		var re *claude.RequestError
		if errors.As(err, &re) {
			return c.Status(re.Status).JSON(fiber.Map{"error": re.Message})
		}
		return err
	}
	return c.JSON(res)
}
