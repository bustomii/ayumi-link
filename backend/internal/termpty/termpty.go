// Package termpty: terminal PTY penuh per tab, terikat session chat Ayumi Link.
package termpty

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/creack/pty"

	"ayumi-link/internal/fsbrowse"
	"ayumi-link/internal/store"
)

const (
	defaultIdle  = 10 * time.Minute
	defaultGrace = 2 * time.Minute
	maxGlobal    = 20
	scrollMax    = 200 * 1024
)

var (
	ErrNotFound   = errors.New("tab terminal tidak ditemukan")
	ErrLimit      = errors.New("batas tab terminal tercapai")
	ErrBadSession = errors.New("sessionId tidak valid")
)

// TabInfo dikirim ke klien.
type TabInfo struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	Title     string    `json:"title"`
	Cwd       string    `json:"cwd"`
	Alive     bool      `json:"alive"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type session struct {
	id, chatID, title string
	cmd               *exec.Cmd
	ptmx              *os.File
	cwd               string
	created, updated  time.Time
	lastActive        time.Time
	lastDetach        time.Time // zero = connected
	connected         int
	scroll            []byte
	mu                sync.Mutex
	closed            bool
	subs              map[chan []byte]struct{}
}

// Manager mengelola PTY per tab.
type Manager struct {
	browser *fsbrowse.Browser
	store   *store.Store
	svcCwd  func(sessionID string) (string, error)
	idle    time.Duration
	grace   time.Duration
	mu      sync.Mutex
	tabs    map[string]*session // tabId -> session
}

func New(browser *fsbrowse.Browser, st *store.Store, svcCwd func(string) (string, error), idle time.Duration) *Manager {
	if idle <= 0 {
		idle = defaultIdle
	}
	m := &Manager{
		browser: browser,
		store:   st,
		svcCwd:  svcCwd,
		idle:    idle,
		grace:   defaultGrace,
		tabs:    map[string]*session{},
	}
	go m.reaper()
	return m
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func shellBin() (string, []string) {
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh, []string{"-l"}
	}
	if runtime.GOOS == "windows" {
		return "cmd.exe", nil
	}
	return "/bin/bash", []string{"-l"}
}

func (m *Manager) resolveStartCwd(chatID string) string {
	if chatID != "" && m.svcCwd != nil {
		if c, err := m.svcCwd(chatID); err == nil {
			if real, ok := m.browser.ResolveDir(c); ok {
				return real
			}
		}
	}
	roots := m.browser.Roots()
	if len(roots) > 0 {
		return roots[0]
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return os.TempDir()
}

func (m *Manager) reaper() {
	for range time.Tick(30 * time.Second) {
		m.mu.Lock()
		now := time.Now()
		var kill []string
		for id, s := range m.tabs {
			s.mu.Lock()
			idleTooLong := now.Sub(s.lastActive) > m.idle
			graceExpired := s.connected == 0 && !s.lastDetach.IsZero() && now.Sub(s.lastDetach) > m.grace && idleTooLong
			// Kill bila idle penuh, atau (disconnect + grace lewat dan tetap idle).
			doKill := idleTooLong || graceExpired
			s.mu.Unlock()
			if doKill {
				kill = append(kill, id)
			}
		}
		m.mu.Unlock()
		for _, id := range kill {
			log.Printf("terminal tab %s dimatikan (idle/grace)", id)
			_ = m.CloseTab(id)
		}
	}
}

// ListTabs mengembalikan tab hidup + metadata tersimpan untuk chat.
func (m *Manager) ListTabs(chatID string) []TabInfo {
	meta := m.store.GetToolsTabs(chatID)
	m.mu.Lock()
	defer m.mu.Unlock()
	alive := map[string]*session{}
	for id, s := range m.tabs {
		if s.chatID == chatID {
			alive[id] = s
		}
	}
	out := make([]TabInfo, 0, len(meta.Terminal)+len(alive))
	seen := map[string]bool{}
	for _, t := range meta.Terminal {
		info := TabInfo{
			ID: t.ID, SessionID: chatID, Title: t.Title, Cwd: t.Cwd,
			Active: t.Active, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
		if s, ok := alive[t.ID]; ok {
			s.mu.Lock()
			info.Alive = !s.closed
			info.Cwd = s.cwd
			info.Title = s.title
			s.mu.Unlock()
		}
		out = append(out, info)
		seen[t.ID] = true
	}
	for id, s := range alive {
		if seen[id] {
			continue
		}
		s.mu.Lock()
		out = append(out, TabInfo{
			ID: id, SessionID: chatID, Title: s.title, Cwd: s.cwd, Alive: !s.closed,
			CreatedAt: s.created, UpdatedAt: s.updated,
		})
		s.mu.Unlock()
	}
	return out
}

func (m *Manager) persistMeta(chatID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.persistMetaLocked(chatID)
}

func (m *Manager) persistMetaLocked(chatID string) {
	st := m.store.GetToolsTabs(chatID)
	var terms []store.ToolTabMeta
	activeSet := false
	for _, s := range m.tabs {
		if s.chatID != chatID {
			continue
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			continue
		}
		terms = append(terms, store.ToolTabMeta{
			ID: s.id, Title: s.title, Cwd: s.cwd,
			CreatedAt: s.created, UpdatedAt: s.updated,
		})
		s.mu.Unlock()
	}
	// Keep order from previous meta when possible
	order := map[string]int{}
	for i, t := range st.Terminal {
		order[t.ID] = i
	}
	for i := range terms {
		if !activeSet {
			terms[i].Active = true
			activeSet = true
		}
	}
	_ = order
	st.Terminal = terms
	_ = m.store.SetToolsTabs(chatID, st)
}

// CreateTab membuat PTY baru untuk chat.
func (m *Manager) CreateTab(chatID, title string) (*TabInfo, error) {
	if chatID == "" {
		return nil, ErrBadSession
	}
	m.mu.Lock()
	nChat, nAll := 0, len(m.tabs)
	for _, s := range m.tabs {
		if s.chatID == chatID {
			nChat++
		}
	}
	if nChat >= store.MaxTabsPerKind() || nAll >= maxGlobal {
		m.mu.Unlock()
		return nil, ErrLimit
	}
	m.mu.Unlock()

	cwd := m.resolveStartCwd(chatID)
	bin, args := shellBin()
	cmd := exec.Command(bin, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka PTY: %w", err)
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	id := newID()
	if title == "" {
		title = "Terminal"
	}
	now := time.Now()
	s := &session{
		id: id, chatID: chatID, title: title, cmd: cmd, ptmx: ptmx, cwd: cwd,
		created: now, updated: now, lastActive: now, subs: map[chan []byte]struct{}{},
	}
	m.mu.Lock()
	m.tabs[id] = s
	m.persistMetaLocked(chatID)
	m.mu.Unlock()

	go m.readLoop(s)

	return &TabInfo{
		ID: id, SessionID: chatID, Title: title, Cwd: cwd, Alive: true, Active: true,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (m *Manager) readLoop(s *session) {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			s.mu.Lock()
			s.lastActive = time.Now()
			s.updated = s.lastActive
			s.scroll = append(s.scroll, chunk...)
			if len(s.scroll) > scrollMax {
				s.scroll = s.scroll[len(s.scroll)-scrollMax:]
			}
			for ch := range s.subs {
				select {
				case ch <- chunk:
				default:
				}
			}
			s.mu.Unlock()
		}
		if err != nil {
			s.mu.Lock()
			s.closed = true
			for ch := range s.subs {
				close(ch)
			}
			s.subs = map[chan []byte]struct{}{}
			s.mu.Unlock()
			return
		}
	}
}

// Attach mendaftarkan subscriber output; kembalikan scrollback + channel.
func (m *Manager) Attach(tabID string) (scroll []byte, out <-chan []byte, detach func(), err error) {
	m.mu.Lock()
	s := m.tabs[tabID]
	m.mu.Unlock()
	if s == nil {
		return nil, nil, nil, ErrNotFound
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, nil, nil, ErrNotFound
	}
	ch := make(chan []byte, 32)
	s.subs[ch] = struct{}{}
	s.connected++
	s.lastDetach = time.Time{}
	s.lastActive = time.Now()
	scroll = append([]byte(nil), s.scroll...)
	s.mu.Unlock()

	detach = func() {
		s.mu.Lock()
		if _, ok := s.subs[ch]; ok {
			delete(s.subs, ch)
			close(ch)
		}
		if s.connected > 0 {
			s.connected--
		}
		if s.connected == 0 {
			s.lastDetach = time.Now()
		}
		s.mu.Unlock()
	}
	return scroll, ch, detach, nil
}

// Write mengirim input ke PTY.
func (m *Manager) Write(tabID string, data []byte) error {
	m.mu.Lock()
	s := m.tabs[tabID]
	m.mu.Unlock()
	if s == nil {
		return ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrNotFound
	}
	s.lastActive = time.Now()
	_, err := s.ptmx.Write(data)
	return err
}

// Resize mengubah ukuran PTY.
func (m *Manager) Resize(tabID string, cols, rows uint16) error {
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	m.mu.Lock()
	s := m.tabs[tabID]
	m.mu.Unlock()
	if s == nil {
		return ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrNotFound
	}
	s.lastActive = time.Now()
	return pty.Setsize(s.ptmx, &pty.Winsize{Cols: cols, Rows: rows})
}

// CloseTab membunuh PTY dan hapus dari meta.
func (m *Manager) CloseTab(tabID string) error {
	m.mu.Lock()
	s := m.tabs[tabID]
	if s == nil {
		m.mu.Unlock()
		return ErrNotFound
	}
	chatID := s.chatID
	delete(m.tabs, tabID)
	m.mu.Unlock()

	s.mu.Lock()
	s.closed = true
	for ch := range s.subs {
		close(ch)
	}
	s.subs = map[chan []byte]struct{}{}
	if s.ptmx != nil {
		_ = s.ptmx.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	s.mu.Unlock()

	m.persistMeta(chatID)
	return nil
}

// KillChat menutup semua tab terminal milik chat (metadata browser tetap).
func (m *Manager) KillChat(chatID string) {
	m.mu.Lock()
	var ids []string
	for id, s := range m.tabs {
		if s.chatID == chatID {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.CloseTab(id)
	}
	st := m.store.GetToolsTabs(chatID)
	st.Terminal = nil
	_ = m.store.SetToolsTabs(chatID, st)
}

// EnsureTab menghidupkan ulang tab dari meta bila belum alive (buat PTY baru dengan cwd tersimpan).
func (m *Manager) EnsureTab(chatID, tabID string) (*TabInfo, error) {
	m.mu.Lock()
	if s, ok := m.tabs[tabID]; ok {
		s.mu.Lock()
		alive := !s.closed
		info := &TabInfo{ID: s.id, SessionID: s.chatID, Title: s.title, Cwd: s.cwd, Alive: alive, CreatedAt: s.created, UpdatedAt: s.updated}
		s.mu.Unlock()
		m.mu.Unlock()
		if alive {
			return info, nil
		}
	} else {
		m.mu.Unlock()
	}
	meta := m.store.GetToolsTabs(chatID)
	var hit *store.ToolTabMeta
	for i := range meta.Terminal {
		if meta.Terminal[i].ID == tabID {
			hit = &meta.Terminal[i]
			break
		}
	}
	if hit == nil {
		return m.CreateTab(chatID, "Terminal")
	}
	// Recreate with same id if possible — CreateTab makes new id; update meta after.
	info, err := m.CreateTab(chatID, hit.Title)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// Owns memeriksa tab milik chat dan masih hidup.
func (m *Manager) Owns(chatID, tabID string) bool {
	m.mu.Lock()
	s := m.tabs[tabID]
	m.mu.Unlock()
	if s == nil || s.chatID != chatID {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.closed
}

// CloseAll saat shutdown.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.tabs))
	for id := range m.tabs {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.CloseTab(id)
	}
}

var _ = io.EOF
