// Package clients provides HTTP clients for KB services.
//
// KB15HTTPClient implements the KB15Client interface for KB-15 Evidence Engine Service.
// It provides clinical evidence search, GRADE assessments, and evidence packaging.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// KB-15 is a RUNTIME KB - called during workflow to fetch evidence for recommendations.
// It provides GRADE grading for KB-19 protocol recommendations.
//
// Key Capabilities:
// 1. Evidence Search: Query clinical literature by condition, intervention, outcome
// 2. GRADE Assessment: Get quality grades (High, Moderate, Low, Very Low)
// 3. Evidence Envelope: Pre-packaged evidence for protocol decision nodes
// 4. Citation Generation: Formatted citations for clinical documentation
//
// Connects to: http://localhost:8095 (Docker: kb15-evidence-engine)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// KB15HTTPClient implements KB15Client by calling the KB-15 Evidence Engine Service REST API.
type KB15HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB15HTTPClient creates a new KB-15 HTTP client.
func NewKB15HTTPClient(baseURL string) *KB15HTTPClient {
	return &KB15HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB15HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB15HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB15HTTPClient {
	return &KB15HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB15Client Interface Implementation (RUNTIME)
// ============================================================================

// SearchEvidence searches for clinical evidence based on query criteria.
// Uses PICO framework (Population, Intervention, Comparison, Outcome).
//
// Parameters:
// - query: EvidenceQuery containing search criteria (condition, intervention, outcome, etc.)
//
// Returns:
// - EvidenceResult containing matched evidence items, each with quality assessment
func (c *KB15HTTPClient) SearchEvidence(
	ctx context.Context,
	query contracts.EvidenceQuery,
) (*contracts.EvidenceResult, error) {

	req := kb15SearchRequest{
		ConditionCodes:    query.ConditionCodes,
		InterventionCodes: query.InterventionCodes,
		OutcomeCodes:      query.OutcomeCodes,
		StudyTypes:        query.StudyTypes,
		MinGrade:          query.MinGrade,
		MaxResults:        query.MaxResults,
	}

	// Convert Period to inline DateRange struct if provided
	if query.DateRange != nil {
		if query.DateRange.Start != nil {
			req.DateRange.Start = *query.DateRange.Start
		}
		if query.DateRange.End != nil {
			req.DateRange.End = *query.DateRange.End
		}
	}

	resp, err := c.callKB15(ctx, "/api/v1/evidence/search", req)
	if err != nil {
		return nil, fmt.Errorf("failed to search evidence: %w", err)
	}

	var result kb15SearchResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	items := make([]contracts.EvidenceItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, contracts.EvidenceItem{
			EvidenceID:      item.EvidenceID,
			Title:           item.Title,
			Authors:         item.Authors,
			PublicationDate: item.PublicationDate,
			Journal:         item.Journal,
			StudyType:       item.StudyType,
			SampleSize:      item.SampleSize,
			Summary:         item.Summary,
			GRADELevel:      item.GRADELevel,
			DOI:             item.DOI,
			PMID:            item.PMID,
			Relevance:       item.Relevance,
		})
	}

	return &contracts.EvidenceResult{
		Query:        query,
		TotalMatches: result.TotalMatches,
		Items:        items,
		SearchTime:   result.SearchTime,
	}, nil
}

// GetEvidenceGrading returns GRADE assessment for a specific evidence item.
// GRADE levels: High, Moderate, Low, Very Low
//
// GRADE Factors assessed:
// - Risk of bias
// - Inconsistency
// - Indirectness
// - Imprecision
// - Publication bias
func (c *KB15HTTPClient) GetEvidenceGrading(
	ctx context.Context,
	evidenceID string,
) (*contracts.GRADEAssessment, error) {

	endpoint := fmt.Sprintf("/api/v1/evidence/%s/grade", evidenceID)
	resp, err := c.callKB15Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get GRADE assessment: %w", err)
	}

	var result kb15GRADEResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse GRADE response: %w", err)
	}

	domains := make([]contracts.GRADEDomain, 0, len(result.Assessment.Domains))
	for _, d := range result.Assessment.Domains {
		domains = append(domains, contracts.GRADEDomain{
			Name:        d.Name,
			Rating:      d.Rating,
			Concern:     d.Concern,
			Explanation: d.Explanation,
		})
	}

	return &contracts.GRADEAssessment{
		EvidenceID:        evidenceID,
		OverallGrade:      result.Assessment.OverallGrade,
		Certainty:         result.Assessment.Certainty,
		Domains:           domains,
		Recommendation:    result.Assessment.Recommendation,
		StrengthOfRec:     result.Assessment.StrengthOfRec,
		LastAssessedAt:    result.Assessment.LastAssessedAt,
		AssessedBy:        result.Assessment.AssessedBy,
		GRADEVersion:      result.Assessment.GRADEVersion,
	}, nil
}

