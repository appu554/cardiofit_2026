#!/usr/bin/env python3
"""
Direct Orchestration Engine Test

This script tests the CAE orchestration engine directly without using gRPC.
It helps isolate issues in the core orchestration logic.
"""

import asyncio
import logging
import sys
import os
from datetime import datetime
from typing import Dict, Any, List

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Import CAE components directly
from app.orchestration.orchestration_engine import OrchestrationEngine
from app.orchestration.request_router import ClinicalRequest
from app.orchestration.parallel_executor import ParallelExecutor
from app.orchestration.decision_aggregator import DecisionAggregator
from app.orchestration.priority_queue import PriorityQueue
from app.reasoners.medication_interaction import MedicationInteractionReasoner
from app.reasoners.dosing_calculator import DosingCalculator
from app.reasoners.contraindication import ContraindicationReasoner
from app.reasoners.duplicate_therapy import DuplicateTherapyReasoner
from app.reasoners.clinical_context import ClinicalContextReasoner
from app.context.patient_context_assembler import PatientContextAssembler

async def test_direct_orchestration():
    """Test the orchestration engine directly"""
    
    logger.info("🚀 Direct Orchestration Engine Test")
    logger.info("=" * 70)
    
    try:
        # Initialize components
        logger.info("Initializing components...")
        
        # Initialize core components
        parallel_executor = ParallelExecutor()
        decision_aggregator = DecisionAggregator()
        priority_queue = PriorityQueue()
        
        # Initialize reasoners
        medication_interaction_reasoner = MedicationInteractionReasoner()
        dosing_calculator = DosingCalculator()
        contraindication_reasoner = ContraindicationReasoner()
        duplicate_therapy_reasoner = DuplicateTherapyReasoner()
        clinical_context_reasoner = ClinicalContextReasoner()
        
        # Initialize context assembler
        patient_context_assembler = PatientContextAssembler()
        
        # Initialize orchestration engine
        orchestration_engine = OrchestrationEngine(
            parallel_executor=parallel_executor,
            decision_aggregator=decision_aggregator,
            priority_queue=priority_queue,
            context_enrichment_engine=None  # We'll test without context enrichment first
        )
        
        # Register reasoners
        orchestration_engine.register_reasoner("interaction", medication_interaction_reasoner)
        orchestration_engine.register_reasoner("dosing", dosing_calculator)
        orchestration_engine.register_reasoner("contraindication", contraindication_reasoner)
        orchestration_engine.register_reasoner("duplicate_therapy", duplicate_therapy_reasoner)
        orchestration_engine.register_reasoner("clinical_context", clinical_context_reasoner)
        
        # Start the orchestration engine
        await orchestration_engine.start()
        logger.info("✅ Orchestration engine initialized and started")
        
        # Create a test request
        test_request = {
            "request_id": "test-direct-001",
            "patient_id": "patient-test-001",
            "medication_ids": ["med-metformin-500mg", "med-lisinopril-10mg"],
            "condition_ids": ["cond-type2diabetes", "cond-hypertension"],
            "clinical_context": {
                "patient_id": "patient-test-001",
                "demographics": {
                    "age": 65,
                    "gender": "male",
                    "weight_kg": 80
                },
                "allergies": [],
                "lab_results": [
                    {"code": "glucose", "value": 180, "unit": "mg/dL"},
                    {"code": "hba1c", "value": 7.8, "unit": "%"}
                ]
            }
        }
        
        logger.info(f"📤 Testing with request: {test_request['request_id']}")
        logger.info(f"   Patient ID: {test_request['patient_id']}")
        logger.info(f"   Medications: {test_request['medication_ids']}")
        logger.info(f"   Conditions: {test_request['condition_ids']}")
        
        # Process the request
        start_time = datetime.now()
        logger.info("⏳ Processing request...")
        
        assertions = await orchestration_engine.generate_clinical_assertions(test_request)
        
        end_time = datetime.now()
        processing_time = (end_time - start_time).total_seconds() * 1000
        
        # Display results
        logger.info(f"✅ Request processed in {processing_time:.1f}ms")
        logger.info(f"📊 Generated {len(assertions)} assertions")
        
        for i, assertion in enumerate(assertions, 1):
            logger.info(f"   {i}. Type: {assertion.type}")
            logger.info(f"      Title: {assertion.title}")
            logger.info(f"      Severity: {assertion.severity}")
            logger.info(f"      Description: {assertion.description}")
            logger.info(f"      Confidence: {assertion.confidence_score}")
            logger.info("")
        
        # Stop the orchestration engine
        await orchestration_engine.stop()
        logger.info("✅ Orchestration engine stopped")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return False

if __name__ == "__main__":
    success = asyncio.run(test_direct_orchestration())
    sys.exit(0 if success else 1)
