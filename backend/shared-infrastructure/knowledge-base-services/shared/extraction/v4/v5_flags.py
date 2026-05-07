"""V5 feature flag resolver.

Single source of truth for V5_<feature> on/off decisions. Precedence:

    1. V5_DISABLE_ALL=1 in env  -> always False (emergency rollback)
    2. profile.v5_features[feature] is True/False  -> profile wins
    3. profile.v5_features[feature] is None        -> fall through
    4. env V5_<FEATURE>=0 (or any non-"1" value)   -> False (explicit opt-out)
    5. anything else                               -> True (default-ON)

All V5 subsystems are ON by default. To disable a specific feature set the
corresponding env var to "0" (e.g. V5_DECOMPOSITION=0). Emergency rollback
via V5_DISABLE_ALL=1 forces every feature off regardless of other settings.

`profile` may be any object; if it lacks `v5_features`, treat as empty dict.

Recognised flags (non-exhaustive — any string works because the resolver is
generic, but these are the ones the pipeline reads today):

    bbox_provenance        - per-channel ChannelProvenance on MergedSpan
    consensus_entropy      - CE gate on conflicting cluster votes
    decomposition          - sentence-level proposition graph
    schema_first           - schema validator gate before merge
    table_specialist       - umbrella: enables Channel D's V5 lane chain
    vlm_table_specialist   - lane: MonkeyOCR Qwen2.5-VL cell_data path
    nemotron_parse         - lane: NVIDIA Nemotron Parse v1.1 — table
                             specialist with two backends:
                               * NEMOTRON_PARSE_URL → self-hosted sidecar
                                 (preferred — see docker-compose.specialists.yml,
                                 runs nvidia/NVIDIA-Nemotron-Parse-v1.1-TC).
                               * NVIDIA_API_KEY     → NIM cloud fallback
                                 (no GPU required, per-call cost).
                             Sidecar wins when both are set. Lane reports
                             Unavailable and falls through silently when
                             neither env var is set, so flipping this flag
                             on without infra is safe.
    figure_specialist      - lane: Nemotron Nano VL 8B sidecar (figures /
                             algorithms / flowcharts). Requires NEMOTRON_VL_URL
                             env var pointing at the sidecar; falls through
                             when unset.
    lightonocr             - lane: LightOnOCR-2-1B-bbox sidecar (body OCR
                             with per-word bbox → FieldProvenance for NER
                             channels). Requires LIGHTONOCR_URL env var.

Lane priority within Channel D (table specialist):
    nemotron_parse > vlm_table_specialist > docling
"""
from __future__ import annotations

import os
from typing import Any


def is_v5_enabled(feature: str, profile: Any) -> bool:
    """Resolve a V5 feature flag with profile-override > env-var > default-ON.

    Args:
        feature: lowercase feature name, e.g. "bbox_provenance".
        profile: object with optional `v5_features` dict attribute.

    Returns:
        True iff the resolved value is on (defaults to True).
    """
    # Emergency rollback always wins.
    if os.environ.get("V5_DISABLE_ALL") == "1":
        return False

    # Profile override (if present and not None) wins over env.
    overrides = getattr(profile, "v5_features", None) or {}
    profile_value = overrides.get(feature)
    if profile_value is True:
        return True
    if profile_value is False:
        return False

    # Env var fallback — default is "1" (ON); set to "0" to disable.
    env_value = os.environ.get(f"V5_{feature.upper()}", "1")
    return env_value == "1"
