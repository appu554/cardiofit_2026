// Package caregaps provides the core care gap detection and quality measure evaluation logic.
package caregaps

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-9-care-gaps/internal/config"
	"kb-9-care-gaps/internal/cql"
	"kb-9-care-gaps/internal/fhir"
	"kb-9-care-gaps/internal/kb3"
	"kb-9-care-gaps/internal/models"

	// Import vaidshala contracts for CQL integration
	"vaidshala/clinical-runtime-platform/contracts"
)

// Service provides care gap detection and quality measure evaluation.
type Service struct {
	config         *config.Config
	logger         *zap.Logger
	measures       map[models.MeasureType]models.MeasureInfo
	fhirClient     *fhir.Client
	queryBuilder   *fhir.QueryBuilder
	cqlExecutor    *cql.Executor       // vaidshala CQL/Measure engine executor
	contextBuilder *cql.ContextBuilder // Builds ClinicalExecutionContext from FHIR
	useCQLEngine   bool                // Whether to use vaidshala CQL engine

	// KB-3 Temporal Integration (Tier 7 sibling)
	kb3Client      *kb3.Client      // HTTP client for KB-3 Temporal/Guidelines
	kb3Integration *kb3.Integration // Gap ↔ Schedule conversion logic
}

// NewService creates a new care gaps service.
func NewService(cfg *config.Config, logger *zap.Logger) *Service {
	s := &Service{
		config:       cfg,
		logger:       logger,
		measures:     make(map[models.MeasureType]models.MeasureInfo),
		useCQLEngine: cfg.UseCQLEngine, // Enable vaidshala CQL engine if configured
	}

	// Initialize FHIR client - Google FHIR is required
	if cfg.GoogleCloudProjectID == "" {
		logger.Fatal("GOOGLE_CLOUD_PROJECT_ID is required - Google FHIR client must be configured")
	}

	fhirCfg := fhir.ClientConfig{
		ProjectID:       cfg.GoogleCloudProjectID,
		Location:        cfg.GoogleCloudLocation,
		DatasetID:       cfg.GoogleCloudDatasetID,
		FHIRStoreID:     cfg.GoogleCloudFHIRStoreID,
		CredentialsPath: cfg.GoogleCredentialsPath,
		Timeout:         cfg.FHIRTimeout,
	}
	s.fhirClient = fhir.NewClient(fhirCfg, logger)
	s.queryBuilder = fhir.NewQueryBuilder(s.fhirClient, logger)
	logger.Info("Google FHIR client configured",
		zap.String("project", cfg.GoogleCloudProjectID),
		zap.String("location", cfg.GoogleCloudLocation),
		zap.String("dataset", cfg.GoogleCloudDatasetID),
		zap.String("fhirStore", cfg.GoogleCloudFHIRStoreID),
	)

	// Initialize vaidshala CQL executor and context builder
	cqlConfig := cql.ExecutorConfig{
		Region: cfg.Region, // AU, IN, US
	}
	s.cqlExecutor = cql.NewExecutor(cqlConfig, logger)
	s.contextBuilder = cql.NewContextBuilder(s.fhirClient, logger)
	logger.Info("Vaidshala CQL executor initialized",
		zap.String("region", cfg.Region),
		zap.Bool("cql_engine_enabled", s.useCQLEngine),
	)

	// Initialize KB-3 Temporal/Guidelines client (Tier 7 integration)
	kb3Config := kb3.ClientConfig{
		BaseURL: cfg.KB3URL,
		Timeout: cfg.KB3Timeout,
		Enabled: cfg.KB3Enabled,
	}
	s.kb3Client = kb3.NewClient(kb3Config, logger)
	s.kb3Integration = kb3.NewIntegration(s.kb3Client, logger)
	logger.Info("KB-3 Temporal integration initialized",
		zap.String("kb3_url", cfg.KB3URL),
		zap.Bool("kb3_enabled", cfg.KB3Enabled),
	)

	// Initialize available measures
	s.initMeasures()

	return s
}

// Initialize initializes the FHIR client connection.
func (s *Service) Initialize(ctx context.Context) error {
	if s.fhirClient != nil {
		return s.fhirClient.Initialize(ctx)
	}
	return nil
}

// HealthCheck performs a health check on FHIR connectivity.
func (s *Service) HealthCheck(ctx context.Context) error {
	if s.fhirClient != nil {
		return s.fhirClient.HealthCheck(ctx)
	}
	return nil // No FHIR client configured
}

// initMeasures initializes the available quality measures.
func (s *Service) initMeasures() {
	s.measures[models.MeasureCMS122DiabetesHbA1c] = models.MeasureInfo{
		Type:        models.MeasureCMS122DiabetesHbA1c,
		CMSID:       "CMS122v11",
		Name:        "Diabetes: Hemoglobin A1c Poor Control",
		Description: "Patients 18-75 years of age with diabetes who had hemoglobin A1c > 9.0%",
		Domain:      "Chronic Disease",
		Steward:     "National Committee for Quality Assurance",
		Version:     "11.0.0",
		CQLLibrary:  "CMS122-DiabetesHbA1c",
	}

	s.measures[models.MeasureCMS165BPControl] = models.MeasureInfo{
		Type:        models.MeasureCMS165BPControl,
		CMSID:       "CMS165v11",
		Name:        "Controlling High Blood Pressure",
		Description: "Patients 18-85 years of age who had a diagnosis of hypertension and whose BP was adequately controlled during the measurement period",
		Domain:      "Chronic Disease",
		Steward:     "National Committee for Quality Assurance",
		Version:     "11.0.0",
		CQLLibrary:  "CMS165-BloodPressure",
	}

	s.measures[models.MeasureCMS130ColorectalScreening] = models.MeasureInfo{
		Type:        models.MeasureCMS130ColorectalScreening,
		CMSID:       "CMS130v11",
		Name:        "Colorectal Cancer Screening",
		Description: "Patients 50-75 years of age who had appropriate screening for colorectal cancer",
		Domain:      "Preventive Care",
		Steward:     "National Committee for Quality Assurance",
		Version:     "11.0.0",
		CQLLibrary:  "CMS130-ColorectalScreening",
	}

	s.measures[models.MeasureCMS2DepressionScreening] = models.MeasureInfo{
		Type:        models.MeasureCMS2DepressionScreening,
		CMSID:       "CMS2v12",
		Name:        "Preventive Care and Screening: Screening for Depression and Follow-Up Plan",
		Description: "Patients aged 12 years and older screened for depression using an age-appropriate standardized tool and follow-up documented",
		Domain:      "Behavioral Health",
		Steward:     "Centers for Medicare & Medicaid Services",
		Version:     "12.0.0",
		CQLLibrary:  "CMS2-DepressionScreening",
	}

	s.measures[models.MeasureIndiaDiabetesCare] = models.MeasureInfo{
		Type:        models.MeasureIndiaDiabetesCare,
		CMSID:       "INDIA-DM-001",
		Name:        "India Diabetes Comprehensive Care",
		Description: "Comprehensive annual diabetes care including HbA1c, kidney function, foot exam, and eye exam",
		Domain:      "Chronic Disease",
		Steward:     "ICMR",
		Version:     "1.0.0",
		CQLLibrary:  "IndiaDiabetesCare",
	}

	s.measures[models.MeasureIndiaHypertensionCare] = models.MeasureInfo{
		Type:        models.MeasureIndiaHypertensionCare,
		CMSID:       "INDIA-HTN-001",
		Name:        "India Hypertension Care",
		Description: "Blood pressure control with kidney function monitoring for hypertensive patients",
		Domain:      "Chronic Disease",
		Steward:     "ICMR",
		Version:     "1.0.0",
		CQLLibrary:  "IndiaHypertensionCare",
	}

	s.logger.Info("Initialized quality measures",
		zap.Int("measure_count", len(s.measures)),
	)
}

