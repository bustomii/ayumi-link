package uploads

import (
	"archive/zip"
	"bytes"
	"unicode/utf8"
)

// sniff mengenali bentuk isi dari byte awal file.
func sniff(head []byte) string {
	switch {
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	case bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF}):
		return "jpeg"
	case bytes.HasPrefix(head, []byte("GIF87a")), bytes.HasPrefix(head, []byte("GIF89a")):
		return "gif"
	case len(head) >= 12 && bytes.Equal(head[0:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		return "webp"
	case bytes.Contains(head[:min(len(head), 1024)], []byte("%PDF-")):
		return "pdf"
	case bytes.HasPrefix(head, []byte("PK\x03\x04")):
		return "zip"
	}
	return ""
}

// isText: UTF-8 valid tanpa byte NUL (untuk CSV). BOM diizinkan.
func isText(data []byte) bool {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	return bytes.IndexByte(data, 0) < 0 && utf8.Valid(data)
}

const (
	maxZipEntries      = 10000
	maxZipUncompressed = 500 << 20 // 500 MB total terurai: batas anti zip-bomb
)

// checkOOXML memastikan zip memang dokumen Office yang sah dan tidak berbahaya.
// part: bagian wajib ("word/document.xml" untuk .docx, "xl/workbook.xml" untuk .xlsx).
func checkOOXML(zr *zip.Reader, part string) *Error {
	if len(zr.File) == 0 || len(zr.File) > maxZipEntries {
		return errInvalid("struktur dokumen tidak valid")
	}
	var total uint64
	have := map[string]bool{}
	for _, f := range zr.File {
		total += f.UncompressedSize64
		if total > maxZipUncompressed {
			return errInvalid("dokumen terlalu besar setelah diekstrak")
		}
		have[f.Name] = true
		// Makro VBA tidak boleh ada di .docx/.xlsx yang sah.
		if f.Name == "word/vbaProject.bin" || f.Name == "xl/vbaProject.bin" {
			return errInvalid("dokumen berisi makro dan tidak diizinkan")
		}
	}
	if !have["[Content_Types].xml"] || !have[part] {
		return errInvalid("bukan dokumen Office yang valid (atau terenkripsi/rusak)")
	}
	return nil
}
