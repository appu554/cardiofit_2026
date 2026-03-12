package cql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	// Import vaidshala contracts
	"vaidshala/clinical-runtime-platform/contracts"

	// Import KB-9 FHIR client
	"kb-9-care-gaps/internal/fhir"
)

// ContextBuilder builds vaidshala ClinicalExecutionContext from FHIR data.
// This is the bridge between Google FHIR data and the CQL engine.
type ContextBuilder struct {
	fhirClient *fhir.Client
	logger     *zap.Logger
}

// NewContextBuilder creates a new context builder.
func NewContextBuilder(fhirClient *fhir.Client, logger *zap.Logger) *ContextBuilder {
	return &ContextBuilder{
		fhirClient: fhirClient,
		logger:     logger,
	}
}

// BuildContext creates a ClinicalExecutionContext for a patient.
// This fetches FHIR data and transforms it into vaidshala's FROZEN contract format.
func (b *ContextBuilder) BuildContext(
	ctx context.Context,
	patientID string,
	measurementPeriod contracts.Period,
	region string,
) (*contracts.ClinicalExecutionContext, error) {
	startTime := time.Now()
	b.logger.Info("Building clinical execution context",
		zap.String("patient_id", patientID),
		zap.String("region", region),
	)

	// Build patient context from FHIR data
	patientCtx, err := b.buildPatientContext(ctx, patientID, measurementPeriod)
	if err != nil {
		return nil, err
	}

	// Build knowledge snapshot (terminology, calculations, safety)
	knowledge := b.buildKnowledgeSnapshot(patientCtx)

	// Build runtime metadata
	runtime := contracts.ExecutionMetadata{
		RequestID:         uuid.New().String(),
		RequestedBy:       "kb-9-care-gaps",
		RequestedAt:       time.Now().UTC(),
		Region:            region,
		MeasurementPeriod: &measurementPeriod,
		ExecutionMode:     "sync",
	}

	execCtx := &contracts.ClinicalExecutionContext{
		Patient:   *patientCtx,
		Knowledge: *knowledge,
		Runtime:   runtime,
	}

	b.logger.Info("Clinical execution context built",
		zap.String("patient_id", patientID),
		zap.Int("conditions", len(patientCtx.ActiveConditions)),
		zap.Int("medications", len(patientCtx.ActiveMedications)),
		zap.Int("labs", len(patientCtx.RecentLabResults)),
		zap.Int("vitals", len(patientCtx.RecentVitalSigns)),
		zap.Duration("duration", time.Since(startTime)),
	)

	return execCtx, nil
}

// buildPatientContext fetches and transforms FHIR resources into PatientContext.
func (b *ContextBuilder) buildPatientContext(
	ctx context.Context,
	patientID string,
	period contracts.Period,
) (*contracts.PatientContext, error) {
	// Convert to FHIR period format (handle pointer fields)
	var fhirPeriod *fhir.Period
	if period.Start != nil && period.End != nil {
		fhirPeriod = &fhir.Period{
			Start: period.Start.Format("2006-01-02"),
			End:   period.End.Format("2006-01-02"),
		}
	}

	patientCtx := &contracts.PatientContext{
		Demographics: contracts.PatientDemographics{
			PatientID: patientID,
		},
		ActiveConditions:  make([]contracts.ClinicalCondition, 0),
		ActiveMedications: make([]contracts.Medication, 0),
		RecentLabResults:  make([]contracts.LabResult, 0),
		RecentVitalSigns:  make([]contracts.VitalSign, 0),
		RecentEncounters:  make([]contracts.Encounter, 0),
		Allergies:         make([]contracts.Allergy, 0),
		RiskProfile:       contracts.RiskProfile{ComputedAt: time.Now().UTC()},
		ClinicalSummary:   contracts.ClinicalSummary{GeneratedAt: time.Now().UTC()},
	}

	// Fetch all patient data in one call (Google FHIR client pattern)
	patientData, err := b.fhirClient.GetPatientData(ctx, patientID, fhirPeriod)
	if err != nil {
		return nil, err
	}

	// Transform patient demographics
	if patientData.Patient != nil {
		patientCtx.Demographics.Gender = patientData.Patient.Gender
		if patientData.Patient.BirthDate != "" {
			if birthDate, err := time.Parse("2006-01-02", patientData.Patient.BirthDate); err == nil {
				patientCtx.Demographics.BirthDate = &birthDate
			}
		}
	}

	// Transform conditions
	patientCtx.ActiveConditions = b.transformConditions(patientData.Conditions)

	// Transform observations (labs and vitals)
	patientCtx.RecentLabResults, patientCtx.RecentVitalSigns = b.transformObservations(patientData.Observations)

	// Transform medications
	patientCtx.ActiveMedications = b.transformMedications(patientData.MedicationRequests)

	return patientCtx, nil
}

