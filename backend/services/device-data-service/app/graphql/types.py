"""
GraphQL types for Device Data Service
"""
import strawberry
from typing import List, Optional
from datetime import datetime
from enum import Enum


@strawberry.enum
class ReadingType(Enum):
    """Types of device readings"""
    HEART_RATE = "heart_rate"
    BLOOD_PRESSURE_SYSTOLIC = "blood_pressure_systolic"
    BLOOD_PRESSURE_DIASTOLIC = "blood_pressure_diastolic"
    BLOOD_GLUCOSE = "blood_glucose"
    TEMPERATURE = "temperature"
    OXYGEN_SATURATION = "oxygen_saturation"
    WEIGHT = "weight"
    STEPS = "steps"
    SLEEP_DURATION = "sleep_duration"
    RESPIRATORY_RATE = "respiratory_rate"


@strawberry.enum
class AlertLevel(Enum):
    """Alert levels for readings"""
    NORMAL = "normal"
    LOW = "low"
    HIGH = "high"
    CRITICAL = "critical"


@strawberry.enum
class ReadingCategory(Enum):
    """Categories for grouping readings"""
    CARDIOVASCULAR = "cardiovascular"
    METABOLIC = "metabolic"
    VITAL_SIGNS = "vital_signs"
    RESPIRATORY = "respiratory"
    ANTHROPOMETRIC = "anthropometric"
    ACTIVITY = "activity"
    OTHER = "other"


@strawberry.type
class DeviceMetadata:
    """Device metadata information"""
    battery_level: Optional[int] = None
    signal_quality: Optional[str] = None
    firmware_version: Optional[str] = None
    manufacturer: Optional[str] = None
    model: Optional[str] = None


@strawberry.type
class VendorInfo:
    """Vendor information"""
    vendor_id: str
    vendor_name: str


@strawberry.type
class DeviceReading:
    """A device reading with processed information"""
    id: str
    device_id: str
    patient_id: Optional[str]
    reading_timestamp: int
    reading_datetime: str
    reading_date: str
    reading_time: str
    reading_type: ReadingType
    reading_type_display: str
    reading_value: float
    reading_unit: str
    reading_category: ReadingCategory
    alert_level: AlertLevel
    is_critical: bool
    is_abnormal: bool
    vendor_info: Optional[VendorInfo]
    device_metadata: Optional[DeviceMetadata]
    indexed_at: str
    year: int
    month: int
    day: int
    hour: int


@strawberry.type
class ReadingConnection:
    """Paginated connection for device readings"""
    items: List[DeviceReading]
    total: int
    page: int
    limit: int
    has_next_page: bool
    has_previous_page: bool


@strawberry.type
class ReadingTypeCount:
    """Count of readings by type"""
    reading_type: ReadingType
    count: int


@strawberry.type
class AlertLevelCount:
    """Count of readings by alert level"""
    alert_level: AlertLevel
    count: int


@strawberry.type
class DateRange:
    """Date range for readings"""
    start_date: str
    end_date: str


@strawberry.type
class ReadingStats:
    """Statistics for device readings"""
    total_readings: int
    readings_by_type: List[ReadingTypeCount]
    readings_by_alert_level: List[AlertLevelCount]
    latest_reading: Optional[DeviceReading]
    date_range: Optional[DateRange]


@strawberry.federation.type(keys=["id"])
class Patient:
    """Patient entity extended with device data"""
    id: strawberry.ID = strawberry.federation.field(external=True)
    
    @strawberry.field
    async def device_readings(
        self,
        reading_type: Optional[ReadingType] = None,
        alert_level: Optional[AlertLevel] = None,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        page: int = 1,
        limit: int = 50
    ) -> ReadingConnection:
        """Get device readings for this patient"""
        from ..services.device_data_service import get_device_data_service
        
        service = get_device_data_service()
        return await service.get_patient_readings(
            patient_id=str(self.id),
            reading_type=reading_type.value if reading_type else None,
            alert_level=alert_level.value if alert_level else None,
            start_date=start_date,
            end_date=end_date,
            page=page,
            limit=limit
        )
    
    @strawberry.field
    async def device_reading_stats(
        self,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None
    ) -> ReadingStats:
        """Get device reading statistics for this patient"""
        from ..services.device_data_service import get_device_data_service
        
        service = get_device_data_service()
        return await service.get_patient_reading_stats(
            patient_id=str(self.id),
            start_date=start_date,
            end_date=end_date
        )
    
    @strawberry.field
    async def latest_readings(
        self,
        limit: int = 10
    ) -> List[DeviceReading]:
        """Get latest device readings for this patient"""
        from ..services.device_data_service import get_device_data_service
        
        service = get_device_data_service()
        result = await service.get_patient_readings(
            patient_id=str(self.id),
            page=1,
            limit=limit
        )
        return result.items
    
    @strawberry.field
    async def critical_readings(
        self,
        hours: int = 24,
        limit: int = 20
    ) -> List[DeviceReading]:
        """Get critical readings for this patient in the last N hours"""
        from ..services.device_data_service import get_device_data_service
        from datetime import datetime, timedelta
        
        end_date = datetime.utcnow()
        start_date = end_date - timedelta(hours=hours)
        
        service = get_device_data_service()
        result = await service.get_patient_readings(
            patient_id=str(self.id),
            alert_level="critical",
            start_date=start_date.isoformat(),
            end_date=end_date.isoformat(),
            page=1,
            limit=limit
        )
        return result.items


@strawberry.federation.type(keys=["id"])
class Device:
    """Device entity with readings"""
    id: strawberry.ID
    
    @strawberry.field
    async def readings(
        self,
        reading_type: Optional[ReadingType] = None,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        page: int = 1,
        limit: int = 50
    ) -> ReadingConnection:
        """Get readings for this device"""
        from ..services.device_data_service import get_device_data_service
        
        service = get_device_data_service()
        return await service.get_device_readings(
            device_id=str(self.id),
            reading_type=reading_type.value if reading_type else None,
            start_date=start_date,
            end_date=end_date,
            page=page,
            limit=limit
        )
    
    @strawberry.field
    async def latest_reading(self) -> Optional[DeviceReading]:
        """Get the latest reading from this device"""
        from ..services.device_data_service import get_device_data_service
        
        service = get_device_data_service()
        result = await service.get_device_readings(
            device_id=str(self.id),
            page=1,
            limit=1
        )
        return result.items[0] if result.items else None
