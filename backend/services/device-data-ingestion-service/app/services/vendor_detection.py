"""
Vendor Detection Service for Medical Devices

Integrates with Universal Device Handler to automatically detect:
1. Device vendor (Fitbit, Garmin, Apple Health, etc.)
2. Device type (heart_rate, blood_pressure, etc.)
3. Medical grade classification
4. Appropriate outbox table routing

This ensures all medical devices are properly routed to the correct
vendor-specific outbox table for true fault isolation.
"""
import logging
import re
from typing import Dict, Any, Optional, Tuple
from dataclasses import dataclass

from app.universal_handler.universal_handler import get_universal_handler
from app.universal_handler.device_processor import DeviceType
from app.db.models import SUPPORTED_VENDORS, is_supported_vendor

logger = logging.getLogger(__name__)


@dataclass
class VendorDetectionResult:
    """Result of vendor detection process"""
    vendor_id: str
    vendor_name: str
    device_type: str
    confidence: float
    is_medical_grade: bool
    outbox_table: str
    dead_letter_table: str
    detection_method: str
    metadata: Dict[str, Any]


class VendorDetectionService:
    """
    Service for detecting device vendor and routing to appropriate outbox table
    
    Detection Methods:
    1. Explicit vendor identification (device_id, metadata)
    2. Universal device handler classification
    3. Device type pattern matching
    4. Fallback to generic_device
    """
    
    def __init__(self):
        self.universal_handler = None
        self._vendor_patterns = self._build_vendor_patterns()
        self._device_type_mapping = self._build_device_type_mapping()
    
    async def _get_universal_handler(self):
        """Lazy load universal handler"""
        if self.universal_handler is None:
            self.universal_handler = await get_universal_handler()
        return self.universal_handler
    
    def _build_vendor_patterns(self) -> Dict[str, list]:
        """Build regex patterns for vendor detection"""
        return {
            "fitbit": [
                r"fitbit",
                r"fb_",
                r"charge\d+",
                r"versa\d*",
                r"ionic",
                r"inspire\d*"
            ],
            "garmin": [
                r"garmin",
                r"forerunner\d+",
                r"fenix\d*",
                r"vivoactive\d*",
                r"vivosmart\d*"
            ],
            "apple_health": [
                r"apple",
                r"iphone",
                r"apple_watch",
                r"healthkit",
                r"com\.apple\."
            ],
            "samsung_health": [
                r"samsung",
                r"galaxy_watch",
                r"gear_s\d*",
                r"samsung_health"
            ],
            "withings": [
                r"withings",
                r"nokia_health",
                r"body\+*",
                r"bpm\+*"
            ],
            "omron": [
                r"omron",
                r"bp\d+",
                r"hem-\d+"
            ],
            "polar": [
                r"polar",
                r"h\d+",
                r"vantage",
                r"ignite"
            ],
            "suunto": [
                r"suunto",
                r"spartan",
                r"ambit\d*"
            ]
        }
    
    def _build_device_type_mapping(self) -> Dict[str, str]:
        """Map device types to medical categories"""
        return {
            DeviceType.HEART_RATE.value: "cardiovascular",
            DeviceType.BLOOD_PRESSURE.value: "cardiovascular", 
            DeviceType.ECG.value: "cardiovascular",
            DeviceType.OXYGEN_SATURATION.value: "respiratory",
            DeviceType.BLOOD_GLUCOSE.value: "metabolic",
            DeviceType.TEMPERATURE.value: "vital_signs",
            DeviceType.WEIGHT.value: "anthropometric",
            DeviceType.STEPS.value: "activity",
            DeviceType.SLEEP_DURATION.value: "sleep"
        }
    
    async def detect_vendor_and_route(self, device_data: Dict[str, Any]) -> VendorDetectionResult:
        """
        Main method to detect vendor and determine routing
        
        Args:
            device_data: Raw device data from ingestion
            
        Returns:
            VendorDetectionResult with routing information
        """
        # Method 1: Explicit vendor detection
        explicit_result = await self._detect_explicit_vendor(device_data)
        if explicit_result and explicit_result.confidence > 0.8:
            return explicit_result
        
        # Method 2: Universal device handler classification
        universal_result = await self._detect_via_universal_handler(device_data)
        if universal_result and universal_result.confidence > 0.6:
            return universal_result
        
        # Method 3: Pattern matching
        pattern_result = await self._detect_via_patterns(device_data)
        if pattern_result and pattern_result.confidence > 0.5:
            return pattern_result
        
        # Method 4: Fallback to generic device
        return await self._fallback_to_generic(device_data)
    
    async def _detect_explicit_vendor(self, device_data: Dict[str, Any]) -> Optional[VendorDetectionResult]:
        """Detect vendor from explicit fields"""
        try:
            # Check metadata for vendor information
            metadata = device_data.get("metadata", {})
            
            # Direct vendor specification
            if "vendor" in metadata:
                vendor_id = metadata["vendor"].lower().replace(" ", "_")
                if is_supported_vendor(vendor_id):
                    return await self._create_detection_result(
                        vendor_id=vendor_id,
                        device_data=device_data,
                        confidence=0.95,
                        detection_method="explicit_metadata"
                    )
            
            # Check device_id for vendor prefixes
            device_id = device_data.get("device_id", "").lower()
            for vendor_id in SUPPORTED_VENDORS.keys():
                if device_id.startswith(vendor_id) or vendor_id in device_id:
                    return await self._create_detection_result(
                        vendor_id=vendor_id,
                        device_data=device_data,
                        confidence=0.9,
                        detection_method="device_id_prefix"
                    )
            
            return None
            
        except Exception as e:
            logger.error(f"Error in explicit vendor detection: {e}")
            return None
    
    async def _detect_via_universal_handler(self, device_data: Dict[str, Any]) -> Optional[VendorDetectionResult]:
        """Use universal device handler for classification"""
        try:
            handler = await self._get_universal_handler()
            
            # Get device type classification
            classification = await handler.detect_device_type(device_data)
            device_type = classification.get("device_type")
            confidence = classification.get("confidence", 0.0)
            
            if not device_type or confidence < 0.5:
                return None
            
            # Determine best vendor based on device type and capabilities
            vendor_id = await self._select_vendor_for_device_type(device_type, device_data)
            
            return await self._create_detection_result(
                vendor_id=vendor_id,
                device_data=device_data,
                confidence=confidence * 0.8,  # Slightly reduce confidence
                detection_method="universal_handler",
                device_type=device_type
            )
            
        except Exception as e:
            logger.error(f"Error in universal handler detection: {e}")
            return None
    
    async def _detect_via_patterns(self, device_data: Dict[str, Any]) -> Optional[VendorDetectionResult]:
        """Use regex patterns for vendor detection"""
        try:
            # Combine all text fields for pattern matching
            text_fields = [
                device_data.get("device_id", ""),
                device_data.get("device_name", ""),
                str(device_data.get("metadata", {})),
                device_data.get("source", "")
            ]
            
            combined_text = " ".join(text_fields).lower()
            
            # Test patterns for each vendor
            best_match = None
            best_score = 0.0
            
            for vendor_id, patterns in self._vendor_patterns.items():
                score = 0.0
                matches = 0
                
                for pattern in patterns:
                    if re.search(pattern, combined_text, re.IGNORECASE):
                        matches += 1
                        score += 1.0 / len(patterns)  # Weight by pattern count
                
                if matches > 0:
                    # Boost score for multiple matches
                    final_score = min(score * (1 + matches * 0.1), 1.0)
                    
                    if final_score > best_score:
                        best_score = final_score
                        best_match = vendor_id
            
            if best_match and best_score > 0.3:
                return await self._create_detection_result(
                    vendor_id=best_match,
                    device_data=device_data,
                    confidence=best_score,
                    detection_method="pattern_matching"
                )
            
            return None
            
        except Exception as e:
            logger.error(f"Error in pattern detection: {e}")
            return None
    
    async def _fallback_to_generic(self, device_data: Dict[str, Any]) -> VendorDetectionResult:
        """Fallback to generic device vendor"""
        return await self._create_detection_result(
            vendor_id="generic_device",
            device_data=device_data,
            confidence=0.1,
            detection_method="fallback"
        )
    
    async def _select_vendor_for_device_type(self, device_type: str, device_data: Dict[str, Any]) -> str:
        """Select best vendor based on device type and medical grade"""
        metadata = device_data.get("metadata", {})
        is_medical_grade = metadata.get("medical_grade", False)
        
        # Medical grade devices go to medical_device vendor
        if is_medical_grade:
            medical_device_types = SUPPORTED_VENDORS["medical_device"]["device_types"]
            if device_type in medical_device_types:
                return "medical_device"
        
        # Blood pressure devices often go to Omron or Withings
        if device_type == "blood_pressure":
            return "withings"  # Default for BP devices
        
        # Blood glucose devices go to medical_device
        if device_type == "blood_glucose":
            return "medical_device"
        
        # ECG devices go to Apple Health or medical_device
        if device_type == "ecg":
            return "apple_health"
        
        # Default fitness devices go to Fitbit
        if device_type in ["heart_rate", "steps", "sleep_duration", "weight"]:
            return "fitbit"
        
        # Everything else goes to generic
        return "generic_device"
    
    async def _create_detection_result(
        self, 
        vendor_id: str, 
        device_data: Dict[str, Any], 
        confidence: float,
        detection_method: str,
        device_type: Optional[str] = None
    ) -> VendorDetectionResult:
        """Create a detection result object"""
        
        vendor_config = SUPPORTED_VENDORS.get(vendor_id, SUPPORTED_VENDORS["generic_device"])
        
        # Determine device type if not provided
        if not device_type:
            device_type = device_data.get("reading_type", "unknown")
        
        # Determine if medical grade
        metadata = device_data.get("metadata", {})
        is_medical_grade = (
            metadata.get("medical_grade", False) or
            vendor_id == "medical_device" or
            device_type in ["blood_pressure", "blood_glucose", "ecg", "temperature"]
        )
        
        return VendorDetectionResult(
            vendor_id=vendor_id,
            vendor_name=vendor_config.get("vendor_name", vendor_id.replace("_", " ").title()),
            device_type=device_type,
            confidence=confidence,
            is_medical_grade=is_medical_grade,
            outbox_table=vendor_config["outbox_table"],
            dead_letter_table=vendor_config["dead_letter_table"],
            detection_method=detection_method,
            metadata={
                "supported_device_types": vendor_config.get("device_types", []),
                "kafka_topic": vendor_config.get("kafka_topic", "raw-device-data.v1"),
                "original_device_data_keys": list(device_data.keys())
            }
        )
    
    async def get_vendor_capabilities(self, vendor_id: str) -> Dict[str, Any]:
        """Get capabilities for a specific vendor"""
        if not is_supported_vendor(vendor_id):
            return {}
        
        vendor_config = SUPPORTED_VENDORS[vendor_id]
        return {
            "vendor_id": vendor_id,
            "supported_device_types": vendor_config.get("device_types", []),
            "outbox_table": vendor_config["outbox_table"],
            "dead_letter_table": vendor_config["dead_letter_table"],
            "kafka_topic": vendor_config.get("kafka_topic", "raw-device-data.v1"),
            "is_medical_grade": vendor_id in ["medical_device", "withings", "omron"]
        }
    
    async def list_all_supported_vendors(self) -> Dict[str, Dict[str, Any]]:
        """List all supported vendors and their capabilities"""
        result = {}
        for vendor_id in SUPPORTED_VENDORS.keys():
            result[vendor_id] = await self.get_vendor_capabilities(vendor_id)
        return result


# Global vendor detection service instance
vendor_detection_service = VendorDetectionService()
