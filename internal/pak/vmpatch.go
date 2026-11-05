package pak

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// VMPatch recolours a CPU Niagara emitter whose particle colour is built
// from constants in its cooked VectorVM script (e.g. the DVJ beam, whose
// Initial.Color is (10, 0.3, 0.3): red, with green and blue sharing one
// literal).
//
// Same-length rewrite: Edits repoint u16 operands so each colour channel
// reads its own literal (spare slots come from profiling-only stat-scope
// ids, freed by pointing those scopes at another id), then Sets write the
// chosen colour into the literals. A patch can hold several colour slots
// (spawn groups) that take different colours.
//
// Hash pins the exact game bytecode, so a game update that changes the
// script makes the patch refuse instead of writing garbage.
type VMPatch struct {
	Asset  string
	Len    int
	Sig    string // hex: int32 length + first bytes of the bytecode
	Hash   string // sha256 of the original bytecode
	LitSig string // hex: the script's whole literal table
	Edits  []OpEdit
	Sets   []LitSet
}

// OpEdit changes the u16 (or, with Byte, the byte) at At+Pos (bytecode
// offsets) from Want to Val.
type OpEdit struct {
	At, Pos   int
	Want, Val uint16
	Byte      bool
}

// LitSet writes colour channel Chan of colour slot Slot, times Gain, to the
// literal at byte Off. Orig is written when that slot has no colour, so a
// partly recoloured patch keeps the game's look for the rest.
type LitSet struct {
	Off, Chan, Slot int
	Gain, Orig      float64
}

