package pak

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInstallUninstall(t *testing.T) {
	root := t.TempDir()
	paks := filepath.Join(root, "Paks")
	os.MkdirAll(paks, 0o755)
	os.WriteFile(filepath.Join(paks, "pakchunk0-Windows.utoc"), []byte("x"), 0o644)

	ua, ue, _ := fakeAsset(map[int][4]float32{1: {4, 0.04, 0.5, 1}, 2: {1, 0.2, 0.3, 0.5}}, 0)
	var calls []string
	var zenInput string
	b := &Builder{Dir: filepath.Join(root, "work")}
	b.Run = func(args ...string) error {
		calls = append(calls, args[0])
		switch args[0] {
		case "to-legacy":
			d := filepath.Join(args[2], "Polaris", "Content", "Effect", "swl")
			os.MkdirAll(d, 0o755)
			os.WriteFile(filepath.Join(d, "MI_EF_SWL_trail01_blade.uasset"), ua, 0o644)
			os.WriteFile(filepath.Join(d, "MI_EF_SWL_trail01_blade.uexp"), ue, 0o644)
		case "to-zen":
			zenInput = args[1]
			base := strings.TrimSuffix(args[2], ".utoc")
			for _, ext := range modExts {
				os.WriteFile(base+ext, []byte("mod"), 0o644)
			}
		}
		return nil
	}

	files, skipped, err := b.Build(paks, []Edit{{Asset: "mi_ef_swl_trail01_blade", Color: [3]float64{0, 1, 0}}, {Asset: "MI_missing"}})
	_ = calls
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 || len(skipped) != 1 {
		t.Fatalf("files %v skipped %v", files, skipped)
	}
	// patched copy keeps the folder layout and scales to the original peak
	got, err := os.ReadFile(filepath.Join(zenInput, "Polaris", "Content", "Effect", "swl", "MI_EF_SWL_trail01_blade.uexp"))
	if err != nil {
		t.Fatal(err)
	}
	names, _ := LegacyNames(ua)
	ps := FindParams(got, names)
	if v := ps["basetexcolor"].Value; v != [4]float32{0, 4, 0, 1} {
		t.Fatalf("BaseTexColor = %v", v)
	}
	if v := ps["basetex2color"].Value; v != [4]float32{0, 1, 0, 0.5} {
		t.Fatalf("BaseTex2Color = %v", v)
	}

	// second build reuses the extracted originals
	if _, _, err := b.Build(paks, []Edit{{Asset: "MI_EF_SWL_trail01_blade", Color: [3]float64{1, 0, 0}}}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "to-legacy,to-zen,to-zen" {
		t.Fatalf("calls: %v", calls)
	}

	// a copy from an older build in ~mods is cleaned up
	os.MkdirAll(filepath.Join(paks, "~mods"), 0o755)
	os.WriteFile(filepath.Join(paks, "~mods", ModName+".utoc"), []byte("old"), 0o644)
	if err := Install(paks, ModName, files); err != nil || !Installed(paks, ModName) {
		t.Fatalf("install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(paks, "mods", ModName+".pak")); err != nil {
		t.Fatal("not in Paks\\mods")
	}
	if _, err := os.Stat(filepath.Join(paks, "~mods", ModName+".utoc")); err == nil {
		t.Fatal("old ~mods copy left behind")
	}
	if err := Install(paks, ModName, files); err != nil || !Installed(paks, ModName) {
		t.Fatalf("install: %v", err)
	}
	if err := Uninstall(paks, ModName); err != nil || Installed(paks, ModName) {
		t.Fatalf("uninstall: %v", err)
	}
}

func TestPaksFromExe(t *testing.T) {
	exe := filepath.Join("C:", "Steam", "TEKKEN 8", "Polaris", "Binaries", "Win64", "Polaris-Win64-Shipping.exe")
	want := filepath.Join("C:", "Steam", "TEKKEN 8", "Polaris", "Content", "Paks")
	if got := PaksFromExe(exe); got != want {
		t.Fatalf("%s", got)
	}
}

func TestBundledRetoc(t *testing.T) {
	b := &Builder{Dir: t.TempDir()}
	exe, err := b.retoc()
	if err != nil {
		t.Fatal(err)
	}
	if sum, _ := fileSHA(exe); sum != retocExeSHA {
		t.Fatal("checksum")
	}
}

func TestExpectGuardsAnyColor(t *testing.T) {
	ua, ue, _ := fakeAsset(map[int][4]float32{1: {1, 0.0104, 0.142, 1}, 2: {1, 0.229, 0.293, 0}}, 0)
	// memory says BaseTexColor is something else: nothing may be written
	n, err := patchColors(ua, ue, Edit{Color: [3]float64{0, 1, 0}, AnyColor: true,
		Expect: map[string][4]float32{"basetexcolor": {9, 9, 9, 1}}})
	if err != nil || n != 0 {
		t.Fatalf("patched %d (%v) despite mismatch", n, err)
	}
	n, _ = patchColors(ua, ue, Edit{Color: [3]float64{0, 1, 0}, AnyColor: true,
		Expect: map[string][4]float32{"basetex2color": {1, 0.229, 0.293, 0}}})
	if n != 1 {
		t.Fatalf("patched %d, want only the verified param", n)
	}
}

func TestRawSet(t *testing.T) {
	ua, ue, _ := fakeAsset(map[int][4]float32{1: {1, 0, 0, 1}}, 2)
	n, err := patchColors(ua, ue, Edit{Set: map[string][]float32{"basetexcolor": {0, 1, 0, 1}}})
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
	names, _ := LegacyNames(ua)
	if v := FindParams(ue, names)["basetexcolor"].Value; v != [4]float32{0, 1, 0, 1} {
		t.Fatal(v)
	}
	if _, err := patchColors(ua, ue, Edit{Set: map[string][]float32{"BaseTex2Color": {1}}}); err == nil {
		t.Fatal("omitted param should be an error")
	}
}

func TestCloneRename(t *testing.T) {
	root := t.TempDir()
	paks := filepath.Join(root, "Paks")
	os.MkdirAll(paks, 0o755)
	os.WriteFile(filepath.Join(paks, "a.utoc"), []byte("x"), 0o644)
	ua, ue, _ := fakeAsset(map[int][4]float32{1: {1, 0, 0, 1}}, 0)
	ua = append(ua, []byte("/Game/Effect/Character/ant/MI_EF_Ant_x\x00")...)
	var zenIn string
	b := &Builder{Dir: filepath.Join(root, "w")}
	b.Run = func(args ...string) error {
		if args[0] == "to-legacy" {
			d := filepath.Join(args[2], "Polaris", "Content", "Effect", "Character", "ant")
			os.MkdirAll(d, 0o755)
			os.WriteFile(filepath.Join(d, "MI_EF_Ant_x.uasset"), ua, 0o644)
			os.WriteFile(filepath.Join(d, "MI_EF_Ant_x.uexp"), ue, 0o644)
		} else {
			zenIn = args[1]
			base := strings.TrimSuffix(args[2], ".utoc")
			for _, ext := range modExts {
				os.WriteFile(base+ext, []byte("m"), 0o644)
			}
		}
		return nil
	}
	ren := [][2]string{{"/Game/Effect/Character/ant/MI_EF_Ant_x", "/Game/Effect/Character/swl/MI_EF_Swl_x"}, {"MI_EF_Ant_x", "MI_EF_Swl_x"}}
	_, _, err := b.Build(paks, []Edit{{Asset: "MI_EF_Swl_x", From: "MI_EF_Ant_x", Rename: ren, Color: [3]float64{0, 1, 0}}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(zenIn, "Polaris", "Content", "Effect", "Character", "swl", "MI_EF_Swl_x.uasset"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte("MI_EF_Ant_x")) || !bytes.Contains(got, []byte("/Game/Effect/Character/swl/MI_EF_Swl_x")) {
		t.Fatal("rename not applied")
	}
	if len(got) != len(ua) {
		t.Fatal("size changed")
	}
}

func TestFindsShareAPattern(t *testing.T) {
	ua, ue, _ := fakeAsset(map[int][4]float32{1: {1, 0, 0, 1}}, 0)
	pat := []byte{1, 2, 3, 4, 5, 6, 7, 0x07, 0x00}
	n, err := patchColors(ua, ue, Edit{Finds: []Find{{Pattern: pat, Off: 0, Data: []byte{9}}, {Pattern: pat, Off: 1, Data: []byte{8}}}})
	if err != nil || n != 2 || ue[0] != 9 || ue[1] != 8 {
		t.Fatalf("%d %v %v", n, err, ue[:4])
	}
}
