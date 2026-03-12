"""
L2: Clinical NER Extraction using GLiNER.

This module provides Named Entity Recognition for clinical text using
GLiNER (Generalist and Lightweight NER), a zero-shot NER model that
accepts custom descriptive labels at inference time.

Key advantage of GLiNER over traditional NER:
- Zero-shot: No need for pre-trained entity types
- Descriptive labels: Uses rich semantic descriptions like
  "medication name or active pharmaceutical ingredient" instead of "drug"
- Domain flexibility: Works well on clinical/biomedical text with custom labels

Pipeline Position:
    L1 (Marker PDF) → L2 (GLiNER NER) → L3 (Claude Structured) → L4 (KB-7)

Usage:
    from extraction.gliner.extractor import ClinicalNERExtractor

    extractor = ClinicalNERExtractor(threshold=0.6)
    result = extractor.extract_entities(
        text="Metformin is contraindicated when eGFR falls below 30 mL/min/1.73m2",
    )

    # Access entities
    for entity in result.entities:
        print(f"{entity.text} -> {entity.label} ({entity.score:.2f})")
"""

import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional, Literal, List, Dict
from datetime import datetime, timezone

from .entity_types import (
    ClinicalEntityTypes,
    EntityType,
    get_labels_for_kb,
    get_all_clinical_labels,
)


@dataclass
class Entity:
    """A recognized clinical entity with span information."""
    text: str  # The matched text
    label: str  # Entity type label
    start: int  # Character start position
    end: int  # Character end position
    score: float  # Confidence score (0-1)

    # Optional enrichment fields (populated by L4)
    rxnorm_code: Optional[str] = None
    loinc_code: Optional[str] = None
    snomed_code: Optional[str] = None
    icd10_code: Optional[str] = None

    # Normalization
    normalized_text: Optional[str] = None  # Canonical form

    def to_dict(self) -> dict:
        result = {
            "text": self.text,
            "label": self.label,
            "start": self.start,
            "end": self.end,
            "score": self.score,
        }
        if self.rxnorm_code:
            result["rxnorm_code"] = self.rxnorm_code
        if self.loinc_code:
            result["loinc_code"] = self.loinc_code
        if self.snomed_code:
            result["snomed_code"] = self.snomed_code
        if self.icd10_code:
            result["icd10_code"] = self.icd10_code
        if self.normalized_text:
            result["normalized_text"] = self.normalized_text
        return result


@dataclass
class NERResult:
    """Complete NER extraction result."""
    text: str
    entities: List[Entity]
    labels_used: List[str]
    extraction_timestamp: str
    model_name: str
    model_version: str

    def to_dict(self) -> dict:
        return {
            "text": self.text,
            "entities": [e.to_dict() for e in self.entities],
            "labels_used": self.labels_used,
            "extraction_timestamp": self.extraction_timestamp,
            "model_name": self.model_name,
            "model_version": self.model_version,
            "entity_count": len(self.entities),
            "unique_labels": list(set(e.label for e in self.entities)),
        }

    def to_json(self, indent: int = 2) -> str:
        return json.dumps(self.to_dict(), indent=indent)

    def get_entities_by_label(self, label: str) -> List[Entity]:
        """Filter entities by label."""
        return [e for e in self.entities if e.label == label]

    def get_entities_by_kb(self, kb: Literal["KB-1", "KB-4", "KB-16"]) -> List[Entity]:
        """Get entities destined for a specific KB."""
        kb_labels = ClinicalEntityTypes.get_labels_by_kb(kb)
        return [e for e in self.entities if e.label in kb_labels]

    def to_gliner_format(self) -> List[dict]:
        """Convert to format expected by L3 Claude extraction."""
        return [
            {
                "entity": e.text,
                "type": e.label,
                "confidence": e.score,
                "span": [e.start, e.end],
                "codes": {
                    k: v for k, v in [
                        ("rxnorm", e.rxnorm_code),
                        ("loinc", e.loinc_code),
                        ("snomed", e.snomed_code),
                        ("icd10", e.icd10_code),
                    ] if v is not None
                },
            }
            for e in self.entities
        ]


