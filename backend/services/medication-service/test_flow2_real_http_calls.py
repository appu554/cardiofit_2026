"""
Real Flow 2 HTTP Integration Test

This test makes ACTUAL HTTP calls to the running services to test Flow 2:
1. Calls Context Service GraphQL endpoint (port 8016)
2. Context Service should call Medication Service (port 8009)
3. Validates the complete Flow 2 integration with real HTTP traffic
"""

import requests
import json
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_context_service_graphql_available_recipes():
    """
    Test the Context Service GraphQL getAvailableRecipes query
    This should trigger the Context Service to call Medication Service
    """
    try:
        logger.info("🧪 Testing Context Service GraphQL getAvailableRecipes")
        logger.info("   This should trigger Context Service → Medication Service calls")
        
        # Context Service GraphQL endpoint
        graphql_url = "http://localhost:8016/graphql"
        
        # GraphQL query for available recipes
        query = """
        query GetAvailableRecipes {
            getAvailableRecipes {
                recipeId
                recipeName
                version
                clinicalScenario
                workflowCategory
                executionPattern
                slaMs
                governanceApproved
                effectiveDate
                expiryDate
            }
        }
        """
        
        payload = {
            "query": query
        }
        
        logger.info(f"📡 Making GraphQL request to: {graphql_url}")
        
        response = requests.post(
            graphql_url,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        logger.info(f"📡 GraphQL Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            if "errors" in data:
                logger.error("❌ GraphQL Errors:")
                for error in data["errors"]:
                    logger.error(f"   - {error.get('message', 'Unknown error')}")
                return False
            
            recipes = data.get("data", {}).get("getAvailableRecipes", [])
            
            if recipes:
                logger.info(f"✅ Retrieved {len(recipes)} recipes from Context Service")
                
                for recipe in recipes[:3]:
                    logger.info(f"   - {recipe.get('recipeId', 'Unknown')}: {recipe.get('recipeName', 'Unknown')}")
                
                if len(recipes) > 3:
                    logger.info(f"   ... and {len(recipes) - 3} more recipes")
                
                return True
            else:
                logger.warning("⚠️ No recipes returned from Context Service")
                return False
        else:
            logger.error(f"❌ GraphQL request failed: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        logger.error("❌ Cannot connect to Context Service on port 8016")
        logger.error("🔧 Make sure Context Service is running: cd backend/services/context-service && python run_service.py")
        return False
    except Exception as e:
        logger.error(f"❌ GraphQL test failed: {str(e)}")
        return False


def test_context_service_medication_context():
    """
    Test getting medication context from Context Service
    This should trigger the complete Flow 2 workflow
    """
    try:
        logger.info("🧪 Testing Context Service Medication Context")
        logger.info("   This should trigger the complete Flow 2 workflow")
        
        # Context Service GraphQL endpoint
        graphql_url = "http://localhost:8016/graphql"
        
        # GraphQL query for context by recipe
        query = """
        query GetContextByRecipe(
            $patientId: String!,
            $recipeId: String!,
            $providerId: String,
            $workflowId: String
        ) {
            getContextByRecipe(
                patientId: $patientId,
                recipeId: $recipeId,
                providerId: $providerId,
                workflowId: $workflowId
            ) {
                contextId
                patientId
                recipeUsed
                assembledData
                completenessScore
                dataFreshness
                sourceMetadata
                safetyFlags {
                    flagType
                    severity
                    message
                    dataPoint
                }
                governanceTags
                status
                assembledAt
                assemblyDurationMs
                connectionErrors {
                    dataPoint
                    source
                    error
                    timestamp
                }
            }
        }
        """
        
        variables = {
            "patientId": "905a60cb-8241-418f-b29b-5b020e851392",
            "recipeId": "medication_safety_base_context_v2",
            "providerId": "provider_123",
            "workflowId": "flow2_test_001"
        }
        
        payload = {
            "query": query,
            "variables": variables
        }
        
        logger.info(f"📡 Making GraphQL context request to: {graphql_url}")
        logger.info(f"   Patient: {variables['patientId']}")
        logger.info(f"   Recipe: {variables['recipeId']}")
        
        response = requests.post(
            graphql_url,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        logger.info(f"📡 GraphQL Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            if "errors" in data:
                logger.error("❌ GraphQL Errors:")
                for error in data["errors"]:
                    logger.error(f"   - {error.get('message', 'Unknown error')}")
                return False
            
            context = data.get("data", {}).get("getContextByRecipe")
            
            if context:
                logger.info("✅ Context retrieved successfully!")
                logger.info(f"   Context ID: {context.get('contextId', 'Unknown')}")
                logger.info(f"   Recipe Used: {context.get('recipeUsed', 'Unknown')}")
                logger.info(f"   Completeness: {context.get('completenessScore', 0):.2%}")
                logger.info(f"   Assembly Time: {context.get('assemblyDurationMs', 0):.1f}ms")
                logger.info(f"   Status: {context.get('status', 'Unknown')}")
                
                # Check assembled data
                assembled_data = context.get('assembledData', {})
                if assembled_data:
                    logger.info(f"   Assembled Data Keys: {list(assembled_data.keys())}")
                
                # Check safety flags
                safety_flags = context.get('safetyFlags', [])
                if safety_flags:
                    logger.info(f"   Safety Flags: {len(safety_flags)}")
                    for flag in safety_flags[:3]:
                        logger.info(f"     - {flag.get('flagType', 'Unknown')}: {flag.get('message', 'No message')}")
                
                # Check connection errors
                connection_errors = context.get('connectionErrors', [])
                if connection_errors:
                    logger.warning(f"   Connection Errors: {len(connection_errors)}")
                    for error in connection_errors:
                        logger.warning(f"     - {error.get('source', 'Unknown')}: {error.get('error', 'Unknown error')}")
                
                return True
            else:
                logger.warning("⚠️ No context returned from Context Service")
                return False
        else:
            logger.error(f"❌ GraphQL request failed: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        logger.error("❌ Cannot connect to Context Service on port 8016")
        return False
    except Exception as e:
        logger.error(f"❌ Context request test failed: {str(e)}")
        return False


def test_medication_service_endpoints():
    """
    Test that Medication Service Flow 2 endpoints are accessible
    """
    try:
        logger.info("🧪 Testing Medication Service Flow 2 Endpoints")
        
        # Test clinical recipes endpoint
        recipes_url = "http://localhost:8009/api/flow2/medication-safety/clinical-recipes"
        
        logger.info(f"📡 Testing: {recipes_url}")
        
        response = requests.get(recipes_url, timeout=10)
        
        if response.status_code == 200:
            data = response.json()
            recipes = data.get('recipes', [])
            logger.info(f"✅ Medication Service has {len(recipes)} clinical recipes available")
            return True
        else:
            logger.error(f"❌ Medication Service clinical recipes endpoint failed: {response.status_code}")
            return False
            
    except requests.exceptions.ConnectionError:
        logger.error("❌ Cannot connect to Medication Service on port 8009")
        logger.error("🔧 Make sure Medication Service is running: cd backend/services/medication-service && python run_service.py")
        return False
    except Exception as e:
        logger.error(f"❌ Medication Service test failed: {str(e)}")
        return False


def main():
    """Main test function"""
    logger.info("🚀 Starting REAL Flow 2 HTTP Integration Test")
    logger.info("🎯 This makes actual HTTP calls to running services")
    logger.info("=" * 80)
    
    try:
        # Test 1: Check Medication Service endpoints
        logger.info("📋 Step 1: Testing Medication Service endpoints...")
        medication_ok = test_medication_service_endpoints()
        
        if not medication_ok:
            logger.error("❌ Medication Service not accessible - cannot proceed")
            return 1
        
        # Test 2: Test Context Service GraphQL
        logger.info("📋 Step 2: Testing Context Service GraphQL...")
        graphql_ok = test_context_service_graphql_available_recipes()
        
        # Test 3: Test complete Flow 2 workflow
        logger.info("📋 Step 3: Testing complete Flow 2 workflow...")
        context_ok = test_context_service_medication_context()
        
        # Results
        logger.info("=" * 80)
        logger.info("📊 FLOW 2 HTTP INTEGRATION TEST RESULTS")
        logger.info("=" * 80)
        
        logger.info(f"✅ Medication Service Endpoints: {'PASS' if medication_ok else 'FAIL'}")
        logger.info(f"✅ Context Service GraphQL: {'PASS' if graphql_ok else 'FAIL'}")
        logger.info(f"✅ Complete Flow 2 Workflow: {'PASS' if context_ok else 'FAIL'}")
        
        if medication_ok and graphql_ok and context_ok:
            logger.info("🎉 REAL Flow 2 HTTP Integration: PASSED")
            logger.info("✅ All HTTP calls are working correctly!")
            logger.info("")
            logger.info("🔄 Flow 2 Architecture Validated:")
            logger.info("1. ✅ Medication Service exposes clinical recipes")
            logger.info("2. ✅ Context Service GraphQL is accessible")
            logger.info("3. ✅ Context Service can assemble medication context")
            logger.info("4. 🔄 Context Service should call Medication Service (check logs)")
            
            return 0
        else:
            logger.error("❌ REAL Flow 2 HTTP Integration: FAILED")
            logger.error("🔧 Check the failed components above")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1


if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
