package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileChange adalah satu file yang dilacak di file-history session Claude.
type FileChange struct {
	Path       string    `json:"path"`       // relatif (kunci snapshot)
	AbsPath    string    `json:"absPath"`    // path absolut di disk
	BackupName string    `json:"backupName"` // nama file di ~/.claude/file-history/<id>/
	Version    int       `json:"version"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Status     string    `json:"status"` // modified | added | deleted | unknown
}

// ChangeDiff adalah isi sebelum/sesudah + unified diff untuk satu file.
type ChangeDiff struct {
	FileChange
	Before string `json:"before"`
	After  string `json:"after"`
	Diff   string `json:"diff"`
}

type backupMeta struct {
	BackupFileName string `json:"backupFileName"`
	Version        int    `json:"version"`
	BackupTime     string `json:"backupTime"`
	RealParentDir  string `json:"realParentDir"`
}

func historyDir(home, id string) string {
	return filepath.Join(home, "file-history", id)
}

// latestTrackedBackups membaca snapshot file-history terakhir dari transcript.
func latestTrackedBackups(jsonlPath string) map[string]backupMeta {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxJSONLLine)
	var latest map[string]backupMeta
	for sc.Scan() {
		var raw struct {
			Type     string `json:"type"`
			Snapshot struct {
				TrackedFileBackups map[string]backupMeta `json:"trackedFileBackups"`
			} `json:"snapshot"`
		}
		if json.Unmarshal(sc.Bytes(), &raw) != nil || raw.Type != "file-history-snapshot" {
			continue
		}
		if raw.Snapshot.TrackedFileBackups == nil {
			continue
		}
		latest = raw.Snapshot.TrackedFileBackups
	}
	return latest
}

func absFromMeta(rel string, meta backupMeta) string {
	base := filepath.Base(rel)
	if meta.RealParentDir != "" {
		return filepath.Join(meta.RealParentDir, base)
	}
	if filepath.IsAbs(rel) {
		return rel
	}
	return rel
}

func parseBackupTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return time.Time{}
	}
	return t
}

func changeStatus(beforeExists, afterExists bool) string {
	switch {
	case !beforeExists && afterExists:
		return "added"
	case beforeExists && !afterExists:
		return "deleted"
	case beforeExists && afterExists:
		return "modified"
	default:
		return "unknown"
	}
}

// ListChanges mengembalikan file yang berubah menurut file-history Claude untuk session.
func (s *Service) ListChanges(id string) ([]FileChange, error) {
	if !IsUUID(id) {
		return nil, ErrBadID
	}
	jsonl := FindJSONL(s.Home, id)
	if jsonl == "" {
		return nil, ErrNotFound
	}
	backups := latestTrackedBackups(jsonl)
	out := make([]FileChange, 0, len(backups))
	hist := historyDir(s.Home, id)
	for rel, meta := range backups {
		abs := absFromMeta(rel, meta)
		beforeExists := meta.BackupFileName != ""
		if beforeExists {
			if _, err := os.Stat(filepath.Join(hist, meta.BackupFileName)); err != nil {
				beforeExists = false
			}
		}
		_, afterErr := os.Stat(abs)
		afterExists := afterErr == nil
		out = append(out, FileChange{
			Path:       rel,
			AbsPath:    abs,
			BackupName: meta.BackupFileName,
			Version:    meta.Version,
			UpdatedAt:  parseBackupTime(meta.BackupTime),
			Status:     changeStatus(beforeExists, afterExists),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].Path < out[j].Path
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

// GetChangeDiff membandingkan backup file-history dengan isi file di disk.
func (s *Service) GetChangeDiff(id, relOrAbs string) (*ChangeDiff, error) {
	list, err := s.ListChanges(id)
	if err != nil {
		return nil, err
	}
	relOrAbs = strings.TrimSpace(relOrAbs)
	var hit *FileChange
	for i := range list {
		if list[i].Path == relOrAbs || list[i].AbsPath == relOrAbs {
			hit = &list[i]
			break
		}
	}
	if hit == nil {
		return nil, ErrNotFound
	}
	hist := historyDir(s.Home, id)
	var before string
	if hit.BackupName != "" {
		if b, err := os.ReadFile(filepath.Join(hist, hit.BackupName)); err == nil {
			before = string(b)
		}
	}
	var after string
	if b, err := os.ReadFile(hit.AbsPath); err == nil {
		after = string(b)
	}
	d := &ChangeDiff{FileChange: *hit, Before: before, After: after, Diff: unifiedDiff(hit.Path, before, after)}
	d.Status = changeStatus(before != "" || hit.BackupName != "", after != "" || fileExists(hit.AbsPath))
	if before == "" && after != "" {
		d.Status = "added"
	} else if before != "" && after == "" {
		d.Status = "deleted"
	} else if before != after {
		d.Status = "modified"
	}
	return d, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// unifiedDiff menghasilkan diff sederhana baris-per-baris (cukup untuk UI Changes).
func unifiedDiff(name, before, after string) string {
	if before == after {
		return ""
	}
	a := splitLines(before)
	b := splitLines(after)
	var bld strings.Builder
	bld.WriteString("--- a/" + name + "\n")
	bld.WriteString("+++ b/" + name + "\n")
	// Myers sederhana: tampilkan semua penghapusan lalu penambahan per blok (bukan optimal, tapi jelas).
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		if i < len(a) && j < len(b) && a[i] == b[j] {
			bld.WriteString(" " + a[i] + "\n")
			i++
			j++
			continue
		}
		// cari kecocokan berikutnya di jendela kecil
		matched := false
		if i < len(a) && j < len(b) {
			for look := 1; look <= 40; look++ {
				if i+look < len(a) && a[i+look] == b[j] {
					for k := 0; k < look; k++ {
						bld.WriteString("-" + a[i+k] + "\n")
					}
					i += look
					matched = true
					break
				}
				if j+look < len(b) && a[i] == b[j+look] {
					for k := 0; k < look; k++ {
						bld.WriteString("+" + b[j+k] + "\n")
					}
					j += look
					matched = true
					break
				}
			}
		}
		if matched {
			continue
		}
		if i < len(a) {
			bld.WriteString("-" + a[i] + "\n")
			i++
		}
		if j < len(b) {
			bld.WriteString("+" + b[j] + "\n")
			j++
		}
	}
	return bld.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}
