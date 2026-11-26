package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Session adalah ringkasan satu percakapan Claude Code.
type Session struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Cwd        string    `json:"cwd"`
	Project    string    `json:"project"`
	Group      string    `json:"group"`
	GroupOrder int       `json:"groupOrder"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Live       bool      `json:"live"`
	Status     string    `json:"status,omitempty"`
	Local      bool      `json:"local"` // dibuat lewat Ayumi Link (bisa dihapus permanen)
	// CwdMissing: folder kerja session sudah tidak ada / tidak diizinkan; perlu fork ke folder lain.
	CwdMissing bool `json:"cwdMissing,omitempty"`
}

// Message adalah satu pesan yang sudah dibersihkan dari noise tool.
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Tools     []string  `json:"tools,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	// Attachments: lampiran yang dikirim user pada pesan ini (dibaca dari blok lampiran di transcript).
	Attachments []Attachment `json:"attachments,omitempty"`
}

func cwdFromJSONL(jsonlPath string) string {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxJSONLLine)
	for sc.Scan() {
		var raw map[string]any
		if json.Unmarshal(sc.Bytes(), &raw) != nil {
			continue
		}
		if cwd, ok := raw["cwd"].(string); ok && cwd != "" {
			return cwd
		}
	}
	return ""
}

func projectKeyToCwd(key string) string {
	// Lossy fallback: Claude encodes "/" as "-". Prefer cwdFromJSONL.
	if key == "" {
		return ""
	}
	return "/" + strings.ReplaceAll(strings.TrimPrefix(key, "-"), "-", "/")
}

func readCustomTitle(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "custom-title.json"))
	if err != nil {
		return ""
	}
	var t struct {
		CustomTitle string `json:"customTitle"`
	}
	if json.Unmarshal(b, &t) != nil {
		return ""
	}
	return t.CustomTitle
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func firstUserPrompt(jsonlPath string) string {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxJSONLLine)
	for sc.Scan() {
		var raw map[string]any
		if json.Unmarshal(sc.Bytes(), &raw) != nil {
			continue
		}
		if raw["type"] != "user" {
			continue
		}
		msg, _ := raw["message"].(map[string]any)
		if msg == nil {
			continue
		}
		text, atts := SplitAttachments(strings.TrimSpace(extractText(msg["content"])))
		if text == "" && len(atts) > 0 {
			text = "Lampiran: " + atts[0].Name
		}
		if text != "" {
			return truncate(text, 80)
		}
	}
	return ""
}

var imageMetaRe = regexp.MustCompile(`(?is)\[Image:[^\]]*\]`)

// ayumiContextRe: blok konteks Agents & Skills yang disisipkan Ayumi Link di depan pesan user.
// Hanya untuk Claude; tidak ditampilkan di chat.
var ayumiContextRe = regexp.MustCompile(`(?s)^\s*\[Konteks Ayumi Link[^\]]*\].*?\[/Konteks Ayumi Link\]\s*`)

func cleanUserText(s string) string {
	s = ayumiContextRe.ReplaceAllString(s, "")
	s = imageMetaRe.ReplaceAllString(s, "📷 [gambar]")
	// Fallback if path/brackets odd
	if strings.Contains(strings.ToLower(s), "[image:") {
		s = regexp.MustCompile(`(?is)\[Image:.*?\]`).ReplaceAllString(s, "📷 [gambar]")
	}
	s = strings.TrimSpace(s)
	// Collapse duplicate placeholders
	for strings.Contains(s, "📷 [gambar]\n📷 [gambar]") {
		s = strings.ReplaceAll(s, "📷 [gambar]\n📷 [gambar]", "📷 [gambar]")
	}
	return s
}

func extractParts(content any) (text string, tools []string, onlyTools bool, onlyToolResult bool) {
	switch c := content.(type) {
	case string:
		return cleanUserText(c), nil, false, false
	case []any:
		var b strings.Builder
		hasText := false
		hasTool := false
		hasToolResult := false
		hasOther := false
		for _, part := range c {
			m, _ := part.(map[string]any)
			if m == nil {
				continue
			}
			switch m["type"] {
			case "text":
				if t, ok := m["text"].(string); ok {
					t = cleanUserText(t)
					if t == "" {
						continue
					}
					hasText = true
					if b.Len() > 0 {
						b.WriteByte('\n')
					}
					b.WriteString(t)
				}
			case "tool_use":
				hasTool = true
				name, _ := m["name"].(string)
				if name == "" {
					name = "tool"
				}
				// short display name: mcp__x__y → y, Bash → Bash
				if i := strings.LastIndex(name, "__"); i >= 0 && i+2 < len(name) {
					name = name[i+2:]
				}
				tools = append(tools, name)
			case "tool_result":
				hasToolResult = true
			case "thinking":
				// skip internal thinking
			case "image":
				hasOther = true
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString("📷 [gambar]")
			default:
				hasOther = true
			}
		}
		text = b.String()
		onlyTools = hasTool && !hasText && !hasOther && !hasToolResult
		onlyToolResult = hasToolResult && !hasText && !hasTool && !hasOther
		return text, tools, onlyTools, onlyToolResult
	default:
		return "", nil, false, false
	}
}

