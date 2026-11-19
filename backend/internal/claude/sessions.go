package claude

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ayumi-link/internal/store"
)

// Service membaca dan memodifikasi session Claude Code milik pengguna.
type Service struct {
	Home       string // ~/.claude
	DesktopDir string // folder data Claude Desktop
	Store      *store.Store
}

var (
	ErrNotFound = errors.New("session not found")
	ErrBadID    = errors.New("invalid session id")
)

func listLiveSessions(home string) map[string]Session {
	out := map[string]Session{}
	dir := filepath.Join(home, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var raw struct {
			SessionID string `json:"sessionId"`
			Cwd       string `json:"cwd"`
			Name      string `json:"name"`
			Status    string `json:"status"`
			UpdatedAt int64  `json:"updatedAt"`
		}
		if json.Unmarshal(b, &raw) != nil || raw.SessionID == "" {
			continue
		}
		updated := time.UnixMilli(raw.UpdatedAt)
		if raw.UpdatedAt > 1_000_000_000_000_000 {
			updated = time.Unix(0, raw.UpdatedAt)
		}
		title := raw.Name
		if title == "" {
			title = filepath.Base(raw.Cwd)
		}
		out[raw.SessionID] = Session{
			ID:        raw.SessionID,
			Title:     title,
			Cwd:       raw.Cwd,
			Project:   filepath.Base(raw.Cwd),
			UpdatedAt: updated,
			Live:      true,
			Status:    raw.Status,
		}
	}
	return out
}

func listDiskSessions(home string) []Session {
	root := filepath.Join(home, "projects")
	var sessions []Session

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) != 2 {
			return nil
		}
		projectKey := parts[0]
		id := strings.TrimSuffix(parts[1], ".jsonl")
		info, err := d.Info()
		if err != nil {
			return nil
		}
		titleDir := filepath.Join(root, projectKey, id)
		title := readCustomTitle(titleDir)
		if title == "" {
			title = firstUserPrompt(path)
		}
		if title == "" && len(id) >= 8 {
			title = id[:8]
		} else if title == "" {
			title = id
		}
		cwd := cwdFromJSONL(path)
		if cwd == "" {
			cwd = projectKeyToCwd(projectKey)
		}
		sessions = append(sessions, Session{
			ID:        id,
			Title:     title,
			Cwd:       cwd,
			Project:   filepath.Base(cwd),
			UpdatedAt: info.ModTime(),
			Live:      false,
		})
		return nil
	})
	return sessions
}

