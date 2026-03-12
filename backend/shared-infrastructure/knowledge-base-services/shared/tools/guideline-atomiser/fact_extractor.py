"""
V3 Fact Extractor - KB-Specific Clinical Fact Extraction

This module implements L3 of the 7-Layer Guideline Curation Pipeline.
It extracts FACTS (parameters, thresholds, values) from clinical guidelines
into KB-specific Pydantic schemas that match Go structs exactly.

Key Principle: Extract FACTS, not RULES. CQL defines logic; KBs store values.

Usage:
    extractor = KBFactExtractor(anthropic_client)
    result = extractor.extract_facts(
        markdown_text="...",
        gliner_entities=[...],
        target_kb="dosing",
        guideline_context={...}
    )

Target KBs:
    - "dosing" -> KB-1: RenalAdjustment, HepaticAdjustment
    - "safety" -> KB-4: Contraindication, Warning
    - "monitoring" -> KB-16: LabRequirement, MonitoringEntry
"""

import json
from datetime import date
from typing import Literal, Union, Optional
from pathlib import Path

try:
    from anthropic import Anthropic
except ImportError:
    Anthropic = None  # Type hint only when not installed

# Import KB-specific schemas
import sys
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from extraction.schemas.kb1_dosing import KB1ExtractionResult
from extraction.schemas.kb4_safety import KB4ExtractionResult
from extraction.schemas.kb16_labs import KB16ExtractionResult
from extraction.schemas.kb20_contextual import KB20ExtractionResult

# V4 multi-channel models (for dossier-based extraction)
try:
    from extraction.v4.models import DrugDossier, VerifiedSpan
except ImportError:
    DrugDossier = None
    VerifiedSpan = None


