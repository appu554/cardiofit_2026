# KB-7 Terminology Service - Developer Integration Guide

This guide provides comprehensive information for developers integrating with the KB-7 Terminology Service, including API usage, code examples, and best practices.

## 🚀 Quick Start Integration

### Authentication Setup

All API requests require authentication via API key or JWT token:

```bash
# Using API key
curl -H "X-API-Key: your-api-key" \
     "http://localhost:8087/v1/concepts?q=paracetamol"

# Using JWT token
curl -H "Authorization: Bearer your-jwt-token" \
     "http://localhost:8087/v1/concepts/snomed/387517004"
```

### Basic Code Lookup

```javascript
// JavaScript/Node.js example
const KB7_BASE_URL = 'http://localhost:8087';
const API_KEY = 'your-api-key';

async function lookupConcept(system, code) {
    const response = await fetch(`${KB7_BASE_URL}/v1/concepts/${system}/${code}`, {
        headers: {
            'X-API-Key': API_KEY,
            'Content-Type': 'application/json'
        }
    });
    
    if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    
    return response.json();
}

// Usage
const concept = await lookupConcept('snomed', '387517004');
console.log(concept.display); // "Paracetamol"
```

## 📚 Core API Usage Patterns

### 1. Individual Concept Lookup

**Endpoint**: `GET /v1/concepts/{system}/{code}`

```python
# Python example
import requests
from typing import Optional, Dict, Any

class KB7Client:
    def __init__(self, base_url: str, api_key: str):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({'X-API-Key': api_key})
    
    def lookup_concept(self, system: str, code: str) -> Optional[Dict[str, Any]]:
        """Look up a single concept by system and code."""
        response = self.session.get(f"{self.base_url}/v1/concepts/{system}/{code}")
        
        if response.status_code == 404:
            return None
        
        response.raise_for_status()
        return response.json()

# Usage
client = KB7Client('http://localhost:8087', 'your-api-key')
concept = client.lookup_concept('snomed', '387517004')

if concept:
    print(f"Code: {concept['code']}")
    print(f"Display: {concept['display']}")
    print(f"Definition: {concept.get('definition', 'N/A')}")
```

### 2. Full-Text Search

**Endpoint**: `GET /v1/concepts?q={query}&system={system}&count={limit}`

```go
// Go example
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
)

type KB7Client struct {
    BaseURL string
    APIKey  string
    Client  *http.Client
}

type SearchResult struct {
    Total    int64     `json:"total"`
    Concepts []Concept `json:"concepts"`
}

type Concept struct {
    Code       string `json:"code"`
    System     string `json:"system"`
    Display    string `json:"display"`
    Definition string `json:"definition"`
}

func (c *KB7Client) SearchConcepts(query, system string, limit int) (*SearchResult, error) {
    params := url.Values{}
    params.Add("q", query)
    if system != "" {
        params.Add("system", system)
    }
    if limit > 0 {
        params.Add("count", fmt.Sprintf("%d", limit))
    }
    
    req, err := http.NewRequest("GET", 
        fmt.Sprintf("%s/v1/concepts?%s", c.BaseURL, params.Encode()), nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("X-API-Key", c.APIKey)
    
    resp, err := c.Client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result SearchResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}

// Usage
client := &KB7Client{
    BaseURL: "http://localhost:8087",
    APIKey:  "your-api-key",
    Client:  &http.Client{},
}

results, err := client.SearchConcepts("paracetamol", "snomed", 10)
if err != nil {
    log.Fatal(err)
}

for _, concept := range results.Concepts {
    fmt.Printf("%s - %s (%s)\n", concept.Code, concept.Display, concept.System)
}
```

### 3. Code Validation

**Endpoint**: `POST /v1/concepts/validate`

