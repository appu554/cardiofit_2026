#!/usr/bin/env python3
"""
Test script to isolate startup issues in Stage 2
"""

import os
import sys
import asyncio
import traceback

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

print("🚀 Starting Stage 2 component tests...")

def test_imports():
    """Test if all imports work"""
    print("\n1️⃣ Testing imports...")
    
    try:
        print("   - Testing basic imports...")
        import structlog
        import fastapi
        print("   ✅ Basic imports OK")
        
        print("   - Testing app config...")
        from app.config import settings
        print(f"   ✅ Config loaded - Port: {settings.PORT}")
        
        print("   - Testing sink imports...")
        from app.sinks.fhir_store_sink import FHIRStoreSink
        from app.sinks.elasticsearch_sink import ElasticsearchSink  
        from app.sinks.mongodb_sink import MongoDBSink
        print("   ✅ Sink imports OK")
        
        print("   - Testing service imports...")
        from app.services.multi_sink_writer import MultiSinkWriterService
        from app.services.kafka_consumer import KafkaConsumerService
        from app.services.fhir_transformation import FHIRTransformationService
        print("   ✅ Service imports OK")
        
        return True
        
    except Exception as e:
        print(f"   ❌ Import failed: {e}")
        traceback.print_exc()
        return False

async def test_fhir_store():
    """Test FHIR Store sink initialization"""
    print("\n2️⃣ Testing FHIR Store sink...")
    
    try:
        from app.sinks.fhir_store_sink import FHIRStoreSink
        
        print("   - Creating FHIR Store sink...")
        sink = FHIRStoreSink()
        print("   ✅ FHIR Store sink created")
        
        print("   - Initializing FHIR Store sink...")
        await asyncio.wait_for(sink.initialize(), timeout=10.0)
        print("   ✅ FHIR Store sink initialized")
        
        return True
        
    except asyncio.TimeoutError:
        print("   ❌ FHIR Store sink initialization timed out")
        return False
    except Exception as e:
        print(f"   ❌ FHIR Store sink failed: {e}")
        traceback.print_exc()
        return False

async def test_elasticsearch():
    """Test Elasticsearch sink initialization"""
    print("\n3️⃣ Testing Elasticsearch sink...")
    
    try:
        from app.sinks.elasticsearch_sink import ElasticsearchSink
        
        print("   - Creating Elasticsearch sink...")
        sink = ElasticsearchSink()
        print("   ✅ Elasticsearch sink created")
        
        print("   - Initializing Elasticsearch sink...")
        await asyncio.wait_for(sink.initialize(), timeout=10.0)
        print("   ✅ Elasticsearch sink initialized")
        
        return True
        
    except asyncio.TimeoutError:
        print("   ❌ Elasticsearch sink initialization timed out")
        return False
    except Exception as e:
        print(f"   ❌ Elasticsearch sink failed: {e}")
        traceback.print_exc()
        return False

async def test_mongodb():
    """Test MongoDB sink initialization"""
    print("\n4️⃣ Testing MongoDB sink...")
    
    try:
        from app.sinks.mongodb_sink import MongoDBSink
        
        print("   - Creating MongoDB sink...")
        sink = MongoDBSink()
        print("   ✅ MongoDB sink created")
        
        print("   - Initializing MongoDB sink...")
        await asyncio.wait_for(sink.initialize(), timeout=10.0)
        print("   ✅ MongoDB sink initialized")
        
        return True
        
    except asyncio.TimeoutError:
        print("   ❌ MongoDB sink initialization timed out")
        return False
    except Exception as e:
        print(f"   ❌ MongoDB sink failed: {e}")
        traceback.print_exc()
        return False

async def test_multi_sink_writer():
    """Test Multi-Sink Writer service"""
    print("\n5️⃣ Testing Multi-Sink Writer...")
    
    try:
        from app.services.multi_sink_writer import MultiSinkWriterService
        
        print("   - Creating Multi-Sink Writer...")
        writer = MultiSinkWriterService()
        print("   ✅ Multi-Sink Writer created")
        
        print("   - Initializing Multi-Sink Writer...")
        await asyncio.wait_for(writer.initialize(), timeout=30.0)
        print("   ✅ Multi-Sink Writer initialized")
        
        return True
        
    except asyncio.TimeoutError:
        print("   ❌ Multi-Sink Writer initialization timed out")
        return False
    except Exception as e:
        print(f"   ❌ Multi-Sink Writer failed: {e}")
        traceback.print_exc()
        return False

async def main():
    """Run all tests"""
    print("🧪 Stage 2 Component Test Suite")
    print("=" * 50)
    
    # Test imports first
    if not test_imports():
        print("\n❌ Import tests failed - stopping")
        return
    
    # Test each component
    tests = [
        ("FHIR Store", test_fhir_store),
        ("Elasticsearch", test_elasticsearch), 
        ("MongoDB", test_mongodb),
        ("Multi-Sink Writer", test_multi_sink_writer)
    ]
    
    results = {}
    
    for name, test_func in tests:
        try:
            results[name] = await test_func()
        except Exception as e:
            print(f"\n❌ {name} test crashed: {e}")
            results[name] = False
    
    # Summary
    print("\n" + "=" * 50)
    print("🏁 Test Results Summary:")
    
    for name, passed in results.items():
        status = "✅ PASS" if passed else "❌ FAIL"
        print(f"   {name}: {status}")
    
    all_passed = all(results.values())
    if all_passed:
        print("\n🎉 All tests passed! Stage 2 should work.")
    else:
        print("\n⚠️  Some tests failed. Fix these issues before running Stage 2.")

if __name__ == "__main__":
    asyncio.run(main())
