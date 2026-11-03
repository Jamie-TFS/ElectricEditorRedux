package store

import (
	"strings"
	"testing"

	"app/internal/fx"
)

func tmpStore(t *testing.T) *Store {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	return Load()
}

func TestRoundTripCode(t *testing.T) {
	s := tmpStore(t)
	set := fx.Setting{Mode: "gradient", Brightness: 5, KeepFade: true,
		Stops: []fx.Stop{{Pos: 0, Color: "#00e5ff"}, {Pos: 0.6, Color: "#7c5cff"}, {Pos: 1, Color: "#ff3d81"}}}
	if err := s.Set("deviljin.electric.trail", set); err != nil {
		t.Fatal(err)
	}
	s.Set("kazuya.electric.bolt", fx.Setting{Mode: "solid", Color: "#ffffff", Brightness: 8})

	code, err := s.Export("deviljin.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(code, "EE1.") || len(code) > 400 {
		t.Fatalf("bad code %q (%d chars)", code, len(code))
	}

	s.Create("Other")
	s.Set("deviljin.electric.trail", fx.Setting{Mode: "native"})
	n, err := s.Import(code, fx.AllParts())
	if err != nil || n != 1 {
		t.Fatalf("import n=%d err=%v", n, err)
	}
	got := s.Active()["deviljin.electric.trail"]
	if got.Mode != "gradient" || len(got.Stops) != 3 || got.Stops[1].Color != "#7c5cff" {
		t.Fatalf("round trip lost data: %+v", got)
	}
	if _, ok := s.Active()["kazuya.electric.bolt"]; !ok {
		t.Fatal("create should copy the previous profile")
	}
}

func TestImportRejectsJunk(t *testing.T) {
	s := tmpStore(t)
	for _, c := range []string{"hello", "EE1.!!!", "EE1.AAAA"} {
		if _, err := s.Import(c, fx.AllParts()); err == nil {
			t.Fatalf("accepted %q", c)
		}
	}
}

func TestProfiles(t *testing.T) {
	s := tmpStore(t)
	if err := s.Delete(); err == nil {
		t.Fatal("must keep one profile")
	}
	if err := s.Create("  "); err == nil {
		t.Fatal("blank name accepted")
	}
	s.Create("B")
	if err := s.Rename("C"); err != nil || s.ActiveName() != "C" {
		t.Fatal("rename failed")
	}
	if err := s.Delete(); err != nil || s.ActiveName() != "Main" {
		t.Fatalf("delete failed: %v %s", err, s.ActiveName())
	}
	// persisted?
	s2 := Load()
	if len(s2.ProfileNames()) != 1 {
		t.Fatalf("not persisted: %v", s2.ProfileNames())
	}
}
