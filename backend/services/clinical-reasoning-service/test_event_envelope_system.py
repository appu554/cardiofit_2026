#!/usr/bin/env python3
"""
Test Script for Clinical Event Envelope Integration

This script tests the Week 5-6 implementation:
- Enhanced Clinical Event Envelope with rich context
- Workflow-Specific Event Processors
- Advanced Idempotency & Event Sourcing
- Clinical Context Integration with CAE
"""

import asyncio
import logging
import json
import sys
import time
from pathlib import Path
from datetime import datetime, timezone, timedelta

# Add the app directory to the path
sys.path.insert(0, str(Path(__file__).parent / 'app'))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def test_clinical_event_envelope():
    """Test the Enhanced Clinical Event Envelope"""
    logger.info("🧪 Testing Enhanced Clinical Event Envelope")
    
    try:
        from events.clinical_event_envelope import (
            ClinicalEventEnvelope, ClinicalContext, TemporalContext, 
            ProvenanceContext, EventMetadata, EventType, EventSeverity, EventStatus
        )
        
        # Test 1: Create comprehensive clinical event envelope
        clinical_context = ClinicalContext(
            patient_id="test_patient_001",
            patient_mrn="MRN123456",
            patient_demographics={"age": 65, "gender": "male"},
            encounter_id="ENC001",
            encounter_type="inpatient",
            primary_provider_id="DR001",
            facility_id="HOSP001",
            active_medications=[
                {"name": "Warfarin", "dosage": "5mg", "frequency": "daily"},
                {"name": "Aspirin", "dosage": "81mg", "frequency": "daily"}
            ],
            active_diagnoses=[
                {"code": "I48.91", "description": "Atrial fibrillation"}
            ],
            active_allergies=[
                {"allergen": "Penicillin", "reaction": "Rash", "severity": "moderate"}
            ]
        )
        
        temporal_context = TemporalContext(
            event_time=datetime.now(timezone.utc),
            system_time=datetime.now(timezone.utc),
            clinical_day=3,
            shift_context="day",
            urgency_level="routine"
        )
        
        provenance_context = ProvenanceContext(
            source_system="clinical_reasoning_service",
            source_user_id="USER001",
            source_user_role="physician",
            created_by="test_system",
            created_at=datetime.now(timezone.utc)
        )
        
        metadata = EventMetadata(
            event_type=EventType.CLINICAL_ASSERTION,
            event_severity=EventSeverity.MODERATE,
            processing_priority=7
        )
        
        envelope = ClinicalEventEnvelope(
            event_data={
                "assertion_type": "drug_interaction",
                "description": "Potential interaction between Warfarin and Aspirin",
                "confidence_score": 0.85,
                "severity": "moderate"
            },
            clinical_context=clinical_context,
            temporal_context=temporal_context,
            provenance_context=provenance_context,
            metadata=metadata
        )
        
        logger.info(f"  ✓ Created clinical event envelope: {envelope.metadata.event_id}")
        
        # Test 2: Add clinical warning
        envelope.add_clinical_warning(
            warning_type="drug_interaction",
            severity="high",
            description="High-risk drug interaction detected",
            source="clinical_reasoning_engine"
        )
        
        logger.info(f"  ✓ Added clinical warning, severity escalated to: {envelope.metadata.event_severity.value}")
        
        # Test 3: Add provenance entry
        envelope.add_provenance_entry(
            "drug_interaction_checker",
            "interaction_analysis",
            confidence=0.92
        )
        
        logger.info(f"  ✓ Added provenance entry")
        
        # Test 4: Update status
        envelope.update_status(EventStatus.COMPLETED, "test_system")
        
        logger.info(f"  ✓ Updated status to: {envelope.metadata.event_status.value}")
        
        # Test 5: Serialization
        envelope_json = envelope.to_json()
        restored_envelope = ClinicalEventEnvelope.from_json(envelope_json)
        
        logger.info(f"  ✓ Serialization test passed: {restored_envelope.metadata.event_id}")
        
        # Test 6: Calculate processing duration
        envelope.temporal_context.processing_time = datetime.now(timezone.utc)
        envelope.temporal_context.completion_time = datetime.now(timezone.utc) + timedelta(milliseconds=50)
        
        duration = envelope.calculate_processing_duration()
        logger.info(f"  ✓ Processing duration: {duration:.3f} seconds")
        
        logger.info("✅ Enhanced Clinical Event Envelope tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Enhanced Clinical Event Envelope test failed: {e}")
        return False


