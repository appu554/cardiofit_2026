#!/usr/bin/env python3
"""
Vaidshala Phase 5: Constrained Atomiser

LLM extraction for genuine guideline gaps only.
All output: DRAFT status + mandatory SME review + confidence cap at 0.85

ONLY invoke when:
- No existing CQL in registry
- Table extraction failed (needs_llm_review=True)
- Complex sequencing/titration logic required

Usage:
    from atomiser import ConstrainedAtomiser, AtomiserConfig

    atomiser = ConstrainedAtomiser()
    result = atomiser.atomise(recommendation, extraction_type="titration")
"""

import json
import os
import re
from dataclasses import dataclass, field, asdict
from datetime import datetime
from typing import List, Dict, Optional, Any
from enum import Enum
from pathlib import Path

# Optional: Anthropic SDK for LLM calls
try:
    import anthropic
    HAS_ANTHROPIC = True
except ImportError:
    HAS_ANTHROPIC = False
    anthropic = None


class ExtractionType(Enum):
    """Types of extractions the Atomiser can perform."""
    TITRATION = "titration"           # Medication titration sequences
    EXCEPTION = "exception"           # Exception/contraindication logic
    BUNDLE = "bundle"                 # Care bundle with temporal sequence
    COMPLEX_CRITERIA = "criteria"     # Complex eligibility criteria
    MONITORING = "monitoring"         # Monitoring intervals and thresholds


class GovernanceStatus(Enum):
    """Governance status for atomised content."""
    DRAFT = "DRAFT"                   # LLM-generated, awaiting SME review
    PENDING_REVIEW = "PENDING_REVIEW" # Deterministic extraction, needs verification
    SME_APPROVED = "SME_APPROVED"     # Reviewed and approved by SME
    ACTIVE = "ACTIVE"                 # Approved and activated for clinical use


@dataclass
class AtomiserConfig:
    """Configuration for the Constrained Atomiser."""
    max_confidence: float = 0.85      # LLM cannot self-certify higher
    require_sme_review: bool = True   # All LLM output requires SME review
    model: str = "claude-sonnet-4-20250514"  # Model for extraction (sonnet for speed/cost)
    max_retries: int = 2              # Max retry attempts for failed extractions
    enable_caching: bool = True       # Cache successful extractions
    cache_dir: str = ".atomiser_cache"
    enable_llm: bool = False          # Enable actual LLM calls (requires ANTHROPIC_API_KEY)
    deterministic_threshold: float = 0.6  # Use LLM if deterministic confidence < this


@dataclass
class Population:
    """Target population for a recommendation."""
    condition: str
    qualifiers: List[str] = field(default_factory=list)
    exclusions: List[str] = field(default_factory=list)
    age_range: Optional[str] = None
    clinical_setting: Optional[str] = None


@dataclass
class Intervention:
    """Clinical intervention specification."""
    action: str  # RECOMMEND, CONSIDER, AVOID, CONTRAINDICATED
    medication_or_procedure: str
    dose_or_parameters: Optional[str] = None
    route: Optional[str] = None
    frequency: Optional[str] = None


@dataclass
class TitrationStep:
    """Single step in a titration sequence."""
    step: int
    dose: str
    duration: Optional[str] = None
    escalation_criteria: Optional[str] = None
    hold_criteria: List[str] = field(default_factory=list)
    max_dose: Optional[str] = None


@dataclass
class TemporalConstraint:
    """Temporal constraint for clinical actions."""
    step_id: str
    deadline_type: str  # RELATIVE, ABSOLUTE, RECURRING
    deadline_value: str
    deadline_from_event: Optional[str] = None
    iso8601_duration: Optional[str] = None


@dataclass
class EvidenceMetadata:
    """Evidence grading metadata."""
    cor: Optional[str] = None         # Class of Recommendation
    loe: Optional[str] = None         # Level of Evidence
    source_text: Optional[str] = None
    guideline_source: Optional[str] = None
    page_reference: Optional[int] = None


