# Ingestion Service — Ingestion Core Plan (Phase 2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the complete ingestion pipeline — adapters (patient self-report, device), pipeline stages (normalizer, validator, mapper, router), Kafka producer, DLQ publisher, and FHIR Store writes — so that health observations can flow end-to-end from submission to FHIR Store persistence and Kafka topic routing.

**Architecture:** All code lives in `vaidshala/clinical-runtime-platform/services/ingestion-service/`. Adapters convert source-specific payloads into `CanonicalObservation`. Pipeline stages process observations sequentially: Normalize → Validate → Map → Route. The orchestrator wires everything together and replaces the Phase 1 stub handlers with real implementations.

**Tech Stack:** Go 1.25, Gin, pgx/v5, redis/go-redis/v9, zap, prometheus/client_golang, segmentio/kafka-go, stretchr/testify, net/http/httptest

**Spec:** `docs/superpowers/specs/2026-03-21-ingestion-intake-onboarding-design.md`

**Prerequisite:** Phase 1 (Foundation) must be implemented first — `go.mod`, config, health endpoints, `CanonicalObservation`, pipeline interfaces, Kafka envelope, `dlq_messages` migration, and `pkg/fhirclient` already exist.

---

## File Structure

All paths relative to `vaidshala/clinical-runtime-platform/services/ingestion-service/`.

| File | Responsibility |
|------|---------------|
| `internal/coding/unit_converter.go` | Unit conversion: mmol/L→mg/dL, kPa→mmHg, degF→degC |
| `internal/coding/unit_converter_test.go` | Unit conversion tests |
| `internal/coding/loinc_mapper.go` | LOINC code lookup table (common lab codes) |
| `internal/coding/snomed_mapper.go` | SNOMED CT code lookup table (observation categories) |
| `internal/coding/loinc_mapper_test.go` | LOINC mapping tests |
| `internal/pipeline/normalizer.go` | Normalizer stage: unit conversion + code mapping |
| `internal/pipeline/normalizer_test.go` | Normalizer tests |
| `internal/pipeline/validator.go` | Validator stage: clinical range checks + quality scoring |
| `internal/pipeline/validator_test.go` | Validator tests |
| `internal/fhir/observation_mapper.go` | CanonicalObservation → FHIR R4 Observation JSON (ABDM IG v7.0) |
| `internal/fhir/diagnostic_report_mapper.go` | CanonicalObservation → FHIR DiagnosticReport JSON |
| `internal/fhir/medication_mapper.go` | CanonicalObservation → FHIR MedicationStatement JSON |
| `internal/fhir/mapper.go` | Composite mapper implementing pipeline.Mapper interface |
| `internal/fhir/mapper_test.go` | FHIR mapper tests |
| `internal/kafka/producer.go` | Kafka producer with segmentio/kafka-go |
| `internal/kafka/router.go` | Topic selection by ObservationType + urgency partitioning |
| `internal/kafka/router_test.go` | Router tests |
| `internal/dlq/publisher.go` | DLQ publisher: PostgreSQL insert + Kafka DLQ topic |
| `internal/dlq/replay.go` | DLQ replay endpoint handler |
| `internal/dlq/publisher_test.go` | DLQ publisher tests |
| `internal/adapters/patient_reported/app_checkin.go` | Flutter app structured JSON → CanonicalObservation |
| `internal/adapters/patient_reported/whatsapp.go` | WhatsApp NLU parsed intent → CanonicalObservation |
| `internal/adapters/patient_reported/app_checkin_test.go` | App checkin adapter tests |
| `internal/adapters/devices/device_adapter.go` | BLE device reading → CanonicalObservation |
| `internal/adapters/devices/device_adapter_test.go` | Device adapter tests |
| `internal/pipeline/orchestrator.go` | Wires Receiver→Parser→Normalizer→Validator→Mapper→Router |
| `internal/pipeline/orchestrator_test.go` | Orchestrator tests |
| `internal/metrics/collectors.go` | 10 Prometheus metrics from spec section 7.3 |
| `internal/api/routes.go` | **Modify**: replace stub handlers with real implementations |
| `internal/api/server.go` | **Modify**: add pipeline + Kafka producer to Server struct |
| `internal/api/handlers.go` | Real FHIR + ingest endpoint handlers |

---

## Task 1: Unit Converter + Code Mappers (`internal/coding/`)

**Files:**
- Create: `internal/coding/unit_converter.go`
- Create: `internal/coding/unit_converter_test.go`
- Create: `internal/coding/loinc_mapper.go`
- Create: `internal/coding/snomed_mapper.go`
- Create: `internal/coding/loinc_mapper_test.go`

- [ ] **Step 1: Write unit_converter_test.go (TDD — test first)**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/coding/unit_converter_test.go
package coding

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertGlucose_MmolToMgdL(t *testing.T) {
	result, err := ConvertUnit("mmol/L", "mg/dL", 5.5, "glucose")
	require.NoError(t, err)
	assert.InDelta(t, 99.0, result, 0.5) // 5.5 * 18.0 = 99.0
}

func TestConvertGlucose_MgdLToMmol(t *testing.T) {
	result, err := ConvertUnit("mg/dL", "mmol/L", 126.0, "glucose")
	require.NoError(t, err)
	assert.InDelta(t, 7.0, result, 0.01) // 126 / 18.0 = 7.0
}

func TestConvertBP_KPaToMmHg(t *testing.T) {
	result, err := ConvertUnit("kPa", "mmHg", 16.0, "blood_pressure")
	require.NoError(t, err)
	assert.InDelta(t, 120.0, result, 0.5) // 16.0 * 7.50062 ≈ 120.01
}

func TestConvertTemp_FahrenheitToCelsius(t *testing.T) {
	result, err := ConvertUnit("degF", "degC", 98.6, "temperature")
	require.NoError(t, err)
	assert.InDelta(t, 37.0, result, 0.01) // (98.6 - 32) * 5/9 = 37.0
}

func TestConvertTemp_CelsiusToFahrenheit(t *testing.T) {
	result, err := ConvertUnit("degC", "degF", 37.0, "temperature")
	require.NoError(t, err)
	assert.InDelta(t, 98.6, result, 0.01)
}

func TestConvertCholesterol_MmolToMgdL(t *testing.T) {
	result, err := ConvertUnit("mmol/L", "mg/dL", 5.0, "cholesterol")
	require.NoError(t, err)
	assert.InDelta(t, 193.3, result, 0.5) // 5.0 * 38.67 = 193.35
}

func TestConvertTriglycerides_MmolToMgdL(t *testing.T) {
	result, err := ConvertUnit("mmol/L", "mg/dL", 1.7, "triglycerides")
	require.NoError(t, err)
	assert.InDelta(t, 150.5, result, 0.5) // 1.7 * 88.57 = 150.57
}

func TestConvertCreatinine_UmolToMgdL(t *testing.T) {
	result, err := ConvertUnit("umol/L", "mg/dL", 88.4, "creatinine")
	require.NoError(t, err)
	assert.InDelta(t, 1.0, result, 0.01) // 88.4 / 88.4 = 1.0
}

func TestConvertHbA1c_MmolMolToPercent(t *testing.T) {
	result, err := ConvertUnit("mmol/mol", "%", 48.0, "hba1c")
	require.NoError(t, err)
	assert.InDelta(t, 6.5, result, 0.1) // (48 / 10.929) + 2.15 ≈ 6.54
}

func TestConvertUnit_SameUnit(t *testing.T) {
	result, err := ConvertUnit("mg/dL", "mg/dL", 126.0, "glucose")
	require.NoError(t, err)
	assert.Equal(t, 126.0, result) // No conversion needed
}

func TestConvertUnit_UnsupportedConversion(t *testing.T) {
	_, err := ConvertUnit("furlongs", "mg/dL", 1.0, "glucose")
	assert.Error(t, err)
}

func TestConvertWeight_KgToLbs(t *testing.T) {
	result, err := ConvertUnit("kg", "lbs", 70.0, "weight")
	require.NoError(t, err)
	assert.InDelta(t, 154.32, result, 0.1) // 70 * 2.20462
}

func TestConvertWeight_LbsToKg(t *testing.T) {
	result, err := ConvertUnit("lbs", "kg", 154.32, "weight")
	require.NoError(t, err)
	assert.InDelta(t, 70.0, result, 0.1)
}

func TestNormalizeToStandardUnit(t *testing.T) {
	tests := []struct {
		name     string
		fromUnit string
		value    float64
		analyte  string
		wantVal  float64
		wantUnit string
	}{
		{"glucose mmol→mg/dL", "mmol/L", 7.0, "glucose", 126.0, "mg/dL"},
		{"glucose mg/dL stays", "mg/dL", 126.0, "glucose", 126.0, "mg/dL"},
		{"BP kPa→mmHg", "kPa", 16.0, "blood_pressure", 120.0, "mmHg"},
		{"temp degF→degC", "degF", 98.6, "temperature", 37.0, "degC"},
		{"cholesterol mmol→mg/dL", "mmol/L", 5.0, "cholesterol", 193.3, "mg/dL"},
		{"creatinine umol→mg/dL", "umol/L", 88.4, "creatinine", 1.0, "mg/dL"},
		{"hba1c mmol/mol→%", "mmol/mol", 48.0, "hba1c", 6.5, "%"},
		{"weight lbs→kg", "lbs", 154.32, "weight", 70.0, "kg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, unit, err := NormalizeToStandardUnit(tt.fromUnit, tt.value, tt.analyte)
			require.NoError(t, err)
			assert.Equal(t, tt.wantUnit, unit)
			assert.InDelta(t, tt.wantVal, val, 0.5)
			_ = math.Abs(0) // suppress unused import
		})
	}
}
```

- [ ] **Step 2: Verify test fails (no implementation yet)**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/coding/... -v -count=1 2>&1 | head -5`
Expected: Compilation error — `ConvertUnit` and `NormalizeToStandardUnit` undefined.

- [ ] **Step 3: Write unit_converter.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/coding/unit_converter.go
package coding

import "fmt"

// conversionKey uniquely identifies a unit conversion path.
type conversionKey struct {
	FromUnit string
	ToUnit   string
	Analyte  string
}

// conversionFunc converts a value from one unit to another.
type conversionFunc func(float64) float64

// standardUnits maps analyte names to their canonical (standard) unit.
var standardUnits = map[string]string{
	"glucose":        "mg/dL",
	"blood_pressure": "mmHg",
	"temperature":    "degC",
	"cholesterol":    "mg/dL",
	"hdl":            "mg/dL",
	"ldl":            "mg/dL",
	"triglycerides":  "mg/dL",
	"creatinine":     "mg/dL",
	"hba1c":          "%",
	"weight":         "kg",
	"height":         "cm",
	"potassium":      "mEq/L",
	"sodium":         "mEq/L",
	"egfr":           "mL/min/1.73m2",
	"heart_rate":     "bpm",
	"spo2":           "%",
}

// conversions holds all known unit conversion functions.
var conversions = map[conversionKey]conversionFunc{
	// Glucose: mmol/L ⇄ mg/dL (factor 18.0)
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "glucose"}:  func(v float64) float64 { return v * 18.0 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "glucose"}:  func(v float64) float64 { return v / 18.0 },

	// Blood pressure: kPa ⇄ mmHg (factor 7.50062)
	{FromUnit: "kPa", ToUnit: "mmHg", Analyte: "blood_pressure"}: func(v float64) float64 { return v * 7.50062 },
	{FromUnit: "mmHg", ToUnit: "kPa", Analyte: "blood_pressure"}: func(v float64) float64 { return v / 7.50062 },

	// Temperature: degF ⇄ degC
	{FromUnit: "degF", ToUnit: "degC", Analyte: "temperature"}: func(v float64) float64 { return (v - 32.0) * 5.0 / 9.0 },
	{FromUnit: "degC", ToUnit: "degF", Analyte: "temperature"}: func(v float64) float64 { return v*9.0/5.0 + 32.0 },

	// Cholesterol: mmol/L ⇄ mg/dL (factor 38.67)
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "cholesterol"}: func(v float64) float64 { return v * 38.67 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "cholesterol"}: func(v float64) float64 { return v / 38.67 },

	// HDL: same factor as total cholesterol
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "hdl"}: func(v float64) float64 { return v * 38.67 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "hdl"}: func(v float64) float64 { return v / 38.67 },

	// LDL: same factor as total cholesterol
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "ldl"}: func(v float64) float64 { return v * 38.67 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "ldl"}: func(v float64) float64 { return v / 38.67 },

	// Triglycerides: mmol/L ⇄ mg/dL (factor 88.57)
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "triglycerides"}: func(v float64) float64 { return v * 88.57 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "triglycerides"}: func(v float64) float64 { return v / 88.57 },

	// Creatinine: umol/L ⇄ mg/dL (factor 88.4)
	{FromUnit: "umol/L", ToUnit: "mg/dL", Analyte: "creatinine"}: func(v float64) float64 { return v / 88.4 },
	{FromUnit: "mg/dL", ToUnit: "umol/L", Analyte: "creatinine"}: func(v float64) float64 { return v * 88.4 },

	// HbA1c: mmol/mol ⇄ % (NGSP/DCCT) using IFCC formula
	{FromUnit: "mmol/mol", ToUnit: "%", Analyte: "hba1c"}: func(v float64) float64 { return (v / 10.929) + 2.15 },
	{FromUnit: "%", ToUnit: "mmol/mol", Analyte: "hba1c"}: func(v float64) float64 { return (v - 2.15) * 10.929 },

	// Weight: lbs ⇄ kg
	{FromUnit: "lbs", ToUnit: "kg", Analyte: "weight"}: func(v float64) float64 { return v / 2.20462 },
	{FromUnit: "kg", ToUnit: "lbs", Analyte: "weight"}: func(v float64) float64 { return v * 2.20462 },

	// Height: in ⇄ cm
	{FromUnit: "in", ToUnit: "cm", Analyte: "height"}: func(v float64) float64 { return v * 2.54 },
	{FromUnit: "cm", ToUnit: "in", Analyte: "height"}: func(v float64) float64 { return v / 2.54 },
}

// ConvertUnit converts a value between two units for a given analyte.
// Returns the original value unchanged if fromUnit == toUnit.
func ConvertUnit(fromUnit, toUnit string, value float64, analyte string) (float64, error) {
	if fromUnit == toUnit {
		return value, nil
	}

	key := conversionKey{FromUnit: fromUnit, ToUnit: toUnit, Analyte: analyte}
	fn, ok := conversions[key]
	if !ok {
		return 0, fmt.Errorf("unsupported conversion: %s → %s for analyte %q", fromUnit, toUnit, analyte)
	}
	return fn(value), nil
}

// NormalizeToStandardUnit converts a value to the canonical standard unit
// for the given analyte. Returns the converted value, the standard unit
// name, and any error. If the value is already in the standard unit, it
// is returned unchanged.
func NormalizeToStandardUnit(fromUnit string, value float64, analyte string) (float64, string, error) {
	stdUnit, ok := standardUnits[analyte]
	if !ok {
		// Unknown analyte — pass through without conversion
		return value, fromUnit, nil
	}

	if fromUnit == stdUnit {
		return value, stdUnit, nil
	}

	converted, err := ConvertUnit(fromUnit, stdUnit, value, analyte)
	if err != nil {
		return 0, "", err
	}
	return converted, stdUnit, nil
}
```

- [ ] **Step 4: Run unit converter tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/coding/... -v -count=1`
Expected: All 14 tests PASS

- [ ] **Step 5: Write loinc_mapper.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/coding/loinc_mapper.go
package coding

// LOINCEntry represents a LOINC code mapping.
type LOINCEntry struct {
	Code     string
	Display  string
	Analyte  string // Internal analyte key for unit conversion
	StdUnit  string // Standard unit for this LOINC code
	Category string // Kafka routing category
}

