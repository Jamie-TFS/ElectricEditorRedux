// Thin layer over the Go backend. When opened in a plain browser (no Wails),
// a mock backend is used so the UI can be developed and screenshotted.
import mockCatalog from "./mock-catalog.json";

export type Mode = "native" | "solid" | "gradient";
export interface Stop { pos: number; color: string }
export interface Setting {
  mode: Mode;
  color: string;
  stops: Stop[];
  brightness: number;
  keepFade: boolean;
}
export interface Part { id: string; label: string; role: "core" | "glow" | "outer" | "trail" }
export interface Effect { id: string; label: string; kind?: "pak" | "rgb"; note?: string; parts: Part[] }
export interface Character { id: string; name: string; effects: Effect[] }
export interface State { active: string; profiles: string[]; parts: Record<string, Setting> }
export interface Status {
  state: "waiting" | "connecting" | "connected" | "error";
  message: string;
  process: string;
  curves: number;
  loaded: Record<string, boolean>;
  applied: number;
}

export interface Edition {
  tier: string; title: string; theme: string; channel: string; devTools: boolean;
  badge: string; greeting: string; keyName: string; keysOn: boolean; version: string; locked: boolean;
}
export interface PakStatus { state: "off" | "building" | "pending" | "ready" | "error" | "nogame"; message: string }
export interface FinderEntry { color: string; colorName: string; name: string; detail: string }
export interface FinderView {
  active: boolean; kind: "live" | "pak"; scope: string; round: number; total: number;
  entries: FinderEntry[] | null; colors: FinderEntry[] | null; done: boolean; pak: PakStatus; legend: string;
}
export interface UpdateStatus { state: "disabled" | "checking" | "current" | "ready" | "error"; version: string; message: string }

interface Backend {
  Edition(): Promise<Edition>;
  ActivateKey(key: string): Promise<Edition>;
  RemoveKey(): Promise<Edition>;
  UpdateStatus(): Promise<UpdateStatus>;
  RestartToUpdate(): Promise<void>;
  Background(): Promise<string>;
  ChooseBackground(): Promise<string>;
  ClearBackground(): Promise<void>;
  Catalog(): Promise<Character[]>;
  DefaultSetting(): Promise<Setting>;
  State(): Promise<State>;
  Status(): Promise<Status>;
  SetPart(key: string, s: Setting): Promise<void>;
  SetMany(keys: string[], s: Setting): Promise<void>;
  SwitchProfile(name: string): Promise<State>;
  CreateProfile(name: string): Promise<State>;
  RenameProfile(name: string): Promise<State>;
  DeleteProfile(): Promise<State>;
  ExportCode(charID: string): Promise<string>;
  DumpCurves(): Promise<string>;
  PakStatus(): Promise<PakStatus>;
  FinderView(): Promise<FinderView>;
  FinderStart(kind: string, scope: string): Promise<FinderView>;
  FinderPick(hex: string): Promise<FinderView>;
  FinderBack(): Promise<FinderView>;
  FinderStop(): Promise<FinderView>;
  OpenFinderLegend(): Promise<void>;
  ExportAssets(filters: string): Promise<string>;
  BuildTestMod(text: string): Promise<PakStatus>;
  TestModStatus(): Promise<PakStatus>;
  OpenPakFolder(): Promise<void>;
  ImportCode(code: string): Promise<{ state: State; count: number }>;
}

declare global {
  interface Window {
    go?: { main?: { App?: Backend } };
    runtime?: { EventsOn(name: string, cb: (...a: unknown[]) => void): () => void };
  }
}

const isApp = () => !!window.go?.main?.App;

// ------------------------------------------------------------ browser mock

function mockEdition(tier: string): Edition {
  const t: Record<string, Partial<Edition>> = {
    "": { title: "Electric Editor", theme: "storm" },
    donator: { title: "Electric Editor", theme: "storm", badge: "Supporter", keyName: "Sam" },
    dev: { title: "Electric Editor dev", theme: "storm", devTools: true, badge: "Dev", channel: "beta", keyName: "Jamie" },
    amy: { title: "Amy's Electric Editor", theme: "amy", greeting: "Hi Amy", keyName: "Amy" },
  };
  const beta = new URLSearchParams(location.search).get("beta") !== null;
  return { tier, title: "", theme: "storm", channel: "stable", devTools: false, badge: "", greeting: "", keyName: "",
    keysOn: true, version: beta ? "1.1.0-beta.2" : "1.0.0", locked: beta && !tier, ...t[tier] } as Edition;
}

