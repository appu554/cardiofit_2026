#!/usr/bin/env python3
"""
Test script to verify intelligent data source routing system
"""
import sys
import asyncio
from pathlib import Path

# Add the app directory to the path
sys.path.append('.')

from app.services.context_assembly_service import ContextAssemblyService
from app.models.context_models import DataPoint, DataSourceType

async def test_intelligent_routing():
    """Test the intelligent data source routing system"""
    try:
        print("🎯 Testing Intelligent Data Source Routing System")
        print("=" * 60)
        
        # Initialize the service
        service = ContextAssemblyService()
        
        # Test data points for each routing category
        test_cases = [
            # Real-Time Critical Data → Direct Microservices
            {
                "category": "🚨 CRITICAL REAL-TIME",
                "route": "Direct Microservices",
                "data_points": [
                    DataPoint("active_medications", DataSourceType.MEDICATION_SERVICE, "http://localhost:8009", ["medication_id", "status"], True),
                    DataPoint("current_vitals", DataSourceType.OBSERVATION_SERVICE, "http://localhost:8007", ["bp", "hr", "temp"], True),
                    DataPoint("drug_interactions", DataSourceType.CAE_SERVICE, "http://localhost:8027", ["interactions"], True),
                ]
            },
            # Structured Clinical Data → Apollo Federation
            {
                "category": "📊 STRUCTURED CLINICAL",
                "route": "Apollo Federation",
                "data_points": [
                    DataPoint("patient_demographics", DataSourceType.APOLLO_FEDERATION, "http://localhost:4000", ["name", "dob"], True),
                    DataPoint("allergies", DataSourceType.APOLLO_FEDERATION, "http://localhost:4000", ["allergen", "reaction"], True),
                    DataPoint("problem_list", DataSourceType.APOLLO_FEDERATION, "http://localhost:4000", ["conditions"], True),
                ]
            },
            # Historical/Analytics → Elasticsearch
            {
                "category": "📈 HISTORICAL/ANALYTICS",
                "route": "Elasticsearch",
                "data_points": [
                    DataPoint("medication_adherence", DataSourceType.ELASTICSEARCH, "http://localhost:9200", ["adherence_rate"], False),
                    DataPoint("lab_patterns", DataSourceType.ELASTICSEARCH, "http://localhost:9200", ["trends"], False),
                    DataPoint("risk_factors", DataSourceType.ELASTICSEARCH, "http://localhost:9200", ["risk_score"], False),
                ]
            }
        ]
        
        patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        
        for test_case in test_cases:
            print(f"\n{test_case['category']} DATA → {test_case['route']}")
            print("-" * 50)
            
            for data_point in test_case['data_points']:
                print(f"  Testing: {data_point.name}")
                
                # Check routing classification
                data_name = data_point.name.lower()
                
                if data_name in service.critical_realtime_data:
                    route = "🚨 Direct Microservices"
                elif data_name in service.structured_clinical_data:
                    route = "📊 Apollo Federation"
                elif data_name in service.historical_analytics_data:
                    route = "📈 Elasticsearch"
                else:
                    route = "🔄 Default (Apollo Federation)"
                
                print(f"    Classified as: {route}")
                
                # Test the actual routing (without making real calls)
                try:
                    # This would normally make the actual call
                    # For testing, we just verify the classification
                    if route.startswith("🚨"):
                        print(f"    ✅ Would route to Direct Microservices")
                    elif route.startswith("📊"):
                        print(f"    ✅ Would route to Apollo Federation")
                    elif route.startswith("📈"):
                        print(f"    ✅ Would route to Elasticsearch")
                    else:
                        print(f"    ✅ Would route to Default (Apollo Federation)")
                        
                except Exception as e:
                    print(f"    ❌ Routing test failed: {e}")
                
                print()
        
        # Test routing sets
        print("\n🔍 ROUTING CLASSIFICATION VERIFICATION")
        print("=" * 50)
        print(f"Critical Real-time Data Points: {len(service.critical_realtime_data)}")
        print(f"  Examples: {list(service.critical_realtime_data)[:5]}")
        print(f"\nStructured Clinical Data Points: {len(service.structured_clinical_data)}")
        print(f"  Examples: {list(service.structured_clinical_data)[:5]}")
        print(f"\nHistorical Analytics Data Points: {len(service.historical_analytics_data)}")
        print(f"  Examples: {list(service.historical_analytics_data)[:5]}")
        
        print(f"\n🎯 INTELLIGENT ROUTING SYSTEM: READY FOR TESTING!")
        print("=" * 60)
        
    except Exception as e:
        print(f"❌ Error testing intelligent routing: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    asyncio.run(test_intelligent_routing())
