package safety

import "github.com/cardiofit/intake-onboarding-service/internal/slots"

// Each HARD_STOP is a pure function: no I/O, no external deps, deterministic.
// Missing slot values => rule does NOT trigger (safe default).

// CheckH1TypeOneDM blocks enrollment if diabetes_type == "T1DM".
func CheckH1TypeOneDM(snap slots.SlotSnapshot) (bool, string, string) {
	dt, ok := snap.GetString("diabetes_type")
	if !ok {
		return false, "H1", ""
	}
	if dt == "T1DM" {
		return true, "H1", "Type 1 DM — T1DM protocol differs, requires endocrinology management"
	}
	return false, "H1", ""
}

// CheckH2Pregnancy blocks enrollment if pregnant == true.
func CheckH2Pregnancy(snap slots.SlotSnapshot) (bool, string, string) {
	pregnant, ok := snap.GetBool("pregnant")
	if !ok {
		return false, "H2", ""
	}
	if pregnant {
		return true, "H2", "Pregnancy — obstetric care required, medication contraindications"
	}
	return false, "H2", ""
}

// CheckH3Dialysis blocks enrollment if dialysis == true OR eGFR < 15.
func CheckH3Dialysis(snap slots.SlotSnapshot) (bool, string, string) {
	dialysis, ok := snap.GetBool("dialysis")
	if ok && dialysis {
		return true, "H3", "Dialysis — nephrology management required"
	}
	egfr, ok := snap.GetFloat64("egfr")
	if ok && egfr < 15 {
		return true, "H3", "eGFR < 15 — CKD stage 5 / pre-dialysis, nephrology management required"
	}
	return false, "H3", ""
}

// CheckH4ActiveCancer blocks enrollment if active_cancer == true.
func CheckH4ActiveCancer(snap slots.SlotSnapshot) (bool, string, string) {
	cancer, ok := snap.GetBool("active_cancer")
	if !ok {
		return false, "H4", ""
	}
	if cancer {
		return true, "H4", "Active cancer — oncology priority, treatment interactions"
	}
	return false, "H4", ""
}

// CheckH5EGFRCritical blocks enrollment if eGFR < 15.
func CheckH5EGFRCritical(snap slots.SlotSnapshot) (bool, string, string) {
	egfr, ok := snap.GetFloat64("egfr")
	if !ok {
		return false, "H5", ""
	}
	if egfr < 15 {
		return true, "H5", "eGFR < 15 — CKD stage 5, requires nephrology specialist"
	}
	return false, "H5", ""
}

// CheckH6RecentMIStroke blocks enrollment if mi_stroke_days < 90.
func CheckH6RecentMIStroke(snap slots.SlotSnapshot) (bool, string, string) {
	days, ok := snap.GetInt("mi_stroke_days")
	if !ok {
		return false, "H6", ""
	}
	if days < 90 {
		return true, "H6", "Recent MI/stroke (< 90 days) — acute cardiac event, specialist management required"
	}
	return false, "H6", ""
}

// CheckH7HeartFailureSevere blocks enrollment if nyha_class >= 3.
func CheckH7HeartFailureSevere(snap slots.SlotSnapshot) (bool, string, string) {
	nyha, ok := snap.GetInt("nyha_class")
	if !ok {
		return false, "H7", ""
	}
	if nyha >= 3 {
		return true, "H7", "Heart failure NYHA class III/IV — HF specialist management required"
	}
	return false, "H7", ""
}

// CheckH8Child blocks enrollment if age < 18.
func CheckH8Child(snap slots.SlotSnapshot) (bool, string, string) {
	age, ok := snap.GetInt("age")
	if !ok {
		return false, "H8", ""
	}
	if age < 18 {
		return true, "H8", "Patient under 18 — pediatric protocol required"
	}
	return false, "H8", ""
}

// CheckH9BariatricSurgery blocks enrollment if bariatric_surgery_months < 12.
func CheckH9BariatricSurgery(snap slots.SlotSnapshot) (bool, string, string) {
	months, ok := snap.GetInt("bariatric_surgery_months")
	if !ok {
		return false, "H9", ""
	}
	if months < 12 {
		return true, "H9", "Bariatric surgery < 12 months ago — surgical follow-up required"
	}
	return false, "H9", ""
}

// CheckH10OrganTransplant blocks enrollment if organ_transplant == true.
func CheckH10OrganTransplant(snap slots.SlotSnapshot) (bool, string, string) {
	transplant, ok := snap.GetBool("organ_transplant")
	if !ok {
		return false, "H10", ""
	}
	if transplant {
		return true, "H10", "Organ transplant — immunosuppression management required"
	}
	return false, "H10", ""
}

// CheckH11SubstanceAbuse blocks enrollment if active_substance_abuse == true.
func CheckH11SubstanceAbuse(snap slots.SlotSnapshot) (bool, string, string) {
	abuse, ok := snap.GetBool("active_substance_abuse")
	if !ok {
		return false, "H11", ""
	}
	if abuse {
		return true, "H11", "Active substance abuse — addiction medicine referral required"
	}
	return false, "H11", ""
}