// loincRegistry maps LOINC codes to their metadata.
// This covers the most common cardiometabolic observations.
// Production: extend with PostgreSQL-backed lab_code_mappings table.
var loincRegistry = map[string]LOINCEntry{
	// Glucose
	"1558-6":  {Code: "1558-6", Display: "Fasting glucose [Mass/volume] in Serum or Plasma", Analyte: "glucose", StdUnit: "mg/dL", Category: "LABS"},
	"2345-7":  {Code: "2345-7", Display: "Glucose [Mass/volume] in Serum or Plasma", Analyte: "glucose", StdUnit: "mg/dL", Category: "LABS"},
	"2339-0":  {Code: "2339-0", Display: "Glucose [Mass/volume] in Blood", Analyte: "glucose", StdUnit: "mg/dL", Category: "LABS"},
	"14749-6": {Code: "14749-6", Display: "Glucose [Moles/volume] in Serum or Plasma", Analyte: "glucose", StdUnit: "mg/dL", Category: "LABS"},
	"4548-4":  {Code: "4548-4", Display: "Hemoglobin A1c/Hemoglobin.total in Blood", Analyte: "hba1c", StdUnit: "%", Category: "LABS"},

	// Renal
	"33914-3": {Code: "33914-3", Display: "Glomerular filtration rate/1.73 sq M.predicted [Volume Rate/Area]", Analyte: "egfr", StdUnit: "mL/min/1.73m2", Category: "LABS"},
	"2160-0":  {Code: "2160-0", Display: "Creatinine [Mass/volume] in Serum or Plasma", Analyte: "creatinine", StdUnit: "mg/dL", Category: "LABS"},
	"6299-2":  {Code: "6299-2", Display: "Urea nitrogen [Mass/volume] in Blood", Analyte: "bun", StdUnit: "mg/dL", Category: "LABS"},
	"2823-3":  {Code: "2823-3", Display: "Potassium [Moles/volume] in Serum or Plasma", Analyte: "potassium", StdUnit: "mEq/L", Category: "LABS"},
	"2951-2":  {Code: "2951-2", Display: "Sodium [Moles/volume] in Serum or Plasma", Analyte: "sodium", StdUnit: "mEq/L", Category: "LABS"},
	"5811-5":  {Code: "5811-5", Display: "Specific gravity of Urine by Test strip", Analyte: "urine_sg", StdUnit: "", Category: "LABS"},
	"14959-1": {Code: "14959-1", Display: "Microalbumin [Mass/volume] in Urine", Analyte: "urine_albumin", StdUnit: "mg/L", Category: "LABS"},

	// Lipids
	"2093-3": {Code: "2093-3", Display: "Cholesterol [Mass/volume] in Serum or Plasma", Analyte: "cholesterol", StdUnit: "mg/dL", Category: "LABS"},
	"2085-9": {Code: "2085-9", Display: "Cholesterol in HDL [Mass/volume] in Serum or Plasma", Analyte: "hdl", StdUnit: "mg/dL", Category: "LABS"},
	"2089-1": {Code: "2089-1", Display: "Cholesterol in LDL [Mass/volume] in Serum or Plasma", Analyte: "ldl", StdUnit: "mg/dL", Category: "LABS"},
	"2571-8": {Code: "2571-8", Display: "Triglyceride [Mass/volume] in Serum or Plasma", Analyte: "triglycerides", StdUnit: "mg/dL", Category: "LABS"},

	// Vitals
	"8480-6": {Code: "8480-6", Display: "Systolic blood pressure", Analyte: "blood_pressure", StdUnit: "mmHg", Category: "VITALS"},
	"8462-4": {Code: "8462-4", Display: "Diastolic blood pressure", Analyte: "blood_pressure", StdUnit: "mmHg", Category: "VITALS"},
	"8867-4": {Code: "8867-4", Display: "Heart rate", Analyte: "heart_rate", StdUnit: "bpm", Category: "VITALS"},
	"2708-6": {Code: "2708-6", Display: "Oxygen saturation in Arterial blood", Analyte: "spo2", StdUnit: "%", Category: "VITALS"},
	"8310-5": {Code: "8310-5", Display: "Body temperature", Analyte: "temperature", StdUnit: "degC", Category: "VITALS"},
	"29463-7": {Code: "29463-7", Display: "Body weight", Analyte: "weight", StdUnit: "kg", Category: "VITALS"},
	"8302-2": {Code: "8302-2", Display: "Body height", Analyte: "height", StdUnit: "cm", Category: "VITALS"},
	"39156-5": {Code: "39156-5", Display: "Body mass index (BMI) [Ratio]", Analyte: "bmi", StdUnit: "kg/m2", Category: "VITALS"},

	// Thyroid
	"3016-3": {Code: "3016-3", Display: "Thyrotropin [Units/volume] in Serum or Plasma", Analyte: "tsh", StdUnit: "mIU/L", Category: "LABS"},
	"3024-7": {Code: "3024-7", Display: "Thyroxine (T4) free [Mass/volume] in Serum or Plasma", Analyte: "ft4", StdUnit: "ng/dL", Category: "LABS"},

	// Liver
	"1742-6": {Code: "1742-6", Display: "Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma", Analyte: "alt", StdUnit: "U/L", Category: "LABS"},
	"1920-8": {Code: "1920-8", Display: "Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma", Analyte: "ast", StdUnit: "U/L", Category: "LABS"},

	// Hematology
	"718-7": {Code: "718-7", Display: "Hemoglobin [Mass/volume] in Blood", Analyte: "hemoglobin", StdUnit: "g/dL", Category: "LABS"},

	// Uric acid
	"3084-1": {Code: "3084-1", Display: "Urate [Mass/volume] in Serum or Plasma", Analyte: "uric_acid", StdUnit: "mg/dL", Category: "LABS"},
}

// LookupLOINC returns the LOINCEntry for a given LOINC code, or false if not found.
func LookupLOINC(code string) (LOINCEntry, bool) {
	entry, ok := loincRegistry[code]
	return entry, ok
}

// AnalyteToLOINC returns the primary LOINC code for a given analyte name.
// This is used when adapters provide analyte names without LOINC codes.
var analyteToLOINC = map[string]string{
	"glucose":          "1558-6",
	"fasting_glucose":  "1558-6",
	"random_glucose":   "2345-7",
	"hba1c":            "4548-4",
	"egfr":             "33914-3",
	"creatinine":       "2160-0",
	"bun":              "6299-2",
	"potassium":        "2823-3",
	"sodium":           "2951-2",
	"cholesterol":      "2093-3",
	"hdl":              "2085-9",
	"ldl":              "2089-1",
	"triglycerides":    "2571-8",
	"systolic_bp":      "8480-6",
	"diastolic_bp":     "8462-4",
	"heart_rate":       "8867-4",
	"spo2":             "2708-6",
	"temperature":      "8310-5",
	"weight":           "29463-7",
	"height":           "8302-2",
	"bmi":              "39156-5",
	"tsh":              "3016-3",
	"ft4":              "3024-7",
	"alt":              "1742-6",
	"ast":              "1920-8",
	"hemoglobin":       "718-7",
	"uric_acid":        "3084-1",
	"urine_albumin":    "14959-1",
}

// LookupLOINCByAnalyte returns the primary LOINC code for an analyte name.
func LookupLOINCByAnalyte(analyte string) (string, bool) {
	code, ok := analyteToLOINC[analyte]
	return code, ok
}
```

- [ ] **Step 6: Write snomed_mapper.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/coding/snomed_mapper.go
package coding

// SNOMEDEntry represents a SNOMED CT code mapping.
type SNOMEDEntry struct {
	Code    string
	Display string
}

// snomedRegistry maps common clinical terms to SNOMED CT codes.
// Used for observation category and body site coding.
var snomedRegistry = map[string]SNOMEDEntry{
	// Observation categories
	"vital_signs":          {Code: "vital-signs", Display: "Vital Signs"},
	"laboratory":           {Code: "laboratory", Display: "Laboratory"},
	"survey":               {Code: "survey", Display: "Survey"},
	"activity":             {Code: "activity", Display: "Activity"},
	"social_history":       {Code: "social-history", Display: "Social History"},

	// Body sites
	"left_arm":             {Code: "368208006", Display: "Left upper arm structure"},
	"right_arm":            {Code: "368209003", Display: "Right upper arm structure"},
	"left_wrist":           {Code: "5951000", Display: "Structure of left wrist"},
	"right_wrist":          {Code: "9736006", Display: "Structure of right wrist"},
	"finger":               {Code: "7569003", Display: "Finger structure"},

	// Methods
	"automated":            {Code: "17146006", Display: "Automated measurement"},
	"manual":               {Code: "258104002", Display: "Manual measurement"},
	"self_reported":        {Code: "self-reported", Display: "Patient self-reported"},

	// Condition codes
	"diabetes_mellitus_2":  {Code: "44054006", Display: "Diabetes mellitus type 2"},
	"hypertension":         {Code: "38341003", Display: "Hypertensive disorder"},
	"ckd":                  {Code: "709044004", Display: "Chronic kidney disease"},
	"dyslipidemia":         {Code: "55822004", Display: "Hyperlipidemia"},
}

// LookupSNOMED returns the SNOMEDEntry for a given key, or false if not found.
func LookupSNOMED(key string) (SNOMEDEntry, bool) {
	entry, ok := snomedRegistry[key]
	return entry, ok
}
```

- [ ] **Step 7: Write loinc_mapper_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/coding/loinc_mapper_test.go
package coding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupLOINC_Found(t *testing.T) {
	entry, ok := LookupLOINC("33914-3")
	require.True(t, ok)
	assert.Equal(t, "egfr", entry.Analyte)
	assert.Equal(t, "mL/min/1.73m2", entry.StdUnit)
	assert.Equal(t, "LABS", entry.Category)
}

func TestLookupLOINC_NotFound(t *testing.T) {
	_, ok := LookupLOINC("99999-9")
	assert.False(t, ok)
}

func TestLookupLOINCByAnalyte_Found(t *testing.T) {
	code, ok := LookupLOINCByAnalyte("fasting_glucose")
	require.True(t, ok)
	assert.Equal(t, "1558-6", code)
}

func TestLookupLOINCByAnalyte_NotFound(t *testing.T) {
	_, ok := LookupLOINCByAnalyte("unknown_analyte")
	assert.False(t, ok)
}

func TestLookupSNOMED_Found(t *testing.T) {
	entry, ok := LookupSNOMED("diabetes_mellitus_2")
	require.True(t, ok)
	assert.Equal(t, "44054006", entry.Code)
}

func TestLookupSNOMED_NotFound(t *testing.T) {
	_, ok := LookupSNOMED("nonexistent")
	assert.False(t, ok)
}

func TestLOINCRegistry_CoversCoreAnalytes(t *testing.T) {
	coreAnalytes := []string{
		"fasting_glucose", "hba1c", "egfr", "creatinine", "potassium",
		"cholesterol", "hdl", "ldl", "triglycerides",
		"systolic_bp", "diastolic_bp", "heart_rate", "weight",
	}
	for _, a := range coreAnalytes {
		code, ok := LookupLOINCByAnalyte(a)
		assert.True(t, ok, "missing LOINC mapping for analyte %q", a)
		_, found := LookupLOINC(code)
		assert.True(t, found, "LOINC code %s for analyte %q not in registry", code, a)
	}
}
```

- [ ] **Step 8: Run all coding tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/coding/... -v -count=1`
Expected: All tests PASS (unit converter + LOINC + SNOMED tests)

- [ ] **Step 9: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/coding/
git commit -m "feat(ingestion): add unit converter and LOINC/SNOMED code mappers

Unit conversions for glucose, cholesterol, triglycerides, creatinine,
HbA1c, BP, temperature, weight, height. LOINC registry covering 30
common cardiometabolic codes. SNOMED registry for observation categories,
body sites, methods, and conditions."
```

---

## Task 2: Normalizer Pipeline Stage (`internal/pipeline/normalizer.go`)

**Files:**
- Create: `internal/pipeline/normalizer.go`
- Create: `internal/pipeline/normalizer_test.go`

- [ ] **Step 1: Write normalizer_test.go (TDD)**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/normalizer_test.go
package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestNormalizer_ConvertsMmolToMgdL(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6", // Fasting glucose
		Value:           7.0,
		Unit:            "mmol/L",
		Timestamp:       time.Now(),
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "mg/dL", obs.Unit)
	assert.InDelta(t, 126.0, obs.Value, 0.5)
}

func TestNormalizer_KeepsMgdLUnchanged(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6",
		Value:           126.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "mg/dL", obs.Unit)
	assert.Equal(t, 126.0, obs.Value)
}

func TestNormalizer_MapsAnalyteToLOINC(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourcePatientReported,
		ObservationType: canonical.ObsPatientReported,
		LOINCCode:       "",
		ValueString:     "fasting_glucose",
		Value:           180.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "1558-6", obs.LOINCCode)
}

func TestNormalizer_FlagsUnmappedCode(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "",
		ValueString:     "unknown_test_xyz",
		Value:           42.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err) // Should not error — flags instead
	assert.Contains(t, obs.Flags, canonical.FlagUnmappedCode)
}

func TestNormalizer_ConvertsBPKpaToMmHg(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceDevice,
		ObservationType: canonical.ObsVitals,
		LOINCCode:       "8480-6", // Systolic BP
		Value:           16.0,
		Unit:            "kPa",
		Timestamp:       time.Now(),
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "mmHg", obs.Unit)
	assert.InDelta(t, 120.0, obs.Value, 0.5)
}

func TestNormalizer_FlagsStaleObservation(t *testing.T) {
	n := NewNormalizer(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6",
		Value:           100.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now().Add(-25 * time.Hour), // >24h old
	}

	err := n.Normalize(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagStale)
}
```

- [ ] **Step 2: Verify test fails**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/pipeline/... -v -count=1 -run TestNormalizer 2>&1 | head -5`
Expected: Compilation error — `NewNormalizer` undefined.

- [ ] **Step 3: Write normalizer.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/normalizer.go
package pipeline

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// staleThreshold defines how old an observation can be before being flagged.
const staleThreshold = 24 * time.Hour

// DefaultNormalizer applies unit conversion, LOINC code mapping, and
// temporal staleness checks to a CanonicalObservation.
type DefaultNormalizer struct {
	logger *zap.Logger
}

// NewNormalizer creates a new DefaultNormalizer.
func NewNormalizer(logger *zap.Logger) *DefaultNormalizer {
	return &DefaultNormalizer{logger: logger}
}

// Normalize applies unit conversion, code mapping, and staleness checks.
// It modifies the observation in place.
func (n *DefaultNormalizer) Normalize(ctx context.Context, obs *canonical.CanonicalObservation) error {
	// Step 1: Map analyte name to LOINC code if missing
	if obs.LOINCCode == "" && obs.ValueString != "" {
		if code, ok := coding.LookupLOINCByAnalyte(obs.ValueString); ok {
			obs.LOINCCode = code
			n.logger.Debug("mapped analyte to LOINC",
				zap.String("analyte", obs.ValueString),
				zap.String("loinc", code),
			)
		} else {
			obs.Flags = append(obs.Flags, canonical.FlagUnmappedCode)
			n.logger.Warn("unmapped analyte — no LOINC code found",
				zap.String("analyte", obs.ValueString),
				zap.String("source_type", string(obs.SourceType)),
			)
		}
	}

	// Step 2: Unit conversion using LOINC registry metadata
	if obs.LOINCCode != "" {
		entry, ok := coding.LookupLOINC(obs.LOINCCode)
		if ok && entry.StdUnit != "" && obs.Unit != entry.StdUnit {
			converted, stdUnit, err := coding.NormalizeToStandardUnit(obs.Unit, obs.Value, entry.Analyte)
			if err != nil {
				n.logger.Warn("unit conversion failed — keeping original",
					zap.String("loinc", obs.LOINCCode),
					zap.String("from_unit", obs.Unit),
					zap.String("to_unit", entry.StdUnit),
					zap.Error(err),
				)
			} else {
				n.logger.Debug("converted unit",
					zap.String("loinc", obs.LOINCCode),
					zap.String("from", obs.Unit),
					zap.String("to", stdUnit),
					zap.Float64("from_val", obs.Value),
					zap.Float64("to_val", converted),
				)
				obs.Value = converted
				obs.Unit = stdUnit
			}
		}
	}

	// Step 3: Temporal staleness check
	if !obs.Timestamp.IsZero() && time.Since(obs.Timestamp) > staleThreshold {
		obs.Flags = append(obs.Flags, canonical.FlagStale)
		n.logger.Debug("observation flagged as stale",
			zap.Time("timestamp", obs.Timestamp),
			zap.Duration("age", time.Since(obs.Timestamp)),
		)
	}

	return nil
}
```

- [ ] **Step 4: Run normalizer tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/pipeline/... -v -count=1 -run TestNormalizer`
Expected: All 5 normalizer tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/normalizer.go \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/normalizer_test.go
git commit -m "feat(ingestion): add normalizer pipeline stage

Unit conversion (mmol→mg/dL, kPa→mmHg, etc), LOINC code mapping from
analyte names, temporal staleness check (>24h → STALE flag). Implements
pipeline.Normalizer interface."
```

---

## Task 3: Validator Pipeline Stage (`internal/pipeline/validator.go`)

**Files:**
- Create: `internal/pipeline/validator.go`
- Create: `internal/pipeline/validator_test.go`

- [ ] **Step 1: Write validator_test.go (TDD)**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/validator_test.go
package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func makeObs(loinc string, value float64, unit string) *canonical.CanonicalObservation {
	return &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       loinc,
		Value:           value,
		Unit:            unit,
		Timestamp:       time.Now(),
	}
}

func TestValidator_NormalGlucose(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("1558-6", 95.0, "mg/dL")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, obs.QualityScore, 0.1) // Normal value, high quality
	assert.Empty(t, obs.Flags)
}

func TestValidator_CriticalGlucoseHigh(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("1558-6", 450.0, "mg/dL")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
	assert.True(t, obs.QualityScore > 0) // Value is real but critical
}

func TestValidator_ImplausibleGlucose(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("1558-6", 1500.0, "mg/dL") // >600 mg/dL is implausible

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagImplausible)
	assert.True(t, obs.QualityScore < 0.3) // Implausible = very low quality
}

