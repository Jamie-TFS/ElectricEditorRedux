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
				for _, path := range p.Paths {
					if other, ok := paths[path]; ok {
						t.Fatalf("path used twice: %s (%s, %s)", path, other, k)
					}
					paths[path] = k
				}
			}
		}
	}
	if len(seen) != 28 {
		t.Fatalf("expected 28 parts, got %d", len(seen))
	}
}
