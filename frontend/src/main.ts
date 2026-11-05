import "@fontsource/unbounded/500.css";
import "@fontsource/unbounded/700.css";
import "@fontsource/manrope/400.css";
import "@fontsource/manrope/600.css";
import "@fontsource/manrope/700.css";
import "./style.css";

import { api, errText, keyOf, onEdition, onPak, onStatus, onUpdate, type Character, type FinderView, type PakStatus, type Edition, type Mode, type Setting, type State, type Status } from "./api";
import { cssGradient, inkFor, keyColor } from "./color";
import { GradientEditor } from "./gradient";
import { ColorPicker } from "./picker";
import { NATIVE } from "./preview";
import { LifeStrips } from "./strips";

const $ = <T extends HTMLElement = HTMLElement>(sel: string, root: ParentNode = document) => root.querySelector<T>(sel)!;

let catalog: Character[] = [];
let state: State = { active: "", profiles: [], parts: {} };
let status: Status | null = null;
let defaults: Setting;
let pakStatus: PakStatus = { state: "off", message: "" };
const sel = { char: "", effect: "", part: "" };
// Electrics (live colour curves) and mod effects (baked into a pak, need a
// Tekken restart) live in separate tabs so it's clear which is which.
type Section = "live" | "pak";
let section: Section = "live";
const kindOf = (e: { kind?: string }): Section => (e.kind === "pak" ? "pak" : "live");
const effectsOf = (c: Character) => c.effects.filter((e) => kindOf(e) === section);
const visibleChars = () => catalog.filter((c) => effectsOf(c).length > 0);

// ------------------------------------------------------------------ shell

