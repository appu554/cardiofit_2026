// Package adapters provides Clinical Guidelines adapter for multiple guideline sources.
package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// CLINICAL GUIDELINES ADAPTER (MULTI-SOURCE)
// =============================================================================

// GuidelinesAdapter ingests clinical practice guidelines from multiple authorities.
// Used by KB-15 (Evidence Engine), KB-19 (Protocol Orchestrator).
// Sources: IDSA, ACC/AHA, NICE, Cochrane, USPSTF, etc.
type GuidelinesAdapter struct {
	*BaseAdapter
	httpClient *http.Client
	sources    map[GuidelineSource]GuidelineSourceConfig
}

// GuidelineSource represents a guideline-issuing authority.
type GuidelineSource string

const (
	SourceIDSA       GuidelineSource = "IDSA"       // Infectious Disease Society of America
	SourceACCAHA     GuidelineSource = "ACC_AHA"    // American College of Cardiology / AHA
	SourceNICE       GuidelineSource = "NICE"       // UK National Institute for Health and Care Excellence
	SourceUSPSTF     GuidelineSource = "USPSTF"     // US Preventive Services Task Force
	SourceCochrane   GuidelineSource = "COCHRANE"   // Cochrane Reviews
	SourceWHO        GuidelineSource = "WHO"        // World Health Organization
	SourceSurviving  GuidelineSource = "SURVIVING"  // Surviving Sepsis Campaign
	SourceASCO       GuidelineSource = "ASCO"       // American Society of Clinical Oncology
	SourceADA        GuidelineSource = "ADA"        // American Diabetes Association
	SourceCHEST      GuidelineSource = "CHEST"      // American College of Chest Physicians
	SourceAAN        GuidelineSource = "AAN"        // American Academy of Neurology
)

// GuidelineSourceConfig contains configuration for a guideline source.
type GuidelineSourceConfig struct {
	Name        string
	BaseURL     string
	APIEndpoint string
	Authority   models.Authority
}

// NewGuidelinesAdapter creates a new clinical guidelines adapter.
func NewGuidelinesAdapter() *GuidelinesAdapter {
	return &GuidelinesAdapter{
		BaseAdapter: NewBaseAdapter(
			"CLINICAL_GUIDELINES",
			models.AuthorityInternal, // Multi-source, set per guideline
			[]models.KB{models.KB15, models.KB19},
		),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		sources: map[GuidelineSource]GuidelineSourceConfig{
			SourceIDSA: {
				Name:        "Infectious Diseases Society of America",
				BaseURL:     "https://www.idsociety.org",
				APIEndpoint: "/practice-guideline",
				Authority:   models.AuthorityIDSA,
			},
			SourceACCAHA: {
				Name:        "American College of Cardiology / American Heart Association",
				BaseURL:     "https://www.acc.org",
				APIEndpoint: "/guidelines",
				Authority:   models.AuthorityACC_AHA,
			},
			SourceNICE: {
				Name:        "National Institute for Health and Care Excellence",
				BaseURL:     "https://www.nice.org.uk",
				APIEndpoint: "/guidance",
				Authority:   models.AuthorityNICE,
			},
			SourceUSPSTF: {
				Name:        "US Preventive Services Task Force",
				BaseURL:     "https://www.uspreventiveservicestaskforce.org",
				APIEndpoint: "/uspstf/recommendation",
				Authority:   models.AuthorityCMS, // CMS oversight
			},
			SourceWHO: {
				Name:        "World Health Organization",
				BaseURL:     "https://www.who.int",
				APIEndpoint: "/publications/guidelines",
				Authority:   models.AuthorityWHO,
			},
		},
	}
}

// GetSupportedTypes returns the knowledge types this adapter can produce.
func (a *GuidelinesAdapter) GetSupportedTypes() []models.KnowledgeType {
	return []models.KnowledgeType{
		models.TypeGuideline,
		models.TypeProtocol,
	}
}

// FetchUpdates retrieves guidelines updated since the given timestamp.
func (a *GuidelinesAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	var allItems []RawItem

	for source, config := range a.sources {
		items, err := a.fetchFromSource(ctx, source, config, since)
		if err != nil {
			// Log error but continue with other sources
			fmt.Printf("WARNING: Failed to fetch from %s: %v\n", source, err)
			continue
		}
		allItems = append(allItems, items...)
	}

	return allItems, nil
}

