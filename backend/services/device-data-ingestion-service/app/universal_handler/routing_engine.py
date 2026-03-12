"""
Universal Device Handler - Routing Engine

Intelligent routing engine with device detection, content-based classification,
and fallback mechanisms for universal device processing.
"""

import logging
import time
import re
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Tuple
from dataclasses import dataclass
from enum import Enum

from .device_processor import DeviceType, ProcessingContext, AbstractDeviceProcessor
from .device_registry import get_device_registry

logger = logging.getLogger(__name__)


class RoutingStrategy(Enum):
    """Routing strategy types"""
    DIRECT_MAPPING = "direct_mapping"          # Explicit device type mapping
    CONTENT_DETECTION = "content_detection"    # Content-based device detection
    PATTERN_MATCHING = "pattern_matching"      # Pattern-based classification
    FALLBACK_GENERIC = "fallback_generic"      # Generic processor fallback


@dataclass
class RoutingResult:
    """Result of routing decision"""
    processor: Optional[AbstractDeviceProcessor]
    device_type: DeviceType
    confidence: float
    strategy_used: RoutingStrategy
    routing_time_ms: float
    fallback_used: bool
    detection_metadata: Dict[str, Any]


@dataclass
class DevicePattern:
    """Device detection pattern"""
    device_type: DeviceType
    patterns: List[str]
    required_fields: List[str]
    optional_fields: List[str]
    value_patterns: Dict[str, str]
    confidence_weight: float


