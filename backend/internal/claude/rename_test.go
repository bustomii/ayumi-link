package claude

import "testing"

func TestSetTitleOverride(t *testing.T) {
	svc, home := newService(t)
	writeTranscript(t, home, sid, line("user", "u1", `"pertanyaan awal"`))
	svc.Store.Register(sid, "/tmp/demo")

	title := func() string {
		for _, s := range svc.List() {
			if s.ID == sid {
				return s.Title
			}
		}
		t.Fatal("session tidak ditemukan")
		return ""
	}
	if got := title(); got != "pertanyaan awal" {
		t.Fatalf("judul awal = %q", got)
	}
	if err := svc.SetTitle(sid, "Nama Baru"); err != nil {
		t.Fatal(err)
	}
	if got := title(); got != "Nama Baru" {
		t.Fatalf("setelah rename = %q", got)
	}
	d, err := svc.Detail(sid, 20, "")
	if err != nil || d.Session.Title != "Nama Baru" {
		t.Fatalf("detail harus memakai nama baru: %v %+v", err, d)
	}
	_ = svc.SetTitle(sid, "")
	if got := title(); got != "pertanyaan awal" {
		t.Fatalf("reset = %q", got)
	}
}

func groupOf(t *testing.T, svc *Service, id string) string {
	t.Helper()
	for _, s := range svc.List() {
		if s.ID == id {
			return s.Group
		}
	}
	t.Fatal("session tidak ditemukan")
	return ""
}

func TestRenameGroupChain(t *testing.T) {
	svc, home := newService(t)
	writeTranscript(t, home, sid, line("user", "u1", `"a"`))
	svc.Store.Register(sid, "/tmp/demo") // group asal "demo"

	if g := groupOf(t, svc, sid); g != "demo" {
		t.Fatalf("awal = %q", g)
	}
	if err := svc.RenameGroup("demo", "Proyek X"); err != nil {
		t.Fatal(err)
	}
	if g := groupOf(t, svc, sid); g != "Proyek X" {
		t.Fatalf("rename 1 = %q", g)
	}
	if err := svc.RenameGroup("Proyek X", "Proyek Y"); err != nil {
		t.Fatal(err)
	}
	if g := groupOf(t, svc, sid); g != "Proyek Y" {
		t.Fatalf("rename berantai = %q", g)
	}
	if err := svc.RenameGroup("Proyek Y", "demo"); err != nil {
		t.Fatal(err)
	}
	if g := groupOf(t, svc, sid); g != "demo" {
		t.Fatalf("kembali ke nama asal = %q", g)
	}
	if n := len(svc.Store.State().GroupNames); n != 0 {
		t.Fatalf("pemetaan identitas harus dibersihkan, tersisa %d", n)
	}
}

func TestRenameGroupFollowsMovedSessions(t *testing.T) {
	svc, home := newService(t)
	writeTranscript(t, home, sid, line("user", "u1", `"a"`))
	svc.Store.Register(sid, "/tmp/demo")

	_ = svc.SetGroup(sid, "Pindahan")
	if err := svc.RenameGroup("Pindahan", "Pindahan Baru"); err != nil {
		t.Fatal(err)
	}
	if g := groupOf(t, svc, sid); g != "Pindahan Baru" {
		t.Fatalf("session yang dipindah harus ikut folder yang diganti: %q", g)
	}
}

func TestDeleteLocalClearsTitle(t *testing.T) {
	svc, home := newService(t)
	writeTranscript(t, home, sid, line("user", "u1", `"a"`))
	svc.Store.Register(sid, "/tmp/demo")
	_ = svc.SetTitle(sid, "Sementara")
	if _, err := svc.Delete(sid); err != nil {
		t.Fatal(err)
	}
	if _, ok := svc.Store.State().Titles[sid]; ok {
		t.Fatal("override judul harus ikut terhapus")
	}
}
