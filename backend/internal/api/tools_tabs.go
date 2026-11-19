package api

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/claude"
	"ayumi-link/internal/store"
	"ayumi-link/internal/termpty"
)

func (s *Server) toolsTabsReady() error {
	if s.Term == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "terminal penuh tidak tersedia")
	}
	return nil
}

func (s *Server) listToolsTabs(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	term := []termpty.TabInfo{}
	if s.Term != nil {
		term = s.Term.ListTabs(id)
	}
	browser := []fiber.Map{}
	if s.Web != nil {
		for _, t := range s.Web.ListTabs(id) {
			browser = append(browser, fiber.Map{
				"id": t.ID, "title": t.Title, "url": t.URL, "active": t.Active,
				"alive": t.Alive, "createdAt": t.CreatedAt, "updatedAt": t.UpdatedAt,
			})
		}
	} else {
		// metadata saja dari store
		st := s.Store.GetToolsTabs(id)
		for _, t := range st.Browser {
			browser = append(browser, fiber.Map{
				"id": t.ID, "title": t.Title, "url": t.URL, "active": t.Active,
				"alive": false, "createdAt": t.CreatedAt, "updatedAt": t.UpdatedAt,
			})
		}
	}
	return c.JSON(fiber.Map{"terminal": term, "browser": browser})
}

func (s *Server) createTerminalTab(c *fiber.Ctx) error {
	if err := s.toolsTabsReady(); err != nil {
		return err
	}
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	var b struct {
		Title string `json:"title"`
	}
	_ = c.BodyParser(&b)
	info, err := s.Term.CreateTab(id, strings.TrimSpace(b.Title))
	if err != nil {
		if errors.Is(err, termpty.ErrLimit) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(info)
}

func (s *Server) deleteTerminalTab(c *fiber.Ctx) error {
	if err := s.toolsTabsReady(); err != nil {
		return err
	}
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	tabID := strings.TrimSpace(c.Params("tabId"))
	if tabID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tabId wajib")
	}
	if !s.Term.Owns(id, tabID) {
		// masih boleh hapus meta
		st := s.Store.GetToolsTabs(id)
		filtered := st.Terminal[:0]
		for _, t := range st.Terminal {
			if t.ID != tabID {
				filtered = append(filtered, t)
			}
		}
		st.Terminal = filtered
		_ = s.Store.SetToolsTabs(id, st)
		return c.JSON(fiber.Map{"ok": true})
	}
	if err := s.Term.CloseTab(tabID); err != nil && !errors.Is(err, termpty.ErrNotFound) {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) handleTerminalWS(conn *websocket.Conn) {
	chatID := strings.TrimSpace(conn.Query("sessionId"))
	tabID := strings.TrimSpace(conn.Query("tabId"))
	if s.Term == nil {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "terminal tidak tersedia"))
		return
	}
	if !claude.IsUUID(chatID) || tabID == "" {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1008, "sessionId/tabId wajib"))
		return
	}
	if !s.Term.Owns(chatID, tabID) {
		if _, err := s.Term.EnsureTab(chatID, tabID); err != nil {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1011, err.Error()))
			return
		}
	}
	scroll, out, detach, err := s.Term.Attach(tabID)
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1011, err.Error()))
		return
	}
	defer detach()

	if len(scroll) > 0 {
		_ = conn.WriteMessage(websocket.BinaryMessage, scroll)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for chunk := range out {
			if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
				return
			}
		}
	}()

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if mt == websocket.TextMessage {
			var ctrl struct {
				Type string `json:"type"`
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
				Data string `json:"data"`
			}
			if json.Unmarshal(msg, &ctrl) == nil {
				switch ctrl.Type {
				case "resize":
					_ = s.Term.Resize(tabID, ctrl.Cols, ctrl.Rows)
				case "input":
					_ = s.Term.Write(tabID, []byte(ctrl.Data))
				}
				continue
			}
			_ = s.Term.Write(tabID, msg)
			continue
		}
		if mt == websocket.BinaryMessage {
			_ = s.Term.Write(tabID, msg)
		}
	}
	<-done
}

func (s *Server) createBrowserTab(c *fiber.Ctx) error {
	if s.Web == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "browser tidak tersedia")
	}
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	var b struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	_ = c.BodyParser(&b)
	url := ""
	if strings.TrimSpace(b.URL) != "" {
		var e error
		if url, e = normalizeWebURL(b.URL); e != nil {
			return fiber.NewError(fiber.StatusBadRequest, e.Error())
		}
	}
	info, err := s.Web.CreateTab(c.UserContext(), id, strings.TrimSpace(b.Title), url)
	if err != nil {
		return s.webErr(err)
	}
	return c.JSON(info)
}

func (s *Server) deleteBrowserTab(c *fiber.Ctx) error {
	if s.Web == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "browser tidak tersedia")
	}
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	tabID := strings.TrimSpace(c.Params("tabId"))
	if err := s.Web.CloseTab(id, tabID); err != nil {
		return s.webErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) activateBrowserTab(c *fiber.Ctx) error {
	if s.Web == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "browser tidak tersedia")
	}
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	tabID := strings.TrimSpace(c.Params("tabId"))
	info, err := s.Web.ActivateTab(c.UserContext(), id, tabID)
	if err != nil {
		return s.webErr(err)
	}
	return c.JSON(info)
}

// touchToolsMeta helper untuk update browser meta timestamps.
func touchMeta(t *store.ToolTabMeta) {
	t.UpdatedAt = time.Now()
}
