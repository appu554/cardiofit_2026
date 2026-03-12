"""
Simple Google FHIR Client for Context Service
Uses the EXACT same pattern as patient service - imports the shared GoogleHealthcareClient
"""
import logging
import sys
import os
from typing import Dict, List, Any, Optional

# Add backend directory to path (same as patient service)
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared Google Healthcare client (same as patient service)
try:
    from services.shared.google_healthcare import GoogleHealthcareClient
    GOOGLE_CLIENT_AVAILABLE = True
    print("✅ Successfully imported GoogleHealthcareClient")
except ImportError as e:
    print(f"❌ Could not import GoogleHealthcareClient: {e}")
    GOOGLE_CLIENT_AVAILABLE = False

logger = logging.getLogger(__name__)


class SimpleGoogleFHIRClient:
    """
    Simple FHIR client using the shared GoogleHealthcareClient.
    Uses the EXACT same pattern as patient service.
    """
    
    def __init__(self):
        """Initialize the client with same settings as patient service"""
        # Same configuration as patient service
        self.project_id = "cardiofit-905a8"
        self.location = "asia-south1"
        self.dataset_id = "clinical-synthesis-hub"
        self.fhir_store_id = "fhir-store"
        self.credentials_path = "credentials/google-credentials.json"
        
        self.client = None
        self._initialized = False
        
        if GOOGLE_CLIENT_AVAILABLE:
            # Create client with same pattern as patient service
            self.client = GoogleHealthcareClient(
                project_id=self.project_id,
                location=self.location,
                dataset_id=self.dataset_id,
                fhir_store_id=self.fhir_store_id,
                credentials_path=self.credentials_path
            )
            print(f"✅ Created GoogleHealthcareClient instance")
        else:
            print("❌ GoogleHealthcareClient not available")
    
    async def initialize(self) -> bool:
        """Initialize the client (same pattern as patient service)"""
        if self._initialized:
            return True
        
        if not self.client:
            print("❌ No client available for initialization")
            return False
        
        try:
            # Initialize the client (same as patient service)
            success = self.client.initialize()
            if success:
                self._initialized = True
                print("✅ Google Healthcare API client initialized successfully")
                return True
            else:
                print("❌ Failed to initialize Google Healthcare API client")
                return False
                
        except Exception as e:
            print(f"❌ Error initializing client: {e}")
            return False
    
    async def get_patient(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Get a patient resource (same pattern as patient service)"""
        if not self._initialized:
            await self.initialize()
        
        if not self._initialized:
            return None
        
        try:
            # Same method call as patient service
            patient = await self.client.get_resource(patient_id)
            if patient:
                print(f"✅ Retrieved patient: {patient_id}")
                return patient
            else:
                print(f"⚠️ Patient not found: {patient_id}")
                return None
                
        except Exception as e:
            print(f"❌ Error getting patient {patient_id}: {e}")
            return None
    
    async def search_patients(self, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Search for patients (same pattern as patient service)"""
        if not self._initialized:
            await self.initialize()
        
        if not self._initialized:
            return []
        
        try:
            # Same method call as patient service
            if params is None:
                params = {"_count": "10"}
            
            patients = await self.client.search_resources(params)
            print(f"✅ Found {len(patients)} patients")
            return patients
            
        except Exception as e:
            print(f"❌ Error searching patients: {e}")
            return []
    
    async def get_medication_requests(self, patient_id: str) -> List[Dict[str, Any]]:
        """Get medication requests for a patient"""
        if not self._initialized:
            await self.initialize()
        
        if not self._initialized:
            return []
        
        try:
            # Search for MedicationRequest resources
            params = {
                "patient": patient_id,
                "status": "active",
                "_count": "50"
            }
            
            # Use the client's search method for MedicationRequest
            medications = await self.client.search_resources(params, resource_type="MedicationRequest")
            print(f"✅ Found {len(medications)} medication requests for patient {patient_id}")
            return medications
            
        except Exception as e:
            print(f"❌ Error getting medication requests for {patient_id}: {e}")
            return []
    
    async def get_observations(self, patient_id: str, category: str = None) -> List[Dict[str, Any]]:
        """Get observations for a patient"""
        if not self._initialized:
            await self.initialize()
        
        if not self._initialized:
            return []
        
        try:
            # Search for Observation resources
            params = {
                "patient": patient_id,
                "_count": "100",
                "_sort": "-date"
            }
            
            if category:
                params["category"] = category
            
            # Use the client's search method for Observation
            observations = await self.client.search_resources(params, resource_type="Observation")
            print(f"✅ Found {len(observations)} observations for patient {patient_id}")
            return observations
            
        except Exception as e:
            print(f"❌ Error getting observations for {patient_id}: {e}")
            return []
    
    async def get_conditions(self, patient_id: str) -> List[Dict[str, Any]]:
        """Get conditions for a patient"""
        if not self._initialized:
            await self.initialize()
        
        if not self._initialized:
            return []
        
        try:
            # Search for Condition resources
            params = {
                "patient": patient_id,
                "_count": "50"
            }
            
            # Use the client's search method for Condition
            conditions = await self.client.search_resources(params, resource_type="Condition")
            print(f"✅ Found {len(conditions)} conditions for patient {patient_id}")
            return conditions
            
        except Exception as e:
            print(f"❌ Error getting conditions for {patient_id}: {e}")
            return []
    
    def is_available(self) -> bool:
        """Check if the client is available"""
        return GOOGLE_CLIENT_AVAILABLE and self.client is not None
    
    def get_status(self) -> Dict[str, Any]:
        """Get client status"""
        return {
            "google_client_available": GOOGLE_CLIENT_AVAILABLE,
            "client_created": self.client is not None,
            "initialized": self._initialized,
            "project_id": self.project_id,
            "location": self.location,
            "dataset_id": self.dataset_id,
            "fhir_store_id": self.fhir_store_id,
            "credentials_path": self.credentials_path
        }


# Global instance (same pattern as patient service)
_fhir_client = None

def get_fhir_client() -> SimpleGoogleFHIRClient:
    """Get the global FHIR client instance (same pattern as patient service)"""
    global _fhir_client
    
    if _fhir_client is None:
        _fhir_client = SimpleGoogleFHIRClient()
    
    return _fhir_client


async def test_fhir_client():
    """Test the FHIR client"""
    print("🧪 Testing Simple Google FHIR Client")
    print("=" * 60)
    
    # Get the client
    client = get_fhir_client()
    
    # Check status
    status = client.get_status()
    print("Client Status:")
    for key, value in status.items():
        print(f"   {key}: {value}")
    
    if not client.is_available():
        print("\n❌ Client not available - cannot run tests")
        return False
    
    # Initialize
    print("\n1. Initializing client...")
    success = await client.initialize()
    
    if not success:
        print("❌ Failed to initialize client")
        return False
    
    # Test patient search
    print("\n2. Testing patient search...")
    patients = await client.search_patients({"_count": "5"})
    print(f"   Found {len(patients)} patients")
    
    # If we have patients, test getting one
    if patients:
        patient_id = patients[0].get("id")
        if patient_id:
            print(f"\n3. Testing get patient: {patient_id}")
            patient = await client.get_patient(patient_id)
            
            if patient:
                print(f"   ✅ Retrieved patient: {patient.get('resourceType')} {patient.get('id')}")
                
                # Test getting related resources
                print(f"\n4. Testing related resources for patient: {patient_id}")
                
                medications = await client.get_medication_requests(patient_id)
                print(f"   Medications: {len(medications)}")
                
                observations = await client.get_observations(patient_id)
                print(f"   Observations: {len(observations)}")
                
                conditions = await client.get_conditions(patient_id)
                print(f"   Conditions: {len(conditions)}")
    
    print("\n✅ FHIR client test completed successfully!")
    return True


if __name__ == "__main__":
    import asyncio
    asyncio.run(test_fhir_client())
