#!/usr/bin/env python3
"""
Load RXNREL.RRF into KB-7 concept_relationships table
Optimized for fast bulk loading with PostgreSQL COPY
"""

import os
import sys
import csv
import tempfile
import subprocess
from datetime import datetime

# Configuration
RXNREL_PATH = "/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/data/rxnorm/extracted/rrf/RXNREL.RRF"
DB_CONTAINER = "kb7-postgres"
DB_NAME = "kb_terminology"
DB_USER = "postgres"

def parse_rxnrel_line(line):
    """
    Parse RXNREL.RRF format:
    RXCUI1|RXAUI1|STYPE1|REL|RXCUI2|RXAUI2|STYPE2|RELA|RUI|SRUI|SAB|SL|DIR|RG|SUPPRESS|CVF|
    """
    fields = line.strip().split('|')
    if len(fields) < 15:
        return None

    rxcui1 = fields[0]      # Source code
    rel = fields[3]         # Relationship type (RN, RB, RO, SY, etc.)
    rxcui2 = fields[4]      # Target code
    rela = fields[7]        # Additional relationship attribute
    sab = fields[10]        # Source abbreviation
    suppress = fields[14]   # Suppress flag

    # Only keep RxNorm relationships that are not suppressed
    if sab != "RXNORM" or suppress == "Y":
        return None

    # Skip self-relationships
    if rxcui1 == rxcui2 or not rxcui1 or not rxcui2:
        return None

    return {
        'source_code': rxcui1,
        'target_code': rxcui2,
        'relationship_type': rel,
        'relationship_attr': rela if rela else None
    }

def main():
    print(f"[{datetime.now()}] Starting RXNREL.RRF loading...")

    if not os.path.exists(RXNREL_PATH):
        print(f"ERROR: RXNREL.RRF not found at {RXNREL_PATH}")
        sys.exit(1)

    # Create temp CSV for COPY
    temp_csv = tempfile.NamedTemporaryFile(mode='w', suffix='.csv', delete=False)
    writer = csv.writer(temp_csv)

    print(f"[{datetime.now()}] Parsing RXNREL.RRF...")

    count = 0
    skipped = 0

    with open(RXNREL_PATH, 'r', encoding='utf-8', errors='ignore') as f:
        for line in f:
            parsed = parse_rxnrel_line(line)
            if parsed:
                writer.writerow([
                    parsed['source_code'],
                    parsed['target_code'],
                    parsed['relationship_type'],
                    parsed['relationship_attr'] or ''
                ])
                count += 1
            else:
                skipped += 1

            if count % 500000 == 0:
                print(f"  Processed {count:,} relationships...")

    temp_csv.close()
    print(f"[{datetime.now()}] Parsed {count:,} relationships (skipped {skipped:,})")

    # Copy CSV into container
    print(f"[{datetime.now()}] Copying data to container...")
    subprocess.run([
        'docker', 'cp', temp_csv.name, f'{DB_CONTAINER}:/tmp/rxnrel.csv'
    ], check=True)

    # Load using PostgreSQL COPY
    print(f"[{datetime.now()}] Loading into database with COPY...")

    copy_sql = """
    COPY concept_relationships(source_code, target_code, relationship_type, relationship_attr)
    FROM '/tmp/rxnrel.csv'
    WITH (FORMAT csv);
    """

    result = subprocess.run([
        'docker', 'exec', DB_CONTAINER, 'psql', '-U', DB_USER, '-d', DB_NAME, '-c', copy_sql
    ], capture_output=True, text=True)

    if result.returncode != 0:
        print(f"ERROR: {result.stderr}")
        sys.exit(1)

    print(result.stdout)

    # Create indexes
    print(f"[{datetime.now()}] Creating indexes...")

    index_sql = """
    CREATE INDEX IF NOT EXISTS idx_rel_source ON concept_relationships(source_code);
    CREATE INDEX IF NOT EXISTS idx_rel_target ON concept_relationships(target_code);
    CREATE INDEX IF NOT EXISTS idx_rel_type ON concept_relationships(relationship_type);
    CREATE INDEX IF NOT EXISTS idx_rel_source_type ON concept_relationships(source_code, relationship_type);
    """

    subprocess.run([
        'docker', 'exec', DB_CONTAINER, 'psql', '-U', DB_USER, '-d', DB_NAME, '-c', index_sql
    ], check=True)

    # Verify
    print(f"[{datetime.now()}] Verifying load...")

    result = subprocess.run([
        'docker', 'exec', DB_CONTAINER, 'psql', '-U', DB_USER, '-d', DB_NAME, '-t', '-c',
        "SELECT relationship_type, COUNT(*) FROM concept_relationships GROUP BY relationship_type ORDER BY count DESC LIMIT 10;"
    ], capture_output=True, text=True)

    print("Relationship types loaded:")
    print(result.stdout)

    # Cleanup
    os.unlink(temp_csv.name)
    subprocess.run(['docker', 'exec', DB_CONTAINER, 'rm', '/tmp/rxnrel.csv'])

    print(f"[{datetime.now()}] DONE! Loaded {count:,} RxNorm relationships")

if __name__ == "__main__":
    main()
