"""
KB-20 Contextual Modifiers & ADR Profiles Schema - Pydantic models.

Extracts adverse drug reaction profiles with onset windows and contextual
modifiers (population, comorbidity, concomitant drug, lab value, temporal)
from clinical guidelines.

These facts are consumed by P2's context modifier rules (CM03/CM07) for
stratum-conditional medication overlay in the Bayesian differential engine.

L3 template structure matches P2's context modifier format:
  {drug_class, symptom, mechanism, onset_window, context_modifier_rule,
   source, confidence}

Go struct alignment:
- Future kb-20-contextual-modifiers/ service (not yet implemented)
- Field aliases map snake_case (Python) to camelCase (Go JSON)

Completeness grading:
- FULL: All required fields populated (drug + reaction + onset_window +
        mechanism + at least 1 contextual_modifier)
- PARTIAL: Drug + reaction present, but onset_window OR mechanism missing
- STUB: Only drug class identified, reaction/modifiers absent
"""

from typing import Optional, Literal
from pydantic import BaseModel, Field, ConfigDict


class ClinicalGovernance(BaseModel):
    """
    Provenance and governance metadata for extracted contextual facts.
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


class ContextualModifierFact(BaseModel):
    """
    A contextual modifier that alters the clinical significance of a drug
    reaction or dosing recommendation.

    Represents a single modifier condition (population characteristic,
    comorbidity, concomitant drug, lab value, or temporal factor) that
    changes how a drug reaction should be interpreted or managed.
    """

    model_config = ConfigDict(populate_by_name=True)

    modifier_type: Literal[
        "POPULATION",
        "COMORBIDITY",
        "CONCOMITANT_DRUG",
        "LAB_VALUE",
        "TEMPORAL",
    ] = Field(
        ...,
        alias="modifierType",
        description="Category of contextual modifier",
    )
    modifier_value: str = Field(
        ...,
        alias="modifierValue",
        description="Specific modifier (e.g., 'elderly >75', 'CKD stage 4', 'potassium >5.5')",
    )
    effect: str = Field(
        ...,
        description="How this modifier changes clinical interpretation "
        "(e.g., 'increases risk', 'contraindicated', 'requires dose reduction')",
    )
    effect_magnitude: Optional[Literal["MAJOR", "MODERATE", "MINOR"]] = Field(
        None,
        alias="effectMagnitude",
        description="Magnitude of the modifier's effect on clinical decision",
    )

    # For LAB_VALUE modifiers: structured threshold
    lab_parameter: Optional[str] = Field(
        None,
        alias="labParameter",
        description="Lab parameter name (e.g., 'eGFR', 'potassium')",
    )
    lab_operator: Optional[Literal["<", "<=", ">", ">=", "=="]] = Field(
        None,
        alias="labOperator",
        description="Comparison operator for lab threshold",
    )
    lab_threshold: Optional[float] = Field(
        None,
        alias="labThreshold",
        description="Numeric threshold value",
    )
    lab_unit: Optional[str] = Field(
        None,
        alias="labUnit",
        description="Unit for the lab value",
    )

    # For CONCOMITANT_DRUG modifiers
    concomitant_drug: Optional[str] = Field(
        None,
        alias="concomitantDrug",
        description="Drug name causing the interaction",
    )
    concomitant_rxnorm: Optional[str] = Field(
        None,
        alias="concomitantRxnorm",
        description="RxNorm code for concomitant drug",
    )

    # P2 context modifier rule mapping
    context_modifier_rule: Optional[str] = Field(
        None,
        alias="contextModifierRule",
        description="P2 context modifier rule ID (e.g., 'CM03', 'CM07')",
    )

    # Completeness grading
    completeness_grade: Literal["FULL", "PARTIAL", "STUB"] = Field(
        "STUB",
        alias="completenessGrade",
        description="Data completeness: FULL (all fields), PARTIAL (core present), STUB (minimal)",
    )

    def model_post_init(self, __context) -> None:
        """Auto-calculate completeness grade based on populated fields."""
        has_core = bool(self.modifier_value and self.effect)
        has_structured = False
        if self.modifier_type == "LAB_VALUE":
            has_structured = bool(
                self.lab_parameter and self.lab_operator and self.lab_threshold
            )
        elif self.modifier_type == "CONCOMITANT_DRUG":
            has_structured = bool(self.concomitant_drug)
        else:
            has_structured = bool(self.effect_magnitude)

        if has_core and has_structured:
            grade = "FULL"
        elif has_core:
            grade = "PARTIAL"
        else:
            grade = "STUB"
        object.__setattr__(self, "completeness_grade", grade)


class AdverseReactionProfile(BaseModel):
    """
    Adverse drug reaction profile with onset window and contextual modifiers.

    Represents a single ADR for a drug, including when it typically occurs,
    how frequent it is, and what contextual factors modify the risk.
    This is the primary data structure P2 consumes for medication overlay.
    """

    model_config = ConfigDict(populate_by_name=True)

    # Drug identification
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
        description="Therapeutic class (e.g., 'SGLT2i', 'Biguanide')",
    )

    # Reaction details
    reaction: str = Field(
        ...,
        description="Adverse reaction description (e.g., 'lactic acidosis', 'hyperkalemia')",
    )
    reaction_snomed: Optional[str] = Field(
        None,
        alias="reactionSnomed",
        description="SNOMED-CT code for the reaction if available",
    )
    mechanism: Optional[str] = Field(
        None,
        description="Pharmacological mechanism (e.g., 'impaired hepatic lactate clearance')",
    )
    symptom: Optional[str] = Field(
        None,
        description="Presenting symptom mapped to P2 differential "
        "(e.g., 'breathlessness', 'nausea', 'hypotension')",
    )

    # Onset window
    onset_window: Optional[str] = Field(
        None,
        alias="onsetWindow",
        description="Typical onset timeframe (e.g., '2-4 weeks', 'hours', 'days to weeks')",
    )
    onset_category: Optional[Literal[
        "IMMEDIATE", "ACUTE", "SUBACUTE", "CHRONIC", "DELAYED", "IDIOSYNCRATIC"
    ]] = Field(
        None,
        alias="onsetCategory",
        description="Categorized onset: IMMEDIATE (<1h), ACUTE (1h-7d), "
        "SUBACUTE (1-6wk), CHRONIC (>6wk), DELAYED (variable), "
        "IDIOSYNCRATIC (unpredictable, not PK-determinable)",
    )

    # Frequency and severity
    frequency: Optional[Literal[
        "VERY_COMMON", "COMMON", "UNCOMMON", "RARE", "VERY_RARE", "UNKNOWN"
    ]] = Field(
        None,
        description="ADR frequency: VERY_COMMON (>10%), COMMON (1-10%), "
        "UNCOMMON (0.1-1%), RARE (0.01-0.1%), VERY_RARE (<0.01%)",
    )
    severity: Optional[Literal["CRITICAL", "HIGH", "MODERATE", "LOW"]] = Field(
        None,
        description="Clinical severity of the adverse reaction",
    )

    # Risk factors and contextual modifiers
    risk_factors: list[str] = Field(
        default_factory=list,
        alias="riskFactors",
        description="Known risk factors (e.g., 'renal impairment', 'dehydration')",
    )
    contextual_modifiers: list[ContextualModifierFact] = Field(
        default_factory=list,
        alias="contextualModifiers",
        description="Contextual modifiers that change ADR risk",
    )

    # Completeness grading
    completeness_grade: Literal["FULL", "PARTIAL", "STUB"] = Field(
        "STUB",
        alias="completenessGrade",
        description="Data completeness: FULL (all fields), PARTIAL (core present), STUB (minimal)",
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

    def model_post_init(self, __context) -> None:
        """Auto-calculate completeness grade based on populated fields."""
        has_drug = bool(self.drug_name and self.reaction)
        has_onset = bool(self.onset_window)
        has_mechanism = bool(self.mechanism)
        has_modifier = len(self.contextual_modifiers) > 0

        if has_drug and has_onset and has_mechanism and has_modifier:
            grade = "FULL"
        elif has_drug and (has_onset or has_mechanism):
            grade = "PARTIAL"
        else:
            grade = "STUB"
        object.__setattr__(self, "completeness_grade", grade)


class KB20ExtractionResult(BaseModel):
    """
    L3 output for KB-20: Complete extraction result for contextual modifiers
    and adverse drug reaction profiles.

    This is the top-level schema returned by the fact extractor when
    target_kb="contextual".
    """

    model_config = ConfigDict(populate_by_name=True)

    adr_profiles: list[AdverseReactionProfile] = Field(
        ...,
        alias="adrProfiles",
        description="List of extracted adverse drug reaction profiles",
    )
    standalone_modifiers: list[ContextualModifierFact] = Field(
        default_factory=list,
        alias="standaloneModifiers",
        description="Contextual modifiers not tied to a specific ADR",
    )
    extraction_date: str = Field(
        ...,
        alias="extractionDate",
        description="ISO date when extraction was performed",
    )
    extractor_version: str = Field(
        "v4.0.0-facts",
        alias="extractorVersion",
        description="Version of the fact extraction pipeline",
    )
    source_guideline: str = Field(
        ...,
        alias="sourceGuideline",
        description="Identifier for the source guideline",
    )
    total_adr_profiles: int = Field(
        0,
        alias="totalAdrProfiles",
        description="Count of ADR profiles extracted",
    )
    total_contextual_modifiers: int = Field(
        0,
        alias="totalContextualModifiers",
        description="Count of contextual modifiers (within ADRs + standalone)",
    )
    completeness_summary: dict[str, int] = Field(
        default_factory=dict,
        alias="completenessSummary",
        description="Count of profiles by completeness grade: {FULL: N, PARTIAL: N, STUB: N}",
    )

    def model_post_init(self, __context) -> None:
        """Auto-calculate totals and completeness summary after initialization."""
        object.__setattr__(self, "total_adr_profiles", len(self.adr_profiles))

        modifier_count = len(self.standalone_modifiers)
        for adr in self.adr_profiles:
            modifier_count += len(adr.contextual_modifiers)
        object.__setattr__(self, "total_contextual_modifiers", modifier_count)

        summary = {"FULL": 0, "PARTIAL": 0, "STUB": 0}
        for adr in self.adr_profiles:
            summary[adr.completeness_grade] = summary.get(adr.completeness_grade, 0) + 1
        object.__setattr__(self, "completeness_summary", summary)
