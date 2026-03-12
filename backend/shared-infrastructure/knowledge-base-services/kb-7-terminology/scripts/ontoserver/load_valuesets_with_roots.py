#!/usr/bin/env python3
"""
Load Ontoserver ValueSets into PostgreSQL with ROOT CODE extraction.

This script:
1. Parses 23,706 Ontoserver ValueSet JSON files
2. Extracts root SNOMED codes for intensional definitions
3. Loads ValueSet metadata into value_sets table with root_code columns

Root Code Patterns Supported:
- Pattern 1: RefSet - "^ 300161000210100 |display|"
- Pattern 2: Concept In - {"property": "concept", "op": "in", "value": "32570581000036105"}
- Pattern 3: ECL - "413836008 |display|"

This script populates the `value_sets` table with root codes.
The `materialize_expansions.py` script then uses these root codes to
run Neo4j traversal and populate `precomputed_valueset_codes`.

Usage:
    python load_valuesets_with_roots.py                    # Load all
    python load_valuesets_with_roots.py --filter sepsis    # Load matching
    python load_valuesets_with_roots.py --dry-run          # Preview only
"""

import os
import sys
import json
import re
import argparse
import psycopg2
from pathlib import Path
from datetime import datetime
from typing import Optional, Tuple, Dict, List
import hashlib

# ============================================================================
# Configuration
# ============================================================================

DATA_DIR = Path(__file__).parent.parent.parent / "data" / "ontoserver-valuesets"
DEFINITIONS_DIR = DATA_DIR / "definitions"

# Database connection
DB_CONFIG = {
    "host": os.getenv("POSTGRES_HOST", "localhost"),
    "port": int(os.getenv("POSTGRES_PORT", "5432")),  # Default port for kb_terminology
    "database": os.getenv("POSTGRES_DB", "kb_terminology"),
    "user": os.getenv("POSTGRES_USER", "postgres"),
    "password": os.getenv("POSTGRES_PASSWORD", "password")
}

SNOMED_SYSTEM = "http://snomed.info/sct"

# ============================================================================
# Root Code Extraction
# ============================================================================

def extract_root_code(valueset: Dict) -> Tuple[Optional[str], Optional[str], str]:
    """
    Extract root code from Ontoserver ValueSet compose.include.filter.

    Returns: (root_code, root_system, definition_type)

    Definition Types:
    - 'refset': Reference set (ECL "^" prefix)
    - 'concept_in': Direct concept "in" filter
    - 'ecl': ECL expression with codes
    - 'explicit': Enumerated concepts (no hierarchy needed)
    - 'unknown': Could not determine
    """
    compose = valueset.get("compose", {})
    includes = compose.get("include", [])

    for include in includes:
        system = include.get("system", "")
        filters = include.get("filter", [])

        for f in filters:
            prop = f.get("property", "")
            op = f.get("op", "")
            value = f.get("value", "")

            if not value:
                continue

            # Pattern 1: Reference Set (ECL "^" prefix)
            # Example: "^ 300161000210100 |New Zealand WHO classification...|"
            if prop == "constraint" and value.strip().startswith("^"):
                match = re.match(r'\^\s*(\d+)', value)
                if match:
                    return (match.group(1), system or SNOMED_SYSTEM, "refset")

            # Pattern 2: Direct concept "in" reference
            # Example: {"property": "concept", "op": "in", "value": "32570581000036105"}
            if prop == "concept" and op == "in":
                # Value is direct SNOMED code
                code = value.strip()
                if code.isdigit():
                    return (code, system or SNOMED_SYSTEM, "concept_in")

            # Pattern 3: ECL expression with direct codes
            # Example: "413836008 |Chronic eosinophilic leukaemia| OR 74964007 |Other|"
            if prop == "constraint":
                # Extract first numeric code from ECL
                match = re.match(r'(\d{6,})', value)
                if match:
                    return (match.group(1), system or SNOMED_SYSTEM, "ecl")

        # Check for explicit concept enumeration
        concepts = include.get("concept", [])
        if concepts:
            # Has explicit concepts - no hierarchy traversal needed
            return (None, system or SNOMED_SYSTEM, "explicit")

    return (None, None, "unknown")


