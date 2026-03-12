"""
Request Router for Clinical Assertion Engine

Handles priority classification, request validation, and routing to appropriate
reasoners based on clinical context and urgency.
"""

import asyncio
import logging
from datetime import datetime, timedelta
from enum import Enum
from typing import Dict, List, Optional, Any
from dataclasses import dataclass

logger = logging.getLogger(__name__)


class RequestPriority(Enum):
    """Request priority levels with SLA targets"""
    CRITICAL = 1      # <50ms - Life-threatening situations
    HIGH = 2         # <100ms - Urgent clinical decisions
    NORMAL = 3       # <500ms - Standard clinical workflow
    BATCH = 4        # <5000ms - Background processing

    def __lt__(self, other):
        if self.__class__ is other.__class__:
            return self.value < other.value
        return NotImplemented

    def __le__(self, other):
        if self.__class__ is other.__class__:
            return self.value <= other.value
        return NotImplemented

    def __gt__(self, other):
        if self.__class__ is other.__class__:
            return self.value > other.value
        return NotImplemented

    def __ge__(self, other):
        if self.__class__ is other.__class__:
            return self.value >= other.value
        return NotImplemented

    @property
    def display_name(self) -> str:
        """Get display name for priority"""
        display_names = {
            1: "critical",
            2: "high",
            3: "normal",
            4: "batch"
        }
        return display_names[self.value]


@dataclass
class ClinicalRequest:
    """Enhanced clinical request with priority and context"""
    patient_id: str
    correlation_id: str
    reasoner_types: List[str]
    medication_ids: List[str]
    condition_ids: List[str]
    allergy_ids: List[str]
    priority: RequestPriority
    clinical_context: Dict[str, Any]
    temporal_context: Dict[str, Any]
    timeout_ms: int
    created_at: datetime