class ClinicalNERExtractor:
    """
    L2 Clinical NER Extractor using GLiNER.

    GLiNER is a zero-shot NER model that:
    - Accepts custom descriptive labels at inference time
    - Works well on clinical/biomedical text
    - Provides confidence scores for each entity

    The key innovation is using DESCRIPTIVE labels that give GLiNER
    semantic context about what to find, rather than terse labels.

    Recommended Model: urchade/gliner_mediumv2.1
    """

    DEFAULT_MODEL = "urchade/gliner_mediumv2.1"
    VERSION = "2.2.0"  # Added span-splitting + OCR correction
    DEFAULT_THRESHOLD = 0.6  # Confidence threshold

    # ═══════════════════════════════════════════════════════════════════════
    # OCR CORRECTION PATTERNS
    # Common OCR errors in clinical PDF extraction
    # ═══════════════════════════════════════════════════════════════════════

    OCR_CORRECTIONS: Dict[str, str] = {
        # Common letter substitutions
        "OUICK": "QUICK",
        "Ouick": "Quick",
        "ouick": "quick",
        "eGER": "eGFR",
        "EGFR": "eGFR",
        "rnetformin": "metformin",
        "Rnetformin": "Metformin",
        "rnL/min": "mL/min",
        "1.73rn": "1.73m",
        "rng/dL": "mg/dL",
        "rnEq/L": "mEq/L",
        "CrCI": "CrCl",
        "HbAlc": "HbA1c",
        "HbA 1c": "HbA1c",
        "Hb A1c": "HbA1c",
        # Number-letter confusion
        "eGFR3O": "eGFR30",
        "eGFR2O": "eGFR20",
        "eGFR6O": "eGFR60",
        "eGFR45": "eGFR45",
        # Ligature issues
        "ﬁnerenone": "finerenone",
        "ﬂuid": "fluid",
        "ﬁrst": "first",
    }

    # Known drug ingredients for span splitting
    KNOWN_DRUG_INGREDIENTS = {
        "metformin", "dapagliflozin", "empagliflozin", "canagliflozin",
        "ertugliflozin", "sotagliflozin", "finerenone", "lisinopril",
        "enalapril", "ramipril", "perindopril", "losartan", "valsartan",
        "irbesartan", "candesartan", "telmisartan", "olmesartan",
        "spironolactone", "eplerenone", "furosemide", "bumetanide",
        "hydrochlorothiazide", "chlorthalidone", "indapamide",
        "semaglutide", "liraglutide", "dulaglutide", "exenatide",
        "sitagliptin", "linagliptin", "saxagliptin", "alogliptin",
        "pioglitazone", "rosiglitazone", "glyburide", "glipizide",
        "glimepiride", "insulin", "amlodipine", "nifedipine",
        "diltiazem", "verapamil", "atenolol", "metoprolol",
        "carvedilol", "bisoprolol", "atorvastatin", "rosuvastatin",
    }

    # Known drug class abbreviations for span splitting
    KNOWN_CLASS_ABBREVIATIONS = {
        "sglt2i", "sglt-2i", "acei", "ace-i", "arb", "arbs",
        "mra", "mras", "ns-mra", "rasi", "ras-i", "dpp4i", "dpp-4i",
        "glp-1", "glp1", "glp-1ra", "tzd", "tzds", "su", "sus",
    }

    # ═══════════════════════════════════════════════════════════════════════
    # DESCRIPTIVE LABELS for GLiNER
    # These rich descriptions help GLiNER understand what to extract
    # ═══════════════════════════════════════════════════════════════════════

    DESCRIPTIVE_LABELS: Dict[str, str] = {
        # KB-1: Drug Dosing
        "drug_ingredient": "medication name or active pharmaceutical ingredient",
        "drug_class": "pharmacological drug class or medication category",
        "drug_product": "brand name or commercial medication product",
        "dose_value": "numeric dosage amount",
        "dose_unit": "dosage unit of measurement",
        "dose_frequency": "medication dosing frequency or schedule",
        "dose_route": "route of drug administration",
        "dose_adjustment": "instruction for dose modification or change",

        # KB-4: Patient Safety
        "condition": "disease, medical condition, or clinical disorder",
        "contraindication_marker": "word indicating contraindication or avoidance",
        "severity": "severity or urgency level indicator",
        "adverse_event": "side effect or adverse drug reaction",
        "population": "patient population or demographic group",
        "caution_marker": "word indicating caution or warning",

        # KB-16: Lab Monitoring
        "lab_test": "laboratory test or clinical measurement",
        "lab_value": "numeric lab result or threshold value",
        "lab_unit": "unit of measurement for lab values",
        "monitoring_frequency": "how often to monitor or check",
        "monitoring_action": "action based on monitoring result",
        "baseline_marker": "indicator of baseline requirement",

        # SHARED: Cross-KB
        "egfr_threshold": "kidney function eGFR or creatinine clearance threshold",
        "recommendation_level": "evidence strength or recommendation grade",
        "guideline_reference": "reference to guideline section or recommendation",
        "temporal_marker": "time-related indicator or duration",
    }

    # Reverse mapping: descriptive label -> short label
    LABEL_REVERSE_MAP: Dict[str, str] = {v: k for k, v in DESCRIPTIVE_LABELS.items()}

    # Drug class patterns for post-processing reclassification
    DRUG_CLASSES = {
        "sglt2 inhibitor", "sglt2i", "sglt2-i", "sglt-2 inhibitor", "sglt-2i",
        "sglt2 inhibitors", "sglt-2 inhibitors",
        "ace inhibitor", "acei", "ace-i", "ace inhibitors",
        "arb", "arbs", "angiotensin receptor blocker", "angiotensin receptor blockers",
        "glp-1 agonist", "glp-1 ra", "glp1-ra", "glp-1 receptor agonist",
        "glp-1 agonists", "glp-1 receptor agonists",
        "mra", "mras", "mineralocorticoid receptor antagonist",
        "mineralocorticoid receptor antagonists",
        "dpp-4 inhibitor", "dpp4i", "dpp-4 inhibitors",
        "sulfonylurea", "sulfonylureas",
        "biguanide", "biguanides",
        "thiazolidinedione", "thiazolidinediones", "tzd", "tzds",
        "rasi", "ras inhibitor", "ras inhibitors",
        "beta blocker", "beta blockers", "beta-blocker", "beta-blockers",
        "calcium channel blocker", "calcium channel blockers",
        "diuretic", "diuretics",
        "statin", "statins",
        "loop diuretic", "loop diuretics",
        "thiazide", "thiazides",
        "nsaid", "nsaids",
    }

    def __init__(
        self,
        model_name: Optional[str] = None,
        device: str = "cpu",
        threshold: float = DEFAULT_THRESHOLD,
    ):
        """
        Initialize the GLiNER extractor.

        Args:
            model_name: GLiNER model to use (default: urchade/gliner_mediumv2.1)
            device: Device for inference ("cpu", "cuda", "mps")
            threshold: Minimum confidence threshold for entities (default: 0.6)
        """
        self.model_name = model_name or self.DEFAULT_MODEL
        self.device = device
        self.threshold = threshold
        self._model = None
        self._model_loaded = False

    def _load_model(self):
        """Lazy-load the GLiNER model."""
        if self._model_loaded:
            return

        try:
            from gliner import GLiNER

            self._model = GLiNER.from_pretrained(self.model_name)
            if self.device == "cuda":
                self._model = self._model.to("cuda")
            elif self.device == "mps":
                self._model = self._model.to("mps")
            self._model_loaded = True
        except ImportError:
            # GLiNER not installed, will use regex fallback
            self._model = None
            self._model_loaded = True
        except Exception as e:
            print(f"⚠️ Failed to load GLiNER model: {e}")
            self._model = None
            self._model_loaded = True

    def _apply_ocr_corrections(self, text: str) -> str:
        """
        Apply OCR corrections to text before extraction.

        Fixes common OCR errors in clinical PDFs:
        - Letter substitutions (rn → m, O → 0)
        - Ligature issues (ﬁ → fi)
        - Header-specific typos (OUICK → QUICK)
        """
        corrected = text
        for wrong, right in self.OCR_CORRECTIONS.items():
            corrected = corrected.replace(wrong, right)
        return corrected

    def _split_multi_drug_spans(self, entities: List[Entity]) -> List[Entity]:
        """
        Split entities containing multiple drugs/classes into separate entities.

        GLiNER often merges adjacent tokens like "metformin SGLT2i RASi" into
        a single span. This post-processor splits them based on known drug
        names and class abbreviations.

        Example:
            Input:  "metformin SGLT2i RASi" (drug_ingredient, 0.41)
            Output: ["metformin" (drug_ingredient, 0.85),
                     "SGLT2i" (drug_class, 0.85),
                     "RASi" (drug_class, 0.85)]
        """
        refined = []

        for ent in entities:
            # Only process drug-related entities
            if ent.label not in ("drug_ingredient", "drug_class", "drug_product"):
                refined.append(ent)
                continue

            # Check if span contains multiple known drugs/classes
            text = ent.text.strip()
            tokens = re.split(r'[\s,/]+', text)

            # If only one token, keep as-is
            if len(tokens) <= 1:
                refined.append(ent)
                continue

            # Check if multiple known entities are present
            found_entities = []
            for token in tokens:
                token_lower = token.lower()
                token_clean = token_lower.rstrip('s')  # Handle plurals

                # Check if it's a known drug ingredient
                if token_lower in self.KNOWN_DRUG_INGREDIENTS:
                    found_entities.append((token, "drug_ingredient"))
                # Check if it's a known class abbreviation
                elif token_lower in self.KNOWN_CLASS_ABBREVIATIONS:
                    found_entities.append((token, "drug_class"))
                # Check in DRUG_CLASSES set (full names)
                elif token_lower in self.DRUG_CLASSES:
                    found_entities.append((token, "drug_class"))

            # If we found multiple known entities, split them
            if len(found_entities) > 1:
                # Calculate position offset for each split entity
                offset = ent.start
                for drug_text, drug_label in found_entities:
                    # Find position in original text
                    try:
                        pos = text.lower().find(drug_text.lower())
                        if pos >= 0:
                            refined.append(Entity(
                                text=drug_text,
                                label=drug_label,
                                start=ent.start + pos,
                                end=ent.start + pos + len(drug_text),
                                score=0.85,  # Boost confidence after splitting
                            ))
                    except:
                        pass
            elif len(found_entities) == 1:
                # Single known entity, use it with proper label
                drug_text, drug_label = found_entities[0]
                refined.append(Entity(
                    text=drug_text,
                    label=drug_label,
                    start=ent.start,
                    end=ent.start + len(drug_text),
                    score=0.85,
                ))
            else:
                # No known entities found, keep original
                refined.append(ent)

        return refined

    def _get_descriptive_labels(self, short_labels: List[str]) -> List[str]:
        """Convert short labels to descriptive labels for GLiNER."""
        return [
            self.DESCRIPTIVE_LABELS.get(label, label)
            for label in short_labels
            if label in self.DESCRIPTIVE_LABELS
        ]

    def _map_descriptive_to_short(self, descriptive_label: str) -> str:
        """Map descriptive label back to short label."""
        return self.LABEL_REVERSE_MAP.get(descriptive_label, descriptive_label)

    def extract_entities(
        self,
        text: str,
        labels: Optional[List[str]] = None,
        threshold: Optional[float] = None,
    ) -> NERResult:
        """
        Extract clinical entities from text.

        Args:
            text: Input text to extract entities from
            labels: Entity labels to filter (optional, uses all if not specified)
            threshold: Confidence threshold (default: instance threshold)

        Returns:
            NERResult with extracted entities
        """
        self._load_model()

        if labels is None:
            labels = get_all_clinical_labels()

        threshold = threshold or self.threshold

        # Apply OCR correction before extraction
        corrected_text = self._apply_ocr_corrections(text)

        if self._model is not None:
            entities = self._extract_with_gliner(corrected_text, labels, threshold)
        else:
            entities = self._extract_with_clinical_regex(corrected_text, labels)

        # Apply post-processing pipeline
        entities = self._merge_adjacent_spans(entities)
        entities = self._split_multi_drug_spans(entities)  # NEW: Split concatenated drugs
        entities = self._apply_drug_class_postprocessor(entities)
        entities = self._apply_clinical_rules(entities, text)
        entities = self._deduplicate_entities(entities)

        # Filter by labels if specified
        if labels:
            entities = [e for e in entities if e.label in labels]

        return NERResult(
            text=text,
            entities=entities,
            labels_used=labels,
            extraction_timestamp=datetime.now(timezone.utc).isoformat(),
            model_name=self.model_name if self._model else "clinical_regex_fallback",
            model_version=self.VERSION,
        )

    def _extract_with_gliner(
        self,
        text: str,
        labels: List[str],
        threshold: float,
    ) -> List[Entity]:
        """Extract entities using GLiNER with descriptive labels."""
        # Convert short labels to descriptive labels for GLiNER
        descriptive_labels = self._get_descriptive_labels(labels)

        if not descriptive_labels:
            return []

        try:
            # GLiNER prediction with descriptive labels
            raw_entities = self._model.predict_entities(
                text,
                descriptive_labels,
                threshold=threshold,
            )
        except Exception as e:
            print(f"⚠️ GLiNER extraction failed: {e}")
            return self._extract_with_clinical_regex(text, labels)

        entities = []
        for ent in raw_entities:
            # Map descriptive label back to short label
            short_label = self._map_descriptive_to_short(ent.get("label", ""))

            entities.append(Entity(
                text=ent.get("text", ""),
                label=short_label,
                start=ent.get("start", 0),
                end=ent.get("end", 0),
                score=ent.get("score", 0.0),
            ))

        return entities

    def _apply_drug_class_postprocessor(self, entities: List[Entity]) -> List[Entity]:
        """
        Post-processor: Reclassify drug ingredients that are actually drug classes.

        GLiNER may extract "SGLT2 inhibitors" as drug_ingredient, but it's
        actually a drug_class. This post-processor corrects these cases.
        """
        refined = []
        for ent in entities:
            if ent.label in ("drug_ingredient", "drug_product"):
                # Check if this is actually a drug class
                text_lower = ent.text.lower().strip()
                if text_lower in self.DRUG_CLASSES:
                    # Reclassify as drug_class
                    ent = Entity(
                        text=ent.text,
                        label="drug_class",
                        start=ent.start,
                        end=ent.end,
                        score=ent.score,
                    )
            refined.append(ent)
        return refined

    def _merge_adjacent_spans(self, entities: List[Entity]) -> List[Entity]:
        """
        Merge adjacent entities of the same type.

        Handles cases like "every 3-6 months" being split into multiple tokens.
        """
        if not entities:
            return entities

        entities = sorted(entities, key=lambda e: e.start)
        merged = [entities[0]]

        for current in entities[1:]:
            prev = merged[-1]

            # Merge if same label and adjacent (within 3 chars for whitespace/hyphen)
            if (current.label == prev.label and
                current.start - prev.end <= 3):
                # Create merged entity with combined text
                merged[-1] = Entity(
                    text=f"{prev.text} {current.text}".strip(),
                    label=prev.label,
                    start=prev.start,
                    end=current.end,
                    score=min(prev.score, current.score),
                )
            else:
                merged.append(current)

        return merged

    def _apply_clinical_rules(self, entities: List[Entity], text: str) -> List[Entity]:
        """
        Apply clinical domain rules for entity refinement.

        Rules:
        1. Detect eGFR thresholds not caught by GLiNER
        2. Identify monitoring frequencies
        3. Detect lab tests and values
        4. Identify contraindication markers
        """
        refined = list(entities)

        # Add eGFR threshold detection
        egfr_patterns = [
            r'\b(e?GFR|CrCl)\s*[<>≤≥=]\s*\d+(?:\.\d+)?',
            r'\b(e?GFR|CrCl)\s+(?:of\s+)?(?:less than|greater than|below|above)\s+\d+',
            r'\b(e?GFR|CrCl)\s+\d+\s*[-–]\s*\d+',
        ]
        for pattern in egfr_patterns:
            for match in re.finditer(pattern, text, re.IGNORECASE):
                # Check if not already captured
                if not any(e.start <= match.start() < e.end for e in refined):
                    refined.append(Entity(
                        text=match.group(),
                        label="egfr_threshold",
                        start=match.start(),
                        end=match.end(),
                        score=0.95,
                    ))

        # Add monitoring frequency detection
        freq_patterns = [
            r'\b(every\s+\d+\s*[-–]\s*\d+\s*(?:months?|weeks?|days?))',
            r'\b(Q\d+[-–]?\d*\s*(?:mo|months?|wk|weeks?))',
            r'\b(at\s+week\s+\d+)',
            r'\b(after\s+\d+\s+(?:weeks?|months?))',
            r'\b(annually|quarterly|monthly|weekly|daily)',
        ]
        for pattern in freq_patterns:
            for match in re.finditer(pattern, text, re.IGNORECASE):
                if not any(e.start <= match.start() < e.end for e in refined):
                    refined.append(Entity(
                        text=match.group(),
                        label="monitoring_frequency",
                        start=match.start(),
                        end=match.end(),
                        score=0.90,
                    ))

        # Add lab test detection
        lab_patterns = [
            r'\b(eGFR|serum\s+creatinine|creatinine|potassium|K\+|sodium|Na\+|'
            r'HbA1c|A1C|fasting\s+glucose|UACR|urine\s+albumin|'
            r'albumin[-\s]to[-\s]creatinine|BUN|blood\s+urea|'
            r'liver\s+function|LFTs?|AST|ALT|hemoglobin|hematocrit)\b'
        ]
        for pattern in lab_patterns:
            for match in re.finditer(pattern, text, re.IGNORECASE):
                if not any(e.start <= match.start() < e.end for e in refined):
                    refined.append(Entity(
                        text=match.group(),
                        label="lab_test",
                        start=match.start(),
                        end=match.end(),
                        score=0.85,
                    ))

        # Add contraindication markers
        contra_patterns = [
            r'\b(contraindicated|avoid(?:ed)?|do\s+not\s+(?:use|initiate|start)|'
            r'should\s+not\s+be\s+used|not\s+recommended|discontinue|stop|hold)\b'
        ]
        for pattern in contra_patterns:
            for match in re.finditer(pattern, text, re.IGNORECASE):
                if not any(e.start <= match.start() < e.end for e in refined):
                    refined.append(Entity(
                        text=match.group(),
                        label="contraindication_marker",
                        start=match.start(),
                        end=match.end(),
                        score=0.95,
                    ))

        return sorted(refined, key=lambda e: e.start)

    def _deduplicate_entities(self, entities: List[Entity]) -> List[Entity]:
        """
        Remove duplicate and overlapping entities, keeping highest confidence.
        """
        if not entities:
            return entities

        # Sort by start position, then by score descending
        entities = sorted(entities, key=lambda e: (e.start, -e.score))

        unique = []
        for ent in entities:
            # Check if overlaps with any existing entity
            overlaps = False
            for existing in unique:
                # Check for overlap
                if (ent.start < existing.end and ent.end > existing.start):
                    overlaps = True
                    break

            if not overlaps:
                unique.append(ent)

        return sorted(unique, key=lambda e: e.start)

    def _extract_with_clinical_regex(
        self,
        text: str,
        labels: Optional[List[str]],
    ) -> List[Entity]:
        """
        Fallback extraction using regex patterns when GLiNER unavailable.

        This provides robust entity recognition for clinical text using
        curated patterns for common KDIGO guideline entities.
        """
        entities = []
        labels = labels or []

        # Drug name patterns (common drugs from KDIGO)
        if not labels or "drug_ingredient" in labels or "drug_name" in labels:
            drug_patterns = [
                r'\b(metformin|dapagliflozin|empagliflozin|canagliflozin|ertugliflozin|'
                r'sotagliflozin|finerenone|lisinopril|enalapril|ramipril|perindopril|'
                r'losartan|valsartan|irbesartan|candesartan|telmisartan|olmesartan|'
                r'spironolactone|eplerenone|furosemide|bumetanide|hydrochlorothiazide|'
                r'chlorthalidone|indapamide|semaglutide|liraglutide|dulaglutide|'
                r'sitagliptin|linagliptin|saxagliptin|alogliptin|pioglitazone|'
                r'glyburide|glipizide|glimepiride|insulin)\b'
            ]
            for pattern in drug_patterns:
                for match in re.finditer(pattern, text, re.IGNORECASE):
                    entities.append(Entity(
                        text=match.group(),
                        label="drug_ingredient",
                        start=match.start(),
                        end=match.end(),
                        score=0.90,
                    ))

        # Drug class patterns
        if not labels or "drug_class" in labels:
            class_patterns = [
                r'\b(SGLT2\s*inhibitors?|SGLT2i|ACE\s*inhibitors?|ACEi|ARBs?|'
                r'biguanides?|sulfonylureas?|thiazolidinediones?|TZDs?|'
                r'GLP[-\s]?1\s*(?:receptor\s+)?agonists?|GLP[-\s]?1\s*RAs?|'
                r'MRAs?|mineralocorticoid\s+receptor\s+antagonists?|'
                r'DPP[-\s]?4\s*inhibitors?|RASi|RAS\s*inhibitors?)\b'
            ]
            for pattern in class_patterns:
                for match in re.finditer(pattern, text, re.IGNORECASE):
                    entities.append(Entity(
                        text=match.group(),
                        label="drug_class",
                        start=match.start(),
                        end=match.end(),
                        score=0.88,
                    ))

        # Condition patterns
        if not labels or "condition" in labels:
            condition_patterns = [
                r'\b(CKD|chronic\s+kidney\s+disease|heart\s+failure|HF|HFrEF|HFpEF|'
                r'AKI|acute\s+kidney\s+injury|ESKD|ESRD|end[-\s]stage\s+(?:renal|kidney)|'
                r'hyperkalemia|hypokalemia|hyperglycemia|hypoglycemia|'
                r'lactic\s+acidosis|diabetic\s+ketoacidosis|DKA|'
                r'type\s*[12]\s*diabetes|T[12]DM?|diabetes\s+mellitus|'
                r'hypertension|HTN|albuminuria|proteinuria)\b'
            ]
            for pattern in condition_patterns:
                for match in re.finditer(pattern, text, re.IGNORECASE):
                    entities.append(Entity(
                        text=match.group(),
                        label="condition",
                        start=match.start(),
                        end=match.end(),
                        score=0.88,
                    ))

        # Lab unit patterns
        if not labels or "lab_unit" in labels:
            unit_patterns = [
                r'\b(mL/min/1\.73\s*m[²2]|mg/dL|mEq/L|mmol/L|mg/g|g/mol|%)\b'
            ]
            for pattern in unit_patterns:
                for match in re.finditer(pattern, text):
                    entities.append(Entity(
                        text=match.group(),
                        label="lab_unit",
                        start=match.start(),
                        end=match.end(),
                        score=0.92,
                    ))

        # Severity patterns
        if not labels or "severity" in labels:
            severity_patterns = [
                r'\b(severe|moderate|mild|critical|stage\s*[1-5]|'
                r'life[-\s]threatening|serious|major|minor)\b'
            ]
            for pattern in severity_patterns:
                for match in re.finditer(pattern, text, re.IGNORECASE):
                    entities.append(Entity(
                        text=match.group(),
                        label="severity",
                        start=match.start(),
                        end=match.end(),
                        score=0.80,
                    ))

        return entities

    def extract_for_kb(
        self,
        text: str,
        target_kb: Literal["dosing", "safety", "monitoring"],
        threshold: Optional[float] = None,
    ) -> NERResult:
        """
        Extract entities relevant for a specific KB.

        Args:
            text: Input text
            target_kb: Target KB ("dosing", "safety", or "monitoring")
            threshold: Confidence threshold

        Returns:
            NERResult with KB-specific entities
        """
        labels = get_labels_for_kb(target_kb)
        return self.extract_entities(text, labels, threshold)

    def extract_batch(
        self,
        texts: List[str],
        labels: Optional[List[str]] = None,
        threshold: Optional[float] = None,
    ) -> List[NERResult]:
        """
        Extract entities from multiple texts.

        Args:
            texts: List of input texts
            labels: Entity labels to extract
            threshold: Confidence threshold

        Returns:
            List of NERResult objects
        """
        return [
            self.extract_entities(text, labels, threshold)
            for text in texts
        ]

    def annotate_text(
        self,
        text: str,
        labels: Optional[List[str]] = None,
        format: Literal["inline", "standoff"] = "inline",
    ) -> str:
        """
        Annotate text with entity tags.

        Args:
            text: Input text
            labels: Entity labels to extract
            format: "inline" for XML-style or "standoff" for offset-based

        Returns:
            Annotated text string
        """
        result = self.extract_entities(text, labels)

        if format == "standoff":
            lines = [text, "---ENTITIES---"]
            for e in result.entities:
                lines.append(f"{e.start}\t{e.end}\t{e.label}\t{e.text}\t{e.score:.2f}")
            return "\n".join(lines)

        # Inline format: insert tags from end to start to preserve positions
        annotated = text
        for e in reversed(result.entities):
            tag_open = f"<{e.label}>"
            tag_close = f"</{e.label}>"
            annotated = (
                annotated[:e.start] +
                tag_open +
                annotated[e.start:e.end] +
                tag_close +
                annotated[e.end:]
            )

        return annotated


