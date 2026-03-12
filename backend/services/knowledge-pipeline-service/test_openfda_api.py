#!/usr/bin/env python3
"""
Test OpenFDA API connection and data retrieval
"""

import requests
import json

# OpenFDA API Configuration
OPENFDA_API_KEY = "Fd4NqfzTO03RYq4KINOZwg8lYz7sgkDriTeGYMnB"
OPENFDA_BASE_URL = "https://api.fda.gov/drug/event.json"

def test_basic_api_call():
    """Test basic API call without search parameters"""
    print("🧪 TESTING BASIC OPENFDA API CALL")
    print("=" * 50)
    
    params = {
        'api_key': OPENFDA_API_KEY,
        'limit': 5
    }
    
    try:
        print(f"📡 Making request to: {OPENFDA_BASE_URL}")
        print(f"📋 Parameters: {params}")
        
        response = requests.get(OPENFDA_BASE_URL, params=params, timeout=30)
        
        print(f"📊 Response status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            results = data.get('results', [])
            meta = data.get('meta', {})
            
            print(f"✅ Success! Received {len(results)} records")
            print(f"📈 Total available: {meta.get('results', {}).get('total', 'Unknown')}")
            
            if results:
                print("\n📋 Sample record structure:")
                sample = results[0]
                print(f"   Keys: {list(sample.keys())}")
                
                # Show patient info if available
                if 'patient' in sample:
                    patient = sample['patient']
                    print(f"   Patient keys: {list(patient.keys())}")
                    
                    if 'drug' in patient:
                        drugs = patient['drug']
                        if drugs:
                            print(f"   First drug keys: {list(drugs[0].keys())}")
                    
                    if 'reaction' in patient:
                        reactions = patient['reaction']
                        if reactions:
                            print(f"   First reaction keys: {list(reactions[0].keys())}")
            
            return True
        else:
            print(f"❌ API request failed: {response.status_code}")
            print(f"Response: {response.text[:500]}...")
            return False
            
    except Exception as e:
        print(f"❌ API request error: {e}")
        return False

def test_search_queries():
    """Test different search query formats"""
    print("\n🔍 TESTING DIFFERENT SEARCH QUERIES")
    print("=" * 50)
    
    search_queries = [
        'serious:1',
        'patient.drug.medicinalproduct:"aspirin"',
        'receivedate:[20240101+TO+20241231]',
        'patient.reaction.reactionmeddrapt:"headache"'
    ]
    
    for search_query in search_queries:
        print(f"\n🔎 Testing search: {search_query}")
        
        params = {
            'api_key': OPENFDA_API_KEY,
            'limit': 3,
            'search': search_query
        }
        
        try:
            response = requests.get(OPENFDA_BASE_URL, params=params, timeout=30)
            
            if response.status_code == 200:
                data = response.json()
                results = data.get('results', [])
                total = data.get('meta', {}).get('results', {}).get('total', 0)
                print(f"   ✅ Success: {len(results)} records (total: {total:,})")
            else:
                print(f"   ❌ Failed: {response.status_code}")
                print(f"   Response: {response.text[:200]}...")
                
        except Exception as e:
            print(f"   ❌ Error: {e}")

def test_pagination():
    """Test pagination with skip parameter"""
    print("\n📄 TESTING PAGINATION")
    print("=" * 30)
    
    params = {
        'api_key': OPENFDA_API_KEY,
        'limit': 5,
        'skip': 0,
        'search': 'serious:1'
    }
    
    try:
        response = requests.get(OPENFDA_BASE_URL, params=params, timeout=30)
        
        if response.status_code == 200:
            data = response.json()
            results = data.get('results', [])
            meta = data.get('meta', {})
            
            print(f"✅ Pagination test successful")
            print(f"   Records received: {len(results)}")
            print(f"   Skip: {params['skip']}")
            print(f"   Limit: {params['limit']}")
            print(f"   Total available: {meta.get('results', {}).get('total', 'Unknown')}")
            
            return True
        else:
            print(f"❌ Pagination test failed: {response.status_code}")
            return False
            
    except Exception as e:
        print(f"❌ Pagination test error: {e}")
        return False

def main():
    """Main test function"""
    print("🧪 OPENFDA API TESTING SUITE")
    print("=" * 60)
    
    # Test basic API call
    basic_success = test_basic_api_call()
    
    if basic_success:
        # Test search queries
        test_search_queries()
        
        # Test pagination
        pagination_success = test_pagination()
        
        print("\n" + "=" * 60)
        print("📊 TEST SUMMARY")
        print("=" * 60)
        print(f"Basic API call: {'✅ Success' if basic_success else '❌ Failed'}")
        print(f"Pagination: {'✅ Success' if pagination_success else '❌ Failed'}")
        
        if basic_success and pagination_success:
            print("🎉 OpenFDA API is working correctly!")
            print("Ready to proceed with adverse event ingestion.")
        else:
            print("⚠️ Some tests failed. Check API configuration.")
    else:
        print("\n❌ Basic API test failed. Cannot proceed with other tests.")
        print("Please check your API key and network connection.")

if __name__ == "__main__":
    main()
