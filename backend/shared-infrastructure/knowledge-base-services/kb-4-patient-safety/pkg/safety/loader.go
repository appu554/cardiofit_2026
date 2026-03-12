// Package safety provides the knowledge loader for governed clinical safety data
package safety

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// =============================================================================
// KNOWLEDGE LOADER
// =============================================================================
// The KnowledgeLoader is responsible for loading clinically governed safety
// knowledge from YAML files at runtime. This separates clinical knowledge from
// code, enabling:
// - Version-controlled knowledge updates
// - Clinical governance workflows
// - Jurisdiction-specific rules
// - Audit trails and provenance tracking

// KnowledgeStore holds all loaded clinical safety knowledge
type KnowledgeStore struct {
	mu sync.RWMutex

	// Drug Safety Knowledge
	BlackBoxWarnings       map[string]BlackBoxWarning       // keyed by RxNormCode
	Contraindications      map[string][]Contraindication    // keyed by RxNormCode (can have multiple)
	DoseLimits             map[string]DoseLimit             // keyed by RxNormCode
	AgeLimits              map[string]AgeLimit              // keyed by RxNormCode
	PregnancySafety        map[string]PregnancySafety       // keyed by RxNormCode
	LactationSafety        map[string]LactationSafety       // keyed by RxNormCode
	HighAlertMedications   map[string]HighAlertMedication   // keyed by RxNormCode
	BeersEntries           map[string]BeersEntry            // keyed by RxNormCode
	AnticholinergicBurdens map[string]AnticholinergicBurden // keyed by RxNormCode
	LabRequirements        map[string]LabRequirement        // keyed by RxNormCode

	// STOPP/START Criteria (European Geriatric Prescribing)
	StoppEntries map[string]StoppEntry // keyed by criterion ID (e.g., "A1", "B2")
	StartEntries map[string]StartEntry // keyed by criterion ID (e.g., "A1", "B2")

	// India-Specific Knowledge (CDSCO/NLEM)
	BannedCombinations map[string]BannedCombinationEntry // keyed by combination ID (e.g., "BAN-ANA-001")
	NLEMMedications    map[string]NLEMMedication         // keyed by RxNormCode

	// Cross-Reference Indices
	DrugNameToRxNorm map[string]string // Drug name -> RxNorm code
	ATCToRxNorm      map[string]string // ATC code -> RxNorm code

	// Metadata
	LoadedAt         string
	KnowledgeVersion string
	TotalEntries     int
	LoadErrors       []string
}

// BlackBoxWarningsFile represents the YAML file structure for black box warnings
type BlackBoxWarningsFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries []BlackBoxWarning `yaml:"entries"`
}

// HighAlertMedicationsFile represents the YAML file structure for high-alert medications
type HighAlertMedicationsFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries []HighAlertMedication `yaml:"entries"`
}

// BeersEntriesFile represents the YAML file structure for Beers Criteria
type BeersEntriesFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries      []BeersEntry `yaml:"entries"`
	BeersEntries []BeersEntry `yaml:"beers_entries"` // Alternative key name used in some YAML files
}

// GetEntries returns entries from either format
func (f *BeersEntriesFile) GetEntries() []BeersEntry {
	if len(f.Entries) > 0 {
		return f.Entries
	}
	return f.BeersEntries
}

// PregnancySafetyFile represents the YAML file structure for pregnancy safety
type PregnancySafetyFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries          []PregnancySafety `yaml:"entries"`
	PregnancyEntries []PregnancySafety `yaml:"pregnancy_entries"` // Alternative key name
}

// GetEntries returns entries from either format
func (f *PregnancySafetyFile) GetEntries() []PregnancySafety {
	if len(f.Entries) > 0 {
		return f.Entries
	}
	return f.PregnancyEntries
}

// LactationSafetyFile represents the YAML file structure for lactation safety
type LactationSafetyFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries          []LactationSafety `yaml:"entries"`
	LactationEntries []LactationSafety `yaml:"lactation_entries"` // Alternative key name
}

// GetEntries returns entries from either format
func (f *LactationSafetyFile) GetEntries() []LactationSafety {
	if len(f.Entries) > 0 {
		return f.Entries
	}
	return f.LactationEntries
}

// LabRequirementsFile represents the YAML file structure for lab monitoring
type LabRequirementsFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries              []LabRequirement `yaml:"entries"`
	LabMonitoringEntries []LabRequirement `yaml:"lab_monitoring_entries"` // Alternative key name
}

// GetEntries returns entries from either format
func (f *LabRequirementsFile) GetEntries() []LabRequirement {
	if len(f.Entries) > 0 {
		return f.Entries
	}
	return f.LabMonitoringEntries
}

// AnticholinergicBurdensFile represents the YAML file structure for ACB scores
type AnticholinergicBurdensFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries []AnticholinergicBurden `yaml:"entries"`
}

// ContraindicationsFile represents the YAML file structure for contraindications
type ContraindicationsFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries []Contraindication `yaml:"entries"`
}

// DoseLimitsFile represents the YAML file structure for dose limits
type DoseLimitsFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries    []DoseLimit `yaml:"entries"`
	DoseLimits []DoseLimit `yaml:"dose_limits"` // Alternative key name
}

// GetEntries returns entries from either format
func (f *DoseLimitsFile) GetEntries() []DoseLimit {
	if len(f.Entries) > 0 {
		return f.Entries
	}
	return f.DoseLimits
}

// AgeLimitsFile represents the YAML file structure for age limits
type AgeLimitsFile struct {
	Metadata struct {
		KnowledgeType     string `yaml:"knowledgeType"`
		KnowledgeVersion  string `yaml:"knowledgeVersion"`
		PrimaryAuthority  string `yaml:"primaryAuthority"`
		Jurisdiction      string `yaml:"jurisdiction"`
		EffectiveDate     string `yaml:"effectiveDate"`
		ReviewDate        string `yaml:"reviewDate"`
		TotalEntries      int    `yaml:"totalEntries"`
		Description       string `yaml:"description"`
	} `yaml:"metadata"`
	Entries   []AgeLimit `yaml:"entries"`
	AgeLimits []AgeLimit `yaml:"age_limits"` // Alternative key name
}

// GetEntries returns entries from either format
func (f *AgeLimitsFile) GetEntries() []AgeLimit {
	if len(f.Entries) > 0 {
		return f.Entries
	}
	return f.AgeLimits
}

