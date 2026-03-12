"""
KB-4 Patient Safety Schema - Pydantic models matching Go structs.

Extracts contraindication and safety FACTS from clinical guidelines.
These facts are consumed by CQL rules in vaidshala/tier-4-guidelines/.

Go struct alignment:
- kb-4-patient-safety/pkg/safety/types.go -> Contraindication
- Field aliases map snake_case (Python) to camelCase (Go JSON)
"""

from typing import Optional, Literal
from pydantic import BaseModel, Field, ConfigDict


class ClinicalGovernance(BaseModel):
    """
    Provenance and governance metadata for extracted safety facts.

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
        description="Specific section/recommendation ID",
    )
    evidence_level: str = Field(
        "",
        alias="evidenceLevel",
        description="Evidence grade (e.g., '1A', '2B')",
    )
    effective_date: str = Field(
        ...,
        alias="effectiveDate",
        description="When guideline became effective (YYYY-MM-DD)",
    )
    guideline_doi: Optional[str] = Field(
        None,
        alias="guidelineDoi",
        description="DOI of source guideline",
    )


class ContraindicationFact(BaseModel):
    """
    Contraindication fact matching KB-4 Go struct.

    Represents a condition (disease, lab value, patient characteristic)
    under which a drug should not be used or used with caution.
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
        description="Therapeutic class for class-wide contraindications",
    )

    # Condition identification
    condition_codes: list[str] = Field(
        ...,
        alias="conditionCodes",
        description="ICD-10-CM codes for the contraindicated condition",
    )
    condition_descriptions: list[str] = Field(
        ...,
        alias="conditionDescriptions",
        description="Human-readable condition descriptions",
    )
    snomed_codes: list[str] = Field(
        default_factory=list,
        alias="snomedCodes",
        description="SNOMED-CT codes if available",
    )

    # Lab-based contraindication (optional)
    lab_parameter: Optional[str] = Field(
        None,
        alias="labParameter",
        description="Lab parameter triggering contraindication (e.g., 'eGFR')",
    )
    lab_loinc: Optional[str] = Field(
        None,
        alias="labLoinc",
        description="LOINC code for the lab parameter",
    )
    lab_threshold: Optional[float] = Field(
        None,
        alias="labThreshold",
        description="Threshold value triggering contraindication",
    )
    lab_operator: Optional[Literal["<", "<=", ">", ">=", "=="]] = Field(
        None,
        alias="labOperator",
        description="Comparison operator for lab threshold",
    )
    lab_unit: Optional[str] = Field(
        None,
        alias="labUnit",
        description="Unit for the lab value",
    )

    # Contraindication classification
    contraindication_type: Literal["absolute", "relative"] = Field(
        ...,
        alias="type",
        description="Whether contraindication is absolute or relative",
    )
    severity: Literal["CRITICAL", "HIGH", "MODERATE", "LOW"] = Field(
        ...,
        description="Clinical severity of the contraindication",
    )

    # Clinical context
    clinical_rationale: str = Field(
        ...,
        alias="clinicalRationale",
        description="WHY this is contraindicated (mechanism, risk)",
    )
    risk_description: Optional[str] = Field(
        None,
        alias="riskDescription",
        description="Description of adverse outcome if used",
    )
    alternative_considerations: Optional[str] = Field(
        None,
        alias="alternativeConsiderations",
        description="Alternative drugs or approaches",
    )

    # Provenance
    source_snippet: Optional[str] = Field(
        None,
        alias="sourceSnippet",
        description="Verbatim text from guideline (for audit)",
        max_length=1000,
    )
    governance: ClinicalGovernance = Field(
        ...,
        description="Full governance metadata",
    )


class WarningFact(BaseModel):
    """
    Drug warning that doesn't rise to contraindication level.

    Represents precautions, monitoring requirements, or dose adjustments
    needed for specific populations or conditions.
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

    warning_type: Literal[
        "BLACK_BOX",
        "PRECAUTION",
        "SPECIAL_POPULATION",
        "MONITORING_REQUIRED",
        "DOSE_ADJUSTMENT",
    ] = Field(
        ...,
        alias="warningType",
        description="Category of warning",
    )
    warning_text: str = Field(
        ...,
        alias="warningText",
        description="Warning description",
    )

    affected_population: Optional[str] = Field(
        None,
        alias="affectedPopulation",
        description="Population this warning applies to",
    )
    condition_codes: list[str] = Field(
        default_factory=list,
        alias="conditionCodes",
        description="ICD-10 codes for conditions triggering warning",
    )

    action_required: str = Field(
        ...,
        alias="actionRequired",
        description="What clinician should do",
    )

    governance: ClinicalGovernance = Field(
        ...,
        description="Full governance metadata",
    )


class KB4ExtractionResult(BaseModel):
    """
    L3 output for KB-4: Complete extraction result for safety facts.

    This is the top-level schema returned by the fact extractor when
    target_kb="safety".
    """

    model_config = ConfigDict(populate_by_name=True)

    contraindications: list[ContraindicationFact] = Field(
        ...,
        description="List of extracted contraindication facts",
    )
    warnings: list[WarningFact] = Field(
        default_factory=list,
        description="List of extracted warning facts",
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
    total_contraindications: int = Field(
        0,
        alias="totalContraindications",
        description="Count of contraindications extracted",
    )

    def model_post_init(self, __context) -> None:
        """Auto-calculate totals after initialization."""
        object.__setattr__(self, "total_contraindications", len(self.contraindications))