// GetPatientCareGaps returns care gaps for a patient.
func (s *Service) GetPatientCareGaps(
	ctx context.Context,
	patientID string,
	measures []models.MeasureType,
	period models.Period,
	includeClosedGaps bool,
	includeEvidence bool,
) (*models.CareGapReport, error) {
	s.logger.Info("Getting care gaps for patient",
		zap.String("patient_id", patientID),
		zap.Int("measure_count", len(measures)),
	)

	// If no measures specified, evaluate all
	if len(measures) == 0 {
		for m := range s.measures {
			measures = append(measures, m)
		}
	}

	// Collect open gaps
	var openGaps []models.CareGap
	var closedGaps []models.CareGap

	for _, measureType := range measures {
		measure, exists := s.measures[measureType]
		if !exists {
			continue
		}

		// Evaluate measure and detect gaps
		gap := s.evaluateMeasureForGaps(ctx, patientID, measure, period, includeEvidence)
		if gap != nil {
			if gap.Status == models.GapStatusOpen {
				openGaps = append(openGaps, *gap)
			} else if includeClosedGaps && gap.Status == models.GapStatusClosed {
				closedGaps = append(closedGaps, *gap)
			}
		}
	}

	// Build summary
	summary := s.buildSummary(openGaps)

	report := &models.CareGapReport{
		PatientID:         patientID,
		ReportDate:        time.Now().UTC(),
		MeasurementPeriod: period,
		OpenGaps:          openGaps,
		Summary:           summary,
		DataCompleteness:  models.DataComplete,
	}

	if includeClosedGaps {
		report.ClosedGaps = closedGaps
	}

	return report, nil
}

// evaluateMeasureForGaps evaluates a single measure and returns any gaps.
func (s *Service) evaluateMeasureForGaps(
	ctx context.Context,
	patientID string,
	measure models.MeasureInfo,
	period models.Period,
	includeEvidence bool,
) *models.CareGap {
	var inDenominator bool
	var inNumerator bool
	var gapReason string
	var evidence *models.CQLEvidence

	// Convert period to FHIR format
	fhirPeriod := &fhir.Period{
		Start: period.Start.Format("2006-01-02"),
		End:   period.End.Format("2006-01-02"),
	}

	// Evaluate measure using FHIR data from Google Healthcare API
	switch measure.Type {
	case models.MeasureCMS122DiabetesHbA1c:
		inDenominator, inNumerator, gapReason, evidence = s.evaluateDiabetesHbA1c(ctx, patientID, fhirPeriod, includeEvidence)

	case models.MeasureCMS165BPControl:
		inDenominator, inNumerator, gapReason, evidence = s.evaluateBPControl(ctx, patientID, fhirPeriod, includeEvidence)

	case models.MeasureCMS2DepressionScreening:
		inDenominator, inNumerator, gapReason, evidence = s.evaluateDepressionScreening(ctx, patientID, fhirPeriod, includeEvidence)

	case models.MeasureCMS130ColorectalScreening:
		inDenominator, inNumerator, gapReason, evidence = s.evaluateColorectalScreening(ctx, patientID, fhirPeriod, includeEvidence)

	case models.MeasureIndiaDiabetesCare, models.MeasureIndiaHypertensionCare:
		// India measures use FHIR data with extended criteria
		inDenominator, inNumerator, gapReason, evidence = s.evaluateIndiaMeasure(ctx, patientID, measure.Type, fhirPeriod, includeEvidence)

	default:
		s.logger.Warn("Unknown measure type", zap.String("measure", string(measure.Type)))
		return nil
	}

	if !inDenominator {
		return nil // Patient not eligible for this measure
	}

	if inNumerator {
		// No gap - patient met the quality target
		return &models.CareGap{
			ID:             uuid.New().String(),
			Measure:        measure,
			Status:         models.GapStatusClosed,
			Priority:       models.GapPriorityLow,
			Reason:         "Patient met quality target",
			Recommendation: "No action required",
			IdentifiedDate: time.Now().UTC(),
		}
	}

	// Gap exists
	gap := &models.CareGap{
		ID:             uuid.New().String(),
		Measure:        measure,
		Status:         models.GapStatusOpen,
		Priority:       s.calculatePriority(measure),
		Reason:         gapReason,
		Recommendation: s.getRecommendation(measure),
		IdentifiedDate: time.Now().UTC(),
		Interventions:  s.getInterventions(measure),
	}

	// Add evidence if requested
	if includeEvidence {
		if evidence != nil {
			gap.Evidence = evidence
		} else {
			gap.Evidence = s.buildEvidence(measure, patientID)
		}
	}

	return gap
}

