// Code generated from FModel dumps of Devil Jin's chain systems. DO NOT EDIT by hand.

package pak

// The auras, rings and bursts around Devil Jin's chains (NS_EF_Swl_Chain01
// and 02). Their particle colour is a fixed red or violet, so in each
// emitter's update script the multiply that reads Initial.Color R/G/B
// points at the colour literals instead (Gain keeps the original strength).
// The chain meshes themselves are recoloured through their materials.
func init() {
	for k, v := range swlChainPatches {
		VMPatches[k] = v
		SystemPatches[v.Asset] = append(SystemPatches[v.Asset], k)
	}
}

var swlChainPatches = map[string]VMPatch{
	"NS_EF_Swl_Chain01:Burst001_1": {Asset: "NS_EF_Swl_Chain01", Len: 1362,
		Sig:    "520500005b84024d0000000000004d0000010001004d00000e0002004d00000f0003004d",
		Hash:   "a78d0228da174b4cac4210fa4d19533962d7bf20ccedb4d14f59fcd82b418390",
		LitSig: "07000000080000000000000001000000acc52737bd378635000000000000803f0200000004000000100000000d661900945c667d5ff36e3cffffff000000804b",
		Edits:  []OpEdit{{0, 1, 0x284, 0x288, false}, {265, 1, 0x2, 0x3, true}, {265, 2, 0x2, 0x284, false}, {167, 1, 0x28c, 0x288, false}, {273, 1, 0x2, 0x3, true}, {273, 2, 0x3, 0x28c, false}, {318, 1, 0x2a8, 0x288, false}, {281, 1, 0x2, 0x3, true}, {281, 2, 0x4, 0x2a8, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 1}, {Off: 8, Chan: 1, Gain: 1}, {Off: 36, Chan: 2, Gain: 1}}},
	"NS_EF_Swl_Chain01:NiagaraEmitter_2": {Asset: "NS_EF_Swl_Chain01", Len: 679,
		Sig:    "a70200005b88024d0000000000004d0000010001004e0000000002004e0000010003004d",
		Hash:   "efd0ee8fe99b684d7fc0e794f5c0523e3d3aed1e17816d56c81daf4c71087855",
		LitSig: "03000000040000000000000001000000acc52737bd378635000000000000803fcdcccc3d02000000",
		Edits:  []OpEdit{{0, 1, 0x288, 0x290, false}, {277, 1, 0x2, 0x3, true}, {277, 2, 0x4, 0x288, false}, {150, 1, 0x28c, 0x290, false}, {285, 1, 0x2, 0x3, true}, {285, 2, 0x5, 0x28c, false}, {162, 1, 0x294, 0x290, false}, {293, 1, 0x2, 0x3, true}, {293, 2, 0x6, 0x294, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 2}, {Off: 4, Chan: 1, Gain: 2}, {Off: 12, Chan: 2, Gain: 2}}},
	"NS_EF_Swl_Chain01:NiagaraEmitter_5": {Asset: "NS_EF_Swl_Chain01", Len: 1565,
		Sig:    "1d0600005b84024d0000000000004d0000010001004d0000090002004d00000a0003004d",
		Hash:   "b3465f2bcff91b8b552da674d345f3954fa109449e73ce7d8cb844ee7a63474c",
		LitSig: "07000000080000000000000001000000acc52737bd378635000000000000803f0000a04003000000db0fc940fee6db2e00000040000080bf9a99993e040000000500000000007041",
		Edits:  []OpEdit{{0, 1, 0x284, 0x28c, false}, {1034, 1, 0x0, 0x1, true}, {1034, 2, 0x3, 0x284, false}, {171, 1, 0x288, 0x28c, false}, {1042, 1, 0x0, 0x1, true}, {1042, 2, 0x4, 0x288, false}, {183, 1, 0x290, 0x28c, false}, {1050, 1, 0x0, 0x1, true}, {1050, 2, 0x5, 0x290, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 10}, {Off: 4, Chan: 1, Gain: 10}, {Off: 12, Chan: 2, Gain: 10}}},
	"NS_EF_Swl_Chain01:NiagaraEmitter_6": {Asset: "NS_EF_Swl_Chain01", Len: 1359,
		Sig:    "4f0500005b84024d0000000000004d0000110001004d0000120002004d0000130003004d",
		Hash:   "e50ee53e19423f2685bfa9d41e90bfada5e08e81784d7b72dabbb96c8e49327e",
		LitSig: "05000000060000000000000001000000acc52737bd378635000000000000803f0000f04102000000100000000d661900d23569bb5ff36e3c08000000ffffff000000804b000080bf0000003f3333333f04000000",
		Edits:  []OpEdit{{0, 1, 0x284, 0x290, false}, {266, 1, 0x2, 0x3, true}, {266, 2, 0x1, 0x284, false}, {157, 1, 0x288, 0x290, false}, {274, 1, 0x2, 0x3, true}, {274, 2, 0x2, 0x288, false}, {160, 1, 0x28c, 0x290, false}, {282, 1, 0x2, 0x3, true}, {282, 2, 0x3, 0x28c, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 1}, {Off: 4, Chan: 1, Gain: 1}, {Off: 8, Chan: 2, Gain: 1}}},
	"NS_EF_Swl_Chain01:WindRing001_1": {Asset: "NS_EF_Swl_Chain01", Len: 1331,
		Sig:    "330500005b84024d0000000000004d0000010001004d0000120002004d0000130003004d",
		Hash:   "246053e80c9d4f8077ad673ae750d15a261c59d2252993567a00de2924f31c00",
		LitSig: "05000000060000000000000001000000acc52737bd378635000000000000803f02000000100000000d6619000b30e1be5ff36e3c08000000ffffff000000804b000080bf04000000cdcccc3d",
		Edits:  []OpEdit{{0, 1, 0x284, 0x290, false}, {258, 1, 0x2, 0x3, true}, {258, 2, 0x2, 0x284, false}, {157, 1, 0x288, 0x290, false}, {266, 1, 0x2, 0x3, true}, {266, 2, 0x3, 0x288, false}, {160, 1, 0x28c, 0x290, false}, {274, 1, 0x2, 0x3, true}, {274, 2, 0x4, 0x28c, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 1}, {Off: 4, Chan: 1, Gain: 1}, {Off: 8, Chan: 2, Gain: 1}}},
	"NS_EF_Swl_Chain02:Burst001_1": {Asset: "NS_EF_Swl_Chain02", Len: 1362,
		Sig:    "520500005b84024d0000000000004d0000010001004d00000e0002004d00000f0003004d",
		Hash:   "a78d0228da174b4cac4210fa4d19533962d7bf20ccedb4d14f59fcd82b418390",
		LitSig: "07000000080000000000000001000000acc52737bd378635000000000000803f0200000004000000100000000d661900945c667d5ff36e3cffffff000000804b",
		Edits:  []OpEdit{{0, 1, 0x284, 0x288, false}, {265, 1, 0x2, 0x3, true}, {265, 2, 0x2, 0x284, false}, {167, 1, 0x28c, 0x288, false}, {273, 1, 0x2, 0x3, true}, {273, 2, 0x3, 0x28c, false}, {318, 1, 0x2a8, 0x288, false}, {281, 1, 0x2, 0x3, true}, {281, 2, 0x4, 0x2a8, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 1}, {Off: 8, Chan: 1, Gain: 1}, {Off: 36, Chan: 2, Gain: 1}}},
	"NS_EF_Swl_Chain02:RED001_14": {Asset: "NS_EF_Swl_Chain02", Len: 1711,
		Sig:    "af0600005b84024d0000000000004d0000010001004d0000090002004d00000a0003004d",
		Hash:   "a29e28a172f0c3ce4ebe940dad6cad589f4dac978a31196def893f921ac405e6",
		LitSig: "08000000090000000000000001000000acc52737bd378635000000000000803f0000a04003000000db0fc940fee6db2e00000040000080bf04000000000070419a99993e0500000006000000",
		Edits:  []OpEdit{{0, 1, 0x284, 0x28c, false}, {1110, 1, 0x0, 0x1, true}, {1110, 2, 0x3, 0x284, false}, {199, 1, 0x288, 0x28c, false}, {1118, 1, 0x0, 0x1, true}, {1118, 2, 0x4, 0x288, false}, {211, 1, 0x290, 0x28c, false}, {1126, 1, 0x0, 0x1, true}, {1126, 2, 0x5, 0x290, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 10}, {Off: 4, Chan: 1, Gain: 10}, {Off: 12, Chan: 2, Gain: 10}}},
	"NS_EF_Swl_Chain02:Ring_2": {Asset: "NS_EF_Swl_Chain02", Len: 1864,
		Sig:    "480700005b88024d0000000000004d0000010001004d00000e0002004d00000f0003004d",
		Hash:   "33359792b22cfc0cd64f3007b202c35dce7c7517411c7b28ca857e6a266d1372",
		LitSig: "06000000070000000000000001000000acc52737bd378635000000000000803f020000000000484203000000100000000d6619008dc4618d5ff36e3c08000000ffffff000000804b0000404005000000db0fc940fee6db2e00000040000080bf",
		Edits:  []OpEdit{{0, 1, 0x288, 0x290, false}, {265, 1, 0x2, 0x3, true}, {265, 2, 0x2, 0x288, false}, {164, 1, 0x28c, 0x290, false}, {273, 1, 0x2, 0x3, true}, {273, 2, 0x3, 0x28c, false}, {262, 1, 0x2a8, 0x290, false}, {281, 1, 0x2, 0x3, true}, {281, 2, 0x4, 0x2a8, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 1}, {Off: 4, Chan: 1, Gain: 1}, {Off: 32, Chan: 2, Gain: 1}}},
	"NS_EF_Swl_Chain02:Shitaji001_1": {Asset: "NS_EF_Swl_Chain02", Len: 679,
		Sig:    "a70200005b88024d0000000000004d0000010001004e0000000002004e0000010003004d",
		Hash:   "3df99bdfe1dbc7e526614c5b09634c0ddde7236a2e9dcfafc8377e78e4a124b3",
		LitSig: "03000000040000000000000001000000acc52737bd378635000000000000803f0000003fcdcc4c3d02000000",
		Edits:  []OpEdit{{0, 1, 0x288, 0x290, false}, {277, 1, 0x2, 0x3, true}, {277, 2, 0x4, 0x288, false}, {150, 1, 0x28c, 0x290, false}, {285, 1, 0x2, 0x3, true}, {285, 2, 0x5, 0x28c, false}, {162, 1, 0x294, 0x290, false}, {293, 1, 0x2, 0x3, true}, {293, 2, 0x6, 0x294, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 2}, {Off: 4, Chan: 1, Gain: 2}, {Off: 12, Chan: 2, Gain: 2}}},
}