class KBFactExtractor:
    """
    L3: Extract KB-specific FACTS (not rules) from clinical guidelines.

    This extractor uses Claude's structured output capability to extract
    clinical facts into Pydantic schemas that exactly match the Go structs
    used by KB-1, KB-4, and KB-16.

    The extractor does NOT generate rules or CQL. It harvests parameters
    (thresholds, factors, flags) that existing CQL references.
    """

    KB_SCHEMA_MAP = {
        "dosing": KB1ExtractionResult,
        "safety": KB4ExtractionResult,
        "monitoring": KB16ExtractionResult,
        "contextual": KB20ExtractionResult,
    }

    def __init__(
        self,
        client: Optional["Anthropic"] = None,
        model: str = "claude-sonnet-4-20250514",
    ):
        """
        Initialize the fact extractor.

        Args:
            client: Anthropic client instance (optional, for testing)
            model: Model to use for extraction
        """
        self.client = client
        self.model = model

    def extract_facts(
        self,
        markdown_text: str,
        gliner_entities: list[dict],
        target_kb: Literal["dosing", "safety", "monitoring", "contextual"],
        guideline_context: dict,
    ) -> Union[KB1ExtractionResult, KB4ExtractionResult, KB16ExtractionResult, KB20ExtractionResult]:
        """
        Extract facts for a specific KB from guideline text.

        Args:
            markdown_text: Extracted text from guideline (L1 Marker output)
            gliner_entities: Pre-tagged clinical entities (L2 GLiNER output)
            target_kb: Target knowledge base ("dosing", "safety", "monitoring")
            guideline_context: Metadata about the source guideline

        Returns:
            KB-specific extraction result (KB1, KB4, or KB16)

        Raises:
            ValueError: If target_kb is not valid
            RuntimeError: If Anthropic client is not available
        """
        if target_kb not in self.KB_SCHEMA_MAP:
            raise ValueError(
                f"Invalid target_kb: {target_kb}. "
                f"Must be one of: {list(self.KB_SCHEMA_MAP.keys())}"
            )

        if self.client is None:
            raise RuntimeError(
                "Anthropic client not available. "
                "Install anthropic package and provide client."
            )

        schema = self.KB_SCHEMA_MAP[target_kb]
        prompt = self._build_prompt(
            target_kb, markdown_text, gliner_entities, guideline_context
        )

        response = self.client.messages.create(
            model=self.model,
            max_tokens=8192,
            tool_choice={"type": "any"},
            tools=[
                {
                    "name": f"extract_{target_kb}_facts",
                    "description": f"Extract {target_kb} facts for KB storage. "
                    f"Output must match the exact schema for Go struct compatibility.",
                    "input_schema": schema.model_json_schema(),
                }
            ],
            messages=[{"role": "user", "content": prompt}],
        )

        # Extract tool use result
        tool_result = None
        for block in response.content:
            if block.type == "tool_use":
                tool_result = block.input
                break

        if tool_result is None:
            raise RuntimeError("No tool use response from model")

        # Handle case where API returns JSON string instead of dict
        if isinstance(tool_result, str):
            tool_result = json.loads(tool_result)

        # Validate and return
        return schema.model_validate(tool_result)

    def extract_facts_from_dossier(
        self,
        dossier: "DrugDossier",
        target_kb: Literal["dosing", "safety", "monitoring", "contextual"],
        guideline_context: dict,
    ) -> Union[KB1ExtractionResult, KB4ExtractionResult, KB16ExtractionResult, KB20ExtractionResult]:
        """Extract facts for a specific KB from a reviewer-verified drug dossier.

        V4 Pipeline 2 entry point. Replaces extract_facts() for dossier-based
        extraction where text spans have been multi-channel extracted, merged,
        and human-reviewed before reaching L3.

        Args:
            dossier: Per-drug dossier with verified spans and source text
            target_kb: Target knowledge base ("dosing", "safety", "monitoring")
            guideline_context: Metadata about the source guideline

        Returns:
            KB-specific extraction result (KB1, KB4, or KB16)
        """
        if DrugDossier is None:
            raise ImportError(
                "V4 models not available. "
                "Ensure extraction.v4.models is importable."
            )

        if target_kb not in self.KB_SCHEMA_MAP:
            raise ValueError(
                f"Invalid target_kb: {target_kb}. "
                f"Must be one of: {list(self.KB_SCHEMA_MAP.keys())}"
            )

        if self.client is None:
            raise RuntimeError(
                "Anthropic client not available. "
                "Install anthropic package and provide client."
            )

        schema = self.KB_SCHEMA_MAP[target_kb]
        prompt = self._build_dossier_prompt(
            dossier, target_kb, guideline_context
        )

        response = self.client.messages.create(
            model=self.model,
            max_tokens=8192,
            tool_choice={"type": "any"},
            tools=[
                {
                    "name": f"extract_{target_kb}_facts",
                    "description": f"Extract {target_kb} facts for KB storage. "
                    f"Output must match the exact schema for Go struct compatibility.",
                    "input_schema": schema.model_json_schema(),
                }
            ],
            messages=[{"role": "user", "content": prompt}],
        )

        tool_result = None
        for block in response.content:
            if block.type == "tool_use":
                tool_result = block.input
                break

        if tool_result is None:
            raise RuntimeError("No tool use response from model")

        if isinstance(tool_result, str):
            tool_result = json.loads(tool_result)

        return schema.model_validate(tool_result)

    def _build_dossier_prompt(
        self,
        dossier: "DrugDossier",
        target_kb: str,
        context: dict,
    ) -> str:
        """Build extraction prompt from a per-drug dossier.

        Replaces the GLiNER entity section with reviewer-verified spans.
        """
        base_instructions = """You are extracting clinical FACTS from a medical guideline.

CRITICAL: You are extracting PARAMETERS that existing CQL rules reference, NOT generating rules.
- CQL defines WHEN something should happen (logic - already exists)
- KB values define WHAT VALUES it happens with (facts - what you extract)

Extract structured FACTS that match the provided schema exactly.
Include provenance information for every fact extracted.
"""

        # Format verified spans for the prompt
        span_lines = []
        for i, span in enumerate(dossier.verified_spans, 1):
            channels = ", ".join(span.contributing_channels) if span.contributing_channels else "reviewer-added"
            page_info = f" [page {span.page_number}]" if span.page_number else ""
            section_info = f" [section {span.section_id}]" if span.section_id else ""
            ctx_hints = ""
            if span.extraction_context:
                hint_parts = []
                for k, v in span.extraction_context.items():
                    if k.startswith("channel_") and k.endswith("_confidence"):
                        continue  # Skip confidence repetition
                    hint_parts.append(f"{k}={v}")
                if hint_parts:
                    ctx_hints = f" ({', '.join(hint_parts)})"
            span_lines.append(
                f'{i}. "{span.text}" (confidence: {span.confidence:.2f}, '
                f'channels: [{channels}]{page_info}{section_info}){ctx_hints}'
            )

        verified_spans_text = "\n".join(span_lines) if span_lines else "No spans available."

        # Signal summary
        summary_parts = []
        for signal_type, count in dossier.signal_summary.items():
            summary_parts.append(f"{signal_type}: {count}")
        signal_summary_text = ", ".join(summary_parts) if summary_parts else "none"

        # KB-specific extraction instructions
        if target_kb == "dosing":
            kb_instructions = """## Target: KB-1 Drug Dosing Facts

Extract RENAL DOSING FACTS for this drug.
- eGFR thresholds where dosing changes
- Adjustment factor (e.g., 0.5 for 50% reduction)
- Maximum dose limits at each threshold
- Whether the drug is contraindicated at that level
- Action type: CONTRAINDICATED, REDUCE_DOSE, REDUCE_FREQUENCY, MONITOR, NO_CHANGE"""
        elif target_kb == "safety":
            kb_instructions = """## Target: KB-4 Patient Safety Facts

Extract CONTRAINDICATION FACTS for this drug.
- Conditions that trigger contraindication
- Whether absolute or relative
- Severity: CRITICAL, HIGH, MODERATE, LOW
- Clinical rationale
- Lab-based thresholds if applicable"""
        elif target_kb == "monitoring":
            kb_instructions = """## Target: KB-16 Lab Monitoring Facts

Extract LAB MONITORING FACTS for this drug.
- Lab tests required
- Monitoring frequency
- Critical value thresholds
- Actions when critical values reached"""
        else:  # contextual
            kb_instructions = """## Target: KB-20 Contextual Modifiers & ADR Profiles

Extract ADVERSE DRUG REACTION PROFILES with onset windows and CONTEXTUAL MODIFIERS.

For each adverse reaction:
- Drug name, class, and RxNorm code
- Reaction description and mechanism
- Presenting symptom (map to P2 differential: breathlessness, nausea, hypotension, etc.)
- Onset window (e.g., "2-4 weeks", "hours") and category (IMMEDIATE/ACUTE/SUBACUTE/CHRONIC/DELAYED)
- Frequency (VERY_COMMON/COMMON/UNCOMMON/RARE/VERY_RARE)
- Severity (CRITICAL/HIGH/MODERATE/LOW)
- Risk factors (e.g., "renal impairment", "dehydration")

For each contextual modifier:
- Type: POPULATION, COMORBIDITY, CONCOMITANT_DRUG, LAB_VALUE, or TEMPORAL
- Value (e.g., "elderly >75", "CKD stage 4", "potassium >5.5")
- Effect on clinical interpretation
- For LAB_VALUE: structured threshold (parameter, operator, value, unit)
- For CONCOMITANT_DRUG: drug name and RxNorm code
- P2 context modifier rule if identifiable (CM03, CM07, etc.)"""

        rxnorm_info = ""
        if dossier.rxnorm_candidate:
            rxnorm_info = f"\nRxNorm candidate (from Channel B, verify): {dossier.rxnorm_candidate}"

        verified_codes = context.get("verified_rxnorm_codes", {})

        return f"""{base_instructions}

{kb_instructions}

## Drug: {dossier.drug_name}{rxnorm_info}
Signal summary: {signal_summary_text}
Source sections: {', '.join(dossier.source_sections) if dossier.source_sections else 'unknown'}
Source pages: {', '.join(str(p) for p in dossier.source_pages) if dossier.source_pages else 'unknown'}

## Reviewer-Verified Text Spans:

The following text spans have been confirmed by a human text reviewer.
The text is VERIFIED CORRECT.
Machine-generated extraction context is included but may be incorrect.

YOUR TASK: Classify each span and extract structured facts for {dossier.drug_name}.

Verified Spans:
{verified_spans_text}

## Source Text (enclosing sections):
{dossier.source_text[:8000]}

## Guideline Context:
- Authority: {context.get('authority', 'Unknown')}
- Document: {context.get('document', 'Unknown')}
- Effective Date: {context.get('effective_date', 'Unknown')}
- DOI: {context.get('doi', 'Unknown')}

## Pre-verified RxNorm Codes (from KB-7):
{self._format_verified_codes(verified_codes)}

## Instructions:
1. Use the verified spans above as your primary evidence
2. Extract ALL relevant facts for {dossier.drug_name} from the source text
3. Use exact numeric values from the text
4. Include the verbatim source snippet for each fact (max 500 chars)
5. Set extraction_date to today's date: {date.today().isoformat()}
6. Set source_guideline to: {context.get('document', 'Unknown')}

IMPORTANT - RxNorm Code Usage:
- Use the pre-verified RxNorm codes from KB-7 above when available
- Do NOT invent or guess RxNorm codes
- If a drug is not in the verified list, use "<LOOKUP_REQUIRED>" as the rxnormCode

Extract structured FACTS matching the schema for {dossier.drug_name}."""

    def _build_prompt(
        self,
        target_kb: str,
        text: str,
        entities: list[dict],
        context: dict,
    ) -> str:
        """
        Build KB-specific extraction prompt.

        Args:
            target_kb: Target knowledge base
            text: Guideline text to extract from
            entities: Pre-tagged entities from GLiNER
            context: Guideline metadata

        Returns:
            Formatted prompt for extraction
        """
        base_instructions = """You are extracting clinical FACTS from a medical guideline.

CRITICAL: You are extracting PARAMETERS that existing CQL rules reference, NOT generating rules.
- CQL defines WHEN something should happen (logic - already exists)
- KB values define WHAT VALUES it happens with (facts - what you extract)

Extract structured FACTS that match the provided schema exactly.
Include provenance information for every fact extracted.
"""

        if target_kb == "dosing":
            return self._build_dosing_prompt(text, entities, context, base_instructions)
        elif target_kb == "safety":
            return self._build_safety_prompt(text, entities, context, base_instructions)
        elif target_kb == "monitoring":
            return self._build_monitoring_prompt(
                text, entities, context, base_instructions
            )
        elif target_kb == "contextual":
            return self._build_contextual_prompt(
                text, entities, context, base_instructions
            )
        else:
            raise ValueError(f"Unknown target_kb: {target_kb}")

    def _build_dosing_prompt(
        self,
        text: str,
        entities: list[dict],
        context: dict,
        base_instructions: str,
    ) -> str:
        """Build prompt for KB-1 dosing fact extraction."""
        return f"""{base_instructions}

## Target: KB-1 Drug Dosing Facts

Extract RENAL DOSING FACTS from this guideline text.

For each drug mentioned, extract:
- RxNorm code (if available) and drug name
- eGFR thresholds where dosing changes
- Adjustment factor (e.g., 0.5 for 50% reduction)
- Maximum dose limits at each threshold
- Whether the drug is contraindicated at that level
- Action type: CONTRAINDICATED, REDUCE_DOSE, REDUCE_FREQUENCY, MONITOR, NO_CHANGE

## Pre-tagged Clinical Entities (from GLiNER):
```json
{json.dumps(entities, indent=2)}
```

## Guideline Context:
- Authority: {context.get('authority', 'Unknown')}
- Document: {context.get('document', 'Unknown')}
- Effective Date: {context.get('effective_date', 'Unknown')}
- DOI: {context.get('doi', 'Unknown')}

## Guideline Text:
{text}

## Pre-verified RxNorm Codes (from KB-7):
{self._format_verified_codes(context.get('verified_rxnorm_codes', {}))}

## Instructions:
1. Identify all drugs with renal dosing guidance
2. For each drug, extract ALL eGFR thresholds mentioned
3. Use exact numeric values from the text (e.g., 30, 45, 60)
4. Include the verbatim source snippet for each fact (max 500 chars)
5. Set extraction_date to today's date: {date.today().isoformat()}
6. Set source_guideline to: {context.get('document', 'Unknown')}

IMPORTANT - RxNorm Code Usage:
- Use the pre-verified RxNorm codes from KB-7 above when available
- Do NOT invent or guess RxNorm codes
- If a drug is not in the verified list, use "<LOOKUP_REQUIRED>" as the rxnormCode

Extract structured FACTS matching the KB1ExtractionResult schema."""

    def _build_safety_prompt(
        self,
        text: str,
        entities: list[dict],
        context: dict,
        base_instructions: str,
    ) -> str:
        """Build prompt for KB-4 safety fact extraction."""
        return f"""{base_instructions}

## Target: KB-4 Patient Safety Facts

Extract CONTRAINDICATION FACTS from this guideline text.

For each contraindication mentioned, extract:
- Drug (RxNorm code if available, or drug class)
- Condition that triggers contraindication (with ICD-10 code if mentioned)
- Whether it's absolute or relative
- Severity level: CRITICAL, HIGH, MODERATE, or LOW
- Clinical rationale (WHY it's contraindicated - mechanism, risk)
- Alternative considerations if mentioned

For lab-based contraindications, also extract:
- Lab parameter name (e.g., "eGFR", "potassium")
- LOINC code if available
- Threshold value and operator
- Unit of measure

## Pre-tagged Clinical Entities (from GLiNER):
```json
{json.dumps(entities, indent=2)}
```

## Guideline Context:
- Authority: {context.get('authority', 'Unknown')}
- Document: {context.get('document', 'Unknown')}
- Effective Date: {context.get('effective_date', 'Unknown')}
- DOI: {context.get('doi', 'Unknown')}

## Guideline Text:
{text}

## Pre-verified RxNorm Codes (from KB-7):
{self._format_verified_codes(context.get('verified_rxnorm_codes', {}))}

## Instructions:
1. Identify all contraindications and warnings
2. Classify each as absolute or relative
3. Assign severity based on clinical impact
4. Include the verbatim source snippet for audit
5. Set extraction_date to today: {date.today().isoformat()}
6. Set source_guideline to: {context.get('document', 'Unknown')}

IMPORTANT - RxNorm Code Usage:
- Use the pre-verified RxNorm codes from KB-7 above when available
- Do NOT invent or guess RxNorm codes
- If a drug is not in the verified list, use "<LOOKUP_REQUIRED>" as the rxnormCode

Extract structured FACTS matching the KB4ExtractionResult schema."""

    def _build_monitoring_prompt(
        self,
        text: str,
        entities: list[dict],
        context: dict,
        base_instructions: str,
    ) -> str:
        """Build prompt for KB-16 lab monitoring fact extraction."""
        return f"""{base_instructions}

## Target: KB-16 Lab Monitoring Facts

Extract LAB MONITORING FACTS from this guideline text.

For each monitoring requirement mentioned, extract:
- Drug requiring monitoring (RxNorm code if available)
- Lab test name and LOINC code
- Baseline requirement (yes/no)
- Monitoring frequency (e.g., "Q3-6 months", "weekly x 4 then monthly")
- Initial monitoring schedule if different from maintenance
- Target range if specified
- Critical values that trigger action (with operator and threshold)
- Action required when critical value reached

## Pre-tagged Clinical Entities (from GLiNER):
```json
{json.dumps(entities, indent=2)}
```

## Guideline Context:
- Authority: {context.get('authority', 'Unknown')}
- Document: {context.get('document', 'Unknown')}
- Effective Date: {context.get('effective_date', 'Unknown')}
- DOI: {context.get('doi', 'Unknown')}

## Guideline Text:
{text}

## Pre-verified RxNorm Codes (from KB-7):
{self._format_verified_codes(context.get('verified_rxnorm_codes', {}))}

## Instructions:
1. Identify all lab monitoring requirements
2. Extract specific frequencies (not vague terms like "regularly")
3. Include critical value thresholds with actions
4. Include the verbatim source snippet for audit
5. Set extraction_date to today: {date.today().isoformat()}
6. Set source_guideline to: {context.get('document', 'Unknown')}

IMPORTANT - RxNorm Code Usage:
- Use the pre-verified RxNorm codes from KB-7 above when available
- Do NOT invent or guess RxNorm codes
- If a drug is not in the verified list, use "<LOOKUP_REQUIRED>" as the rxnormCode

Extract structured FACTS matching the KB16ExtractionResult schema."""

    def _build_contextual_prompt(
        self,
        text: str,
        entities: list[dict],
        context: dict,
        base_instructions: str,
    ) -> str:
        """Build prompt for KB-20 contextual modifier & ADR profile extraction."""
        return f"""{base_instructions}

## Target: KB-20 Contextual Modifiers & ADR Profiles

Extract ADVERSE DRUG REACTION PROFILES and CONTEXTUAL MODIFIERS from this
guideline text. This data feeds P2's medication overlay in the Bayesian
differential diagnosis engine.

For each drug mentioned with adverse effects, extract:
- Drug identification (RxNorm code, name, class)
- Reaction description and pharmacological mechanism
- Presenting symptom mapped to P2 differential (breathlessness, nausea, etc.)
- Onset window and category (IMMEDIATE/ACUTE/SUBACUTE/CHRONIC/DELAYED)
- Frequency (VERY_COMMON >10%, COMMON 1-10%, UNCOMMON 0.1-1%, RARE <0.1%)
- Severity (CRITICAL, HIGH, MODERATE, LOW)
- Risk factors that increase ADR probability

For each contextual modifier, extract:
- Type: POPULATION (age, sex, ethnicity), COMORBIDITY (CKD, HF, liver disease),
  CONCOMITANT_DRUG (drug-drug interactions), LAB_VALUE (eGFR, potassium thresholds),
  TEMPORAL (duration-dependent effects)
- Specific value and effect on clinical interpretation
- For LAB_VALUE type: structured threshold (parameter, operator, value, unit)
- For CONCOMITANT_DRUG type: interacting drug name and RxNorm code

## Pre-tagged Clinical Entities (from GLiNER):
```json
{json.dumps(entities, indent=2)}
```

## Guideline Context:
- Authority: {context.get('authority', 'Unknown')}
- Document: {context.get('document', 'Unknown')}
- Effective Date: {context.get('effective_date', 'Unknown')}
- DOI: {context.get('doi', 'Unknown')}

## Guideline Text:
{text}

## Pre-verified RxNorm Codes (from KB-7):
{self._format_verified_codes(context.get('verified_rxnorm_codes', {{}}))}

## Instructions:
1. Identify all adverse reactions and contextual modifiers
2. Map each reaction to the presenting symptom a patient would report
3. Include onset timing — this is critical for P2's temporal reasoning
4. Classify each contextual modifier by type
5. For lab-based modifiers, extract the structured threshold
6. Include the verbatim source snippet for audit (max 500 chars)
7. Set extraction_date to today: {date.today().isoformat()}
8. Set source_guideline to: {context.get('document', 'Unknown')}

IMPORTANT - RxNorm Code Usage:
- Use the pre-verified RxNorm codes from KB-7 above when available
- Do NOT invent or guess RxNorm codes
- If a drug is not in the verified list, use "<LOOKUP_REQUIRED>" as the rxnormCode

Extract structured FACTS matching the KB20ExtractionResult schema."""

    def _format_verified_codes(self, verified_codes: dict) -> str:
        """
        Format verified RxNorm codes for inclusion in prompt.

        Args:
            verified_codes: Dict mapping drug_name_lower to code info

        Returns:
            Formatted string for prompt, or "No pre-verified codes available"
        """
        if not verified_codes:
            return "No pre-verified codes available. Use '<LOOKUP_REQUIRED>' for all rxnormCode values."

        lines = []
        for drug_name, info in verified_codes.items():
            code = info.get("code", "UNKNOWN")
            display = info.get("display", drug_name)
            lines.append(f"- {drug_name}: {code} (verified as: {display})")

        return "\n".join(lines)

    def validate_extraction(
        self,
        result: Union[KB1ExtractionResult, KB4ExtractionResult, KB16ExtractionResult, KB20ExtractionResult],
    ) -> dict:
        """
        Validate extraction result for completeness and consistency.

        Args:
            result: Extraction result to validate

        Returns:
            Validation report with issues and warnings
        """
        issues = []
        warnings = []

        if isinstance(result, KB1ExtractionResult):
            for drug in result.drugs:
                # Check for missing RxNorm codes
                if not drug.rxnorm_code or drug.rxnorm_code == "UNKNOWN":
                    warnings.append(
                        f"Missing RxNorm code for {drug.drug_name}"
                    )

                # Check for overlapping eGFR ranges
                adjustments = sorted(
                    drug.renal_adjustments, key=lambda x: x.egfr_min
                )
                for i in range(len(adjustments) - 1):
                    if adjustments[i].egfr_max > adjustments[i + 1].egfr_min:
                        issues.append(
                            f"Overlapping eGFR ranges for {drug.drug_name}: "
                            f"{adjustments[i].egfr_min}-{adjustments[i].egfr_max} and "
                            f"{adjustments[i + 1].egfr_min}-{adjustments[i + 1].egfr_max}"
                        )

        elif isinstance(result, KB4ExtractionResult):
            for ci in result.contraindications:
                # Check for missing condition codes
                if not ci.condition_codes:
                    warnings.append(
                        f"Missing ICD-10 codes for {ci.drug_name} contraindication"
                    )

                # Check for missing rationale
                if len(ci.clinical_rationale) < 10:
                    warnings.append(
                        f"Brief clinical rationale for {ci.drug_name}"
                    )

        elif isinstance(result, KB16ExtractionResult):
            for req in result.lab_requirements:
                for lab in req.labs:
                    # Check for missing LOINC codes
                    if not lab.loinc_code or lab.loinc_code == "UNKNOWN":
                        warnings.append(
                            f"Missing LOINC code for {lab.lab_name} ({req.drug_name})"
                        )

                    # Check for vague frequencies
                    vague_terms = ["regularly", "periodically", "as needed"]
                    if any(term in lab.frequency.lower() for term in vague_terms):
                        warnings.append(
                            f"Vague monitoring frequency for {lab.lab_name}: {lab.frequency}"
                        )

        elif isinstance(result, KB20ExtractionResult):
            for adr in result.adr_profiles:
                # Check for missing RxNorm codes
                if not adr.rxnorm_code or adr.rxnorm_code == "UNKNOWN":
                    warnings.append(
                        f"Missing RxNorm code for {adr.drug_name}"
                    )

                # Check for STUB completeness (missing onset or mechanism)
                if adr.completeness_grade == "STUB":
                    issues.append(
                        f"STUB-grade ADR for {adr.drug_name}/{adr.reaction}: "
                        f"missing onset_window and mechanism"
                    )
                elif adr.completeness_grade == "PARTIAL":
                    warnings.append(
                        f"PARTIAL-grade ADR for {adr.drug_name}/{adr.reaction}: "
                        f"onset_window={'present' if adr.onset_window else 'MISSING'}, "
                        f"mechanism={'present' if adr.mechanism else 'MISSING'}"
                    )

                # Check for missing symptom mapping (critical for P2)
                if not adr.symptom:
                    warnings.append(
                        f"Missing P2 symptom mapping for {adr.drug_name}/{adr.reaction}"
                    )

                # Validate LAB_VALUE modifiers have structured thresholds
                for mod in adr.contextual_modifiers:
                    if mod.modifier_type == "LAB_VALUE" and not mod.lab_threshold:
                        warnings.append(
                            f"LAB_VALUE modifier for {adr.drug_name} missing "
                            f"structured threshold: {mod.modifier_value}"
                        )

        return {
            "valid": len(issues) == 0,
            "issues": issues,
            "warnings": warnings,
            "total_issues": len(issues),
            "total_warnings": len(warnings),
        }