var VMPatches = map[string]VMPatch{
	// DVJ beam lightning (NiagaraEmitter_4 "Beam" spawn): Initial.Color = (10, 0.3, 0.3), x5 in update
	"swl-beam-bolt": {
		Asset:  "NS_EF_Swl_Beam",
		Len:    1593,
		Sig:    "390600005b50055800002c0164020000010058000048000000020005059c020200a00200000201040000000200030200000800030006032c0584020300000006",
		Hash:   "409e66b393832860943b40378a44378d6d73755c90b17646872e42b3f90f6374",
		LitSig: "0a0000000b0000000000803f0100000090c2f53c0ad7a33c0000000002000000db0fc940fee6db2e0000000000000040000080bff30435bff304353f0c00000004000000acc52737bd37863505000000000020410000a0409a99993e100000000d6619002b02676a5ff36e3c08000000ffffff000000804b0000004109000000",
		Edits: []OpEdit{
			{At: 79, Pos: 1, Want: 0x554, Val: 0x550},
			{At: 741, Pos: 2, Want: 0x5a8, Val: 0x554},
			{At: 1231, Pos: 6, Want: 0x5a8, Val: 0x554},
		},
		Sets: []LitSet{
			{Off: 80, Chan: 0, Slot: 0, Gain: 10, Orig: 10.0},
			{Off: 88, Chan: 1, Slot: 0, Gain: 10, Orig: 0.30000001192092896},
			{Off: 4, Chan: 2, Slot: 0, Gain: 10, Orig: 0.30000001192092896},
		},
	},

	// DVJ beam core (NiagaraEmitter_8 "BeamCoreGlow001" spawn), Initial.Color per spawn group: slot 0 = glow (3, 0.28, 0.28), slot 1 = the straight beam (10, 0.3, 0.3), x50 in update, and the default group (1, 0, 0) at its start
	"swl-beam-e8": {
		Asset:  "NS_EF_Swl_Beam",
		Len:    2339,
		Sig:    "230900005b60055e0000000001005802002c016402020003005802004800020004000505a8020400ac0202000201040002000400030202000800050006033805",
		Hash:   "66256367b29bd7ad95970503bf4695364fb48ca6a96075212a93a8f002dc3ad1",
		LitSig: "0b0000000c000000000000000000010001000000000040400000803f0000903e000000000200000003000000000020419a99993e9a99993fdb0fc940fee6db2e00000040000080bff30435bff304353f0d000000060000000000c03f3333f33fcdcccc3d00004842cdcc4c3d0000a04107000000100000000d661900f7a60bb75ff36e3c08000000ffffff000000804b0000804000009041000010410a000000",
		Edits: []OpEdit{
			{At: 89, Pos: 4, Want: 0x568, Val: 0x580},
			{At: 105, Pos: 4, Want: 0x568, Val: 0x580},
			{At: 516, Pos: 4, Want: 0x568, Val: 0x580},
			{At: 915, Pos: 4, Want: 0x568, Val: 0x580},
			{At: 931, Pos: 2, Want: 0x568, Val: 0x580},
			{At: 1049, Pos: 4, Want: 0x568, Val: 0x580},
			{At: 1119, Pos: 4, Want: 0x568, Val: 0x580},
			{At: 1541, Pos: 4, Want: 0x568, Val: 0x580},
			{At: 129, Pos: 4, Want: 0x574, Val: 0x568},
			{At: 0, Pos: 1, Want: 0x560, Val: 0x570},
			{At: 149, Pos: 4, Want: 0x57c, Val: 0x560},
			{At: 86, Pos: 1, Want: 0x564, Val: 0x570},
			{At: 235, Pos: 4, Want: 0x590, Val: 0x564},
			{At: 940, Pos: 1, Want: 0x5b0, Val: 0x570},
			{At: 245, Pos: 4, Want: 0x590, Val: 0x5b0},
			{At: 975, Pos: 1, Want: 0x5b4, Val: 0x570},
			{At: 129, Pos: 6, Want: 0x578, Val: 0x5b4},
			{At: 1223, Pos: 1, Want: 0x5d0, Val: 0x570},
			{At: 139, Pos: 6, Want: 0x580, Val: 0x5d0},
			{At: 1789, Pos: 1, Want: 0x5fc, Val: 0x570},
			{At: 149, Pos: 6, Want: 0x580, Val: 0x5fc},
		},
		Sets: []LitSet{
			{Off: 8, Chan: 0, Slot: 0, Gain: 3, Orig: 3.0},
			{Off: 28, Chan: 1, Slot: 0, Gain: 3, Orig: 0.28125},
			{Off: 0, Chan: 2, Slot: 0, Gain: 3, Orig: 0.28125},
			{Off: 44, Chan: 0, Slot: 1, Gain: 10, Orig: 10.0},
			{Off: 4, Chan: 1, Slot: 1, Gain: 10, Orig: 0.30000001192092896},
			{Off: 80, Chan: 2, Slot: 1, Gain: 10, Orig: 0.30000001192092896},
			{Off: 84, Chan: 0, Slot: 1, Gain: 1, Orig: 1.0},
			{Off: 112, Chan: 1, Slot: 1, Gain: 1, Orig: 0.0},
			{Off: 156, Chan: 2, Slot: 1, Gain: 1, Orig: 0.0},
		},
	},

	// DVJ beam light line (NiagaraEmitter_9 "BeamLight" spawn): Initial.Color = (3, 0.1, 0.1), x50 in update
	"swl-beam-light": {
		Asset:  "NS_EF_Swl_Beam",
		Len:    513,
		Sig:    "010200005b60055800002c016402000001005800004800000002000505a4020200a8020000020104000000020003020000080003000603340584020300000006",
		Hash:   "c783a10eb7e8fb735de0498a03cd411a21d0417eb58c2de1d8b392db6733670b",
		LitSig: "06000000070000000000803f0800000003000000000000000000c03f040000000000404000004842cdcccc3dcecccc3ecdcc4c3f0500000000007041",
		Edits: []OpEdit{
			{At: 79, Pos: 1, Want: 0x564, Val: 0x560},
			{At: 195, Pos: 2, Want: 0x588, Val: 0x564},
			{At: 331, Pos: 6, Want: 0x588, Val: 0x564},
		},
		Sets: []LitSet{
			{Off: 32, Chan: 0, Slot: 0, Gain: 3, Orig: 3.0},
			{Off: 40, Chan: 1, Slot: 0, Gain: 3, Orig: 0.10000000149011612},
			{Off: 4, Chan: 2, Slot: 0, Gain: 3, Orig: 0.10000000149011612},
		},
	},
}

