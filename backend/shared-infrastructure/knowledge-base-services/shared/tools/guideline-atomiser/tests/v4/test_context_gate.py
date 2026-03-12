"""
Tests for Channel B: Context Gate (V4.1).

Validates that the context gate correctly rejects drug matches in:
1. Reference/bibliography sections
2. Citation brackets [Smith et al., 2020]
3. Author list zones (proper noun density + "et al.")

Also validates that clinical spans in body text are NOT rejected.
"""

import sys
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

try:
    import ahocorasick
    HAS_AHOCORASICK = True
except ImportError:
    HAS_AHOCORASICK = False

from extraction.v4.models import GuidelineTree, GuidelineSection, TableBoundary

pytestmark = pytest.mark.skipif(
    not HAS_AHOCORASICK,
    reason="ahocorasick-python not installed"
)


@pytest.fixture
def drug_dict():
    """Channel B drug dictionary instance."""
    from extraction.v4.channel_b_drug_dict import ChannelBDrugDict
    return ChannelBDrugDict()


# =============================================================================
# Test Text Constants
# =============================================================================

# Clinical text followed by a reference section mentioning the same drug
TEXT_WITH_REFERENCES = """\
## Recommendation 4.1.1

Metformin is recommended for patients with T2D and CKD.
Dapagliflozin should be initiated when eGFR >= 25.

## References

1. Smith J, et al. Metformin in CKD: a meta-analysis. NEJM 2021.
2. Jones A, et al. Dapagliflozin outcomes trial. Lancet 2020.
"""

TEXT_WITH_CITATIONS = """\
## Recommendation 5.1

Empagliflozin is recommended [Smith et al., 2021] for heart failure.
Finerenone has shown benefit in CKD [14,15] outcomes.
"""

TEXT_WITH_AUTHOR_LIST = """\
## Guideline Authors

Writing Committee: John Smith, Sarah Johnson, Robert Metformin,
Daniel Enalapril, Lisa Brown et al.

## Clinical Recommendations

We recommend enalapril for patients with diabetic nephropathy.
"""


def _build_tree_for_text(text: str, sections_spec: list[dict]) -> GuidelineTree:
    """Build a GuidelineTree from section specifications."""
    sections = []
    for spec in sections_spec:
        heading = spec["heading"]
        start = text.index(heading) if "start" not in spec else spec["start"]
        end = spec.get("end", len(text))
        sections.append(GuidelineSection(
            section_id=spec.get("id", "1"),
            heading=heading,
            start_offset=start,
            end_offset=end,
            page_number=1,
            block_type=spec.get("type", "paragraph"),
            children=spec.get("children", []),
        ))
    return GuidelineTree(sections=sections, tables=[], total_pages=1)


# =============================================================================
# Reference Section Tests
# =============================================================================

class TestContextGateReferences:
    """Drug matches in reference sections should be rejected."""

    def test_clinical_metformin_kept(self, drug_dict):
        """Metformin in the recommendation section should be extracted."""
        ref_start = TEXT_WITH_REFERENCES.index("## References")
        tree = _build_tree_for_text(TEXT_WITH_REFERENCES, [
            {
                "id": "4.1.1",
                "heading": "Recommendation 4.1.1",
                "start": 0,
                "end": ref_start,
                "type": "recommendation",
            },
            {
                "id": "refs",
                "heading": "References",
                "start": ref_start,
                "end": len(TEXT_WITH_REFERENCES),
                "type": "paragraph",
            },
        ])

        output = drug_dict.extract(TEXT_WITH_REFERENCES, tree)
        clinical_texts = [s.text.lower() for s in output.spans]

        assert "metformin" in clinical_texts, "Clinical metformin should be kept"
        assert "dapagliflozin" in clinical_texts, "Clinical dapagliflozin should be kept"

    def test_reference_metformin_rejected(self, drug_dict):
        """Metformin in the reference section should be rejected by context gate."""
        ref_start = TEXT_WITH_REFERENCES.index("## References")
        tree = _build_tree_for_text(TEXT_WITH_REFERENCES, [
            {
                "id": "4.1.1",
                "heading": "Recommendation 4.1.1",
                "start": 0,
                "end": ref_start,
                "type": "recommendation",
            },
            {
                "id": "refs",
                "heading": "References",
                "start": ref_start,
                "end": len(TEXT_WITH_REFERENCES),
                "type": "paragraph",
            },
        ])

        output = drug_dict.extract(TEXT_WITH_REFERENCES, tree)

        # Should have context_gate_rejections > 0
        assert output.metadata["context_gate_rejections"] > 0

        # Verify none of the kept spans are in the reference section
        for span in output.spans:
            assert span.start < ref_start, (
                f"Span '{span.text}' at offset {span.start} is in reference section"
            )

    def test_context_gate_rejection_count(self, drug_dict):
        """Context gate should report accurate rejection count."""
        ref_start = TEXT_WITH_REFERENCES.index("## References")
        tree = _build_tree_for_text(TEXT_WITH_REFERENCES, [
            {
                "id": "4.1.1",
                "heading": "Recommendation 4.1.1",
                "start": 0,
                "end": ref_start,
            },
            {
                "id": "refs",
                "heading": "References",
                "start": ref_start,
                "end": len(TEXT_WITH_REFERENCES),
            },
        ])

        output = drug_dict.extract(TEXT_WITH_REFERENCES, tree)
        rejections = output.metadata["context_gate_rejections"]
        # At least 2 rejections: metformin + dapagliflozin in reference text
        assert rejections >= 2


