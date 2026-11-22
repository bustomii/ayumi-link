package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	ErrNotFound     = errors.New("tidak ditemukan")
	ErrConflict     = errors.New("slug sudah dipakai")
	ErrBadInput     = errors.New("input tidak valid")
	slugPattern     = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	maxNameLen      = 80
	maxLabelLen     = 40
	maxDescLen      = 2000
	maxBodyLen      = 100_000
	maxAgents       = 100
	maxSkills       = 200
	maxSkillsPerAgt = 50
)

// Agent: "tim" di perusahaan (devops, frontend, …) yang memakai banyak skill.
type Agent struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Label       string   `json:"label,omitempty"`
	Description string   `json:"description,omitempty"`
	SkillIDs    []string `json:"skillIds"`
	// Source "disk" = diimpor dari file (~/.claude atau .claude/ proyek) yang dibuat/diubah/dihapus
	// lewat chat Claude; SourcePath = file asalnya.
	Source     string    `json:"source,omitempty"`
	SourcePath string    `json:"sourcePath,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Skill: instruksi terperinci; bisa ditempel ke banyak agent.
type Skill struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Label       string    `json:"label,omitempty"`
	Description string    `json:"description,omitempty"`
	Body        string    `json:"body"`
	Source      string    `json:"source,omitempty"`
	SourcePath  string    `json:"sourcePath,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type agentsSkillsFile struct {
	Agents []Agent `json:"agents"`
	Skills []Skill `json:"skills"`
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Slugify membuat slug dari nama (huruf kecil, strip non-alnum).
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if b.Len() > 0 && !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "item"
	}
	if len(out) > 64 {
		out = out[:64]
		out = strings.Trim(out, "-")
	}
	return out
}

func validSlug(s string) bool {
	return slugPattern.MatchString(s) && len(s) <= 64
}

func (s *Store) loadAgentsSkills() agentsSkillsFile {
	out := agentsSkillsFile{Agents: []Agent{}, Skills: []Skill{}}
	if b, err := os.ReadFile(s.path("agents-skills.json")); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if out.Agents == nil {
		out.Agents = []Agent{}
	}
	if out.Skills == nil {
		out.Skills = []Skill{}
	}
	for i := range out.Agents {
		if out.Agents[i].SkillIDs == nil {
			out.Agents[i].SkillIDs = []string{}
		}
	}
	return out
}

func (s *Store) saveAgentsSkills(v agentsSkillsFile) error {
	if v.Agents == nil {
		v.Agents = []Agent{}
	}
	if v.Skills == nil {
		v.Skills = []Skill{}
	}
	return s.writeAtomic("agents-skills.json", v)
}

func (s *Store) ListAgents() []Agent {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.loadAgentsSkills().Agents
	// Salin ke slice non-nil agar JSON [] bukan null.
	out := make([]Agent, 0, len(src))
	return append(out, src...)
}

func (s *Store) ListSkills() []Skill {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.loadAgentsSkills().Skills
	out := make([]Skill, 0, len(src))
	return append(out, src...)
}

func (s *Store) GetAgent(idOrSlug string) (Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	for _, a := range f.Agents {
		if a.ID == idOrSlug || a.Slug == idOrSlug {
			return a, nil
		}
	}
	return Agent{}, ErrNotFound
}

func (s *Store) GetSkill(idOrSlug string) (Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	for _, sk := range f.Skills {
		if sk.ID == idOrSlug || sk.Slug == idOrSlug {
			return sk, nil
		}
	}
	return Skill{}, ErrNotFound
}

type AgentInput struct {
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	SkillIDs    []string `json:"skillIds"`
}

type SkillInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

func normalizeAgentInput(in AgentInput, existing *Agent) (AgentInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Label = strings.TrimSpace(in.Label)
	in.Description = strings.TrimSpace(in.Description)
	in.Slug = strings.TrimSpace(in.Slug)
	if in.Name == "" {
		return in, ErrBadInput
	}
	if len(in.Name) > maxNameLen || len(in.Label) > maxLabelLen || len(in.Description) > maxDescLen {
		return in, ErrBadInput
	}
	if in.Slug == "" {
		if existing != nil {
			in.Slug = existing.Slug
		} else {
			in.Slug = Slugify(in.Name)
		}
	} else {
		in.Slug = Slugify(in.Slug)
	}
	if !validSlug(in.Slug) {
		return in, ErrBadInput
	}
	if in.SkillIDs == nil {
		in.SkillIDs = []string{}
	}
	if len(in.SkillIDs) > maxSkillsPerAgt {
		return in, ErrBadInput
	}
	return in, nil
}

func normalizeSkillInput(in SkillInput, existing *Skill) (SkillInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Label = strings.TrimSpace(in.Label)
	in.Description = strings.TrimSpace(in.Description)
	in.Body = strings.TrimSpace(in.Body)
	in.Slug = strings.TrimSpace(in.Slug)
	if in.Name == "" {
		return in, ErrBadInput
	}
	if len(in.Name) > maxNameLen || len(in.Label) > maxLabelLen || len(in.Description) > maxDescLen || len(in.Body) > maxBodyLen {
		return in, ErrBadInput
	}
	if in.Slug == "" {
		if existing != nil {
			in.Slug = existing.Slug
		} else {
			in.Slug = Slugify(in.Name)
		}
	} else {
		in.Slug = Slugify(in.Slug)
	}
	if !validSlug(in.Slug) {
		return in, ErrBadInput
	}
	return in, nil
}

