package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type desktopGroupMeta struct {
	// cliSessionId → group name (mis. "JABAR", "VelocInfra")
	byCLI map[string]string
	// cliSessionId → cwd from Desktop index (preferred for folder fallback)
	cwdByCLI map[string]string
	// cliSessionId → judul di sidebar Claude Desktop
	titleByCLI map[string]string
	// group name → order index
	order map[string]int
}

func loadDesktopGroups(root string) desktopGroupMeta {
	meta := desktopGroupMeta{
		byCLI:      map[string]string{},
		cwdByCLI:   map[string]string{},
		titleByCLI: map[string]string{},
		order:      map[string]int{},
	}
	if root == "" {
		return meta
	}

	// local_<desktopId>.json → cliSessionId (+ cwd/title)
	localToCLI := map[string]string{}
	sessionsRoot := filepath.Join(root, "claude-code-sessions")
	_ = filepath.WalkDir(sessionsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "local_") || !strings.HasSuffix(name, ".json") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var raw struct {
			SessionID    string `json:"sessionId"`
			CLISessionID string `json:"cliSessionId"`
			Cwd          string `json:"cwd"`
			OriginCwd    string `json:"originCwd"`
			Title        string `json:"title"`
			IsArchived   bool   `json:"isArchived"`
		}
		if json.Unmarshal(b, &raw) != nil || raw.CLISessionID == "" || raw.IsArchived {
			return nil
		}
		key := raw.SessionID
		if key == "" {
			key = strings.TrimSuffix(name, ".json")
		}
		localToCLI[key] = raw.CLISessionID
		cwd := raw.OriginCwd
		if cwd == "" {
			cwd = raw.Cwd
		}
		if cwd != "" {
			meta.cwdByCLI[raw.CLISessionID] = cwd
		}
		if t := strings.TrimSpace(raw.Title); t != "" {
			meta.titleByCLI[raw.CLISessionID] = t
		}
		return nil
	})

	cfgPath := filepath.Join(root, "claude_desktop_config.json")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return meta
	}
	var cfg struct {
		Preferences struct {
			EpitaxyPrefs map[string]json.RawMessage `json:"epitaxyPrefs"`
		} `json:"preferences"`
	}
	if json.Unmarshal(b, &cfg) != nil {
		return meta
	}

	// groups + assignments
	rawScopes, ok := cfg.Preferences.EpitaxyPrefs["dframe-group-scopes"]
	if ok {
		var scopes map[string]struct {
			Groups []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"groups"`
			Assignments map[string]string `json:"assignments"`
		}
		if json.Unmarshal(rawScopes, &scopes) == nil {
			idToName := map[string]string{}
			for _, scope := range scopes {
				for _, g := range scope.Groups {
					idToName[g.ID] = g.Name
				}
				for codeKey, groupID := range scope.Assignments {
					name := idToName[groupID]
					if name == "" {
						continue
					}
					localID := strings.TrimPrefix(codeKey, "code:")
					if cli, ok := localToCLI[localID]; ok {
						meta.byCLI[cli] = name
					}
				}
			}
		}
	}

	// section order (urutan group seperti di sidebar Claude Desktop)
	rawSections, ok := cfg.Preferences.EpitaxyPrefs["dframe-code-sections"]
	if ok {
		var sections map[string]struct {
			Sections []struct {
				ID    string `json:"id"`
				Kind  string `json:"kind"`
				Name  string `json:"name"`
				Order int    `json:"order"`
				Shown bool   `json:"shown"`
			} `json:"sections"`
		}
		if json.Unmarshal(rawSections, &sections) == nil {
			for _, scope := range sections {
				for _, s := range scope.Sections {
					if !s.Shown {
						continue
					}
					if s.Kind == "manual" && s.Name != "" {
						meta.order[s.Name] = s.Order
					}
				}
			}
		}
	}

	return meta
}

func folderGroupLabel(cwd string) string {
	base := filepath.Base(strings.TrimRight(cwd, string(os.PathSeparator)))
	if base == "" || base == "." || base == "/" {
		return "Lainnya"
	}
	if strings.Contains(cwd, "scratch-workspaces") || strings.HasPrefix(base, "scratch-") {
		return "Scratch"
	}
	return base
}

// applyDesktopGroups menyelaraskan keanggotaan group dengan sidebar Claude Desktop.
// Session yang tidak ada di indeks Desktop disembunyikan, kecuali yang dibuat lewat Ayumi Link (registry).
func applyDesktopGroups(sessions []Session, desktopDir string, registry map[string]string) []Session {
	meta := loadDesktopGroups(desktopDir)
	// Prefer Claude Desktop sidebar membership when index exists
	filterToDesktop := len(meta.cwdByCLI) > 0

	out := make([]Session, 0, len(sessions))
	for _, s := range sessions {
		if _, mine := registry[s.ID]; filterToDesktop && !mine {
			if _, ok := meta.cwdByCLI[s.ID]; !ok {
				if _, assigned := meta.byCLI[s.ID]; !assigned {
					continue // not shown in Claude Desktop Code tab
				}
			}
		}
		if desktopCwd, ok := meta.cwdByCLI[s.ID]; ok && desktopCwd != "" {
			s.Cwd = desktopCwd
			s.Project = filepath.Base(desktopCwd)
		}
		if title, ok := meta.titleByCLI[s.ID]; ok && title != "" {
			s.Title = title
		}
		if name, ok := meta.byCLI[s.ID]; ok && name != "" {
			s.Group = name
			s.GroupOrder = meta.order[name]
		} else {
			s.Group = folderGroupLabel(s.Cwd)
			if ord, ok := meta.order[s.Group]; ok {
				s.GroupOrder = ord
			} else {
				s.GroupOrder = 1000
			}
		}
		out = append(out, s)
	}
	return out
}
