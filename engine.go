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
	"app/internal/game"
	"app/internal/store"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Status is what the UI shows about the game connection.
type Status struct {
	State   string          `json:"state"` // waiting | connecting | connected | error
	Message string          `json:"message"`
	Process string          `json:"process"`
	Curves  int             `json:"curves"`
	Loaded  map[string]bool `json:"loaded"` // part key -> its curves are in memory
	Applied int             `json:"applied"`
}

// Engine keeps the game in sync with the active profile.
type Engine struct {
	ctx   context.Context
	store *store.Store
	parts map[string]fx.Part
	wake  chan struct{}

	mu      sync.Mutex
	sess    *game.Session
	status  Status
	exe     string              // running game's exe path (for finding Paks)
	tracked map[string]*tracked // curves of the current Tekken session
	nextTry time.Time
	dumpReq chan chan dumpResult

	jobs chan func()

	// effect finder, live mode: curve path -> test colour, and the
	// originals of every curve it has written
	finderLive map[string]string
	finderTrk  map[string]*tracked
}

// words that pick out trail / swipe materials (own path or parent path)
var trailWords = []string{"trail", "_tl", "blade", "slash", "claw", "swing"}

// tracked remembers a curve's untouched colours for one Tekken session, so
// "Game colours" can unhook exactly. Keyed to the object's address: when the
// game reloads the effect, it comes back clean and we start over.
type tracked struct {
	Addr    uint64    `json:"addr"`
	Orig    []float32 `json:"orig"`
	Written bool      `json:"written"`
}

type sessionFile struct {
	PID    uint32              `json:"pid"`
	Curves map[string]*tracked `json:"curves"`
}

type dumpResult struct {
	path string
	err  error
}

func NewEngine(s *store.Store) *Engine {
	e := &Engine{
		store:   s,
		parts:   fx.AllParts(),
		wake:    make(chan struct{}, 1),
		tracked: map[string]*tracked{},
		dumpReq: make(chan chan dumpResult),
		jobs:    make(chan func()),
		status:  Status{State: "waiting", Message: "Start Tekken 8 and this will connect on its own.", Loaded: map[string]bool{}},
	}
	return e
}

func (e *Engine) Start(ctx context.Context) {
	e.ctx = ctx
	go func() {
		t := time.NewTicker(400 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			case <-e.wake:
			case reply := <-e.dumpReq:
				reply <- e.dump()
				continue
			case fn := <-e.jobs:
				fn()
				continue
			}
			e.tick()
		}
	}()
}

// Poke applies changes now instead of on the next tick.
func (e *Engine) Poke() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

func (e *Engine) Status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.status
}

func (e *Engine) setStatus(s Status) {
	e.mu.Lock()
	changed := !sameStatus(e.status, s)
	e.status = s
	e.mu.Unlock()
	if changed && e.ctx != nil {
		runtime.EventsEmit(e.ctx, "status", s)
	}
}

func sameStatus(a, b Status) bool {
	if a.State != b.State || a.Message != b.Message || a.Curves != b.Curves || a.Applied != b.Applied || len(a.Loaded) != len(b.Loaded) {
		return false
	}
	for k, v := range a.Loaded {
		if b.Loaded[k] != v {
			return false
		}
	}
	return true
}

func (e *Engine) tick() {
	defer func() {
		// never let a bad read take the app down; drop the session and retry
		if r := recover(); r != nil {
			e.dropSession()
			e.setStatus(Status{State: "error", Message: "Lost track of the game - reconnecting.", Loaded: map[string]bool{}})
		}
	}()

	if e.sess == nil {
		if time.Now().Before(e.nextTry) {
			return
		}
		e.nextTry = time.Now().Add(2 * time.Second)
		sess, err := game.Open()
		if err != nil {
			st := Status{State: "error", Message: err.Error(), Loaded: map[string]bool{}}
			if errors.Is(err, game.ErrNotRunning) {
				st = Status{State: "waiting", Message: "Start Tekken 8 and this will connect on its own.", Loaded: map[string]bool{}}
			}
			e.setStatus(st)
			return
		}
		e.sess = sess
		e.tracked = loadSession(sess.PID())
		e.mu.Lock()
		e.exe = sess.ExePath()
		e.mu.Unlock()
	}

	if !e.sess.Alive() {
		e.dropSession()
		e.setStatus(Status{State: "waiting", Message: "Tekken closed. Waiting for it to start again.", Loaded: map[string]bool{}})
		return
	}
	e.sess.MaybeRescan()

	loaded, applied, dirty := syncCurves(e.sess, partsWithout(e.parts, e.finderLive), e.store.Active(), e.tracked)
	if l2, a2, d2 := syncRGB(e.sess, e.parts, e.store.Active(), e.tracked); d2 || a2 > 0 || len(l2) > 0 {
		for k := range l2 {
			loaded[k] = true
		}
		applied += a2
		dirty = dirty || d2
	}
	if dirty {
		saveSession(e.sess.PID(), e.tracked)
	}
	e.applyFinder()

	msg := "Connected. Load a match to see effects here."
	if len(loaded) > 0 {
		msg = "Connected. Changes show up in game straight away."
	}
	e.setStatus(Status{State: "connected", Message: msg, Process: e.sess.Process(),
		Curves: e.sess.CurveCount(), Loaded: loaded, Applied: applied})
}

