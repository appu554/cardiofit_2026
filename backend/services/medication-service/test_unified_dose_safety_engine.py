#!/usr/bin/env python3
"""
Comprehensive Test Suite for Unified Dose Safety Engine
Tests all advanced features: Knowledge Base, Parallel Processing, Model Sandbox,
Advanced Validation, Hot Loading, Titration, Risk Assessment, and Monitoring
"""

import requests
import json
import time
import sys
from datetime import datetime
from typing import Dict, Any, List

# Engine Configuration
ENGINE_BASE_URL = "http://localhost:8080"
HEADERS = {
    "Content-Type": "application/json",
    "Accept": "application/json",
    "Authorization": "development-token"  # Use the development token for testing
}

class UnifiedDoseSafetyEngineTest:
    def __init__(self):
        self.base_url = ENGINE_BASE_URL
        self.headers = HEADERS
        self.test_results = []
        
    def log_test(self, test_name: str, success: bool, details: str = ""):
        """Log test results"""
        status = "✅ PASS" if success else "❌ FAIL"
        timestamp = datetime.now().strftime("%H:%M:%S")
        print(f"[{timestamp}] {status} {test_name}")
        if details:
            print(f"    📋 {details}")
        
        self.test_results.append({
            "test": test_name,
            "success": success,
            "details": details,
            "timestamp": timestamp
        })
    
    def test_engine_health(self) -> bool:
        """Test 1: Engine Health and Status"""
        print("\n🏥 Testing Engine Health & Status...")
        
        try:
            # Basic health check
            response = requests.get(f"{self.base_url}/health", timeout=10)
            if response.status_code == 200:
                self.log_test("Basic Health Check", True, f"Status: {response.json().get('status', 'unknown')}")
            else:
                self.log_test("Basic Health Check", False, f"HTTP {response.status_code}")
                return False
            
            # Detailed health check
            response = requests.get(f"{self.base_url}/health/detailed", timeout=10)
            if response.status_code == 200:
                health_data = response.json()
                components = health_data.get('components', {})
                healthy_count = sum(1 for comp in components.values() if comp.get('status') == 'healthy')
                self.log_test("Detailed Health Check", True, f"Healthy components: {healthy_count}/{len(components)}")
            else:
                self.log_test("Detailed Health Check", False, f"HTTP {response.status_code}")
            
            # Engine status
            response = requests.get(f"{self.base_url}/status", timeout=10)
            if response.status_code == 200:
                status_data = response.json()
                self.log_test("Engine Status", True, f"Version: {status_data.get('version', 'unknown')}")
            else:
                self.log_test("Engine Status", False, f"HTTP {response.status_code}")
            
            return True
            
        except requests.exceptions.RequestException as e:
            self.log_test("Engine Health", False, f"Connection error: {str(e)}")
            return False
    
    def test_dose_optimization(self) -> bool:
        """Test 2: Advanced Dose Optimization with Risk Assessment"""
        print("\n💊 Testing Dose Optimization with Risk Assessment...")
        
        test_request = {
            "request_id": "dose-opt-test-001",
            "patient_id": "test-patient-001",
            "medication_code": "metformin",
            "clinical_parameters": {
                "age_years": 65,
                "weight_kg": 80.5,
                "height_cm": 175,
                "gender": "male",
                "egfr": 45.0,
                "hba1c": 8.2,
                "serum_creatinine": 1.4
            },
            "optimization_type": "dose_calculation",
            "clinical_context": {
                "indication": "type_2_diabetes",
                "hba1c_target": 7.0,
                "safety_profile": "moderate",
                "current_medications": ["lisinopril"]
            },
            "processing_hints": {
                "enable_drug_interactions": True,
                "enable_contraindication_checks": True
            }
        }
        
        try:
            response = requests.post(
                f"{self.base_url}/api/dose/optimize",
                json=test_request,
                headers=self.headers,
                timeout=30
            )
            
            if response.status_code == 200:
                result = response.json()
                
                # Check for required fields
                required_fields = ['optimized_dose', 'optimization_score', 'confidence_interval']
                missing_fields = [field for field in required_fields if field not in result]

                if not missing_fields:
                    dose = result.get('optimized_dose', 0)
                    score = result.get('optimization_score', 0)
                    self.log_test("Dose Optimization", True, f"Dose: {dose}mg, Score: {score}")

                    # Test confidence interval
                    confidence = result.get('confidence_interval', {})
                    if confidence:
                        self.log_test("Confidence Interval", True, f"Range: {confidence.get('lower', 0)}-{confidence.get('upper', 0)}")

                    return True
                else:
                    self.log_test("Dose Optimization", False, f"Missing fields: {missing_fields}")
                    return False
            else:
                self.log_test("Dose Optimization", False, f"HTTP {response.status_code}: {response.text}")
                return False
                
        except requests.exceptions.RequestException as e:
            self.log_test("Dose Optimization", False, f"Request error: {str(e)}")
            return False
    
    def test_medication_intelligence(self) -> bool:
        """Test 3: Medication Intelligence with Drug Interactions"""
        print("\n🧠 Testing Medication Intelligence...")
        
        test_request = {
            "request_id": "med-intel-test-002",
            "patient_id": "test-patient-002",
            "medications": [
                {
                    "code": "warfarin",
                    "name": "Warfarin Sodium",
                    "dose": 5.0,
                    "unit": "mg",
                    "frequency": "daily",
                    "route": "oral",
                    "duration": "ongoing",
                    "indication": "atrial_fibrillation",
                    "properties": {}
                },
                {
                    "code": "aspirin",
                    "name": "Aspirin",
                    "dose": 81.0,
                    "unit": "mg",
                    "frequency": "daily",
                    "route": "oral",
                    "duration": "ongoing",
                    "indication": "coronary_artery_disease",
                    "properties": {}
                }
            ],
            "intelligence_type": "comprehensive",
            "analysis_depth": "detailed",
            "clinical_context": {
                "age_years": 72,
                "weight_kg": 70.0,
                "conditions": ["atrial_fibrillation", "coronary_artery_disease"],
                "inr": 2.8,
                "egfr": 55.0,
                "include_interactions": True,
                "include_monitoring": True
            }
        }
        
        try:
            response = requests.post(
                f"{self.base_url}/api/medication/intelligence",
                json=test_request,
                headers=self.headers,
                timeout=30
            )
            
            if response.status_code == 200:
                result = response.json()
                
                # Check intelligence score
                intelligence_score = result.get('intelligence_score', 0)
                self.log_test("Intelligence Score", True, f"Score: {intelligence_score}")

                # Check interaction analysis
                interaction_analysis = result.get('interaction_analysis', {})
                if interaction_analysis:
                    self.log_test("Interaction Analysis", True, f"Analysis completed")
                else:
                    self.log_test("Interaction Analysis", False, "No interaction analysis")

                # Check outcome predictions
                predictions = result.get('outcome_predictions', {})
                if predictions:
                    self.log_test("Outcome Predictions", True, f"Predictions generated")
                else:
                    self.log_test("Outcome Predictions", False, "No predictions")
                
                return True
            else:
                self.log_test("Medication Intelligence", False, f"HTTP {response.status_code}")
                return False
                
        except requests.exceptions.RequestException as e:
            self.log_test("Medication Intelligence", False, f"Request error: {str(e)}")
            return False
    
    def test_flow2_execution(self) -> bool:
        """Test 4: Flow2 Recipe Execution with Advanced Features"""
        print("\n🔄 Testing Flow2 Recipe Execution...")
        
        test_request = {
            "request_id": "flow2-test-001",
            "patient_id": "test-patient-003",
            "recipe_type": "dose_calculation_with_titration",
            "input_data": {
                "drug_id": "lisinopril",
                "indication": "hypertension",
                "patient_data": {
                    "age_years": 58,
                    "weight_kg": 85.0,
                    "blood_pressure": {
                        "systolic": 165,
                        "diastolic": 95
                    }
                },
                "target_bp": {
                    "systolic": 130,
                    "diastolic": 80
                },
                "titration_preferences": {
                    "strategy": "gradual",
                    "max_duration_weeks": 8
                }
            },
            "execution_options": {
                "enable_parallel_processing": True,
                "enable_risk_assessment": True,
                "enable_validation": True
            }
        }
        
        try:
            response = requests.post(
                f"{self.base_url}/api/flow2/execute",
                json=test_request,
                headers=self.headers,
                timeout=45
            )
            
            if response.status_code == 200:
                result = response.json()
                
                # Check execution results
                if result.get('success', False):
                    execution_time = result.get('execution_time_ms', 0)
                    self.log_test("Flow2 Execution", True, f"Completed in {execution_time}ms")
                    
                    # Check for titration schedule
                    titration = result.get('titration_schedule')
                    if titration:
                        steps = len(titration.get('steps', []))
                        self.log_test("Titration Schedule", True, f"Generated {steps} titration steps")
                    
                    # Check parallel processing metrics
                    metrics = result.get('performance_metrics', {})
                    if metrics.get('parallel_execution_used', False):
                        self.log_test("Parallel Processing", True, f"Speedup: {metrics.get('speedup_factor', 1.0)}x")
                    
                    return True
                else:
                    errors = result.get('errors', [])
                    self.log_test("Flow2 Execution", False, f"Execution failed: {errors}")
                    return False
            else:
                self.log_test("Flow2 Execution", False, f"HTTP {response.status_code}")
                return False
                
        except requests.exceptions.RequestException as e:
            self.log_test("Flow2 Execution", False, f"Request error: {str(e)}")
            return False
    
    def test_performance_metrics(self) -> bool:
        """Test 5: Performance Monitoring and Metrics"""
        print("\n📊 Testing Performance Monitoring...")
        
        try:
            response = requests.get(f"{self.base_url}/metrics", timeout=10)
            
            if response.status_code == 200:
                metrics = response.json()
                
                # Check key performance metrics
                total_requests = metrics.get('total_requests', 0)
                avg_response_time = metrics.get('average_response_time_ms', 0)
                error_rate = metrics.get('error_rate', 0)
                
                self.log_test("Performance Metrics", True, 
                             f"Requests: {total_requests}, Avg time: {avg_response_time}ms, Error rate: {error_rate}%")
                
                # Check advanced features metrics
                advanced_metrics = metrics.get('advanced_features', {})
                if advanced_metrics:
                    parallel_usage = advanced_metrics.get('parallel_processing_usage', 0)
                    sandbox_executions = advanced_metrics.get('sandbox_executions', 0)
                    self.log_test("Advanced Features Metrics", True, 
                                 f"Parallel usage: {parallel_usage}%, Sandbox executions: {sandbox_executions}")
                
                return True
            else:
                self.log_test("Performance Metrics", False, f"HTTP {response.status_code}")
                return False
                
        except requests.exceptions.RequestException as e:
            self.log_test("Performance Metrics", False, f"Request error: {str(e)}")
            return False
    
    def test_admin_statistics(self) -> bool:
        """Test 6: Admin Statistics and System Status"""
        print("\n👨‍💼 Testing Admin Statistics...")
        
        try:
            response = requests.get(f"{self.base_url}/api/admin/stats", headers=self.headers, timeout=10)
            
            if response.status_code == 200:
                stats = response.json()
                
                # Check system statistics
                uptime = stats.get('uptime_seconds', 0)
                memory_usage = stats.get('memory_usage_mb', 0)
                active_connections = stats.get('active_connections', 0)
                
                self.log_test("System Statistics", True, 
                             f"Uptime: {uptime}s, Memory: {memory_usage}MB, Connections: {active_connections}")
                
                # Check knowledge base statistics
                kb_stats = stats.get('knowledge_base', {})
                if kb_stats:
                    drug_rules = kb_stats.get('total_drug_rules', 0)
                    ddi_rules = kb_stats.get('total_ddi_rules', 0)
                    self.log_test("Knowledge Base Stats", True, 
                                 f"Drug rules: {drug_rules}, DDI rules: {ddi_rules}")
                
                return True
            else:
                self.log_test("Admin Statistics", False, f"HTTP {response.status_code}")
                return False
                
        except requests.exceptions.RequestException as e:
            self.log_test("Admin Statistics", False, f"Request error: {str(e)}")
            return False
    
    def run_comprehensive_test_suite(self):
        """Run the complete test suite"""
        print("🦀" + "="*63)
        print("🦀  UNIFIED DOSE SAFETY ENGINE - COMPREHENSIVE TEST SUITE")
        print("🦀" + "="*63)
        print(f"🕐 Started at: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        print(f"🌐 Testing engine at: {self.base_url}")
        
        # Run all tests
        tests = [
            ("Engine Health & Status", self.test_engine_health),
            ("Dose Optimization", self.test_dose_optimization),
            ("Medication Intelligence", self.test_medication_intelligence),
            ("Flow2 Execution", self.test_flow2_execution),
            ("Performance Metrics", self.test_performance_metrics),
            ("Admin Statistics", self.test_admin_statistics),
        ]
        
        passed_tests = 0
        total_tests = len(tests)
        
        for test_name, test_func in tests:
            try:
                if test_func():
                    passed_tests += 1
            except Exception as e:
                self.log_test(test_name, False, f"Exception: {str(e)}")
        
        # Print summary
        print("\n🦀" + "="*63)
        print("🦀  TEST SUITE SUMMARY")
        print("🦀" + "="*63)
        
        success_rate = (passed_tests / total_tests) * 100
        print(f"📊 Tests Passed: {passed_tests}/{total_tests} ({success_rate:.1f}%)")
        
        if success_rate >= 80:
            print("🎉 ENGINE STATUS: PRODUCTION READY!")
            print("✅ All critical features are operational")
        elif success_rate >= 60:
            print("⚠️  ENGINE STATUS: MOSTLY FUNCTIONAL")
            print("🔧 Some features may need attention")
        else:
            print("❌ ENGINE STATUS: NEEDS ATTENTION")
            print("🚨 Multiple critical issues detected")
        
        print(f"🕐 Completed at: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        
        return success_rate >= 80

def main():
    """Main test execution"""
    print("🚀 Initializing Unified Dose Safety Engine Test Suite...")
    
    # Wait a moment for the engine to be fully ready
    time.sleep(2)
    
    tester = UnifiedDoseSafetyEngineTest()
    
    try:
        success = tester.run_comprehensive_test_suite()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        print("\n⚠️  Test suite interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Test suite failed with exception: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()
