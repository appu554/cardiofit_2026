"""
Patient Service Client for Context Service
Connects to Patient Service HTTP API to get FHIR data (no Google libraries needed)
"""
import logging
import httpx
from typing import Dict, List, Any, Optional

logger = logging.getLogger(__name__)


class PatientServiceClient:
    """
    Client to connect to Patient Service HTTP API.
    Gets FHIR data via Patient Service (which handles Google FHIR Store connection).
    """
    
    def __init__(self, base_url: str = "http://localhost:8003"):
        """Initialize the client"""
        self.base_url = base_url.rstrip('/')
        self.timeout = 15.0

        # Default headers (including required auth headers for Patient Service)
        self.headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
            # Required headers for Patient Service authentication
            "X-User-ID": "context-service-user",
            "X-User-Email": "context-service@example.com",
            "X-User-Role": "system",
            "X-User-Roles": "system",
            "X-User-Permissions": "patient:read,patient:write,observation:read,observation:write,condition:read,condition:write,medication:read,medication:write,encounter:read,encounter:write,timeline:read"
        }

        logger.info(f"Initialized PatientServiceClient with base URL: {self.base_url}")
    
    async def health_check(self) -> Dict[str, Any]:
        """Check if Patient Service is healthy (no auth required)"""
        try:
            # Use minimal headers for health check (no auth required)
            health_headers = {
                "Content-Type": "application/json",
                "Accept": "application/json"
            }

            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.base_url}/health",
                    headers=health_headers,
                    timeout=self.timeout
                )

                if response.status_code == 200:
                    health_data = response.json()
                    logger.info("Patient Service is healthy")
                    return {"status": "healthy", "data": health_data}
                else:
                    logger.warning(f"Patient Service health check failed: HTTP {response.status_code}")
                    return {"status": "unhealthy", "code": response.status_code}

        except Exception as e:
            logger.error(f"Error checking Patient Service health: {e}")
            return {"status": "error", "error": str(e)}

    async def test_auth_endpoints(self) -> Dict[str, Any]:
        """Test if authentication headers work"""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.base_url}/api/patients",
                    params={"limit": 1},
                    headers=self.headers,
                    timeout=self.timeout
                )

                if response.status_code == 200:
                    logger.info("Authentication headers work correctly")
                    return {"status": "auth_success", "code": response.status_code}
                elif response.status_code == 401:
                    logger.warning("Authentication failed - headers may be incorrect")
                    return {"status": "auth_failed", "code": response.status_code}
                else:
                    logger.warning(f"Unexpected response: HTTP {response.status_code}")
                    return {"status": "unexpected", "code": response.status_code}

        except Exception as e:
            logger.error(f"Error testing auth endpoints: {e}")
            return {"status": "error", "error": str(e)}
    
    async def get_patient(self, patient_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get a patient by ID via Patient Service (using non-auth context endpoint)"""
        try:
            # Use simple headers (no auth required for context endpoints)
            headers = {
                "Content-Type": "application/json",
                "Accept": "application/json"
            }

            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.base_url}/api/context/patients/{patient_id}",
                    headers=headers,
                    timeout=self.timeout
                )

                if response.status_code == 200:
                    response_data = response.json()
                    # Extract patient from the response wrapper
                    patient_data = response_data.get("patient", response_data)
                    logger.info(f"Retrieved patient {patient_id} via Patient Service context endpoint")
                    return patient_data
                elif response.status_code == 404:
                    logger.warning(f"Patient {patient_id} not found")
                    return None
                else:
                    logger.error(f"Error getting patient {patient_id}: HTTP {response.status_code}")
                    return None

        except Exception as e:
            logger.error(f"Error getting patient {patient_id}: {e}")
            return None
    
    async def search_patients(self, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Search for patients via Patient Service (using non-auth context endpoint)"""
        try:
            # Use simple headers (no auth required for context endpoints)
            headers = {
                "Content-Type": "application/json",
                "Accept": "application/json"
            }

            # Default search parameters
            if params is None:
                params = {"limit": 10}

            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.base_url}/api/context/patients",
                    params=params,
                    headers=headers,
                    timeout=self.timeout
                )

                if response.status_code == 200:
                    response_data = response.json()

                    # Extract patients from the response wrapper
                    if "patients" in response_data:
                        patients = response_data["patients"]
                        logger.info(f"Found {len(patients)} patients via Patient Service context endpoint")
                        return patients
                    else:
                        logger.warning(f"Unexpected response format: {response_data.keys()}")
                        return []
                else:
                    logger.error(f"Error searching patients: HTTP {response.status_code}")
                    return []

        except Exception as e:
            logger.error(f"Error searching patients: {e}")
            return []
    
    async def get_patient_fhir(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Get a patient in FHIR format via Patient Service context endpoint"""
        try:
            # Use simple headers (no auth required for context endpoints)
            headers = {
                "Content-Type": "application/json",
                "Accept": "application/json"
            }

            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.base_url}/api/context/patients/{patient_id}/fhir",
                    headers=headers,
                    timeout=self.timeout
                )

                if response.status_code == 200:
                    fhir_data = response.json()
                    logger.info(f"Retrieved FHIR patient {patient_id} via Patient Service context endpoint")
                    return fhir_data
                elif response.status_code == 404:
                    logger.warning(f"FHIR patient {patient_id} not found")
                    return None
                else:
                    logger.error(f"Error getting FHIR patient {patient_id}: HTTP {response.status_code}")
                    return None

        except Exception as e:
            logger.error(f"Error getting FHIR patient {patient_id}: {e}")
            return None
    
    async def search_patients_fhir(self, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Search for patients in FHIR format via Patient Service context endpoint"""
        try:
            # Use simple headers (no auth required for context endpoints)
            headers = {
                "Content-Type": "application/json",
                "Accept": "application/json"
            }

            # Default search parameters
            if params is None:
                params = {"limit": 10}

            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.base_url}/api/context/patients",
                    params=params,
                    headers=headers,
                    timeout=self.timeout
                )

                if response.status_code == 200:
                    response_data = response.json()

                    # Extract patients from the response wrapper
                    if "patients" in response_data:
                        patients = response_data["patients"]
                        # Ensure patients are in FHIR format
                        fhir_patients = []
                        for patient in patients:
                            if not patient.get("resourceType"):
                                patient["resourceType"] = "Patient"
                            fhir_patients.append(patient)

                        logger.info(f"Found {len(fhir_patients)} FHIR patients via Patient Service context endpoint")
                        return fhir_patients
                    else:
                        logger.warning(f"Unexpected response format: {response_data.keys()}")
                        return []
                else:
                    logger.error(f"Error searching FHIR patients: HTTP {response.status_code}")
                    return []

        except Exception as e:
            logger.error(f"Error searching FHIR patients: {e}")
            return []
    
    def is_available(self) -> bool:
        """Check if the client is properly configured"""
        return bool(self.base_url)
    
    def get_status(self) -> Dict[str, Any]:
        """Get client status"""
        return {
            "base_url": self.base_url,
            "timeout": self.timeout,
            "available": self.is_available()
        }


