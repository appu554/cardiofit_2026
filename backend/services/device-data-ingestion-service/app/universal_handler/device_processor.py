"""
Universal Device Handler - Device Processor Interface

Abstract base class and concrete implementations for device-specific processing
following the processor pattern with standardized interface.
"""

import logging
import time
from abc import ABC, abstractmethod
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Tuple, Union
from dataclasses import dataclass
from enum import Enum

logger = logging.getLogger(__name__)


class DeviceType(Enum):
    """Supported device types"""
    HEART_RATE = "heart_rate"
    BLOOD_PRESSURE = "blood_pressure"
    BLOOD_GLUCOSE = "blood_glucose"
    WEIGHT = "weight"
    STEPS = "steps"
    SLEEP_DURATION = "sleep_duration"
    ECG = "ecg"
    TEMPERATURE = "temperature"
    OXYGEN_SATURATION = "oxygen_saturation"
    UNKNOWN = "unknown"


class ParameterCriticality(Enum):
    """Medical parameter criticality levels"""
    EMERGENCY = "emergency"      # Life-threatening requiring immediate attention
    CRITICAL = "critical"        # Significant findings needing prompt review
    ROUTINE = "routine"          # Standard medical monitoring
    WELLNESS = "wellness"        # Fitness and lifestyle data


class ProcessingResult(Enum):
    """Processing result status"""
    SUCCESS = "success"
    WARNING = "warning"
    ERROR = "error"
    EMERGENCY = "emergency"


@dataclass
class DeviceCapability:
    """Device capability metadata"""
    device_type: DeviceType
    parameter_types: List[str]
    medical_grade: bool
    real_time_required: bool
    batch_eligible: bool
    emergency_detection: bool
    validation_rules: Dict[str, Any]
    accuracy_specs: Dict[str, Any]
    compliance_metadata: Dict[str, Any]


@dataclass
class ProcessingContext:
    """Context for device processing"""
    device_id: str
    device_type: DeviceType
    timestamp: int
    patient_id: Optional[str]
    user_context: Dict[str, Any]
    request_metadata: Dict[str, Any]
    processing_hints: Dict[str, Any]


@dataclass
class ValidationResult:
    """Result of device data validation"""
    is_valid: bool
    criticality: ParameterCriticality
    warnings: List[str]
    errors: List[str]
    emergency_detected: bool
    quality_score: float
    recommendations: List[str]


@dataclass
class ProcessedDeviceData:
    """Result of device data processing"""
    original_data: Dict[str, Any]
    processed_data: Dict[str, Any]
    validation_result: ValidationResult
    processing_result: ProcessingResult
    metadata: Dict[str, Any]
    processing_time_ms: float
    processor_version: str


class AbstractDeviceProcessor(ABC):
    """
    Abstract base class for all device processors
    
    Defines the standardized interface that all device processors must implement
    for consistent processing across different device types.
    """
    
    def __init__(self, processor_id: str, version: str = "1.0.0"):
        self.processor_id = processor_id
        self.version = version
        self.created_at = datetime.now(timezone.utc)
        self.processing_count = 0
        self.error_count = 0
        self.last_error = None
        
    @abstractmethod
    def get_supported_device_types(self) -> List[DeviceType]:
        """Return list of device types this processor can handle"""
        pass
    
    @abstractmethod
    def get_device_capabilities(self, device_type: DeviceType) -> DeviceCapability:
        """Return capabilities for a specific device type"""
        pass
    
    @abstractmethod
    def can_process(self, device_data: Dict[str, Any], context: ProcessingContext) -> bool:
        """Check if this processor can handle the given device data"""
        pass
    
    @abstractmethod
    def validate_data(self, device_data: Dict[str, Any], context: ProcessingContext) -> ValidationResult:
        """Validate device data and assess criticality"""
        pass
    
    @abstractmethod
    def process_data(self, device_data: Dict[str, Any], context: ProcessingContext) -> ProcessedDeviceData:
        """Process device data with validation and transformation"""
        pass
    
    @abstractmethod
    def detect_emergency(self, device_data: Dict[str, Any], context: ProcessingContext) -> Tuple[bool, str]:
        """Detect emergency conditions in device data"""
        pass
    
    def get_processor_info(self) -> Dict[str, Any]:
        """Get processor metadata and statistics"""
        return {
            "processor_id": self.processor_id,
            "version": self.version,
            "created_at": self.created_at.isoformat(),
            "supported_types": [dt.value for dt in self.get_supported_device_types()],
            "processing_count": self.processing_count,
            "error_count": self.error_count,
            "error_rate": self.error_count / max(self.processing_count, 1),
            "last_error": self.last_error,
            "health_status": "healthy" if self.error_count / max(self.processing_count, 1) < 0.1 else "degraded"
        }
    
    def _update_stats(self, success: bool, error_msg: Optional[str] = None):
        """Update processor statistics"""
        self.processing_count += 1
        if not success:
            self.error_count += 1
            self.last_error = error_msg