```java
// Java example using Spring WebClient
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;
import java.util.List;
import java.util.Map;

public class KB7Client {
    private final WebClient webClient;
    
    public KB7Client(String baseUrl, String apiKey) {
        this.webClient = WebClient.builder()
            .baseUrl(baseUrl)
            .defaultHeader("X-API-Key", apiKey)
            .build();
    }
    
    public Mono<ValidationResponse> validateCode(String code, String system) {
        ValidationRequest request = new ValidationRequest();
        request.setCode(code);
        request.setSystem(system);
        
        return webClient.post()
            .uri("/v1/concepts/validate")
            .bodyValue(request)
            .retrieve()
            .bodyToMono(ValidationResponse.class);
    }
    
    public static class ValidationRequest {
        private String code;
        private String system;
        private String version;
        
        // Getters and setters
        public String getCode() { return code; }
        public void setCode(String code) { this.code = code; }
        public String getSystem() { return system; }
        public void setSystem(String system) { this.system = system; }
    }
    
    public static class ValidationResponse {
        private boolean valid;
        private String code;
        private String system;
        private String display;
        private String message;
        private String severity;
        
        // Getters and setters
        public boolean isValid() { return valid; }
        public String getDisplay() { return display; }
        public String getMessage() { return message; }
    }
}

// Usage
KB7Client client = new KB7Client("http://localhost:8087", "your-api-key");
ValidationResponse response = client.validateCode("387517004", "snomed").block();

if (response.isValid()) {
    System.out.println("Valid code: " + response.getDisplay());
} else {
    System.out.println("Invalid code: " + response.getMessage());
}
```

### 4. Batch Operations

**Endpoint**: `POST /v1/concepts/batch-lookup`

```typescript
// TypeScript example
interface BatchLookupRequest {
    requests: Array<{
        system: string;
        code: string;
        include_hierarchy?: boolean;
    }>;
}

interface BatchLookupResponse {
    results: Array<{
        code: string;
        system: string;
        found: boolean;
        entry?: TerminologyEntry;
        error?: string;
    }>;
}

interface TerminologyEntry {
    code: string;
    system: string;
    display: string;
    definition?: string;
    synonyms?: string[];
    relationships?: Relationship[];
}

class KB7Client {
    constructor(
        private baseUrl: string,
        private apiKey: string
    ) {}
    
    async batchLookup(requests: Array<{system: string, code: string}>): Promise<BatchLookupResponse> {
        const response = await fetch(`${this.baseUrl}/v1/concepts/batch-lookup`, {
            method: 'POST',
            headers: {
                'X-API-Key': this.apiKey,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ requests })
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        return response.json();
    }
}

// Usage
const client = new KB7Client('http://localhost:8087', 'your-api-key');

const codes = [
    { system: 'snomed', code: '387517004' },
    { system: 'icd10', code: 'Z51.11' },
    { system: 'rxnorm', code: '161' }
];

const results = await client.batchLookup(codes);

results.results.forEach(result => {
    if (result.found && result.entry) {
        console.log(`${result.code}: ${result.entry.display}`);
    } else {
        console.log(`${result.code}: Not found - ${result.error}`);
    }
});
```

## 🔗 Cross-System Mapping

### Concept Mapping

**Endpoint**: `GET /v1/mappings/{source_system}/{source_code}/{target_system}`

```csharp
// C# example
using System.Text.Json;
using System.Net.Http.Headers;

public class KB7Client
{
    private readonly HttpClient _httpClient;
    
    public KB7Client(string baseUrl, string apiKey)
    {
        _httpClient = new HttpClient { BaseAddress = new Uri(baseUrl) };
        _httpClient.DefaultRequestHeaders.Add("X-API-Key", apiKey);
    }
    
    public async Task<MappingResponse?> GetMappingAsync(
        string sourceSystem, 
        string sourceCode, 
        string targetSystem)
    {
        var response = await _httpClient.GetAsync(
            $"/v1/mappings/{sourceSystem}/{sourceCode}/{targetSystem}");
        
        if (response.StatusCode == HttpStatusCode.NotFound)
            return null;
            
        response.EnsureSuccessStatusCode();
        
        var json = await response.Content.ReadAsStringAsync();
        return JsonSerializer.Deserialize<MappingResponse>(json);
    }
}

public class MappingResponse
{
    public string SourceCode { get; set; } = "";
    public string SourceSystem { get; set; } = "";
    public List<Mapping> Mappings { get; set; } = new();
}

public class Mapping
{
    public string TargetCode { get; set; } = "";
    public string TargetSystem { get; set; } = "";
    public string TargetDisplay { get; set; } = "";
    public string MappingType { get; set; } = "";
    public double Confidence { get; set; }
}

// Usage
var client = new KB7Client("http://localhost:8087", "your-api-key");
var mapping = await client.GetMappingAsync("snomed", "387517004", "rxnorm");

if (mapping != null)
{
    foreach (var map in mapping.Mappings)
    {
        Console.WriteLine($"{map.TargetCode}: {map.TargetDisplay} (confidence: {map.Confidence})");
    }
}
```

## 🌳 Hierarchy Navigation

