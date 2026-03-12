# Flow 2: Context Recipes Integration Implementation Plan

## 🎯 Overview

This document outlines the implementation plan for **Flow 2: Context Recipes in Context Service** architecture, which integrates the Medication Service with the Context Service using recipe-based data optimization.

## 🏗️ Architecture Summary

### Current State
- ✅ **Medication Service**: 29 Clinical Logic Recipes implemented (MedicationRecipeBook.txt)
- ✅ **Context Service**: Exists with ContextRecipeBook.txt specifications
- ❌ **Integration Layer**: Missing connection between services

### Target Architecture
```
Workflow Engine
    ↓
Medication Service (Port 8009)
├── Clinical Recipe Engine ✅
├── Context Service Client ❌ (IMPLEMENT)
└── Recipe Orchestrator ❌ (IMPLEMENT)
    ↓
Context Service (Port 8XXX)
├── Context Recipe Engine ❌ (IMPLEMENT)
├── Data Aggregation Layer ❌ (IMPLEMENT)
└── API Endpoints ❌ (IMPLEMENT)
    ↓
Data Sources (Lab, Patient, Insurance Services)
```

## 🔄 Complete Flow

### Step-by-Step Execution
1. **Medication Request** arrives at Medication Service
2. **Recipe Orchestrator** determines which context recipe is needed
3. **Context Service Client** calls Context Service with recipe ID
4. **Context Recipe Engine** executes recipe method (not YAML parsing)
5. **Parallel Data Fetcher** gathers data from multiple sources
6. **Data Aggregation Engine** combines data into optimized context
7. **Context Service** returns optimized context to Medication Service
8. **Clinical Recipe Engine** executes with real context data
9. **Recipe Orchestrator** generates final medication proposal

### Example Flow
```python
# 1. Request arrives
request = {
    "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
    "medication": {"name": "warfarin", "is_anticoagulant": True}
}

# 2. Determine context recipe
context_recipe_id = "anticoagulation_context_v1_0"

# 3. Call Context Service
context_data = await context_service_client.execute_recipe(
    patient_id=request.patient_id,
    recipe_id=context_recipe_id
)

# 4. Execute clinical recipes with real context
clinical_results = await clinical_recipe_engine.execute_recipes(
    context=context_data,
    medication=request.medication
)
```

## 📋 Implementation Tasks

### Phase 1: Context Service Implementation

#### Task 3.1: Context Recipe Engine Implementation
**Location**: `backend/services/context-service/`
**Description**: Implement Context Recipe Engine classes based on ContextRecipeBook.txt specifications

```python
# context_recipe_engine.py
class ContextRecipeEngine:
    """Execute context recipes based on ContextRecipeBook.txt specs"""
    
    async def execute_recipe(self, recipe_id: str, patient_id: str):
        """Main entry point for recipe execution"""
        if recipe_id == "anticoagulation_context_v1_0":
            return await self._execute_anticoagulation_context(patient_id)
        elif recipe_id == "chemotherapy_context_v1_0":
            return await self._execute_chemotherapy_context(patient_id)
        # ... implement all 30+ context recipes
    
    async def _execute_anticoagulation_context(self, patient_id: str):
        """Based on ContextRecipeBook.txt anticoagulation specs"""
        # Parallel data fetching
        lab_data = await self.lab_service.get_coagulation_studies(patient_id)
        medication_data = await self.medication_service.get_current_meds(patient_id)
        risk_data = await self.patient_service.get_bleeding_risk(patient_id)
        
        # Aggregate and return optimized context
        return {
            "patient": {"demographics": {...}},
            "labs": lab_data,
            "medications": medication_data,
            "risk_scores": risk_data,
            "metadata": {"recipe_id": "anticoagulation_context_v1_0"}
        }
```

#### Task 3.2: Parallel Data Fetcher Implementation
**Description**: Implement parallel data fetching from multiple sources

