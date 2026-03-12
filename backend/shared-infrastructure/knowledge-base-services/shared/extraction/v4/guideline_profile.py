"""
GuidelineProfile: Configuration abstraction for multi-guideline pipelines.

Replaces KDIGO-hardcoded values in run_pipeline_targeted.py with a
profile-driven system that supports multiple guideline sources (KDIGO, ADA,
RSSDI, etc.) from a single codebase.

Usage:
    # Default (backward-compatible — identical to current KDIGO code):
    profile = GuidelineProfile.kdigo_default()

    # From YAML (new guidelines):
    profile = GuidelineProfile.from_yaml("profiles/ada_2024_soc.yaml")

    # In pipeline:
    context = profile.guideline_context()
    pdf_path = profile.resolve_pdf("quick-reference", pdfs_dir)
"""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional


@dataclass(frozen=True)
class GuidelineProfile:
    """Immutable configuration for a specific clinical guideline.

    Each field maps to a previously hardcoded value in the pipeline.
    The ``kdigo_default()`` classmethod produces the exact same values
    that were hardcoded before this abstraction was introduced.
    """

    # ── Identity ──────────────────────────────────────────────────────
    profile_id: str             # "kdigo_2022_diabetes_ckd"
    display_name: str           # "KDIGO 2022 Diabetes in CKD"
    authority: str              # "KDIGO"
    document_title: str         # Same as display_name (used in CoverageGuard + metadata)
    effective_date: str         # ISO date: "2022-11-01"
    doi: str                    # "10.1016/j.kint.2022.06.008"
    version: str                # "2022"

    # ── PDF Sources ───────────────────────────────────────────────────
    # Maps source key → PDF filename (resolved relative to pdfs_dir)
    pdf_sources: dict[str, str] = field(default_factory=dict)

    # ── Channel B Supplements ─────────────────────────────────────────
    # Extra drug ingredients to merge into Channel B's automaton
    # Format: {"drug_name": "rxnorm_code" or None}
    extra_drug_ingredients: dict[str, Optional[str]] = field(default_factory=dict)

    # Extra drug class variants to merge into Channel B
    # Format: {"variant_text": "CanonicalClassName"}
    extra_drug_classes: dict[str, str] = field(default_factory=dict)

    # ── Channel C Supplements ─────────────────────────────────────────
    # Extra regex patterns for Channel C grammar
    # Format: list of (pattern_regex, confidence, category)
    extra_patterns: list[tuple[str, float, str]] = field(default_factory=list)

    # ── L2.5 RxNorm Pre-Lookup ────────────────────────────────────────
    # Drug class names to skip during RxNorm code lookup (they aren't
    # individual ingredients so KB-7 won't have codes for them)
    drug_class_skip_list: list[str] = field(default_factory=list)

    # ── Context Filtering ─────────────────────────────────────────────
    # Section headings that indicate non-clinical zones (e.g. "References")
    reference_section_headings: list[str] = field(default_factory=lambda: [
        "References", "Bibliography", "Supplementary Materials",
    ])

    # ── Channel B Context Gate ──────────────────────────────────────────
    # Number of characters scanned in each direction around a drug match
    # for clinical context validation (clinical verbs, co-occurring drugs)
    context_window_size: int = 200

    # ── Tiering ───────────────────────────────────────────────────────
    tiering_classifier: str = "rule_based"
    tiering_golden_dataset: Optional[str] = None

    # ── Range Integrity Engine ────────────────────────────────────────
    # Path to JSON file with severity keywords (None = use built-in defaults)
    severity_keywords_path: Optional[str] = None

    # ── Channel A: Authority-Specific Heading Sets ─────────────────
    # Subordinate headings that ALWAYS follow a numbered Recommendation
    # and must be reparented as children of that Recommendation.
    # KDIGO examples: "Rationale", "Key information", "Values and preferences"
    # ADA examples: "Recommendation", "Supporting text" (section-level)
    # If empty, the reparenting pass is skipped entirely for this guideline.
    subordinate_headings: list[str] = field(default_factory=list)

    # Chapter-level headings that RESET the numbered-section tracker to None.
    # Prevents cross-chapter contamination in the reparenting pass.
    chapter_reset_headings: list[str] = field(default_factory=list)

    # ── Chunked L1 Configuration ────────────────────────────────────
    # Manual chunk boundaries for the chunked L1 runner.
    # If empty, the auto-chunker computes boundaries deterministically.
    # Each entry: {start: int, end: int, name: str, extractor: str}
    # start/end are 0-based page indices (PyMuPDF convention).
    chunk_defs: list[dict] = field(default_factory=list)

    # Maximum pages per MonkeyOCR chunk (auto-chunker parameter).
    # Smaller = less VRAM but more subprocess overhead.
    max_pages_per_chunk: int = 10

    # Minimum text chars/page to classify a page as born-digital.
    # Pages above this threshold use PyMuPDF (fast, no GPU).
    # Pages below use MonkeyOCR (VLM-based OCR).
    born_digital_char_threshold: int = 200

    # ══════════════════════════════════════════════════════════════════
    # Factory Methods
    # ══════════════════════════════════════════════════════════════════

    @classmethod
    def kdigo_default(cls) -> GuidelineProfile:
        """Construct the KDIGO 2022 Diabetes-in-CKD profile.

        Returns the exact same values that were previously hardcoded in
        ``run_pipeline_targeted.py``.  This ensures byte-identical pipeline
        output when no ``--guideline`` flag is provided.
        """
        return cls(
            profile_id="kdigo_2022_diabetes_ckd",
            display_name="KDIGO 2022 Diabetes in CKD",
            authority="KDIGO",
            document_title="KDIGO 2022 Diabetes in CKD",
            effective_date="2022-11-01",
            doi="10.1016/j.kint.2022.06.008",
            version="2022",
            pdf_sources={
                "quick-reference": "KDIGO-2022-Diabetes-Guideline-Quick-Reference-Guide.pdf",
                "full-guide": "KDIGO-2022-Clinical-Practice-Guideline-for-Diabetes-Management-in-CKD.pdf",
                "dosing-report": "KDIGO-DrugDosingReportFinal.pdf",
            },
            extra_drug_ingredients={},
            extra_drug_classes={},
            extra_patterns=[],
            drug_class_skip_list=[
                # Canonical class labels (lowercased) from DossierAssembler
                "sglt2i", "acei", "arb", "glp-1 ra", "mra", "nsmra",
                "dpp-4i", "sulfonylurea", "tzd", "rasi",
                "beta-blocker", "ccb", "diuretic", "loop diuretic",
                "statin", "nsaid",
            ],
            reference_section_headings=[
                "References", "Bibliography", "Supplementary Materials",
            ],
            # Shadow mode: runs both rule-based and trained classifiers.
            # Rule-based output is used (no behavior change); trained predictions
            # are logged to shadow_classifier_log.json for comparison.
            # Switch to "trained" when agreement rate > 95% for 2 consecutive weeks.
            tiering_classifier="shadow",
            tiering_golden_dataset="models/tier_assigner.joblib",
            # KDIGO subordinate heading vocabulary (Format for writing guidelines v21)
            subordinate_headings=[
                "key information",
                "balance of benefits and harms",
                "certainty of evidence",
                "certainty of the evidence",
                "values and preferences",
                "resource use and costs",
                "considerations for implementation",
                "rationale",
                "rationale and evidence",
                "evidence base",
            ],
            chapter_reset_headings=[
                "research recommendations",
                "practice implications",
            ],
        )

    @classmethod
    def from_yaml(cls, path: str | Path) -> GuidelineProfile:
        """Load a GuidelineProfile from a YAML file.

        Raises:
            FileNotFoundError: If the YAML file does not exist.
            ImportError: If PyYAML is not installed.
            ValueError: If required fields are missing.
        """
        try:
            import yaml
        except ImportError:
            raise ImportError(
                "PyYAML is required for YAML profile loading. "
                "Install with: pip install pyyaml"
            )

        path = Path(path)
        if not path.exists():
            raise FileNotFoundError(f"Profile YAML not found: {path}")

        with open(path) as f:
            data = yaml.safe_load(f)

        if not isinstance(data, dict):
            raise ValueError(f"Profile YAML must be a mapping, got {type(data).__name__}")

        # Validate required fields
        required = ["profile_id", "display_name", "authority", "document_title",
                     "effective_date", "doi", "version"]
        missing = [k for k in required if k not in data]
        if missing:
            raise ValueError(f"Missing required profile fields: {missing}")

        # Normalize extra_patterns from list-of-lists to list-of-tuples
        raw_patterns = data.get("extra_patterns", [])
        extra_patterns = [tuple(p) for p in raw_patterns]

        return cls(
            profile_id=data["profile_id"],
            display_name=data["display_name"],
            authority=data["authority"],
            document_title=data["document_title"],
            effective_date=data["effective_date"],
            doi=data["doi"],
            version=data["version"],
            pdf_sources=data.get("pdf_sources", {}),
            extra_drug_ingredients=data.get("extra_drug_ingredients", {}),
            extra_drug_classes=data.get("extra_drug_classes", {}),
            extra_patterns=extra_patterns,
            drug_class_skip_list=data.get("drug_class_skip_list", []),
            reference_section_headings=data.get(
                "reference_section_headings",
                ["References", "Bibliography", "Supplementary Materials"],
            ),
            context_window_size=data.get("context_window_size", 200),
            tiering_classifier=data.get("tiering_classifier", "rule_based"),
            tiering_golden_dataset=data.get("tiering_golden_dataset"),
            severity_keywords_path=data.get("severity_keywords_path"),
            chunk_defs=data.get("chunk_defs", []),
            max_pages_per_chunk=data.get("max_pages_per_chunk", 10),
            born_digital_char_threshold=data.get("born_digital_char_threshold", 200),
            subordinate_headings=[
                h.lower() for h in data.get("subordinate_headings", [])
            ],
            chapter_reset_headings=[
                h.lower() for h in data.get("chapter_reset_headings", [])
            ],
        )

    # ══════════════════════════════════════════════════════════════════
    # Pipeline Convenience Methods
    # ══════════════════════════════════════════════════════════════════

    def guideline_context(self) -> dict:
        """Build the guideline_context dict consumed by L3 extraction.

        Returns the same structure as the former ``guideline_context_kdigo()``
        function, with values drawn from this profile.
        """
        return {
            "authority": self.authority,
            "document": self.document_title,
            "effective_date": self.effective_date,
            "doi": self.doi,
            "version": self.version,
        }

    def resolve_pdf(self, source_key: str, pdfs_dir: str | Path) -> Path:
        """Resolve a PDF source key to an absolute path.

        Args:
            source_key: One of the keys in ``pdf_sources`` (e.g. "quick-reference").
            pdfs_dir: Base directory containing the PDF files.

        Returns:
            Absolute Path to the PDF file.

        Raises:
            KeyError: If source_key is not in this profile's pdf_sources.
            FileNotFoundError: If the resolved path does not exist.
        """
        if source_key not in self.pdf_sources:
            available = ", ".join(sorted(self.pdf_sources.keys()))
            raise KeyError(
                f"Unknown PDF source '{source_key}' for profile '{self.profile_id}'. "
                f"Available: {available}"
            )
        pdf_path = Path(pdfs_dir) / self.pdf_sources[source_key]
        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF not found: {pdf_path}")
        return pdf_path

    @property
    def source_choices(self) -> list[str]:
        """List of available --source choices for this profile."""
        return sorted(self.pdf_sources.keys())