### Parent/Child Traversal

**Endpoint**: `GET /v1/hierarchy/{code}?system={system}&depth={depth}&direction={direction}`

```python
# Python example for hierarchy navigation
class TerminologyHierarchy:
    def __init__(self, kb7_client):
        self.client = kb7_client
    
    def get_ancestors(self, system: str, code: str, max_depth: int = 5) -> List[Dict]:
        """Get all ancestor concepts up the hierarchy."""
        response = self.client.session.get(
            f"{self.client.base_url}/v1/hierarchy/{code}",
            params={
                'system': system,
                'depth': max_depth,
                'direction': 'up'
            }
        )
        response.raise_for_status()
        return response.json().get('parents', [])
    
    def get_descendants(self, system: str, code: str, max_depth: int = 3) -> List[Dict]:
        """Get all descendant concepts down the hierarchy."""
        response = self.client.session.get(
            f"{self.client.base_url}/v1/hierarchy/{code}",
            params={
                'system': system,
                'depth': max_depth,
                'direction': 'down'
            }
        )
        response.raise_for_status()
        return response.json().get('children', [])
    
    def get_siblings(self, system: str, code: str) -> List[Dict]:
        """Get sibling concepts (same parents)."""
        ancestors = self.get_ancestors(system, code, 1)
        siblings = []
        
        for parent in ancestors:
            children = self.get_descendants(system, parent['code'], 1)
            siblings.extend([child for child in children if child['code'] != code])
        
        return siblings

# Usage
hierarchy = TerminologyHierarchy(client)

# Get all parents of "Paracetamol"
parents = hierarchy.get_ancestors('snomed', '387517004')
for parent in parents:
    print(f"Parent: {parent['code']} - {parent['display']}")

# Get all children of "Pharmaceutical / biologic product"
children = hierarchy.get_descendants('snomed', '373873005', max_depth=2)
for child in children:
    print(f"Child: {child['code']} - {child['display']} (level: {child['level']})")
```

## 🎯 Advanced Integration Patterns

### 1. Caching Strategy

```javascript
// JavaScript client with built-in caching
class CachedKB7Client {
    constructor(baseUrl, apiKey, cacheOptions = {}) {
        this.baseUrl = baseUrl;
        this.apiKey = apiKey;
        this.cache = new Map();
        this.cacheTTL = cacheOptions.ttl || 3600000; // 1 hour default
        this.maxCacheSize = cacheOptions.maxSize || 1000;
    }
    
    async lookupConcept(system, code) {
        const cacheKey = `${system}:${code}`;
        const cached = this.cache.get(cacheKey);
        
        if (cached && Date.now() - cached.timestamp < this.cacheTTL) {
            return cached.data;
        }
        
        const response = await fetch(`${this.baseUrl}/v1/concepts/${system}/${code}`, {
            headers: { 'X-API-Key': this.apiKey }
        });
        
        if (response.ok) {
            const data = await response.json();
            this.setCacheEntry(cacheKey, data);
            return data;
        }
        
        return null;
    }
    
    setCacheEntry(key, data) {
        // Implement LRU cache eviction
        if (this.cache.size >= this.maxCacheSize) {
            const firstKey = this.cache.keys().next().value;
            this.cache.delete(firstKey);
        }
        
        this.cache.set(key, {
            data,
            timestamp: Date.now()
        });
    }
}
```

### 2. Retry and Circuit Breaker Pattern

