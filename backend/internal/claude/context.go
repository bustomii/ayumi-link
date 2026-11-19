package claude

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
)

// ContextInfo: pemakaian jendela konteks pada giliran terakhir sebuah session.
type ContextInfo struct {
	Used   int    `json:"used"`   // token input giliran terakhir (baru + cache)
	Window int    `json:"window"` // ukuran jendela konteks model
	Model  string `json:"model,omitempty"`
}

const contextTailBytes = 4 << 20

// ReadContextUsage membaca usage dari pesan assistant terakhir (bagian ekor file saja).
// Mengembalikan nil bila tidak ada data usage.
func ReadContextUsage(jsonlPath string) *ContextInfo {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil
	}
	start := st.Size() - contextTailBytes
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil
	}
	buf, err := io.ReadAll(f)
	if err != nil {
		return nil
	}
	lines := bytes.Split(buf, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if len(line) == 0 || !bytes.Contains(line, []byte(`"usage"`)) {
			continue
		}
		var e struct {
			IsSidechain bool `json:"isSidechain"`
			Message     struct {
				Role  string `json:"role"`
				Model string `json:"model"`
				Usage struct {
					In     int `json:"input_tokens"`
					Create int `json:"cache_creation_input_tokens"`
					Read   int `json:"cache_read_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &e) != nil || e.IsSidechain || e.Message.Role != "assistant" {
			continue
		}
		if strings.HasPrefix(e.Message.Model, "<") { // "<synthetic>"
			continue
		}
		used := e.Message.Usage.In + e.Message.Usage.Create + e.Message.Usage.Read
		if used <= 0 {
			continue
		}
		return &ContextInfo{Used: used, Window: contextWindow(e.Message.Model, used), Model: e.Message.Model}
	}
	return nil
}

// contextWindow: 200k secara default; bila pemakaian sudah melewatinya, jendela 1M.
func contextWindow(model string, used int) int {
	if strings.Contains(model, "[1m]") || used > 200_000 {
		return 1_000_000
	}
	return 200_000
}
