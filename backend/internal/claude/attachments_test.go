package claude

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
	"ayumi-link/internal/uploads"
)

var pdfBytes = []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n")

func TestSplitAttachmentsRoundTrip(t *testing.T) {
	id1, id2 := strings.Repeat("a", 32), strings.Repeat("b", 32)
	footer := "\n\n" + attachOpen + "\n" +
		"1. laporan keuangan.pdf | pdf | /data/uploads/" + id1 + "/laporan keuangan.pdf\n" +
		"2. data.xlsx | excel | /data/uploads/" + id2 + "/data.xlsx.txt\n" +
		attachClose
	text, atts := SplitAttachments("Tolong ringkas ini" + footer)
	if text != "Tolong ringkas ini" {
		t.Fatalf("teks = %q", text)
	}
	if len(atts) != 2 || atts[0] != (Attachment{ID: id1, Name: "laporan keuangan.pdf", Kind: "pdf"}) ||
		atts[1] != (Attachment{ID: id2, Name: "data.xlsx", Kind: "excel"}) {
		t.Fatalf("lampiran = %+v", atts)
	}
}

func TestSplitAttachmentsLeavesNormalTextAlone(t *testing.T) {
	for _, in := range []string{
		"pesan biasa",
		"mengandung [Lampiran dari pengguna tapi tanpa penutup",
		"[Lampiran dari pengguna — x]\n1. a | pdf | /x/y\nteks setelahnya\n" + attachClose + "\nlanjutan", // bukan di akhir pesan
	} {
		if text, atts := SplitAttachments(in); text != in || len(atts) != 0 {
			t.Errorf("%q seharusnya tidak diubah: %q %v", in, text, atts)
		}
	}
}

func TestSplitAttachmentsRejectsForgedIDs(t *testing.T) {
	footer := "\n\n" + attachOpen + "\n1. x.pdf | pdf | /etc/passwd\n" + attachClose
	_, atts := SplitAttachments("hi" + footer)
	if len(atts) != 1 || atts[0].ID != "" {
		t.Fatalf("id yang bukan 32 hex harus dikosongkan: %+v", atts)
	}
}

func TestTranscriptShowsAttachmentsAsChips(t *testing.T) {
	home := t.TempDir()
	id := strings.Repeat("c", 32)
	content := `"Lihat ini\n\n` + attachOpen + `\n1. foto.png | image | /data/uploads/` + id + `/foto.png\n` + attachClose + `"`
	only := `"` + attachOpen + `\n1. a.pdf | pdf | /d/uploads/` + id + `/a.pdf\n` + attachClose + `"`
	p := writeTranscript(t, home, sid,
		line("user", "u1", content),
		line("assistant", "a1", `[{"type":"text","text":"ok"}]`),
		line("user", "u2", only),
	)
	msgs, err := LoadMessages(p)
	if err != nil || len(msgs) != 3 {
		t.Fatalf("pesan: %d %v", len(msgs), err)
	}
	if msgs[0].Content != "Lihat ini" || len(msgs[0].Attachments) != 1 || msgs[0].Attachments[0].ID != id || msgs[0].Attachments[0].Kind != "image" {
		t.Fatalf("pesan 0: %+v", msgs[0])
	}
	if msgs[2].Content != "" || len(msgs[2].Attachments) != 1 {
		t.Fatalf("pesan hanya-lampiran harus tetap tampil: %+v", msgs[2])
	}
	if title := firstUserPrompt(p); title != "Lihat ini" {
		t.Fatalf("judul session tidak boleh memuat blok lampiran: %q", title)
	}
}

// ---- runner dengan CLI palsu ----

type fakeEnv struct {
	runner *Runner
	up     *uploads.Manager
	out    string
	cwd    string
}

func newFake(t *testing.T) *fakeEnv {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "claude")
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + out + "/args\ncat > " + out + "/stdin\n" +
		`echo '{"type":"result","is_error":false,"result":"ok"}'` + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(dir, "proj")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	st := store.New(filepath.Join(dir, "data"))
	svc := &Service{Home: filepath.Join(dir, "home"), DesktopDir: dir, Store: st}
	up := uploads.New(filepath.Join(dir, "uploads"), uploads.Options{MaxBytes: 1 << 20})
	r := &Runner{Svc: svc, Browser: fsbrowse.New([]string{dir}), Store: st, Bin: script, DefaultPerm: "auto", Uploads: up}
	return &fakeEnv{runner: r, up: up, out: out, cwd: cwd}
}

func (f *fakeEnv) run(t *testing.T, o ChatOptions) error {
	t.Helper()
	o.Cwd = f.cwd
	ch, err := f.runner.Start(context.Background(), o)
	if err != nil {
		return err
	}
	deadline := time.After(10 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return nil
			}
		case <-deadline:
			t.Fatal("runner tidak selesai")
		}
	}
}

