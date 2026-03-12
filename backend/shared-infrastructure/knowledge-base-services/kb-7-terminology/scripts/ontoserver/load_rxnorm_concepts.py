#!/usr/bin/env python3
"""
load_rxnorm_concepts.py - Load ALL RxNorm concepts into PostgreSQL concepts table

This script reads RXNCONSO.RRF and loads all RxNorm concepts into the concepts table,
enabling general RxNorm code lookups via KB-7 API.

PURPOSE:
    Load complete RxNorm terminology for:
    - Code validation (is this a valid RxNorm code?)
    - Display name lookup (what is the name for code X?)
    - General terminology services

USAGE:
    python load_rxnorm_concepts.py                    # Full load
    python load_rxnorm_concepts.py --dry-run         # Preview changes
    python load_rxnorm_concepts.py --limit 1000      # Test with limited set

ENVIRONMENT VARIABLES:
    POSTGRES_HOST     PostgreSQL host (default: localhost)
    POSTGRES_PORT     PostgreSQL port (default: 5437)
    POSTGRES_DB       PostgreSQL database (default: kb_terminology)
    POSTGRES_USER     PostgreSQL user (default: postgres)
    POSTGRES_PASSWORD PostgreSQL password (default: password)
"""

import os
import sys
import argparse
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Tuple

try:
    import psycopg2
    from psycopg2.extras import execute_batch
except ImportError:
    print("ERROR: psycopg2 not installed. Run: pip install psycopg2-binary")
    sys.exit(1)

# ============================================================================
# Configuration
# ============================================================================

SCRIPT_DIR = Path(__file__).parent
DATA_DIR = SCRIPT_DIR.parent.parent / "data" / "rxnorm"
RRF_DIR = DATA_DIR / "extracted" / "rrf"

RXNORM_SYSTEM = "http://www.nlm.nih.gov/research/umls/rxnorm"
RXNORM_SYSTEM_SHORT = "RxNorm"  # For concepts table - MUST match partition key (case-sensitive!)
RXNORM_VERSION = os.getenv("RXNORM_VERSION", "2024")

POSTGRES_CONFIG = {
    "host": os.getenv("POSTGRES_HOST", "localhost"),
    "port": int(os.getenv("POSTGRES_PORT", "5437")),
    "database": os.getenv("POSTGRES_DB", "kb_terminology"),
    "user": os.getenv("POSTGRES_USER", "postgres"),
    "password": os.getenv("POSTGRES_PASSWORD", "password")
}

# RxNorm Term Types to include
# IN = Ingredient, SCD = Semantic Clinical Drug, SBD = Semantic Branded Drug
# MIN = Multiple Ingredient, BN = Brand Name, PIN = Precise Ingredient
# GPCK = Generic Pack, BPCK = Brand Name Pack
INCLUDED_TTY = {'IN', 'PIN', 'MIN', 'SCD', 'SBD', 'GPCK', 'BPCK', 'BN', 'SCDC', 'SBDC', 'SCDF', 'SBDF', 'SCDG', 'SBDG', 'DF', 'DFG'}

# ============================================================================
# Logging
# ============================================================================

def log_header(msg):
    print(f"\n{'=' * 70}")
    print(f"  {msg}")
    print(f"{'=' * 70}")

def log_info(msg):
    print(f"  {msg}")

def log_success(msg):
    print(f"  ✅ {msg}")

def log_error(msg):
    print(f"  ❌ {msg}")

def log_warning(msg):
    print(f"  ⚠️  {msg}")

# ============================================================================
# RxNorm Concept Loader
# ============================================================================

