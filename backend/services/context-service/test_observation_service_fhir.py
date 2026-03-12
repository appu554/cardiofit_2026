#!/usr/bin/env python3
"""
Test how observation service connects to FHIR Store.
Check if it uses Google FHIR Store or MongoDB.
"""
import asyncio
import httpx
import os
import sys

# Add backend directory to path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, backend_dir)


async def test_observation_service():
    """Test observation service connection"""
    print("🔍 Testing Observation Service FHIR Connection")
    print("=" * 60)
    
    observation_service_url = "http://localhost:8007"
    
    try:
        print(f"Testing: {observation_service_url}")
        
        # Test health endpoint
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{observation_service_url}/health",
                timeout=10.0
            )
            
            print(f"Health check: HTTP {response.status_code}")
            
            if response.status_code == 200:
                health_data = response.json()
                print(f"✅ Observation service is running")
                print(f"   Service: {health_data.get('service', 'Unknown')}")
                print(f"   Status: {health_data.get('status', 'Unknown')}")
                
                # Test observations API
                try:
                    obs_response = await client.get(
                        f"{observation_service_url}/api/observations",
                        params={"limit": 5},
                        timeout=10.0
                    )
                    
                    print(f"Observations API: HTTP {obs_response.status_code}")
                    
                    if obs_response.status_code == 200:
                        obs_data = obs_response.json()
                        print(f"✅ Retrieved observations via service API")
                        
                        if isinstance(obs_data, list):
                            print(f"   Found {len(obs_data)} observations")
                            if obs_data:
                                # Check first observation structure
                                first_obs = obs_data[0]
                                print(f"   Sample observation keys: {list(first_obs.keys())[:5]}")
                        elif isinstance(obs_data, dict):
                            print(f"   Response keys: {list(obs_data.keys())}")
                        
                        return True, "service_api"
                    else:
                        print(f"⚠️ Observations API returned: {obs_response.status_code}")
                        
                except Exception as e:
                    print(f"⚠️ Observations API error: {e}")
                
                return True, "health_only"
            else:
                print(f"❌ Observation service not healthy: {response.status_code}")
                return False, "unhealthy"
                
    except Exception as e:
        print(f"❌ Observation service not accessible: {e}")
        print("   This is normal if the observation service is not running")
        return False, "not_running"


async def check_observation_service_config():
    """Check observation service configuration"""
    print("\n🔍 Checking Observation Service Configuration")
    print("=" * 60)
    
    try:
        # Import observation service settings
        from services.observation_service.app.core.config import settings
        
        print("Configuration found:")
        print(f"   USE_GOOGLE_HEALTHCARE_API: {settings.USE_GOOGLE_HEALTHCARE_API}")
        print(f"   GOOGLE_CLOUD_PROJECT_ID: {settings.GOOGLE_CLOUD_PROJECT_ID}")
        print(f"   GOOGLE_CLOUD_LOCATION: {settings.GOOGLE_CLOUD_LOCATION}")
        print(f"   GOOGLE_CLOUD_DATASET_ID: {settings.GOOGLE_CLOUD_DATASET_ID}")
        print(f"   GOOGLE_CLOUD_FHIR_STORE_ID: {settings.GOOGLE_CLOUD_FHIR_STORE_ID}")
        
        if settings.USE_GOOGLE_HEALTHCARE_API:
            print("✅ Observation service is configured to use Google FHIR Store")
            print(f"   FHIR Store: {settings.fhir_store_name}")
        else:
            print("⚠️ Observation service is configured to use MongoDB (not Google FHIR Store)")
            print("   To enable Google FHIR Store, set: USE_GOOGLE_HEALTHCARE_API=true")
        
        return settings.USE_GOOGLE_HEALTHCARE_API
        
    except ImportError as e:
        print(f"❌ Could not import observation service config: {e}")
        return None
    except Exception as e:
        print(f"❌ Error checking config: {e}")
        return None