class RequestRouter:
    """
    Intelligent request router with priority classification and context-aware routing
    
    Features:
    - Priority-based SLA enforcement
    - Clinical context analysis for routing decisions
    - Circuit breaker patterns for resilience
    - Request validation and sanitization
    """
    
    def __init__(self):
        self.circuit_breakers = {}
        self.request_stats = {
            'total_requests': 0,
            'priority_distribution': {p.display_name: 0 for p in RequestPriority},
            'average_response_times': {p.display_name: [] for p in RequestPriority}
        }
        logger.info("Request Router initialized")
    
    async def route_request(self, raw_request: Dict[str, Any]) -> ClinicalRequest:
        """
        Route and classify incoming clinical reasoning request
        
        Args:
            raw_request: Raw gRPC request data
            
        Returns:
            ClinicalRequest: Classified and validated request
        """
        try:
            # Validate request
            validated_request = await self._validate_request(raw_request)
            
            # Classify priority based on clinical context
            priority = await self._classify_priority(validated_request)
            
            # Determine timeout based on priority
            timeout_ms = self._get_timeout_for_priority(priority)
            
            # Build enhanced clinical request
            clinical_request = ClinicalRequest(
                patient_id=validated_request['patient_id'],
                correlation_id=validated_request.get('correlation_id', f"corr_{validated_request['patient_id']}"),
                reasoner_types=validated_request.get('reasoner_types', ['interaction', 'dosing', 'contraindication']),
                medication_ids=validated_request.get('medication_ids', []),
                condition_ids=validated_request.get('condition_ids', []),
                allergy_ids=validated_request.get('allergy_ids', []),
                priority=priority,
                clinical_context=validated_request.get('clinical_context', {}),
                temporal_context=validated_request.get('temporal_context', {}),
                timeout_ms=timeout_ms,
                created_at=datetime.utcnow()
            )
            
            # Update statistics
            self._update_stats(clinical_request)
            
            logger.info(f"Routed request for patient {clinical_request.patient_id} "
                       f"with priority {priority.display_name} (timeout: {timeout_ms}ms)")
            
            return clinical_request
            
        except Exception as e:
            logger.error(f"Error routing request: {e}")
            raise
    
    async def _validate_request(self, request: Dict[str, Any]) -> Dict[str, Any]:
        """Validate and sanitize incoming request"""
        if not request.get('patient_id'):
            raise ValueError("Patient ID is required")
        
        # Sanitize patient ID
        patient_id = str(request['patient_id']).strip()
        if not patient_id:
            raise ValueError("Invalid patient ID")
        
        # Validate medication IDs if provided
        medication_ids = request.get('medication_ids', [])
        if medication_ids and not isinstance(medication_ids, list):
            raise ValueError("medication_ids must be a list")
        
        # Validate reasoner types
        valid_reasoners = ['interaction', 'dosing', 'contraindication', 'duplicate_therapy', 'clinical_context']
        reasoner_types = request.get('reasoner_types', ['interaction', 'dosing', 'contraindication', 'duplicate_therapy', 'clinical_context'])
        
        for reasoner in reasoner_types:
            if reasoner not in valid_reasoners:
                raise ValueError(f"Invalid reasoner type: {reasoner}")
        
        return {
            'patient_id': patient_id,
            'correlation_id': request.get('correlation_id'),
            'reasoner_types': reasoner_types,
            'medication_ids': [str(mid).strip() for mid in medication_ids],
            'condition_ids': request.get('condition_ids', []),
            'allergy_ids': request.get('allergy_ids', []),
            'clinical_context': request.get('clinical_context', {}),
            'temporal_context': request.get('temporal_context', {})
        }
    
    async def _classify_priority(self, request: Dict[str, Any]) -> RequestPriority:
        """
        Classify request priority based on clinical context
        
        Priority classification logic:
        - CRITICAL: Emergency medications, life-threatening interactions
        - HIGH: Urgent care, high-risk medications
        - NORMAL: Standard clinical workflow
        - BATCH: Background processing, analytics
        """
        clinical_context = request.get('clinical_context', {})
        medication_ids = request.get('medication_ids', [])
        
        # Check for emergency/critical medications (life-threatening situations)
        critical_medications = [
            'epinephrine', 'norepinephrine', 'dopamine', 'dobutamine',
            'vasopressin', 'nitroglycerin'
        ]

        # Check for high-risk medications (urgent but not immediately life-threatening)
        high_risk_medications = [
            'heparin', 'warfarin', 'insulin', 'digoxin', 'amiodarone', 'lidocaine'
        ]

        # Ensure medication_ids are strings before comparison
        medication_strings = [str(med) for med in medication_ids]
        if any(med.lower() in critical_medications for med in medication_strings):
            return RequestPriority.CRITICAL

        if any(med.lower() in high_risk_medications for med in medication_strings):
            return RequestPriority.HIGH
        
        # Check clinical context for urgency indicators
        encounter_type = clinical_context.get('encounter_type', '').lower()
        if encounter_type in ['emergency', 'critical_care', 'icu', 'trauma']:
            return RequestPriority.CRITICAL
        
        if encounter_type in ['urgent_care', 'surgery', 'procedure']:
            return RequestPriority.HIGH
        
        # Check for high-risk patient conditions
        conditions = request.get('condition_ids', [])
        high_risk_conditions = [
            'renal_failure', 'liver_failure', 'heart_failure',
            'pregnancy', 'pediatric', 'geriatric'
        ]
        
        # Ensure conditions are strings before comparison
        condition_strings = [str(cond) for cond in conditions]
        if any(condition.lower() in high_risk_conditions for condition in condition_strings):
            return RequestPriority.HIGH
        
        # Default to normal priority
        return RequestPriority.NORMAL
    
    def _get_timeout_for_priority(self, priority: RequestPriority) -> int:
        """Get timeout in milliseconds based on priority"""
        timeout_map = {
            RequestPriority.CRITICAL: 50,
            RequestPriority.HIGH: 100,
            RequestPriority.NORMAL: 500,
            RequestPriority.BATCH: 5000
        }
        return timeout_map[priority]
    
    def _update_stats(self, request: ClinicalRequest):
        """Update routing statistics"""
        self.request_stats['total_requests'] += 1
        self.request_stats['priority_distribution'][request.priority.display_name] += 1
    
    def get_stats(self) -> Dict[str, Any]:
        """Get routing statistics"""
        return self.request_stats.copy()