func TestValidator_CriticalEGFR(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("33914-3", 12.0, "mL/min/1.73m2") // eGFR < 15 = critical

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
}

func TestValidator_NormalBP(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("8480-6", 120.0, "mmHg")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Empty(t, obs.Flags)
	assert.InDelta(t, 1.0, obs.QualityScore, 0.1)
}

func TestValidator_CriticalBPHigh(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("8480-6", 195.0, "mmHg") // SBP >= 180 = critical

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
}

func TestValidator_ImplausibleBP(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("8480-6", 350.0, "mmHg") // > 300 mmHg is implausible

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagImplausible)
}

func TestValidator_CriticalPotassiumHigh(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("2823-3", 6.5, "mEq/L") // K+ >= 6.0 = critical

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
}

func TestValidator_CriticalPotassiumLow(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("2823-3", 2.8, "mEq/L") // K+ <= 3.0 = critical

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Contains(t, obs.Flags, canonical.FlagCriticalValue)
}

func TestValidator_NormalHbA1c(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("4548-4", 6.5, "%")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.Empty(t, obs.Flags)
}

func TestValidator_QualityScorePatientReported(t *testing.T) {
	v := NewValidator(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourcePatientReported,
		ObservationType: canonical.ObsPatientReported,
		LOINCCode:       "1558-6",
		Value:           140.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	// Patient-reported gets lower base quality than lab
	assert.True(t, obs.QualityScore >= 0.6 && obs.QualityScore <= 0.8,
		"patient-reported quality should be 0.6-0.8, got %f", obs.QualityScore)
}

func TestValidator_QualityScoreDevice(t *testing.T) {
	v := NewValidator(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceDevice,
		ObservationType: canonical.ObsDeviceData,
		LOINCCode:       "8480-6",
		Value:           130.0,
		Unit:            "mmHg",
		Timestamp:       time.Now(),
	}

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	assert.True(t, obs.QualityScore >= 0.85 && obs.QualityScore <= 0.95,
		"device quality should be 0.85-0.95, got %f", obs.QualityScore)
}

func TestValidator_MissingPatientID(t *testing.T) {
	v := NewValidator(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6",
		Value:           100.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	err := v.Validate(context.Background(), obs)
	assert.Error(t, err) // Missing patient ID is a validation error
}

func TestValidator_UnknownLOINC(t *testing.T) {
	v := NewValidator(testLogger())
	obs := makeObs("99999-9", 42.0, "mg/dL")

	err := v.Validate(context.Background(), obs)
	require.NoError(t, err)
	// Unknown LOINC still passes — just gets default quality score
	assert.True(t, obs.QualityScore >= 0.5)
}
```

- [ ] **Step 2: Verify test fails**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/pipeline/... -v -count=1 -run TestValidator 2>&1 | head -5`
Expected: Compilation error — `NewValidator` undefined.

- [ ] **Step 3: Write validator.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/validator.go
package pipeline

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// clinicalRange defines the valid, critical, and implausible ranges for an observation.
type clinicalRange struct {
	// Plausible range — values outside are flagged IMPLAUSIBLE
	PlausibleMin float64
	PlausibleMax float64
	// Critical range — values in critical zone flagged CRITICAL_VALUE
	CriticalLow  float64 // value <= CriticalLow is critical (0 = no low critical)
	CriticalHigh float64 // value >= CriticalHigh is critical (0 = no high critical)
}

// clinicalRanges maps LOINC codes to their clinical range definitions.
var clinicalRanges = map[string]clinicalRange{
	// Fasting glucose (mg/dL)
	"1558-6": {PlausibleMin: 10, PlausibleMax: 600, CriticalLow: 40, CriticalHigh: 400},
	// Random glucose (mg/dL)
	"2345-7": {PlausibleMin: 10, PlausibleMax: 600, CriticalLow: 40, CriticalHigh: 400},
	// Blood glucose (mg/dL)
	"2339-0": {PlausibleMin: 10, PlausibleMax: 600, CriticalLow: 40, CriticalHigh: 400},
	// HbA1c (%)
	"4548-4": {PlausibleMin: 2.0, PlausibleMax: 20.0, CriticalHigh: 14.0},
	// eGFR (mL/min/1.73m2)
	"33914-3": {PlausibleMin: 0, PlausibleMax: 200, CriticalLow: 15},
	// Creatinine (mg/dL)
	"2160-0": {PlausibleMin: 0.1, PlausibleMax: 30.0, CriticalHigh: 10.0},
	// Potassium (mEq/L)
	"2823-3": {PlausibleMin: 1.0, PlausibleMax: 10.0, CriticalLow: 3.0, CriticalHigh: 6.0},
	// Sodium (mEq/L)
	"2951-2": {PlausibleMin: 100, PlausibleMax: 180, CriticalLow: 120, CriticalHigh: 160},
	// Total cholesterol (mg/dL)
	"2093-3": {PlausibleMin: 50, PlausibleMax: 500, CriticalHigh: 400},
	// HDL (mg/dL)
	"2085-9": {PlausibleMin: 5, PlausibleMax: 150},
	// LDL (mg/dL)
	"2089-1": {PlausibleMin: 10, PlausibleMax: 400, CriticalHigh: 300},
	// Triglycerides (mg/dL)
	"2571-8": {PlausibleMin: 10, PlausibleMax: 2000, CriticalHigh: 500},
	// Systolic BP (mmHg)
	"8480-6": {PlausibleMin: 40, PlausibleMax: 300, CriticalLow: 70, CriticalHigh: 180},
	// Diastolic BP (mmHg)
	"8462-4": {PlausibleMin: 20, PlausibleMax: 200, CriticalLow: 40, CriticalHigh: 120},
	// Heart rate (bpm)
	"8867-4": {PlausibleMin: 20, PlausibleMax: 250, CriticalLow: 40, CriticalHigh: 150},
	// SpO2 (%)
	"2708-6": {PlausibleMin: 50, PlausibleMax: 100, CriticalLow: 90},
	// Body temperature (degC)
	"8310-5": {PlausibleMin: 30, PlausibleMax: 45, CriticalLow: 35, CriticalHigh: 40},
	// Body weight (kg)
	"29463-7": {PlausibleMin: 1, PlausibleMax: 500},
	// Body height (cm)
	"8302-2": {PlausibleMin: 30, PlausibleMax: 300},
	// BMI
	"39156-5": {PlausibleMin: 5, PlausibleMax: 80},
	// TSH (mIU/L)
	"3016-3": {PlausibleMin: 0.01, PlausibleMax: 100, CriticalHigh: 50},
	// ALT (U/L)
	"1742-6": {PlausibleMin: 0, PlausibleMax: 5000, CriticalHigh: 1000},
	// AST (U/L)
	"1920-8": {PlausibleMin: 0, PlausibleMax: 5000, CriticalHigh: 1000},
	// Hemoglobin (g/dL)
	"718-7": {PlausibleMin: 2, PlausibleMax: 25, CriticalLow: 7, CriticalHigh: 20},
	// Uric acid (mg/dL)
	"3084-1": {PlausibleMin: 0.5, PlausibleMax: 20, CriticalHigh: 12},
}

// sourceQualityBase assigns a base quality score by source type.
var sourceQualityBase = map[canonical.SourceType]float64{
	canonical.SourceLab:             0.95,
	canonical.SourceEHR:             0.90,
	canonical.SourceABDM:            0.85,
	canonical.SourceDevice:          0.90,
	canonical.SourceWearable:        0.80,
	canonical.SourcePatientReported: 0.70,
	canonical.SourceHPI:             0.75,
}

// DefaultValidator checks clinical ranges, flags critical/implausible values,
// and computes a quality score (0.0-1.0).
type DefaultValidator struct {
	logger *zap.Logger
}

// NewValidator creates a new DefaultValidator.
func NewValidator(logger *zap.Logger) *DefaultValidator {
	return &DefaultValidator{logger: logger}
}

// Validate checks clinical ranges and computes quality score.
// Modifies the observation in place. Returns error only for structural
// issues (missing required fields). Clinical flags are set on the observation,
// not returned as errors.
func (v *DefaultValidator) Validate(ctx context.Context, obs *canonical.CanonicalObservation) error {
	// Structural validation — required fields
	if obs.PatientID == uuid.Nil {
		return fmt.Errorf("observation missing patient_id")
	}
	if obs.Timestamp.IsZero() {
		return fmt.Errorf("observation missing timestamp")
	}

	// Start with base quality score from source type
	baseQuality, ok := sourceQualityBase[obs.SourceType]
	if !ok {
		baseQuality = 0.70
	}
	obs.QualityScore = baseQuality

	// Clinical range check
	if obs.LOINCCode != "" {
		r, found := clinicalRanges[obs.LOINCCode]
		if found {
			v.applyRangeChecks(obs, r)
		}
	}

	// Deductions for existing flags (from normalizer)
	for _, f := range obs.Flags {
		switch f {
		case canonical.FlagStale:
			obs.QualityScore -= 0.10
		case canonical.FlagUnmappedCode:
			obs.QualityScore -= 0.15
		case canonical.FlagManualEntry:
			obs.QualityScore -= 0.05
		}
	}

	// Clamp quality score to [0.0, 1.0]
	if obs.QualityScore < 0.0 {
		obs.QualityScore = 0.0
	}
	if obs.QualityScore > 1.0 {
		obs.QualityScore = 1.0
	}

	return nil
}

// applyRangeChecks checks the observation value against clinical ranges.
func (v *DefaultValidator) applyRangeChecks(obs *canonical.CanonicalObservation, r clinicalRange) {
	val := obs.Value

	// Check implausible range first (superset of critical)
	if val < r.PlausibleMin || val > r.PlausibleMax {
		obs.Flags = append(obs.Flags, canonical.FlagImplausible)
		obs.QualityScore = 0.10 // Very low quality for implausible
		v.logger.Warn("implausible observation value",
			zap.String("loinc", obs.LOINCCode),
			zap.Float64("value", val),
			zap.Float64("plausible_min", r.PlausibleMin),
			zap.Float64("plausible_max", r.PlausibleMax),
		)
		return
	}

	// Check critical ranges
	isCritical := false
	if r.CriticalLow > 0 && val <= r.CriticalLow {
		isCritical = true
	}
	if r.CriticalHigh > 0 && val >= r.CriticalHigh {
		isCritical = true
	}

	if isCritical {
		obs.Flags = append(obs.Flags, canonical.FlagCriticalValue)
		v.logger.Warn("critical observation value detected",
			zap.String("loinc", obs.LOINCCode),
			zap.Float64("value", val),
			zap.String("patient_id", obs.PatientID.String()),
		)
	}
}
```

- [ ] **Step 4: Run validator tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/pipeline/... -v -count=1 -run TestValidator`
Expected: All 14 validator tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/validator.go \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/validator_test.go
git commit -m "feat(ingestion): add validator pipeline stage with clinical range checks

Plausible/critical ranges for 25 LOINC codes. Quality scoring 0.0-1.0
based on source type (lab=0.95, device=0.90, patient-reported=0.70) with
deductions for staleness, unmapped codes, and manual entry. CRITICAL_VALUE
flag for dangerous values (glucose>400, K+>6.0, eGFR<15, SBP>180)."
```

---

## Task 4: FHIR Mapper (`internal/fhir/`)

**Files:**
- Create: `internal/fhir/observation_mapper.go`
- Create: `internal/fhir/diagnostic_report_mapper.go`
- Create: `internal/fhir/medication_mapper.go`
- Create: `internal/fhir/mapper.go`
- Create: `internal/fhir/mapper_test.go`

- [ ] **Step 1: Write observation_mapper.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/fhir/observation_mapper.go
package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// abdmObservationProfile is the ABDM IG v7.0 profile URL for vitals.
const abdmObservationProfile = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/ObservationVitalSignsIN"

// MapObservation converts a CanonicalObservation to a FHIR R4 Observation resource JSON.
// Conforms to ABDM IG v7.0 ObservationVitalSignsIN profile.
func MapObservation(obs *canonical.CanonicalObservation) ([]byte, error) {
	resource := map[string]interface{}{
		"resourceType": "Observation",
		"meta": map[string]interface{}{
			"profile": []string{abdmObservationProfile},
		},
		"status": "final",
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", obs.PatientID.String()),
		},
		"effectiveDateTime": obs.Timestamp.UTC().Format(time.RFC3339),
	}

	// Category
	category := observationCategory(obs)
	resource["category"] = []map[string]interface{}{
		{
			"coding": []map[string]interface{}{
				{
					"system":  "http://terminology.hl7.org/CodeSystem/observation-category",
					"code":    category,
					"display": categoryDisplay(category),
				},
			},
		},
	}

	// Code (LOINC)
	if obs.LOINCCode != "" {
		codeCoding := []map[string]interface{}{
			{
				"system": "http://loinc.org",
				"code":   obs.LOINCCode,
			},
		}
		if entry, ok := coding.LookupLOINC(obs.LOINCCode); ok {
			codeCoding[0]["display"] = entry.Display
		}
		resource["code"] = map[string]interface{}{
			"coding": codeCoding,
		}
	}

	// Value
	if obs.ValueString != "" && obs.Value == 0 {
		resource["valueString"] = obs.ValueString
	} else {
		valueQuantity := map[string]interface{}{
			"value": obs.Value,
		}
		if obs.Unit != "" {
			valueQuantity["unit"] = obs.Unit
			valueQuantity["system"] = "http://unitsofmeasure.org"
			valueQuantity["code"] = ucumCode(obs.Unit)
		}
		resource["valueQuantity"] = valueQuantity
	}

	// Interpretation for critical values
	for _, flag := range obs.Flags {
		if flag == canonical.FlagCriticalValue {
			resource["interpretation"] = []map[string]interface{}{
				{
					"coding": []map[string]interface{}{
						{
							"system":  "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation",
							"code":    "AA",
							"display": "Critical abnormal",
						},
					},
				},
			}
			break
		}
	}

	// Device reference
	if obs.DeviceContext != nil {
		resource["device"] = map[string]interface{}{
			"display": fmt.Sprintf("%s %s (%s)",
				obs.DeviceContext.Manufacturer,
				obs.DeviceContext.Model,
				obs.DeviceContext.DeviceID,
			),
		}
	}

	// Method
	if obs.ClinicalContext != nil && obs.ClinicalContext.Method != "" {
		if entry, ok := coding.LookupSNOMED(obs.ClinicalContext.Method); ok {
			resource["method"] = map[string]interface{}{
				"coding": []map[string]interface{}{
					{
						"system":  "http://snomed.info/sct",
						"code":    entry.Code,
						"display": entry.Display,
					},
				},
			}
		}
	}

	// Body site
	if obs.ClinicalContext != nil && obs.ClinicalContext.BodySite != "" {
		if entry, ok := coding.LookupSNOMED(obs.ClinicalContext.BodySite); ok {
			resource["bodySite"] = map[string]interface{}{
				"coding": []map[string]interface{}{
					{
						"system":  "http://snomed.info/sct",
						"code":    entry.Code,
						"display": entry.Display,
					},
				},
			}
		}
	}

	return json.Marshal(resource)
}

// observationCategory returns the FHIR observation category string.
func observationCategory(obs *canonical.CanonicalObservation) string {
	switch obs.ObservationType {
	case canonical.ObsVitals, canonical.ObsDeviceData:
		return "vital-signs"
	case canonical.ObsLabs:
		return "laboratory"
	case canonical.ObsPatientReported:
		return "survey"
	default:
		return "laboratory"
	}
}

// categoryDisplay returns the display string for a category code.
func categoryDisplay(code string) string {
	switch code {
	case "vital-signs":
		return "Vital Signs"
	case "laboratory":
		return "Laboratory"
	case "survey":
		return "Survey"
	case "social-history":
		return "Social History"
	case "activity":
		return "Activity"
	default:
		return code
	}
}

// ucumCode maps common unit display strings to UCUM codes.
func ucumCode(unit string) string {
	ucumMap := map[string]string{
		"mg/dL":          "mg/dL",
		"mmol/L":         "mmol/L",
		"mmHg":           "mm[Hg]",
		"bpm":            "/min",
		"%":              "%",
		"degC":           "Cel",
		"degF":           "[degF]",
		"kg":             "kg",
		"lbs":            "[lb_av]",
		"cm":             "cm",
		"kg/m2":          "kg/m2",
		"mL/min/1.73m2":  "mL/min/{1.73_m2}",
		"mEq/L":          "meq/L",
		"U/L":            "U/L",
		"g/dL":           "g/dL",
		"mIU/L":          "m[IU]/L",
		"ng/dL":          "ng/dL",
		"mg/L":           "mg/L",
		"mmol/mol":       "mmol/mol",
	}
	if code, ok := ucumMap[unit]; ok {
		return code
	}
	return unit
}
```

- [ ] **Step 2: Write diagnostic_report_mapper.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/fhir/diagnostic_report_mapper.go
package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// abdmDiagnosticReportProfile is the ABDM IG v7.0 profile URL.
const abdmDiagnosticReportProfile = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/DiagnosticReportLabIN"

// MapDiagnosticReport creates a FHIR DiagnosticReport resource wrapping a lab observation.
// This is the required ABDM wrapper for lab results — it references the Observation.
func MapDiagnosticReport(obs *canonical.CanonicalObservation, observationID string) ([]byte, error) {
	resource := map[string]interface{}{
		"resourceType": "DiagnosticReport",
		"meta": map[string]interface{}{
			"profile": []string{abdmDiagnosticReportProfile},
		},
		"status": "final",
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", obs.PatientID.String()),
		},
		"effectiveDateTime": obs.Timestamp.UTC().Format(time.RFC3339),
		"issued":            time.Now().UTC().Format(time.RFC3339),
		"result": []map[string]interface{}{
			{
				"reference": fmt.Sprintf("Observation/%s", observationID),
			},
		},
	}

	// Category — always laboratory for DiagnosticReport
	resource["category"] = []map[string]interface{}{
		{
			"coding": []map[string]interface{}{
				{
					"system":  "http://terminology.hl7.org/CodeSystem/v2-0074",
					"code":    "LAB",
					"display": "Laboratory",
				},
			},
		},
	}

	// Code from LOINC
	if obs.LOINCCode != "" {
		codeCoding := []map[string]interface{}{
			{
				"system": "http://loinc.org",
				"code":   obs.LOINCCode,
			},
		}
		if entry, ok := coding.LookupLOINC(obs.LOINCCode); ok {
			codeCoding[0]["display"] = entry.Display
		}
		resource["code"] = map[string]interface{}{
			"coding": codeCoding,
		}
	}

	// Performer (source)
	if obs.SourceID != "" {
		resource["performer"] = []map[string]interface{}{
			{
				"display": obs.SourceID,
			},
		}
	}

	// Conclusion for critical values
	for _, flag := range obs.Flags {
		if flag == canonical.FlagCriticalValue {
			resource["conclusion"] = "CRITICAL VALUE — immediate clinical review required"
			break
		}
	}

	return json.Marshal(resource)
}
```

- [ ] **Step 3: Write medication_mapper.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/fhir/medication_mapper.go
package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// abdmMedicationStatementProfile is the ABDM IG v7.0 profile URL.
const abdmMedicationStatementProfile = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/MedicationStatementIN"

// MapMedicationStatement creates a FHIR MedicationStatement resource from
// a medication adherence observation.
func MapMedicationStatement(obs *canonical.CanonicalObservation) ([]byte, error) {
	resource := map[string]interface{}{
		"resourceType": "MedicationStatement",
		"meta": map[string]interface{}{
			"profile": []string{abdmMedicationStatementProfile},
		},
		"status": "active",
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", obs.PatientID.String()),
		},
		"effectiveDateTime": obs.Timestamp.UTC().Format(time.RFC3339),
		"dateAsserted":      time.Now().UTC().Format(time.RFC3339),
	}

	// Medication reference from value string (drug name or code)
	if obs.ValueString != "" {
		resource["medicationCodeableConcept"] = map[string]interface{}{
			"text": obs.ValueString,
		}
	}

	// SNOMED code if available
	if obs.SNOMEDCode != "" {
		resource["medicationCodeableConcept"] = map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://snomed.info/sct",
					"code":    obs.SNOMEDCode,
					"display": obs.ValueString,
				},
			},
			"text": obs.ValueString,
		}
	}

	// Category — patient-reported vs clinician
	categoryCode := "patientreported"
	categoryDisplay := "Patient Reported"
	if obs.SourceType == canonical.SourceEHR || obs.SourceType == canonical.SourceABDM {
		categoryCode = "inpatient"
		categoryDisplay = "Inpatient"
	}
	resource["category"] = map[string]interface{}{
		"coding": []map[string]interface{}{
			{
				"system":  "http://terminology.hl7.org/CodeSystem/medication-statement-category",
				"code":    categoryCode,
				"display": categoryDisplay,
			},
		},
	}

	return json.Marshal(resource)
}
```

- [ ] **Step 4: Write mapper.go (composite)**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/fhir/mapper.go
package fhir

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// CompositeMapper implements the pipeline.Mapper interface.
// It selects the correct FHIR resource mapper based on observation type.
type CompositeMapper struct {
	logger *zap.Logger
}

// NewCompositeMapper creates a new CompositeMapper.
func NewCompositeMapper(logger *zap.Logger) *CompositeMapper {
	return &CompositeMapper{logger: logger}
}

// MapToFHIR converts a CanonicalObservation to a FHIR R4 resource JSON.
// For lab results, it produces both an Observation and a DiagnosticReport.
// For medications, it produces a MedicationStatement.
// For vitals and other types, it produces an Observation.
func (m *CompositeMapper) MapToFHIR(ctx context.Context, obs *canonical.CanonicalObservation) ([]byte, error) {
	switch obs.ObservationType {
	case canonical.ObsMedications:
		m.logger.Debug("mapping to MedicationStatement",
			zap.String("patient_id", obs.PatientID.String()),
		)
		return MapMedicationStatement(obs)

	case canonical.ObsLabs:
		// Lab results map to Observation (the DiagnosticReport wrapper
		// is created separately after the Observation ID is known)
		m.logger.Debug("mapping lab to Observation",
			zap.String("loinc", obs.LOINCCode),
			zap.String("patient_id", obs.PatientID.String()),
		)
		return MapObservation(obs)

	case canonical.ObsVitals, canonical.ObsDeviceData, canonical.ObsPatientReported,
		canonical.ObsHPI, canonical.ObsABDMRecords, canonical.ObsGeneral:
		m.logger.Debug("mapping to Observation",
			zap.String("type", string(obs.ObservationType)),
			zap.String("patient_id", obs.PatientID.String()),
		)
		return MapObservation(obs)

	default:
		return nil, fmt.Errorf("unsupported observation type for FHIR mapping: %s", obs.ObservationType)
	}
}
```

- [ ] **Step 5: Write mapper_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/fhir/mapper_test.go
package fhir

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestMapObservation_LabResult(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		SourceID:        "thyrocare",
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "33914-3",
		Value:           42.0,
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
	}

	data, err := MapObservation(obs)
	require.NoError(t, err)

	var resource map[string]interface{}
	err = json.Unmarshal(data, &resource)
	require.NoError(t, err)

	assert.Equal(t, "Observation", resource["resourceType"])
	assert.Equal(t, "final", resource["status"])

	subject := resource["subject"].(map[string]interface{})
	assert.Equal(t, "Patient/a1b2c3d4-e5f6-7890-abcd-ef1234567890", subject["reference"])

	// Check LOINC code
	code := resource["code"].(map[string]interface{})
	codings := code["coding"].([]interface{})
	firstCoding := codings[0].(map[string]interface{})
	assert.Equal(t, "http://loinc.org", firstCoding["system"])
	assert.Equal(t, "33914-3", firstCoding["code"])

	// Check value
	vq := resource["valueQuantity"].(map[string]interface{})
	assert.Equal(t, 42.0, vq["value"])
	assert.Equal(t, "mL/min/1.73m2", vq["unit"])
	assert.Equal(t, "http://unitsofmeasure.org", vq["system"])

	// Check category = laboratory
	categories := resource["category"].([]interface{})
	firstCat := categories[0].(map[string]interface{})
	catCodings := firstCat["coding"].([]interface{})
	firstCatCoding := catCodings[0].(map[string]interface{})
	assert.Equal(t, "laboratory", firstCatCoding["code"])

	// Check ABDM profile
	meta := resource["meta"].(map[string]interface{})
	profiles := meta["profile"].([]interface{})
	assert.Equal(t, abdmObservationProfile, profiles[0])
}

func TestMapObservation_CriticalValue(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "2823-3",
		Value:           6.5,
		Unit:            "mEq/L",
		Flags:           []canonical.Flag{canonical.FlagCriticalValue},
		Timestamp:       time.Now(),
	}

	data, err := MapObservation(obs)
	require.NoError(t, err)

	var resource map[string]interface{}
	json.Unmarshal(data, &resource)

	// Should have interpretation = AA (critical abnormal)
	interpretation := resource["interpretation"].([]interface{})
	firstInterp := interpretation[0].(map[string]interface{})
	interpCodings := firstInterp["coding"].([]interface{})
	firstInterpCoding := interpCodings[0].(map[string]interface{})
	assert.Equal(t, "AA", firstInterpCoding["code"])
}

func TestMapObservation_VitalSigns(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceDevice,
		ObservationType: canonical.ObsVitals,
		LOINCCode:       "8480-6",
		Value:           130.0,
		Unit:            "mmHg",
		Timestamp:       time.Now(),
		DeviceContext: &canonical.DeviceContext{
			DeviceID:     "bp-001",
			DeviceType:   "blood_pressure_monitor",
			Manufacturer: "Omron",
			Model:        "HEM-7120",
		},
	}

	data, err := MapObservation(obs)
	require.NoError(t, err)

	var resource map[string]interface{}
	json.Unmarshal(data, &resource)

	categories := resource["category"].([]interface{})
	firstCat := categories[0].(map[string]interface{})
	catCodings := firstCat["coding"].([]interface{})
	firstCatCoding := catCodings[0].(map[string]interface{})
	assert.Equal(t, "vital-signs", firstCatCoding["code"])

	// Device reference
	device := resource["device"].(map[string]interface{})
	assert.Contains(t, device["display"], "Omron")
	assert.Contains(t, device["display"], "HEM-7120")
}

func TestMapDiagnosticReport(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		SourceID:        "thyrocare",
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "33914-3",
		Value:           42.0,
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Now(),
	}

	data, err := MapDiagnosticReport(obs, "obs-123")
	require.NoError(t, err)

	var resource map[string]interface{}
	json.Unmarshal(data, &resource)

	assert.Equal(t, "DiagnosticReport", resource["resourceType"])
	assert.Equal(t, "final", resource["status"])

	results := resource["result"].([]interface{})
	firstResult := results[0].(map[string]interface{})
	assert.Equal(t, "Observation/obs-123", firstResult["reference"])

	performer := resource["performer"].([]interface{})
	firstPerf := performer[0].(map[string]interface{})
	assert.Equal(t, "thyrocare", firstPerf["display"])
}

func TestMapMedicationStatement(t *testing.T) {
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourcePatientReported,
		ObservationType: canonical.ObsMedications,
		ValueString:     "Metformin 500mg",
		Timestamp:       time.Now(),
	}

	data, err := MapMedicationStatement(obs)
	require.NoError(t, err)

	var resource map[string]interface{}
	json.Unmarshal(data, &resource)

	assert.Equal(t, "MedicationStatement", resource["resourceType"])
	assert.Equal(t, "active", resource["status"])

	medConcept := resource["medicationCodeableConcept"].(map[string]interface{})
	assert.Equal(t, "Metformin 500mg", medConcept["text"])

	cat := resource["category"].(map[string]interface{})
	catCodings := cat["coding"].([]interface{})
	assert.Equal(t, "patientreported", catCodings[0].(map[string]interface{})["code"])
}

func TestCompositeMapper_RoutesToCorrectMapper(t *testing.T) {
	m := NewCompositeMapper(testLogger())
	ctx := context.Background()

	tests := []struct {
		name         string
		obsType      canonical.ObservationType
		wantResource string
	}{
		{"lab → Observation", canonical.ObsLabs, "Observation"},
		{"vitals → Observation", canonical.ObsVitals, "Observation"},
		{"device → Observation", canonical.ObsDeviceData, "Observation"},
		{"patient-reported → Observation", canonical.ObsPatientReported, "Observation"},
		{"medications → MedicationStatement", canonical.ObsMedications, "MedicationStatement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &canonical.CanonicalObservation{
				ID:              uuid.New(),
				PatientID:       uuid.New(),
				TenantID:        uuid.New(),
				SourceType:      canonical.SourceLab,
				ObservationType: tt.obsType,
				LOINCCode:       "1558-6",
				Value:           100.0,
				Unit:            "mg/dL",
				ValueString:     "Metformin 500mg",
				Timestamp:       time.Now(),
			}

			data, err := m.MapToFHIR(ctx, obs)
			require.NoError(t, err)

			var resource map[string]interface{}
			json.Unmarshal(data, &resource)
			assert.Equal(t, tt.wantResource, resource["resourceType"])
		})
	}
}
```

- [ ] **Step 6: Run FHIR mapper tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/fhir/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/fhir/
git commit -m "feat(ingestion): add FHIR R4 mapper with ABDM IG v7.0 profiles

Observation mapper (ObservationVitalSignsIN), DiagnosticReport mapper
(DiagnosticReportLabIN), MedicationStatement mapper (MedicationStatementIN).
CompositeMapper routes by ObservationType. UCUM unit codes, LOINC coding,
critical value interpretation (AA), device reference, method/body site."
```

---

## Task 5: Kafka Producer + Router (`internal/kafka/`)

**Files:**
- Create: `internal/kafka/producer.go`
- Create: `internal/kafka/router.go`
- Create: `internal/kafka/router_test.go`

- [ ] **Step 1: Write router_test.go (TDD)**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/router_test.go
package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestRouter_LabsToIngestionLabs(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.MustParse("aaaabbbb-cccc-dddd-eeee-ffffffffffff"),
		ObservationType: canonical.ObsLabs,
		Timestamp:       time.Now(),
	}

	topic, key, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.labs", topic)
	assert.Equal(t, "aaaabbbb-cccc-dddd-eeee-ffffffffffff", key)
}

func TestRouter_VitalsToIngestionVitals(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsVitals,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.vitals", topic)
}

func TestRouter_DeviceDataToIngestionDeviceData(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsDeviceData,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.device-data", topic)
}

func TestRouter_PatientReportedToIngestionPatientReported(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsPatientReported,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.patient-reported", topic)
}

func TestRouter_MedicationsToIngestionMedications(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsMedications,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.medications", topic)
}

func TestRouter_HPIToIngestionHPI(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsHPI,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.hpi", topic)
}

func TestRouter_ABDMRecordsToIngestionABDM(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsABDMRecords,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.abdm-records", topic)
}

func TestRouter_GeneralToIngestionObservations(t *testing.T) {
	r := NewTopicRouter(testLogger())
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ObservationType: canonical.ObsGeneral,
		Timestamp:       time.Now(),
	}

	topic, _, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, "ingestion.observations", topic)
}

func TestRouter_PartitionKeyIsPatientID(t *testing.T) {
	r := NewTopicRouter(testLogger())
	patientID := uuid.New()
	obs := &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       patientID,
		ObservationType: canonical.ObsLabs,
		Timestamp:       time.Now(),
	}

	_, key, err := r.Route(context.Background(), obs)
	require.NoError(t, err)
	assert.Equal(t, patientID.String(), key)
}
```

- [ ] **Step 2: Verify test fails**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/kafka/... -v -count=1 -run TestRouter 2>&1 | head -5`
Expected: Compilation error — `NewTopicRouter` undefined.

- [ ] **Step 3: Write router.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/router.go
package kafka

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// topicMap maps ObservationType to Kafka topic names.
// Topics follow the pattern ingestion.{domain} per spec section 6.1.
var topicMap = map[canonical.ObservationType]string{
	canonical.ObsLabs:            "ingestion.labs",
	canonical.ObsVitals:          "ingestion.vitals",
	canonical.ObsDeviceData:      "ingestion.device-data",
	canonical.ObsPatientReported: "ingestion.patient-reported",
	canonical.ObsMedications:     "ingestion.medications",
	canonical.ObsHPI:             "ingestion.hpi",
	canonical.ObsABDMRecords:     "ingestion.abdm-records",
	canonical.ObsGeneral:         "ingestion.observations",
}

// TopicRouter selects the Kafka topic and partition key based on
// observation type and patient ID. Implements the pipeline.Router interface.
type TopicRouter struct {
	logger *zap.Logger
}

// NewTopicRouter creates a new TopicRouter.
func NewTopicRouter(logger *zap.Logger) *TopicRouter {
	return &TopicRouter{logger: logger}
}

// Route returns the Kafka topic and partition key for an observation.
// Partition key is always the patientId (UUID string) to ensure ordered
// processing per patient.
func (r *TopicRouter) Route(ctx context.Context, obs *canonical.CanonicalObservation) (string, string, error) {
	topic, ok := topicMap[obs.ObservationType]
	if !ok {
		topic = "ingestion.observations" // Fallback to general topic
		r.logger.Warn("unknown observation type — routing to ingestion.observations",
			zap.String("observation_type", string(obs.ObservationType)),
		)
	}

	partitionKey := obs.PatientID.String()
	if partitionKey == "00000000-0000-0000-0000-000000000000" {
		return "", "", fmt.Errorf("cannot route observation with nil patient_id")
	}

	r.logger.Debug("routed observation",
		zap.String("topic", topic),
		zap.String("partition_key", partitionKey),
		zap.String("observation_type", string(obs.ObservationType)),
	)

	return topic, partitionKey, nil
}
```

- [ ] **Step 4: Write producer.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/producer.go
package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// Producer publishes messages to Kafka topics.
type Producer struct {
	writers map[string]*kafkago.Writer
	logger  *zap.Logger
}

// NewProducer creates a Kafka producer that can write to multiple topics.
// Pass the broker addresses; topic-specific writers are created lazily.
func NewProducer(brokers []string, logger *zap.Logger) *Producer {
	return &Producer{
		writers: make(map[string]*kafkago.Writer),
		logger:  logger,
	}
}

// NewProducerWithWriters creates a Kafka producer with pre-configured writers (for testing).
func NewProducerWithWriters(writers map[string]*kafkago.Writer, logger *zap.Logger) *Producer {
	return &Producer{
		writers: writers,
		logger:  logger,
	}
}

// writerFor returns the writer for a topic, creating it lazily if needed.
func (p *Producer) writerFor(topic string, brokers []string) *kafkago.Writer {
	if w, ok := p.writers[topic]; ok {
		return w
	}
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafkago.Hash{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafkago.RequireAll,
		MaxAttempts:  3,
	}
	p.writers[topic] = w
	return w
}

// Publish sends a CanonicalObservation to the appropriate Kafka topic
// wrapped in the standard Envelope format.
func (p *Producer) Publish(
	ctx context.Context,
	topic string,
	partitionKey string,
	obs *canonical.CanonicalObservation,
	fhirResourceType string,
	fhirResourceID string,
	brokers []string,
) error {
	envelope := Envelope{
		EventID:          uuid.New(),
		EventType:        eventTypeFromObservationType(obs.ObservationType),
		SourceType:       string(obs.SourceType),
		PatientID:        obs.PatientID,
		TenantID:         obs.TenantID,
		Timestamp:        time.Now().UTC(),
		FHIRResourceType: fhirResourceType,
		FHIRResourceID:   fhirResourceID,
		Payload: map[string]interface{}{
			"loinc_code":       obs.LOINCCode,
			"value":            obs.Value,
			"unit":             obs.Unit,
			"observation_type": string(obs.ObservationType),
		},
		QualityScore: obs.QualityScore,
		Flags:        flagsToStrings(obs.Flags),
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	writer := p.writerFor(topic, brokers)
	err = writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(partitionKey),
		Value: data,
	})
	if err != nil {
		p.logger.Error("kafka publish failed",
			zap.String("topic", topic),
			zap.String("partition_key", partitionKey),
			zap.Error(err),
		)
		return err
	}

	p.logger.Info("published to kafka",
		zap.String("topic", topic),
		zap.String("event_id", envelope.EventID.String()),
		zap.String("patient_id", obs.PatientID.String()),
	)
	return nil
}

