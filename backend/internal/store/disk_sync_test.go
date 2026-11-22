package store

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSyncFromDisk(t *testing.T) {
	st := New(t.TempDir())
	root := t.TempDir()
	write(t, filepath.Join(root, "skills", "deploy-app", "SKILL.md"), "---\nname: Deploy App\ndescription: Deploy ke staging\n---\n\nLangkah 1\n")
	write(t, filepath.Join(root, "agents", "devops.md"), "---\nname: devops\ndescription: Tim devops\nskills: deploy-app\n---\nIsi\n")

	if !st.SyncFromDisk([]string{root}) {
		t.Fatal("harus ada perubahan")
	}
	sk, ag := st.ListSkills(), st.ListAgents()
	if len(sk) != 1 || sk[0].Slug != "deploy-app" || sk[0].Body != "Langkah 1" || sk[0].Source != "disk" {
		t.Fatalf("skills: %+v", sk)
	}
	if len(ag) != 1 || ag[0].Slug != "devops" || len(ag[0].SkillIDs) != 1 || ag[0].SkillIDs[0] != sk[0].ID {
		t.Fatalf("agents: %+v", ag)
	}
	if st.SyncFromDisk([]string{root}) {
		t.Fatal("tanpa perubahan seharusnya false")
	}

	// update
	write(t, filepath.Join(root, "skills", "deploy-app", "SKILL.md"), "---\nname: Deploy App\ndescription: baru\n---\nLangkah 2\n")
	st.SyncFromDisk([]string{root})
	if got := st.ListSkills()[0]; got.Body != "Langkah 2" || got.Description != "baru" {
		t.Fatalf("update: %+v", got)
	}

	// delete
	os.RemoveAll(filepath.Join(root, "skills", "deploy-app"))
	os.Remove(filepath.Join(root, "agents", "devops.md"))
	st.SyncFromDisk([]string{root})
	if len(st.ListSkills()) != 0 || len(st.ListAgents()) != 0 {
		t.Fatal("item disk yang filenya hilang harus terhapus")
	}
}