// StoppEntriesFile represents the YAML file structure for STOPP criteria
// Source: O'Mahony D, et al. STOPP/START v3. Age and Ageing, 2023
type StoppEntriesFile struct {
	Metadata struct {
		Version           string   `yaml:"version"`
		LastUpdated       string   `yaml:"lastUpdated"`
		Source            string   `yaml:"source"`
		SourceYear        int      `yaml:"sourceYear"`
		Publication       string   `yaml:"publication"`
		DOI               string   `yaml:"doi"`
		Jurisdiction      string   `yaml:"jurisdiction"`
		KnowledgeVersion  string   `yaml:"knowledgeVersion"`
		Applicability     string   `yaml:"applicability"`
		Regions           []string `yaml:"regions"`
	} `yaml:"metadata"`
	Summary struct {
		TotalEntries int            `yaml:"totalEntries"`
		BySection    map[string]int `yaml:"bySection"`
	} `yaml:"summary"`
	Entries      []StoppEntry `yaml:"entries"`
	StoppEntries []StoppEntry `yaml:"stopp_entries"` // Alternative key name
}

// GetEntries returns entries from either format
func (f *StoppEntriesFile) GetEntries() []StoppEntry {
	if len(f.Entries) > 0 {
		return f.Entries
	}
	return f.StoppEntries
}

// StartEntriesFile represents the YAML file structure for START criteria
// Source: O'Mahony D, et al. STOPP/START v3. Age and Ageing, 2023
type StartEntriesFile struct {
	Metadata struct {
		Version           string   `yaml:"version"`
		LastUpdated       string   `yaml:"lastUpdated"`
		Source            string   `yaml:"source"`
		SourceYear        int      `yaml:"sourceYear"`
		Publication       string   `yaml:"publication"`
		DOI               string   `yaml:"doi"`
		Jurisdiction      string   `yaml:"jurisdiction"`
		KnowledgeVersion  string   `yaml:"knowledgeVersion"`
		Applicability     string   `yaml:"applicability"`
		Regions           []string `yaml:"regions"`
	} `yaml:"metadata"`
	Summary struct {
		TotalEntries int            `yaml:"totalEntries"`
		BySection    map[string]int `yaml:"bySection"`
	} `yaml:"summary"`
	Entries      []StartEntry `yaml:"entries"`
	StartEntries []StartEntry `yaml:"start_entries"` // Alternative key name
}

// GetEntries returns entries from either format
func (f *StartEntriesFile) GetEntries() []StartEntry {
	if len(f.Entries) > 0 {
		return f.Entries
	}
	return f.StartEntries
}

// =============================================================================
// INDIA-SPECIFIC FILE TYPES (CDSCO, NLEM)
// =============================================================================

// BannedCombinationsFile represents the YAML file structure for CDSCO banned FDCs
// India banned 344+ fixed-dose combinations in 2016 under Drugs and Cosmetics Act
type BannedCombinationsFile struct {
	Metadata struct {
		KnowledgeType        string   `yaml:"knowledgeType"`
		KnowledgeVersion     string   `yaml:"knowledgeVersion"`
		PrimaryAuthority     string   `yaml:"primaryAuthority"`
		Jurisdiction         string   `yaml:"jurisdiction"`
		EffectiveDate        string   `yaml:"effectiveDate"`
		ReviewDate           string   `yaml:"reviewDate"`
		TotalEntries         int      `yaml:"totalEntries"`
		Description          string   `yaml:"description"`
		RegulatoryBasis      string   `yaml:"regulatoryBasis"`
		GazetteNotifications []string `yaml:"gazetteNotifications"`
	} `yaml:"metadata"`
	Entries            []BannedCombinationEntry `yaml:"entries"`
	BannedCombinations []BannedCombinationEntry `yaml:"banned_combinations"` // Alternative key
}

// GetEntries returns entries from either format
func (f *BannedCombinationsFile) GetEntries() []BannedCombinationEntry {
	if len(f.Entries) > 0 {
		return f.Entries
	}
	return f.BannedCombinations
}

// NLEMFile represents the YAML file structure for India's National List of Essential Medicines
// NLEM 2022 has nested therapeutic category sections with essential level classification
type NLEMFile struct {
	Metadata struct {
		Version         string `yaml:"version"`
		EffectiveDate   string `yaml:"effectiveDate"`
		TotalMedicines  int    `yaml:"totalMedicines"`
		SourceAuthority string `yaml:"sourceAuthority"`
		SourceDocument  string `yaml:"sourceDocument"`
		SourceUrl       string `yaml:"sourceUrl"`
		Jurisdiction    string `yaml:"jurisdiction"`
	} `yaml:"metadata"`
	// Therapeutic category sections - each contains nested subcategory arrays
	Anaesthetics      NLEMSection `yaml:"anaesthetics"`
	Analgesics        NLEMSection `yaml:"analgesics"`
	Antiallergics     NLEMSection `yaml:"antiallergics"`
	Antiepileptics    NLEMSection `yaml:"antiepileptics"`
	Antibiotics       NLEMSection `yaml:"antibiotics"`
	Antitubercular    NLEMSection `yaml:"antitubercular"`
	Antifungals       NLEMSection `yaml:"antifungals"`
	Antivirals        NLEMSection `yaml:"antivirals"`
	Antimalarials     NLEMSection `yaml:"antimalarials"`
	Cardiovascular    NLEMSection `yaml:"cardiovascular"`
	Antidiabetics     NLEMSection `yaml:"antidiabetics"`
	Gastrointestinal  NLEMSection `yaml:"gastrointestinal"`
	Respiratory       NLEMSection `yaml:"respiratory"`
	Psychotropics     NLEMSection `yaml:"psychotropics"`
	Thyroid           NLEMSection `yaml:"thyroid"`
	VitaminsMinerals  NLEMSection `yaml:"vitamins_minerals"`
	Antidotes         NLEMSection `yaml:"antidotes"`
	Antiparasitics    NLEMSection `yaml:"antiparasitics"`
	Dermatological    NLEMSection `yaml:"dermatological"`
	Ophthalmological  NLEMSection `yaml:"ophthalmological"`
	Oxytocics         NLEMSection `yaml:"oxytocics"`
	IVFluids          NLEMSection `yaml:"iv_fluids"`
}

