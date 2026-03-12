#!/usr/bin/env python3
"""
Load EXPLICIT (Extensional) ValueSets into precomputed_valueset_codes.

These ValueSets have codes directly in compose.include.concept[] -
no Neo4j traversal needed. Just parse JSON and INSERT into PostgreSQL.

This completes the data loading for CQL integration:
- Intensional ValueSets (864): Loaded via materialize_expansions.py (Neo4j)
- Explicit ValueSets (~17,795): Loaded via THIS SCRIPT (direct JSON parse)

Usage:
    python load_explicit_valuesets.py                    # Load all
    python load_explicit_valuesets.py --limit 100       # Test with limited set
    python load_explicit_valuesets.py --dry-run         # Preview only
"""

import os
import sys
import json
import argparse
import psycopg2
from psycopg2.extras import execute_values
from pathlib import Path
from datetime import datetime
from typing import Optional, List, Tuple, Dict
from tqdm import tqdm

# ============================================================================
# Configuration
# ============================================================================

DATA_DIR = Path(__file__).parent.parent.parent / "data" / "ontoserver-valuesets"
DEFINITIONS_DIR = DATA_DIR / "definitions"
EXPANSIONS_DIR = DATA_DIR / "expansions"

# SNOMED Release Version - must match materialization job
SNOMED_RELEASE = os.getenv("SNOMED_RELEASE", "20241130")
SNOMED_SYSTEM = "http://snomed.info/sct"

# PostgreSQL config
POSTGRES_CONFIG = {
    "host": os.getenv("POSTGRES_HOST", "localhost"),
    "port": int(os.getenv("POSTGRES_PORT", "5432")),
    "database": os.getenv("POSTGRES_DB", "kb_terminology"),
    "user": os.getenv("POSTGRES_USER", "postgres"),
    "password": os.getenv("POSTGRES_PASSWORD", "password")
}

# ============================================================================
# Explicit ValueSet Loader
# ============================================================================

