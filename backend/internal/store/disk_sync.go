package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SyncFromDisk menyelaraskan daftar Agents & Skills dengan file yang dibuat, diubah, atau dihapus
// di luar menu (mis. lewat chat Claude): <root>/skills/<slug>/SKILL.md dan <root>/agents/<slug>.md.
// roots berisi direktori ".claude" (~/.claude dan .claude/ di folder proyek).
//
// Hanya item bersumber disk (Source == "disk") yang diubah/dihapus; item buatan manual tidak disentuh,
// dan slug yang sudah dipakai item manual dilewati. Mengembalikan true bila ada perubahan.
func (s *Store) SyncFromDisk(roots []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	changed := false

	skillFiles, agentFiles := scanDiskRoots(roots)

	// ---- skills ----
	bySkillPath := map[string]int{}
	skillSlugs := map[string]bool{}
	for i, sk := range f.Skills {
		skillSlugs[sk.Slug] = true
		if sk.Source == "disk" {
			bySkillPath[sk.SourcePath] = i
		}
	}
	for _, df := range skillFiles {
		fm, body, mod, ok := readDiskDoc(df.path)
		if !ok {
			continue
		}
		slug := Slugify(df.slug)
		name := clip(firstNonEmpty(fm["name"], slug), maxNameLen)
		desc := clip(fm["description"], maxDescLen)
		body = clip(strings.TrimSpace(body), maxBodyLen)
		if i, exists := bySkillPath[df.path]; exists {
			cur := f.Skills[i]
			if cur.Name != name || cur.Description != desc || cur.Body != body {
				cur.Name, cur.Description, cur.Body, cur.UpdatedAt = name, desc, body, mod
				f.Skills[i] = cur
				changed = true
			}
			continue
		}
		if skillSlugs[slug] || len(f.Skills) >= maxSkills || !validSlug(slug) {
			continue
		}
		now := time.Now()
		f.Skills = append(f.Skills, Skill{
			ID: newID(), Slug: slug, Name: name, Description: desc, Body: body,
			Source: "disk", SourcePath: df.path, CreatedAt: now, UpdatedAt: now,
		})
		skillSlugs[slug] = true
		bySkillPath[df.path] = len(f.Skills) - 1
		changed = true
	}
	// Skill dari disk yang filenya sudah dihapus.
	removed := map[string]bool{}
	keptSkills := f.Skills[:0]
	for _, sk := range f.Skills {
		if sk.Source == "disk" && fileGone(sk.SourcePath) {
			removed[sk.ID] = true
			changed = true
			continue
		}
		keptSkills = append(keptSkills, sk)
	}
	f.Skills = keptSkills

	// ---- agents ----
	skillIDBySlug := map[string]string{}
	for _, sk := range f.Skills {
		skillIDBySlug[sk.Slug] = sk.ID
	}
	byAgentPath := map[string]int{}
	agentSlugs := map[string]bool{}
	for i, a := range f.Agents {
		agentSlugs[a.Slug] = true
		if a.Source == "disk" {
			byAgentPath[a.SourcePath] = i
		}
	}
	for _, df := range agentFiles {
		fm, body, mod, ok := readDiskDoc(df.path)
		if !ok {
			continue
		}
		slug := Slugify(df.slug)
		name := clip(firstNonEmpty(fm["name"], slug), maxNameLen)
		desc := fm["description"]
		if strings.TrimSpace(desc) == "" {
			desc = firstParagraph(body)
		}
		desc = clip(desc, maxDescLen)
		instr := clip(body, maxBodyLen)
		var ids []string
		for _, ref := range splitList(fm["skills"]) {
			if id, ok := skillIDBySlug[Slugify(ref)]; ok {
				ids = append(ids, id)
			}
		}
		ids = filterKnownSkills(f.Skills, ids)
		if i, exists := byAgentPath[df.path]; exists {
			cur := f.Agents[i]
			if cur.Name != name || cur.Description != desc || cur.Instructions != instr || !sameStrings(cur.SkillIDs, ids) {
				cur.Name, cur.Description, cur.Instructions, cur.SkillIDs, cur.UpdatedAt = name, desc, instr, ids, mod
				f.Agents[i] = cur
				changed = true
			}
			continue
		}
		if agentSlugs[slug] || len(f.Agents) >= maxAgents || !validSlug(slug) {
			continue
		}
		now := time.Now()
		f.Agents = append(f.Agents, Agent{
			ID: newID(), Slug: slug, Name: name, Description: desc, Instructions: instr, SkillIDs: nonNil(ids),
			Source: "disk", SourcePath: df.path, CreatedAt: now, UpdatedAt: now,
		})
		agentSlugs[slug] = true
		byAgentPath[df.path] = len(f.Agents) - 1
		changed = true
	}
	keptAgents := f.Agents[:0]
	for _, a := range f.Agents {
		if a.Source == "disk" && fileGone(a.SourcePath) {
			changed = true
			continue
		}
		keptAgents = append(keptAgents, a)
	}
	f.Agents = keptAgents
	if len(removed) > 0 {
		for i := range f.Agents {
			for id := range removed {
				f.Agents[i].SkillIDs = removeID(f.Agents[i].SkillIDs, id)
			}
		}
	}

	if !changed {
		return false
	}
	return s.saveAgentsSkills(f) == nil
}

