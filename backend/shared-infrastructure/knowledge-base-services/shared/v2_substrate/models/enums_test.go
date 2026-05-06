package models

import "testing"

func TestCareIntensityIsValid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"palliative", true},
		{"comfort", true},
		{"active", true},
		{"rehabilitation", true},
		{"", false},
		{"unknown", false},
	}
	for _, c := range cases {
		if got := IsValidCareIntensity(c.in); got != c.want {
			t.Errorf("IsValidCareIntensity(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRoleKindIsValid(t *testing.T) {
	valid := []string{"RN", "EN", "NP", "DRNP", "GP", "pharmacist", "ACOP", "PCW", "SDM", "family", "ATSIHP", "medical_practitioner", "dentist"}
	for _, k := range valid {
		if !IsValidRoleKind(k) {
			t.Errorf("IsValidRoleKind(%q) = false, want true", k)
		}
	}
	if IsValidRoleKind("nurse") {
		t.Errorf("IsValidRoleKind(\"nurse\") = true, want false (must use RN/EN)")
	}
}

func TestResidentStatusIsValid(t *testing.T) {
	valid := []string{"active", "deceased", "transferred", "discharged"}
	for _, s := range valid {
		if !IsValidResidentStatus(s) {
			t.Errorf("IsValidResidentStatus(%q) = false, want true", s)
		}
	}
	if IsValidResidentStatus("inactive") {
		t.Errorf("inactive should not be valid")
	}
}

func TestMedicineUseStatusIsValid(t *testing.T) {
	valid := []string{"active", "paused", "ceased", "completed"}
	for _, s := range valid {
		if !IsValidMedicineUseStatus(s) {
			t.Errorf("IsValidMedicineUseStatus(%q) = false, want true", s)
		}
	}
	if IsValidMedicineUseStatus("done") {
		t.Errorf("IsValidMedicineUseStatus(\"done\") = true, want false")
	}
}

func TestIntentCategoryIsValid(t *testing.T) {
	valid := []string{"therapeutic", "preventive", "symptomatic", "trial", "deprescribing", "unspecified"}
	for _, c := range valid {
		if !IsValidIntentCategory(c) {
			t.Errorf("IsValidIntentCategory(%q) = false, want true", c)
		}
	}
	if IsValidIntentCategory("curative") {
		t.Errorf("IsValidIntentCategory(\"curative\") = true, want false")
	}
}

func TestTargetKindIsValid(t *testing.T) {
	valid := []string{"BP_threshold", "completion_date", "symptom_resolution", "HbA1c_band", "open"}
	for _, k := range valid {
		if !IsValidTargetKind(k) {
			t.Errorf("IsValidTargetKind(%q) = false, want true", k)
		}
	}
	if IsValidTargetKind("LDL_target") {
		t.Errorf("IsValidTargetKind(\"LDL_target\") = true, want false (must add to enum first)")
	}
}

func TestStopTriggerIsValid(t *testing.T) {
	valid := []string{"adverse_event", "target_achieved", "review_due", "patient_request",
		"carer_request", "completion", "interaction"}
	for _, s := range valid {
		if !IsValidStopTrigger(s) {
			t.Errorf("IsValidStopTrigger(%q) = false, want true", s)
		}
	}
	if IsValidStopTrigger("died") {
		t.Errorf("IsValidStopTrigger(\"died\") = true, want false")
	}
}

func TestIsValidObservationKind(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"vital", true},
		{"lab", true},
		{"behavioural", true},
		{"mobility", true},
		{"weight", true},
		{"", false},
		{"behavioral", false}, // US spelling rejected — AU spelling only
		{"unknown", false},
	}
	for _, c := range cases {
		if got := IsValidObservationKind(c.in); got != c.want {
			t.Errorf("IsValidObservationKind(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestIsValidDeltaFlag(t *testing.T) {
	valid := []string{"within_baseline", "elevated", "severely_elevated", "low", "severely_low", "no_baseline"}
	for _, f := range valid {
		if !IsValidDeltaFlag(f) {
			t.Errorf("IsValidDeltaFlag(%q) = false, want true", f)
		}
	}
	if IsValidDeltaFlag("") {
		t.Errorf("IsValidDeltaFlag(\"\") = true, want false")
	}
	if IsValidDeltaFlag("normal") {
		t.Errorf("IsValidDeltaFlag(\"normal\") = true, want false (must use within_baseline)")
	}
}
