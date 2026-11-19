package claude

import (
	"encoding/json"
	"strings"
)

type streamEvent struct {
	kind  string // delta | tool | assistant | result | ""
	text  string
	isErr bool
}

// parseStreamLine menerjemahkan satu baris `--output-format stream-json` dari claude.
func parseStreamLine(line []byte) streamEvent {
	var ev map[string]any
	if json.Unmarshal(line, &ev) != nil {
		return streamEvent{}
	}
	switch ev["type"] {
	case "stream_event":
		e, _ := ev["event"].(map[string]any)
		if e == nil {
			return streamEvent{}
		}
		switch e["type"] {
		case "content_block_delta":
			d, _ := e["delta"].(map[string]any)
			if d != nil && d["type"] == "text_delta" {
				if t, _ := d["text"].(string); t != "" {
					return streamEvent{kind: "delta", text: t}
				}
			}
		case "content_block_start":
			cb, _ := e["content_block"].(map[string]any)
			if cb != nil && cb["type"] == "tool_use" {
				name, _ := cb["name"].(string)
				if i := strings.LastIndex(name, "__"); i >= 0 && i+2 < len(name) {
					name = name[i+2:]
				}
				return streamEvent{kind: "tool", text: name}
			}
		}
	case "assistant":
		if m, _ := ev["message"].(map[string]any); m != nil {
			txt, _, _, _ := extractParts(m["content"])
			return streamEvent{kind: "assistant", text: txt}
		}
	case "result":
		isErr, _ := ev["is_error"].(bool)
		res, _ := ev["result"].(string)
		return streamEvent{kind: "result", text: res, isErr: isErr}
	}
	return streamEvent{}
}
