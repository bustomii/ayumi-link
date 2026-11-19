package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ayumi-link/internal/store"
)

const sid = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"

func writeTranscript(t *testing.T, home, id string, lines ...string) string {
	t.Helper()
	dir := filepath.Join(home, "projects", "-tmp-demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func line(role, uuid, content string) string {
	return fmt.Sprintf(`{"type":"%s","uuid":"%s","timestamp":"2026-11-18T10:00:00Z","cwd":"/tmp/demo","message":{"role":"%s","content":%s}}`, role, uuid, role, content)
}

func TestLoadMessagesFiltersToolNoise(t *testing.T) {
	home := t.TempDir()
	p := writeTranscript(t, home, sid,
		line("user", "u1", `"halo"`),
		line("assistant", "a1", `[{"type":"tool_use","name":"mcp__x__Bash","input":{}}]`),
		line("user", "u2", `[{"type":"tool_result","content":"ok"}]`),
		line("assistant", "a2", `[{"type":"text","text":"selesai"}]`),
	)
	msgs, err := LoadMessages(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Content != "halo" || msgs[1].Content != "selesai" {
		t.Fatalf("pesan = %+v", msgs)
	}
	if len(msgs[1].Tools) != 1 || msgs[1].Tools[0] != "Bash" {
		t.Fatalf("tool harus menempel pada balasan berikutnya: %+v", msgs[1].Tools)
	}
}

func TestPaginate(t *testing.T) {
	var all []Message
	for i := 0; i < 45; i++ {
		all = append(all, Message{ID: fmt.Sprintf("m%02d", i)})
	}
	page, more := Paginate(all, 20, "")
	if len(page) != 20 || page[0].ID != "m25" || !more {
		t.Fatalf("halaman terakhir salah: %d %s %v", len(page), page[0].ID, more)
	}
	page, more = Paginate(all, 20, "m25")
	if len(page) != 20 || page[0].ID != "m05" || !more {
		t.Fatalf("halaman sebelumnya salah: %d %s %v", len(page), page[0].ID, more)
	}
	page, more = Paginate(all, 20, "m05")
	if len(page) != 5 || more {
		t.Fatalf("halaman awal salah: %d %v", len(page), more)
	}
}

func newService(t *testing.T) (*Service, string) {
	t.Helper()
	home := t.TempDir()
	return &Service{Home: home, DesktopDir: t.TempDir(), Store: store.New(t.TempDir())}, home
}

func TestDeleteLocalIsPermanentDesktopIsHidden(t *testing.T) {
	svc, home := newService(t)
	local := writeTranscript(t, home, sid, line("user", "u1", `"halo"`))
	other := "bbbbbbbb-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	desktop := writeTranscript(t, home, other, line("user", "u1", `"dari desktop"`))
	svc.Store.Register(sid, "/tmp/demo")

	if mode, err := svc.Delete(sid); err != nil || mode != "deleted" {
		t.Fatalf("local: %q %v", mode, err)
	}
	if _, err := os.Stat(local); !os.IsNotExist(err) {
		t.Fatal("transcript session lokal harus terhapus")
	}
	if _, ok := svc.Store.Registry()[sid]; ok {
		t.Fatal("registry harus bersih")
	}

	if mode, err := svc.Delete(other); err != nil || mode != "hidden" {
		t.Fatalf("desktop: %q %v", mode, err)
	}
	if _, err := os.Stat(desktop); err != nil {
		t.Fatal("transcript session Desktop tidak boleh dihapus")
	}
	for _, s := range svc.List() {
		if s.ID == other {
			t.Fatal("session Desktop yang disembunyikan tidak boleh tampil")
		}
	}
}

func TestSetGroupOverride(t *testing.T) {
	svc, home := newService(t)
	writeTranscript(t, home, sid, line("user", "u1", `"halo"`))
	svc.Store.Register(sid, "/tmp/demo")

	find := func() Session {
		for _, s := range svc.List() {
			if s.ID == sid {
				return s
			}
		}
		t.Fatal("session tidak ditemukan")
		return Session{}
	}
	if g := find().Group; g != "demo" {
		t.Fatalf("group awal = %q", g)
	}
	if err := svc.SetGroup(sid, "Proyek Baru"); err != nil {
		t.Fatal(err)
	}
	if s := find(); s.Group != "Proyek Baru" || !s.Local {
		t.Fatalf("setelah pindah: %+v", s)
	}
	_ = svc.SetGroup(sid, "")
	if g := find().Group; g != "demo" {
		t.Fatalf("reset group = %q", g)
	}
}

func TestEffortAdjustments(t *testing.T) {
	cases := []struct{ model, effort, want string }{
		{"claude-opus-5-5", "xhigh", "xhigh"},
		{"claude-opus-4-6", "xhigh", "high"},
		{"claude-sonnet-4-6", "max", "max"},
		{"claude-haiku-4-5", "medium", ""},
		{"claude-opus-5-5", "ngawur", ""},
	}
	for _, c := range cases {
		if got := effortFor(c.model, c.effort); got != c.want {
			t.Errorf("effortFor(%s,%s)=%q, want %q", c.model, c.effort, got, c.want)
		}
	}
}

func TestParseStreamLine(t *testing.T) {
	if ev := parseStreamLine([]byte(`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hal"}}}`)); ev.kind != "delta" || ev.text != "Hal" {
		t.Errorf("delta: %+v", ev)
	}
	if ev := parseStreamLine([]byte(`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"mcp__a__WebFetch"}}}`)); ev.kind != "tool" || ev.text != "WebFetch" {
		t.Errorf("tool: %+v", ev)
	}
	if ev := parseStreamLine([]byte(`{"type":"result","is_error":true,"result":"gagal"}`)); ev.kind != "result" || !ev.isErr {
		t.Errorf("result: %+v", ev)
	}
	if ev := parseStreamLine([]byte(`bukan json`)); ev.kind != "" {
		t.Errorf("baris rusak harus diabaikan: %+v", ev)
	}
}

func TestVersionOrdering(t *testing.T) {
	if !versionLess("2.1.235", "2.1.325") || versionLess("2.1.325", "2.1.235") || !versionLess("2.1.9", "2.1.10") {
		t.Fatal("urutan versi salah")
	}
}