// NLEMSection represents a therapeutic category with nested subcategories
type NLEMSection struct {
	// Each field represents a subcategory containing medication entries
	GeneralAnaesthetics  []NLEMMedication `yaml:"general_anaesthetics"`
	LocalAnaesthetics    []NLEMMedication `yaml:"local_anaesthetics"`
	Opioids              []NLEMMedication `yaml:"opioids"`
	NonOpioids           []NLEMMedication `yaml:"non_opioids"`
	Antihistamines       []NLEMMedication `yaml:"antihistamines"`
	Penicillins          []NLEMMedication `yaml:"penicillins"`
	Cephalosporins       []NLEMMedication `yaml:"cephalosporins"`
	Macrolides           []NLEMMedication `yaml:"macrolides"`
	Aminoglycosides      []NLEMMedication `yaml:"aminoglycosides"`
	Fluoroquinolones     []NLEMMedication `yaml:"fluoroquinolones"`
	Carbapenems          []NLEMMedication `yaml:"carbapenems"`
	Glycopeptides        []NLEMMedication `yaml:"glycopeptides"`
	Tetracyclines        []NLEMMedication `yaml:"tetracyclines"`
	Lincosamides         []NLEMMedication `yaml:"lincosamides"`
	Oxazolidinones       []NLEMMedication `yaml:"oxazolidinones"`
	Sulfonamides         []NLEMMedication `yaml:"sulfonamides"`
	Nitroimidazoles      []NLEMMedication `yaml:"nitroimidazoles"`
	FirstLine            []NLEMMedication `yaml:"first_line"`
	SecondLine           []NLEMMedication `yaml:"second_line"`
	Systemic             []NLEMMedication `yaml:"systemic"`
	Topical              []NLEMMedication `yaml:"topical"`
	Antiretroviral       []NLEMMedication `yaml:"antiretroviral"`
	Hepatitis            []NLEMMedication `yaml:"hepatitis"`
	Treatment            []NLEMMedication `yaml:"treatment"`
	Prophylaxis          []NLEMMedication `yaml:"prophylaxis"`
	Antihypertensives    []NLEMMedication `yaml:"antihypertensives"`
	Antiarrhythmics      []NLEMMedication `yaml:"antiarrhythmics"`
	HeartFailure         []NLEMMedication `yaml:"heart_failure"`
	Antianginals         []NLEMMedication `yaml:"antianginals"`
	Antithrombotics      []NLEMMedication `yaml:"antithrombotics"`
	Lipidlowering        []NLEMMedication `yaml:"lipid_lowering"`
	Insulins             []NLEMMedication `yaml:"insulins"`
	OralHypoglycemics    []NLEMMedication `yaml:"oral_hypoglycemics"`
	Antacids             []NLEMMedication `yaml:"antacids"`
	Antiemetics          []NLEMMedication `yaml:"antiemetics"`
	Laxatives            []NLEMMedication `yaml:"laxatives"`
	Antidiarrheals       []NLEMMedication `yaml:"antidiarrheals"`
	Bronchodilators      []NLEMMedication `yaml:"bronchodilators"`
	Corticosteroids      []NLEMMedication `yaml:"corticosteroids"`
	Antipsychotics       []NLEMMedication `yaml:"antipsychotics"`
	Antidepressants      []NLEMMedication `yaml:"antidepressants"`
	Anxiolytics          []NLEMMedication `yaml:"anxiolytics"`
	Thyroid              []NLEMMedication `yaml:"thyroid"`
	Antithyroid          []NLEMMedication `yaml:"antithyroid"`
	Vitamins             []NLEMMedication `yaml:"vitamins"`
	Minerals             []NLEMMedication `yaml:"minerals"`
	Specific             []NLEMMedication `yaml:"specific"`
	Anthelminthics       []NLEMMedication `yaml:"anthelminthics"`
	Scabicides           []NLEMMedication `yaml:"scabicides"`
	AntifungalsDerm      []NLEMMedication `yaml:"antifungals"`
	Antibacterials       []NLEMMedication `yaml:"antibacterials"`
	AntiinflammatoryDerm []NLEMMedication `yaml:"anti_inflammatory"`
	Antiglaucoma         []NLEMMedication `yaml:"antiglaucoma"`
	AntibioticsOph       []NLEMMedication `yaml:"antibiotics"`
	Mydriatics           []NLEMMedication `yaml:"mydriatics"`
	Uterotonics          []NLEMMedication `yaml:"uterotonics"`
	Tocolytics           []NLEMMedication `yaml:"tocolytics"`
	Crystalloids         []NLEMMedication `yaml:"crystalloids"`
	Colloids             []NLEMMedication `yaml:"colloids"`
	// Additional fields for sections without subcategories
	Anticonvulsants      []NLEMMedication `yaml:"anticonvulsants"`
	Antimalarials        []NLEMMedication `yaml:"antimalarials"`
	GeneralMedications   []NLEMMedication `yaml:"general"`
}

// GetAllMedications returns all medications from all sections flattened
func (f *NLEMFile) GetAllMedications() []NLEMMedication {
	var meds []NLEMMedication
	sections := []NLEMSection{
		f.Anaesthetics, f.Analgesics, f.Antiallergics, f.Antiepileptics,
		f.Antibiotics, f.Antitubercular, f.Antifungals, f.Antivirals,
		f.Antimalarials, f.Cardiovascular, f.Antidiabetics, f.Gastrointestinal,
		f.Respiratory, f.Psychotropics, f.Thyroid, f.VitaminsMinerals,
		f.Antidotes, f.Antiparasitics, f.Dermatological, f.Ophthalmological,
		f.Oxytocics, f.IVFluids,
	}
	for _, section := range sections {
		meds = append(meds, section.getAllFromSection()...)
	}
	return meds
}

