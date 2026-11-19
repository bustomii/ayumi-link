package api

import (
	"errors"
	"io"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/claude"
	"ayumi-link/internal/uploads"
)

// uploadsPath: satu-satunya rute yang menerima multipart dan badan besar.
const uploadsPath = "/api/uploads"

// maxJSONBody: batas badan untuk semua rute selain unggahan.
const maxJSONBody = 1 << 20

func uploadError(err error) error {
	var ue *uploads.Error
	if errors.As(err, &ue) {
		return fiber.NewError(ue.Status, ue.Message)
	}
	return err
}

func (s *Server) uploadLimits(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"maxBytes": s.Uploads.MaxBytes(),
		"maxFiles": claude.MaxAttachments,
		"types":    uploads.Allowed(),
	})
}

// uploadFile menerima satu file (field "file"). Jenis divalidasi dari isinya.
func (s *Server) uploadFile(c *fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file tidak ditemukan pada permintaan")
	}
	if fh.Size > s.Uploads.MaxBytes() {
		return uploadError(&uploads.Error{Status: fiber.StatusRequestEntityTooLarge, Code: "size",
			Message: "file terlalu besar (maksimal " + strconv.FormatInt(s.Uploads.MaxBytes()>>20, 10) + " MB)"})
	}
	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := s.Uploads.Save(fh.Filename, f)
	if err != nil {
		return uploadError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(info)
}

func isInlineImage(k uploads.Kind) bool { return k == uploads.KindImage }

// uploadDownload menyajikan lampiran. Gambar tampil inline; selain itu selalu diunduh.
func (s *Server) uploadDownload(c *fiber.Ctx) error {
	info, err := s.Uploads.Get(c.Params("id"))
	if err != nil {
		return uploadError(err)
	}
	f, err := os.Open(s.Uploads.Path(info))
	if err != nil {
		return uploadError(uploads.ErrNotFound)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, s.Uploads.MaxBytes()+1))
	if err != nil {
		return err
	}
	disp := "attachment"
	if isInlineImage(info.Kind) {
		disp = "inline"
	}
	c.Set(fiber.HeaderContentType, info.Mime)
	c.Set(fiber.HeaderContentDisposition, disp+"; filename*=UTF-8''"+urlEscape(info.Name))
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set(fiber.HeaderContentSecurityPolicy, "default-src 'none'; sandbox")
	c.Set(fiber.HeaderCacheControl, "private, max-age=3600")
	return c.Send(data)
}

func (s *Server) uploadDelete(c *fiber.Ctx) error {
	if err := s.Uploads.Delete(c.Params("id")); err != nil {
		return uploadError(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

// urlEscape: percent-encode nama file untuk header Content-Disposition (RFC 5987).
func urlEscape(s string) string {
	const hex = "0123456789ABCDEF"
	var b []byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_' || ch == '.' {
			b = append(b, ch)
		} else {
			b = append(b, '%', hex[ch>>4], hex[ch&15])
		}
	}
	return string(b)
}
