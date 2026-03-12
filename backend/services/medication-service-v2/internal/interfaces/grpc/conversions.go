package grpc

import (
	"time"
	"errors"

	"medication-service-v2/internal/domain/entities"
	pb "medication-service-v2/proto/medication/v1"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/structpb"
)

// Conversion utilities for protobuf <-> domain entities

// UUID conversion utilities
func parseUUID(s string) uuid.UUID {
	if s == "" {
		return uuid.Nil
	}
	id, _ := uuid.Parse(s)
	return id
}

func parseUUIDOptional(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	if id, err := uuid.Parse(s); err == nil {
		return &id
	}
	return nil
}

func uuidToString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

// Timestamp conversion utilities
func timestampNow() *timestamppb.Timestamp {
	return timestamppb.Now()
}

func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func timeFromTimestamp(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// Error checking utilities
func isNotFoundError(err error) bool {
	// Implementation depends on your error types
	// This is a placeholder
	return errors.Is(err, entities.ErrNotFound)
}

// MedicationProposal conversions

func convertToProposalPB(proposal *entities.MedicationProposal) (*pb.MedicationProposal, error) {
	if proposal == nil {
		return nil, nil
	}

	// Convert clinical context
	pbClinicalContext, err := convertClinicalContextToPB(proposal.ClinicalContext)
	if err != nil {
		return nil, err
	}

	// Convert medication details
	pbMedicationDetails, err := convertMedicationDetailsToPB(proposal.MedicationDetails)
	if err != nil {
		return nil, err
	}

	// Convert dosage recommendations
	pbDosageRecs := make([]*pb.DosageRecommendation, len(proposal.DosageRecommendations))
	for i, rec := range proposal.DosageRecommendations {
		pbRec, err := convertToDosageRecommendationPB(&rec)
		if err != nil {
			return nil, err
		}
		pbDosageRecs[i] = pbRec
	}

	// Convert safety constraints
	pbSafetyConstraints := make([]*pb.SafetyConstraint, len(proposal.SafetyConstraints))
	for i, constraint := range proposal.SafetyConstraints {
		pbConstraint, err := convertToSafetyConstraintPB(&constraint)
		if err != nil {
			return nil, err
		}
		pbSafetyConstraints[i] = pbConstraint
	}

	return &pb.MedicationProposal{
		Id:                      uuidToString(proposal.ID),
		PatientId:              uuidToString(proposal.PatientID),
		ProtocolId:             proposal.ProtocolID,
		Indication:             proposal.Indication,
		Status:                 convertProposalStatusToPB(proposal.Status),
		ClinicalContext:        pbClinicalContext,
		MedicationDetails:      pbMedicationDetails,
		DosageRecommendations:  pbDosageRecs,
		SafetyConstraints:      pbSafetyConstraints,
		SnapshotId:             uuidToString(proposal.SnapshotID),
		CreatedAt:              timestampFromTime(proposal.CreatedAt),
		UpdatedAt:              timestampFromTime(proposal.UpdatedAt),
		CreatedBy:              proposal.CreatedBy,
		ValidatedBy:            stringPtrToString(proposal.ValidatedBy),
		ValidationTimestamp:    timestampFromTimePtr(proposal.ValidationTimestamp),
	}, nil
}

func convertFromProposalPB(pbProposal *pb.MedicationProposal) (*entities.MedicationProposal, error) {
	if pbProposal == nil {
		return nil, nil
	}

	// Convert clinical context
	clinicalContext, err := convertPBClinicalContext(pbProposal.ClinicalContext)
	if err != nil {
		return nil, err
	}

	// Convert medication details
	medicationDetails, err := convertPBMedicationDetails(pbProposal.MedicationDetails)
	if err != nil {
		return nil, err
	}

	// Convert dosage recommendations
	dosageRecs := make([]entities.DosageRecommendation, len(pbProposal.DosageRecommendations))
	for i, pbRec := range pbProposal.DosageRecommendations {
		rec, err := convertFromDosageRecommendationPB(pbRec)
		if err != nil {
			return nil, err
		}
		dosageRecs[i] = *rec
	}

	// Convert safety constraints
	safetyConstraints := make([]entities.SafetyConstraint, len(pbProposal.SafetyConstraints))
	for i, pbConstraint := range pbProposal.SafetyConstraints {
		constraint, err := convertFromSafetyConstraintPB(pbConstraint)
		if err != nil {
			return nil, err
		}
		safetyConstraints[i] = *constraint
	}

	return &entities.MedicationProposal{
		ID:                    parseUUID(pbProposal.Id),
		PatientID:             parseUUID(pbProposal.PatientId),
		ProtocolID:            pbProposal.ProtocolId,
		Indication:            pbProposal.Indication,
		Status:                convertProposalStatusFromPB(pbProposal.Status),
		ClinicalContext:       clinicalContext,
		MedicationDetails:     medicationDetails,
		DosageRecommendations: dosageRecs,
		SafetyConstraints:     safetyConstraints,
		SnapshotID:            parseUUID(pbProposal.SnapshotId),
		CreatedAt:             timeFromTimestamp(pbProposal.CreatedAt),
		UpdatedAt:             timeFromTimestamp(pbProposal.UpdatedAt),
		CreatedBy:             pbProposal.CreatedBy,
		ValidatedBy:           stringToStringPtr(pbProposal.ValidatedBy),
		ValidationTimestamp:   timePtrFromTimestamp(pbProposal.ValidationTimestamp),
	}, nil
}

// ClinicalContext conversions

func convertClinicalContextToPB(ctx *entities.ClinicalContext) (*pb.ClinicalContext, error) {
	if ctx == nil {
		return nil, nil
	}

	// Convert current medications
	pbCurrentMeds := make([]*pb.CurrentMedication, len(ctx.Medications))
	for i, med := range ctx.Medications {
		pbCurrentMeds[i] = &pb.CurrentMedication{
			MedicationName: med.MedicationName,
			DoseMg:         med.DoseMg,
			Frequency:      med.Frequency,
			StartDate:      timestampFromTime(med.StartDate),
			Route:          med.Route,
		}
	}

	// Convert lab values
	pbLabValues := make(map[string]*pb.LabValue)
	for key, lab := range ctx.LabValues {
		pbLabValues[key] = &pb.LabValue{
			Value:          lab.Value,
			Unit:           lab.Unit,
			Timestamp:      timestampFromTime(lab.Timestamp),
			ReferenceRange: lab.Reference,
		}
	}

	return &pb.ClinicalContext{
		PatientId:          uuidToString(ctx.PatientID),
		WeightKg:           float64PtrToFloat64(ctx.WeightKg),
		HeightCm:           float64PtrToFloat64(ctx.HeightCm),
		AgeYears:           int32(ctx.AgeYears),
		Gender:             ctx.Gender,
		BsaM2:              float64PtrToFloat64(ctx.BSAm2),
		CreatinineMgDl:     float64PtrToFloat64(ctx.CreatinineMgdL),
		Egfr:               float64PtrToFloat64(ctx.eGFR),
		Allergies:          ctx.Allergies,
		Conditions:         ctx.Conditions,
		CurrentMedications: pbCurrentMeds,
		LabValues:          pbLabValues,
	}, nil
}

func convertPBClinicalContext(pbCtx *pb.ClinicalContext) (*entities.ClinicalContext, error) {
	if pbCtx == nil {
		return nil, nil
	}

	// Convert current medications
	currentMeds := make([]entities.CurrentMedication, len(pbCtx.CurrentMedications))
	for i, pbMed := range pbCtx.CurrentMedications {
		currentMeds[i] = entities.CurrentMedication{
			MedicationName: pbMed.MedicationName,
			DoseMg:         pbMed.DoseMg,
			Frequency:      pbMed.Frequency,
			StartDate:      timeFromTimestamp(pbMed.StartDate),
			Route:          pbMed.Route,
		}
	}

	// Convert lab values
	labValues := make(map[string]entities.LabValue)
	for key, pbLab := range pbCtx.LabValues {
		labValues[key] = entities.LabValue{
			Value:     pbLab.Value,
			Unit:      pbLab.Unit,
			Timestamp: timeFromTimestamp(pbLab.Timestamp),
			Reference: pbLab.ReferenceRange,
		}
	}

	return &entities.ClinicalContext{
		PatientID:      parseUUID(pbCtx.PatientId),
		WeightKg:       float64ToFloat64Ptr(pbCtx.WeightKg),
		HeightCm:       float64ToFloat64Ptr(pbCtx.HeightCm),
		AgeYears:       int(pbCtx.AgeYears),
		Gender:         pbCtx.Gender,
		BSAm2:          float64ToFloat64Ptr(pbCtx.BsaM2),
		CreatinineMgdL: float64ToFloat64Ptr(pbCtx.CreatinineMgDl),
		eGFR:           float64ToFloat64Ptr(pbCtx.Egfr),
		Allergies:      pbCtx.Allergies,
		Conditions:     pbCtx.Conditions,
		Medications:    currentMeds,
		LabValues:      labValues,
	}, nil
}

// MedicationDetails conversions

func convertMedicationDetailsToPB(details *entities.MedicationDetails) (*pb.MedicationDetails, error) {
	if details == nil {
		return nil, nil
	}

	// Convert drug interactions
	pbInteractions := make([]*pb.DrugInteraction, len(details.Interactions))
	for i, interaction := range details.Interactions {
		pbInteractions[i] = &pb.DrugInteraction{
			InteractingDrug: interaction.InteractingDrug,
			Severity:        string(interaction.Severity),
			Description:     interaction.Description,
			Management:      interaction.Management,
		}
	}

	// Convert formulation types
	pbFormulationTypes := make([]*pb.FormulationType, len(details.FormulationTypes))
	for i, formType := range details.FormulationTypes {
		pbFormulationTypes[i] = &pb.FormulationType{
			Form:         formType.Form,
			Strengths:    formType.Strengths,
			Route:        formType.Route,
			Availability: formType.Availability,
		}
	}

	// Convert pharmacology profile
	var pbPharmProfile *pb.PharmacologyProfile
	if details.PharmacologyProfile != nil {
		profile := details.PharmacologyProfile
		pbPharmProfile = &pb.PharmacologyProfile{
			HalfLifeHours:     float64PtrToFloat64(profile.HalfLifeHours),
			OnsetMinutes:      int32PtrToInt32(profile.OnsetMinutes),
			PeakHours:         float64PtrToFloat64(profile.PeakHours),
			DurationHours:     float64PtrToFloat64(profile.DurationHours),
			Bioavailability:   float64PtrToFloat64(profile.Bioavailability),
			ProteinBinding:    float64PtrToFloat64(profile.ProteinBinding),
			Metabolism:        profile.Metabolism,
			Excretion:         profile.Excretion,
			RenalAdjustment:   profile.RenalAdjustment,
			HepaticAdjustment: profile.HepaticAdjustment,
		}
	}

	return &pb.MedicationDetails{
		DrugName:            details.DrugName,
		GenericName:         details.GenericName,
		BrandName:           details.BrandName,
		DrugClass:           details.DrugClass,
		Mechanism:           details.Mechanism,
		Indication:          details.Indication,
		Contraindications:   details.Contraindications,
		Interactions:        pbInteractions,
		FormulationTypes:    pbFormulationTypes,
		TherapeuticClass:    details.TherapeuticClass,
		PharmacologyProfile: pbPharmProfile,
	}, nil
}

func convertPBMedicationDetails(pbDetails *pb.MedicationDetails) (*entities.MedicationDetails, error) {
	if pbDetails == nil {
		return nil, nil
	}

	// Convert drug interactions
	interactions := make([]entities.DrugInteraction, len(pbDetails.Interactions))
	for i, pbInteraction := range pbDetails.Interactions {
		interactions[i] = entities.DrugInteraction{
			InteractingDrug: pbInteraction.InteractingDrug,
			Severity:        entities.InteractionSeverity(pbInteraction.Severity),
			Description:     pbInteraction.Description,
			Management:      pbInteraction.Management,
		}
	}

	// Convert formulation types
	formulationTypes := make([]entities.FormulationType, len(pbDetails.FormulationTypes))
	for i, pbFormType := range pbDetails.FormulationTypes {
		formulationTypes[i] = entities.FormulationType{
			Form:         pbFormType.Form,
			Strengths:    pbFormType.Strengths,
			Route:        pbFormType.Route,
			Availability: pbFormType.Availability,
		}
	}

	// Convert pharmacology profile
	var pharmProfile *entities.PharmacologyProfile
	if pbDetails.PharmacologyProfile != nil {
		pbProfile := pbDetails.PharmacologyProfile
		pharmProfile = &entities.PharmacologyProfile{
			HalfLifeHours:     float64ToFloat64Ptr(pbProfile.HalfLifeHours),
			OnsetMinutes:      int32ToIntPtr(pbProfile.OnsetMinutes),
			PeakHours:         float64ToFloat64Ptr(pbProfile.PeakHours),
			DurationHours:     float64ToFloat64Ptr(pbProfile.DurationHours),
			Bioavailability:   float64ToFloat64Ptr(pbProfile.Bioavailability),
			ProteinBinding:    float64ToFloat64Ptr(pbProfile.ProteinBinding),
			Metabolism:        pbProfile.Metabolism,
			Excretion:         pbProfile.Excretion,
			RenalAdjustment:   pbProfile.RenalAdjustment,
			HepaticAdjustment: pbProfile.HepaticAdjustment,
		}
	}

	return &entities.MedicationDetails{
		DrugName:            pbDetails.DrugName,
		GenericName:         pbDetails.GenericName,
		BrandName:           pbDetails.BrandName,
		DrugClass:           pbDetails.DrugClass,
		Mechanism:           pbDetails.Mechanism,
		Indication:          pbDetails.Indication,
		Contraindications:   pbDetails.Contraindications,
		Interactions:        interactions,
		FormulationTypes:    formulationTypes,
		TherapeuticClass:    pbDetails.TherapeuticClass,
		PharmacologyProfile: pharmProfile,
	}, nil
}

// DosageRecommendation conversions

func convertToDosageRecommendationPB(rec *entities.DosageRecommendation) (*pb.DosageRecommendation, error) {
	if rec == nil {
		return nil, nil
	}

	// Convert monitoring requirements
	pbMonitoring := make([]*pb.MonitoringRequirement, len(rec.MonitoringRequired))
	for i, monitor := range rec.MonitoringRequired {
		pbMonitoring[i] = &pb.MonitoringRequirement{
			Parameter:      monitor.Parameter,
			Frequency:      string(monitor.Frequency),
			TargetRange:    stringPtrToString(monitor.TargetRange),
			AlertThreshold: float64PtrToFloat64(monitor.AlertThreshold),
			Notes:          monitor.Notes,
		}
	}

	return &pb.DosageRecommendation{
		Id:                 uuidToString(rec.ID),
		RecommendationType: string(rec.RecommendationType),
		DoseMg:             rec.DoseMg,
		FrequencyPerDay:    int32(rec.FrequencyPerDay),
		Route:              rec.Route,
		DurationDays:       int32PtrToInt32(rec.DurationDays),
		MaxDoseMg:          float64PtrToFloat64(rec.MaxDoseMg),
		MinDoseMg:          float64PtrToFloat64(rec.MinDoseMg),
		AdjustmentReason:   rec.AdjustmentReason,
		CalculationMethod:  string(rec.CalculationMethod),
		ConfidenceScore:    rec.ConfidenceScore,
		ClinicalNotes:      rec.ClinicalNotes,
		MonitoringRequired: pbMonitoring,
	}, nil
}

func convertFromDosageRecommendationPB(pbRec *pb.DosageRecommendation) (*entities.DosageRecommendation, error) {
	if pbRec == nil {
		return nil, nil
	}

	// Convert monitoring requirements
	monitoring := make([]entities.MonitoringRequirement, len(pbRec.MonitoringRequired))
	for i, pbMonitor := range pbRec.MonitoringRequired {
		monitoring[i] = entities.MonitoringRequirement{
			Parameter:      pbMonitor.Parameter,
			Frequency:      entities.MonitoringFrequency(pbMonitor.Frequency),
			TargetRange:    stringToStringPtr(pbMonitor.TargetRange),
			AlertThreshold: float64ToFloat64Ptr(pbMonitor.AlertThreshold),
			Notes:          pbMonitor.Notes,
		}
	}

	return &entities.DosageRecommendation{
		ID:                 parseUUID(pbRec.Id),
		RecommendationType: entities.RecommendationType(pbRec.RecommendationType),
		DoseMg:             pbRec.DoseMg,
		FrequencyPerDay:    int(pbRec.FrequencyPerDay),
		Route:              pbRec.Route,
		DurationDays:       int32ToIntPtr(pbRec.DurationDays),
		MaxDoseMg:          float64ToFloat64Ptr(pbRec.MaxDoseMg),
		MinDoseMg:          float64ToFloat64Ptr(pbRec.MinDoseMg),
		AdjustmentReason:   pbRec.AdjustmentReason,
		CalculationMethod:  entities.CalculationMethod(pbRec.CalculationMethod),
		ConfidenceScore:    pbRec.ConfidenceScore,
		ClinicalNotes:      pbRec.ClinicalNotes,
		MonitoringRequired: monitoring,
	}, nil
}

// SafetyConstraint conversions

func convertToSafetyConstraintPB(constraint *entities.SafetyConstraint) (*pb.SafetyConstraint, error) {
	if constraint == nil {
		return nil, nil
	}

	return &pb.SafetyConstraint{
		Id:             uuidToString(constraint.ID),
		ConstraintType: string(constraint.ConstraintType),
		Severity:       string(constraint.Severity),
		Parameter:      constraint.Parameter,
		Operator:       constraint.Operator,
		ThresholdValue: constraint.ThresholdValue,
		Unit:           constraint.Unit,
		Message:        constraint.Message,
		Action:         constraint.Action,
		Source:         constraint.Source,
	}, nil
}

func convertFromSafetyConstraintPB(pbConstraint *pb.SafetyConstraint) (*entities.SafetyConstraint, error) {
	if pbConstraint == nil {
		return nil, nil
	}

	return &entities.SafetyConstraint{
		ID:             parseUUID(pbConstraint.Id),
		ConstraintType: entities.ConstraintType(pbConstraint.ConstraintType),
		Severity:       entities.ConstraintSeverity(pbConstraint.Severity),
		Parameter:      pbConstraint.Parameter,
		Operator:       pbConstraint.Operator,
		ThresholdValue: pbConstraint.ThresholdValue,
		Unit:           pbConstraint.Unit,
		Message:        pbConstraint.Message,
		Action:         pbConstraint.Action,
		Source:         pbConstraint.Source,
	}, nil
}

// Enum conversions

func convertProposalStatusToPB(status entities.ProposalStatus) pb.ProposalStatus {
	switch status {
	case entities.ProposalStatusDraft:
		return pb.ProposalStatus_PROPOSAL_STATUS_DRAFT
	case entities.ProposalStatusProposed:
		return pb.ProposalStatus_PROPOSAL_STATUS_PROPOSED
	case entities.ProposalStatusValidated:
		return pb.ProposalStatus_PROPOSAL_STATUS_VALIDATED
	case entities.ProposalStatusRejected:
		return pb.ProposalStatus_PROPOSAL_STATUS_REJECTED
	case entities.ProposalStatusCommitted:
		return pb.ProposalStatus_PROPOSAL_STATUS_COMMITTED
	case entities.ProposalStatusExpired:
		return pb.ProposalStatus_PROPOSAL_STATUS_EXPIRED
	default:
		return pb.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED
	}
}

func convertProposalStatusFromPB(pbStatus pb.ProposalStatus) entities.ProposalStatus {
	switch pbStatus {
	case pb.ProposalStatus_PROPOSAL_STATUS_DRAFT:
		return entities.ProposalStatusDraft
	case pb.ProposalStatus_PROPOSAL_STATUS_PROPOSED:
		return entities.ProposalStatusProposed
	case pb.ProposalStatus_PROPOSAL_STATUS_VALIDATED:
		return entities.ProposalStatusValidated
	case pb.ProposalStatus_PROPOSAL_STATUS_REJECTED:
		return entities.ProposalStatusRejected
	case pb.ProposalStatus_PROPOSAL_STATUS_COMMITTED:
		return entities.ProposalStatusCommitted
	case pb.ProposalStatus_PROPOSAL_STATUS_EXPIRED:
		return entities.ProposalStatusExpired
	default:
		return ""
	}
}

// Helper conversion functions for common types

func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func stringToStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func float64PtrToFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func float64ToFloat64Ptr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

func int32PtrToInt32(i *int) int32 {
	if i == nil {
		return 0
	}
	return int32(*i)
}

func int32ToIntPtr(i int32) *int {
	if i == 0 {
		return nil
	}
	result := int(i)
	return &result
}

func timePtrFromTimestamp(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func timestampFromTimePtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// Recipe conversions (placeholder implementations)

func convertToRecipePB(recipe interface{}) (*pb.Recipe, error) {
	// Implementation would depend on the actual Recipe entity structure
	// This is a placeholder
	return &pb.Recipe{}, nil
}

// Additional conversion functions would be implemented here for completeness