package fx

// Part is one recolourable layer of an effect: one or more colour curves
// that should always share a colour.
type Part struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Role  string   `json:"role"` // core | glow | outer | trail - drives the preview
	Paths []string `json:"-"`
	// Asset is set for material parts (Effect.Kind "pak"): the material
	// instance that gets recoloured in the generated mod pak.
	Asset string `json:"-"`
	// Params are the material's colour parameters (default BaseTexColor,
	// BaseTex2Color).
	Params []string `json:"-"`
	// RGB drives colour through three float curves (red, green, blue), each
	// listed with every copy of it. Used by effects whose colour is a Niagara
	// dynamic material parameter (Effect.Kind "rgb", live like the electrics).
	RGB [3][]string `json:"-"`
	// Texture marks Asset as a DXT texture: it's recoloured by tinting its
	// pixel blocks (keeps brightness, takes the hue), patched straight into
	// the game's packaged bytes (textures crash after a legacy round trip).
	Texture bool `json:"-"`
	// RGBScale evens out channels whose curves are multiplied differently
	// in the effect (0 = 1).
	RGBScale [3]float32 `json:"-"`
	// Vec lists vector curves (xyz per sample) that scale an effect's colour,
	// e.g. Niagara "Scale RGB". Recoloured live (Effect.Kind "rgb").
	Vec []string `json:"-"`
	// Copy makes Asset a private, renamed copy of a shared game material, so
	// one fighter can be recoloured without the other. Users lists the
	// assets (with the same renames) that must point at the copy instead.
	Copy *Copy `json:"-"`
	// VM names script patches (pak.VMPatches) of the Niagara system
	// VMAssets that are recoloured with this part, for emitters whose colour
	// is written by their script. Works alongside Asset or on its own.
	VM       []string `json:"-"`
	VMAssets []string `json:"-"`
}

type Copy struct {
	From   string
	Rename [][2]string
	Users  []string
}

type Effect struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Kind is "" for live colour curves, "rgb" for live float-curve colour
	// (solid colours only), "pak" for materials baked into a mod pak (shows
	// next time Tekken starts).
	Kind  string `json:"kind,omitempty"`
	Note  string `json:"note,omitempty"`
	Parts []Part `json:"parts"`
}

type Character struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Effects []Effect `json:"effects"`
}

// Paths are copied verbatim from the original Electric Editor Lua mod.
// Do not "tidy" casing: Polariswork / PolarisBEEwork / Niagarasystem are
// exactly how the game names them.
const (
	grl  = "/Game/Effect/Polariswork/Character/grl/NiagaraSystem/NS_EF_Grl_Saifu02.NS_EF_Grl_Saifu02:"
	zbrS = "/Game/Effect/Polariswork/Character/zbr/NiagaraSystem/NS_EF_Zbr_Saifu02.NS_EF_Zbr_Saifu02:"
	zbrK = "/Game/Effect/Polariswork/Character/zbr/NiagaraSystem/NS_EF_Zbr_Bujinkyaku01.NS_EF_Zbr_Bujinkyaku01:"
	zbrT = "/Game/Effect/Polariswork/Character/zbr/NiagaraSystem/NS_EF_Zbr_LightningTrail02.NS_EF_Zbr_LightningTrail02:"
	beeS = "/Game/Effect/PolarisBEEwork/Character/BEE/Niagarasystem/NS_EF_Bee_Saifu02.NS_EF_Bee_Saifu02:"
	beeR = "/Game/Effect/PolarisBEEwork/Character/BEE/Niagarasystem/NS_EF_Bee_RaijinTrail.NS_EF_Bee_RaijinTrail:"
	ant  = "/Game/Effect/Polariswork/Character/ant/NiagaraSystem/NS_EF_Ant_Saifu02.NS_EF_Ant_Saifu02:"
	swl  = "/Game/Effect/Polariswork/Character/swl/NiagaraSystem/NS_EF_Swl_Saifu02.NS_EF_Swl_Saifu02:"

	swlBeam = "/Game/Effect/Polariswork/Character/swl/NiagaraSystem/NS_EF_Swl_Beam.NS_EF_Swl_Beam:"

	grlNS = "/Game/Effect/Polariswork/Character/grl/NiagaraSystem/"
	swlNS = "/Game/Effect/Polariswork/Character/swl/NiagaraSystem/"
)