class RxNormConceptLoader:
    """Load all RxNorm concepts into PostgreSQL concepts table."""

    def __init__(self, dry_run: bool = False, limit: int = None):
        self.dry_run = dry_run
        self.limit = limit
        self.conn = None
        self.stats = {
            "concepts_read": 0,
            "concepts_inserted": 0,
            "concepts_skipped": 0,
            "errors": []
        }
        self.batch_size = 1000

    def connect(self):
        """Connect to PostgreSQL."""
        if self.dry_run:
            log_info("[DRY RUN] Would connect to PostgreSQL")
            return

        self.conn = psycopg2.connect(**POSTGRES_CONFIG)
        self.conn.autocommit = False
        log_success(f"Connected to PostgreSQL: {POSTGRES_CONFIG['host']}:{POSTGRES_CONFIG['port']}")

    def close(self):
        """Close database connection."""
        if self.conn:
            self.conn.close()

    def load_concepts_from_rxnconso(self) -> List[Tuple]:
        """
        Load RxNorm concepts from RXNCONSO.RRF.

        Returns list of tuples: (code, preferred_term, tty, suppress)
        """
        conso_file = RRF_DIR / "RXNCONSO.RRF"
        if not conso_file.exists():
            log_error(f"RXNCONSO.RRF not found at {conso_file}")
            return []

        log_info(f"Loading RxNorm concepts from {conso_file}")

        concepts = {}  # rxcui -> {name, tty}
        count = 0

        with open(conso_file, 'r', encoding='utf-8') as f:
            for line in f:
                fields = line.strip().split('|')
                if len(fields) < 18:
                    continue

                rxcui = fields[0]
                lat = fields[1]    # Language
                sab = fields[11]   # Source (RXNORM)
                tty = fields[12]   # Term type
                name = fields[14]  # String/name
                suppress = fields[16]

                # Only English, non-suppressed, from RXNORM source
                if lat == 'ENG' and suppress == 'N' and sab == 'RXNORM':
                    if tty in INCLUDED_TTY:
                        # Keep first occurrence (usually preferred term)
                        if rxcui not in concepts:
                            concepts[rxcui] = {'name': name, 'tty': tty}
                        count += 1

                        if self.limit and len(concepts) >= self.limit:
                            break

        log_success(f"Loaded {len(concepts):,} unique RxNorm concepts")
        self.stats["concepts_read"] = len(concepts)
        return concepts

    def insert_concepts(self, concepts: Dict):
        """Insert concepts into PostgreSQL concepts table."""
        if self.dry_run:
            log_info(f"[DRY RUN] Would insert {len(concepts):,} concepts")
            return

        cursor = self.conn.cursor()

        # First, check if concepts table has the right structure
        cursor.execute("""
            SELECT column_name FROM information_schema.columns
            WHERE table_name = 'concepts' AND table_schema = 'public'
        """)
        columns = [row[0] for row in cursor.fetchall()]
        log_info(f"Concepts table columns: {columns}")

        # Delete existing RxNorm concepts (to allow re-runs)
        cursor.execute("""
            DELETE FROM concepts WHERE system = %s
        """, (RXNORM_SYSTEM_SHORT,))
        deleted = cursor.rowcount
        if deleted > 0:
            log_info(f"Deleted {deleted:,} existing RxNorm concepts")

        # Prepare batch insert
        batch = []
        for rxcui, data in concepts.items():
            batch.append((
                RXNORM_SYSTEM_SHORT,  # system (varchar(20))
                rxcui,                 # code
                RXNORM_VERSION,        # version
                data['name'][:200],    # preferred_term (truncate to 200 for phonetic triggers)
                None,                  # fully_specified_name
                [],                    # synonyms
                [],                    # parent_codes
                True,                  # is_leaf (assume leaf for RxNorm)
                True                   # active
            ))

            # Insert in batches
            if len(batch) >= self.batch_size:
                self._insert_batch(cursor, batch)
                batch = []

        # Insert remaining batch
        if batch:
            self._insert_batch(cursor, batch)

        self.conn.commit()
        log_success(f"Inserted {self.stats['concepts_inserted']:,} RxNorm concepts")

    def _insert_batch(self, cursor, batch: List[Tuple]):
        """Insert a batch of concepts."""
        insert_query = """
            INSERT INTO concepts (
                system, code, version, preferred_term, fully_specified_name,
                synonyms, parent_codes, is_leaf, active
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (system, code, version) DO UPDATE SET
                preferred_term = EXCLUDED.preferred_term,
                updated_at = NOW()
        """

        try:
            execute_batch(cursor, insert_query, batch, page_size=100)
            self.stats['concepts_inserted'] += len(batch)
            if self.stats['concepts_inserted'] % 10000 == 0:
                log_info(f"  Inserted {self.stats['concepts_inserted']:,} concepts...")
        except Exception as e:
            self.conn.rollback()
            self.stats['errors'].append(str(e))
            log_error(f"Batch insert error: {e}")

    def load_all(self):
        """Main entry point - load all RxNorm concepts."""
        log_header("RXNORM CONCEPT LOADER")
        log_info(f"Time: {datetime.now().isoformat()}")
        log_info(f"Dry Run: {self.dry_run}")
        log_info(f"Limit: {self.limit or 'None'}")
        log_info(f"RxNorm Version: {RXNORM_VERSION}")
        log_info(f"Data Directory: {RRF_DIR}")

        # Load concepts from RXNCONSO.RRF
        log_header("LOADING RXNORM DATA")
        concepts = self.load_concepts_from_rxnconso()
        if not concepts:
            return False

        # Connect and insert
        self.connect()

        log_header("INSERTING INTO POSTGRESQL")
        self.insert_concepts(concepts)

        self.close()
        self._print_summary()
        return True

    def _print_summary(self):
        """Print loading summary."""
        log_header("SUMMARY")
        log_info(f"Concepts read: {self.stats['concepts_read']:,}")
        log_info(f"Concepts inserted: {self.stats['concepts_inserted']:,}")
        log_info(f"Concepts skipped: {self.stats['concepts_skipped']:,}")
        log_info(f"Errors: {len(self.stats['errors'])}")

        if self.stats['errors']:
            log_error("Errors encountered:")
            for err in self.stats['errors'][:5]:
                log_error(f"  {err}")

        if not self.dry_run:
            print("\n" + "=" * 70)
            print("  NEXT STEPS:")
            print("=" * 70)
            print("  1. Verify concepts in PostgreSQL:")
            print("     SELECT system, COUNT(*) FROM concepts GROUP BY system;")
            print("  2. Test RxNorm lookup:")
            print("     curl http://localhost:8092/v1/concepts/RXNORM/314076")
            print("")

# ============================================================================
# Main
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Load ALL RxNorm concepts into PostgreSQL concepts table"
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Preview without making database changes"
    )
    parser.add_argument(
        "--limit", "-l",
        type=int,
        help="Limit number of concepts to load (for testing)"
    )

    args = parser.parse_args()

    loader = RxNormConceptLoader(dry_run=args.dry_run, limit=args.limit)
    success = loader.load_all()

    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