```go
// Go client with retry and circuit breaker
package main

import (
    "context"
    "time"
    "errors"
    "math"
)

type CircuitBreaker struct {
    maxFailures     int
    resetTimeout    time.Duration
    failures        int
    lastFailureTime time.Time
    state          CircuitState
}

type CircuitState int

const (
    CircuitClosed CircuitState = iota
    CircuitOpen
    CircuitHalfOpen
)

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
    if cb.state == CircuitOpen {
        if time.Since(cb.lastFailureTime) > cb.resetTimeout {
            cb.state = CircuitHalfOpen
        } else {
            return errors.New("circuit breaker is open")
        }
    }
    
    err := fn()
    
    if err != nil {
        cb.failures++
        cb.lastFailureTime = time.Now()
        
        if cb.failures >= cb.maxFailures {
            cb.state = CircuitOpen
        }
        return err
    }
    
    // Success - reset circuit breaker
    cb.failures = 0
    cb.state = CircuitClosed
    return nil
}

type ResilientKB7Client struct {
    baseClient     *KB7Client
    circuitBreaker *CircuitBreaker
    retryConfig    RetryConfig
}

type RetryConfig struct {
    MaxRetries      int
    InitialDelay    time.Duration
    MaxDelay        time.Duration
    BackoffMultiplier float64
}

func (c *ResilientKB7Client) LookupConceptWithRetry(ctx context.Context, system, code string) (*Concept, error) {
    var lastErr error
    
    for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
        err := c.circuitBreaker.Call(ctx, func() error {
            concept, err := c.baseClient.LookupConcept(system, code)
            if err != nil {
                lastErr = err
                return err
            }
            return nil
        })
        
        if err == nil {
            return lastErr.(*Concept), nil
        }
        
        if attempt < c.retryConfig.MaxRetries {
            delay := c.calculateDelay(attempt)
            select {
            case <-time.After(delay):
                continue
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }
    
    return nil, lastErr
}

func (c *ResilientKB7Client) calculateDelay(attempt int) time.Duration {
    delay := time.Duration(float64(c.retryConfig.InitialDelay) * 
        math.Pow(c.retryConfig.BackoffMultiplier, float64(attempt)))
    
    if delay > c.retryConfig.MaxDelay {
        delay = c.retryConfig.MaxDelay
    }
    
    return delay
}
```

### 3. Async/Streaming Operations

```javascript
// JavaScript streaming search with Server-Sent Events
class StreamingKB7Client {
    constructor(baseUrl, apiKey) {
        this.baseUrl = baseUrl;
        this.apiKey = apiKey;
    }
    
    async *streamSearch(query, systems = []) {
        const params = new URLSearchParams({
            q: query,
            stream: 'true'
        });
        
        if (systems.length > 0) {
            params.append('systems', systems.join(','));
        }
        
        const response = await fetch(`${this.baseUrl}/v1/search/stream?${params}`, {
            headers: {
                'X-API-Key': this.apiKey,
                'Accept': 'text/event-stream'
            }
        });
        
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        
        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;
                
                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop(); // Keep incomplete line in buffer
                
                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        const data = line.slice(6);
                        if (data === '[DONE]') return;
                        
                        try {
                            const concept = JSON.parse(data);
                            yield concept;
                        } catch (e) {
                            console.warn('Failed to parse SSE data:', data);
                        }
                    }
                }
            }
        } finally {
            reader.releaseLock();
        }
    }
}

// Usage
const streamingClient = new StreamingKB7Client('http://localhost:8087', 'your-api-key');

for await (const concept of streamingClient.streamSearch('heart disease', ['snomed'])) {
    console.log(`Found: ${concept.code} - ${concept.display}`);
}
```

## 📊 Monitoring and Observability

### Health Check Integration

```python
# Python health check integration
import time
from typing import Dict, Any

class KB7HealthMonitor:
    def __init__(self, kb7_client):
        self.client = kb7_client
        
    def check_service_health(self) -> Dict[str, Any]:
        """Comprehensive health check of KB7 service."""
        health_status = {
            'service': 'kb7-terminology',
            'status': 'healthy',
            'checks': {},
            'timestamp': time.time()
        }
        
        try:
            # 1. Basic connectivity check
            response = self.client.session.get(f"{self.client.base_url}/health", timeout=5)
            response.raise_for_status()
            
            service_health = response.json()
            health_status['checks']['connectivity'] = {
                'status': 'healthy',
                'response_time_ms': response.elapsed.total_seconds() * 1000
            }
            
            # 2. Database check (via service health)
            if service_health.get('checks', {}).get('database', {}).get('status') != 'healthy':
                health_status['status'] = 'degraded'
                health_status['checks']['database'] = service_health['checks']['database']
            
            # 3. Functional test - basic lookup
            start_time = time.time()
            test_concept = self.client.lookup_concept('snomed', '387517004')
            end_time = time.time()
            
            if test_concept:
                health_status['checks']['lookup_functionality'] = {
                    'status': 'healthy',
                    'response_time_ms': (end_time - start_time) * 1000
                }
            else:
                health_status['status'] = 'degraded'
                health_status['checks']['lookup_functionality'] = {
                    'status': 'unhealthy',
                    'error': 'Test concept lookup failed'
                }
                
        except Exception as e:
            health_status['status'] = 'unhealthy'
            health_status['checks']['connectivity'] = {
                'status': 'unhealthy',
                'error': str(e)
            }
            
        return health_status

# Usage in application health endpoint
monitor = KB7HealthMonitor(kb7_client)
health = monitor.check_service_health()

if health['status'] != 'healthy':
    # Alert or failover logic
    logger.warning(f"KB7 service health degraded: {health}")
```

