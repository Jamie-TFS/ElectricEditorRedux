package main

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"app/internal/fx"
	"app/internal/store"
)

// App is the API the frontend calls (generated into frontend/wailsjs).
type App struct {
	ctx     context.Context
	store   *store.Store
	engine  *Engine
	keys    *Keyring
	updater *Updater
}

func NewApp() *App {
	s := store.Load()
	a := &App{store: s, engine: NewEngine(s), keys: NewKeyring(pubKey())}
	a.updater = NewUpdater(func() string { return a.Edition().Channel })
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.engine.Start(ctx)
	a.updater.CleanupOld()
	go func() {
		if keysAvailable && a.keys.LoadRevoked(revokedURL(), httpClient) {
			wruntime.EventsEmit(ctx, "edition", a.Edition())
		}
		a.updater.Run(ctx)
	}()
}

func (a *App) Edition() Edition { return editionFor(a.keys.License()) }

// ActivateKey checks and saves a key; the UI re-themes from the result.
func (a *App) ActivateKey(key string) (Edition, error) {
	_, err := a.keys.Activate(key)
	ed := a.Edition()
	wruntime.WindowSetTitle(a.ctx, ed.Title)
	return ed, err
}

func (a *App) RemoveKey() Edition {
	a.keys.Remove()
	ed := a.Edition()
	wruntime.WindowSetTitle(a.ctx, ed.Title)
	return ed
}

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

// TrailTest turns loaded trail materials green (on) or restores them (off).
func (a *App) TrailTest(on bool) (int, error) { return a.engine.TrailTest(on) }

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