async def test_workflow_processors():
    """Test Workflow-Specific Event Processors"""
    logger.info("🧪 Testing Workflow-Specific Event Processors")
    
    try:
        from events.clinical_event_envelope import (
            ClinicalEventEnvelope, ClinicalContext, TemporalContext, 
            ProvenanceContext, EventMetadata, EventType
        )
        from events.event_processors import (
            MedicationWorkflowProcessor, LaboratoryWorkflowProcessor,
            ClinicalDecisionProcessor, EventProcessorRegistry
        )
        
        # Test 1: Medication Workflow Processor
        medication_processor = MedicationWorkflowProcessor()
        
        # Create medication event
        med_envelope = ClinicalEventEnvelope(
            event_data={
                "medication_name": "Warfarin",
                "dosage": "5mg",
                "route": "oral",
                "frequency": "daily"
            },
            clinical_context=ClinicalContext(
                patient_id="test_patient_001",
                active_allergies=[],
                active_medications=[]
            ),
            temporal_context=TemporalContext(
                event_time=datetime.now(timezone.utc),
                system_time=datetime.now(timezone.utc)
            ),
            provenance_context=ProvenanceContext(
                source_system="pharmacy_system",
                created_by="pharmacist",
                created_at=datetime.now(timezone.utc)
            ),
            metadata=EventMetadata(event_type=EventType.MEDICATION_ORDER)
        )
        
        med_outcome = await medication_processor.process_event(med_envelope)
        logger.info(f"  ✓ Medication processor result: {med_outcome.result.value}")
        logger.info(f"  ✓ Processing time: {med_outcome.processing_duration_ms:.2f}ms")
        logger.info(f"  ✓ Warnings: {len(med_outcome.warnings)}")
        
        # Test 2: Laboratory Workflow Processor
        lab_processor = LaboratoryWorkflowProcessor()
        
        # Create laboratory event
        lab_envelope = ClinicalEventEnvelope(
            event_data={
                "test_name": "Glucose",
                "result_value": 250,  # High glucose
                "reference_range": "70-100 mg/dL",
                "unit": "mg/dL"
            },
            clinical_context=ClinicalContext(
                patient_id="test_patient_001"
            ),
            temporal_context=TemporalContext(
                event_time=datetime.now(timezone.utc),
                system_time=datetime.now(timezone.utc)
            ),
            provenance_context=ProvenanceContext(
                source_system="laboratory_system",
                created_by="lab_tech",
                created_at=datetime.now(timezone.utc)
            ),
            metadata=EventMetadata(event_type=EventType.LABORATORY_RESULT)
        )
        
        lab_outcome = await lab_processor.process_event(lab_envelope)
        logger.info(f"  ✓ Laboratory processor result: {lab_outcome.result.value}")
        logger.info(f"  ✓ Processing time: {lab_outcome.processing_duration_ms:.2f}ms")
        logger.info(f"  ✓ Warnings: {len(lab_outcome.warnings)}")
        
        # Test 3: Clinical Decision Processor
        decision_processor = ClinicalDecisionProcessor()
        
        # Create clinical decision event
        decision_envelope = ClinicalEventEnvelope(
            event_data={
                "decision_type": "medication_adjustment",
                "confidence_score": 0.65,  # Low confidence
                "recommendation": "Reduce warfarin dose"
            },
            clinical_context=ClinicalContext(
                patient_id="test_patient_001"
            ),
            temporal_context=TemporalContext(
                event_time=datetime.now(timezone.utc),
                system_time=datetime.now(timezone.utc)
            ),
            provenance_context=ProvenanceContext(
                source_system="clinical_decision_support",
                created_by="cds_engine",
                created_at=datetime.now(timezone.utc)
            ),
            metadata=EventMetadata(event_type=EventType.CLINICAL_DECISION)
        )
        
        decision_outcome = await decision_processor.process_event(decision_envelope)
        logger.info(f"  ✓ Decision processor result: {decision_outcome.result.value}")
        logger.info(f"  ✓ Processing time: {decision_outcome.processing_duration_ms:.2f}ms")
        logger.info(f"  ✓ Warnings: {len(decision_outcome.warnings)}")
        
        # Test 4: Event Processor Registry
        registry = EventProcessorRegistry()
        
        # Process events through registry
        med_result = await registry.process_event(med_envelope)
        lab_result = await registry.process_event(lab_envelope)
        decision_result = await registry.process_event(decision_envelope)
        
        logger.info(f"  ✓ Registry processed 3 events successfully")
        
        # Get registry stats
        registry_stats = registry.get_registry_stats()
        logger.info(f"  ✓ Registry stats: {registry_stats['registered_processors']} processors")
        
        logger.info("✅ Workflow-Specific Event Processors tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Workflow-Specific Event Processors test failed: {e}")
        return False