type diskFile struct{ path, slug string }

func scanDiskRoots(roots []string) (skills, agents []diskFile) {
	seen := map[string]bool{}
	for _, root := range roots {
		root = filepath.Clean(root)
		if root == "" || seen[root] {
			continue
		}
		seen[root] = true
		if ents, err := os.ReadDir(filepath.Join(root, "skills")); err == nil {
			for _, e := range ents {
				if !e.IsDir() && e.Type()&os.ModeSymlink == 0 {
					continue
				}
				p := filepath.Join(root, "skills", e.Name(), "SKILL.md")
				if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
					skills = append(skills, diskFile{path: p, slug: e.Name()})
				}
			}
		}
		if ents, err := os.ReadDir(filepath.Join(root, "agents")); err == nil {
			for _, e := range ents {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				p := filepath.Join(root, "agents", e.Name())
				agents = append(agents, diskFile{path: p, slug: strings.TrimSuffix(e.Name(), ".md")})
			}
		}
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].path < skills[j].path })
	sort.Slice(agents, func(i, j int) bool { return agents[i].path < agents[j].path })
	return
}

func fileGone(p string) bool {
	_, err := os.Stat(p)
	return os.IsNotExist(err)
}

// readDiskDoc membaca file markdown ber-frontmatter (--- … ---). Batas ukuran 1 MB.
func readDiskDoc(path string) (fm map[string]string, body string, mod time.Time, ok bool) {
	st, err := os.Stat(path)
	if err != nil || st.Size() > 1<<20 {
		return nil, "", time.Time{}, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", time.Time{}, false
	}
	fm, body = parseFrontmatter(string(b))
	return fm, body, st.ModTime(), true
}

func parseFrontmatter(src string) (map[string]string, string) {
	fm := map[string]string{}
	src = strings.ReplaceAll(src, "\r\n", "\n")
	if !strings.HasPrefix(src, "---\n") {
		return fm, src
	}
	rest := src[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return fm, src
	}
	head := rest[:end]
	body := strings.TrimPrefix(rest[end+4:], "\n")
	lines := strings.Split(head, "\n")
	for i := 0; i < len(lines); i++ {
		ln := lines[i]
		k, v, found := strings.Cut(ln, ":")
		if !found || strings.HasPrefix(ln, " ") || strings.HasPrefix(ln, "\t") || strings.HasPrefix(ln, "-") {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch {
		case v == "" || v == ">" || v == "|" || v == ">-" || v == "|-":
			var parts []string
			for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t") || strings.HasPrefix(strings.TrimSpace(lines[i+1]), "- ")) {
				i++
				parts = append(parts, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), "- ")))
			}
			if v == "" {
				v = strings.Join(parts, ", ")
			} else {
				v = strings.Join(parts, " ")
			}
		default:
			v = strings.Trim(v, `"'`)
		}
		fm[k] = v
	}
	return fm, body
}

func splitList(v string) []string {
	v = strings.Trim(strings.TrimSpace(v), "[]")
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.Trim(strings.TrimSpace(p), `"'`); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstParagraph(body string) string {
	for _, p := range strings.Split(strings.TrimSpace(body), "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			return p
		}
	}
	return ""
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return b
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		s = s[:n]
		for len(s) > 0 && s[len(s)-1]&0xC0 == 0x80 { // jangan potong di tengah rune
			s = s[:len(s)-1]
		}
	}
	return s
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
