"""
Clinical Context Assembler

Assembles comprehensive clinical context for events by integrating
data from multiple sources including EHR, FHIR stores, and real-time
clinical systems to provide rich contextual information.
"""

import logging
import asyncio
from datetime import datetime, timezone, timedelta
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, field
from enum import Enum

from .clinical_event_envelope import (
    ClinicalEventEnvelope, ClinicalContext, TemporalContext, 
    ProvenanceContext, EventMetadata, EventType
)

logger = logging.getLogger(__name__)


class ContextSource(Enum):
    """Sources of clinical context data"""
    FHIR_STORE = "fhir_store"
    EHR_SYSTEM = "ehr_system"
    REAL_TIME_MONITORING = "real_time_monitoring"
    CLINICAL_DECISION_SUPPORT = "clinical_decision_support"
    LABORATORY_SYSTEM = "laboratory_system"
    PHARMACY_SYSTEM = "pharmacy_system"
    CACHE = "cache"


class EnrichmentLevel(Enum):
    """Levels of context enrichment"""
    MINIMAL = "minimal"          # Basic patient and encounter info
    STANDARD = "standard"        # Standard clinical context
    COMPREHENSIVE = "comprehensive"  # Full clinical context
    REAL_TIME = "real_time"     # Real-time enrichment with latest data


@dataclass
class ContextEnrichmentConfig:
    """Configuration for context enrichment"""
    enrichment_level: EnrichmentLevel = EnrichmentLevel.STANDARD
    include_historical_data: bool = True
    historical_lookback_days: int = 30
    include_real_time_vitals: bool = False
    include_active_orders: bool = True
    include_recent_labs: bool = True
    recent_labs_hours: int = 72
    cache_ttl_minutes: int = 15
    max_enrichment_time_ms: int = 500


