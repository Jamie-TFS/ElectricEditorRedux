//go:build windows

package game

import (
	"errors"
	"time"
)

const (
	colorCurveClass = "NiagaraDataInterfaceColorCurve"
	// Float curves share the curve base class, so their ShaderLUT sits at
	// the same offset. Some effects (DVJ's beam) drive colour through them.
	floatCurveClass = "NiagaraDataInterfaceCurve"
	// Vector curves (xyz per sample) carry "Scale RGB" colour multipliers.
	vectorCurveClass = "NiagaraDataInterfaceVectorCurve"
)

type curve struct {
	obj   uint64
	num   int
	float bool // float or vector curve (not RGBA)
	class string
}

// Session is a live connection to the game with an index of its colour curves.
type Session struct {
	p        *Process
	n        *names
	o        *objects
	lutOff   uint64
	curves   map[string]curve
	lastNum  int
	lastScan time.Time
}

func Open() (*Session, error) {
	p, err := Attach()
	if err != nil {
		return nil, err
	}
	n, err := findNames(p)
	if err != nil {
		p.Close()
		return nil, err
	}
	o, err := findObjects(p, n)
	if err != nil {
		p.Close()
		return nil, err
	}
	s := &Session{p: p, n: n, o: o, curves: map[string]curve{}}
	s.Rescan()
	return s, nil
}

func (s *Session) Close()          { s.p.Close() }
func (s *Session) Alive() bool     { return s.p.Alive() }
func (s *Session) Process() string { return s.p.Name }

// ExePath is the full path of the running Polaris exe (finds the game folder).
func (s *Session) ExePath() string { return s.p.ExePath }

// MaybeRescan re-indexes curves when the object count changes (a match or
// character loaded) or every 10s as a fallback.
func (s *Session) MaybeRescan() bool {
	if s.o.count() != s.lastNum || time.Since(s.lastScan) > 10*time.Second {
		s.Rescan()
		return true
	}
	return false
}

func (s *Session) Rescan() {
	num := s.o.count()
	var objs, floats, vecs []uint64
	for i := 0; i < num; i++ {
		obj := s.o.item(i)
		if obj == 0 {
			continue
		}
		switch s.o.className(obj) {
		case colorCurveClass:
			objs = append(objs, obj)
		case floatCurveClass:
			floats = append(floats, obj)
		case vectorCurveClass:
			vecs = append(vecs, obj)
		}
	}

	if s.lutOff == 0 {
		votes := map[uint64]int{}
		for _, obj := range objs {
			for _, l := range lutCandidates(s.p, obj) {
				votes[l.off]++
			}
		}
		best := 0
		for off, v := range votes {
			if v > best && v >= 3 {
				s.lutOff, best = off, v
			}
		}
	}

	curves := map[string]curve{}
	if s.lutOff != 0 {
		for _, obj := range objs {
			if l, ok := s.lut(obj); ok {
				curves[s.o.path(obj)] = curve{obj: obj, num: int(l.num)}
			}
		}
		for _, obj := range floats {
			if l, ok := s.flut(obj); ok {
				curves[s.o.path(obj)] = curve{obj: obj, num: int(l.num), float: true, class: floatCurveClass}
			}
		}
		for _, obj := range vecs {
			if l, ok := s.flut(obj); ok && l.num%3 == 0 {
				curves[s.o.path(obj)] = curve{obj: obj, num: int(l.num), float: true, class: vectorCurveClass}
			}
		}
	}
	s.curves = curves
	s.lastNum = num
	s.lastScan = time.Now()
}

func (s *Session) lut(obj uint64) (lutRef, bool) {
	ptr := s.p.u64(obj + s.lutOff)
	num := int32(s.p.u32(obj + s.lutOff + 8))
	if !plausiblePtr(ptr) || num < 4 || num > 4096 || num%4 != 0 {
		return lutRef{}, false
	}
	return lutRef{off: s.lutOff, ptr: ptr, num: num}, true
}

func (s *Session) flut(obj uint64) (lutRef, bool) {
	ptr := s.p.u64(obj + s.lutOff)
	num := int32(s.p.u32(obj + s.lutOff + 8))
	if !plausiblePtr(ptr) || num < 2 || num > 4096 {
		return lutRef{}, false
	}
	return lutRef{off: s.lutOff, ptr: ptr, num: num}, true
}

func (s *Session) ref(c curve) (lutRef, bool) {
	if c.float {
		return s.flut(c.obj)
	}
	return s.lut(c.obj)
}

// IsFloat reports whether a path is a float curve (one value per sample)
// rather than an RGBA colour curve.
func (s *Session) IsFloat(path string) bool { return s.curves[path].float }

// Loaded reports whether a curve path is currently in memory.
func (s *Session) Loaded(path string) bool {
	_, ok := s.curves[path]
	return ok
}

func (s *Session) CurveCount() int { return len(s.curves) }

func (s *Session) PID() uint32 { return s.p.PID }

// Addr is the curve object's address; it changes when the game reloads it.
func (s *Session) Addr(path string) uint64 { return s.curves[path].obj }

// CurveList returns every indexed colour curve with its float count.
func (s *Session) CurveList() map[string]int {
	out := map[string]int{}
	for k, c := range s.curves {
		out[k] = c.num
	}
	return out
}

var errGone = errors.New("curve no longer loaded")

// Read returns a curve's current RGBA floats.
func (s *Session) Read(path string) ([]float32, error) {
	c, ok := s.curves[path]
	if !ok {
		return nil, errGone
	}
	want := colorCurveClass
	if c.float {
		want = c.class
	}
	if s.o.className(c.obj) != want {
		delete(s.curves, path)
		return nil, errGone
	}
	l, ok := s.ref(c)
	if !ok {
		return nil, errGone
	}
	v := s.p.readFloats(l.ptr, int(l.num))
	if v == nil {
		return nil, errGone
	}
	return v, nil
}

// Write replaces a curve's RGBA floats (length must match).
func (s *Session) Write(path string, v []float32) error {
	c, ok := s.curves[path]
	if !ok {
		return errGone
	}
	l, ok := s.ref(c)
	if !ok || int(l.num) != len(v) {
		return errGone
	}
	return s.p.writeFloats(l.ptr, v)
}
