package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"ayumi-link/internal/store"
)

// encodeProjectDir: nama folder di ~/.claude/projects untuk sebuah cwd
// (semua karakter selain huruf/angka menjadi "-", sama seperti Claude Code).
func encodeProjectDir(cwd string) string {
	b := []byte(cwd)
	for i, c := range b {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			b[i] = '-'
		}
	}
	return string(b)
}

// cwdUsable: apakah folder kerja session masih bisa dipakai (ada, dan diizinkan bila CwdOK diatur).
func (s *Service) cwdUsable(cwd string) bool {
	if cwd == "" {
		return true // tidak diketahui: jangan menandai rusak
	}
	if s.CwdOK != nil {
		return s.CwdOK(cwd)
	}
	st, err := os.Stat(cwd)
	return err == nil && st.IsDir()
}

// Fork menyalin transcript session ke folder kerja baru sebagai session baru (id baru), seperti
// "fork" di Claude Desktop. Session asal tidak diubah. Mengembalikan id session baru.
func (s *Service) Fork(id, newCwd string) (string, error) {
	if !IsUUID(id) {
		return "", ErrBadID
	}
	src := FindJSONL(s.Home, id)
	if src == "" {
		return "", ErrNotFound
	}
	if newCwd == "" {
		return "", errors.New("folder tujuan kosong")
	}
	newID := newUUID()
	dir := filepath.Join(s.Home, "projects", encodeProjectDir(newCwd))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, newID+".jsonl")
	tmp := dst + ".tmp"
	if err := copyRewritten(src, tmp, newID, newCwd); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}

	s.Store.Register(newID, newCwd)
	// Nama session ikut terbawa.
	title := ""
	for _, x := range s.List() {
		if x.ID == id {
			title = x.Title
			break
		}
	}
	if title != "" {
		_ = s.Store.Update(func(st *store.State) { st.Titles[newID] = title })
	}
	return newID, nil
}

// copyRewritten menyalin file JSONL baris demi baris; field "sessionId" dan "cwd" pada tiap
// baris diganti. Baris yang bukan JSON objek disalin apa adanya.
func copyRewritten(src, dst, newID, newCwd string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(out, 256<<10)
	idJSON, _ := json.Marshal(newID)
	cwdJSON, _ := json.Marshal(newCwd)

	r := bufio.NewReaderSize(in, 256<<10)
	for {
		line, rerr := r.ReadBytes('\n')
		if len(line) > 0 {
			trimmed := bytes.TrimRight(line, "\r\n")
			if _, err := w.Write(rewriteLine(trimmed, idJSON, cwdJSON)); err != nil {
				out.Close()
				return err
			}
			if err := w.WriteByte('\n'); err != nil {
				out.Close()
				return err
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return rerr
		}
	}
	if err := w.Flush(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func rewriteLine(line, idJSON, cwdJSON []byte) []byte {
	if len(bytes.TrimSpace(line)) == 0 {
		return line
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(line, &m) != nil {
		return line
	}
	changed := false
	if _, ok := m["sessionId"]; ok {
		m["sessionId"] = idJSON
		changed = true
	}
	if _, ok := m["cwd"]; ok {
		m["cwd"] = cwdJSON
		changed = true
	}
	if !changed {
		return line
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if enc.Encode(m) != nil {
		return line
	}
	return bytes.TrimRight(buf.Bytes(), "\n")
}
