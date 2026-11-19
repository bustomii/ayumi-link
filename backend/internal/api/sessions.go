package api

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/claude"
)

func sessionID(c *fiber.Ctx) (string, error) {
	id := c.Params("id")
	if !claude.IsUUID(id) {
		return "", fiber.NewError(fiber.StatusBadRequest, "sessionId tidak valid")
	}
	return id, nil
}

func (s *Server) listSessions(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"sessions": s.Svc.List()})
}

func (s *Server) getSession(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	d, err := s.Svc.Detail(id, limit, c.Query("before"))
	if errors.Is(err, claude.ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, "session tidak ditemukan")
	}
	if err != nil {
		return err
	}
	return c.JSON(d)
}

func (s *Server) moveSession(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	var b struct {
		Group string `json:"group"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	g := strings.TrimSpace(b.Group)
	if len([]rune(g)) > 60 {
		return fiber.NewError(fiber.StatusBadRequest, "nama folder terlalu panjang")
	}
	if err := s.Svc.SetGroup(id, g); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true, "group": g})
}

func (s *Server) deleteSession(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	mode, err := s.Svc.Delete(id)
	if err != nil {
		return err
	}
	if s.Term != nil {
		s.Term.KillChat(id)
	}
	if s.Web != nil {
		s.Web.KillChatTabs(id)
	}
	_ = s.Store.DeleteToolsTabs(id)
	return c.JSON(fiber.Map{"ok": true, "mode": mode})
}
