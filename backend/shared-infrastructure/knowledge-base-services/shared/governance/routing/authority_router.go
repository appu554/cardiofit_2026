// Package routing provides authority-based routing for clinical fact extraction.
// This implements the Phase 3b Authority Router that maps SPL LOINC sections
// to their authoritative data sources.
//
// DESIGN PRINCIPLE: "When authoritative sources exist, we ROUTE — we do NOT EXTRACT"
// The goal is 60-80% of KB-1 and KB-4 facts populated WITHOUT LLM extraction.
//
// Phase 3b Implementation - Authority Router
// Document: Phase3b_Ground_Truth_Ingestion.docx
package routing

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cardiofit/shared/datasources"
)

// =============================================================================
// AUTHORITY ROUTER
// =============================================================================

// AuthorityRouter routes SPL sections and fact requests to appropriate authority sources.
// It implements the LOINC section → Authority mapping defined in Phase 3b specification.
type AuthorityRouter struct {
	// Registered authority clients
	authorities map[string]datasources.AuthorityClient
	mu          sync.RWMutex

	// Routing configuration
	loincRoutes    map[string][]RouteConfig   // LOINC code → authority routes
	factTypeRoutes map[datasources.FactType][]RouteConfig // Fact type → authority routes
	contentRoutes  []ContentRouteRule          // Content-based routing rules

	// Fallback behavior
	defaultLLMPolicy datasources.LLMPolicy
}

// RouteConfig defines how to route to an authority
type RouteConfig struct {
	AuthorityName   string                  `json:"authority_name"`
	Priority        int                     `json:"priority"`        // Lower = higher priority
	FactTypes       []datasources.FactType  `json:"fact_types"`
	LLMPolicy       datasources.LLMPolicy   `json:"llm_policy"`
	ContentTriggers []string                `json:"content_triggers,omitempty"` // Keywords that trigger this route
}

// ContentRouteRule defines content-based routing (e.g., "QT" mentions → CredibleMeds)
type ContentRouteRule struct {
	Keywords        []string                `json:"keywords"`         // Trigger keywords
	AuthorityName   string                  `json:"authority_name"`
	FactType        datasources.FactType    `json:"fact_type"`
	CaseSensitive   bool                    `json:"case_sensitive"`
}

// RoutingDecision contains the result of a routing decision
type RoutingDecision struct {
	// Primary authority to use
	PrimaryAuthority string                    `json:"primary_authority"`
	FactType         datasources.FactType      `json:"fact_type"`
	LLMPolicy        datasources.LLMPolicy     `json:"llm_policy"`

	// Fallback authorities if primary fails
	Fallbacks        []string                  `json:"fallbacks,omitempty"`

	// Extraction guidance
	ExtractionMethod string                    `json:"extraction_method"` // "AUTHORITY_LOOKUP", "TABLE_PARSE", "LLM_CONSENSUS"

	// Audit info
	RouteReason      string                    `json:"route_reason"`
	LOINCCode        string                    `json:"loinc_code,omitempty"`
}

// NewAuthorityRouter creates a new router with default LOINC routing configuration
func NewAuthorityRouter() *AuthorityRouter {
	router := &AuthorityRouter{
		authorities:      make(map[string]datasources.AuthorityClient),
		loincRoutes:      make(map[string][]RouteConfig),
		factTypeRoutes:   make(map[datasources.FactType][]RouteConfig),
		contentRoutes:    make([]ContentRouteRule, 0),
		defaultLLMPolicy: datasources.LLMWithConsensus,
	}

	// Initialize default routing configuration
	router.initializeDefaultRoutes()

	return router
}

// =============================================================================
// LOINC ROUTING TABLE (Phase 3b Specification)
// =============================================================================

// LOINC codes for SPL sections
const (
	LOINCNursingMothers       = "34080-2" // NURSING MOTHERS → LactMed
	LOINCGeriatricUse         = "34082-8" // GERIATRIC USE → OHDSI Beers/STOPP
	LOINCClinicalPharmacology = "34090-1" // CLINICAL PHARMACOLOGY → DrugBank PK
	LOINCWarningsPrecautions  = "43685-7" // WARNINGS AND PRECAUTIONS → Multiple
	LOINCDosageAdministration = "34068-7" // DOSAGE AND ADMINISTRATION → Multiple
	LOINCContraindications    = "34070-3" // CONTRAINDICATIONS → KB-4, KB-5
	LOINCDrugInteractions     = "34073-7" // DRUG INTERACTIONS → KB-5
	LOINCBoxedWarning         = "34066-1" // BOXED WARNING → KB-4
	LOINCPregnancy            = "42228-7" // PREGNANCY → KB-4
)