document.querySelector("#app")!.innerHTML = `
  <header class="top">
    <div class="brand"><span class="mark" aria-hidden="true"></span><span class="brand-name">Electric Editor</span></div>
    <span class="greeting" hidden></span>
    <span class="badge" hidden></span>
    <div class="conn" role="status"><span class="dot"></span><span class="conn-text">Looking for Tekken…</span></div>
    <div class="sections" role="tablist" aria-label="Effect type">
      <button role="tab" data-section="live" aria-selected="true">Electrics <small>live</small></button>
      <button role="tab" data-section="pak" aria-selected="false">Mod effects <small>restart</small></button>
    </div>
    <div class="update" hidden>
      <span class="update-text"></span>
      <button class="primary small" id="restart">Restart now</button>
    </div>
    <div class="top-right">
      <label class="profile">
        <span class="visually-hidden">Profile</span>
        <select id="profile"></select>
      </label>
      <button class="ghost" id="profile-menu" aria-haspopup="menu">Manage</button>
      <div class="menu" id="menu" role="menu" hidden>
        <button role="menuitem" data-act="new">New profile</button>
        <button role="menuitem" data-act="rename">Rename</button>
        <button role="menuitem" data-act="delete" class="danger">Delete</button>
        <hr>
        <button role="menuitem" data-act="key" class="keys-on">Enter a key…</button>
        <button role="menuitem" data-act="key-remove" class="key-remove" hidden></button>
        <hr class="keys-on">
        <button role="menuitem" data-act="bg">Set background image…</button>
        <button role="menuitem" data-act="bg-clear">Remove background image</button>
        <hr>
        <button role="menuitem" data-act="dump" class="dev-only">Save curve list…</button>
        <button role="menuitem" data-act="finder" class="dev-only">Find effects…</button>
        <button role="menuitem" data-act="export" class="dev-only">Export game files…</button>
        <button role="menuitem" data-act="testmod" class="dev-only">Test mod…</button>
        <span class="menu-version"></span>
      </div>
      <button class="primary" id="share">Share</button>
    </div>
  </header>
  <nav class="rail" aria-label="Fighters"></nav>
  <main class="stage">
    <div class="stage-head">
      <h1 class="fighter-name"></h1>
      <div class="effects" role="tablist" aria-label="Effect"></div>
    </div>
    <figure class="life">
      <div class="strips" role="radiogroup" aria-label="Layer"></div>
      <figcaption class="cap-live">What each layer sends to the game, from the start of the electric (left) to where it fades out (right). Click a bar to edit it.</figcaption>
      <figcaption class="cap-pak"><span class="effect-note"></span> These colours are built into a small mod file in Tekken's <code>Paks\\mods</code> folder, so they show the next time Tekken starts. Click a bar to edit it.</figcaption>
    </figure>
    <div class="pak-banner" role="status" hidden>
      <span class="pak-dot" aria-hidden="true"></span>
      <span class="pak-text"></span>
      <button class="ghost small" id="pak-folder" hidden>Open log folder</button>
    </div>
    <section class="editor">
      <div class="editor-head">
        <h2 class="layer-name"></h2>
        <div class="modes" role="radiogroup" aria-label="Colour mode">
          <button role="radio" data-mode="native">Game colours</button>
          <button role="radio" data-mode="solid">Solid</button>
          <button role="radio" data-mode="gradient" class="live-only">Gradient</button>
        </div>
      </div>
      <div class="editor-body">
        <div class="native-note">This layer uses Tekken's own colours. Pick Solid or Gradient to recolour it.</div>
        <div class="grad-slot"></div>
        <div class="picker-slot"></div>
      </div>
      <div class="tuning">
        <label class="slider">
          <span>Brightness</span>
          <input type="range" id="brightness" min="0.5" max="20" step="0.1">
          <output id="brightness-out"></output>
        </label>
        <label class="switch live-only">
          <input type="checkbox" id="keepfade">
          <span class="track" aria-hidden="true"></span>
          <span>Keep the original fade</span>
        </label>
      </div>
      <div class="editor-actions">
        <button class="ghost" id="copy-all">Use on every layer</button>
        <button class="ghost" id="reset">Reset to game colours</button>
      </div>
    </section>
  </main>

  <dialog id="share-dlg" aria-labelledby="share-title">
    <form method="dialog" class="dlg">
      <h2 id="share-title">Share colours</h2>
      <p class="muted">Codes work in anyone's Electric Editor. Paste one into Discord or a mod page.</p>
      <fieldset class="scope">
        <label><input type="radio" name="scope" value="char" checked> <span class="scope-char"></span></label>
        <label><input type="radio" name="scope" value="all"> Whole profile</label>
      </fieldset>
      <div class="code-row">
        <textarea id="code-out" readonly rows="3" aria-label="Share code"></textarea>
        <button type="button" class="primary" id="copy-code">Copy code</button>
      </div>
      <h3>Import a code</h3>
      <div class="code-row">
        <textarea id="code-in" rows="3" placeholder="Paste a code starting with EE1." aria-label="Code to import"></textarea>
        <button type="button" class="ghost" id="import-code">Import</button>
      </div>
      <p class="form-error" id="share-error" role="alert"></p>
      <div class="dlg-actions"><button class="ghost" value="close">Done</button></div>
    </form>
  </dialog>

  <dialog id="name-dlg" aria-labelledby="name-title">
    <form method="dialog" class="dlg small">
      <h2 id="name-title"></h2>
      <input id="name-in" maxlength="40" autocomplete="off" aria-labelledby="name-title">
      <p class="form-error" id="name-error" role="alert"></p>
      <div class="dlg-actions">
        <button class="ghost" value="cancel" formnovalidate>Cancel</button>
        <button class="primary" value="ok" id="name-ok"></button>
      </div>
    </form>
  </dialog>
  <dialog id="key-dlg" aria-labelledby="key-title">
    <form method="dialog" class="dlg small">
      <h2 id="key-title">Enter a key</h2>
      <p class="muted">Keys unlock extras like supporter badges. Paste the whole key, starting with EE-.</p>
      <textarea id="key-in" rows="3" spellcheck="false" placeholder="EE-…" aria-labelledby="key-title"></textarea>
      <p class="form-error" id="key-error" role="alert"></p>
      <div class="dlg-actions">
        <button class="ghost" value="cancel" formnovalidate>Cancel</button>
        <button class="primary" value="ok" id="key-ok">Activate key</button>
      </div>
    </form>
  </dialog>
  <dialog id="finder-dlg" aria-labelledby="finder-title">
    <div class="dlg finder">
      <div class="finder-top">
        <h2 id="finder-title">Effect finder</h2>
        <button class="ghost small" id="finder-close" aria-label="Close">Close</button>
      </div>
      <div class="finder-setup">
        <p class="muted">Gives every effect of a fighter a test colour. Do the move, say which colour it turned, and it narrows down until each colour is one effect. Load a match with the fighter and do the moves once before starting.</p>
        <label class="field">
          <span>Fighter</span>
          <select id="finder-scope">
            <option value="/character/grl/">Kazuya (grl)</option>
            <option value="/character/ant/">Jin (ant)</option>
            <option value="/character/swl/">Devil Jin (swl)</option>
            <option value="/character/zbr/">Reina (zbr)</option>
            <option value="/character/bee/">Heihachi (bee)</option>
            <option value="custom">Other path…</option>
          </select>
        </label>
        <label class="field finder-custom" hidden>
          <span>Path contains</span>
          <input id="finder-custom" placeholder="/effect/polariswork/common/" spellcheck="false">
        </label>
        <fieldset class="scope finder-kind">
          <label><input type="radio" name="finder-kind" value="live" checked> <span><b>Colour curves</b><br><small class="muted">Instant. Electrics, beams, most particles.</small></span></label>
          <label><input type="radio" name="finder-kind" value="pak"> <span><b>Materials</b><br><small class="muted">Needs a Tekken restart each round. Trails, meshes, shockwaves.</small></span></label>
        </fieldset>
        <div class="dlg-actions"><button class="primary" id="finder-start">Start</button></div>
      </div>
      <div class="finder-run" hidden>
        <p class="finder-round"></p>
        <p class="finder-pak" role="status"></p>
        <p class="finder-ask">Do the move. What colour did the effect turn?</p>
        <div class="finder-colors"></div>
        <ol class="finder-legend"></ol>
        <p class="muted small-note">Effect stayed its normal colour? It isn't in this list: try the other mode, or another path.</p>
        <p class="form-error" id="finder-error" role="alert"></p>
        <div class="dlg-actions">
          <button class="ghost" id="finder-back">Back a round</button>
          <button class="ghost" id="finder-legend">Open finder.txt</button>
          <button class="ghost danger" id="finder-stop">Stop and restore colours</button>
        </div>
      </div>
    </div>
  </dialog>
  <dialog id="export-dlg" aria-labelledby="export-title">
    <form method="dialog" class="dlg">
      <h2 id="export-title">Export game files</h2>
      <p class="muted">Pulls the original files out of Tekken into research.zip, with readable names. One filter per line; a file is included if its path contains any of them. Filters are case-sensitive.</p>
      <textarea id="export-in" rows="7" spellcheck="false" aria-labelledby="export-title"></textarea>
      <p class="form-error" id="export-error" role="alert"></p>
      <div class="dlg-actions">
        <button class="ghost" value="cancel" formnovalidate>Cancel</button>
        <button class="primary" value="ok" id="export-ok">Export</button>
      </div>
    </form>
  </dialog>
  <dialog id="test-dlg" aria-labelledby="test-title">
    <form method="dialog" class="dlg">
      <h2 id="test-title">Test mod</h2>
      <p class="muted">Builds ElectricEditorTest_P from raw material values, for experiments. One material per line, then Param=value (1 number) or Param=r,g,b,a.</p>
      <textarea id="test-in" rows="6" spellcheck="false" aria-labelledby="test-title"></textarea>
      <p class="finder-pak" id="test-status" role="status"></p>
      <div class="dlg-actions">
        <button class="ghost" value="close" formnovalidate>Close</button>
        <button class="ghost danger" type="button" id="test-remove">Remove test mod</button>
        <button class="primary" type="button" id="test-build">Build and install</button>
      </div>
    </form>
  </dialog>
  <div class="lock" hidden>
    <form class="lock-card">
      <span class="mark" aria-hidden="true"></span>
      <h1>Supporter beta</h1>
      <p>This is an early build of the next update. It's open to supporters while it's being finished, and everyone gets it when it's released.</p>
      <p class="muted">Got a supporter key? Paste it here. It starts with EE-.</p>
      <textarea id="lock-key" rows="3" spellcheck="false" placeholder="EE-…" aria-label="Supporter key"></textarea>
      <p class="form-error" id="lock-error" role="alert"></p>
      <button class="primary" id="lock-ok">Unlock</button>
      <span class="lock-version"></span>
    </form>
  </div>
  <div class="toast" role="status" aria-live="polite"></div>
`;

