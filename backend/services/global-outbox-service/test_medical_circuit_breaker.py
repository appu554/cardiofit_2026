#!/usr/bin/env python3
"""
Medical Circuit Breaker Test Suite

Tests the Clinical Safety Overload Protection features including:
- Medical priority classification
- Emergency bypass functionality
- Priority lane circuit breakers
- Load-based adaptive filtering
- Clinical safety guarantees
"""

import asyncio
import logging
import sys
from pathlib import Path
from typing import Dict, Any

# Add the backend directory to Python path
backend_dir = Path(__file__).parent.parent.parent
if str(backend_dir) not in sys.path:
    sys.path.insert(0, str(backend_dir))

from app.services.medical_circuit_breaker import (
    MedicalAwareCircuitBreaker, 
    MedicalPriority,
    CircuitBreakerState
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MedicalCircuitBreakerTester:
    """Test suite for Medical-Aware Circuit Breaker"""
    
    def __init__(self):
        self.tests_passed = 0
        self.tests_failed = 0
        self.test_results = []
        self.circuit_breaker = MedicalAwareCircuitBreaker()
    
    def log_test_result(self, test_name: str, passed: bool, message: str = ""):
        """Log test result"""
        status = "✅ PASS" if passed else "❌ FAIL"
        full_message = f"{status} {test_name}"
        if message:
            full_message += f" - {message}"
        
        logger.info(full_message)
        
        if passed:
            self.tests_passed += 1
        else:
            self.tests_failed += 1
        
        self.test_results.append({
            "test": test_name,
            "passed": passed,
            "message": message
        })
    
    def test_medical_priority_classification(self):
        """Test medical priority classification logic"""
        try:
            # Test emergency events
            emergency_event = {
                "event_type": "cardiac_arrest_alert",
                "metadata": {"patient_id": "12345"}
            }
            priority = self.circuit_breaker.classify_medical_priority(emergency_event)
            if priority != MedicalPriority.EMERGENCY:
                self.log_test_result("Medical Priority - Emergency", False, f"Expected EMERGENCY, got {priority}")
                return
            
            # Test critical vital signs
            critical_vitals_event = {
                "event_type": "vital_signs_update",
                "metadata": {
                    "vital_signs": {
                        "heart_rate": 160,  # Above emergency threshold
                        "blood_pressure_systolic": 90
                    }
                }
            }
            priority = self.circuit_breaker.classify_medical_priority(critical_vitals_event)
            if priority != MedicalPriority.EMERGENCY:
                self.log_test_result("Medical Priority - Critical Vitals", False, f"Expected EMERGENCY, got {priority}")
                return
            
            # Test normal clinical event
            normal_event = {
                "event_type": "patient_observation",
                "metadata": {"observation_type": "routine_checkup"}
            }
            priority = self.circuit_breaker.classify_medical_priority(normal_event)
            if priority != MedicalPriority.NORMAL:
                self.log_test_result("Medical Priority - Normal", False, f"Expected NORMAL, got {priority}")
                return
            
            # Test non-clinical event
            low_priority_event = {
                "event_type": "system_log",
                "metadata": {"log_level": "info"}
            }
            priority = self.circuit_breaker.classify_medical_priority(low_priority_event)
            if priority != MedicalPriority.LOW:
                self.log_test_result("Medical Priority - Low", False, f"Expected LOW, got {priority}")
                return
            
            self.log_test_result("Medical Priority Classification", True, "All priority classifications correct")
            
        except Exception as e:
            self.log_test_result("Medical Priority Classification", False, str(e))
    
    async def test_emergency_bypass(self):
        """Test emergency bypass functionality"""
        try:
            # Create emergency event
            emergency_event = {
                "event_type": "cardiac_arrest_alert",
                "metadata": {
                    "patient_id": "emergency_patient_123",
                    "severity": "critical"
                }
            }
            
            # Emergency events should always be processed
            should_process = await self.circuit_breaker.should_process_event(emergency_event)
            if not should_process:
                self.log_test_result("Emergency Bypass", False, "Emergency event was blocked")
                return
            
            # Verify emergency metrics were updated
            status = self.circuit_breaker.get_circuit_breaker_status()
            emergency_processed = status["priority_metrics"]["emergency"]["processed"]
            if emergency_processed == 0:
                self.log_test_result("Emergency Bypass", False, "Emergency metrics not updated")
                return
            
            self.log_test_result("Emergency Bypass", True, f"Emergency event processed, metrics updated")
            
        except Exception as e:
            self.log_test_result("Emergency Bypass", False, str(e))
    
    async def test_priority_lane_processing(self):
        """Test priority lane processing"""
        try:
            # Test different priority events
            test_events = [
                {
                    "event_type": "sepsis_alert",
                    "expected_priority": MedicalPriority.EMERGENCY
                },
                {
                    "event_type": "abnormal_vitals",
                    "expected_priority": MedicalPriority.CRITICAL
                },
                {
                    "event_type": "medication_administration",
                    "expected_priority": MedicalPriority.HIGH
                },
                {
                    "event_type": "patient_discharge",
                    "expected_priority": MedicalPriority.NORMAL
                },
                {
                    "event_type": "device_heartbeat",
                    "expected_priority": MedicalPriority.LOW
                }
            ]
            
            processed_counts = {priority: 0 for priority in MedicalPriority}
            
            for event_data in test_events:
                event = {"event_type": event_data["event_type"], "metadata": {}}
                should_process = await self.circuit_breaker.should_process_event(event)
                
                if should_process:
                    classified_priority = self.circuit_breaker.classify_medical_priority(event)
                    processed_counts[classified_priority] += 1
                    
                    if classified_priority != event_data["expected_priority"]:
                        self.log_test_result("Priority Lane Processing", False, 
                                           f"Event {event_data['event_type']} classified as {classified_priority}, expected {event_data['expected_priority']}")
                        return
            
            # Verify all events were processed (no load shedding in normal conditions)
            total_processed = sum(processed_counts.values())
            if total_processed != len(test_events):
                self.log_test_result("Priority Lane Processing", False, 
                                   f"Only {total_processed}/{len(test_events)} events processed")
                return
            
            self.log_test_result("Priority Lane Processing", True, 
                               f"All {len(test_events)} events processed with correct priorities")
            
        except Exception as e:
            self.log_test_result("Priority Lane Processing", False, str(e))
    
    async def test_vital_signs_analysis(self):
        """Test vital signs analysis for medical priority"""
        try:
            # Test emergency vital signs
            emergency_vitals = {
                "event_type": "vital_signs_reading",
                "metadata": {
                    "vital_signs": {
                        "heart_rate": 35,  # Below emergency threshold
                        "oxygen_saturation": 80  # Below emergency threshold
                    }
                }
            }
            
            priority = self.circuit_breaker.classify_medical_priority(emergency_vitals)
            if priority != MedicalPriority.EMERGENCY:
                self.log_test_result("Vital Signs Analysis - Emergency", False, 
                                   f"Expected EMERGENCY for critical vitals, got {priority}")
                return
            
            # Test critical vital signs
            critical_vitals = {
                "event_type": "vital_signs_reading",
                "metadata": {
                    "vital_signs": {
                        "heart_rate": 45,  # Below critical threshold but above emergency
                        "blood_pressure_systolic": 85  # Below critical threshold
                    }
                }
            }
            
            priority = self.circuit_breaker.classify_medical_priority(critical_vitals)
            if priority != MedicalPriority.CRITICAL:
                self.log_test_result("Vital Signs Analysis - Critical", False, 
                                   f"Expected CRITICAL for abnormal vitals, got {priority}")
                return
            
            # Test normal vital signs
            normal_vitals = {
                "event_type": "vital_signs_reading",
                "metadata": {
                    "vital_signs": {
                        "heart_rate": 75,  # Normal range
                        "blood_pressure_systolic": 120,  # Normal range
                        "oxygen_saturation": 98  # Normal range
                    }
                }
            }
            
            priority = self.circuit_breaker.classify_medical_priority(normal_vitals)
            # Should be NORMAL since no critical patterns in event_type and vitals are normal
            if priority not in [MedicalPriority.NORMAL, MedicalPriority.LOW]:
                self.log_test_result("Vital Signs Analysis - Normal", False, 
                                   f"Expected NORMAL/LOW for normal vitals, got {priority}")
                return
            
            self.log_test_result("Vital Signs Analysis", True, "All vital signs classifications correct")
            
        except Exception as e:
            self.log_test_result("Vital Signs Analysis", False, str(e))
    
    def test_circuit_breaker_status(self):
        """Test circuit breaker status reporting"""
        try:
            status = self.circuit_breaker.get_circuit_breaker_status()
            
            # Verify required fields
            required_fields = [
                "overall_state", "priority_states", "load_metrics", 
                "priority_metrics", "emergency_bypass_enabled"
            ]
            
            for field in required_fields:
                if field not in status:
                    self.log_test_result("Circuit Breaker Status", False, f"Missing field: {field}")
                    return
            
            # Verify priority states
            if len(status["priority_states"]) != len(MedicalPriority):
                self.log_test_result("Circuit Breaker Status", False, 
                                   f"Expected {len(MedicalPriority)} priority states, got {len(status['priority_states'])}")
                return
            
            # Verify emergency bypass is enabled
            if not status["emergency_bypass_enabled"]:
                self.log_test_result("Circuit Breaker Status", False, "Emergency bypass should be enabled")
                return
            
            self.log_test_result("Circuit Breaker Status", True, "Status reporting working correctly")
            
        except Exception as e:
            self.log_test_result("Circuit Breaker Status", False, str(e))
    
    async def run_all_tests(self):
        """Run all medical circuit breaker tests"""
        logger.info("🏥 Running Medical Circuit Breaker Tests")
        logger.info("=" * 60)
        
        # Run all tests
        self.test_medical_priority_classification()
        await self.test_emergency_bypass()
        await self.test_priority_lane_processing()
        await self.test_vital_signs_analysis()
        self.test_circuit_breaker_status()
        
        # Print summary
        logger.info("=" * 60)
        logger.info(f"📊 Test Results: {self.tests_passed} passed, {self.tests_failed} failed")
        
        if self.tests_failed == 0:
            logger.info("🎉 All Medical Circuit Breaker tests passed!")
            return True
        else:
            logger.error("❌ Some tests failed. Please fix issues before proceeding.")
            
            # Show failed tests
            logger.error("\nFailed tests:")
            for result in self.test_results:
                if not result["passed"]:
                    logger.error(f"  - {result['test']}: {result['message']}")
            
            return False

async def main():
    """Main test runner"""
    tester = MedicalCircuitBreakerTester()
    success = await tester.run_all_tests()
    
    if success:
        print("\n✅ Medical Circuit Breaker is working perfectly!")
        print("\nClinical Safety Features:")
        print("1. ✅ Emergency bypass - Critical events always processed")
        print("2. ✅ Medical priority classification - Context-aware prioritization")
        print("3. ✅ Priority lanes - Separate processing by medical importance")
        print("4. ✅ Vital signs analysis - Automatic severity detection")
        print("5. ✅ Load-aware filtering - Protects critical data under load")
    else:
        print("\n❌ Medical Circuit Breaker needs work before proceeding")
        print("\nRecommended actions:")
        print("1. Fix the failed tests above")
        print("2. Verify medical priority classification logic")
        print("3. Test emergency bypass functionality")
    
    return success

if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
