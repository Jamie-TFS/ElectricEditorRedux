//go:build windows

package game

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"syscall"
)

// Locates Unreal's name table and object array by looking for the data
// itself (not version-specific code patterns), so game updates that move
// code around don't break it. Proven on Tekken 8 3.02.02 by eeprobe.

type section struct {
	addr, size uint64
	writable   bool
}

func (p *Process) sections() []section {
	hdr := make([]byte, 0x1000)
	if !p.read(p.base, hdr) {
		return nil
	}
	nt := binary.LittleEndian.Uint32(hdr[0x3C:])
	nsec := int(binary.LittleEndian.Uint16(hdr[nt+6:]))
	optSize := int(binary.LittleEndian.Uint16(hdr[nt+20:]))
	off := int(nt) + 24 + optSize
	var out []section
	for i := 0; i < nsec && off+40 <= len(hdr); i++ {
		s := hdr[off : off+40]
		out = append(out, section{
			size:     uint64(binary.LittleEndian.Uint32(s[8:])),
			addr:     p.base + uint64(binary.LittleEndian.Uint32(s[12:])),
			writable: binary.LittleEndian.Uint32(s[36:])&0x80000000 != 0,
		})
		off += 40
	}
	return out
}

func (p *Process) readSection(s section) []byte {
	buf := make([]byte, s.size)
	const page = 0x1000
	for o := uint64(0); o < s.size; o += page {
		end := min(o+page, s.size)
		p.read(s.addr+o, buf[o:end])
	}
	return buf
}

func (p *Process) inModule(a uint64) bool { return a >= p.base && a < p.base+p.size }

func plausiblePtr(a uint64) bool { return a > 0x10000 && a < 0x7FFFFFFFFFFF && a%8 == 0 }

// ------------------------------------------------------------------ names

type names struct {
	p      *Process
	blocks uint64
	shift  uint
	cache  map[uint32]string
}

func findNames(p *Process) (*names, error) {
	needle := []byte("ByteProperty")
	var block0 uint64
	buf := make([]byte, 16<<20)
	for _, r := range p.heapRegions() {
		for o := uint64(0); o < r.size && block0 == 0; o += uint64(len(buf)) - 64 {
			n := min(uint64(len(buf)), r.size-o)
			chunk := buf[:n]
			if !p.read(r.base+o, chunk) {
				continue
			}
			for i := bytes.Index(chunk, needle); i >= 0; {
				if i >= 8 && string(chunk[i-6:i-2]) == "None" {
					block0 = r.base + o + uint64(i-8)
					break
				}
				j := bytes.Index(chunk[i+1:], needle)
				if j < 0 {
					break
				}
				i += 1 + j
			}
		}
		if block0 != 0 {
			break
		}
	}
	if block0 == 0 {
		return nil, errors.New("name table not found (is a match or menu fully loaded?)")
	}

	hdr := p.u16(block0)
	var shift uint = 99
	for _, s := range []uint{6, 1, 0} {
		if (hdr>>s)&0x3FF == 4 && hdr&1 == 0 {
			shift = s
			break
		}
	}
	if shift == 99 {
		return nil, fmt.Errorf("unknown name format %#x", hdr)
	}

	var want [8]byte
	binary.LittleEndian.PutUint64(want[:], block0)
	for _, s := range p.sections() {
		if !s.writable {
			continue
		}
		data := p.readSection(s)
		for i := 0; i+8 <= len(data); i += 8 {
			if bytes.Equal(data[i:i+8], want[:]) {
				n := &names{p: p, blocks: s.addr + uint64(i), shift: shift, cache: map[uint32]string{}}
				if n.get(0, 0) == "None" {
					return n, nil
				}
			}
		}
	}
	return nil, errors.New("name table pointer not found")
}

func (n *names) get(idx, number uint32) string {
	s, ok := n.cache[idx]
	if !ok {
		blk := n.p.u64(n.blocks + 8*uint64(idx>>16))
		entry := blk + 2*uint64(idx&0xFFFF)
		hdr := n.p.u16(entry)
		ln := int((hdr >> n.shift) & 0x3FF)
		switch {
		case blk == 0 || ln == 0:
			s = ""
		case hdr&1 != 0:
			b := make([]byte, ln*2)
			n.p.read(entry+2, b)
			u := make([]uint16, ln)
			for i := range u {
				u[i] = binary.LittleEndian.Uint16(b[i*2:])
			}
			s = syscall.UTF16ToString(u)
		default:
			b := make([]byte, ln)
			n.p.read(entry+2, b)
			s = string(b)
		}
		n.cache[idx] = s
	}
	if number > 0 {
		return fmt.Sprintf("%s_%d", s, number-1)
	}
	return s
}

