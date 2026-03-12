#!/usr/bin/env python3
"""
Test script to verify Clinical Context Service connections to other services.
This script tests the actual connections to real services.
"""
import asyncio
import aiohttp
import json
import sys
from datetime import datetime


class ServiceConnectionTester:
    """Test connections to all services that the Context Service depends on"""
    
    def __init__(self):
        self.services = {
            "Patient Service": "http://localhost:8003/health",
            "Medication Service": "http://localhost:8009/health", 
            "Lab Service": "http://localhost:8000/health",
            "Condition Service": "http://localhost:8010/health",
            "Encounter Service": "http://localhost:8020/health",
            "Observation Service": "http://localhost:8007/health",
            "Auth Service": "http://localhost:8001/health",
            "API Gateway": "http://localhost:8005/health",
            "Context Service": "http://localhost:8016/health"
        }
        
        self.graphql_endpoints = {
            "Context Service GraphQL": "http://localhost:8016/graphql"
        }
        
        self.results = {}
    
    async def test_service_health(self, service_name: str, url: str) -> dict:
        """Test health endpoint of a service"""
        try:
            timeout = aiohttp.ClientTimeout(total=10)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(url) as response:
                    if response.status == 200:
                        data = await response.json()
                        return {
                            "status": "✅ HEALTHY",
                            "response_time": response.headers.get("X-Response-Time", "N/A"),
                            "data": data
                        }
                    else:
                        return {
                            "status": f"⚠️ UNHEALTHY (HTTP {response.status})",
                            "response_time": "N/A",
                            "data": None
                        }
        except asyncio.TimeoutError:
            return {
                "status": "❌ TIMEOUT",
                "response_time": ">10s",
                "data": None
            }
        except Exception as e:
            return {
                "status": f"❌ ERROR: {str(e)}",
                "response_time": "N/A", 
                "data": None
            }
    
    async def test_graphql_endpoint(self, service_name: str, url: str) -> dict:
        """Test GraphQL endpoint with introspection query"""
        try:
            introspection_query = {
                "query": """
                query IntrospectionQuery {
                    __schema {
                        queryType {
                            name
                            fields {
                                name
                            }
                        }
                    }
                }
                """
            }
            
            timeout = aiohttp.ClientTimeout(total=10)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.post(
                    url,
                    json=introspection_query,
                    headers={"Content-Type": "application/json"}
                ) as response:
                    if response.status == 200:
                        data = await response.json()
                        if "data" in data and "__schema" in data["data"]:
                            query_fields = data["data"]["__schema"]["queryType"]["fields"]
                            field_names = [field["name"] for field in query_fields]
                            return {
                                "status": "✅ HEALTHY",
                                "response_time": response.headers.get("X-Response-Time", "N/A"),
                                "data": {
                                    "available_queries": field_names[:5],  # Show first 5
                                    "total_queries": len(field_names)
                                }
                            }
                        else:
                            return {
                                "status": "⚠️ INVALID RESPONSE",
                                "response_time": "N/A",
                                "data": data
                            }
                    else:
                        return {
                            "status": f"⚠️ UNHEALTHY (HTTP {response.status})",
                            "response_time": "N/A",
                            "data": None
                        }
        except Exception as e:
            return {
                "status": f"❌ ERROR: {str(e)}",
                "response_time": "N/A",
                "data": None
            }
    
    async def test_context_service_recipe_query(self) -> dict:
        """Test Context Service recipe functionality"""
        try:
            query = {
                "query": """
                query GetAvailableRecipes {
                    getAvailableRecipes {
                        recipeId
                        recipeName
                        version
                        clinicalScenario
                        governanceApproved
                    }
                }
                """
            }
            
            timeout = aiohttp.ClientTimeout(total=15)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.post(
                    "http://localhost:8016/graphql",
                    json=query,
                    headers={"Content-Type": "application/json"}
                ) as response:
                    if response.status == 200:
                        data = await response.json()
                        if "data" in data and "getAvailableRecipes" in data["data"]:
                            recipes = data["data"]["getAvailableRecipes"]
                            return {
                                "status": "✅ RECIPES LOADED",
                                "response_time": response.headers.get("X-Response-Time", "N/A"),
                                "data": {
                                    "total_recipes": len(recipes),
                                    "recipes": [r["recipeId"] for r in recipes]
                                }
                            }
                        else:
                            return {
                                "status": "⚠️ NO RECIPES",
                                "response_time": "N/A",
                                "data": data
                            }
                    else:
                        return {
                            "status": f"⚠️ FAILED (HTTP {response.status})",
                            "response_time": "N/A",
                            "data": None
                        }
        except Exception as e:
            return {
                "status": f"❌ ERROR: {str(e)}",
                "response_time": "N/A",
                "data": None
            }
    
    async def run_all_tests(self):
        """Run all connection tests"""
        print("🔍 Clinical Context Service - Connection Test Suite")
        print("=" * 80)
        print(f"Test started at: {datetime.now().isoformat()}")
        print("=" * 80)
        
        # Test service health endpoints
        print("\n📡 Testing Service Health Endpoints:")
        print("-" * 50)
        
        for service_name, url in self.services.items():
            print(f"Testing {service_name}...", end=" ")
            result = await self.test_service_health(service_name, url)
            self.results[service_name] = result
            print(f"{result['status']}")
            if result['data'] and 'service' in str(result['data']):
                print(f"    Response: {result['data']}")
        
        # Test GraphQL endpoints
        print("\n🔗 Testing GraphQL Endpoints:")
        print("-" * 50)
        
        for service_name, url in self.graphql_endpoints.items():
            print(f"Testing {service_name}...", end=" ")
            result = await self.test_graphql_endpoint(service_name, url)
            self.results[service_name] = result
            print(f"{result['status']}")
            if result['data']:
                print(f"    Available queries: {result['data'].get('total_queries', 0)}")
        
        # Test Context Service specific functionality
        print("\n📋 Testing Context Service Recipe System:")
        print("-" * 50)
        
        print("Testing Recipe Loading...", end=" ")
        recipe_result = await self.test_context_service_recipe_query()
        self.results["Recipe System"] = recipe_result
        print(f"{recipe_result['status']}")
        if recipe_result['data']:
            print(f"    Loaded recipes: {recipe_result['data'].get('total_recipes', 0)}")
            if recipe_result['data'].get('recipes'):
                print(f"    Recipe IDs: {', '.join(recipe_result['data']['recipes'])}")
        
        # Print summary
        print("\n" + "=" * 80)
        print("📊 CONNECTION TEST SUMMARY")
        print("=" * 80)
        
        healthy_count = 0
        total_count = len(self.results)
        
        for service_name, result in self.results.items():
            status_icon = "✅" if "✅" in result['status'] else "❌"
            print(f"{status_icon} {service_name}: {result['status']}")
            if "✅" in result['status']:
                healthy_count += 1
        
        print("-" * 80)
        print(f"Overall Health: {healthy_count}/{total_count} services healthy")
        
        if healthy_count == total_count:
            print("🎉 ALL SERVICES CONNECTED AND HEALTHY!")
            print("   The Clinical Context Service is ready to assemble clinical context.")
        elif healthy_count >= total_count * 0.8:
            print("⚠️  Most services are healthy, but some issues detected.")
            print("   The Clinical Context Service may have limited functionality.")
        else:
            print("❌ MULTIPLE SERVICE ISSUES DETECTED!")
            print("   The Clinical Context Service may not function properly.")
        
        print("=" * 80)
        
        return healthy_count == total_count


async def main():
    """Main test function"""
    tester = ServiceConnectionTester()
    
    try:
        all_healthy = await tester.run_all_tests()
        
        if all_healthy:
            print("\n🚀 Ready to test clinical context assembly!")
            print("   Try these GraphQL queries:")
            print("   • getAvailableRecipes - List all available recipes")
            print("   • getContextByRecipe - Assemble clinical context")
            print("   • getCacheStats - View cache performance")
            print(f"   • GraphQL Playground: http://localhost:8016/graphql")
            sys.exit(0)
        else:
            print("\n⚠️  Some services are not healthy.")
            print("   Please start missing services and try again.")
            sys.exit(1)
            
    except KeyboardInterrupt:
        print("\n🛑 Test interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n💥 Test failed with error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    print("🧪 Starting Clinical Context Service Connection Tests...")
    asyncio.run(main())
