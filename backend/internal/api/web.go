package api

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/browser"
)

// normalizeWebURL: hanya http/https; tanpa skema dianggap https://.
func normalizeWebURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 2048 {
		return "", errors.New("alamat kosong atau terlalu panjang")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", errors.New("alamat tidak valid")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("hanya http/https yang diizinkan")
	}
	return u.String(), nil
}

func (s *Server) webErr(err error) error {
	switch {
	case errors.Is(err, browser.ErrUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	case errors.Is(err, browser.ErrNotRunning):
		return c409(err)
	}
	return fiber.NewError(fiber.StatusBadGateway, err.Error())
}

func c409(err error) error { return fiber.NewError(fiber.StatusConflict, err.Error()) }

func (s *Server) webReady() error {
	if s.Web == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "browser tidak tersedia")
	}
	return nil
}

func (s *Server) webStatus(c *fiber.Ctx) error {
	if s.Web == nil {
		return c.JSON(fiber.Map{"available": false, "running": false})
	}
	info := s.Web.Status(c.UserContext())
	return c.JSON(fiber.Map{"available": s.Web.Available(), "running": info.Running, "url": info.URL, "title": info.Title})
}

func (s *Server) webOpen(c *fiber.Ctx) error {
	if err := s.webReady(); err != nil {
		return err
	}
	var b struct {
		URL       string `json:"url"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		SessionID string `json:"sessionId"`
		TabID     string `json:"tabId"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	target := ""
	if strings.TrimSpace(b.URL) != "" {
		var err error
		if target, err = normalizeWebURL(b.URL); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}
	sid := strings.TrimSpace(b.SessionID)
	if sid != "" {
		if b.TabID != "" {
			if _, err := s.Web.ActivateTab(c.UserContext(), sid, b.TabID); err != nil {
				return s.webErr(err)
			}
			info, err := s.Web.Open(c.UserContext(), target, b.Width, b.Height)
			if err != nil {
				return s.webErr(err)
			}
			return c.JSON(fiber.Map{"available": true, "running": info.Running, "url": info.URL, "title": info.Title, "tabId": b.TabID})
		}
		tab, err := s.Web.OpenForChat(c.UserContext(), sid, target, b.Width, b.Height)
		if err != nil {
			return s.webErr(err)
		}
		return c.JSON(fiber.Map{"available": true, "running": true, "url": tab.URL, "title": tab.Title, "tabId": tab.ID})
	}
	info, err := s.Web.Open(c.UserContext(), target, b.Width, b.Height)
	if err != nil {
		return s.webErr(err)
	}
	return c.JSON(fiber.Map{"available": true, "running": info.Running, "url": info.URL, "title": info.Title})
}

func (s *Server) webFrame(c *fiber.Ctx) error {
	if err := s.webReady(); err != nil {
		return err
	}
	w, _ := strconv.Atoi(c.Query("w"))
	h, _ := strconv.Atoi(c.Query("h"))
	img, info, err := s.Web.Frame(c.UserContext(), w, h)
	if err != nil {
		return s.webErr(err)
	}
	c.Set("X-Browser-Url", url.QueryEscape(info.URL))
	c.Set("X-Browser-Title", url.QueryEscape(info.Title))
	c.Set(fiber.HeaderContentType, "image/jpeg")
	return c.Send(img)
}

func (s *Server) webInput(c *fiber.Ctx) error {
	if err := s.webReady(); err != nil {
		return err
	}
	var in browser.Input
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	if err := s.Web.Do(c.UserContext(), in); err != nil {
		if errors.Is(err, browser.ErrNotRunning) || errors.Is(err, browser.ErrUnavailable) {
			return s.webErr(err)
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) webClose(c *fiber.Ctx) error {
	if err := s.webReady(); err != nil {
		return err
	}
	s.Web.Close()
	return c.JSON(fiber.Map{"ok": true})
}