// transformConditions converts FHIR Conditions to vaidshala ClinicalConditions.
func (b *ContextBuilder) transformConditions(fhirConditions []fhir.Condition) []contracts.ClinicalCondition {
	conditions := make([]contracts.ClinicalCondition, 0, len(fhirConditions))

	for _, fc := range fhirConditions {
		if fc.Code == nil || len(fc.Code.Coding) == 0 {
			continue
		}

		coding := fc.Code.Coding[0]
		condition := contracts.ClinicalCondition{
			Code: contracts.ClinicalCode{
				System:  coding.System,
				Code:    coding.Code,
				Display: getDisplay(fc.Code, coding),
			},
			ClinicalStatus: "active",
		}

		// Parse onset date if available
		if fc.OnsetDateTime != "" {
			if onsetDate, err := time.Parse(time.RFC3339, fc.OnsetDateTime); err == nil {
				condition.OnsetDate = &onsetDate
			}
		}

		// Get clinical status
		if fc.ClinicalStatus != nil && len(fc.ClinicalStatus.Coding) > 0 {
			condition.ClinicalStatus = fc.ClinicalStatus.Coding[0].Code
		}

		conditions = append(conditions, condition)
	}

	return conditions
}

// transformObservations converts FHIR Observations to vaidshala LabResults and VitalSigns.
func (b *ContextBuilder) transformObservations(fhirObs []fhir.Observation) ([]contracts.LabResult, []contracts.VitalSign) {
	labs := make([]contracts.LabResult, 0)
	vitals := make([]contracts.VitalSign, 0)

	for _, fo := range fhirObs {
		if fo.Code == nil || len(fo.Code.Coding) == 0 {
			continue
		}

		coding := fo.Code.Coding[0]
		effectiveDate := time.Now()
		if fo.EffectiveDateTime != "" {
			if parsed, err := time.Parse(time.RFC3339, fo.EffectiveDateTime); err == nil {
				effectiveDate = parsed
			}
		}

		// Determine if this is a vital sign or lab
		isVital := b.isVitalSign(coding.Code)

		if isVital {
			vital := contracts.VitalSign{
				Code: contracts.ClinicalCode{
					System:  coding.System,
					Code:    coding.Code,
					Display: getDisplay(fo.Code, coding),
				},
				EffectiveDateTime: &effectiveDate,
			}

			// Handle composite vitals (blood pressure)
			if len(fo.Component) > 0 {
				vital.ComponentValues = b.transformVitalComponents(fo.Component)
			} else if fo.ValueQuantity != nil {
				vital.Value = &contracts.Quantity{
					Value: fo.ValueQuantity.Value,
					Unit:  fo.ValueQuantity.Unit,
				}
			}

			vitals = append(vitals, vital)
		} else {
			lab := contracts.LabResult{
				Code: contracts.ClinicalCode{
					System:  coding.System,
					Code:    coding.Code,
					Display: getDisplay(fo.Code, coding),
				},
				EffectiveDateTime: &effectiveDate,
			}

			if fo.ValueQuantity != nil {
				lab.Value = &contracts.Quantity{
					Value: fo.ValueQuantity.Value,
					Unit:  fo.ValueQuantity.Unit,
				}
			}

			if fo.Interpretation != nil && len(fo.Interpretation) > 0 {
				if len(fo.Interpretation[0].Coding) > 0 {
					lab.Interpretation = fo.Interpretation[0].Coding[0].Code
				}
			}

			labs = append(labs, lab)
		}
	}

	return labs, vitals
}

