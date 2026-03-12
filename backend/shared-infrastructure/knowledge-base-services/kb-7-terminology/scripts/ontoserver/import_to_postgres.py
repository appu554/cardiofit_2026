#!/usr/bin/env python3
"""
Import downloaded Ontoserver ValueSets into PostgreSQL
Loads JSON files from data/ontoserver-valuesets/ into value_sets and value_set_concepts tables

Usage:
    python import_to_postgres.py                    # Import all
    python import_to_postgres.py --filter sepsis    # Import only matching name
    python import_to_postgres.py --dry-run          # Preview without importing
"""

import os
import sys
import json
import argparse
import psycopg2
from pathlib import Path
from datetime import datetime
from typing import Optional, Dict, List, Any
import hashlib

# ============================================================================
# Configuration
# ============================================================================

DATA_DIR = Path(__file__).parent.parent.parent / "data" / "ontoserver-valuesets"
DEFINITIONS_DIR = DATA_DIR / "definitions"
EXPANSIONS_DIR = DATA_DIR / "expansions"

# Database connection (match KB7 service config)
DB_CONFIG = {
    "host": os.getenv("POSTGRES_HOST", "localhost"),
    "port": int(os.getenv("POSTGRES_PORT", "5433")),
    "database": os.getenv("POSTGRES_DB", "kb_terminology"),
    "user": os.getenv("POSTGRES_USER", "postgres"),
    "password": os.getenv("POSTGRES_PASSWORD", "password")
}

# ============================================================================
# Database Operations
# ============================================================================

