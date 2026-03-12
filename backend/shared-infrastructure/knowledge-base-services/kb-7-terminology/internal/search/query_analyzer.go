package search

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"

	"go.uber.org/zap"
)

// QueryAnalyzer analyzes search queries to determine intent and optimize search strategy
type QueryAnalyzer struct {
	cache           cache.EnhancedCache
	logger          *zap.Logger
	metrics         *metrics.Collector
	config          *QueryAnalysisConfig
	patterns        *QueryPatterns
}

// QueryAnalysisConfig holds configuration for query analysis
type QueryAnalysisConfig struct {
	EnableCaching       bool          `json:"enable_caching"`
	CacheTTL           time.Duration `json:"cache_ttl"`
	EnableMLAnalysis   bool          `json:"enable_ml_analysis"`
	ConfidenceThreshold float64      `json:"confidence_threshold"`
	EnablePatternLearning bool       `json:"enable_pattern_learning"`
	MaxAnalysisTime    time.Duration `json:"max_analysis_time"`
}

// QueryPatterns contains regex patterns for query analysis
type QueryPatterns struct {
	// Code patterns
	SNOMEDCodePattern    *regexp.Regexp
	ICD10CodePattern     *regexp.Regexp
	RxNormCodePattern    *regexp.Regexp
	LOINCCodePattern     *regexp.Regexp
	CPTCodePattern       *regexp.Regexp

	// Clinical domain patterns
	DiagnosticPatterns   []*regexp.Regexp
	ProceduralPatterns   []*regexp.Regexp
	MedicationPatterns   []*regexp.Regexp
	LaboratoryPatterns   []*regexp.Regexp
	AnatomyPatterns      []*regexp.Regexp
	SymptomPatterns      []*regexp.Regexp

	// Intent patterns
	LookupPatterns       []*regexp.Regexp
	ExplorationPatterns  []*regexp.Regexp
	ValidationPatterns   []*regexp.Regexp
	TranslationPatterns  []*regexp.Regexp
	BrowsingPatterns     []*regexp.Regexp

	// Language patterns
	NegationPatterns     []*regexp.Regexp
	UncertaintyPatterns  []*regexp.Regexp
	TemporalPatterns     []*regexp.Regexp
	SeverityPatterns     []*regexp.Regexp
}

// QueryAnalysisRequest represents a request for query analysis
type QueryAnalysisRequest struct {
	Query           string                 `json:"query"`
	Context         *AnalysisContext       `json:"context,omitempty"`
	Options         *AnalysisOptions       `json:"options,omitempty"`
	PreviousQueries []string               `json:"previous_queries,omitempty"`
}

// AnalysisContext provides context for query analysis
type AnalysisContext struct {
	UserRole        string            `json:"user_role,omitempty"`
	UserSpecialty   string            `json:"user_specialty,omitempty"`
	SessionContext  map[string]string `json:"session_context,omitempty"`
	GeographicRegion string           `json:"geographic_region,omitempty"`
	Language        string            `json:"language,omitempty"`
	TimezoneOffset  int               `json:"timezone_offset,omitempty"`
}

// AnalysisOptions defines options for query analysis
type AnalysisOptions struct {
	IncludeEntityExtraction bool `json:"include_entity_extraction"`
	IncludeIntentPrediction bool `json:"include_intent_prediction"`
	IncludeDomainClassification bool `json:"include_domain_classification"`
	IncludeQueryExpansion   bool `json:"include_query_expansion"`
	IncludeSpellCheck       bool `json:"include_spell_check"`
	DetailLevel             AnalysisDetailLevel `json:"detail_level"`
}

// AnalysisDetailLevel defines the level of analysis detail
type AnalysisDetailLevel string

const (
	AnalysisDetailBasic      AnalysisDetailLevel = "basic"
	AnalysisDetailStandard   AnalysisDetailLevel = "standard"
	AnalysisDetailComprehensive AnalysisDetailLevel = "comprehensive"
)

