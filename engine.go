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
	tracked map[string]*tracked // curves of the current Tekken session
	nextTry time.Time
	dumpReq chan chan dumpResult

	trailReq   chan trailRequest
	trailSaved map[string][4]float32 // original colours while the trail test is on
}

type trailRequest struct {
	on    bool
	reply chan dumpResult
}

// words that pick out trail / swipe materials (own path or parent path)
var trailWords = []string{"trail", "_tl", "blade", "slash", "claw", "swing"}

// colour parameters a trail material uses
var trailColorParams = map[string]bool{"basetexcolor": true, "basetex2color": true, "edgecolor": true, "color": true}

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
		store:    s,
		parts:    fx.AllParts(),
		wake:     make(chan struct{}, 1),
		tracked:  map[string]*tracked{},
		dumpReq:  make(chan chan dumpResult),
		trailReq: make(chan trailRequest),
		status:   Status{State: "waiting", Message: "Start Tekken 8 and this will connect on its own.", Loaded: map[string]bool{}},
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
			case req := <-e.trailReq:
				req.reply <- e.trailTest(req.on)
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
	}

	if !e.sess.Alive() {
		e.dropSession()
		e.setStatus(Status{State: "waiting", Message: "Tekken closed. Waiting for it to start again.", Loaded: map[string]bool{}})
		return
	}
	e.sess.MaybeRescan()

	loaded, applied, dirty := syncCurves(e.sess, e.parts, e.store.Active(), e.tracked)
	if dirty {
		saveSession(e.sess.PID(), e.tracked)
	}

	msg := "Connected. Load a match to see effects here."
	if len(loaded) > 0 {
		msg = "Connected. Changes show up in game straight away."
	}
	e.setStatus(Status{State: "connected", Message: msg, Process: e.sess.Process(),
		Curves: e.sess.CurveCount(), Loaded: loaded, Applied: applied})
}

func (e *Engine) dropSession() {
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
		fmt.Fprintf(&b, "%s | %d%s\n", p, list[p], tag)
	}
	b.WriteString("\n--- trail, claw and slash materials (for finding claw colours) ---\n\n")
	for _, m := range e.sess.Materials(trailWords) {
		fmt.Fprintf(&b, "%s\n    parent: %s\n", m.Path, m.Parent)
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
		for _, path := range part.Paths {
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
			want := fx.Render(tr.Orig, setting)
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

// TrailTest turns every loaded trail material's colours green in memory (on)
// or puts them back (off). It answers one question: does Tekken's trail
// renderer pick up live material changes? If yes, claws can be recoloured
// live like the electrics; if not, they need a pak.
func (e *Engine) TrailTest(on bool) (int, error) {
	reply := make(chan dumpResult, 1)
	select {
	case e.trailReq <- trailRequest{on: on, reply: reply}:
	case <-time.After(3 * time.Second):
		return 0, errors.New("the app is busy, try again")
	}
	r := <-reply
	n := 0
	fmt.Sscan(r.path, &n)
	return n, r.err
}

func (e *Engine) trailTest(on bool) dumpResult {
	if e.sess == nil {
		return dumpResult{err: errors.New("connect to Tekken first, then load a match")}
	}
	if on && e.trailSaved == nil {
		e.trailSaved = map[string][4]float32{}
	}
	n := 0
	for _, m := range e.sess.Materials(trailWords) {
		for _, p := range m.Params {
			if !trailColorParams[strings.ToLower(p.Name)] {
				continue
			}
			key := m.Path + "|" + p.Name
			if on {
				if _, ok := e.trailSaved[key]; !ok {
					e.trailSaved[key] = p.Value
				}
				if e.sess.SetVector(p, [4]float32{0, 1, 0, p.Value[3]}) == nil {
					n++
				}
			} else if orig, ok := e.trailSaved[key]; ok {
				if e.sess.SetVector(p, orig) == nil {
					n++
				}
			}
		}
	}
	if !on {
		e.trailSaved = nil
	}
	return dumpResult{path: fmt.Sprint(n)}
}