// evaluateDiabetesHbA1c evaluates CMS122 diabetes HbA1c measure.
func (s *Service) evaluateDiabetesHbA1c(
	ctx context.Context,
	patientID string,
	period *fhir.Period,
	includeEvidence bool,
) (inDenominator, inNumerator bool, reason string, evidence *models.CQLEvidence) {
	data, err := s.queryBuilder.GetDiabetesData(ctx, patientID, period)
	if err != nil {
		s.logger.Warn("Failed to get diabetes data", zap.Error(err))
		return true, false, "Unable to evaluate - data fetch error", nil
	}

	// Denominator: Patient has diabetes diagnosis
	inDenominator = data.HasDiabetes
	if !inDenominator {
		return false, false, "", nil
	}

	// Numerator: HbA1c <= 9.0% within measurement period
	if data.MostRecentHbA1c != nil && data.HbA1cInPeriod && !data.HbA1cPoorControl {
		inNumerator = true
		reason = fmt.Sprintf("HbA1c %.1f%% is within target", data.HbA1cValue)
	} else if data.MostRecentHbA1c == nil {
		reason = "No HbA1c result found in measurement period"
	} else if data.HbA1cPoorControl {
		reason = fmt.Sprintf("HbA1c %.1f%% exceeds 9.0%% target", data.HbA1cValue)
	} else {
		reason = "HbA1c test not performed within measurement period"
	}

	if includeEvidence {
		evidence = &models.CQLEvidence{
			LibraryID:      "CMS122-DiabetesHbA1c",
			LibraryVersion: "11.0.0",
			Populations: []models.PopulationMembership{
				{Population: models.PopulationDenominator, IsMember: inDenominator, Reason: "Has diabetes diagnosis"},
				{Population: models.PopulationNumerator, IsMember: inNumerator, Reason: reason},
			},
			DataElements: []models.EvaluatedDataElement{
				{Name: "Has Diabetes", Value: boolPtr(data.HasDiabetes), ContributedToGap: !data.HasDiabetes},
				{Name: "Most Recent HbA1c", Value: floatToStringPtr(data.HbA1cValue), ContributedToGap: data.HbA1cPoorControl},
			},
			EvaluatedAt: time.Now().UTC(),
		}
	}

	return
}

// evaluateBPControl evaluates CMS165 blood pressure control measure.
func (s *Service) evaluateBPControl(
	ctx context.Context,
	patientID string,
	period *fhir.Period,
	includeEvidence bool,
) (inDenominator, inNumerator bool, reason string, evidence *models.CQLEvidence) {
	data, err := s.queryBuilder.GetBloodPressureData(ctx, patientID, period)
	if err != nil {
		s.logger.Warn("Failed to get BP data", zap.Error(err))
		return true, false, "Unable to evaluate - data fetch error", nil
	}

	// Denominator: Patient has hypertension diagnosis
	inDenominator = data.HasHypertension
	if !inDenominator {
		return false, false, "", nil
	}

	// Numerator: BP < 140/90 within measurement period
	if data.BPInPeriod && data.BPControlled {
		inNumerator = true
		reason = fmt.Sprintf("BP %.0f/%.0f is controlled", data.SystolicValue, data.DiastolicValue)
	} else if data.SystolicValue == 0 || data.DiastolicValue == 0 {
		reason = "No blood pressure reading found in measurement period"
	} else {
		reason = fmt.Sprintf("BP %.0f/%.0f exceeds 140/90 target", data.SystolicValue, data.DiastolicValue)
	}

	if includeEvidence {
		evidence = &models.CQLEvidence{
			LibraryID:      "CMS165-BloodPressure",
			LibraryVersion: "11.0.0",
			Populations: []models.PopulationMembership{
				{Population: models.PopulationDenominator, IsMember: inDenominator, Reason: "Has hypertension diagnosis"},
				{Population: models.PopulationNumerator, IsMember: inNumerator, Reason: reason},
			},
			DataElements: []models.EvaluatedDataElement{
				{Name: "Systolic BP", Value: floatToStringPtr(data.SystolicValue), ContributedToGap: data.SystolicValue >= 140},
				{Name: "Diastolic BP", Value: floatToStringPtr(data.DiastolicValue), ContributedToGap: data.DiastolicValue >= 90},
			},
			EvaluatedAt: time.Now().UTC(),
		}
	}

	return
}

// evaluateDepressionScreening evaluates CMS2 depression screening measure.
func (s *Service) evaluateDepressionScreening(
	ctx context.Context,
	patientID string,
	period *fhir.Period,
	includeEvidence bool,
) (inDenominator, inNumerator bool, reason string, evidence *models.CQLEvidence) {
	data, err := s.queryBuilder.GetDepressionScreeningData(ctx, patientID, period)
	if err != nil {
		s.logger.Warn("Failed to get depression data", zap.Error(err))
		return true, false, "Unable to evaluate - data fetch error", nil
	}

	// All patients >= 12 are in denominator (simplified)
	inDenominator = true

	// Numerator: Screening performed and (negative OR positive with follow-up)
	if data.HasScreening && data.ScreeningInPeriod {
		if !data.PositiveScreen || (data.PositiveScreen && data.HasFollowUp) {
			inNumerator = true
			reason = "Depression screening completed with appropriate follow-up"
		} else {
			reason = "Positive PHQ-2 screen without documented PHQ-9 follow-up"
		}
	} else {
		reason = "No depression screening documented in measurement period"
	}

	if includeEvidence {
		evidence = &models.CQLEvidence{
			LibraryID:      "CMS2-DepressionScreening",
			LibraryVersion: "12.0.0",
			Populations: []models.PopulationMembership{
				{Population: models.PopulationDenominator, IsMember: inDenominator, Reason: "Patient >= 12 years"},
				{Population: models.PopulationNumerator, IsMember: inNumerator, Reason: reason},
			},
			DataElements: []models.EvaluatedDataElement{
				{Name: "Has Screening", Value: boolPtr(data.HasScreening), ContributedToGap: !data.HasScreening},
				{Name: "Positive Screen", Value: boolPtr(data.PositiveScreen), ContributedToGap: data.PositiveScreen && !data.HasFollowUp},
			},
			EvaluatedAt: time.Now().UTC(),
		}
	}

	return
}

