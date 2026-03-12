"""
V4 Multi-Channel Extraction Test Fixtures.

Provides shared test data, sample texts, and factory functions for all V4 tests.
Uses real clinical text patterns from KDIGO guidelines — not synthetic.
"""

import sys
from pathlib import Path
from uuid import uuid4

import pytest

# Add shared extraction module to path
SHARED_DIR = Path(__file__).resolve().parents[4]  # -> shared/
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.models import (
    ChannelOutput,
    DrugDossier,
    GuidelineSection,
    GuidelineTree,
    MergedSpan,
    RawSpan,
    ReviewerDecision,
    TableBoundary,
    VerifiedSpan,
)
from extraction.v4.channel_0_normalizer import Channel0Normalizer


# =============================================================================
# Sample Clinical Text (from real KDIGO 2022 Diabetes-CKD patterns)
# =============================================================================

SAMPLE_KDIGO_TEXT = """\
## Chapter 4: Glucose-Lowering Therapies in Patients with T2D and CKD

### Recommendation 4.1.1

We recommend using metformin in patients with T2D and CKD with eGFR >= 30 mL/min/1.73m\u00b2.

When eGFR is 30-44 mL/min/1.73m\u00b2, reduce maximum daily dose of metformin to 1000 mg.

Metformin is contraindicated when eGFR falls below 30 mL/min/1.73m\u00b2.

### Recommendation 4.1.2

We suggest using SGLT2 inhibitors for patients with T2D and CKD.

Dapagliflozin can be initiated when eGFR >= 25 mL/min/1.73m\u00b2 and continued
until dialysis or transplant. Monitor eGFR and potassium every 3-6 months.

### Recommendation 4.2.1

Finerenone is recommended for patients with T2D, CKD, and normal or elevated
potassium. Hold finerenone if potassium > 5.5 mEq/L.
Monitor potassium and eGFR at baseline, within 1 month of initiation,
and every 3-6 months thereafter.

| Drug | eGFR Threshold | Max Dose | Monitoring |
| --- | --- | --- | --- |
| Metformin | >= 30 | 1000 mg (if 30-44) | eGFR every 3-6 months |
| Dapagliflozin | >= 25 (initiate) | 10 mg | eGFR, K+ every 3-6 months |
| Finerenone | No minimum | 20 mg | K+, eGFR at baseline + Q3-6mo |
"""

SAMPLE_KDIGO_TEXT_WITH_LIGATURES = """\
## Chapter 4: Glucose-Lowering Therapies

/uniFB01nerenone is recommended for patients with T2D and CKD.
Dapag/uniFB02ozin has shown bene/uniFB01t in renal outcomes.
The /uniFB01rst-line treatment is metformin when eGFR \u2021 30 mL/min/1.73m2.
"""

SAMPLE_TABLE_MARKDOWN = """\
| Drug | eGFR Threshold | Max Dose | Monitoring Frequency |
| --- | --- | --- | --- |
| Metformin | >= 30 mL/min/1.73m\u00b2 | 1000 mg (if eGFR 30-44) | eGFR every 3-6 months |
| Dapagliflozin | >= 25 mL/min/1.73m\u00b2 | 10 mg daily | eGFR, K+ every 3-6 months |
| Finerenone | No minimum threshold | 20 mg daily | K+, eGFR at baseline + Q3-6mo |
| Canagliflozin | >= 30 mL/min/1.73m\u00b2 | 100 mg daily | eGFR, K+ annually |
"""


# =============================================================================
# Factory Functions
# =============================================================================

@pytest.fixture
def sample_text():
    """Clean KDIGO sample text."""
    return SAMPLE_KDIGO_TEXT


@pytest.fixture
def sample_text_with_ligatures():
    """KDIGO text with Docling ligature corruption."""
    return SAMPLE_KDIGO_TEXT_WITH_LIGATURES


@pytest.fixture
def sample_table_markdown():
    """Markdown table from KDIGO guidelines."""
    return SAMPLE_TABLE_MARKDOWN


@pytest.fixture
def normalizer():
    """Channel 0 normalizer instance."""
    return Channel0Normalizer()


@pytest.fixture
def sample_job_id():
    """A fixed job UUID for test consistency."""
    return uuid4()


@pytest.fixture
def sample_guideline_tree() -> GuidelineTree:
    """A GuidelineTree representing the KDIGO sample structure."""
    return GuidelineTree(
        sections=[
            GuidelineSection(
                section_id="4",
                heading="Chapter 4: Glucose-Lowering Therapies in Patients with T2D and CKD",
                start_offset=0,
                end_offset=len(SAMPLE_KDIGO_TEXT),
                page_number=1,
                block_type="heading",
                children=[
                    GuidelineSection(
                        section_id="4.1.1",
                        heading="Recommendation 4.1.1",
                        start_offset=SAMPLE_KDIGO_TEXT.index("### Recommendation 4.1.1"),
                        end_offset=SAMPLE_KDIGO_TEXT.index("### Recommendation 4.1.2"),
                        page_number=1,
                        block_type="recommendation",
                        children=[],
                    ),
                    GuidelineSection(
                        section_id="4.1.2",
                        heading="Recommendation 4.1.2",
                        start_offset=SAMPLE_KDIGO_TEXT.index("### Recommendation 4.1.2"),
                        end_offset=SAMPLE_KDIGO_TEXT.index("### Recommendation 4.2.1"),
                        page_number=1,
                        block_type="recommendation",
                        children=[],
                    ),
                    GuidelineSection(
                        section_id="4.2.1",
                        heading="Recommendation 4.2.1",
                        start_offset=SAMPLE_KDIGO_TEXT.index("### Recommendation 4.2.1"),
                        end_offset=SAMPLE_KDIGO_TEXT.index("| Drug |"),
                        page_number=1,
                        block_type="recommendation",
                        children=[],
                    ),
                ],
            ),
        ],
        tables=[
            TableBoundary(
                table_id="table_1",
                section_id="4",
                start_offset=SAMPLE_KDIGO_TEXT.index("| Drug |"),
                end_offset=len(SAMPLE_KDIGO_TEXT),
                headers=["Drug", "eGFR Threshold", "Max Dose", "Monitoring"],
                row_count=3,
                page_number=1,
            ),
        ],
        total_pages=1,
    )


