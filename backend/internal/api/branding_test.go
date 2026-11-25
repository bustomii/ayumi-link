package api

import (
	"io"
	"net/http/httptest"
	"testing"
)

func TestBrandingPutAndPublicAssets(t *testing.T) {
	e := newEnv(t)
	cookie, _, _ := setupAccount(t, e)

	res, _ := e.do(t, "PUT", "/api/profile/branding", `{"icon":"android","bg":"#1e3a8a","fg":"#ffffff"}`, cookie, nil)
	if res.StatusCode != 200 {
		t.Fatalf("put branding: %d", res.StatusCode)
	}
	res, got := e.do(t, "GET", "/api/profile/branding", "", cookie, nil)
	if res.StatusCode != 200 || got["icon"] != "android" || got["custom"] != true {
		t.Fatalf("get branding: %v", got)
	}
	// publik
	req := httptest.NewRequest("GET", "/branding/pwa-192.png", nil)
	pngRes, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(pngRes.Body)
	if pngRes.StatusCode != 200 || len(body) < 100 || pngRes.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("png publik: %d len=%d ct=%s", pngRes.StatusCode, len(body), pngRes.Header.Get("Content-Type"))
	}
	res, _ = e.do(t, "DELETE", "/api/profile/branding", "", cookie, nil)
	if res.StatusCode != 200 {
		t.Fatalf("reset: %d", res.StatusCode)
	}
}