// Running reports whether Tekken is open (connected).
func (e *Engine) Running() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.status.State == "connected"
}

// ExePath is the last seen Tekken exe path ("" if never connected).
func (e *Engine) ExePath() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.exe
}

// do runs fn on the engine goroutine (where the game session lives).
func (e *Engine) do(fn func()) error {
	done := make(chan struct{})
	select {
	case e.jobs <- func() { defer close(done); fn() }:
	case <-time.After(5 * time.Second):
		return errors.New("the app is busy, try again")
	}
	<-done
	return nil
}

func (e *Engine) dropSession() {
	e.finderTrk = nil // addresses are only valid for one Tekken process
	if e.sess != nil {
		e.sess.Close()
		e.sess = nil
	}
}

// The session file lets the app restart mid-match without mistaking its own
// colours for Tekken's: it's only trusted for the same Tekken process.
func sessionPath() string { return filepath.Join(store.Dir(), "session.json") }

func loadSession(pid uint32) map[string]*tracked {
	var f sessionFile
	if b, err := os.ReadFile(sessionPath()); err == nil {
		_ = json.Unmarshal(b, &f)
	}
	if f.PID != pid || f.Curves == nil {
		return map[string]*tracked{}
	}
	return f.Curves
}

func saveSession(pid uint32, m map[string]*tracked) {
	_ = os.MkdirAll(store.Dir(), 0o755)
	b, _ := json.Marshal(sessionFile{PID: pid, Curves: m})
	_ = os.WriteFile(sessionPath(), b, 0o644)
	_ = os.Remove(filepath.Join(store.Dir(), "originals.json")) // old format, could hold modded colours
}

// DumpCurves writes every colour curve currently loaded to a text file, for
// mapping new moves. Runs on the engine goroutine.
func (e *Engine) DumpCurves() (string, error) {
	reply := make(chan dumpResult, 1)
	select {
	case e.dumpReq <- reply:
	case <-time.After(3 * time.Second):
		return "", errors.New("the app is busy, try again")
	}
	r := <-reply
	return r.path, r.err
}

func (e *Engine) dump() dumpResult {
	if e.sess == nil {
		return dumpResult{err: errors.New("connect to Tekken first, then load a match")}
	}
	e.sess.Rescan()
	list := e.sess.CurveList()
	known := map[string]string{}
	for key, part := range e.parts {
		for _, p := range part.Paths {
			known[p] = key
		}
		for _, ch := range part.RGB {
			for _, p := range ch {
				known[p] = key
			}
		}
		for _, p := range part.Vec {
			known[p] = key
		}
	}
	paths := make([]string, 0, len(list))
	for p := range list {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var b strings.Builder
	fmt.Fprintf(&b, "Electric Editor curve list - %s - %d curves\n\n", time.Now().Format(time.RFC3339), len(paths))
	for _, p := range paths {
		tag := ""
		if k, ok := known[p]; ok {
			tag = "  [" + k + "]"
		}
		if e.sess.IsFloat(p) {
			if strings.Contains(p, "VectorCurve") {
				tag += "  [vector]"
			} else {
				tag += "  [float]"
			}
			if v, err := e.sess.Read(p); err == nil && len(v) > 0 {
				lo, hi := v[0], v[0]
				for _, x := range v {
					lo, hi = min(lo, x), max(hi, x)
				}
				tag += fmt.Sprintf(" %.3g..%.3g", lo, hi)
			}
		}
		fmt.Fprintf(&b, "%s | %d%s\n", p, list[p], tag)
	}
	b.WriteString("\n--- trail, claw and slash materials (for finding claw colours) ---\n\n")
	for _, m := range e.sess.Materials(trailWords) {
		kind := ""
		if m.Dynamic {
			kind = "[runtime MID] "
		}
		fmt.Fprintf(&b, "%s%s\n    parent: %s\n", kind, m.Path, m.Parent)
		for _, p := range m.Params {
			fmt.Fprintf(&b, "    %-20s %.3f, %.3f, %.3f, %.3f\n", p.Name, p.Value[0], p.Value[1], p.Value[2], p.Value[3])
		}
	}

	out := filepath.Join(store.Dir(), "curves.txt")
	_ = os.MkdirAll(store.Dir(), 0o755)
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		return dumpResult{err: err}
	}
	return dumpResult{path: out}
}

