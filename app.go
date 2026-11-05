package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"app/internal/fx"
	"app/internal/pak"
	"app/internal/store"
)

// App is the API the frontend calls (generated into frontend/wailsjs).
type App struct {
	ctx     context.Context
	store   *store.Store
	engine  *Engine
	keys    *Keyring
	updater *Updater
	paks    *PakMods
	finder  *Finder
	running bool // engine started (not while a beta build is locked)
}

func NewApp() *App {
	s := store.Load()
	a := &App{store: s, engine: NewEngine(s), keys: NewKeyring(pubKey())}
	a.paks = NewPakMods(s, a.engine)
	a.finder = NewFinder(a.engine, a.paks)
	a.updater = NewUpdater(func() string { return a.Edition().Channel })
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if !a.Edition().Locked {
		a.startEngine()
	}
	a.updater.CleanupOld()
	go func() {
		if keysAvailable && a.keys.LoadRevoked(revokedURL(), httpClient) {
			wruntime.EventsEmit(ctx, "edition", a.Edition())
		}
		a.updater.Run(ctx)
	}()
}

// startEngine begins syncing colours to the game (once).
func (a *App) startEngine() {
	if a.running {
		return
	}
	a.running = true
	a.engine.Start(a.ctx)
	a.paks.Start(a.ctx)
	go a.finder.Resume()
}

func (a *App) Edition() Edition { return editionFor(a.keys.License()) }

// ActivateKey checks and saves a key; the UI re-themes from the result.
func (a *App) ActivateKey(key string) (Edition, error) {
	_, err := a.keys.Activate(key)
	ed := a.Edition()
	if err == nil && !ed.Locked {
		a.startEngine()
	}
	wruntime.WindowSetTitle(a.ctx, ed.Title)
	return ed, err
}

func (a *App) RemoveKey() Edition {
	a.keys.Remove()
	ed := a.Edition()
	wruntime.WindowSetTitle(a.ctx, ed.Title)
	return ed
}

func (a *App) PakStatus() PakStatus { return a.paks.Status() }

// OpenPakFolder shows the claw-colour working folder (log, cache).
func (a *App) OpenPakFolder() { _ = os.MkdirAll(a.paks.Folder(), 0o755); openFolder(a.paks.Folder()) }

// ExportAssets extracts matching game files (comma or line separated,
// case-sensitive path fragments) into research.zip and shows it.
func (a *App) ExportAssets(filters string) (string, error) {
	var fs []string
	for _, f := range strings.FieldsFunc(filters, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' }) {
		if f = strings.TrimSpace(f); f != "" {
			fs = append(fs, f)
		}
	}
	paks := a.paks.paksDir()
	if paks == "" {
		return "", errors.New("start Tekken once so the game folder can be found")
	}
	zipPath := filepath.Join(store.Dir(), "research.zip")
	n, err := a.paks.claws.builder.Export(paks, fs, zipPath)
	if err != nil {
		return "", err
	}
	openFolder(store.Dir())
	return fmt.Sprintf("Saved %d files to research.zip", n), nil
}

