package claude

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"ayumi-link/internal/uploads"
)

// MaxAttachments: jumlah lampiran maksimal per pesan.
const MaxAttachments = 5

// Attachment adalah lampiran yang tampil di bubble chat (dibaca balik dari transcript).
type Attachment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

const (
	attachOpen  = "[Lampiran dari pengguna — baca file berikut bila relevan]"
	attachClose = "[/Lampiran]"
	// defaultAttachPrompt dipakai bila pesan dikirim hanya berisi lampiran.
	defaultAttachPrompt = "Tolong baca dan rangkum lampiran ini."
)

// attachmentFooter menyusun blok lampiran yang ditambahkan ke pesan untuk Claude. Untuk Word/Excel
// yang dipakai adalah teks hasil ekstraksi; gambar, PDF, dan CSV dibaca langsung dari filenya.
func attachmentFooter(infos []*uploads.Info, m *uploads.Manager) string {
	if len(infos) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n" + attachOpen + "\n")
	for i, in := range infos {
		path := m.Path(in)
		if t := m.TextPath(in); t != "" {
			path = t
		}
		fmt.Fprintf(&b, "%d. %s | %s | %s\n", i+1, in.Name, in.Kind, path)
	}
	b.WriteString(attachClose)
	return b.String()
}

var attachBlockRe = regexp.MustCompile(`(?s)\n*` + regexp.QuoteMeta("[Lampiran dari pengguna") + `[^\]]*\]\n(.*?)\n` + regexp.QuoteMeta(attachClose) + `\s*$`)

// SplitAttachments memisahkan blok lampiran dari teks pesan user (dipakai saat membaca transcript).
func SplitAttachments(content string) (string, []Attachment) {
	m := attachBlockRe.FindStringSubmatchIndex(content)
	if m == nil {
		return content, nil
	}
	var atts []Attachment
	for _, line := range strings.Split(content[m[2]:m[3]], "\n") {
		// "1. nama.pdf | pdf | /jalur/uploads/<id>/nama.pdf" (nama tidak memuat '|': SafeName menggantinya)
		_, rest, ok := strings.Cut(line, ". ")
		if !ok {
			continue
		}
		parts := strings.Split(rest, " | ")
		if len(parts) < 3 {
			continue
		}
		path := parts[len(parts)-1]
		id := filepath.Base(filepath.Dir(path))
		if !uploads.ValidID(id) {
			id = ""
		}
		atts = append(atts, Attachment{ID: id, Name: strings.Join(parts[:len(parts)-2], " | "), Kind: parts[len(parts)-2]})
	}
	return strings.TrimSpace(content[:m[0]]), atts
}

// resolveAttachments memuat info lampiran dari daftar id; error jika ada yang tidak valid.
func (r *Runner) resolveAttachments(ids []string) ([]*uploads.Info, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if r.Uploads == nil {
		return nil, badRequest("lampiran tidak didukung di server ini")
	}
	if len(ids) > MaxAttachments {
		return nil, badRequest(fmt.Sprintf("maksimal %d lampiran per pesan", MaxAttachments))
	}
	seen := map[string]bool{}
	var out []*uploads.Info
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		in, err := r.Uploads.Get(id)
		if err != nil {
			return nil, badRequest("lampiran tidak ditemukan atau sudah kedaluwarsa; unggah ulang")
		}
		out = append(out, in)
	}
	return out, nil
}