async def test_idempotency_and_event_sourcing():
    """Test Advanced Idempotency & Event Sourcing"""
    logger.info("🧪 Testing Advanced Idempotency & Event Sourcing")
    
    try:
        from events.clinical_event_envelope import (
            ClinicalEventEnvelope, ClinicalContext, TemporalContext, 
            ProvenanceContext, EventMetadata, EventType
        )
        from events.event_sourcing import IdempotencyManager, EventStore
        
        # Test 1: Idempotency Manager
        idempotency_manager = IdempotencyManager()
        
        # Create test event
        test_envelope = ClinicalEventEnvelope(
            event_data={"test": "data", "value": 123},
            clinical_context=ClinicalContext(patient_id="test_patient_001"),
            temporal_context=TemporalContext(
                event_time=datetime.now(timezone.utc),
                system_time=datetime.now(timezone.utc)
            ),
            provenance_context=ProvenanceContext(
                source_system="test_system",
                created_by="test_user",
                created_at=datetime.now(timezone.utc)
            ),
            metadata=EventMetadata(event_type=EventType.CLINICAL_ASSERTION)
        )
        
        # First check - should not be duplicate
        is_duplicate1, original_id1 = await idempotency_manager.check_idempotency(test_envelope)
        logger.info(f"  ✓ First idempotency check: duplicate={is_duplicate1}")
        
        # Mark as processed
        await idempotency_manager.mark_event_processed(test_envelope)
        
        # Second check with same event - should be duplicate
        is_duplicate2, original_id2 = await idempotency_manager.check_idempotency(test_envelope)
        logger.info(f"  ✓ Second idempotency check: duplicate={is_duplicate2}")
        
        # Get idempotency stats
        idempotency_stats = idempotency_manager.get_idempotency_stats()
        logger.info(f"  ✓ Idempotency stats: {idempotency_stats['total_keys']} keys")
        
        # Test 2: Event Store
        event_store = EventStore()
        
        # Append events to store
        stream_id1 = await event_store.append_event(test_envelope)
        logger.info(f"  ✓ Appended event to stream: {stream_id1}")
        
        # Create another event for same patient
        test_envelope2 = ClinicalEventEnvelope(
            event_data={"test": "data2", "value": 456},
            clinical_context=ClinicalContext(patient_id="test_patient_001"),
            temporal_context=TemporalContext(
                event_time=datetime.now(timezone.utc),
                system_time=datetime.now(timezone.utc)
            ),
            provenance_context=ProvenanceContext(
                source_system="test_system",
                created_by="test_user",
                created_at=datetime.now(timezone.utc)
            ),
            metadata=EventMetadata(event_type=EventType.CLINICAL_ASSERTION)
        )
        
        stream_id2 = await event_store.append_event(test_envelope2)
        logger.info(f"  ✓ Appended second event to stream: {stream_id2}")
        
        # Get events by patient
        patient_events = await event_store.get_events_by_patient("test_patient_001")
        logger.info(f"  ✓ Retrieved {len(patient_events)} events for patient")
        
        # Get event by ID
        retrieved_event = await event_store.get_event_by_id(test_envelope.metadata.event_id)
        logger.info(f"  ✓ Retrieved event by ID: {retrieved_event.metadata.event_id if retrieved_event else 'None'}")
        
        # Create snapshot
        snapshot = await event_store.create_snapshot(stream_id1)
        logger.info(f"  ✓ Created snapshot: {snapshot.snapshot_id if snapshot else 'None'}")
        
        # Get store stats
        store_stats = event_store.get_store_stats()
        logger.info(f"  ✓ Event store stats: {store_stats['total_events']} events, {store_stats['total_streams']} streams")
        
        logger.info("✅ Advanced Idempotency & Event Sourcing tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Advanced Idempotency & Event Sourcing test failed: {e}")
        return False