```python
# parallel_data_fetcher.py
class ParallelDataFetcher:
    async def fetch_multiple_sources(self, patient_id: str, data_requirements: dict):
        """Fetch data from multiple sources in parallel"""
        tasks = []
        
        for source, requirements in data_requirements.items():
            if source == "lab-service":
                tasks.append(self.lab_service.get_data(patient_id, requirements))
            elif source == "patient-service":
                tasks.append(self.patient_service.get_data(patient_id, requirements))
            # ... add more sources
        
        results = await asyncio.gather(*tasks, return_exceptions=True)
        return self._handle_partial_failures(results)
```

#### Task 3.3: Data Aggregation Engine Implementation
**Description**: Combine data from multiple sources into optimized context payload

#### Task 3.4: Performance Optimization Implementation
**Description**: Implement caching, timeouts, and performance strategies for sub-100ms targets

### Phase 2: Medication Service Integration

#### Task 1.1: HTTP Client Implementation
**Location**: `backend/services/medication-service/app/infrastructure/external/`
**Description**: Create ContextServiceClient for API communication

```python
# context_service_client.py
class ContextServiceClient:
    def __init__(self):
        self.base_url = "http://localhost:8XXX"  # Context Service port
        self.timeout = 100  # 100ms timeout
    
    async def execute_recipe(self, patient_id: str, recipe_id: str):
        """Call Context Service to execute recipe"""
        try:
            response = await self.http_client.post(
                f"{self.base_url}/api/context/execute",
                json={
                    "patient_id": patient_id,
                    "recipe_id": recipe_id,
                    "urgency": "routine"
                },
                timeout=self.timeout
            )
            return response.json()
        except Exception as e:
            return await self._handle_context_failure(e, patient_id, recipe_id)
```

#### Task 2.1: Recipe Orchestrator Implementation
**Location**: `backend/services/medication-service/app/domain/services/`
**Description**: Build orchestrator that coordinates context and clinical recipes

```python
# recipe_orchestrator.py
class RecipeOrchestrator:
    def __init__(self):
        self.context_service_client = ContextServiceClient()
        self.clinical_recipe_engine = ClinicalRecipeEngine()
    
    async def execute_medication_safety(self, request):
        """Main orchestration method"""
        # 1. Determine context recipe needed
        context_recipe_id = self._determine_context_recipe(request)
        
        # 2. Get context from Context Service
        context_data = await self.context_service_client.execute_recipe(
            patient_id=request.patient_id,
            recipe_id=context_recipe_id
        )
        
        # 3. Transform context for clinical recipes
        recipe_context = self._transform_context(context_data, request)
        
        # 4. Execute clinical recipes with real context
        clinical_results = await self.clinical_recipe_engine.execute_recipes(
            context=recipe_context
        )
        
        return clinical_results
    
    def _determine_context_recipe(self, request):
        """Determine which context recipe is needed"""
        medication = request.medication
        
        if medication.get('is_anticoagulant'):
            return "anticoagulation_context_v1_0"
        elif medication.get('is_chemotherapy'):
            return "chemotherapy_context_v1_0"
        elif medication.get('requires_renal_adjustment'):
            return "renal_safety_context_v1_0"
        else:
            return "medication_safety_base_context_v1_0"
```

### Phase 3: Integration & Testing

#### Task 4.1: Update Clinical Recipes with Context Integration
**Description**: Update existing clinical recipes to use real context instead of mock data

#### Task 4.2: End-to-End Recipe Flow Implementation
**Description**: Implement complete flow with error handling and performance monitoring

#### Task 4.3: Integration Testing
**Description**: Test complete flow from request to response

## 🎯 Key Implementation Points

### Recipe Books Are Implementation Guides
- **ContextRecipeBook.txt**: Specifications for implementing context recipe methods
- **MedicationRecipeBook.txt**: Specifications for clinical recipe integration
- **NOT YAML files**: These are architectural guides, not configuration files to parse

