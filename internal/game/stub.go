//go:build !windows

package game

import "errors"

// Non-Windows stub so the app (and Wails binding generation) builds anywhere.
// Tekken 8 only runs on Windows, so there is nothing to attach to here.

var ErrNotRunning = errors.New("Tekken 8 is not running")

type Session struct{}

func Open() (*Session, error)                     { return nil, ErrNotRunning }
func (s *Session) Close()                         {}
func (s *Session) Alive() bool                    { return false }
func (s *Session) Process() string                { return "" }
func (s *Session) ExePath() string                { return "" }
func (s *Session) MaybeRescan() bool              { return false }
func (s *Session) Rescan()                        {}
func (s *Session) Loaded(string) bool             { return false }
func (s *Session) IsFloat(string) bool            { return false }
func (s *Session) CurveCount() int                { return 0 }
func (s *Session) PID() uint32                    { return 0 }
func (s *Session) Addr(string) uint64             { return 0 }
func (s *Session) CurveList() map[string]int      { return nil }
func (s *Session) Read(string) ([]float32, error) { return nil, ErrNotRunning }
func (s *Session) Write(string, []float32) error  { return ErrNotRunning }

type VectorParam struct {
	Name  string     `json:"name"`
	Value [4]float32 `json:"value"`
}

type Material struct {
	Dynamic bool          `json:"dynamic"`
	Path    string        `json:"path"`
	Parent  string        `json:"parent"`
	Params  []VectorParam `json:"params"`
}

func (s *Session) Materials([]string) []Material           { return nil }
func (s *Session) SetVector(VectorParam, [4]float32) error { return ErrNotRunning }

func (s *Session) ScriptTables(string, [][]byte) [][]uint64 { return nil }
func (s *Session) ScriptCount(string) int                   { return 0 }
func (s *Session) WriteChecked(uint64, []byte) error        { return ErrNotRunning }
