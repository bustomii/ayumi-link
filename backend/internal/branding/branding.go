// Package branding: ikon PWA / favicon (preset + warna latar).
package branding

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultIcon = "ayumi"
	DefaultBg   = "#0c0d10"
	DefaultFg   = "#ffffff"
)

// Preset ikon yang diizinkan (sinkron dengan FE).
var Icons = []string{
	"ayumi", "apple", "android", "linux", "windows",
	"terminal", "spark", "chat", "hex", "bot", "globe",
}

var iconSet = func() map[string]bool {
	m := make(map[string]bool, len(Icons))
	for _, id := range Icons {
		m[id] = true
	}
	return m
}()

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func ValidIcon(id string) bool { return iconSet[id] }

func ValidColor(c string) bool { return hexColorRe.MatchString(c) }

func Normalize(icon, bg, fg string) (string, string, string) {
	if !ValidIcon(icon) {
		icon = DefaultIcon
	}
	if !ValidColor(bg) {
		bg = DefaultBg
	}
	if !ValidColor(fg) {
		fg = DefaultFg
	}
	return icon, strings.ToLower(bg), strings.ToLower(fg)
}

// GlyphPath: path SVG 24×24 (isi glyph mono).
func GlyphPath(icon string) string {
	switch icon {
	case "apple":
		// Siluet apple sederhana (Mirip ikon macOS).
		return `M16.4 12.2c0-2.1 1.7-3.1 1.8-3.2-1-1.4-2.5-1.6-3-1.7-1.3-.1-2.5.8-3.1.8-.7 0-1.7-.7-2.8-.7-1.4 0-2.8.9-3.5 2.2-1.5 2.6-.4 6.5 1.1 8.6.7 1 1.6 2.2 2.7 2.1 1.1-.1 1.5-.7 2.8-.7s1.7.7 2.8.7c1.2 0 1.9-1 2.6-2 .8-1.2 1.1-2.3 1.1-2.4-.1 0-2.2-.9-2.2-3.5zm-2-6.5c.6-.7 1-1.7.9-2.7-1 .1-2.1.6-2.7 1.4-.6.7-1.1 1.7-.9 2.7 1 .1 2-.6 2.7-1.4z`
	case "android":
		return `M17.6 9.3l1.4-2.4c.1-.2 0-.4-.2-.5-.2-.1-.4 0-.5.2l-1.4 2.4A7.4 7.4 0 0 0 12 7.5c-1.8 0-3.4.6-4.9 1.5L5.7 6.6c-.1-.2-.3-.3-.5-.2-.2.1-.3.3-.2.5l1.4 2.4A7.2 7.2 0 0 0 4.5 14v.8c0 .4.3.7.7.7h1.1v3.8c0 .5.4.9.9.9h.7c.5 0 .9-.4.9-.9V15.5h5.4v3.8c0 .5.4.9.9.9h.7c.5 0 .9-.4.9-.9v-3.8h1.1c.4 0 .7-.3.7-.7V14a7.2 7.2 0 0 0-1.9-4.7zM9.2 12.2a.8.8 0 1 1 0-1.6.8.8 0 0 1 0 1.6zm5.6 0a.8.8 0 1 1 0-1.6.8.8 0 0 1 0 1.6z`
	case "linux":
		// Tux sederhana: kepala + badan + kaki.
		return `M12 3.2c-2.2 0-3.8 1.9-3.8 4.1 0 1.1.3 2 .9 2.7-.9.4-1.6 1.3-1.8 2.5-.3 1.5.3 2.8 1.4 3.5-.2.5-.3 1-.3 1.5 0 1.6 1 2.8 2.6 3.2.3.6.9 1 1.6 1h.8c.7 0 1.3-.4 1.6-1 1.6-.4 2.6-1.6 2.6-3.2 0-.5-.1-1-.3-1.5 1.1-.7 1.7-2 1.4-3.5-.2-1.2-.9-2.1-1.8-2.5.6-.7.9-1.6.9-2.7 0-2.2-1.6-4.1-3.8-4.1zm-1.7 3.3a.7.7 0 1 1 0 1.4.7.7 0 0 1 0-1.4zm3.4 0a.7.7 0 1 1 0 1.4.7.7 0 0 1 0-1.4zM12 11.2c.9 0 1.6.3 2 .7l-4 0c.4-.4 1.1-.7 2-.7z`
	case "windows":
		return `M3.5 5.2l7.2-.9v7.1H3.5V5.2zm8.3-.9l8.7-1.1v8.2h-8.7V4.3zM3.5 12.6h7.2v7.2l-7.2-.9v-6.3zm8.3 0h8.7v8.3l-8.7-1.1v-7.2z`
	case "terminal":
		return `M4 5.5A1.5 1.5 0 0 1 5.5 4h13A1.5 1.5 0 0 1 20 5.5v13a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 18.5v-13zM8.2 9.1L6.8 10.5 8.2 11.9l1.1-1.1-1-1 1-1-1.1-1.1zm3.3 4.3h5v1.5h-5v-1.5z`
	case "spark":
		return `M12 2.5l1.4 6.1 6.1 1.4-6.1 1.4L12 17.5l-1.4-6.1L4.5 10l6.1-1.4L12 2.5zm6.5 9.2l.7 3.1 3.1.7-3.1.7-.7 3.1-.7-3.1-3.1-.7 3.1-.7.7-3.1z`
	case "chat":
		return `M5 5.5A2.5 2.5 0 0 1 7.5 3h9A2.5 2.5 0 0 1 19 5.5v8a2.5 2.5 0 0 1-2.5 2.5H11l-3.8 3.2c-.5.4-1.2 0-1.2-.6V16H7.5A2.5 2.5 0 0 1 5 13.5v-8z`
	case "hex":
		return `M12 2.4l7.5 4.3v8.6L12 19.6l-7.5-4.3V6.7L12 2.4zm0 3.2L7.8 8v6L12 16.4 16.2 14V8L12 5.6z`
	case "bot":
		return `M11 3h2v2.2h3.5A2.3 2.3 0 0 1 18.8 7.5v9A2.3 2.3 0 0 1 16.5 18.8h-9A2.3 2.3 0 0 1 5.2 16.5v-9A2.3 2.3 0 0 1 7.5 5.2H11V3zm-2.2 7.2a1.2 1.2 0 1 0 0 2.4 1.2 1.2 0 0 0 0-2.4zm6.4 0a1.2 1.2 0 1 0 0 2.4 1.2 1.2 0 0 0 0-2.4zM9 14.2h6v1.4H9v-1.4z`
	case "globe":
		return `M12 3.2a8.8 8.8 0 1 1 0 17.6 8.8 8.8 0 0 1 0-17.6zm0 1.5c-.9 1.3-1.5 3.2-1.6 5.3h3.2c-.1-2.1-.7-4-1.6-5.3zm-3.2.7C7.3 6.6 6 8.6 5.5 11h2.7c.1-1.8.5-3.4 1.1-4.6zM18.5 11c-.5-2.4-1.8-4.4-3.3-5.6.6 1.2 1 2.8 1.1 4.6h2.2zM5.5 13c.5 2.4 1.8 4.4 3.3 5.6-.6-1.2-1-2.8-1.1-4.6H5.5zm5.7 0c.1 2.1.7 4 1.6 5.3.9-1.3 1.5-3.2 1.6-5.3h-3.2zm5.1 0c-.1 1.8-.5 3.4-1.1 4.6 1.5-1.2 2.8-3.2 3.3-5.6h-2.2z`
	default: // ayumi — monogram A / link mark
		return `M12 3.5L4.5 19h3.2l1.5-3.4h5.6L16.3 19h3.2L12 3.5zm0 4.2l2.1 4.8H9.9L12 7.7z`
	}
}

// SVG membuat ikon persegi dengan latar + glyph (untuk favicon).
func SVG(icon, bg, fg string, size int) string {
	icon, bg, fg = Normalize(icon, bg, fg)
	if size <= 0 {
		size = 512
	}
	// Padding ~20% agar aman untuk maskable / rounded OS.
	scale := float64(size) * 0.52 / 24.0
	tx := float64(size) * 0.24
	ty := tx
	path := GlyphPath(icon)
	return fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+
			`<rect width="%d" height="%d" rx="%d" fill="%s"/>`+
			`<g transform="translate(%.2f %.2f) scale(%.4f)" fill="%s"><path d="%s"/></g></svg>`,
		size, size, size, size,
		size, size, size/5, bg,
		tx, ty, scale, fg, path,
	)
}
