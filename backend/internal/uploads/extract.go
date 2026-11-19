package uploads

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

const (
	maxPartBytes   = 64 << 20 // satu bagian XML maksimal 64 MB setelah diurai
	maxTextChars   = 400_000  // teks hasil ekstraksi maksimal (dipotong dengan catatan)
	maxSheetRows   = 3000
	maxSheets      = 50
	truncatedNote  = "\n\n[... dipotong: dokumen terlalu panjang ...]\n"
	nsRelationship = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
)

func readPart(zr *zip.Reader, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, maxPartBytes))
	}
	return nil, fmt.Errorf("bagian %s tidak ada", name)
}

type textBuilder struct {
	sb        strings.Builder
	truncated bool
}

func (b *textBuilder) write(s string) {
	if b.truncated {
		return
	}
	if b.sb.Len()+len(s) > maxTextChars {
		b.sb.WriteString(truncatedNote)
		b.truncated = true
		return
	}
	b.sb.WriteString(s)
}

// docxText mengambil teks dari word/document.xml: paragraf per baris, sel tabel dipisah tab.
func docxText(zr *zip.Reader) (string, error) {
	data, err := readPart(zr, "word/document.xml")
	if err != nil {
		return "", err
	}
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var out textBuilder
	inText := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("dokumen Word tidak bisa dibaca: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inText = true
			case "tab":
				out.write("\t")
			case "br", "cr":
				out.write("\n")
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "p":
				out.write("\n")
			case "tc":
				out.write("\t")
			}
		case xml.CharData:
			if inText {
				out.write(string(t))
			}
		}
	}
	return strings.TrimSpace(out.sb.String()) + "\n", nil
}

// ---- xlsx ----

func attr(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local && a.Name.Space == "" {
			return a.Value
		}
	}
	return ""
}

func relAttr(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local && (a.Name.Space == nsRelationship || strings.HasSuffix(a.Name.Space, "relationships")) {
			return a.Value
		}
	}
	return ""
}

func sharedStrings(zr *zip.Reader) []string {
	data, err := readPart(zr, "xl/sharedStrings.xml")
	if err != nil {
		return nil
	}
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var out []string
	var cur strings.Builder
	inSI, inT := false, false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "si":
				inSI = true
				cur.Reset()
			case "t":
				inT = inSI
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "si":
				out = append(out, cur.String())
				inSI = false
			case "t":
				inT = false
			}
		case xml.CharData:
			if inT {
				cur.WriteString(string(t))
			}
		}
	}
	return out
}

type sheetRef struct{ name, target string }

func sheetList(zr *zip.Reader) []sheetRef {
	wb, err := readPart(zr, "xl/workbook.xml")
	if err != nil {
		return nil
	}
	rels := map[string]string{}
	if rb, err := readPart(zr, "xl/_rels/workbook.xml.rels"); err == nil {
		dec := xml.NewDecoder(strings.NewReader(string(rb)))
		for {
			tok, err := dec.Token()
			if err != nil {
				break
			}
			if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "Relationship" {
				rels[attr(se, "Id")] = attr(se, "Target")
			}
		}
	}
	var out []sheetRef
	dec := xml.NewDecoder(strings.NewReader(string(wb)))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "sheet" {
			target := rels[relAttr(se, "id")]
			if target == "" {
				continue
			}
			if strings.HasPrefix(target, "/") {
				target = strings.TrimPrefix(target, "/")
			} else {
				target = path.Join("xl", target)
			}
			out = append(out, sheetRef{attr(se, "name"), target})
		}
	}
	return out
}

// colIndex mengubah "AB12" menjadi indeks kolom 0-based (AB = 27).
func colIndex(ref string) int {
	n := 0
	for _, r := range ref {
		if r < 'A' || r > 'Z' {
			break
		}
		n = n*26 + int(r-'A'+1)
	}
	return n - 1
}

func sheetText(zr *zip.Reader, target string, shared []string, out *textBuilder) {
	data, err := readPart(zr, target)
	if err != nil {
		return
	}
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var row []string
	rows := 0
	var cellType string
	cellCol := 0
	var val strings.Builder
	inV, inIS, inT := false, false, false
	flushCell := func() {
		v := val.String()
		if cellType == "s" {
			if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && i >= 0 && i < len(shared) {
				v = shared[i]
			}
		}
		for len(row) <= cellCol {
			row = append(row, "")
		}
		row[cellCol] = strings.NewReplacer("\t", " ", "\r", " ", "\n", " ").Replace(v)
		val.Reset()
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				row = row[:0]
			case "c":
				cellType = attr(t, "t")
				if ref := attr(t, "r"); ref != "" {
					cellCol = colIndex(ref)
				} else {
					cellCol = len(row)
				}
				val.Reset()
			case "v":
				inV = true
			case "is":
				inIS = true
			case "t":
				inT = inIS
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v":
				inV = false
			case "is":
				inIS = false
			case "t":
				inT = false
			case "c":
				flushCell()
			case "row":
				end := len(row)
				for end > 0 && row[end-1] == "" {
					end--
				}
				if end > 0 {
					out.write(strings.Join(row[:end], "\t") + "\n")
					rows++
					if rows >= maxSheetRows {
						out.write(fmt.Sprintf("[... lembar dipotong di %d baris ...]\n", maxSheetRows))
						return
					}
				}
			}
		case xml.CharData:
			if inV || inT {
				val.WriteString(string(t))
			}
		}
	}
}

// xlsxText mengubah setiap lembar menjadi teks TSV. Tanggal tampil sebagai angka serial
// (format sel tidak diterapkan).
func xlsxText(zr *zip.Reader) (string, error) {
	sheets := sheetList(zr)
	if len(sheets) == 0 {
		return "", fmt.Errorf("tidak ada lembar kerja pada file Excel")
	}
	shared := sharedStrings(zr)
	var out textBuilder
	for i, sh := range sheets {
		if i >= maxSheets {
			out.write("[... lembar lainnya diabaikan ...]\n")
			break
		}
		out.write(fmt.Sprintf("=== Lembar: %s ===\n", sh.name))
		sheetText(zr, sh.target, shared, &out)
		out.write("\n")
	}
	return out.sb.String(), nil
}
