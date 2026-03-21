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
	"8480-6":  {Code: "8480-6", Display: "Systolic blood pressure", Analyte: "blood_pressure", StdUnit: "mmHg", Category: "VITALS"},
	"8462-4":  {Code: "8462-4", Display: "Diastolic blood pressure", Analyte: "blood_pressure", StdUnit: "mmHg", Category: "VITALS"},
	"8867-4":  {Code: "8867-4", Display: "Heart rate", Analyte: "heart_rate", StdUnit: "bpm", Category: "VITALS"},
	"2708-6":  {Code: "2708-6", Display: "Oxygen saturation in Arterial blood", Analyte: "spo2", StdUnit: "%", Category: "VITALS"},
	"8310-5":  {Code: "8310-5", Display: "Body temperature", Analyte: "temperature", StdUnit: "degC", Category: "VITALS"},
	"29463-7": {Code: "29463-7", Display: "Body weight", Analyte: "weight", StdUnit: "kg", Category: "VITALS"},
	"8302-2":  {Code: "8302-2", Display: "Body height", Analyte: "height", StdUnit: "cm", Category: "VITALS"},
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

// analyteToLOINC maps analyte names to their primary LOINC code.
// This is used when adapters provide analyte names without LOINC codes.
var analyteToLOINC = map[string]string{
	"glucose":         "1558-6",
	"fasting_glucose": "1558-6",
	"random_glucose":  "2345-7",
	"hba1c":           "4548-4",
	"egfr":            "33914-3",
	"creatinine":      "2160-0",
	"bun":             "6299-2",
	"potassium":       "2823-3",
	"sodium":          "2951-2",
	"cholesterol":     "2093-3",
	"hdl":             "2085-9",
	"ldl":             "2089-1",
	"triglycerides":   "2571-8",
	"systolic_bp":     "8480-6",
	"diastolic_bp":    "8462-4",
	"heart_rate":      "8867-4",
	"spo2":            "2708-6",
	"temperature":     "8310-5",
	"weight":          "29463-7",
	"height":          "8302-2",
	"bmi":             "39156-5",
	"tsh":             "3016-3",
	"ft4":             "3024-7",
	"alt":             "1742-6",
	"ast":             "1920-8",
	"hemoglobin":      "718-7",
	"uric_acid":       "3084-1",
	"urine_albumin":   "14959-1",
}

// LookupLOINCByAnalyte returns the primary LOINC code for an analyte name.
func LookupLOINCByAnalyte(analyte string) (string, bool) {
	code, ok := analyteToLOINC[analyte]
	return code, ok
}
