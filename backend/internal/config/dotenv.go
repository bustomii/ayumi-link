package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv membaca file .env (default: ./.env lalu ../.env, atau path di AYUMI_ENV_FILE) dan mengisi
// environment untuk variabel yang BELUM di-set. Variabel environment sungguhan selalu menang,
// jadi `docker run -e ...` atau `export ...` tetap mengalahkan isi .env.
// Mengembalikan path file yang dibaca ("" bila tidak ada file).
func LoadDotEnv() (string, error) {
	path := os.Getenv("AYUMI_ENV_FILE")
	explicit := path != ""
	candidates := []string{path}
	if !explicit {
		// `cd backend && go run .` memakai .env di root repo (yang juga dipakai docker compose).
		candidates = []string{".env", filepath.Join("..", ".env")}
	}
	var f *os.File
	var err error
	for _, c := range candidates {
		if f, err = os.Open(c); err == nil {
			path = c
			break
		}
		if !os.IsNotExist(err) || explicit {
			return "", err
		}
	}
	if f == nil {
		return "", nil
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		key, val, ok := parseEnvLine(sc.Text())
		if !ok {
			continue
		}
		if _, set := os.LookupEnv(key); !set {
			_ = os.Setenv(key, val)
		}
	}
	return path, sc.Err()
}

// parseEnvLine mengurai satu baris `KEY=VALUE`. Mendukung komentar (#), awalan `export`,
// nilai berkutip tunggal/ganda, dan komentar di akhir nilai tanpa kutip (` # ...`).
func parseEnvLine(line string) (key, val string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
	eq := strings.IndexByte(line, '=')
	if eq <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:eq])
	if key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false
	}
	val = strings.TrimSpace(line[eq+1:])
	if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
		if end := strings.IndexByte(val[1:], val[0]); end >= 0 {
			return key, val[1 : 1+end], true
		}
	}
	if i := strings.Index(val, " #"); i >= 0 {
		val = strings.TrimSpace(val[:i])
	}
	return key, val, true
}