// QueryAnalysisResponse contains the results of query analysis
type QueryAnalysisResponse struct {
	// Original query information
	OriginalQuery   string            `json:"original_query"`
	NormalizedQuery string            `json:"normalized_query"`
	QueryTokens     []QueryToken      `json:"query_tokens"`

	// Intent and classification
	DetectedIntent     SearchIntent         `json:"detected_intent"`
	IntentConfidence   float64              `json:"intent_confidence"`
	QueryType          QueryType            `json:"query_type"`
	TypeConfidence     float64              `json:"type_confidence"`
	ClinicalDomains    []DomainClassification `json:"clinical_domains"`

	// Entity extraction
	ExtractedEntities  []*ExtractedEntity   `json:"extracted_entities"`
	DetectedCodes      []*DetectedCode      `json:"detected_codes"`
	MedicalTerms       []*MedicalTerm       `json:"medical_terms"`

	// Query optimization suggestions
	SearchStrategy     RecommendedStrategy  `json:"search_strategy"`
	SuggestedFilters   map[string][]string  `json:"suggested_filters"`
	QueryExpansions    []string             `json:"query_expansions"`
	SpellCorrections   []SpellSuggestion    `json:"spell_corrections"`

	// Semantic analysis
	SemanticAnalysis   *SemanticAnalysis    `json:"semantic_analysis,omitempty"`
	SentimentAnalysis  *SentimentAnalysis   `json:"sentiment_analysis,omitempty"`
	ComplexityAnalysis *ComplexityAnalysis  `json:"complexity_analysis,omitempty"`

	// Processing metadata
	ProcessingTime     time.Duration        `json:"processing_time"`
	AnalysisVersion    string               `json:"analysis_version"`
	CacheHit           bool                 `json:"cache_hit"`
	ConfidenceScore    float64              `json:"confidence_score"`
}

// QueryToken represents a tokenized part of the query
type QueryToken struct {
	Text        string         `json:"text"`
	Type        TokenType      `json:"type"`
	Position    int            `json:"position"`
	Length      int            `json:"length"`
	Confidence  float64        `json:"confidence"`
	Annotations []TokenAnnotation `json:"annotations,omitempty"`
}

// TokenType defines the type of query token
type TokenType string

const (
	TokenTypeWord           TokenType = "word"
	TokenTypeCode           TokenType = "code"
	TokenTypeMedicalTerm    TokenType = "medical_term"
	TokenTypeModifier       TokenType = "modifier"
	TokenTypeOperator       TokenType = "operator"
	TokenTypeStopWord       TokenType = "stopword"
	TokenTypeNumber         TokenType = "number"
	TokenTypePunctuation    TokenType = "punctuation"
)

// TokenAnnotation provides additional information about a token
type TokenAnnotation struct {
	Type        string      `json:"type"`
	Value       interface{} `json:"value"`
	Confidence  float64     `json:"confidence"`
	Source      string      `json:"source"`
}

// DomainClassification represents classification into clinical domains
type DomainClassification struct {
	Domain      string  `json:"domain"`
	Confidence  float64 `json:"confidence"`
	Indicators  []string `json:"indicators"`
}

