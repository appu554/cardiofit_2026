"""KB-25 Lifestyle Knowledge Graph Schema — Pydantic models.

Extracts lifestyle / behavior / nutrition / activity recommendations from
clinical guidelines (ADA SOC 2026 Section 5 + cross-section lifestyle text).

Consumed by:
- KB-25 Lifestyle Knowledge Graph service
- KB-23 Decision Cards (lifestyle component of INITIATE/INTENSIFY scenarios)
- KB-21 Behavioral Intelligence (DSMES + adherence linkage)

Authority:
- TIER_2_GUIDELINE: ADA-grade A/B/C/E recommendations
- TIER_3_OBSERVATIONAL: study-derived effect magnitudes (CAVIAR, Look AHEAD,
  DiRECT, etc.) when explicitly cited
"""
from typing import Optional, Literal
from pydantic import BaseModel, Field, ConfigDict


class TargetPopulation(BaseModel):
    """Who the recommendation applies to."""
    model_config = ConfigDict(populate_by_name=True)

    description: str = Field(...,
        description="Free-text population description (e.g., "
                    "'adults with T2D and overweight/obesity')")
    age_min: Optional[int] = Field(None, alias="ageMin")
    age_max: Optional[int] = Field(None, alias="ageMax")
    diabetes_type: Optional[Literal["T1D", "T2D", "GDM", "PREDIABETES", "ANY"]] = \
        Field(None, alias="diabetesType")
    bmi_min: Optional[float] = Field(None, alias="bmiMin")
    egfr_min: Optional[float] = Field(None, alias="egfrMin")
    egfr_max: Optional[float] = Field(None, alias="egfrMax")
    excludes: Optional[str] = Field(None,
        description="Population the recommendation excludes "
                    "(e.g., 'pregnant women', 'frail older adults')")


class IntensitySpec(BaseModel):
    """Quantitative description of intervention dose."""
    model_config = ConfigDict(populate_by_name=True)

    amount: Optional[str] = Field(None,
        description="Dose / amount (e.g., '150 min', '0.6-0.8 g/kg/day', '5-7%')")
    frequency: Optional[str] = Field(None,
        description="How often (e.g., 'weekly', 'daily', '2-3 sessions/week')")
    duration: Optional[str] = Field(None,
        description="Length of intervention (e.g., '12 weeks', '6 months')")


class ExpectedEffect(BaseModel):
    """Quantitative outcome the intervention targets."""
    model_config = ConfigDict(populate_by_name=True)

    outcome: str = Field(...,
        description="What changes (e.g., 'A1C', 'body weight', 'LDL')")
    direction: Literal["DECREASE", "INCREASE", "MAINTAIN"] = Field(...)
    magnitude: Optional[str] = Field(None,
        description="Numeric magnitude (e.g., '0.5-1%', '5-10 lb', "
                    "'reduce ASCVD events by 21%')")
    timeframe: Optional[str] = Field(None,
        description="When effect is observed (e.g., '3-6 months')")


class LifestyleRecommendation(BaseModel):
    """A single lifestyle recommendation extracted from guideline."""
    model_config = ConfigDict(populate_by_name=True)

    intervention_type: Literal[
        "PHYSICAL_ACTIVITY",     # exercise, aerobic, resistance, balance
        "MEDICAL_NUTRITION",     # MNT, dietary patterns, macronutrient targets
        "WEIGHT_MANAGEMENT",     # weight loss/gain targets, intensive lifestyle
        "BEHAVIORAL",            # cognitive-behavioral, motivational interviewing
        "SLEEP",                 # sleep hygiene, OSA screening linkage
        "TOBACCO_CESSATION",
        "ALCOHOL_MODERATION",
        "DSMES",                 # diabetes self-management education and support
        "PSYCHOSOCIAL",          # depression/distress screening + intervention
        "SOCIAL_DETERMINANT",    # food security, housing, transportation
    ] = Field(..., alias="interventionType")

    title: str = Field(..., max_length=200,
        description="Short label, e.g., 'Aerobic + resistance for T2D'")
    recommendation_text: str = Field(..., alias="recommendationText",
        description="The core advice, paraphrased to ≤500 chars")

    target_population: TargetPopulation = Field(..., alias="targetPopulation")
    intensity: Optional[IntensitySpec] = Field(None)
    expected_effect: Optional[ExpectedEffect] = Field(None,
        alias="expectedEffect")

    mechanism: Optional[str] = Field(None,
        description="Why it works clinically (e.g., 'improves insulin "
                    "sensitivity via skeletal-muscle GLUT4 translocation')")

    contraindications: Optional[str] = Field(None,
        description="Populations or conditions where the intervention should "
                    "be avoided or modified")

    paired_with_drug: Optional[str] = Field(None, alias="pairedWithDrug",
        description="If the recommendation is explicitly co-prescribed with "
                    "a drug class (e.g., 'high-intensity statin therapy + "
                    "lifestyle for LDL reduction')")

    evidence_level: Optional[Literal["A", "B", "C", "E"]] = \
        Field(None, alias="evidenceLevel",
              description="ADA evidence grade if specified")

    source_snippet: str = Field(..., alias="sourceSnippet", max_length=500,
        description="Verbatim guideline text supporting the extraction")
    source_section: Optional[str] = Field(None, alias="sourceSection")
    source_page: Optional[int] = Field(None, alias="sourcePage")
    recommendation_id: Optional[str] = Field(None, alias="recommendationId",
        description="ADA recommendation number if labeled (e.g., '5.4')")


class KB25ExtractionResult(BaseModel):
    """Top-level KB-25 extraction result for one lifestyle dossier.

    A dossier is a topical chunk of verified guideline spans (e.g., all
    physical-activity spans across sections), not a single drug.
    """
    model_config = ConfigDict(populate_by_name=True)

    recommendations: list[LifestyleRecommendation] = Field(default_factory=list)
    extraction_date: str = Field(..., alias="extractionDate")
    extractor_version: str = Field("v3.0.0-kb25", alias="extractorVersion")
    source_guideline: str = Field("Unknown", alias="sourceGuideline")
    dossier_topic: str = Field(..., alias="dossierTopic",
        description="Topic label of the dossier "
                    "(physical_activity / nutrition / weight / behavioral / etc.)")
    total_recommendations: int = Field(0, alias="totalRecommendations")