class UniversalRoutingEngine:
    """
    Intelligent routing engine for universal device processing
    
    Provides device type detection, processor selection, and fallback
    mechanisms with sub-5ms routing performance target.
    """
    
    def __init__(self):
        self.routing_cache: Dict[str, RoutingResult] = {}
        self.cache_ttl = 300  # 5 minutes
        self.routing_stats = {
            "total_routes": 0,
            "cache_hits": 0,
            "direct_mappings": 0,
            "content_detections": 0,
            "pattern_matches": 0,
            "fallback_uses": 0,
            "routing_times": []
        }
        
        # Initialize device detection patterns
        self.device_patterns = self._initialize_device_patterns()
        
        # Direct mapping for explicit device types
        self.direct_mappings = {
            "heart_rate": DeviceType.HEART_RATE,
            "hr": DeviceType.HEART_RATE,
            "pulse": DeviceType.HEART_RATE,
            "bpm": DeviceType.HEART_RATE,
            "blood_pressure": DeviceType.BLOOD_PRESSURE,
            "bp": DeviceType.BLOOD_PRESSURE,
            "systolic": DeviceType.BLOOD_PRESSURE,
            "diastolic": DeviceType.BLOOD_PRESSURE,
            "blood_glucose": DeviceType.BLOOD_GLUCOSE,
            "glucose": DeviceType.BLOOD_GLUCOSE,
            "bg": DeviceType.BLOOD_GLUCOSE,
            "weight": DeviceType.WEIGHT,
            "steps": DeviceType.STEPS,
            "step_count": DeviceType.STEPS,
            "sleep": DeviceType.SLEEP_DURATION,
            "sleep_duration": DeviceType.SLEEP_DURATION,
            "ecg": DeviceType.ECG,
            "electrocardiogram": DeviceType.ECG,
            "temperature": DeviceType.TEMPERATURE,
            "temp": DeviceType.TEMPERATURE,
            "oxygen_saturation": DeviceType.OXYGEN_SATURATION,
            "spo2": DeviceType.OXYGEN_SATURATION,
            "pulse_ox": DeviceType.OXYGEN_SATURATION
        }
    
    def _initialize_device_patterns(self) -> List[DevicePattern]:
        """Initialize device detection patterns"""
        return [
            DevicePattern(
                device_type=DeviceType.HEART_RATE,
                patterns=["heart.*rate", "pulse", "bpm", "beats.*minute"],
                required_fields=["value"],
                optional_fields=["unit", "rr_interval"],
                value_patterns={"unit": r"bpm|beats.*min"},
                confidence_weight=0.9
            ),
            DevicePattern(
                device_type=DeviceType.BLOOD_PRESSURE,
                patterns=["blood.*pressure", "systolic", "diastolic", "mmhg"],
                required_fields=["value"],
                optional_fields=["systolic", "diastolic", "unit"],
                value_patterns={"unit": r"mmhg|mmHg"},
                confidence_weight=0.9
            ),
            DevicePattern(
                device_type=DeviceType.BLOOD_GLUCOSE,
                patterns=["glucose", "blood.*sugar", "mg.*dl", "mmol.*l"],
                required_fields=["value"],
                optional_fields=["unit", "meal_context"],
                value_patterns={"unit": r"mg/dl|mmol/l"},
                confidence_weight=0.9
            ),
            DevicePattern(
                device_type=DeviceType.WEIGHT,
                patterns=["weight", "mass", "kg", "lbs", "pounds"],
                required_fields=["value"],
                optional_fields=["unit", "bmi"],
                value_patterns={"unit": r"kg|lbs|pounds"},
                confidence_weight=0.8
            ),
            DevicePattern(
                device_type=DeviceType.STEPS,
                patterns=["steps", "step.*count", "walking", "activity"],
                required_fields=["value"],
                optional_fields=["distance", "calories"],
                value_patterns={},
                confidence_weight=0.8
            ),
            DevicePattern(
                device_type=DeviceType.TEMPERATURE,
                patterns=["temperature", "temp", "fever", "celsius", "fahrenheit"],
                required_fields=["value"],
                optional_fields=["unit", "location"],
                value_patterns={"unit": r"°?[cf]|celsius|fahrenheit"},
                confidence_weight=0.9
            ),
            DevicePattern(
                device_type=DeviceType.OXYGEN_SATURATION,
                patterns=["oxygen.*saturation", "spo2", "pulse.*ox", "o2.*sat"],
                required_fields=["value"],
                optional_fields=["unit"],
                value_patterns={"unit": r"%|percent"},
                confidence_weight=0.9
            )
        ]
    
    async def route_device_data(self, device_data: Dict[str, Any], context: ProcessingContext) -> RoutingResult:
        """
        Route device data to appropriate processor
        
        Args:
            device_data: Device data to route
            context: Processing context
            
        Returns:
            RoutingResult with processor and routing metadata
        """
        start_time = time.time()
        
        try:
            # Check cache first
            cache_key = self._generate_cache_key(device_data, context)
            cached_result = self._get_cached_result(cache_key)
            if cached_result:
                self.routing_stats["cache_hits"] += 1
                return cached_result
            
            # Get device registry
            registry = await get_device_registry()
            
            # Strategy 1: Direct mapping
            device_type, confidence = self._direct_mapping_detection(device_data, context)
            if device_type != DeviceType.UNKNOWN and confidence > 0.8:
                processor = self._get_processor_for_device_type(registry, device_type, device_data, context)
                if processor:
                    result = self._create_routing_result(
                        processor, device_type, confidence, RoutingStrategy.DIRECT_MAPPING,
                        start_time, False, {"mapping_source": "direct"}
                    )
                    self._cache_result(cache_key, result)
                    self.routing_stats["direct_mappings"] += 1
                    return result
            
            # Strategy 2: Content-based detection
            device_type, confidence = self._content_based_detection(device_data, context)
            if device_type != DeviceType.UNKNOWN and confidence > 0.7:
                processor = self._get_processor_for_device_type(registry, device_type, device_data, context)
                if processor:
                    result = self._create_routing_result(
                        processor, device_type, confidence, RoutingStrategy.CONTENT_DETECTION,
                        start_time, False, {"detection_source": "content"}
                    )
                    self._cache_result(cache_key, result)
                    self.routing_stats["content_detections"] += 1
                    return result
            
            # Strategy 3: Pattern matching
            device_type, confidence = self._pattern_matching_detection(device_data, context)
            if device_type != DeviceType.UNKNOWN and confidence > 0.6:
                processor = self._get_processor_for_device_type(registry, device_type, device_data, context)
                if processor:
                    result = self._create_routing_result(
                        processor, device_type, confidence, RoutingStrategy.PATTERN_MATCHING,
                        start_time, False, {"detection_source": "pattern"}
                    )
                    self._cache_result(cache_key, result)
                    self.routing_stats["pattern_matches"] += 1
                    return result
            
            # Strategy 4: Fallback to best available processor
            processor = registry.get_best_processor_for_device(device_data, context)
            if processor:
                result = self._create_routing_result(
                    processor, DeviceType.UNKNOWN, 0.5, RoutingStrategy.FALLBACK_GENERIC,
                    start_time, True, {"fallback_reason": "no_specific_match"}
                )
                self._cache_result(cache_key, result)
                self.routing_stats["fallback_uses"] += 1
                return result
            
            # No processor found
            result = self._create_routing_result(
                None, DeviceType.UNKNOWN, 0.0, RoutingStrategy.FALLBACK_GENERIC,
                start_time, True, {"error": "no_processor_found"}
            )
            self.routing_stats["fallback_uses"] += 1
            return result
            
        except Exception as e:
            logger.error(f"Routing error: {e}")
            result = self._create_routing_result(
                None, DeviceType.UNKNOWN, 0.0, RoutingStrategy.FALLBACK_GENERIC,
                start_time, True, {"error": str(e)}
            )
            return result
        
        finally:
            self.routing_stats["total_routes"] += 1
    
    def _direct_mapping_detection(self, device_data: Dict[str, Any], context: ProcessingContext) -> Tuple[DeviceType, float]:
        """Direct mapping based on explicit device type"""
        # Check context device type first
        if context.device_type != DeviceType.UNKNOWN:
            return context.device_type, 1.0
        
        # Check reading_type field
        reading_type = device_data.get("reading_type", "").lower()
        if reading_type in self.direct_mappings:
            return self.direct_mappings[reading_type], 0.95
        
        # Check device_type field
        device_type = device_data.get("device_type", "").lower()
        if device_type in self.direct_mappings:
            return self.direct_mappings[device_type], 0.95
        
        return DeviceType.UNKNOWN, 0.0
    
    def _content_based_detection(self, device_data: Dict[str, Any], context: ProcessingContext) -> Tuple[DeviceType, float]:
        """Content-based device type detection"""
        # Analyze unit field
        unit = device_data.get("unit", "").lower()
        if unit:
            if unit in ["bpm", "beats/min"]:
                return DeviceType.HEART_RATE, 0.9
            elif unit in ["mmhg"]:
                return DeviceType.BLOOD_PRESSURE, 0.9
            elif unit in ["mg/dl", "mmol/l"]:
                return DeviceType.BLOOD_GLUCOSE, 0.9
            elif unit in ["kg", "lbs", "pounds"]:
                return DeviceType.WEIGHT, 0.8
            elif unit in ["°c", "°f", "celsius", "fahrenheit"]:
                return DeviceType.TEMPERATURE, 0.9
            elif unit in ["%", "percent"] and "oxygen" in str(device_data).lower():
                return DeviceType.OXYGEN_SATURATION, 0.9
        
        # Analyze value ranges
        try:
            value = float(device_data.get("value", 0))
            if 30 <= value <= 220 and not unit:
                return DeviceType.HEART_RATE, 0.7  # Likely heart rate range
            elif 50 <= value <= 200 and not unit:
                return DeviceType.BLOOD_PRESSURE, 0.6  # Possible blood pressure
            elif 70 <= value <= 400 and not unit:
                return DeviceType.BLOOD_GLUCOSE, 0.6  # Possible glucose
        except (ValueError, TypeError):
            pass
        
        return DeviceType.UNKNOWN, 0.0
    
    def _pattern_matching_detection(self, device_data: Dict[str, Any], context: ProcessingContext) -> Tuple[DeviceType, float]:
        """Pattern-based device type detection"""
        # Convert all data to searchable text
        search_text = " ".join(str(v).lower() for v in device_data.values() if v is not None)
        
        best_match = DeviceType.UNKNOWN
        best_confidence = 0.0
        
        for pattern in self.device_patterns:
            confidence = 0.0
            
            # Check text patterns
            pattern_matches = 0
            for pattern_text in pattern.patterns:
                if re.search(pattern_text, search_text, re.IGNORECASE):
                    pattern_matches += 1
            
            if pattern_matches > 0:
                confidence += (pattern_matches / len(pattern.patterns)) * 0.6
            
            # Check required fields
            required_matches = sum(1 for field in pattern.required_fields if field in device_data)
            if required_matches == len(pattern.required_fields):
                confidence += 0.3
            
            # Check optional fields
            optional_matches = sum(1 for field in pattern.optional_fields if field in device_data)
            if pattern.optional_fields:
                confidence += (optional_matches / len(pattern.optional_fields)) * 0.1
            
            # Check value patterns
            for field, value_pattern in pattern.value_patterns.items():
                if field in device_data:
                    field_value = str(device_data[field]).lower()
                    if re.search(value_pattern, field_value, re.IGNORECASE):
                        confidence += 0.1
            
            # Apply confidence weight
            confidence *= pattern.confidence_weight
            
            if confidence > best_confidence:
                best_confidence = confidence
                best_match = pattern.device_type
        
        return best_match, best_confidence
    
    def _get_processor_for_device_type(self, registry, device_type: DeviceType, 
                                     device_data: Dict[str, Any], context: ProcessingContext) -> Optional[AbstractDeviceProcessor]:
        """Get the best processor for a specific device type"""
        processors = registry.get_processors_for_device_type(device_type)
        
        # Return the first processor that can handle the data
        for processor in processors:
            if processor.can_process(device_data, context):
                return processor
        
        return None
    
    def _create_routing_result(self, processor: Optional[AbstractDeviceProcessor], device_type: DeviceType,
                             confidence: float, strategy: RoutingStrategy, start_time: float,
                             fallback_used: bool, metadata: Dict[str, Any]) -> RoutingResult:
        """Create routing result with timing"""
        routing_time = (time.time() - start_time) * 1000
        self.routing_stats["routing_times"].append(routing_time)
        
        return RoutingResult(
            processor=processor,
            device_type=device_type,
            confidence=confidence,
            strategy_used=strategy,
            routing_time_ms=routing_time,
            fallback_used=fallback_used,
            detection_metadata=metadata
        )
    
    def _generate_cache_key(self, device_data: Dict[str, Any], context: ProcessingContext) -> str:
        """Generate cache key for routing result"""
        key_parts = [
            device_data.get("reading_type", ""),
            device_data.get("device_type", ""),
            device_data.get("unit", ""),
            context.device_type.value
        ]
        return "|".join(key_parts)
    
    def _get_cached_result(self, cache_key: str) -> Optional[RoutingResult]:
        """Get cached routing result if still valid"""
        if cache_key in self.routing_cache:
            # For simplicity, not implementing TTL check here
            # In production, you'd check timestamp
            return self.routing_cache[cache_key]
        return None
    
    def _cache_result(self, cache_key: str, result: RoutingResult):
        """Cache routing result"""
        # Simple cache without TTL implementation
        # In production, you'd add timestamp and cleanup
        if len(self.routing_cache) < 1000:  # Prevent unlimited growth
            self.routing_cache[cache_key] = result
    
    def get_routing_stats(self) -> Dict[str, Any]:
        """Get routing engine statistics"""
        routing_times = self.routing_stats["routing_times"]
        
        stats = {
            "total_routes": self.routing_stats["total_routes"],
            "cache_hit_rate": (self.routing_stats["cache_hits"] / max(self.routing_stats["total_routes"], 1)) * 100,
            "strategy_distribution": {
                "direct_mappings": self.routing_stats["direct_mappings"],
                "content_detections": self.routing_stats["content_detections"],
                "pattern_matches": self.routing_stats["pattern_matches"],
                "fallback_uses": self.routing_stats["fallback_uses"]
            },
            "cache_size": len(self.routing_cache)
        }
        
        if routing_times:
            stats["performance"] = {
                "avg_routing_time_ms": sum(routing_times) / len(routing_times),
                "min_routing_time_ms": min(routing_times),
                "max_routing_time_ms": max(routing_times),
                "p95_routing_time_ms": sorted(routing_times)[int(len(routing_times) * 0.95)] if len(routing_times) > 20 else max(routing_times)
            }
        
        return stats


# Global routing engine instance
_routing_engine: Optional[UniversalRoutingEngine] = None


def get_routing_engine() -> UniversalRoutingEngine:
    """Get or create the global routing engine"""
    global _routing_engine
    
    if _routing_engine is None:
        _routing_engine = UniversalRoutingEngine()
        logger.info("Global routing engine initialized")
    
    return _routing_engine
