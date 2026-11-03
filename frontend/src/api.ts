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
export interface Effect { id: string; label: string; parts: Part[] }
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
  badge: string; greeting: string; keyName: string; keysOn: boolean; version: string;
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
  TrailTest(on: boolean): Promise<number>;
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
  return { tier, title: "", theme: "storm", channel: "stable", devTools: false, badge: "", greeting: "", keyName: "",
    keysOn: true, version: "1.0.0", ...t[tier] } as Edition;
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
    TrailTest: async () => 6,
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
