// Package adapters provides ICD-10-CM/PCS (International Classification of Diseases) adapter.
package adapters

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// ICD-10 ADAPTER (DIAGNOSTIC AND PROCEDURE CODES)
// =============================================================================

// ICD10Adapter ingests ICD-10-CM (diagnoses) and ICD-10-PCS (procedures) codes.
// Used by KB-7 (Terminology), KB-13 (Quality Measures).
// Source: CMS (US) and WHO (International).
type ICD10Adapter struct {
	*BaseAdapter
	baseURL     string
	httpClient  *http.Client
	fiscalYear  int // CMS fiscal year (e.g., 2024)
}

// NewICD10Adapter creates a new ICD-10 adapter.
func NewICD10Adapter(fiscalYear int) *ICD10Adapter {
	return &ICD10Adapter{
		BaseAdapter: NewBaseAdapter(
			"ICD10",
			models.AuthorityCMS,
			[]models.KB{models.KB7, models.KB13},
		),
		baseURL: "https://www.cms.gov/medicare/coding-billing/icd-10-codes",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		fiscalYear: fiscalYear,
	}
}

// GetSupportedTypes returns the knowledge types this adapter can produce.
func (a *ICD10Adapter) GetSupportedTypes() []models.KnowledgeType {
	return []models.KnowledgeType{
		models.TypeTerminology,
		models.TypeValueSet,
	}
}

// FetchUpdates retrieves ICD-10 codes updated since the given timestamp.
func (a *ICD10Adapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// ICD-10 is released annually (October 1 for US fiscal year)
	var items []RawItem

	// Check if we're in a new fiscal year since 'since'
	releaseDate := time.Date(a.fiscalYear-1, time.October, 1, 0, 0, 0, 0, time.UTC)
	if releaseDate.After(since) {
		// Fetch CM (diagnosis) codes
		cmCodes, err := a.fetchCMCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch ICD-10-CM codes: %w", err)
		}
		items = append(items, cmCodes...)

		// Fetch PCS (procedure) codes
		pcsCodes, err := a.fetchPCSCodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch ICD-10-PCS codes: %w", err)
		}
		items = append(items, pcsCodes...)
	}

	return items, nil
}

// Transform converts raw ICD-10 code to a KnowledgeItem.
func (a *ICD10Adapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	var code ICD10Code
	if err := json.Unmarshal(raw.RawData, &code); err != nil {
		return nil, fmt.Errorf("failed to parse ICD-10 code: %w", err)
	}

	codeType := "CM"
	kb := models.KB7
	if code.CodeType == ICD10TypePCS {
		codeType = "PCS"
	}

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb7:icd10%s:%s", strings.ToLower(codeType), code.Code),
		Type:    models.TypeTerminology,
		KB:      kb,
		Version: fmt.Sprintf("FY%d", a.fiscalYear),
		Name:    code.ShortDescription,
		Description: code.LongDescription,
		Source: models.SourceAttribution{
			Authority:     models.AuthorityCMS,
			Document:      fmt.Sprintf("ICD-10-%s FY%d", codeType, a.fiscalYear),
			Section:       code.Chapter,
			Jurisdiction:  models.JurisdictionUS,
			URL:           fmt.Sprintf("https://www.cms.gov/medicare/coding-billing/icd-10-codes/%d-icd-10-%s", a.fiscalYear, strings.ToLower(codeType)),
			EffectiveDate: fmt.Sprintf("%d-10-01", a.fiscalYear-1),
		},
		ContentRef:  fmt.Sprintf("icd10:%s:%s", strings.ToLower(codeType), code.Code),
		ContentHash: "",
		State:       models.StateDraft,
		RiskLevel:   models.RiskLow,
		WorkflowTemplate: models.TemplateInfraLow,
		RequiresDualReview: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs ICD-10-specific validation.
func (a *ICD10Adapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	if item.Source.Authority != models.AuthorityCMS && item.Source.Authority != models.AuthorityWHO {
		return fmt.Errorf("invalid authority: expected CMS or WHO, got %s", item.Source.Authority)
	}
	return nil
}

// =============================================================================
// ICD-10 DATA STRUCTURES
// =============================================================================

// ICD10CodeType distinguishes between CM and PCS codes.
type ICD10CodeType string

const (
	ICD10TypeCM  ICD10CodeType = "CM"  // Clinical Modification (Diagnoses)
	ICD10TypePCS ICD10CodeType = "PCS" // Procedure Coding System
)

// ICD10Code represents an ICD-10 code entry.
type ICD10Code struct {
	Code             string        `json:"code"`
	CodeType         ICD10CodeType `json:"code_type"`
	ShortDescription string        `json:"short_description"`
	LongDescription  string        `json:"long_description"`
	Chapter          string        `json:"chapter,omitempty"`
	ChapterTitle     string        `json:"chapter_title,omitempty"`
	BlockRange       string        `json:"block_range,omitempty"`
	CategoryCode     string        `json:"category_code,omitempty"`
	IsBillable       bool          `json:"is_billable"`
	IsHeader         bool          `json:"is_header"` // Category header vs. specific code
	ParentCode       string        `json:"parent_code,omitempty"`
	FiscalYear       int           `json:"fiscal_year"`

	// CM-specific fields
	SeventhCharExt   []SeventhCharExtension `json:"seventh_char_extensions,omitempty"`
	Excludes1        []string               `json:"excludes1,omitempty"` // Mutually exclusive
	Excludes2        []string               `json:"excludes2,omitempty"` // Not included here
	Includes         []string               `json:"includes,omitempty"`
	CodeFirst        []string               `json:"code_first,omitempty"`
	UseAdditional    []string               `json:"use_additional,omitempty"`

	// PCS-specific fields
	PCSSection       string `json:"pcs_section,omitempty"`
	PCSBodySystem    string `json:"pcs_body_system,omitempty"`
	PCSOperation     string `json:"pcs_operation,omitempty"`
	PCSBodyPart      string `json:"pcs_body_part,omitempty"`
	PCSApproach      string `json:"pcs_approach,omitempty"`
	PCSDevice        string `json:"pcs_device,omitempty"`
	PCSQualifier     string `json:"pcs_qualifier,omitempty"`
}

// SeventhCharExtension represents 7th character extensions for certain ICD-10-CM codes.
type SeventhCharExtension struct {
	Character   string `json:"character"`
	Description string `json:"description"`
}

// =============================================================================
// ICD-10-CM CHAPTER STRUCTURE
// =============================================================================

// ICD10CMChapter represents an ICD-10-CM chapter.
type ICD10CMChapter struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	CodeRange  string `json:"code_range"`
	Categories []ICD10CMCategory `json:"categories,omitempty"`
}