class HeartRateProcessor(AbstractDeviceProcessor):
    """Heart rate device processor with medical-grade validation"""
    
    def __init__(self):
        super().__init__("heart_rate_processor", "1.0.0")
    
    def get_supported_device_types(self) -> List[DeviceType]:
        return [DeviceType.HEART_RATE]
    
    def get_device_capabilities(self, device_type: DeviceType) -> DeviceCapability:
        if device_type != DeviceType.HEART_RATE:
            raise ValueError(f"Unsupported device type: {device_type}")
        
        return DeviceCapability(
            device_type=DeviceType.HEART_RATE,
            parameter_types=["heart_rate", "rr_interval"],
            medical_grade=True,
            real_time_required=True,
            batch_eligible=False,
            emergency_detection=True,
            validation_rules={
                "heart_rate": {"min": 30, "max": 220, "unit": "bpm"},
                "emergency_thresholds": {"low": 40, "high": 180}
            },
            accuracy_specs={
                "heart_rate": {"accuracy": "±2 bpm", "precision": "1 bpm"}
            },
            compliance_metadata={
                "medical_device": True,
                "fda_approved": True,
                "hipaa_required": True
            }
        )
    
    def can_process(self, device_data: Dict[str, Any], context: ProcessingContext) -> bool:
        """Check if this is heart rate data"""
        reading_type = device_data.get("reading_type", "").lower()
        return reading_type in ["heart_rate", "hr", "pulse", "bpm"]
    
    def validate_data(self, device_data: Dict[str, Any], context: ProcessingContext) -> ValidationResult:
        """Validate heart rate data with medical standards"""
        warnings = []
        errors = []
        emergency_detected = False
        
        # Extract heart rate value
        value = device_data.get("value")
        if value is None:
            errors.append("Missing heart rate value")
            return ValidationResult(
                is_valid=False,
                criticality=ParameterCriticality.ROUTINE,
                warnings=warnings,
                errors=errors,
                emergency_detected=False,
                quality_score=0.0,
                recommendations=["Provide heart rate value"]
            )
        
        try:
            hr_value = float(value)
        except (ValueError, TypeError):
            errors.append(f"Invalid heart rate value: {value}")
            return ValidationResult(
                is_valid=False,
                criticality=ParameterCriticality.ROUTINE,
                warnings=warnings,
                errors=errors,
                emergency_detected=False,
                quality_score=0.0,
                recommendations=["Provide numeric heart rate value"]
            )
        
        # Validate range
        if hr_value < 30 or hr_value > 220:
            errors.append(f"Heart rate {hr_value} outside valid range (30-220 bpm)")
        
        # Check for emergency conditions
        criticality = ParameterCriticality.ROUTINE
        if hr_value < 40:
            emergency_detected = True
            criticality = ParameterCriticality.EMERGENCY
            warnings.append(f"Bradycardia detected: {hr_value} bpm")
        elif hr_value > 180:
            emergency_detected = True
            criticality = ParameterCriticality.EMERGENCY
            warnings.append(f"Tachycardia detected: {hr_value} bpm")
        elif hr_value < 50 or hr_value > 150:
            criticality = ParameterCriticality.CRITICAL
            warnings.append(f"Heart rate outside normal range: {hr_value} bpm")
        
        # Calculate quality score
        quality_score = 1.0
        if warnings:
            quality_score -= 0.1 * len(warnings)
        if errors:
            quality_score = 0.0
        
        return ValidationResult(
            is_valid=len(errors) == 0,
            criticality=criticality,
            warnings=warnings,
            errors=errors,
            emergency_detected=emergency_detected,
            quality_score=max(0.0, quality_score),
            recommendations=self._get_recommendations(hr_value, criticality)
        )
    
    def process_data(self, device_data: Dict[str, Any], context: ProcessingContext) -> ProcessedDeviceData:
        """Process heart rate data with medical validation"""
        start_time = time.time()
        
        try:
            # Validate data
            validation_result = self.validate_data(device_data, context)
            
            # Process data
            processed_data = {
                **device_data,
                "device_type": DeviceType.HEART_RATE.value,
                "parameter_type": "heart_rate",
                "medical_grade": True,
                "processed_at": datetime.now(timezone.utc).isoformat(),
                "processor_id": self.processor_id,
                "processor_version": self.version
            }
            
            # Add medical metadata
            if validation_result.is_valid:
                hr_value = float(device_data["value"])
                processed_data.update({
                    "heart_rate_zone": self._get_heart_rate_zone(hr_value),
                    "medical_interpretation": self._get_medical_interpretation(hr_value),
                    "clinical_significance": validation_result.criticality.value
                })
            
            processing_result = ProcessingResult.SUCCESS
            if validation_result.emergency_detected:
                processing_result = ProcessingResult.EMERGENCY
            elif validation_result.warnings:
                processing_result = ProcessingResult.WARNING
            elif not validation_result.is_valid:
                processing_result = ProcessingResult.ERROR
            
            processing_time = (time.time() - start_time) * 1000
            
            self._update_stats(True)
            
            return ProcessedDeviceData(
                original_data=device_data,
                processed_data=processed_data,
                validation_result=validation_result,
                processing_result=processing_result,
                metadata={
                    "processor_type": "heart_rate",
                    "medical_grade": True,
                    "emergency_capable": True
                },
                processing_time_ms=processing_time,
                processor_version=self.version
            )
            
        except Exception as e:
            self._update_stats(False, str(e))
            logger.error(f"Heart rate processing failed: {e}")
            raise
    
    def detect_emergency(self, device_data: Dict[str, Any], context: ProcessingContext) -> Tuple[bool, str]:
        """Detect heart rate emergencies"""
        try:
            value = float(device_data.get("value", 0))
            
            if value < 40:
                return True, f"Severe bradycardia: {value} bpm - Immediate medical attention required"
            elif value > 180:
                return True, f"Severe tachycardia: {value} bpm - Immediate medical attention required"
            
            return False, ""
            
        except (ValueError, TypeError):
            return False, ""
    
    def _get_heart_rate_zone(self, hr_value: float) -> str:
        """Get heart rate training zone"""
        if hr_value < 60:
            return "resting"
        elif hr_value < 100:
            return "light"
        elif hr_value < 140:
            return "moderate"
        elif hr_value < 180:
            return "vigorous"
        else:
            return "maximum"
    
    def _get_medical_interpretation(self, hr_value: float) -> str:
        """Get medical interpretation of heart rate"""
        if hr_value < 40:
            return "severe_bradycardia"
        elif hr_value < 60:
            return "bradycardia"
        elif hr_value <= 100:
            return "normal"
        elif hr_value <= 150:
            return "elevated"
        elif hr_value <= 180:
            return "tachycardia"
        else:
            return "severe_tachycardia"
    
    def _get_recommendations(self, hr_value: float, criticality: ParameterCriticality) -> List[str]:
        """Get medical recommendations based on heart rate"""
        recommendations = []
        
        if criticality == ParameterCriticality.EMERGENCY:
            recommendations.append("Seek immediate medical attention")
            recommendations.append("Contact emergency services if symptoms present")
        elif criticality == ParameterCriticality.CRITICAL:
            recommendations.append("Consult healthcare provider")
            recommendations.append("Monitor closely for symptoms")
        
        return recommendations