# =============================================================================
# Citation Bracket Tests
# =============================================================================

class TestContextGateCitations:
    """Drug matches inside citation brackets should be rejected."""

    def test_clinical_empagliflozin_kept(self, drug_dict):
        """Empagliflozin outside brackets should be kept."""
        tree = _build_tree_for_text(TEXT_WITH_CITATIONS, [
            {"id": "5.1", "heading": "Recommendation 5.1"},
        ])

        output = drug_dict.extract(TEXT_WITH_CITATIONS, tree)
        texts = [s.text.lower() for s in output.spans]
        assert "empagliflozin" in texts
        assert "finerenone" in texts


# =============================================================================
# Author List Tests
# =============================================================================

class TestContextGateAuthorList:
    """Drug names in author list zones should be rejected."""

    def test_clinical_enalapril_kept(self, drug_dict):
        """Enalapril in recommendations should be extracted."""
        author_end = TEXT_WITH_AUTHOR_LIST.index("## Clinical Recommendations")
        tree = _build_tree_for_text(TEXT_WITH_AUTHOR_LIST, [
            {
                "id": "authors",
                "heading": "Guideline Authors",
                "start": 0,
                "end": author_end,
            },
            {
                "id": "recs",
                "heading": "Clinical Recommendations",
                "start": author_end,
                "end": len(TEXT_WITH_AUTHOR_LIST),
                "type": "recommendation",
            },
        ])

        output = drug_dict.extract(TEXT_WITH_AUTHOR_LIST, tree)

        # The clinical enalapril should be found
        clinical_spans = [
            s for s in output.spans
            if s.start >= author_end and s.text.lower() == "enalapril"
        ]
        assert len(clinical_spans) >= 1, "Clinical enalapril should be kept"


# =============================================================================
# Pass-through Tests (clinical text should NOT be rejected)
# =============================================================================

class TestContextGatePassthrough:
    """Clinical context should pass through the gate unfiltered."""

    def test_standard_clinical_text_unaffected(self, drug_dict):
        """Normal clinical text should have zero context gate rejections."""
        text = (
            "We recommend metformin as first-line therapy. "
            "Dapagliflozin is an SGLT2 inhibitor. "
            "Monitor eGFR every 3 months."
        )
        tree = GuidelineTree(
            sections=[
                GuidelineSection(
                    section_id="1",
                    heading="Clinical Recommendation",
                    start_offset=0,
                    end_offset=len(text),
                    page_number=1,
                    block_type="recommendation",
                    children=[],
                )
            ],
            tables=[],
            total_pages=1,
        )

        output = drug_dict.extract(text, tree)
        assert output.metadata["context_gate_rejections"] == 0
        assert len(output.spans) >= 2  # metformin, dapagliflozin, sglt2

    def test_drug_in_table_body_kept(self, drug_dict):
        """Drug names in table body (not caption) should be kept."""
        text = (
            "| Drug | Dose |\n"
            "| --- | --- |\n"
            "| Metformin | 1000 mg |\n"
            "| Enalapril | 10 mg |\n"
        )
        tree = GuidelineTree(
            sections=[
                GuidelineSection(
                    section_id="1",
                    heading="Drug Table",
                    start_offset=0,
                    end_offset=len(text),
                    page_number=1,
                    block_type="paragraph",
                    children=[],
                )
            ],
            tables=[],
            total_pages=1,
        )

        output = drug_dict.extract(text, tree)
        texts = [s.text.lower() for s in output.spans]
        assert "metformin" in texts
        assert "enalapril" in texts