const strips = new LifeStrips($(".strips"), (key) => {
  sel.part = key.split(".")[2];
  grad.selected = 0;
  renderLayers();
  renderAccents();
  renderEditor();
});
const picker = new ColorPicker((hex) => {
  const s = current();
  if (s.mode === "gradient") grad.setSelectedColor(hex);
  else update({ color: hex });
});
const grad = new GradientEditor(
  (stops) => update({ stops }),
  () => picker.set(grad.selectedColor()),
);
$(".picker-slot").appendChild(picker.el);
$(".grad-slot").appendChild(grad.el);

// ------------------------------------------------------------------ helpers

const char = () => catalog.find((c) => c.id === sel.char)!;
const effect = () => char().effects.find((e) => e.id === sel.effect)!;
const part = () => effect().parts.find((p) => p.id === sel.part)!;
const curKey = () => keyOf(sel.char, sel.effect, sel.part);
const settingFor = (key: string): Setting => {
  if (state.parts[key]) return state.parts[key];
  const [c, e] = key.split(".");
  const k = catalog.find((x) => x.id === c)?.effects.find((x) => x.id === e)?.kind;
  const pak = k === "pak" || k === "rgb";
  // mod effects use brightness as a multiplier of the game's own glow (1 = unchanged)
  return { ...structuredClone(defaults), mode: "native", ...(pak ? { brightness: 1 } : {}) };
};
const current = () => settingFor(curKey());
const isPak = () => effect().kind === "pak";
// solid-colour-only effects: mod effects and live float-curve colour
const isSimple = () => effect().kind === "pak" || effect().kind === "rgb";

