"""
Tests for GuidelineProfile (Phase 2).

Validates:
1. kdigo_default() produces identical context to former guideline_context_kdigo()
2. YAML round-trip: from_yaml() loads the KDIGO profile and matches kdigo_default()
3. PDF source resolution and error handling
4. Drug class skip list matches original hardcoded values
5. guideline_context() returns correct structure
6. Validation errors for missing required fields
7. source_choices property
8. extra_ingredients / extra_classes passthrough
"""

import sys
import os
import tempfile
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.guideline_profile import GuidelineProfile


# =============================================================================
# kdigo_default() Tests
# =============================================================================

class TestKdigoDefault:
    """Verify kdigo_default() produces the exact same values as the former
    hardcoded pipeline constants."""

    def test_profile_id(self):
        profile = GuidelineProfile.kdigo_default()
        assert profile.profile_id == "kdigo_2022_diabetes_ckd"

    def test_authority(self):
        profile = GuidelineProfile.kdigo_default()
        assert profile.authority == "KDIGO"

    def test_display_name(self):
        profile = GuidelineProfile.kdigo_default()
        assert profile.display_name == "KDIGO 2022 Diabetes in CKD"

    def test_document_title(self):
        profile = GuidelineProfile.kdigo_default()
        assert profile.document_title == "KDIGO 2022 Diabetes in CKD"

    def test_effective_date(self):
        profile = GuidelineProfile.kdigo_default()
        assert profile.effective_date == "2022-11-01"

    def test_doi(self):
        profile = GuidelineProfile.kdigo_default()
        assert profile.doi == "10.1016/j.kint.2022.06.008"

    def test_version(self):
        profile = GuidelineProfile.kdigo_default()
        assert profile.version == "2022"

    def test_pdf_sources_match_original(self):
        """PDF source keys and filenames must match the former PDF_PATHS dict."""
        profile = GuidelineProfile.kdigo_default()
        assert set(profile.pdf_sources.keys()) == {
            "quick-reference", "full-guide", "dosing-report",
        }
        assert "KDIGO-2022-Diabetes-Guideline-Quick-Reference-Guide.pdf" in profile.pdf_sources["quick-reference"]
        assert "KDIGO-2022-Clinical-Practice-Guideline-for-Diabetes-Management-in-CKD.pdf" in profile.pdf_sources["full-guide"]
        assert "KDIGO-DrugDosingReportFinal.pdf" in profile.pdf_sources["dosing-report"]

    def test_drug_class_skip_list_matches_original(self):
        """Skip list must match the expanded canonical class labels from DossierAssembler."""
        profile = GuidelineProfile.kdigo_default()
        original = [
            "sglt2i", "acei", "arb", "glp-1 ra", "mra", "nsmra",
            "dpp-4i", "sulfonylurea", "tzd", "rasi",
            "beta-blocker", "ccb", "diuretic", "loop diuretic",
            "statin", "nsaid",
        ]
        assert profile.drug_class_skip_list == original

    def test_extra_ingredients_empty(self):
        """KDIGO profile has no extra ingredients (base dict suffices)."""
        profile = GuidelineProfile.kdigo_default()
        assert profile.extra_drug_ingredients == {}

    def test_extra_classes_empty(self):
        profile = GuidelineProfile.kdigo_default()
        assert profile.extra_drug_classes == {}

    def test_extra_patterns_empty(self):
        profile = GuidelineProfile.kdigo_default()
        assert profile.extra_patterns == []

    def test_reference_headings(self):
        profile = GuidelineProfile.kdigo_default()
        assert "References" in profile.reference_section_headings
        assert "Bibliography" in profile.reference_section_headings

    def test_immutable(self):
        """GuidelineProfile is frozen — cannot modify after construction."""
        profile = GuidelineProfile.kdigo_default()
        with pytest.raises(AttributeError):
            profile.authority = "ADA"


# =============================================================================
# guideline_context() Tests
# =============================================================================

class TestGuidelineContext:
    """Verify guideline_context() matches the former guideline_context_kdigo()."""

    def test_context_structure(self):
        profile = GuidelineProfile.kdigo_default()
        ctx = profile.guideline_context()
        assert ctx == {
            "authority": "KDIGO",
            "document": "KDIGO 2022 Diabetes in CKD",
            "effective_date": "2022-11-01",
            "doi": "10.1016/j.kint.2022.06.008",
            "version": "2022",
        }

    def test_context_does_not_include_source(self):
        """guideline_context() doesn't include 'source' — pipeline adds it."""
        profile = GuidelineProfile.kdigo_default()
        ctx = profile.guideline_context()
        assert "source" not in ctx


