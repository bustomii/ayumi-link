package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"ayumi-link/internal/claude"
)

const heartbeat = 15 * time.Second

// chat menjalankan claude dan menyiarkan hasilnya lewat Server-Sent Events.
func (s *Server) chat(c *fiber.Ctx) error {
	var b struct {
		SessionID      string   `json:"sessionId"`
		Cwd            string   `json:"cwd"`
		Message        string   `json:"message"`
		PermissionMode string   `json:"permissionMode"`
		Model          string   `json:"model"`
		Effort         string   `json:"effort"`
		Attachments    []string `json:"attachments"`
		Agents         []string `json:"agents"` // slug/id/nama @agent
		Skills         []string `json:"skills"` // slug/id/nama #skill
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}

	msg := b.Message
	agentRefs, skillRefs := extractMentionRefs(b.Message, b.Agents, b.Skills)
	if len(agentRefs) > 0 || len(skillRefs) > 0 {
		s.syncDisk()
		ags, sks := s.Store.ResolveMentions(agentRefs, skillRefs)
		if prefix := BuildAgentsSkillsPrompt(ags, sks); prefix != "" {
			msg = prefix + msg
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	events, err := s.Runner.Start(ctx, claude.ChatOptions{
		SessionID: b.SessionID, Cwd: b.Cwd, Message: msg,
		PermissionMode: b.PermissionMode, Model: b.Model, Effort: b.Effort, Attachments: b.Attachments,
	})
	if err != nil {
		cancel()
		var re *claude.RequestError
		if errors.As(err, &re) {
			return fiber.NewError(re.Status, re.Message)
		}
		return err
	}

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache, no-transform")
	c.Set("X-Accel-Buffering", "no")
	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		// Klien terputus => penulisan gagal => cancel() menghentikan proses claude.
		defer cancel()
		tick := time.NewTicker(heartbeat)
		defer tick.Stop()
		write := func(format string, args ...any) bool {
			if _, err := fmt.Fprintf(w, format, args...); err != nil {
				return false
			}
			return w.Flush() == nil
		}
		for {
			select {
			case ev, ok := <-events:
				if !ok {
					return
				}
				data, _ := json.Marshal(ev.Data)
				if !write("event: %s\ndata: %s\n\n", ev.Kind, data) {
					return
				}
			case <-tick.C:
				if !write(": ping\n\n") { // menjaga koneksi lewat proxy/tunnel
					return
				}
			}
		}
	}))
	return nil
}
