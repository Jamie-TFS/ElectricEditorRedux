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
			if e.Asset == "NS_EF_Swl_Beam" && len(e.VM) > 0 && allNative(e.VM) {
				continue // live anchors ride along with every mod
			}
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
	if len(edits) != 2 || edits[0].Asset != "NS_EF_Grl_Beam" || len(edits[0].VM) < 10 || !allNative(edits[1].VM) {
		t.Fatalf("%+v", edits)
	}
	for k := range edits[0].VM {
		if _, ok := pak.VMPatches[k]; !ok {
			t.Fatal("no patch", k)
		}
	}
}

func TestChainAuraScripts(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	s := store.Load()
	m := NewPakMods(s, NewEngine(s))
	edits := m.edits(store.Profile{"deviljin.chains.aura": fx.Setting{Mode: "solid", Color: "#ffffff", Brightness: 1}})
	got := map[string]int{}
	for _, e := range edits {
		got[e.Asset] = len(e.VM) + len(e.Params)
	}
	if got["NS_EF_Swl_Chain01"] == 0 || got["NS_EF_Swl_Chain02"] == 0 || got["MI_EF_Ant_S_Aura24"] == 0 {
		t.Fatalf("%v", got)
	}
}

func TestShapeIgnoresScriptColours(t *testing.T) {
	a := []pak.Edit{{Asset: "NS_EF_Swl_Beam", VM: map[string][3]float64{"swl-beam-light": {1, 0, 0}}}, {Asset: "MI_X", Color: [3]float64{1, 0, 0}}}
	b := []pak.Edit{{Asset: "NS_EF_Swl_Beam", VM: map[string][3]float64{"swl-beam-light": {0, 1, 0}}}, {Asset: "MI_X", Color: [3]float64{1, 0, 0}}}
	c := []pak.Edit{{Asset: "NS_EF_Swl_Beam", VM: map[string][3]float64{"swl-beam-light": {0, 1, 0}}}, {Asset: "MI_X", Color: [3]float64{0, 1, 0}}}
	if shape(a) != shape(b) {
		t.Fatal("script colour change should keep the shape")
	}
	if shape(a) == shape(c) {
		t.Fatal("material colour change must change the shape")
	}
}

func allNative(vm map[string][3]float64) bool {
	for k := range vm {
		if !strings.HasSuffix(k, "#native") {
			return false
		}
	}
	return true
}

func TestAnchorsMakeColourChangesLive(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	s := store.Load()
	m := NewPakMods(s, NewEngine(s))
	claws := store.Profile{"deviljin.claws.swipe": fx.Setting{Mode: "solid", Color: "#00ff00", Brightness: 1}}
	beam := store.Profile{"deviljin.claws.swipe": fx.Setting{Mode: "solid", Color: "#00ff00", Brightness: 1},
		"deviljin.beam.main": fx.Setting{Mode: "solid", Color: "#0000ff", Brightness: 1}}
	a, b := m.edits(claws), m.edits(beam)
	if shape(a) != shape(b) {
		t.Fatal("colouring an anchored beam layer must not need a new mod")
	}
	// the anchored tables differ only in the colour floats
	ca, cb := pak.VMColours(a), pak.VMColours(b)
	for _, n := range []string{"swl-beam-e8", "swl-beam-light"} {
		ta, _ := pak.LiveTable(n, ca[n])
		tb, _ := pak.LiveTable(n, cb[n])
		if len(ta) != len(tb) || string(ta) == string(tb) {
			t.Fatal(n)
		}
	}
}
