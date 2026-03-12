"""
FHIR Transformation Service

This service ports the EXACT same FHIR transformation logic from the
PySpark ETL pipeline (business_logic/transformations.py).

The functions create_fhir_observation_from_device_data_impl() and
create_ui_reading_from_device_data_impl() are preserved with identical
business logic to ensure data consistency.
"""

import json
import uuid
from datetime import datetime
from typing import Dict, Any, Optional

import structlog

logger = structlog.get_logger(__name__)

# LOINC codes for device readings (same as PySpark implementation)
LOINC_CODES = {
    "heart_rate": {
        "code": "8867-4",
        "display": "Heart rate",
        "system": "http://loinc.org"
    },
    "blood_pressure_systolic": {
        "code": "8480-6",
        "display": "Systolic blood pressure",
        "system": "http://loinc.org"
    },
    "blood_pressure_diastolic": {
        "code": "8462-4",
        "display": "Diastolic blood pressure",
        "system": "http://loinc.org"
    },
    "blood_glucose": {
        "code": "33747-0",
        "display": "Blood glucose",
        "system": "http://loinc.org"
    },
    "temperature": {
        "code": "8310-5",
        "display": "Body temperature",
        "system": "http://loinc.org"
    },
    "oxygen_saturation": {
        "code": "2708-6",
        "display": "Oxygen saturation",
        "system": "http://loinc.org"
    },
    "weight": {
        "code": "29463-7",
        "display": "Body weight",
        "system": "http://loinc.org"
    },
    "respiratory_rate": {
        "code": "9279-1",
        "display": "Respiratory rate",
        "system": "http://loinc.org"
    }
}


