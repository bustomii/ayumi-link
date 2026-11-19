package api

import (
	"strings"
	"unicode"

	"github.com/gofiber/fiber/v2"
)

const (
	maxTitleLen = 120
	maxGroupLen = 60
)

// cleanName merapikan nama: spasi di pinggir dibuang, karakter kontrol/baris baru jadi spasi,
// spasi ganda diringkas. ok=false bila lebih panjang dari max.
func cleanName(s string, max int) (string, bool) {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	return s, len([]rune(s)) <= max
}

func (s *Server) renameSession(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	var b struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	title, ok := cleanName(b.Title, maxTitleLen)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "nama session terlalu panjang (maks 120 karakter)")
	}
	if err := s.Svc.SetTitle(id, title); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true, "title": title})
}

func (s *Server) renameGroup(c *fiber.Ctx) error {
	var b struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	from, _ := cleanName(b.From, 200)
	to, ok := cleanName(b.To, maxGroupLen)
	if from == "" {
		return fiber.NewError(fiber.StatusBadRequest, "folder asal wajib diisi")
	}
	if to == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nama folder tidak boleh kosong")
	}
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "nama folder terlalu panjang (maks 60 karakter)")
	}
	if from != to {
		if err := s.Svc.RenameGroup(from, to); err != nil {
			return err
		}
	}
	return c.JSON(fiber.Map{"ok": true, "group": to})
}