func (s *Store) CreateAgent(in AgentInput) (Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	if len(f.Agents) >= maxAgents {
		return Agent{}, ErrBadInput
	}
	in, err := normalizeAgentInput(in, nil)
	if err != nil {
		return Agent{}, err
	}
	for _, a := range f.Agents {
		if a.Slug == in.Slug {
			return Agent{}, ErrConflict
		}
	}
	in.SkillIDs = filterKnownSkills(f.Skills, in.SkillIDs)
	now := time.Now()
	a := Agent{
		ID: newID(), Slug: in.Slug, Name: in.Name, Label: in.Label,
		Description: in.Description, SkillIDs: in.SkillIDs, CreatedAt: now, UpdatedAt: now,
	}
	f.Agents = append(f.Agents, a)
	if err := s.saveAgentsSkills(f); err != nil {
		return Agent{}, err
	}
	return a, nil
}

func (s *Store) UpdateAgent(id string, in AgentInput) (Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	idx := -1
	for i, a := range f.Agents {
		if a.ID == id || a.Slug == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Agent{}, ErrNotFound
	}
	cur := f.Agents[idx]
	in, err := normalizeAgentInput(in, &cur)
	if err != nil {
		return Agent{}, err
	}
	for i, a := range f.Agents {
		if i != idx && a.Slug == in.Slug {
			return Agent{}, ErrConflict
		}
	}
	in.SkillIDs = filterKnownSkills(f.Skills, in.SkillIDs)
	cur.Slug, cur.Name, cur.Label = in.Slug, in.Name, in.Label
	cur.Description, cur.SkillIDs = in.Description, in.SkillIDs
	cur.UpdatedAt = time.Now()
	f.Agents[idx] = cur
	if err := s.saveAgentsSkills(f); err != nil {
		return Agent{}, err
	}
	return cur, nil
}

func (s *Store) DeleteAgent(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	out := f.Agents[:0]
	found := false
	for _, a := range f.Agents {
		if a.ID == id || a.Slug == id {
			found = true
			continue
		}
		out = append(out, a)
	}
	if !found {
		return ErrNotFound
	}
	f.Agents = out
	return s.saveAgentsSkills(f)
}

func (s *Store) CreateSkill(in SkillInput) (Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	if len(f.Skills) >= maxSkills {
		return Skill{}, ErrBadInput
	}
	in, err := normalizeSkillInput(in, nil)
	if err != nil {
		return Skill{}, err
	}
	for _, sk := range f.Skills {
		if sk.Slug == in.Slug {
			return Skill{}, ErrConflict
		}
	}
	now := time.Now()
	sk := Skill{
		ID: newID(), Slug: in.Slug, Name: in.Name, Label: in.Label,
		Description: in.Description, Body: in.Body, CreatedAt: now, UpdatedAt: now,
	}
	f.Skills = append(f.Skills, sk)
	if err := s.saveAgentsSkills(f); err != nil {
		return Skill{}, err
	}
	return sk, nil
}

func (s *Store) UpdateSkill(id string, in SkillInput) (Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	idx := -1
	for i, sk := range f.Skills {
		if sk.ID == id || sk.Slug == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Skill{}, ErrNotFound
	}
	cur := f.Skills[idx]
	in, err := normalizeSkillInput(in, &cur)
	if err != nil {
		return Skill{}, err
	}
	for i, sk := range f.Skills {
		if i != idx && sk.Slug == in.Slug {
			return Skill{}, ErrConflict
		}
	}
	cur.Slug, cur.Name, cur.Label = in.Slug, in.Name, in.Label
	cur.Description, cur.Body = in.Description, in.Body
	cur.UpdatedAt = time.Now()
	f.Skills[idx] = cur
	if err := s.saveAgentsSkills(f); err != nil {
		return Skill{}, err
	}
	return cur, nil
}

func (s *Store) DeleteSkill(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	var removed string
	out := f.Skills[:0]
	for _, sk := range f.Skills {
		if sk.ID == id || sk.Slug == id {
			removed = sk.ID
			continue
		}
		out = append(out, sk)
	}
	if removed == "" {
		return ErrNotFound
	}
	f.Skills = out
	for i := range f.Agents {
		f.Agents[i].SkillIDs = removeID(f.Agents[i].SkillIDs, removed)
	}
	return s.saveAgentsSkills(f)
}

// ResolveMentions mengembalikan agent + skill bodies untuk inject ke prompt chat.
func (s *Store) ResolveMentions(agentRefs, skillRefs []string) (agents []Agent, skills []Skill) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.loadAgentsSkills()
	seenA, seenS := map[string]bool{}, map[string]bool{}
	for _, ref := range agentRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		for _, a := range f.Agents {
			if a.ID == ref || a.Slug == ref || strings.EqualFold(a.Name, ref) {
				if !seenA[a.ID] {
					agents = append(agents, a)
					seenA[a.ID] = true
				}
				for _, sid := range a.SkillIDs {
					for _, sk := range f.Skills {
						if sk.ID == sid && !seenS[sk.ID] {
							skills = append(skills, sk)
							seenS[sk.ID] = true
						}
					}
				}
				break
			}
		}
	}
	for _, ref := range skillRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		for _, sk := range f.Skills {
			if sk.ID == ref || sk.Slug == ref || strings.EqualFold(sk.Name, ref) {
				if !seenS[sk.ID] {
					skills = append(skills, sk)
					seenS[sk.ID] = true
				}
				break
			}
		}
	}
	return agents, skills
}

func filterKnownSkills(skills []Skill, ids []string) []string {
	known := map[string]bool{}
	for _, sk := range skills {
		known[sk.ID] = true
	}
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || !known[id] || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func removeID(ids []string, id string) []string {
	out := ids[:0]
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}