### Metrics Collection

```go
// Go metrics collection
package main

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "time"
)

type KB7Metrics struct {
    requestsTotal       *prometheus.CounterVec
    requestDuration     *prometheus.HistogramVec
    cacheHits          prometheus.Counter
    cacheMisses        prometheus.Counter
    errorRate          *prometheus.CounterVec
}

func NewKB7Metrics() *KB7Metrics {
    return &KB7Metrics{
        requestsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "kb7_client_requests_total",
                Help: "Total number of requests to KB7 service",
            },
            []string{"method", "endpoint", "status"},
        ),
        requestDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "kb7_client_request_duration_seconds",
                Help:    "Duration of requests to KB7 service",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "endpoint"},
        ),
        cacheHits: promauto.NewCounter(prometheus.CounterOpts{
            Name: "kb7_client_cache_hits_total",
            Help: "Total number of cache hits",
        }),
        cacheMisses: promauto.NewCounter(prometheus.CounterOpts{
            Name: "kb7_client_cache_misses_total",
            Help: "Total number of cache misses",
        }),
        errorRate: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "kb7_client_errors_total",
                Help: "Total number of errors by type",
            },
            []string{"error_type"},
        ),
    }
}

type InstrumentedKB7Client struct {
    client  *KB7Client
    metrics *KB7Metrics
}

func (c *InstrumentedKB7Client) LookupConcept(system, code string) (*Concept, error) {
    start := time.Now()
    endpoint := "lookup_concept"
    
    defer func() {
        duration := time.Since(start)
        c.metrics.requestDuration.WithLabelValues("GET", endpoint).Observe(duration.Seconds())
    }()
    
    concept, err := c.client.LookupConcept(system, code)
    
    if err != nil {
        c.metrics.requestsTotal.WithLabelValues("GET", endpoint, "error").Inc()
        c.metrics.errorRate.WithLabelValues("lookup_error").Inc()
        return nil, err
    }
    
    c.metrics.requestsTotal.WithLabelValues("GET", endpoint, "success").Inc()
    return concept, nil
}
```

## 🔧 Testing and Validation

### Unit Testing

```python
# Python unit test example
import unittest
from unittest.mock import Mock, patch
import responses

class TestKB7Client(unittest.TestCase):
    def setUp(self):
        self.client = KB7Client('http://localhost:8087', 'test-api-key')
        
    @responses.activate
    def test_lookup_concept_success(self):
        # Mock successful response
        responses.add(
            responses.GET,
            'http://localhost:8087/v1/concepts/snomed/387517004',
            json={
                'code': '387517004',
                'system': 'snomed',
                'display': 'Paracetamol',
                'definition': 'A para-aminophenol derivative that is used as an analgesic and antipyretic.'
            },
            status=200
        )
        
        concept = self.client.lookup_concept('snomed', '387517004')
        
        self.assertIsNotNone(concept)
        self.assertEqual(concept['code'], '387517004')
        self.assertEqual(concept['display'], 'Paracetamol')
        
    @responses.activate
    def test_lookup_concept_not_found(self):
        responses.add(
            responses.GET,
            'http://localhost:8087/v1/concepts/snomed/invalid-code',
            json={'error': 'Concept not found'},
            status=404
        )
        
        concept = self.client.lookup_concept('snomed', 'invalid-code')
        self.assertIsNone(concept)
        
    def test_search_concepts_validation(self):
        # Test input validation
        with self.assertRaises(ValueError):
            self.client.search_concepts('')  # Empty query
            
        with self.assertRaises(ValueError):
            self.client.search_concepts('test', count=0)  # Invalid count
```

### Integration Testing

