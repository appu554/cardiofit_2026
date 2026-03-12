"""
KB-1 Drug Dosing Schema - Pydantic models matching Go structs.

Extracts renal and hepatic dosing adjustment FACTS from clinical guidelines.
These facts are consumed by CQL rules in vaidshala/tier-4-guidelines/.

Go struct alignment:
- kb-1-drug-rules/internal/models/models.go -> RenalAdjustment
- Field aliases map snake_case (Python) to camelCase (Go JSON)
"""

from datetime import date
from typing import Optional, Literal
from pydantic import BaseModel, Field, ConfigDict


class RenalAdjustment(BaseModel):
    """
    Renal dosing adjustment matching KB-1 Go struct.

    Represents a single eGFR threshold range and the corresponding
    dosing modification (adjustment factor, max dose, or contraindication).
    """

    model_config = ConfigDict(populate_by_name=True)

    egfr_min: float = Field(
        ...,
        alias="egfrMin",
        description="Lower bound of eGFR range (mL/min/1.73m²)",
        ge=0,
    )
    egfr_max: float = Field(
        ...,
        alias="egfrMax",
        description="Upper bound of eGFR range (mL/min/1.73m²)",
        ge=0,
    )
    adjustment_factor: Optional[float] = Field(
        None,
        alias="adjustmentFactor",
        description="Dose adjustment factor (e.g., 0.5 for 50% reduction)",
        ge=0,
        le=1,
    )
    max_dose: Optional[float] = Field(
        None,
        alias="maxDose",
        description="Maximum dose at this eGFR level",
        ge=0,
    )
    max_dose_unit: Optional[str] = Field(
        None,
        alias="maxDoseUnit",
        description="Unit for max_dose (e.g., 'mg', 'mg/day')",
    )
    frequency: Optional[str] = Field(
        None,
        description="Dosing frequency adjustment (e.g., 'every 48h')",
    )
    recommendation: str = Field(
        ...,
        description="Human-readable dosing recommendation",
    )
    contraindicated: bool = Field(
        False,
        description="Whether the drug is contraindicated at this eGFR level",
    )
    action_type: Literal[
        "CONTRAINDICATED", "REDUCE_DOSE", "REDUCE_FREQUENCY", "MONITOR", "NO_CHANGE"
    ] = Field(
        "NO_CHANGE",
        alias="actionType",
        description="Type of dosing action required",
    )


class HepaticAdjustment(BaseModel):
    """
    Hepatic dosing adjustment for drugs with liver metabolism concerns.

    Similar structure to RenalAdjustment but uses Child-Pugh classification.
    """

    model_config = ConfigDict(populate_by_name=True)

    child_pugh_class: Literal["A", "B", "C"] = Field(
        ...,
        alias="childPughClass",
        description="Child-Pugh hepatic impairment class",
    )
    adjustment_factor: Optional[float] = Field(
        None,
        alias="adjustmentFactor",
        description="Dose adjustment factor",
        ge=0,
        le=1,
    )
    max_dose: Optional[float] = Field(
        None,
        alias="maxDose",
        description="Maximum dose for this hepatic class",
    )
    max_dose_unit: Optional[str] = Field(
        None,
        alias="maxDoseUnit",
        description="Unit for max_dose",
    )
    recommendation: str = Field(
        ...,
        description="Human-readable dosing recommendation",
    )
    contraindicated: bool = Field(
        False,
        description="Whether drug is contraindicated",
    )


class ClinicalGovernance(BaseModel):
    """
    Provenance and governance metadata for extracted facts.

    Tracks the authoritative source, evidence level, and extraction details.
    """

    model_config = ConfigDict(populate_by_name=True)

    source_authority: str = Field(
        ...,
        alias="sourceAuthority",
        description="Authoritative body (e.g., 'KDIGO', 'FDA', 'ADA')",
    )
    source_document: str = Field(
        ...,
        alias="sourceDocument",
        description="Full document title",
    )
    source_section: Optional[str] = Field(
        None,
        alias="sourceSection",
        description="Specific section/recommendation ID (e.g., '4.1.1')",
    )
    source_page: Optional[int] = Field(
        None,
        alias="sourcePage",
        description="PDF page number where fact was extracted",
    )
    evidence_level: str = Field(
        "",
        alias="evidenceLevel",
        description="Evidence grade (e.g., '1A', '2B', 'Expert Opinion')",
    )
    effective_date: str = Field(
        ...,
        alias="effectiveDate",
        description="When the guideline became effective (YYYY-MM-DD)",
    )
    guideline_doi: Optional[str] = Field(
        None,
        alias="guidelineDoi",
        description="DOI of the source guideline",
    )


class DrugRenalFacts(BaseModel):
    """
    Complete renal dosing facts for a single drug.

    Contains all eGFR thresholds and corresponding adjustments extracted
    from the guideline for this drug.
    """

    model_config = ConfigDict(populate_by_name=True)

    rxnorm_code: str = Field(
        ...,
        alias="rxnormCode",
        description="RxNorm CUI for the drug",
    )
    drug_name: str = Field(
        ...,
        alias="drugName",
        description="Generic drug name",
    )
    drug_class: Optional[str] = Field(
        None,
        alias="drugClass",
        description="Therapeutic class (e.g., 'SGLT2_inhibitors', 'biguanides')",
    )
    renal_adjustments: list[RenalAdjustment] = Field(
        ...,
        alias="renalAdjustments",
        description="List of eGFR-based dosing adjustments",
    )
    hepatic_adjustments: Optional[list[HepaticAdjustment]] = Field(
        None,
        alias="hepaticAdjustments",
        description="List of hepatic impairment adjustments if applicable",
    )

    # Provenance fields
    source_page: int = Field(
        ...,
        alias="sourcePage",
        description="PDF page number of primary extraction",
    )
    source_snippet: str = Field(
        ...,
        alias="sourceSnippet",
        description="Verbatim text snippet from guideline (for audit)",
        max_length=1000,
    )
    guideline_version: str = Field(
        ...,
        alias="guidelineVersion",
        description="Guideline version identifier",
    )
    guideline_doi: str = Field(
        "",
        alias="guidelineDoi",
        description="DOI of source guideline",
    )
    recommendation_id: str = Field(
        "",
        alias="recommendationId",
        description="Guideline recommendation identifier (e.g., '4.1.1')",
    )

    governance: Optional[ClinicalGovernance] = Field(
        None,
        description="Full governance metadata",
    )


class KB1ExtractionResult(BaseModel):
    """
    L3 output for KB-1: Complete extraction result for drug dosing facts.

    This is the top-level schema returned by the fact extractor when
    target_kb="dosing".
    """

    model_config = ConfigDict(populate_by_name=True)

    drugs: list[DrugRenalFacts] = Field(
        ...,
        description="List of extracted drug dosing facts",
    )
    extraction_date: str = Field(
        ...,
        alias="extractionDate",
        description="ISO date when extraction was performed",
    )
    extractor_version: str = Field(
        "v3.0.0-facts",
        alias="extractorVersion",
        description="Version of the fact extraction pipeline",
    )
    source_guideline: str = Field(
        ...,
        alias="sourceGuideline",
        description="Identifier for the source guideline",
    )
    total_drugs_extracted: int = Field(
        0,
        alias="totalDrugsExtracted",
        description="Count of drugs with facts extracted",
    )

    def model_post_init(self, __context) -> None:
        """Auto-calculate total drugs after initialization."""
        object.__setattr__(self, "total_drugs_extracted", len(self.drugs))
