package api

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/gitrepos"
)

type gitFile struct {
	gitrepos.File
	FromSession bool `json:"fromSession"` // diedit oleh chat session ini
}

type gitRepo struct {
	Path   string    `json:"path"`
	Name   string    `json:"name"`
	Branch string    `json:"branch,omitempty"`
	Setup  bool      `json:"setup"` // ditambahkan lewat setup (bisa dihapus)
	Files  []gitFile `json:"files"`
	Error  string    `json:"error,omitempty"`
}

func (s *Server) sessionCwd(id string) string {
	for _, x := range s.Svc.List() {
		if x.ID == id {
			return x.Cwd
		}
	}
	return s.Store.Registry()[id]
}

// collectRepos menggabungkan repo yang di-setup, repo folder kerja, dan repo dari file yang diedit session.
// Hanya folder di dalam root yang diizinkan.
func (s *Server) collectRepos(id string) (cwd string, repos []gitRepo, err error) {
	cwd, ok := s.Browser.ResolveDir(s.sessionCwd(id))
	if !ok {
		return "", nil, fiber.NewError(fiber.StatusForbidden, "folder session tidak diizinkan atau tidak ada")
	}
	found := map[string]bool{} // path repo -> setup?
	add := func(dir string, setup bool) {
		real, ok := s.Browser.ResolveDir(dir)
		if !ok {
			return
		}
		root := gitrepos.Root(real)
		if root == "" {
			return
		}
		if _, ok := s.Browser.ResolveDir(root); !ok {
			return
		}
		found[root] = found[root] || setup
	}
	for _, p := range gitrepos.Load(cwd) {
		add(p, true)
	}
	add(cwd, false)
	edited := map[string]bool{}
	if changes, err := s.Svc.ListChanges(id); err == nil {
		for _, ch := range changes {
			abs := ch.AbsPath
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(cwd, abs)
			}
			dir := filepath.Dir(abs)
			if real, err := filepath.EvalSymlinks(dir); err == nil {
				dir = real
			}
			edited[filepath.Join(dir, filepath.Base(abs))] = true
			add(dir, false)
		}
	}
	for path, setup := range found {
		r := gitRepo{Path: path, Name: filepath.Base(path), Branch: gitrepos.Branch(path), Setup: setup, Files: []gitFile{}}
		files, err := gitrepos.Status(path)
		if err != nil {
			r.Error = "git status gagal"
		}
		for _, f := range files {
			r.Files = append(r.Files, gitFile{File: f, FromSession: edited[filepath.Join(path, filepath.FromSlash(f.Path))]})
		}
		repos = append(repos, r)
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].Path < repos[j].Path })
	return cwd, repos, nil
}

func (s *Server) listSessionRepos(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	cwd, repos, err := s.collectRepos(id)
	if err != nil {
		return err
	}
	if repos == nil {
		repos = []gitRepo{}
	}
	return c.JSON(fiber.Map{"cwd": cwd, "repos": repos})
}

func (s *Server) getSessionRepoDiff(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	_, repos, err := s.collectRepos(id)
	if err != nil {
		return err
	}
	// Hanya file yang muncul di daftar status yang boleh dibaca.
	repo, path := c.Query("repo"), c.Query("path")
	for _, r := range repos {
		if r.Path != repo {
			continue
		}
		for _, f := range r.Files {
			if f.Path == path {
				diff, err := gitrepos.Diff(r.Path, f.File)
				if err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "git diff gagal")
				}
				return c.JSON(fiber.Map{"diff": diff, "path": path})
			}
		}
	}
	return fiber.NewError(fiber.StatusNotFound, "file tidak ditemukan")
}

func (s *Server) addSessionRepo(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	var b struct {
		Path string `json:"path"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	cwd, ok := s.Browser.ResolveDir(s.sessionCwd(id))
	if !ok {
		return fiber.NewError(fiber.StatusForbidden, "folder session tidak diizinkan atau tidak ada")
	}
	dir, ok := s.Browser.ResolveDir(strings.TrimSpace(b.Path))
	if !ok {
		return fiber.NewError(fiber.StatusForbidden, "folder tidak diizinkan")
	}
	root := gitrepos.Root(dir)
	if root == "" {
		return fiber.NewError(fiber.StatusBadRequest, "folder ini bukan repo git")
	}
	repos := gitrepos.Load(cwd)
	for _, p := range repos {
		if p == root {
			return c.JSON(fiber.Map{"ok": true})
		}
	}
	if err := gitrepos.Save(cwd, append(repos, root)); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) removeSessionRepo(c *fiber.Ctx) error {
	id, err := sessionID(c)
	if err != nil {
		return err
	}
	cwd, ok := s.Browser.ResolveDir(s.sessionCwd(id))
	if !ok {
		return fiber.NewError(fiber.StatusForbidden, "folder session tidak diizinkan atau tidak ada")
	}
	target := c.Query("path")
	keep := []string{}
	for _, p := range gitrepos.Load(cwd) {
		real, ok := s.Browser.ResolveDir(p)
		if p == target || (ok && gitrepos.Root(real) == target) {
			continue
		}
		keep = append(keep, p)
	}
	if err := gitrepos.Save(cwd, keep); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}
