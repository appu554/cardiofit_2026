"""
Medical-Aware Circuit Breaker for Clinical Safety Overload Protection

Implements priority lanes and medical context awareness to ensure critical
clinical data can be processed even when the system is under heavy load.
"""

import asyncio
import logging
import time
from enum import Enum
from typing import Dict, Any, List, Optional, Set
from dataclasses import dataclass
from datetime import datetime, timedelta

from app.core.config import settings

logger = logging.getLogger(__name__)

class MedicalPriority(Enum):
    """Medical priority levels for clinical safety"""
    EMERGENCY = "emergency"      # Life-threatening: cardiac arrest, severe bleeding
    CRITICAL = "critical"        # Urgent medical attention: abnormal vitals
    HIGH = "high"               # Important clinical data: lab results, medications
    NORMAL = "normal"           # Standard clinical data: routine observations
    LOW = "low"                 # Non-clinical data: device metadata, logs

class CircuitBreakerState(Enum):
    """Circuit breaker states"""
    CLOSED = "closed"           # Normal operation
    HALF_OPEN = "half_open"     # Testing recovery
    OPEN = "open"               # Blocking requests

@dataclass
class MedicalContext:
    """Medical context for event classification"""
    event_type: str
    patient_id: Optional[str] = None
    vital_signs: Optional[Dict[str, Any]] = None
    emergency_indicators: Optional[List[str]] = None
    clinical_severity: Optional[str] = None

@dataclass
class LoadMetrics:
    """System load metrics"""
    queue_depth: int = 0
    processing_rate: float = 0.0
    error_rate: float = 0.0
    memory_usage: float = 0.0
    cpu_usage: float = 0.0
    timestamp: datetime = None

