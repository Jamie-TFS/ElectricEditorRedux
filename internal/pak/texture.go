package pak

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Recolouring cooked BC1/BC3 (DXT1/DXT5) textures without decoding them:
// every 4x4 block stores two endpoint colours (RGB565) and per-pixel indices
// that blend between them. Tinting the two endpoints recolours the whole
// block; indices and alpha stay as they are, so no re-encoding is needed.
//
// Legacy cooked Texture2D layout (retoc to-legacy, unversioned), after the
// property data:
//   i32 SizeX, i32 SizeY, u32 PackedData, FString PixelFormat ("PF_DXT5"),
//   i32 FirstMip, i32 NumMips, then per mip:
//     i32 DataResourceIndex, [inline payload], i32 SizeX, SizeY, SizeZ
// The largest mips live in the .ubulk, back to back; the rest are inline.

type mipRegion struct {
	inBulk bool
	off    int // offset in .ubulk or .uexp
	size   int
}

func blockBytes(format string) (int, error) {
	switch format {
	case "PF_DXT5":
		return 16, nil
	case "PF_DXT1":
		return 8, nil
	}
	return 0, fmt.Errorf("unsupported texture format %s (only DXT1/DXT5)", format)
}

// textureMips finds where each mip's block data sits.
func textureMips(uexp []byte, bulkLen int) ([]mipRegion, int, error) {
	var at int
	var format string
	for _, f := range []string{"PF_DXT5", "PF_DXT1"} {
		pat := append([]byte{byte(len(f) + 1), 0, 0, 0}, []byte(f+"\x00")...)
		if i := bytes.Index(uexp, pat); i >= 12 {
			at, format = i, f
			break
		}
	}
	if format == "" {
		return nil, 0, errors.New("not a DXT1/DXT5 texture")
	}
	bb, _ := blockBytes(format)
	sx := int(int32(binary.LittleEndian.Uint32(uexp[at-12:])))
	sy := int(int32(binary.LittleEndian.Uint32(uexp[at-8:])))
	p := at + 4 + len(format) + 1
	if p+8 > len(uexp) || sx <= 0 || sy <= 0 || sx > 16384 || sy > 16384 {
		return nil, 0, errors.New("texture header not understood")
	}
	num := int(int32(binary.LittleEndian.Uint32(uexp[p+4:])))
	p += 8
	if num < 1 || num > 16 {
		return nil, 0, errors.New("texture mip count not understood")
	}
	var mips []mipRegion
	used := 0
	for m := 0; m < num; m++ {
		w, h := max(1, sx>>m), max(1, sy>>m)
		size := max(1, (w+3)/4) * max(1, (h+3)/4) * bb
		p += 4 // data resource index
		if used+size <= bulkLen && used < bulkLen {
			mips = append(mips, mipRegion{inBulk: true, off: used, size: size})
			used += size
		} else {
			if p+size > len(uexp) {
				return nil, 0, errors.New("texture mip data runs past the file")
			}
			mips = append(mips, mipRegion{off: p, size: size})
			p += size
		}
		if p+12 > len(uexp) {
			return nil, 0, errors.New("texture mip table runs past the file")
		}
		if int(binary.LittleEndian.Uint32(uexp[p:])) != w || int(binary.LittleEndian.Uint32(uexp[p+4:])) != h {
			return nil, 0, fmt.Errorf("mip %d size mismatch - texture layout not understood", m)
		}
		p += 12
	}
	if used != bulkLen {
		return nil, 0, fmt.Errorf("bulk data size %d doesn't match the mips (%d)", bulkLen, used)
	}
	return mips, bb, nil
}

func unpack565(c uint16) (float64, float64, float64) {
	return float64(c>>11&31) / 31, float64(c>>5&63) / 63, float64(c&31) / 31
}

func pack565(r, g, b float64) uint16 {
	q := func(v float64, n float64) uint16 {
		return uint16(math.Round(math.Max(0, math.Min(1, v)) * n))
	}
	return q(r, 31)<<11 | q(g, 63)<<5 | q(b, 31)
}

// tintBlocks recolours every colour block: each endpoint keeps its
// brightness (strongest channel) and takes the tint's hue.
func tintBlocks(data []byte, bb int, tint [3]float64) {
	m := math.Max(tint[0], math.Max(tint[1], tint[2]))
	if m <= 0 {
		m = 1
	}
	t := [3]float64{tint[0] / m, tint[1] / m, tint[2] / m}
	colour := bb - 8 // colour half of the block (after BC3's alpha half)
	for i := 0; i+bb <= len(data); i += bb {
		o := i + colour
		c0 := binary.LittleEndian.Uint16(data[o:])
		c1 := binary.LittleEndian.Uint16(data[o+2:])
		n := [2]uint16{}
		for e, c := range [2]uint16{c0, c1} {
			r, g, b := unpack565(c)
			v := math.Max(r, math.Max(g, b))
			n[e] = pack565(v*t[0], v*t[1], v*t[2])
		}
		if bb == 8 && (c0 > c1) != (n[0] > n[1]) {
			// DXT1: the endpoint order picks 4-colour vs 3-colour+transparent
			// mode, so it must survive. Swap the endpoints and remap indices.
			if n[0] == n[1] {
				if n[0] > 0 {
					n[1]--
				} else {
					n[0]++
				}
			} else {
				n[0], n[1] = n[1], n[0]
				idx := binary.LittleEndian.Uint32(data[o+4:])
				var out uint32
				for k := 0; k < 16; k++ {
					v := idx >> (2 * k) & 3
					v = [4]uint32{1, 0, 3, 2}[v]
					out |= v << (2 * k)
				}
				binary.LittleEndian.PutUint32(data[o+4:], out)
			}
		}
		binary.LittleEndian.PutUint16(data[o:], n[0])
		binary.LittleEndian.PutUint16(data[o+2:], n[1])
	}
}

// TintTexture recolours a DXT texture's .uexp (inline mips) and .ubulk
// (large mips) in place.
func TintTexture(uexp, bulk []byte, tint [3]float64) error {
	mips, bb, err := textureMips(uexp, len(bulk))
	if err != nil {
		return err
	}
	for _, m := range mips {
		if m.inBulk {
			tintBlocks(bulk[m.off:m.off+m.size], bb, tint)
		} else {
			tintBlocks(uexp[m.off:m.off+m.size], bb, tint)
		}
	}
	return nil
}
