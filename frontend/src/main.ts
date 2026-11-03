import "@fontsource/unbounded/500.css";
import "@fontsource/unbounded/700.css";
import "@fontsource/manrope/400.css";
import "@fontsource/manrope/600.css";
import "@fontsource/manrope/700.css";
import "./style.css";

import { api, errText, keyOf, onStatus, onUpdate, type Character, type Mode, type Setting, type State, type Status } from "./api";
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
const sel = { char: "", effect: "", part: "" };

// ------------------------------------------------------------------ shell

document.querySelector("#app")!.innerHTML = `
  <header class="top">
    <div class="brand"><span class="mark" aria-hidden="true"></span><span class="brand-name">Electric Editor</span></div>
    <span class="greeting" hidden></span>
    <div class="conn" role="status"><span class="dot"></span><span class="conn-text">Looking for Tekken…</span></div>
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
        <button role="menuitem" data-act="bg">Set background image…</button>
        <button role="menuitem" data-act="bg-clear">Remove background image</button>
        <hr>
        <button role="menuitem" data-act="dump">Save curve list…</button>
        <button role="menuitem" data-act="trail-on" class="dev-only">Test: turn claw trails green</button>
        <button role="menuitem" data-act="trail-off" class="dev-only">Undo claw trail test</button>
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
      <figcaption>What each layer sends to the game, from the start of the electric (left) to where it fades out (right). Click a bar to edit it.</figcaption>
    </figure>
    <section class="editor">
      <div class="editor-head">
        <h2 class="layer-name"></h2>
        <div class="modes" role="radiogroup" aria-label="Colour mode">
          <button role="radio" data-mode="native">Game colours</button>
          <button role="radio" data-mode="solid">Solid</button>
          <button role="radio" data-mode="gradient">Gradient</button>
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
        <label class="switch">
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
const settingFor = (key: string): Setting => state.parts[key] ?? { ...structuredClone(defaults), mode: "native" };
const current = () => settingFor(curKey());

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
  rail.innerHTML = catalog.map((c) => {
    const swatches = c.effects[0].parts.slice(0, 4).map((p) => {
      const s = state.parts[keyOf(c.id, c.effects[0].id, p.id)];
      return `<i style="background:${s ? cssGradient(s, NATIVE[p.role], "180deg") : "transparent"}" class="${s ? "" : "empty"}"></i>`;
    }).join("");
    return `<button class="fighter${c.id === sel.char ? " on" : ""}" data-char="${c.id}" aria-current="${c.id === sel.char}">
      <span class="fighter-label">${c.name}</span><span class="swatches" aria-hidden="true">${swatches}</span></button>`;
  }).join("");
}

function renderHead() {
  $(".fighter-name").textContent = char().name;
  const tabs = $(".effects");
  tabs.innerHTML = char().effects.map((e) =>
    `<button role="tab" aria-selected="${e.id === sel.effect}" data-effect="${e.id}">${e.label}</button>`).join("");
  tabs.hidden = char().effects.length < 2;
}

function renderLayers() {
  const loaded = status?.loaded ?? {};
  strips.set(effect().parts.map((p) => {
    const key = keyOf(sel.char, sel.effect, p.id);
    return { key, label: p.label, setting: settingFor(key), focus: p.id === sel.part, inGame: !!loaded[key] };
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
  br.value = String(s.brightness);
  $("#brightness-out").textContent = s.brightness.toFixed(1);
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
}

function select(charId: string, effectId?: string, partId?: string) {
  sel.char = charId;
  const c = char();
  sel.effect = effectId ?? c.effects[0].id;
  sel.part = partId ?? effect().parts[0].id;
  grad.selected = 0;
  renderAll();
}

// ------------------------------------------------------------------ events

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
  if (act === "bg") {
    try { setBackground(await api.ChooseBackground()); } catch (err) { toast(errText(err)); }
  }
  if (act === "bg-clear") { await api.ClearBackground(); setBackground(""); }
  if (act === "dump") {
    try { const path = await api.DumpCurves(); toast(`Saved ${path.split(/[\\/]/).pop()}`); }
    catch (err) { toast(errText(err)); }
  }
  if (act === "trail-on" || act === "trail-off") {
    try {
      const n = await api.TrailTest(act === "trail-on");
      toast(act === "trail-on"
        ? (n ? `Set ${n} trail colours to green. Try MCR1 or 1,1,2.` : "No trail materials loaded yet. Start a match first.")
        : `Restored ${n} trail colours`);
    } catch (err) { toast(errText(err)); }
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

// ------------------------------------------------------------------ edition, updates, background

function setBackground(url: string) {
  document.body.style.setProperty("--bg-image", url ? `url("${url}")` : "none");
  document.body.classList.toggle("has-bg", !!url);
}

async function applyEdition() {
  const ed = await api.Edition();
  document.body.dataset.theme = ed.theme;
  document.body.classList.toggle("dev", ed.devTools);
  document.title = ed.title;
  $(".brand-name").textContent = ed.title;
  const g = $(".greeting");
  g.textContent = ed.greeting;
  g.hidden = !ed.greeting;
  $(".menu-version").textContent = ed.version === "dev" ? "Development build" : `Version ${ed.version}`;
  setBackground(await api.Background().catch(() => ""));
}

onUpdate((u) => {
  const box = $(".update");
  box.hidden = u.state !== "ready";
  if (u.state === "ready") $(".update-text").textContent = `Version ${u.version} is installed.`;
  if (u.state === "error" && u.message) console.warn(u.message);
});
$("#restart").addEventListener("click", () => api.RestartToUpdate().catch((e) => toast(errText(e))));

// ------------------------------------------------------------------ boot

(async () => {
  await applyEdition();
  [catalog, state, defaults] = await Promise.all([api.Catalog(), api.State(), api.DefaultSetting()]);
  select(catalog[0].id);
  onStatus((s) => { status = s; renderStatus(); renderLayers(); });
})();