// BuildTestMod builds a research mod from lines like
//
//	MI_EF_Ant_trail01_Edge MaskTexTilingU=12
//	MI_EF_Ant_trail03_blade BaseTexColor=0,1,0,1
//
// Several params may share a line. Empty text removes the test mod.
func (a *App) BuildTestMod(text string) (PakStatus, error) {
	byAsset := map[string]map[string][]float32{}
	raw := map[string]map[int][]byte{}
	finds := map[string][]pak.Find{}
	tints := map[string]*[3]float64{}
	rawZen := map[string]bool{}
	var order []string
	for i, line := range strings.Split(text, "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		if len(f) < 2 {
			return PakStatus{}, fmt.Errorf("line %d: expected an asset name then Param=value", i+1)
		}
		asset := f[0]
		if byAsset[asset] == nil {
			byAsset[asset] = map[string][]float32{}
			raw[asset] = map[int][]byte{}
			order = append(order, asset)
		}
		for _, kv := range f[1:] {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				return PakStatus{}, fmt.Errorf("line %d: %q should look like Param=1, Param=1,0,0,1 or @offset=hexbytes", i+1, kv)
			}
			if strings.HasPrefix(k, "~") {
				// ~<hex pattern>+<offset>=<hex bytes>
				pat, offs, _ := strings.Cut(k[1:], "+")
				pb, err1 := hex.DecodeString(pat)
				var off int
				_, err2 := fmt.Sscan(offs, &off)
				data, err3 := hex.DecodeString(v)
				if err1 != nil || err2 != nil || err3 != nil || len(pb) < 8 || len(data) == 0 {
					return PakStatus{}, fmt.Errorf("line %d: use ~<hex pattern>+<offset>=<hex bytes>", i+1)
				}
				finds[asset] = append(finds[asset], pak.Find{Pattern: pb, Off: off, Data: data})
				continue
			}
			if strings.HasPrefix(k, "@") {
				var off int
				if _, err := fmt.Sscan(k[1:], &off); err != nil {
					return PakStatus{}, fmt.Errorf("line %d: bad offset %q", i+1, k)
				}
				data, err := hex.DecodeString(strings.ReplaceAll(v, " ", ""))
				if err != nil || len(data) == 0 {
					return PakStatus{}, fmt.Errorf("line %d: %q isn't hex bytes", i+1, v)
				}
				raw[asset][off] = data
				continue
			}
			var vals []float32
			for _, part := range strings.Split(v, ",") {
				var x float64
				if _, err := fmt.Sscan(strings.TrimSpace(part), &x); err != nil {
					return PakStatus{}, fmt.Errorf("line %d: %q isn't a number", i+1, part)
				}
				vals = append(vals, float32(x))
			}
			if strings.EqualFold(k, "TINT") || strings.EqualFold(k, "ZTINT") {
				// ZTINT: patch the game's packaged bytes directly (no conversion)
				rawZen[asset] = strings.EqualFold(k, "ZTINT")
				if len(vals) != 3 {
					return PakStatus{}, fmt.Errorf("line %d: TINT needs r,g,b", i+1)
				}
				tints[asset] = &[3]float64{float64(vals[0]), float64(vals[1]), float64(vals[2])}
				continue
			}
			if len(vals) != 1 && len(vals) != 4 {
				return PakStatus{}, fmt.Errorf("line %d: give 1 value (scalar) or 4 (colour) for %s", i+1, k)
			}
			byAsset[asset][k] = vals
		}
	}
	var edits []pak.Edit
	for _, asset := range order {
		edits = append(edits, pak.Edit{Asset: asset, Set: byAsset[asset], Raw: raw[asset], Finds: finds[asset], Tint: tints[asset], RawZen: rawZen[asset]})
	}
	a.paks.SetTest(edits)
	return a.paks.TestStatus(), nil
}

func (a *App) TestModStatus() PakStatus { return a.paks.TestStatus() }

// Effect finder (dev tools).
func (a *App) FinderView() FinderView                             { return a.finder.View() }
func (a *App) FinderStart(kind, scope string) (FinderView, error) { return a.finder.Start(kind, scope) }
func (a *App) FinderPick(hex string) (FinderView, error)          { return a.finder.Pick(hex) }
func (a *App) FinderBack() FinderView                             { return a.finder.Back() }
func (a *App) FinderStop() FinderView                             { return a.finder.Stop() }
func (a *App) OpenFinderLegend()                                  { openFolder(store.Dir()) }

func (a *App) UpdateStatus() UpdateStatus { return a.updater.Status() }
func (a *App) RestartToUpdate() error     { return a.updater.Restart(a.ctx) }

// ------------------------------------------------------------ background image

var bgTypes = map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".webp": "image/webp", ".gif": "image/gif"}

func bgPath() (string, string) {
	for ext := range bgTypes {
		p := filepath.Join(store.Dir(), "background"+ext)
		if _, err := os.Stat(p); err == nil {
			return p, ext
		}
	}
	return "", ""
}

