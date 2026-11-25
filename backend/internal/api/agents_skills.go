package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"ayumi-link/internal/store"
)

var (
	reAtAgent = regexp.MustCompile(`(?:^|[\s([{])@([a-zA-Z0-9][a-zA-Z0-9._-]{0,63})`)
	reHashSk  = regexp.MustCompile(`(?:^|[\s([{])#([a-zA-Z0-9][a-zA-Z0-9._-]{0,63})`)
)

func extractMentionRefs(msg string, agents, skills []string) (outA, outS []string) {
	seenA, seenS := map[string]bool{}, map[string]bool{}
	add := func(m map[string]bool, list *[]string, v string) {
		v = strings.TrimSpace(v)
		if v == "" || m[strings.ToLower(v)] {
			return
		}
		m[strings.ToLower(v)] = true
		*list = append(*list, v)
	}
	for _, v := range agents {
		add(seenA, &outA, v)
	}
	for _, v := range skills {
		add(seenS, &outS, v)
	}
	// Tanda baca di ujung kalimat ("@agent.") bukan bagian dari slug.
	for _, m := range reAtAgent.FindAllStringSubmatch(msg, -1) {
		add(seenA, &outA, strings.TrimRight(m[1], "._-"))
	}
	for _, m := range reHashSk.FindAllStringSubmatch(msg, -1) {
		add(seenS, &outS, strings.TrimRight(m[1], "._-"))
	}
	return outA, outS
}

func (s *Server) listAgents(c *fiber.Ctx) error {
	s.syncDisk()
	return c.JSON(fiber.Map{"agents": s.Store.ListAgents()})
}

func (s *Server) getAgent(c *fiber.Ctx) error {
	s.syncDisk()
	a, err := s.Store.GetAgent(c.Params("id"))
	if err != nil {
		return agentsSkillsErr(err)
	}
	skills := make([]store.Skill, 0)
	all := s.Store.ListSkills()
	byID := map[string]store.Skill{}
	for _, sk := range all {
		byID[sk.ID] = sk
	}
	for _, id := range a.SkillIDs {
		if sk, ok := byID[id]; ok {
			skills = append(skills, sk)
		}
	}
	return c.JSON(fiber.Map{"agent": a, "skills": skills})
}

