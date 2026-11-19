package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadContextUsage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	data := `{"message":{"role":"assistant","model":"claude-sonnet-5-5","usage":{"input_tokens":2,"cache_creation_input_tokens":100,"cache_read_input_tokens":900}}}
{"message":{"role":"user","content":"hi"}}
{"isSidechain":true,"message":{"role":"assistant","model":"x","usage":{"input_tokens":99999}}}
`
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	c := ReadContextUsage(p)
	if c == nil || c.Used != 1002 || c.Window != 200000 {
		t.Fatalf("got %+v", c)
	}
}