// getAllFromSection extracts all medications from a section's subcategories
func (s *NLEMSection) getAllFromSection() []NLEMMedication {
	var meds []NLEMMedication
	// Collect from all possible subcategory fields
	meds = append(meds, s.GeneralAnaesthetics...)
	meds = append(meds, s.LocalAnaesthetics...)
	meds = append(meds, s.Opioids...)
	meds = append(meds, s.NonOpioids...)
	meds = append(meds, s.Antihistamines...)
	meds = append(meds, s.Penicillins...)
	meds = append(meds, s.Cephalosporins...)
	meds = append(meds, s.Macrolides...)
	meds = append(meds, s.Aminoglycosides...)
	meds = append(meds, s.Fluoroquinolones...)
	meds = append(meds, s.Carbapenems...)
	meds = append(meds, s.Glycopeptides...)
	meds = append(meds, s.Tetracyclines...)
	meds = append(meds, s.Lincosamides...)
	meds = append(meds, s.Oxazolidinones...)
	meds = append(meds, s.Sulfonamides...)
	meds = append(meds, s.Nitroimidazoles...)
	meds = append(meds, s.FirstLine...)
	meds = append(meds, s.SecondLine...)
	meds = append(meds, s.Systemic...)
	meds = append(meds, s.Topical...)
	meds = append(meds, s.Antiretroviral...)
	meds = append(meds, s.Hepatitis...)
	meds = append(meds, s.Treatment...)
	meds = append(meds, s.Prophylaxis...)
	meds = append(meds, s.Antihypertensives...)
	meds = append(meds, s.Antiarrhythmics...)
	meds = append(meds, s.HeartFailure...)
	meds = append(meds, s.Antianginals...)
	meds = append(meds, s.Antithrombotics...)
	meds = append(meds, s.Lipidlowering...)
	meds = append(meds, s.Insulins...)
	meds = append(meds, s.OralHypoglycemics...)
	meds = append(meds, s.Antacids...)
	meds = append(meds, s.Antiemetics...)
	meds = append(meds, s.Laxatives...)
	meds = append(meds, s.Antidiarrheals...)
	meds = append(meds, s.Bronchodilators...)
	meds = append(meds, s.Corticosteroids...)
	meds = append(meds, s.Antipsychotics...)
	meds = append(meds, s.Antidepressants...)
	meds = append(meds, s.Anxiolytics...)
	meds = append(meds, s.Thyroid...)
	meds = append(meds, s.Antithyroid...)
	meds = append(meds, s.Vitamins...)
	meds = append(meds, s.Minerals...)
	meds = append(meds, s.Specific...)
	meds = append(meds, s.Anthelminthics...)
	meds = append(meds, s.Scabicides...)
	meds = append(meds, s.AntifungalsDerm...)
	meds = append(meds, s.Antibacterials...)
	meds = append(meds, s.AntiinflammatoryDerm...)
	meds = append(meds, s.Antiglaucoma...)
	meds = append(meds, s.AntibioticsOph...)
	meds = append(meds, s.Mydriatics...)
	meds = append(meds, s.Uterotonics...)
	meds = append(meds, s.Tocolytics...)
	meds = append(meds, s.Crystalloids...)
	meds = append(meds, s.Colloids...)
	// Additional fields
	meds = append(meds, s.Anticonvulsants...)
	meds = append(meds, s.Antimalarials...)
	meds = append(meds, s.GeneralMedications...)
	return meds
}

// NewKnowledgeStore creates a new empty knowledge store
func NewKnowledgeStore() *KnowledgeStore {
	return &KnowledgeStore{
		BlackBoxWarnings:       make(map[string]BlackBoxWarning),
		Contraindications:      make(map[string][]Contraindication),
		DoseLimits:             make(map[string]DoseLimit),
		AgeLimits:              make(map[string]AgeLimit),
		PregnancySafety:        make(map[string]PregnancySafety),
		LactationSafety:        make(map[string]LactationSafety),
		HighAlertMedications:   make(map[string]HighAlertMedication),
		BeersEntries:           make(map[string]BeersEntry),
		AnticholinergicBurdens: make(map[string]AnticholinergicBurden),
		LabRequirements:        make(map[string]LabRequirement),
		StoppEntries:           make(map[string]StoppEntry),
		StartEntries:           make(map[string]StartEntry),
		BannedCombinations:     make(map[string]BannedCombinationEntry),
		NLEMMedications:        make(map[string]NLEMMedication),
		DrugNameToRxNorm:       make(map[string]string),
		ATCToRxNorm:            make(map[string]string),
		LoadErrors:             []string{},
	}
}

// KnowledgeLoader handles loading clinical knowledge from YAML files
type KnowledgeLoader struct {
	basePath     string
	jurisdiction Jurisdiction
	store        *KnowledgeStore
}

// NewKnowledgeLoader creates a new knowledge loader (loads from flat directory structure)
func NewKnowledgeLoader(basePath string) *KnowledgeLoader {
	return &KnowledgeLoader{
		basePath:     basePath,
		jurisdiction: JurisdictionGlobal,
		store:        NewKnowledgeStore(),
	}
}

// NewJurisdictionAwareLoader creates a loader that loads jurisdiction-specific knowledge
// with fallback to global knowledge. Directory structure expected:
//   - basePath/us/ for US-specific knowledge (FDA, ISMP, AGS)
//   - basePath/au/ for Australia-specific knowledge (TGA, ACSQHC, PBS)
//   - basePath/in/ for India-specific knowledge (CDSCO, PvPI)
//   - basePath/global/ for universal knowledge (WHO, ICH, LactMed)
//
// Loading order: jurisdiction-specific → global → legacy flat directories
func NewJurisdictionAwareLoader(basePath string, jurisdiction Jurisdiction) *KnowledgeLoader {
	return &KnowledgeLoader{
		basePath:     basePath,
		jurisdiction: jurisdiction,
		store:        NewKnowledgeStore(),
	}
}

// getSearchPaths returns the ordered list of directories to search for a knowledge type
// Order: jurisdiction-specific → global → legacy (flat directory)
func (kl *KnowledgeLoader) getSearchPaths(knowledgeType string) []string {
	paths := []string{}

	// Map jurisdiction to directory name
	jurisdictionDir := strings.ToLower(string(kl.jurisdiction))
	if jurisdictionDir == "global" {
		jurisdictionDir = "" // Skip jurisdiction-specific for global
	}

	// 1. Jurisdiction-specific directory (e.g., us/blackbox, au/blackbox)
	if jurisdictionDir != "" {
		paths = append(paths, filepath.Join(kl.basePath, jurisdictionDir, knowledgeType))
	}

	// 2. Global directory (e.g., global/blackbox)
	paths = append(paths, filepath.Join(kl.basePath, "global", knowledgeType))

	// 3. Legacy flat directory (e.g., blackbox) for backwards compatibility
	paths = append(paths, filepath.Join(kl.basePath, knowledgeType))

	return paths
}

