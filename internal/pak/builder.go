package pak

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Builds a small mod pak (ElectricEditor_P.utoc/.ucas/.pak) that recolours
// trail materials, using retoc (MIT, github.com/trumank/retoc):
//
//   1. retoc to-legacy pulls the original materials out of Tekken's paks
//      once, into a cache (redone when the game updates)
//   2. the colour parameters are patched in Go (legacy.go)
//   3. retoc to-zen packs the patched copies for UE 5.2
//   4. the three files are copied into Paks\mods

//go:embed retoc.zip
var retocZip []byte

//go:embed RETOC_LICENSE.txt
var RetocLicense string

const (
	retocExeSHA = "a145f3557ec60b163ed1b78b2f380232d2753fa30350c8013c98e171db3ae7e7"
	ModName     = "ElectricEditor_P"
	FinderName  = "ElectricEditorFinder_P"
	TestName    = "ElectricEditorTest_P"
	ueVersion   = "UE5_2"
)

var modExts = []string{".utoc", ".ucas", ".pak"}

// ErrLocked means Tekken has the mod files open; try again once it closes.
var ErrLocked = errors.New("Tekken is using the mod files")

// Edit recolours one material: every colour parameter it has becomes Color
// scaled to that parameter's original peak, so the glow strength is kept.
type Edit struct {
	Asset string     // e.g. MI_EF_SWL_trail01_blade (matched case-insensitively)
	Color [3]float64 // 0..1 picked colour, already multiplied by brightness
	// Params names the colour parameters to patch (default ColorParams).
	Params []string
	// AnyColor patches every colour-looking vector parameter instead of
	// just ColorParams (used by the effect finder).
	AnyColor bool
	// Expect, when set, limits AnyColor to parameters whose stored value
	// matches what the game has in memory (lower-case name -> value). A
	// match proves the bytes really are that parameter, so nothing else in
	// the file can be overwritten by a false hit.
	Expect map[string][4]float32
	// Set writes raw values to named scalar/vector parameters (research
	// test mods): 1 float for a scalar, 4 for a colour.
	Set map[string][]float32
	// Raw overwrites bytes of the .uexp at fixed offsets (research only).
	Raw map[int][]byte
	// Finds overwrites bytes relative to a byte pattern that must occur
	// exactly once in the .uexp (research only).
	Finds []Find

	// RawZen patches the game's own packaged bytes for this asset instead of
	// the legacy round trip (for assets that crash after conversion). Only
	// Tint is supported this way for now.
	RawZen bool

	// Tint recolours a DXT1/DXT5 texture asset (keeps brightness, takes the
	// hue of the colour), including the mips in its .ubulk.
	Tint *[3]float64

	// VM rewrites Niagara script constants (VMPatches key, or "key#slot"
	// for patches with several colours -> colour) so an emitter whose
	// colour is baked into its script takes a new one.
	VM map[string][3]float64

	// From builds Asset as a renamed copy of another game asset (Asset
	// doesn't exist in the game). Rename lists same-length text swaps
	// applied to both files and the package path, e.g. the package path
	// and object name. Keep skips colour patching (copy/rename only).
	From   string
	Rename [][2]string
	Keep   bool
}

type Find struct {
	Pattern []byte
	Off     int
	Data    []byte
}

func (e Edit) source() string {
	if e.From != "" {
		return e.From
	}
	return e.Asset
}

// applyRenames swaps text in place; every pair must keep its length so no
// offsets in the package move.
func applyRenames(data []byte, pairs [][2]string) ([]byte, error) {
	for _, p := range pairs {
		if len(p[0]) != len(p[1]) {
			return nil, fmt.Errorf("rename %q -> %q changes length", p[0], p[1])
		}
		data = bytes.ReplaceAll(data, []byte(p[0]), []byte(p[1]))
	}
	return data, nil
}

func near(a, b [4]float32) bool {
	for i := range a {
		d := float64(a[i] - b[i])
		if d < 0 {
			d = -d
		}
		if d > 1e-4+1e-4*float64(max(a[i], -a[i])) {
			return false
		}
	}
	return true
}

// Params patched on trail materials.
var ColorParams = []string{"BaseTexColor", "BaseTex2Color"}

