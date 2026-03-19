package services

import "kb-21-behavioral-intelligence/internal/models"

// TechniqueInfo describes a technique's metadata and default priors.
type TechniqueInfo struct {
	ID          models.TechniqueID
	Name        string
	Description string
	// DefaultAlpha/Beta: population priors from published literature.
	// BCE v2.0 E3 (Population Learning) will replace these with data-driven priors.
	DefaultAlpha float64
	DefaultBeta  float64
}

// TechniqueLibrary provides the canonical catalog of 12 coaching techniques.
var TechniqueLibrary = map[models.TechniqueID]TechniqueInfo{
	models.TechMicroCommitment: {
		ID: models.TechMicroCommitment, Name: "Micro-Commitment",
		Description:  "Small, achievable daily goals that build self-efficacy through tiny wins",
		DefaultAlpha: 2.0, DefaultBeta: 2.0, // moderate prior — works for most patients
	},
	models.TechHabitStacking: {
		ID: models.TechHabitStacking, Name: "Habit Stacking",
		Description:  "Attach new behavior to an existing daily habit (e.g., walk after lunch)",
		DefaultAlpha: 2.0, DefaultBeta: 2.0,
	},
	models.TechLossAversion: {
		ID: models.TechLossAversion, Name: "Loss Aversion",
		Description:  "Frame adherence as avoiding loss of progress rather than gaining new benefit",
		DefaultAlpha: 1.5, DefaultBeta: 2.5, // weaker prior — not effective early in cycle
	},
	models.TechSocialNorms: {
		ID: models.TechSocialNorms, Name: "Social Norms",
		Description:  "Anonymous district-level peer comparison to leverage community identity",
		DefaultAlpha: 1.5, DefaultBeta: 2.5, // needs baseline data to be meaningful
	},
	models.TechMicroEducation: {
		ID: models.TechMicroEducation, Name: "Micro-Education",
		Description:  "Brief educational content explaining why a behavior matters clinically",
		DefaultAlpha: 2.0, DefaultBeta: 2.0,
	},
	models.TechProgressVisualization: {
		ID: models.TechProgressVisualization, Name: "Progress Visualization",
		Description:  "Show data-driven progress (FBG trend, adherence streak, waist measurement)",
		DefaultAlpha: 2.0, DefaultBeta: 1.5, // generally effective
	},
	models.TechEnvironmentRestructure: {
		ID: models.TechEnvironmentRestructure, Name: "Environment Restructuring",
		Description:  "Modify physical environment cues (pill placement, walking shoes by door)",
		DefaultAlpha: 1.5, DefaultBeta: 2.0,
	},
	models.TechImplementIntention: {
		ID: models.TechImplementIntention, Name: "Implementation Intention",
		Description:  "If-then planning for anticipated disruptions (travel, festivals, social events)",
		DefaultAlpha: 1.5, DefaultBeta: 2.0,
	},
	models.TechCostAwareSubstitution: {
		ID: models.TechCostAwareSubstitution, Name: "Cost-Aware Substitution",
		Description:  "Suggest affordable alternatives when cost is an adherence barrier",
		DefaultAlpha: 1.5, DefaultBeta: 2.0,
	},
	models.TechFamilyInclusion: {
		ID: models.TechFamilyInclusion, Name: "Family Inclusion",
		Description:  "Involve consented family member in coaching and celebration",
		DefaultAlpha: 2.0, DefaultBeta: 2.0,
	},
	models.TechRecoveryProtocol: {
		ID: models.TechRecoveryProtocol, Name: "Recovery Protocol",
		Description:  "Post-disruption re-engagement without guilt, lowered targets for 3 days",
		DefaultAlpha: 2.5, DefaultBeta: 1.5, // strong prior — recovery is almost always the right choice after disruption
	},
	models.TechKinshipTone: {
		ID: models.TechKinshipTone, Name: "Kinship Tone",
		Description:  "Culturally warm, elder-respectful communication style (Hindi/regional)",
		DefaultAlpha: 2.0, DefaultBeta: 1.5, // strong for Indian demographic
	},
}

// GetDefaultPriors returns Alpha, Beta priors for a technique.
func GetDefaultPriors(tech models.TechniqueID) (float64, float64) {
	if info, ok := TechniqueLibrary[tech]; ok {
		return info.DefaultAlpha, info.DefaultBeta
	}
	return 1.0, 1.0 // uniform prior fallback
}
