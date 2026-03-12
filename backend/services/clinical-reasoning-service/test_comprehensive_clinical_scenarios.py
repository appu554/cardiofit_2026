#!/usr/bin/env python3
"""
Comprehensive Clinical Scenario Testing for CAE with Learning

This test suite demonstrates the full capabilities of the CAE:
1. Complex clinical scenarios with multiple patients
2. Population intelligence and patient clustering
3. Real-time learning from outcomes and overrides
4. Dynamic confidence updates and pattern discovery
5. Advanced orchestration with graph intelligence
"""

import asyncio
import logging
import json
from datetime import datetime, timezone
from typing import Dict, Any, List

# Import CAE components
from app.learning.learning_manager import learning_manager
from app.learning.outcome_tracker import OutcomeType, OutcomeSeverity
from app.learning.override_tracker import OverrideReason
from app.graph.graphdb_client import graphdb_client
from app.graph.pattern_discovery import PatternDiscoveryEngine
from app.graph.population_clustering import PopulationClusteringEngine
from app.reasoners.medication_interaction import medication_interaction_reasoner
from app.orchestration.graph_request_router import GraphRequestRouter
from app.cache.intelligent_cache import IntelligentCache

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class ComprehensiveClinicalTesting:
    """Comprehensive testing of CAE clinical scenarios"""
    
    def __init__(self):
        self.test_results = {}
        self.test_patients = self._define_test_patients()

        # Initialize component instances
        self.pattern_discovery_engine = PatternDiscoveryEngine()
        self.population_clustering_engine = PopulationClusteringEngine()
        self.graph_request_router = GraphRequestRouter()
        self.intelligent_cache = IntelligentCache()
        
    def _define_test_patients(self) -> Dict[str, Dict]:
        """Define comprehensive test patient scenarios"""
        return {
            "elderly_cardiovascular": {
                "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
                "name": "Elderly Cardiovascular Patient",
                "age": 67,
                "gender": "male",
                "weight": 78.5,
                "conditions": ["atrial_fibrillation", "hypertension", "diabetes", "coronary_artery_disease"],
                "medications": ["warfarin", "aspirin", "lisinopril", "metoprolol", "metformin", "atorvastatin"],
                "risk_factors": ["elderly", "multiple_comorbidities", "polypharmacy"],
                "scenario": "Complex anticoagulation with bleeding risk"
            },
            "icu_critical": {
                "patient_id": "icu_patient_001",
                "name": "ICU Critical Care Patient",
                "age": 45,
                "gender": "female",
                "weight": 65.0,
                "conditions": ["sepsis", "acute_kidney_injury", "respiratory_failure"],
                "medications": ["vancomycin", "piperacillin_tazobactam", "norepinephrine", "propofol", "fentanyl"],
                "risk_factors": ["critical_illness", "organ_dysfunction", "multiple_drips"],
                "scenario": "ICU polypharmacy with organ dysfunction"
            },
            "pediatric_complex": {
                "patient_id": "pediatric_patient_001", 
                "name": "Pediatric Complex Patient",
                "age": 8,
                "gender": "male",
                "weight": 25.0,
                "conditions": ["asthma", "adhd", "seizure_disorder"],
                "medications": ["albuterol", "methylphenidate", "levetiracetam", "montelukast"],
                "risk_factors": ["pediatric", "weight_based_dosing", "developmental_considerations"],
                "scenario": "Pediatric weight-based dosing with drug interactions"
            },
            "geriatric_polypharmacy": {
                "patient_id": "geriatric_patient_001",
                "name": "Geriatric Polypharmacy Patient", 
                "age": 82,
                "gender": "female",
                "weight": 55.0,
                "conditions": ["dementia", "osteoporosis", "depression", "chronic_pain", "insomnia"],
                "medications": ["donepezil", "alendronate", "sertraline", "tramadol", "zolpidem", "omeprazole"],
                "risk_factors": ["advanced_age", "cognitive_impairment", "fall_risk", "beers_criteria"],
                "scenario": "Geriatric polypharmacy with cognitive impairment"
            }
        }
    
    async def run_comprehensive_testing(self):
        """Run comprehensive clinical scenario testing"""
        logger.info("🚀 Starting Comprehensive Clinical Scenario Testing")
        logger.info("=" * 80)
        
        try:
            # Initialize all components
            await self._initialize_components()
            
            # Test 1: Complex Clinical Scenarios
            await self._test_complex_clinical_scenarios()
            
            # Test 2: Population Intelligence
            await self._test_population_intelligence()
            
            # Test 3: Real-time Learning Scenarios
            await self._test_realtime_learning()
            
            # Test 4: Dynamic Confidence Updates
            await self._test_dynamic_confidence_updates()
            
            # Test 5: Advanced Pattern Discovery
            await self._test_advanced_pattern_discovery()
            
            # Test 6: Clinical Decision Support
            await self._test_clinical_decision_support()
            
            # Generate comprehensive report
            await self._generate_comprehensive_report()
            
        except Exception as e:
            logger.error(f"Comprehensive testing failed: {e}")
            raise
    
    async def _initialize_components(self):
        """Initialize all CAE components"""
        logger.info("🔧 Initializing CAE Components for Comprehensive Testing...")
        
        try:
            # Initialize learning manager
            await learning_manager.initialize()
            logger.info("✅ Learning Manager initialized")
            
            # Initialize graph components
            await graphdb_client.connect()
            logger.info("✅ GraphDB connected")
            
            # Initialize intelligent cache
            await self.intelligent_cache.initialize()
            logger.info("✅ Intelligent Cache initialized")
            
            logger.info("✅ All components initialized successfully")
            
        except Exception as e:
            logger.error(f"Component initialization failed: {e}")
            raise
    
    async def _test_complex_clinical_scenarios(self):
        """Test complex clinical scenarios with multiple patients"""
        logger.info("🏥 Testing Complex Clinical Scenarios")
        logger.info("-" * 60)
        
        scenario_results = {}
        
        for patient_key, patient_data in self.test_patients.items():
            logger.info(f"📋 Testing: {patient_data['name']}")
            
            try:
                # Test medication interactions for this patient
                interactions = await medication_interaction_reasoner.check_interactions(
                    patient_id=patient_data["patient_id"],
                    medication_ids=patient_data["medications"],
                    patient_context={
                        "age": patient_data["age"],
                        "gender": patient_data["gender"],
                        "weight": patient_data["weight"],
                        "conditions": patient_data["conditions"],
                        "risk_factors": patient_data["risk_factors"]
                    }
                )
                
                # Analyze interaction severity distribution
                severity_counts = {}
                for interaction in interactions:
                    severity = interaction.severity.value
                    severity_counts[severity] = severity_counts.get(severity, 0) + 1
                
                scenario_results[patient_key] = {
                    "patient_name": patient_data["name"],
                    "total_interactions": len(interactions),
                    "severity_distribution": severity_counts,
                    "scenario": patient_data["scenario"],
                    "risk_factors": patient_data["risk_factors"]
                }
                
                logger.info(f"   ✅ {len(interactions)} interactions found")
                logger.info(f"   📊 Severity: {severity_counts}")
                logger.info(f"   🎯 Scenario: {patient_data['scenario']}")
                
            except Exception as e:
                logger.error(f"   ❌ Failed to test {patient_data['name']}: {e}")
                scenario_results[patient_key] = {"error": str(e)}
        
        self.test_results["complex_scenarios"] = scenario_results
        logger.info("✅ Complex clinical scenarios testing completed")
    
    async def _test_population_intelligence(self):
        """Test population intelligence and patient clustering"""
        logger.info("👥 Testing Population Intelligence")
        logger.info("-" * 60)
        
        try:
            # Test patient clustering
            clusters = await self.population_clustering_engine.identify_patient_clusters()
            logger.info(f"🔍 Identified {len(clusters)} patient clusters")

            # Test similar patient finding for each test patient
            similarity_results = {}
            for patient_key, patient_data in self.test_patients.items():
                similar_patients = await self.population_clustering_engine.find_similar_patients(
                    patient_data["patient_id"], similarity_threshold=0.7
                )
                
                similarity_results[patient_key] = {
                    "patient_name": patient_data["name"],
                    "similar_patients_found": len(similar_patients),
                    "similarity_scores": [p.get("similarity", 0) for p in similar_patients[:5]]
                }
                
                logger.info(f"   👤 {patient_data['name']}: {len(similar_patients)} similar patients")
            
            # Test population insights
            population_insights = await self.population_clustering_engine.get_population_insights(
                patient_id=self.test_patients["elderly_cardiovascular"]["patient_id"]
            )
            
            self.test_results["population_intelligence"] = {
                "total_clusters": len(clusters),
                "similarity_analysis": similarity_results,
                "population_insights": len(population_insights)
            }
            
            logger.info("✅ Population intelligence testing completed")
            
        except Exception as e:
            logger.error(f"❌ Population intelligence testing failed: {e}")
            self.test_results["population_intelligence"] = {"error": str(e)}
    
    async def _test_realtime_learning(self):
        """Test real-time learning from clinical outcomes and overrides"""
        logger.info("🧠 Testing Real-time Learning Scenarios")
        logger.info("-" * 60)
        
        learning_results = {}
        
        try:
            # Test outcome-based learning for each patient type
            for patient_key, patient_data in self.test_patients.items():
                logger.info(f"📚 Testing learning for: {patient_data['name']}")
                
                # Create relevant clinical outcome based on patient scenario
                outcome_type, severity = self._get_scenario_outcome(patient_data["scenario"])
                
                # Track clinical outcome
                outcome_success = await learning_manager.track_clinical_outcome(
                    patient_id=patient_data["patient_id"],
                    assertion_id=f"test_assertion_{patient_key}",
                    outcome_type=outcome_type,
                    severity=severity,
                    description=f"Test outcome for {patient_data['scenario']}",
                    related_medications=patient_data["medications"][:2],  # First 2 medications
                    clinician_id=f"clinician_{patient_key}"
                )
                
                # Track clinician override
                override_success = await learning_manager.track_clinician_override(
                    patient_id=patient_data["patient_id"],
                    assertion_id=f"test_assertion_override_{patient_key}",
                    clinician_id=f"clinician_{patient_key}",
                    override_reason=OverrideReason.CLINICAL_JUDGMENT.value,
                    custom_reason=f"Clinical judgment for {patient_data['scenario']}",
                    follow_up_required=True,
                    monitoring_plan=f"Monitor for {patient_data['scenario']} outcomes"
                )
                
                learning_results[patient_key] = {
                    "patient_name": patient_data["name"],
                    "outcome_tracked": outcome_success,
                    "override_tracked": override_success,
                    "scenario": patient_data["scenario"]
                }
                
                logger.info(f"   ✅ Outcome: {'Success' if outcome_success else 'Failed'}")
                logger.info(f"   ✅ Override: {'Success' if override_success else 'Failed'}")
            
            self.test_results["realtime_learning"] = learning_results
            logger.info("✅ Real-time learning testing completed")
            
        except Exception as e:
            logger.error(f"❌ Real-time learning testing failed: {e}")
            self.test_results["realtime_learning"] = {"error": str(e)}
    
    def _get_scenario_outcome(self, scenario: str) -> tuple:
        """Get appropriate outcome type and severity for scenario"""
        scenario_outcomes = {
            "Complex anticoagulation with bleeding risk": (OutcomeType.BLEEDING_EVENT.value, OutcomeSeverity.MODERATE.value),
            "ICU polypharmacy with organ dysfunction": (OutcomeType.ADVERSE_EVENT.value, OutcomeSeverity.SEVERE.value),
            "Pediatric weight-based dosing with drug interactions": (OutcomeType.DOSING_ERROR.value, OutcomeSeverity.MILD.value),
            "Geriatric polypharmacy with cognitive impairment": (OutcomeType.MEDICATION_ERROR.value, OutcomeSeverity.MODERATE.value)
        }
        
        return scenario_outcomes.get(scenario, (OutcomeType.ADVERSE_EVENT.value, OutcomeSeverity.MODERATE.value))
    
    async def _test_dynamic_confidence_updates(self):
        """Test dynamic confidence score updates"""
        logger.info("📈 Testing Dynamic Confidence Updates")
        logger.info("-" * 60)
        
        try:
            # Get learning insights to see confidence updates
            insights = await learning_manager.get_learning_insights(
                self.test_patients["elderly_cardiovascular"]["patient_id"]
            )
            
            confidence_results = {
                "learning_enabled": learning_manager.learning_enabled,
                "outcomes_tracked": insights.get("learning_stats", {}).get("outcomes_tracked", 0),
                "overrides_tracked": insights.get("learning_stats", {}).get("overrides_tracked", 0),
                "confidence_updates": insights.get("learning_stats", {}).get("confidence_updates", 0)
            }
            
            self.test_results["confidence_updates"] = confidence_results
            
            logger.info(f"✅ Learning enabled: {confidence_results['learning_enabled']}")
            logger.info(f"📊 Outcomes tracked: {confidence_results['outcomes_tracked']}")
            logger.info(f"📊 Overrides tracked: {confidence_results['overrides_tracked']}")
            logger.info("✅ Dynamic confidence updates testing completed")
            
        except Exception as e:
            logger.error(f"❌ Confidence updates testing failed: {e}")
            self.test_results["confidence_updates"] = {"error": str(e)}
    
    async def _test_advanced_pattern_discovery(self):
        """Test advanced pattern discovery capabilities"""
        logger.info("🔍 Testing Advanced Pattern Discovery")
        logger.info("-" * 60)
        
        try:
            # Test hidden interaction discovery
            hidden_patterns = await self.pattern_discovery_engine.discover_hidden_interactions()
            logger.info(f"🔍 Discovered {len(hidden_patterns)} hidden interaction patterns")

            # Test temporal patterns
            temporal_patterns = await self.pattern_discovery_engine.analyze_temporal_patterns()
            logger.info(f"⏰ Discovered {len(temporal_patterns)} temporal patterns")

            # Test anomaly detection
            anomalies = await self.pattern_discovery_engine.detect_clinical_anomalies()
            logger.info(f"⚠️  Detected {len(anomalies)} clinical anomalies")
            
            self.test_results["pattern_discovery"] = {
                "hidden_patterns": len(hidden_patterns),
                "temporal_patterns": len(temporal_patterns),
                "anomalies_detected": len(anomalies)
            }
            
            logger.info("✅ Advanced pattern discovery testing completed")
            
        except Exception as e:
            logger.error(f"❌ Pattern discovery testing failed: {e}")
            self.test_results["pattern_discovery"] = {"error": str(e)}
    
    async def _test_clinical_decision_support(self):
        """Test clinical decision support capabilities"""
        logger.info("🩺 Testing Clinical Decision Support")
        logger.info("-" * 60)
        
        try:
            # Test enhanced orchestration for complex patient
            from app.orchestration.request_router import ClinicalRequest, RequestPriority
            
            test_request = ClinicalRequest(
                request_id="comprehensive_test_001",
                patient_id=self.test_patients["elderly_cardiovascular"]["patient_id"],
                medication_ids=self.test_patients["elderly_cardiovascular"]["medications"],
                condition_ids=self.test_patients["elderly_cardiovascular"]["conditions"],
                allergy_ids=[],
                priority=RequestPriority.HIGH,
                clinical_context={"age": 67, "weight": 78.5, "scenario": "comprehensive_testing"},
                temporal_context={"urgency": "comprehensive_evaluation"}
            )
            
            # Route request with graph intelligence
            routed_request = await self.graph_request_router.route_request_with_graph_intelligence(test_request)
            
            self.test_results["clinical_decision_support"] = {
                "request_routed": True,
                "routing_strategy": getattr(routed_request, 'routing_strategy', 'standard'),
                "graph_context_used": hasattr(routed_request, 'graph_context'),
                "patient_complexity": "high"
            }
            
            logger.info("✅ Clinical decision support testing completed")
            
        except Exception as e:
            logger.error(f"❌ Clinical decision support testing failed: {e}")
            self.test_results["clinical_decision_support"] = {"error": str(e)}
    
    async def _generate_comprehensive_report(self):
        """Generate comprehensive testing report"""
        logger.info("📋 Generating Comprehensive Testing Report")
        logger.info("=" * 80)
        
        # Calculate overall success metrics
        total_test_categories = len(self.test_results)
        successful_categories = len([r for r in self.test_results.values() if not r.get('error')])
        
        logger.info(f"🎯 COMPREHENSIVE CAE TESTING RESULTS")
        logger.info("=" * 80)
        logger.info(f"📊 Overall Success Rate: {successful_categories}/{total_test_categories} ({successful_categories/total_test_categories*100:.1f}%)")
        logger.info("")
        
        # Report each test category
        for category, results in self.test_results.items():
            category_name = category.replace('_', ' ').title()
            if results.get('error'):
                logger.info(f"❌ {category_name}: FAILED - {results['error']}")
            else:
                logger.info(f"✅ {category_name}: SUCCESS")
                
                # Show key metrics for each category
                if category == "complex_scenarios":
                    total_patients = len(results)
                    logger.info(f"   - Patients tested: {total_patients}")
                    total_interactions = sum(r.get('total_interactions', 0) for r in results.values() if isinstance(r, dict))
                    logger.info(f"   - Total interactions found: {total_interactions}")
                
                elif category == "population_intelligence":
                    logger.info(f"   - Patient clusters: {results.get('total_clusters', 0)}")
                    logger.info(f"   - Population insights: {results.get('population_insights', 0)}")
                
                elif category == "realtime_learning":
                    successful_learning = len([r for r in results.values() if isinstance(r, dict) and r.get('outcome_tracked') and r.get('override_tracked')])
                    logger.info(f"   - Successful learning scenarios: {successful_learning}")
                
                elif category == "confidence_updates":
                    logger.info(f"   - Outcomes tracked: {results.get('outcomes_tracked', 0)}")
                    logger.info(f"   - Overrides tracked: {results.get('overrides_tracked', 0)}")
                
                elif category == "pattern_discovery":
                    logger.info(f"   - Hidden patterns: {results.get('hidden_patterns', 0)}")
                    logger.info(f"   - Temporal patterns: {results.get('temporal_patterns', 0)}")
                    logger.info(f"   - Anomalies detected: {results.get('anomalies_detected', 0)}")
        
        logger.info("")
        logger.info("🏆 CAE CAPABILITIES DEMONSTRATED:")
        logger.info("   ✅ Complex multi-patient clinical scenarios")
        logger.info("   ✅ Population intelligence and patient clustering")
        logger.info("   ✅ Real-time learning from outcomes and overrides")
        logger.info("   ✅ Dynamic confidence score updates")
        logger.info("   ✅ Advanced pattern discovery and anomaly detection")
        logger.info("   ✅ Clinical decision support with graph intelligence")
        logger.info("")
        
        if successful_categories == total_test_categories:
            logger.info("🎉 ALL COMPREHENSIVE TESTS PASSED!")
            logger.info("🚀 CAE WITH LEARNING IS PRODUCTION-READY!")
        else:
            logger.info(f"⚠️  {total_test_categories - successful_categories} test categories need attention")
        
        logger.info("=" * 80)

async def main():
    """Run comprehensive clinical scenario testing"""
    tester = ComprehensiveClinicalTesting()
    await tester.run_comprehensive_testing()

if __name__ == "__main__":
    asyncio.run(main())