// transformVitalComponents converts FHIR observation components.
func (b *ContextBuilder) transformVitalComponents(components []fhir.ObservationComponent) []contracts.ComponentValue {
	result := make([]contracts.ComponentValue, 0, len(components))

	for _, comp := range components {
		if comp.Code == nil || len(comp.Code.Coding) == 0 {
			continue
		}

		coding := comp.Code.Coding[0]
		cv := contracts.ComponentValue{
			Code: contracts.ClinicalCode{
				System:  coding.System,
				Code:    coding.Code,
				Display: coding.Display,
			},
		}

		if comp.ValueQuantity != nil {
			cv.Value = &contracts.Quantity{
				Value: comp.ValueQuantity.Value,
				Unit:  comp.ValueQuantity.Unit,
			}
		}

		result = append(result, cv)
	}

	return result
}

// transformMedications converts FHIR MedicationRequests to vaidshala Medications.
func (b *ContextBuilder) transformMedications(fhirMeds []fhir.MedicationRequest) []contracts.Medication {
	meds := make([]contracts.Medication, 0, len(fhirMeds))

	for _, fm := range fhirMeds {
		med := contracts.Medication{
			Status: fm.Status,
		}

		if fm.MedicationCodeableConcept != nil && len(fm.MedicationCodeableConcept.Coding) > 0 {
			coding := fm.MedicationCodeableConcept.Coding[0]
			med.Code = contracts.ClinicalCode{
				System:  coding.System,
				Code:    coding.Code,
				Display: fm.MedicationCodeableConcept.Text,
			}
			if med.Code.Display == "" {
				med.Code.Display = coding.Display
			}
		}

		// Parse dates
		if fm.AuthoredOn != "" {
			if authDate, err := time.Parse(time.RFC3339, fm.AuthoredOn); err == nil {
				med.AuthoredOn = &authDate
			}
		}

		meds = append(meds, med)
	}

	return meds
}

// buildKnowledgeSnapshot creates a KnowledgeSnapshot from patient context.
// This populates terminology memberships and pre-computed calculations.
func (b *ContextBuilder) buildKnowledgeSnapshot(patientCtx *contracts.PatientContext) *contracts.KnowledgeSnapshot {
	snapshot := &contracts.KnowledgeSnapshot{
		Terminology: contracts.TerminologySnapshot{
			ValueSetMemberships: make(map[string]bool),
		},
		Calculators:       contracts.CalculatorSnapshot{},
		Safety:            contracts.SafetySnapshot{},
		Interactions:      contracts.InteractionSnapshot{},
		Formulary:         contracts.FormularySnapshot{},
		Dosing:            contracts.DosingSnapshot{},
		CDI:               contracts.CDIFacts{},
		SnapshotTimestamp: time.Now().UTC(),
		SnapshotVersion:   "1.0.0",
		KBVersions: map[string]string{
			"KB-7": "1.0.0",
			"KB-8": "1.0.0",
		},
	}

	// Check conditions for value set memberships
	for _, cond := range patientCtx.ActiveConditions {
		if isDiabetesCode(cond.Code.Code) {
			snapshot.Terminology.ValueSetMemberships["Diabetes"] = true
		}
		if isHypertensionCode(cond.Code.Code) {
			snapshot.Terminology.ValueSetMemberships["Essential Hypertension"] = true
		}
		if isCKDCode(cond.Code.Code) {
			snapshot.Terminology.ValueSetMemberships["Chronic Kidney Disease"] = true
		}
	}

	// Calculate eGFR if creatinine is available
	for _, lab := range patientCtx.RecentLabResults {
		if isCreatinineCode(lab.Code.Code) && lab.Value != nil && lab.Value.Value > 0 {
			age := 0
			gender := "male"
			if patientCtx.Demographics.BirthDate != nil {
				age = calculateAge(*patientCtx.Demographics.BirthDate)
			}
			if patientCtx.Demographics.Gender != "" {
				gender = patientCtx.Demographics.Gender
			}

			egfr := calculateEGFR(lab.Value.Value, age, gender)
			snapshot.Calculators.EGFR = &contracts.CalculationResult{
				Name:         "eGFR",
				Value:        egfr,
				Unit:         "mL/min/1.73m2",
				Category:     getEGFRStage(egfr),
				CalculatedAt: time.Now().UTC(),
			}
			break
		}
	}

	return snapshot
}

