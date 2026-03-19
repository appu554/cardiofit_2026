package services

import (
	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ColdStartEngine assigns behavioral phenotypes to new patients based on intake signals (E1 §3).
// Phenotype assignment replaces uniform population priors with calibrated cluster-specific priors,
// reducing the Bayesian exploration period from 4-5 weeks to 1-2 weeks.
type ColdStartEngine struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewColdStartEngine(db *gorm.DB, logger *zap.Logger) *ColdStartEngine {
	return &ColdStartEngine{db: db, logger: logger}
}

// AssignPhenotype scores intake signals against each phenotype cluster and returns the best match.
// Scoring weights from BCE v2.0 spec §3.2 Table.
func (cs *ColdStartEngine) AssignPhenotype(intake models.IntakeProfile) models.ColdStartPhenotype {
	scores := map[models.ColdStartPhenotype]float64{
		models.PhenotypeAchiever:         cs.scoreAchiever(intake),
		models.PhenotypeRoutineBuilder:   cs.scoreRoutineBuilder(intake),
		models.PhenotypeKnowledgeSeeker:  cs.scoreKnowledgeSeeker(intake),
		models.PhenotypeSupportDependent: cs.scoreSupportDependent(intake),
		models.PhenotypeRewardResponsive: cs.scoreRewardResponsive(intake),
	}

	best := models.PhenotypeRoutineBuilder // safest default
	bestScore := -1.0
	for phenotype, score := range scores {
		if score > bestScore {
			bestScore = score
			best = phenotype
		}
	}
	return best
}

// GetPhenotypePriors returns calibrated Alpha/Beta priors for a phenotype cluster.
func (cs *ColdStartEngine) GetPhenotypePriors(phenotype models.ColdStartPhenotype) map[models.TechniqueID]models.TechniquePrior {
	if priors, ok := phenotypePriorLibrary[phenotype]; ok {
		return priors
	}
	// Fallback: return uniform population priors
	result := make(map[models.TechniqueID]models.TechniquePrior)
	for _, tech := range models.AllTechniques() {
		alpha, beta := GetDefaultPriors(tech)
		result[tech] = models.TechniquePrior{Alpha: alpha, Beta: beta}
	}
	return result
}

// GetOrAssignPhenotype loads existing assignment or creates one from intake profile.
func (cs *ColdStartEngine) GetOrAssignPhenotype(patientID string) (models.ColdStartPhenotype, error) {
	if cs.db == nil {
		return models.PhenotypeRoutineBuilder, nil
	}

	var intake models.IntakeProfile
	err := cs.db.Where("patient_id = ?", patientID).First(&intake).Error
	if err != nil {
		// No intake profile → can't assign phenotype, use default
		return models.PhenotypeRoutineBuilder, nil
	}

	return cs.AssignPhenotype(intake), nil
}

// --- Scoring functions (spec §3.2 weights) ---

func (cs *ColdStartEngine) scoreAchiever(i models.IntakeProfile) float64 {
	// Weights: self-efficacy 0.30, education 0.15, response speed 0.10, prior success 0.50.
	// PriorProgramSuccess is the dominant signal (spec §3.2): a validated track record
	// overrides all other Achiever signals, giving a max score of 1.05 vs RR max of 1.0.
	score := 0.30*i.SelfEfficacy + 0.15*educationScore(i.EducationLevel) + 0.10*responseSpeedScore(i.FirstResponseLatency)
	if i.PriorProgramSuccess != nil && *i.PriorProgramSuccess {
		score += 0.50
	}
	return score
}

func (cs *ColdStartEngine) scoreRoutineBuilder(i models.IntakeProfile) float64 {
	score := 0.3 * familySupportScore(i.FamilyStructure)
	// Stability proxy: employed + middle-aged + moderate self-efficacy
	if i.EmploymentStatus == "WORKING" {
		score += 0.2
	}
	if i.AgeBand == "45-60" {
		score += 0.2
	}
	score += 0.3 * (1.0 - noveltyPreferenceProxy(i))
	return score
}

func (cs *ColdStartEngine) scoreKnowledgeSeeker(i models.IntakeProfile) float64 {
	score := 0.5*educationScore(i.EducationLevel) + 0.2*responseSpeedScore(i.FirstResponseLatency)
	// Moderate self-efficacy (not high enough for Achiever, not low for Support Dependent)
	if i.SelfEfficacy >= 0.40 && i.SelfEfficacy <= 0.75 {
		score += 0.3
	}
	return score
}

func (cs *ColdStartEngine) scoreSupportDependent(i models.IntakeProfile) float64 {
	score := 0.4 * (1.0 - i.SelfEfficacy) // low self-efficacy → high score
	if i.PriorProgramSuccess != nil && !*i.PriorProgramSuccess {
		score += 0.3 // prior failure
	}
	score += 0.3 * familyDependenceScore(i.FamilyStructure, i.AgeBand)
	return score
}

func (cs *ColdStartEngine) scoreRewardResponsive(i models.IntakeProfile) float64 {
	score := 0.4*responseSpeedScore(i.FirstResponseLatency) + 0.3*smartphoneLiteracyScore(i.SmartphoneLiteracy)
	if i.AgeBand == "30-45" {
		score += 0.3 // younger demographic
	} else if i.AgeBand == "45-60" {
		score += 0.1
	}
	return score
}

// --- Signal normalization helpers ---

func educationScore(level string) float64 {
	switch level {
	case "HIGH":
		return 1.0
	case "MODERATE":
		return 0.5
	default:
		return 0.2
	}
}

func responseSpeedScore(latencyMs int64) float64 {
	// <30min = 1.0, 30min-2hr = 0.7, 2hr-4hr = 0.4, >4hr = 0.1
	switch {
	case latencyMs <= 0:
		return 0.5 // unknown
	case latencyMs <= 1800000: // 30 min
		return 1.0
	case latencyMs <= 7200000: // 2 hours
		return 0.7
	case latencyMs <= 14400000: // 4 hours
		return 0.4
	default:
		return 0.1
	}
}

func smartphoneLiteracyScore(level string) float64 {
	switch level {
	case "HIGH":
		return 1.0
	case "MODERATE":
		return 0.5
	default:
		return 0.2
	}
}

func familySupportScore(structure string) float64 {
	switch structure {
	case "JOINT":
		return 1.0
	case "NUCLEAR":
		return 0.5
	default:
		return 0.2
	}
}

func familyDependenceScore(structure string, ageBand string) float64 {
	score := 0.0
	if structure == "JOINT" {
		score += 0.5
	}
	if ageBand == "60+" {
		score += 0.5
	} else if ageBand == "45-60" {
		score += 0.2
	}
	return score
}

func noveltyPreferenceProxy(i models.IntakeProfile) float64 {
	// Proxy: fast responders + high smartphone literacy → novelty-seeking
	speed := responseSpeedScore(i.FirstResponseLatency)
	literacy := smartphoneLiteracyScore(i.SmartphoneLiteracy)
	return (speed + literacy) / 2.0
}

// --- Phenotype-specific calibrated prior library (spec §3.2) ---

var phenotypePriorLibrary = map[models.ColdStartPhenotype]map[models.TechniqueID]models.TechniquePrior{
	models.PhenotypeAchiever: {
		models.TechProgressVisualization:  {Alpha: 3.0, Beta: 1.5}, // strongest
		models.TechSocialNorms:            {Alpha: 2.5, Beta: 1.5},
		models.TechLossAversion:           {Alpha: 2.5, Beta: 2.0},
		models.TechMicroCommitment:        {Alpha: 2.0, Beta: 2.0},
		models.TechHabitStacking:          {Alpha: 2.0, Beta: 2.0},
		models.TechMicroEducation:         {Alpha: 2.0, Beta: 2.0},
		models.TechEnvironmentRestructure: {Alpha: 1.5, Beta: 2.0},
		models.TechImplementIntention:     {Alpha: 1.5, Beta: 2.0},
		models.TechCostAwareSubstitution:  {Alpha: 1.5, Beta: 2.5},
		models.TechFamilyInclusion:        {Alpha: 1.5, Beta: 2.5},
		models.TechRecoveryProtocol:       {Alpha: 2.5, Beta: 1.5},
		models.TechKinshipTone:            {Alpha: 1.5, Beta: 2.0},
	},
	models.PhenotypeRoutineBuilder: {
		models.TechHabitStacking:          {Alpha: 3.0, Beta: 1.5}, // strongest
		models.TechEnvironmentRestructure: {Alpha: 2.5, Beta: 1.5},
		models.TechImplementIntention:     {Alpha: 2.5, Beta: 2.0},
		models.TechFamilyInclusion:        {Alpha: 2.5, Beta: 2.0},
		models.TechMicroCommitment:        {Alpha: 2.0, Beta: 2.0},
		models.TechMicroEducation:         {Alpha: 2.0, Beta: 2.0},
		models.TechProgressVisualization:  {Alpha: 2.0, Beta: 2.0},
		models.TechCostAwareSubstitution:  {Alpha: 2.0, Beta: 2.0},
		models.TechSocialNorms:            {Alpha: 1.5, Beta: 2.0},
		models.TechLossAversion:           {Alpha: 1.5, Beta: 2.5},
		models.TechRecoveryProtocol:       {Alpha: 2.5, Beta: 1.5},
		models.TechKinshipTone:            {Alpha: 2.0, Beta: 1.5},
	},
	models.PhenotypeKnowledgeSeeker: {
		models.TechMicroEducation:         {Alpha: 3.0, Beta: 1.5}, // strongest
		models.TechProgressVisualization:  {Alpha: 2.5, Beta: 1.5},
		models.TechLossAversion:           {Alpha: 2.5, Beta: 2.0},
		models.TechCostAwareSubstitution:  {Alpha: 2.0, Beta: 2.0},
		models.TechMicroCommitment:        {Alpha: 2.0, Beta: 2.0},
		models.TechHabitStacking:          {Alpha: 2.0, Beta: 2.0},
		models.TechImplementIntention:     {Alpha: 2.0, Beta: 2.0},
		models.TechEnvironmentRestructure: {Alpha: 1.5, Beta: 2.0},
		models.TechSocialNorms:            {Alpha: 1.5, Beta: 2.5},
		models.TechFamilyInclusion:        {Alpha: 1.5, Beta: 2.5},
		models.TechRecoveryProtocol:       {Alpha: 2.5, Beta: 1.5},
		models.TechKinshipTone:            {Alpha: 1.5, Beta: 2.0},
	},
	models.PhenotypeSupportDependent: {
		models.TechMicroCommitment:        {Alpha: 3.0, Beta: 1.5}, // strongest — tiny wins
		models.TechFamilyInclusion:        {Alpha: 3.0, Beta: 1.5}, // external accountability
		models.TechHabitStacking:          {Alpha: 2.5, Beta: 2.0},
		models.TechRecoveryProtocol:       {Alpha: 2.5, Beta: 1.5},
		models.TechKinshipTone:            {Alpha: 2.5, Beta: 1.5}, // warm elder tone
		models.TechMicroEducation:         {Alpha: 2.0, Beta: 2.0},
		models.TechEnvironmentRestructure: {Alpha: 2.0, Beta: 2.0},
		models.TechCostAwareSubstitution:  {Alpha: 2.0, Beta: 2.0},
		models.TechImplementIntention:     {Alpha: 1.5, Beta: 2.0},
		models.TechProgressVisualization:  {Alpha: 1.5, Beta: 2.5},
		models.TechSocialNorms:            {Alpha: 1.0, Beta: 3.0}, // suppressed
		models.TechLossAversion:           {Alpha: 1.0, Beta: 3.0}, // too harsh
	},
	models.PhenotypeRewardResponsive: {
		models.TechProgressVisualization:  {Alpha: 3.0, Beta: 1.5}, // gamification core
		models.TechSocialNorms:            {Alpha: 2.5, Beta: 1.5}, // competition
		models.TechMicroCommitment:        {Alpha: 2.5, Beta: 2.0},
		models.TechHabitStacking:          {Alpha: 2.0, Beta: 2.0},
		models.TechLossAversion:           {Alpha: 2.5, Beta: 2.0},
		models.TechMicroEducation:         {Alpha: 1.5, Beta: 2.0},
		models.TechEnvironmentRestructure: {Alpha: 1.5, Beta: 2.0},
		models.TechImplementIntention:     {Alpha: 2.0, Beta: 2.0},
		models.TechCostAwareSubstitution:  {Alpha: 1.5, Beta: 2.5},
		models.TechFamilyInclusion:        {Alpha: 1.5, Beta: 2.5},
		models.TechRecoveryProtocol:       {Alpha: 2.5, Beta: 1.5},
		models.TechKinshipTone:            {Alpha: 1.5, Beta: 2.0},
	},
}