// GetEvidenceEnvelope returns pre-packaged evidence for a protocol decision node.
// Used by KB-19 Protocol Orchestration to attach evidence to recommendations.
//
// Parameters:
// - protocolID: Identifier of the clinical protocol (e.g., "sepsis-bundle-v3")
// - decisionNodeID: Specific decision point in the protocol (e.g., "lactate-target-decision")
//
// Returns:
// - EvidenceEnvelope containing relevant evidence items and summary for the decision
func (c *KB15HTTPClient) GetEvidenceEnvelope(
	ctx context.Context,
	protocolID string,
	decisionNodeID string,
) (*contracts.EvidenceEnvelope, error) {

	req := kb15EnvelopeRequest{
		ProtocolID:     protocolID,
		DecisionNodeID: decisionNodeID,
	}

	resp, err := c.callKB15(ctx, "/api/v1/evidence/envelope", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence envelope: %w", err)
	}

	var result kb15EnvelopeResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse envelope response: %w", err)
	}

	items := make([]contracts.EvidenceItem, 0, len(result.Envelope.Items))
	for _, item := range result.Envelope.Items {
		items = append(items, contracts.EvidenceItem{
			EvidenceID:      item.EvidenceID,
			Title:           item.Title,
			Authors:         item.Authors,
			PublicationDate: item.PublicationDate,
			Journal:         item.Journal,
			StudyType:       item.StudyType,
			Summary:         item.Summary,
			GRADELevel:      item.GRADELevel,
			DOI:             item.DOI,
		})
	}

	return &contracts.EvidenceEnvelope{
		ProtocolID:       protocolID,
		DecisionNodeID:   decisionNodeID,
		Items:            items,
		SummaryStatement: result.Envelope.SummaryStatement,
		OverallGrade:     result.Envelope.OverallGrade,
		StrengthOfRec:    result.Envelope.StrengthOfRec,
		LastUpdated:      result.Envelope.LastUpdated,
		GuidelineSources: result.Envelope.GuidelineSources,
	}, nil
}

// GenerateCitation generates a formatted citation for an evidence item.
// Supports multiple citation styles.
//
// Parameters:
// - evidenceID: The evidence item to cite
// - style: Citation style ("ama", "apa", "vancouver", "harvard")
func (c *KB15HTTPClient) GenerateCitation(
	ctx context.Context,
	evidenceID string,
	style string,
) (string, error) {

	req := kb15CitationRequest{
		EvidenceID: evidenceID,
		Style:      style,
	}

	resp, err := c.callKB15(ctx, "/api/v1/evidence/citation", req)
	if err != nil {
		return "", fmt.Errorf("failed to generate citation: %w", err)
	}

	var result kb15CitationResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse citation response: %w", err)
	}

	return result.Citation, nil
}

// GetSystematicReviews returns relevant systematic reviews for a condition.
// Systematic reviews are the highest level of clinical evidence.
//
// Parameters:
// - conditionCode: SNOMED or ICD code for the condition
func (c *KB15HTTPClient) GetSystematicReviews(
	ctx context.Context,
	conditionCode string,
) ([]contracts.SystematicReview, error) {

	endpoint := fmt.Sprintf("/api/v1/evidence/systematic-reviews/%s", conditionCode)
	resp, err := c.callKB15Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get systematic reviews: %w", err)
	}

	var result kb15SystematicReviewsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse systematic reviews response: %w", err)
	}

	reviews := make([]contracts.SystematicReview, 0, len(result.Reviews))
	for _, r := range result.Reviews {
		reviews = append(reviews, contracts.SystematicReview{
			ReviewID:        r.ReviewID,
			Title:           r.Title,
			Authors:         r.Authors,
			PublicationDate: r.PublicationDate,
			CochraneID:      r.CochraneID,
			StudiesIncluded: r.StudiesIncluded,
			Participants:    r.Participants,
			Summary:         r.Summary,
			MainFindings:    r.MainFindings,
			AuthorConclusion: r.AuthorConclusion,
			GRADELevel:      r.GRADELevel,
			LastUpdated:     r.LastUpdated,
			DOI:             r.DOI,
		})
	}

	return reviews, nil
}

