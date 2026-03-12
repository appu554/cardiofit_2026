# KB-6 Formulary Management Service - API Reference

## Base Information
- **Service**: KB-6 Formulary Management Service  
- **Version**: 1.0.0
- **Base URL**: `http://localhost:8087/api/v1`
- **gRPC Port**: 8086
- **REST Port**: 8087

## Authentication
All endpoints require Bearer token authentication:
```
Authorization: Bearer <your-jwt-token>
```

## Core Formulary Endpoints

### Get Drug Coverage
**GET** `/formulary/coverage`

Check formulary coverage for a specific drug and insurance plan.

#### Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| drug_id | string | Yes | RxNorm code for the drug |
| payer_id | string | Yes | Insurance payer identifier |
| member_id | string | No | Member identifier |
| formulary_id | string | No | Specific formulary identifier |

#### Response
```json
{
  "dataset_version": "kb6.formulary.2025Q3.v1",
  "covered": true,
  "coverage_status": "preferred",
  "tier": "tier2_preferred_brand",
  "cost": {
    "copay_amount": 25.00,
    "coinsurance_percent": 0,
    "deductible_applies": false,
    "estimated_patient_cost": 25.00,
    "drug_cost": 125.50
  },
  "prior_authorization_required": false,
  "step_therapy_required": false,
  "restrictions": [],
  "alternatives": []
}
```

### Get Drug Alternatives
**GET** `/formulary/alternatives`

Find alternative medications for a given drug.

#### Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| drug_id | string | Yes | RxNorm code for primary drug |
| payer_id | string | Yes | Insurance payer identifier |
| therapeutic_class | string | No | Filter by therapeutic class |
| max_results | int | No | Maximum alternatives to return (default: 10) |

### Search Formulary
**GET** `/formulary/search`

Search formulary drugs with filters and pagination.

#### Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| query | string | Yes | Search term (drug name, generic name, etc.) |
| payer_id | string | No | Filter by payer |
| limit | int | No | Results per page (default: 50) |
| offset | int | No | Pagination offset (default: 0) |

### Get Formulary Information
**GET** `/formulary/info/{formulary_id}`

Retrieve metadata about a specific formulary.

## Cost Analysis Endpoints

### Analyze Drug Costs
**POST** `/cost/analyze`

Perform comprehensive cost analysis with intelligent alternative discovery.

#### Request Body
```json
{
  "transaction_id": "cost-analysis-20250903-001",
  "drug_rxnorms": ["197361", "308136", "284635"],
  "payer_id": "aetna-001", 
  "plan_id": "aetna-standard-2025",
  "quantity": 30,
  "include_alternatives": true,
  "optimization_goal": "balanced"
}
```

#### Optimization Goals
- **cost**: Maximize cost savings
- **efficacy**: Prioritize clinical efficacy
- **safety**: Prioritize safety profile
- **balanced**: Multi-criteria optimization (default)

#### Response
```json
{
  "dataset_version": "kb6.formulary.2025Q3.v1",
  "total_primary_cost": 285.75,
  "total_alternative_cost": 167.25,
  "total_savings": 118.50,
  "savings_percent": 41.47,
  "drug_analysis": [
    {
      "drug_rxnorm": "197361",
      "drug_name": "Lipitor 20mg",
      "primary_cost": 125.00,
      "best_alternative": {
        "drug_rxnorm": "83367",
        "drug_name": "Atorvastatin 20mg",
        "alternative_type": "enhanced_generic",
        "tier": "tier1_generic",
        "estimated_cost": 45.00,
        "cost_savings": 80.00,
        "cost_savings_percent": 64.0,
        "switch_complexity": "simple",
        "efficacy_rating": 0.98,
        "safety_profile": "equivalent"
      },
      "all_alternatives": [...],
      "potential_savings": 80.00
    }
  ],
  "recommendations": [
    {
      "recommendation_type": "intelligent_generic_substitution",
      "description": "AI-optimized generic substitution with $80.00 monthly savings",
      "estimated_savings": 80.00,
      "implementation_complexity": "simple",
      "required_actions": [
        "automated_generic_switching",
        "patient_notification", 
        "pharmacy_coordination"
      ],
      "clinical_impact_score": 0.95
    }
  ],
  "evidence": {
    "dataset_version": "kb6.formulary.2025Q3.v1",
    "source_system": "kb-6-formulary-intelligent-engine",
    "data_sources": ["formulary", "generics", "therapeutics", "tier_optimization", "elasticsearch"],
    "engine_version": "v2.1.0"
  }
}
```

### Optimize Medication Costs
**POST** `/cost/optimize`

Get targeted cost optimization recommendations with implementation strategies.

#### Request Body
```json
{
  "transaction_id": "optimize-20250903-001",
  "drug_rxnorms": ["197361", "308136"],
  "payer_id": "aetna-001",
  "plan_id": "aetna-standard-2025", 
  "optimization_goal": "cost",
  "include_implementation_plan": true
}
```

### Portfolio Cost Analysis
**POST** `/cost/portfolio`

Analyze entire medication portfolios with synergy identification.

#### Request Body
```json
{
  "transaction_id": "portfolio-20250903-001",
  "drug_portfolios": [
    {
      "patient_id": "patient-001",
      "drug_rxnorms": ["197361", "308136", "284635"]
    },
    {
      "patient_id": "patient-002", 
      "drug_rxnorms": ["308136", "284635", "104894"]
    }
  ],
  "payer_id": "aetna-001",
  "plan_id": "aetna-standard-2025",
  "include_risk_analysis": true,
  "optimization_goal": "balanced"
}
```

#### Response Features
- **Cross-Portfolio Analysis**: Identifies optimization patterns across multiple patients
- **Risk Assessment**: Clinical risk scoring for recommended changes
- **Implementation Planning**: Phased rollout recommendations
- **Synergy Identification**: Therapeutic class clustering with coordinated switching bonuses