// gpu lists both copies of a GPU emitter's curve (GPUComputeScript is what
// renders; UpdateScript is kept in sync so a reload can't pick the old one).
func gpu(system, emitter, curve string) []string {
	return []string{
		system + ":" + emitter + ".GPUComputeScript." + curve,
		system + ":" + emitter + ".UpdateScript." + curve,
	}
}

func beamParts(root, suffix string) []Part {
	return []Part{
		{ID: "bolt", Label: "Bolt", Role: "core", Paths: []string{root + "DynamicBeam003_4.EventScript_0.NiagaraDataInterfaceColorCurve_" + suffix}},
		{ID: "glow", Label: "Glow", Role: "glow", Paths: []string{root + "DynamicBeam002_11.EventScript_0.NiagaraDataInterfaceColorCurve_" + suffix}},
		{ID: "outer", Label: "Outer arc", Role: "outer", Paths: []string{root + "DynamicBeam001_3.EventScript_0.NiagaraDataInterfaceColorCurve_" + suffix}},
		{ID: "outer2", Label: "Outer arc 2", Role: "outer", Paths: []string{root + "DynamicBeam_4.EventScript_0.NiagaraDataInterfaceColorCurve_" + suffix}},
	}
}

func lightningParts(root string) []Part {
	return []Part{
		{ID: "trail", Label: "Trail", Role: "trail", Paths: []string{root + "Aura_2.EventScript_0.NiagaraDataInterfaceColorCurve_0"}},
		{ID: "trail2", Label: "Trail 2", Role: "trail", Paths: []string{root + "Aura002_3.EventScript_0.NiagaraDataInterfaceColorCurve_0"}},
		{ID: "lightning", Label: "Lightning", Role: "core", Paths: []string{root + "LIGHTNING_0.EventScript_0.NiagaraDataInterfaceColorCurve_0"}},
		{ID: "lightning2", Label: "Lightning 2", Role: "glow", Paths: []string{root + "Lightning001_1.EventScript_0.NiagaraDataInterfaceColorCurve_0"}},
	}
}