// VMKey splits "name#slot" (slot 0 when there's no #).
func VMKey(k string) (string, int) {
	if i := strings.LastIndexByte(k, '#'); i >= 0 {
		if n, err := strconv.Atoi(k[i+1:]); err == nil {
			return k[:i], n
		}
	}
	return k, 0
}

// applyVMs groups Edit.VM by patch and applies each once.
func applyVMs(uexp []byte, vm map[string][3]float64) (int, error) {
	byPatch := map[string]map[int][3]float64{}
	for k, c := range vm {
		name, slot := VMKey(k)
		if byPatch[name] == nil {
			byPatch[name] = map[int][3]float64{}
		}
		byPatch[name][slot] = c
	}
	names := make([]string, 0, len(byPatch))
	for n := range byPatch {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		p, ok := VMPatches[n]
		if !ok {
			return 0, fmt.Errorf("unknown script patch %s", n)
		}
		if err := ApplyVM(uexp, p, byPatch[n]); err != nil {
			return 0, fmt.Errorf("%s: %v", n, err)
		}
	}
	return len(names), nil
}

// ApplyVM patches one script in uexp (in place). cols maps colour slot to
// colour; slots without one keep the game's values.
func ApplyVM(uexp []byte, p VMPatch, cols map[int][3]float64) error {
	sig, _ := hex.DecodeString(p.Sig)
	// Several scripts can start alike: the ones whose bytes hash right are
	// it. Emitters with byte-identical scripts are all patched.
	var hits []int
	for from := 0; ; {
		i := bytes.Index(uexp[from:], sig)
		if i < 0 {
			break
		}
		i += from
		from = i + 1
		if int(binary.LittleEndian.Uint32(uexp[i:])) != p.Len || i+4+p.Len > len(uexp) {
			continue
		}
		sum := sha256.Sum256(uexp[i+4 : i+4+p.Len])
		if hex.EncodeToString(sum[:]) == p.Hash {
			hits = append(hits, i)
		}
	}
	if len(hits) == 0 {
		return fmt.Errorf("script bytecode not found (game update?)")
	}
	done := 0
	for _, at := range hits {
		err := applyVMAt(uexp, p, cols, at)
		if errors.Is(err, errNoLits) {
			continue // same code, different constants: another emitter's
		}
		if err != nil {
			return err
		}
		done++
	}
	if done == 0 {
		return errNoLits
	}
	return nil
}

var errNoLits = errors.New("script literals not found")

func applyVMAt(uexp []byte, p VMPatch, cols map[int][3]float64, at int) error {
	bc := uexp[at+4 : at+4+p.Len]
	// the literal table is serialized after its bytecode; take the first
	// copy after it (similar emitters can share identical tables)
	lsig, _ := hex.DecodeString(p.LitSig)
	li := bytes.Index(uexp[at+4+p.Len:], lsig)
	if li < 0 || li > 4096 { // it follows its own bytecode closely
		return errNoLits
	}
	lat := at + 4 + p.Len + li
	nb := append([]byte(nil), bc...)
	for _, e := range p.Edits {
		if e.At+e.Pos+2 > len(nb) {
			return fmt.Errorf("operand at %d outside the script", e.At)
		}
		if e.Byte {
			if uint16(nb[e.At+e.Pos]) != e.Want {
				return fmt.Errorf("unexpected op mask at %d", e.At)
			}
			nb[e.At+e.Pos] = byte(e.Val)
			continue
		}
		if got := binary.LittleEndian.Uint16(nb[e.At+e.Pos:]); got != e.Want {
			return fmt.Errorf("unexpected operand at %d", e.At)
		}
		binary.LittleEndian.PutUint16(nb[e.At+e.Pos:], e.Val)
	}
	for _, l := range p.Sets {
		if l.Off < 0 || l.Off+4 > len(lsig) {
			return fmt.Errorf("literal %d outside the table", l.Off)
		}
	}
	copy(bc, nb)
	for _, l := range p.Sets {
		v := l.Orig
		if c, ok := cols[l.Slot]; ok {
			v = c[l.Chan] * l.Gain
		}
		binary.LittleEndian.PutUint32(uexp[lat+l.Off:], math.Float32bits(float32(v)))
	}
	return nil
}
