// Package fhir provides FHIR query builders for care gap detection.
package fhir

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ============================================================================
// Care Gap Specific Queries
// ============================================================================

// QueryBuilder provides methods for building care gap-specific FHIR queries.
type QueryBuilder struct {
	client *Client
	logger *zap.Logger
}

// NewQueryBuilder creates a new FHIR query builder.
func NewQueryBuilder(client *Client, logger *zap.Logger) *QueryBuilder {
	return &QueryBuilder{
		client: client,
		logger: logger,
	}
}

// ============================================================================
// Diabetes Measure Queries (CMS122)
// ============================================================================

// DiabetesData contains data needed for diabetes quality measures.
type DiabetesData struct {
	HasDiabetes       bool
	MostRecentHbA1c   *Observation
	HbA1cValue        float64
	HbA1cDate         time.Time
	HbA1cInPeriod     bool
	HbA1cPoorControl  bool // > 9.0%
}

// GetDiabetesData retrieves data for CMS122 diabetes measure evaluation.
func (qb *QueryBuilder) GetDiabetesData(ctx context.Context, patientID string, period *Period) (*DiabetesData, error) {
	data := &DiabetesData{}

	// Check for diabetes diagnosis
	condParams := map[string]string{
		"patient":         patientID,
		"code":            fmt.Sprintf("%s|%s,%s|%s", SystemSNOMED, SNOMEDDiabetesType2, SystemSNOMED, SNOMEDDiabetesType1),
		"clinical-status": "active",
	}
	condBundle, err := qb.client.SearchResources(ctx, "Condition", condParams)
	if err != nil {
		return nil, fmt.Errorf("failed to search diabetes conditions: %w", err)
	}
	data.HasDiabetes = condBundle != nil && len(condBundle.Entry) > 0

	// Get HbA1c observations
	obsParams := map[string]string{
		"patient": patientID,
		"code":    fmt.Sprintf("%s|%s", SystemLOINC, LOINCHbA1c),
		"_sort":   "-date",
		"_count":  "1",
	}
	if period != nil && period.Start != "" {
		obsParams["date"] = fmt.Sprintf("ge%s", period.Start)
	}

	obsBundle, err := qb.client.SearchResources(ctx, "Observation", obsParams)
	if err != nil {
		return nil, fmt.Errorf("failed to search HbA1c: %w", err)
	}

	observations := parseObservations(obsBundle)
	if len(observations) > 0 {
		data.MostRecentHbA1c = &observations[0]

		// Extract value
		if observations[0].ValueQuantity != nil {
			data.HbA1cValue = observations[0].ValueQuantity.Value
			data.HbA1cPoorControl = data.HbA1cValue > 9.0
		}

		// Extract date
		if observations[0].EffectiveDateTime != "" {
			if t, err := time.Parse(time.RFC3339, observations[0].EffectiveDateTime); err == nil {
				data.HbA1cDate = t
				data.HbA1cInPeriod = isInPeriod(t, period)
			}
		}
	}

	qb.logger.Info("Diabetes data retrieved",
		zap.String("patientID", patientID),
		zap.Bool("hasDiabetes", data.HasDiabetes),
		zap.Float64("hba1c", data.HbA1cValue),
		zap.Bool("poorControl", data.HbA1cPoorControl),
	)

	return data, nil
}

// ============================================================================
// Blood Pressure Measure Queries (CMS165)
// ============================================================================

// BloodPressureData contains data needed for blood pressure quality measures.
type BloodPressureData struct {
	HasHypertension     bool
	MostRecentSystolic  *Observation
	MostRecentDiastolic *Observation
	SystolicValue       float64
	DiastolicValue      float64
	BPDate              time.Time
	BPInPeriod          bool
	BPControlled        bool // < 140/90
}