class ExplicitValueSetLoader:
    """
    Load explicit (extensional) ValueSets directly from JSON into PostgreSQL.

    These ValueSets have codes in compose.include.concept[] - no Neo4j needed.
    We extract codes from:
    1. Definition JSON: compose.include.concept[]
    2. Expansion JSON: expansion.contains[] (pre-expanded by Ontoserver)
    """

    def __init__(self, dry_run: bool = False):
        self.dry_run = dry_run
        self.pg_conn = None
        self.stats = {
            "valuesets_processed": 0,
            "valuesets_loaded": 0,
            "valuesets_skipped": 0,
            "codes_inserted": 0,
            "errors": []
        }

    def connect(self):
        """Connect to PostgreSQL."""
        if self.dry_run:
            print("DRY RUN MODE - No database changes will be made")
            return

        self.pg_conn = psycopg2.connect(**POSTGRES_CONFIG)
        self.pg_conn.autocommit = False
        print(f"Connected to PostgreSQL: {POSTGRES_CONFIG['host']}:{POSTGRES_CONFIG['port']}")

    def close(self):
        """Close database connection."""
        if self.pg_conn:
            self.pg_conn.close()

    def get_valuesets_needing_explicit_load(self) -> List[Dict]:
        """
        Get ValueSets that need explicit loading.
        These are ValueSets that:
        1. Exist in value_sets table
        2. Have NO root_code (not intensional)
        3. Have NOT been loaded into precomputed_valueset_codes yet
        """
        if self.dry_run:
            # In dry run, just return list of JSON files
            return []

        cursor = self.pg_conn.cursor()

        # Get ValueSets without precomputed codes
        query = """
            SELECT vs.id, vs.url, vs.name
            FROM value_sets vs
            LEFT JOIN precomputed_valueset_codes pvc
                ON vs.url = pvc.valueset_url AND pvc.snomed_version = %s
            WHERE pvc.id IS NULL
            ORDER BY vs.name
        """
        cursor.execute(query, (SNOMED_RELEASE,))
        results = cursor.fetchall()
        cursor.close()

        return [{"id": r[0], "url": r[1], "name": r[2]} for r in results]

    def extract_codes_from_definition(self, definition: Dict) -> List[Tuple[str, str, str]]:
        """
        Extract codes from ValueSet definition JSON.
        Returns: List of (system, code, display) tuples
        """
        codes = []
        compose = definition.get("compose", {})

        for include in compose.get("include", []):
            system = include.get("system", SNOMED_SYSTEM)

            # Extract explicit concept list
            for concept in include.get("concept", []):
                code = concept.get("code")
                display = concept.get("display", "")
                if code:
                    codes.append((system, code, display))

        return codes

    def extract_codes_from_expansion(self, expansion: Dict) -> List[Tuple[str, str, str]]:
        """
        Extract codes from pre-expanded ValueSet JSON (from Ontoserver).
        Returns: List of (system, code, display) tuples
        """
        codes = []
        exp = expansion.get("expansion", {})

        for concept in exp.get("contains", []):
            system = concept.get("system", SNOMED_SYSTEM)
            code = concept.get("code")
            display = concept.get("display", "")
            if code:
                codes.append((system, code, display))

        return codes

    def load_valueset_codes(self, valueset_url: str, valueset_id: str,
                           codes: List[Tuple[str, str, str]]) -> int:
        """
        Load codes into precomputed_valueset_codes table.
        """
        if not codes:
            return 0

        if self.dry_run:
            return len(codes)

        cursor = self.pg_conn.cursor()

        # Prepare data for bulk insert
        data = [
            (valueset_url, valueset_id, SNOMED_RELEASE, system, code, display)
            for system, code, display in codes
        ]

        # Bulk insert with conflict handling
        query = """
            INSERT INTO precomputed_valueset_codes
            (valueset_url, valueset_id, snomed_version, code_system, code, display)
            VALUES %s
            ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING
        """
        execute_values(cursor, query, data)
        inserted = cursor.rowcount if cursor.rowcount > 0 else len(codes)

        self.pg_conn.commit()
        return inserted

    def process_json_files(self, limit: Optional[int] = None):
        """
        Process JSON files directly to load explicit ValueSets.
        This handles both definition and expansion files.
        """
        print(f"\n{'='*70}")
        print(f"  EXPLICIT VALUESET LOADER - Direct JSON to PostgreSQL")
        print(f"{'='*70}")
        print(f"  Source: {DEFINITIONS_DIR}")
        print(f"  SNOMED Release: {SNOMED_RELEASE}")
        print(f"  Limit: {limit or 'None'}")
        print(f"  Dry Run: {self.dry_run}")
        print(f"{'='*70}\n")

        self.connect()

        # Get all definition files
        def_files = sorted(DEFINITIONS_DIR.glob("*.json"))
        if limit:
            def_files = def_files[:limit]

        total = len(def_files)
        print(f"Found {total:,} ValueSet definition files\n")

        # Progress bar
        progress = tqdm(def_files, desc="Loading", unit="vs")

        for def_file in progress:
            vs_id = def_file.stem
            self.stats["valuesets_processed"] += 1
            progress.set_postfix_str(f"{vs_id[:30]}...")

            try:
                # Load definition JSON
                with open(def_file, 'r') as f:
                    definition = json.load(f)

                vs_url = definition.get("url", "")
                vs_name = definition.get("name", vs_id)

                # Skip if already has precomputed codes (check in real mode)
                if not self.dry_run:
                    cursor = self.pg_conn.cursor()
                    cursor.execute("""
                        SELECT COUNT(*) FROM precomputed_valueset_codes
                        WHERE valueset_url = %s AND snomed_version = %s
                    """, (vs_url, SNOMED_RELEASE))
                    existing = cursor.fetchone()[0]
                    cursor.close()

                    if existing > 0:
                        self.stats["valuesets_skipped"] += 1
                        continue

                # Try to get codes from multiple sources
                codes = []

                # Source 1: Definition compose.include.concept[]
                codes = self.extract_codes_from_definition(definition)

                # Source 2: If no codes in definition, try expansion file
                if not codes:
                    exp_file = EXPANSIONS_DIR / f"{vs_id}_expanded.json"
                    if exp_file.exists():
                        with open(exp_file, 'r') as f:
                            expansion = json.load(f)
                        codes = self.extract_codes_from_expansion(expansion)

                if not codes:
                    # No codes found - this might be intensional (handled by materializer)
                    self.stats["valuesets_skipped"] += 1
                    continue

                # Get valueset_id from database
                db_vs_id = None
                if not self.dry_run:
                    cursor = self.pg_conn.cursor()
                    cursor.execute("""
                        SELECT id::text FROM value_sets WHERE url = %s LIMIT 1
                    """, (vs_url,))
                    result = cursor.fetchone()
                    if result:
                        db_vs_id = result[0]
                    cursor.close()

                # Load codes
                inserted = self.load_valueset_codes(vs_url, db_vs_id, codes)

                if inserted > 0:
                    self.stats["valuesets_loaded"] += 1
                    self.stats["codes_inserted"] += inserted

            except Exception as e:
                self.stats["errors"].append({
                    "valueset": vs_id,
                    "error": str(e)
                })
                continue

        self.close()
        self._print_summary()

    def _print_summary(self):
        """Print loading summary."""
        print(f"\n{'='*70}")
        print(f"  EXPLICIT VALUESET LOADING COMPLETE!")
        print(f"{'='*70}")
        print(f"  SNOMED Release: {SNOMED_RELEASE}")
        print(f"  ValueSets Processed:  {self.stats['valuesets_processed']:,}")
        print(f"  ValueSets Loaded:     {self.stats['valuesets_loaded']:,}")
        print(f"  ValueSets Skipped:    {self.stats['valuesets_skipped']:,}")
        print(f"  Total Codes Inserted: {self.stats['codes_inserted']:,}")
        print(f"  Errors:               {len(self.stats['errors'])}")
        print(f"{'='*70}\n")

        if self.stats['errors']:
            print(f"First {min(10, len(self.stats['errors']))} errors:")
            for err in self.stats['errors'][:10]:
                print(f"  - {err['valueset']}: {err['error']}")

        print("\nNEXT STEPS:")
        print("  1. Verify total codes: SELECT COUNT(*) FROM precomputed_valueset_codes;")
        print("  2. Test $expand: curl http://localhost:8087/fhir/ValueSet/<name>/$expand")
        print("  3. All ValueSets should now return precomputed codes!")