// Background returns the saved background as a data URL ("" if none).
func (a *App) Background() string {
	p, ext := bgPath()
	if p == "" {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return "data:" + bgTypes[ext] + ";base64," + base64.StdEncoding.EncodeToString(b)
}

// ChooseBackground lets the user pick an image; it's copied into the app's folder.
func (a *App) ChooseBackground() (string, error) {
	file, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:   "Choose a background image",
		Filters: []wruntime.FileFilter{{DisplayName: "Images", Pattern: "*.png;*.jpg;*.jpeg;*.webp;*.gif"}},
	})
	if err != nil || file == "" {
		return a.Background(), err
	}
	ext := strings.ToLower(filepath.Ext(file))
	if _, ok := bgTypes[ext]; !ok {
		return a.Background(), errors.New("pick a PNG, JPG, WebP or GIF image")
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return a.Background(), err
	}
	if len(b) > 25<<20 {
		return a.Background(), errors.New("that image is over 25 MB - pick a smaller one")
	}
	a.ClearBackground()
	_ = os.MkdirAll(store.Dir(), 0o755)
	if err := os.WriteFile(filepath.Join(store.Dir(), "background"+ext), b, 0o644); err != nil {
		return "", err
	}
	return a.Background(), nil
}

func (a *App) ClearBackground() {
	for ext := range bgTypes {
		_ = os.Remove(filepath.Join(store.Dir(), "background"+ext))
	}
}

type State struct {
	Active   string        `json:"active"`
	Profiles []string      `json:"profiles"`
	Parts    store.Profile `json:"parts"`
}

func (a *App) state() State {
	return State{Active: a.store.ActiveName(), Profiles: a.store.ProfileNames(), Parts: a.store.Active()}
}

func (a *App) Catalog() []fx.Character    { return fx.Catalog }
func (a *App) DefaultSetting() fx.Setting { return fx.DefaultSetting() }
func (a *App) State() State               { return a.state() }
func (a *App) Status() Status             { return a.engine.Status() }

// DumpCurves saves every loaded colour curve to curves.txt and opens its folder.
func (a *App) DumpCurves() (string, error) {
	path, err := a.engine.DumpCurves()
	if err == nil {
		openFolder(filepath.Dir(path))
	}
	return path, err
}

func (a *App) SetPart(key string, s fx.Setting) error {
	if err := a.store.Set(key, s); err != nil {
		return err
	}
	a.engine.Poke()
	return nil
}

// SetMany applies one setting to several parts (e.g. "copy to all parts").
func (a *App) SetMany(keys []string, s fx.Setting) error {
	for _, k := range keys {
		if err := a.store.Set(k, s); err != nil {
			return err
		}
	}
	a.engine.Poke()
	return nil
}

func (a *App) SwitchProfile(name string) (State, error) {
	err := a.store.Switch(name)
	a.engine.Poke()
	return a.state(), err
}

func (a *App) CreateProfile(name string) (State, error) {
	err := a.store.Create(name)
	return a.state(), err
}

func (a *App) RenameProfile(name string) (State, error) {
	err := a.store.Rename(name)
	return a.state(), err
}

func (a *App) DeleteProfile() (State, error) {
	err := a.store.Delete()
	a.engine.Poke()
	return a.state(), err
}

// ExportCode shares the whole profile (charID "") or one character.
func (a *App) ExportCode(charID string) (string, error) {
	prefix := ""
	if charID != "" {
		prefix = charID + "."
	}
	return a.store.Export(prefix)
}

type ImportResult struct {
	State State `json:"state"`
	Count int   `json:"count"`
}

func (a *App) ImportCode(code string) (ImportResult, error) {
	n, err := a.store.Import(strings.TrimSpace(code), fx.AllParts())
	a.engine.Poke()
	return ImportResult{State: a.state(), Count: n}, err
}

func openFolder(dir string) {
	if runtime.GOOS == "windows" {
		_ = exec.Command("explorer", dir).Start()
	}
}