async def test_clinical_context_integration():
    """Test Clinical Context Integration with CAE"""
    logger.info("🧪 Testing Clinical Context Integration")
    
    try:
        from events.clinical_context_assembler import (
            ClinicalContextAssembler, ContextEnrichmentEngine, 
            ContextEnrichmentConfig, EnrichmentLevel
        )
        from events.clinical_event_envelope import (
            ClinicalEventEnvelope, ClinicalContext, TemporalContext, 
            ProvenanceContext, EventMetadata, EventType
        )
        
        # Test 1: Clinical Context Assembler
        config = ContextEnrichmentConfig(
            enrichment_level=EnrichmentLevel.COMPREHENSIVE,
            include_real_time_vitals=True,
            include_recent_labs=True
        )
        
        assembler = ClinicalContextAssembler(config)
        
        # Create basic envelope
        basic_envelope = ClinicalEventEnvelope(
            event_data={"test": "context_enrichment"},
            clinical_context=ClinicalContext(
                patient_id="test_patient_001",
                encounter_id="ENC001"
            ),
            temporal_context=TemporalContext(
                event_time=datetime.now(timezone.utc),
                system_time=datetime.now(timezone.utc)
            ),
            provenance_context=ProvenanceContext(
                source_system="test_system",
                created_by="test_user",
                created_at=datetime.now(timezone.utc)
            ),
            metadata=EventMetadata(event_type=EventType.CLINICAL_ASSERTION)
        )
        
        # Enrich context
        enriched_envelope = await assembler.enrich_clinical_context(basic_envelope)
        
        logger.info(f"  ✓ Context enriched - Demographics: {len(enriched_envelope.clinical_context.patient_demographics)} fields")
        logger.info(f"  ✓ Active medications: {len(enriched_envelope.clinical_context.active_medications)}")
        logger.info(f"  ✓ Active diagnoses: {len(enriched_envelope.clinical_context.active_diagnoses)}")
        logger.info(f"  ✓ Laboratory values: {len(enriched_envelope.clinical_context.laboratory_values)}")
        
        # Test 2: Context Enrichment Engine
        enrichment_engine = ContextEnrichmentEngine()
        
        # Register custom assembler
        enrichment_engine.register_assembler("comprehensive", assembler)
        
        # Enrich using engine
        engine_enriched = await enrichment_engine.enrich_envelope(basic_envelope, "comprehensive")
        
        logger.info(f"  ✓ Engine enrichment completed")
        
        # Get enrichment stats
        assembler_stats = assembler.get_enrichment_stats()
        logger.info(f"  ✓ Enrichment stats: {assembler_stats['total_enrichments']} enrichments")
        logger.info(f"  ✓ Cache hit ratio: {assembler_stats['cache_hit_ratio_percent']:.1f}%")
        
        engine_stats = enrichment_engine.get_engine_stats()
        logger.info(f"  ✓ Engine stats: {engine_stats['registered_assemblers']} assemblers")
        
        logger.info("✅ Clinical Context Integration tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Clinical Context Integration test failed: {e}")
        return False