// ------------------------------------------------------------------ objects

const (
	offIndex = 0x0C
	offClass = 0x10
	offName  = 0x18
	offOuter = 0x20
)

type objects struct {
	p        *Process
	n        *names
	header   uint64 // address of FChunkedFixedUObjectArray
	chunks   uint64
	stride   uint64
	nameMemo map[uint64]string
}

func findObjects(p *Process, n *names) (*objects, error) {
	for _, s := range p.sections() {
		if !s.writable {
			continue
		}
		data := p.readSection(s)
		for i := 0; i+32 <= len(data); i += 4 {
			objs := binary.LittleEndian.Uint64(data[i:])
			maxE := int32(binary.LittleEndian.Uint32(data[i+16:]))
			numE := int32(binary.LittleEndian.Uint32(data[i+20:]))
			maxC := int32(binary.LittleEndian.Uint32(data[i+24:]))
			numC := int32(binary.LittleEndian.Uint32(data[i+28:]))
			if !plausiblePtr(objs) || numE < 10000 || numE > maxE || maxE > 100000000 {
				continue
			}
			if maxC != (maxE+65535)/65536 || numC != (numE+65535)/65536 {
				continue
			}
			for _, stride := range []uint64{24, 16, 32} {
				o := &objects{p: p, n: n, header: s.addr + uint64(i), chunks: objs, stride: stride, nameMemo: map[uint64]string{}}
				ok := true
				for _, k := range []int{1, 2, 3, 100, 1000, 5000} {
					obj := o.item(k)
					if obj == 0 || int(p.u32(obj+offIndex)) != k {
						ok = false
						break
					}
				}
				if ok {
					return o, nil
				}
			}
		}
	}
	return nil, errors.New("object array not found")
}

func (o *objects) count() int { return int(int32(o.p.u32(o.header + 20))) }

func (o *objects) item(i int) uint64 {
	chunk := o.p.u64(o.chunks + 8*uint64(i/65536))
	if chunk == 0 {
		return 0
	}
	return o.p.u64(chunk + o.stride*uint64(i%65536))
}

func (o *objects) name(obj uint64) string {
	var b [8]byte
	o.p.read(obj+offName, b[:])
	return o.n.get(binary.LittleEndian.Uint32(b[:4]), binary.LittleEndian.Uint32(b[4:]))
}

// classes are long-lived, so their names are memoised by address
func (o *objects) className(obj uint64) string {
	cls := o.p.u64(obj + offClass)
	if cls == 0 {
		return ""
	}
	if s, ok := o.nameMemo[cls]; ok {
		return s
	}
	s := o.name(cls)
	o.nameMemo[cls] = s
	return s
}

// /Pkg/Path.Asset:Sub.Sub - same format as UE4SS / GetFullName
func (o *objects) path(obj uint64) string {
	var chain []string
	for cur, d := obj, 0; cur != 0 && d < 16; d++ {
		chain = append([]string{o.name(cur)}, chain...)
		cur = o.p.u64(cur + offOuter)
	}
	var sb strings.Builder
	for i, c := range chain {
		switch {
		case i == 0:
		case i == 2:
			sb.WriteByte(':')
		default:
			sb.WriteByte('.')
		}
		sb.WriteString(c)
	}
	return sb.String()
}

// ------------------------------------------------------------------ LUTs

type lutRef struct {
	off      uint64
	ptr      uint64
	num, max int32
}

func lutCandidates(p *Process, obj uint64) []lutRef {
	buf := make([]byte, 0x300)
	p.read(obj, buf)
	var out []lutRef
	for off := 0x28; off+16 <= len(buf); off += 8 {
		ptr := binary.LittleEndian.Uint64(buf[off:])
		num := int32(binary.LittleEndian.Uint32(buf[off+8:]))
		mx := int32(binary.LittleEndian.Uint32(buf[off+12:]))
		if !plausiblePtr(ptr) || p.inModule(ptr) || num < 8 || num > 4096 || num%4 != 0 || mx < num || mx > num*2+64 {
			continue
		}
		if v := p.readFloats(ptr, int(num)); v != nil && finite(v) {
			out = append(out, lutRef{uint64(off), ptr, num, mx})
		}
	}
	return out
}

func finite(v []float32) bool {
	for _, f := range v {
		if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) || math.Abs(float64(f)) > 10000 {
			return false
		}
	}
	return true
}

func (p *Process) readFloats(addr uint64, n int) []float32 {
	b := make([]byte, 4*n)
	if !p.read(addr, b) {
		return nil
	}
	out := make([]float32, n)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func (p *Process) writeFloats(addr uint64, v []float32) error {
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(f))
	}
	return p.write(addr, b)
}