// evaluateColorectalScreening evaluates CMS130 colorectal screening measure.
func (s *Service) evaluateColorectalScreening(
	ctx context.Context,
	patientID string,
	period *fhir.Period,
	includeEvidence bool,
) (inDenominator, inNumerator bool, reason string, evidence *models.CQLEvidence) {
	data, err := s.queryBuilder.GetColorectalScreeningData(ctx, patientID, period)
	if err != nil {
		s.logger.Warn("Failed to get colorectal data", zap.Error(err))
		return true, false, "Unable to evaluate - data fetch error", nil
	}

	// Denominator: Age 50-75
	inDenominator = data.Age >= 50 && data.Age <= 75
	if !inDenominator {
		return false, false, "", nil
	}

	// Numerator: Colonoscopy within 10 years OR FOBT/FIT within 1 year
	inNumerator = data.ScreeningUpToDate
	if inNumerator {
		if data.HasColonoscopy {
			reason = fmt.Sprintf("Colonoscopy performed on %s", data.ColonoscopyDate.Format("2006-01-02"))
		} else {
			reason = fmt.Sprintf("FOBT/FIT performed on %s", data.FOBTDate.Format("2006-01-02"))
		}
	} else {
		reason = "No colonoscopy in past 10 years or FOBT/FIT in past year"
	}

	if includeEvidence {
		evidence = &models.CQLEvidence{
			LibraryID:      "CMS130-ColorectalScreening",
			LibraryVersion: "11.0.0",
			Populations: []models.PopulationMembership{
				{Population: models.PopulationDenominator, IsMember: inDenominator, Reason: fmt.Sprintf("Age %d (50-75)", data.Age)},
				{Population: models.PopulationNumerator, IsMember: inNumerator, Reason: reason},
			},
			DataElements: []models.EvaluatedDataElement{
				{Name: "Age", Value: intToStringPtr(data.Age), ContributedToGap: false},
				{Name: "Has Colonoscopy", Value: boolPtr(data.HasColonoscopy), ContributedToGap: !data.HasColonoscopy},
				{Name: "Has FOBT/FIT", Value: boolPtr(data.HasFOBT), ContributedToGap: !data.HasFOBT},
			},
			EvaluatedAt: time.Now().UTC(),
		}
	}

	return
}

// evaluateIndiaMeasure evaluates India-specific measures (ICMR guidelines).
func (s *Service) evaluateIndiaMeasure(
	ctx context.Context,
	patientID string,
	measureType models.MeasureType,
	period *fhir.Period,
	includeEvidence bool,
) (inDenominator, inNumerator bool, reason string, evidence *models.CQLEvidence) {
	// India measures combine multiple FHIR queries
	switch measureType {
	case models.MeasureIndiaDiabetesCare:
		// Get diabetes data (reuses CMS122 logic)
		diabetesData, err := s.queryBuilder.GetDiabetesData(ctx, patientID, period)
		if err != nil {
			s.logger.Warn("Failed to get diabetes data for India measure", zap.Error(err))
			return true, false, "Unable to evaluate - data fetch error", nil
		}

		inDenominator = diabetesData.HasDiabetes
		if !inDenominator {
			return false, false, "", nil
		}

		// India measure requires HbA1c + kidney function (eGFR)
		hasHbA1c := diabetesData.MostRecentHbA1c != nil && diabetesData.HbA1cInPeriod
		// For now, assume kidney function check passes if HbA1c is present
		// Full implementation would query for eGFR observations
		inNumerator = hasHbA1c && !diabetesData.HbA1cPoorControl

		if inNumerator {
			reason = fmt.Sprintf("India Diabetes Care: HbA1c %.1f%% within target", diabetesData.HbA1cValue)
		} else if !hasHbA1c {
			reason = "India Diabetes Care: Missing HbA1c or kidney function tests"
		} else {
			reason = fmt.Sprintf("India Diabetes Care: HbA1c %.1f%% exceeds target", diabetesData.HbA1cValue)
		}

		if includeEvidence {
			evidence = &models.CQLEvidence{
				LibraryID:      "IndiaDiabetesCare",
				LibraryVersion: "1.0.0",
				Populations: []models.PopulationMembership{
					{Population: models.PopulationDenominator, IsMember: inDenominator, Reason: "Has diabetes diagnosis"},
					{Population: models.PopulationNumerator, IsMember: inNumerator, Reason: reason},
				},
				DataElements: []models.EvaluatedDataElement{
					{Name: "Has Diabetes", Value: boolPtr(diabetesData.HasDiabetes), ContributedToGap: false},
					{Name: "HbA1c Value", Value: floatToStringPtr(diabetesData.HbA1cValue), ContributedToGap: diabetesData.HbA1cPoorControl},
				},
				EvaluatedAt: time.Now().UTC(),
			}
		}

	case models.MeasureIndiaHypertensionCare:
		// Get BP data (reuses CMS165 logic)
		bpData, err := s.queryBuilder.GetBloodPressureData(ctx, patientID, period)
		if err != nil {
			s.logger.Warn("Failed to get BP data for India measure", zap.Error(err))
			return true, false, "Unable to evaluate - data fetch error", nil
		}

		inDenominator = bpData.HasHypertension
		if !inDenominator {
			return false, false, "", nil
		}

		// India hypertension requires BP control + kidney function monitoring
		inNumerator = bpData.BPControlled && bpData.BPInPeriod

		if inNumerator {
			reason = fmt.Sprintf("India HTN Care: BP %.0f/%.0f controlled", bpData.SystolicValue, bpData.DiastolicValue)
		} else if bpData.SystolicValue == 0 {
			reason = "India HTN Care: No blood pressure documented"
		} else {
			reason = fmt.Sprintf("India HTN Care: BP %.0f/%.0f uncontrolled", bpData.SystolicValue, bpData.DiastolicValue)
		}

		if includeEvidence {
			evidence = &models.CQLEvidence{
				LibraryID:      "IndiaHypertensionCare",
				LibraryVersion: "1.0.0",
				Populations: []models.PopulationMembership{
					{Population: models.PopulationDenominator, IsMember: inDenominator, Reason: "Has hypertension diagnosis"},
					{Population: models.PopulationNumerator, IsMember: inNumerator, Reason: reason},
				},
				DataElements: []models.EvaluatedDataElement{
					{Name: "Systolic BP", Value: floatToStringPtr(bpData.SystolicValue), ContributedToGap: bpData.SystolicValue >= 140},
					{Name: "Diastolic BP", Value: floatToStringPtr(bpData.DiastolicValue), ContributedToGap: bpData.DiastolicValue >= 90},
				},
				EvaluatedAt: time.Now().UTC(),
			}
		}
	}

	return
}

