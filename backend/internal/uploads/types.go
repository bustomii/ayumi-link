// Package uploads menerima lampiran chat. Hanya gambar, PDF, Excel (.xlsx/.csv) dan Word (.docx)
// yang diizinkan, divalidasi dari ISI file (magic bytes dan struktur), bukan dari ekstensi saja.
package uploads

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Kind string

const (
	KindImage Kind = "image"
	KindPDF   Kind = "pdf"
	KindExcel Kind = "excel"
	KindWord  Kind = "word"
)

// TypeSpec menjelaskan satu jenis lampiran yang diizinkan (ditampilkan ke UI).
type TypeSpec struct {
	Kind  Kind     `json:"kind"`
	Label string   `json:"label"`
	Exts  []string `json:"exts"`
}

// Allowed adalah daftar lengkap yang boleh dikirim. Selain ini ditolak.
func Allowed() []TypeSpec {
	return []TypeSpec{
		{KindImage, "Gambar / foto", []string{".png", ".jpg", ".jpeg", ".gif", ".webp"}},
		{KindPDF, "PDF", []string{".pdf"}},
		{KindExcel, "Excel", []string{".xlsx", ".csv"}},
		{KindWord, "Dokumen Word", []string{".docx"}},
	}
}

// rule: jenis, MIME, dan bentuk isi yang harus cocok untuk sebuah ekstensi.
type rule struct {
	kind Kind
	mime string
	body string // png | jpeg | gif | webp | pdf | zip | text
}

var rules = map[string]rule{
	".png":  {KindImage, "image/png", "png"},
	".jpg":  {KindImage, "image/jpeg", "jpeg"},
	".jpeg": {KindImage, "image/jpeg", "jpeg"},
	".gif":  {KindImage, "image/gif", "gif"},
	".webp": {KindImage, "image/webp", "webp"},
	".pdf":  {KindPDF, "application/pdf", "pdf"},
	".xlsx": {KindExcel, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "zip"},
	".csv":  {KindExcel, "text/csv; charset=utf-8", "text"},
	".docx": {KindWord, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "zip"},
}

// legacy: format lama yang sengaja tidak didukung, dengan pesan yang membantu.
var legacy = map[string]string{
	".xls": "format Excel lama (.xls) tidak didukung; simpan ulang sebagai .xlsx",
	".doc": "format Word lama (.doc) tidak didukung; simpan ulang sebagai .docx",
}

// Error adalah penolakan lampiran yang aman ditampilkan ke pengguna.
type Error struct {
	Status  int
	Code    string // type | size | empty | quota | invalid | notfound | count
	Message string
}

func (e *Error) Error() string { return e.Message }

func errType(msg string) *Error    { return &Error{http.StatusUnsupportedMediaType, "type", msg} }
func errInvalid(msg string) *Error { return &Error{http.StatusBadRequest, "invalid", msg} }

var (
	ErrEmpty    = &Error{http.StatusBadRequest, "empty", "file kosong"}
	ErrNotFound = &Error{http.StatusNotFound, "notfound", "lampiran tidak ditemukan"}
)

func errSize(limitMB int64) *Error {
	return &Error{http.StatusRequestEntityTooLarge, "size", "file terlalu besar (maksimal " + strconv.FormatInt(limitMB, 10) + " MB)"}
}

// Info adalah lampiran yang sudah tersimpan.
type Info struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Kind    Kind      `json:"kind"`
	Size    int64     `json:"size"`
	Mime    string    `json:"mime"`
	Created time.Time `json:"createdAt"`
	// TextFile: nama file teks hasil ekstraksi (Word/Excel); kosong bila tidak ada.
	TextFile string `json:"-"`
}

// IsAllowedExt memeriksa ekstensi (huruf kecil, dengan titik).
func IsAllowedExt(ext string) bool { _, ok := rules[strings.ToLower(ext)]; return ok }

// MsgUnsupported: pesan bila jenis file tidak ada di daftar yang diizinkan.
const MsgUnsupported = "jenis file tidak diizinkan; hanya gambar, PDF, Excel (.xlsx, .csv), dan Word (.docx)"
