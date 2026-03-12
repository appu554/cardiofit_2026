"""
V3 Clinical Guideline Curation Pipeline - Guideline Atomiser

This package implements the 7-Layer architecture for extracting clinical FACTS
from guidelines (KDIGO, FDA SPL, ADA, etc.) into KB-specific formats.

Key Principle: Extract FACTS, not RULES.
- CQL defines WHEN something should happen (logic - already exists in vaidshala)
- KB values define WHAT VALUES it happens with (facts - what we extract)

Layers:
- L1: PDF Parsing (Marker v1.10)
- L2: Clinical NER (GLiNER-BioMed)
- L3: Structured Extraction (Claude + KB-specific Pydantic schemas)
- L4: Terminology Validation (Snow Owl for RxNorm, LOINC, SNOMED-CT)
- L5: CQL Validation (Registry-based compatibility checking)
- L6: Provenance (Git + FHIR Provenance resources)
- L7: MCP Orchestration (Curator workflow)

Usage:
    from guideline_atomiser import KBFactExtractor, create_extractor_from_env
    from guideline_atomiser import MarkerExtractor, extract_pdf_with_provenance

    # L1: Extract PDF with provenance
    pdf_result = extract_pdf_with_provenance("kdigo_2022.pdf")

    # L3: Extract KB-specific facts
    extractor = create_extractor_from_env()
    result = extractor.extract_facts(
        markdown_text=pdf_result.markdown,
        gliner_entities=[...],
        target_kb="dosing",
        guideline_context={"authority": "KDIGO", ...}
    )
"""

from .fact_extractor import KBFactExtractor, create_extractor_from_env
from .marker_extractor import (
    MarkerExtractor,
    extract_pdf_with_provenance,
    ExtractionResult,
    TextBlock,
    TableBlock,
    ExtractionProvenance,
    BoundingBox,
)
from .snow_owl_client import (
    SnowOwlClient,
    create_client_from_env as create_terminology_client,
    CodeValidationResult,
    SearchResult,
    TerminologyEnrichment,
)

__all__ = [
    # L3: Fact Extraction
    "KBFactExtractor",
    "create_extractor_from_env",
    # L1: PDF Parsing
    "MarkerExtractor",
    "extract_pdf_with_provenance",
    "ExtractionResult",
    "TextBlock",
    "TableBlock",
    "ExtractionProvenance",
    "BoundingBox",
    # L4: Terminology Validation
    "SnowOwlClient",
    "create_terminology_client",
    "CodeValidationResult",
    "SearchResult",
    "TerminologyEnrichment",
]

__version__ = "3.0.0"
