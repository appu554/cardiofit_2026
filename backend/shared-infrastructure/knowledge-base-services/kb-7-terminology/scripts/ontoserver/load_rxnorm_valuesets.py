#!/usr/bin/env python3
"""
load_rxnorm_valuesets.py - Load RxNorm drug class ValueSets into PostgreSQL

This script reads the drug_class_mappings.json file and expands ingredient codes
to all formulations (SCD, SBD) using RXNREL.RRF relationships, then inserts
into precomputed_valueset_codes table.

PURPOSE (CTO/CMO DIRECTIVE):
    "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."

This loads RxNorm medication codes so that ValueSet membership checking works
for patient medications that use RxNorm codes (e.g., "314076" for Lisinopril 10 MG).

USAGE:
    python load_rxnorm_valuesets.py                    # Full load
    python load_rxnorm_valuesets.py --dry-run         # Preview changes
    python load_rxnorm_valuesets.py --class ACEInhibitors  # Load specific class

ENVIRONMENT VARIABLES:
    POSTGRES_HOST     PostgreSQL host (default: localhost)
    POSTGRES_PORT     PostgreSQL port (default: 5437)
    POSTGRES_DB       PostgreSQL database (default: kb_terminology)
    POSTGRES_USER     PostgreSQL user (default: postgres)
    POSTGRES_PASSWORD PostgreSQL password (default: password)
"""

import os
import sys
import json
import argparse
from pathlib import Path
from datetime import datetime
from typing import Dict, Set, List, Tuple

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
MAPPING_FILE = DATA_DIR / "drug_class_mappings.json"
RRF_DIR = DATA_DIR / "extracted" / "rrf"

RXNORM_SYSTEM = "http://www.nlm.nih.gov/research/umls/rxnorm"

POSTGRES_CONFIG = {
    "host": os.getenv("POSTGRES_HOST", "localhost"),
    "port": int(os.getenv("POSTGRES_PORT", "5437")),
    "database": os.getenv("POSTGRES_DB", "kb_terminology"),
    "user": os.getenv("POSTGRES_USER", "postgres"),
    "password": os.getenv("POSTGRES_PASSWORD", "password")
}

# RxNorm Term Types we want to include
# IN = Ingredient, SCD = Semantic Clinical Drug, SBD = Semantic Branded Drug
# MIN = Multiple Ingredient, BN = Brand Name, PIN = Precise Ingredient
INCLUDED_TTY = {'IN', 'PIN', 'MIN', 'SCD', 'SBD', 'GPCK', 'BPCK', 'BN'}

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
# RxNorm Data Loader
# ============================================================================