var Catalog = []Character{
	{ID: "kazuya", Name: "Kazuya", Effects: []Effect{
		{ID: "electric", Label: "Electrics", Parts: beamParts(grl, "1")},
		{ID: "devil", Label: "Devil shockwaves", Kind: "pak",
			Note: "Kazuya's devil shockwaves.",
			Parts: []Part{
				{ID: "b12", Label: "b1,2 / df3+4,1,2", Role: "glow", Asset: "MI_EF_Grl_Aura09", Params: []string{"Col2", "EdgeCol", "EdgeCol2"}},
				{ID: "heat", Label: "Heat db3+4", Role: "core", Asset: "MI_EF_Grl_DvlAura_01", Params: []string{"EDGEcolor"}},
				{ID: "ff2", Label: "f,f2", Role: "outer", Asset: "MI_EF_Grl_Wing_00", Params: []string{"EDGEcolor"}},
				// Not placed yet (hidden for release; add back once the move is known):
				// {ID: "fire", Label: "Fire", Role: "trail", Asset: "MI_EF_Grl_Aura06", Params: []string{"Col1", "Col2"}},
				// {ID: "aura00", Label: "Devil aura 2", Role: "glow", Asset: "MI_EF_Grl_DvlAura_00", Params: []string{"EDGEcolor"}},
			}},
		{ID: "beam", Label: "Beam", Kind: "pak",
			Note: "Kazuya's eye beam: the beam, its lightning, fire and glow. White highlights stay white.",
			Parts: []Part{
				{ID: "main", Label: "Beam", Role: "core", VMAssets: []string{"NS_EF_Grl_Beam"}, VM: []string{"*"}},
				{ID: "beaml", Label: "Beam L", Role: "glow", VMAssets: []string{"NS_EF_Grl_BeamL"}, VM: []string{"*"}},
				{ID: "beamn", Label: "Beam N", Role: "outer", VMAssets: []string{"NS_EF_Grl_BeamN"}, VM: []string{"*"}},
				{ID: "charge", Label: "Charge-up", Role: "trail", VMAssets: []string{"NS_EF_Grl_BeamCharge"}, VM: []string{"*"}},
			}},
	}},
	{ID: "jin", Name: "Jin", Effects: []Effect{
		{ID: "electric", Label: "Electrics", Parts: lightningParts(ant)},
		{ID: "trails", Label: "Trails", Kind: "pak",
			Note: "The blade trail on d2, d/f2 and d/f2,1+2.",
			Parts: []Part{
				{ID: "blade", Label: "Blade trail", Role: "trail", Asset: "MI_EF_Ant_trail03_blade"},
			}},
	}},
	{ID: "deviljin", Name: "Devil Jin", Effects: []Effect{
		{ID: "electric", Label: "Electrics", Parts: lightningParts(swl)},
		{ID: "claws", Label: "Claw trails", Kind: "pak",
			Note: "Claw swipes on WS2, MCR1, MCR 2,2, Rage slash and 1,1,2.",
			Parts: []Part{
				{ID: "ws2", Label: "WS2 claws", Role: "trail", Asset: ws2Blade, Copy: &Copy{
					From:   "MI_EF_Ant_trail03_blade",
					Rename: [][2]string{{"MI_EF_Ant_trail03_blade", ws2Blade}},
					Users:  []string{"DA_SWL_KisekiRender3"},
				}},
				{ID: "swipe", Label: "Claw swipe", Role: "trail", Asset: "MI_EF_SWL_trail01_blade"},
				{ID: "swipe2", Label: "Claw swipe 2", Role: "trail", Asset: "MI_EF_SWL_trail01_blade02"},
				{ID: "trail2", Label: "Trail 2", Role: "trail", Asset: "MI_EF_SWL_trail02_blade"},
				{ID: "single", Label: "Single trail", Role: "trail", Asset: "MI_EF_Swl_SingleTrail01_blade"},
				{ID: "edge", Label: "Single trail edge", Role: "trail", Asset: "MI_EF_Swl_SingleTrail01_edge"},
			}},
		{ID: "beam", Label: "Beam", Kind: "pak",
			Note: "The eye beam: the beam line, its lightning and the glow inside it.",
			Parts: []Part{
				{ID: "lightning", Label: "Lightning", Role: "core", Asset: "T_EF_COLA_Swl_Thunder03", Texture: true,
					VMAssets: []string{"NS_EF_Swl_Beam"}, VM: []string{"swl-beam-bolt", "swl-beam-thunder"}},
				{ID: "main", Label: "Beam", Role: "core", VMAssets: []string{"NS_EF_Swl_Beam"}, VM: []string{"swl-beam-e8#1", "swl-beam-light", "swl-beam-head", "swl-beam-burst"}},
				{ID: "glow", Label: "Glow", Role: "glow", VMAssets: []string{"NS_EF_Swl_Beam"}, VM: []string{"swl-beam-e8#0", "swl-beam-glow001", "swl-beam-glowburst", "swl-beam-firecore001", "swl-beam-fireblue001"}},
			}},
		{ID: "aura", Label: "MCR3", Kind: "pak",
			Note: "The devil aura and chains on MCR3.",
			Parts: []Part{
				{ID: "main", Label: "Aura", Role: "glow", Asset: "MI_EF_Swl_DvlAura_03", Params: []string{"EDGEcolor"}},
				{ID: "fire", Label: "Fire", Role: "outer", Asset: "MI_EF_Swl_DvlAura_04", Params: []string{"EDGEcolor"}},
				// the chain links themselves are textured; this is their red edge glow
				{ID: "chains", Label: "Chain glow", Role: "core", Asset: "MI_EF_Swl_S_Chain_00", Params: []string{"EDGEcolor"}},
			}},
		{ID: "chains", Label: "Chains", Kind: "pak",
			Note: "The chains on U1, UF1 and b,f1+2, and the aura around them. The chains are shared with Jin, so his change too.",
			Parts: []Part{
				{ID: "main", Label: "U1 chain", Role: "core", Asset: "MI_EF_Ant_S_Chain02", Params: chainParams},
				{ID: "chain5", Label: "UF1 chain", Role: "core", Asset: "MI_EF_Ant_S_Chain05", Params: chainParams},
				{ID: "chain1", Label: "b,f1+2 chain", Role: "core", Asset: "MI_EF_Ant_S_Chain01", Params: chainParams},
				{ID: "wrap", Label: "Chain wrap", Role: "glow", Asset: "MI_EF_Ant_S_Chain03", Params: chainParams},
				{ID: "small", Label: "Small links", Role: "outer", Asset: "MI_EF_Ant_S_Chain04", Params: chainParams},
				{ID: "aura", Label: "Chain aura", Role: "trail", Asset: "MI_EF_Ant_S_Aura24", Params: []string{"EDGEcolor"},
					VMAssets: []string{"NS_EF_Swl_Chain01", "NS_EF_Swl_Chain02"}, VM: []string{"*"}},
			}},
	}},
	{ID: "reina", Name: "Reina", Effects: []Effect{
		{ID: "electric", Label: "Electrics", Parts: beamParts(zbrS, "0")},
		{ID: "kick", Label: "Electric kick", Parts: beamParts(zbrK, "0")},
		{ID: "cd14", Label: "CD1,4", Parts: []Part{
			{ID: "trail", Label: "Lightning trail", Role: "core", Paths: []string{zbrT + "DynamicBeam_4.EventScript_0.NiagaraDataInterfaceColorCurve_0"}},
		}},
	}},
	{ID: "heihachi", Name: "Heihachi", Effects: []Effect{
		{ID: "electric", Label: "Electrics", Parts: beamParts(beeS, "1")},
		{ID: "raijin", Label: "CD1 electric", Parts: []Part{
			{ID: "bolt", Label: "Bolt", Role: "core", Paths: []string{beeR + "DynamicBeam003_4.EventScript_0.NiagaraDataInterfaceColorCurve_8"}},
			{ID: "glow", Label: "Glow", Role: "glow", Paths: []string{beeR + "DynamicBeam002_11.EventScript_0.NiagaraDataInterfaceColorCurve_8"}},
			{ID: "outer", Label: "Outer arc", Role: "outer", Paths: []string{beeR + "DynamicBeam001_3.EventScript_0.NiagaraDataInterfaceColorCurve_14"}},
			{ID: "outer2", Label: "Outer arc 2", Role: "outer", Paths: []string{beeR + "DynamicBeam_4.EventScript_0.NiagaraDataInterfaceColorCurve_12"}},
		}},
	}},
}

