package browser

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"ayumi-link/internal/store"
)

// TabInfo: satu tab Browser (CDP page) terikat chat session.
type TabInfo struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Active    bool      `json:"active"`
	Alive     bool      `json:"alive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type pageTab struct {
	id, chatID, title, targetID string
	cdp                         *cdpClient
	created, updated, last      time.Time
}

func newTabID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (m *Manager) ensureTabsMap() {
	if m.pages == nil {
		m.pages = map[string]*pageTab{}
	}
}

// ListTabs menggabungkan meta store + tab hidup.
func (m *Manager) ListTabs(chatID string) []TabInfo {
	meta := []store.ToolTabMeta{}
	if m.store != nil {
		meta = m.store.GetToolsTabs(chatID).Browser
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureTabsMap()
	alive := map[string]*pageTab{}
	for id, p := range m.pages {
		if p.chatID == chatID {
			alive[id] = p
		}
	}
	out := make([]TabInfo, 0, len(meta)+len(alive))
	seen := map[string]bool{}
	for _, t := range meta {
		info := TabInfo{
			ID: t.ID, SessionID: chatID, Title: t.Title, URL: t.URL,
			Active: t.Active || t.ID == m.activeTab, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
		if p, ok := alive[t.ID]; ok {
			info.Alive = p.cdp != nil && p.cdp.alive()
			info.Title = p.title
			if u := m.pageURLLocked(p); u != "" {
				info.URL = u
			}
		}
		out = append(out, info)
		seen[t.ID] = true
	}
	for id, p := range alive {
		if seen[id] {
			continue
		}
		out = append(out, TabInfo{
			ID: id, SessionID: chatID, Title: p.title, URL: m.pageURLLocked(p),
			Alive: true, Active: id == m.activeTab, CreatedAt: p.created, UpdatedAt: p.updated,
		})
	}
	return out
}

func (m *Manager) pageURLLocked(p *pageTab) string {
	if m.port == "" || p.targetID == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	entries, err := m.list(ctx)
	if err != nil {
		return ""
	}
	for _, t := range entries {
		if t.ID == p.targetID {
			if t.URL == "about:blank" {
				return ""
			}
			return t.URL
		}
	}
	return ""
}

func (m *Manager) persistBrowserMeta(chatID string) {
	if m.store == nil {
		return
	}
	st := m.store.GetToolsTabs(chatID)
	var tabs []store.ToolTabMeta
	for _, p := range m.pages {
		if p.chatID != chatID {
			continue
		}
		tabs = append(tabs, store.ToolTabMeta{
			ID: p.id, Title: p.title, URL: m.pageURLLocked(p),
			Active: p.id == m.activeTab, CreatedAt: p.created, UpdatedAt: p.updated,
		})
	}
	st.Browser = tabs
	_ = m.store.SetToolsTabs(chatID, st)
}

// CreateTab menambah halaman CDP baru untuk chat.
func (m *Manager) CreateTab(ctx context.Context, chatID, title, rawURL string) (TabInfo, error) {
	if chatID == "" {
		return TabInfo{}, errors.New("sessionId wajib")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureTabsMap()
	n := 0
	for _, p := range m.pages {
		if p.chatID == chatID {
			n++
		}
	}
	if n >= store.MaxTabsPerKind() {
		return TabInfo{}, fmt.Errorf("batas tab browser (%d) tercapai", store.MaxTabsPerKind())
	}
	if err := m.ensure(ctx); err != nil {
		return TabInfo{}, err
	}
	m.last = time.Now()
	if title == "" {
		title = "Browser"
	}
	now := time.Now()
	id := newTabID()

	var cdp *cdpClient
	var targetID string
	cctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	// Tab pertama: pakai halaman dari ensure(). Berikutnya: /json/new.
	if len(m.pages) == 0 && m.cdp != nil && m.target != "" {
		cdp, targetID = m.cdp, m.target
		_, _ = cdp.call(cctx, "Network.enable", nil)
		if rawURL != "" {
			_, _ = cdp.call(cctx, "Page.navigate", map[string]any{"url": rawURL})
		}
	} else {
		targetURL := "about:blank"
		if rawURL != "" {
			targetURL = rawURL
		}
		var wsPath string
		var err error
		targetID, wsPath, err = m.newPageTarget(ctx, targetURL)
		if err != nil {
			return TabInfo{}, err
		}
		ws, err := wsDial(net.JoinHostPort("127.0.0.1", m.port), wsPath)
		if err != nil {
			return TabInfo{}, err
		}
		cdp = newCDP(ws)
		_, _ = cdp.call(cctx, "Page.enable", nil)
		_, _ = cdp.call(cctx, "Network.enable", nil)
		m.restoreCookiesTo(cctx, cdp)
		_, _ = cdp.call(cctx, "Emulation.setDeviceMetricsOverride", map[string]any{
			"width": m.w, "height": m.h, "deviceScaleFactor": 1, "mobile": false,
		})
	}

	p := &pageTab{id: id, chatID: chatID, title: title, targetID: targetID, cdp: cdp, created: now, updated: now, last: now}
	m.pages[id] = p
	m.activeTab = id
	m.cdp, m.target = cdp, targetID
	m.persistBrowserMeta(chatID)
	u := rawURL
	return TabInfo{ID: id, SessionID: chatID, Title: title, URL: u, Active: true, Alive: true, CreatedAt: now, UpdatedAt: now}, nil
}

func (m *Manager) newPageTarget(ctx context.Context, pageURL string) (id, wsPath string, err error) {
	u := "http://127.0.0.1:" + m.port + "/json/new?" + url.QueryEscape(pageURL)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut, u, nil)
	res, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		// fallback GET (some Chromium builds)
		req, _ = http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		res, err = (&http.Client{Timeout: 8 * time.Second}).Do(req)
	}
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var ent listEntry
	if json.Unmarshal(body, &ent) != nil || ent.WSURL == "" {
		// maybe array
		return m.pageTarget(ctx)
	}
	wu, err := url.Parse(ent.WSURL)
	if err != nil {
		return "", "", err
	}
	return ent.ID, wu.Path, nil
}

func (m *Manager) restoreCookiesTo(ctx context.Context, cdp *cdpClient) {
	b, err := os.ReadFile(m.cookieFile())
	if err != nil {
		return
	}
	var all []map[string]any
	if json.Unmarshal(b, &all) != nil {
		return
	}
	keep := []string{"name", "value", "domain", "path", "httpOnly", "secure", "sameSite"}
	params := make([]map[string]any, 0, len(all))
	for _, c := range all {
		p := map[string]any{}
		for _, k := range keep {
			if v, ok := c[k]; ok {
				p[k] = v
			}
		}
		if session, _ := c["session"].(bool); !session {
			if exp, ok := c["expires"].(float64); ok && exp > 0 {
				p["expires"] = exp
			}
		}
		if p["name"] != nil && p["domain"] != nil {
			params = append(params, p)
		}
	}
	if len(params) > 0 {
		_, _ = cdp.call(ctx, "Network.setCookies", map[string]any{"cookies": params})
	}
}

// ActivateTab memilih tab aktif untuk Frame/Do. Jika proses idle-mati, recreate dari meta tersimpan.
func (m *Manager) ActivateTab(ctx context.Context, chatID, tabID string) (TabInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureTabsMap()
	p := m.pages[tabID]
	if p != nil && p.chatID == chatID && p.cdp != nil && p.cdp.alive() {
		m.activeTab = tabID
		m.cdp, m.target = p.cdp, p.targetID
		m.last = time.Now()
		p.last, p.updated = m.last, m.last
		m.persistBrowserMeta(chatID)
		return TabInfo{
			ID: tabID, SessionID: chatID, Title: p.title, URL: m.pageURLLocked(p),
			Active: true, Alive: true, CreatedAt: p.created, UpdatedAt: p.updated,
		}, nil
	}
	// Recreate dari tools-tabs.json setelah idle/kill.
	title, rawURL := "Browser", ""
	created := time.Now()
	found := false
	if m.store != nil {
		for _, t := range m.store.GetToolsTabs(chatID).Browser {
			if t.ID == tabID {
				found = true
				if t.Title != "" {
					title = t.Title
				}
				rawURL = t.URL
				if !t.CreatedAt.IsZero() {
					created = t.CreatedAt
				}
				break
			}
		}
	}
	if !found {
		return TabInfo{}, ErrNotRunning
	}
	if err := m.ensure(ctx); err != nil {
		return TabInfo{}, err
	}
	m.last = time.Now()
	cctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	var cdp *cdpClient
	var targetID string
	if len(m.pages) == 0 && m.cdp != nil && m.target != "" {
		cdp, targetID = m.cdp, m.target
		_, _ = cdp.call(cctx, "Network.enable", nil)
		if rawURL != "" {
			_, _ = cdp.call(cctx, "Page.navigate", map[string]any{"url": rawURL})
		}
	} else {
		targetURL := "about:blank"
		if rawURL != "" {
			targetURL = rawURL
		}
		id, wsPath, err := m.newPageTarget(ctx, targetURL)
		if err != nil {
			return TabInfo{}, err
		}
		ws, err := wsDial(net.JoinHostPort("127.0.0.1", m.port), wsPath)
		if err != nil {
			return TabInfo{}, err
		}
		cdp = newCDP(ws)
		targetID = id
		_, _ = cdp.call(cctx, "Page.enable", nil)
		_, _ = cdp.call(cctx, "Network.enable", nil)
		m.restoreCookiesTo(cctx, cdp)
		_, _ = cdp.call(cctx, "Emulation.setDeviceMetricsOverride", map[string]any{
			"width": m.w, "height": m.h, "deviceScaleFactor": 1, "mobile": false,
		})
	}
	now := time.Now()
	p = &pageTab{id: tabID, chatID: chatID, title: title, targetID: targetID, cdp: cdp, created: created, updated: now, last: now}
	m.pages[tabID] = p
	m.activeTab = tabID
	m.cdp, m.target = cdp, targetID
	m.persistBrowserMeta(chatID)
	return TabInfo{
		ID: tabID, SessionID: chatID, Title: title, URL: rawURL,
		Active: true, Alive: true, CreatedAt: created, UpdatedAt: now,
	}, nil
}

// CloseTab menutup satu tab; matikan Chromium bila tidak ada tab tersisa.
func (m *Manager) CloseTab(chatID, tabID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureTabsMap()
	p := m.pages[tabID]
	if p == nil {
		if m.store != nil {
			st := m.store.GetToolsTabs(chatID)
			filtered := st.Browser[:0]
			for _, t := range st.Browser {
				if t.ID != tabID {
					filtered = append(filtered, t)
				}
			}
			st.Browser = filtered
			_ = m.store.SetToolsTabs(chatID, st)
		}
		return nil
	}
	if p.chatID != chatID {
		return ErrNotRunning
	}
	if p.cdp != nil {
		p.cdp.close()
	}
	delete(m.pages, tabID)
	if m.activeTab == tabID {
		m.activeTab = ""
		m.cdp, m.target = nil, ""
		for id, q := range m.pages {
			m.activeTab = id
			m.cdp, m.target = q.cdp, q.targetID
			break
		}
	}
	m.persistBrowserMeta(chatID)
	if len(m.pages) == 0 {
		m.stopLocked()
	}
	return nil
}

// KillChatTabs menutup semua tab browser milik chat.
func (m *Manager) KillChatTabs(chatID string) {
	m.mu.Lock()
	var ids []string
	for id, p := range m.pages {
		if p.chatID == chatID {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.CloseTab(chatID, id)
	}
}

// activePage returns CDP for operations; falls back to legacy m.cdp.
func (m *Manager) activePageLocked() *cdpClient {
	if m.activeTab != "" {
		if p := m.pages[m.activeTab]; p != nil && p.cdp != nil && p.cdp.alive() {
			return p.cdp
		}
	}
	return m.cdp
}

// OpenForChat: pastikan ada tab aktif untuk chat, lalu navigate.
func (m *Manager) OpenForChat(ctx context.Context, chatID, rawURL string, w, h int) (TabInfo, error) {
	m.mu.Lock()
	m.ensureTabsMap()
	var existing string
	for id, p := range m.pages {
		if p.chatID == chatID {
			existing = id
			if p.id == m.activeTab {
				break
			}
		}
	}
	metaID := ""
	if existing == "" && m.store != nil {
		for _, t := range m.store.GetToolsTabs(chatID).Browser {
			if metaID == "" {
				metaID = t.ID
			}
			if t.Active {
				metaID = t.ID
				break
			}
		}
	}
	m.mu.Unlock()

	targetTab := existing
	if targetTab == "" {
		targetTab = metaID
	}
	if targetTab == "" {
		info, err := m.CreateTab(ctx, chatID, "Browser", rawURL)
		if err != nil {
			return TabInfo{}, err
		}
		if w > 0 || h > 0 {
			m.mu.Lock()
			cctx, cancel := context.WithTimeout(ctx, callTimeout)
			m.setSizeLocked(cctx, w, h)
			cancel()
			m.mu.Unlock()
		}
		return info, nil
	}
	if _, err := m.ActivateTab(ctx, chatID, targetTab); err != nil {
		return TabInfo{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensure(ctx); err != nil {
		return TabInfo{}, err
	}
	m.last = time.Now()
	cctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	m.setSizeLocked(cctx, w, h)
	cdp := m.activePageLocked()
	if cdp == nil {
		return TabInfo{}, ErrNotRunning
	}
	if rawURL != "" {
		res, err := cdp.call(cctx, "Page.navigate", map[string]any{"url": rawURL})
		if err != nil {
			return TabInfo{}, err
		}
		var nav struct {
			ErrorText string `json:"errorText"`
		}
		if json.Unmarshal(res, &nav) == nil && nav.ErrorText != "" {
			return TabInfo{}, fmt.Errorf("gagal membuka halaman: %s", nav.ErrorText)
		}
	}
	p := m.pages[m.activeTab]
	if p == nil {
		return TabInfo{}, ErrNotRunning
	}
	p.updated = time.Now()
	return TabInfo{
		ID: m.activeTab, SessionID: chatID, Title: p.title, URL: m.pageURLLocked(p),
		Active: true, Alive: true, CreatedAt: p.created, UpdatedAt: p.updated,
	}, nil
}
