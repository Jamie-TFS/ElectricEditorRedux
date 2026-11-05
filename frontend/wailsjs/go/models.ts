export namespace fx {
	
	export class Part {
	    id: string;
	    label: string;
	    role: string;
	
	    static createFrom(source: any = {}) {
	        return new Part(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.role = source["role"];
	    }
	}
	export class Effect {
	    id: string;
	    label: string;
	    kind?: string;
	    note?: string;
	    parts: Part[];
	
	    static createFrom(source: any = {}) {
	        return new Effect(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.kind = source["kind"];
	        this.note = source["note"];
	        this.parts = this.convertValues(source["parts"], Part);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Character {
	    id: string;
	    name: string;
	    effects: Effect[];
	
	    static createFrom(source: any = {}) {
	        return new Character(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.effects = this.convertValues(source["effects"], Effect);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Copy {
	    From: string;
	    Rename: string[][];
	    Users: string[];
	
	    static createFrom(source: any = {}) {
	        return new Copy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.From = source["From"];
	        this.Rename = source["Rename"];
	        this.Users = source["Users"];
	    }
	}
	
	
	export class Stop {
	    pos: number;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new Stop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pos = source["pos"];
	        this.color = source["color"];
	    }
	}
	export class Setting {
	    mode: string;
	    color: string;
	    stops: Stop[];
	    brightness: number;
	    keepFade: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Setting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.color = source["color"];
	        this.stops = this.convertValues(source["stops"], Stop);
	        this.brightness = source["brightness"];
	        this.keepFade = source["keepFade"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class Edition {
	    tier: string;
	    title: string;
	    theme: string;
	    channel: string;
	    devTools: boolean;
	    badge: string;
	    greeting: string;
	    keyName: string;
	    keysOn: boolean;
	    version: string;
	    locked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Edition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tier = source["tier"];
	        this.title = source["title"];
	        this.theme = source["theme"];
	        this.channel = source["channel"];
	        this.devTools = source["devTools"];
	        this.badge = source["badge"];
	        this.greeting = source["greeting"];
	        this.keyName = source["keyName"];
	        this.keysOn = source["keysOn"];
	        this.version = source["version"];
	        this.locked = source["locked"];
	    }
	}
	export class FinderEntry {
	    color: string;
	    colorName: string;
	    name: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new FinderEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.color = source["color"];
	        this.colorName = source["colorName"];
	        this.name = source["name"];
	        this.detail = source["detail"];
	    }
	}
	export class PakStatus {
	    state: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new PakStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.message = source["message"];
	    }
	}
	export class FinderView {
	    active: boolean;
	    kind: string;
	    scope: string;
	    round: number;
	    total: number;
	    entries: FinderEntry[];
	    colors: FinderEntry[];
	    done: boolean;
	    pak: PakStatus;
	    legend: string;
	
	    static createFrom(source: any = {}) {
	        return new FinderView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.kind = source["kind"];
	        this.scope = source["scope"];
	        this.round = source["round"];
	        this.total = source["total"];
	        this.entries = this.convertValues(source["entries"], FinderEntry);
	        this.colors = this.convertValues(source["colors"], FinderEntry);
	        this.done = source["done"];
	        this.pak = this.convertValues(source["pak"], PakStatus);
	        this.legend = source["legend"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class State {
	    active: string;
	    profiles: string[];
	    parts: Record<string, fx.Setting>;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.profiles = source["profiles"];
	        this.parts = this.convertValues(source["parts"], fx.Setting, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportResult {
	    state: State;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = this.convertValues(source["state"], State);
	        this.count = source["count"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class Status {
	    state: string;
	    message: string;
	    process: string;
	    curves: number;
	    loaded: Record<string, boolean>;
	    applied: number;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.message = source["message"];
	        this.process = source["process"];
	        this.curves = source["curves"];
	        this.loaded = source["loaded"];
	        this.applied = source["applied"];
	    }
	}
	export class UpdateStatus {
	    state: string;
	    version: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.version = source["version"];
	        this.message = source["message"];
	    }
	}

}

