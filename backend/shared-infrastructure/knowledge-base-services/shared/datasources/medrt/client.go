// Package medrt provides a client for the MED-RT (Medication Reference Terminology) API.
// MED-RT is part of the NLM RxClass API and provides mechanism-level drug signals.
//
// DESIGN PRINCIPLE: "DDI ≠ NLP problem. Use structured graph queries for interaction mechanisms."
//
// This client provides:
// - Class-level interaction signals (QT prolongation, bleeding risk, CNS depression)
// - Drug-disease relationships (contraindications)
// - Mechanism of action and physiologic effects
// - Class membership for inheritance
//
// NOTE: This uses the RxClass API (https://rxnav.nlm.nih.gov/RxClassAPIs.html),
// NOT the discontinued Drug Interaction API.
package medrt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// =============================================================================
// MED-RT SIGNAL TYPES
// =============================================================================

// SignalType represents the type of clinical signal from MED-RT
type SignalType string

const (
	// SignalQTProlongation - Drug associated with QT interval prolongation
	SignalQTProlongation SignalType = "QT_PROLONGATION"

	// SignalBleedingRisk - Drug associated with increased bleeding risk
	SignalBleedingRisk SignalType = "BLEEDING_RISK"

	// SignalCNSDepression - Drug causes CNS depression
	SignalCNSDepression SignalType = "CNS_DEPRESSION"

	// SignalSerotonergic - Drug has serotonergic activity
	SignalSerotonergic SignalType = "SEROTONERGIC"

	// SignalAnticholinergic - Drug has anticholinergic activity
	SignalAnticholinergic SignalType = "ANTICHOLINERGIC"

	// SignalNephrotoxic - Drug is potentially nephrotoxic
	SignalNephrotoxic SignalType = "NEPHROTOXIC"

	// SignalHepatotoxic - Drug is potentially hepatotoxic
	SignalHepatotoxic SignalType = "HEPATOTOXIC"

	// SignalContraindication - Drug has contraindication with condition
	SignalContraindication SignalType = "CONTRAINDICATION"

	// SignalMayTreat - Drug may treat a condition
	SignalMayTreat SignalType = "MAY_TREAT"

	// SignalMayPrevent - Drug may prevent a condition
	SignalMayPrevent SignalType = "MAY_PREVENT"
)

// RelationshipType maps MED-RT relationship types
type RelationshipType string

const (
	RelaCIWith         RelationshipType = "CI_with"          // Contraindicated with
	RelaMayTreat       RelationshipType = "may_treat"        // May treat
	RelaMayPrevent     RelationshipType = "may_prevent"      // May prevent
	RelaMayDiagnose    RelationshipType = "may_diagnose"     // May diagnose
	RelaInducedBy      RelationshipType = "induced_by"       // Induced by
	RelaHasMechanism   RelationshipType = "has_mechanism"    // Has mechanism of action
	RelaHasEffect      RelationshipType = "has_PE"           // Has physiologic effect
	RelaHasIngredient  RelationshipType = "has_ingredient"   // Has ingredient
	RelaMemberOf       RelationshipType = "member_of"        // Member of class
)

// =============================================================================
// MED-RT SIGNAL MODEL
// =============================================================================

// MEDRTSignal represents a mechanism-level signal from MED-RT
type MEDRTSignal struct {
	// Drug identification
	DrugRxCUI  string `json:"rxcui"`
	DrugName   string `json:"drugName"`

	// Signal details
	SignalType   SignalType       `json:"signalType"`
	RelationType RelationshipType `json:"relationType"`

	// Class information (if signal is from class membership)
	ClassRxCUI string `json:"classRxcui,omitempty"`
	ClassName  string `json:"className,omitempty"`
	ClassType  string `json:"classType,omitempty"` // ATC, EPC, MOA, PE, etc.

	// Disease/condition (for CI_with, may_treat, etc.)
	DiseaseCode string `json:"diseaseCode,omitempty"` // ICD-10 or SNOMED
	DiseaseName string `json:"diseaseName,omitempty"`

	// Metadata
	Source     string    `json:"source"` // Always "MEDRT"
	RetrievedAt time.Time `json:"retrievedAt"`
}

// DrugClassRelationship represents a drug's relationship to a class
type DrugClassRelationship struct {
	ClassID     string           `json:"classId"`
	ClassName   string           `json:"className"`
	ClassType   string           `json:"classType"`
	RelationType RelationshipType `json:"relationType"`
	MinConcept  string           `json:"minConcept,omitempty"` // Most specific concept
}

