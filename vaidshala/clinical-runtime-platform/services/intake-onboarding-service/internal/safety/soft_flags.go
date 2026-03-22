package safety

import "github.com/cardiofit/intake-onboarding-service/internal/slots"

// Each SOFT_FLAG is a pure function: no I/O, no external deps, deterministic.
// Soft flags do NOT block enrollment — they raise pharmacist awareness.

// CheckSF01Elderly flags patients age >= 75 for dose adjustment awareness.
func CheckSF01Elderly(snap slots.SlotSnapshot) (bool, string, string) {
	age, ok := snap.GetInt("age")
	if !ok {
		return false, "SF-01", ""
	}
	if age >= 75 {
		return true, "SF-01", "Elderly patient (age >= 75) — dose adjustment awareness required"
	}
	return false, "SF-01", ""
}

// CheckSF02CKDModerate flags patients with eGFR 15-44 for renal dose adjustment.
func CheckSF02CKDModerate(snap slots.SlotSnapshot) (bool, string, string) {
	egfr, ok := snap.GetFloat64("egfr")
	if !ok {
		return false, "SF-02", ""
	}
	if egfr >= 15 && egfr <= 44 {
		return true, "SF-02", "CKD moderate (eGFR 15-44) — renal dose adjustment required"
	}
	return false, "SF-02", ""
}

// CheckSF03Polypharmacy flags patients with medication_count >= 5.
func CheckSF03Polypharmacy(snap slots.SlotSnapshot) (bool, string, string) {
	count, ok := snap.GetInt("medication_count")
	if !ok {
		return false, "SF-03", ""
	}
	if count >= 5 {
		return true, "SF-03", "Polypharmacy (>= 5 medications) — drug interaction review required"
	}
	return false, "SF-03", ""
}

// CheckSF04LowBMI flags patients with BMI < 18.5 for malnutrition risk.
func CheckSF04LowBMI(snap slots.SlotSnapshot) (bool, string, string) {
	bmi, ok := snap.GetFloat64("bmi")
	if !ok {
		return false, "SF-04", ""
	}
	if bmi < 18.5 {
		return true, "SF-04", "Low BMI (< 18.5) — malnutrition risk assessment required"
	}
	return false, "SF-04", ""
}

// CheckSF05InsulinUse flags patients currently on insulin.
func CheckSF05InsulinUse(snap slots.SlotSnapshot) (bool, string, string) {
	insulin, ok := snap.GetBool("insulin")
	if !ok {
		return false, "SF-05", ""
	}
	if insulin {
		return true, "SF-05", "Insulin use — hypoglycemia monitoring required"
	}
	return false, "SF-05", ""
}

// CheckSF06FallsRisk flags patients with falls_history == true OR age >= 70.
func CheckSF06FallsRisk(snap slots.SlotSnapshot) (bool, string, string) {
	falls, ok := snap.GetBool("falls_history")
	if ok && falls {
		return true, "SF-06", "Falls history — balance assessment and medication review required"
	}
	age, ok := snap.GetInt("age")
	if ok && age >= 70 {
		return true, "SF-06", "Age >= 70 — falls risk, balance assessment required"
	}
	return false, "SF-06", ""
}

// CheckSF07CognitiveImpairment flags patients with cognitive impairment.
func CheckSF07CognitiveImpairment(snap slots.SlotSnapshot) (bool, string, string) {
	impaired, ok := snap.GetBool("cognitive_impairment")
	if !ok {
		return false, "SF-07", ""
	}
	if impaired {
		return true, "SF-07", "Cognitive impairment — caregiver involvement recommended"
	}
	return false, "SF-07", ""
}

// CheckSF08NonAdherent flags patients with adherence_score < 0.5.
func CheckSF08NonAdherent(snap slots.SlotSnapshot) (bool, string, string) {
	score, ok := snap.GetFloat64("adherence_score")
	if !ok {
		return false, "SF-08", ""
	}
	if score < 0.5 {
		return true, "SF-08", "Non-adherent history (score < 0.5) — enhanced follow-up required"
	}
	return false, "SF-08", ""
}
