"""
Tests for Channel 0: Text Normalizer.

Validates:
1. Ligature repair (fi/fl ligatures from Docling)
2. Symbol correction (double-dagger -> >=)
3. OCR simple corrections (rn -> m, etc.)
4. Regex-based corrections (units, terms, drugs, numbers)
5. Whitespace normalization
6. Idempotency (running twice produces same result)
7. Real Docling output validation (if available)
"""

import sys
from pathlib import Path

import pytest

# Add shared extraction module to path
SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.channel_0_normalizer import Channel0Normalizer


class TestLigatureRepair:
    """Test Unicode ligature replacement."""

    def test_fi_ligature_unicode(self, normalizer):
        text = "\ufb01nerenone is recommended"
        result, meta = normalizer.normalize(text)
        assert result == "finerenone is recommended"
        assert meta["ligature_fixes"] > 0

    def test_fl_ligature_unicode(self, normalizer):
        text = "\ufb02uid balance is important"
        result, meta = normalizer.normalize(text)
        assert result == "fluid balance is important"
        assert meta["ligature_fixes"] > 0

    def test_docling_escape_fi(self, normalizer):
        text = "/uniFB01nerenone"
        result, meta = normalizer.normalize(text)
        assert result == "finerenone"
        assert "/uniFB01" not in result

    def test_docling_escape_fl(self, normalizer):
        text = "/uniFB02uid"
        result, meta = normalizer.normalize(text)
        assert result == "fluid"
        assert "/uniFB02" not in result

    def test_multiple_ligatures_in_one_text(self, normalizer):
        text = "/uniFB01nerenone has shown bene/uniFB01t in /uniFB02uid management"
        result, meta = normalizer.normalize(text)
        assert "finerenone" in result
        assert "benefit" in result
        assert "fluid" in result
        assert meta["ligature_fixes"] == 3

    def test_ffi_ligature(self, normalizer):
        text = "e\ufb03cacy of treatment"
        result, _ = normalizer.normalize(text)
        assert "efficacy" in result


class TestSymbolCorrection:
    """Test symbol replacement."""

    def test_double_dagger_to_gte(self, normalizer):
        text = "eGFR \u2021 30 mL/min"
        result, meta = normalizer.normalize(text)
        assert "\u2265" in result or ">=" in result
        assert "\u2021" not in result
        assert meta["symbol_fixes"] > 0

    def test_dagger_to_plus(self, normalizer):
        text = "K\u2020 monitoring"
        result, meta = normalizer.normalize(text)
        assert "K+" in result
        assert meta["symbol_fixes"] > 0


class TestOCRSimpleCorrections:
    """Test simple string-replacement OCR fixes."""

    def test_rn_to_m_metformin(self, normalizer):
        text = "rnetformin should be used"
        result, meta = normalizer.normalize(text)
        assert "metformin" in result
        assert meta["ocr_fixes"] > 0

    def test_rn_to_m_unit(self, normalizer):
        text = "30 rnL/min"
        result, _ = normalizer.normalize(text)
        assert "mL/min" in result

    def test_hba1c_variants(self, normalizer):
        text = "HbAlc should be monitored"
        result, _ = normalizer.normalize(text)
        assert "HbA1c" in result

    def test_crcl_fix(self, normalizer):
        text = "CrCI below 30"
        result, _ = normalizer.normalize(text)
        assert "CrCl" in result

    def test_egfr_all_caps(self, normalizer):
        text = "EGFR below 30"
        result, _ = normalizer.normalize(text)
        assert "eGFR" in result


