#!/usr/bin/env python3
"""
Load ALL Ontoserver ValueSet Expansions into precomputed_valueset_codes.

This is the COMPREHENSIVE loader that uses EXPANSION-FIRST pattern:
1. Iterates through ALL 22,003 expansion files (NOT definitions!)
2. Extracts codes from expansion.contains[]
3. Gets valueset_url from corresponding definition file
4. Bulk inserts into PostgreSQL

CTO/CMO DIRECTIVE:
"CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."

NO Neo4j traversal - pure JSON → PostgreSQL loading.
Ontoserver has ALREADY computed expansions - we just load them.

Usage:
    python load_all_expansions.py                    # Load all
    python load_all_expansions.py --limit 100       # Test with limited set
    python load_all_expansions.py --dry-run         # Preview only
    python load_all_expansions.py --fresh           # Truncate and reload
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

# SNOMED Release Version - must match other scripts
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

# Batch size for bulk inserts
BATCH_SIZE = 1000

# ============================================================================
# Expansion-First Loader
# ============================================================================

class ExpansionFirstLoader:
    """
    Load ALL Ontoserver expansions using EXPANSION-FIRST pattern.

    Key Design Decisions:
    1. Iterate over EXPANSION files, not definitions
    2. Every expansion file = precomputed codes by Ontoserver
    3. No Neo4j, no ECL evaluation, no hierarchy traversal
    4. Pure JSON → PostgreSQL bulk insert
    """

    def __init__(self, dry_run: bool = False, fresh: bool = False):
        self.dry_run = dry_run
        self.fresh = fresh
        self.pg_conn = None
        self.stats = {
            "expansions_found": 0,
            "expansions_processed": 0,
            "expansions_loaded": 0,
            "expansions_skipped": 0,
            "expansions_empty": 0,
            "codes_inserted": 0,
            "errors": []
        }
        # Cache for definition lookups
        self._definition_cache: Dict[str, Dict] = {}

    def connect(self):
        """Connect to PostgreSQL."""
        if self.dry_run:
            print("DRY RUN MODE - No database changes will be made")
            return

        self.pg_conn = psycopg2.connect(**POSTGRES_CONFIG)
        self.pg_conn.autocommit = False
        print(f"Connected to PostgreSQL: {POSTGRES_CONFIG['host']}:{POSTGRES_CONFIG['port']}")

        if self.fresh:
            print("\n⚠️  FRESH MODE: Truncating precomputed_valueset_codes...")
            cursor = self.pg_conn.cursor()
            cursor.execute("TRUNCATE TABLE precomputed_valueset_codes")
            self.pg_conn.commit()
            print("✓ Table truncated")

    def close(self):
        """Close database connection."""
        if self.pg_conn:
            self.pg_conn.close()

    def get_definition(self, valueset_id: str) -> Optional[Dict]:
        """
        Get definition JSON for a ValueSet ID.
        Uses caching to avoid repeated file reads.
        """
        if valueset_id in self._definition_cache:
            return self._definition_cache[valueset_id]

        def_file = DEFINITIONS_DIR / f"{valueset_id}.json"
        if def_file.exists():
            with open(def_file, 'r') as f:
                definition = json.load(f)
                self._definition_cache[valueset_id] = definition
                return definition
        return None

    def extract_oid(self, definition: Dict) -> Optional[str]:
        """Extract OID from ValueSet identifier array."""
        for ident in definition.get("identifier", []):
            system = ident.get("system", "")
            value = ident.get("value", "")
            if "oid" in system.lower() and value:
                # Strip urn:oid: prefix if present
                return value.replace("urn:oid:", "")
        return None

    def extract_codes_from_expansion(self, expansion: Dict) -> List[Tuple[str, str, str]]:
        """
        Extract codes from pre-expanded ValueSet JSON.
        Returns: List of (system, code, display) tuples
        """
        codes = []
        exp_section = expansion.get("expansion", {})

        for concept in exp_section.get("contains", []):
            system = concept.get("system", SNOMED_SYSTEM)
            code = concept.get("code")
            display = concept.get("display", "")

            if code:
                codes.append((system, code, display))

        return codes

    def valueset_already_loaded(self, valueset_url: str) -> bool:
        """Check if ValueSet already has precomputed codes (idempotency guard)."""
        if self.dry_run:
            return False

        cursor = self.pg_conn.cursor()
        cursor.execute("""
            SELECT COUNT(*) FROM precomputed_valueset_codes
            WHERE valueset_url = %s AND snomed_version = %s
        """, (valueset_url, SNOMED_RELEASE))
        count = cursor.fetchone()[0]
        cursor.close()
        return count > 0

    def get_valueset_db_id(self, valueset_url: str) -> Optional[str]:
        """Get UUID from value_sets table if exists."""
        if self.dry_run:
            return None

        cursor = self.pg_conn.cursor()
        cursor.execute("""
            SELECT id::text FROM value_sets WHERE url = %s LIMIT 1
        """, (valueset_url,))
        result = cursor.fetchone()
        cursor.close()
        return result[0] if result else None

    def bulk_insert_codes(self, valueset_url: str, valueset_id: Optional[str],
                          codes: List[Tuple[str, str, str]]) -> int:
        """
        Bulk insert codes into precomputed_valueset_codes.
        Returns number of codes inserted.
        """
        if not codes or self.dry_run:
            return len(codes) if self.dry_run else 0

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

        # Insert in batches
        total_inserted = 0
        for i in range(0, len(data), BATCH_SIZE):
            batch = data[i:i + BATCH_SIZE]
            execute_values(cursor, query, batch)
            total_inserted += len(batch)

        self.pg_conn.commit()
        return total_inserted

    def process_expansion_file(self, exp_file: Path) -> int:
        """
        Process a single expansion file.
        Returns number of codes inserted.
        """
        # Extract ValueSet ID from filename: "xyz_expanded.json" → "xyz"
        valueset_id = exp_file.stem.replace("_expanded", "")

        try:
            # Load expansion JSON
            with open(exp_file, 'r') as f:
                expansion = json.load(f)

            # Get valueset_url from expansion or definition
            valueset_url = expansion.get("url")

            if not valueset_url:
                # Try to get from definition
                definition = self.get_definition(valueset_id)
                if definition:
                    valueset_url = definition.get("url")

            if not valueset_url:
                # Last resort: construct from ID
                valueset_url = f"http://ontoserver.csiro.au/fhir/ValueSet/{valueset_id}"

            # Idempotency guard - skip if already loaded
            if self.valueset_already_loaded(valueset_url):
                self.stats["expansions_skipped"] += 1
                return 0

            # Extract codes from expansion
            codes = self.extract_codes_from_expansion(expansion)

            if not codes:
                self.stats["expansions_empty"] += 1
                return 0

            # Get database ID if exists
            db_id = self.get_valueset_db_id(valueset_url)

            # Bulk insert codes
            inserted = self.bulk_insert_codes(valueset_url, db_id, codes)

            self.stats["expansions_loaded"] += 1
            self.stats["codes_inserted"] += inserted

            return inserted

        except json.JSONDecodeError as e:
            self.stats["errors"].append({
                "file": str(exp_file),
                "error": f"JSON decode error: {e}"
            })
            return 0
        except Exception as e:
            self.stats["errors"].append({
                "file": str(exp_file),
                "error": str(e)
            })
            return 0

    def load_all(self, limit: Optional[int] = None):
        """
        Load ALL expansion files using EXPANSION-FIRST pattern.

        This is the main entry point that:
        1. Lists ALL expansion files (22,003 expected)
        2. Processes each one (extract codes, bulk insert)
        3. No Neo4j, no ECL evaluation - pure JSON parsing
        """
        print(f"\n{'='*70}")
        print(f"  EXPANSION-FIRST LOADER - ALL Ontoserver Expansions")
        print(f"{'='*70}")
        print(f"  Source: {EXPANSIONS_DIR}")
        print(f"  SNOMED Release: {SNOMED_RELEASE}")
        print(f"  Limit: {limit or 'None (ALL files)'}")
        print(f"  Dry Run: {self.dry_run}")
        print(f"  Fresh Start: {self.fresh}")
        print(f"{'='*70}\n")

        # Verify directories exist
        if not EXPANSIONS_DIR.exists():
            print(f"ERROR: Expansions directory not found: {EXPANSIONS_DIR}")
            print("Run download_valuesets.py first!")
            sys.exit(1)

        self.connect()

        # Get ALL expansion files (this is the key - expansion-first!)
        exp_files = sorted(EXPANSIONS_DIR.glob("*_expanded.json"))
        self.stats["expansions_found"] = len(exp_files)

        if limit:
            exp_files = exp_files[:limit]

        print(f"Found {self.stats['expansions_found']:,} expansion files")
        if limit:
            print(f"Processing first {limit:,} files (--limit)\n")
        else:
            print(f"Processing ALL files\n")

        # Progress bar
        progress = tqdm(exp_files, desc="Loading expansions", unit="vs")

        for exp_file in progress:
            self.stats["expansions_processed"] += 1
            vs_id = exp_file.stem.replace("_expanded", "")
            progress.set_postfix_str(f"{vs_id[:25]}...")

            self.process_expansion_file(exp_file)

        self.close()
        self._print_summary()

    def _print_summary(self):
        """Print loading summary."""
        print(f"\n{'='*70}")
        print(f"  EXPANSION-FIRST LOADING COMPLETE!")
        print(f"{'='*70}")
        print(f"  SNOMED Release:        {SNOMED_RELEASE}")
        print(f"  Expansions Found:      {self.stats['expansions_found']:,}")
        print(f"  Expansions Processed:  {self.stats['expansions_processed']:,}")
        print(f"  Expansions Loaded:     {self.stats['expansions_loaded']:,}")
        print(f"  Expansions Skipped:    {self.stats['expansions_skipped']:,} (already loaded)")
        print(f"  Expansions Empty:      {self.stats['expansions_empty']:,}")
        print(f"  Total Codes Inserted:  {self.stats['codes_inserted']:,}")
        print(f"  Errors:                {len(self.stats['errors'])}")
        print(f"{'='*70}\n")

        if self.stats['errors']:
            print(f"First {min(10, len(self.stats['errors']))} errors:")
            for err in self.stats['errors'][:10]:
                print(f"  - {err['file']}: {err['error']}")
            print()

        print("VERIFICATION QUERIES:")
        print("  SELECT COUNT(*) FROM precomputed_valueset_codes;")
        print("  SELECT COUNT(DISTINCT valueset_url) FROM precomputed_valueset_codes;")
        print()
        print("NEXT STEPS:")
        print("  1. Run materialize_expansions.py for hierarchical ValueSets (gaps)")
        print("  2. Test $expand: curl http://localhost:8087/fhir/ValueSet/<name>/$expand")
        print("  3. Verify <50ms response time (pure PostgreSQL read)")


# ============================================================================
# Main Entry Point
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Load ALL Ontoserver ValueSet expansions (EXPANSION-FIRST pattern)"
    )
    parser.add_argument(
        "--limit", "-l",
        type=int,
        help="Limit number of expansion files to process (for testing)"
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Preview without making database changes"
    )
    parser.add_argument(
        "--fresh",
        action="store_true",
        help="Truncate table and reload all expansions"
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

    # Run loader
    loader = ExpansionFirstLoader(dry_run=args.dry_run, fresh=args.fresh)
    loader.load_all(limit=args.limit)


if __name__ == "__main__":
    main()
