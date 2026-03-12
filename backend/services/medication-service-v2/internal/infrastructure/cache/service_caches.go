package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ServiceSpecificCaches provides optimized caching for different service types
type ServiceSpecificCaches struct {
	cacheManager *MultiLevelCache
	logger       *zap.Logger
}

// NewServiceSpecificCaches creates service-specific cache implementations
func NewServiceSpecificCaches(cacheManager *MultiLevelCache, logger *zap.Logger) *ServiceSpecificCaches {
	return &ServiceSpecificCaches{
		cacheManager: cacheManager,
		logger:       logger.Named("service_caches"),
	}
}

// Recipe Resolver Cache - Optimized for <10ms target response time
type RecipeResolverCache struct {
	cache  *MultiLevelCache
	logger *zap.Logger
}

// RecipeCache represents cached recipe data
type RecipeCache struct {
	ProtocolID      string                 `json:"protocol_id"`
	Recipe          map[string]interface{} `json:"recipe"`
	ComputedHash    string                 `json:"computed_hash"`
	Dependencies    []string               `json:"dependencies"`
	ClinicalContext map[string]interface{} `json:"clinical_context,omitempty"`
	CachedAt        time.Time              `json:"cached_at"`
	TTL             time.Duration          `json:"ttl"`
}

func (ssc *ServiceSpecificCaches) RecipeResolver() *RecipeResolverCache {
	return &RecipeResolverCache{
		cache:  ssc.cacheManager,
		logger: ssc.logger.Named("recipe_resolver"),
	}
}

// GetRecipe retrieves a cached recipe with aggressive optimization
func (rc *RecipeResolverCache) GetRecipe(ctx context.Context, protocolID string, patientContext map[string]interface{}) (*RecipeCache, error) {
	// Create context-aware cache key
	contextHash := rc.hashContext(patientContext)
	cacheKey := fmt.Sprintf("recipe:%s:ctx:%s", protocolID, contextHash)
	
	var cached RecipeCache
	if err := rc.cache.Get(ctx, cacheKey, &cached); err != nil {
		return nil, err
	}
	
	// Verify cache validity with computed hash
	currentHash := rc.computeRecipeHash(cached.Recipe, patientContext)
	if cached.ComputedHash != currentHash {
		// Cache invalidation due to context change
		rc.cache.Delete(ctx, cacheKey)
		return nil, ErrCacheMiss
	}
	
	rc.logger.Debug("Recipe cache hit",
		zap.String("protocol_id", protocolID),
		zap.String("context_hash", contextHash),
		zap.Duration("age", time.Since(cached.CachedAt)),
	)
	
	return &cached, nil
}

// SetRecipe caches a recipe with smart TTL based on complexity
func (rc *RecipeResolverCache) SetRecipe(ctx context.Context, protocolID string, recipe map[string]interface{}, patientContext map[string]interface{}, dependencies []string) error {
	contextHash := rc.hashContext(patientContext)
	cacheKey := fmt.Sprintf("recipe:%s:ctx:%s", protocolID, contextHash)
	
	// Calculate TTL based on recipe complexity and dependencies
	ttl := rc.calculateRecipeTTL(recipe, dependencies)
	
	cachedRecipe := RecipeCache{
		ProtocolID:      protocolID,
		Recipe:          recipe,
		ComputedHash:    rc.computeRecipeHash(recipe, patientContext),
		Dependencies:    dependencies,
		ClinicalContext: patientContext,
		CachedAt:        time.Now(),
		TTL:             ttl,
	}
	
	// Tag for invalidation
	tags := []string{
		fmt.Sprintf("protocol:%s", protocolID),
		"recipe_resolver",
	}
	for _, dep := range dependencies {
		tags = append(tags, fmt.Sprintf("dep:%s", dep))
	}
	
	return rc.cache.Set(ctx, cacheKey, cachedRecipe, ttl, tags...)
}

// InvalidateRecipesByProtocol invalidates all recipes for a protocol
func (rc *RecipeResolverCache) InvalidateRecipesByProtocol(ctx context.Context, protocolID string) error {
	return rc.cache.InvalidateByTags(ctx, fmt.Sprintf("protocol:%s", protocolID))
}

// Clinical Engine Results Cache - For Rust engine calculation results
type ClinicalEngineCache struct {
	cache  *MultiLevelCache
	logger *zap.Logger
}