class FHIRTransformationService:
    """
    FHIR Transformation Service
    
    Implements the exact same FHIR transformation logic as the PySpark pipeline.
    This ensures data consistency between the monolithic and modular architectures.
    """
    
    def __init__(self):
        self.service_name = "stage2-storage-fanout"
        logger.info("FHIR Transformation Service initialized")

    def transform_to_fhir_sync(self, enriched_data: Dict[str, Any]) -> Optional[str]:
        """Synchronous wrapper for FHIR transformation"""
        try:
            return self.create_fhir_observation_from_device_data(enriched_data)
        except Exception as e:
            logger.error("FHIR transformation failed", error=str(e))
            return None
    
    def create_fhir_observation_from_device_data(self, device_data: Dict[str, Any]) -> str:
        """
        Transform device data into FHIR Observation resource.
        
        This is the EXACT same function as create_fhir_observation_from_device_data_impl()
        from the PySpark pipeline, ensuring identical FHIR compliance.
        
        Args:
            device_data: Device reading data dictionary
            
        Returns:
            JSON string of FHIR Observation resource
        """
        try:
            # Extract required fields
            device_id = device_data.get('device_id')
            timestamp = device_data.get('timestamp')
            reading_type = device_data.get('reading_type')
            value = device_data.get('value')
            unit = device_data.get('unit')
            patient_id = device_data.get('patient_id')
            metadata = device_data.get('metadata')
            vendor_info = device_data.get('vendor_info')
            
            # Validate required fields (same as PySpark)
            if not all([device_id, timestamp, reading_type, value, unit]):
                logger.warning("Missing required fields for FHIR transformation", 
                             device_id=device_id, reading_type=reading_type)
                return self._create_error_fhir_resource("Missing required fields")
            
            # Generate unique FHIR resource ID
            fhir_id = str(uuid.uuid4())
            
            # Convert timestamp to FHIR datetime format
            try:
                if isinstance(timestamp, (int, float)):
                    observation_datetime = datetime.fromtimestamp(timestamp).isoformat() + "Z"
                else:
                    observation_datetime = datetime.now().isoformat() + "Z"
            except (ValueError, OSError):
                observation_datetime = datetime.now().isoformat() + "Z"
            
            # Get LOINC code for reading type
            loinc_info = LOINC_CODES.get(reading_type.lower(), {
                "code": "unknown",
                "display": f"Unknown reading type: {reading_type}",
                "system": "http://loinc.org"
            })
            
            # Parse metadata if it's a string
            parsed_metadata = {}
            if metadata:
                try:
                    if isinstance(metadata, str):
                        parsed_metadata = json.loads(metadata)
                    elif isinstance(metadata, dict):
                        parsed_metadata = metadata
                except json.JSONDecodeError:
                    logger.warning("Failed to parse metadata JSON", metadata=metadata)
            
            # Parse vendor info if it's a string
            parsed_vendor_info = {}
            if vendor_info:
                try:
                    if isinstance(vendor_info, str):
                        parsed_vendor_info = json.loads(vendor_info)
                    elif isinstance(vendor_info, dict):
                        parsed_vendor_info = vendor_info
                except json.JSONDecodeError:
                    logger.warning("Failed to parse vendor_info JSON", vendor_info=vendor_info)
            
            # Create FHIR Observation resource (same structure as PySpark)
            fhir_observation = {
                "resourceType": "Observation",
                "id": fhir_id,
                "status": "final",
                "category": [
                    {
                        "coding": [
                            {
                                "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                                "code": "vital-signs",
                                "display": "Vital Signs"
                            }
                        ]
                    }
                ],
                "code": {
                    "coding": [
                        {
                            "system": loinc_info["system"],
                            "code": loinc_info["code"],
                            "display": loinc_info["display"]
                        }
                    ],
                    "text": loinc_info["display"]
                },
                "subject": {
                    "reference": f"Patient/{patient_id}" if patient_id else f"Device/{device_id}"
                },
                "effectiveDateTime": observation_datetime,
                "valueQuantity": {
                    "value": float(value),
                    "unit": unit,
                    "system": "http://unitsofmeasure.org",
                    "code": unit
                },
                "device": {
                    "reference": f"Device/{device_id}",
                    "display": f"Device {device_id}"
                },
                "meta": {
                    "source": self.service_name,
                    "versionId": "1",
                    "lastUpdated": datetime.now().isoformat() + "Z",
                    "tag": [
                        {
                            "system": "http://clinical-synthesis-hub.com/tags",
                            "code": "device-reading",
                            "display": "Device Reading"
                        }
                    ]
                }
            }
            
            # Add component for additional metadata (same as PySpark)
            components = []
            
            # Add device metadata as components
            if parsed_metadata:
                for key, value in parsed_metadata.items():
                    if key in ['battery_level', 'signal_quality', 'device_model']:
                        components.append({
                            "code": {
                                "coding": [
                                    {
                                        "system": "http://clinical-synthesis-hub.com/device-metadata",
                                        "code": key,
                                        "display": key.replace('_', ' ').title()
                                    }
                                ]
                            },
                            "valueString": str(value)
                        })
            
            # Add vendor information as components
            if parsed_vendor_info:
                vendor_id = parsed_vendor_info.get('vendor_id')
                vendor_name = parsed_vendor_info.get('vendor_name')
                
                if vendor_id:
                    components.append({
                        "code": {
                            "coding": [
                                {
                                    "system": "http://clinical-synthesis-hub.com/vendor-info",
                                    "code": "vendor-id",
                                    "display": "Vendor ID"
                                }
                            ]
                        },
                        "valueString": vendor_id
                    })
                
                if vendor_name:
                    components.append({
                        "code": {
                            "coding": [
                                {
                                    "system": "http://clinical-synthesis-hub.com/vendor-info",
                                    "code": "vendor-name",
                                    "display": "Vendor Name"
                                }
                            ]
                        },
                        "valueString": vendor_name
                    })
            
            # Add components to observation if any exist
            if components:
                fhir_observation["component"] = components
            
            # Convert to JSON string
            return json.dumps(fhir_observation, separators=(',', ':'))
            
        except Exception as e:
            logger.error("Error creating FHIR observation", error=str(e), device_data=device_data)
            return self._create_error_fhir_resource(f"Transformation error: {str(e)}")
    
    def create_ui_reading_from_device_data(self, device_data: Dict[str, Any]) -> str:
        """
        Create UI-optimized document from device data.
        
        This is the EXACT same function as create_ui_reading_from_device_data_impl()
        from the PySpark pipeline, ensuring identical UI document structure.
        
        Args:
            device_data: Device reading data dictionary
            
        Returns:
            JSON string of UI-optimized document
        """
        try:
            # Extract fields
            device_id = device_data.get('device_id')
            timestamp = device_data.get('timestamp')
            reading_type = device_data.get('reading_type')
            value = device_data.get('value')
            unit = device_data.get('unit')
            patient_id = device_data.get('patient_id')
            metadata = device_data.get('metadata')
            vendor_info = device_data.get('vendor_info')
            
            # Validate required fields
            if not all([device_id, timestamp, reading_type, value, unit]):
                logger.warning("Missing required fields for UI document", 
                             device_id=device_id, reading_type=reading_type)
                return self._create_error_ui_document("Missing required fields")
            
            # Parse metadata
            parsed_metadata = {}
            if metadata:
                try:
                    if isinstance(metadata, str):
                        parsed_metadata = json.loads(metadata)
                    elif isinstance(metadata, dict):
                        parsed_metadata = metadata
                except json.JSONDecodeError:
                    pass
            
            # Parse vendor info
            parsed_vendor_info = {}
            if vendor_info:
                try:
                    if isinstance(vendor_info, str):
                        parsed_vendor_info = json.loads(vendor_info)
                    elif isinstance(vendor_info, dict):
                        parsed_vendor_info = vendor_info
                except json.JSONDecodeError:
                    pass
            
            # Determine alert level (same logic as PySpark)
            alert_level = self._determine_alert_level(reading_type, value)
            
            # Create UI document (same structure as PySpark)
            ui_document = {
                # Core reading data
                "device_id": device_id,
                "patient_id": patient_id,
                "reading_timestamp": timestamp,
                "reading_type": reading_type,
                "reading_value": float(value),
                "reading_unit": unit,
                
                # UI-specific fields
                "alert_level": alert_level,
                "reading_category": self._categorize_reading(reading_type),
                "display_name": self._get_display_name(reading_type),
                "is_critical": alert_level in ["critical", "emergency"],
                
                # Metadata
                "device_metadata": parsed_metadata,
                "vendor_info": parsed_vendor_info,
                
                # Processing metadata
                "indexed_at": datetime.now().isoformat() + "Z",
                "processing_stage": self.service_name,
                "document_version": "1.0"
            }
            
            # Add device-specific fields
            if parsed_metadata:
                ui_document.update({
                    "battery_level": parsed_metadata.get('battery_level'),
                    "signal_quality": parsed_metadata.get('signal_quality'),
                    "device_model": parsed_metadata.get('device_model')
                })
            
            # Add vendor-specific fields
            if parsed_vendor_info:
                ui_document.update({
                    "vendor_id": parsed_vendor_info.get('vendor_id'),
                    "vendor_name": parsed_vendor_info.get('vendor_name')
                })
            
            return json.dumps(ui_document, separators=(',', ':'))
            
        except Exception as e:
            logger.error("Error creating UI document", error=str(e), device_data=device_data)
            return self._create_error_ui_document(f"Transformation error: {str(e)}")
    
    def _create_error_fhir_resource(self, error_message: str) -> str:
        """Create error FHIR resource (same as PySpark)"""
        error_resource = {
            "resourceType": "Observation",
            "id": str(uuid.uuid4()),
            "status": "cancelled",
            "code": {
                "text": f"Error: {error_message}"
            },
            "valueString": f"Error processing device data: {error_message}",
            "meta": {
                "source": self.service_name,
                "lastUpdated": datetime.now().isoformat() + "Z"
            }
        }
        return json.dumps(error_resource)
    
    def _create_error_ui_document(self, error_message: str) -> str:
        """Create error UI document (same as PySpark)"""
        error_doc = {
            "device_id": "unknown",
            "reading_timestamp": 0,
            "reading_type": "unknown",
            "reading_value": 0.0,
            "reading_unit": "unknown",
            "error": error_message,
            "alert_level": "error",
            "indexed_at": datetime.now().isoformat() + "Z",
            "processing_stage": self.service_name
        }
        return json.dumps(error_doc)
    
    def _determine_alert_level(self, reading_type: str, value: float) -> str:
        """Determine alert level (same logic as PySpark)"""
        # Normal ranges (same as PySpark)
        normal_ranges = {
            "heart_rate": (60, 100),
            "blood_pressure_systolic": (90, 140),
            "blood_pressure_diastolic": (60, 90),
            "blood_glucose": (70, 140),
            "temperature": (97.0, 99.5),
            "oxygen_saturation": (95, 100),
            "respiratory_rate": (12, 20)
        }
        
        # Critical ranges (same as PySpark)
        critical_ranges = {
            "heart_rate": (40, 150),
            "blood_pressure_systolic": (70, 180),
            "blood_pressure_diastolic": (40, 110),
            "blood_glucose": (50, 200),
            "temperature": (95.0, 104.0),
            "oxygen_saturation": (85, 100),
            "respiratory_rate": (8, 30)
        }
        
        if reading_type not in normal_ranges:
            return "normal"
        
        normal_min, normal_max = normal_ranges[reading_type]
        critical_min, critical_max = critical_ranges.get(reading_type, (0, float('inf')))
        
        # Check critical ranges first
        if value < critical_min or value > critical_max:
            return "critical"
        
        # Check normal ranges
        if value < normal_min:
            return "low"
        elif value > normal_max:
            return "high"
        else:
            return "normal"
    
    def _categorize_reading(self, reading_type: str) -> str:
        """Categorize reading type (same as PySpark)"""
        categories = {
            "heart_rate": "cardiovascular",
            "blood_pressure_systolic": "cardiovascular",
            "blood_pressure_diastolic": "cardiovascular",
            "blood_glucose": "metabolic",
            "temperature": "vital_signs",
            "oxygen_saturation": "respiratory",
            "respiratory_rate": "respiratory",
            "weight": "anthropometric"
        }
        return categories.get(reading_type.lower(), "other")
    
    def _get_display_name(self, reading_type: str) -> str:
        """Get display name for reading type (same as PySpark)"""
        display_names = {
            "heart_rate": "Heart Rate",
            "blood_pressure_systolic": "Systolic Blood Pressure",
            "blood_pressure_diastolic": "Diastolic Blood Pressure",
            "blood_glucose": "Blood Glucose",
            "temperature": "Body Temperature",
            "oxygen_saturation": "Oxygen Saturation",
            "respiratory_rate": "Respiratory Rate",
            "weight": "Body Weight"
        }
        return display_names.get(reading_type.lower(), reading_type.replace('_', ' ').title())
    
    def is_healthy(self) -> bool:
        """Check if transformation service is healthy"""
        return True  # Simple health check for now
