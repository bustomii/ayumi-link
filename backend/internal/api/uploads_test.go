package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ayumi-link/internal/auth"
	"ayumi-link/internal/claude"
	"ayumi-link/internal/config"
	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
	"ayumi-link/internal/uploads"
)

var (
	pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...)
	pdfBytes = []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF\n")
)

type uploadEnv struct {
	*env
	cwd string
	out string // folder berisi args/stdin yang diterima "claude" palsu
	up  *uploads.Manager
}

func newUploadEnv(t *testing.T) *uploadEnv {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	_ = os.MkdirAll(out, 0o755)
	script := filepath.Join(dir, "claude")
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + out + "/args\ncat > " + out + "/stdin\n" +
		`echo '{"type":"result","is_error":false,"result":"ok"}'` + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(dir, "proj")
	_ = os.MkdirAll(cwd, 0o755)

	cfg := &config.Config{DataDir: dir, ClaudeHome: dir, DesktopDir: dir, PermissionMode: "auto", AuthRateLimit: 1000}
	st := store.New(dir)
	am, err := auth.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	br := fsbrowse.New([]string{dir})
	up := uploads.New(filepath.Join(dir, "uploads"), uploads.Options{MaxBytes: 64 << 10, MaxTotal: 10 << 20})
	svc := &claude.Service{Home: dir, DesktopDir: dir, Store: st}
	run := &claude.Runner{Svc: svc, Browser: br, Store: st, Bin: script, DefaultPerm: "auto", Uploads: up}
	e := &env{app: New(Deps{Config: cfg, Auth: am, Svc: svc, Runner: run, Browser: br, Store: st, Uploads: up}), auth: am}
	return &uploadEnv{env: e, cwd: cwd, out: out, up: up}
}

func multipartBody(t *testing.T, field, filename string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if filename != "" {
		w, err := mw.CreateFormFile(field, filename)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(data)
	}
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

func (e *uploadEnv) upload(t *testing.T, cookie, filename string, data []byte) (*http.Response, map[string]any) {
	t.Helper()
	buf, ct := multipartBody(t, "file", filename, data)
	req := httptest.NewRequest("POST", "/api/uploads", buf)
	req.Header.Set("Content-Type", ct)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	return res, decode(res)
}

func decode(res *http.Response) map[string]any {
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func zipDocx(t *testing.T) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"[Content_Types].xml": "<Types/>",
		"word/document.xml":   `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Halo</w:t></w:r></w:p></w:body></w:document>`,
	} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func TestUploadRequiresLogin(t *testing.T) {
	e := newUploadEnv(t)
	if res, _ := e.upload(t, "", "a.pdf", pdfBytes); res.StatusCode != 401 {
		t.Fatalf("unggah tanpa login harus 401: %d", res.StatusCode)
	}
	for _, p := range []string{"/api/uploads/limits", "/api/uploads/" + strings.Repeat("a", 32) + "/file"} {
		if res, _ := e.do(t, "GET", p, "", "", nil); res.StatusCode != 401 {
			t.Errorf("GET %s tanpa login harus 401: %d", p, res.StatusCode)
		}
	}
	if res, _ := e.do(t, "DELETE", "/api/uploads/"+strings.Repeat("a", 32), "", "", nil); res.StatusCode != 401 {
		t.Errorf("DELETE tanpa login harus 401: %d", res.StatusCode)
	}
}

func TestUploadAcceptsAllowedAndServesBack(t *testing.T) {
	e := newUploadEnv(t)
	cookie, _, _ := setupAccount(t, e.env)

	res, info := e.upload(t, cookie, "foto saya.png", pngBytes)
	if res.StatusCode != 201 || info["kind"] != "image" || info["name"] != "foto saya.png" || info["mime"] != "image/png" {
		t.Fatalf("unggah png: %d %v", res.StatusCode, info)
	}
	id := info["id"].(string)

	// gambar tampil inline, tidak boleh dijalankan sebagai konten lain
	req := httptest.NewRequest("GET", "/api/uploads/"+id+"/file", nil)
	req.Header.Set("Cookie", cookie)
	got, _ := e.app.Test(req, -1)
	body, _ := io.ReadAll(got.Body)
	if got.StatusCode != 200 || !bytes.Equal(body, pngBytes) || got.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("unduh: %d %q", got.StatusCode, got.Header)
	}
	if got.Header.Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(got.Header.Get("Content-Disposition"), "inline") ||
		!strings.Contains(got.Header.Get("Content-Security-Policy"), "sandbox") {
		t.Fatalf("header keamanan unduhan: %v", got.Header)
	}

	// PDF selalu diunduh (attachment)
	_, pdfInfo := e.upload(t, cookie, "laporan.pdf", pdfBytes)
	req = httptest.NewRequest("GET", "/api/uploads/"+pdfInfo["id"].(string)+"/file", nil)
	req.Header.Set("Cookie", cookie)
	got, _ = e.app.Test(req, -1)
	if !strings.HasPrefix(got.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("PDF harus attachment: %v", got.Header.Get("Content-Disposition"))
	}

	// docx lolos
	if res, _ := e.upload(t, cookie, "surat.docx", zipDocx(t)); res.StatusCode != 201 {
		t.Fatalf("docx: %d", res.StatusCode)
	}

	// hapus
	if res, _ := e.do(t, "DELETE", "/api/uploads/"+id, "", cookie, nil); res.StatusCode != 200 {
		t.Fatalf("hapus: %d", res.StatusCode)
	}
	if res, _ := e.do(t, "GET", "/api/uploads/"+id+"/file", "", cookie, nil); res.StatusCode != 404 {
		t.Fatalf("setelah hapus harus 404: %d", res.StatusCode)
	}
}