// GetBloodPressureData retrieves data for CMS165 blood pressure measure.
func (qb *QueryBuilder) GetBloodPressureData(ctx context.Context, patientID string, period *Period) (*BloodPressureData, error) {
	data := &BloodPressureData{}

	// Check for hypertension diagnosis
	condParams := map[string]string{
		"patient":         patientID,
		"code":            fmt.Sprintf("%s|%s", SystemSNOMED, SNOMEDHypertension),
		"clinical-status": "active",
	}
	condBundle, err := qb.client.SearchResources(ctx, "Condition", condParams)
	if err != nil {
		return nil, fmt.Errorf("failed to search hypertension: %w", err)
	}
	data.HasHypertension = condBundle != nil && len(condBundle.Entry) > 0

	// Get systolic BP
	sysParams := map[string]string{
		"patient": patientID,
		"code":    fmt.Sprintf("%s|%s", SystemLOINC, LOINCSystolicBP),
		"_sort":   "-date",
		"_count":  "1",
	}
	if period != nil && period.Start != "" {
		sysParams["date"] = fmt.Sprintf("ge%s", period.Start)
	}

	sysBundle, err := qb.client.SearchResources(ctx, "Observation", sysParams)
	if err != nil {
		qb.logger.Warn("Failed to search systolic BP", zap.Error(err))
	} else {
		observations := parseObservations(sysBundle)
		if len(observations) > 0 {
			data.MostRecentSystolic = &observations[0]
			if observations[0].ValueQuantity != nil {
				data.SystolicValue = observations[0].ValueQuantity.Value
			}
			if observations[0].EffectiveDateTime != "" {
				if t, err := time.Parse(time.RFC3339, observations[0].EffectiveDateTime); err == nil {
					data.BPDate = t
					data.BPInPeriod = isInPeriod(t, period)
				}
			}
		}
	}

	// Get diastolic BP
	diaParams := map[string]string{
		"patient": patientID,
		"code":    fmt.Sprintf("%s|%s", SystemLOINC, LOINCDiastolicBP),
		"_sort":   "-date",
		"_count":  "1",
	}
	if period != nil && period.Start != "" {
		diaParams["date"] = fmt.Sprintf("ge%s", period.Start)
	}

	diaBundle, err := qb.client.SearchResources(ctx, "Observation", diaParams)
	if err != nil {
		qb.logger.Warn("Failed to search diastolic BP", zap.Error(err))
	} else {
		observations := parseObservations(diaBundle)
		if len(observations) > 0 {
			data.MostRecentDiastolic = &observations[0]
			if observations[0].ValueQuantity != nil {
				data.DiastolicValue = observations[0].ValueQuantity.Value
			}
		}
	}

	// Check if BP is controlled
	data.BPControlled = data.SystolicValue > 0 && data.DiastolicValue > 0 &&
		data.SystolicValue < 140 && data.DiastolicValue < 90

	qb.logger.Info("Blood pressure data retrieved",
		zap.String("patientID", patientID),
		zap.Bool("hasHypertension", data.HasHypertension),
		zap.Float64("systolic", data.SystolicValue),
		zap.Float64("diastolic", data.DiastolicValue),
		zap.Bool("controlled", data.BPControlled),
	)

	return data, nil
}

// ============================================================================
// Depression Screening Queries (CMS2)
// ============================================================================

// DepressionScreeningData contains data for depression screening measure.
type DepressionScreeningData struct {
	HasScreening    bool
	PHQ2Score       *float64
	PHQ9Score       *float64
	ScreeningDate   time.Time
	ScreeningInPeriod bool
	PositiveScreen  bool
	HasFollowUp     bool
}

// GetDepressionScreeningData retrieves data for CMS2 depression screening.
func (qb *QueryBuilder) GetDepressionScreeningData(ctx context.Context, patientID string, period *Period) (*DepressionScreeningData, error) {
	data := &DepressionScreeningData{}

	// Get PHQ-2 screening
	phq2Params := map[string]string{
		"patient": patientID,
		"code":    fmt.Sprintf("%s|%s", SystemLOINC, LOINCPHQ2),
		"_sort":   "-date",
		"_count":  "1",
	}
	if period != nil && period.Start != "" {
		phq2Params["date"] = fmt.Sprintf("ge%s", period.Start)
	}

	phq2Bundle, err := qb.client.SearchResources(ctx, "Observation", phq2Params)
	if err != nil {
		qb.logger.Warn("Failed to search PHQ-2", zap.Error(err))
	} else {
		observations := parseObservations(phq2Bundle)
		if len(observations) > 0 {
			data.HasScreening = true
			if observations[0].ValueQuantity != nil {
				val := observations[0].ValueQuantity.Value
				data.PHQ2Score = &val
				data.PositiveScreen = val >= 3 // PHQ-2 positive if >= 3
			}
			if observations[0].EffectiveDateTime != "" {
				if t, err := time.Parse(time.RFC3339, observations[0].EffectiveDateTime); err == nil {
					data.ScreeningDate = t
					data.ScreeningInPeriod = isInPeriod(t, period)
				}
			}
		}
	}

	// Get PHQ-9 (follow-up for positive PHQ-2)
	phq9Params := map[string]string{
		"patient": patientID,
		"code":    fmt.Sprintf("%s|%s", SystemLOINC, LOINCPHQ9),
		"_sort":   "-date",
		"_count":  "1",
	}
	if period != nil && period.Start != "" {
		phq9Params["date"] = fmt.Sprintf("ge%s", period.Start)
	}

	phq9Bundle, err := qb.client.SearchResources(ctx, "Observation", phq9Params)
	if err != nil {
		qb.logger.Warn("Failed to search PHQ-9", zap.Error(err))
	} else {
		observations := parseObservations(phq9Bundle)
		if len(observations) > 0 {
			if observations[0].ValueQuantity != nil {
				val := observations[0].ValueQuantity.Value
				data.PHQ9Score = &val
			}
			data.HasFollowUp = true
		}
	}

	qb.logger.Info("Depression screening data retrieved",
		zap.String("patientID", patientID),
		zap.Bool("hasScreening", data.HasScreening),
		zap.Bool("positiveScreen", data.PositiveScreen),
		zap.Bool("hasFollowUp", data.HasFollowUp),
	)

	return data, nil
}

