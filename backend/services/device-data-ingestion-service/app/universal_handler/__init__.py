"""
Universal Device Handler Module

Single intelligent handler for all device types with processor pattern,
dynamic routing, and medical-grade processing capabilities.

This module implements the Universal Device Handler architecture with:
- Abstract device processor interface
- Dynamic device registry with hot-pluggable support
- Intelligent routing engine with fallback mechanisms
- Medical-grade parameter classification
- Performance optimization and monitoring
"""

from .device_processor import (
    AbstractDeviceProcessor,
    DeviceType,
    ParameterCriticality,
    ProcessingResult,
    DeviceCapability,
    ProcessingContext,
    ValidationResult,
    ProcessedDeviceData,
    HeartRateProcessor
)

from .device_registry import (
    DeviceRegistry,
    ProcessorRegistration,
    RegistryStats,
    get_device_registry,
    cleanup_device_registry
)

from .routing_engine import (
    UniversalRoutingEngine,
    RoutingStrategy,
    RoutingResult,
    DevicePattern,
    get_routing_engine
)

from .universal_handler import (
    UniversalDeviceHandler,
    UniversalProcessingResult,
    get_universal_handler
)

__all__ = [
    # Device Processor
    "AbstractDeviceProcessor",
    "DeviceType", 
    "ParameterCriticality",
    "ProcessingResult",
    "DeviceCapability",
    "ProcessingContext",
    "ValidationResult",
    "ProcessedDeviceData",
    "HeartRateProcessor",
    
    # Device Registry
    "DeviceRegistry",
    "ProcessorRegistration",
    "RegistryStats",
    "get_device_registry",
    "cleanup_device_registry",
    
    # Routing Engine
    "UniversalRoutingEngine",
    "RoutingStrategy",
    "RoutingResult", 
    "DevicePattern",
    "get_routing_engine",
    
    # Universal Handler
    "UniversalDeviceHandler",
    "UniversalProcessingResult",
    "get_universal_handler"
]
