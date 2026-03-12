"""
Integration Tests for GraphQL API
Tests Pillar 1: Federated GraphQL API (The "Unified Data Graph")
"""
import pytest
import asyncio
from datetime import datetime
from typing import Dict, Any
import json

from fastapi.testclient import TestClient
from strawberry.fastapi import GraphQLRouter
import strawberry

from app.main import app
from app.api.graphql.schema import schema
from app.api.graphql.resolvers import context_resolver


class TestGraphQLAPI:
    """
    Integration tests for the GraphQL API.
    Tests the unified data graph implementation.
    """
    
    @pytest.fixture
    def client(self):
        """Create test client"""
        return TestClient(app)
    
    @pytest.fixture
    def graphql_client(self, client):
        """Create GraphQL test client"""
        return client
    
    def execute_graphql_query(self, client, query: str, variables: Dict[str, Any] = None) -> Dict[str, Any]:
        """Execute GraphQL query and return response"""
        response = client.post(
            "/graphql",
            json={
                "query": query,
                "variables": variables or {}
            }
        )
        return response.json()
    
    @pytest.mark.asyncio
    async def test_get_context_by_recipe_query(self, graphql_client):
        """Test getContextByRecipe GraphQL query"""
        query = """
        query GetContextByRecipe($patientId: String!, $recipeId: String!, $providerId: String) {
            getContextByRecipe(
                patientId: $patientId,
                recipeId: $recipeId,
                providerId: $providerId,
                forceRefresh: false
            ) {
                contextId
                patientId
                recipeUsed
                assembledData
                completenessScore
                status
                safetyFlags {
                    flagType
                    severity
                    message
                    dataPoint
                    timestamp
                }
                assembledAt
                assemblyDurationMs
                cacheHit
                cacheKey
                ttlSeconds
            }
        }
        """
        
        variables = {
            "patientId": "test_patient_123",
            "recipeId": "medication_prescribing_v2",
            "providerId": "test_provider_456"
        }
        
        response = self.execute_graphql_query(graphql_client, query, variables)
        
        # Check response structure
        assert "data" in response
        assert "getContextByRecipe" in response["data"]
        
        context_data = response["data"]["getContextByRecipe"]
        
        # Verify context structure
        assert "contextId" in context_data
        assert "patientId" in context_data
        assert "recipeUsed" in context_data
        assert "assembledData" in context_data
        assert "completenessScore" in context_data
        assert "status" in context_data
        assert "safetyFlags" in context_data
        
        # Verify patient ID matches request
        assert context_data["patientId"] == "test_patient_123"
        
        print("✅ getContextByRecipe GraphQL query test passed")
    
    @pytest.mark.asyncio
    async def test_get_context_fields_query(self, graphql_client):
        """Test getContextFields GraphQL query"""
        query = """
        query GetContextFields($patientId: String!, $fields: [String!]!, $maxAgeHours: Int) {
            getContextFields(
                patientId: $patientId,
                fields: $fields,
                maxAgeHours: $maxAgeHours
            ) {
                data
                completeness
                metadata
                status
            }
        }
        """
        
        variables = {
            "patientId": "test_patient_123",
            "fields": ["patient_demographics", "current_medications", "allergies"],
            "maxAgeHours": 24
        }
        
        response = self.execute_graphql_query(graphql_client, query, variables)
        
        # Check response structure
        assert "data" in response
        assert "getContextFields" in response["data"]
        
        fields_data = response["data"]["getContextFields"]
        
        # Verify fields response structure
        assert "data" in fields_data
        assert "completeness" in fields_data
        assert "metadata" in fields_data
        assert "status" in fields_data
        
        print("✅ getContextFields GraphQL query test passed")
    
    @pytest.mark.asyncio
    async def test_validate_context_availability_query(self, graphql_client):
        """Test validateContextAvailability GraphQL query"""
        query = """
        query ValidateContextAvailability($patientId: String!, $recipeId: String!, $providerId: String) {
            validateContextAvailability(
                patientId: $patientId,
                recipeId: $recipeId,
                providerId: $providerId
            ) {
                available
                recipeId
                patientId
                estimatedCompleteness
                unavailableSources
                estimatedAssemblyTimeMs
                cacheAvailable
            }
        }
        """
        
        variables = {
            "patientId": "test_patient_123",
            "recipeId": "medication_prescribing_v2",
            "providerId": "test_provider_456"
        }
        
        response = self.execute_graphql_query(graphql_client, query, variables)
        
        # Check response structure
        assert "data" in response
        assert "validateContextAvailability" in response["data"]
        
        availability_data = response["data"]["validateContextAvailability"]
        
        # Verify availability response structure
        assert "available" in availability_data
        assert "recipeId" in availability_data
        assert "patientId" in availability_data
        assert "estimatedCompleteness" in availability_data
        assert "unavailableSources" in availability_data
        assert "estimatedAssemblyTimeMs" in availability_data
        assert "cacheAvailable" in availability_data
        
        # Verify patient and recipe IDs match request
        assert availability_data["patientId"] == "test_patient_123"
        assert availability_data["recipeId"] == "medication_prescribing_v2"
        
        print("✅ validateContextAvailability GraphQL query test passed")
    
    @pytest.mark.asyncio
    async def test_get_available_recipes_query(self, graphql_client):
        """Test getAvailableRecipes GraphQL query"""
        query = """
        query GetAvailableRecipes($clinicalScenario: String, $workflowCategory: String) {
            getAvailableRecipes(
                clinicalScenario: $clinicalScenario,
                workflowCategory: $workflowCategory
            ) {
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
        
        variables = {
            "clinicalScenario": "medication_ordering",
            "workflowCategory": "command_initiated"
        }
        
        response = self.execute_graphql_query(graphql_client, query, variables)
        
        # Check response structure
        assert "data" in response
        assert "getAvailableRecipes" in response["data"]
        
        recipes_data = response["data"]["getAvailableRecipes"]
        
        # Verify recipes response structure
        assert isinstance(recipes_data, list)
        
        if len(recipes_data) > 0:
            recipe = recipes_data[0]
            assert "recipeId" in recipe
            assert "recipeName" in recipe
            assert "version" in recipe
            assert "clinicalScenario" in recipe
            assert "workflowCategory" in recipe
            assert "executionPattern" in recipe
            assert "slaMs" in recipe
            assert "governanceApproved" in recipe
        
        print("✅ getAvailableRecipes GraphQL query test passed")
    
    @pytest.mark.asyncio
    async def test_get_recipe_info_query(self, graphql_client):
        """Test getRecipeInfo GraphQL query"""
        query = """
        query GetRecipeInfo($recipeId: String!, $version: String) {
            getRecipeInfo(
                recipeId: $recipeId,
                version: $version
            ) {
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
        
        variables = {
            "recipeId": "medication_prescribing_v2",
            "version": "latest"
        }
        
        response = self.execute_graphql_query(graphql_client, query, variables)
        
        # Check response structure
        assert "data" in response
        assert "getRecipeInfo" in response["data"]
        
        recipe_info = response["data"]["getRecipeInfo"]
        
        if recipe_info:  # Recipe might not exist in test environment
            # Verify recipe info structure
            assert "recipeId" in recipe_info
            assert "recipeName" in recipe_info
            assert "version" in recipe_info
            assert "clinicalScenario" in recipe_info
            assert "workflowCategory" in recipe_info
            assert "executionPattern" in recipe_info
            assert "slaMs" in recipe_info
            assert "governanceApproved" in recipe_info
            
            # Verify recipe ID matches request
            assert recipe_info["recipeId"] == "medication_prescribing_v2"
        
        print("✅ getRecipeInfo GraphQL query test passed")
    
    @pytest.mark.asyncio
    async def test_get_cache_stats_query(self, graphql_client):
        """Test getCacheStats GraphQL query"""
        query = """
        query GetCacheStats {
            getCacheStats {
                totalEntries
                hitRatio
                l1Entries
                l2Entries
                lastUpdated
                performanceMetrics
            }
        }
        """
        
        response = self.execute_graphql_query(graphql_client, query)
        
        # Check response structure
        assert "data" in response
        assert "getCacheStats" in response["data"]
        
        cache_stats = response["data"]["getCacheStats"]
        
        # Verify cache stats structure
        assert "totalEntries" in cache_stats
        assert "hitRatio" in cache_stats
        assert "l1Entries" in cache_stats
        assert "l2Entries" in cache_stats
        assert "lastUpdated" in cache_stats
        assert "performanceMetrics" in cache_stats
        
        # Verify data types
        assert isinstance(cache_stats["totalEntries"], int)
        assert isinstance(cache_stats["hitRatio"], (int, float))
        assert isinstance(cache_stats["l1Entries"], int)
        assert isinstance(cache_stats["l2Entries"], int)
        
        print("✅ getCacheStats GraphQL query test passed")
    
    @pytest.mark.asyncio
    async def test_get_service_health_query(self, graphql_client):
        """Test getServiceHealth GraphQL query"""
        query = """
        query GetServiceHealth {
            getServiceHealth {
                serviceType
                endpoint
                healthy
                responseTimeMs
                lastCheck
                errorMessage
            }
        }
        """
        
        response = self.execute_graphql_query(graphql_client, query)
        
        # Check response structure
        assert "data" in response
        assert "getServiceHealth" in response["data"]
        
        service_health = response["data"]["getServiceHealth"]
        
        # Verify service health response structure
        assert isinstance(service_health, list)
        
        if len(service_health) > 0:
            health_item = service_health[0]
            assert "serviceType" in health_item
            assert "endpoint" in health_item
            assert "healthy" in health_item
            assert "responseTimeMs" in health_item
            assert "lastCheck" in health_item
            
            # Verify data types
            assert isinstance(health_item["healthy"], bool)
            assert isinstance(health_item["responseTimeMs"], int)
        
        print("✅ getServiceHealth GraphQL query test passed")
    
    @pytest.mark.asyncio
    async def test_invalidate_patient_context_mutation(self, graphql_client):
        """Test invalidatePatientContext GraphQL mutation"""
        mutation = """
        mutation InvalidatePatientContext($patientId: String!, $recipeIds: [String!]) {
            invalidatePatientContext(
                patientId: $patientId,
                recipeIds: $recipeIds
            )
        }
        """
        
        variables = {
            "patientId": "test_patient_123",
            "recipeIds": ["medication_prescribing_v2", "routine_medication_refill_v1"]
        }
        
        response = self.execute_graphql_query(graphql_client, mutation, variables)
        
        # Check response structure
        assert "data" in response
        assert "invalidatePatientContext" in response["data"]
        
        # Verify mutation result
        result = response["data"]["invalidatePatientContext"]
        assert isinstance(result, bool)
        
        print("✅ invalidatePatientContext GraphQL mutation test passed")
    
    @pytest.mark.asyncio
    async def test_invalidate_context_cache_mutation(self, graphql_client):
        """Test invalidateContextCache GraphQL mutation"""
        mutation = """
        mutation InvalidateContextCache($cacheKey: String!) {
            invalidateContextCache(cacheKey: $cacheKey)
        }
        """
        
        variables = {
            "cacheKey": "context:test_patient_123:medication_prescribing_v2"
        }
        
        response = self.execute_graphql_query(graphql_client, mutation, variables)
        
        # Check response structure
        assert "data" in response
        assert "invalidateContextCache" in response["data"]
        
        # Verify mutation result
        result = response["data"]["invalidateContextCache"]
        assert isinstance(result, bool)
        
        print("✅ invalidateContextCache GraphQL mutation test passed")
    
    @pytest.mark.asyncio
    async def test_warm_context_cache_mutation(self, graphql_client):
        """Test warmContextCache GraphQL mutation"""
        mutation = """
        mutation WarmContextCache($patientIds: [String!]!, $recipeIds: [String!]!) {
            warmContextCache(
                patientIds: $patientIds,
                recipeIds: $recipeIds
            )
        }
        """
        
        variables = {
            "patientIds": ["test_patient_123", "test_patient_456"],
            "recipeIds": ["medication_prescribing_v2", "routine_medication_refill_v1"]
        }
        
        response = self.execute_graphql_query(graphql_client, mutation, variables)
        
        # Check response structure
        assert "data" in response
        assert "warmContextCache" in response["data"]
        
        # Verify mutation result
        result = response["data"]["warmContextCache"]
        assert isinstance(result, bool)
        
        print("✅ warmContextCache GraphQL mutation test passed")
    
    @pytest.mark.asyncio
    async def test_graphql_error_handling(self, graphql_client):
        """Test GraphQL error handling"""
        # Test query with invalid patient ID
        query = """
        query GetContextByRecipe($patientId: String!, $recipeId: String!) {
            getContextByRecipe(
                patientId: $patientId,
                recipeId: $recipeId
            ) {
                contextId
                patientId
                status
            }
        }
        """
        
        variables = {
            "patientId": "",  # Invalid empty patient ID
            "recipeId": "nonexistent_recipe"
        }
        
        response = self.execute_graphql_query(graphql_client, query, variables)
        
        # Should handle error gracefully
        assert "data" in response
        
        # Check if error context is returned
        context_data = response["data"]["getContextByRecipe"]
        if context_data:
            assert context_data["status"] in ["FAILED", "UNAVAILABLE"]
        
        print("✅ GraphQL error handling test passed")
    
    @pytest.mark.asyncio
    async def test_graphql_schema_introspection(self, graphql_client):
        """Test GraphQL schema introspection"""
        introspection_query = """
        query IntrospectionQuery {
            __schema {
                types {
                    name
                    kind
                }
                queryType {
                    name
                    fields {
                        name
                        type {
                            name
                        }
                    }
                }
                mutationType {
                    name
                    fields {
                        name
                        type {
                            name
                        }
                    }
                }
            }
        }
        """
        
        response = self.execute_graphql_query(graphql_client, introspection_query)
        
        # Check introspection response
        assert "data" in response
        assert "__schema" in response["data"]
        
        schema_data = response["data"]["__schema"]
        
        # Verify schema structure
        assert "types" in schema_data
        assert "queryType" in schema_data
        assert "mutationType" in schema_data
        
        # Verify query type has expected fields
        query_type = schema_data["queryType"]
        assert query_type["name"] == "Query"
        
        query_field_names = [field["name"] for field in query_type["fields"]]
        expected_query_fields = [
            "getContextByRecipe",
            "getContextFields",
            "validateContextAvailability",
            "getAvailableRecipes",
            "getRecipeInfo",
            "getCacheStats",
            "getServiceHealth"
        ]
        
        for expected_field in expected_query_fields:
            assert expected_field in query_field_names
        
        # Verify mutation type has expected fields
        mutation_type = schema_data["mutationType"]
        assert mutation_type["name"] == "Mutation"
        
        mutation_field_names = [field["name"] for field in mutation_type["fields"]]
        expected_mutation_fields = [
            "invalidatePatientContext",
            "invalidateContextCache",
            "warmContextCache"
        ]
        
        for expected_field in expected_mutation_fields:
            assert expected_field in mutation_field_names
        
        print("✅ GraphQL schema introspection test passed")
    
    @pytest.mark.asyncio
    async def test_end_to_end_graphql_workflow(self, graphql_client):
        """Test complete end-to-end GraphQL workflow"""
        print("🔄 Starting end-to-end GraphQL workflow test")
        
        # 1. Get available recipes
        recipes_query = """
        query GetAvailableRecipes {
            getAvailableRecipes {
                recipeId
                recipeName
                governanceApproved
            }
        }
        """
        
        recipes_response = self.execute_graphql_query(graphql_client, recipes_query)
        assert "data" in recipes_response
        print("   ✅ Available recipes retrieved")
        
        # 2. Validate context availability
        availability_query = """
        query ValidateContextAvailability($patientId: String!, $recipeId: String!) {
            validateContextAvailability(
                patientId: $patientId,
                recipeId: $recipeId
            ) {
                available
                estimatedCompleteness
                cacheAvailable
            }
        }
        """
        
        availability_variables = {
            "patientId": "test_patient_123",
            "recipeId": "medication_prescribing_v2"
        }
        
        availability_response = self.execute_graphql_query(
            graphql_client, availability_query, availability_variables
        )
        assert "data" in availability_response
        print("   ✅ Context availability validated")
        
        # 3. Get context by recipe
        context_query = """
        query GetContextByRecipe($patientId: String!, $recipeId: String!) {
            getContextByRecipe(
                patientId: $patientId,
                recipeId: $recipeId
            ) {
                contextId
                patientId
                completenessScore
                status
                cacheHit
            }
        }
        """
        
        context_variables = {
            "patientId": "test_patient_123",
            "recipeId": "medication_prescribing_v2"
        }
        
        context_response = self.execute_graphql_query(
            graphql_client, context_query, context_variables
        )
        assert "data" in context_response
        print("   ✅ Context retrieved by recipe")
        
        # 4. Get cache statistics
        cache_stats_query = """
        query GetCacheStats {
            getCacheStats {
                totalEntries
                hitRatio
            }
        }
        """
        
        cache_stats_response = self.execute_graphql_query(graphql_client, cache_stats_query)
        assert "data" in cache_stats_response
        print("   ✅ Cache statistics retrieved")
        
        # 5. Invalidate patient context
        invalidate_mutation = """
        mutation InvalidatePatientContext($patientId: String!) {
            invalidatePatientContext(patientId: $patientId)
        }
        """
        
        invalidate_variables = {
            "patientId": "test_patient_123"
        }
        
        invalidate_response = self.execute_graphql_query(
            graphql_client, invalidate_mutation, invalidate_variables
        )
        assert "data" in invalidate_response
        print("   ✅ Patient context invalidated")
        
        print("🎉 End-to-end GraphQL workflow test completed successfully!")


if __name__ == "__main__":
    # Run tests
    pytest.main([__file__, "-v", "-s"])
