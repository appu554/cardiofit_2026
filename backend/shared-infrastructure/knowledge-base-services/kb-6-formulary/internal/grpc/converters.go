package grpc

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"kb-formulary/internal/services"
	pb "kb-formulary/proto/kb6"
)

// convertPatientContext converts protobuf patient context to service struct
func convertPatientContext(patient *pb.PatientContext) *services.PatientContext {
	if patient == nil {
		return nil
	}
	
	return &services.PatientContext{
		Age:            int(patient.Age),
		Gender:         patient.Gender,
		DiagnosisCodes: patient.DiagnosisCodes,
		Allergies:      patient.Allergies,
	}
}

// convertCostDetails converts service cost details to protobuf
func convertCostDetails(cost *services.CostDetails) *pb.CostDetails {
	if cost == nil {
		return nil
	}
	
	return &pb.CostDetails{
		CopayAmount:          cost.CopayAmount,
		CoinsurancePercent:   int32(cost.CoinsurancePercent),
		DeductibleApplies:    cost.DeductibleApplies,
		EstimatedPatientCost: cost.EstimatedPatientCost,
		DrugCost:             cost.DrugCost,
	}
}

// convertQuantityLimits converts service quantity limits to protobuf
func convertQuantityLimits(limits *services.QuantityLimits) *pb.QuantityLimits {
	if limits == nil {
		return nil
	}
	
	return &pb.QuantityLimits{
		MaxQuantity:   int32(limits.MaxQuantity),
		PerDays:       int32(limits.PerDays),
		MaxFillsPerYear: int32(limits.MaxFillsPerYear),
		LimitType:     limits.LimitType,
	}
}

// convertAgeRestrictions converts service age restrictions to protobuf
func convertAgeRestrictions(restrictions *services.AgeRestrictions) *pb.AgeRestrictions {
	if restrictions == nil {
		return nil
	}
	
	return &pb.AgeRestrictions{
		MinAge: int32(restrictions.MinAge),
		MaxAge: int32(restrictions.MaxAge),
	}
}

// convertAlternatives converts service alternatives to protobuf
func convertAlternatives(alternatives []services.Alternative) []*pb.Alternative {
	if alternatives == nil {
		return nil
	}
	
	result := make([]*pb.Alternative, len(alternatives))
	for i, alt := range alternatives {
		result[i] = &pb.Alternative{
			DrugRxnorm:            alt.DrugRxNorm,
			DrugName:              alt.DrugName,
			AlternativeType:       alt.AlternativeType,
			Tier:                  alt.Tier,
			EstimatedCost:         alt.EstimatedCost,
			CostSavings:           alt.CostSavings,
			CostSavingsPercent:    alt.CostSavingsPercent,
			SwitchComplexity:      alt.SwitchComplexity,
			EfficacyRating:        alt.EfficacyRating,
			SafetyProfile:         alt.SafetyProfile,
		}
	}
	return result
}

// convertEvidenceEnvelope converts service evidence envelope to protobuf
func convertEvidenceEnvelope(evidence *services.EvidenceEnvelope) *pb.EvidenceEnvelope {
	if evidence == nil {
		return nil
	}
	
	envelope := &pb.EvidenceEnvelope{
		DatasetVersion:    evidence.DatasetVersion,
		SourceSystem:      evidence.SourceSystem,
		DecisionHash:      evidence.DecisionHash,
		DataSources:       evidence.DataSources,
		Kb7Version:        evidence.KB7Version,
	}
	
	if !evidence.DatasetTimestamp.IsZero() {
		envelope.DatasetTimestamp = timestamppb.New(evidence.DatasetTimestamp)
	}
	
	if evidence.Provenance != nil {
		envelope.Provenance = evidence.Provenance
	}
	
	return envelope
}

// convertLotDetails converts service lot details to protobuf
func convertLotDetails(lots []services.LotDetail) []*pb.LotDetail {
	if lots == nil {
		return nil
	}
	
	result := make([]*pb.LotDetail, len(lots))
	for i, lot := range lots {
		detail := &pb.LotDetail{
			LotNumber:    lot.LotNumber,
			Quantity:     int32(lot.Quantity),
			Manufacturer: lot.Manufacturer,
			UnitCost:     lot.UnitCost,
		}
		
		if !lot.ExpirationDate.IsZero() {
			detail.ExpirationDate = timestamppb.New(lot.ExpirationDate)
		}
		
		result[i] = detail
	}
	return result
}

// convertReorderInfo converts service reorder info to protobuf
func convertReorderInfo(info *services.ReorderInfo) *pb.ReorderInfo {
	if info == nil {
		return nil
	}
	
	return &pb.ReorderInfo{
		ReorderPoint:       int32(info.ReorderPoint),
		ReorderQuantity:    int32(info.ReorderQuantity),
		MaxStockLevel:      int32(info.MaxStockLevel),
		ReorderRecommended: info.ReorderRecommended,
		DaysUntilStockout:  int32(info.DaysUntilStockout),
	}
}

// convertAlternativeStock converts service alternative stock to protobuf
func convertAlternativeStock(stock []services.AlternativeStock) []*pb.AlternativeStock {
	if stock == nil {
		return nil
	}
	
	result := make([]*pb.AlternativeStock, len(stock))
	for i, alt := range stock {
		result[i] = &pb.AlternativeStock{
			DrugRxnorm:        alt.DrugRxNorm,
			DrugName:          alt.DrugName,
			AlternativeType:   alt.AlternativeType,
			QuantityAvailable: int32(alt.QuantityAvailable),
			LocationId:        alt.LocationID,
			DistanceKm:        alt.DistanceKm,
		}
	}
	return result
}

