package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvLine(t *testing.T) {
	cases := []struct {
		line, key, val string
		ok             bool
	}{
		{"PORT=8081", "PORT", "8081", true},
		{"  PORT = 8081  ", "PORT", "8081", true},
		{"export HOST=0.0.0.0", "HOST", "0.0.0.0", true},
		{`NAME="dengan spasi # bukan komentar"`, "NAME", "dengan spasi # bukan komentar", true},
		{`NAME='kutip tunggal'`, "NAME", "kutip tunggal", true},
		{"KEY=nilai # komentar akhir", "KEY", "nilai", true},
		{"KEY=a#b", "KEY", "a#b", true}, // '#' tanpa spasi bukan komentar
		{"EMPTY=", "EMPTY", "", true},
		{"AYUMI_ALLOWED_IPS=1.2.3.4,10.0.0.0/8", "AYUMI_ALLOWED_IPS", "1.2.3.4,10.0.0.0/8", true},
		{"# komentar", "", "", false},
		{"", "", "", false},
		{"tanpa-sama-dengan", "", "", false},
		{"=nilai", "", "", false},
		{"BAD KEY=1", "", "", false},
	}
	for _, c := range cases {
		k, v, ok := parseEnvLine(c.line)
		if k != c.key || v != c.val || ok != c.ok {
			t.Errorf("parseEnvLine(%q) = %q,%q,%v; want %q,%q,%v", c.line, k, v, ok, c.key, c.val, c.ok)
		}
	}
}

func TestLoadDotEnvDoesNotOverrideRealEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.env")
	body := "AYUMI_TEST_A=dari-file\nAYUMI_TEST_B=dari-file\n# komentar\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AYUMI_ENV_FILE", path)
	t.Setenv("AYUMI_TEST_A", "dari-env") // env sungguhan harus menang
	os.Unsetenv("AYUMI_TEST_B")
	t.Cleanup(func() { os.Unsetenv("AYUMI_TEST_B") })

	got, err := LoadDotEnv()
	if err != nil || got != path {
		t.Fatalf("LoadDotEnv = %q, %v", got, err)
	}
	if v := os.Getenv("AYUMI_TEST_A"); v != "dari-env" {
		t.Fatalf("env sungguhan harus menang: %q", v)
	}
	if v := os.Getenv("AYUMI_TEST_B"); v != "dari-file" {
		t.Fatalf("variabel yang belum di-set harus diisi dari file: %q", v)
	}
}

func TestLoadDotEnvMissingFile(t *testing.T) {
	t.Chdir(t.TempDir())
	os.Unsetenv("AYUMI_ENV_FILE")
	if p, err := LoadDotEnv(); err != nil || p != "" {
		t.Fatalf("tanpa .env harus diam saja: %q %v", p, err)
	}
	t.Setenv("AYUMI_ENV_FILE", "/tidak/ada.env")
	if _, err := LoadDotEnv(); err == nil {
		t.Fatal("path eksplisit yang tidak ada harus error")
	}
}

func TestIPList(t *testing.T) {
	l, err := ParseIPList([]string{"203.0.113.7", "192.168.1.0/24", "::1", "2001:db8::/32", " "})
	if err != nil {
		t.Fatal(err)
	}
	allow := []string{"203.0.113.7", "192.168.1.50", "::1", "2001:db8::1234", "::ffff:203.0.113.7"}
	deny := []string{"203.0.113.8", "192.168.2.1", "127.0.0.1", "2001:db9::1", "bukan-ip", ""}
	for _, ip := range allow {
		if !l.Allows(ip) {
			t.Errorf("%q seharusnya diizinkan", ip)
		}
	}
	for _, ip := range deny {
		if l.Allows(ip) {
			t.Errorf("%q seharusnya ditolak", ip)
		}
	}
}

func TestIPListEmptyAllowsAll(t *testing.T) {
	l, _ := ParseIPList(nil)
	if l.Enabled() || !l.Allows("8.8.8.8") || !l.Allows("bukan-ip") {
		t.Fatal("daftar kosong berarti tanpa pembatasan")
	}
	var nilList *IPList
	if nilList.Enabled() || !nilList.Allows("1.1.1.1") {
		t.Fatal("nil aman dipakai")
	}
}

