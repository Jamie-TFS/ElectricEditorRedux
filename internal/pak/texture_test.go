package pak

import (
	"encoding/binary"
	"os"
	"testing"
)

func TestTintBlocksKeepsBrightness(t *testing.T) {
	blk := make([]byte, 16) // BC3: 8 alpha + c0 c1 + indices
	binary.LittleEndian.PutUint16(blk[8:], pack565(1, 0.1, 0.1))
	binary.LittleEndian.PutUint16(blk[10:], pack565(0.5, 0, 0))
	tintBlocks(blk, 16, [3]float64{0, 1, 0})
	r, g, b := unpack565(binary.LittleEndian.Uint16(blk[8:]))
	if r != 0 || g != 1 || b != 0 {
		t.Fatal(r, g, b)
	}
	if _, g2, _ := unpack565(binary.LittleEndian.Uint16(blk[10:])); g2 < 0.47 || g2 > 0.54 {
		t.Fatal(g2)
	}
}

func TestDXT1KeepsMode(t *testing.T) {
	blk := make([]byte, 8)
	c0, c1 := pack565(1, 0, 0), pack565(0, 0, 0.9) // c0 > c1: 4-colour mode
	binary.LittleEndian.PutUint16(blk[0:], c0)
	binary.LittleEndian.PutUint16(blk[2:], c1)
	binary.LittleEndian.PutUint32(blk[4:], 0x0000001b) // pixel indices 3,2,1,0
	tintBlocks(blk, 8, [3]float64{0, 0, 1})
	if binary.LittleEndian.Uint16(blk[0:]) <= binary.LittleEndian.Uint16(blk[2:]) {
		t.Fatal("4-colour mode lost")
	}
}

// Runs against a real exported texture when present (not in the repo).
func TestRealThunderTexture(t *testing.T) {
	base := os.Getenv("EE_TEX")
	if base == "" {
		t.Skip("set EE_TEX to a legacy texture path without extension")
	}
	ue, _ := os.ReadFile(base + ".uexp")
	bulk, _ := os.ReadFile(base + ".ubulk")
	mips, bb, err := textureMips(ue, len(bulk))
	if err != nil || len(mips) != 10 || bb != 16 {
		t.Fatalf("%v %d", err, len(mips))
	}
	if err := TintTexture(ue, bulk, [3]float64{0, 1, 1}); err != nil {
		t.Fatal(err)
	}
}