// convertStockAlerts converts service stock alerts to protobuf
func convertStockAlerts(alerts []services.StockAlert) []*pb.StockAlert {
	if alerts == nil {
		return nil
	}
	
	result := make([]*pb.StockAlert, len(alerts))
	for i, alert := range alerts {
		pbAlert := &pb.StockAlert{
			AlertType:         alert.AlertType,
			Severity:          alert.Severity,
			Message:           alert.Message,
			RecommendedAction: alert.RecommendedAction,
		}
		
		if !alert.TriggeredAt.IsZero() {
			pbAlert.TriggeredAt = timestamppb.New(alert.TriggeredAt)
		}
		
		result[i] = pbAlert
	}
	return result
}

// convertDemandPrediction converts service demand prediction to protobuf
func convertDemandPrediction(prediction *services.DemandPrediction) *pb.DemandPrediction {
	if prediction == nil {
		return nil
	}
	
	return &pb.DemandPrediction{
		PredictedDemand_7D:  int32(prediction.PredictedDemand7d),
		PredictedDemand_30D: int32(prediction.PredictedDemand30d),
		ConfidenceScore:     prediction.ConfidenceScore,
		StockoutRisk:        prediction.StockoutRisk,
	}
}

// convertDrugCostAnalysis converts service drug cost analysis to protobuf
func convertDrugCostAnalysis(analysis []services.DrugCostAnalysis) []*pb.DrugCostAnalysis {
	if analysis == nil {
		return nil
	}
	
	result := make([]*pb.DrugCostAnalysis, len(analysis))
	for i, drug := range analysis {
		pbAnalysis := &pb.DrugCostAnalysis{
			DrugRxnorm:       drug.DrugRxNorm,
			DrugName:         drug.DrugName,
			PrimaryCost:      drug.PrimaryCost,
			PotentialSavings: drug.PotentialSavings,
		}
		
		if drug.BestAlternative != nil {
			pbAnalysis.BestAlternative = &pb.Alternative{
				DrugRxnorm:            drug.BestAlternative.DrugRxNorm,
				DrugName:              drug.BestAlternative.DrugName,
				AlternativeType:       drug.BestAlternative.AlternativeType,
				Tier:                  drug.BestAlternative.Tier,
				EstimatedCost:         drug.BestAlternative.EstimatedCost,
				CostSavings:           drug.BestAlternative.CostSavings,
				CostSavingsPercent:    drug.BestAlternative.CostSavingsPercent,
				SwitchComplexity:      drug.BestAlternative.SwitchComplexity,
				EfficacyRating:        drug.BestAlternative.EfficacyRating,
				SafetyProfile:         drug.BestAlternative.SafetyProfile,
			}
		}
		
		pbAnalysis.AllAlternatives = convertAlternatives(drug.AllAlternatives)
		result[i] = pbAnalysis
	}
	return result
}

// convertCostOptimizations converts service cost optimizations to protobuf
func convertCostOptimizations(optimizations []services.CostOptimization) []*pb.CostOptimization {
	if optimizations == nil {
		return nil
	}
	
	result := make([]*pb.CostOptimization, len(optimizations))
	for i, opt := range optimizations {
		result[i] = &pb.CostOptimization{
			RecommendationType:       opt.RecommendationType,
			Description:              opt.Description,
			EstimatedSavings:         opt.EstimatedSavings,
			ImplementationComplexity: opt.ImplementationComplexity,
			RequiredActions:          opt.RequiredActions,
			ClinicalImpactScore:      opt.ClinicalImpactScore,
		}
	}
	return result
}

// convertFormularyEntries converts service formulary entries to protobuf
func convertFormularyEntries(entries []services.FormularyEntry) []*pb.FormularyEntry {
	if entries == nil {
		return nil
	}
	
	result := make([]*pb.FormularyEntry, len(entries))
	for i, entry := range entries {
		result[i] = &pb.FormularyEntry{
			DrugRxnorm:                entry.DrugRxNorm,
			DrugName:                  entry.DrugName,
			DrugType:                  entry.DrugType,
			Tier:                      entry.Tier,
			CoverageStatus:            entry.CoverageStatus,
			Cost:                      convertCostDetails(entry.Cost),
			PriorAuthorizationRequired: entry.PriorAuthorizationRequired,
			StepTherapyRequired:        entry.StepTherapyRequired,
			RelevanceScore:             entry.RelevanceScore,
		}
	}
	return result
}

// convertSearchMetadata converts service search metadata to protobuf
func convertSearchMetadata(metadata *services.SearchMetadata) *pb.SearchMetadata {
	if metadata == nil {
		return nil
	}
	
	pbMetadata := &pb.SearchMetadata{
		AvgCost:          metadata.AvgCost,
		CoveredCount:     int32(metadata.CoveredCount),
		NotCoveredCount:  int32(metadata.NotCoveredCount),
	}
	
	// Convert tier counts
	if metadata.TierCounts != nil {
		pbMetadata.TierCounts = make(map[string]int32)
		for tier, count := range metadata.TierCounts {
			pbMetadata.TierCounts[tier] = int32(count)
		}
	}
	
	// Convert drug type counts
	if metadata.DrugTypeCounts != nil {
		pbMetadata.DrugTypeCounts = make(map[string]int32)
		for drugType, count := range metadata.DrugTypeCounts {
			pbMetadata.DrugTypeCounts[drugType] = int32(count)
		}
	}
	
	return pbMetadata
}

// Helper function to convert time.Time to timestamppb.Timestamp safely
func convertTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}