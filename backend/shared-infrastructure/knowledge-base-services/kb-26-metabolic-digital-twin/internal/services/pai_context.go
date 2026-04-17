package services

import (
	"math"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ComputeContextScore maps clinical context (CKM stage, post-discharge status,
// acute illness, polypharmacy, NYHA class) into a 0-100 context dimension
// score for the Patient Acuity Index.
//
// Scoring pipeline:
//  1. CKM stage base score lookup
//  2. Additive clinical modifiers (post-discharge, acute illness, hypo, steroid, polypharmacy)
//  3. NYHA class multiplicative amplifier
//  4. Cap at ContextMaxScore (100)
func ComputeContextScore(input models.PAIDimensionInput, cfg *PAIConfig) float64 {
	// 1. CKM stage base
	score := cfg.ContextCKMStageBase[input.CKMStage] // 0 if not found

	// 2. Additive modifiers
	if input.IsPostDischarge30d {
		score += cfg.ContextPostDischarge30d
	}
	if input.IsAcutelyIll {
		score += cfg.ContextAcuteIllness
	}
	if input.HasRecentHypo {
		score += cfg.ContextRecentHypo
	}
	if input.ActiveSteroidCourse {
		score += cfg.ContextActiveSteroid
	}
	if input.Age >= cfg.ContextPolypharmacyAge && input.MedicationCount >= cfg.ContextPolypharmacyMeds {
		score += cfg.ContextPolypharmacyElderly
	}

	// 3. NYHA amplifier
	if amp, ok := cfg.ContextNYHAAmplifier[input.NYHAClass]; ok {
		score *= amp
	}

	// 4. Cap at max
	return math.Min(score, cfg.ContextMaxScore)
}