# =============================================================================
# YAML Loading Tests
# =============================================================================

class TestFromYaml:
    """Tests for GuidelineProfile.from_yaml()."""

    def test_load_kdigo_yaml(self):
        """Load the actual KDIGO YAML and verify it matches kdigo_default()."""
        yaml_path = (
            Path(__file__).resolve().parents[2]
            / "data" / "profiles" / "kdigo_2022_diabetes_ckd.yaml"
        )
        if not yaml_path.exists():
            pytest.skip(f"YAML not found: {yaml_path}")

        from_yaml = GuidelineProfile.from_yaml(yaml_path)
        default = GuidelineProfile.kdigo_default()

        assert from_yaml.profile_id == default.profile_id
        assert from_yaml.authority == default.authority
        assert from_yaml.display_name == default.display_name
        assert from_yaml.document_title == default.document_title
        assert from_yaml.effective_date == default.effective_date
        assert from_yaml.doi == default.doi
        assert from_yaml.version == default.version
        assert from_yaml.pdf_sources == default.pdf_sources
        assert from_yaml.drug_class_skip_list == default.drug_class_skip_list
        assert from_yaml.extra_drug_ingredients == default.extra_drug_ingredients
        assert from_yaml.extra_drug_classes == default.extra_drug_classes
        assert from_yaml.extra_patterns == default.extra_patterns
        assert from_yaml.reference_section_headings == default.reference_section_headings

    def test_guideline_context_from_yaml_matches_default(self):
        """guideline_context() from YAML must match kdigo_default()."""
        yaml_path = (
            Path(__file__).resolve().parents[2]
            / "data" / "profiles" / "kdigo_2022_diabetes_ckd.yaml"
        )
        if not yaml_path.exists():
            pytest.skip(f"YAML not found: {yaml_path}")

        from_yaml = GuidelineProfile.from_yaml(yaml_path)
        default = GuidelineProfile.kdigo_default()

        assert from_yaml.guideline_context() == default.guideline_context()

    def test_missing_file_raises(self):
        with pytest.raises(FileNotFoundError):
            GuidelineProfile.from_yaml("/nonexistent/profile.yaml")

    def test_missing_required_fields(self):
        """YAML with missing required fields should raise ValueError."""
        with tempfile.NamedTemporaryFile(suffix=".yaml", mode="w", delete=False) as f:
            f.write("profile_id: test\nauthority: TEST\n")
            f.flush()
            try:
                with pytest.raises(ValueError, match="Missing required"):
                    GuidelineProfile.from_yaml(f.name)
            finally:
                os.unlink(f.name)

    def test_invalid_yaml_type(self):
        """Non-mapping YAML should raise ValueError."""
        with tempfile.NamedTemporaryFile(suffix=".yaml", mode="w", delete=False) as f:
            f.write("- list_item_1\n- list_item_2\n")
            f.flush()
            try:
                with pytest.raises(ValueError, match="must be a mapping"):
                    GuidelineProfile.from_yaml(f.name)
            finally:
                os.unlink(f.name)

    def test_extra_patterns_round_trip(self):
        """Extra patterns stored as YAML lists should be loaded as tuples."""
        with tempfile.NamedTemporaryFile(suffix=".yaml", mode="w", delete=False) as f:
            f.write(
                "profile_id: test_extra\n"
                "display_name: Test\n"
                "authority: TEST\n"
                "document_title: Test Doc\n"
                "effective_date: '2024-01-01'\n"
                "doi: '10.0/test'\n"
                "version: '1'\n"
                "extra_patterns:\n"
                "  - ['eGFR.*threshold', 0.85, 'renal']\n"
                "  - ['HbA1c.*target', 0.80, 'glycemic']\n"
            )
            f.flush()
            try:
                profile = GuidelineProfile.from_yaml(f.name)
                assert len(profile.extra_patterns) == 2
                assert profile.extra_patterns[0] == ("eGFR.*threshold", 0.85, "renal")
                assert isinstance(profile.extra_patterns[0], tuple)
            finally:
                os.unlink(f.name)


# =============================================================================
# PDF Resolution Tests
# =============================================================================

