package main

import (
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
)

// The effect finder maps unknown effects to game assets by colour:
// every candidate gets one of 8 test colours, the user does the move and
// says which colour it turned, and the candidates narrow to that colour.
// 64 candidates take two rounds, 512 take three.
//
//   live: Niagara colour curves, written into memory - instant
//   pak:  material instances, built into a test mod - needs a Tekken restart

type finderColor struct{ Hex, Name string }

var finderPalette = []finderColor{
	{"#ff1a1a", "Red"}, {"#1aff3c", "Green"}, {"#1a5cff", "Blue"}, {"#ffe81a", "Yellow"},
	{"#1af0ff", "Cyan"}, {"#ff1aff", "Magenta"}, {"#ff8a00", "Orange"}, {"#ffffff", "White"},
}

type finderItem struct {
	Name   string   `json:"name"`
	Detail string   `json:"detail"`
	Paths  []string `json:"paths,omitempty"` // live: curve paths (GPU copies share a colour)
	Asset  string   `json:"asset,omitempty"` // pak: material instance name
	// pak: colour params as the game has them in memory (verifies patches)
	Params map[string][4]float32 `json:"params,omitempty"`
}

type finderSaved struct {
	Kind    string       `json:"kind"`
	Scope   string       `json:"scope"`
	Items   []finderItem `json:"items"`
	History [][]int      `json:"history"` // candidate index lists, last = current
}

type FinderEntry struct {
	Color     string `json:"color"`
	ColorName string `json:"colorName"`
	Name      string `json:"name"`
	Detail    string `json:"detail"`
}

type FinderView struct {
	Active  bool          `json:"active"`
	Kind    string        `json:"kind"`
	Scope   string        `json:"scope"`
	Round   int           `json:"round"`
	Total   int           `json:"total"`
	Entries []FinderEntry `json:"entries"`
	Colors  []FinderEntry `json:"colors"` // colours in use, Name = how many candidates
	Done    bool          `json:"done"`   // every candidate has its own colour
	Pak     PakStatus     `json:"pak"`
	Legend  string        `json:"legend"`
}

type Finder struct {
	engine *Engine
	paks   *PakMods
	mu     sync.Mutex
	st     *finderSaved
}

func finderPath() string { return filepath.Join(store.Dir(), "finder.json") }
func legendPath() string { return filepath.Join(store.Dir(), "finder.txt") }

func NewFinder(e *Engine, p *PakMods) *Finder {
	f := &Finder{engine: e, paks: p}
	if b, err := os.ReadFile(finderPath()); err == nil {
		var st finderSaved
		if json.Unmarshal(b, &st) == nil && len(st.History) > 0 {
			f.st = &st
			for _, it := range st.Items {
				if st.Kind == "pak" && it.Params == nil {
					f.st = nil // from a build that patched unverified params: drop it (its mod is removed)
					_ = os.Remove(finderPath())
					break
				}
			}
		}
	}
	return f
}

// Resume re-applies a finder session left running when the app closed.
func (f *Finder) Resume() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.st != nil {
		f.apply()
	}
}

func (f *Finder) current() []int { return f.st.History[len(f.st.History)-1] }

func colorFor(i int) finderColor { return finderPalette[i%len(finderPalette)] }

// apply pushes the current round's colours to the game.
func (f *Finder) apply() {
	cur := f.current()
	switch f.st.Kind {
	case "live":
		m := map[string]string{}
		for j, idx := range cur {
			for _, p := range f.st.Items[idx].Paths {
				m[p] = colorFor(j).Hex
			}
		}
		_ = f.engine.do(func() { f.engine.finderLive = m })
		f.paks.SetFinder(nil)
	case "pak":
		var edits []pak.Edit
		for j, idx := range cur {
			c, _ := fx.ParseHex(colorFor(j).Hex)
			it := f.st.Items[idx]
			edits = append(edits, pak.Edit{Asset: it.Asset, Color: [3]float64{c.R(), c.G(), c.B()}, AnyColor: true, Expect: it.Params})
		}
		sort.Slice(edits, func(a, b int) bool { return edits[a].Asset < edits[b].Asset })
		f.paks.SetFinder(edits)
		_ = f.engine.do(func() { f.engine.finderLive = nil })
	}
	f.save()
	f.writeLegend()
}

func (f *Finder) save() {
	_ = os.MkdirAll(store.Dir(), 0o755)
	if f.st == nil {
		_ = os.Remove(finderPath())
		return
	}
	b, _ := json.Marshal(f.st)
	_ = os.WriteFile(finderPath(), b, 0o644)
}

