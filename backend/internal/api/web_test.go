package api

import "testing"

func TestNormalizeWebURL(t *testing.T) {
	ok := map[string]string{
		"example.com":              "https://example.com",
		"http://localhost:3000/a":  "http://localhost:3000/a",
		"  https://x.id/p?q=1  ":   "https://x.id/p?q=1",
	}
	for in, want := range ok {
		got, err := normalizeWebURL(in)
		if err != nil || got != want {
			t.Errorf("normalizeWebURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "   ", "javascript:alert(1)", "file:///etc/passwd", "data:text/html,x", "chrome://settings", "ftp://x.com"} {
		if got, err := normalizeWebURL(bad); err == nil {
			t.Errorf("normalizeWebURL(%q) = %q; want error", bad, got)
		}
	}
}
