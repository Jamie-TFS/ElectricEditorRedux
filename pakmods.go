package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"app/internal/fx"
	"app/internal/pak"
	"app/internal/store"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PakStatus drives the claw-trail banner in the UI.
type PakStatus struct {
	State   string `json:"state"` // off | building | pending | ready | error | nogame
	Message string `json:"message"`
}

type pakState struct {
	Paks      string            `json:"paks"`      // last known Tekken Paks folder
	Installed map[string]string `json:"installed"` // mod name -> spec of its files in Paks\mods
}

// slot is one generated mod: the claw colours, or the effect finder's test
// colours. Each builds in its own folder and installs under its own name.
type slot struct {
	builder  *pak.Builder
	built    string // spec currently in builder.Dir\out
	builtOut []string
	failed   string // spec that failed to build (not retried until it changes)
}

// PakMods keeps Tekken's mods folder in step with the material parts of the
// active profile (and the effect finder). It polls instead of hooking every
// profile change, so switching profiles, importing codes and edits all work.
type PakMods struct {
	ctx    context.Context
	store  *store.Store
	engine *Engine
	assets map[string]fx.Part // part key -> material part

	claws  *slot
	finder *slot
	test   *slot

	mu          sync.Mutex
	status      PakStatus
	finderEdits []pak.Edit
	finderStat  PakStatus
	testEdits   []pak.Edit
	testStat    PakStatus

	st        pakState
	seen      string
	seenSince time.Time
}

func NewPakMods(s *store.Store, e *Engine) *PakMods {
	dir := filepath.Join(store.Dir(), "pak")
	m := &PakMods{
		store:  s,
		engine: e,
		assets: map[string]fx.Part{},
		status: PakStatus{State: "off"},
		claws:  &slot{builder: &pak.Builder{Dir: dir, Name: pak.ModName, Always: fx.PakAssets()}},
		finder: &slot{builder: &pak.Builder{Dir: filepath.Join(store.Dir(), "pak-finder"), Name: pak.FinderName}},
		test:   &slot{builder: &pak.Builder{Dir: filepath.Join(store.Dir(), "pak-test"), Name: pak.TestName}},
	}
	for _, c := range fx.Catalog {
		for _, ef := range c.Effects {
			for _, p := range ef.Parts {
				if p.Asset != "" || len(p.VMAssets) > 0 {
					m.assets[fx.Key(c.ID, ef.ID, p.ID)] = p
				}
			}
		}
	}
	m.load()
	return m
}

func (m *PakMods) statePath() string { return filepath.Join(m.claws.builder.Dir, "state.json") }

func (m *PakMods) load() {
	if b, err := os.ReadFile(m.statePath()); err == nil {
		_ = json.Unmarshal(b, &m.st)
	}
	if m.st.Installed == nil {
		m.st.Installed = map[string]string{}
	}
}

func (m *PakMods) save() {
	_ = os.MkdirAll(m.claws.builder.Dir, 0o755)
	b, _ := json.Marshal(m.st)
	_ = os.WriteFile(m.statePath(), b, 0o644)
}

func (m *PakMods) Status() PakStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *PakMods) set(state, msg string) {
	m.mu.Lock()
	changed := m.status.State != state || m.status.Message != msg
	m.status = PakStatus{State: state, Message: msg}
	m.mu.Unlock()
	if changed && m.ctx != nil {
		runtime.EventsEmit(m.ctx, "pak", m.status)
	}
}

// SetFinder replaces the effect finder's test colours (nil = remove its mod).
func (m *PakMods) SetFinder(edits []pak.Edit) {
	m.mu.Lock()
	m.finderEdits = edits
	if len(edits) > 0 {
		m.finderStat = PakStatus{State: "building", Message: "Building test colours…"}
	}
	m.mu.Unlock()
}

// SetTest replaces the research test mod (nil = remove it).
func (m *PakMods) SetTest(edits []pak.Edit) {
	m.mu.Lock()
	m.testEdits = edits
	m.test.failed = ""
	if len(edits) > 0 {
		m.testStat = PakStatus{State: "building", Message: "Building test mod…"}
	} else {
		m.testStat = PakStatus{State: "building", Message: "Removing test mod…"}
	}
	m.mu.Unlock()
}

