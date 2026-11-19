// Package fsbrowse menelusuri folder secara aman untuk pemilih "Session baru".
//
// Aturan keamanan:
//   - hanya folder (tidak pernah file); nama berawalan titik, Library, dan node_modules dilewati
//   - hanya di dalam root yang diizinkan
//   - path di-resolve (symlink terurai) lalu dicek masih di dalam root,
//     sehingga ".." dan symlink tidak bisa keluar
//   - jumlah entri dibatasi
package fsbrowse

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const MaxEntries = 500

var hiddenNames = map[string]bool{"Library": true, "node_modules": true}

type Browser struct{ roots []string }

// New me-resolve root (symlink terurai); root yang tidak ada diabaikan.
func New(roots []string) *Browser {
	b := &Browser{}
	for _, r := range roots {
		if real, err := filepath.EvalSymlinks(r); err == nil && filepath.IsAbs(real) {
			b.roots = append(b.roots, filepath.Clean(real))
		}
	}
	return b
}

func (b *Browser) Roots() []string { return b.roots }

func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// ResolveDir mengembalikan path nyata bila p adalah folder di dalam root.
func (b *Browser) ResolveDir(p string) (string, bool) {
	return b.resolve(p, true)
}

// ResolveFile mengembalikan path nyata bila p adalah file di dalam root.
func (b *Browser) ResolveFile(p string) (string, bool) {
	return b.resolve(p, false)
}

func (b *Browser) resolve(p string, dirOnly bool) (string, bool) {
	if p == "" || !filepath.IsAbs(p) {
		return "", false
	}
	real, err := filepath.EvalSymlinks(filepath.Clean(p))
	if err != nil {
		return "", false
	}
	st, err := os.Stat(real)
	if err != nil {
		return "", false
	}
	if dirOnly {
		if !st.IsDir() {
			return "", false
		}
	} else if st.IsDir() {
		return "", false
	}
	for _, root := range b.roots {
		if within(root, real) {
			return real, true
		}
	}
	return "", false
}

type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Listing struct {
	Path      string  `json:"path"`
	Parent    string  `json:"parent"`
	Dirs      []Entry `json:"dirs"`
	Truncated bool    `json:"truncated"`
}

// List mendaftar subfolder dari p (kosong = root pertama).
func (b *Browser) List(p string) (*Listing, bool) {
	if len(b.roots) == 0 {
		return nil, false
	}
	if p == "" {
		p = b.roots[0]
	}
	real, ok := b.ResolveDir(p)
	if !ok {
		return nil, false
	}

	parent := ""
	if up := filepath.Dir(real); up != real {
		for _, root := range b.roots {
			if within(root, up) {
				parent = up
				break
			}
		}
	}

	entries, err := os.ReadDir(real)
	if err != nil {
		return nil, false
	}
	out := &Listing{Path: real, Parent: parent, Dirs: []Entry{}}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || hiddenNames[name] {
			continue
		}
		full := filepath.Join(real, name)
		// symlink hanya diikuti bila tujuannya folder yang masih diizinkan
		if _, ok := b.ResolveDir(full); !ok {
			continue
		}
		if len(out.Dirs) >= MaxEntries {
			out.Truncated = true
			break
		}
		out.Dirs = append(out.Dirs, Entry{Name: name, Path: full})
	}
	sort.Slice(out.Dirs, func(i, j int) bool {
		return strings.ToLower(out.Dirs[i].Name) < strings.ToLower(out.Dirs[j].Name)
	})
	return out, true
}
