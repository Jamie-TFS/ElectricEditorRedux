package main

import (
	"strings"
	"testing"

	"app/internal/fx"
	"app/internal/pak"
	"app/internal/store"
)

func TestWS2CopyKeepsJinAndDevilJinApart(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	s := store.Load()
	m := NewPakMods(s, NewEngine(s))
	solid := fx.Setting{Mode: "solid", Color: "#00ff00", Brightness: 1}
	list := func(p store.Profile) string {
		var b []string
		for _, e := range m.edits(p) {
			x := e.Asset
			if e.Keep {
				x += "(keep)"
			}
			if e.From != "" {
				x += "<-" + e.From
			}
			b = append(b, x)
		}
		return strings.Join(b, " ")
	}

	// only Jin recoloured: Devil Jin still gets an untouched private copy
	got := list(store.Profile{"jin.trails.blade": solid})
	want := "DA_SWL_KisekiRender3(keep) MI_EF_Ant_trail03_blade MI_EF_Ws2_trail03_blade(keep)<-MI_EF_Ant_trail03_blade"
	if got != want {
		t.Fatalf("jin only:\n got %s\nwant %s", got, want)
	}
	// only WS2: the shared blade is untouched, the copy is coloured
	got = list(store.Profile{"deviljin.claws.ws2": solid})
	want = "DA_SWL_KisekiRender3(keep) MI_EF_Ws2_trail03_blade<-MI_EF_Ant_trail03_blade"
	if got != want {
		t.Fatalf("ws2 only:\n got %s\nwant %s", got, want)
	}
	// neither: nothing
	if got = list(store.Profile{}); got != "" {
		t.Fatal(got)
	}
}

func TestBeamScriptPatches(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	s := store.Load()
	m := NewPakMods(s, NewEngine(s))
	green := fx.Setting{Mode: "solid", Color: "#00ff00", Brightness: 1}
	blue := fx.Setting{Mode: "solid", Color: "#0000ff", Brightness: 2}
	edits := m.edits(store.Profile{"deviljin.beam.lightning": green, "deviljin.beam.glow": blue})
	if len(edits) != 2 {
		t.Fatalf("want texture + system, got %+v", edits)
	}
	ns := edits[0]
	if ns.Asset != "NS_EF_Swl_Beam" || ns.VM["swl-beam-bolt"] != [3]float64{0, 1, 0} || ns.VM["swl-beam-e8#0"] != [3]float64{0, 0, 2} {
		t.Fatalf("system edit %+v", ns)
	}
	if edits[1].Asset != "T_EF_COLA_Swl_Thunder03" || edits[1].Tint == nil {
		t.Fatalf("texture edit %+v", edits[1])
	}
	// glow alone touches only the system
	if e := m.edits(store.Profile{"deviljin.beam.glow": blue}); len(e) != 1 || e[0].Tint != nil {
		t.Fatalf("%+v", e)
	}
}

func TestKazuyaBeamExpands(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	s := store.Load()
	m := NewPakMods(s, NewEngine(s))
	edits := m.edits(store.Profile{"kazuya.beam.main": fx.Setting{Mode: "solid", Color: "#ff0000", Brightness: 1}})
	if len(edits) != 1 || edits[0].Asset != "NS_EF_Grl_Beam" || len(edits[0].VM) < 10 {
		t.Fatalf("%+v", edits)
	}
	for k := range edits[0].VM {
		if _, ok := pak.VMPatches[k]; !ok {
			t.Fatal("no patch", k)
		}
	}
}