// Helper functions for evidence building
func boolPtr(b bool) *string {
	s := fmt.Sprintf("%t", b)
	return &s
}

func floatToStringPtr(f float64) *string {
	if f == 0 {
		return nil
	}
	s := fmt.Sprintf("%.1f", f)
	return &s
}

func intToStringPtr(i int) *string {
	s := fmt.Sprintf("%d", i)
	return &s
}

// calculatePriority determines gap priority based on measure.
func (s *Service) calculatePriority(measure models.MeasureInfo) models.GapPriority {
	switch measure.Type {
	case models.MeasureCMS122DiabetesHbA1c:
		return models.GapPriorityHigh
	case models.MeasureCMS165BPControl:
		return models.GapPriorityHigh
	case models.MeasureCMS130ColorectalScreening:
		return models.GapPriorityMedium
	case models.MeasureCMS2DepressionScreening:
		return models.GapPriorityMedium
	default:
		return models.GapPriorityLow
	}
}

// getGapReason returns a human-readable reason for the gap.
func (s *Service) getGapReason(measure models.MeasureInfo) string {
	switch measure.Type {
	case models.MeasureCMS122DiabetesHbA1c:
		return "HbA1c level above 9% target or no recent HbA1c result"
	case models.MeasureCMS165BPControl:
		return "Blood pressure not adequately controlled (<140/90)"
	case models.MeasureCMS130ColorectalScreening:
		return "No colorectal cancer screening documented in measurement period"
	case models.MeasureCMS2DepressionScreening:
		return "No depression screening documented in measurement period"
	case models.MeasureIndiaDiabetesCare:
		return "Incomplete annual diabetes care components"
	case models.MeasureIndiaHypertensionCare:
		return "Blood pressure or kidney function monitoring not current"
	default:
		return "Quality measure target not met"
	}
}

// getRecommendation returns a recommendation for closing the gap.
func (s *Service) getRecommendation(measure models.MeasureInfo) string {
	switch measure.Type {
	case models.MeasureCMS122DiabetesHbA1c:
		return "Review diabetes management and order HbA1c test if due"
	case models.MeasureCMS165BPControl:
		return "Review antihypertensive therapy and document blood pressure"
	case models.MeasureCMS130ColorectalScreening:
		return "Order colonoscopy, FIT, or FOBT for colorectal cancer screening"
	case models.MeasureCMS2DepressionScreening:
		return "Perform depression screening using PHQ-2/PHQ-9"
	case models.MeasureIndiaDiabetesCare:
		return "Complete annual diabetes assessment including HbA1c, eGFR, foot exam, eye exam"
	case models.MeasureIndiaHypertensionCare:
		return "Document blood pressure and order serum creatinine/eGFR"
	default:
		return "Address quality measure requirements"
	}
}

// getInterventions returns suggested interventions for the gap.
func (s *Service) getInterventions(measure models.MeasureInfo) []models.Intervention {
	switch measure.Type {
	case models.MeasureCMS122DiabetesHbA1c:
		return []models.Intervention{
			{
				Type:        models.InterventionLabOrder,
				Description: "Order HbA1c test",
				Code:        "4548-4",
				CodeSystem:  "http://loinc.org",
				Priority:    models.GapPriorityHigh,
			},
			{
				Type:        models.InterventionPatientEducation,
				Description: "Diabetes self-management education",
				Priority:    models.GapPriorityMedium,
			},
		}
	case models.MeasureCMS165BPControl:
		return []models.Intervention{
			{
				Type:        models.InterventionScreening,
				Description: "Document blood pressure measurement",
				Code:        "85354-9",
				CodeSystem:  "http://loinc.org",
				Priority:    models.GapPriorityHigh,
			},
		}
	case models.MeasureCMS130ColorectalScreening:
		return []models.Intervention{
			{
				Type:        models.InterventionProcedureOrder,
				Description: "Order colonoscopy",
				Code:        "73761001",
				CodeSystem:  "http://snomed.info/sct",
				Priority:    models.GapPriorityMedium,
			},
			{
				Type:        models.InterventionLabOrder,
				Description: "Order FIT (fecal immunochemical test)",
				Code:        "57905-2",
				CodeSystem:  "http://loinc.org",
				Priority:    models.GapPriorityMedium,
			},
		}
	case models.MeasureCMS2DepressionScreening:
		return []models.Intervention{
			{
				Type:        models.InterventionScreening,
				Description: "Administer PHQ-2/PHQ-9 screening",
				Code:        "44261-6",
				CodeSystem:  "http://loinc.org",
				Priority:    models.GapPriorityMedium,
			},
		}
	default:
		return nil
	}
}

// buildEvidence creates CQL evidence for the gap.
func (s *Service) buildEvidence(measure models.MeasureInfo, patientID string) *models.CQLEvidence {
	return &models.CQLEvidence{
		LibraryID:      measure.CQLLibrary,
		LibraryVersion: measure.Version,
		Populations: []models.PopulationMembership{
			{Population: models.PopulationDenominator, IsMember: true, Reason: "Patient meets inclusion criteria"},
			{Population: models.PopulationNumerator, IsMember: false, Reason: "Quality target not met"},
			{Population: models.PopulationDenominatorExcl, IsMember: false},
		},
		DataElements: []models.EvaluatedDataElement{
			{Name: "Most Recent Result", Value: nil, ContributedToGap: true},
		},
		EvaluatedAt: time.Now().UTC(),
	}
}

// buildSummary creates a summary of care gaps.
func (s *Service) buildSummary(gaps []models.CareGap) models.CareGapSummary {
	summary := models.CareGapSummary{
		TotalOpenGaps: len(gaps),
		GapsByDomain:  []models.DomainGapCount{},
	}

	domainCounts := make(map[string]int)

	for _, gap := range gaps {
		// Count by priority
		switch gap.Priority {
		case models.GapPriorityUrgent:
			summary.UrgentGaps++
		case models.GapPriorityHigh:
			summary.HighPriorityGaps++
		}

		// Count by domain
		domainCounts[gap.Measure.Domain]++
	}

	for domain, count := range domainCounts {
		summary.GapsByDomain = append(summary.GapsByDomain, models.DomainGapCount{
			Domain: domain,
			Count:  count,
		})
	}

	// Calculate quality score (simplified: 100 - (gaps * 10))
	score := 100.0 - float64(len(gaps)*10)
	if score < 0 {
		score = 0
	}
	summary.QualityScore = &score

	return summary
}

