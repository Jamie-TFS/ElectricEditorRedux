//go:build windows

package game

import (
	"encoding/binary"
	"math"
	"strings"
)

// Material instances: Tekken draws claw / swipe trails with its own trail
// renderer, coloured by MaterialInstanceConstant vector parameters
// (BaseTexColor etc). This finds them and their colour values in memory.

const (
	vecParamStride = 48 // FMaterialParameterInfo(16) + FLinearColor(16) + FGuid(16)
	micClass       = "MaterialInstanceConstant"
)

type VectorParam struct {
	Name  string     `json:"name"`
	Value [4]float32 `json:"value"`
	addr  uint64     // where Value lives
}

type Material struct {
	Path   string        `json:"path"`
	Parent string        `json:"parent"`
	Params []VectorParam `json:"params"`
	obj    uint64
}

// vectorParams finds the VectorParameterValues array inside a material
// instance by shape: every element must decode to a real name + sane colour.
func (s *Session) vectorParams(obj uint64) []VectorParam {
	buf := make([]byte, 0x400)
	s.p.read(obj, buf)
	for off := 0x28; off+16 <= len(buf); off += 8 {
		ptr := binary.LittleEndian.Uint64(buf[off:])
		num := int32(binary.LittleEndian.Uint32(buf[off+8:]))
		mx := int32(binary.LittleEndian.Uint32(buf[off+12:]))
		if !plausiblePtr(ptr) || s.p.inModule(ptr) || num < 1 || num > 64 || mx < num || mx > 128 {
			continue
		}
		raw := make([]byte, int(num)*vecParamStride)
		if !s.p.read(ptr, raw) {
			continue
		}
		var out []VectorParam
		ok := true
		for i := 0; i < int(num) && ok; i++ {
			e := raw[i*vecParamStride:]
			assoc := e[8]
			index := int32(binary.LittleEndian.Uint32(e[12:]))
			name := s.n.get(binary.LittleEndian.Uint32(e[0:]), binary.LittleEndian.Uint32(e[4:]))
			if assoc > 2 || index < -1 || index > 64 || !printableName(name) {
				ok = false
				break
			}
			var v [4]float32
			for k := 0; k < 4; k++ {
				f := math.Float32frombits(binary.LittleEndian.Uint32(e[16+k*4:]))
				if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) || math.Abs(float64(f)) > 100000 {
					ok = false
					break
				}
				v[k] = f
			}
			out = append(out, VectorParam{Name: name, Value: v, addr: ptr + uint64(i*vecParamStride) + 16})
		}
		if ok && len(out) > 0 {
			return out
		}
	}
	return nil
}

func printableName(n string) bool {
	if n == "" || len(n) > 64 {
		return false
	}
	for _, c := range n {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}

// parent finds the first pointer field that points at a material.
func (s *Session) parent(obj uint64) string {
	buf := make([]byte, 0x200)
	s.p.read(obj, buf)
	for off := 0x28; off+8 <= len(buf); off += 8 {
		ptr := binary.LittleEndian.Uint64(buf[off:])
		if !plausiblePtr(ptr) || s.p.inModule(ptr) {
			continue
		}
		switch s.o.className(ptr) {
		case "Material", micClass:
			return s.o.path(ptr)
		}
	}
	return ""
}

// Materials lists loaded material instances whose own path or parent path
// contains any of the (lower-case) words.
func (s *Session) Materials(words []string) []Material {
	match := func(p string) bool {
		l := strings.ToLower(p)
		for _, w := range words {
			if strings.Contains(l, w) {
				return true
			}
		}
		return false
	}
	var out []Material
	num := s.o.count()
	for i := 0; i < num; i++ {
		obj := s.o.item(i)
		if obj == 0 || s.o.className(obj) != micClass {
			continue
		}
		path := s.o.path(obj)
		if strings.Contains(path, "Default__") {
			continue
		}
		parent := s.parent(obj)
		if !match(path) && !match(parent) {
			continue
		}
		out = append(out, Material{Path: path, Parent: parent, Params: s.vectorParams(obj), obj: obj})
	}
	return out
}

// SetVector overwrites one vector parameter's value in memory.
func (s *Session) SetVector(p VectorParam, v [4]float32) error {
	return s.p.writeFloats(p.addr, v[:])
}