// Close closes all Kafka writers.
func (p *Producer) Close() error {
	var lastErr error
	for topic, w := range p.writers {
		if err := w.Close(); err != nil {
			p.logger.Error("failed to close kafka writer",
				zap.String("topic", topic),
				zap.Error(err),
			)
			lastErr = err
		}
	}
	return lastErr
}

// eventTypeFromObservationType maps observation types to Kafka event type strings.
func eventTypeFromObservationType(obsType canonical.ObservationType) string {
	switch obsType {
	case canonical.ObsLabs:
		return "LAB_RESULT"
	case canonical.ObsVitals:
		return "VITAL_SIGN"
	case canonical.ObsDeviceData:
		return "DEVICE_READING"
	case canonical.ObsPatientReported:
		return "PATIENT_REPORT"
	case canonical.ObsMedications:
		return "MEDICATION_UPDATE"
	case canonical.ObsHPI:
		return "HPI_SLOT_DATA"
	case canonical.ObsABDMRecords:
		return "ABDM_RECORD"
	default:
		return "OBSERVATION"
	}
}

// flagsToStrings converts canonical flags to string slice.
func flagsToStrings(flags []canonical.Flag) []string {
	if len(flags) == 0 {
		return nil
	}
	result := make([]string, len(flags))
	for i, f := range flags {
		result[i] = string(f)
	}
	return result
}
```

- [ ] **Step 5: Run router tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/kafka/... -v -count=1 -run TestRouter`
Expected: All 9 router tests PASS