type Builder struct {
	Dir  string // working folder, e.g. %APPDATA%\ElectricEditor\pak
	Name string // mod file name without extension, e.g. ElectricEditor_P
	// Always lists assets to extract alongside whatever a build needs, so
	// switching between them doesn't re-run the extraction.
	Always []string

	// Run replaces the bundled retoc (tests).
	Run func(args ...string) error
}

func (b *Builder) name() string {
	if b.Name == "" {
		return ModName
	}
	return b.Name
}

func (b *Builder) log(format string, args ...any) {
	_ = os.MkdirAll(b.Dir, 0o755)
	f, err := os.OpenFile(filepath.Join(b.Dir, "log.txt"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

// retoc unpacks the bundled retoc.exe into a writable folder (it downloads
// its Oodle DLL next to itself on first use, with a hash check).
func (b *Builder) retoc() (string, error) {
	dir := filepath.Join(b.Dir, "tools")
	exe := filepath.Join(dir, "retoc.exe")
	if sum, err := fileSHA(exe); err == nil && sum == retocExeSHA {
		return exe, nil
	}
	zr, err := zip.NewReader(bytes.NewReader(retocZip), int64(len(retocZip)))
	if err != nil {
		return "", err
	}
	for _, f := range zr.File {
		if !strings.EqualFold(filepath.Base(f.Name), "retoc.exe") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}
		h := sha256.Sum256(data)
		if hex.EncodeToString(h[:]) != retocExeSHA {
			return "", errors.New("bundled retoc.exe failed its checksum")
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(exe, data, 0o755); err != nil {
			return "", err
		}
		_ = os.WriteFile(filepath.Join(dir, "LICENSE-retoc.txt"), []byte(RetocLicense), 0o644)
		return exe, nil
	}
	return "", errors.New("retoc.exe missing from bundle")
}

func fileSHA(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (b *Builder) run(args ...string) error {
	if b.Run != nil {
		return b.Run(args...)
	}
	exe, err := b.retoc()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = filepath.Dir(exe)
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	b.log("retoc %s\n%s", strings.Join(args, " "), tail(out, 4000))
	if err != nil {
		return fmt.Errorf("retoc %s failed: %v (details in %s)", args[0], err, filepath.Join(b.Dir, "log.txt"))
	}
	return nil
}

func tail(b []byte, n int) string {
	if len(b) > n {
		b = b[len(b)-n:]
	}
	return string(b)
}

// gameStamp changes whenever Tekken is updated (its own containers change).
func gameStamp(paks string) (string, error) {
	ents, err := os.ReadDir(paks)
	if err != nil {
		return "", fmt.Errorf("can't read the game's Paks folder: %v", err)
	}
	var parts []string
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".utoc") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%d:%d", e.Name(), info.Size(), info.ModTime().Unix()))
	}
	if len(parts) == 0 {
		return "", errors.New("no game containers found in " + paks)
	}
	sort.Strings(parts)
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:8]), nil
}

// originals makes sure the untouched materials are extracted for this game
// version and returns them indexed by lower-case asset name -> .uasset path.
// The cache is reused while the game is unchanged and it was extracted with
// a request covering every asset in need.
func (b *Builder) originals(paks string, need []string) (map[string]string, error) {
	need = append(append([]string(nil), need...), b.Always...)
	dir := filepath.Join(b.Dir, "original")
	stampFile := filepath.Join(dir, ".stamp")
	reqFile := filepath.Join(dir, ".requested")
	stamp, err := gameStamp(paks)
	if err != nil {
		return nil, err
	}
	requested := map[string]bool{}
	if old, _ := os.ReadFile(stampFile); string(old) == stamp {
		raw, _ := os.ReadFile(reqFile)
		for _, n := range strings.Split(string(raw), "\n") {
			requested[strings.ToLower(n)] = true
		}
	}
	fresh := len(requested) == 0
	for _, n := range need {
		if !requested[strings.ToLower(n)] {
			fresh = true
		}
	}
	if fresh {
		_ = os.RemoveAll(dir)
		args := []string{"to-legacy", paks, dir, "--no-shaders"}
		for _, n := range need {
			args = append(args, "-f", "/"+n+".") // retoc: case-sensitive substring of the package file path
		}
		if err := b.run(args...); err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
		_ = os.MkdirAll(dir, 0o755)
		if err := os.WriteFile(stampFile, []byte(stamp), 0o644); err != nil {
			return nil, err
		}
		_ = os.WriteFile(reqFile, []byte(strings.Join(need, "\n")), 0o644)
	}
	idx := map[string]string{}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".uasset") {
			idx[strings.ToLower(strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())))] = p
		}
		return nil
	})
	if len(idx) == 0 {
		_ = os.Remove(stampFile)
		return nil, errors.New("couldn't find those materials in the game files")
	}
	return idx, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// findRaw locates the value of a scalar or vector parameter by name (must