// ICD10CMCategory represents a category within a chapter.
type ICD10CMCategory struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Subcategories []ICD10Code `json:"subcategories,omitempty"`
}

// ICD-10-CM Chapters
var ICD10CMChapters = []ICD10CMChapter{
	{1, "Certain infectious and parasitic diseases", "A00-B99", nil},
	{2, "Neoplasms", "C00-D49", nil},
	{3, "Diseases of the blood and blood-forming organs", "D50-D89", nil},
	{4, "Endocrine, nutritional and metabolic diseases", "E00-E89", nil},
	{5, "Mental, Behavioral and Neurodevelopmental disorders", "F01-F99", nil},
	{6, "Diseases of the nervous system", "G00-G99", nil},
	{7, "Diseases of the eye and adnexa", "H00-H59", nil},
	{8, "Diseases of the ear and mastoid process", "H60-H95", nil},
	{9, "Diseases of the circulatory system", "I00-I99", nil},
	{10, "Diseases of the respiratory system", "J00-J99", nil},
	{11, "Diseases of the digestive system", "K00-K95", nil},
	{12, "Diseases of the skin and subcutaneous tissue", "L00-L99", nil},
	{13, "Diseases of the musculoskeletal system and connective tissue", "M00-M99", nil},
	{14, "Diseases of the genitourinary system", "N00-N99", nil},
	{15, "Pregnancy, childbirth and the puerperium", "O00-O9A", nil},
	{16, "Certain conditions originating in the perinatal period", "P00-P96", nil},
	{17, "Congenital malformations, deformations and chromosomal abnormalities", "Q00-Q99", nil},
	{18, "Symptoms, signs and abnormal clinical and laboratory findings", "R00-R99", nil},
	{19, "Injury, poisoning and certain other consequences of external causes", "S00-T88", nil},
	{20, "External causes of morbidity", "V00-Y99", nil},
	{21, "Factors influencing health status and contact with health services", "Z00-Z99", nil},
}

// =============================================================================
// ICD-10-PCS STRUCTURE
// =============================================================================

// ICD10PCSSection represents a PCS section.
type ICD10PCSSection struct {
	Value       string `json:"value"`       // Single character 0-9, B-D, F-H, X
	Title       string `json:"title"`
	BodySystems []ICD10PCSBodySystem `json:"body_systems,omitempty"`
}