- [ ] **Step 6: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/producer.go \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/router.go \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/router_test.go
git commit -m "feat(ingestion): add Kafka producer and topic router

TopicRouter maps 8 ObservationTypes to 8 ingestion.* topics (spec 6.1).
Partition key = patientId for ordered per-patient processing. Producer
uses segmentio/kafka-go with Hash balancer, RequireAll acks, 3 retries.
Envelope format matches spec section 6.3."
```

---

## Task 6: DLQ Publisher (`internal/dlq/`)

**Files:**
- Create: `internal/dlq/publisher.go`
- Create: `internal/dlq/replay.go`
- Create: `internal/dlq/publisher_test.go`

- [ ] **Step 1: Write publisher_test.go (TDD)**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/publisher_test.go
package dlq

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestDLQEntry_Validate(t *testing.T) {
	entry := &DLQEntry{
		ErrorClass:   ErrorClassParse,
		SourceType:   "LAB",
		SourceID:     "thyrocare",
		RawPayload:   []byte(`{"invalid": "json`),
		ErrorMessage: "unexpected end of JSON input",
	}

	err := entry.Validate()
	require.NoError(t, err)
}

func TestDLQEntry_ValidateEmptyPayload(t *testing.T) {
	entry := &DLQEntry{
		ErrorClass:   ErrorClassParse,
		SourceType:   "LAB",
		ErrorMessage: "some error",
	}

	err := entry.Validate()
	assert.Error(t, err) // Missing raw payload
}

func TestDLQEntry_ValidateEmptyErrorClass(t *testing.T) {
	entry := &DLQEntry{
		SourceType:   "LAB",
		RawPayload:   []byte("data"),
		ErrorMessage: "some error",
	}

	err := entry.Validate()
	assert.Error(t, err) // Missing error class
}

func TestErrorClasses(t *testing.T) {
	classes := []ErrorClass{
		ErrorClassParse,
		ErrorClassNormalization,
		ErrorClassValidation,
		ErrorClassMapping,
		ErrorClassPublish,
		ErrorClassFHIRWrite,
	}
	assert.Len(t, classes, 6)
}

func TestPublisher_PublishToMemory(t *testing.T) {
	// Test with in-memory store (no real DB)
	p := NewMemoryPublisher(testLogger())

	entry := &DLQEntry{
		ErrorClass:   ErrorClassParse,
		SourceType:   "LAB",
		SourceID:     "thyrocare",
		RawPayload:   []byte(`{"bad": "data"`),
		ErrorMessage: "invalid JSON",
	}

	err := p.Publish(context.Background(), entry)
	require.NoError(t, err)

	entries := p.ListPending(context.Background())
	require.Len(t, entries, 1)
	assert.Equal(t, ErrorClassParse, entries[0].ErrorClass)
	assert.Equal(t, StatusPending, entries[0].Status)
}

func TestPublisher_ReplayEntry(t *testing.T) {
	p := NewMemoryPublisher(testLogger())

	entry := &DLQEntry{
		ErrorClass:   ErrorClassValidation,
		SourceType:   "DEVICE",
		RawPayload:   []byte(`{"value": -1}`),
		ErrorMessage: "negative value",
	}

	_ = p.Publish(context.Background(), entry)
	entries := p.ListPending(context.Background())
	require.Len(t, entries, 1)

	err := p.MarkReplayed(context.Background(), entries[0].ID)
	require.NoError(t, err)

	pending := p.ListPending(context.Background())
	assert.Len(t, pending, 0)
}
```

- [ ] **Step 2: Verify test fails**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/dlq/... -v -count=1 2>&1 | head -5`
Expected: Compilation error — types and functions undefined.

- [ ] **Step 3: Write publisher.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/publisher.go
package dlq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ErrorClass categorizes the type of error that caused DLQ entry.
// Maps to spec section 7.1 error classes.
type ErrorClass string

const (
	ErrorClassParse         ErrorClass = "PARSE"
	ErrorClassNormalization ErrorClass = "NORMALIZATION"
	ErrorClassValidation    ErrorClass = "VALIDATION"
	ErrorClassMapping       ErrorClass = "MAPPING"
	ErrorClassPublish       ErrorClass = "PUBLISH"
	ErrorClassFHIRWrite     ErrorClass = "FHIR_WRITE"
)

// DLQStatus represents the lifecycle state of a DLQ entry.
type DLQStatus string

const (
	StatusPending   DLQStatus = "PENDING"
	StatusReplayed  DLQStatus = "REPLAYED"
	StatusDiscarded DLQStatus = "DISCARDED"
)

// DLQEntry represents a message that failed processing and was sent to the DLQ.
type DLQEntry struct {
	ID           uuid.UUID  `json:"id"`
	ErrorClass   ErrorClass `json:"error_class"`
	SourceType   string     `json:"source_type"`
	SourceID     string     `json:"source_id,omitempty"`
	RawPayload   []byte     `json:"raw_payload"`
	ErrorMessage string     `json:"error_message"`
	RetryCount   int        `json:"retry_count"`
	Status       DLQStatus  `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

// Validate checks that the DLQ entry has required fields.
func (e *DLQEntry) Validate() error {
	if e.ErrorClass == "" {
		return fmt.Errorf("DLQ entry missing error_class")
	}
	if len(e.RawPayload) == 0 {
		return fmt.Errorf("DLQ entry missing raw_payload")
	}
	return nil
}

// Publisher handles writing failed messages to the DLQ.
type Publisher interface {
	Publish(ctx context.Context, entry *DLQEntry) error
	ListPending(ctx context.Context) []*DLQEntry
	MarkReplayed(ctx context.Context, id uuid.UUID) error
}

// PostgresPublisher writes DLQ entries to the dlq_messages PostgreSQL table.
type PostgresPublisher struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresPublisher creates a DLQ publisher backed by PostgreSQL.
func NewPostgresPublisher(db *pgxpool.Pool, logger *zap.Logger) *PostgresPublisher {
	return &PostgresPublisher{db: db, logger: logger}
}

// Publish inserts a DLQ entry into the dlq_messages table.
func (p *PostgresPublisher) Publish(ctx context.Context, entry *DLQEntry) error {
	if err := entry.Validate(); err != nil {
		return err
	}

	entry.ID = uuid.New()
	entry.Status = StatusPending
	entry.CreatedAt = time.Now().UTC()

	_, err := p.db.Exec(ctx,
		`INSERT INTO dlq_messages (id, error_class, source_type, source_id, raw_payload, error_message, retry_count, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		entry.ID, string(entry.ErrorClass), entry.SourceType, entry.SourceID,
		entry.RawPayload, entry.ErrorMessage, entry.RetryCount, string(entry.Status), entry.CreatedAt,
	)
	if err != nil {
		p.logger.Error("failed to insert DLQ entry",
			zap.String("error_class", string(entry.ErrorClass)),
			zap.Error(err),
		)
		return fmt.Errorf("insert DLQ entry: %w", err)
	}

	p.logger.Info("published to DLQ",
		zap.String("id", entry.ID.String()),
		zap.String("error_class", string(entry.ErrorClass)),
		zap.String("source_type", entry.SourceType),
	)
	return nil
}

// ListPending returns all DLQ entries with PENDING status.
func (p *PostgresPublisher) ListPending(ctx context.Context) []*DLQEntry {
	rows, err := p.db.Query(ctx,
		`SELECT id, error_class, source_type, source_id, raw_payload, error_message, retry_count, status, created_at, resolved_at
		 FROM dlq_messages WHERE status = $1 ORDER BY created_at ASC`, string(StatusPending))
	if err != nil {
		p.logger.Error("failed to list pending DLQ entries", zap.Error(err))
		return nil
	}
	defer rows.Close()

	var entries []*DLQEntry
	for rows.Next() {
		e := &DLQEntry{}
		var errorClass, sourceType, status string
		err := rows.Scan(&e.ID, &errorClass, &sourceType, &e.SourceID,
			&e.RawPayload, &e.ErrorMessage, &e.RetryCount, &status, &e.CreatedAt, &e.ResolvedAt)
		if err != nil {
			p.logger.Error("failed to scan DLQ entry", zap.Error(err))
			continue
		}
		e.ErrorClass = ErrorClass(errorClass)
		e.SourceType = sourceType
		e.Status = DLQStatus(status)
		entries = append(entries, e)
	}
	return entries
}

// MarkReplayed marks a DLQ entry as replayed.
func (p *PostgresPublisher) MarkReplayed(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := p.db.Exec(ctx,
		`UPDATE dlq_messages SET status = $1, resolved_at = $2 WHERE id = $3`,
		string(StatusReplayed), now, id)
	if err != nil {
		return fmt.Errorf("mark DLQ entry replayed: %w", err)
	}
	p.logger.Info("DLQ entry marked as replayed", zap.String("id", id.String()))
	return nil
}

// MemoryPublisher is an in-memory DLQ publisher for testing.
type MemoryPublisher struct {
	mu      sync.Mutex
	entries []*DLQEntry
	logger  *zap.Logger
}

// NewMemoryPublisher creates an in-memory DLQ publisher.
func NewMemoryPublisher(logger *zap.Logger) *MemoryPublisher {
	return &MemoryPublisher{logger: logger}
}

// Publish adds a DLQ entry to the in-memory store.
func (p *MemoryPublisher) Publish(ctx context.Context, entry *DLQEntry) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	entry.ID = uuid.New()
	entry.Status = StatusPending
	entry.CreatedAt = time.Now().UTC()
	p.entries = append(p.entries, entry)
	return nil
}

// ListPending returns all pending entries.
func (p *MemoryPublisher) ListPending(ctx context.Context) []*DLQEntry {
	p.mu.Lock()
	defer p.mu.Unlock()

	var pending []*DLQEntry
	for _, e := range p.entries {
		if e.Status == StatusPending {
			pending = append(pending, e)
		}
	}
	return pending
}

// MarkReplayed marks an entry as replayed.
func (p *MemoryPublisher) MarkReplayed(ctx context.Context, id uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, e := range p.entries {
		if e.ID == id {
			now := time.Now().UTC()
			e.Status = StatusReplayed
			e.ResolvedAt = &now
			return nil
		}
	}
	return fmt.Errorf("DLQ entry %s not found", id)
}
```

- [ ] **Step 4: Write replay.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/replay.go
package dlq

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ReplayHandler handles the DLQ replay endpoint.
// POST /fhir/OperationOutcome/:id/$replay
type ReplayHandler struct {
	publisher Publisher
	logger    *zap.Logger
}

// NewReplayHandler creates a new ReplayHandler.
func NewReplayHandler(publisher Publisher, logger *zap.Logger) *ReplayHandler {
	return &ReplayHandler{publisher: publisher, logger: logger}
}

// HandleReplay replays a single DLQ message by its ID.
func (h *ReplayHandler) HandleReplay(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid DLQ entry ID",
		})
		return
	}

	err = h.publisher.MarkReplayed(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to replay DLQ entry",
			zap.String("id", idStr),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to replay DLQ entry: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "replayed",
		"dlq_id":  id.String(),
		"message": "DLQ entry marked as replayed and queued for reprocessing",
	})
}

// HandleListPending lists all pending DLQ entries.
// GET /fhir/OperationOutcome?category=dlq
func (h *ReplayHandler) HandleListPending(c *gin.Context) {
	entries := h.publisher.ListPending(c.Request.Context())

	fhirEntries := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		fhirEntries = append(fhirEntries, gin.H{
			"resourceType": "OperationOutcome",
			"id":           e.ID.String(),
			"issue": []gin.H{
				{
					"severity":    "error",
					"code":        "processing",
					"diagnostics": e.ErrorMessage,
					"details": gin.H{
						"text": string(e.ErrorClass),
					},
				},
			},
			"extension": []gin.H{
				{
					"url":         "source_type",
					"valueString": e.SourceType,
				},
				{
					"url":         "source_id",
					"valueString": e.SourceID,
				},
				{
					"url":         "created_at",
					"valueString": e.CreatedAt.String(),
				},
				{
					"url":         "retry_count",
					"valueInteger": e.RetryCount,
				},
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        len(fhirEntries),
		"entry":        fhirEntries,
	})
}
```

- [ ] **Step 5: Run DLQ tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/dlq/... -v -count=1`
Expected: All 6 DLQ tests PASS

- [ ] **Step 6: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/
git commit -m "feat(ingestion): add DLQ publisher with PostgreSQL backend and replay endpoint

6 error classes (PARSE, NORMALIZATION, VALIDATION, MAPPING, PUBLISH,
FHIR_WRITE). PostgresPublisher writes to dlq_messages table from Phase 1
migration. MemoryPublisher for testing. Replay handler marks entries as
REPLAYED. List endpoint returns FHIR OperationOutcome Bundle."
```

---

## Task 7: Patient Self-Report Adapter (`internal/adapters/patient_reported/`)

**Files:**
- Create: `internal/adapters/patient_reported/app_checkin.go`
- Create: `internal/adapters/patient_reported/whatsapp.go`
- Create: `internal/adapters/patient_reported/app_checkin_test.go`

