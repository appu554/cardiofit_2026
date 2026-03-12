#!/usr/bin/env python3

"""
Simple test script to verify EncounterManagementService federation endpoint
"""

import requests
import json
import sys

def test_federation_endpoint():
    """Test the federation endpoint with a simple introspection query"""
    
    url = "http://localhost:8020/api/federation"
    
    # Simple introspection query
    query = {
        "query": """
        query {
            __schema {
                types {
                    name
                    kind
                }
            }
        }
        """
    }
    
    try:
        print("🔍 Testing federation endpoint...")
        print(f"URL: {url}")
        
        response = requests.post(
            url,
            json=query,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        print(f"Status Code: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            if "data" in data and "__schema" in data["data"]:
                types = data["data"]["__schema"]["types"]
                print(f"✅ Federation endpoint working! Found {len(types)} types")
                
                # Look for encounter-related types
                encounter_types = [t for t in types if "encounter" in t["name"].lower() or "location" in t["name"].lower()]
                
                if encounter_types:
                    print(f"✅ Found {len(encounter_types)} encounter-related types:")
                    for t in encounter_types[:10]:  # Show first 10
                        print(f"  - {t['name']} ({t['kind']})")
                else:
                    print("⚠️ No encounter-related types found")
                
                return True
            else:
                print("❌ Invalid response structure")
                print(f"Response: {json.dumps(data, indent=2)}")
                return False
        else:
            print(f"❌ HTTP Error: {response.status_code}")
            print(f"Response: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        print("❌ Connection failed - is the service running on port 8020?")
        return False
    except requests.exceptions.Timeout:
        print("❌ Request timeout")
        return False
    except Exception as e:
        print(f"❌ Error: {e}")
        return False

def test_encounter_type():
    """Test if the Encounter type is properly defined"""
    
    url = "http://localhost:8020/api/federation"
    
    query = {
        "query": """
        query {
            __type(name: "Encounter") {
                name
                fields {
                    name
                    type {
                        name
                        kind
                    }
                }
            }
        }
        """
    }
    
    try:
        print("\n🔍 Testing Encounter type definition...")
        
        response = requests.post(
            url,
            json=query,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code == 200:
            data = response.json()
            
            if "data" in data and "__type" in data["data"] and data["data"]["__type"]:
                encounter_type = data["data"]["__type"]
                fields = encounter_type.get("fields", [])
                
                print(f"✅ Encounter type found with {len(fields)} fields")
                
                # Check for required fields
                required_fields = ["id", "status", "encounterClass", "subject"]
                found_fields = [f["name"] for f in fields]
                
                for field in required_fields:
                    if field in found_fields:
                        print(f"  ✅ {field}")
                    else:
                        print(f"  ❌ {field} (missing)")
                
                return True
            else:
                print("❌ Encounter type not found")
                return False
        else:
            print(f"❌ HTTP Error: {response.status_code}")
            return False
            
    except Exception as e:
        print(f"❌ Error testing Encounter type: {e}")
        return False

if __name__ == "__main__":
    print("🚀 Testing EncounterManagementService Federation Endpoint\n")
    
    # Test federation endpoint
    federation_ok = test_federation_endpoint()
    
    # Test encounter type
    encounter_ok = test_encounter_type()
    
    print(f"\n📊 Results:")
    print(f"Federation Endpoint: {'✅ PASS' if federation_ok else '❌ FAIL'}")
    print(f"Encounter Type: {'✅ PASS' if encounter_ok else '❌ FAIL'}")
    
    if federation_ok and encounter_ok:
        print(f"\n🎉 All tests passed! Federation endpoint is ready.")
        print(f"You can now run: node apollo-federation/generate-supergraph.js")
    else:
        print(f"\n❌ Some tests failed. Check the service configuration.")
        sys.exit(1)
