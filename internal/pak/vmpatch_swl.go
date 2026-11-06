// Code generated from the FModel dump of NS_EF_Swl_Beam. DO NOT EDIT by hand.

package pak

// The smaller DVJ eye-beam emitters around the forehead: the burst core and
// head burst (constant colours, given a literal per channel), the thunder
// (all channels taken from its red brightness chain), and the fire and glow
// sprites (the update multiply that reads Initial.Color R/G/B points at the
// colour literals instead).
func init() {
	for k, v := range swlBeamExtra {
		VMPatches[k] = v
	}
}

var swlBeamExtra = map[string]VMPatch{
	"swl-beam-burst": {Asset: "NS_EF_Swl_Beam", Len: 605,
		Sig:    "5d0200005b60055800002c016402000001005800004800000002000505a4020200a80200",
		Hash:   "6e31d2cd629302f250374cb915dcc1c6d5b1a036ebaa55fbd380fd6e7937af5d",
		LitSig: "07000000080000000000803f0900000004000000000000000000c03f05000000000020410000c8429a99993e06000000cdcc4c3ef30435bff304353f",
		Edits:  []OpEdit{{0, 1, 0x560, 0x570, false}, {195, 2, 0x588, 0x560, false}, {313, 6, 0x588, 0x560, false}},
		Sets:   []LitSet{{Off: 32, Chan: 0, Gain: 10, Orig: 10.0}, {Off: 40, Chan: 1, Gain: 10, Orig: 0.3}, {Off: 0, Chan: 2, Gain: 10, Orig: 0.3}}},
	"swl-beam-fireblue001": {Asset: "NS_EF_Swl_Beam", Len: 1974,
		Sig:    "b60700005b84024d0000000000004d00000d0001004d00000e0002004d00000f0003004d",
		Hash:   "fdf994c257499c26e54cf3a476a00bec79247fa6db4245813f94f6718cbc8e16",
		LitSig: "08000000090000000000000001000000acc52737bd378635000000000000803f030000000b0000000d6619005ff36e3c0000fa42cdcc4c3dffffff000000804b000080456666263f0000003ea470bd3e0000fa43040000000000a04005000000100000000200000016a29ada0000f04100008c420600000000000040cdcccc3d",
		Edits:  []OpEdit{{136, 1, 0x288, 0x284, false}, {1035, 1, 0x0, 0x1, true}, {1035, 2, 0x1, 0x288, false}, {139, 1, 0x28c, 0x284, false}, {1043, 1, 0x0, 0x1, true}, {1043, 2, 0x2, 0x28c, false}, {734, 1, 0x2e0, 0x284, false}, {1051, 1, 0x0, 0x1, true}, {1051, 2, 0x3, 0x2e0, false}},
		Sets:   []LitSet{{Off: 4, Chan: 0, Gain: 1, Orig: 0.0}, {Off: 8, Chan: 1, Gain: 1, Orig: 0.0}, {Off: 92, Chan: 2, Gain: 1, Orig: 0.0}}},
	"swl-beam-firecore001": {Asset: "NS_EF_Swl_Beam", Len: 1958,
		Sig:    "a60700005b84024d0000000000004d0000050001004d0000060002004d0000070003004d",
		Hash:   "5206dd7806a91a35bc312549e869788ec75faaa19b2b64b2e4516dcf95485803",
		LitSig: "08000000090000000000000001000000acc52737bd378635000000000000803f100000000d66190057fe2a575ff36e3cffffff000000804b000020410000a0c002000000040000000b0000000000fa426f12833a000080456666263f0000003ea470bd3e0000fa430500000000000040060000000300000058fe2a57000048430080a24407000000",
		Edits:  []OpEdit{{192, 1, 0x288, 0x284, false}, {1513, 1, 0x0, 0x1, true}, {1513, 2, 0x9, 0x288, false}, {195, 1, 0x28c, 0x284, false}, {1521, 1, 0x0, 0x1, true}, {1521, 2, 0xa, 0x28c, false}, {694, 1, 0x2c8, 0x284, false}, {1529, 1, 0x0, 0x1, true}, {1529, 2, 0xb, 0x2c8, false}},
		Sets:   []LitSet{{Off: 4, Chan: 0, Gain: 1, Orig: 0.0}, {Off: 8, Chan: 1, Gain: 1, Orig: 0.0}, {Off: 68, Chan: 2, Gain: 1, Orig: 0.0}}},
	"swl-beam-glow001": {Asset: "NS_EF_Swl_Beam", Len: 1999,
		Sig:    "cf0700005b84024d0000000000004d0000010001004d00000e0002004d00000f0003004d",
		Hash:   "8691ded722bb3f939b05e66870d39f1f814c82cd08d50603638763707e0b4b41",
		LitSig: "090000000a0000000000000001000000acc52737bd378635000000000000803f030000000b0000000d6619005ff36e3c0000fa42cdcc4c3d08000000ffffff000000804b000080456666263f0000003ea470bd3e0000fa43040000000000a040050000001000000002000000352a1844cdcc4c3e06000000000000400000f0c1cdcccc3d",
		Edits:  []OpEdit{{0, 1, 0x284, 0x290, false}, {1040, 1, 0x0, 0x1, true}, {1040, 2, 0x2, 0x284, false}, {143, 1, 0x288, 0x290, false}, {1048, 1, 0x0, 0x1, true}, {1048, 2, 0x3, 0x288, false}, {146, 1, 0x28c, 0x290, false}, {1056, 1, 0x0, 0x1, true}, {1056, 2, 0x4, 0x28c, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 1, Orig: 0.0}, {Off: 4, Chan: 1, Gain: 1, Orig: 0.0}, {Off: 8, Chan: 2, Gain: 1, Orig: 0.0}}},
	"swl-beam-head": {Asset: "NS_EF_Swl_Beam", Len: 1141,
		Sig:    "750400005b40055800002c01640200000100580000480000000200050594020200980200",
		Hash:   "f255e1b72900220faf63fb0d98d9c7d36d1eab75582fc85afea7b73539494b03",
		LitSig: "07000000080000000000803f0000000002000000db0fc940fee6db2e0000000000000040000080bff304353f09000000030000000000c03f00002041000020c19a99193f9a99993e0000003f00004843",
		Edits:  []OpEdit{{0, 1, 0x540, 0x56c, false}, {41, 1, 0x544, 0x56c, false}, {799, 6, 0x578, 0x540, false}, {809, 6, 0x578, 0x544, false}},
		Sets:   []LitSet{{Off: 76, Chan: 0, Gain: 200, Orig: 200.0}, {Off: 0, Chan: 1, Gain: 200, Orig: 10.0}, {Off: 4, Chan: 2, Gain: 200, Orig: 10.0}}},
	"swl-beam-glowburst": {Asset: "NS_EF_Swl_Beam", Len: 627,
		Sig:    "730200005b88024d0000000000004d0000010001004d0000060002004d0000070003004d",
		Hash:   "3ba509bde5a0889c659d0e6c69b2f0c11a60aa4ae4a83f751f18a73ed69c4854",
		LitSig: "04000000050000000000000001000000acc52737bd378635000000000000803f8fc2f53ccdcc4c3d020000000000f04103000000",
		Edits:  []OpEdit{{0, 1, 0x288, 0x290, false}, {256, 1, 0x2, 0x3, true}, {256, 2, 0x2, 0x288, false}, {129, 1, 0x28c, 0x290, false}, {264, 1, 0x2, 0x3, true}, {264, 2, 0x3, 0x28c, false}, {141, 1, 0x294, 0x290, false}, {272, 1, 0x2, 0x3, true}, {272, 2, 0x4, 0x294, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 1, Orig: 0.0}, {Off: 4, Chan: 1, Gain: 1, Orig: 0.0}, {Off: 12, Chan: 2, Gain: 1, Orig: 0.0}}},
	"swl-beam-thunder": {Asset: "NS_EF_Swl_Beam", Len: 675,
		Sig:    "a30200005b84024d0000000000004d0000090001004d00000a0002004d00000b0003004d",
		Hash:   "cda9100292b8ee044351b982b8d8970b8dc7f4d68bbca72ff2981ced98b6cd4a",
		LitSig: "04000000050000000000000001000000acc52737bd378635000000000000803f0000f04102000000",
		Edits:  []OpEdit{{0, 1, 0x284, 0x290, false}, {129, 1, 0x288, 0x290, false}, {132, 1, 0x28c, 0x290, false}, {248, 1, 0x0, 0x1, true}, {248, 2, 0x1, 0x284, false}, {256, 1, 0x0, 0x1, true}, {256, 2, 0x2, 0x288, false}, {256, 4, 0x15, 0x17, false}, {256, 6, 0x17, 0x15, false}, {264, 1, 0x0, 0x1, true}, {264, 2, 0x3, 0x28c, false}, {264, 4, 0x16, 0x17, false}, {264, 6, 0x15, 0x17, false}, {323, 6, 0x17, 0x15, false}, {333, 6, 0x15, 0x17, false}},
		Sets:   []LitSet{{Off: 0, Chan: 0, Gain: 1, Orig: 0.0}, {Off: 4, Chan: 1, Gain: 1, Orig: 0.0}, {Off: 8, Chan: 2, Gain: 1, Orig: 0.0}}},
}
