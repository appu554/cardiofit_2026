"""
KB-5 Drug Interactions Schema — Pydantic models.

Extracts drug-drug interaction (DDI) facts from clinical guidelines.

These facts are consumed by:
- KB-5 Drug Interactions Service (interaction_matrix table)
- Medication Advisor for safety alerts
- CQL libraries that gate prescribing decisions

Authority hierarchy:
- TIER_0_ONC_HIGH: oncology high-priority (TKI, anticoagulants)
- TIER_1_SEVERE: regulatory black-box (FDA labels, EMA contraindications)
- TIER_2_MODERATE: guideline-anchored (ADA, KDIGO, AHA) — **this extractor**
- TIER_3_MECHANISM: mechanism-based reasoning (CYP, transporters)

ADA-2026 extracted DDIs are TIER_2_MODERATE: guideline-anchored. They are
SUPPLEMENTAL to a primary DrugBank/Lexicomp feed; this extractor does not
replace those primary sources.

Go struct alignment:
- interaction_matrix table in canonical_facts DB
- Field aliases map snake_case (Python) to camelCase (Go JSON)
"""

from typing import Optional, Literal
from pydantic import BaseModel, Field, ConfigDict


class ClinicalGovernance(BaseModel):
    """Provenance metadata for extracted DDI facts."""
    model_config = ConfigDict(populate_by_name=True)

    source_authority: str = Field(..., alias="sourceAuthority",
                                  description="Authority body (ADA, KDIGO, etc.)")
    source_document: str = Field(..., alias="sourceDocument",
                                 description="Full document title")
    source_section: Optional[str] = Field(None, alias="sourceSection",
                                          description="Recommendation ID or section")
    evidence_level: str = Field("", alias="evidenceLevel",
                                description="Evidence grade (A/B/C/E)")
    effective_date: str = Field(..., alias="effectiveDate",
                                description="Guideline effective date (ISO)")
    guideline_doi: Optional[str] = Field(None, alias="guidelineDoi")


class InteractionPartner(BaseModel):
    """The other drug or class in a DDI pair."""
    model_config = ConfigDict(populate_by_name=True)

    drug_name: str = Field(..., alias="drugName",
                           description="Other drug or class name")
    rxnorm_code: Optional[str] = Field(None, alias="rxnormCode",
                                       description="RxNorm code or '<LOOKUP_REQUIRED>'")
    drug_class: Optional[str] = Field(None, alias="drugClass",
                                      description="ATC class or descriptive class")


class DrugInteraction(BaseModel):
    """A single drug-drug interaction fact extracted from guideline."""
    model_config = ConfigDict(populate_by_name=True)

    target_drug: str = Field(..., alias="targetDrug",
                             description="The dossier's drug (left side of the pair)")
    target_rxnorm: Optional[str] = Field(None, alias="targetRxnorm",
                                         description="RxNorm of target_drug")
    target_drug_class: Optional[str] = Field(None, alias="targetDrugClass")

    partner: InteractionPartner = Field(...,
                                        description="The interacting drug/class")

    severity: Literal["CRITICAL", "HIGH", "MODERATE", "LOW"] = Field(...,
        description="Severity of the interaction")

    clinical_effect: str = Field(..., alias="clinicalEffect",
                                 description="What happens (e.g., 'increased risk of "
                                             "hyperkalemia', 'reduced glycemic control')")
    mechanism: Optional[str] = Field(None,
                                     description="Pharmacological mechanism if specified")
    management: str = Field(..., description="What to do (avoid combo, monitor X, "
                                             "adjust dose Y, etc.)")

    is_bidirectional: bool = Field(True, alias="isBidirectional",
                                   description="True if both drugs affect each other; "
                                               "False if precipitant→object only")
    precipitant_drug: Optional[str] = Field(None, alias="precipitantDrug",
                                            description="If unidirectional, the drug doing the affecting")
    object_drug: Optional[str] = Field(None, alias="objectDrug",
                                       description="If unidirectional, the drug being affected")

    evidence_level: Optional[str] = Field(None, alias="evidenceLevel",
                                          description="ADA evidence grade A/B/C/E")
    documentation: Optional[Literal[
        "ESTABLISHED", "PROBABLE", "SUSPECTED", "POSSIBLE", "UNLIKELY"
    ]] = Field(None, description="Strength of clinical evidence")

    source_snippet: str = Field(..., alias="sourceSnippet",
                                description="Verbatim guideline text (≤500 chars)")
    source_page: Optional[int] = Field(None, alias="sourcePage")
    recommendation_id: Optional[str] = Field(None, alias="recommendationId")

    governance: Optional[ClinicalGovernance] = Field(None)


class KB5ExtractionResult(BaseModel):
    """Top-level KB-5 extraction result for one drug dossier.

    A single dossier produces many interaction rows, one per partner drug or
    class identified in the guideline as interacting with the target drug.
    """
    model_config = ConfigDict(populate_by_name=True)

    interactions: list[DrugInteraction] = Field(default_factory=list,
        description="All DDIs found involving the target drug")
    extraction_date: str = Field(..., alias="extractionDate",
                                 description="ISO date of extraction")
    extractor_version: str = Field("v3.0.0-facts", alias="extractorVersion")
    source_guideline: str = Field("Unknown", alias="sourceGuideline")
    total_interactions: int = Field(0, alias="totalInteractions")