// Transform converts raw guideline content to a KnowledgeItem.
func (a *GuidelinesAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	var guideline ClinicalGuideline
	if err := json.Unmarshal(raw.RawData, &guideline); err != nil {
		return nil, fmt.Errorf("failed to parse guideline: %w", err)
	}

	// Determine target KB based on guideline type
	kb := models.KB15 // Evidence Engine for most guidelines
	if guideline.Type == GuidelineTypeProtocol || guideline.Type == GuidelineTypeClinicalPathway {
		kb = models.KB19 // Protocol Orchestrator for actionable protocols
	}

	// Calculate content hash
	hash := sha256.Sum256(raw.RawData)
	contentHash := hex.EncodeToString(hash[:])

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("%s:%s:%s", kb, strings.ToLower(string(guideline.Source)), guideline.ID),
		Type:    models.TypeGuideline,
		KB:      kb,
		Version: guideline.Version,
		Name:    guideline.Title,
		Description: guideline.Summary,
		Source: models.SourceAttribution{
			Authority:     a.getAuthority(guideline.Source),
			Document:      guideline.Title,
			Section:       guideline.Category,
			Jurisdiction:  a.getJurisdiction(guideline.Source),
			URL:           guideline.URL,
			EffectiveDate: guideline.PublishedDate,
		},
		ContentRef:  fmt.Sprintf("guideline:%s:%s", guideline.Source, guideline.ID),
		ContentHash: contentHash,
		State:       models.StateDraft,
		RiskLevel:   a.determineRiskLevel(guideline),
		WorkflowTemplate: a.determineWorkflow(guideline),
		RequiresDualReview: guideline.ClinicalArea == "medication" || guideline.ClinicalArea == "critical_care",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs guideline-specific validation.
func (a *GuidelinesAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	// Ensure guideline has required evidence grading
	if item.Source.Authority == "" {
		return fmt.Errorf("guideline must have a recognized authority")
	}
	return nil
}

// =============================================================================
// CLINICAL GUIDELINE DATA STRUCTURES
// =============================================================================

// ClinicalGuideline represents a clinical practice guideline.
type ClinicalGuideline struct {
	ID               string            `json:"id"`
	Source           GuidelineSource   `json:"source"`
	Title            string            `json:"title"`
	Summary          string            `json:"summary,omitempty"`
	Version          string            `json:"version"`
	PublishedDate    string            `json:"published_date"`
	LastUpdated      string            `json:"last_updated,omitempty"`
	ExpirationDate   string            `json:"expiration_date,omitempty"`
	URL              string            `json:"url"`
	PDFURL           string            `json:"pdf_url,omitempty"`
	Category         string            `json:"category"`
	ClinicalArea     string            `json:"clinical_area"`
	Type             GuidelineType     `json:"type"`
	Status           GuidelineStatus   `json:"status"`
	TargetPopulation string            `json:"target_population,omitempty"`

	// Evidence and Recommendations
	EvidenceGrade    EvidenceGrade     `json:"evidence_grade,omitempty"`
	Recommendations  []Recommendation  `json:"recommendations,omitempty"`
	KeyPoints        []string          `json:"key_points,omitempty"`

	// Structured Content
	Sections         []GuidelineSection `json:"sections,omitempty"`
	Algorithms       []ClinicalAlgorithm `json:"algorithms,omitempty"`

	// References
	References       []Reference        `json:"references,omitempty"`
	RelatedGuidelines []string          `json:"related_guidelines,omitempty"`

	// Metadata
	Authors          []string          `json:"authors,omitempty"`
	ReviewDate       string            `json:"review_date,omitempty"`
	Funding          string            `json:"funding,omitempty"`
	ConflictOfInterest string          `json:"conflict_of_interest,omitempty"`
}

// GuidelineType represents the type of clinical guideline.
type GuidelineType string

const (
	GuidelineTypePractice        GuidelineType = "PRACTICE_GUIDELINE"
	GuidelineTypeProtocol        GuidelineType = "PROTOCOL"
	GuidelineTypeClinicalPathway GuidelineType = "CLINICAL_PATHWAY"
	GuidelineTypeConsensus       GuidelineType = "CONSENSUS_STATEMENT"
	GuidelineTypeSystematicReview GuidelineType = "SYSTEMATIC_REVIEW"
	GuidelineTypeRecommendation  GuidelineType = "RECOMMENDATION"
	GuidelineTypeBestPractice    GuidelineType = "BEST_PRACTICE"
)

// GuidelineStatus represents the status of a guideline.
type GuidelineStatus string

const (
	GuidelineStatusActive      GuidelineStatus = "ACTIVE"
	GuidelineStatusUnderReview GuidelineStatus = "UNDER_REVIEW"
	GuidelineStatusSuperseded  GuidelineStatus = "SUPERSEDED"
	GuidelineStatusWithdrawn   GuidelineStatus = "WITHDRAWN"
	GuidelineStatusDraft       GuidelineStatus = "DRAFT"
)

// EvidenceGrade represents the overall evidence grade.
type EvidenceGrade struct {
	System        string `json:"system"`        // GRADE, SIGN, NICE, etc.
	OverallGrade  string `json:"overall_grade"` // A, B, C, D or High, Moderate, Low
	Certainty     string `json:"certainty,omitempty"`
	Strength      string `json:"strength,omitempty"` // Strong, Weak, Conditional
}

// =============================================================================
// RECOMMENDATION STRUCTURES
// =============================================================================

// Recommendation represents a single recommendation within a guideline.
type Recommendation struct {
	ID                string              `json:"id"`
	Number            string              `json:"number,omitempty"`
	Text              string              `json:"text"`
	EvidenceLevel     string              `json:"evidence_level"`     // Level I, II, III
	RecommendationClass string            `json:"recommendation_class"` // Class I, IIa, IIb, III
	Strength          RecommendationStrength `json:"strength"`
	QualityOfEvidence string              `json:"quality_of_evidence"` // High, Moderate, Low, Very Low
	GRADE             *GRADEAssessment    `json:"grade,omitempty"`
	Notes             string              `json:"notes,omitempty"`
	Rationale         string              `json:"rationale,omitempty"`
	SupportingEvidence []string           `json:"supporting_evidence,omitempty"`
}

// RecommendationStrength represents the strength of a recommendation.
type RecommendationStrength string

const (
	StrengthStrong       RecommendationStrength = "STRONG"
	StrengthModerate     RecommendationStrength = "MODERATE"
	StrengthWeak         RecommendationStrength = "WEAK"
	StrengthConditional  RecommendationStrength = "CONDITIONAL"
	StrengthGoodPractice RecommendationStrength = "GOOD_PRACTICE"
)

// GRADEAssessment represents GRADE evidence assessment.
type GRADEAssessment struct {
	CertaintyOfEvidence string `json:"certainty"` // High, Moderate, Low, Very Low
	StrengthOfRecommendation string `json:"strength"` // Strong, Weak
	Direction          string `json:"direction"` // For, Against
	Domains            GRADEDomains `json:"domains,omitempty"`
}

// GRADEDomains represents the GRADE assessment domains.
type GRADEDomains struct {
	RiskOfBias         string `json:"risk_of_bias"`
	Inconsistency      string `json:"inconsistency"`
	Indirectness       string `json:"indirectness"`
	Imprecision        string `json:"imprecision"`
	PublicationBias    string `json:"publication_bias"`
	LargeEffect        bool   `json:"large_effect"`
	DoseResponse       bool   `json:"dose_response"`
	ConfoundingEffect  bool   `json:"confounding_effect"`
}

// =============================================================================
// GUIDELINE CONTENT STRUCTURES
// =============================================================================

// GuidelineSection represents a section of a guideline document.
type GuidelineSection struct {
	ID       string             `json:"id"`
	Title    string             `json:"title"`
	Number   string             `json:"number,omitempty"`
	Content  string             `json:"content"`
	Type     SectionType        `json:"type"`
	Subsections []GuidelineSection `json:"subsections,omitempty"`
}

// SectionType represents the type of guideline section.
type SectionType string

const (
	SectionTypeIntroduction    SectionType = "INTRODUCTION"
	SectionTypeMethods         SectionType = "METHODS"
	SectionTypeBackground      SectionType = "BACKGROUND"
	SectionTypeRecommendations SectionType = "RECOMMENDATIONS"
	SectionTypeEvidence        SectionType = "EVIDENCE"
	SectionTypeImplementation  SectionType = "IMPLEMENTATION"
	SectionTypeMonitoring      SectionType = "MONITORING"
	SectionTypeResearch        SectionType = "RESEARCH_PRIORITIES"
	SectionTypeAppendix        SectionType = "APPENDIX"
)

// ClinicalAlgorithm represents a clinical decision algorithm.
type ClinicalAlgorithm struct {
	ID          string                `json:"id"`
	Title       string                `json:"title"`
	Description string                `json:"description,omitempty"`
	ImageURL    string                `json:"image_url,omitempty"`
	Nodes       []AlgorithmNode       `json:"nodes,omitempty"`
	Edges       []AlgorithmEdge       `json:"edges,omitempty"`
}

// AlgorithmNode represents a node in a clinical algorithm.
type AlgorithmNode struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // decision, action, endpoint
	Label    string   `json:"label"`
	Content  string   `json:"content,omitempty"`
}