function makeMock(): Backend {
  const def = (): Setting => ({
    mode: "native", color: "#7c5cff", brightness: 6, keepFade: true,
    stops: [{ pos: 0, color: "#00e5ff" }, { pos: 1, color: "#7c5cff" }],
  });
  let active = "Main";
  const profiles: Record<string, Record<string, Setting>> = {
    Main: {
      "deviljin.electric.trail": { ...def(), mode: "gradient", stops: [
        { pos: 0, color: "#00e5ff" }, { pos: 0.55, color: "#7c5cff" }, { pos: 1, color: "#ff3d81" }] },
      "deviljin.electric.lightning": { ...def(), mode: "solid", color: "#c9b6ff", brightness: 8 },
      "kazuya.electric.bolt": { ...def(), mode: "solid", color: "#ffd23f" },
    },
    Tournament: {},
  };
  const state = (): State => ({ active, profiles: Object.keys(profiles).sort(), parts: structuredClone(profiles[active]) });
  const loaded: Record<string, boolean> = {};
  for (const c of mockCatalog as Character[]) {
    if (c.id === "deviljin") for (const e of c.effects) for (const p of e.parts) loaded[`${c.id}.${e.id}.${p.id}`] = true;
  }
  const params = new URLSearchParams(location.search);
  const pal = [["#ff1a1a", "Red"], ["#1aff3c", "Green"], ["#1a5cff", "Blue"], ["#ffe81a", "Yellow"],
    ["#1af0ff", "Cyan"], ["#ff1aff", "Magenta"], ["#ff8a00", "Orange"], ["#ffffff", "White"]];
  let fItems: string[] = [];
  let fHist: number[][] = [];
  let fKind: "live" | "pak" = "live";
  let fScope = "";
  const fView = (): FinderView => {
    if (!fHist.length) return { active: false } as FinderView;
    const cur = fHist[fHist.length - 1];
    const entries = cur.map((i, j) => ({ color: pal[j % 8][0], colorName: pal[j % 8][1], name: fItems[i], detail: `/Game/Effect/Polariswork/Character/swl/${fItems[i]}` }));
    const colors = pal.map(([color, colorName], k) => ({ color, colorName, name: String(cur.filter((_, j) => j % 8 === k).length), detail: "" })).filter((c) => c.name !== "0");
    return { active: true, kind: fKind, scope: fScope, round: fHist.length, total: fItems.length, entries, colors,
      done: cur.length <= 8, legend: "C:\\Users\\you\\AppData\\Roaming\\ElectricEditor\\finder.txt",
      pak: fKind === "pak" ? { state: "ready", message: "Test colours installed. Restart Tekken, do the move, then pick the colour you see." } : { state: "off", message: "" } };
  };
  return {
    Edition: async () => mockEdition(params.get("key") ?? ""),
    ActivateKey: async (k: string) => {
      const tier = /amy/i.test(k) ? "amy" : /dev/i.test(k) ? "dev" : /don/i.test(k) ? "donator" : "";
      if (!tier) throw "that key isn't valid - check you copied all of it";
      return mockEdition(tier);
    },
    RemoveKey: async () => mockEdition(""),
    UpdateStatus: async () => ({ state: params.get("update") === "ready" ? "ready" : "current", version: "1.1.0", message: "" }),
    RestartToUpdate: async () => {},
    Background: async () => "",
    ChooseBackground: async () => "",
    ClearBackground: async () => {},
    Catalog: async () => mockCatalog as Character[],
    DefaultSetting: async () => def(),
    State: async () => state(),
    Status: async () => ({ state: "connected", message: "Connected. Changes show up in game straight away.",
      process: "Polaris-Win64-Shipping.exe", curves: 54, loaded, applied: 2 }),
    SetPart: async (k, s) => { if (s.mode === "native") delete profiles[active][k]; else profiles[active][k] = s; },
    SetMany: async (ks, s) => { for (const k of ks) profiles[active][k] = structuredClone(s); },
    SwitchProfile: async (n) => { active = n; return state(); },
    CreateProfile: async (n) => {
      if (!n.trim()) throw "profile names need 1 to 40 characters";
      if (profiles[n]) throw `a profile called "${n}" already exists`;
      profiles[n] = structuredClone(profiles[active]); active = n; return state();
    },
    RenameProfile: async (n) => { profiles[n] = profiles[active]; delete profiles[active]; active = n; return state(); },
    DeleteProfile: async () => {
      if (Object.keys(profiles).length < 2) throw "you need at least one profile";
      delete profiles[active]; active = Object.keys(profiles).sort()[0]; return state();
    },
    PakStatus: async () => {
      const pak = new Set((mockCatalog as Character[]).flatMap((c) => c.effects.filter((e) => e.kind === "pak").map((e) => `${c.id}.${e.id}.`)));
      const on = Object.entries(profiles[active]).some(([k, v]) => v.mode !== "native" && [...pak].some((p) => k.startsWith(p)));
      return on ? { state: "ready", message: "Mod colours installed. They show next time you start Tekken." } : { state: "off", message: "" };
    },
    OpenPakFolder: async () => {},
    FinderView: async () => fView(),
    FinderStart: async (kind, scope) => {
      fKind = kind as "live" | "pak"; fScope = scope;
      fItems = Array.from({ length: 37 }, (_, i) => kind === "live" ? `NS_EF_Swl_Effect${i} · Emitter_${i % 4} · ColorCurve_0` : `MI_EF_Swl_Material${String(i).padStart(2, "0")}`);
      fHist = [fItems.map((_, i) => i)];
      return fView();
    },
    FinderPick: async (hex) => {
      const cur = fHist[fHist.length - 1];
      fHist.push(cur.filter((_, j) => pal[j % 8][0] === hex));
      return fView();
    },
    FinderBack: async () => { if (fHist.length > 1) fHist.pop(); return fView(); },
    FinderStop: async () => { fHist = []; return fView(); },
    OpenFinderLegend: async () => {},
    ExportAssets: async () => "Saved 23 files to research.zip",
    BuildTestMod: async (t) => t.trim() ? { state: "ready", message: "Test mod installed. Restart Tekken to try it." } : { state: "off", message: "No test mod installed." },
    TestModStatus: async () => ({ state: "off", message: "No test mod installed." }),
    DumpCurves: async () => "C:\\Users\\you\\AppData\\Roaming\\ElectricEditor\\curves.txt",
    ExportCode: async () => "EE1.jVLNbtswDH6XnBVAP7YT59bTMDSXrAN2KHxQZFoholeZ1mQmx98-Sko3YNg6v4oGjQ0MjhkE0j4",
    ImportCode: async () => ({ state: state(), count: 3 }),
  };
}

