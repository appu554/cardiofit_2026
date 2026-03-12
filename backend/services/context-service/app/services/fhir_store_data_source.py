"""
Direct FHIR Store Data Source for Clinical Context Service
Connects directly to Google Cloud Healthcare API FHIR Store using the same pattern as other services
"""
import logging
import json
import os
import sys
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
import asyncio
import aiohttp
import httpx

from app.models.context_models import DataPoint, SourceMetadata, DataSourceType

logger = logging.getLogger(__name__)


class FHIRStoreDataSource:
    """
    Direct FHIR Store data source for clinical context assembly.
    Uses the same pattern as other services - shared Google Healthcare client.
    """

    def __init__(self):
        # Your FHIR Store configuration (same as other services)
        self.project_id = "cardiofit-905a8"
        self.location = "asia-south1"
        self.dataset_id = "clinical-synthesis-hub"
        self.fhir_store_id = "fhir-store"

        # Build FHIR Store path (same pattern as other services)
        self.fhir_store_path = f"projects/{self.project_id}/locations/{self.location}/datasets/{self.dataset_id}/fhirStores/{self.fhir_store_id}"
        self.base_url = f"https://healthcare.googleapis.com/v1/{self.fhir_store_path}/fhir"

        # Use shared Google Healthcare client (same as other services)
        self.client = None

        # Connection status
        self.connection_healthy = False
        
        # FHIR resource type mappings
        self.resource_mappings = {
            "patient_demographics": "Patient",
            "patient_medications": "MedicationRequest",
            "patient_conditions": "Condition",
            "patient_allergies": "AllergyIntolerance",
            "lab_results": "Observation",
            "vital_signs": "Observation",
            "encounters": "Encounter",
            "procedures": "Procedure",
            "diagnostic_reports": "DiagnosticReport"
        }
    
    async def initialize(self):
        """Initialize FHIR Store connection using shared client pattern"""
        try:
            logger.info("🏥 Initializing FHIR Store connection...")

            # Try to import the shared Google Healthcare client (same as other services)
            try:
                # Add backend directory to path
                backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
                sys.path.insert(0, backend_dir)

                from services.shared.google_healthcare.client import GoogleHealthcareClient

                # Initialize the client with same settings as other services
                self.client = GoogleHealthcareClient(
                    project_id=self.project_id,
                    location=self.location,
                    dataset_id=self.dataset_id,
                    fhir_store_id=self.fhir_store_id,
                    credentials_path="../services/encounter-service/credentials/google-credentials.json"
                )

                # Initialize the client
                if self.client.initialize():
                    self.connection_healthy = True
                    logger.info("✅ FHIR Store connection established using shared client")
                    logger.info(f"   Project: {self.project_id}")
                    logger.info(f"   Dataset: {self.dataset_id}")
                    logger.info(f"   FHIR Store: {self.fhir_store_id}")
                    logger.info(f"   Base URL: {self.base_url}")
                    return True
                else:
                    logger.warning("⚠️ Shared client initialization failed, using direct HTTP")
                    self.client = None

            except ImportError as e:
                logger.warning(f"⚠️ Could not import shared client: {e}, using direct HTTP")
                self.client = None

            # Fallback to direct HTTP requests (like medication service)
            await self._test_direct_connection()

            self.connection_healthy = True
            logger.info("✅ FHIR Store connection established using direct HTTP")
            return True

        except Exception as e:
            logger.error(f"❌ Failed to connect to FHIR Store: {e}")
            self.connection_healthy = False
            return False
    
    async def _test_direct_connection(self):
        """Test FHIR Store connection using direct HTTP (like medication service)"""
        try:
            # Get FHIR URLs (same pattern as medication service)
            fhir_urls = self._get_fhir_urls()

            for url in fhir_urls:
                try:
                    logger.info(f"Testing FHIR server URL: {url}")

                    # Simple metadata request to test connection
                    metadata_url = f"{url}/metadata"

                    async with httpx.AsyncClient() as client:
                        response = await client.get(
                            metadata_url,
                            timeout=10.0
                        )

                        if response.status_code == 200:
                            metadata = response.json()
                            logger.info(f"✅ FHIR Store metadata retrieved from {url}")
                            logger.info(f"   FHIR Version: {metadata.get('fhirVersion', 'Unknown')}")
                            return True
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code}")

                except Exception as e:
                    logger.warning(f"Exception testing {url}: {str(e)}")
                    continue

            raise Exception("All FHIR URLs failed connection test")

        except Exception as e:
            logger.error(f"❌ FHIR Store connection test failed: {e}")
            raise

    def _get_fhir_urls(self):
        """Get FHIR URLs (same pattern as medication service)"""
        # Try multiple FHIR endpoints (same as medication service pattern)
        fhir_urls = [
            f"https://healthcare.googleapis.com/v1/{self.fhir_store_path}/fhir",
            f"http://localhost:8014/fhir",  # Local FHIR service fallback
        ]
        return fhir_urls
    
    async def fetch_patient_demographics(self, patient_id: str, data_point: DataPoint) -> Dict[str, Any]:
        """Fetch patient demographics from FHIR Store (same pattern as other services)"""
        try:
            if not self.connection_healthy:
                await self.initialize()

            # Use shared client if available
            if self.client:
                try:
                    patient_resource = await self.client.get_fhir_resource("Patient", patient_id)
                    if patient_resource:
                        demographics = self._extract_patient_demographics(patient_resource)

                        source_metadata = SourceMetadata(
                            source_type=DataSourceType.FHIR_STORE,
                            source_endpoint=self.base_url,
                            retrieved_at=datetime.utcnow(),
                            data_version="FHIR R4",
                            completeness=self._calculate_completeness(demographics, data_point.fields),
                            response_time_ms=0.0,
                            cache_hit=False
                        )

                        return {
                            "data": demographics,
                            "metadata": source_metadata,
                            "success": True
                        }
                except Exception as e:
                    logger.warning(f"Shared client failed, trying direct HTTP: {e}")

            # Fallback to direct HTTP (same pattern as medication service)
            fhir_urls = self._get_fhir_urls()

            for url in fhir_urls:
                try:
                    patient_url = f"{url}/Patient/{patient_id}"

                    async with httpx.AsyncClient() as client:
                        response = await client.get(
                            patient_url,
                            timeout=10.0
                        )

                        if response.status_code == 200:
                            patient_resource = response.json()
                            demographics = self._extract_patient_demographics(patient_resource)

                            source_metadata = SourceMetadata(
                                source_type=DataSourceType.FHIR_STORE,
                                source_endpoint=url,
                                retrieved_at=datetime.utcnow(),
                                data_version="FHIR R4",
                                completeness=self._calculate_completeness(demographics, data_point.fields),
                                response_time_ms=0.0,
                                cache_hit=False
                            )

                            return {
                                "data": demographics,
                                "metadata": source_metadata,
                                "success": True
                            }
                        elif response.status_code == 404:
                            logger.info(f"Patient {patient_id} not found in {url}")
                            continue
                        else:
                            logger.warning(f"Error from {url}: {response.status_code}")
                            continue

                except Exception as e:
                    logger.warning(f"Exception with {url}: {str(e)}")
                    continue

            return {
                "data": {},
                "metadata": None,
                "success": False,
                "error": f"Patient {patient_id} not found in any FHIR Store"
            }

        except Exception as e:
            logger.error(f"❌ Error fetching patient demographics: {e}")
            return {
                "data": {},
                "metadata": None,
                "success": False,
                "error": str(e)
            }
    
    async def fetch_patient_medications(self, patient_id: str, data_point: DataPoint) -> Dict[str, Any]:
        """Fetch patient medications from FHIR Store"""
        try:
            if not self.connection_healthy:
                await self.initialize()
            
            await self._ensure_valid_token()
            
            # FHIR MedicationRequest search
            url = f"{self.base_url}/MedicationRequest"
            params = {
                "patient": patient_id,
                "status": "active",
                "_count": "50",
                "_sort": "-_lastUpdated"
            }
            headers = {
                "Authorization": f"Bearer {self.access_token}",
                "Content-Type": "application/fhir+json"
            }
            
            async with aiohttp.ClientSession() as session:
                async with session.get(url, headers=headers, params=params) as response:
                    if response.status == 200:
                        bundle = await response.json()
                        
                        # Extract medications from FHIR Bundle
                        medications = self._extract_medications_from_bundle(bundle)
                        
                        source_metadata = SourceMetadata(
                            source_type=DataSourceType.FHIR_STORE,
                            source_endpoint=self.base_url,
                            retrieved_at=datetime.utcnow(),
                            data_version="FHIR R4",
                            completeness=1.0 if medications else 0.0,
                            response_time_ms=0.0,
                            cache_hit=False
                        )
                        
                        return {
                            "data": {
                                "medications": medications,
                                "total_count": len(medications)
                            },
                            "metadata": source_metadata,
                            "success": True
                        }
                    else:
                        error_text = await response.text()
                        return {
                            "data": {"medications": [], "total_count": 0},
                            "metadata": None,
                            "success": False,
                            "error": f"FHIR Store error: HTTP {response.status} - {error_text}"
                        }
                        
        except Exception as e:
            logger.error(f"❌ Error fetching patient medications: {e}")
            return {
                "data": {"medications": [], "total_count": 0},
                "metadata": None,
                "success": False,
                "error": str(e)
            }
    
    async def fetch_patient_observations(self, patient_id: str, data_point: DataPoint, category: str = None) -> Dict[str, Any]:
        """Fetch patient observations (labs, vitals) from FHIR Store"""
        try:
            if not self.connection_healthy:
                await self.initialize()
            
            await self._ensure_valid_token()
            
            # FHIR Observation search
            url = f"{self.base_url}/Observation"
            params = {
                "patient": patient_id,
                "_count": "100",
                "_sort": "-date"
            }
            
            # Add category filter if specified
            if category:
                params["category"] = category
            
            headers = {
                "Authorization": f"Bearer {self.access_token}",
                "Content-Type": "application/fhir+json"
            }
            
            async with aiohttp.ClientSession() as session:
                async with session.get(url, headers=headers, params=params) as response:
                    if response.status == 200:
                        bundle = await response.json()
                        
                        # Extract observations from FHIR Bundle
                        observations = self._extract_observations_from_bundle(bundle)
                        
                        source_metadata = SourceMetadata(
                            source_type=DataSourceType.FHIR_STORE,
                            source_endpoint=self.base_url,
                            retrieved_at=datetime.utcnow(),
                            data_version="FHIR R4",
                            completeness=1.0 if observations else 0.0,
                            response_time_ms=0.0,
                            cache_hit=False
                        )
                        
                        return {
                            "data": {
                                "observations": observations,
                                "total_count": len(observations),
                                "category": category or "all"
                            },
                            "metadata": source_metadata,
                            "success": True
                        }
                    else:
                        error_text = await response.text()
                        return {
                            "data": {"observations": [], "total_count": 0},
                            "metadata": None,
                            "success": False,
                            "error": f"FHIR Store error: HTTP {response.status} - {error_text}"
                        }
                        
        except Exception as e:
            logger.error(f"❌ Error fetching patient observations: {e}")
            return {
                "data": {"observations": [], "total_count": 0},
                "metadata": None,
                "success": False,
                "error": str(e)
            }
    
    async def search_patient_resources(self, patient_id: str, resource_type: str, search_params: Dict[str, str] = None) -> Dict[str, Any]:
        """Generic search for patient resources in FHIR Store"""
        try:
            if not self.connection_healthy:
                await self.initialize()
            
            await self._ensure_valid_token()
            
            # Build search URL
            url = f"{self.base_url}/{resource_type}"
            params = {"patient": patient_id}
            
            if search_params:
                params.update(search_params)
            
            headers = {
                "Authorization": f"Bearer {self.access_token}",
                "Content-Type": "application/fhir+json"
            }
            
            async with aiohttp.ClientSession() as session:
                async with session.get(url, headers=headers, params=params) as response:
                    if response.status == 200:
                        bundle = await response.json()
                        
                        # Extract resources from bundle
                        resources = []
                        if bundle.get("entry"):
                            resources = [entry["resource"] for entry in bundle["entry"]]
                        
                        return {
                            "data": {
                                "resources": resources,
                                "total_count": bundle.get("total", len(resources)),
                                "resource_type": resource_type
                            },
                            "success": True
                        }
                    else:
                        error_text = await response.text()
                        return {
                            "data": {"resources": [], "total_count": 0},
                            "success": False,
                            "error": f"FHIR Store error: HTTP {response.status} - {error_text}"
                        }
                        
        except Exception as e:
            logger.error(f"❌ Error searching FHIR resources: {e}")
            return {
                "data": {"resources": [], "total_count": 0},
                "success": False,
                "error": str(e)
            }
    
    async def get_connection_health(self) -> Dict[str, Any]:
        """Check FHIR Store connection health"""
        try:
            if not self.connection_healthy:
                return {"healthy": False, "error": "Connection not initialized"}
            
            await self._ensure_valid_token()
            
            # Test with metadata endpoint
            url = f"{self.base_url}/metadata"
            headers = {
                "Authorization": f"Bearer {self.access_token}",
                "Content-Type": "application/fhir+json"
            }
            
            async with aiohttp.ClientSession() as session:
                async with session.get(url, headers=headers) as response:
                    if response.status == 200:
                        metadata = await response.json()
                        return {
                            "healthy": True,
                            "fhir_version": metadata.get("fhirVersion", "Unknown"),
                            "software": metadata.get("software", {}).get("name", "Unknown"),
                            "project": self.project_id,
                            "dataset": self.dataset_id,
                            "fhir_store": self.fhir_store_id
                        }
                    else:
                        return {
                            "healthy": False,
                            "error": f"HTTP {response.status}"
                        }
                        
        except Exception as e:
            return {
                "healthy": False,
                "error": str(e)
            }
    
    async def _ensure_valid_token(self):
        """Ensure access token is valid"""
        if not self.access_token or (self.token_expiry and datetime.utcnow() >= self.token_expiry):
            await self._refresh_access_token()
    
    def _extract_patient_demographics(self, patient_resource: Dict[str, Any]) -> Dict[str, Any]:
        """Extract demographics from FHIR Patient resource"""
        demographics = {
            "patient_id": patient_resource.get("id"),
            "resource_type": "Patient"
        }
        
        # Name
        if patient_resource.get("name"):
            name = patient_resource["name"][0]
            demographics["family_name"] = name.get("family")
            demographics["given_names"] = name.get("given", [])
            demographics["full_name"] = f"{' '.join(name.get('given', []))} {name.get('family', '')}"
        
        # Gender
        demographics["gender"] = patient_resource.get("gender")
        
        # Birth date
        demographics["birth_date"] = patient_resource.get("birthDate")
        
        # Calculate age if birth date available
        if demographics["birth_date"]:
            try:
                birth_date = datetime.strptime(demographics["birth_date"], "%Y-%m-%d")
                age = (datetime.now() - birth_date).days // 365
                demographics["age"] = age
            except:
                pass
        
        # Contact info
        if patient_resource.get("telecom"):
            for contact in patient_resource["telecom"]:
                if contact.get("system") == "phone":
                    demographics["phone"] = contact.get("value")
                elif contact.get("system") == "email":
                    demographics["email"] = contact.get("value")
        
        # Address
        if patient_resource.get("address"):
            address = patient_resource["address"][0]
            demographics["address"] = {
                "line": address.get("line", []),
                "city": address.get("city"),
                "state": address.get("state"),
                "postal_code": address.get("postalCode"),
                "country": address.get("country")
            }
        
        return demographics
    
    def _extract_medications_from_bundle(self, bundle: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Extract medications from FHIR Bundle"""
        medications = []
        
        if bundle.get("entry"):
            for entry in bundle["entry"]:
                resource = entry["resource"]
                
                medication = {
                    "id": resource.get("id"),
                    "status": resource.get("status"),
                    "intent": resource.get("intent"),
                    "authored_on": resource.get("authoredOn")
                }
                
                # Medication reference or coding
                if resource.get("medicationCodeableConcept"):
                    coding = resource["medicationCodeableConcept"].get("coding", [])
                    if coding:
                        medication["medication_name"] = coding[0].get("display")
                        medication["medication_code"] = coding[0].get("code")
                        medication["medication_system"] = coding[0].get("system")
                
                # Dosage instructions
                if resource.get("dosageInstruction"):
                    dosage = resource["dosageInstruction"][0]
                    medication["dosage_text"] = dosage.get("text")
                    
                    if dosage.get("doseAndRate"):
                        dose_rate = dosage["doseAndRate"][0]
                        if dose_rate.get("doseQuantity"):
                            dose_qty = dose_rate["doseQuantity"]
                            medication["dose_value"] = dose_qty.get("value")
                            medication["dose_unit"] = dose_qty.get("unit")
                
                medications.append(medication)
        
        return medications
    
    def _extract_observations_from_bundle(self, bundle: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Extract observations from FHIR Bundle"""
        observations = []
        
        if bundle.get("entry"):
            for entry in bundle["entry"]:
                resource = entry["resource"]
                
                observation = {
                    "id": resource.get("id"),
                    "status": resource.get("status"),
                    "effective_date": resource.get("effectiveDateTime"),
                    "issued": resource.get("issued")
                }
                
                # Category
                if resource.get("category"):
                    category = resource["category"][0]
                    if category.get("coding"):
                        observation["category"] = category["coding"][0].get("display")
                
                # Code (what was observed)
                if resource.get("code"):
                    coding = resource["code"].get("coding", [])
                    if coding:
                        observation["code_display"] = coding[0].get("display")
                        observation["code_code"] = coding[0].get("code")
                        observation["code_system"] = coding[0].get("system")
                
                # Value
                if resource.get("valueQuantity"):
                    value_qty = resource["valueQuantity"]
                    observation["value"] = value_qty.get("value")
                    observation["unit"] = value_qty.get("unit")
                elif resource.get("valueString"):
                    observation["value"] = resource["valueString"]
                elif resource.get("valueCodeableConcept"):
                    value_concept = resource["valueCodeableConcept"]
                    if value_concept.get("coding"):
                        observation["value"] = value_concept["coding"][0].get("display")
                
                # Reference range
                if resource.get("referenceRange"):
                    ref_range = resource["referenceRange"][0]
                    observation["reference_range"] = {
                        "low": ref_range.get("low", {}).get("value"),
                        "high": ref_range.get("high", {}).get("value"),
                        "unit": ref_range.get("low", {}).get("unit")
                    }
                
                observations.append(observation)
        
        return observations
    
    def _calculate_completeness(self, data: Dict[str, Any], required_fields: List[str]) -> float:
        """Calculate data completeness score"""
        if not required_fields:
            return 1.0
        
        present_fields = sum(1 for field in required_fields if data.get(field) is not None)
        return present_fields / len(required_fields)
    
    async def close(self):
        """Close FHIR Store connection"""
        self.connection_healthy = False
        logger.info("FHIR Store connection closed")