class TestRegexPatternCorrections:
    """Test compiled regex-based corrections."""

    def test_egfr_unit_normalization(self, normalizer):
        text = "eGFR of 30 mL/min/1.73 m2"
        result, meta = normalizer.normalize(text)
        assert "mL/min/1.73m\u00b2" in result
        assert meta["regex_fixes"] > 0

    def test_egfr_unit_caret(self, normalizer):
        text = "30 mL/min/1.73m^2"
        result, _ = normalizer.normalize(text)
        assert "m\u00b2" in result
        assert "^" not in result

    def test_drug_name_l1_confusion(self, normalizer):
        text = "Dapag1if1ozin 10 mg"
        result, _ = normalizer.normalize(text)
        assert "dapagliflozin" in result

    def test_metformin_o0_confusion(self, normalizer):
        text = "Metf0rmin is first-line"
        result, _ = normalizer.normalize(text)
        assert "metformin" in result

    def test_egfr_O_to_0(self, normalizer):
        text = "eGFR < 3O mL/min"
        result, _ = normalizer.normalize(text)
        assert "eGFR < 30" in result

    def test_potassium_threshold(self, normalizer):
        text = "K+ > 5,5 mEq/L"
        result, _ = normalizer.normalize(text)
        assert "K+ > 5.5" in result

    def test_mg_dl_normalization(self, normalizer):
        text = "creatinine 1.5 mg/dl"
        result, _ = normalizer.normalize(text)
        assert "mg/dL" in result

    def test_finerenone_variant(self, normalizer):
        text = "Fineren0ne should be held"
        result, _ = normalizer.normalize(text)
        assert "finerenone" in result


class TestWhitespaceNormalization:
    """Test whitespace cleanup."""

    def test_multiple_spaces(self, normalizer):
        text = "eGFR  >=  30  mL/min"
        result, meta = normalizer.normalize(text)
        assert "  " not in result.split("\n")[0]
        assert meta["whitespace_fixes"] > 0

    def test_trailing_whitespace(self, normalizer):
        text = "metformin 1000 mg   \nsome other text   "
        result, _ = normalizer.normalize(text)
        for line in result.split("\n"):
            assert line == line.rstrip()

    def test_preserves_markdown_indent(self, normalizer):
        text = "  - List item with indent"
        result, _ = normalizer.normalize(text)
        assert result.startswith("  - ")

    def test_excessive_blank_lines(self, normalizer):
        text = "section 1\n\n\n\n\n\nsection 2"
        result, _ = normalizer.normalize(text)
        assert "\n\n\n\n" not in result


class TestIdempotency:
    """Test that running normalizer twice produces same result."""

    def test_double_run_same_result(self, normalizer):
        text = "/uniFB01nerenone eGFR \u2021 30 rnL/min rnetformin"
        result1, meta1 = normalizer.normalize(text)
        result2, meta2 = normalizer.normalize(result1)
        assert result1 == result2
        assert meta2["fix_count"] == 0  # no new fixes on second pass

    def test_idempotent_on_clean_text(self, normalizer):
        text = "metformin 500 mg when eGFR >= 30 mL/min/1.73m\u00b2"
        result, meta = normalizer.normalize(text)
        assert result == text
        assert meta["fix_count"] == 0


class TestMetadataTracking:
    """Test that fix counts are accurate."""

    def test_fix_count_total(self, normalizer):
        text = "/uniFB01nerenone \u2021 30 rnetformin"
        _, meta = normalizer.normalize(text)
        total = (
            meta["ligature_fixes"]
            + meta["symbol_fixes"]
            + meta["ocr_fixes"]
            + meta["regex_fixes"]
            + meta["whitespace_fixes"]
        )
        assert meta["fix_count"] == total
        assert meta["fix_count"] > 0

    def test_zero_fixes_on_clean(self, normalizer):
        text = "This is clean text with no issues"
        _, meta = normalizer.normalize(text)
        assert meta["fix_count"] == 0


class TestRealDoclingOutput:
    """Test against real Docling output file (if available)."""

    DOCLING_OUTPUT_PATH = Path(__file__).resolve().parents[6] / "KDIGO-2022-Diabetes-CKD-Docling-Output.md"

    @pytest.mark.skipif(
        not DOCLING_OUTPUT_PATH.exists(),
        reason="Real Docling output not available"
    )
    def test_channel_0_on_real_docling_output(self, normalizer):
        """Run normalizer on actual Docling output, assert no residual corruption."""
        text = self.DOCLING_OUTPUT_PATH.read_text(encoding="utf-8")
        normalized, meta = normalizer.normalize(text)

        # Verify no residual Docling ligature escapes
        assert "/uniFB01" not in normalized
        assert "/uniFB02" not in normalized

        # Verify double-dagger symbols replaced
        assert "\u2021" not in normalized

        # Confirm normalizer actually did work
        assert meta["fix_count"] > 0
