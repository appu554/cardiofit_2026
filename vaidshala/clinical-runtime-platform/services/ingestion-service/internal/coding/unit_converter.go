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
	// Glucose: mmol/L <-> mg/dL (factor 18.0)
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "glucose"}:  func(v float64) float64 { return v * 18.0 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "glucose"}:  func(v float64) float64 { return v / 18.0 },

	// Blood pressure: kPa <-> mmHg (factor 7.50062)
	{FromUnit: "kPa", ToUnit: "mmHg", Analyte: "blood_pressure"}: func(v float64) float64 { return v * 7.50062 },
	{FromUnit: "mmHg", ToUnit: "kPa", Analyte: "blood_pressure"}: func(v float64) float64 { return v / 7.50062 },

	// Temperature: degF <-> degC
	{FromUnit: "degF", ToUnit: "degC", Analyte: "temperature"}: func(v float64) float64 { return (v - 32.0) * 5.0 / 9.0 },
	{FromUnit: "degC", ToUnit: "degF", Analyte: "temperature"}: func(v float64) float64 { return v*9.0/5.0 + 32.0 },

	// Cholesterol: mmol/L <-> mg/dL (factor 38.67)
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "cholesterol"}: func(v float64) float64 { return v * 38.67 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "cholesterol"}: func(v float64) float64 { return v / 38.67 },

	// HDL: same factor as total cholesterol
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "hdl"}: func(v float64) float64 { return v * 38.67 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "hdl"}: func(v float64) float64 { return v / 38.67 },

	// LDL: same factor as total cholesterol
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "ldl"}: func(v float64) float64 { return v * 38.67 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "ldl"}: func(v float64) float64 { return v / 38.67 },

	// Triglycerides: mmol/L <-> mg/dL (factor 88.57)
	{FromUnit: "mmol/L", ToUnit: "mg/dL", Analyte: "triglycerides"}: func(v float64) float64 { return v * 88.57 },
	{FromUnit: "mg/dL", ToUnit: "mmol/L", Analyte: "triglycerides"}: func(v float64) float64 { return v / 88.57 },

	// Creatinine: umol/L <-> mg/dL (factor 88.4)
	{FromUnit: "umol/L", ToUnit: "mg/dL", Analyte: "creatinine"}: func(v float64) float64 { return v / 88.4 },
	{FromUnit: "mg/dL", ToUnit: "umol/L", Analyte: "creatinine"}: func(v float64) float64 { return v * 88.4 },

	// HbA1c: mmol/mol <-> % (NGSP/DCCT) using IFCC formula
	{FromUnit: "mmol/mol", ToUnit: "%", Analyte: "hba1c"}: func(v float64) float64 { return (v / 10.929) + 2.15 },
	{FromUnit: "%", ToUnit: "mmol/mol", Analyte: "hba1c"}: func(v float64) float64 { return (v - 2.15) * 10.929 },

	// Weight: lbs <-> kg
	{FromUnit: "lbs", ToUnit: "kg", Analyte: "weight"}: func(v float64) float64 { return v / 2.20462 },
	{FromUnit: "kg", ToUnit: "lbs", Analyte: "weight"}: func(v float64) float64 { return v * 2.20462 },

	// Height: in <-> cm
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
		return 0, fmt.Errorf("unsupported conversion: %s -> %s for analyte %q", fromUnit, toUnit, analyte)
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
		// Unknown analyte -- pass through without conversion
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
