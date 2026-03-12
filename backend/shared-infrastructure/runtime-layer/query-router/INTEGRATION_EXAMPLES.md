# Query Router Integration Examples

This guide provides practical integration examples for the CardioFit Multi-KB Query Router across different service contexts.

## Table of Contents

1. [Service Integration Patterns](#service-integration-patterns)
2. [Clinical Workflow Examples](#clinical-workflow-examples)
3. [Error Handling Patterns](#error-handling-patterns)
4. [Testing Strategies](#testing-strategies)
5. [Performance Tuning](#performance-tuning)

## Service Integration Patterns

### 1. Medication Service Integration

```python
# medication_service.py
from typing import List, Dict, Any
from shared_infrastructure.runtime_layer.query_router import (
    MultiKBQueryRouter,
    MultiKBQueryRequest,
    QueryPattern
)

class MedicationService:
    """Service for medication management with multi-KB support"""

    def __init__(self, config: Dict[str, Any]):
        self.router = MultiKBQueryRouter(config)
        self.service_id = "medication-service"

    async def initialize(self):
        """Initialize router and connections"""
        await self.router.initialize_clients()

    async def get_medication_profile(self, patient_id: str) -> Dict[str, Any]:
        """
        Get comprehensive medication profile for a patient

        Queries:
        - KB1: Current medications
        - KB5: Drug interactions
        - KB3: Dosing calculations
        - KB7: Terminology mappings
        """
        # Step 1: Get patient's current medications from KB1
        medications_request = MultiKBQueryRequest(
            service_id=self.service_id,
            kb_id="kb1",
            pattern=QueryPattern.KB1_PATIENT_LOOKUP,
            params={"patient_id": patient_id, "resource_type": "MedicationStatement"}
        )
        medications_response = await self.router.route_query(medications_request)

        medications = medications_response.data.get('medications', [])
        if not medications:
            return {"patient_id": patient_id, "medications": [], "interactions": [], "alerts": []}

        # Step 2: Extract drug codes for interaction checking
        drug_rxnorms = [med['rxnorm'] for med in medications if 'rxnorm' in med]

        # Step 3: Check drug interactions (Cross-KB query)
        interactions_request = MultiKBQueryRequest(
            service_id=self.service_id,
            kb_id=None,  # Cross-KB query
            pattern=QueryPattern.CROSS_KB_DRUG_ANALYSIS,
            params={
                "drug_codes": drug_rxnorms,
                "patient_id": patient_id
            },
            cross_kb_scope=["kb3", "kb5", "kb7"],
            priority="high"  # High priority for safety checks
        )
        interactions_response = await self.router.route_query(interactions_request)

        # Step 4: Get terminology details for each medication
        terminology_tasks = []
        for drug_code in drug_rxnorms:
            term_request = MultiKBQueryRequest(
                service_id=self.service_id,
                kb_id="kb7",
                pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
                params={"code": drug_code, "system": "RxNorm"}
            )
            terminology_tasks.append(self.router.route_query(term_request))

        terminology_responses = await asyncio.gather(*terminology_tasks)

        # Step 5: Compile comprehensive profile
        return {
            "patient_id": patient_id,
            "medications": medications,
            "interactions": interactions_response.data.get('interactions', []),
            "dosing": interactions_response.data.get('calculations', []),
            "terminology": [resp.data for resp in terminology_responses],
            "query_metadata": {
                "sources_used": interactions_response.sources_used,
                "total_latency": interactions_response.latency,
                "cache_status": interactions_response.cache_status
            }
        }

    async def check_drug_safety(self, drug_code: str, patient_context: Dict) -> Dict[str, Any]:
        """
        Comprehensive drug safety check before prescribing
        """
        # Safety analysis pattern
        safety_request = MultiKBQueryRequest(
            service_id=self.service_id,
            kb_id=None,
            pattern=QueryPattern.CROSS_KB_DRUG_ANALYSIS,
            params={
                "drug_code": drug_code,
                "patient_weight": patient_context.get('weight'),
                "patient_age": patient_context.get('age'),
                "kidney_function": patient_context.get('egfr'),
                "liver_function": patient_context.get('alt'),
                "current_medications": patient_context.get('current_meds', [])
            },
            cross_kb_scope=["kb3", "kb4", "kb5", "kb6"],
            require_snapshot=True  # Ensure consistency for safety check
        )

        safety_response = await self.router.route_query(safety_request)

        # Interpret safety results
        alerts = []
        if safety_response.data.get('interactions'):
            alerts.append({
                "type": "interaction",
                "severity": "high",
                "details": safety_response.data['interactions']
            })

        if safety_response.data.get('contraindications'):
            alerts.append({
                "type": "contraindication",
                "severity": "critical",
                "details": safety_response.data['contraindications']
            })

        return {
            "drug_code": drug_code,
            "is_safe": len(alerts) == 0,
            "alerts": alerts,
            "recommended_dose": safety_response.data.get('recommended_dose'),
            "adjustments": safety_response.data.get('dose_adjustments', []),
            "evidence_score": safety_response.data.get('evidence_score')
        }
```

### 2. Clinical Reasoning Service Integration

```python
# clinical_reasoning_service.py
from shared_infrastructure.runtime_layer.query_router import (
    MultiKBQueryRouter,
    MultiKBQueryRequest,
    QueryPattern
)

class ClinicalReasoningService:
    """Advanced clinical reasoning with multi-KB integration"""

    def __init__(self, config: Dict[str, Any]):
        self.router = MultiKBQueryRouter(config)
        self.service_id = "clinical-reasoning-service"

    async def analyze_clinical_scenario(self, scenario: Dict[str, Any]) -> Dict[str, Any]:
        """
        Analyze complex clinical scenario using multiple knowledge bases
        """
        patient_id = scenario['patient_id']
        symptoms = scenario.get('symptoms', [])
        lab_results = scenario.get('lab_results', [])

        # Phase 1: Gather patient context from multiple KBs
        context_request = MultiKBQueryRequest(
            service_id=self.service_id,
            kb_id=None,
            pattern=QueryPattern.CROSS_KB_PATIENT_VIEW,
            params={"patient_id": patient_id},
            cross_kb_scope=["kb1", "kb7"],
            require_snapshot=True  # Consistent view important for reasoning
        )
        context = await self.router.route_query(context_request)

        # Phase 2: Semantic analysis of symptoms
        semantic_request = MultiKBQueryRequest(
            service_id=self.service_id,
            kb_id="kb7",
            pattern=QueryPattern.SEMANTIC_INFERENCE,
            params={
                "concepts": symptoms,
                "context": "differential_diagnosis"
            }
        )
        semantic_analysis = await self.router.route_query(semantic_request)

        # Phase 3: Guideline matching from KB2
        guideline_request = MultiKBQueryRequest(
            service_id=self.service_id,
            kb_id="kb2",
            pattern=QueryPattern.KB2_GUIDELINE_SEARCH,
            params={
                "conditions": semantic_analysis.data.get('inferred_conditions', []),
                "patient_demographics": context.data.get('demographics')
            }
        )
        guidelines = await self.router.route_query(guideline_request)

        # Phase 4: Clinical reasoning synthesis
        reasoning_request = MultiKBQueryRequest(
            service_id=self.service_id,
            kb_id=None,
            pattern=QueryPattern.CLINICAL_REASONING,
            params={
                "patient_context": context.data,
                "clinical_features": symptoms + lab_results,
                "semantic_analysis": semantic_analysis.data,
                "applicable_guidelines": guidelines.data
            },
            cross_kb_scope=["kb1", "kb2", "kb6", "kb7"],
            priority="high"
        )
        reasoning_result = await self.router.route_query(reasoning_request)

        return {
            "scenario_id": scenario.get('id'),
            "patient_id": patient_id,
            "differential_diagnosis": reasoning_result.data.get('differentials', []),
            "recommended_tests": reasoning_result.data.get('tests', []),
            "treatment_options": reasoning_result.data.get('treatments', []),
            "confidence_scores": reasoning_result.data.get('confidence', {}),
            "evidence_basis": {
                "guidelines_applied": guidelines.data,
                "semantic_concepts": semantic_analysis.data,
                "sources_consulted": reasoning_result.kb_sources
            }
        }

    async def validate_clinical_decision(self, decision: Dict[str, Any]) -> Dict[str, Any]:
        """
        Validate a clinical decision against evidence and safety rules
        """
        # KB4: Safety rules validation
        safety_validation = MultiKBQueryRequest(
            service_id=self.service_id,
            kb_id="kb4",
            pattern="safety_rule_check",
            params={
                "decision_type": decision['type'],
                "parameters": decision['parameters']
            }
        )
        safety_result = await self.router.route_query(safety_validation)

        # KB6: Evidence scoring
        evidence_request = MultiKBQueryRequest(
            service_id=self.service_id,
            kb_id="kb6",
            pattern="evidence_scoring",
            params={
                "intervention": decision['intervention'],
                "condition": decision['condition']
            }
        )
        evidence_result = await self.router.route_query(evidence_request)

        return {
            "decision": decision,
            "is_valid": safety_result.data.get('passes_safety', False),
            "safety_concerns": safety_result.data.get('violations', []),
            "evidence_level": evidence_result.data.get('level', 'unknown'),
            "evidence_score": evidence_result.data.get('score', 0.0),
            "recommendations": evidence_result.data.get('recommendations', [])
        }
```

## Clinical Workflow Examples

### Complete Admission Workflow

```python
async def patient_admission_workflow(patient_data: Dict[str, Any]):
    """
    Complete workflow for patient admission with all necessary checks
    """
    router = MultiKBQueryRouter(config)
    workflow_results = {
        "patient_id": patient_data['id'],
        "admission_time": datetime.utcnow(),
        "checks_performed": []
    }

    # 1. Terminology standardization
    terminology_request = MultiKBQueryRequest(
        service_id="admission-workflow",
        kb_id="kb7",
        pattern=QueryPattern.KB7_TERMINOLOGY_SEARCH,
        params={
            "diagnosis_text": patient_data['chief_complaint'],
            "map_to": ["ICD10", "SNOMED-CT"]
        }
    )
    terminology = await router.route_query(terminology_request)
    workflow_results['standardized_diagnoses'] = terminology.data

    # 2. Check medication history and interactions
    med_history_request = MultiKBQueryRequest(
        service_id="admission-workflow",
        kb_id=None,
        pattern=QueryPattern.CROSS_KB_DRUG_ANALYSIS,
        params={
            "patient_id": patient_data['id'],
            "include_history": True
        },
        cross_kb_scope=["kb1", "kb3", "kb5"]
    )
    med_history = await router.route_query(med_history_request)
    workflow_results['medication_review'] = med_history.data

    # 3. Retrieve applicable clinical guidelines
    guidelines_request = MultiKBQueryRequest(
        service_id="admission-workflow",
        kb_id="kb2",
        pattern=QueryPattern.KB2_GUIDELINE_SEARCH,
        params={
            "conditions": terminology.data.get('icd10_codes', []),
            "care_setting": "inpatient"
        }
    )
    guidelines = await router.route_query(guidelines_request)
    workflow_results['clinical_guidelines'] = guidelines.data

    # 4. Generate care plan based on all information
    care_plan_request = MultiKBQueryRequest(
        service_id="admission-workflow",
        kb_id=None,
        pattern=QueryPattern.CLINICAL_REASONING,
        params={
            "patient_data": patient_data,
            "diagnoses": terminology.data,
            "medications": med_history.data,
            "guidelines": guidelines.data
        },
        cross_kb_scope=["kb1", "kb2", "kb8"],  # Include clinical workflows (KB8)
        require_snapshot=True  # Ensure consistency
    )
    care_plan = await router.route_query(care_plan_request)
    workflow_results['care_plan'] = care_plan.data

    workflow_results['checks_performed'] = [
        "terminology_standardization",
        "medication_reconciliation",
        "guideline_retrieval",
        "care_plan_generation"
    ]

    return workflow_results
```

## Error Handling Patterns

### Comprehensive Error Handling

```python
class RobustQueryHandler:
    """Robust query handling with comprehensive error management"""

    def __init__(self, router: MultiKBQueryRouter):
        self.router = router
        self.max_retries = 3
        self.fallback_chain = {
            QueryPattern.KB7_TERMINOLOGY_SEARCH: [
                QueryPattern.KB7_TERMINOLOGY_LOOKUP,
                QueryPattern.SEMANTIC_INFERENCE
            ]
        }

    async def execute_with_retry(self, request: MultiKBQueryRequest):
        """Execute query with retry logic and fallback patterns"""

        last_error = None
        original_pattern = request.pattern

        for attempt in range(self.max_retries):
            try:
                # Attempt primary query
                response = await self.router.route_query(request)

                # Check if response is valid
                if response.data and 'error' not in response.data:
                    return response

                # If error in response, log and retry
                if 'error' in response.data:
                    logger.warning(f"Query error on attempt {attempt + 1}: {response.data['error']}")
                    last_error = response.data['error']

            except Exception as e:
                logger.error(f"Exception on attempt {attempt + 1}: {e}")
                last_error = str(e)

                # Try fallback patterns if available
                if original_pattern in self.fallback_chain and attempt < self.max_retries - 1:
                    fallback_patterns = self.fallback_chain[original_pattern]
                    if attempt < len(fallback_patterns):
                        request.pattern = fallback_patterns[attempt]
                        logger.info(f"Trying fallback pattern: {request.pattern}")
                        continue

            # Exponential backoff between retries
            if attempt < self.max_retries - 1:
                await asyncio.sleep(2 ** attempt)

        # All retries exhausted, return error response
        return MultiKBQueryResponse(
            data={"error": f"Query failed after {self.max_retries} attempts", "last_error": last_error},
            sources_used=[],
            kb_sources=[],
            latency=0.0,
            cache_status="error"
        )

    async def execute_with_timeout(self, request: MultiKBQueryRequest, timeout_seconds: int = 30):
        """Execute query with timeout"""
        try:
            response = await asyncio.wait_for(
                self.router.route_query(request),
                timeout=timeout_seconds
            )
            return response
        except asyncio.TimeoutError:
            logger.error(f"Query timeout after {timeout_seconds} seconds")
            return MultiKBQueryResponse(
                data={"error": f"Query timeout after {timeout_seconds} seconds"},
                sources_used=[],
                kb_sources=[],
                latency=timeout_seconds * 1000,
                cache_status="timeout"
            )
```

## Testing Strategies

### Integration Testing

```python
import pytest
from unittest.mock import AsyncMock, patch

class TestQueryRouterIntegration:
    """Integration tests for Query Router"""

    @pytest.fixture
    async def router(self):
        """Create router with test configuration"""
        test_config = {
            'neo4j': {'uri': 'bolt://test:7687', 'auth': ('test', 'test')},
            'postgres': {'host': 'test', 'port': 5432, 'database': 'test'}
        }
        router = MultiKBQueryRouter(test_config)

        # Mock client initialization
        router._neo4j_manager = AsyncMock()
        router._postgres_client = AsyncMock()

        return router

    @pytest.mark.asyncio
    async def test_single_kb_query(self, router):
        """Test single KB query routing"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            kb_id="kb7",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"}
        )

        # Mock response
        router._postgres_client.query.return_value = {
            'code': 'I10',
            'display': 'Essential hypertension',
            'system': 'ICD10'
        }

        response = await router.route_query(request)

        assert response.data['code'] == 'I10'
        assert 'postgres' in response.sources_used
        assert response.kb_sources == ['kb7']

    @pytest.mark.asyncio
    async def test_cross_kb_query(self, router):
        """Test cross-KB query orchestration"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            kb_id=None,
            pattern=QueryPattern.CROSS_KB_PATIENT_VIEW,
            params={"patient_id": "12345"},
            cross_kb_scope=["kb1", "kb7"]
        )

        # Mock Neo4j responses
        router._neo4j_manager.cross_kb_query.return_value = {
            'patient': {'id': '12345', 'name': 'Test Patient'},
            'medications': [{'rxnorm': '123', 'name': 'Test Med'}],
            'terminology': [{'code': 'I10', 'display': 'Hypertension'}]
        }

        response = await router.route_query(request)

        assert response.data['patient']['id'] == '12345'
        assert len(response.kb_sources) == 2
        assert 'kb1' in response.kb_sources
        assert 'kb7' in response.kb_sources

    @pytest.mark.asyncio
    async def test_fallback_mechanism(self, router):
        """Test fallback routing on primary failure"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            kb_id="kb7",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"}
        )

        # Simulate Neo4j failure
        router._neo4j_manager.query.side_effect = Exception("Connection failed")

        # Mock PostgreSQL fallback success
        router._postgres_client.query.return_value = {
            'code': 'I10',
            'display': 'Essential hypertension',
            'system': 'ICD10'
        }

        response = await router.route_query(request)

        assert response.cache_status == 'fallback'
        assert 'postgres_fallback' in response.sources_used

    @pytest.mark.asyncio
    async def test_caching_behavior(self, router):
        """Test caching behavior"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            kb_id="kb7",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"}
        )

        # Mock cache miss then hit
        router._redis_l2_client = AsyncMock()
        router._redis_l2_client.get.side_effect = [None, {'cached': 'data'}]

        # First call - cache miss
        response1 = await router.route_query(request)
        assert response1.cache_status == 'miss'

        # Second call - cache hit
        response2 = await router.route_query(request)
        assert response2.cache_status == 'hit'
```

## Performance Tuning

### Optimization Configuration

```python
# performance_tuning.py

class PerformanceOptimizedRouter:
    """Performance-optimized router configuration"""

    def __init__(self):
        self.config = self._build_optimized_config()
        self.router = MultiKBQueryRouter(self.config)

    def _build_optimized_config(self):
        """Build performance-optimized configuration"""
        return {
            # Connection pooling
            'neo4j': {
                'uri': 'bolt://localhost:7687',
                'auth': ('neo4j', 'password'),
                'max_connection_pool_size': 100,  # Increased pool size
                'connection_acquisition_timeout': 5,
                'max_transaction_retry_time': 10
            },

            # Caching configuration
            'redis_l2': {
                'host': 'localhost',
                'port': 6379,
                'db': 0,
                'ttl': 300,  # 5 minutes
                'max_connections': 50,
                'socket_keepalive': True,
                'socket_keepalive_options': {
                    1: 1,  # TCP_KEEPIDLE
                    2: 3,  # TCP_KEEPINTVL
                    3: 5   # TCP_KEEPCNT
                }
            },

            'redis_l3': {
                'host': 'localhost',
                'port': 6380,
                'db': 0,
                'ttl': 3600,  # 1 hour for complex queries
                'max_connections': 50
            },

            # Query optimization
            'query_optimization': {
                'enable_parallel_execution': True,
                'max_parallel_queries': 10,
                'query_timeout': 30,  # seconds
                'enable_query_batching': True,
                'batch_size': 100
            },

            # Performance monitoring
            'monitoring': {
                'enable_metrics': True,
                'metrics_interval': 60,  # seconds
                'slow_query_threshold': 1000,  # ms
                'enable_query_logging': True
            }
        }

    async def warmup_caches(self):
        """Pre-warm caches with common queries"""
        common_queries = [
            # Common terminology lookups
            MultiKBQueryRequest(
                service_id="cache-warmer",
                kb_id="kb7",
                pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
                params={"code": code, "system": "ICD10"}
            ) for code in ['I10', 'E11', 'J44', 'N18']
        ]

        # Execute queries to populate cache
        for query in common_queries:
            await self.router.route_query(query)

        logger.info(f"Cache warmed with {len(common_queries)} common queries")

    async def optimize_for_batch_processing(self, requests: List[MultiKBQueryRequest]):
        """Optimize batch query processing"""

        # Group queries by KB and pattern
        grouped = {}
        for request in requests:
            key = (request.kb_id, request.pattern)
            if key not in grouped:
                grouped[key] = []
            grouped[key].append(request)

        # Process groups in parallel
        tasks = []
        for (kb_id, pattern), group_requests in grouped.items():
            if len(group_requests) > 1:
                # Batch similar queries
                batch_request = self._create_batch_request(kb_id, pattern, group_requests)
                tasks.append(self.router.route_query(batch_request))
            else:
                tasks.append(self.router.route_query(group_requests[0]))

        # Execute all queries in parallel
        responses = await asyncio.gather(*tasks)
        return responses

    def _create_batch_request(self, kb_id: str, pattern: QueryPattern,
                            requests: List[MultiKBQueryRequest]):
        """Create batched request for similar queries"""
        # Combine parameters for batch processing
        combined_params = {
            'batch_queries': [req.params for req in requests],
            'batch_size': len(requests)
        }

        return MultiKBQueryRequest(
            service_id="batch-processor",
            kb_id=kb_id,
            pattern=pattern,
            params=combined_params
        )
```

## Summary

The Query Router integration examples demonstrate:

1. **Service Integration**: How different services (medication, clinical reasoning) integrate with the router
2. **Clinical Workflows**: Complete workflows that leverage multiple Knowledge Bases
3. **Error Handling**: Robust patterns for handling failures and implementing fallbacks
4. **Testing Strategies**: Comprehensive testing approaches for integration validation
5. **Performance Tuning**: Optimization techniques for production deployments

These patterns ensure reliable, performant, and maintainable integration with the CardioFit runtime layer's Query Router component.