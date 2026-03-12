"""
Flow 2 Detailed Response Analysis

This test captures and displays the complete GraphQL response from the Context Service
to understand exactly what's happening with the data quality and errors.
"""

import requests
import json
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_detailed_context_response():
    """
    Get detailed Context Service response and analyze all fields
    """
    try:
        logger.info("🔍 DETAILED Flow 2 Context Response Analysis")
        logger.info("=" * 80)
        
        # Context Service GraphQL endpoint
        graphql_url = "http://localhost:8016/graphql"
        
        # GraphQL query for complete context details
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
                    timestamp
                    details
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
                cacheHit
                cacheKey
                ttlSeconds
            }
        }
        """
        
        variables = {
            "patientId": "905a60cb-8241-418f-b29b-5b020e851392",
            "recipeId": "medication_safety_base_context_v2",
            "providerId": "provider_123",
            "workflowId": "detailed_analysis_001"
        }
        
        payload = {
            "query": query,
            "variables": variables
        }
        
        logger.info(f"📡 Making detailed GraphQL request...")
        logger.info(f"   Patient: {variables['patientId']}")
        logger.info(f"   Recipe: {variables['recipeId']}")
        
        response = requests.post(
            graphql_url,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        logger.info(f"📡 Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            # Check for GraphQL errors first
            if "errors" in data:
                logger.error("❌ GraphQL Errors:")
                for error in data["errors"]:
                    logger.error(f"   - {error.get('message', 'Unknown error')}")
                    if "locations" in error:
                        logger.error(f"     Location: {error['locations']}")
                    if "path" in error:
                        logger.error(f"     Path: {error['path']}")
                return False
            
            context = data.get("data", {}).get("getContextByRecipe")
            
            if context:
                logger.info("=" * 80)
                logger.info("📊 COMPLETE CONTEXT RESPONSE ANALYSIS")
                logger.info("=" * 80)
                
                # Basic Information
                logger.info("🔍 BASIC INFORMATION:")
                logger.info(f"   Context ID: {context.get('contextId', 'Unknown')}")
                logger.info(f"   Patient ID: {context.get('patientId', 'Unknown')}")
                logger.info(f"   Recipe Used: {context.get('recipeUsed', 'Unknown')}")
                logger.info(f"   Status: {context.get('status', 'Unknown')}")
                logger.info(f"   Assembled At: {context.get('assembledAt', 'Unknown')}")
                logger.info(f"   Assembly Duration: {context.get('assemblyDurationMs', 0):.1f}ms")
                
                # Completeness Analysis
                logger.info("")
                logger.info("📊 COMPLETENESS ANALYSIS:")
                completeness = context.get('completenessScore', 0)
                logger.info(f"   Completeness Score: {completeness:.2%}")
                
                if completeness == 0:
                    logger.error("   ❌ ZERO COMPLETENESS - Critical data assembly failure!")
                elif completeness < 0.5:
                    logger.warning(f"   ⚠️ LOW COMPLETENESS - Insufficient data quality")
                elif completeness < 0.9:
                    logger.info(f"   ⚠️ MODERATE COMPLETENESS - Some data missing")
                else:
                    logger.info(f"   ✅ HIGH COMPLETENESS - Good data quality")
                
                # Assembled Data Analysis
                logger.info("")
                logger.info("📋 ASSEMBLED DATA ANALYSIS:")
                assembled_data = context.get('assembledData', {})
                
                if assembled_data:
                    logger.info(f"   Data Points Retrieved: {len(assembled_data)}")
                    for key, value in assembled_data.items():
                        if isinstance(value, dict):
                            logger.info(f"   - {key}: {len(value)} fields")
                            # Show first few fields for debugging
                            if value:
                                sample_fields = list(value.keys())[:3]
                                logger.info(f"     Sample fields: {sample_fields}")
                        elif isinstance(value, list):
                            logger.info(f"   - {key}: {len(value)} items")
                        else:
                            logger.info(f"   - {key}: {type(value).__name__}")
                else:
                    logger.error("   ❌ NO ASSEMBLED DATA - Complete assembly failure!")
                
                # Safety Flags Analysis
                logger.info("")
                logger.info("🚨 SAFETY FLAGS ANALYSIS:")
                safety_flags = context.get('safetyFlags', [])
                
                if safety_flags:
                    logger.info(f"   Total Safety Flags: {len(safety_flags)}")
                    
                    # Group by severity
                    critical_flags = []
                    warning_flags = []
                    info_flags = []
                    
                    for flag in safety_flags:
                        severity = flag.get('severity', 'UNKNOWN')
                        if severity == 'CRITICAL':
                            critical_flags.append(flag)
                        elif severity == 'WARNING':
                            warning_flags.append(flag)
                        else:
                            info_flags.append(flag)
                    
                    if critical_flags:
                        logger.error(f"   ❌ CRITICAL FLAGS ({len(critical_flags)}):")
                        for flag in critical_flags:
                            logger.error(f"     - {flag.get('flagType', 'Unknown')}: {flag.get('message', 'No message')}")
                    
                    if warning_flags:
                        logger.warning(f"   ⚠️ WARNING FLAGS ({len(warning_flags)}):")
                        for flag in warning_flags:
                            logger.warning(f"     - {flag.get('flagType', 'Unknown')}: {flag.get('message', 'No message')}")
                    
                    if info_flags:
                        logger.info(f"   ℹ️ INFO FLAGS ({len(info_flags)}):")
                        for flag in info_flags:
                            logger.info(f"     - {flag.get('flagType', 'Unknown')}: {flag.get('message', 'No message')}")
                else:
                    logger.info("   ✅ No safety flags")
                
                # Connection Errors Analysis
                logger.info("")
                logger.info("🔌 CONNECTION ERRORS ANALYSIS:")
                connection_errors = context.get('connectionErrors', [])
                
                if connection_errors:
                    logger.error(f"   ❌ CONNECTION ERRORS ({len(connection_errors)}):")
                    for error in connection_errors:
                        logger.error(f"     - Source: {error.get('source', 'Unknown')}")
                        logger.error(f"       Data Point: {error.get('dataPoint', 'Unknown')}")
                        logger.error(f"       Error: {error.get('error', 'Unknown error')}")
                        logger.error(f"       Timestamp: {error.get('timestamp', 'Unknown')}")
                else:
                    logger.info("   ✅ No connection errors")
                
                # Source Metadata Analysis
                logger.info("")
                logger.info("📡 SOURCE METADATA ANALYSIS:")
                source_metadata = context.get('sourceMetadata', [])
                
                if source_metadata:
                    logger.info(f"   Data Sources Used: {len(source_metadata)}")
                    for metadata in source_metadata:
                        if isinstance(metadata, dict):
                            source_type = metadata.get('sourceType', 'Unknown')
                            endpoint = metadata.get('sourceEndpoint', 'Unknown')
                            logger.info(f"   - {source_type}: {endpoint}")
                else:
                    logger.warning("   ⚠️ No source metadata available")
                
                # Cache Information
                logger.info("")
                logger.info("💾 CACHE INFORMATION:")
                cache_hit = context.get('cacheHit', False)
                cache_key = context.get('cacheKey', 'Unknown')
                ttl_seconds = context.get('ttlSeconds', 0)
                
                logger.info(f"   Cache Hit: {cache_hit}")
                logger.info(f"   Cache Key: {cache_key}")
                logger.info(f"   TTL: {ttl_seconds} seconds")
                
                # Data Freshness
                logger.info("")
                logger.info("🕒 DATA FRESHNESS:")
                data_freshness = context.get('dataFreshness', {})
                if data_freshness:
                    for data_point, freshness in data_freshness.items():
                        logger.info(f"   - {data_point}: {freshness}")
                else:
                    logger.warning("   ⚠️ No data freshness information")
                
                # Overall Assessment
                logger.info("")
                logger.info("=" * 80)
                logger.info("🎯 OVERALL FLOW 2 ASSESSMENT")
                logger.info("=" * 80)
                
                status = context.get('status', 'UNKNOWN')
                
                if status == 'SUCCESS' and completeness >= 0.9:
                    logger.info("✅ FLOW 2 STATUS: EXCELLENT")
                    logger.info("   Context assembly successful with high data quality")
                elif status == 'WARNING' and completeness >= 0.7:
                    logger.info("⚠️ FLOW 2 STATUS: GOOD")
                    logger.info("   Context assembly successful with acceptable data quality")
                elif status == 'PARTIAL' and completeness >= 0.5:
                    logger.warning("⚠️ FLOW 2 STATUS: PARTIAL")
                    logger.warning("   Context assembly partially successful, some data missing")
                else:
                    logger.error("❌ FLOW 2 STATUS: FAILED")
                    logger.error("   Context assembly failed or data quality too low for clinical use")
                
                return True
            else:
                logger.error("❌ No context data returned")
                return False
        else:
            logger.error(f"❌ HTTP request failed: {response.status_code}")
            logger.error(f"Response body: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        logger.error("❌ Cannot connect to Context Service on port 8016")
        return False
    except Exception as e:
        logger.error(f"❌ Detailed analysis failed: {str(e)}")
        return False

def main():
    """Main test function"""
    logger.info("🚀 Flow 2 Detailed Response Analysis")
    logger.info("🎯 This provides complete visibility into Context Service responses")
    logger.info("")
    
    try:
        success = test_detailed_context_response()
        
        if success:
            logger.info("✅ Detailed analysis completed successfully")
            return 0
        else:
            logger.error("❌ Detailed analysis failed")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
