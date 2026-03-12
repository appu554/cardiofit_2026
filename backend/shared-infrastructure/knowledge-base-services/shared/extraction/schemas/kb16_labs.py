"""
KB-16 Lab Monitoring Schema - Pydantic models matching Go structs.

Extracts lab monitoring requirement FACTS from clinical guidelines.
These facts are consumed by CQL rules in vaidshala/tier-4-guidelines/.

Go struct alignment:
- kb-4-patient-safety/pkg/safety/types.go -> LabRequirement, LabMonitoringEntry
- kb-16-lab-interpretation/ for lab interpretation specifics
- Field aliases map snake_case (Python) to camelCase (Go JSON)
"""

from typing import Optional, Union, Any
from pydantic import BaseModel, Field, ConfigDict


class ClinicalGovernance(BaseModel):
    """
    Provenance and governance metadata for extracted lab monitoring facts.
    """

    model_config = ConfigDict(populate_by_name=True)

    source_authority: str = Field(
        ...,
        alias="sourceAuthority",
        description="Authoritative body (e.g., 'KDIGO', 'FDA')",
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
        description="Evidence grade",
    )
    effective_date: str = Field(
        ...,
        alias="effectiveDate",
        description="When guideline became effective",
    )


class CriticalValueThreshold(BaseModel):
    """
    Critical value threshold that triggers clinical action.

    When a lab result crosses this threshold, specific action is required.
    """

    model_config = ConfigDict(populate_by_name=True)

    operator: str = Field(
        ...,
        description="Comparison operator: '<', '<=', '>', '>=', '=='",
    )
    value: float = Field(
        ...,
        description="Threshold value",
    )
    unit: str = Field(
        ...,
        description="Unit for the value",
    )
    action: str = Field(
        ...,
        description="Action to take when threshold crossed (e.g., 'HOLD', 'STOP', 'NOTIFY')",
    )
    urgency: Optional[str] = Field(
        None,
        description="Urgency level: 'IMMEDIATE', 'WITHIN_24H', 'NEXT_VISIT'",
    )


class TargetRange(BaseModel):
    """
    Target therapeutic range for a lab value.
    """

    model_config = ConfigDict(populate_by_name=True)

    low: Optional[float] = Field(
        None,
        description="Lower bound of target range",
    )
    high: Optional[float] = Field(
        None,
        description="Upper bound of target range",
    )
    unit: str = Field(
        ...,
        description="Unit for the range values",
    )
    population_context: Optional[str] = Field(
        None,
        alias="populationContext",
        description="Population this range applies to (e.g., 'CKD Stage 4')",
    )


class LabMonitoringEntry(BaseModel):
    """
    Single lab monitoring requirement matching KB-16 Go struct.

    Represents a specific lab test that must be monitored when using a drug.
    """

    model_config = ConfigDict(populate_by_name=True)

    lab_name: str = Field(
        ...,
        alias="labName",
        description="Human-readable lab test name",
    )
    loinc_code: str = Field(
        ...,
        alias="loincCode",
        description="LOINC code for the lab test",
    )

    # Monitoring schedule
    frequency: str = Field(
        ...,
        description="Monitoring frequency (e.g., 'Q3-6 months', 'weekly x 4 then monthly')",
    )
    baseline_timing: Optional[str] = Field(
        None,
        alias="baselineTiming",
        description="When to obtain baseline (e.g., 'before initiation')",
    )
    initial_monitoring: Optional[str] = Field(
        None,
        alias="initialMonitoring",
        description="Initial monitoring schedule (e.g., 'week 1, 2, 4')",
    )
    maintenance_frequency: Optional[str] = Field(
        None,
        alias="maintenanceFrequency",
        description="Ongoing monitoring frequency after stabilization",
    )

    # Thresholds and targets
    target_range: Optional[TargetRange] = Field(
        None,
        alias="targetRange",
        description="Target therapeutic range if applicable",
    )
    critical_high: Optional[CriticalValueThreshold] = Field(
        None,
        alias="criticalHigh",
        description="High critical value threshold",
    )
    critical_low: Optional[CriticalValueThreshold] = Field(
        None,
        alias="criticalLow",
        description="Low critical value threshold",
    )

    # Actions
    action_required: Optional[str] = Field(
        None,
        alias="actionRequired",
        description="Default action when monitoring shows concern",
    )
    clinical_context: Optional[str] = Field(
        None,
        alias="clinicalContext",
        description="Why this monitoring is important",
    )


class LabRequirementFact(BaseModel):
    """
    Complete lab monitoring requirements for a drug matching KB-16 Go struct.

    Contains all lab tests that must be monitored when prescribing this drug.
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
        description="Therapeutic class for class-wide requirements",
    )

    # Lab monitoring entries
    labs: list[LabMonitoringEntry] = Field(
        ...,
        description="List of lab tests requiring monitoring",
    )

    # Requirements flags
    baseline_required: bool = Field(
        ...,
        alias="baselineRequired",
        description="Whether baseline labs are required before starting",
    )
    monitoring_required: bool = Field(
        True,
        alias="monitoringRequired",
        description="Whether ongoing monitoring is required",
    )

    # Special populations
    special_population_notes: Optional[str] = Field(
        None,
        alias="specialPopulationNotes",
        description="Notes for special populations (elderly, renal impairment)",
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


class KB16ExtractionResult(BaseModel):
    """
    L3 output for KB-16: Complete extraction result for lab monitoring facts.

    This is the top-level schema returned by the fact extractor when
    target_kb="monitoring".
    """

    model_config = ConfigDict(populate_by_name=True)

    lab_requirements: list[LabRequirementFact] = Field(
        ...,
        alias="labRequirements",
        description="List of extracted lab monitoring requirements",
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
    total_drugs: int = Field(
        0,
        alias="totalDrugs",
        description="Count of drugs with monitoring requirements",
    )
    total_lab_tests: int = Field(
        0,
        alias="totalLabTests",
        description="Count of unique lab tests across all drugs",
    )

    def model_post_init(self, __context) -> None:
        """Auto-calculate totals after initialization."""
        object.__setattr__(self, "total_drugs", len(self.lab_requirements))
        unique_loincs = set()
        for req in self.lab_requirements:
            for lab in req.labs:
                unique_loincs.add(lab.loinc_code)
        object.__setattr__(self, "total_lab_tests", len(unique_loincs))