var chainParams = []string{"Col2", "EdgeCol", "EdgeCol2"}

// Devil Jin's own copy of Jin's blade (same length name, so the copy is a
// byte-for-byte rename). Packages are matched case-insensitively, so it must
// not collide with any real asset.
const ws2Blade = "MI_EF_Ws2_trail03_blade"

// PakAssets lists every game asset the material parts read (extracted
// together, so switching between them never re-extracts).
func PakAssets() []string {
	seen := map[string]bool{}
	var out []string
	add := func(a string) {
		if a != "" && !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	for _, c := range Catalog {
		for _, e := range c.Effects {
			for _, p := range e.Parts {
				if p.Copy != nil {
					add(p.Copy.From)
					for _, u := range p.Copy.Users {
						add(u)
					}
				} else {
					add(p.Asset)
				}
				for _, a := range p.VMAssets {
					add(a)
				}
			}
		}
	}
	return out
}

// Key identifies a part across the app and in saved profiles.
func Key(char, effect, part string) string { return char + "." + effect + "." + part }

// AllParts flattens the catalog keyed by Key().
func AllParts() map[string]Part {
	out := map[string]Part{}
	for _, c := range Catalog {
		for _, e := range c.Effects {
			for _, p := range e.Parts {
				out[Key(c.ID, e.ID, p.ID)] = p
			}
		}
	}
	return out
}