- [ ] **Step 1: Write app_checkin_test.go (TDD)**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/patient_reported/app_checkin_test.go
package patient_reported

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestParseAppCheckin_SingleObservation(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		PatientID: uuid.MustParse("aaaabbbb-cccc-dddd-eeee-ffffffffffff"),
		TenantID:  uuid.MustParse("11112222-3333-4444-5555-666677778888"),
		Timestamp: time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC),
		Readings: []AppReading{
			{
				Analyte: "fasting_glucose",
				Value:   142.0,
				Unit:    "mg/dL",
			},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	require.Len(t, observations, 1)

	obs := observations[0]
	assert.Equal(t, canonical.SourcePatientReported, obs.SourceType)
	assert.Equal(t, "app_checkin", obs.SourceID)
	assert.Equal(t, canonical.ObsPatientReported, obs.ObservationType)
	assert.Equal(t, 142.0, obs.Value)
	assert.Equal(t, "mg/dL", obs.Unit)
	assert.Equal(t, "fasting_glucose", obs.ValueString)
	assert.Equal(t, payload.PatientID, obs.PatientID)
	assert.Equal(t, payload.TenantID, obs.TenantID)
	assert.Contains(t, obs.Flags, canonical.FlagManualEntry)
}

func TestParseAppCheckin_MultipleReadings(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Readings: []AppReading{
			{Analyte: "systolic_bp", Value: 130.0, Unit: "mmHg"},
			{Analyte: "diastolic_bp", Value: 85.0, Unit: "mmHg"},
			{Analyte: "heart_rate", Value: 72.0, Unit: "bpm"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	assert.Len(t, observations, 3)

	// Verify each observation has correct analyte
	analytes := make([]string, 3)
	for i, obs := range observations {
		analytes[i] = obs.ValueString
	}
	assert.Contains(t, analytes, "systolic_bp")
	assert.Contains(t, analytes, "diastolic_bp")
	assert.Contains(t, analytes, "heart_rate")
}

func TestParseAppCheckin_EmptyReadings(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Readings:  []AppReading{},
	}

	_, err := adapter.Parse(payload)
	assert.Error(t, err) // Empty readings should error
}

func TestParseAppCheckin_MissingPatientID(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Readings: []AppReading{
			{Analyte: "weight", Value: 75.0, Unit: "kg"},
		},
	}

	_, err := adapter.Parse(payload)
	assert.Error(t, err) // Missing patient ID should error
}

func TestParseAppCheckin_VitalsGetCorrectObservationType(t *testing.T) {
	adapter := NewAppCheckinAdapter(testLogger())

	payload := AppCheckinPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Readings: []AppReading{
			{Analyte: "systolic_bp", Value: 130.0, Unit: "mmHg"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	assert.Equal(t, canonical.ObsVitals, observations[0].ObservationType)
}
```

- [ ] **Step 2: Verify test fails**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/adapters/patient_reported/... -v -count=1 2>&1 | head -5`
Expected: Compilation error.

- [ ] **Step 3: Write app_checkin.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/patient_reported/app_checkin.go
package patient_reported

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// AppCheckinPayload represents the JSON body from the Flutter app checkin.
type AppCheckinPayload struct {
	PatientID uuid.UUID    `json:"patient_id"`
	TenantID  uuid.UUID    `json:"tenant_id"`
	Timestamp time.Time    `json:"timestamp"`
	Readings  []AppReading `json:"readings"`
}

// AppReading is a single observation reading from the app.
type AppReading struct {
	Analyte string  `json:"analyte"` // e.g., "fasting_glucose", "systolic_bp", "weight"
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
}

// vitalsAnalytes lists analyte names that should be categorized as VITALS.
var vitalsAnalytes = map[string]bool{
	"systolic_bp":  true,
	"diastolic_bp": true,
	"heart_rate":   true,
	"spo2":         true,
	"temperature":  true,
	"weight":       true,
	"height":       true,
	"bmi":          true,
}

// AppCheckinAdapter converts Flutter app structured JSON into CanonicalObservations.
type AppCheckinAdapter struct {
	logger *zap.Logger
}

// NewAppCheckinAdapter creates a new AppCheckinAdapter.
func NewAppCheckinAdapter(logger *zap.Logger) *AppCheckinAdapter {
	return &AppCheckinAdapter{logger: logger}
}

// Parse converts an AppCheckinPayload into one or more CanonicalObservations.
func (a *AppCheckinAdapter) Parse(payload AppCheckinPayload) ([]canonical.CanonicalObservation, error) {
	if payload.PatientID == uuid.Nil {
		return nil, fmt.Errorf("app checkin missing patient_id")
	}
	if len(payload.Readings) == 0 {
		return nil, fmt.Errorf("app checkin has no readings")
	}

	timestamp := payload.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	observations := make([]canonical.CanonicalObservation, 0, len(payload.Readings))

	for _, reading := range payload.Readings {
		obs := canonical.CanonicalObservation{
			ID:          uuid.New(),
			PatientID:   payload.PatientID,
			TenantID:    payload.TenantID,
			SourceType:  canonical.SourcePatientReported,
			SourceID:    "app_checkin",
			Value:       reading.Value,
			Unit:        reading.Unit,
			ValueString: reading.Analyte,
			Timestamp:   timestamp,
			Flags:       []canonical.Flag{canonical.FlagManualEntry},
		}

		// Categorize as VITALS or PATIENT_REPORTED based on analyte
		if vitalsAnalytes[reading.Analyte] {
			obs.ObservationType = canonical.ObsVitals
		} else {
			obs.ObservationType = canonical.ObsPatientReported
		}

		// Try to resolve LOINC code from analyte name
		if loincCode, ok := coding.LookupLOINCByAnalyte(reading.Analyte); ok {
			obs.LOINCCode = loincCode
		}

		observations = append(observations, obs)
	}

	a.logger.Info("parsed app checkin",
		zap.String("patient_id", payload.PatientID.String()),
		zap.Int("reading_count", len(observations)),
	)

	return observations, nil
}
```

- [ ] **Step 4: Write whatsapp.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/patient_reported/whatsapp.go
package patient_reported

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// WhatsAppNLUPayload represents the parsed output from the Tier-1 NLU service.
// The NLU service extracts intent and entities from Hindi/regional language
// free text and sends structured JSON to the ingestion service.
type WhatsAppNLUPayload struct {
	PatientID  uuid.UUID          `json:"patient_id"`
	TenantID   uuid.UUID          `json:"tenant_id"`
	MessageID  string             `json:"message_id"`
	Timestamp  time.Time          `json:"timestamp"`
	Intent     string             `json:"intent"`     // e.g., "report_glucose", "report_bp", "report_symptom"
	Entities   []WhatsAppEntity   `json:"entities"`
	Confidence float64            `json:"confidence"` // NLU confidence 0.0-1.0
	RawText    string             `json:"raw_text"`   // Original message text
}

// WhatsAppEntity is an extracted entity from NLU parsing.
type WhatsAppEntity struct {
	Type  string  `json:"type"`  // e.g., "glucose_value", "systolic_bp", "medication_name"
	Value float64 `json:"value,omitempty"`
	Text  string  `json:"text,omitempty"`
	Unit  string  `json:"unit,omitempty"`
}

// intentToAnalyte maps WhatsApp NLU intents to analyte names.
var intentToAnalyte = map[string]string{
	"report_glucose":   "glucose",
	"report_fasting":   "fasting_glucose",
	"report_bp":        "systolic_bp",
	"report_weight":    "weight",
	"report_symptom":   "",
	"report_hba1c":     "hba1c",
	"report_heart_rate": "heart_rate",
}

// WhatsAppAdapter converts NLU-parsed WhatsApp messages into CanonicalObservations.
type WhatsAppAdapter struct {
	logger *zap.Logger
}

// NewWhatsAppAdapter creates a new WhatsAppAdapter.
func NewWhatsAppAdapter(logger *zap.Logger) *WhatsAppAdapter {
	return &WhatsAppAdapter{logger: logger}
}

// Parse converts a WhatsAppNLUPayload into CanonicalObservations.
func (a *WhatsAppAdapter) Parse(payload WhatsAppNLUPayload) ([]canonical.CanonicalObservation, error) {
	if payload.PatientID == uuid.Nil {
		return nil, fmt.Errorf("whatsapp message missing patient_id")
	}
	if len(payload.Entities) == 0 {
		return nil, fmt.Errorf("whatsapp NLU extracted no entities")
	}

	timestamp := payload.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	observations := make([]canonical.CanonicalObservation, 0, len(payload.Entities))

	for _, entity := range payload.Entities {
		obs := canonical.CanonicalObservation{
			ID:              uuid.New(),
			PatientID:       payload.PatientID,
			TenantID:        payload.TenantID,
			SourceType:      canonical.SourcePatientReported,
			SourceID:        "whatsapp",
			ObservationType: canonical.ObsPatientReported,
			Value:           entity.Value,
			Unit:            entity.Unit,
			ValueString:     entity.Type,
			Timestamp:       timestamp,
			Flags:           []canonical.Flag{canonical.FlagManualEntry},
			RawPayload:      []byte(payload.RawText),
		}

		// Low NLU confidence adds LOW_QUALITY flag
		if payload.Confidence < 0.70 {
			obs.Flags = append(obs.Flags, canonical.FlagLowQuality)
		}

		// Categorize vitals
		if vitalsAnalytes[entity.Type] {
			obs.ObservationType = canonical.ObsVitals
		}

		// Resolve LOINC code
		analyte := entity.Type
		if mapped, ok := intentToAnalyte[payload.Intent]; ok && mapped != "" && analyte == "" {
			analyte = mapped
		}
		if analyte != "" {
			if loincCode, ok := coding.LookupLOINCByAnalyte(analyte); ok {
				obs.LOINCCode = loincCode
			}
		}

		observations = append(observations, obs)
	}

	a.logger.Info("parsed whatsapp message",
		zap.String("patient_id", payload.PatientID.String()),
		zap.String("intent", payload.Intent),
		zap.Float64("confidence", payload.Confidence),
		zap.Int("entity_count", len(observations)),
	)

	return observations, nil
}
```

- [ ] **Step 5: Run adapter tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/adapters/patient_reported/... -v -count=1`
Expected: All 5 app checkin tests PASS

- [ ] **Step 6: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/patient_reported/
git commit -m "feat(ingestion): add patient self-report adapters (app checkin + WhatsApp)

AppCheckinAdapter parses Flutter app structured JSON with multiple
readings per checkin. WhatsAppAdapter parses Tier-1 NLU output with
confidence-based LOW_QUALITY flagging. Both map analytes to LOINC codes
and categorize vitals vs patient-reported observations."
```

---

## Task 8: Device Adapter (`internal/adapters/devices/`)

**Files:**
- Create: `internal/adapters/devices/device_adapter.go`
- Create: `internal/adapters/devices/device_adapter_test.go`

- [ ] **Step 1: Write device_adapter_test.go (TDD)**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/devices/device_adapter_test.go
package devices

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestDeviceAdapter_BPReading(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:     "bp-omron-001",
			DeviceType:   "blood_pressure_monitor",
			Manufacturer: "Omron",
			Model:        "HEM-7120",
			FirmwareVer:  "2.1.0",
		},
		Readings: []DeviceReading{
			{Analyte: "systolic_bp", Value: 135.0, Unit: "mmHg"},
			{Analyte: "diastolic_bp", Value: 88.0, Unit: "mmHg"},
			{Analyte: "heart_rate", Value: 74.0, Unit: "bpm"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	require.Len(t, observations, 3)

	// All should be device source with device context
	for _, obs := range observations {
		assert.Equal(t, canonical.SourceDevice, obs.SourceType)
		assert.Equal(t, canonical.ObsDeviceData, obs.ObservationType)
		require.NotNil(t, obs.DeviceContext)
		assert.Equal(t, "Omron", obs.DeviceContext.Manufacturer)
		assert.Equal(t, "HEM-7120", obs.DeviceContext.Model)
		assert.Equal(t, "bp-omron-001", obs.DeviceContext.DeviceID)
	}
}

func TestDeviceAdapter_GlucometerReading(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:     "gluco-accu-001",
			DeviceType:   "glucometer",
			Manufacturer: "Accu-Chek",
			Model:        "Active",
		},
		Readings: []DeviceReading{
			{Analyte: "glucose", Value: 155.0, Unit: "mg/dL"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	require.Len(t, observations, 1)

	obs := observations[0]
	assert.Equal(t, 155.0, obs.Value)
	assert.Equal(t, "mg/dL", obs.Unit)
	assert.NotEmpty(t, obs.LOINCCode) // Should have resolved glucose LOINC
}

func TestDeviceAdapter_PulseOximeter(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:     "spo2-001",
			DeviceType:   "pulse_oximeter",
			Manufacturer: "Masimo",
			Model:        "MightySat",
		},
		Readings: []DeviceReading{
			{Analyte: "spo2", Value: 97.0, Unit: "%"},
			{Analyte: "heart_rate", Value: 68.0, Unit: "bpm"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	assert.Len(t, observations, 2)
}

func TestDeviceAdapter_EmptyReadings(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:   "dev-001",
			DeviceType: "bp_monitor",
		},
		Readings: []DeviceReading{},
	}

	_, err := adapter.Parse(payload)
	assert.Error(t, err)
}

func TestDeviceAdapter_MissingPatientID(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{DeviceID: "dev-001", DeviceType: "bp_monitor"},
		Readings:  []DeviceReading{{Analyte: "systolic_bp", Value: 120.0, Unit: "mmHg"}},
	}

	_, err := adapter.Parse(payload)
	assert.Error(t, err)
}

func TestDeviceAdapter_WeighingScale(t *testing.T) {
	adapter := NewDeviceAdapter(testLogger())

	payload := DevicePayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Timestamp: time.Now(),
		Device: DeviceInfo{
			DeviceID:     "scale-001",
			DeviceType:   "weighing_scale",
			Manufacturer: "Xiaomi",
			Model:        "Mi Scale 2",
		},
		Readings: []DeviceReading{
			{Analyte: "weight", Value: 72.5, Unit: "kg"},
		},
	}

	observations, err := adapter.Parse(payload)
	require.NoError(t, err)
	require.Len(t, observations, 1)
	assert.Equal(t, 72.5, observations[0].Value)
	assert.Equal(t, "kg", observations[0].Unit)
}
```

- [ ] **Step 2: Verify test fails**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/adapters/devices/... -v -count=1 2>&1 | head -5`
Expected: Compilation error.

- [ ] **Step 3: Write device_adapter.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/devices/device_adapter.go
package devices

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// DevicePayload represents BLE device data relayed through the Flutter app.
type DevicePayload struct {
	PatientID uuid.UUID       `json:"patient_id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Timestamp time.Time       `json:"timestamp"`
	Device    DeviceInfo      `json:"device"`
	Readings  []DeviceReading `json:"readings"`
}

// DeviceInfo holds BLE device metadata.
type DeviceInfo struct {
	DeviceID     string `json:"device_id"`
	DeviceType   string `json:"device_type"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	FirmwareVer  string `json:"firmware_version,omitempty"`
}

// DeviceReading is a single measurement from a BLE device.
type DeviceReading struct {
	Analyte string  `json:"analyte"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
}

// DeviceAdapter converts BLE device readings (relayed via app) into CanonicalObservations.
type DeviceAdapter struct {
	logger *zap.Logger
}

// NewDeviceAdapter creates a new DeviceAdapter.
func NewDeviceAdapter(logger *zap.Logger) *DeviceAdapter {
	return &DeviceAdapter{logger: logger}
}

// Parse converts a DevicePayload into one or more CanonicalObservations.
func (a *DeviceAdapter) Parse(payload DevicePayload) ([]canonical.CanonicalObservation, error) {
	if payload.PatientID == uuid.Nil {
		return nil, fmt.Errorf("device reading missing patient_id")
	}
	if len(payload.Readings) == 0 {
		return nil, fmt.Errorf("device payload has no readings")
	}

	timestamp := payload.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	deviceCtx := &canonical.DeviceContext{
		DeviceID:     payload.Device.DeviceID,
		DeviceType:   payload.Device.DeviceType,
		Manufacturer: payload.Device.Manufacturer,
		Model:        payload.Device.Model,
		FirmwareVer:  payload.Device.FirmwareVer,
	}

	observations := make([]canonical.CanonicalObservation, 0, len(payload.Readings))

	for _, reading := range payload.Readings {
		obs := canonical.CanonicalObservation{
			ID:              uuid.New(),
			PatientID:       payload.PatientID,
			TenantID:        payload.TenantID,
			SourceType:      canonical.SourceDevice,
			SourceID:        payload.Device.DeviceID,
			ObservationType: canonical.ObsDeviceData,
			Value:           reading.Value,
			Unit:            reading.Unit,
			ValueString:     reading.Analyte,
			Timestamp:       timestamp,
			DeviceContext:   deviceCtx,
			ClinicalContext: &canonical.ClinicalContext{
				Method: "automated",
			},
		}

		// Resolve LOINC code from analyte name
		if loincCode, ok := coding.LookupLOINCByAnalyte(reading.Analyte); ok {
			obs.LOINCCode = loincCode
		}

		observations = append(observations, obs)
	}

	a.logger.Info("parsed device reading",
		zap.String("patient_id", payload.PatientID.String()),
		zap.String("device_id", payload.Device.DeviceID),
		zap.String("device_type", payload.Device.DeviceType),
		zap.Int("reading_count", len(observations)),
	)

	return observations, nil
}
```

- [ ] **Step 4: Run device adapter tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/adapters/devices/... -v -count=1`
Expected: All 5 device adapter tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/devices/
git commit -m "feat(ingestion): add BLE device adapter for app-relayed readings

Parses structured JSON from Flutter app BLE relay. Supports BP monitors,
glucometers, pulse oximeters, weighing scales. DeviceContext attached
to all observations. Automated measurement method via SNOMED coding.
LOINC resolution from analyte names."
```

---

## Task 9: Pipeline Orchestrator (`internal/pipeline/orchestrator.go`)

**Files:**
- Create: `internal/pipeline/orchestrator.go`
- Create: `internal/pipeline/orchestrator_test.go`

- [ ] **Step 1: Write orchestrator_test.go (TDD)**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/orchestrator_test.go
package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/dlq"
)

func TestOrchestrator_ProcessSingle(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, // FHIR mapper — nil for unit test (skip FHIR Store write)
		nil, // Router — nil for unit test (skip Kafka publish)
		dlqPub,
		logger,
	)

	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		SourceID:        "thyrocare",
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "33914-3", // eGFR
		Value:           42.0,
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Should have quality score from validator
	assert.True(t, results[0].QualityScore > 0)
	// eGFR 42 is not critical (>15)
	assert.NotContains(t, results[0].Flags, canonical.FlagCriticalValue)
}