### Context Recipe Mapping
Each clinical recipe has a corresponding context recipe:
- `medication-safety-anticoagulation-v3.0` → `anticoagulation_context_v1_0`
- `medication-safety-chemotherapy-v3.0` → `chemotherapy_context_v1_0`
- `population-pregnancy-v3.0` → `pregnancy_safety_context_v1_0`

### Performance Targets
- **Context fetching**: <100ms
- **Clinical recipe execution**: <5ms
- **Total response time**: <200ms

## 🚀 Implementation Priority

1. **Start with**: Task 3.1 (Context Recipe Engine)
2. **Then**: Task 1.1 (HTTP Client Implementation)
3. **Then**: Task 2.1 (Recipe Orchestrator)
4. **Finally**: Task 4.2 (Integration Testing)

## 📊 Success Criteria

- ✅ All 29 clinical recipes use real context data
- ✅ Context Service responds in <100ms
- ✅ Parallel data fetching works correctly
- ✅ Graceful degradation when Context Service unavailable
- ✅ End-to-end flow tested and validated

## 🔗 Related Files

- `MedicationRecipeBook.txt`: Clinical recipe specifications
- `ContextRecipeBook.txt`: Context recipe specifications  
- `clinical_recipe_engine.py`: Existing clinical recipe implementation
- `IMPLEMENTATION_PLAN.md`: Original implementation plan

## 🔧 Technical Implementation Details

### Context Service API Design

#### REST Endpoints
```
POST /api/context/execute
{
    "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
    "recipe_id": "anticoagulation_context_v1_0",
    "urgency": "routine|urgent|emergency"
}

Response:
{
    "context": {
        "patient": {...},
        "labs": {...},
        "medications": {...}
    },
    "metadata": {
        "recipe_id": "anticoagulation_context_v1_0",
        "execution_time_ms": 85,
        "data_sources": ["lab-service", "patient-service"],
        "cache_hit": false
    }
}
```

#### GraphQL Alternative
```graphql
mutation ExecuteContextRecipe($input: ContextRecipeInput!) {
    executeContextRecipe(input: $input) {
        context {
            patient { demographics, conditions }
            labs { latest, historical }
            medications { current, recent }
        }
        metadata {
            recipeId
            executionTimeMs
            dataSources
        }
    }
}
```

### Error Handling Strategy

#### Context Service Failures
```python
class ContextServiceClient:
    async def _handle_context_failure(self, error, patient_id, recipe_id):
        """Graceful degradation when Context Service fails"""
        if isinstance(error, TimeoutError):
            # Use cached context if available
            cached_context = await self.cache.get(f"context:{patient_id}:{recipe_id}")
            if cached_context:
                return cached_context

        # Fallback to minimal context
        return await self._get_minimal_context(patient_id)

    async def _get_minimal_context(self, patient_id):
        """Minimal context for basic safety checks"""
        return {
            "patient": {"id": patient_id},
            "metadata": {"fallback": True, "limited_context": True}
        }
```

#### Partial Data Failures
```python
class ParallelDataFetcher:
    def _handle_partial_failures(self, results):
        """Handle when some data sources fail"""
        successful_data = {}
        failed_sources = []

        for i, result in enumerate(results):
            if isinstance(result, Exception):
                failed_sources.append(self.source_names[i])
            else:
                successful_data.update(result)

        return {
            "data": successful_data,
            "metadata": {
                "partial_failure": len(failed_sources) > 0,
                "failed_sources": failed_sources
            }
        }
```

### Caching Strategy

