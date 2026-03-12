"""
Apollo Federation GraphQL Client for Context Service.

This client allows the Context Service to query the Apollo Federation Gateway
to get data from microservices, following the correct architecture:

Context Service → Apollo Federation → Microservices
"""

import aiohttp
import json
import logging
from typing import Dict, Any, Optional, List
from datetime import datetime

logger = logging.getLogger(__name__)


class ApolloFederationClient:
    """
    GraphQL client for querying Apollo Federation Gateway from Context Service.
    
    This implements the correct architecture where Context Service acts as a consumer
    of Apollo Federation, not a provider to it.
    """
    
    def __init__(self, apollo_federation_url: str = "http://localhost:4000/graphql"):
        self.apollo_federation_url = apollo_federation_url
        self.session = None
        
    async def __aenter__(self):
        """Create HTTP session for GraphQL requests."""
        self.session = aiohttp.ClientSession()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Close HTTP session."""
        if self.session:
            await self.session.close()
    
    async def get_patient_data(self, patient_id: str) -> Dict[str, Any]:
        """
        Get patient data from Patient Service via Apollo Federation.
        
        Args:
            patient_id: The patient ID
            
        Returns:
            Patient data from Patient Service
        """
        query = """
        query GetPatientData($patientId: ID!) {
            patient(id: $patientId) {
                id
                resourceType
                name {
                    given
                    family
                    use
                }
                gender
                birthDate
                address {
                    line
                    city
                    state
                    postalCode
                    country
                }
                telecom {
                    system
                    value
                    use
                }
            }
        }
        """
        
        variables = {"patientId": patient_id}
        
        try:
            logger.info(f"🔍 Querying Apollo Federation for patient data: {patient_id}")
            result = await self._execute_query(query, variables)
            
            if result and "patient" in result:
                logger.info(f"✅ Successfully retrieved patient data via Apollo Federation")
                return result["patient"]
            else:
                logger.warning(f"⚠️ No patient data found for ID: {patient_id}")
                return {}
                
        except Exception as e:
            logger.error(f"❌ Failed to get patient data via Apollo Federation: {e}")
            return {}
    
    async def get_patient_medications(self, patient_id: str) -> List[Dict[str, Any]]:
        """
        Get patient medications from Medication Service via Apollo Federation.

        Args:
            patient_id: The patient ID

        Returns:
            List of medications from Medication Service
        """
        query = """
        query GetPatientMedications($patientId: String!) {
            medications(patientId: $patientId) {
                id
                status
                medicationCodeableConcept {
                    coding {
                        system
                        code
                        display
                    }
                    text
                }
                dosageInstruction {
                    text
                }
                effectiveDateTime
            }
        }
        """
        
        variables = {"patientId": patient_id}
        
        try:
            logger.info(f"🔍 Querying Apollo Federation for patient medications: {patient_id}")
            result = await self._execute_query(query, variables)
            
            if result and "medications" in result:
                medications = result["medications"]
                logger.info(f"✅ Successfully retrieved {len(medications)} medications via Apollo Federation")
                return medications
            else:
                logger.warning(f"⚠️ No medications found for patient ID: {patient_id}")
                return []
                
        except Exception as e:
            logger.error(f"❌ Failed to get patient medications via Apollo Federation: {e}")
            return []
    
    async def get_patient_conditions(self, patient_id: str) -> List[Dict[str, Any]]:
        """
        Get patient conditions from Condition Service via Apollo Federation.
        
        Args:
            patient_id: The patient ID
            
        Returns:
            List of conditions from Condition Service
        """
        query = """
        query GetPatientConditions($patientId: String!) {
            conditionsByPatient(patientId: $patientId) {
                id
                clinicalStatus
                verificationStatus
                code {
                    coding {
                        system
                        code
                        display
                    }
                    text
                }
                onsetDateTime
                recordedDate
            }
        }
        """
        
        variables = {"patientId": patient_id}
        
        try:
            logger.info(f"🔍 Querying Apollo Federation for patient conditions: {patient_id}")
            result = await self._execute_query(query, variables)
            
            if result and "conditionsByPatient" in result:
                conditions = result["conditionsByPatient"]
                logger.info(f"✅ Successfully retrieved {len(conditions)} conditions via Apollo Federation")
                return conditions
            else:
                logger.warning(f"⚠️ No conditions found for patient ID: {patient_id}")
                return []
                
        except Exception as e:
            logger.error(f"❌ Failed to get patient conditions via Apollo Federation: {e}")
            return []
    
    async def get_patient_observations(self, patient_id: str) -> List[Dict[str, Any]]:
        """
        Get patient observations from Observation Service via Apollo Federation.
        
        Args:
            patient_id: The patient ID
            
        Returns:
            List of observations from Observation Service
        """
        query = """
        query GetPatientObservations($patientId: String!) {
            observationsByPatient(patientId: $patientId) {
                id
                status
                code {
                    coding {
                        system
                        code
                        display
                    }
                    text
                }
                valueQuantity {
                    value
                    unit
                    system
                }
                effectiveDateTime
                issued
            }
        }
        """
        
        variables = {"patientId": patient_id}
        
        try:
            logger.info(f"🔍 Querying Apollo Federation for patient observations: {patient_id}")
            result = await self._execute_query(query, variables)
            
            if result and "observationsByPatient" in result:
                observations = result["observationsByPatient"]
                logger.info(f"✅ Successfully retrieved {len(observations)} observations via Apollo Federation")
                return observations
            else:
                logger.warning(f"⚠️ No observations found for patient ID: {patient_id}")
                return []
                
        except Exception as e:
            logger.error(f"❌ Failed to get patient observations via Apollo Federation: {e}")
            return []
    
    async def _execute_query(self, query: str, variables: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Execute a GraphQL query against Apollo Federation.
        
        Args:
            query: The GraphQL query string
            variables: Query variables
            
        Returns:
            Query result data or None if failed
        """
        if not self.session:
            raise RuntimeError("HTTP session not initialized. Use 'async with' context manager.")
        
        payload = {
            "query": query,
            "variables": variables
        }
        
        try:
            logger.debug(f"🔗 Executing GraphQL query to Apollo Federation: {self.apollo_federation_url}")
            
            async with self.session.post(
                self.apollo_federation_url,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=aiohttp.ClientTimeout(total=30)
            ) as response:
                
                if response.status != 200:
                    logger.error(f"❌ Apollo Federation returned status {response.status}")
                    return None
                
                result = await response.json()
                
                if "errors" in result:
                    logger.error(f"❌ GraphQL errors from Apollo Federation: {result['errors']}")
                    return None
                
                if "data" not in result:
                    logger.error(f"❌ No data in Apollo Federation response")
                    return None
                
                logger.debug(f"✅ Successfully executed GraphQL query via Apollo Federation")
                return result["data"]
                
        except aiohttp.ClientTimeout:
            logger.error(f"❌ Timeout querying Apollo Federation")
            return None
        except Exception as e:
            logger.error(f"❌ Error executing GraphQL query: {e}")
            return None


# Singleton instance for use across the Context Service
_apollo_client_instance = None

def get_apollo_federation_client() -> ApolloFederationClient:
    """Get or create singleton Apollo Federation client."""
    global _apollo_client_instance
    if _apollo_client_instance is None:
        _apollo_client_instance = ApolloFederationClient()
    return _apollo_client_instance
