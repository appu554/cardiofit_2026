"""Pytest configuration for L3 seed data tests.

Adds the shared/ directory to sys.path so that extraction.* imports resolve
correctly, matching the same path convention used by the V4 pipeline scripts.
"""

import sys
from pathlib import Path

# seed data lives at: shared/extraction/v4/l3_seed_data/
# parents: [0]=l3_seed_data, [1]=v4, [2]=extraction, [3]=shared
SHARED_DIR = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(SHARED_DIR))