func TestOrchestrator_ProcessWithUnitConversion(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, nil,
		dlqPub,
		logger,
	)

	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6", // Fasting glucose
		Value:           7.0,
		Unit:            "mmol/L",
		Timestamp:       time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Should be converted to mg/dL
	assert.Equal(t, "mg/dL", results[0].Unit)
	assert.InDelta(t, 126.0, results[0].Value, 0.5)
}

func TestOrchestrator_CriticalValueFlagged(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, nil,
		dlqPub,
		logger,
	)

	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "33914-3", // eGFR
		Value:           12.0,      // < 15 = critical
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Flags, canonical.FlagCriticalValue)
}

func TestOrchestrator_ValidationErrorGoesToDLQ(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, nil,
		dlqPub,
		logger,
	)

	// Missing patient ID — structural validation error
	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      canonical.SourceLab,
		ObservationType: canonical.ObsLabs,
		LOINCCode:       "1558-6",
		Value:           100.0,
		Unit:            "mg/dL",
		Timestamp:       time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs})
	require.NoError(t, err) // Orchestrator does not error — sends to DLQ
	assert.Len(t, results, 0)

	// Check DLQ
	pending := dlqPub.ListPending(context.Background())
	require.Len(t, pending, 1)
	assert.Equal(t, dlq.ErrorClassValidation, pending[0].ErrorClass)
}

func TestOrchestrator_ProcessMultiple(t *testing.T) {
	logger := testLogger()
	dlqPub := dlq.NewMemoryPublisher(logger)

	orch := NewOrchestrator(
		NewNormalizer(logger),
		NewValidator(logger),
		nil, nil,
		dlqPub,
		logger,
	)

	obs1 := canonical.CanonicalObservation{
		ID: uuid.New(), PatientID: uuid.New(), TenantID: uuid.New(),
		SourceType: canonical.SourceLab, ObservationType: canonical.ObsLabs,
		LOINCCode: "8480-6", Value: 130.0, Unit: "mmHg", Timestamp: time.Now(),
	}
	obs2 := canonical.CanonicalObservation{
		ID: uuid.New(), PatientID: uuid.New(), TenantID: uuid.New(),
		SourceType: canonical.SourceDevice, ObservationType: canonical.ObsDeviceData,
		LOINCCode: "8867-4", Value: 72.0, Unit: "bpm", Timestamp: time.Now(),
	}

	results, err := orch.Process(context.Background(), []canonical.CanonicalObservation{obs1, obs2})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}
```

- [ ] **Step 2: Verify test fails**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/pipeline/... -v -count=1 -run TestOrchestrator 2>&1 | head -5`
Expected: Compilation error — `NewOrchestrator` undefined.

- [ ] **Step 3: Write orchestrator.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/orchestrator.go
package pipeline

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/dlq"
)

// Orchestrator wires pipeline stages together:
// Normalizer → Validator → (Mapper → Router are optional for unit testing)
// Failed observations are sent to the DLQ instead of causing pipeline errors.
type Orchestrator struct {
	normalizer Normalizer
	validator  Validator
	mapper     Mapper
	router     Router
	dlqPub     dlq.Publisher
	logger     *zap.Logger
}

// NewOrchestrator creates a new pipeline Orchestrator.
// mapper and router may be nil (for unit testing without FHIR Store / Kafka).
func NewOrchestrator(
	normalizer Normalizer,
	validator Validator,
	mapper Mapper,
	router Router,
	dlqPub dlq.Publisher,
	logger *zap.Logger,
) *Orchestrator {
	return &Orchestrator{
		normalizer: normalizer,
		validator:  validator,
		mapper:     mapper,
		router:     router,
		dlqPub:     dlqPub,
		logger:     logger,
	}
}

// Process runs a batch of CanonicalObservations through the pipeline stages.
// Returns the successfully processed observations. Failed observations are
// sent to the DLQ — the orchestrator never returns an error for individual
// observation failures.
func (o *Orchestrator) Process(ctx context.Context, observations []canonical.CanonicalObservation) ([]canonical.CanonicalObservation, error) {
	var processed []canonical.CanonicalObservation

	for i := range observations {
		obs := &observations[i]

		// Stage 1: Normalize
		if err := o.normalizer.Normalize(ctx, obs); err != nil {
			o.sendToDLQ(ctx, obs, dlq.ErrorClassNormalization, err)
			continue
		}

		// Stage 2: Validate
		if err := o.validator.Validate(ctx, obs); err != nil {
			o.sendToDLQ(ctx, obs, dlq.ErrorClassValidation, err)
			continue
		}

		// Stage 3: Map to FHIR (optional)
		if o.mapper != nil {
			fhirJSON, err := o.mapper.MapToFHIR(ctx, obs)
			if err != nil {
				o.sendToDLQ(ctx, obs, dlq.ErrorClassMapping, err)
				continue
			}
			// Store FHIR JSON in raw payload for downstream use
			obs.RawPayload = fhirJSON
		}

		// Stage 4: Route (optional — topic/key selection only, actual publish is separate)
		if o.router != nil {
			topic, key, err := o.router.Route(ctx, obs)
			if err != nil {
				o.sendToDLQ(ctx, obs, dlq.ErrorClassPublish, err)
				continue
			}
			o.logger.Debug("observation routed",
				zap.String("topic", topic),
				zap.String("key", key),
				zap.String("loinc", obs.LOINCCode),
			)
		}

		processed = append(processed, *obs)
	}

	o.logger.Info("pipeline batch complete",
		zap.Int("input", len(observations)),
		zap.Int("processed", len(processed)),
		zap.Int("dlq", len(observations)-len(processed)),
	)

	return processed, nil
}

// sendToDLQ publishes a failed observation to the DLQ.
func (o *Orchestrator) sendToDLQ(ctx context.Context, obs *canonical.CanonicalObservation, errorClass dlq.ErrorClass, origErr error) {
	rawPayload, _ := json.Marshal(obs)

	entry := &dlq.DLQEntry{
		ErrorClass:   errorClass,
		SourceType:   string(obs.SourceType),
		SourceID:     obs.SourceID,
		RawPayload:   rawPayload,
		ErrorMessage: origErr.Error(),
	}

	if err := o.dlqPub.Publish(ctx, entry); err != nil {
		o.logger.Error("CRITICAL: failed to publish to DLQ",
			zap.String("error_class", string(errorClass)),
			zap.Error(err),
		)
	}
}
```

- [ ] **Step 4: Run orchestrator tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/pipeline/... -v -count=1 -run TestOrchestrator`
Expected: All 5 orchestrator tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/orchestrator.go \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/orchestrator_test.go
git commit -m "feat(ingestion): add pipeline orchestrator wiring Normalizer→Validator→Mapper→Router

Batch processing with per-observation DLQ routing on failure. Orchestrator
never errors on individual observation failures — sends to DLQ instead.
Mapper and Router optional (nil-safe) for unit testing without FHIR/Kafka."
```

---

## Task 10: Prometheus Metrics (`internal/metrics/collectors.go`)

**Files:**
- Create: `internal/metrics/collectors.go`

- [ ] **Step 1: Write collectors.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/metrics/collectors.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Ingestion Prometheus metrics — 10 metrics from spec section 7.3.

var (
	// MessagesReceived counts total messages received by source type.
	MessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_messages_received_total",
			Help: "Total messages received by the ingestion service",
		},
		[]string{"source_type", "source_id", "tenant_id"},
	)

	// MessagesProcessed counts messages processed by stage and status.
	MessagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_messages_processed_total",
			Help: "Total messages processed by pipeline stage and status",
		},
		[]string{"source_type", "stage", "status"},
	)

	// PipelineDuration tracks the duration of each pipeline stage.
	PipelineDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ingestion_pipeline_duration_seconds",
			Help:    "Duration of each pipeline stage in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source_type", "stage"},
	)

	// CriticalValues counts critical values detected.
	CriticalValues = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_critical_values_total",
			Help: "Total critical values detected during validation",
		},
		[]string{"observation_type", "tenant_id"},
	)

	// DLQMessages counts messages sent to the DLQ.
	DLQMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_dlq_messages_total",
			Help: "Total messages sent to the dead letter queue",
		},
		[]string{"error_class", "source_type"},
	)

	// WALMessagesPending tracks messages waiting in the write-ahead log.
	WALMessagesPending = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ingestion_wal_messages_pending",
			Help: "Number of messages pending in the Kafka WAL failover buffer",
		},
	)

	// PatientResolutionPending tracks unresolved patient identifiers.
	PatientResolutionPending = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ingestion_patient_resolution_pending",
			Help: "Number of observations pending patient resolution",
		},
		[]string{"tenant_id"},
	)

	// ABDMConsentOperations counts ABDM consent operations.
	ABDMConsentOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_abdm_consent_operations_total",
			Help: "Total ABDM consent operations by type and status",
		},
		[]string{"operation", "status"},
	)

	// FHIRValidationFailures counts FHIR validation failures.
	FHIRValidationFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestion_fhir_validation_failures_total",
			Help: "Total FHIR validation failures by profile and violation type",
		},
		[]string{"profile", "violation_type"},
	)

	// SourceFreshness tracks the freshness of data from each source.
	SourceFreshness = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ingestion_source_freshness_seconds",
			Help: "Seconds since last message from each data source",
		},
		[]string{"source_type", "source_id"},
	)
)
```

- [ ] **Step 2: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/metrics/
git commit -m "feat(ingestion): add 10 Prometheus metric collectors from spec section 7.3

messages_received_total, messages_processed_total, pipeline_duration_seconds,
critical_values_total, dlq_messages_total, wal_messages_pending,
patient_resolution_pending, abdm_consent_operations_total,
fhir_validation_failures_total, source_freshness_seconds."
```

---

## Task 11: Wire Handlers — Replace Stubs with Real Implementations

**Files:**
- Create: `internal/api/handlers.go`
- Modify: `internal/api/server.go` (add pipeline dependencies)
- Modify: `internal/api/routes.go` (replace stubs with real handlers)

- [ ] **Step 1: Write handlers.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/handlers.go
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/adapters/devices"
	"github.com/cardiofit/ingestion-service/internal/adapters/patient_reported"
	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/dlq"
	"github.com/cardiofit/ingestion-service/internal/fhir"
	kafkapkg "github.com/cardiofit/ingestion-service/internal/kafka"
	"github.com/cardiofit/ingestion-service/internal/metrics"
	"github.com/cardiofit/ingestion-service/internal/pipeline"
)

// handleFHIRObservation handles POST /fhir/Observation.
// Accepts a FHIR-like observation payload, converts to canonical, runs pipeline.
func (s *Server) handleFHIRObservation(c *gin.Context) {
	start := time.Now()
	metrics.MessagesReceived.WithLabelValues("FHIR", "direct", "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Parse the incoming observation into canonical form
	var incoming struct {
		PatientID string  `json:"patient_id"`
		TenantID  string  `json:"tenant_id"`
		LOINCCode string  `json:"loinc_code"`
		Value     float64 `json:"value"`
		Unit      string  `json:"unit"`
		Timestamp string  `json:"timestamp,omitempty"`
	}
	if err := json.Unmarshal(body, &incoming); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	patientID, err := uuid.Parse(incoming.PatientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient_id"})
		return
	}

	tenantID := uuid.Nil
	if incoming.TenantID != "" {
		tenantID, _ = uuid.Parse(incoming.TenantID)
	}

	ts := time.Now().UTC()
	if incoming.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, incoming.Timestamp); err == nil {
			ts = parsed
		}
	}

	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       patientID,
		TenantID:        tenantID,
		SourceType:      canonical.SourceEHR,
		SourceID:        "fhir_direct",
		ObservationType: canonical.ObsGeneral,
		LOINCCode:       incoming.LOINCCode,
		Value:           incoming.Value,
		Unit:            incoming.Unit,
		Timestamp:       ts,
		RawPayload:      body,
	}

	results, err := s.orchestrator.Process(c.Request.Context(), []canonical.CanonicalObservation{obs})
	if err != nil {
		s.logger.Error("pipeline processing failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "observation rejected — check DLQ for details",
			"dlq_url": "/fhir/OperationOutcome?category=dlq",
		})
		return
	}

	// Write to FHIR Store if available
	var fhirResourceID string
	if s.fhirClient != nil && len(results[0].RawPayload) > 0 {
		resp, err := s.fhirClient.Create("Observation", results[0].RawPayload)
		if err != nil {
			s.logger.Error("FHIR Store write failed", zap.Error(err))
			// Continue — Kafka publish is more important than FHIR Store sync
		} else {
			var created map[string]interface{}
			if json.Unmarshal(resp, &created) == nil {
				if id, ok := created["id"].(string); ok {
					fhirResourceID = id
				}
			}
		}
	}

	// Publish to Kafka
	if s.kafkaProducer != nil && s.topicRouter != nil {
		topic, key, err := s.topicRouter.Route(c.Request.Context(), &results[0])
		if err == nil {
			_ = s.kafkaProducer.Publish(
				c.Request.Context(), topic, key, &results[0],
				"Observation", fhirResourceID, s.config.Kafka.Brokers,
			)
		}
	}

	metrics.PipelineDuration.WithLabelValues(string(obs.SourceType), "total").Observe(time.Since(start).Seconds())

	c.JSON(http.StatusCreated, gin.H{
		"status":           "accepted",
		"observation_id":   results[0].ID.String(),
		"fhir_resource_id": fhirResourceID,
		"quality_score":    results[0].QualityScore,
		"flags":            results[0].Flags,
	})
}

// handleDeviceIngest handles POST /ingest/devices.
func (s *Server) handleDeviceIngest(c *gin.Context) {
	metrics.MessagesReceived.WithLabelValues("DEVICE", "", "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var payload devices.DevicePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	adapter := devices.NewDeviceAdapter(s.logger)
	observations, err := adapter.Parse(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := s.orchestrator.Process(c.Request.Context(), observations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	// Write to FHIR Store and Kafka for each result
	for i := range results {
		s.publishResult(c, &results[i])
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "accepted",
		"processed": len(results),
		"total":     len(observations),
		"rejected":  len(observations) - len(results),
	})
}

// handleAppCheckin handles POST /ingest/app-checkin (patient self-report from Flutter app).
func (s *Server) handleAppCheckin(c *gin.Context) {
	metrics.MessagesReceived.WithLabelValues("PATIENT_REPORTED", "app_checkin", "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var payload patient_reported.AppCheckinPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	adapter := patient_reported.NewAppCheckinAdapter(s.logger)
	observations, err := adapter.Parse(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := s.orchestrator.Process(c.Request.Context(), observations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	for i := range results {
		s.publishResult(c, &results[i])
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "accepted",
		"processed": len(results),
		"total":     len(observations),
	})
}

// handleWhatsAppIngest handles POST /ingest/whatsapp (NLU-parsed WhatsApp messages).
func (s *Server) handleWhatsAppIngest(c *gin.Context) {
	metrics.MessagesReceived.WithLabelValues("PATIENT_REPORTED", "whatsapp", "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var payload patient_reported.WhatsAppNLUPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	adapter := patient_reported.NewWhatsAppAdapter(s.logger)
	observations, err := adapter.Parse(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := s.orchestrator.Process(c.Request.Context(), observations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	for i := range results {
		s.publishResult(c, &results[i])
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "accepted",
		"processed": len(results),
	})
}

// handleHPIIngest handles POST /internal/hpi (slot data from Intake M0, service-to-service).
func (s *Server) handleHPIIngest(c *gin.Context) {
	metrics.MessagesReceived.WithLabelValues("HPI", "intake_m0", "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// HPI data arrives as an array of CanonicalObservations from Intake
	var observations []canonical.CanonicalObservation
	if err := json.Unmarshal(body, &observations); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	// Mark as HPI source
	for i := range observations {
		observations[i].SourceType = canonical.SourceHPI
		observations[i].ObservationType = canonical.ObsHPI
	}

	results, err := s.orchestrator.Process(c.Request.Context(), observations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	for i := range results {
		s.publishResult(c, &results[i])
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "accepted",
		"processed": len(results),
	})
}

// publishResult writes an observation to FHIR Store and publishes to Kafka.
func (s *Server) publishResult(c *gin.Context, obs *canonical.CanonicalObservation) {
	// Check for critical values
	for _, flag := range obs.Flags {
		if flag == canonical.FlagCriticalValue {
			metrics.CriticalValues.WithLabelValues(string(obs.ObservationType), obs.TenantID.String()).Inc()
			break
		}
	}

	// Map to FHIR if not already mapped
	if len(obs.RawPayload) == 0 {
		mapper := fhir.NewCompositeMapper(s.logger)
		fhirJSON, err := mapper.MapToFHIR(c.Request.Context(), obs)
		if err != nil {
			s.logger.Error("FHIR mapping failed", zap.Error(err))
			return
		}
		obs.RawPayload = fhirJSON
	}

	// Write to FHIR Store
	var fhirResourceID string
	if s.fhirClient != nil {
		resourceType := "Observation"
		if obs.ObservationType == canonical.ObsMedications {
			resourceType = "MedicationStatement"
		}
		resp, err := s.fhirClient.Create(resourceType, obs.RawPayload)
		if err != nil {
			s.logger.Error("FHIR Store write failed",
				zap.String("patient_id", obs.PatientID.String()),
				zap.Error(err),
			)
		} else {
			var created map[string]interface{}
			if json.Unmarshal(resp, &created) == nil {
				if id, ok := created["id"].(string); ok {
					fhirResourceID = id
				}
			}
		}

		// For lab results, also create a DiagnosticReport
		if obs.ObservationType == canonical.ObsLabs && fhirResourceID != "" {
			drJSON, err := fhir.MapDiagnosticReport(obs, fhirResourceID)
			if err == nil {
				_, _ = s.fhirClient.Create("DiagnosticReport", drJSON)
			}
		}
	}

	// Publish to Kafka
	if s.kafkaProducer != nil && s.topicRouter != nil {
		topic, key, err := s.topicRouter.Route(c.Request.Context(), obs)
		if err == nil {
			resourceType := "Observation"
			if obs.ObservationType == canonical.ObsMedications {
				resourceType = "MedicationStatement"
			}
			if pubErr := s.kafkaProducer.Publish(
				c.Request.Context(), topic, key, obs,
				resourceType, fhirResourceID, s.config.Kafka.Brokers,
			); pubErr != nil {
				metrics.DLQMessages.WithLabelValues("PUBLISH", string(obs.SourceType)).Inc()
				s.logger.Error("Kafka publish failed",
					zap.String("topic", topic),
					zap.Error(pubErr),
				)
			}
		}
	}
}
```