// be present exactly once).
func findRaw(uexp []byte, names []string, name string) (int, error) {
	for idx, n := range names {
		if !strings.EqualFold(n, name) {
			continue
		}
		pat := make([]byte, 8, 13)
		binary.LittleEndian.PutUint32(pat, uint32(idx))
		pat = append(pat, paramTail...)
		first := bytes.Index(uexp, pat)
		if first < 0 {
			return 0, fmt.Errorf("%s isn't stored in this material (it uses the parent's default)", name)
		}
		if bytes.Index(uexp[first+1:], pat) >= 0 {
			return 0, fmt.Errorf("%s: ambiguous", name)
		}
		return first + len(pat), nil
	}
	return 0, fmt.Errorf("%s: no such parameter name in this material", name)
}

// colourish picks parameters that are likely colours by name.
func colourish(name string) bool {
	l := strings.ToLower(name)
	for _, w := range []string{"color", "colour", "col", "tint", "emissive", "glow"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// patchColors applies one Edit to a .uexp in memory. Returns how many
// parameters changed.
func patchColors(uasset, uexp []byte, e Edit) (int, error) {
	names, err := LegacyNames(uasset)
	if err != nil {
		return 0, err
	}
	found := FindParams(uexp, names)
	var targets []Param
	if e.AnyColor {
		for key, p := range found {
			if e.Expect != nil {
				if want, ok := e.Expect[key]; !ok || !near(want, p.Value) {
					continue
				}
			} else if !colourish(p.Name) {
				continue
			}
			targets = append(targets, p)
		}
	} else if (len(e.Set) == 0 && len(e.Raw) == 0 && len(e.Finds) == 0 && len(e.VM) == 0) || len(e.Params) > 0 {
		names := e.Params
		if len(names) == 0 {
			names = ColorParams
		}
		for _, name := range names {
			if p, ok := found[strings.ToLower(name)]; ok {
				targets = append(targets, p) // all-zero originals aren't stored: nothing to patch
			}
		}
	}
	n := 0
	for off, data := range e.Raw {
		if off < 0 || off+len(data) > len(uexp)-4 { // never touch the package footer
			return n, fmt.Errorf("@%d is outside this file (%d bytes)", off, len(uexp)-4)
		}
		copy(uexp[off:], data)
		n++
	}
	// locate every pattern before writing, so patches that share a pattern
	// (or overlap another's) still find it
	at := make([]int, len(e.Finds))
	for i, f := range e.Finds {
		p := bytes.Index(uexp, f.Pattern)
		if p < 0 {
			return n, fmt.Errorf("byte pattern not found")
		}
		if bytes.Index(uexp[p+1:], f.Pattern) >= 0 {
			return n, fmt.Errorf("byte pattern found more than once")
		}
		if p+f.Off < 0 || p+f.Off+len(f.Data) > len(uexp)-4 {
			return n, fmt.Errorf("patch outside the file")
		}
		at[i] = p
	}
	for i, f := range e.Finds {
		copy(uexp[at[i]+f.Off:], f.Data)
		n++
	}
	if len(e.VM) > 0 {
		k, err := applyVMs(uexp, e.VM)
		if err != nil {
			return n, err
		}
		n += k
	}
	for name, vals := range e.Set {
		off, err := findRaw(uexp, names, name)
		if err != nil {
			return n, err
		}
		if off+4*len(vals) > len(uexp) {
			return n, fmt.Errorf("%s: out of range", name)
		}
		// a zero value isn't stored at all (the bytes there belong to the
		// next field), so refuse anything that doesn't look like a value
		for k := range vals {
			f := float64(math.Float32frombits(binary.LittleEndian.Uint32(uexp[off+4*k:])))
			if math.IsNaN(f) || math.Abs(f) >= 10000 || (f != 0 && math.Abs(f) < 1e-5) {
				return n, fmt.Errorf("%s is 0 in this material, so it isn't stored and can't be changed yet", name)
			}
		}
		for k, f := range vals {
			binary.LittleEndian.PutUint32(uexp[off+4*k:], math.Float32bits(f))
		}
		n++
	}
	for _, p := range targets {
		peak := float64(max(p.Value[0], p.Value[1], p.Value[2]))
		if peak <= 0 || (e.AnyColor && peak < 1) {
			peak = 1
		}
		c := e.Color
		v := [4]float32{float32(c[0] * peak), float32(c[1] * peak), float32(c[2] * peak), p.Value[3]}
		if err := SetParam(uexp, p, v); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// Build makes the mod files in <Dir>\out. Returns the paths of the three
// files and any assets it couldn't patch (missing or no colour params).
func (b *Builder) Build(paks string, edits []Edit) (files []string, skipped []string, err error) {
	if len(edits) == 0 {
		return nil, nil, errors.New("nothing to build")
	}
	need := make([]string, 0, len(edits))
	for _, e := range edits {
		need = append(need, e.source())
	}
	orig, err := b.originals(paks, need)
	if err != nil {
		return nil, nil, err
	}
	origRoot := filepath.Join(b.Dir, "original")
	work := filepath.Join(b.Dir, "work")
	out := filepath.Join(b.Dir, "out")
	_ = os.RemoveAll(work)
	_ = os.RemoveAll(out)
	patched := 0
	for _, e := range edits {
		ua, ok := orig[strings.ToLower(e.source())]
		if !ok {
			skipped = append(skipped, e.Asset)
			continue
		}
		ue := strings.TrimSuffix(ua, filepath.Ext(ua)) + ".uexp"
		uaData, err1 := os.ReadFile(ua)
		ueData, err2 := os.ReadFile(ue)
		if err1 != nil || err2 != nil {
			skipped = append(skipped, e.Asset)
			continue
		}
		bulkPath := strings.TrimSuffix(ua, filepath.Ext(ua)) + ".ubulk"
		bulkData, _ := os.ReadFile(bulkPath) // nil if the asset has none
		n := 1
		var perr error
		if e.RawZen {
			// packed unchanged; the real bytes are swapped in after to-zen
		} else if e.Tint != nil {
			perr = TintTexture(ueData, bulkData, *e.Tint)
		} else if !e.Keep {
			n, perr = patchColors(uaData, ueData, e)
		}
		if perr == nil && len(e.Rename) > 0 {
			if uaData, perr = applyRenames(uaData, e.Rename); perr == nil {
				ueData, perr = applyRenames(ueData, e.Rename)
			}
		}
		if perr != nil || n == 0 {
			b.log("%s: not patched (%v, %d params)", e.Asset, perr, n)
			if perr != nil {
				skipped = append(skipped, e.Asset+": "+perr.Error())
			} else {
				skipped = append(skipped, e.Asset)
			}
			continue
		}
		rel, _ := filepath.Rel(origRoot, ua)
		if len(e.Rename) > 0 {
			// the package path follows the renames too (it is the package name)
			slash := filepath.ToSlash(rel)
			if i := strings.Index(slash, "/Content/"); i >= 0 {
				game := "/Game/" + slash[i+len("/Content/"):]
				game = strings.TrimSuffix(game, filepath.Ext(game))
				renamed, _ := applyRenames([]byte(game), e.Rename)
				slash = slash[:i] + "/Content/" + strings.TrimPrefix(string(renamed), "/Game/") + ".uasset"
				rel = filepath.FromSlash(slash)
			}
		}
		dstUA := filepath.Join(work, rel)
		if err := os.MkdirAll(filepath.Dir(dstUA), 0o755); err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(dstUA, uaData, 0o644); err != nil {
			return nil, nil, err
		}
		if bulkData != nil {
			if err := os.WriteFile(strings.TrimSuffix(dstUA, filepath.Ext(dstUA))+".ubulk", bulkData, 0o644); err != nil {
				return nil, nil, err
			}
		}
		if err := os.WriteFile(strings.TrimSuffix(dstUA, filepath.Ext(dstUA))+".uexp", ueData, 0o644); err != nil {
			return nil, nil, err
		}
		patched++
	}
	if patched == 0 {
		if len(skipped) > 0 {
			return nil, skipped, errors.New("nothing could be patched - " + strings.Join(skipped, "; "))
		}
		return nil, skipped, errors.New("none of the materials could be patched")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, nil, err
	}
	utoc := filepath.Join(out, b.name()+".utoc")
	if err := b.run("to-zen", work, utoc, "--version", ueVersion); err != nil {
		return nil, skipped, err
	}
	var raw []Edit
	for _, e := range edits {
		if e.RawZen {
			raw = append(raw, e)
		}
	}
	if len(raw) > 0 {
		if err := b.swapRawZen(paks, utoc, raw); err != nil {
			return nil, skipped, err
		}
	}
	for _, ext := range modExts {
		p := filepath.Join(out, b.name()+ext)
		if _, err := os.Stat(p); err != nil {
			return nil, skipped, fmt.Errorf("retoc didn't produce %s", filepath.Base(p))
		}
		files = append(files, p)
	}
	return files, skipped, nil
}

// Tekken 8 loads loose mods from Paks\mods. ~mods is only checked so files
// from the first test build get cleaned up.
var modFolders = []string{"mods", "~mods"}

// Export extracts game assets whose path contains any filter (legacy
// .uasset/.uexp with readable names) and zips them, for research.
func (b *Builder) Export(paks string, filters []string, zipPath string) (int, error) {
	if len(filters) == 0 {
		return 0, errors.New("give at least one filter")
	}
	dir := filepath.Join(b.Dir, "export")
	_ = os.RemoveAll(dir)
	defer os.RemoveAll(dir)
	args := []string{"to-legacy", paks, dir, "--no-shaders"}
	for _, f := range filters {
		args = append(args, "-f", f)
	}
	if err := b.run(args...); err != nil {
		return 0, err
	}
	out, err := os.Create(zipPath)
	if err != nil {
		return 0, err
	}
	zw := zip.NewWriter(out)
	n := 0
	werr := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		n++
		return err
	})
	if err := zw.Close(); werr == nil {
		werr = err
	}
	if err := out.Close(); werr == nil {
		werr = err
	}
	if werr != nil {
		return 0, werr
	}
	if n == 0 {
		_ = os.Remove(zipPath)
		return 0, errors.New("nothing in the game files matched those filters (they're case-sensitive)")
	}
	return n, nil
}

// ModsDir is the folder the mod is installed into.
func ModsDir(paks string) string { return filepath.Join(paks, "mods") }

// Install copies built files into the mods folder. Each file is written
// beside its target and renamed over it, so a locked target leaves the old
// one intact. Copies left in other mod folders by older builds are removed.
func Install(paks, name string, files []string) error {
	dir := ModsDir(paks)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// copy everything first, then swap, so a failure never leaves a mix of
	// old and new files (the game can't load a mismatched set)
	var tmps []string
	defer func() {
		for _, t := range tmps {
			_ = os.Remove(t)
		}
	}()
	for _, src := range files {
		tmp := filepath.Join(dir, filepath.Base(src)) + ".new"
		if err := copyFile(src, tmp); err != nil {
			return err
		}
		tmps = append(tmps, tmp)
	}
	for _, tmp := range tmps {
		dst := strings.TrimSuffix(tmp, ".new")
		if err := replaceFile(tmp, dst); err != nil {
			return fmt.Errorf("%w (%s: %v)", ErrLocked, filepath.Base(dst), err)
		}
	}
	tmps = nil
	for _, f := range modFolders {
		if d := filepath.Join(paks, f); d != dir {
			_ = removeMod(d, name)
		}
	}
	return nil
}

// replaceFile moves tmp over dst, clearing a read-only flag on dst first
// (files copied by hand can carry one) and retrying briefly.
func replaceFile(tmp, dst string) error {
	var err error
	for try := 0; try < 3; try++ {
		if err = os.Rename(tmp, dst); err == nil {
			return nil
		}
		_ = os.Chmod(dst, 0o666)
		if rerr := os.Remove(dst); rerr == nil || errors.Is(rerr, fs.ErrNotExist) {
			if err = os.Rename(tmp, dst); err == nil {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return err
}

func removeMod(dir, name string) error {
	for _, ext := range modExts {
		p := filepath.Join(dir, name+ext)
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			_ = os.Chmod(p, 0o666)
			if err = os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("%w (%s: %v)", ErrLocked, filepath.Base(p), err)
			}
		}
	}
	return nil
}

// Uninstall removes the mod files from every mod folder (back to game colours).
func Uninstall(paks, name string) error {
	var err error
	for _, f := range modFolders {
		if e := removeMod(filepath.Join(paks, f), name); e != nil {
			err = e
		}
	}
	return err
}

// Installed reports whether the mod files are in the mods folder.
func Installed(paks, name string) bool {
	_, err := os.Stat(filepath.Join(ModsDir(paks), name+".utoc"))
	return err == nil
}

// PaksFromExe turns ...\Polaris\Binaries\Win64\Polaris-Win64-Shipping.exe
// into ...\Polaris\Content\Paks.
func PaksFromExe(exe string) string {
	if exe == "" {
		return ""
	}
	polaris := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	return filepath.Join(polaris, "Content", "Paks")
}

// swapRawZen replaces the converted chunks of RawZen assets with the game's
// original packaged bytes (patched). retoc's conversion still provides the
// container header and chunk ids; only the payloads change:
//
//	unpack-raw mod -> get each original chunk from the game -> patch ->
//	pack-raw mod
func (b *Builder) swapRawZen(paks, utoc string, edits []Edit) error {
	rawDir := filepath.Join(b.Dir, "raw")
	_ = os.RemoveAll(rawDir)
	if err := b.run("unpack-raw", utoc, rawDir); err != nil {
		return err
	}
	mf, err := os.ReadFile(filepath.Join(rawDir, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		ChunkPaths map[string]string `json:"chunk_paths"`
	}
	if err := json.Unmarshal(mf, &manifest); err != nil {
		return fmt.Errorf("raw manifest: %v", err)
	}
	for _, e := range edits {
		var pkgID, bulkID string
		for id, path := range manifest.ChunkPaths {
			base := filepath.Base(filepath.FromSlash(path))
			stem := strings.TrimSuffix(base, filepath.Ext(base))
			if !strings.EqualFold(stem, e.Asset) {
				continue
			}
			switch strings.ToLower(filepath.Ext(base)) {
			case ".uasset", ".umap":
				pkgID = id
			case ".ubulk":
				bulkID = id
			}
		}
		if pkgID == "" {
			return fmt.Errorf("%s: package chunk not found in the built mod", e.Asset)
		}
		get := func(id string) ([]byte, error) {
			tmp := filepath.Join(rawDir, id+".orig")
			if err := b.run("get", paks, id, tmp); err != nil {
				return nil, err
			}
			return os.ReadFile(tmp)
		}
		pkg, err := get(pkgID)
		if err != nil {
			return err
		}
		var bulk []byte
		if bulkID != "" {
			if bulk, err = get(bulkID); err != nil {
				return err
			}
		}
		if e.Tint != nil {
			if err := TintTexture(pkg, bulk, *e.Tint); err != nil {
				return fmt.Errorf("%s: %v", e.Asset, err)
			}
		}
		if err := os.WriteFile(filepath.Join(rawDir, "chunks", pkgID), pkg, 0o644); err != nil {
			return err
		}
		if bulkID != "" {
			if err := os.WriteFile(filepath.Join(rawDir, "chunks", bulkID), bulk, 0o644); err != nil {
				return err
			}
		}
		_ = os.Remove(filepath.Join(rawDir, pkgID+".orig"))
		_ = os.Remove(filepath.Join(rawDir, bulkID+".orig"))
	}
	_ = os.Remove(utoc)
	_ = os.Remove(strings.TrimSuffix(utoc, ".utoc") + ".ucas")
	return b.run("pack-raw", rawDir, utoc)
}