// DiseaseRelationship represents a drug-disease relationship
type DiseaseRelationship struct {
	RelationType RelationshipType `json:"relationType"`
	DiseaseCode  string           `json:"diseaseCode"`
	DiseaseName  string           `json:"diseaseName"`
	Source       string           `json:"source"`
}

// =============================================================================
// MED-RT CLIENT
// =============================================================================

// Client provides access to MED-RT via the RxClass API
type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      Cache
	log        *logrus.Entry
	mu         sync.RWMutex
}

// Cache interface for caching API responses
type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
}

// Config holds client configuration
type Config struct {
	BaseURL    string        `json:"baseUrl"`
	Timeout    time.Duration `json:"timeout"`
	CacheTTL   time.Duration `json:"cacheTTL"`
	MaxRetries int           `json:"maxRetries"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		BaseURL:    "https://rxnav.nlm.nih.gov",
		Timeout:    30 * time.Second,
		CacheTTL:   24 * time.Hour, // MED-RT data is stable, cache aggressively
		MaxRetries: 3,
	}
}

// NewClient creates a new MED-RT client
func NewClient(config Config, cache Cache, log *logrus.Entry) *Client {
	if log == nil {
		log = logrus.NewEntry(logrus.StandardLogger())
	}
	if config.BaseURL == "" {
		config.BaseURL = DefaultConfig().BaseURL
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultConfig().Timeout
	}

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		cache: cache,
		log:   log.WithField("client", "MEDRT"),
	}
}

// =============================================================================
// INTERACTION SIGNAL QUERIES
// =============================================================================

// GetInteractionSignals returns all mechanism-level interaction signals for a drug
func (c *Client) GetInteractionSignals(ctx context.Context, rxcui string) ([]*MEDRTSignal, error) {
	var signals []*MEDRTSignal

	// Get class memberships
	classes, err := c.GetDrugClasses(ctx, rxcui)
	if err != nil {
		c.log.WithError(err).WithField("rxcui", rxcui).Warn("Failed to get drug classes")
		// Continue - we might still get other signals
	}

	// Check for specific high-risk signals based on class membership
	for _, class := range classes {
		classSignals := c.detectClassSignals(class)
		for _, signal := range classSignals {
			signal.DrugRxCUI = rxcui
			signal.RetrievedAt = time.Now()
			signals = append(signals, signal)
		}
	}

	// Get disease contraindications
	contraindications, err := c.GetContraindications(ctx, rxcui)
	if err != nil {
		c.log.WithError(err).WithField("rxcui", rxcui).Warn("Failed to get contraindications")
	} else {
		for _, ci := range contraindications {
			signals = append(signals, &MEDRTSignal{
				DrugRxCUI:    rxcui,
				SignalType:   SignalContraindication,
				RelationType: RelaCIWith,
				DiseaseCode:  ci.DiseaseCode,
				DiseaseName:  ci.DiseaseName,
				Source:       "MEDRT",
				RetrievedAt:  time.Now(),
			})
		}
	}

	return signals, nil
}

// detectClassSignals detects interaction signals based on drug class
func (c *Client) detectClassSignals(class DrugClassRelationship) []*MEDRTSignal {
	var signals []*MEDRTSignal

	// Map class names/types to signals
	classNameLower := strings.ToLower(class.ClassName)

	// QT Prolongation signals
	qtClasses := []string{
		"antiarrhythmic", "class ia antiarrhythmic", "class iii antiarrhythmic",
		"fluoroquinolone", "macrolide", "antipsychotic", "antiemetic",
		"tricyclic antidepressant", "qt prolonging",
	}
	for _, qtClass := range qtClasses {
		if strings.Contains(classNameLower, qtClass) {
			signals = append(signals, &MEDRTSignal{
				SignalType:   SignalQTProlongation,
				RelationType: RelaMemberOf,
				ClassRxCUI:   class.ClassID,
				ClassName:    class.ClassName,
				ClassType:    class.ClassType,
				Source:       "MEDRT",
			})
			break
		}
	}

	// Bleeding Risk signals
	bleedingClasses := []string{
		"anticoagulant", "antiplatelet", "thrombolytic", "nsaid", "aspirin",
		"warfarin", "heparin", "factor xa inhibitor", "direct thrombin inhibitor",
	}
	for _, bleedClass := range bleedingClasses {
		if strings.Contains(classNameLower, bleedClass) {
			signals = append(signals, &MEDRTSignal{
				SignalType:   SignalBleedingRisk,
				RelationType: RelaMemberOf,
				ClassRxCUI:   class.ClassID,
				ClassName:    class.ClassName,
				ClassType:    class.ClassType,
				Source:       "MEDRT",
			})
			break
		}
	}

	// CNS Depression signals
	cnsClasses := []string{
		"benzodiazepine", "opioid", "barbiturate", "sedative", "hypnotic",
		"anxiolytic", "muscle relaxant", "antihistamine", "antipsychotic",
	}
	for _, cnsClass := range cnsClasses {
		if strings.Contains(classNameLower, cnsClass) {
			signals = append(signals, &MEDRTSignal{
				SignalType:   SignalCNSDepression,
				RelationType: RelaMemberOf,
				ClassRxCUI:   class.ClassID,
				ClassName:    class.ClassName,
				ClassType:    class.ClassType,
				Source:       "MEDRT",
			})
			break
		}
	}

	// Serotonergic signals
	seroClasses := []string{
		"ssri", "snri", "maoi", "serotonin", "triptans", "tramadol", "fentanyl",
		"meperidine", "linezolid", "methylene blue",
	}
	for _, seroClass := range seroClasses {
		if strings.Contains(classNameLower, seroClass) {
			signals = append(signals, &MEDRTSignal{
				SignalType:   SignalSerotonergic,
				RelationType: RelaMemberOf,
				ClassRxCUI:   class.ClassID,
				ClassName:    class.ClassName,
				ClassType:    class.ClassType,
				Source:       "MEDRT",
			})
			break
		}
	}

	// Anticholinergic signals
	antiChClasses := []string{
		"anticholinergic", "antimuscarinic", "antihistamine", "tricyclic",
		"antipsychotic", "antiparkinsonian",
	}
	for _, acClass := range antiChClasses {
		if strings.Contains(classNameLower, acClass) {
			signals = append(signals, &MEDRTSignal{
				SignalType:   SignalAnticholinergic,
				RelationType: RelaMemberOf,
				ClassRxCUI:   class.ClassID,
				ClassName:    class.ClassName,
				ClassType:    class.ClassType,
				Source:       "MEDRT",
			})
			break
		}
	}

	// Nephrotoxic signals
	nephroClasses := []string{
		"aminoglycoside", "nsaid", "ace inhibitor", "arb", "contrast", "cisplatin",
		"amphotericin", "cyclosporine", "tacrolimus",
	}
	for _, nephroClass := range nephroClasses {
		if strings.Contains(classNameLower, nephroClass) {
			signals = append(signals, &MEDRTSignal{
				SignalType:   SignalNephrotoxic,
				RelationType: RelaMemberOf,
				ClassRxCUI:   class.ClassID,
				ClassName:    class.ClassName,
				ClassType:    class.ClassType,
				Source:       "MEDRT",
			})
			break
		}
	}

	return signals
}

// =============================================================================
// RXCLASS API METHODS
// =============================================================================

// GetDrugClasses returns all class memberships for a drug
func (c *Client) GetDrugClasses(ctx context.Context, rxcui string) ([]DrugClassRelationship, error) {
	// Check cache
	cacheKey := fmt.Sprintf("medrt:classes:%s", rxcui)
	if c.cache != nil {
		if cached, ok := c.cache.Get(cacheKey); ok {
			var classes []DrugClassRelationship
			if err := json.Unmarshal(cached, &classes); err == nil {
				return classes, nil
			}
		}
	}

	// Call RxClass API
	endpoint := fmt.Sprintf("/REST/rxclass/class/byRxcui.json?rxcui=%s&relaSource=MEDRT", rxcui)
	resp, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get drug classes: %w", err)
	}

	// Parse response
	var result struct {
		RxClassDrugInfoList struct {
			RxClassDrugInfo []struct {
				MinConceptItem struct {
					RxCUI string `json:"rxcui"`
					Name  string `json:"name"`
				} `json:"minConceptItem"`
				RxClassMinConceptItem struct {
					ClassID   string `json:"classId"`
					ClassName string `json:"className"`
					ClassType string `json:"classType"`
				} `json:"rxclassMinConceptItem"`
				Rela string `json:"rela"`
			} `json:"rxclassDrugInfo"`
		} `json:"rxclassDrugInfoList"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var classes []DrugClassRelationship
	for _, info := range result.RxClassDrugInfoList.RxClassDrugInfo {
		classes = append(classes, DrugClassRelationship{
			ClassID:     info.RxClassMinConceptItem.ClassID,
			ClassName:   info.RxClassMinConceptItem.ClassName,
			ClassType:   info.RxClassMinConceptItem.ClassType,
			RelationType: RelationshipType(info.Rela),
			MinConcept:  info.MinConceptItem.Name,
		})
	}

	// Cache result
	if c.cache != nil {
		if data, err := json.Marshal(classes); err == nil {
			c.cache.Set(cacheKey, data, 24*time.Hour)
		}
	}

	return classes, nil
}

