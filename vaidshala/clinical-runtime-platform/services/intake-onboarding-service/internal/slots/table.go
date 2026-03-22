// Package slots defines the 50-slot intake table and event-sourced storage.
package slots

// DataType represents the data type of a slot value.
type DataType string

const (
	DataTypeNumeric     DataType = "numeric"
	DataTypeBoolean     DataType = "boolean"
	DataTypeCodedChoice DataType = "coded_choice"
	DataTypeText        DataType = "text"
	DataTypeDate        DataType = "date"
	DataTypeInteger     DataType = "integer"
	DataTypeList        DataType = "list"
)

// SlotDefinition describes a single intake slot.
type SlotDefinition struct {
	Name      string   `json:"name"`
	Domain    string   `json:"domain"`
	LOINCCode string   `json:"loinc_code"`
	DataType  DataType `json:"data_type"`
	Required  bool     `json:"required"`
	Unit      string   `json:"unit,omitempty"`
	Label     string   `json:"label"`
}

// slotTable holds the canonical 50-slot intake definition.
// Organized by domain: demographics (8), glycemic (7), renal (5), cardiac (7),
// lipid (5), medications (5), lifestyle (7), symptoms (6).
var slotTable = []SlotDefinition{
	// -- Demographics (8 slots) --
	{Name: "age", Domain: "demographics", LOINCCode: "30525-0", DataType: DataTypeInteger, Required: true, Unit: "years", Label: "Age"},
	{Name: "sex", Domain: "demographics", LOINCCode: "76689-9", DataType: DataTypeCodedChoice, Required: true, Label: "Biological sex"},
	{Name: "height", Domain: "demographics", LOINCCode: "8302-2", DataType: DataTypeNumeric, Required: true, Unit: "cm", Label: "Height"},
	{Name: "weight", Domain: "demographics", LOINCCode: "29463-7", DataType: DataTypeNumeric, Required: true, Unit: "kg", Label: "Weight"},
	{Name: "bmi", Domain: "demographics", LOINCCode: "39156-5", DataType: DataTypeNumeric, Required: true, Unit: "kg/m2", Label: "BMI"},
	{Name: "pregnant", Domain: "demographics", LOINCCode: "82810-3", DataType: DataTypeBoolean, Required: true, Label: "Currently pregnant"},
	{Name: "ethnicity", Domain: "demographics", LOINCCode: "69490-1", DataType: DataTypeCodedChoice, Required: false, Label: "Ethnicity"},
	{Name: "primary_language", Domain: "demographics", LOINCCode: "54899-0", DataType: DataTypeCodedChoice, Required: false, Label: "Primary language"},

	// -- Glycemic (7 slots) --
	{Name: "diabetes_type", Domain: "glycemic", LOINCCode: "44877-9", DataType: DataTypeCodedChoice, Required: true, Label: "Diabetes type"},
	{Name: "fbg", Domain: "glycemic", LOINCCode: "1558-6", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "Fasting blood glucose"},
	{Name: "hba1c", Domain: "glycemic", LOINCCode: "4548-4", DataType: DataTypeNumeric, Required: true, Unit: "%", Label: "HbA1c"},
	{Name: "ppbg", Domain: "glycemic", LOINCCode: "1521-4", DataType: DataTypeNumeric, Required: false, Unit: "mg/dL", Label: "Post-prandial blood glucose"},
	{Name: "diabetes_duration_years", Domain: "glycemic", LOINCCode: "66519-0", DataType: DataTypeInteger, Required: false, Unit: "years", Label: "Diabetes duration"},
	{Name: "insulin", Domain: "glycemic", LOINCCode: "46239-0", DataType: DataTypeBoolean, Required: true, Label: "Currently on insulin"},
	{Name: "hypoglycemia_episodes", Domain: "glycemic", LOINCCode: "55399-0", DataType: DataTypeInteger, Required: false, Label: "Hypoglycemia episodes (past 3 months)"},

	// -- Renal (5 slots) --
	{Name: "egfr", Domain: "renal", LOINCCode: "33914-3", DataType: DataTypeNumeric, Required: true, Unit: "mL/min/1.73m2", Label: "eGFR"},
	{Name: "serum_creatinine", Domain: "renal", LOINCCode: "2160-0", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "Serum creatinine"},
	{Name: "uacr", Domain: "renal", LOINCCode: "9318-7", DataType: DataTypeNumeric, Required: false, Unit: "mg/g", Label: "Urine albumin-to-creatinine ratio"},
	{Name: "dialysis", Domain: "renal", LOINCCode: "67038-0", DataType: DataTypeBoolean, Required: true, Label: "Currently on dialysis"},
	{Name: "serum_potassium", Domain: "renal", LOINCCode: "2823-3", DataType: DataTypeNumeric, Required: false, Unit: "mEq/L", Label: "Serum potassium"},

	// -- Cardiac (7 slots) --
	{Name: "systolic_bp", Domain: "cardiac", LOINCCode: "8480-6", DataType: DataTypeNumeric, Required: true, Unit: "mmHg", Label: "Systolic blood pressure"},
	{Name: "diastolic_bp", Domain: "cardiac", LOINCCode: "8462-4", DataType: DataTypeNumeric, Required: true, Unit: "mmHg", Label: "Diastolic blood pressure"},
	{Name: "heart_rate", Domain: "cardiac", LOINCCode: "8867-4", DataType: DataTypeNumeric, Required: false, Unit: "bpm", Label: "Resting heart rate"},
	{Name: "nyha_class", Domain: "cardiac", LOINCCode: "88020-3", DataType: DataTypeInteger, Required: false, Label: "NYHA functional class (1-4)"},
	{Name: "mi_stroke_days", Domain: "cardiac", LOINCCode: "67530-6", DataType: DataTypeInteger, Required: false, Unit: "days", Label: "Days since last MI or stroke"},
	{Name: "lvef", Domain: "cardiac", LOINCCode: "10230-1", DataType: DataTypeNumeric, Required: false, Unit: "%", Label: "Left ventricular ejection fraction"},
	{Name: "atrial_fibrillation", Domain: "cardiac", LOINCCode: "44667-4", DataType: DataTypeBoolean, Required: false, Label: "Atrial fibrillation"},

	// -- Lipid (5 slots) --
	{Name: "total_cholesterol", Domain: "lipid", LOINCCode: "2093-3", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "Total cholesterol"},
	{Name: "ldl", Domain: "lipid", LOINCCode: "2089-1", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "LDL cholesterol"},
	{Name: "hdl", Domain: "lipid", LOINCCode: "2085-9", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "HDL cholesterol"},
	{Name: "triglycerides", Domain: "lipid", LOINCCode: "2571-8", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "Triglycerides"},
	{Name: "on_statin", Domain: "lipid", LOINCCode: "82667-7", DataType: DataTypeBoolean, Required: false, Label: "Currently on statin"},

	// -- Medications (5 slots) --
	{Name: "current_medications", Domain: "medications", LOINCCode: "10160-0", DataType: DataTypeList, Required: true, Label: "Current medications list"},
	{Name: "medication_count", Domain: "medications", LOINCCode: "82670-1", DataType: DataTypeInteger, Required: true, Label: "Total medication count"},
	{Name: "adherence_score", Domain: "medications", LOINCCode: "71950-0", DataType: DataTypeNumeric, Required: false, Label: "Medication adherence score (0.0-1.0)"},
	{Name: "allergies", Domain: "medications", LOINCCode: "52473-6", DataType: DataTypeList, Required: true, Label: "Known allergies"},
	{Name: "supplement_list", Domain: "medications", LOINCCode: "29549-3", DataType: DataTypeList, Required: false, Label: "Current supplements"},

	// -- Lifestyle (7 slots) --
	{Name: "smoking_status", Domain: "lifestyle", LOINCCode: "72166-2", DataType: DataTypeCodedChoice, Required: true, Label: "Smoking status"},
	{Name: "alcohol_use", Domain: "lifestyle", LOINCCode: "74013-4", DataType: DataTypeCodedChoice, Required: true, Label: "Alcohol use frequency"},
	{Name: "exercise_minutes_week", Domain: "lifestyle", LOINCCode: "68516-4", DataType: DataTypeInteger, Required: false, Unit: "min/week", Label: "Exercise minutes per week"},
	{Name: "diet_type", Domain: "lifestyle", LOINCCode: "81663-7", DataType: DataTypeCodedChoice, Required: false, Label: "Diet type"},
	{Name: "sleep_hours", Domain: "lifestyle", LOINCCode: "93832-4", DataType: DataTypeNumeric, Required: false, Unit: "hours", Label: "Average sleep hours"},
	{Name: "active_substance_abuse", Domain: "lifestyle", LOINCCode: "68524-8", DataType: DataTypeBoolean, Required: true, Label: "Active substance abuse"},
	{Name: "falls_history", Domain: "lifestyle", LOINCCode: "52552-7", DataType: DataTypeBoolean, Required: false, Label: "Falls history (past 12 months)"},

	// -- Symptoms (6 slots) --
	{Name: "active_cancer", Domain: "symptoms", LOINCCode: "63933-6", DataType: DataTypeBoolean, Required: true, Label: "Active cancer"},
	{Name: "organ_transplant", Domain: "symptoms", LOINCCode: "79829-6", DataType: DataTypeBoolean, Required: true, Label: "Organ transplant recipient"},
	{Name: "cognitive_impairment", Domain: "symptoms", LOINCCode: "72106-8", DataType: DataTypeBoolean, Required: false, Label: "Cognitive impairment"},
	{Name: "bariatric_surgery_months", Domain: "symptoms", LOINCCode: "85359-8", DataType: DataTypeInteger, Required: false, Unit: "months", Label: "Months since bariatric surgery"},
	{Name: "primary_complaint", Domain: "symptoms", LOINCCode: "10164-2", DataType: DataTypeText, Required: false, Label: "Primary complaint (free text)"},
	{Name: "comorbidities", Domain: "symptoms", LOINCCode: "45701-0", DataType: DataTypeList, Required: false, Label: "Comorbidity list"},
}

// slotIndex is a name-to-slot lookup map, built at init time.
var slotIndex map[string]SlotDefinition

func init() {
	slotIndex = make(map[string]SlotDefinition, len(slotTable))
	for _, s := range slotTable {
		slotIndex[s.Name] = s
	}
}

// AllSlots returns the full 50-slot intake definition table.
func AllSlots() []SlotDefinition {
	out := make([]SlotDefinition, len(slotTable))
	copy(out, slotTable)
	return out
}

// LookupSlot returns the slot definition by name.
func LookupSlot(name string) (SlotDefinition, bool) {
	s, ok := slotIndex[name]
	return s, ok
}

// SlotsByDomain returns all slots for a given domain.
func SlotsByDomain(domain string) []SlotDefinition {
	var out []SlotDefinition
	for _, s := range slotTable {
		if s.Domain == domain {
			out = append(out, s)
		}
	}
	return out
}

// RequiredSlots returns all slots with Required=true.
func RequiredSlots() []SlotDefinition {
	var out []SlotDefinition
	for _, s := range slotTable {
		if s.Required {
			out = append(out, s)
		}
	}
	return out
}
