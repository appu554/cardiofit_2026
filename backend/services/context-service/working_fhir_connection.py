#!/usr/bin/env python3
"""
Working FHIR Store Connection - Direct to Google Cloud Healthcare API
Uses the same approach as the ETL pipeline that actually works.
"""
import asyncio
import httpx
import json
import os
import subprocess
import sys
from typing import Dict, Any, Optional


class WorkingFHIRConnection:
    """
    Working FHIR connection using gcloud auth token.
    This is the simplest approach that actually works.
    """
    
    def __init__(self):
        # Your FHIR Store configuration
        self.project_id = "cardiofit-905a8"
        self.location = "asia-south1"
        self.dataset_id = "clinical-synthesis-hub"
        self.fhir_store_id = "fhir-store"
        
        # Build FHIR Store URL
        self.fhir_store_path = f"projects/{self.project_id}/locations/{self.location}/datasets/{self.dataset_id}/fhirStores/{self.fhir_store_id}"
        self.base_url = f"https://healthcare.googleapis.com/v1/{self.fhir_store_path}/fhir"
        
        print(f"🎯 FHIR Store: {self.fhir_store_path}")
        print(f"🌐 Base URL: {self.base_url}")
    
    def get_gcloud_access_token(self) -> Optional[str]:
        """Get access token using gcloud command (simplest approach)"""
        try:
            print("🔑 Getting access token via gcloud...")
            
            # Use gcloud to get access token
            result = subprocess.run(
                ["gcloud", "auth", "print-access-token"],
                capture_output=True,
                text=True,
                timeout=30
            )
            
            if result.returncode == 0:
                token = result.stdout.strip()
                print(f"✅ Got access token: {token[:20]}...")
                return token
            else:
                print(f"❌ gcloud error: {result.stderr}")
                return None
                
        except subprocess.TimeoutExpired:
            print("❌ gcloud command timed out")
            return None
        except FileNotFoundError:
            print("❌ gcloud command not found")
            print("   Install Google Cloud SDK: https://cloud.google.com/sdk/docs/install")
            return None
        except Exception as e:
            print(f"❌ Error getting token: {e}")
            return None
    
    async def test_connection(self) -> bool:
        """Test connection to FHIR Store"""
        try:
            # Get access token
            token = self.get_gcloud_access_token()
            if not token:
                print("❌ No access token available")
                return False
            
            # Test metadata endpoint
            headers = {
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/fhir+json",
                "Accept": "application/fhir+json"
            }
            
            print(f"\n🔍 Testing: {self.base_url}/metadata")
            
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.base_url}/metadata",
                    headers=headers,
                    timeout=15.0
                )
                
                print(f"Response: HTTP {response.status_code}")
                
                if response.status_code == 200:
                    metadata = response.json()
                    print(f"✅ SUCCESS! FHIR Version: {metadata.get('fhirVersion', 'Unknown')}")
                    print(f"   Software: {metadata.get('software', {}).get('name', 'Unknown')}")
                    return True
                elif response.status_code == 401:
                    print("❌ HTTP 401 - Authentication failed")
                    print("   Try: gcloud auth login")
                    return False
                elif response.status_code == 403:
                    print("❌ HTTP 403 - Permission denied")
                    print("   Check if your account has Healthcare API permissions")
                    return False
                else:
                    print(f"❌ HTTP {response.status_code}")
                    print(f"   Response: {response.text[:200]}...")
                    return False
                    
        except Exception as e:
            print(f"❌ Connection error: {e}")
            return False
    
    async def get_patient(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Get a patient from FHIR Store"""
        try:
            token = self.get_gcloud_access_token()
            if not token:
                return None
            
            headers = {
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/fhir+json",
                "Accept": "application/fhir+json"
            }
            
            url = f"{self.base_url}/Patient/{patient_id}"
            print(f"🔍 Getting patient: {url}")
            
            async with httpx.AsyncClient() as client:
                response = await client.get(url, headers=headers, timeout=15.0)
                
                if response.status_code == 200:
                    patient = response.json()
                    print(f"✅ Found patient: {patient.get('id')} ({patient.get('resourceType')})")
                    return patient
                elif response.status_code == 404:
                    print(f"⚠️ Patient not found: {patient_id}")
                    return None
                else:
                    print(f"❌ Error getting patient: HTTP {response.status_code}")
                    return None
                    
        except Exception as e:
            print(f"❌ Error getting patient: {e}")
            return None
    
    async def search_patients(self, count: int = 10) -> list:
        """Search for patients in FHIR Store"""
        try:
            token = self.get_gcloud_access_token()
            if not token:
                return []
            
            headers = {
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/fhir+json",
                "Accept": "application/fhir+json"
            }
            
            url = f"{self.base_url}/Patient"
            params = {"_count": str(count)}
            
            print(f"🔍 Searching patients: {url}")
            
            async with httpx.AsyncClient() as client:
                response = await client.get(url, headers=headers, params=params, timeout=15.0)
                
                if response.status_code == 200:
                    bundle = response.json()
                    entries = bundle.get("entry", [])
                    patients = [entry["resource"] for entry in entries]
                    
                    print(f"✅ Found {len(patients)} patients")
                    for patient in patients[:3]:  # Show first 3
                        print(f"   - {patient.get('id')}: {patient.get('name', [{}])[0].get('family', 'Unknown')}")
                    
                    return patients
                else:
                    print(f"❌ Error searching patients: HTTP {response.status_code}")
                    return []
                    
        except Exception as e:
            print(f"❌ Error searching patients: {e}")
            return []
    
    async def get_medications(self, patient_id: str) -> list:
        """Get medications for a patient"""
        try:
            token = self.get_gcloud_access_token()
            if not token:
                return []
            
            headers = {
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/fhir+json",
                "Accept": "application/fhir+json"
            }
            
            url = f"{self.base_url}/MedicationRequest"
            params = {
                "patient": patient_id,
                "status": "active",
                "_count": "50"
            }
            
            print(f"🔍 Getting medications for patient: {patient_id}")
            
            async with httpx.AsyncClient() as client:
                response = await client.get(url, headers=headers, params=params, timeout=15.0)
                
                if response.status_code == 200:
                    bundle = response.json()
                    entries = bundle.get("entry", [])
                    medications = [entry["resource"] for entry in entries]
                    
                    print(f"✅ Found {len(medications)} medications")
                    return medications
                else:
                    print(f"❌ Error getting medications: HTTP {response.status_code}")
                    return []
                    
        except Exception as e:
            print(f"❌ Error getting medications: {e}")
            return []


async def test_working_connection():
    """Test the working FHIR connection"""
    print("🚀 Testing Working FHIR Store Connection")
    print("=" * 80)
    print("Using gcloud auth token (simplest approach that works)")
    print("=" * 80)
    
    # Create client
    client = WorkingFHIRConnection()
    
    # Test connection
    print("\n1. Testing FHIR Store connection...")
    connection_ok = await client.test_connection()
    
    if not connection_ok:
        print("\n❌ Connection failed. Try these steps:")
        print("   1. Install Google Cloud SDK")
        print("   2. Run: gcloud auth login")
        print("   3. Run: gcloud config set project cardiofit-905a8")
        print("   4. Run this test again")
        return False
    
    # Test patient search
    print("\n2. Testing patient search...")
    patients = await client.search_patients(5)
    
    # Test getting a specific patient
    if patients:
        patient_id = patients[0].get("id")
        print(f"\n3. Testing get patient: {patient_id}")
        patient = await client.get_patient(patient_id)
        
        if patient:
            print(f"\n4. Testing medications for patient: {patient_id}")
            medications = await client.get_medications(patient_id)
    
    print("\n" + "=" * 80)
    print("🎯 SUMMARY")
    print("=" * 80)
    print("✅ Direct FHIR Store connection works!")
    print("✅ This is the simplest approach for Context Service")
    print("✅ No additional Python libraries needed")
    print("✅ Just requires gcloud CLI authentication")
    
    print("\n💡 FOR CONTEXT SERVICE:")
    print("   Use this exact pattern:")
    print("   1. Get token: gcloud auth print-access-token")
    print("   2. Add header: Authorization: Bearer {token}")
    print("   3. Make HTTP requests to FHIR Store")
    print("   4. Same as this working example!")
    
    return True


async def main():
    """Main function"""
    try:
        success = await test_working_connection()
        return 0 if success else 1
    except KeyboardInterrupt:
        print("\n🛑 Test interrupted")
        return 1
    except Exception as e:
        print(f"\n💥 Test failed: {e}")
        return 1


if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