@dataclass
class AtomisedRecommendation:
    """
    Fully atomised clinical recommendation.

    This is the output format for KB-15 Evidence Engine integration.
    """
    recommendation_id: str
    extraction_type: str

    # Core clinical content
    population: Dict
    intervention: Dict
    titration_sequence: List[Dict] = field(default_factory=list)
    temporal_constraints: List[Dict] = field(default_factory=list)

    # Evidence and confidence
    evidence: Dict = field(default_factory=dict)
    confidence: float = 0.0
    uncertainty_flags: List[str] = field(default_factory=list)

    # Governance
    status: str = "DRAFT"
    requires_sme_review: bool = True
    sme_reviewer: Optional[str] = None
    sme_review_date: Optional[str] = None

    # Provenance
    provenance: Dict = field(default_factory=dict)

    def to_kb15_format(self) -> Dict:
        """Convert to KB-15 Evidence Engine format."""
        return {
            "recommendation_id": self.recommendation_id,
            "extraction_type": self.extraction_type,
            "clinical_content": {
                "population": self.population,
                "intervention": self.intervention,
                "titration_sequence": self.titration_sequence,
                "temporal_constraints": self.temporal_constraints
            },
            "evidence_envelope": {
                "class_of_recommendation": self.evidence.get("cor"),
                "level_of_evidence": self.evidence.get("loe"),
                "extraction_confidence": self.confidence,
                "uncertainty_flags": self.uncertainty_flags
            },
            "governance": {
                "status": self.status,
                "requires_sme_review": self.requires_sme_review,
                "sme_reviewer": self.sme_reviewer,
                "sme_review_date": self.sme_review_date,
                "activation_ready": self.status == "SME_APPROVED"
            },
            "provenance": self.provenance
        }


