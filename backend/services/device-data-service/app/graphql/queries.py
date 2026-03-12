"""
GraphQL queries for Device Data Service
"""
import strawberry
from typing import List, Optional
from .types import (
    DeviceReading, ReadingConnection, ReadingStats, Device,
    ReadingType, AlertLevel, ReadingCategory
)


@strawberry.type
class Query:
    """GraphQL queries for device data"""
    
    @strawberry.field
    async def device_reading(self, id: str) -> Optional[DeviceReading]:
        """Get a specific device reading by ID"""
        from ..services.device_data_service import get_device_data_service
        
        service = get_device_data_service()
        return await service.get_reading_by_id(id)
    
    @strawberry.field
    async def device_readings(
        self,
        patient_id: Optional[str] = None,
        device_id: Optional[str] = None,
        reading_type: Optional[ReadingType] = None,
        alert_level: Optional[AlertLevel] = None,
        reading_category: Optional[ReadingCategory] = None,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        page: int = 1,
        limit: int = 50
    ) -> ReadingConnection:
        """Search device readings with filters"""
        from ..services.device_data_service import get_device_data_service
        
        service = get_device_data_service()
        
        if patient_id:
            return await service.get_patient_readings(
                patient_id=patient_id,
                reading_type=reading_type.value if reading_type else None,
                alert_level=alert_level.value if alert_level else None,
                start_date=start_date,
                end_date=end_date,
                page=page,
                limit=limit
            )
        elif device_id:
            return await service.get_device_readings(
                device_id=device_id,
                reading_type=reading_type.value if reading_type else None,
                start_date=start_date,
                end_date=end_date,
                page=page,
                limit=limit
            )
        else:
            return await service.search_readings(
                reading_type=reading_type.value if reading_type else None,
                alert_level=alert_level.value if alert_level else None,
                reading_category=reading_category.value if reading_category else None,
                start_date=start_date,
                end_date=end_date,
                page=page,
                limit=limit
            )
    
    @strawberry.field
    async def critical_readings(
        self,
        hours: int = 24,
        limit: int = 100
    ) -> List[DeviceReading]:
        """Get all critical readings in the last N hours"""
        from ..services.device_data_service import get_device_data_service
        from datetime import datetime, timedelta
        
        end_date = datetime.utcnow()
        start_date = end_date - timedelta(hours=hours)
        
        service = get_device_data_service()
        result = await service.search_readings(
            alert_level="critical",
            start_date=start_date.isoformat(),
            end_date=end_date.isoformat(),
            page=1,
            limit=limit
        )
        return result.items
    
    @strawberry.field
    async def reading_stats(
        self,
        patient_id: Optional[str] = None,
        device_id: Optional[str] = None,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None
    ) -> ReadingStats:
        """Get reading statistics"""
        from ..services.device_data_service import get_device_data_service
        
        service = get_device_data_service()
        
        if patient_id:
            return await service.get_patient_reading_stats(
                patient_id=patient_id,
                start_date=start_date,
                end_date=end_date
            )
        elif device_id:
            return await service.get_device_reading_stats(
                device_id=device_id,
                start_date=start_date,
                end_date=end_date
            )
        else:
            return await service.get_global_reading_stats(
                start_date=start_date,
                end_date=end_date
            )
    
    @strawberry.field
    async def device(self, id: str) -> Optional[Device]:
        """Get device information"""
        # For now, just return a Device with the ID
        # In a full implementation, this would fetch device metadata
        return Device(id=id)
    
    @strawberry.field
    async def devices_for_patient(self, patient_id: str) -> List[Device]:
        """Get all devices associated with a patient"""
        from ..services.device_data_service import get_device_data_service
        
        service = get_device_data_service()
        device_ids = await service.get_patient_devices(patient_id)
        return [Device(id=device_id) for device_id in device_ids]