// EvaluateMeasure evaluates a single measure for a patient.
func (s *Service) EvaluateMeasure(
	ctx context.Context,
	patientID string,
	measureType models.MeasureType,
	period models.Period,
) (*models.MeasureReport, error) {
	measure, exists := s.measures[measureType]
	if !exists {
		return nil, fmt.Errorf("unknown measure type: %s", measureType)
	}

	// Evaluate the measure
	// In production, this would call the vaidshala CQL engine

	report := &models.MeasureReport{
		ID:        uuid.New().String(),
		Measure:   measure,
		PatientID: patientID,
		Period:    period,
		Status:    "complete",
		Type:      "individual",
		Populations: []models.PopulationResult{
			{Population: models.PopulationInitial, Count: 1},
			{Population: models.PopulationDenominator, Count: 1},
			{Population: models.PopulationNumerator, Count: 0},
		},
		GeneratedAt: time.Now().UTC(),
	}

	return report, nil
}

// EvaluatePopulation evaluates a measure across a population.
func (s *Service) EvaluatePopulation(
	ctx context.Context,
	patientIDs []string,
	measureType models.MeasureType,
	period models.Period,
	limit int,
) (*models.PopulationMeasureReport, error) {
	measure, exists := s.measures[measureType]
	if !exists {
		return nil, fmt.Errorf("unknown measure type: %s", measureType)
	}

	startTime := time.Now()

	// Limit patient count
	if len(patientIDs) > limit {
		patientIDs = patientIDs[:limit]
	}

	// Evaluate for each patient
	var patientsWithGaps []models.PatientGapSummary
	numeratorCount := 0

	for _, patientID := range patientIDs {
		// Simulate evaluation - in production would use CQL engine
		hasGap := true // Simplified - would be actual CQL result

		if hasGap {
			patientsWithGaps = append(patientsWithGaps, models.PatientGapSummary{
				PatientID:      patientID,
				Status:         models.GapStatusOpen,
				Recommendation: s.getRecommendation(measure),
			})
		} else {
			numeratorCount++
		}
	}

	// Calculate performance rate
	performanceRate := float64(numeratorCount) / float64(len(patientIDs)) * 100

	report := &models.PopulationMeasureReport{
		ID:            uuid.New().String(),
		Measure:       measure,
		Period:        period,
		TotalPatients: len(patientIDs),
		Populations: []models.PopulationResult{
			{Population: models.PopulationInitial, Count: len(patientIDs)},
			{Population: models.PopulationDenominator, Count: len(patientIDs)},
			{Population: models.PopulationNumerator, Count: numeratorCount},
		},
		PerformanceRate:  &performanceRate,
		PatientsWithGaps: patientsWithGaps,
		GeneratedAt:      time.Now().UTC(),
		ProcessingTimeMs: int(time.Since(startTime).Milliseconds()),
	}

	return report, nil
}

// GetAvailableMeasures returns all available quality measures.
func (s *Service) GetAvailableMeasures() []models.MeasureInfo {
	measures := make([]models.MeasureInfo, 0, len(s.measures))
	for _, m := range s.measures {
		measures = append(measures, m)
	}
	return measures
}

// GetMeasureInfo returns information about a specific measure.
func (s *Service) GetMeasureInfo(measureType models.MeasureType) (*models.MeasureInfo, error) {
	measure, exists := s.measures[measureType]
	if !exists {
		return nil, fmt.Errorf("unknown measure type: %s", measureType)
	}
	return &measure, nil
}

// RecordGapAddressed records that a gap has been addressed.
func (s *Service) RecordGapAddressed(
	ctx context.Context,
	patientID string,
	gapID string,
	intervention models.InterventionType,
	notes string,
) (*models.CareGap, error) {
	// In production, this would update the gap status in storage
	now := time.Now().UTC()
	return &models.CareGap{
		ID:             gapID,
		Status:         models.GapStatusPending,
		Priority:       models.GapPriorityLow,
		Reason:         "Intervention recorded",
		Recommendation: "Await confirmation of intervention completion",
		IdentifiedDate: now,
	}, nil
}

// DismissGap dismisses a gap with a reason.
func (s *Service) DismissGap(
	ctx context.Context,
	patientID string,
	gapID string,
	reason string,
) (*models.CareGap, error) {
	now := time.Now().UTC()
	return &models.CareGap{
		ID:             gapID,
		Status:         models.GapStatusExcluded,
		Priority:       models.GapPriorityLow,
		Reason:         reason,
		Recommendation: "Gap dismissed by clinician",
		IdentifiedDate: now,
		ClosedDate:     &now,
	}, nil
}

// SnoozeGap snoozes a gap until a future date.
func (s *Service) SnoozeGap(
	ctx context.Context,
	patientID string,
	gapID string,
	snoozeUntil time.Time,
	reason string,
) (*models.CareGap, error) {
	now := time.Now().UTC()
	return &models.CareGap{
		ID:             gapID,
		Status:         models.GapStatusPending,
		Priority:       models.GapPriorityLow,
		Reason:         fmt.Sprintf("Snoozed until %s: %s", snoozeUntil.Format("2006-01-02"), reason),
		Recommendation: "Review gap after snooze period",
		IdentifiedDate: now,
		DueDate:        &snoozeUntil,
	}, nil
}

// ============================================================================
// VAIDSHALA CQL ENGINE INTEGRATION
// ============================================================================

