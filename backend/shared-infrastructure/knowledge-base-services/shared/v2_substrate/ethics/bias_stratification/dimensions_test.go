package bias_stratification

import "testing"

func TestAgeBand_Boundaries(t *testing.T) {
	cases := []struct {
		age  int
		want string
	}{
		{0, "under_65"},
		{64, "under_65"},
		{65, "65-74"},
		{74, "65-74"},
		{75, "75-84"},
		{84, "75-84"},
		{85, "85+"},
		{100, "85+"},
	}
	for _, c := range cases {
		got := AgeBand(c.age)
		if got != c.want {
			t.Errorf("AgeBand(%d) = %q, want %q", c.age, got, c.want)
		}
	}
}

func TestAllDimensions_StableOrder(t *testing.T) {
	want := []Dimension{
		DimAgeBand,
		DimSex,
		DimFrailtyTier,
		DimCALD,
		DimSocioecon,
		DimFacility,
	}
	if len(AllDimensions) != len(want) {
		t.Fatalf("AllDimensions length = %d, want %d", len(AllDimensions), len(want))
	}
	for i, d := range want {
		if AllDimensions[i] != d {
			t.Errorf("AllDimensions[%d] = %q, want %q", i, AllDimensions[i], d)
		}
	}
}

func TestDimensionConsts(t *testing.T) {
	cases := map[Dimension]string{
		DimAgeBand:     "age_band",
		DimSex:         "sex",
		DimFrailtyTier: "frailty_tier",
		DimCALD:        "cald_background",
		DimSocioecon:   "socioeconomic_indicator",
		DimFacility:    "facility_geography",
	}
	for d, want := range cases {
		if string(d) != want {
			t.Errorf("Dimension %v string = %q, want %q", d, string(d), want)
		}
	}
}
