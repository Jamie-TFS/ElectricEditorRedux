package main

import (
	"testing"

	"app/internal/fx"
	"app/internal/store"
)

type fakeGame struct {
	curves map[string][]float32
	addr   map[string]uint64
	writes int
}

func (f *fakeGame) Loaded(p string) bool { _, ok := f.curves[p]; return ok }
func (f *fakeGame) Read(p string) ([]float32, error) {
	return append([]float32(nil), f.curves[p]...), nil
}
func (f *fakeGame) Addr(p string) uint64 { return f.addr[p] }
func (f *fakeGame) Write(p string, v []float32) error {
	f.writes++
	f.curves[p] = append([]float32(nil), v...)
	return nil
}

func TestUnhookRestoresExactlyAndThenLeavesAlone(t *testing.T) {
	path := "/Game/x.x:Beam.EventScript_0.Curve_0"
	parts := map[string]fx.Part{"k.e.bolt": {ID: "bolt", Paths: []string{path}}}
	orig := []float32{0.2, 0.4, 3.1, 1, 0.1, 0.3, 2.9, 0.5}
	g := &fakeGame{curves: map[string][]float32{path: append([]float32(nil), orig...)}, addr: map[string]uint64{path: 0x1000}}
	trk := map[string]*tracked{}

	// native from the start: never written
	syncCurves(g, parts, store.Profile{}, trk)
	if g.writes != 0 {
		t.Fatalf("native layer was written %d times", g.writes)
	}

	// recolour
	red := store.Profile{"k.e.bolt": {Mode: "solid", Color: "#ff0000", Brightness: 5, KeepFade: true}}
	_, applied, _ := syncCurves(g, parts, red, trk)
	if applied != 1 || g.curves[path][1] != 0 {
		t.Fatalf("recolour not applied: %v", g.curves[path])
	}
	w := g.writes

	// stable: no rewrites while nothing changes
	syncCurves(g, parts, red, trk)
	if g.writes != w {
		t.Fatal("rewrote an already-correct curve")
	}

	// back to game colours: exact original, once
	syncCurves(g, parts, store.Profile{}, trk)
	if !fx.Same(g.curves[path], orig) {
		t.Fatalf("not restored exactly: %v", g.curves[path])
	}
	w = g.writes
	// the game changes it itself afterwards: we must not fight it
	g.curves[path][0] = 9
	syncCurves(g, parts, store.Profile{}, trk)
	if g.writes != w || g.curves[path][0] != 9 {
		t.Fatal("unhooked layer was touched again")
	}
}

func TestReloadedCurveTakesFreshOriginal(t *testing.T) {
	path := "/Game/x.x:Beam.EventScript_0.Curve_0"
	parts := map[string]fx.Part{"k.e.bolt": {ID: "bolt", Paths: []string{path}}}
	g := &fakeGame{curves: map[string][]float32{path: {1, 1, 1, 1}}, addr: map[string]uint64{path: 0x1000}}
	trk := map[string]*tracked{}
	syncCurves(g, parts, store.Profile{"k.e.bolt": {Mode: "solid", Color: "#00ff00", Brightness: 2, KeepFade: false}}, trk)

	// game reloads the effect at a new address with its own colours
	g.curves[path] = []float32{0.5, 0.5, 0.5, 1}
	g.addr[path] = 0x2000
	syncCurves(g, parts, store.Profile{}, trk)
	if trk[path].Orig[0] != 0.5 || trk[path].Written {
		t.Fatalf("reload not detected: %+v", trk[path])
	}
}

func TestIsBeta(t *testing.T) {
	for v, want := range map[string]bool{"1.1.0-beta.1": true, "1.1.0-preview": true, "1.1.0": false, "dev": false} {
		if isBeta(v) != want {
			t.Errorf("isBeta(%q) = %v", v, !want)
		}
	}
}
