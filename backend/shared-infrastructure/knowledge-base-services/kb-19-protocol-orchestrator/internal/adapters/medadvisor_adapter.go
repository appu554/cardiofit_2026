// Package adapters provides adapter implementations for external service clients.
// This package bridges the gap between client types and transaction types.
package adapters

import (
	"context"

	"kb-19-protocol-orchestrator/internal/clients"
	"kb-19-protocol-orchestrator/internal/transaction"
)

// =============================================================================
// MEDICATION ADVISOR RISK PROVIDER ADAPTER
// Implements transaction.RiskProfileProvider interface
// V3 Architecture: Bridges clients.MedicationAdvisorClient to transaction package
// =============================================================================

// MedAdvisorRiskProvider wraps MedicationAdvisorClient to implement
// the transaction.RiskProfileProvider interface for V3 workflow integration.
type MedAdvisorRiskProvider struct {
	client *clients.MedicationAdvisorClient
}

// NewMedAdvisorRiskProvider creates an adapter that implements RiskProfileProvider.
func NewMedAdvisorRiskProvider(client *clients.MedicationAdvisorClient) *MedAdvisorRiskProvider {
	return &MedAdvisorRiskProvider{client: client}
}

// GetRiskProfile implements transaction.RiskProfileProvider by calling the underlying client
// and converting the response to the expected transaction.RiskProfile type.
func (a *MedAdvisorRiskProvider) GetRiskProfile(ctx context.Context, req *transaction.RiskProfileRequest) (*transaction.RiskProfile, error) {
	// Convert conditions from transaction to client type
	clientConditions := make([]clients.ConditionRefInput, 0, len(req.PatientData.Conditions))
	for _, c := range req.PatientData.Conditions {
		clientConditions = append(clientConditions, clients.ConditionRefInput{
			ICD10Code:  c.ICD10Code,
			SNOMEDCode: c.SNOMEDCode,
			Display:    c.Display,
		})
	}

	// Convert allergies from transaction to client type
	clientAllergies := make([]clients.AllergyRefInput, 0, len(req.PatientData.Allergies))
	for _, a := range req.PatientData.Allergies {
		clientAllergies = append(clientAllergies, clients.AllergyRefInput{
			AllergenCode: a.AllergenCode,
			AllergenType: a.AllergenType,
			Severity:     a.Severity,
		})
	}

	// Convert transaction.RiskProfileRequest to clients.RiskProfileRequest
	clientReq := &clients.RiskProfileRequest{
		PatientID:   req.PatientID,
		EncounterID: req.EncounterID,
		Medications: make([]clients.MedicationInput, 0, len(req.Medications)),
		PatientData: clients.PatientDataInput{
			Sex:            req.PatientData.Gender, // Map Gender -> Sex for Med-Advisor
			Age:            req.PatientData.Age,
			WeightKg:       req.PatientData.WeightKg,
			HeightCm:       req.PatientData.HeightCm,
			EGFR:           req.PatientData.EGFR,
			ChildPughScore: req.PatientData.ChildPughScore,
			IsPregnant:     req.PatientData.IsPregnant,
			IsLactating:    req.PatientData.IsLactating,
			Conditions:     clientConditions,
			Allergies:      clientAllergies,
		},
		LabValues: make([]clients.LabValueInput, 0, len(req.LabValues)),
	}

	// Convert medications
	for _, med := range req.Medications {
		clientReq.Medications = append(clientReq.Medications, clients.MedicationInput{
			RxNormCode:      med.RxNormCode,
			DrugName:        med.DrugName,
			DoseValue:       med.DoseValue,
			DoseUnit:        med.DoseUnit,
			IsProposed:      med.IsProposed,
			RequiresDoseAdj: med.RequiresDoseAdj,
		})
	}

	// Convert lab values
	for _, lab := range req.LabValues {
		clientReq.LabValues = append(clientReq.LabValues, clients.LabValueInput{
			LOINCCode:  lab.LOINCCode,
			Value:      lab.Value,
			Unit:       lab.Unit,
			IsCritical: lab.IsCritical,
		})
	}

	// Call the underlying client
	resp, err := a.client.GetRiskProfile(ctx, clientReq)
	if err != nil {
		return nil, err
	}

	// Convert clients.RiskProfileResponse to transaction.RiskProfile
	result := &transaction.RiskProfile{
		RequestID:           resp.RequestID,
		PatientID:           resp.PatientID,
		EncounterID:         resp.EncounterID,
		CalculatedAt:        resp.CalculatedAt,
		MedicationRisks:     make([]transaction.MedicationRisk, 0, len(resp.MedicationRisks)),
		DDIRisks:            make([]transaction.DDIRisk, 0, len(resp.DDIRisks)),
		LabRisks:            make([]transaction.LabRisk, 0, len(resp.LabRisks)),
		AllergyRisks:        make([]transaction.AllergyRisk, 0, len(resp.AllergyRisks)),
		DoseRecommendations: make([]transaction.DoseRecommendation, 0, len(resp.DoseRecommendations)),
		KBSourcesUsed:       resp.KBSourcesUsed,
		ProcessingMs:        resp.ProcessingMs,
	}

	// Convert medication risks
	for _, mr := range resp.MedicationRisks {
		riskFactors := make([]transaction.RiskFactor, 0, len(mr.RiskFactors))
		for _, rf := range mr.RiskFactors {
			riskFactors = append(riskFactors, transaction.RiskFactor{
				Type:        rf.Type,
				Severity:    rf.Severity,
				Description: rf.Description,
				KBSource:    rf.KBSource,
				RuleID:      rf.RuleID,
			})
		}
		result.MedicationRisks = append(result.MedicationRisks, transaction.MedicationRisk{
			RxNormCode:      mr.RxNormCode,
			DrugName:        mr.DrugName,
			OverallRisk:     mr.OverallRisk,
			RiskCategory:    mr.RiskCategory,
			RiskFactors:     riskFactors,
			IsHighAlert:     mr.IsHighAlert,
			HasBlackBoxWarn: mr.HasBlackBoxWarn,
		})
	}

	// Convert DDI risks
	for _, ddi := range resp.DDIRisks {
		result.DDIRisks = append(result.DDIRisks, transaction.DDIRisk{
			Drug1Code:          ddi.Drug1Code,
			Drug1Name:          ddi.Drug1Name,
			Drug2Code:          ddi.Drug2Code,
			Drug2Name:          ddi.Drug2Name,
			Severity:           ddi.Severity,
			InteractionType:    ddi.InteractionType,
			Mechanism:          ddi.Mechanism,
			ClinicalEffect:     ddi.ClinicalEffect,
			ManagementStrategy: ddi.ManagementStrategy,
			EvidenceLevel:      ddi.EvidenceLevel,
			KBSource:           ddi.KBSource,
			RuleID:             ddi.RuleID,
		})
	}

	// Convert Lab risks
	for _, lab := range resp.LabRisks {
		result.LabRisks = append(result.LabRisks, transaction.LabRisk{
			RxNormCode:     lab.RxNormCode,
			DrugName:       lab.DrugName,
			LOINCCode:      lab.LOINCCode,
			LabName:        lab.LabName,
			CurrentValue:   lab.CurrentValue,
			ThresholdValue: lab.ThresholdValue,
			ThresholdOp:    lab.ThresholdOp,
			Severity:       lab.Severity,
			ClinicalRisk:   lab.ClinicalRisk,
			Recommendation: lab.Recommendation,
			KBSource:       lab.KBSource,
			RuleID:         lab.RuleID,
		})
	}

	// Convert Allergy risks
	for _, allergy := range resp.AllergyRisks {
		result.AllergyRisks = append(result.AllergyRisks, transaction.AllergyRisk{
			RxNormCode:      allergy.RxNormCode,
			DrugName:        allergy.DrugName,
			AllergenCode:    allergy.AllergenCode,
			AllergenName:    allergy.AllergenName,
			IsCrossReactive: allergy.IsCrossReactive,
			Severity:        allergy.Severity,
			ReactionType:    allergy.ReactionType,
			KBSource:        allergy.KBSource,
			RuleID:          allergy.RuleID,
		})
	}

	// Convert Dose recommendations
	for _, dose := range resp.DoseRecommendations {
		result.DoseRecommendations = append(result.DoseRecommendations, transaction.DoseRecommendation{
			RxNormCode:      dose.RxNormCode,
			DrugName:        dose.DrugName,
			OriginalDose:    dose.OriginalDose,
			AdjustedDose:    dose.AdjustedDose,
			DoseUnit:        dose.DoseUnit,
			AdjustmentType:  dose.AdjustmentType,
			AdjustmentRatio: dose.AdjustmentRatio,
			Reason:          dose.Reason,
			KBSource:        dose.KBSource,
			RuleID:          dose.RuleID,
		})
	}

	return result, nil
}
