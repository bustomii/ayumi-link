package api

import (
	"encoding/json"
	"testing"
)

func TestClaudeAuthAPI(t *testing.T) {
	e := newEnv(t)
	if res, _ := e.do(t, "GET", "/api/claude/auth", "", "", nil); res.StatusCode != 401 {
		t.Fatalf("tanpa cookie harus 401, got %d", res.StatusCode)
	}

	cookie, _, _ := setupAccount(t, e)
	res, body := e.do(t, "GET", "/api/claude/auth", "", cookie, nil)
	if res.StatusCode != 200 {
		t.Fatalf("GET status: %d %v", res.StatusCode, body)
	}

	res, body = e.do(t, "PUT", "/api/claude/auth", `{"method":"oauth_token","token":"short"}`, cookie, nil)
	if res.StatusCode != 400 {
		t.Fatalf("token pendek harus 400, got %d %v", res.StatusCode, body)
	}

	tok := "sk-ant-oat01-test-token-abcdefghijklmnop"
	res, body = e.do(t, "PUT", "/api/claude/auth", `{"method":"oauth_token","token":"`+tok+`"}`, cookie, nil)
	if res.StatusCode != 200 {
		t.Fatalf("PUT: %d %v", res.StatusCode, body)
	}
	if body["loggedIn"] != true || body["stored"] != true {
		b, _ := json.Marshal(body)
		t.Fatalf("harus loggedIn+stored: %s", b)
	}

	res, body = e.do(t, "DELETE", "/api/claude/auth", "", cookie, nil)
	if res.StatusCode != 200 {
		t.Fatalf("DELETE: %d %v", res.StatusCode, body)
	}
	if body["stored"] == true {
		t.Fatalf("setelah delete stored harus false: %v", body)
	}
}