func extractText(content any) string {
	text, _, _, _ := extractParts(content)
	return text
}

// LoadMessages membaca transcript .jsonl dan mengembalikan pesan user/assistant.
func LoadMessages(jsonlPath string) ([]Message, error) {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var msgs []Message
	var pendingTools []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxJSONLLine)
	for sc.Scan() {
		var raw map[string]any
		if json.Unmarshal(sc.Bytes(), &raw) != nil {
			continue
		}
		typ, _ := raw["type"].(string)
		if typ != "user" && typ != "assistant" {
			continue
		}
		msg, _ := raw["message"].(map[string]any)
		if msg == nil {
			continue
		}
		role, _ := msg["role"].(string)
		if role == "" {
			role = typ
		}
		if isSyntheticStub(raw, msg) {
			continue
		}
		content, tools, onlyTools, onlyToolResult := extractParts(msg["content"])
		content = strings.TrimSpace(content)
		var atts []Attachment
		if role == "user" {
			content, atts = SplitAttachments(content)
		}

		// Skip tool plumbing noise
		if onlyToolResult {
			continue
		}
		if onlyTools {
			pendingTools = append(pendingTools, tools...)
			continue
		}
		if (content == "" && len(atts) == 0) || content == "📷 [gambar]" {
			continue
		}
		if strings.HasPrefix(content, "[tool:") && !strings.Contains(content, "\n") {
			continue
		}

		id, _ := raw["uuid"].(string)
		tsStr, _ := raw["timestamp"].(string)
		ts, _ := time.Parse(time.RFC3339Nano, tsStr)
		if ts.IsZero() {
			ts, _ = time.Parse(time.RFC3339, tsStr)
		}

		attached := tools
		if role == "assistant" && len(pendingTools) > 0 {
			attached = append(append([]string{}, pendingTools...), tools...)
			pendingTools = nil
		} else if role == "user" {
			pendingTools = nil
		}

		if id == "" {
			id = "msg-" + strconv.Itoa(len(msgs))
		}

		m := Message{
			ID:          id,
			Role:        role,
			Content:     content,
			Tools:       uniqueStrings(attached),
			Timestamp:   ts,
			Attachments: atts,
		}
		if role == "user" {
			msgs = placeUserBeforeEarlyReplies(msgs, m)
		} else {
			msgs = append(msgs, m)
		}
	}
	return msgs, sc.Err()
}

// Paginate mengambil `limit` pesan terakhir sebelum `beforeID`.
func Paginate(all []Message, limit int, beforeID string) (page []Message, hasMore bool) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	n := len(all)
	if n == 0 {
		return nil, false
	}

	end := n
	if beforeID != "" {
		idx := -1
		for i, m := range all {
			if m.ID == beforeID {
				idx = i
				break
			}
		}
		if idx <= 0 {
			return nil, false
		}
		end = idx
	}

	start := end - limit
	if start < 0 {
		start = 0
	}
	page = all[start:end]
	hasMore = start > 0
	return page, hasMore
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// isSyntheticStub mengenali balasan sintetis "No response requested." yang ditulis Claude Code
// tepat sebelum pesan user saat session dilanjutkan. Itu bukan balasan nyata dan membuat balasan
// tampil di atas pesan user. Pesan error API sintetis tetap ditampilkan.
func isSyntheticStub(raw, msg map[string]any) bool {
	if model, _ := msg["model"].(string); model != "<synthetic>" {
		return false
	}
	if isErr, _ := raw["isApiErrorMessage"].(bool); isErr {
		return false
	}
	text, _, _, _ := extractParts(msg["content"])
	return strings.TrimSpace(text) == "No response requested."
}

// placeUserBeforeEarlyReplies memastikan balasan tidak pernah tampil di atas pesan user yang
// dibalasnya. Balasan asli selalu lebih baru dari pesan user; bila ada balasan tepat di depan
// dengan waktu lebih baru dari pesan user ini, urutannya terbalik di transcript,
// sehingga pesan user disisipkan di depannya.
func placeUserBeforeEarlyReplies(msgs []Message, user Message) []Message {
	if user.Timestamp.IsZero() {
		return append(msgs, user)
	}
	i := len(msgs)
	for i > 0 && msgs[i-1].Role == "assistant" && !msgs[i-1].Timestamp.IsZero() && msgs[i-1].Timestamp.After(user.Timestamp) {
		i--
	}
	msgs = append(msgs, Message{})
	copy(msgs[i+1:], msgs[i:])
	msgs[i] = user
	return msgs
}

// maxJSONLLine: batas satu baris transcript/stream. Tool result (mis. file besar atau gambar base64)
// bisa mencapai beberapa MB; batas kecil membuat seluruh session gagal dimuat dengan 500.
const maxJSONLLine = 64 * 1024 * 1024