function toast(msg: string) {
  const t = $(".toast");
  t.textContent = msg;
  t.classList.add("show");
  clearTimeout((t as unknown as { _h: number })._h);
  (t as unknown as { _h: number })._h = window.setTimeout(() => t.classList.remove("show"), 2600);
}

// Save is debounced while dragging; local state updates instantly.
let saveTimer = 0;
function update(patch: Partial<Setting>) {
  const key = curKey();
  const next = { ...current(), ...patch };
  state.parts[key] = next;
  if (next.mode === "native") delete state.parts[key];
  clearTimeout(saveTimer);
  saveTimer = window.setTimeout(() => api.SetPart(key, next).catch((e) => toast(errText(e))), 60);
  renderAccents();
  renderEditor(false);
  renderLayers();
  renderRail();
}

// ------------------------------------------------------------------ render

function renderRail() {
  const rail = $(".rail");
  rail.innerHTML = visibleChars().map((c) => {
    const first = effectsOf(c)[0];
    const swatches = first.parts.slice(0, 4).map((p) => {
      const s = state.parts[keyOf(c.id, first.id, p.id)];
      return `<i style="background:${s ? cssGradient(s, NATIVE[p.role], "180deg") : "transparent"}" class="${s ? "" : "empty"}"></i>`;
    }).join("");
    return `<button class="fighter${c.id === sel.char ? " on" : ""}" data-char="${c.id}" aria-current="${c.id === sel.char}">
      <span class="fighter-label">${c.name}</span><span class="swatches" aria-hidden="true">${swatches}</span></button>`;
  }).join("");
}

function renderHead() {
  $(".fighter-name").textContent = char().name;
  const tabs = $(".effects");
  tabs.innerHTML = effectsOf(char()).map((e) =>
    `<button role="tab" aria-selected="${e.id === sel.effect}" data-effect="${e.id}">${e.label}</button>`).join("");
  tabs.hidden = effectsOf(char()).length < 2;
  document.querySelectorAll<HTMLElement>(".sections [data-section]").forEach((b) =>
    b.setAttribute("aria-selected", String(b.dataset.section === section)));
}

function renderLayers() {
  const loaded = status?.loaded ?? {};
  strips.set(effect().parts.map((p) => {
    const key = keyOf(sel.char, sel.effect, p.id);
    return { key, label: p.label, setting: settingFor(key), focus: p.id === sel.part, inGame: !!loaded[key], flat: isSimple() };
  }));
}

function renderEditor(syncPicker = true) {
  const s = current();
  $(".layer-name").textContent = part().label;
  document.querySelectorAll<HTMLButtonElement>(".modes button").forEach((b) => {
    b.setAttribute("aria-checked", String(b.dataset.mode === s.mode));
  });
  const body = $(".editor");
  body.dataset.view = s.mode;
  const br = $<HTMLInputElement>("#brightness");
  const pak = isSimple();
  document.body.classList.toggle("rgb", effect().kind === "rgb");
  document.body.classList.toggle("pak", pak);
  br.min = pak ? "0.2" : "0.5";
  br.max = pak ? "4" : "20";
  br.step = pak ? "0.05" : "0.1";
  br.value = String(s.brightness);
  $(".native-note").textContent = pak
    ? "This layer uses Tekken's own colours. Pick Solid to recolour it."
    : "This layer uses Tekken's own colours. Pick Solid or Gradient to recolour it.";
  $(".effect-note").textContent = effect().note ?? "";
  $("#brightness-out").textContent = pak ? `${s.brightness.toFixed(2)}×` : s.brightness.toFixed(1);
  $<HTMLInputElement>("#keepfade").checked = s.keepFade;
  if (syncPicker) {
    if (s.mode === "gradient") { grad.set(s.stops); picker.set(grad.selectedColor()); }
    else picker.set(s.color);
  }
}