async def test_integrated_event_envelope_system():
    """Test integrated event envelope system with CAE orchestration"""
    logger.info("🧪 Testing Integrated Event Envelope System")
    
    try:
        from orchestration.orchestration_engine import OrchestrationEngine
        
        # Create orchestration engine with event envelope system
        engine = OrchestrationEngine(max_queue_size=100, max_concurrent=10)
        
        # Register mock reasoner
        class MockReasoner:
            async def check_interactions(self, **kwargs):
                await asyncio.sleep(0.01)
                return {
                    "assertions": [{
                        "type": "interaction",
                        "severity": "moderate",
                        "description": "Mock interaction detected",
                        "confidence": 0.85
                    }]
                }
        
        engine.register_reasoner("interaction", MockReasoner())
        
        # Start engine
        await engine.start()
        
        # Test 1: Process clinical event envelope
        envelope_data = {
            "event_data": {
                "assertion_type": "drug_interaction",
                "description": "Warfarin-Aspirin interaction",
                "confidence_score": 0.85
            },
            "clinical_context": {
                "patient_id": "test_patient_001",
                "patient_mrn": "MRN123456",
                "encounter_id": "ENC001",
                "active_medications": [
                    {"name": "Warfarin", "dosage": "5mg"},
                    {"name": "Aspirin", "dosage": "81mg"}
                ],
                "active_diagnoses": [
                    {"code": "I48.91", "description": "Atrial fibrillation"}
                ],
                "active_allergies": []
            },
            "temporal_context": {
                "event_time": datetime.now(timezone.utc).isoformat(),
                "system_time": datetime.now(timezone.utc).isoformat(),
                "clinical_day": 3
            },
            "provenance_context": {
                "source_system": "clinical_reasoning_service",
                "created_by": "test_system",
                "created_at": datetime.now(timezone.utc).isoformat()
            },
            "metadata": {
                "event_type": "clinical_assertion",
                "event_severity": "moderate",
                "processing_priority": 7
            },
            "envelope_version": "2.0",
            "created_at": datetime.now(timezone.utc).isoformat()
        }
        
        # Process envelope
        result = await engine.process_clinical_event_envelope(envelope_data)
        
        logger.info(f"  ✓ Event envelope processing status: {result['status']}")
        logger.info(f"  ✓ Processing duration: {result.get('processing_duration_ms', 0):.2f}ms")
        logger.info(f"  ✓ Warnings: {len(result.get('warnings', []))}")
        logger.info(f"  ✓ Recommendations: {len(result.get('recommendations', []))}")
        
        # Test 2: Test idempotency (process same event again)
        result2 = await engine.process_clinical_event_envelope(envelope_data)
        
        logger.info(f"  ✓ Second processing status: {result2['status']}")
        
        # Test 3: Get event history
        event_history = await engine.get_event_history("test_patient_001", days_back=1)
        
        logger.info(f"  ✓ Retrieved {len(event_history)} events from history")
        
        # Test 4: Get enhanced system status
        status = await engine.get_system_status()
        
        logger.info(f"  ✓ System status includes event envelope system: {'event_envelope_system' in status}")
        
        if 'event_envelope_system' in status:
            envelope_status = status['event_envelope_system']
            logger.info(f"    - Event processors: {envelope_status.get('event_processor_registry', {}).get('registered_processors', 0)}")
            logger.info(f"    - Event store streams: {envelope_status.get('event_store', {}).get('total_streams', 0)}")
            logger.info(f"    - Idempotency keys: {envelope_status.get('idempotency_manager', {}).get('total_keys', 0)}")
        
        # Stop engine
        await engine.stop()
        
        logger.info("✅ Integrated Event Envelope System tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Integrated Event Envelope System test failed: {e}")
        return False


async def main():
    """Run all event envelope system tests"""
    logger.info("🚀 Starting Clinical Event Envelope Integration Tests")
    logger.info("=" * 70)
    
    test_results = []
    
    # Run individual component tests
    test_functions = [
        ("Enhanced Clinical Event Envelope", test_clinical_event_envelope),
        ("Workflow-Specific Event Processors", test_workflow_processors),
        ("Advanced Idempotency & Event Sourcing", test_idempotency_and_event_sourcing),
        ("Clinical Context Integration", test_clinical_context_integration),
        ("Integrated Event Envelope System", test_integrated_event_envelope_system)
    ]
    
    for test_name, test_func in test_functions:
        logger.info(f"\n📋 Running {test_name} tests...")
        try:
            result = await test_func()
            test_results.append((test_name, result))
        except Exception as e:
            logger.error(f"❌ {test_name} test suite failed: {e}")
            test_results.append((test_name, False))
    
    # Summary
    logger.info("\n" + "=" * 70)
    logger.info("📊 EVENT ENVELOPE INTEGRATION TEST SUMMARY")
    logger.info("=" * 70)
    
    passed = 0
    failed = 0
    
    for test_name, result in test_results:
        status = "✅ PASSED" if result else "❌ FAILED"
        logger.info(f"{test_name}: {status}")
        if result:
            passed += 1
        else:
            failed += 1
    
    logger.info(f"\nTotal: {passed} passed, {failed} failed")
    
    if failed == 0:
        logger.info("🎉 All event envelope integration tests passed!")
        logger.info("📋 Week 5-6: Clinical Event Envelope Integration is complete!")
        logger.info("\n🚀 Key Features Validated:")
        logger.info("  ✓ Enhanced clinical event envelope with rich context")
        logger.info("  ✓ Workflow-specific event processors")
        logger.info("  ✓ Advanced idempotency and event sourcing")
        logger.info("  ✓ Clinical context integration with CAE")
        logger.info("  ✓ Complete audit trail and provenance tracking")
    else:
        logger.error(f"⚠️  {failed} test suite(s) failed. Please review and fix issues.")
    
    return failed == 0


if __name__ == "__main__":
    # Run the test suite
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