// Authority source names
const (
	AuthorityLactMed      = "LactMed"
	AuthorityCPIC         = "CPIC"
	AuthorityCredibleMeds = "CredibleMeds"
	AuthorityLiverTox     = "LiverTox"
	AuthorityDrugBank     = "DrugBank"
	AuthorityOHDSI        = "OHDSI"
	AuthorityFDASPL       = "FDA_SPL"
)

// initializeDefaultRoutes sets up the default LOINC → Authority routing table
// as specified in Phase3b_Ground_Truth_Ingestion.docx
func (r *AuthorityRouter) initializeDefaultRoutes() {
	// ─────────────────────────────────────────────────────────────────────────
	// LOINC 34080-2: NURSING MOTHERS → LactMed (LLM NEVER)
	// ─────────────────────────────────────────────────────────────────────────
	r.loincRoutes[LOINCNursingMothers] = []RouteConfig{
		{
			AuthorityName: AuthorityLactMed,
			Priority:      1,
			FactTypes:     []datasources.FactType{datasources.FactTypeLactationSafety},
			LLMPolicy:     datasources.LLMNever,
		},
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LOINC 34082-8: GERIATRIC USE → OHDSI Beers/STOPP (LLM NEVER)
	// ─────────────────────────────────────────────────────────────────────────
	r.loincRoutes[LOINCGeriatricUse] = []RouteConfig{
		{
			AuthorityName: AuthorityOHDSI,
			Priority:      1,
			FactTypes:     []datasources.FactType{datasources.FactTypeGeriatricPIM},
			LLMPolicy:     datasources.LLMNever,
		},
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LOINC 34090-1: CLINICAL PHARMACOLOGY → DrugBank PK (LLM NO)
	// ─────────────────────────────────────────────────────────────────────────
	r.loincRoutes[LOINCClinicalPharmacology] = []RouteConfig{
		{
			AuthorityName: AuthorityDrugBank,
			Priority:      1,
			FactTypes: []datasources.FactType{
				datasources.FactTypePKParameters,
				datasources.FactTypeProteinBinding,
				datasources.FactTypeCYPInteraction,
			},
			LLMPolicy: datasources.LLMGapFillOnly,
		},
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LOINC 43685-7: WARNINGS AND PRECAUTIONS → Multiple authorities
	// Content-based routing: QT → CredibleMeds, Hepato → LiverTox
	// ─────────────────────────────────────────────────────────────────────────
	r.loincRoutes[LOINCWarningsPrecautions] = []RouteConfig{
		{
			AuthorityName:   AuthorityCredibleMeds,
			Priority:        1,
			FactTypes:       []datasources.FactType{datasources.FactTypeQTProlongation},
			LLMPolicy:       datasources.LLMNever,
			ContentTriggers: []string{"QT", "torsade", "TdP", "arrhythmia", "prolongation"},
		},
		{
			AuthorityName:   AuthorityLiverTox,
			Priority:        1,
			FactTypes:       []datasources.FactType{datasources.FactTypeHepatotoxicity},
			LLMPolicy:       datasources.LLMNever,
			ContentTriggers: []string{"hepat", "liver", "ALT", "AST", "LFT", "jaundice", "hepatotoxic"},
		},
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LOINC 34068-7: DOSAGE AND ADMINISTRATION → Multiple authorities
	// Content-based: PGx → CPIC, Renal → SPL Tables, Hepatic → SPL Tables
	// ─────────────────────────────────────────────────────────────────────────
	r.loincRoutes[LOINCDosageAdministration] = []RouteConfig{
		{
			AuthorityName:   AuthorityCPIC,
			Priority:        1,
			FactTypes:       []datasources.FactType{datasources.FactTypePharmacogenomics},
			LLMPolicy:       datasources.LLMNever,
			ContentTriggers: []string{"CYP2D6", "CYP2C19", "CYP2C9", "CYP3A4", "genotype", "phenotype", "poor metabolizer", "extensive metabolizer", "pharmacogenomic"},
		},
		{
			AuthorityName:   AuthorityFDASPL,
			Priority:        2,
			FactTypes:       []datasources.FactType{datasources.FactTypeRenalDosing},
			LLMPolicy:       datasources.LLMWithConsensus, // Tables first, LLM for prose gaps
			ContentTriggers: []string{"renal", "kidney", "CrCl", "GFR", "creatinine clearance", "dialysis"},
		},
		{
			AuthorityName:   AuthorityFDASPL,
			Priority:        2,
			FactTypes:       []datasources.FactType{datasources.FactTypeHepaticDosing},
			LLMPolicy:       datasources.LLMWithConsensus, // Tables first, LLM for prose gaps
			ContentTriggers: []string{"hepatic impairment", "Child-Pugh", "liver disease", "cirrhosis"},
		},
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LOINC 34073-7: DRUG INTERACTIONS → DrugBank + CredibleMeds
	// ─────────────────────────────────────────────────────────────────────────
	r.loincRoutes[LOINCDrugInteractions] = []RouteConfig{
		{
			AuthorityName: AuthorityDrugBank,
			Priority:      1,
			FactTypes: []datasources.FactType{
				datasources.FactTypeDrugInteraction,
				datasources.FactTypeCYPInteraction,
				datasources.FactTypeTransporterInteraction,
			},
			LLMPolicy: datasources.LLMGapFillOnly,
		},
		{
			AuthorityName:   AuthorityCredibleMeds,
			Priority:        1,
			FactTypes:       []datasources.FactType{datasources.FactTypeQTProlongation},
			LLMPolicy:       datasources.LLMNever,
			ContentTriggers: []string{"QT", "torsade"},
		},
	}

	// ─────────────────────────────────────────────────────────────────────────
	// Setup fact type → authority routes for direct fact type queries
	// ─────────────────────────────────────────────────────────────────────────
	r.factTypeRoutes[datasources.FactTypeLactationSafety] = []RouteConfig{
		{AuthorityName: AuthorityLactMed, Priority: 1, LLMPolicy: datasources.LLMNever},
	}
	r.factTypeRoutes[datasources.FactTypeQTProlongation] = []RouteConfig{
		{AuthorityName: AuthorityCredibleMeds, Priority: 1, LLMPolicy: datasources.LLMNever},
	}
	r.factTypeRoutes[datasources.FactTypeHepatotoxicity] = []RouteConfig{
		{AuthorityName: AuthorityLiverTox, Priority: 1, LLMPolicy: datasources.LLMNever},
	}
	r.factTypeRoutes[datasources.FactTypePharmacogenomics] = []RouteConfig{
		{AuthorityName: AuthorityCPIC, Priority: 1, LLMPolicy: datasources.LLMNever},
	}
	r.factTypeRoutes[datasources.FactTypeGeriatricPIM] = []RouteConfig{
		{AuthorityName: AuthorityOHDSI, Priority: 1, LLMPolicy: datasources.LLMNever},
	}
	r.factTypeRoutes[datasources.FactTypePKParameters] = []RouteConfig{
		{AuthorityName: AuthorityDrugBank, Priority: 1, LLMPolicy: datasources.LLMGapFillOnly},
	}
	r.factTypeRoutes[datasources.FactTypeCYPInteraction] = []RouteConfig{
		{AuthorityName: AuthorityDrugBank, Priority: 1, LLMPolicy: datasources.LLMGapFillOnly},
	}

	// ─────────────────────────────────────────────────────────────────────────
	// Content-based routing rules (keyword detection)
	// ─────────────────────────────────────────────────────────────────────────
	r.contentRoutes = []ContentRouteRule{
		{
			Keywords:      []string{"QT prolongation", "torsades de pointes", "TdP", "QTc"},
			AuthorityName: AuthorityCredibleMeds,
			FactType:      datasources.FactTypeQTProlongation,
		},
		{
			Keywords:      []string{"hepatotoxicity", "liver injury", "DILI", "ALT elevation"},
			AuthorityName: AuthorityLiverTox,
			FactType:      datasources.FactTypeHepatotoxicity,
		},
		{
			Keywords:      []string{"breastfeeding", "lactation", "nursing", "breast milk", "RID"},
			AuthorityName: AuthorityLactMed,
			FactType:      datasources.FactTypeLactationSafety,
		},
		{
			Keywords:      []string{"CYP2D6", "CYP2C19", "poor metabolizer", "pharmacogenomic"},
			AuthorityName: AuthorityCPIC,
			FactType:      datasources.FactTypePharmacogenomics,
		},
		{
			Keywords:      []string{"elderly", "geriatric", "Beers criteria", "STOPP"},
			AuthorityName: AuthorityOHDSI,
			FactType:      datasources.FactTypeGeriatricPIM,
		},
	}
}

// =============================================================================
// AUTHORITY REGISTRATION
// =============================================================================

// RegisterAuthority registers an authority client with the router
func (r *AuthorityRouter) RegisterAuthority(name string, client datasources.AuthorityClient) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.authorities[name]; exists {
		return fmt.Errorf("authority %s already registered", name)
	}

	r.authorities[name] = client
	return nil
}

// GetAuthority retrieves a registered authority client
func (r *AuthorityRouter) GetAuthority(name string) (datasources.AuthorityClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.authorities[name]
	return client, exists
}

// ListAuthorities returns all registered authority names
func (r *AuthorityRouter) ListAuthorities() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.authorities))
	for name := range r.authorities {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// ROUTING METHODS
// =============================================================================

// RouteByLOINC determines the authority to use for a given LOINC section code
func (r *AuthorityRouter) RouteByLOINC(loincCode string, sectionContent string) *RoutingDecision {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes, exists := r.loincRoutes[loincCode]
	if !exists {
		return &RoutingDecision{
			LLMPolicy:        r.defaultLLMPolicy,
			ExtractionMethod: "LLM_CONSENSUS",
			RouteReason:      fmt.Sprintf("No authority route for LOINC %s", loincCode),
			LOINCCode:        loincCode,
		}
	}

	// Check content-triggered routes first
	for _, route := range routes {
		if len(route.ContentTriggers) > 0 {
			if r.matchesContentTriggers(sectionContent, route.ContentTriggers) {
				return r.buildDecision(route, loincCode, "Content trigger matched")
			}
		}
	}

	// Fall back to first non-content-triggered route
	for _, route := range routes {
		if len(route.ContentTriggers) == 0 {
			return r.buildDecision(route, loincCode, "Default LOINC route")
		}
	}

	// Use first route if all have triggers but none matched
	if len(routes) > 0 {
		return r.buildDecision(routes[0], loincCode, "First available route (no trigger match)")
	}

	return &RoutingDecision{
		LLMPolicy:        r.defaultLLMPolicy,
		ExtractionMethod: "LLM_CONSENSUS",
		RouteReason:      "No matching route found",
		LOINCCode:        loincCode,
	}
}

// RouteByFactType determines the authority for a specific fact type
func (r *AuthorityRouter) RouteByFactType(factType datasources.FactType) *RoutingDecision {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes, exists := r.factTypeRoutes[factType]
	if !exists || len(routes) == 0 {
		return &RoutingDecision{
			FactType:         factType,
			LLMPolicy:        r.defaultLLMPolicy,
			ExtractionMethod: "LLM_CONSENSUS",
			RouteReason:      fmt.Sprintf("No authority route for fact type %s", factType),
		}
	}

	route := routes[0]
	return &RoutingDecision{
		PrimaryAuthority: route.AuthorityName,
		FactType:         factType,
		LLMPolicy:        route.LLMPolicy,
		ExtractionMethod: "AUTHORITY_LOOKUP",
		RouteReason:      fmt.Sprintf("Direct route for %s", factType),
	}
}

// RouteByContent analyzes content and determines appropriate authorities
func (r *AuthorityRouter) RouteByContent(content string) []*RoutingDecision {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var decisions []*RoutingDecision

	for _, rule := range r.contentRoutes {
		if r.matchesKeywords(content, rule.Keywords, rule.CaseSensitive) {
			decisions = append(decisions, &RoutingDecision{
				PrimaryAuthority: rule.AuthorityName,
				FactType:         rule.FactType,
				LLMPolicy:        r.getLLMPolicyForAuthority(rule.AuthorityName),
				ExtractionMethod: "AUTHORITY_LOOKUP",
				RouteReason:      fmt.Sprintf("Content matched keywords for %s", rule.AuthorityName),
			})
		}
	}

	return decisions
}

// =============================================================================
// FACT RETRIEVAL
// =============================================================================

// GetFacts retrieves facts from the appropriate authority based on fact type
func (r *AuthorityRouter) GetFacts(ctx context.Context, rxcui string, factType datasources.FactType) ([]datasources.AuthorityFact, error) {
	decision := r.RouteByFactType(factType)

	if decision.PrimaryAuthority == "" {
		return nil, fmt.Errorf("no authority available for fact type %s", factType)
	}

	client, exists := r.GetAuthority(decision.PrimaryAuthority)
	if !exists {
		return nil, fmt.Errorf("authority %s not registered", decision.PrimaryAuthority)
	}

	return client.GetFacts(ctx, rxcui)
}

// GetFactsForSection retrieves facts relevant to an SPL section
func (r *AuthorityRouter) GetFactsForSection(ctx context.Context, rxcui string, loincCode string, sectionContent string) ([]datasources.AuthorityFact, error) {
	decision := r.RouteByLOINC(loincCode, sectionContent)

	if decision.PrimaryAuthority == "" || decision.ExtractionMethod != "AUTHORITY_LOOKUP" {
		return nil, nil // No authority available, caller should use other extraction methods
	}

	client, exists := r.GetAuthority(decision.PrimaryAuthority)
	if !exists {
		return nil, fmt.Errorf("authority %s not registered", decision.PrimaryAuthority)
	}

	return client.GetFacts(ctx, rxcui)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (r *AuthorityRouter) buildDecision(route RouteConfig, loincCode string, reason string) *RoutingDecision {
	var factType datasources.FactType
	if len(route.FactTypes) > 0 {
		factType = route.FactTypes[0]
	}

	return &RoutingDecision{
		PrimaryAuthority: route.AuthorityName,
		FactType:         factType,
		LLMPolicy:        route.LLMPolicy,
		ExtractionMethod: "AUTHORITY_LOOKUP",
		RouteReason:      reason,
		LOINCCode:        loincCode,
	}
}

func (r *AuthorityRouter) matchesContentTriggers(content string, triggers []string) bool {
	contentLower := strings.ToLower(content)
	for _, trigger := range triggers {
		if strings.Contains(contentLower, strings.ToLower(trigger)) {
			return true
		}
	}
	return false
}

func (r *AuthorityRouter) matchesKeywords(content string, keywords []string, caseSensitive bool) bool {
	if !caseSensitive {
		content = strings.ToLower(content)
	}
	for _, keyword := range keywords {
		kw := keyword
		if !caseSensitive {
			kw = strings.ToLower(keyword)
		}
		if strings.Contains(content, kw) {
			return true
		}
	}
	return false
}

func (r *AuthorityRouter) getLLMPolicyForAuthority(authorityName string) datasources.LLMPolicy {
	// Definitive authorities: LLM = NEVER
	switch authorityName {
	case AuthorityLactMed, AuthorityCPIC, AuthorityCredibleMeds, AuthorityLiverTox, AuthorityOHDSI:
		return datasources.LLMNever
	case AuthorityDrugBank:
		return datasources.LLMGapFillOnly
	default:
		return datasources.LLMWithConsensus
	}
}

// =============================================================================
// ROUTING TABLE ACCESS (for debugging/introspection)
// =============================================================================

// GetLOINCRoutes returns the LOINC routing table
func (r *AuthorityRouter) GetLOINCRoutes() map[string][]RouteConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	copy := make(map[string][]RouteConfig)
	for k, v := range r.loincRoutes {
		copy[k] = append([]RouteConfig{}, v...)
	}
	return copy
}

// GetRoutingDecisionSummary returns a summary of routing for a LOINC code
func (r *AuthorityRouter) GetRoutingDecisionSummary(loincCode string) string {
	decision := r.RouteByLOINC(loincCode, "")
	if decision.PrimaryAuthority == "" {
		return fmt.Sprintf("LOINC %s: No authority route, extraction_method=%s, llm_policy=%s",
			loincCode, decision.ExtractionMethod, decision.LLMPolicy)
	}
	return fmt.Sprintf("LOINC %s: authority=%s, fact_type=%s, llm_policy=%s, extraction_method=%s",
		loincCode, decision.PrimaryAuthority, decision.FactType, decision.LLMPolicy, decision.ExtractionMethod)
}