class RxNormValueSetLoader:
    """Load RxNorm drug classes into precomputed_valueset_codes."""

    def __init__(self, dry_run: bool = False):
        self.dry_run = dry_run
        self.conn = None
        self.concepts = {}  # rxcui -> {name, tty}
        self.stats = {
            "classes_processed": 0,
            "codes_inserted": 0,
            "codes_skipped": 0,
            "errors": []
        }

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

    def load_rxnorm_concepts(self):
        """Load RxNorm concepts from RXNCONSO.RRF."""
        conso_file = RRF_DIR / "RXNCONSO.RRF"
        if not conso_file.exists():
            log_error(f"RXNCONSO.RRF not found at {conso_file}")
            return False

        log_info(f"Loading RxNorm concepts from {conso_file}")

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
                        if rxcui not in self.concepts:
                            self.concepts[rxcui] = {'name': name, 'tty': tty}
                        count += 1

        log_success(f"Loaded {len(self.concepts):,} RxNorm concepts")
        return True

    def expand_ingredient_to_codes(self, ingredient_rxcui: str, ingredient_name: str) -> Set[Tuple[str, str]]:
        """
        Expand an ingredient to all related codes using NAME-BASED MATCHING.

        This is more reliable than RXNREL parsing because:
        1. RXNREL has complex format with multiple sources
        2. Name matching catches all formulations (SCD, SBD) containing the ingredient

        Returns set of (rxcui, display_name) tuples.
        """
        codes = set()
        ingredient_lower = ingredient_name.lower()

        # Add the ingredient itself
        if ingredient_rxcui in self.concepts:
            codes.add((ingredient_rxcui, self.concepts[ingredient_rxcui]['name']))

        # Find ALL concepts whose name contains the ingredient name
        # This catches: "lisinopril 10 MG Oral Tablet", "lisinopril 20 MG [Zestril]", etc.
        for rxcui, concept in self.concepts.items():
            concept_name_lower = concept['name'].lower()
            # Match if ingredient name appears at start of word boundary in concept name
            # e.g., "lisinopril" matches "lisinopril 10 MG" but not "hydrochlorothiazide lisinopril"
            if concept_name_lower.startswith(ingredient_lower) or f" {ingredient_lower}" in concept_name_lower:
                codes.add((rxcui, concept['name']))

        return codes

    def load_drug_class(self, class_name: str, class_data: dict) -> int:
        """Load a single drug class into precomputed_valueset_codes."""
        valueset_url = class_data['valueset_url']
        ingredients = class_data['ingredients']

        log_info(f"\nProcessing: {class_name}")
        log_info(f"  ValueSet URL: {valueset_url}")
        log_info(f"  Ingredients: {len(ingredients)}")

        # Expand all ingredients to their formulations using name-based matching
        all_codes = set()
        for ing in ingredients:
            rxcui = ing['rxcui']
            name = ing['name']
            expanded = self.expand_ingredient_to_codes(rxcui, name)
            all_codes.update(expanded)
            log_info(f"    {name} ({rxcui}): {len(expanded)} codes")

        log_info(f"  Total expanded codes: {len(all_codes)}")

        if self.dry_run:
            log_info(f"  [DRY RUN] Would insert {len(all_codes)} codes")
            return len(all_codes)

        # Insert into precomputed_valueset_codes
        cursor = self.conn.cursor()

        # First, delete existing RxNorm codes for this valueset (to allow re-runs)
        cursor.execute("""
            DELETE FROM precomputed_valueset_codes
            WHERE valueset_url = %s AND code_system = %s
        """, (valueset_url, RXNORM_SYSTEM))
        deleted = cursor.rowcount
        if deleted > 0:
            log_info(f"  Deleted {deleted} existing RxNorm codes")

        # Prepare batch insert
        # Note: snomed_version is NOT NULL, we use 'RXNORM-2024' for RxNorm codes
        rxnorm_version = "RXNORM-2024"
        insert_data = [
            (valueset_url, rxnorm_version, RXNORM_SYSTEM, code, display)
            for code, display in all_codes
        ]

        if insert_data:
            execute_batch(cursor, """
                INSERT INTO precomputed_valueset_codes
                    (valueset_url, snomed_version, code_system, code, display, materialized_at)
                VALUES (%s, %s, %s, %s, %s, NOW())
                ON CONFLICT (valueset_url, snomed_version, code_system, code) DO UPDATE SET
                    display = EXCLUDED.display,
                    materialized_at = NOW()
            """, insert_data, page_size=500)

        self.conn.commit()
        log_success(f"  Inserted {len(insert_data)} codes")

        return len(insert_data)

    def load_all(self, specific_class: str = None):
        """Load all drug classes from the mapping file."""
        log_header("RXNORM VALUESET LOADER")
        log_info(f"Time: {datetime.now().isoformat()}")
        log_info(f"Dry Run: {self.dry_run}")
        log_info(f"Mapping File: {MAPPING_FILE}")

        # Load mapping file
        if not MAPPING_FILE.exists():
            log_error(f"Mapping file not found: {MAPPING_FILE}")
            return False

        with open(MAPPING_FILE, 'r') as f:
            mappings = json.load(f)

        drug_classes = mappings.get('drug_classes', {})
        log_info(f"Found {len(drug_classes)} drug classes in mapping file")

        # Filter to specific class if requested
        if specific_class:
            if specific_class not in drug_classes:
                log_error(f"Drug class '{specific_class}' not found in mappings")
                return False
            drug_classes = {specific_class: drug_classes[specific_class]}

        # Load RxNorm data
        log_header("LOADING RXNORM DATA")
        if not self.load_rxnorm_concepts():
            return False

        # Connect to database
        self.connect()

        # Process each drug class
        log_header("LOADING DRUG CLASSES")
        total_codes = 0
        for class_name, class_data in drug_classes.items():
            try:
                codes = self.load_drug_class(class_name, class_data)
                total_codes += codes
                self.stats['classes_processed'] += 1
                self.stats['codes_inserted'] += codes
            except Exception as e:
                log_error(f"Error loading {class_name}: {e}")
                self.stats['errors'].append(f"{class_name}: {e}")

        self.close()
        self._print_summary(total_codes)
        return True

    def _print_summary(self, total_codes: int):
        """Print loading summary."""
        log_header("SUMMARY")
        log_info(f"Drug classes processed: {self.stats['classes_processed']}")
        log_info(f"Total codes inserted: {self.stats['codes_inserted']:,}")
        log_info(f"Errors: {len(self.stats['errors'])}")

        if self.stats['errors']:
            log_error("Errors encountered:")
            for err in self.stats['errors'][:5]:
                log_error(f"  {err}")

        if not self.dry_run:
            print("\n" + "=" * 70)
            print("  NEXT STEPS:")
            print("=" * 70)
            print("  1. Run kb7-materialize-all.py to refresh materialization")
            print("  2. Restart KB-7 service to pick up new codes")
            print("  3. Test: curl 'http://localhost:8092/fhir/ValueSet/meds-ace-inhibitors/$expand'")
            print("")

# ============================================================================
# Main
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Load RxNorm drug class ValueSets into PostgreSQL"
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Preview without making database changes"
    )
    parser.add_argument(
        "--class", "-c",
        dest="drug_class",
        help="Load only a specific drug class (e.g., ACEInhibitors)"
    )

    args = parser.parse_args()

    loader = RxNormValueSetLoader(dry_run=args.dry_run)
    success = loader.load_all(specific_class=args.drug_class)

    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