// ExtractedEntity represents an entity extracted from the query
type ExtractedEntity struct {
	Text        string      `json:"text"`
	Type        EntityType  `json:"type"`
	Subtype     string      `json:"subtype,omitempty"`
	Position    [2]int      `json:"position"` // start, end
	Confidence  float64     `json:"confidence"`
	Value       interface{} `json:"value,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EntityType defines the type of extracted entity
type EntityType string

const (
	EntityTypeCode          EntityType = "code"
	EntityTypeTerm          EntityType = "term"
	EntityTypeSystem        EntityType = "system"
	EntityTypeDomain        EntityType = "domain"
	EntityTypeDosage        EntityType = "dosage"
	EntityTypeFrequency     EntityType = "frequency"
	EntityTypeRoute         EntityType = "route"
	EntityTypeTimeframe     EntityType = "timeframe"
	EntityTypeSeverity      EntityType = "severity"
	EntityTypeAnatomical    EntityType = "anatomical"
)

// DetectedCode represents a detected clinical code
type DetectedCode struct {
	Code        string      `json:"code"`
	System      string      `json:"system"`
	Display     string      `json:"display,omitempty"`
	Confidence  float64     `json:"confidence"`
	Position    [2]int      `json:"position"`
	Validated   bool        `json:"validated"`
}

// MedicalTerm represents a detected medical term
type MedicalTerm struct {
	Term        string      `json:"term"`
	Category    string      `json:"category"`
	Synonyms    []string    `json:"synonyms,omitempty"`
	Definition  string      `json:"definition,omitempty"`
	Confidence  float64     `json:"confidence"`
	Position    [2]int      `json:"position"`
}

// RecommendedStrategy suggests the optimal search strategy
type RecommendedStrategy struct {
	PrimaryStrategy   ClinicalSearchMode   `json:"primary_strategy"`
	FallbackStrategies []ClinicalSearchMode `json:"fallback_strategies"`
	RecommendedSystems []string            `json:"recommended_systems"`
	SortPreference    string               `json:"sort_preference"`
	FilterSuggestions map[string]string    `json:"filter_suggestions"`
	Reasoning         string               `json:"reasoning"`
}

// SpellSuggestion represents a spelling correction suggestion
type SpellSuggestion struct {
	Original    string  `json:"original"`
	Suggestion  string  `json:"suggestion"`
	Confidence  float64 `json:"confidence"`
	EditDistance int    `json:"edit_distance"`
}

// SemanticAnalysis contains semantic analysis results
type SemanticAnalysis struct {
	MainConcepts      []string          `json:"main_concepts"`
	RelatedConcepts   []string          `json:"related_concepts"`
	SemanticDensity   float64           `json:"semantic_density"`
	ConceptualClarity float64           `json:"conceptual_clarity"`
	Relationships     []ConceptRelation `json:"relationships"`
}

// ConceptRelation represents a relationship between concepts
type ConceptRelation struct {
	Source      string  `json:"source"`
	Target      string  `json:"target"`
	Relation    string  `json:"relation"`
	Confidence  float64 `json:"confidence"`
}

// SentimentAnalysis contains sentiment analysis results
type SentimentAnalysis struct {
	OverallSentiment string             `json:"overall_sentiment"`
	Confidence       float64            `json:"confidence"`
	Emotions         map[string]float64 `json:"emotions"`
	Urgency          float64            `json:"urgency"`
}

// ComplexityAnalysis analyzes query complexity
type ComplexityAnalysis struct {
	LexicalComplexity   float64 `json:"lexical_complexity"`
	SyntacticComplexity float64 `json:"syntactic_complexity"`
	SemanticComplexity  float64 `json:"semantic_complexity"`
	OverallComplexity   float64 `json:"overall_complexity"`
	ProcessingDifficulty string `json:"processing_difficulty"`
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer(
	cache cache.EnhancedCache,
	logger *zap.Logger,
	metrics *metrics.Collector,
	config *QueryAnalysisConfig,
) *QueryAnalyzer {
	if config == nil {
		config = DefaultQueryAnalysisConfig()
	}

	analyzer := &QueryAnalyzer{
		cache:    cache,
		logger:   logger,
		metrics:  metrics,
		config:   config,
		patterns: initializeQueryPatterns(),
	}

	return analyzer
}

// DefaultQueryAnalysisConfig returns default configuration for query analysis
func DefaultQueryAnalysisConfig() *QueryAnalysisConfig {
	return &QueryAnalysisConfig{
		EnableCaching:       true,
		CacheTTL:           10 * time.Minute,
		EnableMLAnalysis:   true,
		ConfidenceThreshold: 0.7,
		EnablePatternLearning: true,
		MaxAnalysisTime:    5 * time.Second,
	}
}

// AnalyzeQuery performs comprehensive analysis of a search query
func (qa *QueryAnalyzer) AnalyzeQuery(ctx context.Context, request *QueryAnalysisRequest) (*QueryAnalysisResponse, error) {
	startTime := time.Now()

	// Check cache first
	cacheKey := qa.generateCacheKey(request)
	if qa.config.EnableCaching {
		if cached, exists := qa.getCachedAnalysis(cacheKey); exists {
			cached.ProcessingTime = time.Since(startTime)
			cached.CacheHit = true
			return cached, nil
		}
	}

	qa.logger.Debug("Analyzing query",
		zap.String("query", request.Query),
		zap.String("cache_key", cacheKey),
	)

	// Create analysis context with timeout
	analysisCtx, cancel := context.WithTimeout(ctx, qa.config.MaxAnalysisTime)
	defer cancel()

	// Initialize response
	response := &QueryAnalysisResponse{
		OriginalQuery:   request.Query,
		NormalizedQuery: qa.normalizeQuery(request.Query),
		ProcessingTime:  0,
		AnalysisVersion: "1.0",
		CacheHit:        false,
	}

	// Tokenize query
	response.QueryTokens = qa.tokenizeQuery(response.NormalizedQuery)

	// Extract entities
	if request.Options == nil || request.Options.IncludeEntityExtraction {
		response.ExtractedEntities = qa.extractEntities(analysisCtx, response.NormalizedQuery, response.QueryTokens)
		response.DetectedCodes = qa.detectCodes(response.NormalizedQuery, response.QueryTokens)
		response.MedicalTerms = qa.extractMedicalTerms(response.NormalizedQuery, response.QueryTokens)
	}

	// Classify query intent
	if request.Options == nil || request.Options.IncludeIntentPrediction {
		response.DetectedIntent, response.IntentConfidence = qa.classifyIntent(response.NormalizedQuery, response.QueryTokens, response.ExtractedEntities)
	}

	// Classify query type and domains
	if request.Options == nil || request.Options.IncludeDomainClassification {
		response.QueryType, response.TypeConfidence = qa.classifyQueryType(response.NormalizedQuery, response.QueryTokens, response.ExtractedEntities)
		response.ClinicalDomains = qa.classifyDomains(response.NormalizedQuery, response.QueryTokens)
	}

	// Generate search strategy recommendations
	response.SearchStrategy = qa.recommendSearchStrategy(response)

	// Generate filter suggestions
	response.SuggestedFilters = qa.generateFilterSuggestions(response)

	// Query expansion
	if request.Options != nil && request.Options.IncludeQueryExpansion {
		response.QueryExpansions = qa.generateQueryExpansions(analysisCtx, response.NormalizedQuery)
	}

	// Spell checking
	if request.Options != nil && request.Options.IncludeSpellCheck {
		response.SpellCorrections = qa.generateSpellCorrections(response.NormalizedQuery)
	}

	// Advanced analysis for comprehensive detail level
	if request.Options != nil && request.Options.DetailLevel == AnalysisDetailComprehensive {
		response.SemanticAnalysis = qa.performSemanticAnalysis(response.NormalizedQuery, response.QueryTokens)
		response.SentimentAnalysis = qa.performSentimentAnalysis(response.NormalizedQuery)
		response.ComplexityAnalysis = qa.analyzeComplexity(response.NormalizedQuery, response.QueryTokens)
	}

	// Calculate overall confidence score
	response.ConfidenceScore = qa.calculateOverallConfidence(response)

	// Calculate processing time
	response.ProcessingTime = time.Since(startTime)

	// Cache the response
	if qa.config.EnableCaching {
		qa.cacheAnalysis(cacheKey, response)
	}

	// Record metrics
	qa.recordAnalysisMetrics(request, response)

	qa.logger.Debug("Query analysis completed",
		zap.String("query", request.Query),
		zap.String("detected_intent", string(response.DetectedIntent)),
		zap.String("query_type", string(response.QueryType)),
		zap.Float64("confidence", response.ConfidenceScore),
		zap.Duration("processing_time", response.ProcessingTime),
	)

	return response, nil
}

// normalizeQuery normalizes the input query
func (qa *QueryAnalyzer) normalizeQuery(query string) string {
	// Convert to lowercase
	normalized := strings.ToLower(query)

	// Remove extra whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	// Trim whitespace
	normalized = strings.TrimSpace(normalized)

	return normalized
}

// tokenizeQuery tokenizes the normalized query
func (qa *QueryAnalyzer) tokenizeQuery(query string) []QueryToken {
	tokens := make([]QueryToken, 0)
	words := strings.Fields(query)
	position := 0

	for _, word := range words {
		token := QueryToken{
			Text:       word,
			Type:       qa.classifyToken(word),
			Position:   position,
			Length:     len(word),
			Confidence: 1.0,
		}

		// Add annotations based on token type
		token.Annotations = qa.annotateToken(word, token.Type)

		tokens = append(tokens, token)
		position += len(word) + 1 // +1 for space
	}

	return tokens
}

// classifyToken classifies a token by type
func (qa *QueryAnalyzer) classifyToken(token string) TokenType {
	// Check if it's a code
	if qa.isCode(token) {
		return TokenTypeCode
	}

	// Check if it's a number
	if regexp.MustCompile(`^\d+$`).MatchString(token) {
		return TokenTypeNumber
	}

	// Check if it's punctuation
	if regexp.MustCompile(`^[[:punct:]]+$`).MatchString(token) {
		return TokenTypePunctuation
	}

	// Check if it's a stop word
	if qa.isStopWord(token) {
		return TokenTypeStopWord
	}

	// Check if it's a medical term (simplified check)
	if qa.isMedicalTerm(token) {
		return TokenTypeMedicalTerm
	}

	// Default to word
	return TokenTypeWord
}

// isCode checks if a token is a clinical code
func (qa *QueryAnalyzer) isCode(token string) bool {
	// Check various code patterns
	patterns := []*regexp.Regexp{
		qa.patterns.SNOMEDCodePattern,
		qa.patterns.ICD10CodePattern,
		qa.patterns.RxNormCodePattern,
		qa.patterns.LOINCCodePattern,
		qa.patterns.CPTCodePattern,
	}

	for _, pattern := range patterns {
		if pattern.MatchString(token) {
			return true
		}
	}

	return false
}

// isStopWord checks if a token is a stop word
func (qa *QueryAnalyzer) isStopWord(token string) bool {
	stopWords := []string{
		"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by",
		"is", "are", "was", "were", "be", "been", "being", "have", "has", "had", "do", "does", "did",
	}

	for _, stopWord := range stopWords {
		if token == stopWord {
			return true
		}
	}

	return false
}

// isMedicalTerm checks if a token is likely a medical term (simplified)
func (qa *QueryAnalyzer) isMedicalTerm(token string) bool {
	// Simplified medical term detection
	// In a real implementation, this would use a medical dictionary
	medicalSuffixes := []string{"itis", "oma", "osis", "emia", "pathy", "therapy", "gram", "scopy"}

	for _, suffix := range medicalSuffixes {
		if strings.HasSuffix(token, suffix) {
			return true
		}
	}

	return false
}

// annotateToken adds annotations to a token
func (qa *QueryAnalyzer) annotateToken(token string, tokenType TokenType) []TokenAnnotation {
	annotations := make([]TokenAnnotation, 0)

	switch tokenType {
	case TokenTypeCode:
		if system := qa.detectCodeSystem(token); system != "" {
			annotations = append(annotations, TokenAnnotation{
				Type:       "code_system",
				Value:      system,
				Confidence: 0.9,
				Source:     "pattern_matching",
			})
		}
	case TokenTypeMedicalTerm:
		annotations = append(annotations, TokenAnnotation{
			Type:       "medical_category",
			Value:      "clinical_term",
			Confidence: 0.7,
			Source:     "heuristic",
		})
	}

	return annotations
}

// detectCodeSystem detects which coding system a code belongs to
func (qa *QueryAnalyzer) detectCodeSystem(code string) string {
	if qa.patterns.SNOMEDCodePattern.MatchString(code) {
		return "SNOMED_CT"
	}
	if qa.patterns.ICD10CodePattern.MatchString(code) {
		return "ICD10CM"
	}
	if qa.patterns.RxNormCodePattern.MatchString(code) {
		return "RXNORM"
	}
	if qa.patterns.LOINCCodePattern.MatchString(code) {
		return "LOINC"
	}
	if qa.patterns.CPTCodePattern.MatchString(code) {
		return "CPT"
	}
	return ""
}

// extractEntities extracts entities from the query
func (qa *QueryAnalyzer) extractEntities(ctx context.Context, query string, tokens []QueryToken) []*ExtractedEntity {
	entities := make([]*ExtractedEntity, 0)

	// Extract entities from tokens
	for _, token := range tokens {
		if token.Type == TokenTypeCode || token.Type == TokenTypeMedicalTerm {
			entity := &ExtractedEntity{
				Text:       token.Text,
				Position:   [2]int{token.Position, token.Position + token.Length},
				Confidence: token.Confidence,
				Metadata:   make(map[string]interface{}),
			}

			if token.Type == TokenTypeCode {
				entity.Type = EntityTypeCode
				entity.Subtype = qa.detectCodeSystem(token.Text)
			} else {
				entity.Type = EntityTypeTerm
			}

			entities = append(entities, entity)
		}
	}

	return entities
}

// detectCodes detects clinical codes in the query
func (qa *QueryAnalyzer) detectCodes(query string, tokens []QueryToken) []*DetectedCode {
	codes := make([]*DetectedCode, 0)

	for _, token := range tokens {
		if token.Type == TokenTypeCode {
			code := &DetectedCode{
				Code:       token.Text,
				System:     qa.detectCodeSystem(token.Text),
				Position:   [2]int{token.Position, token.Position + token.Length},
				Confidence: token.Confidence,
				Validated:  false, // Would need validation against actual code systems
			}

			codes = append(codes, code)
		}
	}

	return codes
}

// extractMedicalTerms extracts medical terms from the query
func (qa *QueryAnalyzer) extractMedicalTerms(query string, tokens []QueryToken) []*MedicalTerm {
	terms := make([]*MedicalTerm, 0)

	for _, token := range tokens {
		if token.Type == TokenTypeMedicalTerm {
			term := &MedicalTerm{
				Term:       token.Text,
				Category:   "general",
				Position:   [2]int{token.Position, token.Position + token.Length},
				Confidence: token.Confidence,
			}

			terms = append(terms, term)
		}
	}

	return terms
}

// Implement remaining methods with appropriate logic...

// Initialize query patterns
func initializeQueryPatterns() *QueryPatterns {
	return &QueryPatterns{
		// Code patterns
		SNOMEDCodePattern:    regexp.MustCompile(`^\d{6,18}$`),
		ICD10CodePattern:     regexp.MustCompile(`^[A-Z]\d{2}(\.\d{1,3})?$`),
		RxNormCodePattern:    regexp.MustCompile(`^\d{1,8}$`),
		LOINCCodePattern:     regexp.MustCompile(`^\d{1,5}-\d{1,2}$`),
		CPTCodePattern:       regexp.MustCompile(`^\d{5}$`),

		// Clinical domain patterns
		DiagnosticPatterns:   []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(diagnos\w*|disease|disorder|syndrome|condition)\b`),
		},
		MedicationPatterns:   []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(drug|medication|medicine|pharmaceutical|tablet|capsule|injection)\b`),
		},
		// ... more patterns would be added
	}
}

// Additional helper methods would be implemented here...

// Simplified implementations for remaining methods
func (qa *QueryAnalyzer) classifyIntent(query string, tokens []QueryToken, entities []*ExtractedEntity) (SearchIntent, float64) {
	// Simplified intent classification
	if strings.Contains(query, "what is") || strings.Contains(query, "define") {
		return IntentLookup, 0.8
	}
	if strings.Contains(query, "related") || strings.Contains(query, "similar") {
		return IntentExplore, 0.7
	}
	return IntentLookup, 0.6
}

func (qa *QueryAnalyzer) classifyQueryType(query string, tokens []QueryToken, entities []*ExtractedEntity) (QueryType, float64) {
	// Simplified query type classification
	for _, pattern := range qa.patterns.MedicationPatterns {
		if pattern.MatchString(query) {
			return QueryTypeMedication, 0.8
		}
	}
	for _, pattern := range qa.patterns.DiagnosticPatterns {
		if pattern.MatchString(query) {
			return QueryTypeDiagnostic, 0.8
		}
	}
	return QueryTypeGeneral, 0.6
}

func (qa *QueryAnalyzer) classifyDomains(query string, tokens []QueryToken) []DomainClassification {
	domains := make([]DomainClassification, 0)

	// Check for medication domain
	for _, pattern := range qa.patterns.MedicationPatterns {
		if pattern.MatchString(query) {
			domains = append(domains, DomainClassification{
				Domain:     "medication",
				Confidence: 0.8,
				Indicators: []string{"medication pattern match"},
			})
			break
		}
	}

	return domains
}

func (qa *QueryAnalyzer) recommendSearchStrategy(response *QueryAnalysisResponse) RecommendedStrategy {
	strategy := RecommendedStrategy{
		PrimaryStrategy:    SearchModeStandard,
		FallbackStrategies: []ClinicalSearchMode{SearchModeFuzzy, SearchModePhonetic},
		SortPreference:     "relevance",
		FilterSuggestions:  make(map[string]string),
		Reasoning:         "Default strategy based on general query",
	}

	// Adjust strategy based on detected codes
	if len(response.DetectedCodes) > 0 {
		strategy.PrimaryStrategy = SearchModeExact
		strategy.Reasoning = "Exact search for detected codes"
	}

	// Adjust based on query type
	switch response.QueryType {
	case QueryTypeMedication:
		strategy.RecommendedSystems = []string{"RXNORM"}
		strategy.FilterSuggestions["domain"] = "medication"
	case QueryTypeDiagnostic:
		strategy.RecommendedSystems = []string{"SNOMED_CT", "ICD10CM"}
		strategy.FilterSuggestions["domain"] = "diagnostic"
	}

	return strategy
}

func (qa *QueryAnalyzer) generateFilterSuggestions(response *QueryAnalysisResponse) map[string][]string {
	filters := make(map[string][]string)

	// Add system filters based on detected codes
	if len(response.DetectedCodes) > 0 {
		systems := make([]string, 0)
		for _, code := range response.DetectedCodes {
			if code.System != "" {
				systems = append(systems, code.System)
			}
		}
		if len(systems) > 0 {
			filters["systems"] = systems
		}
	}

	// Add domain filters based on classification
	if len(response.ClinicalDomains) > 0 {
		domains := make([]string, 0)
		for _, domain := range response.ClinicalDomains {
			domains = append(domains, domain.Domain)
		}
		filters["domains"] = domains
	}

	return filters
}

func (qa *QueryAnalyzer) generateQueryExpansions(ctx context.Context, query string) []string {
	// Simplified query expansion
	return []string{}
}

func (qa *QueryAnalyzer) generateSpellCorrections(query string) []SpellSuggestion {
	// Simplified spell correction
	return []SpellSuggestion{}
}

func (qa *QueryAnalyzer) performSemanticAnalysis(query string, tokens []QueryToken) *SemanticAnalysis {
	// Simplified semantic analysis
	return &SemanticAnalysis{
		MainConcepts:      []string{query},
		SemanticDensity:   0.5,
		ConceptualClarity: 0.7,
		Relationships:     []ConceptRelation{},
	}
}

func (qa *QueryAnalyzer) performSentimentAnalysis(query string) *SentimentAnalysis {
	// Simplified sentiment analysis
	return &SentimentAnalysis{
		OverallSentiment: "neutral",
		Confidence:       0.6,
		Emotions:         make(map[string]float64),
		Urgency:          0.3,
	}
}

func (qa *QueryAnalyzer) analyzeComplexity(query string, tokens []QueryToken) *ComplexityAnalysis {
	// Simplified complexity analysis
	lexicalComplexity := float64(len(tokens)) / 10.0
	if lexicalComplexity > 1.0 {
		lexicalComplexity = 1.0
	}

	return &ComplexityAnalysis{
		LexicalComplexity:    lexicalComplexity,
		SyntacticComplexity:  0.3,
		SemanticComplexity:   0.4,
		OverallComplexity:    (lexicalComplexity + 0.3 + 0.4) / 3.0,
		ProcessingDifficulty: "medium",
	}
}

func (qa *QueryAnalyzer) calculateOverallConfidence(response *QueryAnalysisResponse) float64 {
	// Average of intent and type confidence
	return (response.IntentConfidence + response.TypeConfidence) / 2.0
}

// Cache and metrics helper methods
func (qa *QueryAnalyzer) generateCacheKey(request *QueryAnalysisRequest) string {
	return fmt.Sprintf("query_analysis:%s", strings.ToLower(request.Query))
}

func (qa *QueryAnalyzer) getCachedAnalysis(key string) (*QueryAnalysisResponse, bool) {
	if cached, err := qa.cache.Get(key); err == nil {
		if response, ok := cached.(*QueryAnalysisResponse); ok {
			return response, true
		}
	}
	return nil, false
}

func (qa *QueryAnalyzer) cacheAnalysis(key string, response *QueryAnalysisResponse) {
	qa.cache.Set(key, response, qa.config.CacheTTL)
}

func (qa *QueryAnalyzer) recordAnalysisMetrics(request *QueryAnalysisRequest, response *QueryAnalysisResponse) {
	qa.metrics.RecordAnalysisMetric("query_analysis_requests_total", "complete")
	qa.metrics.RecordAnalysisMetric("query_analysis_time_seconds", "recorded")
	qa.metrics.RecordAnalysisMetric("query_analysis_confidence", "scored")

	labels := map[string]string{
		"intent":     string(response.DetectedIntent),
		"query_type": string(response.QueryType),
		"cache_hit":  fmt.Sprintf("%t", response.CacheHit),
	}

	qa.metrics.IncrementCounterWithLabels("query_analyses_total", labels)
}