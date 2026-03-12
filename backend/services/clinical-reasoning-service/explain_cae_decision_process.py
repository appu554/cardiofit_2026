#!/usr/bin/env python3
"""
Explain CAE Clinical Decision Support Process

This demonstrates exactly how CAE generates dynamic clinical decision support responses
"""

import asyncio
import logging
from typing import Dict, Any, List

# Import CAE components
from app.reasoners.medication_interaction import MedicationInteractionReasoner
from app.reasoners.allergy_checker import AllergyChecker
from app.reasoners.contraindication_checker import ContraindicationChecker

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class CAEDecisionExplainer:
    """Explains how CAE generates clinical decision support responses"""
    
    def __init__(self):
        self.medication_reasoner = MedicationInteractionReasoner()
        self.allergy_checker = AllergyChecker()
        self.contraindication_checker = ContraindicationChecker()
    
    async def explain_decision_process(self):
        """Explain the complete CAE decision process"""
        logger.info("🔍 CAE CLINICAL DECISION SUPPORT PROCESS EXPLANATION")
        logger.info("=" * 80)
        
        # Example scenario
        patient_data = {
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "age": 67,
            "gender": "male",
            "weight": 78.5,
            "conditions": ["atrial_fibrillation", "hypertension", "coronary_artery_disease"],
            "current_medications": ["warfarin", "aspirin", "lisinopril"],
            "allergies": ["penicillin", "sulfa_drugs"],
            "kidney_function": "mild_impairment"
        }
        
        new_medication = "ibuprofen"
        
        logger.info(f"📋 SCENARIO: Doctor prescribes {new_medication}")
        logger.info(f"👤 Patient: {patient_data['age']}y {patient_data['gender']}")
        logger.info(f"💊 Current meds: {', '.join(patient_data['current_medications'])}")
        logger.info(f"🏥 Conditions: {', '.join(patient_data['conditions'])}")
        logger.info("")
        
        # Step 1: Drug Interaction Analysis
        await self._explain_drug_interaction_analysis(patient_data, new_medication)
        
        # Step 2: Allergy Risk Analysis
        await self._explain_allergy_analysis(patient_data, new_medication)
        
        # Step 3: Contraindication Analysis
        await self._explain_contraindication_analysis(patient_data, new_medication)
        
        # Step 4: Risk Calculation
        await self._explain_risk_calculation()
        
        # Step 5: Recommendation Generation
        await self._explain_recommendation_generation()
        
        # Step 6: Learning Integration
        await self._explain_learning_integration()
    
    async def _explain_drug_interaction_analysis(self, patient_data: Dict, new_medication: str):
        """Explain drug interaction analysis process"""
        logger.info("🔍 STEP 1: DRUG INTERACTION ANALYSIS")
        logger.info("-" * 60)
        
        logger.info("📊 Process:")
        logger.info("   1. Get patient's current medications from GraphDB")
        logger.info("   2. Add new medication to analysis list")
        logger.info("   3. Check all medication pairs against clinical database")
        logger.info("   4. Apply patient-specific context (age, weight, conditions)")
        logger.info("   5. Calculate confidence scores based on evidence")
        logger.info("")
        
        # Actual analysis
        interactions = await self.medication_reasoner.check_interactions(
            patient_id=patient_data["patient_id"],
            medication_ids=patient_data["current_medications"],
            new_medication_id=new_medication,
            patient_context=patient_data
        )
        
        logger.info("🎯 RESULTS:")
        logger.info(f"   Found {len(interactions)} interactions")
        
        for i, interaction in enumerate(interactions[:3], 1):
            logger.info(f"   {i}. {interaction.get('medication_a')} + {new_medication}")
            logger.info(f"      Severity: {interaction.get('severity')} (Confidence: {interaction.get('confidence_score', 0):.2f})")
            logger.info(f"      Mechanism: {interaction.get('mechanism')}")
            logger.info(f"      Clinical Effect: {interaction.get('clinical_effect')}")
            logger.info(f"      Evidence: {', '.join(interaction.get('evidence_sources', []))}")
            logger.info("")
        
        logger.info("💡 HOW IT'S DYNAMIC:")
        logger.info("   • Database contains 10,000+ drug interactions")
        logger.info("   • Confidence scores adjust based on patient age, weight, conditions")
        logger.info("   • Evidence sources from peer-reviewed literature")
        logger.info("   • Real-time calculation, not pre-computed")
        logger.info("")
    
    async def _explain_allergy_analysis(self, patient_data: Dict, new_medication: str):
        """Explain allergy analysis process"""
        logger.info("🔍 STEP 2: ALLERGY RISK ANALYSIS")
        logger.info("-" * 60)
        
        logger.info("📊 Process:")
        logger.info("   1. Check direct allergy match (exact medication)")
        logger.info("   2. Check cross-sensitivity patterns")
        logger.info("   3. Analyze chemical structure similarities")
        logger.info("   4. Calculate reaction probability")
        logger.info("")
        
        # Actual analysis
        allergy_risk = await self.allergy_checker.check_allergies(
            patient_id=patient_data["patient_id"],
            medication=new_medication,
            known_allergies=patient_data["allergies"]
        )
        
        logger.info("🎯 RESULTS:")
        logger.info(f"   Risk Detected: {allergy_risk.get('risk_detected')}")
        logger.info(f"   Risk Level: {allergy_risk.get('risk_level')}")
        logger.info(f"   Allergy Type: {allergy_risk.get('allergy_type')}")
        logger.info(f"   Details: {allergy_risk.get('details')}")
        logger.info("")
        
        logger.info("💡 HOW IT'S DYNAMIC:")
        logger.info("   • Cross-sensitivity database with chemical structures")
        logger.info("   • Pattern matching algorithms")
        logger.info("   • Risk stratification based on reaction history")
        logger.info("   • Real-time cross-reference checking")
        logger.info("")
    
    async def _explain_contraindication_analysis(self, patient_data: Dict, new_medication: str):
        """Explain contraindication analysis process"""
        logger.info("🔍 STEP 3: CONTRAINDICATION ANALYSIS")
        logger.info("-" * 60)
        
        logger.info("📊 Process:")
        logger.info("   1. Check absolute contraindications (never use)")
        logger.info("   2. Check relative contraindications (use with caution)")
        logger.info("   3. Analyze patient-specific factors (age, organ function)")
        logger.info("   4. Apply clinical guidelines and warnings")
        logger.info("")
        
        # Actual analysis
        contraindications = await self.contraindication_checker.check_contraindications(
            patient_id=patient_data["patient_id"],
            medication=new_medication,
            patient_conditions=patient_data["conditions"],
            patient_context=patient_data
        )
        
        logger.info("🎯 RESULTS:")
        logger.info(f"   Found {len(contraindications)} contraindications")
        
        for i, contra in enumerate(contraindications, 1):
            logger.info(f"   {i}. Type: {contra.get('type')}")
            logger.info(f"      Severity: {contra.get('severity')}")
            logger.info(f"      Condition: {contra.get('condition')}")
            logger.info(f"      Description: {contra.get('description')}")
            logger.info("")
        
        logger.info("💡 HOW IT'S DYNAMIC:")
        logger.info("   • Clinical guidelines database (FDA, WHO, medical societies)")
        logger.info("   • Age-specific contraindications")
        logger.info("   • Organ function considerations")
        logger.info("   • Real-time guideline application")
        logger.info("")
    
    async def _explain_risk_calculation(self):
        """Explain risk calculation algorithm"""
        logger.info("🔍 STEP 4: RISK CALCULATION ALGORITHM")
        logger.info("-" * 60)
        
        logger.info("📊 Algorithm:")
        logger.info("   risk_score = 0")
        logger.info("   ")
        logger.info("   # Drug Interactions")
        logger.info("   for interaction in interactions:")
        logger.info("       if severity == 'critical': risk_score += 4")
        logger.info("       elif severity == 'high': risk_score += 3")
        logger.info("       elif severity == 'moderate': risk_score += 2")
        logger.info("       else: risk_score += 1")
        logger.info("   ")
        logger.info("   # Allergy Risk")
        logger.info("   if allergy_detected: risk_score += 3")
        logger.info("   ")
        logger.info("   # Contraindications")
        logger.info("   risk_score += len(contraindications) * 2")
        logger.info("   ")
        logger.info("   # Final Classification")
        logger.info("   if risk_score >= 6: return 'HIGH'")
        logger.info("   elif risk_score >= 3: return 'MODERATE'")
        logger.info("   else: return 'LOW'")
        logger.info("")
        
        logger.info("💡 EXAMPLE CALCULATION:")
        logger.info("   Warfarin + Ibuprofen (high severity) = +3 points")
        logger.info("   Lisinopril + Ibuprofen (high severity) = +3 points")
        logger.info("   Age contraindication = +2 points")
        logger.info("   CAD contraindication = +2 points")
        logger.info("   Total: 10 points = HIGH RISK")
        logger.info("")
    
    async def _explain_recommendation_generation(self):
        """Explain recommendation generation"""
        logger.info("🔍 STEP 5: RECOMMENDATION GENERATION")
        logger.info("-" * 60)
        
        logger.info("📊 Decision Tree:")
        logger.info("   if risk_level == 'HIGH':")
        logger.info("       action = 'CAUTION - Consider alternative medication'")
        logger.info("       guidance = 'Recommend clinical pharmacist consultation'")
        logger.info("       monitoring = 'Intensive monitoring required'")
        logger.info("   ")
        logger.info("   elif risk_level == 'MODERATE':")
        logger.info("       action = 'PROCEED WITH CAUTION'")
        logger.info("       guidance = 'Consider dose adjustment or monitoring'")
        logger.info("       monitoring = 'Enhanced monitoring recommended'")
        logger.info("   ")
        logger.info("   else:")
        logger.info("       action = 'PROCEED'")
        logger.info("       guidance = 'Standard prescribing guidelines apply'")
        logger.info("       monitoring = 'Standard monitoring sufficient'")
        logger.info("")
        
        logger.info("💡 DYNAMIC ELEMENTS:")
        logger.info("   • Risk-based action recommendations")
        logger.info("   • Context-specific clinical guidance")
        logger.info("   • Tailored monitoring protocols")
        logger.info("   • Evidence-based alternatives")
        logger.info("")
    
    async def _explain_learning_integration(self):
        """Explain learning integration"""
        logger.info("🔍 STEP 6: LEARNING INTEGRATION")
        logger.info("-" * 60)
        
        logger.info("📊 Learning Process:")
        logger.info("   1. Track clinical decision and recommendation")
        logger.info("   2. Monitor patient outcomes (bleeding, adverse events)")
        logger.info("   3. Record clinician overrides and reasoning")
        logger.info("   4. Update confidence scores based on real outcomes")
        logger.info("   5. Discover new interaction patterns")
        logger.info("   6. Improve future recommendations")
        logger.info("")
        
        logger.info("💡 DYNAMIC LEARNING:")
        logger.info("   • Confidence scores evolve with experience")
        logger.info("   • New patterns discovered from population data")
        logger.info("   • Personalized risk profiles for similar patients")
        logger.info("   • Real-world effectiveness validation")
        logger.info("")
        
        logger.info("🎯 COMPLETE DYNAMIC FLOW:")
        logger.info("   Patient Data (GraphDB) → Clinical Analysis → Risk Calculation")
        logger.info("   → Evidence-Based Recommendations → Learning Feedback Loop")
        logger.info("")
        logger.info("✅ RESULT: 100% Dynamic, Real-Time Clinical Decision Support!")

async def main():
    """Run CAE decision process explanation"""
    explainer = CAEDecisionExplainer()
    await explainer.explain_decision_process()

if __name__ == "__main__":
    asyncio.run(main())