class ClinicalContextAssembler:
    """
    Assembles comprehensive clinical context for events
    
    Integrates data from multiple clinical systems to provide
    rich contextual information for clinical event processing.
    """
    
    def __init__(self, config: Optional[ContextEnrichmentConfig] = None):
        self.config = config or ContextEnrichmentConfig()
        
        # Context cache for performance
        self.context_cache: Dict[str, Tuple[ClinicalContext, datetime]] = {}
        
        # Data source clients (would be injected in production)
        self.graphdb_client = None  # Injected dependency
        self.fhir_client = None
        self.ehr_client = None
        self.lab_client = None
        self.pharmacy_client = None
        
        # Enrichment statistics
        self.enrichment_stats = {
            "total_enrichments": 0,
            "cache_hits": 0,
            "cache_misses": 0,
            "average_enrichment_time_ms": 0.0,
            "source_usage": {source.value: 0 for source in ContextSource}
        }
        
        logger.info("Clinical Context Assembler initialized")
    
    async def enrich_clinical_context(self, envelope: ClinicalEventEnvelope) -> ClinicalEventEnvelope:
        """
        Enrich clinical context of event envelope
        
        Args:
            envelope: Event envelope to enrich
            
        Returns:
            Enriched event envelope
        """
        start_time = datetime.now()
        
        try:
            # Check cache first
            cache_key = self._generate_cache_key(envelope)
            cached_context = await self._get_cached_context(cache_key)
            
            if cached_context:
                envelope.clinical_context = cached_context
                self.enrichment_stats["cache_hits"] += 1
                self.enrichment_stats["source_usage"][ContextSource.CACHE.value] += 1
                
                logger.debug(f"Used cached context for {envelope.metadata.event_id}")
            else:
                # Perform enrichment
                enriched_context = await self._perform_enrichment(envelope.clinical_context)
                envelope.clinical_context = enriched_context
                
                # Cache the enriched context
                await self._cache_context(cache_key, enriched_context)
                self.enrichment_stats["cache_misses"] += 1
            
            # Add enrichment provenance
            envelope.add_provenance_entry(
                "clinical_context_assembler",
                f"context_enrichment_{self.config.enrichment_level.value}",
                confidence=0.95
            )
            
            # Update statistics
            enrichment_time = (datetime.now() - start_time).total_seconds() * 1000
            await self._update_enrichment_stats(enrichment_time)
            
            logger.debug(f"Enriched clinical context for {envelope.metadata.event_id} "
                        f"in {enrichment_time:.2f}ms")
            
            return envelope
            
        except Exception as e:
            logger.error(f"Error enriching clinical context: {e}")
            # Return original envelope if enrichment fails
            return envelope
    
    async def _perform_enrichment(self, base_context) -> ClinicalContext:
        """Perform clinical context enrichment"""
        # Handle both dictionary and object types for base_context
        if isinstance(base_context, dict):
            # Create a ClinicalContext from dictionary
            patient_id = base_context.get('patient_id', 'unknown_patient')
            enriched_context = ClinicalContext(
                patient_id=patient_id,
                patient_mrn=base_context.get('patient_mrn'),
                patient_demographics=base_context.get('patient_demographics', {}).copy() if isinstance(base_context.get('patient_demographics'), dict) else {},
                encounter_id=base_context.get('encounter_id'),
                encounter_type=base_context.get('encounter_type'),
                encounter_status=base_context.get('encounter_status'),
                admission_date=base_context.get('admission_date'),
                discharge_date=base_context.get('discharge_date'),
                primary_provider_id=base_context.get('primary_provider_id'),
                attending_physician_id=base_context.get('attending_physician_id'),
                care_team_members=base_context.get('care_team_members', []).copy() if isinstance(base_context.get('care_team_members'), list) else [],
                facility_id=base_context.get('facility_id'),
                facility_name=base_context.get('facility_name'),
                department_id=base_context.get('department_id'),
                department_name=base_context.get('department_name'),
                unit_id=base_context.get('unit_id'),
                unit_name=base_context.get('unit_name'),
                active_diagnoses=base_context.get('active_diagnoses', []).copy() if isinstance(base_context.get('active_diagnoses'), list) else [],
                active_medications=base_context.get('active_medications', []).copy() if isinstance(base_context.get('active_medications'), list) else [],
                active_allergies=base_context.get('active_allergies', []).copy() if isinstance(base_context.get('active_allergies'), list) else [],
                vital_signs=base_context.get('vital_signs', {}).copy() if isinstance(base_context.get('vital_signs'), dict) else {},
                laboratory_values=base_context.get('laboratory_values', {}).copy() if isinstance(base_context.get('laboratory_values'), dict) else {},
                risk_factors=base_context.get('risk_factors', []).copy() if isinstance(base_context.get('risk_factors'), list) else [],
                clinical_warnings=base_context.get('clinical_warnings', []).copy() if isinstance(base_context.get('clinical_warnings'), list) else []
            )
        else:
            # Use the object attributes directly
            enriched_context = ClinicalContext(
                patient_id=base_context.patient_id,
                patient_mrn=base_context.patient_mrn,
                patient_demographics=base_context.patient_demographics.copy(),
                encounter_id=base_context.encounter_id,
                encounter_type=base_context.encounter_type,
                encounter_status=base_context.encounter_status,
                admission_date=base_context.admission_date,
                discharge_date=base_context.discharge_date,
                primary_provider_id=base_context.primary_provider_id,
                attending_physician_id=base_context.attending_physician_id,
                care_team_members=base_context.care_team_members.copy(),
                facility_id=base_context.facility_id,
                facility_name=base_context.facility_name,
                department_id=base_context.department_id,
                department_name=base_context.department_name,
                unit_id=base_context.unit_id,
                unit_name=base_context.unit_name,
                active_diagnoses=base_context.active_diagnoses.copy(),
                active_medications=base_context.active_medications.copy(),
                active_allergies=base_context.active_allergies.copy(),
                vital_signs=base_context.vital_signs.copy(),
                laboratory_values=base_context.laboratory_values.copy(),
                risk_factors=base_context.risk_factors.copy(),
                clinical_warnings=base_context.clinical_warnings.copy()
            )
        
        # Perform enrichment based on configuration
        if self.config.enrichment_level in [EnrichmentLevel.STANDARD, EnrichmentLevel.COMPREHENSIVE]:
            await self._enrich_patient_demographics(enriched_context)
            await self._enrich_active_medications(enriched_context)
            await self._enrich_active_diagnoses(enriched_context)
            await self._enrich_allergies(enriched_context)
        
        if self.config.enrichment_level in [EnrichmentLevel.COMPREHENSIVE, EnrichmentLevel.REAL_TIME]:
            await self._enrich_recent_laboratory_values(enriched_context)
            await self._enrich_care_team_information(enriched_context)
            await self._enrich_facility_information(enriched_context)
        
        if self.config.enrichment_level == EnrichmentLevel.REAL_TIME:
            await self._enrich_real_time_vitals(enriched_context)
            await self._enrich_active_orders(enriched_context)
        
        return enriched_context
    
    async def _enrich_patient_demographics(self, context: ClinicalContext):
        """Enrich patient demographic information"""
        try:
            # Handle both object and dictionary types for context
            patient_id = context.get('patient_id') if isinstance(context, dict) else context.patient_id
            
            demographics = await self._fetch_patient_demographics(patient_id)
            
            if demographics:
                # Handle both object and dictionary types for context.patient_demographics
                if isinstance(context, dict):
                    if 'patient_demographics' not in context or not isinstance(context['patient_demographics'], dict):
                        context['patient_demographics'] = {}
                    context['patient_demographics'].update(demographics)
                else:
                    context.patient_demographics.update(demographics)
                    
                self.enrichment_stats["source_usage"][ContextSource.EHR_SYSTEM.value] += 1
                
                logger.debug(f"Enriched demographics for patient {patient_id}")
            
        except Exception as e:
            logger.warning(f"Failed to enrich patient demographics: {e}")
    
    async def _enrich_active_medications(self, context: ClinicalContext):
        """Enrich active medication information"""
        try:
            # Handle both object and dictionary types for context
            patient_id = context.get('patient_id') if isinstance(context, dict) else context.patient_id
            
            # Simulate medication lookup
            medications = await self._fetch_active_medications(patient_id)
            
            if medications:
                # Handle both object and dictionary types for context.active_medications
                if isinstance(context, dict):
                    if 'active_medications' not in context or not isinstance(context['active_medications'], list):
                        context['active_medications'] = []
                    context['active_medications'].extend(medications)
                else:
                    context.active_medications.extend(medications)
                    
                self.enrichment_stats["source_usage"][ContextSource.PHARMACY_SYSTEM.value] += 1
                
                logger.debug(f"Enriched {len(medications)} active medications for patient {patient_id}")
            
        except Exception as e:
            logger.warning(f"Failed to enrich active medications: {e}")
    
    async def _enrich_active_diagnoses(self, context: ClinicalContext):
        """Enrich active diagnosis information"""
        try:
            # Handle both object and dictionary types for context
            patient_id = context.get('patient_id') if isinstance(context, dict) else context.patient_id
            
            # Simulate diagnosis lookup
            diagnoses = await self._fetch_active_diagnoses(patient_id)
            
            if diagnoses:
                # Handle both object and dictionary types for context.active_diagnoses
                if isinstance(context, dict):
                    if 'active_diagnoses' not in context or not isinstance(context['active_diagnoses'], list):
                        context['active_diagnoses'] = []
                    context['active_diagnoses'].extend(diagnoses)
                else:
                    context.active_diagnoses.extend(diagnoses)
                    
                self.enrichment_stats["source_usage"][ContextSource.EHR_SYSTEM.value] += 1
                
                logger.debug(f"Enriched {len(diagnoses)} active diagnoses for patient {patient_id}")
            
        except Exception as e:
            logger.warning(f"Failed to enrich active diagnoses: {e}")
    
    async def _enrich_allergies(self, context: ClinicalContext):
        """Enrich allergy information"""
        try:
            # Handle both object and dictionary types for context
            patient_id = context.get('patient_id') if isinstance(context, dict) else context.patient_id
            
            # Simulate allergy lookup
            allergies = await self._fetch_allergies(patient_id)
            
            if allergies:
                # Handle both object and dictionary types for context.active_allergies
                if isinstance(context, dict):
                    if 'active_allergies' not in context or not isinstance(context['active_allergies'], list):
                        context['active_allergies'] = []
                    context['active_allergies'].extend(allergies)
                else:
                    context.active_allergies.extend(allergies)
                    
                self.enrichment_stats["source_usage"][ContextSource.EHR_SYSTEM.value] += 1
                
                logger.debug(f"Enriched {len(allergies)} allergies for patient {patient_id}")
            
        except Exception as e:
            logger.warning(f"Failed to enrich allergies: {e}")
    
    async def _enrich_recent_laboratory_values(self, context: ClinicalContext):
        """Enrich recent laboratory values"""
        try:
            # Handle both object and dictionary types for context
            patient_id = context.get('patient_id') if isinstance(context, dict) else context.patient_id
            
            # Simulate recent lab lookup
            lab_values = await self._fetch_recent_lab_values(
                patient_id, 
                hours_back=self.config.recent_labs_hours
            )
            
            if lab_values:
                # Handle both object and dictionary types for context.laboratory_values
                if isinstance(context, dict):
                    if 'laboratory_values' not in context or not isinstance(context['laboratory_values'], dict):
                        context['laboratory_values'] = {}
                    context['laboratory_values'].update(lab_values)
                else:
                    context.laboratory_values.update(lab_values)
                    
                self.enrichment_stats["source_usage"][ContextSource.LABORATORY_SYSTEM.value] += 1
                
                logger.debug(f"Enriched {len(lab_values)} recent lab values for patient {patient_id}")
            
        except Exception as e:
            logger.warning(f"Failed to enrich recent lab values: {e}")
    
    async def _enrich_care_team_information(self, context: ClinicalContext):
        """Enrich care team information"""
        try:
            # Handle both object and dictionary types for context
            if isinstance(context, dict):
                encounter_id = context.get('encounter_id')
                patient_id = context.get('patient_id')
                lookup_id = encounter_id or patient_id
            else:
                lookup_id = context.encounter_id or context.patient_id
                
            # Simulate care team lookup
            care_team = await self._fetch_care_team(lookup_id)
            
            if care_team:
                # Handle both object and dictionary types for context.care_team_members
                if isinstance(context, dict):
                    if 'care_team_members' not in context or not isinstance(context['care_team_members'], list):
                        context['care_team_members'] = []
                    context['care_team_members'].extend(care_team)
                else:
                    context.care_team_members.extend(care_team)
                
                self.enrichment_stats["source_usage"][ContextSource.EHR_SYSTEM.value] += 1
                
                logger.debug(f"Enriched {len(care_team)} care team members for patient/encounter {lookup_id}")
            
        except Exception as e:
            logger.warning(f"Failed to enrich care team information: {e}")
    
    async def _enrich_facility_information(self, context: ClinicalContext):
        """Enrich facility and location information"""
        try:
            # Handle both object and dictionary types for context
            if isinstance(context, dict):
                facility_id = context.get('facility_id')
                if not facility_id:
                    return  # Skip enrichment if facility_id is not available
            else:
                facility_id = context.facility_id
                if not facility_id:
                    return  # Skip enrichment if facility_id is not available
                
            # Simulate facility lookup
            facility_info = await self._fetch_facility_info(facility_id)
            
            if facility_info:
                # Handle both object and dictionary types for updating facility info
                if isinstance(context, dict):
                    # Get current values or empty strings for defaults
                    context['facility_name'] = facility_info.get("name", context.get('facility_name', ''))
                    context['department_name'] = facility_info.get("department_name", context.get('department_name', ''))
                    context['unit_name'] = facility_info.get("unit_name", context.get('unit_name', ''))
                else:
                    context.facility_name = facility_info.get("name", context.facility_name)
                    context.department_name = facility_info.get("department_name", context.department_name)
                    context.unit_name = facility_info.get("unit_name", context.unit_name)
                
                self.enrichment_stats["source_usage"][ContextSource.EHR_SYSTEM.value] += 1
                
                logger.debug(f"Enriched facility information for facility {facility_id}")
            
        except Exception as e:
            logger.warning(f"Failed to enrich facility information: {e}")
    
    async def _enrich_real_time_vitals(self, context: ClinicalContext):
        """Enrich real-time vital signs"""
        try:
            # Handle both object and dictionary types for context
            patient_id = context.get('patient_id') if isinstance(context, dict) else context.patient_id
            
            # Simulate real-time vitals lookup
            vitals = await self._fetch_real_time_vitals(patient_id)
            
            if vitals:
                # Handle both object and dictionary types for context.vital_signs
                if isinstance(context, dict):
                    if 'vital_signs' not in context or not isinstance(context['vital_signs'], dict):
                        context['vital_signs'] = {}
                    context['vital_signs'].update(vitals)
                else:
                    context.vital_signs.update(vitals)
                    
                self.enrichment_stats["source_usage"][ContextSource.REAL_TIME_MONITORING.value] += 1
                
                logger.debug(f"Enriched real-time vitals for patient {patient_id}")
            
        except Exception as e:
            logger.warning(f"Failed to enrich real-time vitals: {e}")
    
    async def _enrich_active_orders(self, context: ClinicalContext):
        """Enrich active orders information"""
        try:
            # Handle both object and dictionary types for context
            patient_id = context.get('patient_id') if isinstance(context, dict) else context.patient_id
            
            # Simulate active orders lookup
            orders = await self._fetch_active_orders(patient_id)
            
            if orders:
                # Add orders to metadata or custom field
                if isinstance(context, dict):
                    # Add to dictionary if active_orders field exists
                    if 'active_orders' in context and isinstance(context['active_orders'], list):
                        context['active_orders'].extend(orders)
                    else:
                        # Just add to stats, don't modify context as this appears to be a placeholder
                        pass
                else:
                    # Add to object if active_orders field exists
                    if "active_orders" in context.__dict__:
                        context.active_orders.extend(orders)
                
                self.enrichment_stats["source_usage"][ContextSource.EHR_SYSTEM.value] += 1
                
                logger.debug(f"Enriched {len(orders)} active orders for patient {patient_id}")
            
        except Exception as e:
            logger.warning(f"Failed to enrich active orders: {e}")
    
    # Simulated data fetch methods (would integrate with real systems)
    
    async def _fetch_patient_demographics(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Fetch patient demographics from GraphDB"""
        if not self.graphdb_client:
            logger.warning("GraphDB client not configured, cannot fetch demographics")
            return None
        
        try:
            logger.info(f"Fetching demographics for patient {patient_id} from GraphDB...")
            demographics = await self.graphdb_client.get_patient_demographics(patient_id)
            if demographics:
                logger.info(f"Successfully fetched demographics for patient {patient_id}")
                self.enrichment_stats["source_usage"][ContextSource.EHR_SYSTEM.value] += 1
            else:
                logger.warning(f"No demographics found for patient {patient_id} in GraphDB")
            return demographics
        except Exception as e:
            logger.error(f"Error fetching demographics from GraphDB for patient {patient_id}: {e}")
            return None
    
    async def _fetch_active_medications(self, patient_id: str) -> List[Dict[str, Any]]:
        """Fetch active medications"""
        await asyncio.sleep(0.02)
        
        return [
            {
                "name": "Lisinopril",
                "dosage": "10mg",
                "frequency": "daily",
                "route": "oral",
                "start_date": "2024-01-15",
                "status": "active"
            },
            {
                "name": "Metformin",
                "dosage": "500mg",
                "frequency": "twice daily",
                "route": "oral",
                "start_date": "2024-02-01",
                "status": "active"
            }
        ]
    
    async def _fetch_active_diagnoses(self, patient_id: str) -> List[Dict[str, Any]]:
        """Fetch active diagnoses"""
        await asyncio.sleep(0.02)
        
        return [
            {
                "code": "I10",
                "description": "Essential hypertension",
                "onset_date": "2024-01-10",
                "status": "active"
            },
            {
                "code": "E11.9",
                "description": "Type 2 diabetes mellitus without complications",
                "onset_date": "2024-01-20",
                "status": "active"
            }
        ]
    
    async def _fetch_allergies(self, patient_id: str) -> List[Dict[str, Any]]:
        """Fetch patient allergies"""
        await asyncio.sleep(0.01)
        
        return [
            {
                "allergen": "Penicillin",
                "reaction": "Rash",
                "severity": "moderate",
                "verified": True
            }
        ]
    
    async def _fetch_recent_lab_values(self, patient_id: str, hours_back: int) -> Dict[str, Any]:
        """Fetch recent laboratory values"""
        await asyncio.sleep(0.03)
        
        return {
            "glucose": {"value": 120, "unit": "mg/dL", "timestamp": "2024-07-07T10:00:00Z"},
            "creatinine": {"value": 1.1, "unit": "mg/dL", "timestamp": "2024-07-07T10:00:00Z"},
            "hemoglobin": {"value": 13.5, "unit": "g/dL", "timestamp": "2024-07-07T10:00:00Z"}
        }
    
    async def _fetch_care_team(self, identifier: str) -> List[Dict[str, Any]]:
        """Fetch care team members"""
        await asyncio.sleep(0.02)
        
        return [
            {
                "provider_id": "DR001",
                "name": "Dr. Smith",
                "role": "attending_physician",
                "specialty": "internal_medicine"
            },
            {
                "provider_id": "RN001",
                "name": "Nurse Johnson",
                "role": "primary_nurse",
                "specialty": "medical_surgical"
            }
        ]
    
    async def _fetch_facility_info(self, facility_id: Optional[str]) -> Optional[Dict[str, Any]]:
        """Fetch facility information"""
        if not facility_id:
            return None
        
        await asyncio.sleep(0.01)
        
        return {
            "name": "General Hospital",
            "department_name": "Internal Medicine",
            "unit_name": "Medical Ward 3A"
        }
    
    async def _fetch_real_time_vitals(self, patient_id: str) -> Dict[str, Any]:
        """Fetch real-time vital signs"""
        await asyncio.sleep(0.05)
        
        return {
            "heart_rate": {"value": 72, "unit": "bpm", "timestamp": datetime.now(timezone.utc).isoformat()},
            "blood_pressure": {"systolic": 130, "diastolic": 80, "unit": "mmHg", "timestamp": datetime.now(timezone.utc).isoformat()},
            "temperature": {"value": 98.6, "unit": "F", "timestamp": datetime.now(timezone.utc).isoformat()}
        }
    
    async def _fetch_active_orders(self, patient_id: str) -> List[Dict[str, Any]]:
        """Fetch active orders"""
        await asyncio.sleep(0.02)
        
        return [
            {
                "order_id": "ORD001",
                "type": "medication",
                "description": "Administer Lisinopril 10mg PO daily",
                "status": "active",
                "ordered_by": "DR001"
            }
        ]
    
    def _generate_cache_key(self, envelope: ClinicalEventEnvelope) -> str:
        """Generate cache key for context"""
        # Handle both object and dictionary types for clinical_context
        if isinstance(envelope.clinical_context, dict):
            patient_id = envelope.clinical_context.get('patient_id', 'unknown_patient')
            encounter_id = envelope.clinical_context.get('encounter_id') or "no_encounter"
        else:
            patient_id = envelope.clinical_context.patient_id
            encounter_id = envelope.clinical_context.encounter_id or "no_encounter"
            
        key_components = [
            patient_id,
            encounter_id,
            self.config.enrichment_level.value
        ]
        return "_".join(key_components)
    
    async def _get_cached_context(self, cache_key: str) -> Optional[ClinicalContext]:
        """Get cached clinical context"""
        if cache_key in self.context_cache:
            context, cached_at = self.context_cache[cache_key]
            
            # Check if cache is still valid
            cache_age = datetime.now() - cached_at
            if cache_age.total_seconds() < (self.config.cache_ttl_minutes * 60):
                return context
            else:
                # Remove expired cache entry
                del self.context_cache[cache_key]
        
        return None
    
    async def _cache_context(self, cache_key: str, context: ClinicalContext):
        """Cache clinical context"""
        self.context_cache[cache_key] = (context, datetime.now())
        
        # Simple cache cleanup (keep last 1000 entries)
        if len(self.context_cache) > 1000:
            # Remove oldest entries
            sorted_cache = sorted(self.context_cache.items(), key=lambda x: x[1][1])
            for key, _ in sorted_cache[:100]:
                del self.context_cache[key]
    
    async def _update_enrichment_stats(self, enrichment_time_ms: float):
        """Update enrichment statistics"""
        self.enrichment_stats["total_enrichments"] += 1
        
        # Update average enrichment time
        total = self.enrichment_stats["total_enrichments"]
        current_avg = self.enrichment_stats["average_enrichment_time_ms"]
        self.enrichment_stats["average_enrichment_time_ms"] = (
            (current_avg * (total - 1) + enrichment_time_ms) / total
        )
    
    def get_enrichment_stats(self) -> Dict[str, Any]:
        """Get enrichment statistics"""
        cache_hit_ratio = 0.0
        total_requests = self.enrichment_stats["cache_hits"] + self.enrichment_stats["cache_misses"]
        
        if total_requests > 0:
            cache_hit_ratio = (self.enrichment_stats["cache_hits"] / total_requests) * 100
        
        return {
            **self.enrichment_stats,
            "cache_hit_ratio_percent": round(cache_hit_ratio, 2),
            "cached_contexts": len(self.context_cache),
            "enrichment_level": self.config.enrichment_level.value
        }


class ContextEnrichmentEngine:
    """
    High-level context enrichment engine
    
    Orchestrates context enrichment across multiple assemblers
    and provides unified interface for context enhancement.
    """
    
    def __init__(self):
        self.assemblers: Dict[str, ClinicalContextAssembler] = {}
        self.default_assembler = ClinicalContextAssembler()
        
        logger.info("Context Enrichment Engine initialized")
    
    def register_assembler(self, name: str, assembler: ClinicalContextAssembler):
        """Register a context assembler"""
        self.assemblers[name] = assembler
        logger.info(f"Registered context assembler: {name}")
    
    async def enrich_envelope(self, envelope: ClinicalEventEnvelope, 
                            assembler_name: Optional[str] = None) -> ClinicalEventEnvelope:
        """Enrich event envelope using specified or default assembler"""
        assembler = self.assemblers.get(assembler_name, self.default_assembler)
        return await assembler.enrich_clinical_context(envelope)

    async def enrich_request_with_context(self, request: 'ClinicalRequest', 
                                          assembler_name: Optional[str] = None) -> 'ClinicalRequest':
        """Enrich a ClinicalRequest object with clinical context"""
        import traceback
        import json
        from ..orchestration.request_router import ClinicalRequest
        from app.events.clinical_event_envelope import (ClinicalEventEnvelope, EventMetadata, 
                                                EventType, ClinicalContext, EventStatus, 
                                                EventSeverity, ProvenanceContext)
        
        logger.info(f"Starting request enrichment - Request type: {type(request)}")
        
        # Debug request contents
        if isinstance(request, dict):
            logger.info(f"Request is a dictionary with keys: {list(request.keys())}")
            if 'patient_id' in request:
                logger.info(f"Dictionary has patient_id: {request['patient_id']}")
            else:
                logger.warning(f"Dictionary does not have patient_id key")
        else:
            logger.info(f"Request attributes: {dir(request)}")
            if hasattr(request, 'patient_id'):
                logger.info(f"Object has patient_id: {request.patient_id}")
            else:
                logger.warning(f"Object does not have patient_id attribute")
                
        try:
            # This is a bit of a workaround to reuse the envelope-based enrichment logic
            # Generate a unique event ID using patient_id since request_id might not be available
            logger.info("Extracting patient_id from request")
            patient_id = request.get('patient_id', 'unknown_patient') if isinstance(request, dict) else request.patient_id
            logger.info(f"Extracted patient_id: {patient_id}")
            
            event_id = f"req_{patient_id}_{datetime.now(timezone.utc).timestamp()}"
            logger.info(f"Generated event_id: {event_id}")
            
            # Handle clinical_context based on request type
            logger.info("Getting clinical_context from request")
            if isinstance(request, dict):
                if request.get('clinical_context') is None:
                    clinical_ctx = ClinicalContext(patient_id=patient_id)
                    logger.info("Created new ClinicalContext with patient_id")
                else:
                    clinical_ctx = request.get('clinical_context')
                    # Ensure patient_id is set in the dictionary if clinical_ctx is a dictionary
                    if isinstance(clinical_ctx, dict):
                        if 'patient_id' not in clinical_ctx:
                            clinical_ctx['patient_id'] = patient_id
                            logger.info(f"Added patient_id to existing clinical_context dictionary: {patient_id}")
                    logger.info(f"Using existing clinical_context: {type(clinical_ctx)}")
            else:
                if request.clinical_context is None:
                    clinical_ctx = ClinicalContext(patient_id=patient_id)
                    logger.info("Created new ClinicalContext with patient_id")
                else:
                    clinical_ctx = request.clinical_context
                    # Ensure patient_id is set in the dictionary if clinical_ctx is a dictionary
                    if isinstance(clinical_ctx, dict) and 'patient_id' not in clinical_ctx:
                        clinical_ctx['patient_id'] = patient_id
                        logger.info(f"Added patient_id to existing clinical_context dictionary: {patient_id}")
                    logger.info(f"Using existing clinical_context: {type(clinical_ctx)}")
            
            logger.info("Creating temporary ClinicalEventEnvelope")
            temp_envelope = ClinicalEventEnvelope(
                event_data={"type": "enrichment_query", "patient_id": patient_id},
                clinical_context=clinical_ctx,
                temporal_context=TemporalContext(event_time=datetime.now(timezone.utc), system_time=datetime.now(timezone.utc)),
                provenance_context=ProvenanceContext(source_system="ContextEnrichmentEngine", created_by="system", created_at=datetime.now(timezone.utc)),
                metadata=EventMetadata(event_type=EventType.CLINICAL_QUERY, event_status=EventStatus.CREATED, event_severity=EventSeverity.INFORMATIONAL, tags=["context-enrichment", "internal-query"])
            )
            logger.info(f"Created envelope with event_data: {temp_envelope.event_data}")

            # Enrich the envelope
            logger.info("Calling enrich_envelope")
            enriched_envelope = await self.enrich_envelope(temp_envelope, assembler_name)
            logger.info(f"Envelope enriched successfully: {type(enriched_envelope.clinical_context)}")

            # Update the original request with the enriched context
            logger.info("Updating original request with enriched context")
            if isinstance(request, dict):
                request['clinical_context'] = enriched_envelope.clinical_context
                logger.info("Updated dictionary with enriched context")
            else:
                request.clinical_context = enriched_envelope.clinical_context
                logger.info("Updated object with enriched context")

            return request
        except Exception as e:
            logger.error(f"Error in enrich_request_with_context: {e}")
            logger.error(f"Traceback: {traceback.format_exc()}")
            raise
    
    def get_engine_stats(self) -> Dict[str, Any]:
        """Get engine statistics"""
        stats = {
            "registered_assemblers": len(self.assemblers),
            "assembler_names": list(self.assemblers.keys()),
            "assembler_stats": {}
        }
        
        # Get stats from each assembler
        for name, assembler in self.assemblers.items():
            stats["assembler_stats"][name] = assembler.get_enrichment_stats()
        
        # Include default assembler stats
        stats["assembler_stats"]["default"] = self.default_assembler.get_enrichment_stats()
        
        return stats
