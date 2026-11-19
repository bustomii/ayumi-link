package api

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/claude"
)

func (s *Server) getClaudeAuth(c *fiber.Ctx) error {
	if s.CliAuth == nil {
		return c.JSON(claude.CliAuthStatus{Message: "auth CLI tidak tersedia"})
	}
	ctx, cancel := context.WithTimeout(c.UserContext(), 8*time.Second)
	defer cancel()
	st, err := s.CliAuth.Status(ctx)
	if err != nil {
		return err
	}
	return c.JSON(st)
}

func (s *Server) putClaudeAuth(c *fiber.Ctx) error {
	if s.CliAuth == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "auth CLI tidak tersedia")
	}
	var b struct {
		Method string `json:"method"`
		Token  string `json:"token"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	b.Method = strings.TrimSpace(b.Method)
	b.Token = strings.TrimSpace(b.Token)
	ctx, cancel := context.WithTimeout(c.UserContext(), 12*time.Second)
	defer cancel()
	st, err := s.CliAuth.Save(ctx, b.Method, b.Token)
	if err != nil {
		if re, ok := err.(*claude.RequestError); ok {
			return fiber.NewError(re.Status, re.Message)
		}
		return err
	}
	return c.JSON(st)
}

func (s *Server) deleteClaudeAuth(c *fiber.Ctx) error {
	if s.CliAuth == nil {
		return c.JSON(fiber.Map{"ok": true})
	}
	if err := s.CliAuth.Clear(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(c.UserContext(), 8*time.Second)
	defer cancel()
	st, err := s.CliAuth.Status(ctx)
	if err != nil {
		return c.JSON(fiber.Map{"ok": true})
	}
	return c.JSON(st)
}