func TestUploadRejectsForbiddenTypes(t *testing.T) {
	e := newUploadEnv(t)
	cookie, _, _ := setupAccount(t, e.env)
	exe := append([]byte("MZ"), bytes.Repeat([]byte{0x90}, 64)...)
	bad := []struct {
		name string
		data []byte
		want int
	}{
		{"virus.exe", exe, 415},
		{"menyamar.pdf", exe, 415},
		{"skrip.sh", []byte("#!/bin/sh"), 415},
		{"halaman.html", []byte("<script>1</script>"), 415},
		{"gambar.svg", []byte("<svg/>"), 415},
		{"arsip.zip", zipDocx(t), 415},
		{"video.mp4", []byte("ftypmp42"), 415},
		{"catatan.txt", []byte("halo"), 415},
		{"lama.xls", []byte("\xD0\xCF\x11\xE0"), 415},
		{"lama.doc", []byte("\xD0\xCF\x11\xE0"), 415},
		{"kosong.pdf", nil, 400},
	}
	for _, c := range bad {
		res, body := e.upload(t, cookie, c.name, c.data)
		if res.StatusCode != c.want || body["error"] == nil {
			t.Errorf("%s harus %d dengan pesan error: %d %v", c.name, c.want, res.StatusCode, body)
		}
	}
	// besar (batas 64 KB di lingkungan uji)
	big := append([]byte("%PDF-"), bytes.Repeat([]byte("x"), 70<<10)...)
	if res, body := e.upload(t, cookie, "besar.pdf", big); res.StatusCode != 413 {
		t.Errorf("file besar harus 413: %d %v", res.StatusCode, body)
	}
	// tidak ada yang tersimpan dari semua percobaan gagal
	if entries, _ := os.ReadDir(e.up.Root()); len(entries) != 0 {
		t.Errorf("unggahan yang ditolak tidak boleh meninggalkan file: %v", entries)
	}
}

