package claude

import "testing"

func TestValidModelCuratedAndCustom(t *testing.T) {
	ok := []string{
		"claude-sonnet-5-5",
		"claude-fable-5-2",
		"claude-sonnet-4-20250514",
		"claude-3-5-haiku-20241022",
		"claude-opus-4-1",
	}
	for _, m := range ok {
		if !ValidModel(m) {
			t.Fatalf("expected valid: %q", m)
		}
	}
	bad := []string{"", "gpt-4", "Claude-Sonnet", "claude-", "claude- Bad", "xclaude-sonnet-5"}
	for _, m := range bad {
		if ValidModel(m) {
			t.Fatalf("expected invalid: %q", m)
		}
	}
}