// FindJSONL mencari file transcript sebuah session.
func FindJSONL(home, id string) string {
	root := filepath.Join(home, "projects")
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == id+".jsonl" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// listBase menggabungkan session live + disk lalu menerapkan group Desktop, session tersembunyi,
// override folder, dan override nama session. Belum termasuk penggantian nama folder.
func (s *Service) listBase() ([]Session, map[string]string) {
	live := listLiveSessions(s.Home)
	byID := map[string]Session{}
	for _, d := range listDiskSessions(s.Home) {
		byID[d.ID] = d
	}
	for id, l := range live {
		if existing, ok := byID[id]; ok {
			if l.Title == "" {
				l.Title = existing.Title
			}
			if l.Cwd == "" {
				l.Cwd = existing.Cwd
				l.Project = existing.Project
			}
		}
		l.Live = true
		byID[id] = l
	}
	out := make([]Session, 0, len(byID))
	for _, v := range byID {
		out = append(out, v)
	}

	registry := s.Store.Registry()
	out = applyDesktopGroups(out, s.DesktopDir, registry)
	return s.applyLocalState(out), registry
}

// List mengembalikan semua session yang tampil di Ayumi Link, terurut per folder lalu waktu.
func (s *Service) List() []Session {
	out, registry := s.listBase()
	names := s.Store.State().GroupNames
	for i := range out {
		if renamed := names[out[i].Group]; renamed != "" {
			out[i].Group = renamed
		}
		if _, ok := registry[out[i].ID]; ok {
			out[i].Local = true
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].GroupOrder != out[j].GroupOrder {
			return out[i].GroupOrder < out[j].GroupOrder
		}
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

// applyLocalState menyembunyikan session yang disembunyikan dan menerapkan override group.
func (s *Service) applyLocalState(in []Session) []Session {
	st := s.Store.State()
	order := map[string]int{}
	for _, x := range in {
		if _, ok := order[x.Group]; !ok {
			order[x.Group] = x.GroupOrder
		}
	}
	out := make([]Session, 0, len(in))
	for _, x := range in {
		if st.Hidden[x.ID] {
			continue
		}
		if g := st.Groups[x.ID]; g != "" {
			x.Group = g
			if ord, ok := order[g]; ok {
				x.GroupOrder = ord
			} else {
				x.GroupOrder = 1000
			}
		}
		if t := st.Titles[x.ID]; t != "" {
			x.Title = t
		}
		out = append(out, x)
	}
	return out
}

type Detail struct {
	Session  Session   `json:"session"`
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
	Total    int       `json:"total"`
	Limit    int       `json:"limit"`
}

// Detail mengembalikan pesan sebuah session dengan paginasi (terbaru dulu).
func (s *Service) Detail(id string, limit int, before string) (*Detail, error) {
	path := FindJSONL(s.Home, id)
	if path == "" {
		return nil, ErrNotFound
	}
	msgs, err := LoadMessages(path)
	if err != nil {
		return nil, err
	}
	page, hasMore := Paginate(msgs, limit, before)
	var meta Session
	for _, x := range s.List() {
		if x.ID == id {
			meta = x
			break
		}
	}
	if page == nil {
		page = []Message{}
	}
	return &Detail{Session: meta, Messages: page, HasMore: hasMore, Total: len(msgs), Limit: limit}, nil
}

// SetGroup memindahkan session ke folder (hanya berlaku di Ayumi Link). group kosong = reset.
func (s *Service) SetGroup(id, group string) error {
	return s.Store.Update(func(st *store.State) {
		if group == "" {
			delete(st.Groups, id)
		} else {
			st.Groups[id] = group
		}
	})
}

// Delete: session buatan Ayumi Link dihapus permanen; session Desktop hanya disembunyikan.
// Mengembalikan "deleted" atau "hidden".
func (s *Service) Delete(id string) (string, error) {
	if _, mine := s.Store.Registry()[id]; !mine {
		err := s.Store.Update(func(st *store.State) { st.Hidden[id] = true })
		return "hidden", err
	}
	if p := FindJSONL(s.Home, id); p != "" {
		root := filepath.Join(s.Home, "projects")
		if rel, err := filepath.Rel(root, p); err != nil || strings.HasPrefix(rel, "..") {
			return "", errors.New("path tidak valid")
		}
		if err := os.Remove(p); err != nil {
			return "", err
		}
		_ = os.RemoveAll(filepath.Join(filepath.Dir(p), id))
	}
	s.Store.Unregister(id)
	_ = s.Store.Update(func(st *store.State) {
		delete(st.Groups, id)
		delete(st.Hidden, id)
		delete(st.Titles, id)
	})
	return "deleted", nil
}

// SetTitle mengganti nama session (hanya berlaku di Ayumi Link). title kosong = kembali ke nama asli.
func (s *Service) SetTitle(id, title string) error {
	return s.Store.Update(func(st *store.State) {
		if title == "" {
			delete(st.Titles, id)
		} else {
			st.Titles[id] = title
		}
	})
}

// RenameGroup mengganti nama folder yang sedang tampil (from) menjadi to, hanya di Ayumi Link.
// Pemetaan hanya disimpan untuk folder yang ada saat ini; bila to sama dengan nama asal,
// pemetaan dihapus (kembali ke nama asli); bila to sudah dipakai folder lain, keduanya digabung.
func (s *Service) RenameGroup(from, to string) error {
	base, _ := s.listBase()
	existing := map[string]bool{}
	for _, x := range base {
		existing[x.Group] = true
	}
	return s.Store.Update(func(st *store.State) {
		shown := func(g string) string {
			if r := st.GroupNames[g]; r != "" {
				return r
			}
			return g
		}
		for g := range existing {
			if shown(g) == from {
				st.GroupNames[g] = to
			}
		}
		for k, v := range st.GroupNames {
			if k == v || !existing[k] { // identitas atau folder yang sudah tidak ada
				delete(st.GroupNames, k)
			}
		}
	})
}
