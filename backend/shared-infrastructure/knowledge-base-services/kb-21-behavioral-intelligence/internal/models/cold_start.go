package models

// ColdStartPhenotype represents one of 5 behavioral phenotype clusters (BCE v2.0 E1 §3.2).
// Assigned on Day 1 from intake signals to replace uniform population priors.
type ColdStartPhenotype string

const (
	PhenotypeAchiever         ColdStartPhenotype = "ACHIEVER"          // High self-efficacy, data-driven, responds to progress
	PhenotypeRoutineBuilder   ColdStartPhenotype = "ROUTINE_BUILDER"   // Moderate self-efficacy, prefers structure and predictability
	PhenotypeKnowledgeSeeker  ColdStartPhenotype = "KNOWLEDGE_SEEKER"  // High education, responds to explanations and data
	PhenotypeSupportDependent ColdStartPhenotype = "SUPPORT_DEPENDENT" // Low self-efficacy, needs external accountability
	PhenotypeRewardResponsive ColdStartPhenotype = "REWARD_RESPONSIVE" // Responds to incentives, competition, milestones
)

// AllColdStartPhenotypes returns the canonical list.
func AllColdStartPhenotypes() []ColdStartPhenotype {
	return []ColdStartPhenotype{
		PhenotypeAchiever, PhenotypeRoutineBuilder, PhenotypeKnowledgeSeeker,
		PhenotypeSupportDependent, PhenotypeRewardResponsive,
	}
}

// PhenotypePriorSet holds calibrated Alpha/Beta priors for all 12 techniques
// for a specific cold-start phenotype cluster.
type PhenotypePriorSet struct {
	Phenotype ColdStartPhenotype
	Priors    map[TechniqueID]TechniquePrior
}

// TechniquePrior holds a single technique's calibrated Alpha/Beta values.
type TechniquePrior struct {
	Alpha float64
	Beta  float64
}