# ============================================================================
# Main Entry Point
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Load EXPLICIT (Extensional) ValueSets into precomputed_valueset_codes"
    )
    parser.add_argument(
        "--limit", "-l",
        type=int,
        help="Limit number of ValueSets to process (for testing)"
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Preview without making database changes"
    )
    parser.add_argument(
        "--snomed-version",
        default=os.getenv("SNOMED_RELEASE", "20241130"),
        help="SNOMED release version (e.g., 20241130)"
    )
    parser.add_argument(
        "--pg-host",
        default=os.getenv("POSTGRES_HOST", "localhost"),
        help="PostgreSQL host"
    )
    parser.add_argument(
        "--pg-port",
        type=int,
        default=int(os.getenv("POSTGRES_PORT", "5432")),
        help="PostgreSQL port"
    )

    args = parser.parse_args()

    # Update config from args
    global SNOMED_RELEASE
    SNOMED_RELEASE = args.snomed_version
    POSTGRES_CONFIG["host"] = args.pg_host
    POSTGRES_CONFIG["port"] = args.pg_port

    # Check if data exists
    if not DEFINITIONS_DIR.exists():
        print(f"ERROR: Definitions directory not found: {DEFINITIONS_DIR}")
        print("Run download_valuesets.py first!")
        sys.exit(1)

    # Run loader
    loader = ExplicitValueSetLoader(dry_run=args.dry_run)
    loader.process_json_files(limit=args.limit)


if __name__ == "__main__":
    main()
