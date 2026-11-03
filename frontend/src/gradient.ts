import type { Stop } from "./api";
import { colorAt, rgbToHex, sortedStops } from "./color";

const MAX_STOPS = 6;

// Gradient bar with draggable colour stops. Click the bar to add a stop.
export class GradientEditor {
  readonly el: HTMLElement;
  private bar: HTMLElement;
  private stops: Stop[] = [];
  selected = 0;

  constructor(private onChange: (stops: Stop[]) => void, private onSelect: (i: number) => void) {
    this.el = document.createElement("div");
    this.el.className = "gradient";
    this.el.innerHTML = `
      <div class="grad-bar" title="Click to add a colour stop"></div>
      <div class="grad-foot">
        <span class="grad-hint">Left is the start of the electric, right is where it fades out.</span>
        <button class="ghost small grad-remove" type="button">Remove stop</button>
      </div>`;
    this.bar = this.el.querySelector(".grad-bar")!;
    this.bar.addEventListener("pointerdown", (e) => {
      if ((e.target as HTMLElement).classList.contains("stop")) return;
      if (this.stops.length >= MAX_STOPS) return;
      const pos = this.posFromEvent(e);
      const color = rgbToHex(colorAt({ mode: "gradient", stops: this.stops, color: "", brightness: 1, keepFade: true }, pos, "#ffffff"));
      this.stops.push({ pos, color });
      this.selected = this.stops.length - 1;
      this.render();
      this.onChange(this.copy());
      this.onSelect(this.selected);
    });
    this.el.querySelector(".grad-remove")!.addEventListener("click", () => this.remove());
  }

  private copy(): Stop[] { return this.stops.map((s) => ({ ...s })); }

  private posFromEvent(e: PointerEvent) {
    const r = this.bar.getBoundingClientRect();
    return Math.round(Math.max(0, Math.min(1, (e.clientX - r.left) / r.width)) * 1000) / 1000;
  }

  set(stops: Stop[]) {
    this.stops = stops.map((s) => ({ ...s }));
    if (this.selected >= this.stops.length) this.selected = 0;
    this.render();
  }

  setSelectedColor(color: string) {
    if (!this.stops[this.selected]) return;
    this.stops[this.selected].color = color;
    this.render();
    this.onChange(this.copy());
  }

  selectedColor() { return this.stops[this.selected]?.color ?? "#ffffff"; }

  private remove() {
    if (this.stops.length <= 2) return;
    this.stops.splice(this.selected, 1);
    this.selected = 0;
    this.render();
    this.onChange(this.copy());
    this.onSelect(0);
  }

  private render() {
    const st = sortedStops(this.stops);
    this.bar.style.background = `linear-gradient(90deg, ${st.map((s) => `${s.color} ${s.pos * 100}%`).join(", ")})`;
    this.bar.querySelectorAll(".stop").forEach((n) => n.remove());
    this.stops.forEach((s, i) => {
      const b = document.createElement("button");
      b.type = "button";
      b.className = "stop" + (i === this.selected ? " on" : "");
      b.style.left = `${s.pos * 100}%`;
      b.style.setProperty("--c", s.color);
      b.setAttribute("aria-label", `Colour stop at ${Math.round(s.pos * 100)}%`);
      b.addEventListener("pointerdown", (e) => {
        e.stopPropagation();
        this.selected = i;
        this.onSelect(i);
        this.render();
        const handle = this.bar.querySelectorAll<HTMLElement>(".stop")[i];
        handle.setPointerCapture(e.pointerId);
        const move = (ev: PointerEvent) => {
          this.stops[i].pos = this.posFromEvent(ev);
          handle.style.left = `${this.stops[i].pos * 100}%`;
          const s2 = sortedStops(this.stops);
          this.bar.style.background = `linear-gradient(90deg, ${s2.map((s) => `${s.color} ${s.pos * 100}%`).join(", ")})`;
          this.onChange(this.copy());
        };
        const up = () => { handle.removeEventListener("pointermove", move); handle.removeEventListener("pointerup", up); };
        handle.addEventListener("pointermove", move);
        handle.addEventListener("pointerup", up);
      });
      b.addEventListener("keydown", (e) => {
        const d = e.key === "ArrowLeft" ? -0.01 : e.key === "ArrowRight" ? 0.01 : 0;
        if (d) {
          e.preventDefault();
          this.stops[i].pos = Math.max(0, Math.min(1, this.stops[i].pos + d * (e.shiftKey ? 5 : 1)));
          this.render();
          this.onChange(this.copy());
          (this.bar.querySelectorAll<HTMLElement>(".stop")[i]).focus();
        } else if (e.key === "Delete" || e.key === "Backspace") {
          this.selected = i;
          this.remove();
        }
      });
      this.bar.appendChild(b);
    });
    (this.el.querySelector(".grad-remove") as HTMLButtonElement).disabled = this.stops.length <= 2;
  }
}