// LoadAll loads all knowledge files from the configured directory
func (kl *KnowledgeLoader) LoadAll() (*KnowledgeStore, error) {
	jurisdictionLabel := string(kl.jurisdiction)
	if kl.jurisdiction == "" {
		jurisdictionLabel = "GLOBAL"
	}
	log.Printf("📚 Loading clinical knowledge from: %s (jurisdiction: %s)", kl.basePath, jurisdictionLabel)

	// Check if directory exists
	if _, err := os.Stat(kl.basePath); os.IsNotExist(err) {
		log.Printf("⚠️  Knowledge directory not found: %s (using built-in data)", kl.basePath)
		return kl.store, nil
	}

	// Load each knowledge type
	var loadCount int

	// Load Black Box Warnings
	if count, err := kl.loadBlackBoxWarnings(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("BlackBox: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ Black Box Warnings: %d entries", count)
	}

	// Load High-Alert Medications
	if count, err := kl.loadHighAlertMedications(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("HighAlert: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ High-Alert Medications: %d entries", count)
	}

	// Load Beers Criteria
	if count, err := kl.loadBeersEntries(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Beers: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ Beers Criteria: %d entries", count)
	}

	// Load Pregnancy Safety
	if count, err := kl.loadPregnancySafety(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Pregnancy: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ Pregnancy Safety: %d entries", count)
	}

	// Load Lactation Safety
	if count, err := kl.loadLactationSafety(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Lactation: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ Lactation Safety: %d entries", count)
	}

	// Load Lab Requirements
	if count, err := kl.loadLabRequirements(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Labs: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ Lab Requirements: %d entries", count)
	}

	// Load Anticholinergic Burdens
	if count, err := kl.loadAnticholinergicBurdens(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("ACB: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ Anticholinergic Burden: %d entries", count)
	}

	// Load Contraindications
	if count, err := kl.loadContraindications(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Contraindications: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ Contraindications: %d entries", count)
	}

	// Load Dose Limits
	if count, err := kl.loadDoseLimits(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("DoseLimits: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ Dose Limits: %d entries", count)
	}

	// Load Age Limits
	if count, err := kl.loadAgeLimits(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("AgeLimits: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ Age Limits: %d entries", count)
	}

	// Load STOPP Criteria (European Geriatric - Potentially Inappropriate Prescribing)
	if count, err := kl.loadStoppEntries(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("STOPP: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ STOPP Criteria: %d entries", count)
	}

	// Load START Criteria (European Geriatric - Prescribing Omissions)
	if count, err := kl.loadStartEntries(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("START: %v", err))
	} else {
		loadCount += count
		log.Printf("   ✅ START Criteria: %d entries", count)
	}

	// Load India-Specific: Banned Fixed-Dose Combinations (CDSCO)
	if count, err := kl.loadBannedCombinations(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("BannedFDC: %v", err))
	} else {
		loadCount += count
		if count > 0 {
			log.Printf("   ✅ Banned FDC Combinations (IN): %d entries", count)
		}
	}

	// Load India-Specific: National List of Essential Medicines (NLEM)
	if count, err := kl.loadNLEMMedications(); err != nil {
		kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("NLEM: %v", err))
	} else {
		loadCount += count
		if count > 0 {
			log.Printf("   ✅ NLEM Essential Medicines (IN): %d entries", count)
		}
	}

	kl.store.TotalEntries = loadCount
	log.Printf("📊 Total knowledge entries loaded: %d", loadCount)

	if len(kl.store.LoadErrors) > 0 {
		log.Printf("⚠️  Load errors: %d", len(kl.store.LoadErrors))
		for _, err := range kl.store.LoadErrors {
			log.Printf("   - %s", err)
		}
	}

	return kl.store, nil
}