def extract_oid(valueset: Dict) -> Optional[str]:
    """Extract OID from ValueSet identifier array."""
    identifiers = valueset.get("identifier", [])
    for ident in identifiers:
        system = ident.get("system", "")
        value = ident.get("value", "")
        if "oid" in system.lower() and value:
            return value.replace("urn:oid:", "")
    return None


# ============================================================================
# Database Operations
# ============================================================================

class ValueSetLoader:
    """Load ValueSets with root code extraction into PostgreSQL."""

    def __init__(self, dry_run: bool = False):
        self.dry_run = dry_run
        self.conn = None
        self.stats = {
            "total_processed": 0,
            "imported": 0,
            "updated": 0,
            "skipped": 0,
            "with_root_code": 0,
            "explicit": 0,
            "unknown": 0,
            "errors": []
        }

    def connect(self):
        """Connect to PostgreSQL."""
        if self.dry_run:
            print("DRY RUN MODE - No database changes will be made")
            return

        self.conn = psycopg2.connect(**DB_CONFIG)
        self.conn.autocommit = False
        print(f"Connected to PostgreSQL: {DB_CONFIG['host']}:{DB_CONFIG['port']}/{DB_CONFIG['database']}")

    def close(self):
        """Close database connection."""
        if self.conn:
            self.conn.close()

    def load_valueset(self, definition: Dict) -> bool:
        """Load a single ValueSet with root code extraction."""
        vs_id = definition.get("id", "")
        vs_url = definition.get("url", "")
        vs_version = definition.get("version", "1.0")
        vs_name = definition.get("name", vs_id)
        vs_title = definition.get("title", vs_name)
        vs_status = definition.get("status", "active")
        vs_publisher = definition.get("publisher", "CSIRO Ontoserver")
        vs_description = definition.get("description", "")[:2000]  # Truncate

        # Extract root code and definition type
        root_code, root_system, definition_type = extract_root_code(definition)
        oid = extract_oid(definition)

        # Track stats
        if definition_type in ("refset", "concept_in", "ecl"):
            self.stats["with_root_code"] += 1
        elif definition_type == "explicit":
            self.stats["explicit"] += 1
        else:
            self.stats["unknown"] += 1

        if self.dry_run:
            status = "intensional" if root_code else definition_type
            print(f"  [DRY RUN] {vs_name[:50]}")
            print(f"            URL: {vs_url}")
            print(f"            Type: {status}, Root: {root_code or 'N/A'}")
            self.stats["imported"] += 1
            return True

        try:
            cursor = self.conn.cursor()

            # Generate deterministic UUID from URL+version
            hash_input = f"{vs_url}|{vs_version}"
            vs_uuid = hashlib.md5(hash_input.encode()).hexdigest()
            vs_uuid = f"{vs_uuid[:8]}-{vs_uuid[8:12]}-{vs_uuid[12:16]}-{vs_uuid[16:20]}-{vs_uuid[20:32]}"

            # Upsert ValueSet with root code columns
            cursor.execute("""
                INSERT INTO value_sets (
                    id, url, version, name, title, description, status, publisher,
                    compose, root_code, root_system, definition_type, oid,
                    created_at, updated_at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (url, version) DO UPDATE SET
                    name = EXCLUDED.name,
                    title = EXCLUDED.title,
                    description = EXCLUDED.description,
                    status = EXCLUDED.status,
                    root_code = EXCLUDED.root_code,
                    root_system = EXCLUDED.root_system,
                    definition_type = EXCLUDED.definition_type,
                    oid = EXCLUDED.oid,
                    updated_at = NOW()
                RETURNING id, (xmax = 0) AS is_insert
            """, (
                vs_uuid,
                vs_url,
                vs_version,
                vs_name,
                vs_title,
                vs_description,
                vs_status,
                vs_publisher,
                json.dumps(definition.get("compose", {})),
                root_code,
                root_system,
                definition_type,
                oid,
                datetime.now(),
                datetime.now()
            ))

            result = cursor.fetchone()
            is_insert = result[1] if result else True

            self.conn.commit()

            if is_insert:
                self.stats["imported"] += 1
            else:
                self.stats["updated"] += 1

            return True

        except Exception as e:
            self.conn.rollback()
            self.stats["errors"].append({"id": vs_id, "url": vs_url, "error": str(e)})
            return False

    def load_all(self, name_filter: Optional[str] = None):
        """Load all ValueSets from downloaded JSON files."""
        print(f"\n{'='*70}")
        print(f"  ONTOSERVER VALUESET LOADER WITH ROOT CODE EXTRACTION")
        print(f"{'='*70}")
        print(f"  Source: {DEFINITIONS_DIR}")
        print(f"  Filter: {name_filter or 'None'}")
        print(f"  Dry Run: {self.dry_run}")
        print(f"{'='*70}\n")

        # Get list of definition files
        def_files = sorted(DEFINITIONS_DIR.glob("*.json"))
        total_files = len(def_files)
        print(f"Found {total_files:,} ValueSet definition files\n")

        self.connect()

        for i, def_file in enumerate(def_files, 1):
            self.stats["total_processed"] += 1

            try:
                # Load definition
                with open(def_file, 'r') as f:
                    definition = json.load(f)

                # Filter by name if specified
                vs_name = definition.get("name", "")
                if name_filter and name_filter.lower() not in vs_name.lower():
                    self.stats["skipped"] += 1
                    continue

                # Load into database
                self.load_valueset(definition)

                # Progress
                if i % 500 == 0:
                    print(f"  Progress: {i:,}/{total_files:,} ({100*i/total_files:.1f}%)")
                    print(f"    - Imported: {self.stats['imported']:,}")
                    print(f"    - With root codes: {self.stats['with_root_code']:,}")

            except Exception as e:
                self.stats["errors"].append({"file": str(def_file), "error": str(e)})

        self.close()
        self._print_summary()

    def _print_summary(self):
        """Print import summary."""
        print(f"\n{'='*70}")
        print(f"  IMPORT COMPLETE!")
        print(f"{'='*70}")
        print(f"  Files Processed:      {self.stats['total_processed']:,}")
        print(f"  ValueSets Imported:   {self.stats['imported']:,}")
        print(f"  ValueSets Updated:    {self.stats['updated']:,}")
        print(f"  ValueSets Skipped:    {self.stats['skipped']:,}")
        print(f"{'='*70}")
        print(f"  Root Code Extraction:")
        print(f"    - With Root Codes:  {self.stats['with_root_code']:,} (need materialization)")
        print(f"    - Explicit Codes:   {self.stats['explicit']:,} (no Neo4j needed)")
        print(f"    - Unknown Type:     {self.stats['unknown']:,}")
        print(f"{'='*70}")
        print(f"  Errors: {len(self.stats['errors'])}")
        print(f"{'='*70}\n")

        if self.stats['errors']:
            print("First 10 errors:")
            for err in self.stats['errors'][:10]:
                print(f"  - {err}")

        print("\nNext steps:")
        print("  1. Run materialize_expansions.py to populate precomputed_valueset_codes")
        print("  2. This will run Neo4j traversal for intensional ValueSets")
        print("  3. Then $expand endpoint will use pure PostgreSQL reads")


# ============================================================================
# Main Entry Point
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Load Ontoserver ValueSets with root code extraction"
    )
    parser.add_argument(
        "--filter", "-f",
        help="Filter ValueSets by name (substring match)"
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Preview without making database changes"
    )
    parser.add_argument(
        "--db-host",
        default=os.getenv("POSTGRES_HOST", "localhost"),
        help="PostgreSQL host"
    )
    parser.add_argument(
        "--db-port",
        type=int,
        default=int(os.getenv("POSTGRES_PORT", "5432")),
        help="PostgreSQL port"
    )

    args = parser.parse_args()

    # Update config from args
    DB_CONFIG["host"] = args.db_host
    DB_CONFIG["port"] = args.db_port

    # Check if data exists
    if not DEFINITIONS_DIR.exists():
        print(f"ERROR: Definitions directory not found: {DEFINITIONS_DIR}")
        print("Run download_valuesets.py first!")
        sys.exit(1)

    loader = ValueSetLoader(dry_run=args.dry_run)
    loader.load_all(name_filter=args.filter)


if __name__ == "__main__":
    main()
