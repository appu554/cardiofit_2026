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
	"vital_signs":    {Code: "vital-signs", Display: "Vital Signs"},
	"laboratory":     {Code: "laboratory", Display: "Laboratory"},
	"survey":         {Code: "survey", Display: "Survey"},
	"activity":       {Code: "activity", Display: "Activity"},
	"social_history": {Code: "social-history", Display: "Social History"},

	// Body sites
	"left_arm":    {Code: "368208006", Display: "Left upper arm structure"},
	"right_arm":   {Code: "368209003", Display: "Right upper arm structure"},
	"left_wrist":  {Code: "5951000", Display: "Structure of left wrist"},
	"right_wrist": {Code: "9736006", Display: "Structure of right wrist"},
	"finger":      {Code: "7569003", Display: "Finger structure"},

	// Methods
	"automated":     {Code: "17146006", Display: "Automated measurement"},
	"manual":        {Code: "258104002", Display: "Manual measurement"},
	"self_reported": {Code: "self-reported", Display: "Patient self-reported"},

	// Condition codes
	"diabetes_mellitus_2": {Code: "44054006", Display: "Diabetes mellitus type 2"},
	"hypertension":        {Code: "38341003", Display: "Hypertensive disorder"},
	"ckd":                 {Code: "709044004", Display: "Chronic kidney disease"},
	"dyslipidemia":        {Code: "55822004", Display: "Hyperlipidemia"},
}

// LookupSNOMED returns the SNOMEDEntry for a given key, or false if not found.
func LookupSNOMED(key string) (SNOMEDEntry, bool) {
	entry, ok := snomedRegistry[key]
	return entry, ok
}
