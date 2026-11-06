package fx

import "testing"

func lut(vals ...float32) []float32 { return vals }

func TestNativeIsCopy(t *testing.T) {
	o := lut(1, 2, 3, 1, 4, 5, 6, 0.5)
	r := Render(o, Setting{Mode: "native"})
	if !Same(o, r) {
		t.Fatal("native should return original")
	}
	r[0] = 9
	if o[0] == 9 {
		t.Fatal("native must not alias the original")
	}
}

func TestSolidKeepFade(t *testing.T) {
	// two samples: bright then half-bright, alpha fades
	o := lut(2, 2, 2, 1, 1, 1, 1, 0.25)
	r := Render(o, Setting{Mode: "solid", Color: "#ff0000", Brightness: 4, KeepFade: true})
	want := lut(4, 0, 0, 1, 2, 0, 0, 0.25)
	if !Same(r, want) {
		t.Fatalf("got %v want %v", r, want)
	}
}

func TestSolidFlat(t *testing.T) {
	o := lut(2, 2, 2, 1, 1, 1, 1, 0.25)
	r := Render(o, Setting{Mode: "solid", Color: "#00ff00", Brightness: 3, KeepFade: false})
	want := lut(0, 3, 0, 1, 0, 3, 0, 1)
	if !Same(r, want) {
		t.Fatalf("got %v want %v", r, want)
	}
}

func TestGradientEnds(t *testing.T) {
	o := make([]float32, 4*3)
	for i := range o {
		o[i] = 1
	}
	s := Setting{Mode: "gradient", Brightness: 1, KeepFade: true,
		Stops: []Stop{{1, "#0000ff"}, {0, "#ff0000"}}} // unsorted on purpose
	r := Render(o, s)
	if r[0] != 1 || r[2] != 0 {
		t.Fatalf("start should be red: %v", r[:4])
	}
	if r[8] != 0 || r[10] != 1 {
		t.Fatalf("end should be blue: %v", r[8:])
	}
	if r[4] < 0.49 || r[4] > 0.51 || r[6] < 0.49 || r[6] > 0.51 {
		t.Fatalf("middle should be half/half: %v", r[4:8])
	}
}

func TestCatalogKeysUnique(t *testing.T) {
	seen := map[string]bool{}
	paths := map[string]string{}
	for _, c := range Catalog {
		for _, e := range c.Effects {
			for _, p := range e.Parts {
				k := Key(c.ID, e.ID, p.ID)
				if seen[k] {
					t.Fatalf("duplicate key %s", k)
				}
				seen[k] = true
				if (e.Kind == "pak") != (p.Asset != "" || len(p.VMAssets) > 0) || (e.Kind == "pak" && len(p.Paths) > 0) {
					t.Fatalf("%s: pak parts need an Asset and no curve paths", k)
				}
				for _, path := range p.Paths {
					if other, ok := paths[path]; ok {
						t.Fatalf("path used twice: %s (%s, %s)", path, other, k)
					}
					paths[path] = k
				}
			}
		}
	}
	if len(seen) != 57 {
		t.Fatalf("expected 57 parts, got %d", len(seen))
	}
}

func TestRenderRGB(t *testing.T) {
	orig := [3][]float32{{0, 1.97, 1}, {0, 0, 0, 0}, {0, 5, 10}}
	out := RenderRGB(orig, [3]int{3, 4, 3}, Setting{Mode: "solid", Color: "#00ff00", Brightness: 1})
	if out[0][1] != 0 || out[1][3] < 0.99 || out[1][3] > 1.01 || out[2][2] != 0 {
		t.Fatalf("%v", out)
	}
	// green takes the red curve's shape, resampled to its own length
	if g := out[1]; g[0] != 0 || g[1] < 1.9 || g[2] < 1.9 {
		t.Fatalf("shape: %v", g)
	}
}

func TestRenderVec(t *testing.T) {
	got := RenderVec([]float32{1, 0.25, 0.25, 0.5, 0.1, 0.1}, Setting{Mode: "solid", Color: "#00ff00", Brightness: 1})
	if got[0] != 0 || got[1] != 1 || got[2] != 0 || got[4] != 0.5 {
		t.Fatal(got)
	}
}