// loadBlackBoxWarnings loads black box warning files from jurisdiction-aware paths
func (kl *KnowledgeLoader) loadBlackBoxWarnings() (int, error) {
	searchPaths := kl.getSearchPaths("blackbox")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var bbFile BlackBoxWarningsFile
			if err := yaml.Unmarshal(data, &bbFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range bbFile.Entries {
				if entry.RxNormCode != "" {
					// Only add if not already present (jurisdiction-specific takes priority)
					if _, exists := kl.store.BlackBoxWarnings[entry.RxNormCode]; !exists {
						kl.store.BlackBoxWarnings[entry.RxNormCode] = entry
						// Build cross-reference indices
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						if entry.ATCCode != "" {
							kl.store.ATCToRxNorm[entry.ATCCode] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadHighAlertMedications loads high-alert medication files from jurisdiction-aware paths
func (kl *KnowledgeLoader) loadHighAlertMedications() (int, error) {
	searchPaths := kl.getSearchPaths("high-alert")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var haFile HighAlertMedicationsFile
			if err := yaml.Unmarshal(data, &haFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range haFile.Entries {
				if entry.RxNormCode != "" {
					if _, exists := kl.store.HighAlertMedications[entry.RxNormCode]; !exists {
						kl.store.HighAlertMedications[entry.RxNormCode] = entry
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						if entry.ATCCode != "" {
							kl.store.ATCToRxNorm[entry.ATCCode] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadBeersEntries loads Beers Criteria files from jurisdiction-aware paths
func (kl *KnowledgeLoader) loadBeersEntries() (int, error) {
	searchPaths := kl.getSearchPaths("beers")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var beersFile BeersEntriesFile
			if err := yaml.Unmarshal(data, &beersFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range beersFile.GetEntries() {
				if entry.RxNormCode != "" {
					if _, exists := kl.store.BeersEntries[entry.RxNormCode]; !exists {
						kl.store.BeersEntries[entry.RxNormCode] = entry
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						if entry.ATCCode != "" {
							kl.store.ATCToRxNorm[entry.ATCCode] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadPregnancySafety loads pregnancy safety files from jurisdiction-aware paths
func (kl *KnowledgeLoader) loadPregnancySafety() (int, error) {
	searchPaths := kl.getSearchPaths("pregnancy")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var pregFile PregnancySafetyFile
			if err := yaml.Unmarshal(data, &pregFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range pregFile.GetEntries() {
				if entry.RxNormCode != "" {
					if _, exists := kl.store.PregnancySafety[entry.RxNormCode]; !exists {
						kl.store.PregnancySafety[entry.RxNormCode] = entry
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						if entry.ATCCode != "" {
							kl.store.ATCToRxNorm[entry.ATCCode] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadLactationSafety loads lactation safety files from jurisdiction-aware paths
func (kl *KnowledgeLoader) loadLactationSafety() (int, error) {
	searchPaths := kl.getSearchPaths("lactation")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var lactFile LactationSafetyFile
			if err := yaml.Unmarshal(data, &lactFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range lactFile.GetEntries() {
				if entry.RxNormCode != "" {
					if _, exists := kl.store.LactationSafety[entry.RxNormCode]; !exists {
						kl.store.LactationSafety[entry.RxNormCode] = entry
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						if entry.ATCCode != "" {
							kl.store.ATCToRxNorm[entry.ATCCode] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadLabRequirements loads lab monitoring requirements files from jurisdiction-aware paths
func (kl *KnowledgeLoader) loadLabRequirements() (int, error) {
	searchPaths := kl.getSearchPaths("lab-monitoring")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var labFile LabRequirementsFile
			if err := yaml.Unmarshal(data, &labFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range labFile.GetEntries() {
				if entry.RxNormCode != "" {
					// Only add if not already present (jurisdiction-specific takes priority)
					if _, exists := kl.store.LabRequirements[entry.RxNormCode]; !exists {
						kl.store.LabRequirements[entry.RxNormCode] = entry
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						if entry.ATCCode != "" {
							kl.store.ATCToRxNorm[entry.ATCCode] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadAnticholinergicBurdens loads ACB score files from jurisdiction-aware paths
func (kl *KnowledgeLoader) loadAnticholinergicBurdens() (int, error) {
	searchPaths := kl.getSearchPaths("anticholinergic")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var acbFile AnticholinergicBurdensFile
			if err := yaml.Unmarshal(data, &acbFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range acbFile.Entries {
				if entry.RxNormCode != "" {
					// Only add if not already present (jurisdiction-specific takes priority)
					if _, exists := kl.store.AnticholinergicBurdens[entry.RxNormCode]; !exists {
						kl.store.AnticholinergicBurdens[entry.RxNormCode] = entry
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						if entry.ATCCode != "" {
							kl.store.ATCToRxNorm[entry.ATCCode] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadContraindications loads contraindication files from jurisdiction-aware paths
// Note: Contraindications support multiple entries per drug, so all are appended.
// Jurisdiction priority is maintained by loading jurisdiction-specific files first.
func (kl *KnowledgeLoader) loadContraindications() (int, error) {
	searchPaths := kl.getSearchPaths("contraindications")

	count := 0
	seenEntries := make(map[string]map[string]bool) // drug -> condition -> seen

	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var ciFile ContraindicationsFile
			if err := yaml.Unmarshal(data, &ciFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range ciFile.Entries {
				if entry.RxNormCode != "" {
					// Initialize seen map for this drug if needed
					if seenEntries[entry.RxNormCode] == nil {
						seenEntries[entry.RxNormCode] = make(map[string]bool)
					}

					// Create unique key from conditions + type to avoid duplicate entries
					conditionsKey := strings.Join(entry.ConditionCodes, ",")
					entryKey := fmt.Sprintf("%s:%s", conditionsKey, entry.Type)

					// Only add if this specific contraindication hasn't been seen
					// (jurisdiction-specific takes priority, loaded first)
					if !seenEntries[entry.RxNormCode][entryKey] {
						seenEntries[entry.RxNormCode][entryKey] = true
						kl.store.Contraindications[entry.RxNormCode] = append(
							kl.store.Contraindications[entry.RxNormCode],
							entry,
						)
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						if entry.ATCCode != "" {
							kl.store.ATCToRxNorm[entry.ATCCode] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadDoseLimits loads dose limit files from jurisdiction-aware paths
func (kl *KnowledgeLoader) loadDoseLimits() (int, error) {
	searchPaths := kl.getSearchPaths("dose_limits")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var dlFile DoseLimitsFile
			if err := yaml.Unmarshal(data, &dlFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range dlFile.GetEntries() {
				if entry.RxNormCode != "" {
					// Only add if not already present (jurisdiction-specific takes priority)
					if _, exists := kl.store.DoseLimits[entry.RxNormCode]; !exists {
						kl.store.DoseLimits[entry.RxNormCode] = entry
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						if entry.ATCCode != "" {
							kl.store.ATCToRxNorm[entry.ATCCode] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadAgeLimits loads age limit files from jurisdiction-aware paths
func (kl *KnowledgeLoader) loadAgeLimits() (int, error) {
	searchPaths := kl.getSearchPaths("age_limits")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var alFile AgeLimitsFile
			if err := yaml.Unmarshal(data, &alFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range alFile.GetEntries() {
				if entry.RxNormCode != "" {
					// Only add if not already present (jurisdiction-specific takes priority)
					if _, exists := kl.store.AgeLimits[entry.RxNormCode]; !exists {
						kl.store.AgeLimits[entry.RxNormCode] = entry
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNormCode
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadStoppEntries loads STOPP criteria files from jurisdiction-aware paths
// STOPP: Screening Tool of Older Persons' Prescriptions
func (kl *KnowledgeLoader) loadStoppEntries() (int, error) {
	searchPaths := kl.getSearchPaths("stopp_start")

	count := 0
	for _, dir := range searchPaths {
		// Match stopp*.yaml files (e.g., stopp_v3.yaml)
		files, err := filepath.Glob(filepath.Join(dir, "stopp*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var stoppFile StoppEntriesFile
			if err := yaml.Unmarshal(data, &stoppFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range stoppFile.GetEntries() {
				if entry.ID != "" {
					// Only add if not already present (jurisdiction-specific takes priority)
					if _, exists := kl.store.StoppEntries[entry.ID]; !exists {
						kl.store.StoppEntries[entry.ID] = entry
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadStartEntries loads START criteria files from jurisdiction-aware paths
// START: Screening Tool to Alert to Right Treatment
func (kl *KnowledgeLoader) loadStartEntries() (int, error) {
	searchPaths := kl.getSearchPaths("stopp_start")

	count := 0
	for _, dir := range searchPaths {
		// Match start*.yaml files (e.g., start_v3.yaml)
		files, err := filepath.Glob(filepath.Join(dir, "start*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var startFile StartEntriesFile
			if err := yaml.Unmarshal(data, &startFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range startFile.GetEntries() {
				if entry.ID != "" {
					// Only add if not already present (jurisdiction-specific takes priority)
					if _, exists := kl.store.StartEntries[entry.ID]; !exists {
						kl.store.StartEntries[entry.ID] = entry
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// =============================================================================
// INDIA-SPECIFIC LOADERS (CDSCO, NLEM)
// =============================================================================

// loadBannedCombinations loads CDSCO banned fixed-dose combination files
// These are India-specific banned FDCs from Gazette Notifications
func (kl *KnowledgeLoader) loadBannedCombinations() (int, error) {
	searchPaths := kl.getSearchPaths("banned-fdc")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var bcFile BannedCombinationsFile
			if err := yaml.Unmarshal(data, &bcFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range bcFile.GetEntries() {
				if entry.ID != "" {
					// Only add if not already present
					if _, exists := kl.store.BannedCombinations[entry.ID]; !exists {
						kl.store.BannedCombinations[entry.ID] = entry
						// Build cross-reference indices for component drugs
						for _, comp := range entry.Components {
							if comp.Drug != "" && comp.RxNorm != "" {
								kl.store.DrugNameToRxNorm[comp.Drug] = comp.RxNorm
							}
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// loadNLEMMedications loads India's National List of Essential Medicines
// NLEM 2022 contains 384 medicines across 27 therapeutic categories
func (kl *KnowledgeLoader) loadNLEMMedications() (int, error) {
	searchPaths := kl.getSearchPaths("nlem")

	count := 0
	for _, dir := range searchPaths {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Read %s: %v", file, err))
				continue
			}

			var nlemFile NLEMFile
			if err := yaml.Unmarshal(data, &nlemFile); err != nil {
				kl.store.LoadErrors = append(kl.store.LoadErrors, fmt.Sprintf("Parse %s: %v", file, err))
				continue
			}

			kl.store.mu.Lock()
			for _, entry := range nlemFile.GetAllMedications() {
				if entry.RxNorm != "" {
					// Only add if not already present
					if _, exists := kl.store.NLEMMedications[entry.RxNorm]; !exists {
						kl.store.NLEMMedications[entry.RxNorm] = entry
						if entry.DrugName != "" {
							kl.store.DrugNameToRxNorm[entry.DrugName] = entry.RxNorm
						}
						count++
					}
				}
			}
			kl.store.mu.Unlock()
		}
	}

	return count, nil
}

// =============================================================================
// KNOWLEDGE STORE QUERY METHODS
// =============================================================================

// GetBlackBoxWarning returns a black box warning for the given RxNorm code
func (ks *KnowledgeStore) GetBlackBoxWarning(rxnormCode string) (BlackBoxWarning, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	bb, ok := ks.BlackBoxWarnings[rxnormCode]
	return bb, ok
}

// GetHighAlertMedication returns high-alert medication info for the given RxNorm code
func (ks *KnowledgeStore) GetHighAlertMedication(rxnormCode string) (HighAlertMedication, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	ha, ok := ks.HighAlertMedications[rxnormCode]
	return ha, ok
}

// GetBeersEntry returns Beers Criteria entry for the given RxNorm code
func (ks *KnowledgeStore) GetBeersEntry(rxnormCode string) (BeersEntry, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	be, ok := ks.BeersEntries[rxnormCode]
	return be, ok
}

// GetPregnancySafety returns pregnancy safety info for the given RxNorm code
func (ks *KnowledgeStore) GetPregnancySafety(rxnormCode string) (PregnancySafety, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	ps, ok := ks.PregnancySafety[rxnormCode]
	return ps, ok
}

// GetLactationSafety returns lactation safety info for the given RxNorm code
func (ks *KnowledgeStore) GetLactationSafety(rxnormCode string) (LactationSafety, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	ls, ok := ks.LactationSafety[rxnormCode]
	return ls, ok
}

// GetLabRequirement returns lab monitoring requirements for the given RxNorm code
func (ks *KnowledgeStore) GetLabRequirement(rxnormCode string) (LabRequirement, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	lr, ok := ks.LabRequirements[rxnormCode]
	return lr, ok
}

// GetAnticholinergicBurden returns ACB score for the given RxNorm code
func (ks *KnowledgeStore) GetAnticholinergicBurden(rxnormCode string) (AnticholinergicBurden, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	ab, ok := ks.AnticholinergicBurdens[rxnormCode]
	return ab, ok
}

// GetContraindications returns contraindications for the given RxNorm code
func (ks *KnowledgeStore) GetContraindications(rxnormCode string) ([]Contraindication, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	ci, ok := ks.Contraindications[rxnormCode]
	return ci, ok
}

// GetDoseLimit returns dose limit for the given RxNorm code
func (ks *KnowledgeStore) GetDoseLimit(rxnormCode string) (DoseLimit, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	dl, ok := ks.DoseLimits[rxnormCode]
	return dl, ok
}

// GetAgeLimit returns age limit for the given RxNorm code
func (ks *KnowledgeStore) GetAgeLimit(rxnormCode string) (AgeLimit, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	al, ok := ks.AgeLimits[rxnormCode]
	return al, ok
}

// GetStoppEntry returns STOPP criterion for the given criterion ID (e.g., "A1", "B2")
func (ks *KnowledgeStore) GetStoppEntry(criterionID string) (StoppEntry, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	se, ok := ks.StoppEntries[criterionID]
	return se, ok
}

// GetStartEntry returns START criterion for the given criterion ID (e.g., "A1", "B2")
func (ks *KnowledgeStore) GetStartEntry(criterionID string) (StartEntry, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	se, ok := ks.StartEntries[criterionID]
	return se, ok
}

// GetAllStoppEntries returns all STOPP criteria entries
func (ks *KnowledgeStore) GetAllStoppEntries() []StoppEntry {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	entries := make([]StoppEntry, 0, len(ks.StoppEntries))
	for _, entry := range ks.StoppEntries {
		entries = append(entries, entry)
	}
	return entries
}

// GetAllStartEntries returns all START criteria entries
func (ks *KnowledgeStore) GetAllStartEntries() []StartEntry {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	entries := make([]StartEntry, 0, len(ks.StartEntries))
	for _, entry := range ks.StartEntries {
		entries = append(entries, entry)
	}
	return entries
}

// GetStoppEntriesBySection returns STOPP entries for a specific section (e.g., "A", "B")
func (ks *KnowledgeStore) GetStoppEntriesBySection(sectionPrefix string) []StoppEntry {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	entries := make([]StoppEntry, 0)
	for _, entry := range ks.StoppEntries {
		if strings.HasPrefix(entry.Section, sectionPrefix) {
			entries = append(entries, entry)
		}
	}
	return entries
}

// GetStartEntriesBySection returns START entries for a specific section (e.g., "A", "B")
func (ks *KnowledgeStore) GetStartEntriesBySection(sectionPrefix string) []StartEntry {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	entries := make([]StartEntry, 0)
	for _, entry := range ks.StartEntries {
		if strings.HasPrefix(entry.Section, sectionPrefix) {
			entries = append(entries, entry)
		}
	}
	return entries
}

// =============================================================================
// INDIA-SPECIFIC QUERY METHODS (CDSCO, NLEM)
// =============================================================================

// GetBannedCombination returns a banned FDC entry for the given combination ID
func (ks *KnowledgeStore) GetBannedCombination(combinationID string) (BannedCombinationEntry, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	bc, ok := ks.BannedCombinations[combinationID]
	return bc, ok
}

// GetAllBannedCombinations returns all banned FDC entries
func (ks *KnowledgeStore) GetAllBannedCombinations() []BannedCombinationEntry {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	entries := make([]BannedCombinationEntry, 0, len(ks.BannedCombinations))
	for _, entry := range ks.BannedCombinations {
		entries = append(entries, entry)
	}
	return entries
}

// GetBannedCombinationsByCategory returns banned FDCs for a specific therapeutic category
func (ks *KnowledgeStore) GetBannedCombinationsByCategory(category string) []BannedCombinationEntry {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	entries := make([]BannedCombinationEntry, 0)
	for _, entry := range ks.BannedCombinations {
		if entry.Category == category {
			entries = append(entries, entry)
		}
	}
	return entries
}

// CheckBannedCombination checks if a set of RxNorm codes forms a banned combination
// Returns the banned combination entry if found, nil otherwise
func (ks *KnowledgeStore) CheckBannedCombination(rxnormCodes []string) *BannedCombinationEntry {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	// Create a set of input RxNorm codes for fast lookup
	inputSet := make(map[string]bool)
	for _, code := range rxnormCodes {
		inputSet[code] = true
	}

	// Check each banned combination
	for _, entry := range ks.BannedCombinations {
		matchCount := 0
		for _, comp := range entry.Components {
			if inputSet[comp.RxNorm] {
				matchCount++
			}
		}
		// If all components of this banned combination are present
		if matchCount == len(entry.Components) && matchCount > 0 {
			return &entry
		}
	}
	return nil
}

// GetNLEMMedication returns NLEM entry for the given RxNorm code
func (ks *KnowledgeStore) GetNLEMMedication(rxnormCode string) (NLEMMedication, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	nlem, ok := ks.NLEMMedications[rxnormCode]
	return nlem, ok
}

// GetAllNLEMMedications returns all NLEM medication entries
func (ks *KnowledgeStore) GetAllNLEMMedications() []NLEMMedication {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	entries := make([]NLEMMedication, 0, len(ks.NLEMMedications))
	for _, entry := range ks.NLEMMedications {
		entries = append(entries, entry)
	}
	return entries
}

// GetNLEMByEssentialLevel returns NLEM medications by essential level (P/S/T)
func (ks *KnowledgeStore) GetNLEMByEssentialLevel(level string) []NLEMMedication {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	entries := make([]NLEMMedication, 0)
	for _, entry := range ks.NLEMMedications {
		if entry.EssentialLevel == level {
			entries = append(entries, entry)
		}
	}
	return entries
}

// GetNLEMByCategory returns NLEM medications by therapeutic category
func (ks *KnowledgeStore) GetNLEMByCategory(category string) []NLEMMedication {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	entries := make([]NLEMMedication, 0)
	for _, entry := range ks.NLEMMedications {
		if entry.Category == category {
			entries = append(entries, entry)
		}
	}
	return entries
}

// IsEssentialMedicine checks if a drug is on India's National List of Essential Medicines
func (ks *KnowledgeStore) IsEssentialMedicine(rxnormCode string) bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	_, ok := ks.NLEMMedications[rxnormCode]
	return ok
}

// ResolveRxNormCode resolves a drug name or ATC code to RxNorm code
func (ks *KnowledgeStore) ResolveRxNormCode(identifier string) string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	// Check if it's already an RxNorm code
	if _, ok := ks.BlackBoxWarnings[identifier]; ok {
		return identifier
	}
	if _, ok := ks.HighAlertMedications[identifier]; ok {
		return identifier
	}

	// Check drug name index
	if rxnorm, ok := ks.DrugNameToRxNorm[identifier]; ok {
		return rxnorm
	}

	// Check ATC code index
	if rxnorm, ok := ks.ATCToRxNorm[identifier]; ok {
		return rxnorm
	}

	return identifier // Return as-is if not found
}

// GetStats returns loading statistics
func (ks *KnowledgeStore) GetStats() map[string]int {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	// Count total contraindication entries across all drugs
	contraIndicationCount := 0
	for _, entries := range ks.Contraindications {
		contraIndicationCount += len(entries)
	}

	return map[string]int{
		"blackbox_warnings":       len(ks.BlackBoxWarnings),
		"high_alert_medications":  len(ks.HighAlertMedications),
		"beers_entries":           len(ks.BeersEntries),
		"pregnancy_safety":        len(ks.PregnancySafety),
		"lactation_safety":        len(ks.LactationSafety),
		"lab_requirements":        len(ks.LabRequirements),
		"anticholinergic_burdens": len(ks.AnticholinergicBurdens),
		"contraindications":       contraIndicationCount,
		"dose_limits":             len(ks.DoseLimits),
		"age_limits":              len(ks.AgeLimits),
		"stopp_entries":           len(ks.StoppEntries),
		"start_entries":           len(ks.StartEntries),
		"banned_combinations_in":  len(ks.BannedCombinations),
		"nlem_medications_in":     len(ks.NLEMMedications),
		"drug_name_index":         len(ks.DrugNameToRxNorm),
		"atc_index":               len(ks.ATCToRxNorm),
		"total_entries":           ks.TotalEntries,
		"load_errors":             len(ks.LoadErrors),
	}
}