func (m *PakMods) TestStatus() PakStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.testStat
}

func (m *PakMods) setTest(state, msg string) {
	m.mu.Lock()
	m.testStat = PakStatus{State: state, Message: msg}
	m.mu.Unlock()
}

func (m *PakMods) FinderStatus() PakStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.finderStat
}

func (m *PakMods) setFinder(state, msg string) {
	m.mu.Lock()
	m.finderStat = PakStatus{State: state, Message: msg}
	m.mu.Unlock()
}

// edits turns the profile into the list of materials to recolour.
func (m *PakMods) edits(profile store.Profile) []pak.Edit {
	var out []pak.Edit
	recoloured := map[string]bool{}   // game assets changed in place
	scripts := map[string]*pak.Edit{} // Niagara systems with script patches
	for key, part := range m.assets {
		s, ok := profile[key]
		if !ok || s.Mode == "native" {
			continue
		}
		c, ok := fx.ParseHex(s.Color)
		if !ok {
			continue
		}
		br := s.Brightness
		if br <= 0 || br > 10 {
			br = 1
		}
		col := [3]float64{c.R() * br, c.G() * br, c.B() * br}
		if len(part.VM) > 0 {
			var keys []string
			for _, k := range part.VM {
				if k == "*" { // every generated patch of the part's systems
					for _, a := range part.VMAssets {
						keys = append(keys, pak.SystemPatches[a]...)
					}
					continue
				}
				keys = append(keys, k)
			}
			for _, k := range keys {
				name, _ := pak.VMKey(k)
				asset := pak.VMPatches[name].Asset
				se := scripts[asset]
				if se == nil {
					se = &pak.Edit{Asset: asset, VM: map[string][3]float64{}}
					scripts[asset] = se
				}
				se.VM[k] = col
			}
		}
		if part.Asset == "" {
			continue
		}
		e := pak.Edit{Asset: part.Asset, Params: part.Params, Color: col}
		if part.Texture {
			tint := e.Color
			e = pak.Edit{Asset: part.Asset, Tint: &tint, RawZen: true}
		}
		if part.Copy != nil {
			e.From, e.Rename = part.Copy.From, part.Copy.Rename
			out = append(out, copyUsers(part.Copy)...)
		} else {
			recoloured[strings.ToLower(part.Asset)] = true
		}
		out = append(out, e)
	}
	// A private copy must also exist when only the shared original is being
	// recoloured, or the other fighter would pick up that colour.
	for key, part := range m.assets {
		if part.Copy == nil || !recoloured[strings.ToLower(part.Copy.From)] {
			continue
		}
		if s, ok := profile[key]; ok && s.Mode != "native" {
			continue // already added with its own colour
		}
		out = append(out, pak.Edit{Asset: part.Asset, From: part.Copy.From, Rename: part.Copy.Rename, Keep: true})
		out = append(out, copyUsers(part.Copy)...)
	}
	for _, se := range scripts {
		out = append(out, *se)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Asset < out[j].Asset })
	return out
}

func copyUsers(c *fx.Copy) []pak.Edit {
	var out []pak.Edit
	for _, u := range c.Users {
		out = append(out, pak.Edit{Asset: u, Rename: c.Rename, Keep: true})
	}
	return out
}

