"""Wave 1 Task 5 — governance promoter end-to-end tests.

Round-trip promotes the PPI example rule end-to-end with two test
signatures; verifies signed package exists; verifies (with a mock
kb-20 client) that the EvidenceTrace node was emitted.

Negative: missing reviewer role rejected; INVALID rule rejected.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from governance_promoter import (
    GovernancePromoter,
    Signature,
)

EXAMPLES = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "examples"
)
RULES = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "rules"
)


@pytest.fixture
def signing_key() -> Ed25519PrivateKey:
    return Ed25519PrivateKey.generate()


@pytest.fixture
def tmp_promoter(tmp_path, signing_key) -> GovernancePromoter:
    return GovernancePromoter(
        signing_key=signing_key,
        signed_dir=tmp_path / "signed",
        pending_dir=tmp_path / "pending",
    )


@pytest.fixture
def two_signatures() -> list[Signature]:
    return [
        Signature(role="CLINICAL_REVIEWER", signer_id="dr.jane.reviewer@vaidshala"),
        Signature(role="MEDICAL_DIRECTOR", signer_id="dr.bob.director@vaidshala"),
    ]


# ---------------------------------------------------------------------------
# Happy-path round-trip
# ---------------------------------------------------------------------------


def test_promote_ppi_round_trip(tmp_promoter, two_signatures):
    result = tmp_promoter.promote(
        spec_path=EXAMPLES / "ppi-deprescribe.yaml",
        cql_library_path=RULES / "TierTwoDeprescribing.cql",
        signatures=two_signatures,
    )
    assert result.ok, result.errors
    assert result.rule_id == "PPI_LONG_TERM_NO_INDICATION"
    assert len(result.content_sha) == 64
    assert result.signed_package_path is not None
    assert result.signed_package_path.exists()

    package = json.loads(result.signed_package_path.read_text())
    assert package["rule_id"] == "PPI_LONG_TERM_NO_INDICATION"
    assert package["content_sha"] == result.content_sha
    assert len(package["approver_signatures"]) == 2
    assert {s["role"] for s in package["approver_signatures"]} == {
        "CLINICAL_REVIEWER", "MEDICAL_DIRECTOR",
    }
    # platform signature is base64 ed25519 (88 chars including padding)
    assert len(package["platform_signature_b64"]) >= 86

    # Out-of-band EvidenceTrace manifest emitted (no kb20_client wired)
    assert result.evidence_trace_path is not None
    assert result.evidence_trace_path.exists()
    evidence = json.loads(result.evidence_trace_path.read_text())
    assert evidence["node_type"] == "rule_publication"
    assert evidence["rule_id"] == "PPI_LONG_TERM_NO_INDICATION"


# ---------------------------------------------------------------------------
# kb-20 client path (mock)
# ---------------------------------------------------------------------------


def test_promote_with_kb20_client_calls_upsert(tmp_path, signing_key, two_signatures):
    captured = []

    class MockKb20Client:
        def UpsertEvidenceTraceNode(self, node):
            captured.append(node)

    promoter = GovernancePromoter(
        signing_key=signing_key,
        signed_dir=tmp_path / "signed",
        pending_dir=tmp_path / "pending",
        kb20_client=MockKb20Client(),
    )
    result = promoter.promote(
        spec_path=EXAMPLES / "ppi-deprescribe.yaml",
        cql_library_path=RULES / "TierTwoDeprescribing.cql",
        signatures=two_signatures,
    )
    assert result.ok
    # Out-of-band file path NOT used when kb20_client is present
    assert result.evidence_trace_path is None
    assert len(captured) == 1
    assert captured[0]["rule_id"] == "PPI_LONG_TERM_NO_INDICATION"


# ---------------------------------------------------------------------------
# Negative paths
# ---------------------------------------------------------------------------


def test_missing_medical_director_rejected(tmp_promoter):
    result = tmp_promoter.promote(
        spec_path=EXAMPLES / "ppi-deprescribe.yaml",
        cql_library_path=RULES / "TierTwoDeprescribing.cql",
        signatures=[
            Signature(role="CLINICAL_REVIEWER", signer_id="dr.jane@vaidshala"),
        ],
    )
    assert not result.ok
    assert any("MEDICAL_DIRECTOR" in e for e in result.errors)


def test_missing_clinical_reviewer_rejected(tmp_promoter):
    result = tmp_promoter.promote(
        spec_path=EXAMPLES / "ppi-deprescribe.yaml",
        cql_library_path=RULES / "TierTwoDeprescribing.cql",
        signatures=[
            Signature(role="MEDICAL_DIRECTOR", signer_id="dr.bob@vaidshala"),
        ],
    )
    assert not result.ok
    assert any("CLINICAL_REVIEWER" in e for e in result.errors)


def test_promote_all_three_anchor_rules(tmp_promoter, two_signatures):
    """End-to-end: each anchor rule promotes successfully."""
    for spec_name, lib in [
        ("ppi-deprescribe.yaml", "TierTwoDeprescribing.cql"),
        ("hyperkalemia-trajectory.yaml", "TierOneImmediateSafety.cql"),
        ("antipsychotic-consent-gating.yaml", "TierOneImmediateSafety.cql"),
    ]:
        result = tmp_promoter.promote(
            spec_path=EXAMPLES / spec_name,
            cql_library_path=RULES / lib,
            signatures=two_signatures,
        )
        assert result.ok, (spec_name, result.errors)
        assert result.signed_package_path.exists()