// GetContraindications returns disease contraindications for a drug
func (c *Client) GetContraindications(ctx context.Context, rxcui string) ([]DiseaseRelationship, error) {
	// Check cache
	cacheKey := fmt.Sprintf("medrt:ci:%s", rxcui)
	if c.cache != nil {
		if cached, ok := c.cache.Get(cacheKey); ok {
			var ci []DiseaseRelationship
			if err := json.Unmarshal(cached, &ci); err == nil {
				return ci, nil
			}
		}
	}

	// Call RxClass API for CI_with relationships
	endpoint := fmt.Sprintf("/REST/rxclass/class/byRxcui.json?rxcui=%s&relaSource=MEDRT&rela=CI_with", rxcui)
	resp, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get contraindications: %w", err)
	}

	// Parse response
	var result struct {
		RxClassDrugInfoList struct {
			RxClassDrugInfo []struct {
				RxClassMinConceptItem struct {
					ClassID   string `json:"classId"`
					ClassName string `json:"className"`
					ClassType string `json:"classType"`
				} `json:"rxclassMinConceptItem"`
			} `json:"rxclassDrugInfo"`
		} `json:"rxclassDrugInfoList"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var contraindications []DiseaseRelationship
	for _, info := range result.RxClassDrugInfoList.RxClassDrugInfo {
		contraindications = append(contraindications, DiseaseRelationship{
			RelationType: RelaCIWith,
			DiseaseCode:  info.RxClassMinConceptItem.ClassID,
			DiseaseName:  info.RxClassMinConceptItem.ClassName,
			Source:       "MEDRT",
		})
	}

	// Cache result
	if c.cache != nil {
		if data, err := json.Marshal(contraindications); err == nil {
			c.cache.Set(cacheKey, data, 24*time.Hour)
		}
	}

	return contraindications, nil
}

// GetClassMembers returns all drugs in a class
func (c *Client) GetClassMembers(ctx context.Context, classID string) ([]string, error) {
	// Check cache
	cacheKey := fmt.Sprintf("medrt:members:%s", classID)
	if c.cache != nil {
		if cached, ok := c.cache.Get(cacheKey); ok {
			var members []string
			if err := json.Unmarshal(cached, &members); err == nil {
				return members, nil
			}
		}
	}

	// Call RxClass API
	endpoint := fmt.Sprintf("/REST/rxclass/classMembers.json?classId=%s&relaSource=MEDRT",
		url.QueryEscape(classID))
	resp, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get class members: %w", err)
	}

	// Parse response
	var result struct {
		DrugMemberGroup struct {
			DrugMember []struct {
				MinConcept struct {
					RxCUI string `json:"rxcui"`
				} `json:"minConcept"`
			} `json:"drugMember"`
		} `json:"drugMemberGroup"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var members []string
	for _, member := range result.DrugMemberGroup.DrugMember {
		members = append(members, member.MinConcept.RxCUI)
	}

	// Cache result
	if c.cache != nil {
		if data, err := json.Marshal(members); err == nil {
			c.cache.Set(cacheKey, data, 24*time.Hour)
		}
	}

	return members, nil
}

// =============================================================================
// HTTP HELPERS
// =============================================================================

// doRequest performs an HTTP request with retries
func (c *Client) doRequest(ctx context.Context, endpoint string) ([]byte, error) {
	fullURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var body []byte
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	return body, nil
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

// Health checks connectivity to the RxClass API
func (c *Client) Health(ctx context.Context) error {
	// Simple health check - get version
	endpoint := "/REST/version.json"
	_, err := c.doRequest(ctx, endpoint)
	return err
}

// Name returns the client identifier
func (c *Client) Name() string {
	return "MEDRT"
}