function renderAccents() {
  const s = current();
  const nativeAccent = getComputedStyle(document.body).getPropertyValue("--native-accent").trim() || NATIVE[part().role];
  const accent = s.mode === "native" ? nativeAccent : keyColor(s, NATIVE[part().role]);
  document.documentElement.style.setProperty("--accent", accent);
  document.documentElement.style.setProperty("--accent-ink", inkFor(accent));
}

function renderProfiles() {
  $<HTMLSelectElement>("#profile").innerHTML = state.profiles
    .map((p) => `<option ${p === state.active ? "selected" : ""}>${p.replace(/</g, "&lt;")}</option>`).join("");
}

function renderStatus() {
  if (!status) return;
  const c = $(".conn");
  c.dataset.state = status.state;
  const short: Record<string, string> = {
    connected: status.curves ? "Connected to Tekken" : "Connected, waiting for a match",
    waiting: "Waiting for Tekken",
    connecting: "Connecting…",
    error: "Can't reach Tekken",
  };
  $(".conn-text").textContent = short[status.state] ?? status.message;
  c.title = status.message;
}

function renderAll() {
  renderRail();
  renderHead();
  renderLayers();
  renderAccents();
  renderEditor();
  renderProfiles();
  renderStatus();
  renderPak();
}

function select(charId: string, effectId?: string, partId?: string) {
  sel.char = charId;
  const c = char();
  sel.effect = effectId ?? effectsOf(c)[0].id;
  sel.part = partId ?? effect().parts[0].id;
  grad.selected = 0;
  renderAll();
}

// ------------------------------------------------------------------ events

$(".sections").addEventListener("click", (e) => {
  const b = (e.target as HTMLElement).closest<HTMLElement>("[data-section]");
  if (!b || b.dataset.section === section) return;
  section = b.dataset.section as Section;
  const keep = visibleChars().find((c) => c.id === sel.char);
  select((keep ?? visibleChars()[0]).id);
});
$(".rail").addEventListener("click", (e) => {
  const b = (e.target as HTMLElement).closest<HTMLElement>("[data-char]");
  if (b) select(b.dataset.char!);
});
$(".effects").addEventListener("click", (e) => {
  const b = (e.target as HTMLElement).closest<HTMLElement>("[data-effect]");
  if (b) select(sel.char, b.dataset.effect!);
});
$(".modes").addEventListener("click", (e) => {
  const b = (e.target as HTMLElement).closest<HTMLElement>("[data-mode]");
  if (!b) return;
  const mode = b.dataset.mode as Mode;
  const s = current();
  const patch: Partial<Setting> = { mode };
  if (mode !== "native" && s.mode === "native") {
    patch.color = s.color || defaults.color;
    patch.stops = s.stops?.length ? s.stops : defaults.stops;
    if (isSimple()) patch.brightness = 1; // 1x = same glow strength as the original
  }
  update(patch);
  renderEditor();
});
$("#brightness").addEventListener("input", (e) => update({ brightness: +(e.target as HTMLInputElement).value }));
$("#keepfade").addEventListener("change", (e) => update({ keepFade: (e.target as HTMLInputElement).checked }));
$("#reset").addEventListener("click", () => { update({ mode: "native" }); renderEditor(); });
$("#copy-all").addEventListener("click", async () => {
  const s = current();
  const keys = effect().parts.map((p) => keyOf(sel.char, sel.effect, p.id));
  for (const k of keys) state.parts[k] = structuredClone(s);
  if (s.mode === "native") for (const k of keys) delete state.parts[k];
  try { await api.SetMany(keys, s); toast(`Applied to all ${keys.length} layers`); } catch (e) { toast(errText(e)); }
  renderAll();
});

