# Apollo Federation Client Integration Guide

## Overview

This document describes the Apollo Federation client implementation for Medication Service V2, which connects to the existing Apollo Federation gateway to query knowledge bases for clinical decision support.

## Architecture

```
Medication Service V2 → Apollo Federation Client → Apollo Federation Gateway (port 4000) → Knowledge Base Services
```

### Key Components

1. **ApolloFederationClient**: Low-level GraphQL client for Apollo Federation gateway
2. **ApolloFederationService**: High-level service layer with caching and health checking
3. **KnowledgeBaseIntegrationService**: Unified knowledge base query interface
4. **GraphQLQueryBuilder**: Optimized GraphQL query construction with fragments
5. **ApolloFederationFactory**: Client management with health monitoring
6. **HTTP Handlers**: REST API endpoints for knowledge base queries

## Configuration

### YAML Configuration

The Apollo Federation client is configured via the existing `config.yaml`:

```yaml
external_services:
  apollo_federation:
    url: "${APOLLO_FEDERATION_URL:-http://localhost:4000/graphql}"
    timeout: "15s"
    max_retries: 2
    circuit_breaker:
      enabled: true
      max_requests: 5
      interval: "30s"
      timeout: "5s"
      failure_threshold: 0.5
```

### Environment Variables

- `APOLLO_FEDERATION_URL`: Apollo Federation gateway URL (default: http://localhost:4000/graphql)

## Usage Examples

### 1. Basic Setup and Initialization

```go
// Initialize Apollo Federation components
deps := &bootstrap.SetupDependencies{
    Logger:             logger,
    Config:             configMap,
    CacheService:       cacheService,
    PerformanceMonitor: performanceMonitor,
    CircuitBreaker:     circuitBreaker,
}

apolloSetup, err := bootstrap.NewApolloFederationSetup(deps)
if err != nil {
    log.Fatal("Failed to setup Apollo Federation:", err)
}

knowledgeBaseService := apolloSetup.GetKnowledgeBaseService()
```

### 2. Basic Dosing Rule Query

```go
// Query dosing rules for a specific drug
request := &services.KnowledgeBaseQueryRequest{
    DrugCode:     "vancomycin",
    Region:       stringPtr("US"),
    QueryTypes:   []string{"dosing"},
    CacheEnabled: true,
    CacheTTL:     30 * time.Minute,
    Priority:     "normal",
}

response, err := knowledgeBaseService.QueryKnowledgeBases(ctx, request)
if err != nil {
    return fmt.Errorf("dosing query failed: %w", err)
}

// Process dosing rules
for _, rule := range response.DosingRules {
    fmt.Printf("Drug: %s, Version: %s, Starting Dose: %.1f %s\n",
        rule.DrugName, rule.Version, rule.BaseDose.Starting, rule.BaseDose.Unit)
}
```

### 3. Patient-Specific Dosing Calculation

```go
// Calculate personalized dosing recommendation
patientContext := &infrastructure.PatientContextInput{
    WeightKg:            70.0,
    EGFR:               85.0,
    AgeYears:            45,
    Sex:                "male",
    Pregnant:           boolPtr(false),
    CreatinineClearance: float64Ptr(90.0),
}

request := &services.KnowledgeBaseQueryRequest{
    DrugCode:       "vancomycin",
    PatientContext: patientContext,
    Region:         stringPtr("US"),
    QueryTypes:     []string{"dosing"},
    CacheEnabled:   true,
    CacheTTL:       15 * time.Minute,
    Priority:       "high",
}

response, err := knowledgeBaseService.QueryKnowledgeBases(ctx, request)
if err != nil {
    return fmt.Errorf("patient dosing calculation failed: %w", err)
}

// Process dosing recommendations
for _, rec := range response.DosingRecommendations {
    fmt.Printf("Recommended dose: %.1f mg %s\n", 
        rec.RecommendedDose.AmountMg, rec.RecommendedDose.Frequency)
    
    // Check safety alerts
    for _, alert := range rec.SafetyAlerts {
        fmt.Printf("Safety Alert (%s): %s\n", alert.Severity, alert.Message)
    }
}
```

### 4. Batch Query Multiple Drugs

```go
// Query multiple drugs efficiently
drugCodes := []string{"vancomycin", "gentamicin", "cefazolin", "warfarin"}
requests := make([]*services.KnowledgeBaseQueryRequest, len(drugCodes))

for i, drugCode := range drugCodes {
    requests[i] = &services.KnowledgeBaseQueryRequest{
        DrugCode:     drugCode,
        Region:       stringPtr("US"),
        QueryTypes:   []string{"dosing", "availability"},
        CacheEnabled: true,
        CacheTTL:     30 * time.Minute,
    }
}

responses, err := knowledgeBaseService.BatchQueryKnowledgeBases(ctx, requests)
if err != nil {
    return fmt.Errorf("batch query failed: %w", err)
}

for drugCode, response := range responses {
    available := len(response.DosingRules) > 0
    fmt.Printf("Drug %s: Available=%t, Rules=%d\n", 
        drugCode, available, len(response.DosingRules))
}
```

### 5. Comprehensive Clinical Intelligence

```go
// Query all available knowledge for a drug
request := &services.KnowledgeBaseQueryRequest{
    DrugCode:     "warfarin",
    Region:       stringPtr("US"),
    QueryTypes:   []string{"dosing", "guidelines", "interactions", "safety", "availability"},
    CacheEnabled: true,
    CacheTTL:     20 * time.Minute,
    Priority:     "normal",
}

response, err := knowledgeBaseService.QueryKnowledgeBases(ctx, request)
if err != nil {
    return fmt.Errorf("clinical intelligence query failed: %w", err)
}

fmt.Printf("Clinical Intelligence for %s:\n", response.DrugCode)
fmt.Printf("- Dosing Rules: %d\n", len(response.DosingRules))
fmt.Printf("- Clinical Guidelines: %d\n", len(response.ClinicalGuidelines))
fmt.Printf("- Drug Interactions: %d\n", len(response.DrugInteractions))
fmt.Printf("- Safety Alerts: %d\n", len(response.SafetyAlerts))
fmt.Printf("- Response Time: %v\n", response.QueryMetrics.TotalDuration)
```

## HTTP API Endpoints

The service exposes REST endpoints for knowledge base queries:

### Single Drug Query
```http
POST /api/federation/query
Content-Type: application/json

{
  "drug_code": "vancomycin",
  "query_types": ["dosing"],
  "region": "US",
  "cache_enabled": true,
  "cache_ttl_minutes": 30
}
```

### Patient-Specific Query
```http
POST /api/federation/personalized
Content-Type: application/json

{
  "drug_code": "vancomycin",
  "patient_context": {
    "weightKg": 70.0,
    "egfr": 85.0,
    "ageYears": 45,
    "sex": "male"
  },
  "region": "US",
  "query_types": ["dosing"]
}
```

### Batch Query
```http
POST /api/federation/batch/dosing
Content-Type: application/json

{
  "drug_codes": ["vancomycin", "gentamicin", "cefazolin"],
  "region": "US"
}
```

### Availability Check
```http
GET /api/federation/availability?drug_code=vancomycin&region=US
```

### Health Check
```http
GET /api/federation/health
```

## Knowledge Base Integration

The client integrates with existing knowledge bases through Apollo Federation:

### KB1 - Drug Rules (port 8081)
- Dosing rules and calculations
- Population-specific adjustments
- Titration schedules
- Safety verification

### KB3 - Clinical Guidelines (port 8084)
- Evidence-based recommendations
- Clinical practice guidelines
- Therapeutic protocols

### KB4 - Patient Safety (port 8085)
- Contraindications and warnings
- Lab monitoring requirements
- Risk assessments

### KB5 - Drug Interactions (port 8086)
- Drug-drug interactions
- Severity classifications
- Management recommendations

### KB6 - Formulary (port 8087)
- Medication formularies
- Cost considerations
- Alternative medications

### KB7 - Terminology (port 8088)
- Drug codes and mappings
- Standardized terminology
- Cross-references

## Performance Optimization

### Caching Strategy
- **Individual Queries**: 30-minute default TTL
- **Personalized Queries**: 15-minute TTL
- **Batch Queries**: Optimized batch retrieval with shared cache
- **Availability Checks**: 5-minute TTL

### Query Optimization
- **GraphQL Fragments**: Reusable field selections
- **Batch Operations**: Single GraphQL query for multiple drugs
- **Field Selection**: Request only needed fields
- **Complexity Limits**: Query complexity estimation and limiting

### Circuit Breaker
- **Failure Threshold**: 50% (configurable)
- **Recovery Time**: 30 seconds
- **Max Requests**: 5 during half-open state

## Error Handling

### Retry Strategy
- **Max Retries**: 2 (configurable)
- **Retry Delay**: 1 second with exponential backoff
- **Non-Retryable Errors**: Validation, syntax, authentication errors

### Graceful Degradation
- **Partial Results**: Return successful queries even if some fail
- **Fallback Behavior**: Continue operation with cached data
- **Health Monitoring**: Background health checks with automatic recovery

## Monitoring and Metrics

### Performance Metrics
- Query count by type
- Average response times
- Error rates and types
- Cache hit/miss ratios
- Circuit breaker events

### Health Monitoring
- Apollo Federation gateway connectivity
- Individual knowledge base availability
- Response time tracking
- Error rate monitoring

### Logging
- Query execution details (when enabled)
- Performance measurements
- Error conditions and stack traces
- Health check results

## Testing

### Unit Tests
```go
// Test basic client functionality
func TestApolloFederationClient_GetDosingRule(t *testing.T) {
    // Mock GraphQL responses
    // Test query construction
    // Verify response parsing
}
```

### Integration Tests
```go
// Test against real Apollo Federation gateway
func TestKnowledgeBaseIntegration_RealGateway(t *testing.T) {
    // Requires running Apollo Federation gateway
    // Test actual GraphQL queries
    // Verify end-to-end functionality
}
```

### Load Tests
- Concurrent query performance
- Batch operation efficiency
- Cache performance under load
- Circuit breaker behavior

## Deployment Considerations

### Prerequisites
1. Apollo Federation gateway running at port 4000
2. Knowledge base services with federation endpoints
3. Redis for caching (optional but recommended)
4. Network connectivity to gateway

### Configuration
1. Set `APOLLO_FEDERATION_URL` environment variable
2. Configure timeout and retry settings
3. Enable circuit breaker for production
4. Set appropriate cache TTL values

### Monitoring
1. Set up health check endpoints
2. Configure performance metrics collection
3. Set up alerting for connectivity issues
4. Monitor cache performance

## Troubleshooting

### Common Issues

1. **Connection Refused**
   - Verify Apollo Federation gateway is running
   - Check network connectivity
   - Validate URL configuration

2. **Query Timeout**
   - Increase timeout configuration
   - Check knowledge base service performance
   - Verify network latency

3. **GraphQL Errors**
   - Validate query syntax using GraphQL playground
   - Check schema compatibility
   - Verify field names and types

4. **Cache Issues**
   - Verify Redis connectivity
   - Check cache key generation
   - Monitor cache hit rates

### Debug Mode
Enable detailed logging by setting:
```yaml
logging:
  level: "debug"
external_services:
  apollo_federation:
    log_queries: true
    log_query_details: true
```

## Security Considerations

### Authentication
- GraphQL queries inherit HTTP authentication
- Support for bearer tokens and API keys
- TLS encryption in transit

### Authorization
- Query-level access control
- Knowledge base-specific permissions
- Patient data access controls

### Data Privacy
- No persistent storage of patient data
- Secure caching with expiration
- Audit logging for compliance

## Future Enhancements

### Planned Features
1. **Subscription Support**: Real-time updates for knowledge base changes
2. **Advanced Caching**: Intelligent cache invalidation
3. **Query Optimization**: Automatic query field selection
4. **Multi-Region Support**: Failover between federation gateways

### Knowledge Base Expansion
1. **KB5 Integration**: Complete drug interaction support
2. **KB4 Enhancement**: Advanced patient safety checks
3. **New Knowledge Bases**: Additional clinical knowledge sources
4. **External APIs**: Integration with external medical databases

## Support and Maintenance

### Configuration Updates
- Dynamic configuration reloading
- Blue-green deployment support
- Configuration validation

### Monitoring and Alerting
- Health check endpoints
- Performance metrics
- Error rate monitoring
- Capacity planning metrics

### Backup and Recovery
- Circuit breaker failover
- Cached data preservation
- Graceful service degradation