// ClinicalCalculationResult represents cached calculation data
type ClinicalCalculationResult struct {
	CalculationID   string                 `json:"calculation_id"`
	InputParams     map[string]interface{} `json:"input_params"`
	Result          map[string]interface{} `json:"result"`
	ComputationTime time.Duration          `json:"computation_time"`
	EngineVersion   string                 `json:"engine_version"`
	CachedAt        time.Time              `json:"cached_at"`
	Confidence      float64                `json:"confidence"`
	ValidationFlags []string               `json:"validation_flags,omitempty"`
}

func (ssc *ServiceSpecificCaches) ClinicalEngine() *ClinicalEngineCache {
	return &ClinicalEngineCache{
		cache:  ssc.cacheManager,
		logger: ssc.logger.Named("clinical_engine"),
	}
}

// GetCalculationResult retrieves cached clinical calculation results
func (cec *ClinicalEngineCache) GetCalculationResult(ctx context.Context, calculationID string, inputParams map[string]interface{}) (*ClinicalCalculationResult, error) {
	paramsHash := cec.hashParams(inputParams)
	cacheKey := fmt.Sprintf("clinical_calc:%s:params:%s", calculationID, paramsHash)
	
	var cached ClinicalCalculationResult
	if err := cec.cache.Get(ctx, cacheKey, &cached); err != nil {
		return nil, err
	}
	
	cec.logger.Debug("Clinical calculation cache hit",
		zap.String("calculation_id", calculationID),
		zap.String("params_hash", paramsHash),
		zap.Duration("computation_time", cached.ComputationTime),
	)
	
	return &cached, nil
}

// SetCalculationResult caches clinical calculation results
func (cec *ClinicalEngineCache) SetCalculationResult(ctx context.Context, result *ClinicalCalculationResult) error {
	paramsHash := cec.hashParams(result.InputParams)
	cacheKey := fmt.Sprintf("clinical_calc:%s:params:%s", result.CalculationID, paramsHash)
	
	// TTL based on confidence and calculation complexity
	ttl := cec.calculateResultTTL(result)
	
	tags := []string{
		fmt.Sprintf("calc_type:%s", result.CalculationID),
		"clinical_engine",
		fmt.Sprintf("engine_version:%s", result.EngineVersion),
	}
	
	return cec.cache.Set(ctx, cacheKey, *result, ttl, tags...)
}

// Workflow State Cache - For 4-Phase orchestration state
type WorkflowStateCache struct {
	cache  *MultiLevelCache
	logger *zap.Logger
}

// WorkflowState represents cached workflow state
type WorkflowState struct {
	WorkflowID    string                 `json:"workflow_id"`
	PatientID     string                 `json:"patient_id"`
	CurrentPhase  int                    `json:"current_phase"`
	PhaseData     map[string]interface{} `json:"phase_data"`
	State         string                 `json:"state"` // PENDING, RUNNING, COMPLETED, FAILED
	Progress      float64                `json:"progress"`
	Metadata      map[string]interface{} `json:"metadata"`
	LastUpdated   time.Time              `json:"last_updated"`
	TTL           time.Duration          `json:"ttl"`
}

func (ssc *ServiceSpecificCaches) WorkflowState() *WorkflowStateCache {
	return &WorkflowStateCache{
		cache:  ssc.cacheManager,
		logger: ssc.logger.Named("workflow_state"),
	}
}

// GetWorkflowState retrieves cached workflow state
func (wsc *WorkflowStateCache) GetWorkflowState(ctx context.Context, workflowID string) (*WorkflowState, error) {
	cacheKey := fmt.Sprintf("workflow_state:%s", workflowID)
	
	var cached WorkflowState
	if err := wsc.cache.Get(ctx, cacheKey, &cached); err != nil {
		return nil, err
	}
	
	return &cached, nil
}

