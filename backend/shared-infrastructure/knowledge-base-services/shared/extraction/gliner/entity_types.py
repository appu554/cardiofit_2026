"""
L2: Clinical Entity Type Definitions for GLiNER.

This module defines the clinical entity types recognized during the NER phase
of the V3 Clinical Guideline Curation Pipeline. Uses GLiNER (zero-shot NER)
with descriptive labels for improved entity recognition on clinical text.

Key Principle: Pre-tagging clinical entities makes L3 Claude extraction
more accurate by providing structural context. OpenMed NER handles the "where"
so Claude can focus on the "what" (values, thresholds, relationships).

Entity Types are grouped by KB destination:
- KB-1 (Dosing): Drug ingredients, drug classes, drug products, doses, frequencies
- KB-4 (Safety): Conditions, contraindication markers, severity levels
- KB-16 (Monitoring): Lab tests, thresholds, monitoring frequencies

V2.0 Changes:
- Split drug_name → drug_ingredient, drug_class, drug_product
- Aligned with OpenMed NER entity type mapping
- Improved separation of generic names vs brand names vs drug classes

Usage:
    from entity_types import ClinicalEntityTypes, get_labels_for_kb

    # Get all entity labels for OpenMed NER
    labels = ClinicalEntityTypes.get_all_labels()

    # Get KB-specific labels
    dosing_labels = get_labels_for_kb("dosing")
"""

from dataclasses import dataclass
from typing import Literal, Optional


@dataclass(frozen=True)
class EntityType:
    """A clinical entity type with metadata."""
    name: str  # Display name
    label: str  # GLiNER label (lowercase, underscore-separated)
    description: str
    kb_destination: Literal["KB-1", "KB-4", "KB-16", "SHARED"]
    examples: tuple[str, ...]
    color: str = "#808080"  # For visualization

    def __hash__(self):
        return hash(self.label)