#### Multi-Level Caching
```python
class ContextCacheManager:
    def __init__(self):
        self.l1_cache = {}  # In-memory cache
        self.l2_cache = RedisCache()  # Redis cache

    async def get_cached_context(self, cache_key):
        # L1 Cache (fastest)
        if cache_key in self.l1_cache:
            return self.l1_cache[cache_key]

        # L2 Cache (Redis)
        cached_data = await self.l2_cache.get(cache_key)
        if cached_data:
            self.l1_cache[cache_key] = cached_data  # Promote to L1
            return cached_data

        return None

    async def cache_context(self, cache_key, context_data, ttl_seconds):
        # Cache in both levels
        self.l1_cache[cache_key] = context_data
        await self.l2_cache.setex(cache_key, context_data, ttl_seconds)
```

#### Cache Key Strategy
```python
def generate_cache_key(patient_id: str, recipe_id: str, context_hash: str):
    """Generate cache key for context data"""
    return f"context:{recipe_id}:{patient_id}:{context_hash[:8]}"

def calculate_context_hash(patient_data, medication_data):
    """Calculate hash to detect when context needs refresh"""
    context_string = f"{patient_data.get('last_updated')}:{medication_data.get('hash')}"
    return hashlib.md5(context_string.encode()).hexdigest()
```

## 🧪 Testing Strategy

### Unit Testing
```python
# test_context_recipe_engine.py
class TestContextRecipeEngine:
    async def test_anticoagulation_context_execution(self):
        engine = ContextRecipeEngine()

        # Mock data sources
        with patch.object(engine.lab_service, 'get_coagulation_studies') as mock_lab:
            mock_lab.return_value = {"inr": 2.5, "pt": 28}

            result = await engine._execute_anticoagulation_context("patient_123")

            assert result["labs"]["inr"] == 2.5
            assert result["metadata"]["recipe_id"] == "anticoagulation_context_v1_0"
```

### Integration Testing
```python
# test_flow2_integration.py
class TestFlow2Integration:
    async def test_complete_medication_safety_flow(self):
        # Test complete flow: Request → Context → Clinical → Response
        request = MedicationSafetyRequest(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            medication={"name": "warfarin", "is_anticoagulant": True}
        )

        orchestrator = RecipeOrchestrator()
        result = await orchestrator.execute_medication_safety(request)

        assert result.overall_safety_status in ["SAFE", "WARNING", "UNSAFE"]
        assert result.context_recipe_used == "anticoagulation_context_v1_0"
        assert result.execution_time_ms < 200
```

### Performance Testing
```python
# test_performance.py
class TestPerformance:
    async def test_context_fetching_performance(self):
        """Validate sub-100ms context fetching"""
        start_time = time.time()

        context = await context_service_client.execute_recipe(
            patient_id="test_patient",
            recipe_id="anticoagulation_context_v1_0"
        )

        execution_time = (time.time() - start_time) * 1000
        assert execution_time < 100, f"Context fetching took {execution_time}ms"
```

## 📈 Monitoring & Observability

### Metrics to Track
```python
# metrics.py
class ContextServiceMetrics:
    def __init__(self):
        self.context_execution_time = Histogram('context_execution_seconds')
        self.context_cache_hits = Counter('context_cache_hits_total')
        self.context_failures = Counter('context_failures_total')
        self.data_source_latency = Histogram('data_source_latency_seconds')

    def record_context_execution(self, recipe_id, execution_time):
        self.context_execution_time.labels(recipe_id=recipe_id).observe(execution_time)

    def record_cache_hit(self, recipe_id):
        self.context_cache_hits.labels(recipe_id=recipe_id).inc()
```

### Logging Strategy
```python
# logging_config.py
LOGGING_CONFIG = {
    'version': 1,
    'formatters': {
        'structured': {
            'format': '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        }
    },
    'handlers': {
        'context_service': {
            'class': 'logging.FileHandler',
            'filename': 'context_service.log',
            'formatter': 'structured'
        }
    },
    'loggers': {
        'context_recipe_engine': {
            'handlers': ['context_service'],
            'level': 'INFO'
        }
    }
}
```

---

**Next Step**: Begin implementation with Task 3.1 - Context Recipe Engine Implementation
