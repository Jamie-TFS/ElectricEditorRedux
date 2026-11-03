import type { Setting, Stop } from "./api";

export type RGB = [number, number, number];

export function hexToRgb(hex: string): RGB {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
  if (!m) return [1, 1, 1];
  const v = parseInt(m[1], 16);
  return [(v >> 16 & 255) / 255, (v >> 8 & 255) / 255, (v & 255) / 255];
}

export function rgbToHex([r, g, b]: RGB): string {
  const h = (x: number) => Math.round(Math.max(0, Math.min(1, x)) * 255).toString(16).padStart(2, "0");
  return `#${h(r)}${h(g)}${h(b)}`;
}

export function hsvToRgb(h: number, s: number, v: number): RGB {
  const f = (n: number) => {
    const k = (n + h / 60) % 6;
    return v - v * s * Math.max(0, Math.min(k, 4 - k, 1));
  };
  return [f(5), f(3), f(1)];
}

export function rgbToHsv([r, g, b]: RGB): [number, number, number] {
  const max = Math.max(r, g, b), min = Math.min(r, g, b), d = max - min;
  let h = 0;
  if (d) {
    if (max === r) h = ((g - b) / d) % 6;
    else if (max === g) h = (b - r) / d + 2;
    else h = (r - g) / d + 4;
    h *= 60;
    if (h < 0) h += 360;
  }
  return [h, max ? d / max : 0, max];
}

export function sortedStops(stops: Stop[]): Stop[] {
  return [...stops].sort((a, b) => a.pos - b.pos);
}

// Colour at t (0..1) across the effect's lifetime - mirrors fx.colorAt in Go.
export function colorAt(s: Setting, t: number, nativeHex: string): RGB {
  if (s.mode === "native") return hexToRgb(nativeHex);
  if (s.mode === "solid" || !s.stops.length) return hexToRgb(s.color);
  const st = sortedStops(s.stops);
  if (t <= st[0].pos) return hexToRgb(st[0].color);
  for (let i = 1; i < st.length; i++) {
    if (t <= st[i].pos) {
      const a = hexToRgb(st[i - 1].color), b = hexToRgb(st[i].color);
      const span = st[i].pos - st[i - 1].pos;
      const k = span > 0 ? (t - st[i - 1].pos) / span : 0;
      return [a[0] + (b[0] - a[0]) * k, a[1] + (b[1] - a[1]) * k, a[2] + (b[2] - a[2]) * k];
    }
  }
  return hexToRgb(st[st.length - 1].color);
}

export function cssGradient(s: Setting, nativeHex: string, dir = "90deg"): string {
  if (s.mode === "native") return nativeHex;
  if (s.mode === "solid") return s.color;
  const st = sortedStops(s.stops);
  return `linear-gradient(${dir}, ${st.map((x) => `${x.color} ${(x.pos * 100).toFixed(1)}%`).join(", ")})`;
}

// Representative single colour (for swatches and the UI accent).
export function keyColor(s: Setting | undefined, nativeHex: string): string {
  if (!s || s.mode === "native") return nativeHex;
  if (s.mode === "solid") return s.color;
  return rgbToHex(colorAt(s, 0.5, nativeHex));
}

// Readable text on top of a colour.
export function inkFor(hex: string): string {
  const [r, g, b] = hexToRgb(hex);
  const l = 0.2126 * r + 0.7152 * g + 0.0722 * b;
  return l > 0.55 ? "#10121f" : "#f4f5fb";
}
