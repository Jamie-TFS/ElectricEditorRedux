package pak

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

// Edits vector (colour) parameters in a cooked, unversioned legacy
// MaterialInstanceConstant (.uasset + .uexp, as produced by retoc to-legacy).
// Values are fixed-size floats, so patching never changes the file layout.
//
// Layout of one FVectorParameterValue in the .uexp:
//   FName (u32 name-map index, u32 number) | u8 association (2 = global) |
//   i32 index (-1) | 4 x f32 colour | 16-byte GUID
// Cooked files OMIT values that are all zero, so a zero colour has no bytes
// to patch; plausibility checks stop us writing over unrelated data.

var paramTail = []byte{0x02, 0xff, 0xff, 0xff, 0xff}

// LegacyNames reads the name map from a legacy .uasset: a run of FString
// entries (int32 len, ascii, NUL) each followed by a 4-byte hash.
func LegacyNames(uasset []byte) ([]string, error) {
	chain := func(start int) []string {
		var names []string
		p := start
		for p+4 <= len(uasset) {
			n := int(int32(binary.LittleEndian.Uint32(uasset[p:])))
			if n < 1 || n > 1024 || p+4+n+4 > len(uasset) {
				break
			}
			raw := uasset[p+4 : p+4+n]
			if raw[n-1] != 0 {
				break
			}
			ok := true
			for _, c := range raw[:n-1] {
				if c < 32 || c > 126 {
					ok = false
					break
				}
			}
			if !ok {
				break
			}
			names = append(names, string(raw[:n-1]))
			p += 4 + n + 4
		}
		return names
	}
	var best []string
	for i := 0; i+8 < len(uasset); i++ {
		if c := chain(i); len(c) > len(best) {
			best = c
		}
	}
	if len(best) < 5 {
		return nil, errors.New("name map not found in .uasset")
	}
	return best, nil
}

// Param is one colour parameter found in a .uexp.
type Param struct {
	Name   string
	Offset int
	Value  [4]float32
}

func plausible(v [4]float32) bool {
	for _, f := range v {
		if math.IsNaN(float64(f)) || math.Abs(float64(f)) >= 10000 {
			return false
		}
	}
	return true
}

// FindParams returns the vector parameters it can safely patch, by name.
func FindParams(uexp []byte, names []string) map[string]Param {
	out := map[string]Param{}
	for idx, name := range names {
		pat := make([]byte, 8, 13)
		binary.LittleEndian.PutUint32(pat, uint32(idx))
		pat = append(pat, paramTail...)
		hits := 0
		at := -1
		for i := 0; i+len(pat) <= len(uexp); i++ {
			if string(uexp[i:i+len(pat)]) == string(pat) {
				hits++
				at = i + len(pat)
			}
		}
		if hits != 1 || at+16 > len(uexp) {
			continue
		}
		var v [4]float32
		for k := 0; k < 4; k++ {
			v[k] = math.Float32frombits(binary.LittleEndian.Uint32(uexp[at+k*4:]))
		}
		if plausible(v) {
			out[strings.ToLower(name)] = Param{Name: name, Offset: at, Value: v}
		}
	}
	return out
}

// SetParam overwrites one parameter's colour in place.
func SetParam(uexp []byte, p Param, v [4]float32) error {
	if p.Offset+16 > len(uexp) {
		return fmt.Errorf("%s: offset out of range", p.Name)
	}
	for k := 0; k < 4; k++ {
		binary.LittleEndian.PutUint32(uexp[p.Offset+k*4:], math.Float32bits(v[k]))
	}
	return nil
}
