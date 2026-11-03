import { hexToRgb, hsvToRgb, rgbToHex, rgbToHsv } from "./color";

const PRESETS = ["#ffffff", "#00e5ff", "#1f6bff", "#7c5cff", "#ff3dd2", "#ff2a4a", "#ff8a1f", "#ffd23f", "#3dff8a"];

// HSV square + hue strip + hex field. Emits on every drag step.
export class ColorPicker {
  readonly el: HTMLElement;
  private sv: HTMLElement;
  private svKnob: HTMLElement;
  private hue: HTMLInputElement;
  private hex: HTMLInputElement;
  private h = 0;
  private s = 1;
  private v = 1;

  constructor(private onChange: (hex: string) => void) {
    this.el = document.createElement("div");
    this.el.className = "picker";
    this.el.innerHTML = `
      <div class="sv" tabindex="0" aria-label="Saturation and brightness"><span class="knob"></span></div>
      <div class="picker-side">
        <input class="hue" type="range" min="0" max="360" step="1" aria-label="Hue">
        <label class="hex-row"><span class="chip"></span><input class="hex" spellcheck="false" maxlength="7" aria-label="Hex colour"></label>
        <div class="presets">${PRESETS.map((c) => `<button class="preset" style="--c:${c}" data-c="${c}" aria-label="Use ${c}"></button>`).join("")}</div>
      </div>`;
    this.sv = this.el.querySelector(".sv")!;
    this.svKnob = this.el.querySelector(".knob")!;
    this.hue = this.el.querySelector(".hue")!;
    this.hex = this.el.querySelector(".hex")!;

    const fromPointer = (e: PointerEvent) => {
      const r = this.sv.getBoundingClientRect();
      this.s = Math.max(0, Math.min(1, (e.clientX - r.left) / r.width));
      this.v = 1 - Math.max(0, Math.min(1, (e.clientY - r.top) / r.height));
      this.emit();
    };
    this.sv.addEventListener("pointerdown", (e) => {
      this.sv.setPointerCapture(e.pointerId);
      fromPointer(e);
      const move = (ev: PointerEvent) => fromPointer(ev);
      const up = () => { this.sv.removeEventListener("pointermove", move); this.sv.removeEventListener("pointerup", up); };
      this.sv.addEventListener("pointermove", move);
      this.sv.addEventListener("pointerup", up);
    });
    this.sv.addEventListener("keydown", (e) => {
      const step = e.shiftKey ? 0.1 : 0.02;
      const k: Record<string, () => void> = {
        ArrowLeft: () => (this.s -= step), ArrowRight: () => (this.s += step),
        ArrowUp: () => (this.v += step), ArrowDown: () => (this.v -= step),
      };
      if (k[e.key]) {
        e.preventDefault();
        k[e.key]();
        this.s = Math.max(0, Math.min(1, this.s));
        this.v = Math.max(0, Math.min(1, this.v));
        this.emit();
      }
    });
    this.hue.addEventListener("input", () => { this.h = +this.hue.value; this.emit(); });
    this.hex.addEventListener("change", () => {
      const v = this.hex.value.trim();
      const norm = v.startsWith("#") ? v : `#${v}`;
      if (/^#[0-9a-f]{6}$/i.test(norm)) { this.set(norm); this.onChange(norm.toLowerCase()); }
      else this.paint();
    });
    this.el.querySelectorAll<HTMLButtonElement>(".preset").forEach((b) =>
      b.addEventListener("click", () => { this.set(b.dataset.c!); this.onChange(b.dataset.c!); }));
  }

  set(hex: string) {
    [this.h, this.s, this.v] = rgbToHsv(hexToRgb(hex));
    this.paint();
  }

  private current() { return rgbToHex(hsvToRgb(this.h, this.s, this.v)); }

  private emit() { this.paint(); this.onChange(this.current()); }

  private paint() {
    const hex = this.current();
    this.sv.style.setProperty("--hue", rgbToHex(hsvToRgb(this.h, 1, 1)));
    this.svKnob.style.left = `${this.s * 100}%`;
    this.svKnob.style.top = `${(1 - this.v) * 100}%`;
    this.svKnob.style.background = hex;
    this.hue.value = String(Math.round(this.h));
    if (document.activeElement !== this.hex) this.hex.value = hex;
    (this.el.querySelector(".chip") as HTMLElement).style.background = hex;
  }
}
