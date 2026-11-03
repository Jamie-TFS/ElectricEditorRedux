package fx

// Part is one recolourable layer of an effect: one or more colour curves
// that should always share a colour.
type Part struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Role  string   `json:"role"` // core | glow | outer | trail - drives the preview
	Paths []string `json:"-"`
}

type Effect struct {
	ID    string `json:"id"`
	Label string `json:"label"`
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
	beeS = "/Game/Effect/PolarisBEEwork/Character/BEE/Niagarasystem/NS_EF_Bee_Saifu02.NS_EF_Bee_Saifu02:"
	beeR = "/Game/Effect/PolarisBEEwork/Character/BEE/Niagarasystem/NS_EF_Bee_RaijinTrail.NS_EF_Bee_RaijinTrail:"
	ant  = "/Game/Effect/Polariswork/Character/ant/NiagaraSystem/NS_EF_Ant_Saifu02.NS_EF_Ant_Saifu02:"
	swl  = "/Game/Effect/Polariswork/Character/swl/NiagaraSystem/NS_EF_Swl_Saifu02.NS_EF_Swl_Saifu02:"

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
	}},
	{ID: "jin", Name: "Jin", Effects: []Effect{
		{ID: "electric", Label: "Electrics", Parts: lightningParts(ant)},
	}},
	{ID: "deviljin", Name: "Devil Jin", Effects: []Effect{
		{ID: "electric", Label: "Electrics", Parts: lightningParts(swl)},
	}},
	{ID: "reina", Name: "Reina", Effects: []Effect{
		{ID: "electric", Label: "Electrics", Parts: beamParts(zbrS, "0")},
		{ID: "kick", Label: "Electric kick", Parts: beamParts(zbrK, "0")},
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
