#!/usr/bin/env python3
"""
Outbox Pattern Logic Test (No Database Required)

This test demonstrates the outbox pattern logic and vendor detection
without requiring a database connection. Perfect for understanding
how the system works.
"""
import asyncio
import json
import logging
import sys
from pathlib import Path
from typing import Dict, Any

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class OutboxLogicDemo:
    """Demonstrate outbox pattern logic without database"""
    
    def __init__(self):
        # Supported vendors (from our implementation)
        self.supported_vendors = {
            # Consumer Fitness Devices
            "fitbit": {
                "outbox_table": "fitbit_outbox",
                "dead_letter_table": "fitbit_dead_letter",
                "device_types": ["heart_rate", "steps", "sleep_duration", "weight"]
            },
            "garmin": {
                "outbox_table": "garmin_outbox", 
                "dead_letter_table": "garmin_dead_letter",
                "device_types": ["heart_rate", "steps", "sleep_duration", "weight", "oxygen_saturation"]
            },
            "apple_health": {
                "outbox_table": "apple_health_outbox",
                "dead_letter_table": "apple_health_dead_letter", 
                "device_types": ["heart_rate", "steps", "sleep_duration", "weight", "ecg"]
            },
            
            # Medical Grade Devices
            "withings": {
                "outbox_table": "withings_outbox",
                "dead_letter_table": "withings_dead_letter",
                "device_types": ["weight", "blood_pressure", "temperature", "heart_rate"]
            },
            "omron": {
                "outbox_table": "omron_outbox",
                "dead_letter_table": "omron_dead_letter",
                "device_types": ["blood_pressure", "heart_rate", "weight"]
            },
            
            # Clinical/Hospital Devices
            "medical_device": {
                "outbox_table": "medical_device_outbox",
                "dead_letter_table": "medical_device_dead_letter",
                "device_types": ["ecg", "blood_pressure", "blood_glucose", "temperature", "oxygen_saturation", "heart_rate"]
            },
            
            # Fallback for Unknown Devices
            "generic_device": {
                "outbox_table": "generic_device_outbox",
                "dead_letter_table": "generic_device_dead_letter",
                "device_types": ["heart_rate", "steps", "weight", "temperature", "blood_pressure", "blood_glucose", "ecg", "oxygen_saturation", "sleep_duration"]
            }
        }
        
        # Sample medical device data
        self.medical_device_samples = {
            "fitbit_heart_rate": {
                "device_id": "fitbit_charge5_001",
                "reading_type": "heart_rate",
                "value": 75.5,
                "unit": "bpm",
                "timestamp": 1703123456,
                "patient_id": "patient-123",
                "metadata": {"vendor": "fitbit", "medical_grade": False}
            },
            "omron_blood_pressure": {
                "device_id": "omron_bp7000_001",
                "reading_type": "blood_pressure",
                "systolic": 120,
                "diastolic": 80,
                "unit": "mmHg",
                "timestamp": 1703123456,
                "patient_id": "patient-123",
                "metadata": {"vendor": "omron", "medical_grade": True}
            },
            "medical_glucose": {
                "device_id": "hospital_glucose_meter_001",
                "reading_type": "blood_glucose",
                "value": 95.0,
                "unit": "mg/dL",
                "timestamp": 1703123456,
                "patient_id": "patient-123",
                "metadata": {"medical_grade": True, "hospital_device": True}
            },
            "apple_ecg": {
                "device_id": "apple_watch_series8_001",
                "reading_type": "ecg",
                "waveform": [0.1, 0.2, 0.8, 0.1, -0.1, 0.0],
                "heart_rate": 72,
                "unit": "mV",
                "timestamp": 1703123456,
                "patient_id": "patient-123",
                "metadata": {"vendor": "apple", "medical_grade": True}
            },
            "unknown_steps": {
                "device_id": "unknown_fitness_tracker_001",
                "reading_type": "steps",
                "value": 8500,
                "unit": "steps",
                "timestamp": 1703123456,
                "patient_id": "patient-123",
                "metadata": {}
            }
        }
    
    async def run_demo(self):
        """Run the complete outbox pattern demo"""
        logger.info("🚀 Outbox Pattern Logic Demo")
        logger.info("="*60)
        
        # Show supported vendors
        self.show_supported_vendors()
        
        # Demonstrate vendor detection
        await self.demo_vendor_detection()
        
        # Show outbox routing
        self.demo_outbox_routing()
        
        # Show fault isolation benefits
        self.demo_fault_isolation()
        
        logger.info("="*60)
        logger.info("🎉 Demo completed! Outbox pattern ready for all medical devices!")
    
    def show_supported_vendors(self):
        """Show all supported vendors and their capabilities"""
        logger.info("\n📋 SUPPORTED VENDORS AND CAPABILITIES:")
        logger.info("-" * 40)
        
        for vendor_id, config in self.supported_vendors.items():
            device_types = config["device_types"]
            outbox_table = config["outbox_table"]
            
            logger.info(f"🏥 {vendor_id.upper().replace('_', ' ')}")
            logger.info(f"   Outbox Table: {outbox_table}")
            logger.info(f"   Device Types: {', '.join(device_types)}")
            logger.info(f"   Medical Grade: {'Yes' if 'medical' in vendor_id or vendor_id in ['withings', 'omron'] else 'No'}")
            logger.info("")
    
    async def demo_vendor_detection(self):
        """Demonstrate vendor detection logic"""
        logger.info("\n🔍 VENDOR DETECTION DEMO:")
        logger.info("-" * 40)
        
        for sample_name, device_data in self.medical_device_samples.items():
            detection_result = self.detect_vendor(device_data)
            
            logger.info(f"📱 {sample_name.upper().replace('_', ' ')}")
            logger.info(f"   Device ID: {device_data['device_id']}")
            logger.info(f"   Reading Type: {device_data['reading_type']}")
            logger.info(f"   ➜ Detected Vendor: {detection_result['vendor_id']}")
            logger.info(f"   ➜ Confidence: {detection_result['confidence']:.2f}")
            logger.info(f"   ➜ Method: {detection_result['method']}")
            logger.info(f"   ➜ Outbox Table: {detection_result['outbox_table']}")
            logger.info(f"   ➜ Medical Grade: {detection_result['is_medical_grade']}")
            logger.info("")
    
    def demo_outbox_routing(self):
        """Show how messages are routed to different outbox tables"""
        logger.info("\n🔄 OUTBOX ROUTING DEMO:")
        logger.info("-" * 40)
        
        # Simulate message routing
        routing_examples = [
            ("Fitbit heart rate", "fitbit_outbox"),
            ("Omron blood pressure", "omron_outbox"),
            ("Apple ECG", "apple_health_outbox"),
            ("Hospital glucose meter", "medical_device_outbox"),
            ("Unknown device", "generic_device_outbox")
        ]
        
        for device_type, outbox_table in routing_examples:
            logger.info(f"📊 {device_type} ➜ {outbox_table}")
        
        logger.info("\n✅ BENEFITS:")
        logger.info("   • True fault isolation per vendor")
        logger.info("   • Independent processing speeds")
        logger.info("   • Vendor-specific optimizations")
        logger.info("   • No cross-contamination of failures")
    
    def demo_fault_isolation(self):
        """Demonstrate fault isolation benefits"""
        logger.info("\n🛡️ FAULT ISOLATION DEMO:")
        logger.info("-" * 40)
        
        logger.info("SCENARIO: Fitbit API is down, causing Kafka publish failures")
        logger.info("")
        
        # Simulate outbox states
        outbox_states = {
            "fitbit_outbox": {"pending": 1500, "failed": 50, "status": "🔴 DEGRADED"},
            "garmin_outbox": {"pending": 10, "failed": 0, "status": "🟢 HEALTHY"},
            "apple_health_outbox": {"pending": 25, "failed": 0, "status": "🟢 HEALTHY"},
            "medical_device_outbox": {"pending": 5, "failed": 0, "status": "🟢 HEALTHY"},
            "generic_device_outbox": {"pending": 8, "failed": 0, "status": "🟢 HEALTHY"}
        }
        
        for table, state in outbox_states.items():
            vendor = table.replace("_outbox", "").replace("_", " ").title()
            logger.info(f"{state['status']} {vendor}")
            logger.info(f"   Pending: {state['pending']}, Failed: {state['failed']}")
        
        logger.info("\n🎯 RESULT:")
        logger.info("   • Only Fitbit users affected")
        logger.info("   • Garmin, Apple, Medical devices continue normally")
        logger.info("   • No cascading failures")
        logger.info("   • Independent recovery when Fitbit API restored")
    
    def detect_vendor(self, device_data: Dict[str, Any]) -> Dict[str, Any]:
        """Simple vendor detection logic"""
        device_id = device_data.get("device_id", "").lower()
        metadata = device_data.get("metadata", {})
        reading_type = device_data.get("reading_type", "")
        
        # Method 1: Explicit vendor in metadata
        if "vendor" in metadata:
            vendor = metadata["vendor"].lower()
            if vendor in self.supported_vendors:
                return {
                    "vendor_id": vendor,
                    "confidence": 0.95,
                    "method": "explicit_metadata",
                    "outbox_table": self.supported_vendors[vendor]["outbox_table"],
                    "is_medical_grade": metadata.get("medical_grade", False)
                }
        
        # Method 2: Device ID pattern matching
        for vendor_id in self.supported_vendors.keys():
            if vendor_id in device_id or device_id.startswith(vendor_id):
                return {
                    "vendor_id": vendor_id,
                    "confidence": 0.85,
                    "method": "device_id_pattern",
                    "outbox_table": self.supported_vendors[vendor_id]["outbox_table"],
                    "is_medical_grade": vendor_id in ["medical_device", "withings", "omron"]
                }
        
        # Method 3: Medical grade device type routing
        if metadata.get("medical_grade", False) or metadata.get("hospital_device", False):
            if reading_type in self.supported_vendors["medical_device"]["device_types"]:
                return {
                    "vendor_id": "medical_device",
                    "confidence": 0.75,
                    "method": "medical_grade_routing",
                    "outbox_table": "medical_device_outbox",
                    "is_medical_grade": True
                }
        
        # Method 4: Device type specific routing
        if reading_type == "blood_pressure":
            return {
                "vendor_id": "omron",
                "confidence": 0.60,
                "method": "device_type_routing",
                "outbox_table": "omron_outbox",
                "is_medical_grade": True
            }
        
        if reading_type == "ecg":
            return {
                "vendor_id": "apple_health",
                "confidence": 0.60,
                "method": "device_type_routing",
                "outbox_table": "apple_health_outbox",
                "is_medical_grade": True
            }
        
        # Method 5: Fallback to generic
        return {
            "vendor_id": "generic_device",
            "confidence": 0.10,
            "method": "fallback",
            "outbox_table": "generic_device_outbox",
            "is_medical_grade": False
        }


async def main():
    """Run the demo"""
    demo = OutboxLogicDemo()
    await demo.run_demo()


if __name__ == "__main__":
    asyncio.run(main())
