#!/usr/bin/env python3
"""
Direct ICD-10-CM Loader for KB-7 Terminology Service
Loads ICD-10-CM codes from CMS text files directly into PostgreSQL.

Usage:
    python load_icd10_direct.py --data ./data/icd10 --batch 5000
"""

import os
import sys
import argparse
from datetime import datetime
from typing import Dict, List, Optional
import psycopg2
from psycopg2.extras import execute_values

# Database connection settings (same as KB-7 Terminology Service)
DB_CONFIG = {
    'host': os.environ.get('DB_HOST', 'localhost'),
    'port': int(os.environ.get('DB_PORT', '5437')),
    'dbname': os.environ.get('DB_NAME', 'kb_terminology'),
    'user': os.environ.get('DB_USER', 'postgres'),
    'password': os.environ.get('DB_PASSWORD', 'password'),
}


def parse_icd10_file(file_path: str) -> List[Dict]:
    """
    Parse CMS ICD-10-CM codes file.

    Format: Fixed-width with code (variable length) followed by tab and description
    Example: A000    Cholera due to Vibrio cholerae 01, biovar cholerae
    """
    records = []

    with open(file_path, 'r', encoding='utf-8', errors='replace') as f:
        for line_num, line in enumerate(f, 1):
            line = line.rstrip('\n\r')
            if not line.strip():
                continue

            # Split on first whitespace to get code and description
            parts = line.split(None, 1)
            if len(parts) >= 2:
                code = parts[0].strip()
                description = parts[1].strip()
            elif len(parts) == 1:
                code = parts[0].strip()
                description = f"ICD-10-CM Code {code}"
            else:
                continue

            # Validate ICD-10-CM code format (starts with letter, followed by digits)
            if not code or not code[0].isalpha():
                continue

            records.append({
                'code': code,
                'description': description,
                'line_num': line_num
            })

    return records