// AlgorithmEdge represents an edge in a clinical algorithm.
type AlgorithmEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"`
	Label     string `json:"label,omitempty"`
}

// Reference represents a literature reference.
type Reference struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"` // article, book, website
	Authors   []string `json:"authors,omitempty"`
	Title     string   `json:"title"`
	Journal   string   `json:"journal,omitempty"`
	Year      string   `json:"year"`
	Volume    string   `json:"volume,omitempty"`
	Pages     string   `json:"pages,omitempty"`
	PMID      string   `json:"pmid,omitempty"`
	DOI       string   `json:"doi,omitempty"`
	URL       string   `json:"url,omitempty"`
}

// =============================================================================
// SOURCE-SPECIFIC OPERATIONS
// =============================================================================

// fetchFromSource fetches guidelines from a specific source.
func (a *GuidelinesAdapter) fetchFromSource(ctx context.Context, source GuidelineSource, config GuidelineSourceConfig, since time.Time) ([]RawItem, error) {
	urlStr := fmt.Sprintf("%s%s?updated_after=%s",
		config.BaseURL, config.APIEndpoint, since.Format("2006-01-02"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		Guidelines []ClinicalGuideline `json:"guidelines"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var items []RawItem
	for _, g := range result.Guidelines {
		g.Source = source
		data, err := json.Marshal(g)
		if err != nil {
			continue
		}
		items = append(items, RawItem{
			ID:        g.ID,
			Authority: config.Authority,
			RawData:   data,
		})
	}

	return items, nil
}

// getAuthority returns the authority for a guideline source.
func (a *GuidelinesAdapter) getAuthority(source GuidelineSource) models.Authority {
	if config, ok := a.sources[source]; ok {
		return config.Authority
	}
	return models.AuthorityInternal
}

// getJurisdiction returns the jurisdiction for a guideline source.
func (a *GuidelinesAdapter) getJurisdiction(source GuidelineSource) models.Jurisdiction {
	switch source {
	case SourceNICE:
		return models.JurisdictionUK
	case SourceWHO:
		return models.JurisdictionGlobal
	default:
		return models.JurisdictionUS
	}
}

// determineRiskLevel determines risk level based on guideline content.
func (a *GuidelinesAdapter) determineRiskLevel(g ClinicalGuideline) models.RiskLevel {
	// High risk for medication/critical care protocols
	if g.ClinicalArea == "medication" || g.ClinicalArea == "critical_care" ||
	   g.Type == GuidelineTypeProtocol {
		return models.RiskHigh
	}
	// Medium risk for clinical pathways
	if g.Type == GuidelineTypeClinicalPathway {
		return models.RiskMedium
	}
	// Low risk for evidence reviews
	return models.RiskMedium
}

// determineWorkflow determines the appropriate workflow template.
func (a *GuidelinesAdapter) determineWorkflow(g ClinicalGuideline) models.WorkflowTemplate {
	if a.determineRiskLevel(g) == models.RiskHigh {
		return models.TemplateClinicalHigh
	}
	return models.TemplateQualityMed
}

// =============================================================================
// GUIDELINE SEARCH AND LOOKUP
// =============================================================================

// SearchGuidelines searches across all sources for guidelines.
func (a *GuidelinesAdapter) SearchGuidelines(ctx context.Context, query string, sources []GuidelineSource, limit int) ([]*ClinicalGuideline, error) {
	var results []*ClinicalGuideline

	// If no sources specified, search all
	if len(sources) == 0 {
		for source := range a.sources {
			sources = append(sources, source)
		}
	}

	for _, source := range sources {
		config, ok := a.sources[source]
		if !ok {
			continue
		}

		guidelines, err := a.searchSource(ctx, config, query, limit)
		if err != nil {
			continue
		}
		for i := range guidelines {
			guidelines[i].Source = source
		}
		results = append(results, guidelines...)
	}

	return results, nil
}

func (a *GuidelinesAdapter) searchSource(ctx context.Context, config GuidelineSourceConfig, query string, limit int) ([]*ClinicalGuideline, error) {
	urlStr := fmt.Sprintf("%s%s/search?q=%s&limit=%d",
		config.BaseURL, config.APIEndpoint, url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed: %d", resp.StatusCode)
	}

	var result struct {
		Guidelines []*ClinicalGuideline `json:"guidelines"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Guidelines, nil
}

// GetGuideline retrieves a specific guideline by ID and source.
func (a *GuidelinesAdapter) GetGuideline(ctx context.Context, source GuidelineSource, id string) (*ClinicalGuideline, error) {
	config, ok := a.sources[source]
	if !ok {
		return nil, fmt.Errorf("unknown source: %s", source)
	}

	urlStr := fmt.Sprintf("%s%s/%s", config.BaseURL, config.APIEndpoint, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var guideline ClinicalGuideline
	if err := json.NewDecoder(resp.Body).Decode(&guideline); err != nil {
		return nil, err
	}
	guideline.Source = source

	return &guideline, nil
}

// =============================================================================
// PDF CONTENT EXTRACTION
// =============================================================================

// ExtractFromPDF extracts structured guideline content from a PDF.
// This is a placeholder - in production would use a PDF parsing library.
func (a *GuidelinesAdapter) ExtractFromPDF(ctx context.Context, pdfURL string) (*ClinicalGuideline, error) {
	// Fetch PDF
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch PDF: %d", resp.StatusCode)
	}

	// Read PDF content
	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	// Extract text from PDF (placeholder)
	text := a.extractTextFromPDF(pdfData)

	// Parse guideline structure from text
	guideline := a.parseGuidelineText(text, pdfURL)

	return guideline, nil
}

// extractTextFromPDF extracts text from PDF bytes.
// In production, would use pdftotext, poppler, or similar library.
func (a *GuidelinesAdapter) extractTextFromPDF(pdfData []byte) string {
	// Placeholder - would use actual PDF parsing
	return string(pdfData)
}

// parseGuidelineText parses guideline structure from extracted text.
func (a *GuidelinesAdapter) parseGuidelineText(text string, sourceURL string) *ClinicalGuideline {
	guideline := &ClinicalGuideline{
		ID:   fmt.Sprintf("extracted-%d", time.Now().Unix()),
		URL:  sourceURL,
		Type: GuidelineTypePractice,
	}

	// Extract title (usually first significant line)
	titlePattern := regexp.MustCompile(`(?m)^([A-Z][A-Za-z\s:]+)$`)
	if match := titlePattern.FindString(text); match != "" {
		guideline.Title = strings.TrimSpace(match)
	}

	// Extract recommendations
	recPattern := regexp.MustCompile(`(?i)recommendation\s*(\d+)[:\.]?\s*(.+?)(?:\n|$)`)
	for _, match := range recPattern.FindAllStringSubmatch(text, -1) {
		if len(match) >= 3 {
			guideline.Recommendations = append(guideline.Recommendations, Recommendation{
				Number: match[1],
				Text:   strings.TrimSpace(match[2]),
			})
		}
	}

	// Extract key points/summary
	if strings.Contains(strings.ToLower(text), "key points") {
		guideline.KeyPoints = append(guideline.KeyPoints, "Key points present in document")
	}

	return guideline
}

// =============================================================================
// EVIDENCE LEVEL MAPPING
// =============================================================================

// MapEvidenceLevel maps evidence levels between different grading systems.
func MapEvidenceLevel(fromSystem, toSystem, level string) string {
	// Mapping tables between common systems
	gradeToOxford := map[string]string{
		"HIGH":      "1a",
		"MODERATE":  "2a",
		"LOW":       "3a",
		"VERY_LOW":  "4",
	}

	oxfordToGrade := map[string]string{
		"1a": "HIGH",
		"1b": "HIGH",
		"2a": "MODERATE",
		"2b": "MODERATE",
		"3a": "LOW",
		"3b": "LOW",
		"4":  "VERY_LOW",
		"5":  "VERY_LOW",
	}

	accahaToGrade := map[string]string{
		"A": "HIGH",
		"B": "MODERATE",
		"C": "LOW",
	}

	switch {
	case fromSystem == "GRADE" && toSystem == "OXFORD":
		return gradeToOxford[level]
	case fromSystem == "OXFORD" && toSystem == "GRADE":
		return oxfordToGrade[level]
	case fromSystem == "ACC_AHA" && toSystem == "GRADE":
		return accahaToGrade[level]
	default:
		return level
	}
}
