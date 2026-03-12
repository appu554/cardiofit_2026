"""
KB-Specific Pydantic Schemas for V4 Guideline Curation Pipeline.

These schemas define the extraction output format for each Knowledge Base:
- KB-1: Drug dosing facts (RenalAdjustment, HepaticAdjustment)
- KB-4: Safety facts (Contraindication, Warning)
- KB-16: Lab monitoring facts (LabRequirement, MonitoringEntry)
- KB-20: Contextual modifiers & ADR profiles (AdverseReactionProfile, ContextualModifierFact)

All schemas use ConfigDict(populate_by_name=True) to support both:
- snake_case (Python convention)
- camelCase (Go JSON convention via aliases)

When serializing for Go consumption, use: model.model_dump(by_alias=True)
"""

from .kb1_dosing import (
    RenalAdjustment,
    HepaticAdjustment,
    DrugRenalFacts,
    KB1ExtractionResult,
)
from .kb4_safety import (
    ClinicalGovernance,
    ContraindicationFact,
    KB4ExtractionResult,
)
from .kb16_labs import (
    LabMonitoringEntry,
    LabRequirementFact,
    KB16ExtractionResult,
)
from .kb20_contextual import (
    AdverseReactionProfile,
    ContextualModifierFact,
    KB20ExtractionResult,
)

__all__ = [
    # KB-1 Dosing
    "RenalAdjustment",
    "HepaticAdjustment",
    "DrugRenalFacts",
    "KB1ExtractionResult",
    # KB-4 Safety
    "ClinicalGovernance",
    "ContraindicationFact",
    "KB4ExtractionResult",
    # KB-16 Labs
    "LabMonitoringEntry",
    "LabRequirementFact",
    "KB16ExtractionResult",
    # KB-20 Contextual Modifiers
    "AdverseReactionProfile",
    "ContextualModifierFact",
    "KB20ExtractionResult",
]