func spec(edits []pak.Edit) string {
	if len(edits) == 0 {
		return ""
	}
	var b strings.Builder
	for _, e := range edits {
		fmt.Fprintf(&b, "%s=%.4f,%.4f,%.4f", strings.ToLower(e.Asset), e.Color[0], e.Color[1], e.Color[2])
		if e.Keep {
			b.WriteString(" keep")
		}
		if e.From != "" {
			fmt.Fprintf(&b, " from %s", e.From)
		}
		b.WriteString(";")
		if len(e.Set) > 0 {
			keys := make([]string, 0, len(e.Set))
			for k := range e.Set {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(&b, "%s:%v;", k, e.Set[k])
			}
		}
		for _, f := range e.Finds {
			fmt.Fprintf(&b, "~%x+%d:%x;", f.Pattern, f.Off, f.Data)
		}
		if e.Tint != nil {
			fmt.Fprintf(&b, "tint%v raw=%v;", *e.Tint, e.RawZen)
		}
		vk := make([]string, 0, len(e.VM))
		for k := range e.VM {
			vk = append(vk, k)
		}
		sort.Strings(vk)
		for _, k := range vk {
			fmt.Fprintf(&b, "vm %s%v;", k, e.VM[k])
		}
		offs := make([]int, 0, len(e.Raw))
		for off := range e.Raw {
			offs = append(offs, off)
		}
		sort.Ints(offs)
		for _, off := range offs {
			fmt.Fprintf(&b, "@%d:%x;", off, e.Raw[off])
		}
	}
	return b.String()
}