## Health and Status Endpoints

### Global Health Check
**GET** `/health`

Returns overall service health status.

```json
{
  "service": "KB-6 Formulary Management Service",
  "version": "1.0.0", 
  "status": "healthy",
  "timestamp": "2025-09-03T10:15:30Z",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Database connection OK",
      "duration": "2ms"
    },
    "cache": {
      "status": "healthy", 
      "message": "Redis connection OK",
      "duration": "1ms"
    }
  }
}
```

### Component Health Checks
**GET** `/health/formulary` - Formulary service health
**GET** `/health/inventory` - Inventory service health

### Service Information
**GET** `/`

Returns service information and available endpoints.

### API Documentation
**GET** `/api/v1/docs`

Returns interactive API documentation.

## Error Codes and Responses

### Standard HTTP Status Codes
- **200**: Success
- **400**: Bad Request (invalid parameters)
- **401**: Unauthorized (missing/invalid token)
- **404**: Not Found (drug/formulary not found)
- **429**: Too Many Requests (rate limit exceeded)
- **500**: Internal Server Error
- **503**: Service Unavailable (degraded mode)

### Error Response Format
```json
{
  "error": {
    "code": "INVALID_DRUG_ID",
    "message": "The provided drug_id is not a valid RxNorm code",
    "details": {
      "drug_id": "invalid-123",
      "expected_format": "numeric RxNorm code"
    },
    "timestamp": "2025-09-03T10:15:30Z",
    "transaction_id": "error-20250903-001"
  }
}
```

### Degraded Mode Responses
When service is in degraded mode (e.g., Elasticsearch unavailable), responses include:
```json
{
  "status": {
    "code": "PARTIAL_SUCCESS",
    "message": "Response generated with limited data",
    "warnings": ["Elasticsearch unavailable - semantic search disabled"],
    "degradation_mode": "cache_only"
  }
}
```

## Rate Limiting

### Default Limits
- **100 requests per minute** per client
- **20 burst requests** allowed
- **Cost analysis endpoints**: 50 requests per minute (higher computational cost)

### Rate Limit Headers
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1693747530
X-RateLimit-Window: 60
```

## Client Integration Examples

### JavaScript/TypeScript
```typescript
interface CostAnalysisRequest {
  transaction_id: string;
  drug_rxnorms: string[];
  payer_id: string;
  plan_id: string;
  quantity?: number;
  include_alternatives?: boolean;
  optimization_goal?: 'cost' | 'efficacy' | 'safety' | 'balanced';
}

async function analyzeCosts(request: CostAnalysisRequest): Promise<CostAnalysisResponse> {
  const response = await fetch('http://localhost:8087/api/v1/cost/analyze', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getAuthToken()}`
    },
    body: JSON.stringify(request)
  });
  
  if (!response.ok) {
    throw new Error(`Cost analysis failed: ${response.statusText}`);
  }
  
  return response.json();
}
```

### Python
```python
import requests
from typing import List, Optional

class KB6FormularyClient:
    def __init__(self, base_url: str, auth_token: str):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {auth_token}',
            'Content-Type': 'application/json'
        })
    
    def analyze_costs(
        self,
        drug_rxnorms: List[str],
        payer_id: str,
        plan_id: str,
        quantity: int = 30,
        include_alternatives: bool = True,
        optimization_goal: str = 'balanced'
    ) -> dict:
        payload = {
            'transaction_id': f'py-analysis-{int(time.time())}',
            'drug_rxnorms': drug_rxnorms,
            'payer_id': payer_id,
            'plan_id': plan_id,
            'quantity': quantity,
            'include_alternatives': include_alternatives,
            'optimization_goal': optimization_goal
        }
        
        response = self.session.post(
            f'{self.base_url}/api/v1/cost/analyze',
            json=payload
        )
        response.raise_for_status()
        return response.json()
```

### Go gRPC Client
```go
import (
    "context"
    "google.golang.org/grpc"
    pb "kb-formulary/proto/kb6"
)

func (c *KB6Client) AnalyzeCosts(ctx context.Context, drugRxNorms []string, payerID, planID string) (*pb.CostAnalysisResponse, error) {
    conn, err := grpc.Dial("localhost:8086", grpc.WithInsecure())
    if err != nil {
        return nil, err
    }
    defer conn.Close()
    
    client := pb.NewKB6ServiceClient(conn)
    
    req := &pb.CostAnalysisRequest{
        TransactionId:       generateTransactionID(),
        DrugRxnorms:        drugRxNorms,
        PayerId:            payerID,
        PlanId:             planID,
        Quantity:           30,
        IncludeAlternatives: true,
        OptimizationGoal:   "balanced",
    }
    
    return client.GetCostAnalysis(ctx, req)
}
```

## Performance Benchmarks

### 📈 Expected Performance
- **Single Drug Analysis**: 25-50ms (p95)
- **Multi-Drug Analysis (5 drugs)**: 75-150ms (p95)
- **Portfolio Analysis (10 patients)**: 200-400ms (p95)
- **Cache Hit Rate**: >90% for repeated queries
- **Elasticsearch Search**: 50-150ms (p95)

### 🔧 Optimization Tips
1. **Batch Requests**: Combine multiple drugs in single analysis request
2. **Cache Strategy**: Implement client-side caching for frequently accessed data
3. **Parallel Processing**: Use concurrent requests for independent operations
4. **Smart Pagination**: Use appropriate limit/offset for search operations

---

**API Reference Status**: ✅ **Complete**
- All endpoint documentation
- Request/response schemas
- Client integration examples
- Performance expectations
- Error handling patterns