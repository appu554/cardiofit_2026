"""
GLiNER Clinical NER Module - L2 of the V3 Guideline Curation Pipeline.

This module provides Named Entity Recognition for clinical text using
GLiNER, a zero-shot NER model optimized for biomedical entities.

Key Principle: Pre-tag clinical entities before L3 Claude extraction.
GLiNER identifies WHERE entities are; Claude extracts WHAT values they have.

Components:
- ClinicalNERExtractor: Main NER extraction class
- ClinicalEntityTypes: Entity type definitions with KB destinations
- Entity/NERResult: Data classes for extraction results

Usage:
    from gliner import ClinicalNERExtractor, get_labels_for_kb

    extractor = ClinicalNERExtractor()

    # Extract all clinical entities
    result = extractor.extract_entities(text)

    # Extract for specific KB
    dosing_entities = extractor.extract_for_kb(text, "dosing")
"""

from .extractor import (
    ClinicalNERExtractor,
    OpenMedNERExtractor,  # Backward compatibility alias
    Entity,
    NERResult,
    create_extractor_from_env,
)
from .entity_types import (
    ClinicalEntityTypes,
    EntityType,
    get_labels_for_kb,
    get_all_clinical_labels,
    get_entity_color_map,
    KDIGO_ENTITY_LABELS,
    SPL_ENTITY_LABELS,
    ADA_ENTITY_LABELS,
)

__all__ = [
    # Extractor
    "ClinicalNERExtractor",
    "OpenMedNERExtractor",  # Backward compatibility alias
    "Entity",
    "NERResult",
    "create_extractor_from_env",
    # Entity Types
    "ClinicalEntityTypes",
    "EntityType",
    "get_labels_for_kb",
    "get_all_clinical_labels",
    "get_entity_color_map",
    # Label Sets
    "KDIGO_ENTITY_LABELS",
    "SPL_ENTITY_LABELS",
    "ADA_ENTITY_LABELS",
]

__version__ = "2.1.0"  # GLiNER with descriptive labels
