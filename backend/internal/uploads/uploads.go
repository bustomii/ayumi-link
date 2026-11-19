package uploads

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Options mengatur batas unggahan.
type Options struct {
	MaxBytes int64         // ukuran maksimal per file
	MaxTotal int64         // total semua lampiran tersimpan
	TTL      time.Duration // lampiran lebih tua dari ini dihapus oleh Cleanup
}

type Manager struct {
	root string
	opt  Options
	now  func() time.Time
}

var idRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// ValidID: id lampiran = 32 karakter hex.
func ValidID(id string) bool { return idRe.MatchString(id) }

func New(root string, opt Options) *Manager {
	if opt.MaxBytes <= 0 {
		opt.MaxBytes = 20 << 20
	}
	if opt.MaxTotal <= 0 {
		opt.MaxTotal = 2 << 30
	}
	return &Manager{root: root, opt: opt, now: time.Now}
}

func (m *Manager) Root() string         { return m.root }
func (m *Manager) MaxBytes() int64      { return m.opt.MaxBytes }
func (m *Manager) dir(id string) string { return filepath.Join(m.root, id) }

// SafeName membersihkan nama file dari klien: tanpa path, tanpa karakter kontrol,
// maksimal 100 karakter dengan ekstensi dipertahankan. Hasilnya aman dijadikan nama file.
func SafeName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	name = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsControl(r):
			return -1
		case r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|':
			return '_'
		}
		return r
	}, name)
	name = strings.TrimRight(strings.TrimSpace(name), ". ")
	ext := filepath.Ext(name)
	base := strings.TrimLeft(strings.TrimSuffix(name, ext), ". ")
	if r := []rune(base); len(r) > 100 {
		base = string(r[:100])
	}
	if strings.TrimSpace(base) == "" {
		base = "file"
	}
	return base + strings.ToLower(ext)
}

// Save memvalidasi dan menyimpan satu lampiran. Semua pemeriksaan memakai ISI file.
func (m *Manager) Save(filename string, r io.Reader) (*Info, error) {
	name := SafeName(filename)
	ext := strings.ToLower(filepath.Ext(name))
	rl, ok := rules[ext]
	if !ok {
		if msg, isLegacy := legacy[ext]; isLegacy {
			return nil, errType(msg)
		}
		return nil, errType(MsgUnsupported)
	}
	if m.totalSize()+m.opt.MaxBytes > m.opt.MaxTotal {
		return nil, &Error{http.StatusInsufficientStorage, "quota", "penyimpanan lampiran penuh; hapus lampiran lama atau coba lagi nanti"}
	}

	var idb [16]byte
	if _, err := rand.Read(idb[:]); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(idb[:])
	dir := m.dir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(dir)
		}
	}()

	final := filepath.Join(dir, name)
	f, err := os.OpenFile(final, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	n, err := io.Copy(f, io.LimitReader(r, m.opt.MaxBytes+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	if n > m.opt.MaxBytes {
		return nil, errSize(m.opt.MaxBytes >> 20)
	}
	if n == 0 {
		return nil, ErrEmpty
	}

	// Isi harus cocok dengan ekstensi.
	data, err := os.ReadFile(final)
	if err != nil {
		return nil, err
	}
	var textFile string
	switch rl.body {
	case "text":
		if !isText(data) {
			return nil, errType("isi file bukan teks CSV yang valid (UTF-8)")
		}
	case "zip":
		if sniff(data[:min(len(data), 4096)]) != "zip" {
			return nil, errType("isi file tidak sesuai dengan ekstensinya " + ext)
		}
		zr, zerr := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if zerr != nil {
			return nil, errInvalid("file rusak atau terenkripsi")
		}
		part := "word/document.xml"
		if ext == ".xlsx" {
			part = "xl/workbook.xml"
		}
		if e := checkOOXML(zr, part); e != nil {
			return nil, e
		}
		var text string
		var xerr error
		if ext == ".xlsx" {
			text, xerr = xlsxText(zr)
		} else {
			text, xerr = docxText(zr)
		}
		if xerr != nil {
			return nil, errInvalid(xerr.Error())
		}
		textFile = name + ".txt"
		if err := os.WriteFile(filepath.Join(dir, textFile), []byte(text), 0o600); err != nil {
			return nil, err
		}
	default:
		if got := sniff(data[:min(len(data), 4096)]); got != rl.body {
			return nil, errType("isi file tidak sesuai dengan ekstensinya " + ext)
		}
	}

	info := &Info{ID: id, Name: name, Kind: rl.kind, Size: n, Mime: rl.mime, Created: m.now().UTC(), TextFile: textFile}
	meta, _ := json.Marshal(struct {
		*Info
		TextFile string `json:"textFile,omitempty"`
	}{info, textFile})
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), meta, 0o600); err != nil {
		return nil, err
	}
	cleanup = false
	return info, nil
}

// Get membaca info lampiran dari disk; ErrNotFound bila id tidak valid atau tidak ada.
func (m *Manager) Get(id string) (*Info, error) {
	if !ValidID(id) {
		return nil, ErrNotFound
	}
	b, err := os.ReadFile(filepath.Join(m.dir(id), "meta.json"))
	if err != nil {
		return nil, ErrNotFound
	}
	var meta struct {
		Info
		TextFile string `json:"textFile"`
	}
	if json.Unmarshal(b, &meta) != nil || meta.ID != id {
		return nil, ErrNotFound
	}
	info := meta.Info
	info.TextFile = meta.TextFile
	// Nama dari meta tidak dipercaya mentah-mentah.
	info.Name = SafeName(info.Name)
	return &info, nil
}

// Path: lokasi file asli. TextPath: lokasi teks hasil ekstraksi ("" bila tidak ada).
func (m *Manager) Path(i *Info) string { return filepath.Join(m.dir(i.ID), i.Name) }
func (m *Manager) TextPath(i *Info) string {
	if i.TextFile == "" {
		return ""
	}
	return filepath.Join(m.dir(i.ID), SafeName(i.TextFile))
}

// Dir: folder lampiran (diberikan ke Claude lewat --add-dir).
func (m *Manager) Dir(i *Info) string { return m.dir(i.ID) }

func (m *Manager) Delete(id string) error {
	if !ValidID(id) {
		return ErrNotFound
	}
	if _, err := os.Stat(m.dir(id)); err != nil {
		return ErrNotFound
	}
	return os.RemoveAll(m.dir(id))
}

// Cleanup menghapus lampiran yang lebih tua dari TTL. Mengembalikan jumlah yang dihapus.
func (m *Manager) Cleanup() int {
	if m.opt.TTL <= 0 {
		return 0
	}
	entries, err := os.ReadDir(m.root)
	if err != nil {
		return 0
	}
	removed := 0
	cutoff := m.now().Add(-m.opt.TTL)
	for _, e := range entries {
		if !e.IsDir() || !ValidID(e.Name()) {
			continue
		}
		st, err := os.Stat(filepath.Join(m.root, e.Name(), "meta.json"))
		if err != nil {
			st, err = os.Stat(filepath.Join(m.root, e.Name())) // sisa unggahan gagal
			if err != nil {
				continue
			}
		}
		if st.ModTime().Before(cutoff) {
			if os.RemoveAll(filepath.Join(m.root, e.Name())) == nil {
				removed++
			}
		}
	}
	return removed
}

func (m *Manager) totalSize() int64 {
	var total int64
	_ = filepath.WalkDir(m.root, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, e := d.Info(); e == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}
