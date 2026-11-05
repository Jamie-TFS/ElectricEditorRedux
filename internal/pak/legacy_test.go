package pak

import (
	"encoding/binary"
	"math"
	"testing"
)

// fakeAsset builds a legacy .uasset name map and a .uexp with vector params,
// laid out like the real files mic_patch.py handled.
func fakeAsset(params map[int][4]float32, omitted int) ([]byte, []byte, []string) {
	names := []string{"None", "BaseTexColor", "BaseTex2Color", "BaseTexture", "/Game/x/MI", "MaterialInstanceConstant"}
	ua := []byte{0xc1, 0x83, 0x2a, 0x9e, 1, 2, 3, 4, 5, 6, 7, 8}
	ua = append(ua, 5, 0, 0, 0, 'N', 'o', 'n', 'e', 0, 9, 9) // stray FString in the "summary"
	for _, n := range names {
		b := append([]byte(n), 0)
		var l [4]byte
		binary.LittleEndian.PutUint32(l[:], uint32(len(b)))
		ua = append(ua, l[:]...)
		ua = append(ua, b...)
		ua = append(ua, 0xaa, 0xbb, 0xcc, 0xdd)
	}
	ue := []byte{1, 2, 3, 4, 5, 6, 7}
	for idx := 1; idx <= 2; idx++ {
		var hdr [8]byte
		binary.LittleEndian.PutUint32(hdr[:], uint32(idx))
		ue = append(ue, 0x07, 0x00)
		ue = append(ue, hdr[:]...)
		ue = append(ue, paramTail...)
		if idx == omitted {
			// zero colour left out by the cooker: next bytes are a GUID / other data
			ue = append(ue, 0x00, 0x40, 0x20, 0xc8, 0x4f, 0x7c, 0xae, 0x43)
		} else {
			for _, f := range params[idx] {
				var b [4]byte
				binary.LittleEndian.PutUint32(b[:], math.Float32bits(f))
				ue = append(ue, b[:]...)
			}
		}
		ue = append(ue, make([]byte, 16)...)
	}
	return ua, ue, names
}

func TestFindAndSet(t *testing.T) {
	ua, ue, _ := fakeAsset(map[int][4]float32{1: {1, 0.0104, 0.142, 1}, 2: {1, 0.229, 0.293, 0}}, 0)
	names, err := LegacyNames(ua)
	if err != nil || names[1] != "BaseTexColor" {
		t.Fatalf("names: %v %v", names, err)
	}
	ps := FindParams(ue, names)
	p, ok := ps["basetexcolor"]
	if !ok || p.Value[1] != float32(0.0104) {
		t.Fatalf("BaseTexColor not found: %+v", ps)
	}
	if err := SetParam(ue, p, [4]float32{0, 1, 0, 1}); err != nil {
		t.Fatal(err)
	}
	if got := FindParams(ue, names)["basetexcolor"].Value; got != [4]float32{0, 1, 0, 1} {
		t.Fatalf("not written: %v", got)
	}
	if got := FindParams(ue, names)["basetex2color"].Value; got[1] != float32(0.229) {
		t.Fatal("neighbour param damaged")
	}
}

func TestOmittedZeroIsNotPatched(t *testing.T) {
	ua, ue, _ := fakeAsset(map[int][4]float32{1: {1, 0, 0.0333, 1}}, 2)
	names, _ := LegacyNames(ua)
	ps := FindParams(ue, names)
	if _, ok := ps["basetex2color"]; ok {
		t.Fatal("omitted (all-zero) param must not be offered for patching")
	}
	if _, ok := ps["basetexcolor"]; !ok {
		t.Fatal("present param missing")
	}
}