// profiles
$("#profile").addEventListener("change", async (e) => {
  try { state = await api.SwitchProfile((e.target as HTMLSelectElement).value); renderAll(); }
  catch (err) { toast(errText(err)); }
});
const menu = $("#menu");
$("#profile-menu").addEventListener("click", () => { menu.hidden = !menu.hidden; });
document.addEventListener("click", (e) => {
  if (!(e.target as HTMLElement).closest("#menu, #profile-menu")) menu.hidden = true;
});
function askName(title: string, ok: string, value: string, run: (n: string) => Promise<State>) {
  const dlg = $<HTMLDialogElement>("#name-dlg");
  $("#name-title").textContent = title;
  $("#name-ok").textContent = ok;
  $("#name-error").textContent = "";
  const inp = $<HTMLInputElement>("#name-in");
  inp.value = value;
  dlg.showModal();
  inp.select();
  dlg.onclose = null;
  $<HTMLFormElement>("form", dlg).onsubmit = async (ev) => {
    if ((ev.submitter as HTMLButtonElement)?.value !== "ok") return;
    ev.preventDefault();
    try { state = await run(inp.value); dlg.close(); renderAll(); }
    catch (err) { $("#name-error").textContent = errText(err); }
  };
}
menu.addEventListener("click", async (e) => {
  const act = (e.target as HTMLElement).closest<HTMLElement>("[data-act]")?.dataset.act;
  menu.hidden = true;
  if (act === "new") askName("New profile", "Create profile", "", (n) => api.CreateProfile(n));
  if (act === "rename") askName("Rename profile", "Rename", state.active, (n) => api.RenameProfile(n));
  if (act === "key") {
    const dlg = $<HTMLDialogElement>("#key-dlg");
    $("#key-error").textContent = "";
    $<HTMLTextAreaElement>("#key-in").value = "";
    dlg.showModal();
    $<HTMLFormElement>("form", dlg).onsubmit = async (ev) => {
      if ((ev.submitter as HTMLButtonElement)?.value !== "ok") return;
      ev.preventDefault();
      try {
        const ed = await api.ActivateKey($<HTMLTextAreaElement>("#key-in").value);
        applyEdition(ed);
        dlg.close();
        toast(ed.badge ? `Key activated: ${ed.badge}` : ed.greeting ? `${ed.greeting}!` : "Key activated");
      } catch (err) { $("#key-error").textContent = errText(err); }
    };
  }
  if (act === "key-remove") {
    if (!confirm("Remove your key from this PC? You can enter it again any time.")) return;
    applyEdition(await api.RemoveKey());
    toast("Key removed");
  }
  if (act === "bg") {
    try { setBackground(await api.ChooseBackground()); } catch (err) { toast(errText(err)); }
  }
  if (act === "bg-clear") { await api.ClearBackground(); setBackground(""); }
  if (act === "dump") {
    try { const path = await api.DumpCurves(); toast(`Saved ${path.split(/[\\/]/).pop()}`); }
    catch (err) { toast(errText(err)); }
  }
  if (act === "finder") openFinder();
  if (act === "testmod") {
    const dlg = $<HTMLDialogElement>("#test-dlg");
    const inp = $<HTMLTextAreaElement>("#test-in");
    inp.value = inp.value || "DA_SWL_KisekiRender @20=fdffffff\nDA_SWL_KisekiRender3 @12=fdffffff @20=fdffffff";
    const show = (s: { state: string; message: string }) => {
      const el = $("#test-status");
      el.dataset.state = s.state;
      el.textContent = s.message || "No test mod installed.";
    };
    show(await api.TestModStatus());
    dlg.showModal();
    const poll = window.setInterval(async () => { if (!dlg.open) return clearInterval(poll); show(await api.TestModStatus()); }, 1200);
    $("#test-build").onclick = async () => { try { show(await api.BuildTestMod(inp.value)); } catch (err) { show({ state: "error", message: errText(err) }); } };
    $("#test-remove").onclick = async () => { show(await api.BuildTestMod("")); };
  }
  if (act === "export") {
    const dlg = $<HTMLDialogElement>("#export-dlg");
    const inp = $<HTMLTextAreaElement>("#export-in");
    if (!inp.value) inp.value = ["/Character/Trace/", "DA_CharacterEffect_", "_Groundwave", "MI_EF_Swl_S_Chain"].join("\n");
    $("#export-error").textContent = "";
    dlg.showModal();
    $<HTMLFormElement>("form", dlg).onsubmit = async (ev) => {
      if ((ev.submitter as HTMLButtonElement)?.value !== "ok") return;
      ev.preventDefault();
      const ok = $<HTMLButtonElement>("#export-ok");
      ok.disabled = true;
      ok.textContent = "Exporting…";
      try { toast(await api.ExportAssets(inp.value)); dlg.close(); }
      catch (err) { $("#export-error").textContent = errText(err); }
      finally { ok.disabled = false; ok.textContent = "Export"; }
    };
  }

  if (act === "delete") {
    if (!confirm(`Delete the profile "${state.active}"?`)) return;
    try { state = await api.DeleteProfile(); renderAll(); toast("Profile deleted"); } catch (err) { toast(errText(err)); }
  }
});

