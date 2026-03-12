"""
GraphQL Schema Introspection Test

This test queries the GraphQL schema to understand the actual structure
of the observations query and types.
"""

import requests
import json
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def introspect_schema():
    """
    Introspect the GraphQL schema to understand the structure
    """
    try:
        logger.info("🔍 Introspecting GraphQL Schema")
        
        # Observation Service Federation GraphQL endpoint
        graphql_url = "http://localhost:8007/api/federation"
        
        # GraphQL introspection query
        query = """
        query IntrospectionQuery {
            __schema {
                queryType {
                    fields {
                        name
                        args {
                            name
                            type {
                                name
                                kind
                                ofType {
                                    name
                                    kind
                                }
                            }
                        }
                        type {
                            name
                            kind
                            ofType {
                                name
                                kind
                                fields {
                                    name
                                    type {
                                        name
                                        kind
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
        """
        
        payload = {
            "query": query
        }
        
        logger.info(f"📡 Making introspection request to: {graphql_url}")
        
        response = requests.post(
            graphql_url,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        logger.info(f"📡 Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            if "errors" in data:
                logger.error("❌ GraphQL Errors:")
                for error in data["errors"]:
                    logger.error(f"   - {error}")
                return None
            
            schema = data.get("data", {}).get("__schema", {})
            query_type = schema.get("queryType", {})
            fields = query_type.get("fields", [])
            
            logger.info("📊 AVAILABLE QUERY FIELDS:")
            
            for field in fields:
                field_name = field.get("name", "Unknown")
                field_type = field.get("type", {})
                args = field.get("args", [])
                
                logger.info(f"   🔹 {field_name}")
                
                # Show arguments
                if args:
                    logger.info(f"      Arguments:")
                    for arg in args:
                        arg_name = arg.get("name", "Unknown")
                        arg_type = arg.get("type", {})
                        type_name = get_type_name(arg_type)
                        logger.info(f"        - {arg_name}: {type_name}")
                
                # Show return type
                return_type = get_type_name(field_type)
                logger.info(f"      Returns: {return_type}")
                
                # If it's observations field, show more details
                if field_name == "observations":
                    logger.info("      🎯 OBSERVATIONS FIELD DETAILS:")
                    logger.info(f"         Type: {return_type}")
                    
                    # Check if it's a list or connection
                    if field_type.get("kind") == "LIST":
                        of_type = field_type.get("ofType", {})
                        logger.info(f"         List of: {get_type_name(of_type)}")
                        
                        # Show fields of the observation type
                        if of_type.get("name") == "Observation":
                            obs_fields = of_type.get("fields", [])
                            if obs_fields:
                                logger.info("         Observation fields:")
                                for obs_field in obs_fields[:10]:  # Show first 10 fields
                                    obs_field_name = obs_field.get("name", "Unknown")
                                    obs_field_type = obs_field.get("type", {})
                                    obs_type_name = get_type_name(obs_field_type)
                                    logger.info(f"           - {obs_field_name}: {obs_type_name}")
                
                logger.info("")
            
            return fields
        else:
            logger.error(f"❌ Introspection request failed: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except Exception as e:
        logger.error(f"❌ Error during introspection: {str(e)}")
        return None

def get_type_name(type_info):
    """
    Extract type name from GraphQL type info
    """
    if not type_info:
        return "Unknown"
    
    kind = type_info.get("kind", "")
    name = type_info.get("name")
    
    if name:
        return name
    elif kind == "LIST":
        of_type = type_info.get("ofType", {})
        return f"[{get_type_name(of_type)}]"
    elif kind == "NON_NULL":
        of_type = type_info.get("ofType", {})
        return f"{get_type_name(of_type)}!"
    else:
        return f"{kind}"

def test_simple_observations_query():
    """
    Test a simple observations query to see what works
    """
    try:
        logger.info("🧪 Testing simple observations query")
        
        # Observation Service Federation GraphQL endpoint
        graphql_url = "http://localhost:8007/api/federation"
        
        # Simple query without patient filter first
        query = """
        query TestObservations {
            observations(count: 5) {
                id
                status
            }
        }
        """
        
        payload = {
            "query": query
        }
        
        logger.info(f"📡 Making test query to: {graphql_url}")
        
        response = requests.post(
            graphql_url,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        logger.info(f"📡 Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            if "errors" in data:
                logger.error("❌ GraphQL Errors:")
                for error in data["errors"]:
                    logger.error(f"   - {error}")
                return False
            
            observations = data.get("data", {}).get("observations", [])
            logger.info(f"✅ Simple query returned {len(observations)} observations")
            
            if observations:
                logger.info("📋 Sample observation:")
                logger.info(f"   {json.dumps(observations[0], indent=2)}")
            
            return True
        else:
            logger.error(f"❌ Test query failed: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Error during test query: {str(e)}")
        return False

def main():
    """Main test function"""
    logger.info("🚀 GraphQL Schema Introspection Test")
    logger.info("🎯 Understanding the actual GraphQL schema structure")
    logger.info("=" * 70)
    
    try:
        # Step 1: Introspect schema
        logger.info("📋 Step 1: Introspecting schema...")
        fields = introspect_schema()
        
        if not fields:
            logger.error("❌ Cannot introspect schema")
            return 1
        
        # Step 2: Test simple query
        logger.info("📋 Step 2: Testing simple observations query...")
        simple_ok = test_simple_observations_query()
        
        logger.info("=" * 70)
        if simple_ok:
            logger.info("✅ GraphQL Schema Introspection: SUCCESS")
            logger.info("🎯 Now we understand the schema structure!")
            return 0
        else:
            logger.error("❌ GraphQL Schema Introspection: FAILED")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