@pytest.fixture
def sample_raw_spans(sample_job_id) -> list[RawSpan]:
    """Sample RawSpans from different channels for merger testing."""
    metformin_start = SAMPLE_KDIGO_TEXT.index("metformin in patients")
    dapa_start = SAMPLE_KDIGO_TEXT.index("Dapagliflozin can")
    egfr_start = SAMPLE_KDIGO_TEXT.index("eGFR >= 30")

    return [
        # Channel B: drug dictionary hits
        RawSpan(
            channel="B",
            text="metformin",
            start=metformin_start,
            end=metformin_start + 9,
            confidence=1.0,
            section_id="4.1.1",
            source_block_type="paragraph",
            channel_metadata={"match_type": "exact", "rxnorm_candidate": "860975"},
        ),
        RawSpan(
            channel="B",
            text="Dapagliflozin",
            start=dapa_start,
            end=dapa_start + 13,
            confidence=1.0,
            section_id="4.1.2",
            source_block_type="paragraph",
            channel_metadata={"match_type": "exact", "rxnorm_candidate": "1488564"},
        ),
        # Channel C: regex pattern hits
        RawSpan(
            channel="C",
            text="eGFR >= 30",
            start=egfr_start,
            end=egfr_start + 10,
            confidence=0.95,
            section_id="4.1.1",
            source_block_type="paragraph",
            channel_metadata={"pattern": "egfr_threshold"},
        ),
        # Channel E: GLiNER residual (overlaps with B for metformin — tests merger)
        RawSpan(
            channel="E",
            text="metformin",
            start=metformin_start,
            end=metformin_start + 9,
            confidence=0.78,
            section_id="4.1.1",
            source_block_type="paragraph",
            channel_metadata={"gliner_label": "drug_ingredient"},
        ),
    ]


@pytest.fixture
def sample_merged_spans(sample_job_id) -> list[MergedSpan]:
    """Sample MergedSpans for reviewer testing."""
    metformin_start = SAMPLE_KDIGO_TEXT.index("metformin in patients")
    egfr_start = SAMPLE_KDIGO_TEXT.index("eGFR >= 30")

    return [
        MergedSpan(
            job_id=sample_job_id,
            text="metformin",
            start=metformin_start,
            end=metformin_start + 9,
            contributing_channels=["B", "E"],
            channel_confidences={"B": 1.0, "E": 0.78},
            merged_confidence=0.94,  # boosted for 2 channels
            section_id="4.1.1",
        ),
        MergedSpan(
            job_id=sample_job_id,
            text="eGFR >= 30",
            start=egfr_start,
            end=egfr_start + 10,
            contributing_channels=["C"],
            channel_confidences={"C": 0.95},
            merged_confidence=0.95,
            section_id="4.1.1",
        ),
    ]


@pytest.fixture
def sample_verified_spans() -> list[VerifiedSpan]:
    """Sample VerifiedSpans for dossier assembly testing."""
    metformin_start = SAMPLE_KDIGO_TEXT.index("metformin in patients")
    egfr_start = SAMPLE_KDIGO_TEXT.index("eGFR >= 30")
    contra_start = SAMPLE_KDIGO_TEXT.index("contraindicated when eGFR falls below 30")
    dapa_start = SAMPLE_KDIGO_TEXT.index("Dapagliflozin can")
    finerenone_start = SAMPLE_KDIGO_TEXT.index("Finerenone is recommended")

    return [
        VerifiedSpan(
            text="metformin",
            start=metformin_start,
            end=metformin_start + 9,
            confidence=0.94,
            contributing_channels=["B", "E"],
            section_id="4.1.1",
            extraction_context={"channel_B_rxnorm_candidate": "860975"},
        ),
        VerifiedSpan(
            text="eGFR >= 30",
            start=egfr_start,
            end=egfr_start + 10,
            confidence=0.95,
            contributing_channels=["C"],
            section_id="4.1.1",
            extraction_context={"channel_C_pattern": "egfr_threshold"},
        ),
        VerifiedSpan(
            text="contraindicated when eGFR falls below 30",
            start=contra_start,
            end=contra_start + 41,
            confidence=0.92,
            contributing_channels=["C", "E"],
            section_id="4.1.1",
            extraction_context={"channel_C_pattern": "contraindication"},
        ),
        VerifiedSpan(
            text="Dapagliflozin",
            start=dapa_start,
            end=dapa_start + 13,
            confidence=1.0,
            contributing_channels=["B"],
            section_id="4.1.2",
            extraction_context={"channel_B_rxnorm_candidate": "1488564"},
        ),
        VerifiedSpan(
            text="Finerenone",
            start=finerenone_start,
            end=finerenone_start + 10,
            confidence=1.0,
            contributing_channels=["B"],
            section_id="4.2.1",
            extraction_context={"channel_B_rxnorm_candidate": "2555902"},
        ),
    ]