func TestIPListInvalid(t *testing.T) {
	for _, bad := range []string{"999.1.1.1", "1.2.3.4/99", "abc", "1.2.3"} {
		if _, err := ParseIPList([]string{bad}); err == nil {
			t.Errorf("%q harus ditolak", bad)
		}
	}
}

func TestLoadRejectsInvalidAllowlist(t *testing.T) {
	t.Setenv("AYUMI_ALLOWED_IPS", "bukan-ip")
	if _, err := Load(); err == nil {
		t.Fatal("allowlist tidak valid harus menggagalkan Load")
	}
	t.Setenv("AYUMI_ALLOWED_IPS", "10.0.0.0/8, 203.0.113.7")
	t.Setenv("AYUMI_TRUSTED_PROXIES", "172.16.0.0/12")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.AllowedIPs.Allows("10.1.2.3") || c.AllowedIPs.Allows("11.0.0.1") || len(c.TrustedProxies) != 1 {
		t.Fatalf("konfigurasi: %+v", c)
	}
}

func TestLoadDotEnvFallsBackToParentDir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "backend")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("AYUMI_TEST_PARENT=ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("AYUMI_ENV_FILE")
	os.Unsetenv("AYUMI_TEST_PARENT")
	t.Cleanup(func() { os.Unsetenv("AYUMI_TEST_PARENT") })
	t.Chdir(sub)

	got, err := LoadDotEnv()
	if err != nil || got != filepath.Join("..", ".env") {
		t.Fatalf("LoadDotEnv = %q, %v", got, err)
	}
	if os.Getenv("AYUMI_TEST_PARENT") != "ok" {
		t.Fatal(".env di folder induk harus terbaca")
	}

	// .env di folder kerja didahulukan
	if err := os.WriteFile(filepath.Join(sub, ".env"), []byte("AYUMI_TEST_PARENT=lokal\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("AYUMI_TEST_PARENT")
	if got, _ = LoadDotEnv(); got != ".env" || os.Getenv("AYUMI_TEST_PARENT") != "lokal" {
		t.Fatalf("file lokal harus didahulukan: %q %q", got, os.Getenv("AYUMI_TEST_PARENT"))
	}
}

func TestHostList(t *testing.T) {
	l, err := ParseHostList([]string{"link.neonn.my.id", " LOCALHOST ", "127.0.0.1", "[::1]", "*.contoh.id", ""})
	if err != nil {
		t.Fatal(err)
	}
	allow := []string{"link.neonn.my.id", "LINK.NEONN.MY.ID", "link.neonn.my.id.", "localhost", "127.0.0.1", "::1", "[::1]", "a.contoh.id", "x.y.contoh.id"}
	deny := []string{"neonn.my.id", "evil.neonn.my.id", "link.neonn.my.id.evil.com", "contoh.id", "evilcontoh.id", "", "0.0.0.0", "attacker.com"}
	for _, h := range allow {
		if !l.Allows(h) {
			t.Errorf("%q seharusnya diizinkan", h)
		}
	}
	for _, h := range deny {
		if l.Allows(h) {
			t.Errorf("%q seharusnya ditolak", h)
		}
	}
}

func TestHostListEmptyAllowsAll(t *testing.T) {
	l, _ := ParseHostList(nil)
	var nilList *HostList
	if l.Enabled() || !l.Allows("apa.saja") || nilList.Enabled() || !nilList.Allows("x") {
		t.Fatal("daftar kosong/nil berarti tanpa pembatasan")
	}
}

func TestHostListRejectsBadEntries(t *testing.T) {
	for _, bad := range []string{"https://link.neonn.my.id", "link.neonn.my.id:8081", "link.neonn.my.id/path", "user@host", "host name", "-awal.id", "*", "**.id", "a b.com"} {
		if _, err := ParseHostList([]string{bad}); err == nil {
			t.Errorf("%q harus ditolak", bad)
		}
	}
}

func TestLoadAllowedHostsFromEnv(t *testing.T) {
	t.Setenv("AYUMI_ALLOWED_HOSTS", "https://salah.id")
	if _, err := Load(); err == nil {
		t.Fatal("host tidak valid harus menggagalkan Load")
	}
	t.Setenv("AYUMI_ALLOWED_HOSTS", "link.neonn.my.id, localhost")
	c, err := Load()
	if err != nil || !c.AllowedHosts.Allows("link.neonn.my.id") || c.AllowedHosts.Allows("lain.com") {
		t.Fatalf("konfigurasi: %v %+v", err, c)
	}
}