class ClinicalEntityTypes:
    """
    Clinical entity type definitions for guideline extraction.

    These entity types are optimized for:
    1. KDIGO Guidelines (nephrology focus)
    2. FDA SPL Labels (drug safety)
    3. ADA Standards (diabetes care)
    4. ACC/AHA Guidelines (cardiovascular)
    """

    # ═══════════════════════════════════════════════════════════════════════
    # KB-1: Drug Dosing Entity Types
    # ═══════════════════════════════════════════════════════════════════════

    DRUG_INGREDIENT = EntityType(
        name="Drug Ingredient",
        label="drug_ingredient",
        description="Active pharmaceutical ingredient (generic name)",
        kb_destination="KB-1",
        examples=("metformin", "dapagliflozin", "lisinopril", "finerenone"),
        color="#4CAF50",
    )

    DRUG_CLASS = EntityType(
        name="Drug Class",
        label="drug_class",
        description="Pharmacological class of medications",
        kb_destination="KB-1",
        examples=("SGLT2 inhibitors", "GLP-1 RA", "ACE inhibitors", "RASi", "biguanides"),
        color="#8BC34A",
    )

    DRUG_PRODUCT = EntityType(
        name="Drug Product",
        label="drug_product",
        description="Brand name or product formulation",
        kb_destination="KB-1",
        examples=("Glucophage", "Farxiga", "Jardiance", "Kerendia"),
        color="#66BB6A",
    )

    # Legacy alias for backward compatibility
    DRUG_NAME = DRUG_INGREDIENT  # Deprecated: use DRUG_INGREDIENT instead

    DOSE_VALUE = EntityType(
        name="Dose Value",
        label="dose_value",
        description="Numeric dose amount",
        kb_destination="KB-1",
        examples=("500", "10", "2.5", "1000"),
        color="#2196F3",
    )

    DOSE_UNIT = EntityType(
        name="Dose Unit",
        label="dose_unit",
        description="Unit of measurement for doses",
        kb_destination="KB-1",
        examples=("mg", "mcg", "mL", "units", "mg/kg"),
        color="#03A9F4",
    )

    DOSE_FREQUENCY = EntityType(
        name="Dose Frequency",
        label="dose_frequency",
        description="How often a dose is taken",
        kb_destination="KB-1",
        examples=("once daily", "twice daily", "BID", "every 12 hours", "weekly"),
        color="#00BCD4",
    )

    DOSE_ROUTE = EntityType(
        name="Dose Route",
        label="dose_route",
        description="Route of administration",
        kb_destination="KB-1",
        examples=("oral", "subcutaneous", "IV", "intramuscular", "topical"),
        color="#009688",
    )

    DOSE_ADJUSTMENT = EntityType(
        name="Dose Adjustment",
        label="dose_adjustment",
        description="Instructions for dose modification",
        kb_destination="KB-1",
        examples=("reduce by 50%", "half dose", "increase to", "discontinue"),
        color="#FF9800",
    )

    # ═══════════════════════════════════════════════════════════════════════
    # KB-4: Patient Safety Entity Types
    # ═══════════════════════════════════════════════════════════════════════

    CONDITION = EntityType(
        name="Clinical Condition",
        label="condition",
        description="Disease, disorder, or clinical state",
        kb_destination="KB-4",
        examples=("CKD", "heart failure", "lactic acidosis", "hyperkalemia"),
        color="#F44336",
    )

    CONTRAINDICATION_MARKER = EntityType(
        name="Contraindication Marker",
        label="contraindication_marker",
        description="Words indicating contraindication",
        kb_destination="KB-4",
        examples=("contraindicated", "avoid", "do not use", "should not be used"),
        color="#E91E63",
    )

    SEVERITY_INDICATOR = EntityType(
        name="Severity Indicator",
        label="severity",
        description="Severity or urgency level",
        kb_destination="KB-4",
        examples=("severe", "moderate", "mild", "life-threatening", "critical"),
        color="#9C27B0",
    )

    ADVERSE_EVENT = EntityType(
        name="Adverse Event",
        label="adverse_event",
        description="Side effects or adverse reactions",
        kb_destination="KB-4",
        examples=("hypoglycemia", "nausea", "lactic acidosis", "volume depletion"),
        color="#673AB7",
    )

    POPULATION = EntityType(
        name="Patient Population",
        label="population",
        description="Specific patient groups or demographics",
        kb_destination="KB-4",
        examples=("elderly", "pregnant", "pediatric", "CKD stage 4-5", "dialysis"),
        color="#3F51B5",
    )

    CAUTION_MARKER = EntityType(
        name="Caution Marker",
        label="caution_marker",
        description="Words indicating caution or warning",
        kb_destination="KB-4",
        examples=("use with caution", "monitor closely", "consider", "may"),
        color="#FF5722",
    )

    # ═══════════════════════════════════════════════════════════════════════
    # KB-16: Lab Monitoring Entity Types
    # ═══════════════════════════════════════════════════════════════════════

    LAB_TEST = EntityType(
        name="Lab Test",
        label="lab_test",
        description="Laboratory test or measurement",
        kb_destination="KB-16",
        examples=("eGFR", "serum creatinine", "potassium", "HbA1c", "UACR"),
        color="#795548",
    )

    LAB_VALUE = EntityType(
        name="Lab Value",
        label="lab_value",
        description="Numeric lab result or threshold",
        kb_destination="KB-16",
        examples=("30", "< 45", "> 5.5", "30-44", "≥ 60"),
        color="#9E9E9E",
    )

    LAB_UNIT = EntityType(
        name="Lab Unit",
        label="lab_unit",
        description="Unit for lab measurements",
        kb_destination="KB-16",
        examples=("mL/min/1.73m2", "mg/dL", "mEq/L", "%", "mg/g"),
        color="#607D8B",
    )

    MONITORING_FREQUENCY = EntityType(
        name="Monitoring Frequency",
        label="monitoring_frequency",
        description="How often to monitor",
        kb_destination="KB-16",
        examples=("every 3 months", "annually", "at week 4", "Q3-6 months"),
        color="#FFC107",
    )

    MONITORING_ACTION = EntityType(
        name="Monitoring Action",
        label="monitoring_action",
        description="Action based on monitoring result",
        kb_destination="KB-16",
        examples=("discontinue", "hold", "reduce dose", "recheck in 1 week"),
        color="#FF9800",
    )

    BASELINE_MARKER = EntityType(
        name="Baseline Marker",
        label="baseline_marker",
        description="Indicators of baseline requirements",
        kb_destination="KB-16",
        examples=("before initiation", "at baseline", "prior to starting"),
        color="#CDDC39",
    )

    # ═══════════════════════════════════════════════════════════════════════
    # SHARED: Cross-KB Entity Types
    # ═══════════════════════════════════════════════════════════════════════

    EGFR_THRESHOLD = EntityType(
        name="eGFR Threshold",
        label="egfr_threshold",
        description="Kidney function threshold (used by all KBs)",
        kb_destination="SHARED",
        examples=("eGFR < 30", "eGFR 30-45", "eGFR ≥ 60", "CrCl < 30"),
        color="#FFEB3B",
    )

    RECOMMENDATION_LEVEL = EntityType(
        name="Recommendation Level",
        label="recommendation_level",
        description="Strength of recommendation",
        kb_destination="SHARED",
        examples=("1A", "1B", "2C", "Grade A", "strong recommendation"),
        color="#E91E63",
    )

    GUIDELINE_REFERENCE = EntityType(
        name="Guideline Reference",
        label="guideline_reference",
        description="Reference to guideline section",
        kb_destination="SHARED",
        examples=("Recommendation 4.1.1", "Section 3.2", "Table 5"),
        color="#9C27B0",
    )

    TEMPORAL_MARKER = EntityType(
        name="Temporal Marker",
        label="temporal_marker",
        description="Time-related indicators",
        kb_destination="SHARED",
        examples=("after 4 weeks", "within 3 months", "at initiation", "during"),
        color="#00BCD4",
    )

    @classmethod
    def get_all_types(cls) -> list[EntityType]:
        """Get all entity types."""
        return [
            value for key, value in vars(cls).items()
            if isinstance(value, EntityType)
        ]

    @classmethod
    def get_all_labels(cls) -> list[str]:
        """Get all GLiNER labels."""
        return [et.label for et in cls.get_all_types()]

    @classmethod
    def get_types_by_kb(cls, kb: Literal["KB-1", "KB-4", "KB-16", "SHARED"]) -> list[EntityType]:
        """Get entity types for a specific KB."""
        return [et for et in cls.get_all_types() if et.kb_destination == kb]

    @classmethod
    def get_labels_by_kb(cls, kb: Literal["KB-1", "KB-4", "KB-16", "SHARED"]) -> list[str]:
        """Get GLiNER labels for a specific KB."""
        return [et.label for et in cls.get_types_by_kb(kb)]

    @classmethod
    def get_type_by_label(cls, label: str) -> Optional[EntityType]:
        """Get entity type by its label."""
        for et in cls.get_all_types():
            if et.label == label:
                return et
        return None