// ============================================================================
// Colorectal Cancer Screening Queries (CMS130)
// ============================================================================

// ColorectalScreeningData contains data for colorectal cancer screening.
type ColorectalScreeningData struct {
	Age                int
	HasColonoscopy     bool
	ColonoscopyDate    time.Time
	HasFOBT            bool
	FOBTDate           time.Time
	ScreeningUpToDate  bool
}

// GetColorectalScreeningData retrieves data for CMS130 colorectal screening.
func (qb *QueryBuilder) GetColorectalScreeningData(ctx context.Context, patientID string, period *Period) (*ColorectalScreeningData, error) {
	data := &ColorectalScreeningData{}

	// Get patient for age calculation
	patientRaw, err := qb.client.GetResource(ctx, "Patient", patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient: %w", err)
	}
	if patientRaw != nil {
		patient := parsePatient(patientRaw)
		if patient != nil && patient.BirthDate != "" {
			if birthDate, err := time.Parse("2006-01-02", patient.BirthDate); err == nil {
				data.Age = calculateAge(birthDate)
			}
		}
	}

	// Search for colonoscopy (last 10 years)
	tenYearsAgo := time.Now().AddDate(-10, 0, 0).Format("2006-01-02")
	colonoscopyParams := map[string]string{
		"patient": patientID,
		"code":    fmt.Sprintf("%s|73761001", SystemSNOMED), // SNOMED code for colonoscopy
		"date":    fmt.Sprintf("ge%s", tenYearsAgo),
		"_sort":   "-date",
		"_count":  "1",
	}

	colonBundle, err := qb.client.SearchResources(ctx, "Procedure", colonoscopyParams)
	if err != nil {
		qb.logger.Warn("Failed to search colonoscopy", zap.Error(err))
	} else {
		procedures := parseProcedures(colonBundle)
		if len(procedures) > 0 {
			data.HasColonoscopy = true
			if procedures[0].PerformedDateTime != "" {
				if t, err := time.Parse(time.RFC3339, procedures[0].PerformedDateTime); err == nil {
					data.ColonoscopyDate = t
				}
			}
		}
	}

	// Search for FOBT/FIT (last year)
	oneYearAgo := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	fobtParams := map[string]string{
		"patient": patientID,
		"code":    fmt.Sprintf("%s|%s", SystemLOINC, LOINCFOBT),
		"date":    fmt.Sprintf("ge%s", oneYearAgo),
		"_sort":   "-date",
		"_count":  "1",
	}

	fobtBundle, err := qb.client.SearchResources(ctx, "Observation", fobtParams)
	if err != nil {
		qb.logger.Warn("Failed to search FOBT", zap.Error(err))
	} else {
		observations := parseObservations(fobtBundle)
		if len(observations) > 0 {
			data.HasFOBT = true
			if observations[0].EffectiveDateTime != "" {
				if t, err := time.Parse(time.RFC3339, observations[0].EffectiveDateTime); err == nil {
					data.FOBTDate = t
				}
			}
		}
	}

	// Screening is up to date if colonoscopy within 10 years OR FOBT within 1 year
	data.ScreeningUpToDate = data.HasColonoscopy || data.HasFOBT

	qb.logger.Info("Colorectal screening data retrieved",
		zap.String("patientID", patientID),
		zap.Int("age", data.Age),
		zap.Bool("hasColonoscopy", data.HasColonoscopy),
		zap.Bool("hasFOBT", data.HasFOBT),
		zap.Bool("upToDate", data.ScreeningUpToDate),
	)

	return data, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// isInPeriod checks if a time falls within a measurement period.
func isInPeriod(t time.Time, period *Period) bool {
	if period == nil {
		return true
	}

	if period.Start != "" {
		start, err := time.Parse("2006-01-02", period.Start)
		if err == nil && t.Before(start) {
			return false
		}
	}

	if period.End != "" {
		end, err := time.Parse("2006-01-02", period.End)
		if err == nil && t.After(end) {
			return false
		}
	}

	return true
}

// calculateAge calculates age from birth date.
func calculateAge(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		age--
	}
	return age
}
