#!/usr/bin/env python3
"""
Load L3 seed data into KB-20 via KB20PushClient.

IMPORTANT: Seed JSON files use camelCase Pydantic aliases (drugClass, onsetWindow,
contextualModifiers). The Go API expects snake_case (drug_class, onset_window,
context_modifier_rule as flat JSONB). This loader handles the transform by
parsing through Pydantic (AdverseReactionProfile) to get validated snake_case
dicts via model_dump(), then passing those to KB20PushClient._transform_adr_profiles()
which builds context_modifier_rule JSONB from nested contextual_modifiers and
maps field names to the Go model format.

Usage:
    python seed_loader.py                    # dry run (validate only)
    python seed_loader.py --push             # push to KB-20
    python seed_loader.py --push --kb20-url http://localhost:8131
"""

import argparse
import json
import pathlib
import sys

# Add the shared/ directory to path so extraction.* imports resolve correctly.
# seed_loader.py lives at shared/extraction/v4/seed_loader.py
# parents[2] == shared/  → this is the correct insertion point.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2]))

from extraction.schemas.kb20_contextual import AdverseReactionProfile
from extraction.v4.kb20_push_client import KB20PushClient

SEED_DIR = pathlib.Path(__file__).parent / "l3_seed_data"


def main():
    parser = argparse.ArgumentParser(description="Load L3 seed data into KB-20")
    parser.add_argument("--push", action="store_true", help="Push to KB-20 (default: dry run)")
    parser.add_argument("--kb20-url", default="http://localhost:8131", help="KB-20 base URL")
    args = parser.parse_args()

    seed_files = sorted(SEED_DIR.glob("*.json"))
    if not seed_files:
        print(f"No seed files found in {SEED_DIR}")
        sys.exit(1)

    print(f"Found {len(seed_files)} seed files in {SEED_DIR}")

    # Validate all files first against Pydantic schema
    # Collect snake_case model_dump() dicts so _transform_adr_profiles() gets
    # the correct key format (it expects drug_class, onset_window, etc.)
    valid_profiles = []
    for f in seed_files:
        try:
            with open(f) as fh:
                data = json.load(fh)
            profile = AdverseReactionProfile(**data)
            grade = profile.completeness_grade
            print(f"  OK {f.name}: {grade} ({profile.drug_class} -> {profile.reaction[:40]})")
            valid_profiles.append((f.name, profile.model_dump()))
        except Exception as e:
            print(f"  FAIL {f.name}: {e}")

    print(f"\nValidated: {len(valid_profiles)}/{len(seed_files)} files")

    if not args.push:
        print("\nDry run complete. Use --push to write to KB-20.")
        return

    # Push to KB-20 as a single batch
    client = KB20PushClient(base_url=args.kb20_url)
    if not client.health_check():
        print(f"\nKB-20 not reachable at {args.kb20_url}")
        sys.exit(1)

    # Transform from Pydantic snake_case model_dump() -> Go snake_case format.
    # _transform_adr_profiles() builds context_modifier_rule JSONB from the
    # nested contextual_modifiers list and maps field names to the Go model.
    all_snake = [snake_dict for _, snake_dict in valid_profiles]
    governance = {"authority": "MANUAL_CURATED", "document": "L3 Seed Data"}
    go_records = client._transform_adr_profiles(all_snake, governance, source="MANUAL_CURATED")

    print(f"\nPushing {len(go_records)} records as single batch...")
    result = client._post_batch("/api/v1/pipeline/adr-profiles", go_records)
    succeeded = result.get("succeeded", 0)
    failed = result.get("failed", 0)
    if result.get("errors"):
        for err in result["errors"][:5]:
            print(f"  WARNING  {err}")

    print(f"\nPush complete: {succeeded} OK, {failed} failed")


if __name__ == "__main__":
    main()
