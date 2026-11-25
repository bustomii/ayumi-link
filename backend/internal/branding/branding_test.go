package branding

import (
	"bytes"
	"image/png"
	"testing"
)

func TestNormalizeAndSVG(t *testing.T) {
	icon, bg, fg := Normalize("nope", "bad", "")
	if icon != DefaultIcon || bg != DefaultBg || fg != DefaultFg {
		t.Fatalf("normalize defaults: %s %s %s", icon, bg, fg)
	}
	svg := SVG("android", "#1e3a8a", "#ffffff", 128)
	if !bytes.Contains([]byte(svg), []byte("#1e3a8a")) || !bytes.Contains([]byte(svg), []byte("<svg")) {
		t.Fatalf("svg lemah: %s", svg[:80])
	}
}

func TestPNGAllIcons(t *testing.T) {
	for _, id := range Icons {
		b, err := PNG(id, "#111827", "#ffffff", 64)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s decode: %v", id, err)
		}
		if img.Bounds().Dx() != 64 {
			t.Fatalf("%s size %v", id, img.Bounds())
		}
	}
}

func TestValidColor(t *testing.T) {
	if !ValidColor("#aB12Cd") || ValidColor("#fff") || ValidColor("red") {
		t.Fatal("ValidColor")
	}
}