class ConstrainedAtomiser:
    """
    Constrained LLM extraction for guideline gaps.

    GOVERNANCE RULES:
    1. Confidence cap at 0.85 - LLM cannot self-certify higher
    2. All output is DRAFT status until SME review
    3. Uncertainty flags must be explicitly listed
    4. Full provenance tracking required

    ONLY invoke when:
    - No existing CQL in registry (verified by AtomiserRegistry)
    - Table extraction failed (needs_llm_review=True from Phase 4)
    - Complex sequencing/titration logic required
    """

    # JSON Schema for structured extraction
    EXTRACTION_SCHEMA = {
        "type": "object",
        "required": ["population", "intervention", "evidence", "confidence"],
        "properties": {
            "population": {
                "type": "object",
                "required": ["condition"],
                "properties": {
                    "condition": {"type": "string", "description": "Primary clinical condition"},
                    "qualifiers": {"type": "array", "items": {"type": "string"}},
                    "exclusions": {"type": "array", "items": {"type": "string"}},
                    "age_range": {"type": "string"},
                    "clinical_setting": {"type": "string"}
                }
            },
            "intervention": {
                "type": "object",
                "required": ["action", "medication_or_procedure"],
                "properties": {
                    "action": {
                        "type": "string",
                        "enum": ["RECOMMEND", "CONSIDER", "AVOID", "CONTRAINDICATED"]
                    },
                    "medication_or_procedure": {"type": "string"},
                    "dose_or_parameters": {"type": "string"},
                    "route": {"type": "string"},
                    "frequency": {"type": "string"}
                }
            },
            "titration_sequence": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "step": {"type": "integer", "minimum": 1},
                        "dose": {"type": "string"},
                        "duration": {"type": "string"},
                        "escalation_criteria": {"type": "string"},
                        "hold_criteria": {"type": "array", "items": {"type": "string"}},
                        "max_dose": {"type": "string"}
                    }
                }
            },
            "temporal_constraints": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "step_id": {"type": "string"},
                        "deadline_type": {
                            "type": "string",
                            "enum": ["RELATIVE", "ABSOLUTE", "RECURRING"]
                        },
                        "deadline_value": {"type": "string"},
                        "deadline_from_event": {"type": "string"}
                    }
                }
            },
            "evidence": {
                "type": "object",
                "properties": {
                    "cor": {"type": "string", "description": "Class of Recommendation (I, IIa, IIb, III)"},
                    "loe": {"type": "string", "description": "Level of Evidence (A, B-R, B-NR, C-LD, C-EO)"},
                    "source_text": {"type": "string"}
                }
            },
            "confidence": {
                "type": "number",
                "minimum": 0,
                "maximum": 0.85,
                "description": "Extraction confidence (capped at 0.85)"
            },
            "uncertainty_flags": {
                "type": "array",
                "items": {"type": "string"},
                "description": "List of uncertainties or ambiguities in extraction"
            }
        }
    }

    def __init__(self, config: AtomiserConfig = None):
        """
        Initialize the Constrained Atomiser.

        Args:
            config: AtomiserConfig with extraction parameters
        """
        self.config = config or AtomiserConfig()
        self.extraction_count = 0
        self.cache = {}
        self.client = None

        # Create cache directory if enabled
        if self.config.enable_caching:
            Path(self.config.cache_dir).mkdir(exist_ok=True)

        # Initialize Anthropic client if LLM enabled and SDK available
        if self.config.enable_llm:
            if not HAS_ANTHROPIC:
                raise ImportError(
                    "Anthropic SDK not installed. Run: pip install anthropic"
                )
            api_key = os.environ.get("ANTHROPIC_API_KEY")
            if not api_key:
                raise ValueError(
                    "ANTHROPIC_API_KEY environment variable not set. "
                    "Set it or disable LLM: config.enable_llm=False"
                )
            self.client = anthropic.Anthropic(api_key=api_key)

    def atomise(
        self,
        text_chunk: str,
        extraction_type: str,
        guideline_source: str = "UNKNOWN",
        existing_evidence: Dict = None
    ) -> AtomisedRecommendation:
        """
        Atomise a text chunk into structured clinical recommendation.

        Args:
            text_chunk: Specific text to atomise (NOT entire guideline)
            extraction_type: Type of extraction (titration, exception, bundle, etc.)
            guideline_source: Source guideline identifier
            existing_evidence: Any COR/LOE already extracted by Phase 4

        Returns:
            AtomisedRecommendation with DRAFT status
        """
        self.extraction_count += 1
        recommendation_id = f"{guideline_source}-ATOM-{self.extraction_count:04d}"

        # Check cache first
        cache_key = self._compute_cache_key(text_chunk, extraction_type)
        if self.config.enable_caching and cache_key in self.cache:
            cached = self.cache[cache_key]
            cached.recommendation_id = recommendation_id
            return cached

        # First, try deterministic extraction
        extraction = self._deterministic_extraction(text_chunk, extraction_type, existing_evidence)

        # If confidence is below threshold and LLM is enabled, use LLM
        if (extraction.get("confidence", 0) < self.config.deterministic_threshold
                and self.config.enable_llm and self.client):
            llm_extraction = self._llm_extraction(text_chunk, extraction_type, existing_evidence)
            if llm_extraction:
                extraction = llm_extraction
                extraction["used_llm"] = True

        # Apply governance constraints (confidence cap, DRAFT status)
        extraction = self._apply_governance(extraction)

        # Build result
        result = AtomisedRecommendation(
            recommendation_id=recommendation_id,
            extraction_type=extraction_type,
            population=extraction.get("population", {}),
            intervention=extraction.get("intervention", {}),
            titration_sequence=extraction.get("titration_sequence", []),
            temporal_constraints=extraction.get("temporal_constraints", []),
            evidence=extraction.get("evidence", existing_evidence or {}),
            confidence=extraction.get("confidence", 0.5),
            uncertainty_flags=extraction.get("uncertainty_flags", []),
            status=GovernanceStatus.DRAFT.value,
            requires_sme_review=True,
            provenance={
                "extraction_method": "ATOMISER_LLM" if extraction.get("used_llm") else "ATOMISER_DETERMINISTIC",
                "model": self.config.model if extraction.get("used_llm") else None,
                "source_text": text_chunk[:500] + "..." if len(text_chunk) > 500 else text_chunk,
                "guideline_source": guideline_source,
                "extraction_timestamp": datetime.utcnow().isoformat() + "Z",
                "schema_version": "1.0"
            }
        )

        # Cache result
        if self.config.enable_caching:
            self.cache[cache_key] = result

        return result

    def _deterministic_extraction(
        self,
        text: str,
        extraction_type: str,
        existing_evidence: Dict = None
    ) -> Dict:
        """
        Perform deterministic extraction where possible.

        This handles common patterns without LLM:
        - Simple dose escalations
        - Clear temporal constraints
        - Explicit eligibility criteria
        """
        extraction = {
            "population": {},
            "intervention": {},
            "titration_sequence": [],
            "temporal_constraints": [],
            "evidence": existing_evidence or {},
            "confidence": 0.6,
            "uncertainty_flags": [],
            "used_llm": False
        }

        # Extract population
        extraction["population"] = self._extract_population(text)

        # Extract intervention
        extraction["intervention"] = self._extract_intervention(text)

        # Extract titration if applicable
        if extraction_type == ExtractionType.TITRATION.value:
            extraction["titration_sequence"] = self._extract_titration(text)
            if extraction["titration_sequence"]:
                extraction["confidence"] = 0.7

        # Extract temporal constraints
        extraction["temporal_constraints"] = self._extract_temporal(text)

        # Add uncertainty flags
        extraction["uncertainty_flags"] = self._detect_uncertainties(text, extraction)

        # Adjust confidence based on completeness
        if not extraction["population"].get("condition"):
            extraction["uncertainty_flags"].append("Population condition unclear")
            extraction["confidence"] *= 0.8

        if not extraction["intervention"].get("medication_or_procedure"):
            extraction["uncertainty_flags"].append("Intervention unclear")
            extraction["confidence"] *= 0.8

        return extraction

    def _llm_extraction(
        self,
        text: str,
        extraction_type: str,
        existing_evidence: Dict = None
    ) -> Optional[Dict]:
        """
        Call Anthropic API for LLM-assisted extraction.

        This is the "Last Resort" when deterministic extraction fails.
        All output is constrained by governance rules.

        Args:
            text: Text to extract from
            extraction_type: Type of extraction
            existing_evidence: Pre-extracted evidence from Phase 4

        Returns:
            Extraction dict or None if LLM call fails
        """
        if not self.client:
            return None

        prompt = self._build_extraction_prompt(text, extraction_type, existing_evidence)

        try:
            # Call Anthropic API with structured output
            response = self.client.messages.create(
                model=self.config.model,
                max_tokens=2000,
                messages=[
                    {
                        "role": "user",
                        "content": prompt
                    }
                ],
                system="""You are a clinical informatics expert extracting structured data from clinical guidelines.
Your extraction must be precise, evidence-based, and acknowledge any uncertainties.
CRITICAL: Your confidence score MUST NOT exceed 0.85 - this is a governance constraint.
Respond with valid JSON only, following the provided schema exactly."""
            )

            # Parse response
            response_text = response.content[0].text.strip()

            # Handle markdown code blocks
            if response_text.startswith("```"):
                # Extract JSON from code block
                lines = response_text.split("\n")
                json_lines = []
                in_json = False
                for line in lines:
                    if line.startswith("```json"):
                        in_json = True
                        continue
                    elif line.startswith("```"):
                        in_json = False
                        continue
                    if in_json:
                        json_lines.append(line)
                response_text = "\n".join(json_lines)

            extraction = json.loads(response_text)

            # Merge with existing evidence
            if existing_evidence:
                extraction.setdefault("evidence", {}).update(existing_evidence)

            extraction["used_llm"] = True
            extraction["llm_model"] = self.config.model

            return extraction

        except json.JSONDecodeError as e:
            # LLM returned invalid JSON
            return {
                "used_llm": True,
                "llm_error": f"Invalid JSON response: {str(e)}",
                "confidence": 0.3,
                "uncertainty_flags": ["LLM extraction failed - invalid response format"],
                "population": {},
                "intervention": {},
                "evidence": existing_evidence or {}
            }
        except Exception as e:
            # API error or other failure
            return {
                "used_llm": True,
                "llm_error": str(e),
                "confidence": 0.3,
                "uncertainty_flags": [f"LLM extraction failed: {str(e)}"],
                "population": {},
                "intervention": {},
                "evidence": existing_evidence or {}
            }

    def _extract_population(self, text: str) -> Dict:
        """Extract population criteria from text."""
        population = {
            "condition": "",
            "qualifiers": [],
            "exclusions": []
        }

        # Common condition patterns
        condition_patterns = [
            r'(?:patients?|adults?|individuals?)\s+with\s+([^,\.]+)',
            r'(?:in|for)\s+(?:patients?|adults?)\s+(?:with|who\s+have)\s+([^,\.]+)',
            r'(?:diagnosis|diagnosed)\s+(?:of|with)\s+([^,\.]+)'
        ]

        for pattern in condition_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                population["condition"] = match.group(1).strip()
                break

        # Exclusion patterns
        exclusion_patterns = [
            r'(?:except|excluding|contraindicated\s+in|not\s+for)\s+([^\.]+)',
            r'(?:should\s+not|do\s+not)\s+(?:use|give|administer)\s+(?:in|to)\s+([^\.]+)'
        ]

        for pattern in exclusion_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            population["exclusions"].extend([m.strip() for m in matches])

        # Qualifier patterns (severity, stage, etc.)
        qualifier_patterns = [
            r'(severe|moderate|mild)\s+',
            r'(stage\s+[IViv123]+)',
            r'(NYHA\s+class\s+[IViv]+)',
            r'(EF\s*[<>≤≥]\s*\d+%?)'
        ]

        for pattern in qualifier_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            population["qualifiers"].extend([m.strip() for m in matches])

        return population

    def _extract_intervention(self, text: str) -> Dict:
        """Extract intervention details from text."""
        intervention = {
            "action": "RECOMMEND",
            "medication_or_procedure": "",
            "dose_or_parameters": None,
            "route": None,
            "frequency": None
        }

        # Action patterns
        if re.search(r'\b(recommend|should\s+(?:be\s+)?(?:given|initiated|started))\b', text, re.IGNORECASE):
            intervention["action"] = "RECOMMEND"
        elif re.search(r'\b(consider|may\s+be|can\s+be)\b', text, re.IGNORECASE):
            intervention["action"] = "CONSIDER"
        elif re.search(r'\b(avoid|should\s+not|do\s+not)\b', text, re.IGNORECASE):
            intervention["action"] = "AVOID"
        elif re.search(r'\b(contraindicated|never|prohibited)\b', text, re.IGNORECASE):
            intervention["action"] = "CONTRAINDICATED"

        # Medication/procedure patterns
        med_patterns = [
            r'(?:administer|give|initiate|start)\s+([A-Za-z][A-Za-z0-9\-]+(?:\s+[A-Za-z]+)?)',
            r'(?:treatment\s+with|therapy\s+with)\s+([A-Za-z][A-Za-z0-9\-]+)',
            r'(ARNi|ACEi|ARB|beta[- ]?blocker|SGLT2i?|MRA|statin|anticoagulant)',
        ]

        for pattern in med_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                intervention["medication_or_procedure"] = match.group(1).strip()
                break

        # Dose patterns
        dose_patterns = [
            r'(\d+(?:\.\d+)?\s*(?:mg|mcg|g|mL|units?|IU)(?:\s*/\s*(?:kg|day|dose))?)',
            r'(target\s+dose[:\s]+[^\.]+)',
            r'(starting\s+dose[:\s]+[^\.]+)'
        ]

        for pattern in dose_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                intervention["dose_or_parameters"] = match.group(1).strip()
                break

        # Route patterns
        route_match = re.search(r'\b(oral(?:ly)?|IV|intravenous(?:ly)?|subcutaneous(?:ly)?|IM|intramuscular(?:ly)?)\b', text, re.IGNORECASE)
        if route_match:
            intervention["route"] = route_match.group(1)

        # Frequency patterns
        freq_match = re.search(r'\b(once\s+daily|twice\s+daily|BID|TID|QID|q\d+h|every\s+\d+\s+hours?|daily|weekly)\b', text, re.IGNORECASE)
        if freq_match:
            intervention["frequency"] = freq_match.group(1)

        return intervention

    def _extract_titration(self, text: str) -> List[Dict]:
        """Extract titration sequence from text."""
        titration = []

        # Look for step patterns
        step_patterns = [
            r'(?:step|week|phase)\s*(\d+)[:\s]+([^\.]+)',
            r'(?:initial(?:ly)?|start(?:ing)?)[:\s]+(\d+(?:\.\d+)?\s*mg)',
            r'(?:increase|titrate)\s+to\s+(\d+(?:\.\d+)?\s*mg)',
            r'(?:target|maximum|max)\s+(?:dose)?[:\s]+(\d+(?:\.\d+)?\s*mg)'
        ]

        step_num = 1
        for pattern in step_patterns:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                step = {
                    "step": step_num,
                    "dose": match.group(1) if len(match.groups()) == 1 else match.group(2),
                    "duration": None,
                    "escalation_criteria": None,
                    "hold_criteria": []
                }

                # Look for duration near this match
                context = text[max(0, match.start()-50):min(len(text), match.end()+100)]
                duration_match = re.search(r'(?:for|after|every)\s+(\d+)\s*(days?|weeks?|months?)', context, re.IGNORECASE)
                if duration_match:
                    step["duration"] = f"{duration_match.group(1)} {duration_match.group(2)}"

                titration.append(step)
                step_num += 1

        # Look for hold criteria
        hold_patterns = [
            r'hold\s+if\s+([^\.]+)',
            r'do\s+not\s+(?:increase|escalate)\s+if\s+([^\.]+)',
            r'reduce\s+dose\s+if\s+([^\.]+)'
        ]

        for pattern in hold_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            for step in titration:
                step["hold_criteria"].extend([m.strip() for m in matches])

        return titration

    def _extract_temporal(self, text: str) -> List[Dict]:
        """Extract temporal constraints from text."""
        constraints = []

        patterns = [
            (r'within\s+(\d+)\s*(hours?|days?|weeks?)', 'RELATIVE'),
            (r'every\s+(\d+)\s*(hours?|days?|weeks?|months?)', 'RECURRING'),
            (r'after\s+(\d+)\s*(hours?|days?|weeks?)', 'RELATIVE'),
            (r'before\s+([a-zA-Z][a-zA-Z\s]+)', 'RELATIVE'),
        ]

        constraint_id = 1
        for pattern, deadline_type in patterns:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                constraint = {
                    "step_id": f"TC-{constraint_id:03d}",
                    "deadline_type": deadline_type,
                    "deadline_value": f"{match.group(1)} {match.group(2)}" if len(match.groups()) > 1 else match.group(1),
                    "deadline_from_event": None
                }

                # Convert to ISO 8601 if possible
                if deadline_type in ('RELATIVE', 'RECURRING') and len(match.groups()) > 1:
                    constraint["iso8601_duration"] = self._to_iso8601(match.group(1), match.group(2))

                constraints.append(constraint)
                constraint_id += 1

        return constraints

    def _to_iso8601(self, value: str, unit: str) -> Optional[str]:
        """Convert value/unit to ISO 8601 duration."""
        try:
            num = int(value)
        except ValueError:
            return None

        unit_map = {
            'hour': 'H', 'hours': 'H',
            'day': 'D', 'days': 'D',
            'week': 'W', 'weeks': 'W',
            'month': 'M', 'months': 'M'
        }

        iso_unit = unit_map.get(unit.lower())
        if not iso_unit:
            return None

        if iso_unit == 'H':
            return f"PT{num}H"
        else:
            return f"P{num}{iso_unit}"

    def _detect_uncertainties(self, text: str, extraction: Dict) -> List[str]:
        """Detect uncertainties and ambiguities in the extraction."""
        uncertainties = []

        # Check for hedging language
        hedging = [
            (r'\b(may|might|could|possibly|potentially)\b', "Contains hedging language"),
            (r'\b(unclear|uncertain|unknown|limited\s+evidence)\b', "Evidence uncertainty noted"),
            (r'\b(expert\s+opinion|consensus)\b', "Based on expert opinion"),
            (r'\b(consider|individualize)\b', "Requires clinical judgment"),
        ]

        for pattern, flag in hedging:
            if re.search(pattern, text, re.IGNORECASE):
                uncertainties.append(flag)

        # Check extraction completeness
        if not extraction.get("titration_sequence") and "titrat" in text.lower():
            uncertainties.append("Titration mentioned but not fully extracted")

        if extraction.get("intervention", {}).get("action") == "CONSIDER":
            uncertainties.append("Conditional recommendation - clinical judgment required")

        return uncertainties

    def _apply_governance(self, extraction: Dict) -> Dict:
        """Apply governance constraints to extraction."""

        # Cap confidence at 0.85
        if extraction.get("confidence", 0) > self.config.max_confidence:
            extraction["confidence"] = self.config.max_confidence
            extraction.setdefault("uncertainty_flags", []).append("Confidence capped at 0.85 per governance rules")

        # Ensure DRAFT status
        extraction["status"] = GovernanceStatus.DRAFT.value
        extraction["requires_sme_review"] = True

        return extraction

    def _build_extraction_prompt(
        self,
        text: str,
        extraction_type: str,
        existing_evidence: Dict = None
    ) -> str:
        """Build LLM extraction prompt with schema."""

        prompt = f"""Extract structured clinical recommendation from the following text.

EXTRACTION TYPE: {extraction_type}

TEXT:
{text}

EXISTING EVIDENCE (from Phase 4):
{json.dumps(existing_evidence or {}, indent=2)}

OUTPUT REQUIREMENTS:
1. Follow the JSON schema exactly
2. Confidence MUST be between 0 and 0.85
3. List ALL uncertainties and ambiguities
4. If information is missing, note it in uncertainty_flags
5. Use standard clinical terminology

OUTPUT SCHEMA:
{json.dumps(self.EXTRACTION_SCHEMA, indent=2)}

Respond with valid JSON only.
"""
        return prompt

    def _compute_cache_key(self, text: str, extraction_type: str) -> str:
        """Compute cache key for extraction."""
        import hashlib
        content = f"{extraction_type}:{text[:500]}"
        return hashlib.md5(content.encode()).hexdigest()

    def validate_extraction(self, extraction: AtomisedRecommendation) -> tuple[bool, List[str]]:
        """
        Validate an atomised recommendation against schema.

        Returns:
            Tuple of (is_valid, list of validation errors)
        """
        errors = []

        # Check required fields
        if not extraction.population.get("condition"):
            errors.append("Missing population condition")

        if not extraction.intervention.get("medication_or_procedure"):
            errors.append("Missing intervention")

        # Check governance
        if extraction.status != GovernanceStatus.DRAFT.value:
            errors.append("Non-DRAFT status not allowed for LLM extractions")

        if not extraction.requires_sme_review:
            errors.append("SME review must be required")

        if extraction.confidence > self.config.max_confidence:
            errors.append(f"Confidence exceeds maximum {self.config.max_confidence}")

        return len(errors) == 0, errors

    def get_stats(self) -> Dict[str, Any]:
        """Get atomiser statistics."""
        return {
            "total_extractions": self.extraction_count,
            "cached_extractions": len(self.cache),
            "llm_enabled": self.config.enable_llm,
            "llm_available": self.client is not None,
            "config": {
                "max_confidence": self.config.max_confidence,
                "require_sme_review": self.config.require_sme_review,
                "model": self.config.model,
                "deterministic_threshold": self.config.deterministic_threshold
            }
        }