class MedicalAwareCircuitBreaker:
    """
    Medical-Aware Circuit Breaker with Priority Lanes
    
    Features:
    - Separate processing lanes by medical priority
    - Emergency bypass (always processes EMERGENCY events)
    - Adaptive load shedding based on clinical importance
    - Medical context classification
    - Clinical safety guarantees
    """
    
    def __init__(self):
        self.state = CircuitBreakerState.CLOSED
        self.priority_states: Dict[MedicalPriority, CircuitBreakerState] = {
            priority: CircuitBreakerState.CLOSED for priority in MedicalPriority
        }
        
        # Configuration
        self.max_queue_depth = settings.MEDICAL_CIRCUIT_BREAKER_MAX_QUEUE_DEPTH
        self.emergency_bypass_enabled = True
        self.critical_threshold = settings.MEDICAL_CIRCUIT_BREAKER_CRITICAL_THRESHOLD
        
        # Metrics tracking
        self.load_metrics = LoadMetrics(timestamp=datetime.utcnow())
        self.priority_metrics: Dict[MedicalPriority, Dict[str, int]] = {
            priority: {"processed": 0, "dropped": 0, "errors": 0}
            for priority in MedicalPriority
        }
        
        # Medical event patterns for classification
        self.emergency_patterns = {
            "cardiac_arrest", "severe_bleeding", "respiratory_failure",
            "stroke_alert", "sepsis_alert", "anaphylaxis"
        }
        
        self.critical_patterns = {
            "abnormal_vitals", "critical_lab", "medication_alert",
            "fall_detection", "arrhythmia", "hypotension", "hypertension"
        }
        
        self.vital_sign_thresholds = {
            "heart_rate": {"emergency": (40, 150), "critical": (50, 130)},
            "blood_pressure_systolic": {"emergency": (70, 200), "critical": (90, 180)},
            "oxygen_saturation": {"emergency": (85, 100), "critical": (90, 100)},
            "temperature": {"emergency": (35.0, 40.0), "critical": (36.0, 39.0)}
        }
    
    def classify_medical_priority(self, event_data: Dict[str, Any]) -> MedicalPriority:
        """
        Classify event medical priority based on clinical context
        
        Args:
            event_data: Event data including type, payload, and metadata
            
        Returns:
            MedicalPriority: Classified priority level
        """
        try:
            event_type = event_data.get("event_type", "").lower()
            metadata = event_data.get("metadata", {})
            
            # Check for emergency patterns
            if any(pattern in event_type for pattern in self.emergency_patterns):
                return MedicalPriority.EMERGENCY
            
            # Check for critical patterns
            if any(pattern in event_type for pattern in self.critical_patterns):
                return MedicalPriority.CRITICAL
            
            # Analyze vital signs if present
            if "vital_signs" in metadata:
                vital_priority = self._analyze_vital_signs(metadata["vital_signs"])
                if vital_priority:
                    return vital_priority
            
            # Check for medication-related events
            if "medication" in event_type or "drug" in event_type:
                return MedicalPriority.HIGH
            
            # Check for patient-related clinical events
            if any(keyword in event_type for keyword in ["patient", "clinical", "diagnosis", "treatment"]):
                return MedicalPriority.NORMAL
            
            # Default to low priority for non-clinical events
            return MedicalPriority.LOW
            
        except Exception as e:
            logger.warning(f"Failed to classify medical priority: {e}")
            return MedicalPriority.NORMAL  # Safe default
    
    def _analyze_vital_signs(self, vital_signs: Dict[str, Any]) -> Optional[MedicalPriority]:
        """Analyze vital signs for medical priority classification"""
        try:
            for vital_type, value in vital_signs.items():
                if vital_type in self.vital_sign_thresholds:
                    thresholds = self.vital_sign_thresholds[vital_type]
                    
                    # Check emergency thresholds
                    emergency_min, emergency_max = thresholds["emergency"]
                    if value < emergency_min or value > emergency_max:
                        return MedicalPriority.EMERGENCY
                    
                    # Check critical thresholds
                    critical_min, critical_max = thresholds["critical"]
                    if value < critical_min or value > critical_max:
                        return MedicalPriority.CRITICAL
            
            return None
            
        except Exception as e:
            logger.warning(f"Failed to analyze vital signs: {e}")
            return None
    
    async def should_process_event(self, event_data: Dict[str, Any]) -> bool:
        """
        Determine if event should be processed based on medical priority and system load
        
        Args:
            event_data: Event data to evaluate
            
        Returns:
            bool: True if event should be processed, False if dropped
        """
        try:
            # Classify medical priority
            medical_priority = self.classify_medical_priority(event_data)
            
            # Emergency bypass - always process emergency events
            if medical_priority == MedicalPriority.EMERGENCY:
                logger.info(f"🚨 Emergency event bypass: {event_data.get('event_type')}")
                self.priority_metrics[medical_priority]["processed"] += 1
                return True
            
            # Update load metrics
            await self._update_load_metrics()
            
            # Check priority-specific circuit breaker state
            priority_state = self.priority_states[medical_priority]
            
            if priority_state == CircuitBreakerState.OPEN:
                # Check if we should attempt recovery
                if await self._should_attempt_recovery(medical_priority):
                    self.priority_states[medical_priority] = CircuitBreakerState.HALF_OPEN
                    logger.info(f"🔄 Medical circuit breaker for {medical_priority.value} transitioning to HALF_OPEN")
                else:
                    self.priority_metrics[medical_priority]["dropped"] += 1
                    return False
            
            # Apply load-based filtering
            if await self._should_drop_due_to_load(medical_priority):
                self.priority_metrics[medical_priority]["dropped"] += 1
                return False
            
            # Process the event
            self.priority_metrics[medical_priority]["processed"] += 1
            return True
            
        except Exception as e:
            logger.error(f"Error in medical circuit breaker evaluation: {e}")
            # Fail safe - process critical and emergency events
            return medical_priority in [MedicalPriority.EMERGENCY, MedicalPriority.CRITICAL]
    
    async def _update_load_metrics(self):
        """Update system load metrics"""
        try:
            # This would integrate with actual system monitoring
            # For now, we'll use placeholder metrics
            current_time = datetime.utcnow()
            
            # Update metrics (integrate with actual monitoring system)
            self.load_metrics = LoadMetrics(
                queue_depth=await self._get_queue_depth(),
                processing_rate=await self._get_processing_rate(),
                error_rate=await self._get_error_rate(),
                timestamp=current_time
            )
            
        except Exception as e:
            logger.warning(f"Failed to update load metrics: {e}")
    
    async def _get_queue_depth(self) -> int:
        """Get current queue depth from database"""
        try:
            from app.core.database import db_manager
            
            async with db_manager.get_connection() as conn:
                queue_depth = await conn.fetchval("""
                    SELECT COUNT(*) FROM global_event_outbox 
                    WHERE status = 'pending'
                """)
                return queue_depth or 0
                
        except Exception as e:
            logger.warning(f"Failed to get queue depth: {e}")
            return 0
    
    async def _get_processing_rate(self) -> float:
        """Get current processing rate"""
        # Placeholder - would integrate with actual metrics
        return 100.0
    
    async def _get_error_rate(self) -> float:
        """Get current error rate"""
        # Placeholder - would integrate with actual metrics
        return 0.05
    
    async def _should_drop_due_to_load(self, priority: MedicalPriority) -> bool:
        """Determine if event should be dropped due to system load"""
        try:
            # Never drop emergency or critical events
            if priority in [MedicalPriority.EMERGENCY, MedicalPriority.CRITICAL]:
                return False
            
            # Check queue depth
            if self.load_metrics.queue_depth > self.max_queue_depth:
                # Drop low priority events first
                if priority == MedicalPriority.LOW:
                    return True
                # Drop normal priority if severely overloaded
                elif priority == MedicalPriority.NORMAL and self.load_metrics.queue_depth > self.max_queue_depth * 1.5:
                    return True
            
            # Check error rate
            if self.load_metrics.error_rate > 0.1:  # 10% error rate
                # Drop non-critical events during high error periods
                return priority not in [MedicalPriority.EMERGENCY, MedicalPriority.CRITICAL]
            
            return False
            
        except Exception as e:
            logger.warning(f"Failed to evaluate load-based dropping: {e}")
            return False
    
    async def _should_attempt_recovery(self, priority: MedicalPriority) -> bool:
        """Check if circuit breaker should attempt recovery"""
        # Implement recovery logic based on time and system health
        return True  # Placeholder
    
    def get_circuit_breaker_status(self) -> Dict[str, Any]:
        """Get comprehensive circuit breaker status"""
        return {
            "overall_state": self.state.value,
            "priority_states": {
                priority.value: state.value 
                for priority, state in self.priority_states.items()
            },
            "load_metrics": {
                "queue_depth": self.load_metrics.queue_depth,
                "processing_rate": self.load_metrics.processing_rate,
                "error_rate": self.load_metrics.error_rate,
                "timestamp": self.load_metrics.timestamp.isoformat() if self.load_metrics.timestamp else None
            },
            "priority_metrics": {
                priority.value: metrics 
                for priority, metrics in self.priority_metrics.items()
            },
            "emergency_bypass_enabled": self.emergency_bypass_enabled
        }

# Global instance
medical_circuit_breaker = MedicalAwareCircuitBreaker()
