package api

import "testing"

func TestExtractMentionRefsInlineAndPunctuation(t *testing.T) {
	msg := "buatkan doc dengan #skill-a dan dijalankan menggunakan @agent-a. simpan ke [Folder: /x/folder-a]"
	a, s := extractMentionRefs(msg, nil, nil)
	if len(a) != 1 || a[0] != "agent-a" || len(s) != 1 || s[0] != "skill-a" {
		t.Fatalf("agents=%v skills=%v", a, s)
	}
}