def main():
    """Demo the Constrained Atomiser."""

    print("=" * 60)
    print("Phase 5: Constrained Atomiser Demo")
    print("=" * 60)

    # Show LLM configuration status
    print(f"\n📦 Anthropic SDK installed: {HAS_ANTHROPIC}")
    print(f"🔑 ANTHROPIC_API_KEY set: {bool(os.environ.get('ANTHROPIC_API_KEY'))}")
    print("\n💡 To enable LLM extraction:")
    print("   1. pip install anthropic")
    print("   2. export ANTHROPIC_API_KEY='your-key'")
    print("   3. AtomiserConfig(enable_llm=True)")
    print("-" * 60)

    # Sample text requiring atomisation
    sample_text = """
    For patients with HFrEF (LVEF ≤40%), initiate beta-blocker therapy with
    carvedilol starting at 3.125 mg twice daily. Titrate every 2 weeks as
    tolerated to target dose of 25 mg twice daily (50 mg twice daily for
    patients >85 kg). Hold if heart rate <50 bpm or symptomatic hypotension.
    Do not initiate in patients with acute decompensation.

    Class I, Level A evidence.
    """

    atomiser = ConstrainedAtomiser()

    result = atomiser.atomise(
        text_chunk=sample_text,
        extraction_type="titration",
        guideline_source="ACC-AHA-HF-2022",
        existing_evidence={"cor": "I", "loe": "A"}
    )

    print("\n--- Atomised Result ---")
    print(f"ID: {result.recommendation_id}")
    print(f"Type: {result.extraction_type}")
    print(f"Status: {result.status}")
    print(f"Confidence: {result.confidence}")
    print(f"Requires SME Review: {result.requires_sme_review}")

    print("\n--- Population ---")
    print(json.dumps(result.population, indent=2))

    print("\n--- Intervention ---")
    print(json.dumps(result.intervention, indent=2))

    print("\n--- Titration Sequence ---")
    print(json.dumps(result.titration_sequence, indent=2))

    print("\n--- Uncertainty Flags ---")
    for flag in result.uncertainty_flags:
        print(f"  - {flag}")

    print("\n--- KB-15 Format ---")
    print(json.dumps(result.to_kb15_format(), indent=2))

    # Validate
    is_valid, errors = atomiser.validate_extraction(result)
    print(f"\n--- Validation: {'PASSED' if is_valid else 'FAILED'} ---")
    for error in errors:
        print(f"  ERROR: {error}")

    print("\n--- Stats ---")
    print(json.dumps(atomiser.get_stats(), indent=2))


if __name__ == "__main__":
    main()