# Global instance
_patient_client = None

def get_patient_client() -> PatientServiceClient:
    """Get the global Patient Service client instance"""
    global _patient_client
    
    if _patient_client is None:
        _patient_client = PatientServiceClient()
    
    return _patient_client


async def test_patient_service_connection():
    """Test the Patient Service connection"""
    print("🧪 Testing Patient Service Connection")
    print("=" * 60)
    
    # Get the client
    client = get_patient_client()
    
    # Check status
    status = client.get_status()
    print("Client Status:")
    for key, value in status.items():
        print(f"   {key}: {value}")
    
    if not client.is_available():
        print("\n❌ Client not available")
        return False
    
    # Test health check
    print("\n1. Testing health check...")
    health = await client.health_check()
    print(f"   Health status: {health.get('status')}")
    
    if health.get('status') != 'healthy':
        print("❌ Patient Service is not healthy")
        return False
    
    # Test patient search
    print("\n2. Testing patient search...")
    patients = await client.search_patients({"limit": 3})
    print(f"   Found {len(patients)} patients")
    
    # Test FHIR patient search
    print("\n3. Testing FHIR patient search...")
    fhir_patients = await client.search_patients_fhir({"_count": "3"})
    print(f"   Found {len(fhir_patients)} FHIR patients")
    
    # If we have patients, test getting one
    if patients:
        patient_id = patients[0].get("id")
        if patient_id:
            print(f"\n4. Testing get patient: {patient_id}")
            patient = await client.get_patient(patient_id)
            
            if patient:
                print(f"   ✅ Retrieved patient: {patient.get('resourceType', 'Unknown')} {patient.get('id')}")
                
                # Test FHIR format
                fhir_patient = await client.get_patient_fhir(patient_id)
                if fhir_patient:
                    print(f"   ✅ Retrieved FHIR patient: {fhir_patient.get('resourceType', 'Unknown')} {fhir_patient.get('id')}")
    
    print("\n✅ Patient Service connection test completed successfully!")
    return True


if __name__ == "__main__":
    import asyncio
    asyncio.run(test_patient_service_connection())