// EvaluateWithCQL evaluates care gaps using the vaidshala CQL engine.
// This is the preferred method when CQL engine is enabled.
//
// Flow:
//  1. Build ClinicalExecutionContext from FHIR patient data
//  2. Run vaidshala CQL Engine to determine clinical facts
//  3. Run vaidshala Measure Engine to evaluate care gaps
//  4. Convert MeasureResults to KB-9 CareGap models
func (s *Service) EvaluateWithCQL(
	ctx context.Context,
	patientID string,
	period models.Period,
) (*models.CareGapReport, error) {
	startTime := time.Now()

	s.logger.Info("Evaluating care gaps with vaidshala CQL engine",
		zap.String("patient_id", patientID),
		zap.String("region", s.config.Region),
	)

	// Step 1: Build ClinicalExecutionContext from FHIR data
	measurementPeriod := contracts.Period{
		Start: &period.Start,
		End:   &period.End,
	}

	execCtx, err := s.contextBuilder.BuildContext(ctx, patientID, measurementPeriod, s.config.Region)
	if err != nil {
		s.logger.Error("Failed to build clinical execution context", zap.Error(err))
		return nil, fmt.Errorf("failed to build execution context: %w", err)
	}

	// Step 2 & 3: Run CQL → Measure Engine pipeline
	result, err := s.cqlExecutor.Evaluate(ctx, execCtx)
	if err != nil {
		s.logger.Error("CQL evaluation failed", zap.Error(err))
		return nil, fmt.Errorf("CQL evaluation failed: %w", err)
	}

	s.logger.Info("CQL evaluation completed",
		zap.Int("clinical_facts", len(result.ClinicalFacts)),
		zap.Int("measure_results", len(result.MeasureResults)),
		zap.Int64("execution_ms", result.ExecutionTimeMs),
	)

	// Step 4: Convert MeasureResults to CareGaps
	openGaps, closedGaps := s.convertMeasureResultsToGaps(result.MeasureResults, result.ClinicalFacts)

	// Build summary
	summary := s.buildSummary(openGaps)

	report := &models.CareGapReport{
		PatientID:         patientID,
		ReportDate:        time.Now().UTC(),
		MeasurementPeriod: period,
		OpenGaps:          openGaps,
		ClosedGaps:        closedGaps,
		Summary:           summary,
		DataCompleteness:  models.DataComplete,
	}

	s.logger.Info("Care gap report generated via CQL engine",
		zap.String("patient_id", patientID),
		zap.Int("open_gaps", len(openGaps)),
		zap.Int("closed_gaps", len(closedGaps)),
		zap.Duration("total_duration", time.Since(startTime)),
	)

	return report, nil
}

// convertMeasureResultsToGaps converts vaidshala MeasureResults to KB-9 CareGaps.
func (s *Service) convertMeasureResultsToGaps(
	measureResults []contracts.MeasureResult,
	clinicalFacts []contracts.ClinicalFact,
) (openGaps []models.CareGap, closedGaps []models.CareGap) {
	// Build fact lookup map
	factMap := make(map[string]contracts.ClinicalFact)
	for _, fact := range clinicalFacts {
		factMap[fact.FactID] = fact
	}

	for _, mr := range measureResults {
		// Get measure info
		measureType := s.mapCMSIDToMeasureType(mr.MeasureID)
		measure, exists := s.measures[measureType]
		if !exists {
			// Create a basic measure info if not found
			measure = models.MeasureInfo{
				Type:        measureType,
				CMSID:       mr.MeasureID,
				Name:        mr.MeasureName,
				Description: mr.Rationale,
				Version:     mr.MeasureVersion,
			}
		}

		gap := models.CareGap{
			ID:             uuid.New().String(),
			Measure:        measure,
			IdentifiedDate: time.Now().UTC(),
			Evidence: &models.CQLEvidence{
				LibraryID:      mr.MeasureID,
				LibraryVersion: mr.MeasureVersion,
				Populations: []models.PopulationMembership{
					{Population: models.PopulationDenominator, IsMember: mr.InDenominator, Reason: "Measure denominator"},
					{Population: models.PopulationNumerator, IsMember: mr.InNumerator, Reason: mr.Rationale},
				},
				EvaluatedAt: time.Now().UTC(),
			},
		}

		if mr.CareGapIdentified {
			// Open gap
			gap.Status = models.GapStatusOpen
			gap.Priority = s.mapPriority(mr.MeasureID)
			gap.Reason = mr.Rationale
			gap.Recommendation = s.getRecommendation(measure)
			gap.Interventions = s.getInterventions(measure)
			openGaps = append(openGaps, gap)
		} else if mr.InDenominator && mr.InNumerator {
			// Closed gap (patient met the target)
			gap.Status = models.GapStatusClosed
			gap.Priority = models.GapPriorityLow
			gap.Reason = mr.Rationale
			gap.Recommendation = "No action required"
			closedGaps = append(closedGaps, gap)
		}
	}

	return openGaps, closedGaps
}

// mapCMSIDToMeasureType maps CMS measure IDs to KB-9 MeasureType.
func (s *Service) mapCMSIDToMeasureType(cmsID string) models.MeasureType {
	switch cmsID {
	case "CMS122":
		return models.MeasureCMS122DiabetesHbA1c
	case "CMS165":
		return models.MeasureCMS165BPControl
	case "CMS130":
		return models.MeasureCMS130ColorectalScreening
	case "CMS2":
		return models.MeasureCMS2DepressionScreening
	case "CMS134":
		return models.MeasureType("CMS134_DIABETES_KIDNEY")
	default:
		return models.MeasureType(cmsID)
	}
}

// mapPriority determines gap priority from CMS measure ID.
func (s *Service) mapPriority(cmsID string) models.GapPriority {
	switch cmsID {
	case "CMS122", "CMS165":
		return models.GapPriorityHigh
	case "CMS130", "CMS2", "CMS134":
		return models.GapPriorityMedium
	default:
		return models.GapPriorityLow
	}
}

// GetPatientCareGapsWithCQL returns care gaps using vaidshala CQL if enabled.
// This method delegates to EvaluateWithCQL if CQL engine is enabled,
// otherwise falls back to the legacy FHIR-based evaluation.
func (s *Service) GetPatientCareGapsWithCQL(
	ctx context.Context,
	patientID string,
	measures []models.MeasureType,
	period models.Period,
	includeClosedGaps bool,
	includeEvidence bool,
) (*models.CareGapReport, error) {
	if s.useCQLEngine {
		// Use vaidshala CQL engine
		return s.EvaluateWithCQL(ctx, patientID, period)
	}

	// Fall back to legacy FHIR-based evaluation
	return s.GetPatientCareGaps(ctx, patientID, measures, period, includeClosedGaps, includeEvidence)
}