// GetEvidenceByPMID retrieves evidence by PubMed ID.
func (c *KB15HTTPClient) GetEvidenceByPMID(
	ctx context.Context,
	pmid string,
) (*contracts.EvidenceItem, error) {

	endpoint := fmt.Sprintf("/api/v1/evidence/pmid/%s", pmid)
	resp, err := c.callKB15Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence by PMID: %w", err)
	}

	var result kb15EvidenceResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse evidence response: %w", err)
	}

	return &contracts.EvidenceItem{
		EvidenceID:      result.Item.EvidenceID,
		Title:           result.Item.Title,
		Authors:         result.Item.Authors,
		PublicationDate: result.Item.PublicationDate,
		Journal:         result.Item.Journal,
		StudyType:       result.Item.StudyType,
		SampleSize:      result.Item.SampleSize,
		Summary:         result.Item.Summary,
		GRADELevel:      result.Item.GRADELevel,
		DOI:             result.Item.DOI,
		PMID:            result.Item.PMID,
	}, nil
}

// GetGuidelineEvidence retrieves evidence supporting a specific guideline recommendation.
func (c *KB15HTTPClient) GetGuidelineEvidence(
	ctx context.Context,
	guidelineID string,
	recommendationID string,
) (*contracts.GuidelineEvidence, error) {

	req := kb15GuidelineEvidenceRequest{
		GuidelineID:      guidelineID,
		RecommendationID: recommendationID,
	}

	resp, err := c.callKB15(ctx, "/api/v1/evidence/guideline", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get guideline evidence: %w", err)
	}

	var result kb15GuidelineEvidenceResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse guideline evidence response: %w", err)
	}

	items := make([]contracts.EvidenceItem, 0, len(result.Evidence.SupportingItems))
	for _, item := range result.Evidence.SupportingItems {
		items = append(items, contracts.EvidenceItem{
			EvidenceID:      item.EvidenceID,
			Title:           item.Title,
			Authors:         item.Authors,
			PublicationDate: item.PublicationDate,
			Journal:         item.Journal,
			StudyType:       item.StudyType,
			Summary:         item.Summary,
			GRADELevel:      item.GRADELevel,
			DOI:             item.DOI,
		})
	}

	return &contracts.GuidelineEvidence{
		GuidelineID:           guidelineID,
		RecommendationID:      recommendationID,
		RecommendationText:    result.Evidence.RecommendationText,
		RecommendationGrade:   result.Evidence.RecommendationGrade,
		StrengthOfRecommendation: result.Evidence.StrengthOfRecommendation,
		SupportingItems:       items,
		EvidenceSummary:       result.Evidence.EvidenceSummary,
		GuidelineOrganization: result.Evidence.GuidelineOrganization,
		PublicationYear:       result.Evidence.PublicationYear,
	}, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB15HTTPClient) callKB15(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	url := c.baseURL + endpoint

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-15 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *KB15HTTPClient) callKB15Get(ctx context.Context, endpoint string) ([]byte, error) {
	url := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-15 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-15 service is healthy.
func (c *KB15HTTPClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-15 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// KB-15 Request/Response Types (internal)
// ============================================================================

type kb15SearchRequest struct {
	ConditionCodes    []string `json:"condition_codes"`
	InterventionCodes []string `json:"intervention_codes"`
	OutcomeCodes      []string `json:"outcome_codes"`
	StudyTypes        []string `json:"study_types"` // RCT, meta-analysis, cohort, etc.
	MinGrade          string   `json:"min_grade"`   // Minimum GRADE level
	MaxResults        int      `json:"max_results"`
	DateRange         struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"date_range"`
}

type kb15SearchResponse struct {
	TotalMatches int                `json:"total_matches"`
	Items        []kb15EvidenceItem `json:"items"`
	SearchTime   int64              `json:"search_time_ms"`
}

type kb15EvidenceItem struct {
	EvidenceID      string    `json:"evidence_id"`
	Title           string    `json:"title"`
	Authors         []string  `json:"authors"`
	PublicationDate time.Time `json:"publication_date"`
	Journal         string    `json:"journal"`
	StudyType       string    `json:"study_type"`
	SampleSize      int       `json:"sample_size"`
	Summary         string    `json:"summary"`
	GRADELevel      string    `json:"grade_level"`
	DOI             string    `json:"doi"`
	PMID            string    `json:"pmid"`
	Relevance       float64   `json:"relevance"`
}

type kb15EvidenceResponse struct {
	Item kb15EvidenceItem `json:"item"`
}

type kb15GRADEResponse struct {
	Assessment kb15GRADEAssessment `json:"assessment"`
}

type kb15GRADEAssessment struct {
	OverallGrade   string           `json:"overall_grade"`
	Certainty      string           `json:"certainty"`
	Domains        []kb15GRADEDomain `json:"domains"`
	Recommendation string           `json:"recommendation"`
	StrengthOfRec  string           `json:"strength_of_recommendation"`
	LastAssessedAt time.Time        `json:"last_assessed_at"`
	AssessedBy     string           `json:"assessed_by"`
	GRADEVersion   string           `json:"grade_version"`
}

type kb15GRADEDomain struct {
	Name        string `json:"name"` // risk_of_bias, inconsistency, indirectness, imprecision, publication_bias
	Rating      string `json:"rating"`
	Concern     string `json:"concern"` // serious, very_serious, not_serious
	Explanation string `json:"explanation"`
}

type kb15EnvelopeRequest struct {
	ProtocolID     string `json:"protocol_id"`
	DecisionNodeID string `json:"decision_node_id"`
}

type kb15EnvelopeResponse struct {
	Envelope kb15EvidenceEnvelope `json:"envelope"`
}

type kb15EvidenceEnvelope struct {
	Items            []kb15EvidenceItem `json:"items"`
	SummaryStatement string             `json:"summary_statement"`
	OverallGrade     string             `json:"overall_grade"`
	StrengthOfRec    string             `json:"strength_of_recommendation"`
	LastUpdated      time.Time          `json:"last_updated"`
	GuidelineSources []string           `json:"guideline_sources"`
}

type kb15CitationRequest struct {
	EvidenceID string `json:"evidence_id"`
	Style      string `json:"style"` // ama, apa, vancouver, harvard
}

type kb15CitationResponse struct {
	Citation string `json:"citation"`
}

type kb15SystematicReviewsResponse struct {
	Reviews []kb15SystematicReview `json:"reviews"`
}

type kb15SystematicReview struct {
	ReviewID         string    `json:"review_id"`
	Title            string    `json:"title"`
	Authors          []string  `json:"authors"`
	PublicationDate  time.Time `json:"publication_date"`
	CochraneID       string    `json:"cochrane_id"`
	StudiesIncluded  int       `json:"studies_included"`
	Participants     int       `json:"participants"`
	Summary          string    `json:"summary"`
	MainFindings     string    `json:"main_findings"`
	AuthorConclusion string    `json:"author_conclusion"`
	GRADELevel       string    `json:"grade_level"`
	LastUpdated      time.Time `json:"last_updated"`
	DOI              string    `json:"doi"`
}

type kb15GuidelineEvidenceRequest struct {
	GuidelineID      string `json:"guideline_id"`
	RecommendationID string `json:"recommendation_id"`
}

type kb15GuidelineEvidenceResponse struct {
	Evidence kb15GuidelineEvidence `json:"evidence"`
}

type kb15GuidelineEvidence struct {
	RecommendationText       string             `json:"recommendation_text"`
	RecommendationGrade      string             `json:"recommendation_grade"`
	StrengthOfRecommendation string             `json:"strength_of_recommendation"`
	SupportingItems          []kb15EvidenceItem `json:"supporting_items"`
	EvidenceSummary          string             `json:"evidence_summary"`
	GuidelineOrganization    string             `json:"guideline_organization"`
	PublicationYear          int                `json:"publication_year"`
}
