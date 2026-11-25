// Package store menyimpan data lokal Ayumi Link (bukan data Claude/Desktop).
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// State: override folder per session dan session Desktop yang disembunyikan.
type State struct {
	Groups map[string]string `json:"groups"`
	Hidden map[string]bool   `json:"hidden"`
	// Titles: nama session yang diubah di Ayumi Link (id -> judul). Tidak mengubah data Claude/Desktop.
	Titles map[string]string `json:"titles,omitempty"`
	// GroupNames: nama folder yang diubah di Ayumi Link (nama asal -> nama baru).
	GroupNames map[string]string `json:"groupNames,omitempty"`
}

type Store struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) *Store { return &Store{dir: dir} }

func (s *Store) Dir() string { return s.dir }

func (s *Store) path(name string) string { return filepath.Join(s.dir, name) }

// writeAtomic menulis lewat file sementara lalu rename, dengan izin 0600.
func (s *Store) writeAtomic(name string, v any) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, name+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path(name))
}

// ---- registry: session yang dibuat lewat Ayumi Link (id -> cwd) ----

func (s *Store) loadRegistry() map[string]string {
	out := map[string]string{}
	if b, err := os.ReadFile(s.path("sessions.json")); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func (s *Store) Registry() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadRegistry()
}

func (s *Store) Register(id, cwd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reg := s.loadRegistry()
	if reg[id] == cwd {
		return
	}
	reg[id] = cwd
	_ = s.writeAtomic("sessions.json", reg)
}

func (s *Store) Unregister(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reg := s.loadRegistry()
	if _, ok := reg[id]; !ok {
		return
	}
	delete(reg, id)
	_ = s.writeAtomic("sessions.json", reg)
}

// ---- state ----

func (s *Store) loadState() State {
	st := State{}
	if b, err := os.ReadFile(s.path("state.json")); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	if st.Groups == nil {
		st.Groups = map[string]string{}
	}
	if st.Hidden == nil {
		st.Hidden = map[string]bool{}
	}
	if st.Titles == nil {
		st.Titles = map[string]string{}
	}
	if st.GroupNames == nil {
		st.GroupNames = map[string]string{}
	}
	return st
}

func (s *Store) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadState()
}

func (s *Store) Update(fn func(*State)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.loadState()
	fn(&st)
	return s.writeAtomic("state.json", st)
}

// ---- preferensi chat default ----

type Prefs struct {
	PermissionMode string `json:"permissionMode"`
	Model          string `json:"model"`
	Effort         string `json:"effort"`
}

// Prefs mengembalikan preferensi tersimpan; ok=false bila belum pernah disimpan.
func (s *Store) Prefs() (Prefs, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var p Prefs
	b, err := os.ReadFile(s.path("prefs.json"))
	if err != nil || json.Unmarshal(b, &p) != nil {
		return Prefs{}, false
	}
	return p, true
}

func (s *Store) SetPrefs(p Prefs) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeAtomic("prefs.json", p)
}

// ---- ikon PWA / branding ----

// Branding: ikon + warna latar aplikasi (PWA / favicon).
type Branding struct {
	Icon      string `json:"icon"`
	Bg        string `json:"bg"`
	Fg        string `json:"fg"`
	UpdatedAt int64  `json:"updatedAt"`
}

func (s *Store) Branding() (Branding, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b Branding
	raw, err := os.ReadFile(s.path("branding.json"))
	if err != nil || json.Unmarshal(raw, &b) != nil {
		return Branding{}, false
	}
	return b, true
}

func (s *Store) SetBranding(b Branding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeAtomic("branding.json", b)
}

// WriteBrandingPNG menyimpan PNG ikon (192 / 512 / apple).
func (s *Store) WriteBrandingPNG(name string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path(name), data, 0o600)
}

func (s *Store) BrandingPNG(name string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path(name))
	if err != nil || len(b) == 0 {
		return nil, false
	}
	return b, true
}

func (s *Store) ClearBranding() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range []string{"branding.json", "branding-192.png", "branding-512.png", "branding-apple.png"} {
		_ = os.Remove(s.path(n))
	}
}