// ICD10PCSBodySystem represents a body system within a section.
type ICD10PCSBodySystem struct {
	Value      string `json:"value"`
	Title      string `json:"title"`
	Operations []ICD10PCSOperation `json:"operations,omitempty"`
}

// ICD10PCSOperation represents an operation root.
type ICD10PCSOperation struct {
	Value       string `json:"value"`
	Title       string `json:"title"`
	Definition  string `json:"definition"`
	Explanation string `json:"explanation,omitempty"`
}

// ICD-10-PCS Sections
var ICD10PCSSections = []ICD10PCSSection{
	{"0", "Medical and Surgical", nil},
	{"1", "Obstetrics", nil},
	{"2", "Placement", nil},
	{"3", "Administration", nil},
	{"4", "Measurement and Monitoring", nil},
	{"5", "Extracorporeal or Systemic Assistance and Performance", nil},
	{"6", "Extracorporeal or Systemic Therapies", nil},
	{"7", "Osteopathic", nil},
	{"8", "Other Procedures", nil},
	{"9", "Chiropractic", nil},
	{"B", "Imaging", nil},
	{"C", "Nuclear Medicine", nil},
	{"D", "Radiation Therapy", nil},
	{"F", "Physical Rehabilitation and Diagnostic Audiology", nil},
	{"G", "Mental Health", nil},
	{"H", "Substance Abuse Treatment", nil},
	{"X", "New Technology", nil},
}

// Common PCS Operations (Medical and Surgical Section)
var ICD10PCSOperations = map[string]ICD10PCSOperation{
	"0": {Value: "0", Title: "Alteration", Definition: "Modifying the anatomic structure of a body part without affecting the function"},
	"1": {Value: "1", Title: "Bypass", Definition: "Altering the route of passage of the contents of a tubular body part"},
	"2": {Value: "2", Title: "Change", Definition: "Taking out or off a device from a body part and putting back an identical or similar device"},
	"3": {Value: "3", Title: "Control", Definition: "Stopping, or attempting to stop, postprocedural or other acute bleeding"},
	"4": {Value: "4", Title: "Creation", Definition: "Putting in or on biological or synthetic material to form a new body part"},
	"5": {Value: "5", Title: "Destruction", Definition: "Physical eradication of all or a portion of a body part"},
	"6": {Value: "6", Title: "Detachment", Definition: "Cutting off all or a portion of the upper or lower extremities"},
	"7": {Value: "7", Title: "Dilation", Definition: "Expanding an orifice or the lumen of a tubular body part"},
	"8": {Value: "8", Title: "Division", Definition: "Cutting into a body part, without draining fluids and/or gases"},
	"9": {Value: "9", Title: "Drainage", Definition: "Taking or letting out fluids and/or gases from a body part"},
	"B": {Value: "B", Title: "Excision", Definition: "Cutting out or off, without replacement, a portion of a body part"},
	"C": {Value: "C", Title: "Extirpation", Definition: "Taking or cutting out solid matter from a body part"},
	"D": {Value: "D", Title: "Extraction", Definition: "Pulling or stripping out or off all or a portion of a body part"},
	"F": {Value: "F", Title: "Fragmentation", Definition: "Breaking solid matter in a body part into pieces"},
	"G": {Value: "G", Title: "Fusion", Definition: "Joining together portions of an articular body part"},
	"H": {Value: "H", Title: "Insertion", Definition: "Putting in a nonbiological appliance that monitors, assists, performs, or prevents"},
	"J": {Value: "J", Title: "Inspection", Definition: "Visually and/or manually exploring a body part"},
	"K": {Value: "K", Title: "Map", Definition: "Locating the route of passage of electrical impulses"},
	"L": {Value: "L", Title: "Occlusion", Definition: "Completely closing an orifice or the lumen of a tubular body part"},
	"M": {Value: "M", Title: "Reattachment", Definition: "Putting back in or on all or a portion of a separated body part"},
	"N": {Value: "N", Title: "Release", Definition: "Freeing a body part from an abnormal physical constraint"},
	"P": {Value: "P", Title: "Removal", Definition: "Taking out or off a device from a body part"},
	"Q": {Value: "Q", Title: "Repair", Definition: "Restoring, to the extent possible, a body part to its normal structure"},
	"R": {Value: "R", Title: "Replacement", Definition: "Putting in or on biological or synthetic material"},
	"S": {Value: "S", Title: "Reposition", Definition: "Moving to its normal location, or other suitable location"},
	"T": {Value: "T", Title: "Resection", Definition: "Cutting out or off, without replacement, all of a body part"},
	"U": {Value: "U", Title: "Supplement", Definition: "Putting in or on biological or synthetic material that reinforces"},
	"V": {Value: "V", Title: "Restriction", Definition: "Partially closing an orifice or the lumen of a tubular body part"},
	"W": {Value: "W", Title: "Revision", Definition: "Correcting, to the extent possible, a portion of a malfunctioning device"},
	"X": {Value: "X", Title: "Transfer", Definition: "Moving, without taking out, all or a portion of a body part"},
	"Y": {Value: "Y", Title: "Transplantation", Definition: "Putting in or on all or a portion of a living body part taken from another individual"},
}

