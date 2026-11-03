package fx

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

type Stop struct {
	Pos   float64 `json:"pos"` // 0..1 across the effect's lifetime
	Color string  `json:"color"`
}

type Setting struct {
	Mode       string  `json:"mode"` // native | solid | gradient
	Color      string  `json:"color"`
	Stops      []Stop  `json:"stops"`
	Brightness float64 `json:"brightness"`
	KeepFade   bool    `json:"keepFade"`
}

func DefaultSetting() Setting {
	return Setting{
		Mode:       "native",
		Color:      "#7c5cff",
		Stops:      []Stop{{0, "#00e5ff"}, {1, "#7c5cff"}},
		Brightness: 6,
		KeepFade:   true,
	}
}

type rgb struct{ r, g, b float64 }

func ParseHex(s string) (rgb, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return rgb{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return rgb{}, false
	}
	return rgb{float64(v>>16&0xFF) / 255, float64(v>>8&0xFF) / 255, float64(v&0xFF) / 255}, true
}

func (s Setting) colorAt(t float64) rgb {
	if s.Mode != "gradient" || len(s.Stops) == 0 {
		c, _ := ParseHex(s.Color)
		return c
	}
	stops := append([]Stop(nil), s.Stops...)
	sort.Slice(stops, func(i, j int) bool { return stops[i].Pos < stops[j].Pos })
	if t <= stops[0].Pos {
		c, _ := ParseHex(stops[0].Color)
		return c
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].Pos {
			a, _ := ParseHex(stops[i-1].Color)
			b, _ := ParseHex(stops[i].Color)
			span := stops[i].Pos - stops[i-1].Pos
			k := 0.0
			if span > 0 {
				k = (t - stops[i-1].Pos) / span
			}
			return rgb{a.r + (b.r-a.r)*k, a.g + (b.g-a.g)*k, a.b + (b.b-a.b)*k}
		}
	}
	c, _ := ParseHex(stops[len(stops)-1].Color)
	return c
}

// Render builds the RGBA curve to write, from the game's original curve.
//   - solid/gradient colour sampled across the curve's lifetime
//   - Brightness is the HDR multiplier (the game's electrics run ~3-8)
//   - KeepFade keeps the original brightness shape and alpha fade
func Render(orig []float32, s Setting) []float32 {
	if s.Mode == "native" || len(orig) < 4 {
		return append([]float32(nil), orig...)
	}
	n := len(orig) / 4
	peak := 0.0
	for i := 0; i < n; i++ {
		peak = math.Max(peak, maxf(orig[i*4], orig[i*4+1], orig[i*4+2]))
	}
	bright := s.Brightness
	if bright <= 0 {
		bright = 1
	}
	out := make([]float32, len(orig))
	for i := 0; i < n; i++ {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		c := s.colorAt(t)
		f, a := 1.0, 1.0
		if s.KeepFade {
			a = float64(orig[i*4+3])
			if peak > 0 {
				f = maxf(orig[i*4], orig[i*4+1], orig[i*4+2]) / peak
			}
		}
		k := bright * f
		out[i*4] = float32(c.r * k)
		out[i*4+1] = float32(c.g * k)
		out[i*4+2] = float32(c.b * k)
		out[i*4+3] = float32(a)
	}
	return out
}

func maxf(a, b, c float32) float64 { return math.Max(float64(a), math.Max(float64(b), float64(c))) }

// Same reports whether two curves are equal within float noise.
func Same(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(float64(a[i]-b[i])) > 1e-4 {
			return false
		}
	}
	return true
}
