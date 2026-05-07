"""Governance promoter (Stage 5) — Layer 3 Wave 1 Task 5.

Orchestrates the Stage 5 promotion workflow for a Layer 3 rule:

  1. Two-gate validation (Stage 1 + Stage 2)
  2. CompatibilityChecker shows the rule as ACTIVE
  3. Compute content_sha over the rule package
     (CQL define text + rule_spec YAML + sorted helper list)
  4. Solicit dual signatures: clinical reviewer + medical director
     (CLI flags / env vars for MVP — real signing UI is Layer 4 work)
  5. Sign the package with platform Ed25519 key
  6. Write the signed package to KB-4 governance/signed/ as a JSON
     manifest (kb-4 governance HTTP API integration deferred to V1)
  7. Emit an EvidenceTrace `rule_publication` node (written to
     kb-4-patient-safety/governance/evidence_trace_pending/<rule>.json
     for an out-of-band kb-20 ingester to pick up — kb-20 KB20Client
     direct-call deferred to V1)

KB-20 KB20Client integration is intentionally deferred (Wave 1 Task 5
plan provision: write to manifest if KB20Client unavailable from
Python). The on-disk manifest is the contract for the kb-20 ingester.

Wave 1 Task 5 — see plan.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import time
import uuid
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml
from cryptography.hazmat.primitives.asymmetric.ed25519 import (
    Ed25519PrivateKey,
)

from compatibility_checker import (  # type: ignore[import-not-found]
    CompatibilityChecker,
    CompatStatus,
)
from rule_specification_validator import load_spec  # type: ignore[import-not-found]
from two_gate_validator import (  # type: ignore[import-not-found]
    _extract_define_body,
    run_two_gate,
)


# ---------------------------------------------------------------------------
# Output paths
# ---------------------------------------------------------------------------

_HERE = Path(__file__).resolve()
_KB_ROOT = _HERE.parents[2].parent  # knowledge-base-services/
DEFAULT_SIGNED_DIR = (
    _KB_ROOT / "kb-4-patient-safety" / "governance" / "signed"
)
DEFAULT_PENDING_DIR = (
    _KB_ROOT / "kb-4-patient-safety" / "governance" / "evidence_trace_pending"
)


# ---------------------------------------------------------------------------
# Inputs
# ---------------------------------------------------------------------------


@dataclass
class Signature:
    role: str  # "CLINICAL_REVIEWER" or "MEDICAL_DIRECTOR"
    signer_id: str
    signature_b64: str = ""

    def is_valid(self) -> bool:
        return bool(self.role) and bool(self.signer_id)


@dataclass
class PromotionResult:
    ok: bool
    rule_id: str
    content_sha: str
    signed_package_path: Path | None
    evidence_trace_path: Path | None
    errors: list[str] = field(default_factory=list)


# ---------------------------------------------------------------------------
# Promoter
# ---------------------------------------------------------------------------


@dataclass
class GovernancePromoter:
    signing_key: Ed25519PrivateKey
    signed_dir: Path = field(default_factory=lambda: DEFAULT_SIGNED_DIR)
    pending_dir: Path = field(default_factory=lambda: DEFAULT_PENDING_DIR)
    kb20_client: Any = None  # injected mock or real client

    def __post_init__(self) -> None:
        self.signed_dir.mkdir(parents=True, exist_ok=True)
        self.pending_dir.mkdir(parents=True, exist_ok=True)

    # ------------------------------------------------------------------
    # Main entry
    # ------------------------------------------------------------------

    def promote(
        self,
        spec_path: Path,
        cql_library_path: Path,
        signatures: list[Signature],
        compatibility_checker: CompatibilityChecker | None = None,
    ) -> PromotionResult:
        spec = load_spec(spec_path)
        rule_id = spec["rule_id"]
        cql_library_text = cql_library_path.read_text()
        cql_body = _extract_define_body(cql_library_text, spec["define"])

        errors: list[str] = []

        # 1. Two-gate validation
        gate_result = run_two_gate(spec, cql_body)
        if not gate_result.ok:
            errors.extend(str(e) for e in gate_result.errors)

        # 2. CompatibilityChecker
        cc = compatibility_checker or CompatibilityChecker()
        cc.register(spec, cql_body)
        cc.OnRuleUpdate(spec, cql_body)
        if cc.status_of(rule_id) != CompatStatus.ACTIVE:
            errors.append(
                f"CompatibilityChecker reports rule '{rule_id}' as "
                f"{cc.status_of(rule_id).value}; cannot promote"
            )

        # 3. Dual-signature precondition
        roles = {s.role for s in signatures if s.is_valid()}
        required = {"CLINICAL_REVIEWER", "MEDICAL_DIRECTOR"}
        missing = required - roles
        if missing:
            errors.append(
                f"missing required signature roles: {sorted(missing)}"
            )

        if errors:
            return PromotionResult(
                ok=False,
                rule_id=rule_id,
                content_sha="",
                signed_package_path=None,
                evidence_trace_path=None,
                errors=errors,
            )

        # 4. Compute content_sha
        helper_names = sorted(_extract_helper_calls(cql_body))
        content_sha = _compute_content_sha(spec, cql_body, helper_names)
        spec["content_sha"] = content_sha

        # 5. Sign
        platform_signature_b64 = _ed25519_sign_b64(
            self.signing_key, content_sha.encode()
        )

        package = {
            "rule_id": rule_id,
            "version": _spec_version(spec),
            "content_sha": content_sha,
            "rule_specification": spec,
            "cql_define_body": cql_body,
            "helper_calls": helper_names,
            "platform_signature_b64": platform_signature_b64,
            "approver_signatures": [
                {
                    "role": s.role,
                    "signer_id": s.signer_id,
                    "signature_b64": s.signature_b64,
                }
                for s in signatures
            ],
            "promoted_at": _utc_now_iso(),
        }

        # 6. Write signed package (kb-4 HTTP API deferred to V1)
        signed_filename = f"{rule_id}-{package['version']}.json"
        signed_path = self.signed_dir / signed_filename
        signed_path.write_text(json.dumps(package, indent=2, sort_keys=True))

        # 7. Emit EvidenceTrace rule_publication node
        evidence_node = {
            "node_type": "rule_publication",
            "rule_id": rule_id,
            "content_sha": content_sha,
            "version": package["version"],
            "platform_signature_b64": platform_signature_b64,
            "approvers": [
                {"role": s.role, "signer_id": s.signer_id}
                for s in signatures
            ],
            "occurred_at": package["promoted_at"],
        }

        evidence_path: Path | None = None
        if self.kb20_client is not None:
            # Real or mock kb-20 client path
            self.kb20_client.UpsertEvidenceTraceNode(evidence_node)
        else:
            # Out-of-band ingester path
            evidence_filename = (
                f"{rule_id}-{package['version']}-{uuid.uuid4().hex[:8]}.json"
            )
            evidence_path = self.pending_dir / evidence_filename
            evidence_path.write_text(json.dumps(evidence_node, indent=2))

        return PromotionResult(
            ok=True,
            rule_id=rule_id,
            content_sha=content_sha,
            signed_package_path=signed_path,
            evidence_trace_path=evidence_path,
        )


# ---------------------------------------------------------------------------
# Hashing / signing
# ---------------------------------------------------------------------------


def _compute_content_sha(
    spec: dict[str, Any], cql_body: str, helper_names: list[str]
) -> str:
    """Canonical SHA over (rule_spec - content_sha) + cql_body + helper list."""
    spec_for_hash = dict(spec)
    spec_for_hash.pop("content_sha", None)
    payload = json.dumps(
        {"spec": spec_for_hash, "cql": cql_body, "helpers": helper_names},
        sort_keys=True,
        separators=(",", ":"),
    ).encode()
    return hashlib.sha256(payload).hexdigest()


def _ed25519_sign_b64(key: Ed25519PrivateKey, data: bytes) -> str:
    import base64

    return base64.b64encode(key.sign(data)).decode()


# ---------------------------------------------------------------------------
# Misc helpers
# ---------------------------------------------------------------------------


def _extract_helper_calls(cql_body: str) -> set[str]:
    """Find HelperLibrary."HelperName"() patterns in the CQL body."""
    pat = re.compile(r'\b[A-Za-z_]+\."([A-Z][A-Za-z0-9_]+)"\s*\(')
    return {m.group(1) for m in pat.finditer(cql_body)}


def _spec_version(spec: dict[str, Any]) -> str:
    return spec.get("schema_version", "2.0")


def _utc_now_iso() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


# ---------------------------------------------------------------------------
# CLI shim
# ---------------------------------------------------------------------------


def signatures_from_env() -> list[Signature]:
    """Pick up reviewer signatures from env vars (MVP entrypoint)."""
    sigs = []
    for role in ("CLINICAL_REVIEWER", "MEDICAL_DIRECTOR"):
        signer = os.environ.get(f"L3_PROMOTE_{role}_ID")
        if signer:
            sigs.append(
                Signature(
                    role=role,
                    signer_id=signer,
                    signature_b64=os.environ.get(
                        f"L3_PROMOTE_{role}_SIG_B64", ""
                    ),
                )
            )
    return sigs


def load_signing_key_from_env() -> Ed25519PrivateKey:
    import base64

    raw = os.environ.get("L3_PROMOTE_PLATFORM_KEY_B64")
    if not raw:
        raise RuntimeError(
            "L3_PROMOTE_PLATFORM_KEY_B64 env var (base64-encoded 32-byte "
            "Ed25519 private key seed) is required"
        )
    seed = base64.b64decode(raw)
    return Ed25519PrivateKey.from_private_bytes(seed)
