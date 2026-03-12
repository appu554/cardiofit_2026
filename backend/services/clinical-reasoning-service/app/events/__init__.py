"""
Clinical Event Envelope System for Clinical Assertion Engine

This module provides comprehensive clinical event handling with:
- Enhanced clinical event envelope with rich context
- Temporal sophistication and clinical time tracking
- Complete provenance and audit trail
- Workflow-specific event processors
- Advanced idempotency and event sourcing
"""

from .clinical_event_envelope import (
    ClinicalEventEnvelope,
    ClinicalContext,
    TemporalContext,
    ProvenanceContext,
    EventMetadata,
    EventType,
    EventSeverity
)

from .event_processors import (
    MedicationWorkflowProcessor,
    LaboratoryWorkflowProcessor,
    ClinicalDecisionProcessor,
    EventProcessorRegistry
)

from .event_sourcing import (
    EventStore,
    EventStream,
    EventSnapshot,
    IdempotencyManager
)

from .clinical_context_assembler import (
    ClinicalContextAssembler,
    ContextEnrichmentEngine
)

__all__ = [
    # Core Event Envelope
    'ClinicalEventEnvelope',
    'ClinicalContext',
    'TemporalContext', 
    'ProvenanceContext',
    'EventMetadata',
    'EventType',
    'EventSeverity',
    
    # Event Processors
    'MedicationWorkflowProcessor',
    'LaboratoryWorkflowProcessor',
    'ClinicalDecisionProcessor',
    'EventProcessorRegistry',
    
    # Event Sourcing
    'EventStore',
    'EventStream',
    'EventSnapshot',
    'IdempotencyManager',
    
    # Context Assembly
    'ClinicalContextAssembler',
    'ContextEnrichmentEngine'
]
