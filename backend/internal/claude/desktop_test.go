package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func writeDesktopFixture(t *testing.T, root string) (cliJabar, cliFolder string) {
	t.Helper()
	cliJabar = "11111111-1111-4111-8111-111111111111"
	cliFolder = "22222222-2222-4222-8222-222222222222"
	localJabar := "local_aaaa-bbbb-cccc-dddd"
	localFolder := "local_eeee-ffff-0000-1111"

	sessDir := filepath.Join(root, "claude-code-sessions", "org")
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(sessDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(localJabar+".json", `{
		"sessionId":"`+localJabar+`",
		"cliSessionId":"`+cliJabar+`",
		"originCwd":"/Users/me/Public/Working/Jabar",
		"title":"BIRO HUKUM"
	}`)
	mustWrite(localFolder+".json", `{
		"sessionId":"`+localFolder+`",
		"cliSessionId":"`+cliFolder+`",
		"originCwd":"/Users/me/Public/Working/Jabar/research",
		"title":"TESTING"
	}`)

	cfg := `{
	  "preferences": {
	    "epitaxyPrefs": {
	      "dframe-group-scopes": {
	        "scope/1": {
	          "groups": [
	            {"id":"cg-jabar","name":"JABAR"},
	            {"id":"cg-others","name":"others"}
	          ],
	          "assignments": {
	            "code:` + localJabar + `":"cg-jabar"
	          }
	        }
	      },
	      "dframe-code-sections": {
	        "scope/1": {
	          "sections": [
	            {"id":"cg-jabar","kind":"manual","name":"JABAR","order":3,"shown":true},
	            {"id":"cg-others","kind":"manual","name":"others","order":4,"shown":true},
	            {"id":"sessions","kind":"sessions","name":"","order":5,"shown":true}
	          ]
	        }
	      }
	    }
	  }
	}`
	if err := os.WriteFile(filepath.Join(root, "claude_desktop_config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return cliJabar, cliFolder
}

func TestApplyDesktopGroupsMatchesDesktopSidebar(t *testing.T) {
	desk := t.TempDir()
	cliJabar, cliFolder := writeDesktopFixture(t, desk)

	in := []Session{
		{ID: cliJabar, Title: "dari-jsonl", Cwd: "/tmp/wrong", Group: "wrong"},
		{ID: cliFolder, Title: "dari-jsonl-2", Cwd: "/tmp/wrong2", Group: "wrong"},
		{ID: "33333333-3333-4333-8333-333333333333", Title: "hantu", Cwd: "/tmp/x"}, // tidak di Desktop
	}
	out := applyDesktopGroups(in, desk, nil)
	if len(out) != 2 {
		t.Fatalf("hanya session indeks Desktop yang tampil: %d", len(out))
	}

	byID := map[string]Session{}
	for _, s := range out {
		byID[s.ID] = s
	}
	j := byID[cliJabar]
	if j.Group != "JABAR" || j.Title != "BIRO HUKUM" || j.GroupOrder != 3 {
		t.Fatalf("assigned: %+v", j)
	}
	if j.Cwd != "/Users/me/Public/Working/Jabar" {
		t.Fatalf("cwd Desktop: %q", j.Cwd)
	}
	f := byID[cliFolder]
	if f.Group != "research" || f.Title != "TESTING" {
		t.Fatalf("unassigned harus folder-label + title Desktop: %+v", f)
	}
}

func TestApplyDesktopGroupsKeepsAyumiLocalSessions(t *testing.T) {
	desk := t.TempDir()
	writeDesktopFixture(t, desk)
	localID := "44444444-4444-4444-8444-444444444444"
	in := []Session{{ID: localID, Title: "Lokal", Cwd: "/Users/me/proj/app"}}
	out := applyDesktopGroups(in, desk, map[string]string{localID: "/Users/me/proj/app"})
	if len(out) != 1 || out[0].Group != "app" {
		t.Fatalf("session Ayumi Link tetap tampil: %+v", out)
	}
}
