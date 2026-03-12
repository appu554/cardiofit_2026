"""
Test Context Service GraphQL Schema
Check what fields are actually available
"""

import requests
import json
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_context_service_schema():
    """Test what GraphQL fields are available in Context Service"""
    logger.info("🔍 TESTING CONTEXT SERVICE GRAPHQL SCHEMA")
    logger.info("=" * 60)
    
    context_url = "http://localhost:8016/graphql"
    
    # GraphQL introspection query to see available fields
    introspection_query = """
    query IntrospectionQuery {
        __schema {
            queryType {
                fields {
                    name
                    description
                    args {
                        name
                        type {
                            name
                        }
                    }
                }
            }
        }
    }
    """
    
    try:
        logger.info("📡 Sending GraphQL introspection query...")
        response = requests.post(
            context_url,
            json={"query": introspection_query},
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        logger.info(f"✅ Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            if "errors" in data:
                logger.error(f"❌ GraphQL Errors: {data['errors']}")
                return False
            
            schema_data = data.get("data", {}).get("__schema", {})
            query_type = schema_data.get("queryType", {})
            fields = query_type.get("fields", [])
            
            logger.info(f"📊 Available Query Fields: {len(fields)}")
            logger.info("")
            
            for field in fields:
                field_name = field.get("name", "Unknown")
                field_desc = field.get("description", "No description")
                args = field.get("args", [])
                
                logger.info(f"🔹 {field_name}")
                if field_desc and field_desc != "No description":
                    logger.info(f"   📝 {field_desc}")
                
                if args:
                    logger.info(f"   📋 Arguments:")
                    for arg in args:
                        arg_name = arg.get("name", "Unknown")
                        arg_type = arg.get("type", {}).get("name", "Unknown")
                        logger.info(f"      - {arg_name}: {arg_type}")
                logger.info("")
            
            return True
        else:
            logger.error(f"❌ HTTP Error: {response.status_code}")
            logger.error(f"   Response: {response.text}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Schema introspection failed: {e}")
        return False

def test_simple_context_query():
    """Test a simple context query to see what works"""
    logger.info("🔍 TESTING SIMPLE CONTEXT QUERIES")
    logger.info("=" * 60)
    
    context_url = "http://localhost:8016/graphql"
    patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
    
    # Try different possible field names based on our earlier working tests
    possible_queries = [
        # Query 1: Try assembleContext (what we used before)
        {
            "name": "assembleContext",
            "query": """
            query GetContext($patientId: String!, $recipeName: String!) {
                assembleContext(patientId: $patientId, recipeName: $recipeName) {
                    contextId
                    patientId
                    status
                }
            }
            """,
            "variables": {
                "patientId": patient_id,
                "recipeName": "medication_safety_base_context_v2"
            }
        },
        # Query 2: Try getContext
        {
            "name": "getContext", 
            "query": """
            query GetContext($patientId: String!, $recipeName: String!) {
                getContext(patientId: $patientId, recipeName: $recipeName) {
                    contextId
                    patientId
                    status
                }
            }
            """,
            "variables": {
                "patientId": patient_id,
                "recipeName": "medication_safety_base_context_v2"
            }
        },
        # Query 3: Try contextAssembly
        {
            "name": "contextAssembly",
            "query": """
            query GetContext($patientId: String!, $recipeName: String!) {
                contextAssembly(patientId: $patientId, recipeName: $recipeName) {
                    contextId
                    patientId
                    status
                }
            }
            """,
            "variables": {
                "patientId": patient_id,
                "recipeName": "medication_safety_base_context_v2"
            }
        },
        # Query 4: Try executeRecipe (from our working tests)
        {
            "name": "executeRecipe",
            "query": """
            query ExecuteRecipe($patientId: String!, $recipeId: String!) {
                executeRecipe(patientId: $patientId, recipeId: $recipeId) {
                    contextId
                    patientId
                    status
                }
            }
            """,
            "variables": {
                "patientId": patient_id,
                "recipeId": "medication_safety_base_context_v2"
            }
        }
    ]
    
    for query_test in possible_queries:
        logger.info(f"🧪 Testing query: {query_test['name']}")
        
        try:
            response = requests.post(
                context_url,
                json={
                    "query": query_test["query"],
                    "variables": query_test["variables"]
                },
                headers={"Content-Type": "application/json"},
                timeout=10
            )
            
            if response.status_code == 200:
                data = response.json()
                
                if "errors" in data:
                    logger.warning(f"   ❌ GraphQL Error: {data['errors'][0].get('message', 'Unknown error')}")
                else:
                    logger.info(f"   ✅ SUCCESS! Query {query_test['name']} works")
                    result_data = data.get("data", {})
                    logger.info(f"   📊 Response: {json.dumps(result_data, indent=2)[:200]}...")
                    return query_test["name"], query_test["query"]
            else:
                logger.warning(f"   ❌ HTTP Error: {response.status_code}")
        
        except Exception as e:
            logger.warning(f"   ❌ Query failed: {e}")
        
        logger.info("")
    
    return None, None

def main():
    """Main test function"""
    logger.info("🚀 Context Service Schema Analysis")
    logger.info("🎯 Finding the correct GraphQL field names")
    logger.info("=" * 80)
    
    try:
        # Test 1: Get schema information
        schema_success = test_context_service_schema()
        logger.info("")
        
        # Test 2: Try different query patterns
        working_field, working_query = test_simple_context_query()
        
        if working_field:
            logger.info("=" * 80)
            logger.info("🎉 SOLUTION FOUND!")
            logger.info(f"✅ Working GraphQL field: {working_field}")
            logger.info("✅ Use this field name in future queries")
        else:
            logger.info("=" * 80)
            logger.info("❌ NO WORKING QUERY FOUND")
            logger.info("❌ Context Service may have schema issues")
        
        return 0 if working_field else 1
        
    except Exception as e:
        logger.error(f"❌ Analysis failed: {str(e)}")
        return 1

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