func (f *fakeEnv) read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.out, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRunnerPassesAttachmentsToClaude(t *testing.T) {
	f := newFake(t)
	pdf, err := f.up.Save("laporan.pdf", bytes.NewReader(pdfBytes))
	if err != nil {
		t.Fatal(err)
	}
	csv, err := f.up.Save("data.csv", strings.NewReader("a,b\n1,2\n"))
	if err != nil {
		t.Fatal(err)
	}

	if err := f.run(t, ChatOptions{Message: "ringkas ya", Attachments: []string{pdf.ID, csv.ID, pdf.ID}}); err != nil {
		t.Fatal(err)
	}
	stdin, args := f.read(t, "stdin"), f.read(t, "args")

	if !strings.HasPrefix(stdin, "ringkas ya\n\n"+attachOpen) || !strings.HasSuffix(stdin, attachClose) {
		t.Fatalf("prompt = %q", stdin)
	}
	if !strings.Contains(stdin, "1. laporan.pdf | pdf | "+f.up.Path(pdf)) || !strings.Contains(stdin, "2. data.csv | excel | "+f.up.Path(csv)) {
		t.Fatalf("daftar lampiran salah: %q", stdin)
	}
	if strings.Count(stdin, "laporan.pdf |") != 1 {
		t.Errorf("lampiran duplikat harus digabung: %q", stdin)
	}
	// folder lampiran diizinkan lewat --add-dir
	for _, dir := range []string{f.up.Dir(pdf), f.up.Dir(csv)} {
		if !strings.Contains(args, dir) {
			t.Errorf("--add-dir harus memuat %s: %q", dir, args)
		}
	}
	if !strings.Contains(args, "--add-dir") {
		t.Errorf("args tanpa --add-dir: %q", args)
	}
}

func TestRunnerUsesExtractedTextForOffice(t *testing.T) {
	f := newFake(t)
	docxBytes := docxFixture(t, "Isi dokumen penting")
	d, err := f.up.Save("surat.docx", bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.run(t, ChatOptions{Message: "baca", Attachments: []string{d.ID}}); err != nil {
		t.Fatal(err)
	}
	stdin := f.read(t, "stdin")
	if !strings.Contains(stdin, "surat.docx | word | "+f.up.TextPath(d)) || !strings.HasSuffix(f.up.TextPath(d), "surat.docx.txt") {
		t.Fatalf("Word harus memakai teks hasil ekstraksi: %q", stdin)
	}
	if b, _ := os.ReadFile(f.up.TextPath(d)); !strings.Contains(string(b), "Isi dokumen penting") {
		t.Fatalf("isi teks ekstraksi: %q", b)
	}
}

func TestRunnerAttachmentOnlyMessage(t *testing.T) {
	f := newFake(t)
	p, _ := f.up.Save("a.pdf", bytes.NewReader(pdfBytes))
	if err := f.run(t, ChatOptions{Message: "   ", Attachments: []string{p.ID}}); err != nil {
		t.Fatalf("pesan kosong dengan lampiran harus boleh: %v", err)
	}
	if !strings.HasPrefix(f.read(t, "stdin"), defaultAttachPrompt) {
		t.Fatalf("prompt bawaan tidak dipakai: %q", f.read(t, "stdin"))
	}
}

func TestRunnerRejectsBadAttachments(t *testing.T) {
	f := newFake(t)
	p, _ := f.up.Save("a.pdf", bytes.NewReader(pdfBytes))

	if err := f.run(t, ChatOptions{Message: "", Attachments: nil}); err == nil {
		t.Error("pesan kosong tanpa lampiran harus ditolak")
	}
	if err := f.run(t, ChatOptions{Message: "x", Attachments: []string{strings.Repeat("0", 32)}}); err == nil {
		t.Error("id lampiran tidak ada harus ditolak")
	}
	if err := f.run(t, ChatOptions{Message: "x", Attachments: []string{"../../etc/passwd"}}); err == nil {
		t.Error("id berisi path harus ditolak")
	}
	many := make([]string, MaxAttachments+1)
	for i := range many {
		many[i] = p.ID
	}
	// id sama dihitung satu, tapi jumlah mentah > batas tetap ditolak
	if err := f.run(t, ChatOptions{Message: "x", Attachments: many}); err == nil {
		t.Error("melebihi jumlah lampiran maksimal harus ditolak")
	}
	f.runner.Uploads = nil
	if err := f.run(t, ChatOptions{Message: "x", Attachments: []string{p.ID}}); err == nil {
		t.Error("tanpa fitur lampiran harus ditolak")
	}
}

// docxFixture membuat .docx minimal yang sah.
func docxFixture(t *testing.T, para string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	add("[Content_Types].xml", "<Types/>")
	add("word/document.xml", `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>`+para+`</w:t></w:r></w:p></w:body></w:document>`)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
