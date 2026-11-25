// Package gitrepos membaca perubahan belum ter-commit pada repo git yang dipantau tab Changes.
package gitrepos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxFiles     = 500
	maxDiffBytes = 400_000
	maxRepos     = 30
)

// File adalah satu file yang belum ter-commit.
type File struct {
	Path   string `json:"path"`   // relatif terhadap repo
	Status string `json:"status"` // modified | added | deleted | renamed | untracked
}

type config struct {
	Repos []string `json:"repos"`
}

// ConfigPath adalah daftar repo yang di-setup untuk sebuah folder kerja (bisa diedit agent).
func ConfigPath(cwd string) string { return filepath.Join(cwd, ".ayumi", "repos.json") }

// Load membaca daftar repo yang di-setup; file tidak ada = kosong.
func Load(cwd string) []string {
	b, err := os.ReadFile(ConfigPath(cwd))
	if err != nil {
		return nil
	}
	var c config
	if json.Unmarshal(b, &c) != nil {
		return nil
	}
	if len(c.Repos) > maxRepos {
		c.Repos = c.Repos[:maxRepos]
	}
	return c.Repos
}

// Save menulis daftar repo. Folder .ayumi diberi .gitignore sendiri agar tidak mengotori repo.
func Save(cwd string, repos []string) error {
	dir := filepath.Join(cwd, ".ayumi")
	if st, err := os.Lstat(dir); err == nil && !st.IsDir() {
		return errors.New(".ayumi bukan folder")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ignore := filepath.Join(dir, ".gitignore")
	if _, err := os.Lstat(ignore); os.IsNotExist(err) {
		if err := os.WriteFile(ignore, []byte("*\n"), 0o644); err != nil {
			return err
		}
	}
	if len(repos) > maxRepos {
		return errors.New("terlalu banyak repo")
	}
	if repos == nil {
		repos = []string{}
	}
	b, err := json.MarshalIndent(config{Repos: repos}, "", "  ")
	if err != nil {
		return err
	}
	tmp := ConfigPath(cwd) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ConfigPath(cwd))
}

// Root mencari folder teratas repo git yang memuat dir ("" bila bukan repo).
func Root(dir string) string {
	dir = filepath.Clean(dir)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func run(repo string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// fsmonitor dan diff eksternal dimatikan agar config repo tidak menjalankan program lain.
	full := append([]string{"-C", repo, "-c", "core.fsmonitor=false", "-c", "core.quotepath=false"}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
	return cmd.Output()
}

// Branch mengembalikan nama branch aktif.
func Branch(repo string) string {
	out, err := run(repo, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Status mendaftar file yang belum ter-commit.
func Status(repo string) ([]File, error) {
	out, err := run(repo, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var files []File
	parts := bytes.Split(out, []byte{0})
	for i := 0; i < len(parts) && len(files) < maxFiles; i++ {
		e := string(parts[i])
		if len(e) < 4 {
			continue
		}
		xy, p := e[:2], e[3:]
		st := "modified"
		switch {
		case xy == "??":
			st = "untracked"
		case strings.ContainsAny(xy, "R"):
			st = "renamed"
			i++ // entri berikutnya adalah nama lama
		case strings.ContainsAny(xy, "D"):
			st = "deleted"
		case strings.ContainsAny(xy, "A"):
			st = "added"
		}
		files = append(files, File{Path: p, Status: st})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// Diff mengembalikan unified diff satu file terhadap HEAD. f harus berasal dari Status.
func Diff(repo string, f File) (string, error) {
	abs := filepath.Join(repo, filepath.FromSlash(f.Path))
	if f.Status != "untracked" {
		out, err := run(repo, "diff", "--no-ext-diff", "--no-textconv", "HEAD", "--", f.Path)
		if err != nil {
			// Repo tanpa commit: bandingkan dengan index.
			out, err = run(repo, "diff", "--no-ext-diff", "--no-textconv", "--cached", "--", f.Path)
		}
		if err != nil {
			return "", err
		}
		return clip(string(out)), nil
	}
	st, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "(bukan file biasa)", nil
	}
	if st.Size() > maxDiffBytes {
		return "(file terlalu besar untuk ditampilkan)", nil
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return "(file biner)", nil
	}
	return "+++ " + f.Path + "\n+" + strings.ReplaceAll(strings.TrimSuffix(string(b), "\n"), "\n", "\n+"), nil
}

func clip(s string) string {
	if len(s) > maxDiffBytes {
		return s[:maxDiffBytes] + "\n… (dipotong)"
	}
	return s
}
