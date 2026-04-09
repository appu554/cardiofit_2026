package services

import (
	"fmt"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// EvidenceChain — structured clinical evidence for therapeutic inertia
// ---------------------------------------------------------------------------

// EvidenceChain captures the clinical narrative, quantified risk, and
// guideline reference backing a therapeutic inertia verdict.
type EvidenceChain struct {
	Summary       string `json:"summary"`
	RiskStatement string `json:"risk_statement"`
	GuidelineRef  string `json:"guideline_ref"`
}

// ---------------------------------------------------------------------------
// BuildEvidenceChain — construct evidence chain from an inertia verdict
// ---------------------------------------------------------------------------

// BuildEvidenceChain generates a structured evidence chain for a single
// inertia verdict, including domain-specific risk quantification from
// landmark trials and current guideline references.
func BuildEvidenceChain(verdict models.InertiaVerdict) EvidenceChain {
	switch verdict.Domain {
	case models.DomainGlycaemic:
		return buildGlycaemicEvidence(verdict)
	case models.DomainHemodynamic:
		return buildHemodynamicEvidence(verdict)
	default:
		return buildGenericEvidence(verdict)
	}
}

// ---------------------------------------------------------------------------
// Domain-specific evidence builders
// ---------------------------------------------------------------------------

func buildGlycaemicEvidence(v models.InertiaVerdict) EvidenceChain {
	weeks := v.InertiaDurationDays / 7

	summary := fmt.Sprintf(
		"HbA1c %.1f%% above target %.1f%% for %d weeks with no medication change",
		v.CurrentValue, v.TargetValue, weeks,
	)

	riskStatement := "Each year at HbA1c >7%%: +37%% microvascular risk (retinopathy, nephropathy, neuropathy), +14%% MI risk (UKPDS 35)"

	guidelineRef := "ADA Standards of Care 2025, Section 9 — Pharmacologic Approaches to Glycaemic Treatment"

	return EvidenceChain{
		Summary:       summary,
		RiskStatement: riskStatement,
		GuidelineRef:  guidelineRef,
	}
}

func buildHemodynamicEvidence(v models.InertiaVerdict) EvidenceChain {
	weeks := v.InertiaDurationDays / 7

	summary := fmt.Sprintf(
		"SBP %.0f mmHg above target %.0f mmHg for %d weeks with no medication change",
		v.CurrentValue, v.TargetValue, weeks,
	)

	riskStatement := "Each 10 mmHg above target: +30%% stroke risk, +20%% CHD risk (Prospective Studies Collaboration)"

	guidelineRef := "ISH 2020 Global Hypertension Practice Guidelines; ESC/ESH 2024 Arterial Hypertension Guidelines"

	return EvidenceChain{
		Summary:       summary,
		RiskStatement: riskStatement,
		GuidelineRef:  guidelineRef,
	}
}

func buildGenericEvidence(v models.InertiaVerdict) EvidenceChain {
	weeks := v.InertiaDurationDays / 7

	summary := fmt.Sprintf(
		"%s domain: value %.1f above target %.1f for %d weeks with no medication change",
		string(v.Domain), v.CurrentValue, v.TargetValue, weeks,
	)

	return EvidenceChain{
		Summary:       summary,
		RiskStatement: "Prolonged time above target increases cumulative organ damage risk",
		GuidelineRef:  "Clinical practice guidelines for " + string(v.Domain) + " management",
	}
}
