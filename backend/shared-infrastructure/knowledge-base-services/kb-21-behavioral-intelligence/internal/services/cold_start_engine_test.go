package services

import (
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestAssignPhenotype_HighSelfEfficacy_PriorSuccess(t *testing.T) {
	engine := NewColdStartEngine(nil, nil)
	intake := models.IntakeProfile{
		SelfEfficacy:         0.85,
		PriorProgramSuccess:  boolPtr(true),
		EducationLevel:       "HIGH",
		AgeBand:              "30-45",
		SmartphoneLiteracy:   "HIGH",
		FirstResponseLatency: 600000, // 10 min — fast
		FamilyStructure:      "NUCLEAR",
		EmploymentStatus:     "WORKING",
	}
	result := engine.AssignPhenotype(intake)
	if result != models.PhenotypeAchiever {
		t.Errorf("high self-efficacy + prior success: got %q, want ACHIEVER", result)
	}
}

func TestAssignPhenotype_LowSelfEfficacy_PriorFailure(t *testing.T) {
	engine := NewColdStartEngine(nil, nil)
	intake := models.IntakeProfile{
		SelfEfficacy:         0.20,
		PriorProgramSuccess:  boolPtr(false),
		EducationLevel:       "LOW",
		AgeBand:              "60+",
		SmartphoneLiteracy:   "LOW",
		FirstResponseLatency: 28800000, // 8 hours — slow
		FamilyStructure:      "JOINT",
		EmploymentStatus:     "RETIRED",
	}
	result := engine.AssignPhenotype(intake)
	if result != models.PhenotypeSupportDependent {
		t.Errorf("low self-efficacy + prior failure: got %q, want SUPPORT_DEPENDENT", result)
	}
}

func TestAssignPhenotype_HighEducation_QuestionAsker(t *testing.T) {
	engine := NewColdStartEngine(nil, nil)
	intake := models.IntakeProfile{
		SelfEfficacy:         0.60,
		PriorProgramSuccess:  nil, // never tried
		EducationLevel:       "HIGH",
		AgeBand:              "45-60",
		SmartphoneLiteracy:   "HIGH",
		FirstResponseLatency: 1800000, // 30 min
		FamilyStructure:      "NUCLEAR",
		EmploymentStatus:     "WORKING",
	}
	result := engine.AssignPhenotype(intake)
	if result != models.PhenotypeKnowledgeSeeker {
		t.Errorf("high education + moderate self-efficacy: got %q, want KNOWLEDGE_SEEKER", result)
	}
}

func TestAssignPhenotype_FastResponder_Young(t *testing.T) {
	engine := NewColdStartEngine(nil, nil)
	intake := models.IntakeProfile{
		SelfEfficacy:         0.55,
		PriorProgramSuccess:  nil,
		EducationLevel:       "MODERATE",
		AgeBand:              "30-45",
		SmartphoneLiteracy:   "HIGH",
		FirstResponseLatency: 300000, // 5 min — very fast
		FamilyStructure:      "NUCLEAR",
		EmploymentStatus:     "WORKING",
	}
	result := engine.AssignPhenotype(intake)
	if result != models.PhenotypeRewardResponsive {
		t.Errorf("fast responder + young + high smartphone: got %q, want REWARD_RESPONSIVE", result)
	}
}

func TestAssignPhenotype_StableSchedule_Family(t *testing.T) {
	engine := NewColdStartEngine(nil, nil)
	intake := models.IntakeProfile{
		SelfEfficacy:         0.55,
		PriorProgramSuccess:  nil,
		EducationLevel:       "MODERATE",
		AgeBand:              "45-60",
		SmartphoneLiteracy:   "MODERATE",
		FirstResponseLatency: 3600000, // 1 hour
		FamilyStructure:      "JOINT",
		EmploymentStatus:     "WORKING",
	}
	result := engine.AssignPhenotype(intake)
	if result != models.PhenotypeRoutineBuilder {
		t.Errorf("stable schedule + joint family + 45-60: got %q, want ROUTINE_BUILDER", result)
	}
}

func TestGetPhenotypePriors_Achiever_BoostsProgress(t *testing.T) {
	engine := NewColdStartEngine(nil, nil)
	priors := engine.GetPhenotypePriors(models.PhenotypeAchiever)
	if priors == nil {
		t.Fatal("expected non-nil priors for ACHIEVER")
	}
	// Achiever: T-06 Progress has highest Alpha relative to Beta
	progressPrior := priors[models.TechProgressVisualization]
	microPrior := priors[models.TechMicroCommitment]
	progressRatio := progressPrior.Alpha / progressPrior.Beta
	microRatio := microPrior.Alpha / microPrior.Beta
	if progressRatio <= microRatio {
		t.Errorf("ACHIEVER: Progress ratio %.2f should be > Micro-Commitment ratio %.2f", progressRatio, microRatio)
	}
}

func TestGetPhenotypePriors_SupportDependent_BoostsMicroCommitment(t *testing.T) {
	engine := NewColdStartEngine(nil, nil)
	priors := engine.GetPhenotypePriors(models.PhenotypeSupportDependent)
	mcPrior := priors[models.TechMicroCommitment]
	lossPrior := priors[models.TechLossAversion]
	mcRatio := mcPrior.Alpha / mcPrior.Beta
	lossRatio := lossPrior.Alpha / lossPrior.Beta
	// Support Dependent: T-01 Micro-Commitment ranked highest
	if mcRatio <= lossRatio {
		t.Errorf("SUPPORT_DEPENDENT: MicroCommitment ratio %.2f should be > LossAversion ratio %.2f", mcRatio, lossRatio)
	}
}

func boolPtr(b bool) *bool { return &b }