// SetWorkflowState caches workflow state with dynamic TTL
func (wsc *WorkflowStateCache) SetWorkflowState(ctx context.Context, state *WorkflowState) error {
	cacheKey := fmt.Sprintf("workflow_state:%s", state.WorkflowID)
	
	// Dynamic TTL based on workflow state
	var ttl time.Duration
	switch state.State {
	case "RUNNING":
		ttl = 30 * time.Minute // Active workflows cached longer
	case "PENDING":
		ttl = 15 * time.Minute
	case "COMPLETED", "FAILED":
		ttl = 24 * time.Hour // Completed workflows cached for history
	default:
		ttl = 10 * time.Minute
	}
	
	state.TTL = ttl
	state.LastUpdated = time.Now()
	
	tags := []string{
		fmt.Sprintf("patient:%s", state.PatientID),
		fmt.Sprintf("phase:%d", state.CurrentPhase),
		fmt.Sprintf("state:%s", state.State),
		"workflow_orchestrator",
	}
	
	return wsc.cache.Set(ctx, cacheKey, *state, ttl, tags...)
}

// Google FHIR Cache - Smart caching of FHIR resources and metadata
type GoogleFHIRCache struct {
	cache  *MultiLevelCache
	logger *zap.Logger
}

// FHIRResourceCache represents cached FHIR resource data
type FHIRResourceCache struct {
	ResourceType  string                 `json:"resource_type"`
	ResourceID    string                 `json:"resource_id"`
	ResourceData  map[string]interface{} `json:"resource_data"`
	Metadata      map[string]interface{} `json:"metadata"`
	ETag          string                 `json:"etag"`
	LastModified  time.Time              `json:"last_modified"`
	CachedAt      time.Time              `json:"cached_at"`
	ProjectID     string                 `json:"project_id"`
	DatasetID     string                 `json:"dataset_id"`
	FHIRStoreID   string                 `json:"fhir_store_id"`
}

func (ssc *ServiceSpecificCaches) GoogleFHIR() *GoogleFHIRCache {
	return &GoogleFHIRCache{
		cache:  ssc.cacheManager,
		logger: ssc.logger.Named("google_fhir"),
	}
}

// GetFHIRResource retrieves cached FHIR resource with ETag validation
func (gfc *GoogleFHIRCache) GetFHIRResource(ctx context.Context, projectID, datasetID, fhirStoreID, resourceType, resourceID string) (*FHIRResourceCache, error) {
	cacheKey := fmt.Sprintf("fhir:%s:%s:%s:%s:%s", projectID, datasetID, fhirStoreID, resourceType, resourceID)
	
	var cached FHIRResourceCache
	if err := gfc.cache.Get(ctx, cacheKey, &cached); err != nil {
		return nil, err
	}
	
	gfc.logger.Debug("FHIR resource cache hit",
		zap.String("resource_type", resourceType),
		zap.String("resource_id", resourceID),
		zap.String("etag", cached.ETag),
	)
	
	return &cached, nil
}

// SetFHIRResource caches FHIR resource with metadata
func (gfc *GoogleFHIRCache) SetFHIRResource(ctx context.Context, resource *FHIRResourceCache) error {
	cacheKey := fmt.Sprintf("fhir:%s:%s:%s:%s:%s", 
		resource.ProjectID, resource.DatasetID, resource.FHIRStoreID, 
		resource.ResourceType, resource.ResourceID)
	
	// FHIR resources cached for longer periods due to medical data stability
	ttl := 4 * time.Hour
	
	// Shorter TTL for frequently changing resources
	if resource.ResourceType == "Observation" || resource.ResourceType == "DiagnosticReport" {
		ttl = 30 * time.Minute
	}
	
	resource.CachedAt = time.Now()
	
	tags := []string{
		fmt.Sprintf("fhir_store:%s", resource.FHIRStoreID),
		fmt.Sprintf("resource_type:%s", resource.ResourceType),
		fmt.Sprintf("project:%s", resource.ProjectID),
		"google_fhir",
	}
	
	return gfc.cache.Set(ctx, cacheKey, *resource, ttl, tags...)
}

// Apollo Federation Query Cache - For GraphQL query result caching
type ApolloFederationCache struct {
	cache  *MultiLevelCache
	logger *zap.Logger
}

// GraphQLQueryCache represents cached GraphQL query results
type GraphQLQueryCache struct {
	QueryHash      string                 `json:"query_hash"`
	Query          string                 `json:"query"`
	Variables      map[string]interface{} `json:"variables"`
	Result         map[string]interface{} `json:"result"`
	ExecutionTime  time.Duration          `json:"execution_time"`
	ServicesCalled []string               `json:"services_called"`
	CachedAt       time.Time              `json:"cached_at"`
	TTL            time.Duration          `json:"ttl"`
}