// paksDir finds Tekken's Paks folder: from the running game, else the last
// one seen, else the default Steam location.
func (m *PakMods) paksDir() string {
	if p := pak.PaksFromExe(m.engine.ExePath()); p != "" {
		if _, err := os.Stat(p); err == nil {
			if m.st.Paks != p {
				m.st.Paks = p
				m.save()
			}
			return p
		}
	}
	if m.st.Paks != "" {
		if _, err := os.Stat(m.st.Paks); err == nil {
			return m.st.Paks
		}
	}
	for _, p := range []string{
		`C:\Program Files (x86)\Steam\steamapps\common\TEKKEN 8\Polaris\Content\Paks`,
		`C:\Program Files\Steam\steamapps\common\TEKKEN 8\Polaris\Content\Paks`,
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (m *PakMods) Start(ctx context.Context) {
	m.ctx = ctx
	go func() {
		t := time.NewTicker(1500 * time.Millisecond)
		defer t.Stop()
		for {
			m.step()
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

// sync brings one slot's installed files to want. Result state is one of
// off | ready | building-failed(error) | pending | error.
func (m *PakMods) sync(s *slot, paks string, edits []pak.Edit) (state string, err error) {
	name := s.builder.Name
	want := spec(edits)
	if m.st.Installed[name] != "" && !pak.Installed(paks, name) {
		delete(m.st.Installed, name) // removed by hand
		m.save()
	}
	if want == m.st.Installed[name] {
		if want == "" {
			return "off", nil
		}
		return "ready", nil
	}
	if want == "" {
		if err := pak.Uninstall(paks, name); err != nil {
			return "pending", err
		}
		delete(m.st.Installed, name)
		m.save()
		return "off", nil
	}
	if want != s.built {
		if want == s.failed {
			return "error", errors.New("build failed, change something to retry")
		}
		files, skipped, err := s.builder.Build(paks, edits)
		if err != nil {
			s.failed = want
			return "error", err
		}
		if len(skipped) > 0 {
			builderLog(s.builder.Dir, "skipped: %s", strings.Join(skipped, ", "))
		}
		s.built, s.builtOut, s.failed = want, files, ""
	}
	if err := pak.Install(paks, name, s.builtOut); err != nil {
		if errors.Is(err, pak.ErrLocked) {
			return "pending", err
		}
		return "error", err
	}
	m.st.Installed[name] = want
	m.save()
	return "installed", nil
}

func (m *PakMods) step() {
	defer func() {
		if r := recover(); r != nil {
			m.set("error", fmt.Sprint("Mod colours hit a problem: ", r))
		}
	}()
	edits := m.edits(m.store.Active())
	want := spec(edits)
	m.mu.Lock()
	fEdits := m.finderEdits
	m.mu.Unlock()

	paks := m.paksDir()
	if paks == "" {
		if want == "" {
			m.set("off", "")
		} else {
			m.set("nogame", "Start Tekken once so Electric Editor can find the game folder for mod colours.")
		}
		if len(fEdits) > 0 {
			m.setFinder("nogame", "Start Tekken once so the finder can find the game folder.")
		}
		return
	}
	running := m.engine.Running()

	// effect finder test mod (no debounce: it only changes on a button press)
	if fEdits != nil || m.st.Installed[pak.FinderName] != "" {
		if len(fEdits) > 0 && spec(fEdits) != m.finder.built && spec(fEdits) != m.finder.failed {
			m.setFinder("building", fmt.Sprintf("Building test colours for %d materials… (first time can take a minute)", len(fEdits)))
		}
		st, err := m.sync(m.finder, paks, fEdits)
		switch st {
		case "installed", "ready":
			if running && st == "installed" {
				m.setFinder("ready", "Test colours installed. Restart Tekken, do the move, then pick the colour you see.")
			} else if st == "installed" {
				m.setFinder("ready", "Test colours installed. Start Tekken, do the move, then pick the colour you see.")
			} else if m.FinderStatus().State != "ready" {
				m.setFinder("ready", "Test colours are installed. Do the move in Tekken, then pick the colour you see.")
			}
		case "pending":
			if len(fEdits) > 0 {
				m.setFinder("pending", "Close Tekken to install the new test colours, then start it again.")
			} else {
				m.setFinder("pending", "Test colours are removed when you close Tekken.")
			}
		case "error":
			m.setFinder("error", "Couldn't build test colours: "+err.Error())
		case "off":
			m.setFinder("off", "")
			m.mu.Lock()
			if len(m.finderEdits) == 0 {
				m.finderEdits = nil
			}
			m.mu.Unlock()
		}
	}

	// research test mod
	m.mu.Lock()
	tEdits := m.testEdits
	m.mu.Unlock()
	if tEdits != nil || m.st.Installed[pak.TestName] != "" {
		st, err := m.sync(m.test, paks, tEdits)
		switch st {
		case "installed", "ready":
			if len(tEdits) > 0 {
				m.setTest("ready", "Test mod installed. Restart Tekken to try it.")
			}
		case "pending":
			m.setTest("pending", "Close Tekken to install or remove the test mod.")
		case "error":
			m.setTest("error", err.Error())
		case "off":
			m.setTest("off", "No test mod installed.")
		}
	}

	// claw colours: wait until the colour has stopped changing (dragging)
	if want != m.seen {
		m.seen, m.seenSince = want, time.Now()
	}
	if time.Since(m.seenSince) < 1200*time.Millisecond {
		return
	}
	if want != m.st.Installed[pak.ModName] && want != "" && want != m.claws.built && want != m.claws.failed {
		m.set("building", "Building mod colours…")
	}
	st, err := m.sync(m.claws, paks, edits)
	switch st {
	case "off":
		m.set("off", "")
	case "ready":
		if running {
			m.set("ready", "Mod colours are installed. Restart Tekken if you changed them this session.")
		} else {
			m.set("ready", "Mod colours are installed and show next time you start Tekken.")
		}
	case "installed":
		if running {
			m.set("ready", "Mod colours installed. Restart Tekken to see them.")
		} else {
			m.set("ready", "Mod colours installed. They show next time you start Tekken.")
		}
	case "pending":
		switch {
		case !running && err != nil:
			// the game is closed, so something else is holding the files
			m.set("error", fmt.Sprintf("Couldn't replace the mod files in %s - %v. Make sure Tekken has fully closed (check Task Manager for Polaris), or copy the files from %s into that folder by hand.",
				pak.ModsDir(paks), err, filepath.Join(m.claws.builder.Dir, "out")))
		case want == "":
			m.set("pending", "Mod effects go back to game colours when you close Tekken.")
		default:
			m.set("pending", "New mod colours install when you close Tekken.")
		}
	case "error":
		m.set("error", "Couldn't build mod colours: "+err.Error())
	}
}

func builderLog(dir, format string, args ...any) {
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(filepath.Join(dir, "log.txt"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		fmt.Fprintf(f, "%s  %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
		f.Close()
	}
}

// Folder opens the claw-colour working folder (log, cache).
func (m *PakMods) Folder() string { return m.claws.builder.Dir }

// FinderFolder is the effect finder's working folder.
func (m *PakMods) FinderFolder() string { return m.finder.builder.Dir }
