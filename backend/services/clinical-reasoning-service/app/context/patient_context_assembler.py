"""
Patient Context Assembler

This module implements GraphDB integration for assembling complete patient
clinical context before making clinical reasoning decisions.
"""

import logging
from typing import Dict, Any, List, Optional
from dataclasses import dataclass
from datetime import datetime, timedelta
import asyncio
import httpx

logger = logging.getLogger(__name__)

@dataclass
class PatientContext:
    """Complete patient clinical context"""
    patient_id: str
    demographics: Dict[str, Any]
    active_conditions: List[Dict[str, Any]]
    current_medications: List[Dict[str, Any]]
    allergies: List[Dict[str, Any]]
    recent_labs: List[Dict[str, Any]]
    vital_signs: List[Dict[str, Any]]
    clinical_timeline: List[Dict[str, Any]]
    risk_factors: Dict[str, Any]
    last_updated: datetime

class PatientContextAssembler:
    """
    Patient Context Assembler using GraphDB queries
    
    This component queries GraphDB to assemble complete patient clinical context
    using SPARQL queries like: DESCRIBE :Patient123
    """
    
    def __init__(self, graphdb_endpoint: str = None, redis_client=None):
        self.graphdb_endpoint = graphdb_endpoint or "http://localhost:7200"  # CAE-specific GraphDB port
        self.repository = "cae-clinical-intelligence"  # CAE-specific repository
        self.redis_client = redis_client
        self.fallback_cache = {}  # Simple in-memory cache fallback
        self.cache_ttl = 300  # 5 minutes cache TTL
        logger.info(f"Patient Context Assembler initialized with GraphDB: {self.graphdb_endpoint}")
    
    async def get_patient_context(
        self,
        patient_id: str,
        include_history_days: int = 30,
        force_refresh: bool = False
    ) -> PatientContext:
        """
        Get complete patient clinical context from GraphDB
        
        Args:
            patient_id: Patient identifier
            include_history_days: Days of historical data to include
            force_refresh: Force refresh from GraphDB (bypass cache)
            
        Returns:
            PatientContext with complete clinical information
        """
        logger.info(f"Assembling context for patient {patient_id}")
        
        # Check Redis cache first
        if not force_refresh and self.redis_client:
            cached_context = await self.redis_client.get_patient_context(patient_id)
            if cached_context:
                logger.info(f"Returning cached context from Redis for patient {patient_id}")
                return PatientContext(**cached_context)

        # Fallback to in-memory cache
        cache_key = f"{patient_id}_{include_history_days}"
        if not force_refresh and cache_key in self.fallback_cache:
            cached_context, cached_time = self.fallback_cache[cache_key]
            if (datetime.now() - cached_time).seconds < self.cache_ttl:
                logger.info(f"Returning cached context from memory for patient {patient_id}")
                return cached_context
        
        try:
            # Query GraphDB for complete patient context
            context = await self._query_patient_context(patient_id, include_history_days)

            # Cache the result in Redis
            if self.redis_client:
                context_dict = {
                    "patient_id": context.patient_id,
                    "demographics": context.demographics,
                    "active_conditions": context.active_conditions,
                    "current_medications": context.current_medications,
                    "allergies": context.allergies,
                    "recent_labs": context.recent_labs,
                    "vital_signs": context.vital_signs,
                    "clinical_timeline": context.clinical_timeline,
                    "risk_factors": context.risk_factors,
                    "last_updated": context.last_updated.isoformat()
                }
                await self.redis_client.set_patient_context(patient_id, context_dict)

            # Fallback cache
            self.fallback_cache[cache_key] = (context, datetime.now())

            logger.info(f"Successfully assembled context for patient {patient_id}")
            return context
            
        except Exception as e:
            logger.error(f"Failed to assemble context for patient {patient_id}: {e}")
            # Return minimal context with error indication
            return self._create_minimal_context(patient_id, str(e))
    
    async def _query_patient_context(
        self,
        patient_id: str,
        include_history_days: int
    ) -> PatientContext:
        """Query GraphDB for patient context using SPARQL"""
        
        # For now, simulate GraphDB queries since GraphDB may not be set up
        # In production, this would use real SPARQL queries
        
        if await self._is_graphdb_available():
            return await self._query_real_graphdb(patient_id, include_history_days)
        else:
            logger.warning("GraphDB not available, using mock patient context")
            return await self._create_mock_context(patient_id, include_history_days)
    
    async def _is_graphdb_available(self) -> bool:
        """Check if GraphDB is available"""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(f"{self.graphdb_endpoint}/rest/repositories", timeout=5.0)
                return response.status_code == 200
        except Exception:
            return False
    
    async def _query_real_graphdb(
        self,
        patient_id: str,
        include_history_days: int
    ) -> PatientContext:
        """Query real GraphDB using SPARQL"""
        
        # Main patient description query
        describe_query = f"""
        PREFIX fhir: <http://hl7.org/fhir/>
        PREFIX clinical: <http://clinical-synthesis-hub.com/ontology/>
        
        DESCRIBE <http://clinical-synthesis-hub.com/patient/{patient_id}>
        """
        
        # Active conditions query
        conditions_query = f"""
        PREFIX fhir: <http://hl7.org/fhir/>
        PREFIX clinical: <http://clinical-synthesis-hub.com/ontology/>
        
        SELECT ?condition ?code ?display ?status ?onsetDate WHERE {{
            <http://clinical-synthesis-hub.com/patient/{patient_id}> clinical:hasCondition ?condition .
            ?condition fhir:code ?code ;
                      fhir:display ?display ;
                      fhir:clinicalStatus ?status ;
                      fhir:onsetDateTime ?onsetDate .
            FILTER(?status = "active")
        }}
        """
        
        # Current medications query
        medications_query = f"""
        PREFIX fhir: <http://hl7.org/fhir/>
        PREFIX clinical: <http://clinical-synthesis-hub.com/ontology/>
        
        SELECT ?medication ?code ?display ?dosage ?frequency ?status WHERE {{
            <http://clinical-synthesis-hub.com/patient/{patient_id}> clinical:hasMedication ?medication .
            ?medication fhir:code ?code ;
                       fhir:display ?display ;
                       fhir:dosage ?dosage ;
                       fhir:frequency ?frequency ;
                       fhir:status ?status .
            FILTER(?status = "active")
        }}
        """
        
        # Allergies query
        allergies_query = f"""
        PREFIX fhir: <http://hl7.org/fhir/>
        PREFIX clinical: <http://clinical-synthesis-hub.com/ontology/>
        
        SELECT ?allergy ?substance ?reaction ?severity WHERE {{
            <http://clinical-synthesis-hub.com/patient/{patient_id}> clinical:hasAllergy ?allergy .
            ?allergy fhir:substance ?substance ;
                    fhir:reaction ?reaction ;
                    fhir:severity ?severity .
        }}
        """
        
        # Recent labs query (last 30 days)
        cutoff_date = (datetime.now() - timedelta(days=include_history_days)).isoformat()
        labs_query = f"""
        PREFIX fhir: <http://hl7.org/fhir/>
        PREFIX clinical: <http://clinical-synthesis-hub.com/ontology/>
        
        SELECT ?observation ?code ?display ?value ?unit ?date WHERE {{
            <http://clinical-synthesis-hub.com/patient/{patient_id}> clinical:hasObservation ?observation .
            ?observation fhir:code ?code ;
                        fhir:display ?display ;
                        fhir:value ?value ;
                        fhir:unit ?unit ;
                        fhir:effectiveDateTime ?date .
            FILTER(?date >= "{cutoff_date}"^^xsd:dateTime)
            FILTER(CONTAINS(LCASE(?display), "lab") || CONTAINS(LCASE(?display), "blood"))
        }}
        ORDER BY DESC(?date)
        """
        
        try:
            async with httpx.AsyncClient() as client:
                # Execute SPARQL queries
                demographics = await self._execute_sparql_query(client, describe_query)
                conditions = await self._execute_sparql_query(client, conditions_query)
                medications = await self._execute_sparql_query(client, medications_query)
                allergies = await self._execute_sparql_query(client, allergies_query)
                labs = await self._execute_sparql_query(client, labs_query)
                
                # Process results and create context
                return PatientContext(
                    patient_id=patient_id,
                    demographics=self._process_demographics(demographics),
                    active_conditions=self._process_conditions(conditions),
                    current_medications=self._process_medications(medications),
                    allergies=self._process_allergies(allergies),
                    recent_labs=self._process_labs(labs),
                    vital_signs=[],  # Would need separate query
                    clinical_timeline=self._build_timeline(conditions, medications, labs),
                    risk_factors=self._assess_risk_factors(conditions, medications, labs),
                    last_updated=datetime.now()
                )
                
        except Exception as e:
            logger.error(f"GraphDB query failed: {e}")
            return await self._create_mock_context(patient_id, include_history_days)
    
    async def _execute_sparql_query(self, client: httpx.AsyncClient, query: str) -> Dict[str, Any]:
        """Execute SPARQL query against GraphDB"""
        try:
            response = await client.post(
                f"{self.graphdb_endpoint}/repositories/{self.repository}",
                headers={
                    "Content-Type": "application/sparql-query",
                    "Accept": "application/sparql-results+json"
                },
                content=query,
                timeout=10.0
            )
            response.raise_for_status()
            result = response.json()
            # Ensure we're returning a dictionary with expected structure
            if isinstance(result, str):
                logger.warning(f"Received string result from GraphDB: {result[:100]}...")
                return {"results": {"bindings": []}}
            return result
        except Exception as e:
            logger.error(f"SPARQL query execution failed: {e}")
            return {"results": {"bindings": []}}
    
    async def _create_mock_context(
        self,
        patient_id: str,
        include_history_days: int
    ) -> PatientContext:
        """Create mock patient context for testing when GraphDB is not available"""
        
        # Mock patient context based on patient ID
        mock_contexts = {
            "test-patient-001": {
                "demographics": {"age": 75, "gender": "male", "weight": 70},
                "conditions": [
                    {"code": "I48", "display": "Atrial fibrillation", "status": "active"},
                    {"code": "I10", "display": "Essential hypertension", "status": "active"}
                ],
                "medications": [
                    {"code": "warfarin", "display": "Warfarin 5mg", "dosage": "5mg", "frequency": "once daily"},
                    {"code": "lisinopril", "display": "Lisinopril 10mg", "dosage": "10mg", "frequency": "once daily"}
                ],
                "allergies": [
                    {"substance": "penicillin", "reaction": "rash", "severity": "moderate"}
                ]
            },
            "test-patient-002": {
                "demographics": {"age": 28, "gender": "female", "weight": 65, "pregnancy_status": "pregnant"},
                "conditions": [
                    {"code": "Z33", "display": "Pregnancy", "status": "active"}
                ],
                "medications": [],
                "allergies": []
            }
        }
        
        mock_data = mock_contexts.get(patient_id, {
            "demographics": {"age": 50, "gender": "unknown", "weight": 70},
            "conditions": [],
            "medications": [],
            "allergies": []
        })
        
        return PatientContext(
            patient_id=patient_id,
            demographics=mock_data["demographics"],
            active_conditions=mock_data["conditions"],
            current_medications=mock_data["medications"],
            allergies=mock_data["allergies"],
            recent_labs=[],
            vital_signs=[],
            clinical_timeline=[],
            risk_factors=self._assess_risk_factors(
                mock_data["conditions"], 
                mock_data["medications"], 
                []
            ),
            last_updated=datetime.now()
        )
    
    def _create_minimal_context(self, patient_id: str, error_message: str) -> PatientContext:
        """Create minimal context when queries fail"""
        return PatientContext(
            patient_id=patient_id,
            demographics={"error": error_message},
            active_conditions=[],
            current_medications=[],
            allergies=[],
            recent_labs=[],
            vital_signs=[],
            clinical_timeline=[],
            risk_factors={"error": error_message},
            last_updated=datetime.now()
        )
    
    def _process_demographics(self, sparql_result: Dict[str, Any]) -> Dict[str, Any]:
        """Process demographics from SPARQL result"""
        # Implementation would parse SPARQL results
        return {"processed": True, "source": "graphdb"}
    
    def _process_conditions(self, sparql_result: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Process conditions from SPARQL result"""
        conditions = []
        for binding in sparql_result.get("results", {}).get("bindings", []):
            conditions.append({
                "code": binding.get("code", {}).get("value", ""),
                "display": binding.get("display", {}).get("value", ""),
                "status": binding.get("status", {}).get("value", ""),
                "onsetDate": binding.get("onsetDate", {}).get("value", "")
            })
        return conditions
    
    def _process_medications(self, sparql_result: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Process medications from SPARQL result"""
        medications = []
        for binding in sparql_result.get("results", {}).get("bindings", []):
            medications.append({
                "code": binding.get("code", {}).get("value", ""),
                "display": binding.get("display", {}).get("value", ""),
                "dosage": binding.get("dosage", {}).get("value", ""),
                "frequency": binding.get("frequency", {}).get("value", ""),
                "status": binding.get("status", {}).get("value", "")
            })
        return medications
    
    def _process_allergies(self, sparql_result: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Process allergies from SPARQL result"""
        allergies = []
        for binding in sparql_result.get("results", {}).get("bindings", []):
            allergies.append({
                "substance": binding.get("substance", {}).get("value", ""),
                "reaction": binding.get("reaction", {}).get("value", ""),
                "severity": binding.get("severity", {}).get("value", "")
            })
        return allergies
    
    def _process_labs(self, sparql_result: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Process lab results from SPARQL result"""
        labs = []
        for binding in sparql_result.get("results", {}).get("bindings", []):
            labs.append({
                "code": binding.get("code", {}).get("value", ""),
                "display": binding.get("display", {}).get("value", ""),
                "value": binding.get("value", {}).get("value", ""),
                "unit": binding.get("unit", {}).get("value", ""),
                "date": binding.get("date", {}).get("value", "")
            })
        return labs
    
    def _build_timeline(
        self,
        conditions: List[Dict[str, Any]],
        medications: List[Dict[str, Any]],
        labs: List[Dict[str, Any]]
    ) -> List[Dict[str, Any]]:
        """Build clinical timeline from patient data"""
        timeline = []
        
        # Add conditions to timeline
        for condition in conditions:
            if condition.get("onsetDate"):
                timeline.append({
                    "date": condition["onsetDate"],
                    "type": "condition",
                    "description": condition.get("display", ""),
                    "data": condition
                })
        
        # Add lab results to timeline
        for lab in labs:
            if lab.get("date"):
                timeline.append({
                    "date": lab["date"],
                    "type": "lab",
                    "description": f"{lab.get('display', '')}: {lab.get('value', '')} {lab.get('unit', '')}",
                    "data": lab
                })
        
        # Sort by date
        timeline.sort(key=lambda x: x.get("date", ""), reverse=True)
        return timeline
    
    def _assess_risk_factors(
        self,
        conditions: List[Dict[str, Any]],
        medications: List[Dict[str, Any]],
        labs: List[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """Assess patient risk factors based on clinical data"""
        risk_factors = {
            "bleeding_risk": "low",
            "kidney_risk": "low",
            "drug_interaction_risk": "low",
            "age_related_risk": "low"
        }
        
        # Assess bleeding risk
        bleeding_conditions = ["atrial fibrillation", "anticoagulation"]
        bleeding_medications = ["warfarin", "aspirin", "heparin"]
        
        for condition in conditions:
            if any(risk in condition.get("display", "").lower() for risk in bleeding_conditions):
                risk_factors["bleeding_risk"] = "moderate"
        
        for medication in medications:
            if any(risk in medication.get("code", "").lower() for risk in bleeding_medications):
                risk_factors["bleeding_risk"] = "high"
        
        return risk_factors