def load_icd10_concepts(conn, data_dir: str, batch_size: int = 5000, version: str = '2025'):
    """Load ICD-10-CM concepts from CMS text file."""

    # Look for the codes file
    codes_file = os.path.join(data_dir, f'icd10cm_codes_{version}.txt')
    if not os.path.exists(codes_file):
        # Try without version in filename
        codes_file = os.path.join(data_dir, 'icd10cm_codes.txt')
        if not os.path.exists(codes_file):
            print(f"❌ ICD-10-CM codes file not found in: {data_dir}")
            return 0

    print(f"📄 Loading ICD-10-CM codes from: {codes_file}")
    codes = parse_icd10_file(codes_file)
    print(f"   Found {len(codes):,} ICD-10-CM code records")

    if not codes:
        print("❌ No valid ICD-10-CM codes found in file")
        return 0

    # Insert into database
    cursor = conn.cursor()

    # Check if data already exists
    cursor.execute("SELECT COUNT(*) FROM concepts_icd10")
    existing = cursor.fetchone()[0]
    if existing > 0:
        print(f"⚠️  concepts_icd10 already has {existing:,} records")
        user_input = input("   Do you want to reload? (y/N): ").strip().lower()
        if user_input != 'y':
            print("   Skipping ICD-10 loading")
            cursor.close()
            return existing
        # Clear existing data
        print("   Clearing existing ICD-10 data...")
        cursor.execute("DELETE FROM concepts_icd10")
        conn.commit()

    # Disable triggers that conflict with partitioned table inserts
    print("🔧 Disabling triggers on concepts_icd10 for bulk insert...")
    cursor.execute("ALTER TABLE concepts_icd10 DISABLE TRIGGER update_concepts_search")
    cursor.execute("ALTER TABLE concepts_icd10 DISABLE TRIGGER trigger_update_concept_search_vector")
    conn.commit()

    # Prepare batch insert
    insert_sql = """
        INSERT INTO concepts_icd10
        (system, code, version, preferred_term, fully_specified_name, synonyms, active, properties)
        VALUES %s
        ON CONFLICT (system, code, version) DO UPDATE SET
            preferred_term = EXCLUDED.preferred_term,
            fully_specified_name = EXCLUDED.fully_specified_name,
            active = EXCLUDED.active,
            updated_at = NOW()
    """

    batch = []
    loaded = 0
    skipped = 0

    print(f"🔄 Loading ICD-10-CM codes in batches of {batch_size}...")

    for code_entry in codes:
        code = code_entry['code']
        description = code_entry['description']

        # Truncate to avoid phonetics trigger 255 byte limit
        preferred_term = description[:200] if description else f'ICD-10-CM {code}'
        fsn = description[:1000] if description else None

        # Build record tuple matching concepts_icd10 table structure
        # Note: Partition constraint requires system = 'ICD10' (not 'ICD10CM')
        record = (
            'ICD10',                              # system (must match partition constraint)
            code,                                 # code
            version,                              # version
            preferred_term,                       # preferred_term (truncated for phonetics)
            fsn,                                  # fully_specified_name
            [],                                   # synonyms (empty for ICD-10)
            True,                                 # active
            '{}'                                  # properties (empty JSON)
        )
        batch.append(record)

        if len(batch) >= batch_size:
            try:
                execute_values(
                    cursor, insert_sql, batch,
                    template="(%s, %s, %s, %s, %s, %s, %s, %s::jsonb)"
                )
                conn.commit()
                loaded += len(batch)
                print(f"   Loaded {loaded:,} / {len(codes):,} ({100*loaded/len(codes):.1f}%)")
            except Exception as e:
                print(f"❌ Error inserting batch: {e}")
                conn.rollback()
                skipped += len(batch)
            batch = []

    # Insert remaining records
    if batch:
        try:
            execute_values(
                cursor, insert_sql, batch,
                template="(%s, %s, %s, %s, %s, %s, %s, %s::jsonb)"
            )
            conn.commit()
            loaded += len(batch)
        except Exception as e:
            print(f"❌ Error inserting final batch: {e}")
            conn.rollback()
            skipped += len(batch)

    print(f"✅ ICD-10-CM loading complete: {loaded:,} codes loaded, {skipped:,} skipped")

    # Update search vectors for full-text search (manually since triggers were disabled)
    print("🔄 Updating search vectors...")
    cursor.execute("""
        UPDATE concepts_icd10
        SET search_vector = to_tsvector('english',
            coalesce(preferred_term, '') || ' ' ||
            coalesce(code, '')
        )
    """)
    conn.commit()
    print("✅ Search vectors updated")

    # Re-enable triggers
    print("🔧 Re-enabling triggers on concepts_icd10...")
    cursor.execute("ALTER TABLE concepts_icd10 ENABLE TRIGGER update_concepts_search")
    cursor.execute("ALTER TABLE concepts_icd10 ENABLE TRIGGER trigger_update_concept_search_vector")
    conn.commit()
    print("✅ Triggers re-enabled")

    cursor.close()
    return loaded


def main():
    parser = argparse.ArgumentParser(description='Load ICD-10-CM into KB-7 Terminology Service')
    parser.add_argument('--data', required=True, help='Data directory containing ICD-10-CM files')
    parser.add_argument('--batch', type=int, default=5000, help='Batch size for inserts')
    parser.add_argument('--version', default='2025', help='ICD-10-CM version year')
    parser.add_argument('--force', action='store_true', help='Force reload even if data exists')
    args = parser.parse_args()

    print("=" * 60)
    print("KB-7 ICD-10-CM Direct Loader")
    print("=" * 60)
    print()

    # Connect to database
    print(f"🔌 Connecting to PostgreSQL at {DB_CONFIG['host']}:{DB_CONFIG['port']}...")
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        print("   ✅ Connected successfully")
    except Exception as e:
        print(f"   ❌ Connection failed: {e}")
        sys.exit(1)

    try:
        # If --force flag, clear existing data first
        if args.force:
            cursor = conn.cursor()
            print("🔄 Force mode: Clearing existing ICD-10 data...")
            cursor.execute("DELETE FROM concepts_icd10")
            conn.commit()
            cursor.close()

        # Load ICD-10-CM
        loaded = load_icd10_concepts(conn, args.data, args.batch, args.version)

        # Show summary
        print()
        print("=" * 60)
        cursor = conn.cursor()
        cursor.execute("SELECT COUNT(*) FROM concepts_icd10")
        total = cursor.fetchone()[0]
        cursor.close()
        print(f"📊 Total ICD-10-CM concepts in database: {total:,}")
        print("=" * 60)

    finally:
        conn.close()

    print()
    print("✅ ICD-10-CM loading complete!")


if __name__ == '__main__':
    main()