# Backward compatibility alias
OpenMedNERExtractor = ClinicalNERExtractor


def create_extractor_from_env() -> ClinicalNERExtractor:
    """
    Create a NER extractor configured from environment variables.

    Environment Variables:
        GLINER_MODEL: Model name (default: urchade/gliner_mediumv2.1)
        NER_DEVICE: Device (default: cpu)
        NER_THRESHOLD: Confidence threshold (default: 0.6)
    """
    import os

    return ClinicalNERExtractor(
        model_name=os.getenv("GLINER_MODEL"),
        device=os.getenv("NER_DEVICE", "cpu"),
        threshold=float(os.getenv("NER_THRESHOLD", "0.6")),
    )


# CLI interface
if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="L2 Clinical NER Extraction with GLiNER"
    )
    parser.add_argument(
        "text",
        type=str,
        nargs="?",
        help="Text to extract entities from"
    )
    parser.add_argument(
        "--file", "-f",
        type=Path,
        help="File containing text to process"
    )
    parser.add_argument(
        "--labels", "-l",
        type=str,
        nargs="+",
        help="Entity labels to extract"
    )
    parser.add_argument(
        "--kb",
        type=str,
        choices=["dosing", "safety", "monitoring"],
        help="Target KB for entity extraction"
    )
    parser.add_argument(
        "--threshold", "-t",
        type=float,
        default=0.6,
        help="Confidence threshold (default: 0.6)"
    )
    parser.add_argument(
        "--annotate", "-a",
        action="store_true",
        help="Output annotated text instead of JSON"
    )
    parser.add_argument(
        "--output", "-o",
        type=Path,
        help="Output file"
    )
    parser.add_argument(
        "--model", "-m",
        type=str,
        default=None,
        help="GLiNER model name"
    )

    args = parser.parse_args()

    # Get input text
    if args.file:
        text = args.file.read_text()
    elif args.text:
        text = args.text
    else:
        # Demo text
        text = (
            "Metformin is contraindicated when eGFR falls below 30 mL/min/1.73m2. "
            "SGLT2 inhibitors like dapagliflozin can be continued until eGFR < 20. "
            "Monitor potassium every 3-6 months with finerenone."
        )

    # Create extractor and process
    extractor = ClinicalNERExtractor(
        model_name=args.model,
        threshold=args.threshold,
    )

    if args.kb:
        result = extractor.extract_for_kb(text, args.kb)
    else:
        result = extractor.extract_entities(text, args.labels)

    # Output
    if args.annotate:
        output = extractor.annotate_text(text, args.labels)
    else:
        output = result.to_json()

    if args.output:
        args.output.write_text(output)
        print(f"Output saved to {args.output}")
    else:
        print(output)