func (s *Server) createAgent(c *fiber.Ctx) error {
	var in store.AgentInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	a, err := s.Store.CreateAgent(in)
	if err != nil {
		return agentsSkillsErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(a)
}

func (s *Server) updateAgent(c *fiber.Ctx) error {
	var in store.AgentInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	a, err := s.Store.UpdateAgent(c.Params("id"), in)
	if err != nil {
		return agentsSkillsErr(err)
	}
	return c.JSON(a)
}

func (s *Server) deleteAgent(c *fiber.Ctx) error {
	if err := s.Store.DeleteAgent(c.Params("id")); err != nil {
		return agentsSkillsErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) listSkills(c *fiber.Ctx) error {
	s.syncDisk()
	return c.JSON(fiber.Map{"skills": s.Store.ListSkills()})
}

func (s *Server) getSkill(c *fiber.Ctx) error {
	s.syncDisk()
	sk, err := s.Store.GetSkill(c.Params("id"))
	if err != nil {
		return agentsSkillsErr(err)
	}
	agents := make([]store.Agent, 0)
	for _, a := range s.Store.ListAgents() {
		for _, sid := range a.SkillIDs {
			if sid == sk.ID {
				agents = append(agents, a)
				break
			}
		}
	}
	return c.JSON(fiber.Map{"skill": sk, "agents": agents})
}

func (s *Server) createSkill(c *fiber.Ctx) error {
	var in store.SkillInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	sk, err := s.Store.CreateSkill(in)
	if err != nil {
		return agentsSkillsErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(sk)
}

func (s *Server) updateSkill(c *fiber.Ctx) error {
	var in store.SkillInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	sk, err := s.Store.UpdateSkill(c.Params("id"), in)
	if err != nil {
		return agentsSkillsErr(err)
	}
	return c.JSON(sk)
}

func (s *Server) deleteSkill(c *fiber.Ctx) error {
	if err := s.Store.DeleteSkill(c.Params("id")); err != nil {
		return agentsSkillsErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func agentsSkillsErr(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, "tidak ditemukan")
	case errors.Is(err, store.ErrConflict):
		return fiber.NewError(fiber.StatusConflict, "slug sudah dipakai")
	case errors.Is(err, store.ErrBadInput):
		return fiber.NewError(fiber.StatusBadRequest, "input tidak valid")
	default:
		return err
	}
}

// BuildAgentsSkillsPrompt menyusun blok konteks agent/skill untuk disisipkan ke pesan chat.
func BuildAgentsSkillsPrompt(agents []store.Agent, skills []store.Skill) string {
	if len(agents) == 0 && len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[Konteks Ayumi Link — Agents & Skills]\n")
	b.WriteString("Kerjakan permintaan pengguna di bawah ini SEBAGAI agent yang di-mention, dengan mengikuti instruksi agent dan skill-nya dan benar-benar menjalankan tugasnya (bukan sekadar menjelaskan). Jangan mengulang blok ini ke pengguna kecuali diminta.\n\n")
	for _, a := range agents {
		fmt.Fprintf(&b, "## Agent @%s (%s)\n", a.Slug, a.Name)
		if a.Label != "" {
			fmt.Fprintf(&b, "Label: %s\n", a.Label)
		}
		if a.Description != "" {
			fmt.Fprintf(&b, "%s\n", a.Description)
		}
		if a.Instructions != "" {
			fmt.Fprintf(&b, "\n%s\n", a.Instructions)
		}
		b.WriteString("\n")
	}
	for _, sk := range skills {
		fmt.Fprintf(&b, "## Skill #%s (%s)\n", sk.Slug, sk.Name)
		if sk.Label != "" {
			fmt.Fprintf(&b, "Label: %s\n", sk.Label)
		}
		if sk.Description != "" {
			fmt.Fprintf(&b, "%s\n\n", sk.Description)
		}
		if sk.Body != "" {
			b.WriteString(sk.Body)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("[/Konteks Ayumi Link]\n\n")
	return b.String()
}

// GeneratePromptAgent / GeneratePromptSkill: teks awal chat untuk minta Claude susun draft.
func GeneratePromptAgent(name, description string) string {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	return fmt.Sprintf(`Buatkan draft **Agent** Ayumi Link (tim internal perusahaan).

Nama yang diinginkan: %s
Deskripsi dari pengguna:
%s

Tugasmu:
1. Usulkan name, slug (kebab-case), label singkat, dan description yang jelas untuk agent ini.
2. Usulkan 1–5 **skill** terperinci yang cocok (masing-masing: name, slug, label, description, body = instruksi operasional lengkap dalam bahasa Indonesia/Inggris sesuai konteks).
3. Di akhir jawaban, keluarkan TEPAT SATU blok JSON (tanpa markdown di luar blok) dengan format:

`+"```"+`json
{
  "agent": { "name": "", "slug": "", "label": "", "description": "" },
  "skills": [
    { "name": "", "slug": "", "label": "", "description": "", "body": "" }
  ]
}
`+"```"+`

Setelah itu, pengguna akan menyimpan lewat menu Skills & Agents. Jangan membuat file di disk.`, nameOrDash(name), descriptionOrDash(description))
}

func GeneratePromptSkill(name, description string) string {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	return fmt.Sprintf(`Buatkan draft **Skill** Ayumi Link (instruksi terperinci untuk agent).

Nama yang diinginkan: %s
Deskripsi dari pengguna:
%s

Tugasmu:
1. Susun name, slug (kebab-case), label singkat, description, dan body (instruksi langkah-demi-langkah yang bisa diikuti Claude saat skill ini diaktifkan dengan #slug).
2. Di akhir jawaban, keluarkan TEPAT SATU blok JSON:

`+"```"+`json
{
  "skill": { "name": "", "slug": "", "label": "", "description": "", "body": "" }
}
`+"```"+`

Pengguna akan menyimpan lewat menu Skills & Agents. Jangan membuat file di disk.`, nameOrDash(name), descriptionOrDash(description))
}

func (s *Server) generateAgentsSkillsPrompt(c *fiber.Ctx) error {
	var b struct {
		Kind        string `json:"kind"` // agent | skill
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json tidak valid")
	}
	kind := strings.ToLower(strings.TrimSpace(b.Kind))
	var prompt string
	switch kind {
	case "agent":
		prompt = GeneratePromptAgent(b.Name, b.Description)
	case "skill":
		prompt = GeneratePromptSkill(b.Name, b.Description)
	default:
		return fiber.NewError(fiber.StatusBadRequest, "kind harus agent atau skill")
	}
	return c.JSON(fiber.Map{"kind": kind, "prompt": prompt})
}

func nameOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(belum ditentukan — usulkan sendiri)"
	}
	return s
}

func descriptionOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(belum ada — tanyakan singkat lalu usulkan draft lengkap)"
	}
	return s
}

// syncDisk mengimpor/menyelaraskan agent & skill yang dibuat lewat chat Claude (file di ~/.claude dan
// .claude/ pada folder session buatan Ayumi Link). Dibatasi sekali per 2 detik.
func (s *Server) syncDisk() {
	s.diskSyncMu.Lock()
	defer s.diskSyncMu.Unlock()
	if time.Since(s.diskSyncAt) < 2*time.Second {
		return
	}
	s.diskSyncAt = time.Now()
	roots := []string{s.Config.ClaudeHome}
	seen := map[string]bool{}
	for _, cwd := range s.Store.Registry() {
		if cwd == "" || seen[cwd] {
			continue
		}
		seen[cwd] = true
		roots = append(roots, filepath.Join(cwd, ".claude"))
	}
	s.Store.SyncFromDisk(roots)
}