// EvaluateMeasureWithCQL evaluates a single measure using the vaidshala CQL engine.
func (s *Service) EvaluateMeasureWithCQL(
	ctx context.Context,
	patientID string,
	measureID string,
	period models.Period,
) (*models.MeasureReport, error) {
	if !s.useCQLEngine {
		// Fall back to legacy method
		measureType := s.mapCMSIDToMeasureType(measureID)
		return s.EvaluateMeasure(ctx, patientID, measureType, period)
	}

	// Build execution context
	measurementPeriod := contracts.Period{
		Start: &period.Start,
		End:   &period.End,
	}

	execCtx, err := s.contextBuilder.BuildContext(ctx, patientID, measurementPeriod, s.config.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to build execution context: %w", err)
	}

	// Evaluate single measure
	result, err := s.cqlExecutor.EvaluateSingleMeasure(ctx, execCtx, measureID)
	if err != nil {
		return nil, fmt.Errorf("CQL measure evaluation failed: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("measure %s not found", measureID)
	}

	// Convert to MeasureReport
	measureType := s.mapCMSIDToMeasureType(measureID)
	measure, exists := s.measures[measureType]
	if !exists {
		measure = models.MeasureInfo{
			Type:  measureType,
			CMSID: measureID,
			Name:  result.MeasureName,
		}
	}

	report := &models.MeasureReport{
		ID:        uuid.New().String(),
		Measure:   measure,
		PatientID: patientID,
		Period:    period,
		Status:    "complete",
		Type:      "individual",
		Populations: []models.PopulationResult{
			{Population: models.PopulationInitial, Count: 1},
			{Population: models.PopulationDenominator, Count: boolToCount(result.InDenominator)},
			{Population: models.PopulationNumerator, Count: boolToCount(result.InNumerator)},
		},
		GeneratedAt: time.Now().UTC(),
	}

	return report, nil
}

// boolToCount converts a boolean to count (0 or 1).
func boolToCount(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ============================================================================
// KB-3 TEMPORAL INTEGRATION (TIER 7: LONGITUDINAL INTELLIGENCE)
// ============================================================================

// EvaluateWithTemporalContext evaluates care gaps and enriches them with temporal
// context from KB-3. This is the primary method for Tier 7 Longitudinal Intelligence.
//
// Flow:
//  1. KB-9 identifies WHAT care obligations exist (via CQL/Measures)
//  2. KB-3 provides WHEN obligations are due (via temporal engines)
//  3. Gaps are enriched with: due dates, overdue status, recurrence patterns
//  4. New gaps are optionally pushed to KB-3 for temporal tracking
//
// This method represents the collaboration between the "Accountability Engine" (KB-9)
// and the "Temporal Brain" (KB-3).
func (s *Service) EvaluateWithTemporalContext(
	ctx context.Context,
	patientID string,
	period models.Period,
	createScheduleItems bool,
) (*models.CareGapReport, error) {
	startTime := time.Now()

	s.logger.Info("Evaluating care gaps with KB-3 temporal enrichment",
		zap.String("patient_id", patientID),
		zap.Bool("create_schedule_items", createScheduleItems),
	)

	// Step 1: Evaluate care gaps using CQL engine
	report, err := s.EvaluateWithCQL(ctx, patientID, period)
	if err != nil {
		return nil, fmt.Errorf("CQL evaluation failed: %w", err)
	}

	// Step 2: Enrich gaps with KB-3 temporal context
	enrichedGaps, err := s.kb3Integration.EnrichGapsWithTemporalContext(
		ctx, report.OpenGaps, patientID,
	)
	if err != nil {
		s.logger.Warn("Failed to enrich gaps with KB-3 temporal context",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		// Continue with unenriched gaps - graceful degradation
	} else {
		report.OpenGaps = enrichedGaps
	}

	// Step 3: Optionally create KB-3 schedule items for new gaps
	if createScheduleItems && len(report.OpenGaps) > 0 {
		items, err := s.kb3Integration.CreateScheduleItemsForGaps(
			ctx, report.OpenGaps, patientID,
		)
		if err != nil {
			s.logger.Warn("Failed to create KB-3 schedule items",
				zap.String("patient_id", patientID),
				zap.Error(err),
			)
		} else if len(items) > 0 {
			s.logger.Info("Created KB-3 schedule items for care gaps",
				zap.String("patient_id", patientID),
				zap.Int("items_created", len(items)),
			)
		}
	}

	// Step 4: Identify upcoming due gaps (approaching deadline)
	report.UpcomingDue = s.filterUpcomingDueGaps(report.OpenGaps)

	// Rebuild summary with temporal-aware priority counts
	report.Summary = s.buildSummary(report.OpenGaps)

	s.logger.Info("Temporal-enriched care gap report generated",
		zap.String("patient_id", patientID),
		zap.Int("open_gaps", len(report.OpenGaps)),
		zap.Int("upcoming_due", len(report.UpcomingDue)),
		zap.Duration("total_duration", time.Since(startTime)),
	)

	return report, nil
}

// filterUpcomingDueGaps filters gaps that are approaching their due date.
func (s *Service) filterUpcomingDueGaps(gaps []models.CareGap) []models.CareGap {
	var upcoming []models.CareGap

	for _, gap := range gaps {
		if gap.TemporalContext != nil {
			// Include gaps that are approaching (within alert threshold)
			if gap.TemporalContext.Status == models.ConstraintApproaching {
				upcoming = append(upcoming, gap)
			}
			// Also include gaps due within 7 days
			if gap.TemporalContext.DaysUntilDue <= 7 && gap.TemporalContext.DaysUntilDue > 0 {
				// Avoid duplicates
				found := false
				for _, u := range upcoming {
					if u.ID == gap.ID {
						found = true
						break
					}
				}
				if !found {
					upcoming = append(upcoming, gap)
				}
			}
		}
	}

	return upcoming
}

// SyncClosedGapToKB3 notifies KB-3 when a care gap is closed.
// This completes the bidirectional sync between KB-9 and KB-3.
func (s *Service) SyncClosedGapToKB3(
	ctx context.Context,
	patientID string,
	closedGap *models.CareGap,
) error {
	return s.kb3Integration.SyncGapClosureWithKB3(ctx, closedGap, patientID)
}

// GetOverdueGapsFromKB3 retrieves gaps that KB-3 has flagged as overdue.
// This provides a temporal perspective on which gaps are most urgent.
func (s *Service) GetOverdueGapsFromKB3(
	ctx context.Context,
	patientID string,
) ([]kb3.OverdueAlert, error) {
	return s.kb3Integration.GetOverdueGapsFromKB3(ctx, patientID)
}

// KB3HealthCheck verifies KB-3 connectivity.
func (s *Service) KB3HealthCheck(ctx context.Context) error {
	return s.kb3Client.HealthCheck(ctx)
}