- [ ] **Step 2: Modify server.go — add pipeline dependencies**

Replace the existing `Server` struct and `NewServer` function to include pipeline components:

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/server.go
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/config"
	"github.com/cardiofit/ingestion-service/internal/dlq"
	kafkapkg "github.com/cardiofit/ingestion-service/internal/kafka"
	"github.com/cardiofit/ingestion-service/internal/pipeline"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// Server holds the HTTP server and all dependencies.
type Server struct {
	Router        *gin.Engine
	config        *config.Config
	db            *pgxpool.Pool
	redis         *redis.Client
	fhirClient    *fhirclient.Client
	logger        *zap.Logger
	orchestrator  *pipeline.Orchestrator
	kafkaProducer *kafkapkg.Producer
	topicRouter   *kafkapkg.TopicRouter
	dlqPublisher  dlq.Publisher
	dlqReplay     *dlq.ReplayHandler
}

// NewServer creates and configures the HTTP server with all dependencies.
func NewServer(
	cfg *config.Config,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	fhirClient *fhirclient.Client,
	logger *zap.Logger,
) *Server {
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Initialize pipeline components
	normalizer := pipeline.NewNormalizer(logger)
	validator := pipeline.NewValidator(logger)

	// DLQ publisher
	var dlqPub dlq.Publisher
	if db != nil {
		dlqPub = dlq.NewPostgresPublisher(db, logger)
	} else {
		dlqPub = dlq.NewMemoryPublisher(logger)
	}

	// FHIR mapper and Kafka router
	fhirMapper := fhir.NewCompositeMapper(logger)
	topicRouter := kafkapkg.NewTopicRouter(logger)

	// Pipeline orchestrator
	orchestrator := pipeline.NewOrchestrator(normalizer, validator, fhirMapper, topicRouter, dlqPub, logger)

	// Kafka producer
	var kafkaProducer *kafkapkg.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		kafkaProducer = kafkapkg.NewProducer(cfg.Kafka.Brokers, logger)
	}

	dlqReplay := dlq.NewReplayHandler(dlqPub, logger)

	s := &Server{
		Router:        router,
		config:        cfg,
		db:            db,
		redis:         redisClient,
		fhirClient:    fhirClient,
		logger:        logger,
		orchestrator:  orchestrator,
		kafkaProducer: kafkaProducer,
		topicRouter:   topicRouter,
		dlqPublisher:  dlqPub,
		dlqReplay:     dlqReplay,
	}

	router.Use(s.metricsMiddleware())
	router.Use(s.corsMiddleware())
	s.setupRoutes()

	return s
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		_ = duration
		_ = strconv.Itoa(c.Writer.Status())
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-User-Role")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
```

**Note:** This `server.go` requires adding the import for `fhir` package. Add `"github.com/cardiofit/ingestion-service/internal/fhir"` to the import block. The `fhir` alias is unused — use the direct package name:

```go
import (
	// ... existing imports ...
	fhir "github.com/cardiofit/ingestion-service/internal/fhir"
	kafkapkg "github.com/cardiofit/ingestion-service/internal/kafka"
	// ...
)
```

- [ ] **Step 3: Replace routes.go with real handlers**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/routes.go
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) setupRoutes() {
	// Infrastructure
	s.Router.GET("/healthz", s.handleHealthz)
	s.Router.GET("/readyz", s.handleReadyz)
	s.Router.GET("/startupz", s.handleStartupz)
	s.Router.GET("/metrics", s.prometheusHandler())

	// FHIR-compliant inbound — Phase 2 real handlers
	fhirGroup := s.Router.Group("/fhir")
	{
		fhirGroup.POST("", s.stubHandler("FHIR Transaction Bundle"))
		fhirGroup.POST("/Observation", s.handleFHIRObservation)
		fhirGroup.POST("/DiagnosticReport", s.stubHandler("FHIR DiagnosticReport"))
		fhirGroup.POST("/MedicationStatement", s.stubHandler("FHIR MedicationStatement"))

		// DLQ endpoints
		fhirGroup.GET("/OperationOutcome", s.dlqReplay.HandleListPending)
		fhirGroup.POST("/OperationOutcome/:id/$replay", s.dlqReplay.HandleReplay)
	}

	// Source-specific receivers
	ingest := s.Router.Group("/ingest")
	{
		ingest.POST("/ehr/hl7v2", s.stubHandler("HL7v2 ingest"))        // Phase 4
		ingest.POST("/ehr/fhir", s.stubHandler("FHIR passthrough"))     // Phase 4
		ingest.POST("/labs/:labId", s.stubHandler("Lab ingest"))        // Phase 4
		ingest.POST("/devices", s.handleDeviceIngest)                    // Phase 2
		ingest.POST("/app-checkin", s.handleAppCheckin)                  // Phase 2
		ingest.POST("/whatsapp", s.handleWhatsAppIngest)                // Phase 2
		ingest.POST("/wearables/:provider", s.stubHandler("Wearable ingest")) // Phase 4
		ingest.POST("/abdm/data-push", s.stubHandler("ABDM data push"))       // Phase 4
	}

	// Internal (service-to-service)
	internal := s.Router.Group("/internal")
	{
		internal.POST("/hpi", s.handleHPIIngest) // Phase 2
	}

	// Admin/Dashboard
	s.Router.GET("/$source-status", s.stubHandler("Source status"))
}

// stubHandler returns a 501 Not Implemented response with the endpoint name.
// These stubs are replaced with real handlers in later phases.
func (s *Server) stubHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"status":   "not_implemented",
			"endpoint": name,
			"message":  "This endpoint will be implemented in a future phase",
		})
	}
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go build ./cmd/ingestion/`
Expected: Binary compiles without errors

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/
git commit -m "feat(ingestion): wire real handlers replacing Phase 1 stubs

POST /fhir/Observation, POST /ingest/devices, POST /ingest/app-checkin,
POST /ingest/whatsapp, POST /internal/hpi now run the full pipeline:
adapter → normalizer → validator → FHIR mapper → FHIR Store write →
Kafka publish. DLQ list + replay endpoints wired. Server struct holds
pipeline orchestrator, Kafka producer, topic router, DLQ publisher."
```

---

## Task 12: Integration Test (End-to-End)

**Files:**
- Create: `internal/api/integration_test.go`

- [ ] **Step 1: Write integration_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/integration_test.go
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/config"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

func testConfig() *config.Config {
	cfg, _ := config.Load()
	cfg.FHIR.Enabled = false
	cfg.Kafka.Brokers = []string{""} // Disable Kafka in tests
	return cfg
}

// mockFHIRServer creates a mock Google FHIR Store that accepts creates.
func mockFHIRServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"resourceType":"Observation","id":"fhir-obs-001"}`))
		case http.MethodGet:
			if r.URL.Path == "/metadata" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"resourceType":"Bundle","total":0}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
}

func TestIntegration_PostFHIRObservation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()

	fhirSrv := mockFHIRServer(t)
	defer fhirSrv.Close()
	mockFHIR := fhirclient.NewWithHTTPClient(fhirSrv.URL, fhirSrv.Client(), logger)

	server := NewServer(cfg, nil, nil, mockFHIR, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"tenant_id":  uuid.New().String(),
		"loinc_code": "1558-6",
		"value":      142.0,
		"unit":       "mg/dL",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "accepted", resp["status"])
	assert.NotEmpty(t, resp["observation_id"])
	assert.NotEmpty(t, resp["fhir_resource_id"])
	assert.True(t, resp["quality_score"].(float64) > 0)
}

func TestIntegration_PostFHIRObservation_UnitConversion(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"loinc_code": "1558-6",
		"value":      7.0,
		"unit":       "mmol/L",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "accepted", resp["status"])
}

func TestIntegration_PostDeviceIngest(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()

	fhirSrv := mockFHIRServer(t)
	defer fhirSrv.Close()
	mockFHIR := fhirclient.NewWithHTTPClient(fhirSrv.URL, fhirSrv.Client(), logger)

	server := NewServer(cfg, nil, nil, mockFHIR, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"tenant_id":  uuid.New().String(),
		"timestamp":  "2026-03-21T08:00:00Z",
		"device": map[string]interface{}{
			"device_id":    "bp-001",
			"device_type":  "blood_pressure_monitor",
			"manufacturer": "Omron",
			"model":        "HEM-7120",
		},
		"readings": []map[string]interface{}{
			{"analyte": "systolic_bp", "value": 135.0, "unit": "mmHg"},
			{"analyte": "diastolic_bp", "value": 88.0, "unit": "mmHg"},
			{"analyte": "heart_rate", "value": 74.0, "unit": "bpm"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ingest/devices", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "accepted", resp["status"])
	assert.Equal(t, float64(3), resp["processed"])
}

func TestIntegration_PostAppCheckin(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"tenant_id":  uuid.New().String(),
		"timestamp":  "2026-03-21T08:00:00Z",
		"readings": []map[string]interface{}{
			{"analyte": "fasting_glucose", "value": 142.0, "unit": "mg/dL"},
			{"analyte": "weight", "value": 72.5, "unit": "kg"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ingest/app-checkin", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "accepted", resp["status"])
	assert.Equal(t, float64(2), resp["processed"])
}

func TestIntegration_PostHPIIngest(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	observations := []map[string]interface{}{
		{
			"id":               uuid.New().String(),
			"patient_id":       uuid.New().String(),
			"tenant_id":        uuid.New().String(),
			"source_type":      "HPI",
			"observation_type": "HPI",
			"loinc_code":       "1558-6",
			"value":            180.0,
			"unit":             "mg/dL",
			"timestamp":        "2026-03-21T10:00:00Z",
		},
	}
	bodyBytes, _ := json.Marshal(observations)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/internal/hpi", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestIntegration_DLQListEmpty(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/fhir/OperationOutcome?category=dlq", nil)
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Bundle", resp["resourceType"])
	assert.Equal(t, float64(0), resp["total"])
}

func TestIntegration_InvalidObservationGoesToDLQ(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	// Missing patient_id → should fail validation → DLQ
	body := map[string]interface{}{
		"loinc_code": "1558-6",
		"value":      100.0,
		"unit":       "mg/dL",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	// Should get 400 (bad request due to invalid patient_id parse)
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusUnprocessableEntity,
		"expected 400 or 422, got %d", w.Code)
}

func TestIntegration_CriticalValueFlagged(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"loinc_code": "2823-3", // Potassium
		"value":      6.8,      // K+ >= 6.0 = critical
		"unit":       "mEq/L",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	flags, ok := resp["flags"].([]interface{})
	require.True(t, ok, "flags should be an array")
	flagStrs := make([]string, len(flags))
	for i, f := range flags {
		flagStrs[i] = f.(string)
	}
	assert.Contains(t, flagStrs, "CRITICAL_VALUE")
}

func TestIntegration_HealthEndpoints(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	endpoints := []string{"/healthz", "/startupz"}
	for _, ep := range endpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, ep, nil)
		server.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "endpoint %s should return 200", ep)
	}
}

func TestIntegration_StubEndpointsReturn501(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	stubs := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/fhir"},
		{http.MethodPost, "/fhir/DiagnosticReport"},
		{http.MethodPost, "/ingest/ehr/hl7v2"},
		{http.MethodPost, "/ingest/ehr/fhir"},
		{http.MethodPost, "/ingest/labs/thyrocare"},
		{http.MethodPost, "/ingest/abdm/data-push"},
	}

	for _, s := range stubs {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(s.method, s.path, bytes.NewReader([]byte("{}")))
		server.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code,
			"stub %s %s should return 501", s.method, s.path)
	}
}
```

- [ ] **Step 2: Run integration tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/api/... -v -count=1 -run TestIntegration`
Expected: All integration tests PASS

- [ ] **Step 3: Run full test suite**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./... -v -count=1`
Expected output (summary):
```
ok  github.com/cardiofit/ingestion-service/internal/coding          (tests: ~20)
ok  github.com/cardiofit/ingestion-service/internal/pipeline         (tests: ~24)
ok  github.com/cardiofit/ingestion-service/internal/fhir             (tests: ~9)
ok  github.com/cardiofit/ingestion-service/internal/kafka            (tests: ~9)
ok  github.com/cardiofit/ingestion-service/internal/dlq              (tests: ~6)
ok  github.com/cardiofit/ingestion-service/internal/adapters/patient_reported (tests: ~5)
ok  github.com/cardiofit/ingestion-service/internal/adapters/devices (tests: ~5)
ok  github.com/cardiofit/ingestion-service/internal/api              (tests: ~10)
```

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/integration_test.go
git commit -m "test(ingestion): add end-to-end integration tests

10 integration tests covering: FHIR Observation create, unit conversion
end-to-end, device ingest, app checkin, HPI ingest, DLQ listing, invalid
observation → DLQ, critical value flagging, health endpoints, stub 501s.
Uses httptest mock FHIR Store."
```

- [ ] **Step 5: Final compilation check**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go mod tidy && go build ./cmd/ingestion/`
Expected: Binary compiles successfully

- [ ] **Step 6: Final commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/
git commit -m "feat(ingestion): complete Phase 2 — Ingestion Core pipeline

Full ingestion pipeline: adapters (app checkin, WhatsApp NLU, BLE device)
→ normalizer (unit conversion, LOINC mapping, staleness check)
→ validator (25 clinical ranges, quality scoring 0.0-1.0)
→ FHIR mapper (Observation, DiagnosticReport, MedicationStatement — ABDM IG v7.0)
→ router (8 ingestion.* Kafka topics, patientId partition key)
→ DLQ publisher (PostgreSQL + replay endpoint)
→ 10 Prometheus metrics.

All stub handlers from Phase 1 replaced for Phase 2 scope endpoints."
```
