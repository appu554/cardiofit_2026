"""
Universal Device Handler - Device Registry

Dynamic device processor discovery and registration system with hot-pluggable
device support and health monitoring.
"""

import logging
import asyncio
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Set
from dataclasses import dataclass, field
from collections import defaultdict
import weakref

from .device_processor import (
    AbstractDeviceProcessor, DeviceType, DeviceCapability,
    ProcessingContext, HeartRateProcessor
)

logger = logging.getLogger(__name__)


@dataclass
class ProcessorRegistration:
    """Processor registration metadata"""
    processor: AbstractDeviceProcessor
    registered_at: datetime
    last_health_check: datetime
    health_status: str
    processing_count: int = 0
    error_count: int = 0
    version: str = "1.0.0"
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class RegistryStats:
    """Device registry statistics"""
    total_processors: int
    healthy_processors: int
    degraded_processors: int
    failed_processors: int
    total_device_types: int
    processing_count: int
    error_count: int
    uptime_seconds: float


class DeviceRegistry:
    """
    Dynamic device processor registry with hot-pluggable support
    
    Manages device processor lifecycle, health monitoring, and discovery
    for the universal device handler system.
    """
    
    def __init__(self):
        self.processors: Dict[str, ProcessorRegistration] = {}
        self.device_type_mapping: Dict[DeviceType, List[str]] = defaultdict(list)
        self.capability_cache: Dict[str, DeviceCapability] = {}
        self.health_check_interval = 60  # seconds
        self.health_check_task: Optional[asyncio.Task] = None
        self.created_at = datetime.now(timezone.utc)
        self._lock = asyncio.Lock()
        
        # Initialize with built-in processors
        self._register_builtin_processors()
    
    def _register_builtin_processors(self):
        """Register built-in device processors"""
        try:
            # Register heart rate processor
            hr_processor = HeartRateProcessor()
            self.register_processor(hr_processor)
            logger.info("Built-in processors registered successfully")
        except Exception as e:
            logger.error(f"Failed to register built-in processors: {e}")
    
    async def initialize(self):
        """Initialize the device registry"""
        # Start health monitoring
        self.health_check_task = asyncio.create_task(self._health_monitor())
        logger.info("Device registry initialized with health monitoring")
    
    async def cleanup(self):
        """Cleanup registry resources"""
        if self.health_check_task:
            self.health_check_task.cancel()
            try:
                await self.health_check_task
            except asyncio.CancelledError:
                pass
        logger.info("Device registry cleaned up")
    
    def register_processor(self, processor: AbstractDeviceProcessor, metadata: Optional[Dict[str, Any]] = None) -> bool:
        """
        Register a device processor
        
        Args:
            processor: Device processor instance
            metadata: Optional metadata for the processor
            
        Returns:
            bool: True if registration successful
        """
        try:
            processor_id = processor.processor_id
            
            # Check if processor already registered
            if processor_id in self.processors:
                logger.warning(f"Processor {processor_id} already registered, updating...")
                self.unregister_processor(processor_id)
            
            # Create registration
            registration = ProcessorRegistration(
                processor=processor,
                registered_at=datetime.now(timezone.utc),
                last_health_check=datetime.now(timezone.utc),
                health_status="healthy",
                version=processor.version,
                metadata=metadata or {}
            )
            
            # Register processor
            self.processors[processor_id] = registration
            
            # Update device type mapping
            for device_type in processor.get_supported_device_types():
                self.device_type_mapping[device_type].append(processor_id)
                
                # Cache capabilities
                try:
                    capability = processor.get_device_capabilities(device_type)
                    cache_key = f"{processor_id}:{device_type.value}"
                    self.capability_cache[cache_key] = capability
                except Exception as e:
                    logger.warning(f"Failed to cache capabilities for {processor_id}:{device_type}: {e}")
            
            logger.info(f"Processor {processor_id} registered successfully")
            return True
            
        except Exception as e:
            logger.error(f"Failed to register processor {processor.processor_id}: {e}")
            return False
    
    def unregister_processor(self, processor_id: str) -> bool:
        """
        Unregister a device processor
        
        Args:
            processor_id: ID of processor to unregister
            
        Returns:
            bool: True if unregistration successful
        """
        try:
            if processor_id not in self.processors:
                logger.warning(f"Processor {processor_id} not found for unregistration")
                return False
            
            registration = self.processors[processor_id]
            processor = registration.processor
            
            # Remove from device type mapping
            for device_type in processor.get_supported_device_types():
                if processor_id in self.device_type_mapping[device_type]:
                    self.device_type_mapping[device_type].remove(processor_id)
                
                # Remove from capability cache
                cache_key = f"{processor_id}:{device_type.value}"
                self.capability_cache.pop(cache_key, None)
            
            # Remove processor
            del self.processors[processor_id]
            
            logger.info(f"Processor {processor_id} unregistered successfully")
            return True
            
        except Exception as e:
            logger.error(f"Failed to unregister processor {processor_id}: {e}")
            return False
    
    def get_processors_for_device_type(self, device_type: DeviceType) -> List[AbstractDeviceProcessor]:
        """
        Get all processors that can handle a specific device type
        
        Args:
            device_type: Device type to find processors for
            
        Returns:
            List of processors that can handle the device type
        """
        processors = []
        processor_ids = self.device_type_mapping.get(device_type, [])
        
        for processor_id in processor_ids:
            if processor_id in self.processors:
                registration = self.processors[processor_id]
                if registration.health_status == "healthy":
                    processors.append(registration.processor)
        
        return processors
    
    def get_best_processor_for_device(self, device_data: Dict[str, Any], context: ProcessingContext) -> Optional[AbstractDeviceProcessor]:
        """
        Get the best processor for specific device data
        
        Args:
            device_data: Device data to process
            context: Processing context
            
        Returns:
            Best processor for the device data, or None if no suitable processor found
        """
        # Try processors for the specified device type first
        if context.device_type != DeviceType.UNKNOWN:
            processors = self.get_processors_for_device_type(context.device_type)
            for processor in processors:
                if processor.can_process(device_data, context):
                    return processor
        
        # Try all processors if device type is unknown or no specific processor found
        for registration in self.processors.values():
            if registration.health_status == "healthy":
                processor = registration.processor
                if processor.can_process(device_data, context):
                    return processor
        
        return None
    
    def get_device_capabilities(self, processor_id: str, device_type: DeviceType) -> Optional[DeviceCapability]:
        """
        Get cached device capabilities
        
        Args:
            processor_id: Processor ID
            device_type: Device type
            
        Returns:
            Device capabilities or None if not found
        """
        cache_key = f"{processor_id}:{device_type.value}"
        return self.capability_cache.get(cache_key)
    
    def get_all_supported_device_types(self) -> Set[DeviceType]:
        """Get all device types supported by registered processors"""
        return set(self.device_type_mapping.keys())
    
    def get_processor_info(self, processor_id: str) -> Optional[Dict[str, Any]]:
        """
        Get processor information and statistics
        
        Args:
            processor_id: Processor ID
            
        Returns:
            Processor information or None if not found
        """
        if processor_id not in self.processors:
            return None
        
        registration = self.processors[processor_id]
        processor_info = registration.processor.get_processor_info()
        
        return {
            **processor_info,
            "registered_at": registration.registered_at.isoformat(),
            "last_health_check": registration.last_health_check.isoformat(),
            "health_status": registration.health_status,
            "registry_metadata": registration.metadata
        }
    
    def get_registry_stats(self) -> RegistryStats:
        """Get comprehensive registry statistics"""
        healthy_count = sum(1 for r in self.processors.values() if r.health_status == "healthy")
        degraded_count = sum(1 for r in self.processors.values() if r.health_status == "degraded")
        failed_count = sum(1 for r in self.processors.values() if r.health_status == "failed")
        
        total_processing = sum(r.processing_count for r in self.processors.values())
        total_errors = sum(r.error_count for r in self.processors.values())
        
        uptime = (datetime.now(timezone.utc) - self.created_at).total_seconds()
        
        return RegistryStats(
            total_processors=len(self.processors),
            healthy_processors=healthy_count,
            degraded_processors=degraded_count,
            failed_processors=failed_count,
            total_device_types=len(self.device_type_mapping),
            processing_count=total_processing,
            error_count=total_errors,
            uptime_seconds=uptime
        )
    
    async def _health_monitor(self):
        """Background health monitoring for processors"""
        while True:
            try:
                await asyncio.sleep(self.health_check_interval)
                await self._perform_health_checks()
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Health monitoring error: {e}")
    
    async def _perform_health_checks(self):
        """Perform health checks on all registered processors"""
        async with self._lock:
            for processor_id, registration in self.processors.items():
                try:
                    processor = registration.processor
                    processor_info = processor.get_processor_info()
                    
                    # Update health status based on error rate
                    error_rate = processor_info.get("error_rate", 0)
                    if error_rate < 0.05:  # Less than 5% error rate
                        registration.health_status = "healthy"
                    elif error_rate < 0.2:  # Less than 20% error rate
                        registration.health_status = "degraded"
                    else:
                        registration.health_status = "failed"
                    
                    # Update statistics
                    registration.processing_count = processor_info.get("processing_count", 0)
                    registration.error_count = processor_info.get("error_count", 0)
                    registration.last_health_check = datetime.now(timezone.utc)
                    
                except Exception as e:
                    logger.error(f"Health check failed for processor {processor_id}: {e}")
                    registration.health_status = "failed"
    
    def list_processors(self) -> Dict[str, Dict[str, Any]]:
        """List all registered processors with their information"""
        return {
            processor_id: self.get_processor_info(processor_id)
            for processor_id in self.processors.keys()
        }


# Global registry instance
_device_registry: Optional[DeviceRegistry] = None


async def get_device_registry() -> DeviceRegistry:
    """Get or create the global device registry"""
    global _device_registry
    
    if _device_registry is None:
        _device_registry = DeviceRegistry()
        await _device_registry.initialize()
        logger.info("Global device registry initialized")
    
    return _device_registry


async def cleanup_device_registry():
    """Cleanup the global device registry"""
    global _device_registry
    
    if _device_registry is not None:
        await _device_registry.cleanup()
        _device_registry = None
        logger.info("Global device registry cleaned up")