def create_extractor_from_env() -> KBFactExtractor:
    """
    Create a fact extractor using environment variables.

    Requires ANTHROPIC_API_KEY environment variable.

    Returns:
        Configured KBFactExtractor instance
    """
    import os

    if Anthropic is None:
        raise ImportError(
            "anthropic package not installed. "
            "Install with: pip install anthropic"
        )

    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        raise ValueError("ANTHROPIC_API_KEY environment variable not set")

    client = Anthropic(api_key=api_key)
    return KBFactExtractor(client=client)


# CLI interface for testing
if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="Extract clinical facts from guideline text"
    )
    parser.add_argument(
        "--target-kb",
        choices=["dosing", "safety", "monitoring", "contextual"],
        required=True,
        help="Target knowledge base",
    )
    parser.add_argument(
        "--text-file",
        type=Path,
        required=True,
        help="Path to markdown text file",
    )
    parser.add_argument(
        "--entities-file",
        type=Path,
        help="Path to GLiNER entities JSON file",
    )
    parser.add_argument(
        "--output",
        type=Path,
        help="Output JSON file path",
    )
    parser.add_argument(
        "--authority",
        default="KDIGO",
        help="Guideline authority (e.g., KDIGO, FDA)",
    )
    parser.add_argument(
        "--document",
        default="Unknown Guideline",
        help="Guideline document name",
    )

    args = parser.parse_args()

    # Load text
    text = args.text_file.read_text()

    # Load entities if provided
    entities = []
    if args.entities_file and args.entities_file.exists():
        entities = json.loads(args.entities_file.read_text())

    # Build context
    context = {
        "authority": args.authority,
        "document": args.document,
        "effective_date": date.today().isoformat(),
    }

    # Create extractor and run
    extractor = create_extractor_from_env()
    result = extractor.extract_facts(
        markdown_text=text,
        gliner_entities=entities,
        target_kb=args.target_kb,
        guideline_context=context,
    )

    # Validate
    validation = extractor.validate_extraction(result)
    print(f"\nValidation: {validation}")

    # Output
    output_json = result.model_dump(by_alias=True)

    if args.output:
        args.output.write_text(json.dumps(output_json, indent=2))
        print(f"\nOutput written to: {args.output}")
    else:
        print(json.dumps(output_json, indent=2))
