"""V5 feature flag resolver.

Single source of truth for V5_<feature> on/off decisions. Precedence:

    1. V5_DISABLE_ALL=1 in env  -> always False (emergency rollback)
    2. profile.v5_features[feature] is True/False  -> profile wins
    3. profile.v5_features[feature] is None        -> fall through
    4. env V5_<FEATURE>=1                          -> True
    5. anything else                               -> False (default-off)

`profile` may be any object; if it lacks `v5_features`, treat as empty dict.
"""
from __future__ import annotations

import os
from typing import Any


def is_v5_enabled(feature: str, profile: Any) -> bool:
    """Resolve a V5 feature flag with profile-override > env-var > default-off.

    Args:
        feature: lowercase feature name, e.g. "bbox_provenance".
        profile: object with optional `v5_features` dict attribute.

    Returns:
        True iff the resolved value is on.
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

    # Env var fallback.
    env_value = os.environ.get(f"V5_{feature.upper()}", "0")
    return env_value == "1"