// isVitalSign determines if a code represents a vital sign.
func (b *ContextBuilder) isVitalSign(code string) bool {
	vitalCodes := map[string]bool{
		"85354-9": true, // BP panel
		"8480-6":  true, // Systolic
		"8462-4":  true, // Diastolic
		"8310-5":  true, // Temperature
		"8867-4":  true, // Heart rate
		"9279-1":  true, // Respiratory rate
		"2708-6":  true, // O2 saturation
		"29463-7": true, // Weight
		"8302-2":  true, // Height
		"39156-5": true, // BMI
	}
	return vitalCodes[code]
}

// Helper functions

func getDisplay(codeableConcept *fhir.CodeableConcept, coding fhir.Coding) string {
	if codeableConcept.Text != "" {
		return codeableConcept.Text
	}
	return coding.Display
}

func isDiabetesCode(code string) bool {
	diabetesCodes := map[string]bool{
		"73211009": true, // SNOMED Diabetes mellitus
		"44054006": true, // SNOMED Type 2
		"46635009": true, // SNOMED Type 1
		"E11":      true, // ICD-10 Type 2
		"E10":      true, // ICD-10 Type 1
	}
	return diabetesCodes[code]
}

func isHypertensionCode(code string) bool {
	htnCodes := map[string]bool{
		"59621000": true, // SNOMED Essential HTN
		"38341003": true, // SNOMED HTN
		"I10":      true, // ICD-10
	}
	return htnCodes[code]
}

func isCKDCode(code string) bool {
	ckdCodes := map[string]bool{
		"709044004": true, // SNOMED CKD
		"N18":       true, // ICD-10
	}
	return ckdCodes[code]
}

func isCreatinineCode(code string) bool {
	return code == "2160-0" || code == "38483-4"
}

func calculateAge(birthDate time.Time) int {
	now := time.Now()
	years := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		years--
	}
	return years
}

func calculateEGFR(creatinine float64, age int, gender string) float64 {
	if creatinine <= 0 || age <= 0 {
		return 0
	}

	// CKD-EPI 2021 (simplified)
	scr := creatinine
	if scr > 20 { // Likely in μmol/L
		scr = creatinine / 88.4
	}

	var kappa, alpha, sexCoeff float64
	if gender == "female" {
		kappa = 0.7
		alpha = -0.241
		sexCoeff = 1.012
	} else {
		kappa = 0.9
		alpha = -0.302
		sexCoeff = 1.0
	}

	ratio := scr / kappa
	var term float64
	if ratio <= 1 {
		term = ratio
	} else {
		term = 1 / ratio
	}

	egfr := 142.0 * pow(term, alpha) * pow(0.9938, float64(age)) * sexCoeff
	return egfr
}

func getEGFRStage(egfr float64) string {
	switch {
	case egfr >= 90:
		return "G1"
	case egfr >= 60:
		return "G2"
	case egfr >= 45:
		return "G3a"
	case egfr >= 30:
		return "G3b"
	case egfr >= 15:
		return "G4"
	default:
		return "G5"
	}
}

func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := 1.0
	absExp := exp
	if exp < 0 {
		absExp = -exp
	}
	for i := 0; i < int(absExp*10); i++ {
		result *= base
	}
	if exp < 0 {
		return 1 / result
	}
	return result
}