```go
// Go integration test
package main

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestKB7Integration(t *testing.T) {
    // Skip if not running integration tests
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    client := &KB7Client{
        BaseURL: "http://localhost:8087",
        APIKey:  "test-api-key",
        Client:  &http.Client{Timeout: 30 * time.Second},
    }
    
    t.Run("HealthCheck", func(t *testing.T) {
        health, err := client.CheckHealth()
        require.NoError(t, err)
        assert.Equal(t, "healthy", health.Status)
    })
    
    t.Run("LookupKnownConcept", func(t *testing.T) {
        concept, err := client.LookupConcept("snomed", "387517004")
        require.NoError(t, err)
        require.NotNil(t, concept)
        assert.Equal(t, "387517004", concept.Code)
        assert.Equal(t, "snomed", concept.System)
        assert.Contains(t, strings.ToLower(concept.Display), "paracetamol")
    })
    
    t.Run("SearchConcepts", func(t *testing.T) {
        results, err := client.SearchConcepts("paracetamol", "snomed", 10)
        require.NoError(t, err)
        require.NotNil(t, results)
        assert.Greater(t, results.Total, int64(0))
        assert.True(t, len(results.Concepts) > 0)
        
        // Verify first result relevance
        firstResult := results.Concepts[0]
        assert.Contains(t, strings.ToLower(firstResult.Display), "paracetamol")
    })
    
    t.Run("ValidateCode", func(t *testing.T) {
        validation, err := client.ValidateCode("387517004", "snomed")
        require.NoError(t, err)
        require.NotNil(t, validation)
        assert.True(t, validation.Valid)
        assert.Equal(t, "387517004", validation.Code)
    })
}
```

## 🔍 Debugging and Troubleshooting

### Debug Logging

```python
# Python debug logging setup
import logging
import requests.adapters
import urllib3

# Enable detailed HTTP logging
logging.basicConfig(level=logging.DEBUG)
logging.getLogger("requests.packages.urllib3").setLevel(logging.DEBUG)
logging.getLogger("urllib3.connectionpool").setLevel(logging.DEBUG)

class DebugKB7Client(KB7Client):
    def __init__(self, base_url: str, api_key: str, debug: bool = False):
        super().__init__(base_url, api_key)
        
        if debug:
            # Add detailed logging
            self.session.hooks['response'].append(self._log_response)
            
    def _log_response(self, response, *args, **kwargs):
        logging.debug(f"Request: {response.request.method} {response.request.url}")
        logging.debug(f"Response: {response.status_code} {response.reason}")
        logging.debug(f"Headers: {dict(response.headers)}")
        
        if response.headers.get('content-type', '').startswith('application/json'):
            try:
                logging.debug(f"Body: {response.json()}")
            except:
                logging.debug(f"Body: {response.text[:500]}")
```

### Performance Profiling

```javascript
// JavaScript performance profiling
class ProfiledKB7Client {
    constructor(baseUrl, apiKey) {
        this.baseUrl = baseUrl;
        this.apiKey = apiKey;
        this.stats = {
            requests: 0,
            totalTime: 0,
            errors: 0,
            cacheHits: 0,
            cacheMisses: 0
        };
    }
    
    async lookupConcept(system, code) {
        const startTime = performance.now();
        this.stats.requests++;
        
        try {
            const result = await this._makeRequest(`/v1/concepts/${system}/${code}`);
            const endTime = performance.now();
            
            this.stats.totalTime += (endTime - startTime);
            
            console.log(`Lookup ${system}:${code} took ${(endTime - startTime).toFixed(2)}ms`);
            return result;
            
        } catch (error) {
            this.stats.errors++;
            throw error;
        }
    }
    
    getPerformanceStats() {
        return {
            ...this.stats,
            averageTime: this.stats.requests > 0 ? this.stats.totalTime / this.stats.requests : 0,
            errorRate: this.stats.requests > 0 ? this.stats.errors / this.stats.requests : 0,
            cacheHitRate: (this.stats.cacheHits + this.stats.cacheMisses) > 0 
                ? this.stats.cacheHits / (this.stats.cacheHits + this.stats.cacheMisses) 
                : 0
        };
    }
}
```

## 📚 Best Practices

### 1. Error Handling
- Always handle HTTP error status codes appropriately
- Implement retry logic with exponential backoff
- Use circuit breakers for resilience
- Log errors with sufficient context for debugging

### 2. Performance Optimization
- Implement client-side caching for frequently accessed concepts
- Use batch operations when looking up multiple codes
- Consider connection pooling for high-volume applications
- Monitor and optimize API call patterns

### 3. Security
- Never hardcode API keys in source code
- Use environment variables or secure credential storage
- Implement proper authentication token management
- Validate all inputs before sending to the API

### 4. Monitoring
- Track API usage metrics and performance
- Set up alerts for error rates and response times
- Monitor cache hit rates and optimize accordingly
- Implement health checks in your application

For additional development support, contact the Clinical Platform Team at clinical-platform@hospital.com.