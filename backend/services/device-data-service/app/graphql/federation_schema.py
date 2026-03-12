"""
Apollo Federation schema for Device Data Service
"""
import strawberry
from .queries import Query
from .types import Patient, Device, DeviceReading, ReadingConnection, ReadingStats


# Create the federated schema
schema = strawberry.federation.Schema(
    query=Query,
    types=[
        Patient,  # Federation extension
        Device,   # Federation type
        DeviceReading,
        ReadingConnection,
        ReadingStats,
    ],
    enable_federation_2=True
)