# Convenience functions for common access patterns

def get_labels_for_kb(kb_name: Literal["dosing", "safety", "monitoring"]) -> list[str]:
    """
    Get GLiNER labels for a target KB extraction.

    Args:
        kb_name: "dosing" (KB-1), "safety" (KB-4), or "monitoring" (KB-16)

    Returns:
        List of entity labels relevant for that KB
    """
    kb_map = {
        "dosing": "KB-1",
        "safety": "KB-4",
        "monitoring": "KB-16",
    }

    kb_id = kb_map.get(kb_name)
    if not kb_id:
        raise ValueError(f"Unknown KB name: {kb_name}")

    # Get KB-specific labels plus shared labels
    labels = ClinicalEntityTypes.get_labels_by_kb(kb_id)
    labels.extend(ClinicalEntityTypes.get_labels_by_kb("SHARED"))

    return labels


def get_all_clinical_labels() -> list[str]:
    """Get all clinical entity labels for comprehensive NER."""
    return ClinicalEntityTypes.get_all_labels()


def get_entity_color_map() -> dict[str, str]:
    """Get a mapping of entity labels to colors for visualization."""
    return {et.label: et.color for et in ClinicalEntityTypes.get_all_types()}


# Entity type groupings for specific extraction tasks

KDIGO_ENTITY_LABELS = [
    "drug_ingredient",
    "drug_class",
    "drug_product",
    "dose_value",
    "dose_unit",
    "dose_adjustment",
    "condition",
    "contraindication_marker",
    "severity",
    "lab_test",
    "lab_value",
    "lab_unit",
    "monitoring_frequency",
    "egfr_threshold",
    "recommendation_level",
    "guideline_reference",
]

SPL_ENTITY_LABELS = [
    "drug_ingredient",
    "drug_product",
    "dose_value",
    "dose_unit",
    "dose_frequency",
    "dose_route",
    "condition",
    "contraindication_marker",
    "adverse_event",
    "severity",
    "population",
    "caution_marker",
    "lab_test",
]

ADA_ENTITY_LABELS = [
    "drug_ingredient",
    "drug_class",
    "drug_product",
    "dose_adjustment",
    "condition",
    "lab_test",
    "lab_value",
    "monitoring_frequency",
    "egfr_threshold",
    "recommendation_level",
    "temporal_marker",
]

# Legacy aliases for backward compatibility
DRUG_NAME_LABELS = ["drug_ingredient", "drug_product"]  # Replacement for old "drug_name"
