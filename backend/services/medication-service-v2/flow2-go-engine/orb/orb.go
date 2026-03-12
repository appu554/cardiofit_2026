package orb

// Re-export types from internal/orb package for external use
import (
	internal_orb "flow2-go-engine/internal/orb"
)

// Core types re-exported from internal package
type IntentManifest = internal_orb.IntentManifest
type MedicationRequest = internal_orb.MedicationRequest
type OrchestratorRuleBase = internal_orb.OrchestratorRuleBase
type ContextServiceRecipeBook = internal_orb.ContextServiceRecipeBook
type ContextRecipe = internal_orb.ContextRecipe
type EvaluationMetrics = internal_orb.EvaluationMetrics
type ORBRule = internal_orb.ORBRule

// Constructor functions re-exported
var NewOrchestratorRuleBase = internal_orb.NewOrchestratorRuleBase
var NewPhase1CompliantORB = internal_orb.NewPhase1CompliantORB

// Execution functions re-exported
var ExecuteLocal = internal_orb.ExecuteLocal
var GetEvaluationMetrics = internal_orb.GetEvaluationMetrics
var GetAvailableRules = internal_orb.GetAvailableRules
var GetRuleByID = internal_orb.GetRuleByID