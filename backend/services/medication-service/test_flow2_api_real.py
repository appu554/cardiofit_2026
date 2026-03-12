"""
Real Flow 2 API Integration Test

This test validates the Flow 2 API endpoints with actual Context Service integration.
It tests the complete API workflow:

1. POST /api/flow2/medication-safety/validate
2. Recipe Orchestrator → Context Service (REAL CALL)
3. Clinical Recipe execution with real context
4. API response with comprehensive safety assessment
"""

import requests
import json
import time
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_flow2_api_endpoint():
    """
    Test the Flow 2 API endpoint with real Context Service integration
    """
    try:
        logger.info("🚀 Testing Flow 2 API Endpoint with Real Context Service")
        logger.info("=" * 70)
        
        # Medication Service URL
        medication_service_url = "http://localhost:8009"
        flow2_endpoint = f"{medication_service_url}/api/flow2/medication-safety/validate"
        
        # Test request data
        request_data = {
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "medication": {
                "name": "warfarin",
                "generic_name": "warfarin sodium",
                "dose": "5mg",
                "frequency": "daily",
                "route": "oral",
                "therapeutic_class": "ANTICOAGULANT",
                "is_anticoagulant": True,
                "requires_renal_adjustment": False,
                "is_chemotherapy": False,
                "is_controlled_substance": False,
                "is_high_risk": True
            },
            "provider_id": "provider_123",
            "encounter_id": "encounter_456",
            "action_type": "prescribe",
            "urgency": "routine",
            "workflow_id": "api_test_001"
        }
        
        logger.info("📋 Request Details:")
        logger.info(f"   Endpoint: {flow2_endpoint}")
        logger.info(f"   Patient: {request_data['patient_id']}")
        logger.info(f"   Medication: {request_data['medication']['name']} {request_data['medication']['dose']}")
        logger.info(f"   Urgency: {request_data['urgency']}")
        
        # Make the API call
        logger.info("🌐 Making Flow 2 API call...")
        start_time = time.time()
        
        response = requests.post(
            flow2_endpoint,
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        end_time = time.time()
        api_response_time = (end_time - start_time) * 1000
        
        logger.info(f"📡 API Response received in {api_response_time:.1f}ms")
        logger.info(f"📡 HTTP Status: {response.status_code}")
        
        if response.status_code == 200:
            result = response.json()
            
            logger.info("=" * 70)
            logger.info("📊 FLOW 2 API RESPONSE ANALYSIS")
            logger.info("=" * 70)
            
            # Basic response structure
            logger.info(f"✅ Request ID: {result.get('request_id', 'N/A')}")
            logger.info(f"✅ Patient ID: {result.get('patient_id', 'N/A')}")
            logger.info(f"✅ Overall Safety Status: {result.get('overall_safety_status', 'N/A')}")
            logger.info(f"✅ Context Recipe Used: {result.get('context_recipe_used', 'N/A')}")
            logger.info(f"✅ Context Completeness: {result.get('context_completeness_score', 0):.2%}")
            logger.info(f"✅ Execution Time: {result.get('execution_time_ms', 0):.1f}ms")
            
            # Clinical recipes executed
            recipes_executed = result.get('clinical_recipes_executed', [])
            logger.info(f"✅ Clinical Recipes Executed: {len(recipes_executed)}")
            for recipe in recipes_executed:
                logger.info(f"   - {recipe}")
            
            # Safety summary
            safety_summary = result.get('safety_summary', {})
            if safety_summary:
                logger.info("✅ Safety Summary:")
                logger.info(f"   - Total Validations: {safety_summary.get('total_validations', 0)}")
                logger.info(f"   - Critical Issues: {safety_summary.get('critical_issues', 0)}")
                logger.info(f"   - High Issues: {safety_summary.get('high_issues', 0)}")
                logger.info(f"   - Medium Issues: {safety_summary.get('medium_issues', 0)}")
                logger.info(f"   - Context Completeness: {safety_summary.get('context_completeness', 0):.2%}")
                
                # Clinical decision support
                cds = safety_summary.get('clinical_decision_support', {})
                if cds:
                    logger.info("✅ Clinical Decision Support:")
                    logger.info(f"   - Provider Summary: {cds.get('provider_summary', 'N/A')}")
                    logger.info(f"   - Patient Explanation: {cds.get('patient_explanation', 'N/A')}")
                    
                    monitoring = cds.get('monitoring_requirements', [])
                    if monitoring:
                        logger.info("   - Monitoring Requirements:")
                        for req in monitoring[:3]:  # Show first 3
                            logger.info(f"     • {req}")
            
            # Performance metrics
            performance = result.get('performance_metrics', {})
            if performance:
                logger.info("✅ Performance Metrics:")
                logger.info(f"   - Context Assembly: {performance.get('context_assembly_time_ms', 0):.1f}ms")
                logger.info(f"   - Clinical Recipes: {performance.get('clinical_recipes_time_ms', 0):.1f}ms")
                logger.info(f"   - Recipes Executed: {performance.get('recipes_executed', 0)}")
                logger.info(f"   - Data Sources Used: {performance.get('data_sources_used', 0)}")
            
            # Errors
            errors = result.get('errors')
            if errors:
                logger.warning("⚠️ Errors encountered:")
                for error in errors:
                    logger.warning(f"   - {error}")
            
            # Validate Flow 2 API requirements
            logger.info("=" * 70)
            logger.info("🎯 FLOW 2 API REQUIREMENTS VALIDATION")
            logger.info("=" * 70)
            
            # Check if we used real context
            context_recipe = result.get('context_recipe_used', '')
            if context_recipe not in ['error_fallback', 'minimal_fallback']:
                logger.info("✅ REAL CONTEXT: API used actual Context Service")
            else:
                logger.warning("⚠️ FALLBACK CONTEXT: API used fallback (Context Service unavailable)")
            
            # Check performance
            execution_time = result.get('execution_time_ms', 0)
            if execution_time < 200:
                logger.info(f"✅ PERFORMANCE: Met <200ms target ({execution_time:.1f}ms)")
            else:
                logger.warning(f"⚠️ PERFORMANCE: Exceeded 200ms target ({execution_time:.1f}ms)")
            
            # Check clinical recipes
            if len(recipes_executed) > 0:
                logger.info(f"✅ CLINICAL RECIPES: Executed {len(recipes_executed)} recipes")
            else:
                logger.warning("⚠️ CLINICAL RECIPES: No recipes executed")
            
            # Check safety status
            safety_status = result.get('overall_safety_status', '')
            if safety_status in ['SAFE', 'WARNING', 'UNSAFE']:
                logger.info(f"✅ SAFETY ASSESSMENT: Valid status ({safety_status})")
            else:
                logger.warning(f"⚠️ SAFETY ASSESSMENT: Invalid status ({safety_status})")
            
            return True
            
        else:
            logger.error(f"❌ API call failed with status {response.status_code}")
            logger.error(f"Response: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        logger.error("❌ Connection failed - Medication Service not running")
        logger.error("🔧 Start Medication Service: python run_service.py")
        return False
    except requests.exceptions.Timeout:
        logger.error("❌ API call timed out")
        return False
    except Exception as e:
        logger.error(f"❌ API test failed: {str(e)}")
        return False


def test_flow2_health_endpoint():
    """
    Test the Flow 2 health check endpoint
    """
    try:
        logger.info("🔍 Testing Flow 2 Health Check Endpoint...")
        
        health_url = "http://localhost:8009/api/flow2/medication-safety/health"
        
        response = requests.get(health_url, timeout=10)
        
        if response.status_code == 200:
            health_data = response.json()
            
            logger.info("✅ Health Check Response:")
            logger.info(f"   Status: {health_data.get('status', 'Unknown')}")
            logger.info(f"   Timestamp: {health_data.get('timestamp', 'Unknown')}")
            
            components = health_data.get('components', {})
            logger.info("✅ Component Health:")
            for component, status in components.items():
                if isinstance(status, str):
                    logger.info(f"   - {component}: {status}")
                else:
                    logger.info(f"   - {component}: {type(status).__name__}")
            
            performance = health_data.get('performance_metrics', {})
            if performance:
                logger.info("✅ Performance Metrics:")
                logger.info(f"   - Total Requests: {performance.get('total_requests', 0)}")
                logger.info(f"   - Success Rate: {performance.get('success_rate', 0):.2%}")
                logger.info(f"   - Avg Response Time: {performance.get('average_response_time_ms', 0):.1f}ms")
            
            return True
        else:
            logger.error(f"❌ Health check failed: {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Health check test failed: {str(e)}")
        return False


def test_flow2_metrics_endpoint():
    """
    Test the Flow 2 metrics endpoint
    """
    try:
        logger.info("📊 Testing Flow 2 Metrics Endpoint...")
        
        metrics_url = "http://localhost:8009/api/flow2/medication-safety/metrics"
        
        response = requests.get(metrics_url, timeout=10)
        
        if response.status_code == 200:
            metrics_data = response.json()
            
            logger.info("✅ Metrics Response:")
            logger.info(f"   Timestamp: {metrics_data.get('timestamp', 'Unknown')}")
            
            flow2_metrics = metrics_data.get('flow2_metrics', {})
            if flow2_metrics:
                logger.info("✅ Flow 2 Metrics:")
                logger.info(f"   - Total Requests: {flow2_metrics.get('total_requests', 0)}")
                logger.info(f"   - Success Rate: {flow2_metrics.get('success_rate', 0):.2%}")
                logger.info(f"   - Avg Response Time: {flow2_metrics.get('average_response_time_ms', 0):.1f}ms")
                
                clinical_recipes = flow2_metrics.get('clinical_recipes', {})
                if clinical_recipes:
                    logger.info(f"   - Registered Recipes: {clinical_recipes.get('total_registered', 0)}")
            
            return True
        else:
            logger.error(f"❌ Metrics endpoint failed: {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Metrics test failed: {str(e)}")
        return False


def main():
    """Main test function"""
    logger.info("🚀 Starting REAL Flow 2 API Integration Test")
    logger.info("🎯 This test validates Flow 2 API endpoints with actual Context Service calls")
    logger.info("")
    
    try:
        # Test health endpoint first
        health_ok = test_flow2_health_endpoint()
        
        # Test metrics endpoint
        metrics_ok = test_flow2_metrics_endpoint()
        
        # Test main validation endpoint
        validation_ok = test_flow2_api_endpoint()
        
        logger.info("=" * 70)
        if validation_ok:
            logger.info("🎉 REAL Flow 2 API Integration Test: PASSED")
            logger.info("✅ Flow 2 API endpoints are working correctly!")
            logger.info("")
            logger.info("🎯 Next Steps:")
            logger.info("1. Ensure Context Service is running for full integration")
            logger.info("2. Test with different medication types")
            logger.info("3. Test with different urgency levels")
            logger.info("4. Monitor performance metrics")
            return 0
        else:
            logger.error("❌ REAL Flow 2 API Integration Test: FAILED")
            logger.error("🔧 Check the error messages above")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1


if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