// share
const shareDlg = $<HTMLDialogElement>("#share-dlg");
async function refreshCode() {
  const scope = ($<HTMLInputElement>('input[name="scope"]:checked', shareDlg)).value;
  $("#share-error").textContent = "";
  try { $<HTMLTextAreaElement>("#code-out").value = await api.ExportCode(scope === "char" ? sel.char : ""); }
  catch (err) { $<HTMLTextAreaElement>("#code-out").value = ""; $("#share-error").textContent = errText(err); }
}
$("#share").addEventListener("click", () => {
  $(".scope-char", shareDlg).textContent = `${char().name} only`;
  shareDlg.showModal();
  refreshCode();
});
shareDlg.querySelectorAll('input[name="scope"]').forEach((r) => r.addEventListener("change", refreshCode));
$("#copy-code").addEventListener("click", async () => {
  const v = $<HTMLTextAreaElement>("#code-out").value;
  if (!v) return;
  try { await navigator.clipboard.writeText(v); } catch { $<HTMLTextAreaElement>("#code-out").select(); document.execCommand("copy"); }
  toast("Code copied");
});
$("#import-code").addEventListener("click", async () => {
  try {
    const { state: s, count: n } = await api.ImportCode($<HTMLTextAreaElement>("#code-in").value);
    state = s;
    $<HTMLTextAreaElement>("#code-in").value = "";
    shareDlg.close();
    renderAll();
    toast(`Imported ${n} layer${n === 1 ? "" : "s"}`);
  } catch (err) { $("#share-error").textContent = errText(err); }
});

// ------------------------------------------------------------------ effect finder (dev)

const finderDlg = $<HTMLDialogElement>("#finder-dlg");
let finderPoll = 0;
const esc = (t: string) => t.replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]!));

function renderFinder(v: FinderView) {
  $(".finder-setup", finderDlg).hidden = v.active;
  $(".finder-run", finderDlg).hidden = !v.active;
  $("#finder-error").textContent = "";
  if (!v.active) return;
  const left = v.entries?.length ?? 0;
  $(".finder-round").textContent = `Round ${v.round} · ${left} of ${v.total} ${v.kind === "live" ? "colour curves" : "materials"} left` +
    (v.done ? " · each colour is one effect now" : "");
  const pk = $(".finder-pak");
  pk.hidden = v.kind !== "pak" || !v.pak.message;
  pk.dataset.state = v.pak.state;
  pk.textContent = v.pak.message;
  $(".finder-colors").innerHTML = (v.colors ?? []).map((c) =>
    `<button class="finder-color" data-hex="${c.color}" ${left < 2 ? "disabled" : ""}>
      <i style="background:${c.color}"></i><span>${c.colorName}</span><small>${c.name}</small></button>`).join("");
  $(".finder-ask").textContent = left < 2 ? "Found it." : "Do the move. What colour did the effect turn?";
  const legend = $(".finder-legend");
  legend.hidden = !v.done;
  legend.innerHTML = v.done ? (v.entries ?? []).map((e) =>
    `<li><i style="background:${e.color}"></i><b>${e.colorName}</b><span>${esc(e.name)}</span><small>${esc(e.detail)}</small></li>`).join("") : "";
}