// curveIO is the slice of game.Session the sync loop needs (fakeable in tests).
type curveIO interface {
	Loaded(path string) bool
	Read(path string) ([]float32, error)
	Write(path string, v []float32) error
	Addr(path string) uint64
}

// syncCurves makes every loaded curve match the profile.
//   - layers set to a colour are written whenever the game's value differs
//   - layers on "native" are never touched, except once to restore the
//     original if this app had changed them earlier in the session
func syncCurves(io curveIO, parts map[string]fx.Part, profile store.Profile, trk map[string]*tracked) (loaded map[string]bool, applied int, dirty bool) {
	loaded = map[string]bool{}
	for key, part := range parts {
		setting, ok := profile[key]
		if !ok {
			setting = fx.Setting{Mode: "native"}
		}
		paths, vec := part.Paths, false
		if len(part.Vec) > 0 {
			paths, vec = part.Vec, true
		}
		for _, path := range paths {
			if !io.Loaded(path) {
				continue
			}
			cur, err := io.Read(path)
			if err != nil {
				continue
			}
			loaded[key] = true
			addr := io.Addr(path)
			tr := trk[path]
			if tr == nil || tr.Addr != addr || len(tr.Orig) != len(cur) {
				// new or freshly reloaded curve: these are the game's own colours
				tr = &tracked{Addr: addr, Orig: cur}
				trk[path] = tr
				dirty = true
			}
			if setting.Mode == "native" {
				if tr.Written && io.Write(path, tr.Orig) == nil {
					tr.Written = false
					dirty = true
				}
				continue
			}
			var want []float32
			if vec {
				want = fx.RenderVec(tr.Orig, setting)
			} else {
				want = fx.Render(tr.Orig, setting)
			}
			if !fx.Same(cur, want) && io.Write(path, want) != nil {
				continue
			}
			if !tr.Written {
				tr.Written = true
				dirty = true
			}
			applied++
		}
	}
	return loaded, applied, dirty
}

// syncRGB handles parts whose colour lives in three float curves (one per
// channel). Same rules as syncCurves: native restores once, then never
// touches; a colour is rewritten whenever the game's value differs.
func syncRGB(io curveIO, parts map[string]fx.Part, profile store.Profile, trk map[string]*tracked) (loaded map[string]bool, applied int, dirty bool) {
	loaded = map[string]bool{}
	for key, part := range parts {
		if len(part.RGB[0]) == 0 {
			continue
		}
		setting, ok := profile[key]
		if !ok {
			setting = fx.Setting{Mode: "native"}
		}
		// every copy index k is one independent set of curves (e.g. CPU/GPU)
		for k := range part.RGB[0] {
			var paths [3]string
			var cur [3][]float32
			var orig [3][]float32
			var lens [3]int
			okAll := true
			for ch := 0; ch < 3; ch++ {
				if k >= len(part.RGB[ch]) {
					okAll = false
					break
				}
				p := part.RGB[ch][k]
				paths[ch] = p
				if !io.Loaded(p) {
					okAll = false
					break
				}
				v, err := io.Read(p)
				if err != nil {
					okAll = false
					break
				}
				cur[ch] = v
				lens[ch] = len(v)
				addr := io.Addr(p)
				tr := trk[p]
				if tr == nil || tr.Addr != addr || len(tr.Orig) != len(v) {
					tr = &tracked{Addr: addr, Orig: v}
					trk[p] = tr
					dirty = true
				}
				orig[ch] = tr.Orig
			}
			if !okAll {
				continue
			}
			loaded[key] = true
			if setting.Mode == "native" {
				for ch := 0; ch < 3; ch++ {
					if tr := trk[paths[ch]]; tr.Written && io.Write(paths[ch], tr.Orig) == nil {
						tr.Written = false
						dirty = true
					}
				}
				continue
			}
			want := fx.RenderRGB(orig, lens, setting)
			for ch := 0; ch < 3; ch++ {
				if sc := part.RGBScale[ch]; sc != 0 && want[ch] != nil {
					for i := range want[ch] {
						want[ch][i] *= sc
					}
				}
			}
			for ch := 0; ch < 3; ch++ {
				if want[ch] == nil || (ch > 0 && paths[ch] == paths[ch-1]) {
					continue // a channel sharing the previous one's curve takes its value
				}
				if !fx.Same(cur[ch], want[ch]) && io.Write(paths[ch], want[ch]) != nil {
					continue
				}
				if tr := trk[paths[ch]]; !tr.Written {
					tr.Written = true
					dirty = true
				}
			}
			applied++
		}
	}
	return loaded, applied, dirty
}