// =============================================================================
// ICD-10 OPERATIONS
// =============================================================================

// fetchCMCodes fetches ICD-10-CM codes from CMS.
func (a *ICD10Adapter) fetchCMCodes(ctx context.Context) ([]RawItem, error) {
	// CMS provides ICD-10-CM as downloadable flat files
	// In production, would download and parse the actual files
	url := fmt.Sprintf("%s/%d-icd-10-cm-codes-file", a.baseURL, a.fiscalYear)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse the flat file format
	return a.parseCMFlatFile(resp.Body)
}

// fetchPCSCodes fetches ICD-10-PCS codes from CMS.
func (a *ICD10Adapter) fetchPCSCodes(ctx context.Context) ([]RawItem, error) {
	url := fmt.Sprintf("%s/%d-icd-10-pcs-codes-file", a.baseURL, a.fiscalYear)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return a.parsePCSFlatFile(resp.Body)
}

// parseCMFlatFile parses CMS ICD-10-CM flat file format.
func (a *ICD10Adapter) parseCMFlatFile(r io.Reader) ([]RawItem, error) {
	var items []RawItem
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 10 {
			continue
		}

		code := ICD10Code{
			Code:             strings.TrimSpace(line[0:7]),
			CodeType:         ICD10TypeCM,
			ShortDescription: strings.TrimSpace(line[77:137]),
			LongDescription:  strings.TrimSpace(line[77:]),
			IsBillable:       len(strings.TrimSpace(line[0:7])) >= 3,
			FiscalYear:       a.fiscalYear,
		}

		// Determine chapter from code
		code.Chapter = a.getChapterForCode(code.Code)

		data, err := json.Marshal(code)
		if err != nil {
			continue
		}

		items = append(items, RawItem{
			ID:        code.Code,
			Authority: models.AuthorityCMS,
			RawData:   data,
		})
	}

	return items, scanner.Err()
}

// parsePCSFlatFile parses CMS ICD-10-PCS flat file format.
func (a *ICD10Adapter) parsePCSFlatFile(r io.Reader) ([]RawItem, error) {
	var items []RawItem
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 10 {
			continue
		}

		// PCS codes are 7 characters
		codeStr := strings.TrimSpace(line[0:7])
		if len(codeStr) != 7 {
			continue
		}

		code := ICD10Code{
			Code:             codeStr,
			CodeType:         ICD10TypePCS,
			ShortDescription: strings.TrimSpace(line[77:137]),
			LongDescription:  strings.TrimSpace(line[77:]),
			IsBillable:       true, // All 7-char PCS codes are billable
			FiscalYear:       a.fiscalYear,
			PCSSection:       codeStr[0:1],
			PCSBodySystem:    codeStr[1:2],
			PCSOperation:     codeStr[2:3],
			PCSBodyPart:      codeStr[3:4],
			PCSApproach:      codeStr[4:5],
			PCSDevice:        codeStr[5:6],
			PCSQualifier:     codeStr[6:7],
		}

		data, err := json.Marshal(code)
		if err != nil {
			continue
		}

		items = append(items, RawItem{
			ID:        code.Code,
			Authority: models.AuthorityCMS,
			RawData:   data,
		})
	}

	return items, scanner.Err()
}

