"""
Tests for Channel B: Drug Dictionary (Aho-Corasick).

Validates:
1. Drug ingredient matching (metformin, dapagliflozin, etc.)
2. Drug class matching (SGLT2 inhibitor, ACE inhibitor, etc.)
3. Word boundary enforcement (no partial matches)
4. Case-insensitive matching
5. RxNorm candidate metadata
6. Canonical class name metadata
7. Channel output structure
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


@pytest.fixture
def simple_tree():
    """A minimal GuidelineTree for testing Channel B."""
    return GuidelineTree(
        sections=[
            GuidelineSection(
                section_id="1",
                heading="Test",
                start_offset=0,
                end_offset=10000,
                page_number=1,
                block_type="paragraph",
                children=[],
            )
        ],
        tables=[],
        total_pages=1,
    )


class TestDrugIngredientMatching:
    """Test exact drug name matching."""

    def test_metformin_matched(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        drug_texts = [s.text.lower() for s in output.spans]
        assert "metformin" in drug_texts

    def test_dapagliflozin_matched(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        drug_texts = [s.text.lower() for s in output.spans]
        assert "dapagliflozin" in drug_texts

    def test_finerenone_matched(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        drug_texts = [s.text.lower() for s in output.spans]
        assert "finerenone" in drug_texts

    def test_rxnorm_in_metadata(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        metformin_spans = [s for s in output.spans if s.text.lower() == "metformin"]
        assert len(metformin_spans) >= 1
        assert metformin_spans[0].channel_metadata["rxnorm_candidate"] == "860975"

    def test_multiple_drug_occurrences(self, drug_dict, simple_tree):
        text = "metformin is used. Later, metformin is continued."
        output = drug_dict.extract(text, simple_tree)
        metformin_spans = [s for s in output.spans if s.text.lower() == "metformin"]
        assert len(metformin_spans) == 2

    def test_confidence_is_1(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        for span in output.spans:
            assert span.confidence == 1.0


class TestDrugClassMatching:
    """Test drug class variant matching."""

    def test_sglt2_inhibitor_matched(self, drug_dict, simple_tree):
        text = "We recommend SGLT2 inhibitors for these patients."
        output = drug_dict.extract(text, simple_tree)
        class_spans = [s for s in output.spans if s.channel_metadata.get("match_type") == "class"]
        class_texts = [s.text.lower() for s in class_spans]
        assert any("sglt2" in t for t in class_texts)

    def test_ace_inhibitor_matched(self, drug_dict, simple_tree):
        text = "ACE inhibitors are first-line therapy."
        output = drug_dict.extract(text, simple_tree)
        class_spans = [s for s in output.spans if s.channel_metadata.get("match_type") == "class"]
        assert len(class_spans) >= 1

    def test_canonical_class_in_metadata(self, drug_dict, simple_tree):
        text = "SGLT2 inhibitors are recommended."
        output = drug_dict.extract(text, simple_tree)
        class_spans = [s for s in output.spans if s.channel_metadata.get("match_type") == "class"]
        if class_spans:
            assert class_spans[0].channel_metadata["canonical_class"] == "SGLT2i"

    def test_arb_class(self, drug_dict, simple_tree):
        text = "ARBs should be used when ACEi is not tolerated."
        output = drug_dict.extract(text, simple_tree)
        texts = [s.text.lower() for s in output.spans]
        assert "arbs" in texts or "arb" in texts or "acei" in texts


class TestWordBoundary:
    """Test word boundary enforcement prevents partial matches."""

    def test_arb_not_in_garbanzo(self, drug_dict, simple_tree):
        text = "Garbanzo beans are a healthy snack."
        output = drug_dict.extract(text, simple_tree)
        # "ARB" should NOT match inside "garbanzo"
        assert len(output.spans) == 0

    def test_arb_not_in_carb(self, drug_dict, simple_tree):
        text = "Low carb diet is recommended."
        output = drug_dict.extract(text, simple_tree)
        assert len(output.spans) == 0

    def test_metformin_not_in_prefix(self, drug_dict, simple_tree):
        text = "premetformine is not a real drug."
        output = drug_dict.extract(text, simple_tree)
        # Should not match "metformin" inside "premetformine"
        assert len(output.spans) == 0

    def test_standalone_arb_matches(self, drug_dict, simple_tree):
        text = "ARB therapy is recommended."
        output = drug_dict.extract(text, simple_tree)
        assert len(output.spans) >= 1


class TestCaseInsensitive:
    """Test case-insensitive matching."""

    def test_uppercase_metformin(self, drug_dict, simple_tree):
        text = "METFORMIN should be used."
        output = drug_dict.extract(text, simple_tree)
        assert len(output.spans) >= 1
        assert output.spans[0].text == "METFORMIN"  # preserves original case

    def test_mixed_case_dapagliflozin(self, drug_dict, simple_tree):
        text = "Dapagliflozin is effective."
        output = drug_dict.extract(text, simple_tree)
        assert len(output.spans) >= 1
        assert output.spans[0].text == "Dapagliflozin"


class TestChannelOutput:
    """Test ChannelOutput structure."""

    def test_output_channel_is_b(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        assert output.channel == "B"

    def test_output_success(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        assert output.success is True

    def test_output_has_metadata(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        assert "dictionary_size" in output.metadata
        assert "matches_found" in output.metadata
        assert output.metadata["matches_found"] == len(output.spans)

    def test_output_elapsed_ms(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        assert output.elapsed_ms > 0

    def test_all_spans_are_channel_b(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        for span in output.spans:
            assert span.channel == "B"

    def test_empty_text_no_matches(self, drug_dict, simple_tree):
        output = drug_dict.extract("", simple_tree)
        assert output.span_count == 0


class TestSectionAssignment:
    """Test that spans get correct section_id from tree."""

    def test_metformin_in_section_4_1_1(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        metformin_spans = [s for s in output.spans if s.text.lower() == "metformin" and s.section_id == "4.1.1"]
        assert len(metformin_spans) >= 1

    def test_dapagliflozin_in_section_4_1_2(self, drug_dict, sample_text, sample_guideline_tree):
        output = drug_dict.extract(sample_text, sample_guideline_tree)
        dapa_spans = [s for s in output.spans if s.text.lower() == "dapagliflozin" and s.section_id == "4.1.2"]
        assert len(dapa_spans) >= 1
