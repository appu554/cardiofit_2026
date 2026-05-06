package scoring

import (
	"context"
	"testing"
)

func TestStaticDrugWeightLookup_PrefixMatch(t *testing.T) {
	l := NewStaticDrugWeightLookup(map[string]DrugWeight{
		"amitriptyline": {DrugName: "amitriptyline", ACBWeight: 3},
	})
	w, found, err := l.Lookup(context.Background(), "Amitriptyline 25mg ORAL")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !found {
		t.Fatal("expected match")
	}
	if w.ACBWeight != 3 {
		t.Errorf("ACBWeight: got %d want 3", w.ACBWeight)
	}
}

func TestStaticDrugWeightLookup_NoMatch(t *testing.T) {
	l := NewStaticDrugWeightLookup(map[string]DrugWeight{
		"amitriptyline": {DrugName: "amitriptyline", ACBWeight: 3},
	})
	_, found, err := l.Lookup(context.Background(), "Paracetamol")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if found {
		t.Error("expected no match")
	}
}

func TestStaticDrugWeightLookup_LongestPrefixWins(t *testing.T) {
	l := NewStaticDrugWeightLookup(map[string]DrugWeight{
		"oxy":       {DrugName: "oxy_short", ACBWeight: 1},
		"oxybutyn":  {DrugName: "oxybutyn_long", ACBWeight: 3},
	})
	w, found, _ := l.Lookup(context.Background(), "Oxybutynin 5mg")
	if !found {
		t.Fatal("expected match")
	}
	if w.DrugName != "oxybutyn_long" {
		t.Errorf("DrugName: got %q want oxybutyn_long (longest prefix should win)", w.DrugName)
	}
}

func TestStaticDrugWeightLookup_EmptyMap(t *testing.T) {
	l := NewStaticDrugWeightLookup(nil)
	_, found, err := l.Lookup(context.Background(), "anything")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if found {
		t.Error("expected no match on empty lookup")
	}
}