// getChapterForCode determines the ICD-10-CM chapter for a code.
func (a *ICD10Adapter) getChapterForCode(code string) string {
	if len(code) < 1 {
		return ""
	}

	// Map first character to chapter range
	firstChar := code[0]

	switch {
	case firstChar >= 'A' && firstChar <= 'B':
		return "Chapter 1: Infectious diseases (A00-B99)"
	case firstChar >= 'C' && firstChar <= 'D' && (len(code) < 2 || code[1] <= '4'):
		return "Chapter 2: Neoplasms (C00-D49)"
	case firstChar == 'D' && len(code) >= 2 && code[1] >= '5':
		return "Chapter 3: Blood diseases (D50-D89)"
	case firstChar == 'E':
		return "Chapter 4: Endocrine diseases (E00-E89)"
	case firstChar == 'F':
		return "Chapter 5: Mental disorders (F01-F99)"
	case firstChar == 'G':
		return "Chapter 6: Nervous system (G00-G99)"
	case firstChar == 'H' && (len(code) < 2 || code[1] <= '5'):
		return "Chapter 7: Eye diseases (H00-H59)"
	case firstChar == 'H':
		return "Chapter 8: Ear diseases (H60-H95)"
	case firstChar == 'I':
		return "Chapter 9: Circulatory system (I00-I99)"
	case firstChar == 'J':
		return "Chapter 10: Respiratory system (J00-J99)"
	case firstChar == 'K':
		return "Chapter 11: Digestive system (K00-K95)"
	case firstChar == 'L':
		return "Chapter 12: Skin diseases (L00-L99)"
	case firstChar == 'M':
		return "Chapter 13: Musculoskeletal system (M00-M99)"
	case firstChar == 'N':
		return "Chapter 14: Genitourinary system (N00-N99)"
	case firstChar == 'O':
		return "Chapter 15: Pregnancy (O00-O9A)"
	case firstChar == 'P':
		return "Chapter 16: Perinatal conditions (P00-P96)"
	case firstChar == 'Q':
		return "Chapter 17: Congenital conditions (Q00-Q99)"
	case firstChar == 'R':
		return "Chapter 18: Symptoms and signs (R00-R99)"
	case firstChar == 'S' || firstChar == 'T':
		return "Chapter 19: Injury and poisoning (S00-T88)"
	case firstChar >= 'V' && firstChar <= 'Y':
		return "Chapter 20: External causes (V00-Y99)"
	case firstChar == 'Z':
		return "Chapter 21: Factors influencing health (Z00-Z99)"
	default:
		return ""
	}
}

// =============================================================================
// ICD-10 LOOKUP AND SEARCH
// =============================================================================

// LookupCode looks up an ICD-10 code.
func (a *ICD10Adapter) LookupCode(ctx context.Context, code string) (*ICD10Code, error) {
	// Determine code type
	codeType := ICD10TypeCM
	if a.isPCSCode(code) {
		codeType = ICD10TypePCS
	}

	url := fmt.Sprintf("%s/lookup?code=%s&type=%s&year=%d",
		a.baseURL, code, codeType, a.fiscalYear)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result ICD10Code
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// SearchCodes searches for ICD-10 codes by description.
func (a *ICD10Adapter) SearchCodes(ctx context.Context, query string, codeType ICD10CodeType, limit int) ([]*ICD10Code, error) {
	// In production, would search the CMS database or local index
	return nil, nil
}

// isPCSCode determines if a code is ICD-10-PCS format.
func (a *ICD10Adapter) isPCSCode(code string) bool {
	// PCS codes are exactly 7 alphanumeric characters
	if len(code) != 7 {
		return false
	}
	matched, _ := regexp.MatchString(`^[0-9A-HJ-NP-Z]{7}$`, code)
	return matched
}

// =============================================================================
// ICD-10 FILE IMPORT
// =============================================================================

// ImportFromFile imports ICD-10 codes from a local file.
func (a *ICD10Adapter) ImportFromFile(ctx context.Context, filepath string, codeType ICD10CodeType) ([]RawItem, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	switch codeType {
	case ICD10TypeCM:
		return a.parseCMFlatFile(file)
	case ICD10TypePCS:
		return a.parsePCSFlatFile(file)
	default:
		return nil, fmt.Errorf("unknown code type: %s", codeType)
	}
}

// GetGEMs retrieves General Equivalence Mappings for code translation.
// GEMs map between ICD-9 and ICD-10 codes.
func (a *ICD10Adapter) GetGEMs(ctx context.Context, fromCode string, direction string) ([]GEMMapping, error) {
	// In production, would look up GEM mappings
	return nil, nil
}

// GEMMapping represents a General Equivalence Mapping entry.
type GEMMapping struct {
	FromCode       string `json:"from_code"`
	FromSystem     string `json:"from_system"` // ICD-9-CM or ICD-10-CM
	ToCode         string `json:"to_code"`
	ToSystem       string `json:"to_system"`
	Flags          string `json:"flags"`          // Approximate, No Map, etc.
	Scenario       int    `json:"scenario"`       // For multiple mappings
	ChoiceList     int    `json:"choice_list"`    // Selection within scenario
}