async def test_google_fhir_service_directly():
    """Test the Google FHIR service directly"""
    print("\n🔍 Testing Google FHIR Service Directly")
    print("=" * 60)
    
    try:
        # Import the Google FHIR service
        from services.observation_service.app.services.google_fhir_service import GoogleObservationFHIRService
        
        print("✅ Successfully imported GoogleObservationFHIRService")
        
        # Try to create an instance
        try:
            fhir_service = GoogleObservationFHIRService()
            print("✅ Successfully created GoogleObservationFHIRService instance")
            
            # Check if client is initialized
            if hasattr(fhir_service, 'client') and fhir_service.client:
                print("✅ Google Healthcare client is available")
                
                # Try to initialize the client
                try:
                    if fhir_service.client.initialize():
                        print("✅ Google Healthcare client initialized successfully")
                        return True
                    else:
                        print("❌ Failed to initialize Google Healthcare client")
                        return False
                except Exception as e:
                    print(f"❌ Error initializing client: {e}")
                    return False
            else:
                print("❌ No Google Healthcare client available")
                return False
                
        except Exception as e:
            print(f"❌ Error creating GoogleObservationFHIRService: {e}")
            return False
            
    except ImportError as e:
        print(f"❌ Could not import GoogleObservationFHIRService: {e}")
        print("   This might be due to missing Google libraries")
        return False
    except Exception as e:
        print(f"❌ Error testing Google FHIR service: {e}")
        return False


async def show_fhir_connection_summary():
    """Show summary of FHIR connection options"""
    print("\n💡 FHIR CONNECTION SUMMARY")
    print("=" * 60)
    
    print("Based on the observation service analysis:")
    
    print("\n🔍 OBSERVATION SERVICE PATTERN:")
    print("   1. Has USE_GOOGLE_HEALTHCARE_API setting (defaults to false)")
    print("   2. Uses MongoDB by default (not Google FHIR Store)")
    print("   3. Can be switched to Google FHIR Store with environment variable")
    print("   4. Uses shared GoogleHealthcareClient when enabled")
    
    print("\n🎯 FOR CONTEXT SERVICE:")
    print("   Option 1: Copy observation service pattern")
    print("   • Use MongoDB by default (same as observation service)")
    print("   • Add USE_GOOGLE_HEALTHCARE_API setting")
    print("   • Switch to Google FHIR Store when needed")
    
    print("\n   Option 2: Enable Google FHIR Store in observation service")
    print("   • Set USE_GOOGLE_HEALTHCARE_API=true")
    print("   • Install Google libraries in observation service")
    print("   • Use observation service as FHIR proxy")
    
    print("\n   Option 3: Direct Google FHIR Store connection")
    print("   • Install Google libraries in context service")
    print("   • Use shared GoogleHealthcareClient directly")
    print("   • Same pattern as patient service (when enabled)")


async def main():
    """Main test function"""
    print("🚀 Observation Service FHIR Connection Analysis")
    print("=" * 80)
    print("Analyzing how observation service connects to FHIR Store")
    print("=" * 80)
    
    try:
        # Test observation service
        service_running, service_status = await test_observation_service()
        
        # Check configuration
        uses_google_fhir = await check_observation_service_config()
        
        # Test Google FHIR service directly
        google_fhir_works = await test_google_fhir_service_directly()
        
        # Show summary
        await show_fhir_connection_summary()
        
        # Final recommendations
        print("\n" + "=" * 80)
        print("🎯 RECOMMENDATIONS")
        print("=" * 80)
        
        if service_running:
            print("✅ Observation service is running")
            if uses_google_fhir:
                print("✅ Observation service uses Google FHIR Store")
                if google_fhir_works:
                    print("✅ Google FHIR connection works")
                    print("\n🎯 RECOMMENDED: Copy observation service pattern exactly")
                else:
                    print("❌ Google FHIR connection has issues")
                    print("\n🎯 RECOMMENDED: Use observation service as proxy")
            else:
                print("⚠️ Observation service uses MongoDB (not Google FHIR Store)")
                print("\n🎯 RECOMMENDED: Either:")
                print("   1. Use MongoDB pattern (same as observation service)")
                print("   2. Enable Google FHIR Store in observation service")
        else:
            print("❌ Observation service is not running")
            print("\n🎯 RECOMMENDED: Start observation service first")
        
        return 0
        
    except KeyboardInterrupt:
        print("\n🛑 Test interrupted by user")
        return 1
    except Exception as e:
        print(f"\n💥 Test failed with error: {e}")
        return 1


if __name__ == "__main__":
    exit_code = asyncio.run(main())
    exit(exit_code)
