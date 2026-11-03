import type { Setting } from "./api";
import { colorAt } from "./color";

// One bar per layer showing exactly what's written to the game: the colour
// across the effect's life (left = start, right = fade), tone-mapped the way
// HDR colour blooms in game, so bright greens read as green, not mud.

export interface Row { key: string; label: string; setting: Setting; focus: boolean; inGame: boolean }

const SAMPLES = 96;

function tone(v: number): number {
  return 1 - Math.exp(-v * 0.42);
}

// HDR rgb -> display rgb, with a little white bloom at high brightness
function display(r: number, g: number, b: number): [number, number, number] {
  const peak = Math.max(r, g, b);
  const bloom = Math.max(0, Math.min(0.5, (peak - 8) / 20));
  const f = (c: number) => Math.round((tone(c) * (1 - bloom) + bloom) * 255);
  return [f(r), f(g), f(b)];
}

export class LifeStrips {
  private rows: Row[] = [];

  constructor(private root: HTMLElement, private onPick: (key: string) => void) {
    new ResizeObserver(() => this.paint()).observe(root);
    root.addEventListener("click", (e) => {
      const r = (e.target as HTMLElement).closest<HTMLElement>("[data-key]");
      if (r) this.onPick(r.dataset.key!);
    });
  }

  set(rows: Row[]) {
    const same = rows.length === this.rows.length && rows.every((r, i) => r.key === this.rows[i].key);
    this.rows = rows;
    if (!same) {
      this.root.innerHTML = rows.map((r) => `
        <button class="strip" data-key="${r.key}" type="button" role="radio">
          <span class="strip-label"><span class="strip-name"></span><span class="ingame"></span></span>
          <span class="strip-track"><canvas></canvas><span class="strip-native">Game colours</span></span>
        </button>`).join("");
    }
    this.paint();
  }

  private paint() {
    const els = this.root.querySelectorAll<HTMLElement>(".strip");
    this.rows.forEach((row, i) => {
      const el = els[i];
      if (!el) return;
      el.classList.toggle("on", row.focus);
      el.classList.toggle("native", row.setting.mode === "native");
      el.setAttribute("aria-checked", String(row.focus));
      el.querySelector(".strip-name")!.textContent = row.label;
      const dot = el.querySelector<HTMLElement>(".ingame")!;
      dot.classList.toggle("yes", row.inGame);
      dot.title = row.inGame ? "Loaded in the game right now" : "Not loaded yet. Pick this fighter in a match and it will apply.";
      const canvas = el.querySelector("canvas")!;
      const w = canvas.clientWidth, h = canvas.clientHeight;
      if (!w || !h) return;
      const dpr = window.devicePixelRatio || 1;
      canvas.width = Math.round(w * dpr);
      canvas.height = Math.round(h * dpr);
      const ctx = canvas.getContext("2d")!;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, w, h);
      if (row.setting.mode === "native") return;

      const k = Math.max(0.5, row.setting.brightness);
      const step = w / SAMPLES;
      for (let s = 0; s < SAMPLES; s++) {
        const t = s / (SAMPLES - 1);
        const [r, g, b] = colorAt(row.setting, t, "#ffffff");
        const [dr, dg, db] = display(r * k, g * k, b * k);
        ctx.fillStyle = `rgb(${dr},${dg},${db})`;
        ctx.fillRect(Math.floor(s * step), 0, Math.ceil(step) + 1, h);
      }
      // soft glow on the bright core line, like the beam's hot centre
      const grad = ctx.createLinearGradient(0, 0, 0, h);
      grad.addColorStop(0, "rgba(0,0,0,.35)");
      grad.addColorStop(0.5, "rgba(255,255,255,.10)");
      grad.addColorStop(1, "rgba(0,0,0,.35)");
      ctx.fillStyle = grad;
      ctx.fillRect(0, 0, w, h);
    });
  }
}