func TestUploadContentTypeRules(t *testing.T) {
	e := newUploadEnv(t)
	cookie, _, _ := setupAccount(t, e.env)

	// JSON ke rute unggah ditolak
	if res, _ := e.do(t, "POST", "/api/uploads", `{"file":"x"}`, cookie, nil); res.StatusCode != 415 {
		t.Errorf("JSON ke /api/uploads harus 415: %d", res.StatusCode)
	}
	// multipart ke rute lain ditolak (hanya unggahan yang boleh)
	buf, ct := multipartBody(t, "file", "a.pdf", pdfBytes)
	req := httptest.NewRequest("POST", "/api/chat", buf)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Cookie", cookie)
	if res, _ := e.app.Test(req, -1); res.StatusCode != 415 {
		t.Errorf("multipart ke /api/chat harus 415: %d", res.StatusCode)
	}
	// tanpa field file
	buf, ct = multipartBody(t, "file", "", nil)
	req = httptest.NewRequest("POST", "/api/uploads", buf)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Cookie", cookie)
	if res, _ := e.app.Test(req, -1); res.StatusCode != 400 {
		t.Errorf("tanpa file harus 400: %d", res.StatusCode)
	}
}

func TestUploadLimitsEndpoint(t *testing.T) {
	e := newUploadEnv(t)
	cookie, _, _ := setupAccount(t, e.env)
	res, body := e.do(t, "GET", "/api/uploads/limits", "", cookie, nil)
	types, _ := body["types"].([]any)
	if res.StatusCode != 200 || body["maxBytes"].(float64) != 64<<10 || body["maxFiles"].(float64) != claude.MaxAttachments || len(types) != 4 {
		t.Fatalf("limits: %d %v", res.StatusCode, body)
	}
}

func TestJSONBodyLimitStillApplies(t *testing.T) {
	e := newUploadEnv(t)
	cookie, _, _ := setupAccount(t, e.env)
	// di atas 1 MB (batas JSON) tetapi di bawah batas badan global yang mengikuti ukuran unggahan
	huge := `{"group":"` + strings.Repeat("a", (1<<20)+10_000) + `"}`
	res, _ := e.do(t, "PUT", "/api/sessions/"+strings.Repeat("a", 8)+"-aaaa-4aaa-8aaa-"+strings.Repeat("a", 12)+"/group", huge, cookie, nil)
	if res.StatusCode != 413 {
		t.Fatalf("badan JSON >1 MB harus 413 walau batas unggahan lebih besar: %d", res.StatusCode)
	}
}

func TestInvalidUploadIDsBlocked(t *testing.T) {
	e := newUploadEnv(t)
	cookie, _, _ := setupAccount(t, e.env)
	for _, id := range []string{"..%2f..%2fetc%2fpasswd", "abc", strings.Repeat("g", 32), strings.Repeat("A", 32)} {
		if res, _ := e.do(t, "GET", "/api/uploads/"+id+"/file", "", cookie, nil); res.StatusCode == 200 {
			t.Errorf("id %q tidak boleh menyajikan file", id)
		}
		if res, _ := e.do(t, "DELETE", "/api/uploads/"+id, "", cookie, nil); res.StatusCode == 200 {
			t.Errorf("id %q tidak boleh menghapus", id)
		}
	}
}

func TestChatWithAttachmentOverHTTP(t *testing.T) {
	e := newUploadEnv(t)
	cookie, _, _ := setupAccount(t, e.env)
	_, info := e.upload(t, cookie, "laporan.pdf", pdfBytes)
	id := info["id"].(string)

	body := `{"cwd":"` + e.cwd + `","message":"ringkas","attachments":["` + id + `"]}`
	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	res, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	stream, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(stream), "event: done") {
		t.Fatalf("chat: %d %s", res.StatusCode, stream)
	}
	stdin, _ := os.ReadFile(filepath.Join(e.out, "stdin"))
	if !strings.Contains(string(stdin), "ringkas") || !strings.Contains(string(stdin), "laporan.pdf | pdf | ") {
		t.Fatalf("prompt ke claude: %q", stdin)
	}

	// id yang tidak ada ditolak dengan pesan jelas
	bad := `{"cwd":"` + e.cwd + `","message":"x","attachments":["` + strings.Repeat("0", 32) + `"]}`
	if res, _ := e.do(t, "POST", "/api/chat", bad, cookie, nil); res.StatusCode != 400 {
		t.Fatalf("lampiran tidak ada harus 400: %d", res.StatusCode)
	}
}
