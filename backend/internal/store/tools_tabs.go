package store

import (
	"encoding/json"
	"os"
	"time"
)

const maxTabsPerKind = 5

// ToolTabMeta: satu tab Terminal atau Browser yang terikat session chat.
type ToolTabMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Cwd       string    `json:"cwd,omitempty"`
	URL       string    `json:"url,omitempty"`
	Active    bool      `json:"active,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SessionToolsTabs: tab per chat session.
type SessionToolsTabs struct {
	Terminal []ToolTabMeta `json:"terminal"`
	Browser  []ToolTabMeta `json:"browser"`
}

type toolsTabsFile map[string]SessionToolsTabs

func (s *Store) loadToolsTabs() toolsTabsFile {
	out := toolsTabsFile{}
	if b, err := os.ReadFile(s.path("tools-tabs.json")); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if out == nil {
		out = toolsTabsFile{}
	}
	return out
}

func (s *Store) saveToolsTabs(v toolsTabsFile) error {
	return s.writeAtomic("tools-tabs.json", v)
}

// GetToolsTabs mengembalikan salinan tab untuk session chat.
func (s *Store) GetToolsTabs(sessionID string) SessionToolsTabs {
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.loadToolsTabs()
	st := all[sessionID]
	if st.Terminal == nil {
		st.Terminal = []ToolTabMeta{}
	}
	if st.Browser == nil {
		st.Browser = []ToolTabMeta{}
	}
	return st
}

// SetToolsTabs menyimpan seluruh set tab untuk satu session (dipakai setelah mutate).
func (s *Store) SetToolsTabs(sessionID string, st SessionToolsTabs) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.loadToolsTabs()
	if st.Terminal == nil {
		st.Terminal = []ToolTabMeta{}
	}
	if st.Browser == nil {
		st.Browser = []ToolTabMeta{}
	}
	if len(st.Terminal) == 0 && len(st.Browser) == 0 {
		delete(all, sessionID)
	} else {
		all[sessionID] = st
	}
	return s.saveToolsTabs(all)
}

// DeleteToolsTabs menghapus semua tab metadata session (saat session dihapus).
func (s *Store) DeleteToolsTabs(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.loadToolsTabs()
	if _, ok := all[sessionID]; !ok {
		return nil
	}
	delete(all, sessionID)
	return s.saveToolsTabs(all)
}

// MaxTabsPerKind batas tab terminal/browser per chat.
func MaxTabsPerKind() int { return maxTabsPerKind }