class PostgresImporter:
    """Import ValueSets into PostgreSQL"""

    def __init__(self, dry_run: bool = False):
        self.dry_run = dry_run
        self.conn = None
        self.stats = {
            "valuesets_imported": 0,
            "valuesets_skipped": 0,
            "concepts_imported": 0,
            "errors": []
        }

    def connect(self):
        """Connect to PostgreSQL"""
        if self.dry_run:
            print("DRY RUN MODE - No database changes will be made")
            return

        self.conn = psycopg2.connect(**DB_CONFIG)
        self.conn.autocommit = False
        print(f"Connected to PostgreSQL: {DB_CONFIG['host']}:{DB_CONFIG['port']}/{DB_CONFIG['database']}")

    def close(self):
        """Close database connection"""
        if self.conn:
            self.conn.close()

    def valueset_exists(self, url: str, version: str) -> bool:
        """Check if ValueSet already exists"""
        if self.dry_run:
            return False

        cursor = self.conn.cursor()
        cursor.execute(
            "SELECT 1 FROM value_sets WHERE url = %s AND version = %s",
            (url, version)
        )
        exists = cursor.fetchone() is not None
        cursor.close()
        return exists

    def import_valueset(self, definition: Dict, expansion: Optional[Dict] = None) -> bool:
        """Import a single ValueSet with its concepts"""
        vs_id = definition.get("id", "")
        vs_url = definition.get("url", "")
        vs_version = definition.get("version", "1.0")
        vs_name = definition.get("name", vs_id)
        vs_title = definition.get("title", vs_name)
        vs_status = definition.get("status", "active")
        vs_publisher = definition.get("publisher", "CSIRO Ontoserver")
        vs_description = definition.get("description", "")

        # Skip if already exists
        if self.valueset_exists(vs_url, vs_version):
            self.stats["valuesets_skipped"] += 1
            return False

        if self.dry_run:
            print(f"  [DRY RUN] Would import: {vs_name} ({vs_url})")
            self.stats["valuesets_imported"] += 1
            return True

        try:
            cursor = self.conn.cursor()

            # Generate UUID for the value set
            vs_uuid = hashlib.md5(f"{vs_url}|{vs_version}".encode()).hexdigest()
            vs_uuid = f"{vs_uuid[:8]}-{vs_uuid[8:12]}-{vs_uuid[12:16]}-{vs_uuid[16:20]}-{vs_uuid[20:32]}"

            # Insert value set
            cursor.execute("""
                INSERT INTO value_sets (
                    id, url, version, name, title, description, status, publisher,
                    compose, expansion, created_at, updated_at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (url, version) DO UPDATE SET
                    name = EXCLUDED.name,
                    title = EXCLUDED.title,
                    description = EXCLUDED.description,
                    status = EXCLUDED.status,
                    updated_at = NOW()
                RETURNING id
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
                json.dumps(expansion.get("expansion", {}) if expansion else {}),
                datetime.now(),
                datetime.now()
            ))

            result = cursor.fetchone()
            actual_uuid = result[0] if result else vs_uuid

            # Import concepts from expansion
            if expansion:
                exp_data = expansion.get("expansion", {})
                contains = exp_data.get("contains", [])

                for concept in contains:
                    self._import_concept(cursor, actual_uuid, concept)
                    self.stats["concepts_imported"] += 1

            self.conn.commit()
            self.stats["valuesets_imported"] += 1
            return True

        except Exception as e:
            self.conn.rollback()
            self.stats["errors"].append({"id": vs_id, "error": str(e)})
            return False

    def _import_concept(self, cursor, value_set_id: str, concept: Dict):
        """Import a single concept into value_set_concepts"""
        code = concept.get("code", "")
        system = concept.get("system", "")
        display = concept.get("display", "")
        version = concept.get("version", "")

        cursor.execute("""
            INSERT INTO value_set_concepts (
                value_set_id, system, code, display, version, inactive
            ) VALUES (
                (SELECT id FROM value_sets WHERE id::text = %s LIMIT 1),
                %s, %s, %s, %s, %s
            )
            ON CONFLICT DO NOTHING
        """, (
            value_set_id,
            system,
            code,
            display,
            version or None,
            concept.get("inactive", False)
        ))

    def import_all(self, name_filter: Optional[str] = None):
        """Import all downloaded ValueSets"""
        print(f"\n{'='*60}")
        print(f"  ONTOSERVER VALUESET IMPORTER")
        print(f"{'='*60}")
        print(f"  Source: {DATA_DIR}")
        print(f"  Filter: {name_filter or 'None'}")
        print(f"  Dry Run: {self.dry_run}")
        print(f"{'='*60}\n")

        # Get list of definition files
        def_files = list(DEFINITIONS_DIR.glob("*.json"))
        print(f"Found {len(def_files):,} ValueSet definition files")

        self.connect()

        for i, def_file in enumerate(def_files, 1):
            vs_id = def_file.stem

            # Load definition
            with open(def_file, 'r') as f:
                definition = json.load(f)

            # Filter by name if specified
            vs_name = definition.get("name", "")
            if name_filter and name_filter.lower() not in vs_name.lower():
                continue

            # Load expansion if available
            exp_file = EXPANSIONS_DIR / f"{vs_id}_expanded.json"
            expansion = None
            if exp_file.exists():
                with open(exp_file, 'r') as f:
                    expansion = json.load(f)

            # Import
            success = self.import_valueset(definition, expansion)
            status = "✓" if success else "○" if not self.valueset_exists(
                definition.get("url", ""), definition.get("version", "1.0")
            ) else "="

            if i % 100 == 0 or success:
                print(f"  [{i:,}/{len(def_files):,}] {status} {vs_name[:50]}")

        self.close()

        # Print summary
        print(f"\n{'='*60}")
        print(f"  IMPORT COMPLETE!")
        print(f"{'='*60}")
        print(f"  ValueSets Imported: {self.stats['valuesets_imported']:,}")
        print(f"  ValueSets Skipped (existing): {self.stats['valuesets_skipped']:,}")
        print(f"  Concepts Imported: {self.stats['concepts_imported']:,}")
        print(f"  Errors: {len(self.stats['errors'])}")
        print(f"{'='*60}\n")

        if self.stats['errors']:
            print("Errors:")
            for err in self.stats['errors'][:10]:
                print(f"  - {err['id']}: {err['error']}")

# ============================================================================
# Main Entry Point
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Import Ontoserver ValueSets into PostgreSQL"
    )
    parser.add_argument(
        "--filter", "-f",
        help="Filter ValueSets by name"
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Preview without making changes"
    )

    args = parser.parse_args()

    # Check if data exists
    if not DEFINITIONS_DIR.exists():
        print(f"ERROR: Definitions directory not found: {DEFINITIONS_DIR}")
        print("Run download_valuesets.py first!")
        sys.exit(1)

    importer = PostgresImporter(dry_run=args.dry_run)
    importer.import_all(name_filter=args.filter)

if __name__ == "__main__":
    main()