func (f *Finder) writeLegend() {
	var b strings.Builder
	fmt.Fprintf(&b, "Electric Editor effect finder - %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "%s, %q, round %d, %d of %d candidates left\n\n", f.st.Kind, f.st.Scope, len(f.st.History), len(f.current()), len(f.st.Items))
	for j, idx := range f.current() {
		it := f.st.Items[idx]
		fmt.Fprintf(&b, "%-8s %s\n         %s\n", colorFor(j).Name, it.Name, it.Detail)
	}
	_ = os.WriteFile(legendPath(), []byte(b.String()), 0o644)
}

func (f *Finder) View() FinderView {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.st == nil {
		return FinderView{}
	}
	cur := f.current()
	v := FinderView{Active: true, Kind: f.st.Kind, Scope: f.st.Scope, Round: len(f.st.History),
		Total: len(f.st.Items), Done: len(cur) <= len(finderPalette), Legend: legendPath()}
	counts := map[int]int{}
	for j, idx := range cur {
		c := colorFor(j)
		counts[j%len(finderPalette)]++
		it := f.st.Items[idx]
		v.Entries = append(v.Entries, FinderEntry{Color: c.Hex, ColorName: c.Name, Name: it.Name, Detail: it.Detail})
	}
	for i, c := range finderPalette {
		if counts[i] > 0 {
			v.Colors = append(v.Colors, FinderEntry{Color: c.Hex, ColorName: c.Name, Name: fmt.Sprint(counts[i])})
		}
	}
	if f.st.Kind == "pak" {
		v.Pak = f.paks.FinderStatus()
	}
	return v
}

// colourParam keeps vector parameters that look like colours: named like
// one, and not all zero (zero values aren't stored in the file, so there'd
// be nothing to patch).
func colourParam(name string, v [4]float32) bool {
	l := strings.ToLower(name)
	named := false
	for _, w := range []string{"color", "colour", "col", "tint", "emissive", "glow"} {
		if strings.Contains(l, w) {
			named = true
		}
	}
	return named && (v[0] != 0 || v[1] != 0 || v[2] != 0)
}

// shortCurve turns a curve path into "NS_X · Emitter · ColorCurve_0".
func shortCurve(path string) (string, string) {
	sys, rest, _ := strings.Cut(path, ":")
	sys = sys[strings.LastIndex(sys, "/")+1:]
	sys, _, _ = strings.Cut(sys, ".")
	segs := strings.Split(rest, ".")
	emitter := segs[0]
	curve := strings.TrimPrefix(segs[len(segs)-1], "NiagaraDataInterface")
	return sys + " · " + emitter + " · " + curve, path
}

// Start scans the game for candidates in scope (a path fragment such as
// "/character/swl/") and colours round 1.
func (f *Finder) Start(kind, scope string) (FinderView, error) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return FinderView{}, errors.New("pick a fighter or type part of a path")
	}
	var items []finderItem
	var err error
	_ = f.engine.do(func() {
		s := f.engine.sess
		if s == nil {
			err = errors.New("connect to Tekken and load a match with that fighter first")
			return
		}
		s.Rescan()
		switch kind {
		case "live":
			groups := map[string][]string{}
			for p := range s.CurveList() {
				if s.IsFloat(p) || !strings.Contains(strings.ToLower(p), scope) {
					continue
				}
				g := strings.NewReplacer(".GPUComputeScript.", ".~.", ".UpdateScript.", ".~.").Replace(p)
				groups[g] = append(groups[g], p)
			}
			for g, paths := range groups {
				sort.Strings(paths)
				name, _ := shortCurve(paths[0])
				items = append(items, finderItem{Name: name, Detail: g, Paths: paths})
			}
		case "pak":
			seen := map[string]bool{}
			for _, m := range s.Materials([]string{scope}) {
				if len(m.Params) == 0 || !strings.Contains(strings.ToLower(m.Path), scope) {
					continue
				}
				name := m.Path[strings.LastIndex(m.Path, "/")+1:]
				name, _, _ = strings.Cut(name, ".")
				if name == "" || seen[strings.ToLower(name)] {
					continue
				}
				seen[strings.ToLower(name)] = true
				var ps []string
				params := map[string][4]float32{}
				for _, p := range m.Params {
					if colourParam(p.Name, p.Value) {
						ps = append(ps, p.Name)
						params[strings.ToLower(p.Name)] = p.Value
					}
				}
				if len(params) == 0 {
					continue
				}
				items = append(items, finderItem{Name: name, Detail: m.Path + "  [" + strings.Join(ps, ", ") + "]", Asset: name, Params: params})
			}
		default:
			err = errors.New("unknown finder mode")
		}
	})
	if err != nil {
		return FinderView{}, err
	}
	if len(items) == 0 {
		if kind == "live" {
			return FinderView{}, errors.New("no colour curves loaded for that fighter - load a match and do the move once first")
		}
		return FinderView{}, errors.New("no materials with colours loaded for that fighter - load a match and do the move once first")
	}
	sort.Slice(items, func(a, b int) bool { return items[a].Detail < items[b].Detail })
	all := make([]int, len(items))
	for i := range all {
		all[i] = i
	}
	f.mu.Lock()
	f.st = &finderSaved{Kind: kind, Scope: scope, Items: items, History: [][]int{all}}
	f.apply()
	f.mu.Unlock()
	return f.View(), nil
}

