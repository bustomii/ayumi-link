package uploads

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---- fixture ----

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const wNS = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`

func docx(t *testing.T, paras ...string) []byte {
	var body strings.Builder
	for _, p := range paras {
		body.WriteString("<w:p><w:r><w:t>" + p + "</w:t></w:r></w:p>")
	}
	return makeZip(t, map[string]string{
		"[Content_Types].xml": "<Types/>",
		"word/document.xml":   "<w:document " + wNS + "><w:body>" + body.String() + "</w:body></w:document>",
	})
}

func xlsx(t *testing.T, sheet string, rows [][]string) []byte {
	var shared []string
	idx := func(s string) int {
		for i, v := range shared {
			if v == s {
				return i
			}
		}
		shared = append(shared, s)
		return len(shared) - 1
	}
	var sheetXML strings.Builder
	for r, row := range rows {
		fmt.Fprintf(&sheetXML, `<row r="%d">`, r+1)
		for c, cell := range row {
			if cell == "" {
				continue
			}
			ref := fmt.Sprintf("%c%d", 'A'+c, r+1)
			if strings.HasPrefix(cell, "#") { // angka
				fmt.Fprintf(&sheetXML, `<c r="%s"><v>%s</v></c>`, ref, cell[1:])
			} else {
				fmt.Fprintf(&sheetXML, `<c r="%s" t="s"><v>%d</v></c>`, ref, idx(cell))
			}
		}
		sheetXML.WriteString("</row>")
	}
	var ss strings.Builder
	for _, s := range shared {
		ss.WriteString("<si><t>" + s + "</t></si>")
	}
	return makeZip(t, map[string]string{
		"[Content_Types].xml": "<Types/>",
		"xl/workbook.xml": `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
			`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>` +
			`<sheet name="` + sheet + `" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/sharedStrings.xml":       "<sst>" + ss.String() + "</sst>",
		"xl/worksheets/sheet1.xml":   "<worksheet><sheetData>" + sheetXML.String() + "</sheetData></worksheet>",
	})
}

var (
	png  = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...)
	jpg  = append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{1}, 32)...)
	gif  = append([]byte("GIF89a"), bytes.Repeat([]byte{0}, 32)...)
	webp = append(append([]byte("RIFF\x24\x00\x00\x00WEBP"), "VP8 "...), bytes.Repeat([]byte{0}, 32)...)
	pdf  = []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF\n")
	csv  = []byte("nama,umur\nAni,30\nBudi,25\n")
)

func newMgr(t *testing.T) *Manager {
	t.Helper()
	return New(t.TempDir(), Options{MaxBytes: 1 << 20, MaxTotal: 8 << 20, TTL: time.Hour})
}

func save(m *Manager, name string, data []byte) (*Info, error) {
	return m.Save(name, bytes.NewReader(data))
}

func wantErr(t *testing.T, err error, code string) {
	t.Helper()
	e, ok := err.(*Error)
	if !ok || e.Code != code {
		t.Fatalf("seharusnya error %q, dapat %v", code, err)
	}
}

// ---- diterima ----

func TestAcceptsAllowedTypes(t *testing.T) {
	m := newMgr(t)
	cases := []struct {
		name string
		data []byte
		kind Kind
	}{
		{"foto.png", png, KindImage},
		{"foto.JPG", jpg, KindImage},
		{"foto.jpeg", jpg, KindImage},
		{"anim.gif", gif, KindImage},
		{"gambar.webp", webp, KindImage},
		{"laporan.pdf", pdf, KindPDF},
		{"data.csv", csv, KindExcel},
		{"data.xlsx", xlsx(t, "Data", [][]string{{"a"}}), KindExcel},
		{"surat.docx", docx(t, "Halo"), KindWord},
	}
	for _, c := range cases {
		info, err := save(m, c.name, c.data)
		if err != nil {
			t.Errorf("%s harus diterima: %v", c.name, err)
			continue
		}
		if info.Kind != c.kind || info.Size != int64(len(c.data)) || !ValidID(info.ID) {
			t.Errorf("%s: info salah %+v", c.name, info)
		}
		if _, err := os.Stat(m.Path(info)); err != nil {
			t.Errorf("%s: file tidak tersimpan: %v", c.name, err)
		}
	}
}

// ---- ditolak ----

func TestRejectsEverythingElse(t *testing.T) {
	m := newMgr(t)
	exe := append([]byte("MZ"), bytes.Repeat([]byte{0x90}, 64)...)
	cases := []struct {
		name, why string
		data      []byte
	}{
		{"virus.exe", "ekstensi tidak diizinkan", exe},
		{"skrip.sh", "skrip", []byte("#!/bin/sh\nrm -rf ~\n")},
		{"halaman.html", "html", []byte("<script>alert(1)</script>")},
		{"gambar.svg", "svg bisa memuat skrip", []byte("<svg onload=alert(1)/>")},
		{"arsip.zip", "zip umum", makeZip(t, map[string]string{"a.txt": "x"})},
		{"catatan.txt", "teks biasa", []byte("halo")},
		{"video.mp4", "video", []byte("\x00\x00\x00\x18ftypmp42")},
		{"tanpa-ekstensi", "tanpa ekstensi", pdf},
		{"makro.docm", "docm", docx(t, "x")},
		{"makro.xlsm", "xlsm", xlsx(t, "A", [][]string{{"x"}})},
		{"kode.js", "js", []byte("alert(1)")},
		{"model.PY", "py", []byte("print(1)")},
	}
	for _, c := range cases {
		if _, err := save(m, c.name, c.data); err == nil {
			t.Errorf("%s (%s) seharusnya ditolak", c.name, c.why)
		} else {
			wantErr(t, err, "type")
		}
	}
}

func TestLegacyOfficeFormatsGetHelpfulMessage(t *testing.T) {
	m := newMgr(t)
	for ext, hint := range map[string]string{".xls": ".xlsx", ".doc": ".docx"} {
		_, err := save(m, "lama"+ext, []byte("\xD0\xCF\x11\xE0"))
		wantErr(t, err, "type")
		if !strings.Contains(err.Error(), hint) {
			t.Errorf("pesan %s harus menyebut %s: %v", ext, hint, err)
		}
	}
}

func TestContentMustMatchExtension(t *testing.T) {
	m := newMgr(t)
	exe := append([]byte("MZ"), bytes.Repeat([]byte{0x90}, 64)...)
	mism := []struct {
		name string
		data []byte
	}{
		{"menyamar.pdf", exe},                        // exe berkedok pdf
		{"menyamar.png", pdf},                        // pdf berkedok png
		{"menyamar.jpg", png},                        // png berkedok jpg
		{"menyamar.docx", pdf},                       // pdf berkedok docx
		{"menyamar.xlsx", exe},                       // exe berkedok xlsx
		{"menyamar.gif", []byte("<html>")},           // html berkedok gif
		{"menyamar.webp", []byte("RIFF....WAVEfmt")}, // wav berkedok webp
	}
	for _, c := range mism {
		_, err := save(m, c.name, c.data)
		wantErr(t, err, "type")
	}
}

func TestOfficeStructureIsVerified(t *testing.T) {
	m := newMgr(t)
	// xlsx diberi ekstensi docx dan sebaliknya
	_, err := save(m, "salah.docx", xlsx(t, "A", [][]string{{"x"}}))
	wantErr(t, err, "invalid")
	_, err = save(m, "salah.xlsx", docx(t, "x"))
	wantErr(t, err, "invalid")
	// zip kosong struktur
	_, err = save(m, "kosong.docx", makeZip(t, map[string]string{"readme.txt": "x"}))
	wantErr(t, err, "invalid")
	// makro VBA
	_, err = save(m, "makro.docx", makeZip(t, map[string]string{
		"[Content_Types].xml": "<Types/>", "word/document.xml": "<w:document " + wNS + "/>", "word/vbaProject.bin": "x",
	}))
	wantErr(t, err, "invalid")
	if !strings.Contains(err.Error(), "makro") {
		t.Errorf("pesan harus menyebut makro: %v", err)
	}
	// zip rusak
	_, err = save(m, "rusak.docx", []byte("PK\x03\x04 bukan zip sungguhan"))
	wantErr(t, err, "invalid")
}

func TestCSVValidation(t *testing.T) {
	m := newMgr(t)
	_, err := save(m, "biner.csv", append([]byte("a,b\n"), 0, 1, 2))
	wantErr(t, err, "type")
	_, err = save(m, "latin1.csv", []byte("nama\n\xE9\xE8\n")) // bukan UTF-8
	wantErr(t, err, "type")
	if _, err := save(m, "bom.csv", append([]byte{0xEF, 0xBB, 0xBF}, csv...)); err != nil {
		t.Errorf("CSV UTF-8 dengan BOM harus diterima: %v", err)
	}
	if _, err := save(m, "unicode.csv", []byte("nama\nÉmilie 日本語\n")); err != nil {
		t.Errorf("CSV unicode harus diterima: %v", err)
	}
}

func TestSizeEmptyAndQuota(t *testing.T) {
	m := New(t.TempDir(), Options{MaxBytes: 1024, MaxTotal: 3000})
	_, err := save(m, "kosong.pdf", nil)
	wantErr(t, err, "empty")
	big := append([]byte("%PDF-"), bytes.Repeat([]byte("x"), 2000)...)
	_, err = save(m, "besar.pdf", big)
	wantErr(t, err, "size")
	// tidak boleh meninggalkan sisa file setelah ditolak
	entries, _ := os.ReadDir(m.Root())
	if len(entries) != 0 {
		t.Errorf("sisa unggahan gagal harus dibersihkan: %v", entries)
	}

	small := append([]byte("%PDF-"), bytes.Repeat([]byte("x"), 900)...)
	for i := 0; i < 2; i++ {
		if _, err := save(m, fmt.Sprintf("a%d.pdf", i), small); err != nil {
			t.Fatalf("unggahan %d: %v", i, err)
		}
	}
	_, err = save(m, "c.pdf", small) // total + MaxBytes melewati kuota
	wantErr(t, err, "quota")
}

// ---- keamanan nama & id ----

func TestFilenameIsSanitized(t *testing.T) {
	m := newMgr(t)
	cases := map[string]string{
		"../../etc/passwd.pdf":            "passwd.pdf",
		`..\..\windows\sys.pdf`:           "sys.pdf",
		"/abs/path/laporan.pdf":           "laporan.pdf",
		"na\x00me\x07.pdf":                "name.pdf",
		"a:b*c?.pdf":                      "a_b_c_.pdf",
		"  ...rahasia.PDF  ":              "rahasia.pdf",
		"Laporan Keuangan Ü.pdf":          "Laporan Keuangan Ü.pdf",
		".pdf":                            "file.pdf",
		strings.Repeat("a", 300) + ".pdf": strings.Repeat("a", 100) + ".pdf",
	}
	for in, want := range cases {
		if got := SafeName(in); got != want {
			t.Errorf("SafeName(%q) = %q; want %q", in, got, want)
		}
	}
	info, err := save(m, "../../etc/passwd.pdf", pdf)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(m.Root(), m.Path(info))
	if err != nil || strings.HasPrefix(rel, "..") || filepath.Dir(rel) != info.ID {
		t.Fatalf("file harus tetap di dalam folder lampirannya: %q %v", rel, err)
	}
}

func TestIDValidationBlocksTraversal(t *testing.T) {
	m := newMgr(t)
	for _, id := range []string{"", "..", "../x", "abc", strings.Repeat("g", 32), strings.Repeat("A", 32), strings.Repeat("a", 31), strings.Repeat("a", 33)} {
		if ValidID(id) {
			t.Errorf("%q tidak boleh valid", id)
		}
		if _, err := m.Get(id); err == nil {
			t.Errorf("Get(%q) harus gagal", id)
		}
		if err := m.Delete(id); err == nil {
			t.Errorf("Delete(%q) harus gagal", id)
		}
	}
}

func TestGetDeleteRoundTrip(t *testing.T) {
	m := newMgr(t)
	info, _ := save(m, "surat.docx", docx(t, "Halo dunia"))
	got, err := m.Get(info.ID)
	if err != nil || got.Name != "surat.docx" || got.Kind != KindWord || got.TextFile == "" {
		t.Fatalf("Get: %+v %v", got, err)
	}
	if err := m.Delete(info.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(info.ID); err == nil {
		t.Fatal("setelah Delete harus tidak ada")
	}
}

func TestCleanupByTTL(t *testing.T) {
	m := newMgr(t)
	old, _ := save(m, "lama.pdf", pdf)
	fresh, _ := save(m, "baru.pdf", pdf)
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(filepath.Join(m.dir(old.ID), "meta.json"), past, past); err != nil {
		t.Fatal(err)
	}
	if n := m.Cleanup(); n != 1 {
		t.Fatalf("seharusnya menghapus 1, dapat %d", n)
	}
	if _, err := m.Get(old.ID); err == nil {
		t.Error("lampiran lama harus terhapus")
	}
	if _, err := m.Get(fresh.ID); err != nil {
		t.Errorf("lampiran baru harus tetap ada: %v", err)
	}
}

// ---- ekstraksi teks ----

func TestDocxTextExtraction(t *testing.T) {
	m := newMgr(t)
	info, err := save(m, "surat.docx", docx(t, "Kepada Yth.", "Isi surat penting", "Hormat kami"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(m.TextPath(info))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != "Kepada Yth.\nIsi surat penting\nHormat kami\n" {
		t.Fatalf("teks docx = %q", got)
	}
}

func TestXlsxTextExtraction(t *testing.T) {
	m := newMgr(t)
	rows := [][]string{
		{"Nama", "Umur", "Kota"},
		{"Ani", "#30", "Jakarta"},
		{"Budi", "", "Bandung"}, // sel kosong di tengah harus mempertahankan kolom
	}
	info, err := save(m, "data.xlsx", xlsx(t, "Penduduk", rows))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(m.TextPath(info))
	want := "=== Lembar: Penduduk ===\nNama\tUmur\tKota\nAni\t30\tJakarta\nBudi\t\tBandung\n\n"
	if string(b) != want {
		t.Fatalf("teks xlsx:\n%q\nseharusnya:\n%q", b, want)
	}
}

func TestXlsxRowLimit(t *testing.T) {
	m := New(t.TempDir(), Options{MaxBytes: 4 << 20})
	var rows [][]string
	for i := 0; i < maxSheetRows+50; i++ {
		rows = append(rows, []string{fmt.Sprintf("baris%d", i)})
	}
	info, err := save(m, "besar.xlsx", xlsx(t, "S", rows))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(m.TextPath(info))
	if !strings.Contains(string(b), "dipotong") || strings.Contains(string(b), fmt.Sprintf("baris%d", maxSheetRows+10)) {
		t.Fatal("lembar yang terlalu panjang harus dipotong dengan catatan")
	}
}

func TestExtractedTextIsNotUserNamedPath(t *testing.T) {
	m := newMgr(t)
	info, _ := save(m, "../../x.docx", docx(t, "a"))
	rel, _ := filepath.Rel(m.Root(), m.TextPath(info))
	if strings.HasPrefix(rel, "..") || filepath.Dir(rel) != info.ID {
		t.Fatalf("teks ekstraksi harus di dalam folder lampiran: %q", rel)
	}
}
