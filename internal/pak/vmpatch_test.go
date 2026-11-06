package pak

import (
	"os"
	"testing"
)

// EE_NSBEAM points at the game's legacy NS_EF_Swl_Beam.uexp (not shipped).
func TestVMPatchReal(t *testing.T) {
	path := os.Getenv("EE_NSBEAM")
	if path == "" {
		t.Skip("EE_NSBEAM not set")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	orig := append([]byte(nil), data...)
	n, err := applyVMs(data, map[string][3]float64{
		"swl-beam-bolt": {0.1, 1, 0.2}, "swl-beam-e8#1": {0.1, 1, 0.2}, "swl-beam-light": {0.1, 1, 0.2},
		"swl-beam-thunder": {0.1, 1, 0.2}, "swl-beam-head": {0.1, 1, 0.2}, "swl-beam-burst": {0.1, 1, 0.2},
		"swl-beam-glow001": {0.1, 1, 0.2}, "swl-beam-glowburst": {0.1, 1, 0.2}, "swl-beam-firecore001": {0.1, 1, 0.2},
		"swl-beam-fireblue001": {0.1, 1, 0.2},
	})
	if err != nil || n != 10 {
		t.Fatal(n, err)
	}
	if len(data) != len(orig) {
		t.Fatal("size changed")
	}
	// a second run must refuse (bytecode no longer matches)
	if err := ApplyVM(data, VMPatches["swl-beam-e8"], nil); err == nil {
		t.Fatal("patched twice")
	}
	if out := os.Getenv("EE_NSBEAM_OUT"); out != "" {
		os.WriteFile(out, data, 0o644)
	}
}

func TestInstallReplacesReadOnly(t *testing.T) {
	paks := t.TempDir()
	src := t.TempDir()
	os.MkdirAll(ModsDir(paks), 0o755)
	var files []string
	for _, ext := range []string{".pak", ".utoc", ".ucas"} {
		dst := ModsDir(paks) + "/X_P" + ext
		os.WriteFile(dst, []byte("old"), 0o444)
		f := src + "/X_P" + ext
		os.WriteFile(f, []byte("new"), 0o644)
		files = append(files, f)
	}
	if err := Install(paks, "X_P", files); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		b, _ := os.ReadFile(ModsDir(paks) + "/" + f[len(src)+1:])
		if string(b) != "new" {
			t.Fatal(f, string(b))
		}
	}
	if m, _ := os.ReadDir(ModsDir(paks)); len(m) != 3 {
		t.Fatal("leftover files", len(m))
	}
}