func (ssc *ServiceSpecificCaches) ApolloFederation() *ApolloFederationCache {
	return &ApolloFederationCache{
		cache:  ssc.cacheManager,
		logger: ssc.logger.Named("apollo_federation"),
	}
}

// GetQueryResult retrieves cached GraphQL query results
func (afc *ApolloFederationCache) GetQueryResult(ctx context.Context, queryHash string, variables map[string]interface{}) (*GraphQLQueryCache, error) {
	variablesHash := afc.hashVariables(variables)
	cacheKey := fmt.Sprintf("graphql:%s:vars:%s", queryHash, variablesHash)
	
	var cached GraphQLQueryCache
	if err := afc.cache.Get(ctx, cacheKey, &cached); err != nil {
		return nil, err
	}
	
	afc.logger.Debug("GraphQL query cache hit",
		zap.String("query_hash", queryHash),
		zap.Duration("execution_time", cached.ExecutionTime),
		zap.Strings("services_called", cached.ServicesCalled),
	)
	
	return &cached, nil
}

// SetQueryResult caches GraphQL query results
func (afc *ApolloFederationCache) SetQueryResult(ctx context.Context, result *GraphQLQueryCache) error {
	variablesHash := afc.hashVariables(result.Variables)
	cacheKey := fmt.Sprintf("graphql:%s:vars:%s", result.QueryHash, variablesHash)
	
	// TTL based on query complexity and data freshness requirements
	ttl := afc.calculateQueryTTL(result)
	
	result.CachedAt = time.Now()
	result.TTL = ttl
	
	tags := []string{"apollo_federation", "graphql_query"}
	for _, service := range result.ServicesCalled {
		tags = append(tags, fmt.Sprintf("service:%s", service))
	}
	
	return afc.cache.Set(ctx, cacheKey, *result, ttl, tags...)
}

// Helper methods for hashing and TTL calculation

func (rc *RecipeResolverCache) hashContext(context map[string]interface{}) string {
	return rc.computeHash(context)
}

func (rc *RecipeResolverCache) computeRecipeHash(recipe map[string]interface{}, context map[string]interface{}) string {
	combined := make(map[string]interface{})
	for k, v := range recipe {
		combined[fmt.Sprintf("recipe_%s", k)] = v
	}
	for k, v := range context {
		combined[fmt.Sprintf("context_%s", k)] = v
	}
	return rc.computeHash(combined)
}

func (rc *RecipeResolverCache) computeHash(data interface{}) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", data)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (rc *RecipeResolverCache) calculateRecipeTTL(recipe map[string]interface{}, dependencies []string) time.Duration {
	// Base TTL
	ttl := 1 * time.Hour
	
	// Reduce TTL for complex recipes with many dependencies
	if len(dependencies) > 5 {
		ttl = 30 * time.Minute
	}
	
	// Increase TTL for simple, static recipes
	if len(recipe) < 10 && len(dependencies) <= 2 {
		ttl = 4 * time.Hour
	}
	
	return ttl
}

func (cec *ClinicalEngineCache) hashParams(params map[string]interface{}) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", params)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (cec *ClinicalEngineCache) calculateResultTTL(result *ClinicalCalculationResult) time.Duration {
	// Base TTL
	ttl := 2 * time.Hour
	
	// Longer TTL for high-confidence results
	if result.Confidence > 0.95 {
		ttl = 6 * time.Hour
	}
	
	// Shorter TTL for low-confidence or complex calculations
	if result.Confidence < 0.8 || result.ComputationTime > 5*time.Second {
		ttl = 30 * time.Minute
	}
	
	return ttl
}

func (afc *ApolloFederationCache) hashVariables(variables map[string]interface{}) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", variables)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (afc *ApolloFederationCache) calculateQueryTTL(result *GraphQLQueryCache) time.Duration {
	// Base TTL
	ttl := 15 * time.Minute
	
	// Longer TTL for expensive queries
	if result.ExecutionTime > 1*time.Second {
		ttl = 1 * time.Hour
	}
	
	// Shorter TTL for queries involving many services (higher chance of data changes)
	if len(result.ServicesCalled) > 3 {
		ttl = 5 * time.Minute
	}
	
	return ttl
}