package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMessagesSkipsSyntheticStub(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	data := `{"type":"assistant","uuid":"a1","timestamp":"2026-11-23T00:31:08Z","message":{"role":"assistant","model":"<synthetic>","content":[{"type":"text","text":"No response requested."}]}}
{"type":"user","uuid":"u1","timestamp":"2026-11-23T00:31:08Z","message":{"role":"user","content":"halo"}}
{"type":"assistant","uuid":"a2","timestamp":"2026-11-23T00:31:09Z","isApiErrorMessage":true,"message":{"role":"assistant","model":"<synthetic>","content":[{"type":"text","text":"API Error: x"}]}}
{"type":"assistant","uuid":"a3","timestamp":"2026-11-23T00:31:10Z","message":{"role":"assistant","model":"claude-sonnet-5-5","content":[{"type":"text","text":"No response requested."}]}}
`
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	msgs, err := LoadMessages(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 || msgs[0].ID != "u1" || msgs[1].ID != "a2" || msgs[2].ID != "a3" {
		t.Fatalf("got %+v", msgs)
	}
}

func TestLoadMessagesUserNeverAfterItsReply(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	data := `{"type":"assistant","uuid":"a0","timestamp":"2026-11-23T00:30:00Z","message":{"role":"assistant","model":"claude-sonnet-5-5","content":[{"type":"text","text":"lama"}]}}
{"type":"assistant","uuid":"a1","timestamp":"2026-11-23T00:31:09Z","message":{"role":"assistant","model":"claude-sonnet-5-5","content":[{"type":"text","text":"terlalu awal"}]}}
{"type":"user","uuid":"u1","timestamp":"2026-11-23T00:31:08Z","message":{"role":"user","content":"halo"}}
{"type":"assistant","uuid":"a2","timestamp":"2026-11-23T00:31:12Z","message":{"role":"assistant","model":"claude-sonnet-5-5","content":[{"type":"text","text":"balasan"}]}}
`
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	msgs, err := LoadMessages(p)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}
	if len(ids) != 4 || ids[0] != "a0" || ids[1] != "u1" || ids[2] != "a1" || ids[3] != "a2" {
		t.Fatalf("urutan salah: %v", ids)
	}
}

func TestLoadMessagesHidesAyumiContextBlock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	data := `{"type":"user","uuid":"u1","timestamp":"2026-11-23T00:27:10Z","message":{"role":"user","content":"[Konteks Ayumi Link — Agents & Skills]\nIkuti peran.\n\n## Agent @git-commit (git-commit)\nisi\n\n[/Konteks Ayumi Link]\n\npush perubahan @git-commit"}}
`
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	msgs, err := LoadMessages(p)
	if err != nil || len(msgs) != 1 || msgs[0].Content != "push perubahan @git-commit" {
		t.Fatalf("got %+v %v", msgs, err)
	}
}
