package main

import (
	"fmt"
	"testing"
	"time"

	"app/internal/store"
)

func TestFinderNarrowing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	s := store.Load()
	e := NewEngine(s)
	go func() { // stand-in engine loop for do()
		for fn := range e.jobs {
			fn()
		}
	}()
	p := NewPakMods(s, e)
	f := NewFinder(e, p)

	var items []finderItem
	all := []int{}
	for i := 0; i < 20; i++ {
		items = append(items, finderItem{Name: fmt.Sprintf("MI_%02d", i), Asset: fmt.Sprintf("MI_%02d", i), Params: map[string][4]float32{"color": {1, 0, 0, 1}}})
		all = append(all, i)
	}
	f.st = &finderSaved{Kind: "pak", Scope: "/character/swl/", Items: items, History: [][]int{all}}
	f.apply()

	v := f.View()
	if v.Done || len(v.Colors) != 8 || v.Colors[0].Name != "3" || v.Colors[7].Name != "2" {
		t.Fatalf("round 1: %+v", v.Colors)
	}
	if got := len(p.finderEdits); got != 20 {
		t.Fatalf("finder edits %d", got)
	}
	v, err := f.Pick("#1a5cff") // blue = candidates 2, 10, 18
	if err != nil || !v.Done || len(v.Entries) != 3 {
		t.Fatalf("pick: %v %+v", err, v)
	}
	if v.Entries[0].Name != "MI_02" || v.Entries[1].Name != "MI_10" || v.Entries[2].Name != "MI_18" || v.Entries[1].ColorName != "Green" {
		t.Fatalf("round 2: %+v", v.Entries)
	}
	if _, err := f.Pick("#ffffff"); err == nil {
		t.Fatal("white isn't used in round 2")
	}
	if v = f.Back(); v.Round != 1 || len(v.Entries) != 20 {
		t.Fatalf("back: %+v", v.Round)
	}

	// survives an app restart
	f2 := NewFinder(e, p)
	if v := f2.View(); !v.Active || v.Total != 20 {
		t.Fatalf("reload: %+v", v)
	}
	f.Stop()
	if p.finderEdits != nil || NewFinder(e, p).View().Active {
		t.Fatal("stop")
	}
	_ = time.Second
}

func TestShortCurve(t *testing.T) {
	n, _ := shortCurve("/Game/Effect/Polariswork/Character/swl/NiagaraSystem/NS_EF_Swl_Saifu02.NS_EF_Swl_Saifu02:Aura_2.EventScript_0.NiagaraDataInterfaceColorCurve_0")
	if n != "NS_EF_Swl_Saifu02 · Aura_2 · ColorCurve_0" {
		t.Fatal(n)
	}
}