const backend: Backend = isApp() ? window.go!.main!.App! : makeMock();

export const api = backend;

export function onEdition(cb: (e: Edition) => void): void {
  if (isApp() && window.runtime) window.runtime.EventsOn("edition", (e) => cb(e as Edition));
}

export function onUpdate(cb: (s: UpdateStatus) => void): void {
  if (isApp() && window.runtime) window.runtime.EventsOn("update", (s) => cb(s as UpdateStatus));
  api.UpdateStatus().then(cb).catch(() => {});
}

export function onPak(cb: (s: PakStatus) => void): void {
  if (isApp() && window.runtime) window.runtime.EventsOn("pak", (s) => cb(s as PakStatus));
  const poll = async () => { try { cb(await api.PakStatus()); } catch { /* ignore */ } };
  poll();
  setInterval(poll, 2000);
}

export function onStatus(cb: (s: Status) => void): void {
  if (isApp() && window.runtime) {
    window.runtime.EventsOn("status", (s) => cb(s as Status));
  }
  // also poll, so the UI recovers if an event is missed
  const poll = async () => { try { cb(await api.Status()); } catch { /* ignore */ } };
  poll();
  setInterval(poll, 1500);
}

export const keyOf = (char: string, effect: string, part: string) => `${char}.${effect}.${part}`;

export function errText(e: unknown): string {
  const s = typeof e === "string" ? e : (e as Error)?.message ?? String(e);
  return s.charAt(0).toUpperCase() + s.slice(1);
}