class TestResolvePdf:
    """Tests for resolve_pdf() method."""

    def test_unknown_source_key_raises(self):
        profile = GuidelineProfile.kdigo_default()
        with pytest.raises(KeyError, match="Unknown PDF source"):
            profile.resolve_pdf("nonexistent-source", "/tmp")

    def test_resolve_existing_pdf(self):
        """resolve_pdf should return a Path when the file exists."""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Create a dummy PDF
            pdf_name = "KDIGO-2022-Diabetes-Guideline-Quick-Reference-Guide.pdf"
            Path(tmpdir, pdf_name).touch()

            profile = GuidelineProfile.kdigo_default()
            result = profile.resolve_pdf("quick-reference", tmpdir)
            assert result.exists()
            assert result.name == pdf_name

    def test_resolve_missing_pdf_raises(self):
        """resolve_pdf should raise FileNotFoundError for missing PDF."""
        profile = GuidelineProfile.kdigo_default()
        with pytest.raises(FileNotFoundError, match="PDF not found"):
            profile.resolve_pdf("quick-reference", "/nonexistent/dir")


# =============================================================================
# source_choices Property Tests
# =============================================================================

class TestSourceChoices:
    """Tests for the source_choices property."""

    def test_kdigo_source_choices(self):
        profile = GuidelineProfile.kdigo_default()
        choices = profile.source_choices
        assert isinstance(choices, list)
        assert "quick-reference" in choices
        assert "full-guide" in choices
        assert "dosing-report" in choices
        assert len(choices) == 3

    def test_choices_are_sorted(self):
        profile = GuidelineProfile.kdigo_default()
        choices = profile.source_choices
        assert choices == sorted(choices)


# =============================================================================
# Channel B Extra Ingredients Integration
# =============================================================================

class TestChannelBExtras:
    """Verify Channel B accepts extra ingredients from profile."""

    def test_channel_b_with_extra_ingredients(self):
        """Channel B should include extra ingredients in its automaton."""
        try:
            import ahocorasick
        except ImportError:
            pytest.skip("ahocorasick-python not installed")

        from extraction.v4.channel_b_drug_dict import ChannelBDrugDict

        extra = {"pembrolizumab": "1547220", "nivolumab": "1597876"}
        channel_b = ChannelBDrugDict(extra_ingredients=extra)

        # Verify the extra drugs are findable
        from extraction.v4.models import GuidelineTree, GuidelineSection
        text = "Patient received pembrolizumab and nivolumab for treatment."
        tree = GuidelineTree(
            sections=[GuidelineSection(
                section_id="1", heading="Treatment",
                start_offset=0, end_offset=len(text),
                page_number=1, block_type="recommendation", children=[],
            )],
            tables=[], total_pages=1,
        )
        output = channel_b.extract(text, tree)
        found_drugs = {s.text.lower() for s in output.spans}
        assert "pembrolizumab" in found_drugs
        assert "nivolumab" in found_drugs

    def test_channel_b_without_extras_backward_compat(self):
        """Channel B with no extras should work exactly as before."""
        try:
            import ahocorasick
        except ImportError:
            pytest.skip("ahocorasick-python not installed")

        from extraction.v4.channel_b_drug_dict import ChannelBDrugDict

        # Default construction (no extras) — backward compatible
        channel_b = ChannelBDrugDict()
        assert channel_b._extra_ingredients == {}
        assert channel_b._extra_classes == {}

    def test_extra_does_not_overwrite_base(self):
        """Extra ingredients should NOT overwrite base dictionary entries."""
        try:
            import ahocorasick
        except ImportError:
            pytest.skip("ahocorasick-python not installed")

        from extraction.v4.channel_b_drug_dict import ChannelBDrugDict

        # Try to overwrite metformin with a wrong RxNorm code
        extra = {"metformin": "999999"}
        channel_b = ChannelBDrugDict(extra_ingredients=extra)

        # The base metformin (860975) should still be in the automaton
        from extraction.v4.models import GuidelineTree, GuidelineSection
        text = "Prescribe metformin for the patient."
        tree = GuidelineTree(
            sections=[GuidelineSection(
                section_id="1", heading="Rx",
                start_offset=0, end_offset=len(text),
                page_number=1, block_type="recommendation", children=[],
            )],
            tables=[], total_pages=1,
        )
        output = channel_b.extract(text, tree)
        metformin_spans = [s for s in output.spans if s.text.lower() == "metformin"]
        assert len(metformin_spans) == 1
        # RxNorm should be the ORIGINAL base value, not the extra
        assert metformin_spans[0].channel_metadata.get("rxnorm_candidate") == "860975"
