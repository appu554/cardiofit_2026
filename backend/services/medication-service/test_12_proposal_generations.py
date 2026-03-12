"""
12 PROPOSAL GENERATION Examples

This test demonstrates the variety of pharmaceutical intelligence and proposal formats
that the Medication Service can generate for different medications and clinical scenarios.

Each test shows the complete STEP 5: PROPOSAL GENERATION output format.
"""

import asyncio
import logging
import sys
import time
from datetime import datetime

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def test_12_proposal_generations():
    """Test 12 different proposal generation scenarios"""
    
    try:
        # Add the medication service to Python path
        import os
        current_dir = os.path.dirname(os.path.abspath(__file__))
        sys.path.insert(0, current_dir)
        
        from app.domain.services.recipe_orchestrator import RecipeOrchestrator, MedicationSafetyRequest
        
        # Initialize orchestrator
        orchestrator = RecipeOrchestrator(
            context_service_url="http://localhost:8016",
            enable_safety_gateway=False
        )
        
        logger.info("🚀 12 PROPOSAL GENERATION Examples")
        logger.info("🎯 Demonstrating pharmaceutical intelligence variety")
        logger.info("=" * 80)
        
        # Define 12 different medication scenarios
        test_scenarios = [
            {
                "name": "1. Pain Management - Acetaminophen",
                "medication": {"name": "Acetaminophen", "indication": "pain_management"},
                "expected": "Standard analgesic with hepatic considerations"
            },
            {
                "name": "2. Anticoagulation - Warfarin",
                "medication": {"name": "Warfarin", "indication": "atrial_fibrillation", "is_high_risk": True},
                "expected": "Complex anticoagulation protocol with INR monitoring"
            },
            {
                "name": "3. Diabetes Management - Metformin",
                "medication": {"name": "Metformin", "indication": "diabetes_type_2"},
                "expected": "Diabetes management with renal function assessment"
            },
            {
                "name": "4. Hypertension - Lisinopril",
                "medication": {"name": "Lisinopril", "indication": "hypertension"},
                "expected": "ACE inhibitor with renal and electrolyte monitoring"
            },
            {
                "name": "5. Infection - Amoxicillin",
                "medication": {"name": "Amoxicillin", "indication": "bacterial_infection"},
                "expected": "Antibiotic with allergy screening"
            },
            {
                "name": "6. Heart Failure - Digoxin",
                "medication": {"name": "Digoxin", "indication": "heart_failure", "is_high_risk": True},
                "expected": "Cardiac glycoside with therapeutic drug monitoring"
            },
            {
                "name": "7. Pain Management - Morphine",
                "medication": {"name": "Morphine", "indication": "severe_pain", "is_high_risk": True},
                "expected": "Opioid with respiratory monitoring and addiction risk"
            },
            {
                "name": "8. Cholesterol - Atorvastatin",
                "medication": {"name": "Atorvastatin", "indication": "hyperlipidemia"},
                "expected": "Statin with liver function monitoring"
            },
            {
                "name": "9. Seizure - Phenytoin",
                "medication": {"name": "Phenytoin", "indication": "epilepsy", "is_high_risk": True},
                "expected": "Antiepileptic with therapeutic drug monitoring"
            },
            {
                "name": "10. Asthma - Albuterol",
                "medication": {"name": "Albuterol", "indication": "asthma"},
                "expected": "Bronchodilator with cardiac monitoring"
            },
            {
                "name": "11. Depression - Sertraline",
                "medication": {"name": "Sertraline", "indication": "depression"},
                "expected": "SSRI with suicide risk assessment"
            },
            {
                "name": "12. Thyroid - Levothyroxine",
                "medication": {"name": "Levothyroxine", "indication": "hypothyroidism"},
                "expected": "Thyroid hormone with TSH monitoring"
            }
        ]
        
        proposals_generated = []
        
        for i, scenario in enumerate(test_scenarios, 1):
            logger.info(f"\n🧪 {scenario['name']}")
            logger.info(f"   Expected: {scenario['expected']}")
            logger.info("-" * 60)
            
            try:
                # Create medication request
                request = MedicationSafetyRequest(
                    patient_id="905a60cb-8241-418f-b29b-5b020e851392",
                    medication=scenario["medication"],
                    provider_id="test-provider-001",
                    action_type="prescribe",
                    urgency="routine"
                )
                
                # Execute pharmaceutical intelligence
                start_time = time.time()
                result = await orchestrator.execute_medication_safety(request)
                execution_time = time.time() - start_time
                
                # Extract proposal information
                proposal = {
                    "scenario": scenario["name"],
                    "medication_name": scenario["medication"]["name"],
                    "indication": scenario["medication"]["indication"],
                    "clinical_recipes_executed": len(result.clinical_recipes_executed),
                    "recipe_names": result.clinical_recipes_executed,
                    "overall_safety_status": result.overall_safety_status,
                    "context_completeness": f"{result.context_completeness_score:.1%}",
                    "execution_time_ms": f"{execution_time * 1000:.1f}ms",
                    "clinical_decision_support": None,
                    "safety_summary": None
                }
                
                # Extract clinical decision support
                if result.safety_summary and result.safety_summary.get('clinical_decision_support'):
                    cds = result.safety_summary['clinical_decision_support']
                    proposal["clinical_decision_support"] = {
                        "provider_summary": cds.get('provider_summary', 'N/A'),
                        "patient_explanation": cds.get('patient_explanation', 'N/A'),
                        "monitoring_requirements": len(cds.get('monitoring_requirements', []))
                    }
                
                # Extract safety summary
                if result.safety_summary:
                    proposal["safety_summary"] = {
                        "total_validations": result.safety_summary.get('total_validations', 0),
                        "critical_issues": result.safety_summary.get('critical_issues', 0),
                        "high_issues": result.safety_summary.get('high_issues', 0),
                        "context_safety_flags": result.safety_summary.get('context_safety_flags', 0),
                        "recommendations": len(result.safety_summary.get('recommendations', []))
                    }
                
                proposals_generated.append(proposal)
                
                # Display proposal details
                logger.info(f"   📊 Clinical Recipes: {proposal['clinical_recipes_executed']}")
                logger.info(f"   📊 Recipe Names: {', '.join(proposal['recipe_names'])}")
                logger.info(f"   📊 Safety Status: {proposal['overall_safety_status']}")
                logger.info(f"   📊 Context Completeness: {proposal['context_completeness']}")
                logger.info(f"   📊 Execution Time: {proposal['execution_time_ms']}")
                
                if proposal["clinical_decision_support"]:
                    cds = proposal["clinical_decision_support"]
                    logger.info(f"   📋 Provider Summary: {cds['provider_summary'][:80]}...")
                    logger.info(f"   📋 Patient Explanation: {cds['patient_explanation'][:80]}...")
                    logger.info(f"   📋 Monitoring Requirements: {cds['monitoring_requirements']}")
                
                if proposal["safety_summary"]:
                    ss = proposal["safety_summary"]
                    logger.info(f"   🛡️ Total Validations: {ss['total_validations']}")
                    logger.info(f"   🛡️ Safety Issues: {ss['critical_issues']} critical, {ss['high_issues']} high")
                    logger.info(f"   🛡️ Safety Flags: {ss['context_safety_flags']}")
                
                logger.info(f"   ✅ PROPOSAL {i}: GENERATED SUCCESSFULLY")
                
            except Exception as e:
                logger.error(f"   ❌ PROPOSAL {i}: FAILED - {e}")
                proposals_generated.append({
                    "scenario": scenario["name"],
                    "error": str(e)
                })
        
        # Summary of all proposals
        logger.info("\n" + "=" * 80)
        logger.info("🎯 12 PROPOSAL GENERATION SUMMARY")
        logger.info("=" * 80)
        
        successful_proposals = [p for p in proposals_generated if "error" not in p]
        failed_proposals = [p for p in proposals_generated if "error" in p]
        
        logger.info(f"📊 SUCCESSFUL PROPOSALS: {len(successful_proposals)}/12")
        logger.info(f"📊 FAILED PROPOSALS: {len(failed_proposals)}/12")
        
        if successful_proposals:
            logger.info("\n🎉 SUCCESSFUL PROPOSAL FORMATS:")
            logger.info("-" * 50)
            
            for proposal in successful_proposals:
                logger.info(f"\n✅ {proposal['scenario']}")
                logger.info(f"   Medication: {proposal['medication_name']}")
                logger.info(f"   Indication: {proposal['indication']}")
                logger.info(f"   Clinical Processing: {proposal['clinical_recipes_executed']} recipes")
                logger.info(f"   Safety Assessment: {proposal['overall_safety_status']}")
                logger.info(f"   Context Integration: {proposal['context_completeness']}")
                logger.info(f"   Performance: {proposal['execution_time_ms']}")
                
                if proposal.get("clinical_decision_support"):
                    logger.info(f"   Clinical Decision Support: ✅ Generated")
                if proposal.get("safety_summary"):
                    logger.info(f"   Safety Summary: ✅ Generated")
        
        if failed_proposals:
            logger.info("\n❌ FAILED PROPOSALS:")
            logger.info("-" * 30)
            for proposal in failed_proposals:
                logger.info(f"   {proposal['scenario']}: {proposal['error']}")
        
        # Analysis of proposal variety
        logger.info("\n📊 PROPOSAL VARIETY ANALYSIS:")
        logger.info("-" * 40)
        
        if successful_proposals:
            # Recipe variety
            all_recipes = []
            for p in successful_proposals:
                all_recipes.extend(p['recipe_names'])
            unique_recipes = list(set(all_recipes))
            
            logger.info(f"   Unique Clinical Recipes Used: {len(unique_recipes)}")
            for recipe in unique_recipes:
                count = sum(1 for p in successful_proposals if recipe in p['recipe_names'])
                logger.info(f"      - {recipe}: {count} times")
            
            # Safety status variety
            safety_statuses = [p['overall_safety_status'] for p in successful_proposals]
            unique_statuses = list(set(safety_statuses))
            logger.info(f"   Safety Status Variety: {unique_statuses}")
            
            # Performance analysis
            execution_times = [float(p['execution_time_ms'].replace('ms', '')) for p in successful_proposals]
            avg_time = sum(execution_times) / len(execution_times)
            logger.info(f"   Average Execution Time: {avg_time:.1f}ms")
            logger.info(f"   Performance Range: {min(execution_times):.1f}ms - {max(execution_times):.1f}ms")
        
        success_rate = len(successful_proposals) / 12 * 100
        
        logger.info(f"\n🎯 OVERALL SUCCESS RATE: {len(successful_proposals)}/12 ({success_rate:.1f}%)")
        
        if success_rate >= 90:
            logger.info("🎉 PROPOSAL GENERATION: EXCELLENT!")
            logger.info("✅ Medication Service demonstrates diverse pharmaceutical intelligence")
        elif success_rate >= 75:
            logger.info("⚠️ PROPOSAL GENERATION: GOOD")
            logger.info("🔧 Most proposal types working well")
        else:
            logger.info("❌ PROPOSAL GENERATION: NEEDS IMPROVEMENT")
            logger.info("🚨 Significant issues with proposal variety")
        
        return success_rate >= 75
        
    except Exception as e:
        logger.error(f"❌ Test execution failed: {e}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Main test execution"""
    try:
        success = await test_12_proposal_generations()
        
        if success:
            logger.info("\n🎉 12 Proposal Generation test completed successfully!")
            logger.info("🧠 Medication Service demonstrates comprehensive pharmaceutical intelligence!")
            sys.exit(0)
        else:
            logger.error("\n💥 Some proposal generation tests failed!")
            sys.exit(1)
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {e}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