async function openFinder() {
  const sc = { kazuya: "/character/grl/", jin: "/character/ant/", deviljin: "/character/swl/", reina: "/character/zbr/", heihachi: "/character/bee/" } as Record<string, string>;
  const v = await api.FinderView();
  if (!v.active && sc[sel.char]) $<HTMLSelectElement>("#finder-scope").value = sc[sel.char];
  renderFinder(v);
  finderDlg.showModal();
  clearInterval(finderPoll);
  finderPoll = window.setInterval(async () => {
    if (!finderDlg.open) return clearInterval(finderPoll);
    const cur = await api.FinderView();
    if (cur.active && cur.kind === "pak") renderFinder(cur);
  }, 1500);
}
const finderAct = async (fn: () => Promise<FinderView>) => {
  try { renderFinder(await fn()); } catch (err) { $("#finder-error").textContent = errText(err); }
};
$("#finder-scope").addEventListener("change", (e) => {
  ($(".finder-custom") as HTMLElement).hidden = (e.target as HTMLSelectElement).value !== "custom";
});
$("#finder-start").addEventListener("click", async () => {
  const pick = $<HTMLSelectElement>("#finder-scope").value;
  const scope = pick === "custom" ? $<HTMLInputElement>("#finder-custom").value : pick;
  const kind = ($<HTMLInputElement>('input[name="finder-kind"]:checked', finderDlg)).value;
  try { renderFinder(await api.FinderStart(kind, scope)); }
  catch (err) { toast(errText(err)); }
});
$(".finder-colors").addEventListener("click", (e) => {
  const b = (e.target as HTMLElement).closest<HTMLElement>("[data-hex]");
  if (b) finderAct(() => api.FinderPick(b.dataset.hex!));
});
$("#finder-back").addEventListener("click", () => finderAct(() => api.FinderBack()));
$("#finder-stop").addEventListener("click", () => finderAct(() => api.FinderStop()));
$("#finder-legend").addEventListener("click", () => api.OpenFinderLegend());
$("#finder-close").addEventListener("click", () => finderDlg.close());

// ------------------------------------------------------------------ claw trail mod status

function renderPak() {
  const b = $(".pak-banner");
  b.hidden = !isPak() || pakStatus.state === "off" || !pakStatus.message;
  b.dataset.state = pakStatus.state;
  $(".pak-text").textContent = pakStatus.message;
  $("#pak-folder").hidden = pakStatus.state !== "error";
}
onPak((s) => { pakStatus = s; if (catalog.length) renderPak(); });
$("#pak-folder").addEventListener("click", () => api.OpenPakFolder());

// ------------------------------------------------------------------ edition, updates, background

function setBackground(url: string) {
  document.body.style.setProperty("--bg-image", url ? `url("${url}")` : "none");
  document.body.classList.toggle("has-bg", !!url);
}

function applyEdition(ed: Edition) {
  document.body.dataset.theme = ed.theme;
  document.body.classList.toggle("dev", ed.devTools);
  document.title = ed.title;
  $(".brand-name").textContent = ed.title;
  const g = $(".greeting");
  g.textContent = ed.greeting;
  g.hidden = !ed.greeting;
  const badge = $(".badge");
  badge.textContent = ed.badge;
  badge.hidden = !ed.badge;
  document.querySelectorAll<HTMLElement>(".keys-on").forEach((el) => (el.hidden = !ed.keysOn || !!ed.tier));
  const rm = $(".key-remove");
  rm.hidden = !ed.tier;
  rm.textContent = `Remove key${ed.keyName ? ` (${ed.keyName})` : ""}`;
  $(".menu-version").textContent = ed.version === "dev" ? "Development build" : `Version ${ed.version}`;
  $(".lock").hidden = !ed.locked;
  $(".lock-version").textContent = `Version ${ed.version}`;
  if (catalog.length) renderAccents();
}
onEdition(applyEdition);
$<HTMLFormElement>(".lock-card").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  $("#lock-error").textContent = "";
  try {
    const ed = await api.ActivateKey($<HTMLTextAreaElement>("#lock-key").value);
    if (ed.locked) {
      $("#lock-error").textContent = "That key is valid but doesn't include beta access.";
      return;
    }
    applyEdition(ed);
    toast(ed.greeting ? `${ed.greeting}!` : "Beta unlocked. Thanks for the support!");
  } catch (err) { $("#lock-error").textContent = errText(err); }
});

onUpdate((u) => {
  const box = $(".update");
  box.hidden = u.state !== "ready";
  if (u.state === "ready") $(".update-text").textContent = `Version ${u.version} is installed.`;
  if (u.state === "error" && u.message) console.warn(u.message);
});
$("#restart").addEventListener("click", () => api.RestartToUpdate().catch((e) => toast(errText(e))));

// ------------------------------------------------------------------ boot

(async () => {
  applyEdition(await api.Edition());
  setBackground(await api.Background().catch(() => ""));
  [catalog, state, defaults] = await Promise.all([api.Catalog(), api.State(), api.DefaultSetting()]);
  select(visibleChars()[0].id);
  onStatus((s) => { status = s; renderStatus(); renderLayers(); });
})();