// Pick narrows the candidates to those showing the colour the user saw.
func (f *Finder) Pick(hex string) (FinderView, error) {
	f.mu.Lock()
	if f.st == nil {
		f.mu.Unlock()
		return FinderView{}, errors.New("start the finder first")
	}
	var next []int
	for j, idx := range f.current() {
		if strings.EqualFold(colorFor(j).Hex, hex) {
			next = append(next, idx)
		}
	}
	if len(next) == 0 {
		f.mu.Unlock()
		return FinderView{}, errors.New("nothing has that colour this round")
	}
	f.st.History = append(f.st.History, next)
	f.apply()
	f.mu.Unlock()
	return f.View(), nil
}

// Back undoes the last pick.
func (f *Finder) Back() FinderView {
	f.mu.Lock()
	if f.st != nil && len(f.st.History) > 1 {
		f.st.History = f.st.History[:len(f.st.History)-1]
		f.apply()
	}
	f.mu.Unlock()
	return f.View()
}

// Stop puts every test colour back.
func (f *Finder) Stop() FinderView {
	f.mu.Lock()
	f.st = nil
	f.save()
	f.mu.Unlock()
	_ = f.engine.do(func() { f.engine.finderLive = map[string]string{} })
	f.paks.SetFinder(nil)
	return FinderView{}
}

// ------------------------------------------------------------ engine side

// applyFinder writes the finder's live test colours and restores curves it
// no longer colours. Runs on the engine goroutine every tick.
func (e *Engine) applyFinder() {
	if e.sess == nil || (len(e.finderLive) == 0 && len(e.finderTrk) == 0) {
		return
	}
	if e.finderTrk == nil {
		e.finderTrk = map[string]*tracked{}
	}
	for path, tr := range e.finderTrk {
		if _, on := e.finderLive[path]; on {
			continue
		}
		if e.sess.Loaded(path) && e.sess.Addr(path) == tr.Addr && tr.Written {
			_ = e.sess.Write(path, tr.Orig)
		}
		delete(e.finderTrk, path)
	}
	for path, hex := range e.finderLive {
		if !e.sess.Loaded(path) {
			continue
		}
		cur, err := e.sess.Read(path)
		if err != nil {
			continue
		}
		addr := e.sess.Addr(path)
		tr := e.finderTrk[path]
		if tr == nil || tr.Addr != addr || len(tr.Orig) != len(cur) {
			orig := cur
			if t := e.tracked[path]; t != nil && t.Addr == addr && len(t.Orig) == len(cur) {
				orig = t.Orig // our own profile colour is on it: remember the game's
			}
			tr = &tracked{Addr: addr, Orig: orig}
			e.finderTrk[path] = tr
		}
		want := fx.Render(tr.Orig, fx.Setting{Mode: "solid", Color: hex, Brightness: 6, KeepFade: true})
		if !fx.Same(cur, want) && e.sess.Write(path, want) == nil {
			tr.Written = true
		}
	}
	if len(e.finderLive) == 0 {
		e.finderLive = nil
	}
}

// partsWithout drops curves the finder is colouring, so the profile sync
// doesn't fight it.
func partsWithout(parts map[string]fx.Part, skip map[string]string) map[string]fx.Part {
	// (vector-curve parts are never finder candidates, so they pass through)
	if len(skip) == 0 {
		return parts
	}
	out := make(map[string]fx.Part, len(parts))
	for k, p := range parts {
		var keep []string
		for _, path := range p.Paths {
			if _, ok := skip[path]; !ok {
				keep = append(keep, path)
			}
		}
		p.Paths = keep
		out[k] = p
	}
	return out
}
