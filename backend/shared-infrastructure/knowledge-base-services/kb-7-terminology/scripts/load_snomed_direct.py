#!/usr/bin/env python3
"""
Direct SNOMED-CT Loader for KB-7 Terminology Service
Loads SNOMED RF2 files directly into PostgreSQL, bypassing Elasticsearch requirement.

Usage:
    python load_snomed_direct.py --data ./data/snomed/snapshot --batch 5000
"""

import os
import sys
import csv
import argparse
from datetime import datetime
from typing import Dict, List, Optional, Tuple
import psycopg2
from psycopg2.extras import execute_values

# Increase CSV field size limit for large SNOMED descriptions
csv.field_size_limit(sys.maxsize)

# Database connection settings
DB_CONFIG = {
    'host': os.environ.get('DB_HOST', 'localhost'),
    'port': int(os.environ.get('DB_PORT', '5437')),
    'dbname': os.environ.get('DB_NAME', 'kb_terminology'),
    'user': os.environ.get('DB_USER', 'postgres'),
    'password': os.environ.get('DB_PASSWORD', 'password'),
}


def parse_rf2_file(file_path: str, delimiter: str = '\t') -> List[Dict]:
    """Parse RF2 tab-separated file into list of dicts."""
    records = []
    with open(file_path, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter=delimiter)
        for row in reader:
            records.append(row)
    return records


def load_snomed_concepts(conn, data_dir: str, batch_size: int = 5000, version: str = '20250701'):
    """Load SNOMED concepts from RF2 files."""

    # Parse concept file
    concept_file = os.path.join(data_dir, 'sct2_Concept_Snapshot_INT.txt')
    if not os.path.exists(concept_file):
        print(f"❌ Concept file not found: {concept_file}")
        return 0

    print(f"📄 Loading concepts from: {concept_file}")
    concepts = parse_rf2_file(concept_file)
    print(f"   Found {len(concepts):,} concept records")

    # Parse description file for preferred terms
    desc_file = os.path.join(data_dir, 'sct2_Description_Snapshot-en_INT.txt')
    if not os.path.exists(desc_file):
        print(f"❌ Description file not found: {desc_file}")
        return 0

    print(f"📄 Loading descriptions from: {desc_file}")
    descriptions = parse_rf2_file(desc_file)
    print(f"   Found {len(descriptions):,} description records")

    # Build description lookup: concept_id -> {preferred_term, fsn, synonyms}
    print("🔄 Building description index...")
    desc_lookup: Dict[str, Dict] = {}

    # Type IDs:
    # 900000000000003001 = Fully Specified Name
    # 900000000000013009 = Synonym
    FSN_TYPE = '900000000000003001'
    SYN_TYPE = '900000000000013009'

    # Acceptability:
    # 900000000000548007 = Preferred
    # 900000000000549004 = Acceptable
    PREFERRED = '900000000000548007'

    for desc in descriptions:
        if desc.get('active') != '1':
            continue

        concept_id = desc.get('conceptId')
        term = desc.get('term', '')
        type_id = desc.get('typeId')

        if concept_id not in desc_lookup:
            desc_lookup[concept_id] = {
                'preferred_term': '',
                'fsn': '',
                'synonyms': []
            }

        if type_id == FSN_TYPE:
            desc_lookup[concept_id]['fsn'] = term
            # Use FSN as preferred term if not set
            if not desc_lookup[concept_id]['preferred_term']:
                # Remove semantic tag for display
                if ' (' in term:
                    desc_lookup[concept_id]['preferred_term'] = term.rsplit(' (', 1)[0]
                else:
                    desc_lookup[concept_id]['preferred_term'] = term
        elif type_id == SYN_TYPE:
            desc_lookup[concept_id]['synonyms'].append(term)
            # Prefer synonym as display name if it's shorter
            if not desc_lookup[concept_id]['preferred_term'] or len(term) < len(desc_lookup[concept_id]['preferred_term']):
                desc_lookup[concept_id]['preferred_term'] = term

    print(f"   Indexed {len(desc_lookup):,} concepts with descriptions")

    # Insert into database
    cursor = conn.cursor()

    # First check if data already exists
    cursor.execute("SELECT COUNT(*) FROM concepts_snomed")
    existing = cursor.fetchone()[0]
    if existing > 0:
        print(f"✅ concepts_snomed already has {existing:,} records - skipping")
        cursor.close()
        return existing

    # Prepare batch insert
    insert_sql = """
        INSERT INTO concepts_snomed
        (system, code, version, preferred_term, fully_specified_name, synonyms, active, properties)
        VALUES %s
        ON CONFLICT (system, code, version) DO UPDATE SET
            preferred_term = EXCLUDED.preferred_term,
            fully_specified_name = EXCLUDED.fully_specified_name,
            synonyms = EXCLUDED.synonyms,
            active = EXCLUDED.active,
            updated_at = NOW()
    """

    batch = []
    loaded = 0
    skipped = 0

    print(f"🔄 Loading concepts in batches of {batch_size}...")

    for concept in concepts:
        concept_id = concept.get('id')
        active = concept.get('active') == '1'

        # Get descriptions
        desc_info = desc_lookup.get(concept_id, {})
        preferred_term = desc_info.get('preferred_term', f'SNOMED Concept {concept_id}')
        fsn = desc_info.get('fsn', '')
        synonyms = desc_info.get('synonyms', [])

        # Skip if no meaningful description
        if not preferred_term and not fsn:
            skipped += 1
            continue

        # Build record
        record = (
            'SNOMED',                           # system
            concept_id,                         # code
            version,                            # version
            preferred_term[:500] if preferred_term else f'Concept {concept_id}',  # preferred_term
            fsn[:1000] if fsn else None,        # fully_specified_name
            synonyms[:20] if synonyms else [],  # synonyms (limit to 20)
            active,                             # active
            '{}'                                # properties (empty JSON)
        )
        batch.append(record)

        if len(batch) >= batch_size:
            execute_values(cursor, insert_sql, batch, template="(%s, %s, %s, %s, %s, %s, %s, %s::jsonb)")
            conn.commit()
            loaded += len(batch)
            print(f"   Loaded {loaded:,} / {len(concepts):,} ({100*loaded/len(concepts):.1f}%)")
            batch = []

    # Insert remaining
    if batch:
        execute_values(cursor, insert_sql, batch, template="(%s, %s, %s, %s, %s, %s, %s, %s::jsonb)")
        conn.commit()
        loaded += len(batch)

    print(f"✅ SNOMED loading complete: {loaded:,} concepts loaded, {skipped:,} skipped")

    # Update search vectors
    print("🔄 Updating search vectors...")
    cursor.execute("""
        UPDATE concepts_snomed
        SET search_vector = to_tsvector('english',
            coalesce(preferred_term, '') || ' ' ||
            coalesce(fully_specified_name, '') || ' ' ||
            coalesce(code, '')
        )
        WHERE search_vector IS NULL
    """)
    conn.commit()
    print("✅ Search vectors updated")

    cursor.close()
    return loaded


def load_loinc_concepts(conn, data_dir: str, batch_size: int = 5000, version: str = '20250321'):
    """Load LOINC concepts from SNOMED-LOINC Extension RF2 files."""

    # Parse concept file
    concept_file = os.path.join(data_dir, 'sct2_Concept_Snapshot_LO1010000_20250321.txt')
    if not os.path.exists(concept_file):
        print(f"❌ LOINC concept file not found: {concept_file}")
        return 0

    print(f"📄 Loading LOINC concepts from: {concept_file}")
    concepts = parse_rf2_file(concept_file)
    print(f"   Found {len(concepts):,} LOINC concept records")

    # Parse description file
    desc_file = os.path.join(data_dir, 'sct2_Description_Snapshot-en_LO1010000_20250321.txt')
    if os.path.exists(desc_file):
        descriptions = parse_rf2_file(desc_file)
        print(f"   Found {len(descriptions):,} description records")
    else:
        descriptions = []
        print("   No description file found")

    # Parse identifier file for LOINC codes
    id_file = os.path.join(data_dir, 'sct2_Identifier_Snapshot_LO1010000_20250321.txt')
    if os.path.exists(id_file):
        identifiers = parse_rf2_file(id_file)
        print(f"   Found {len(identifiers):,} identifier records (LOINC codes)")
    else:
        identifiers = []
        print("   No identifier file found")

    # Build SNOMED -> LOINC code mapping
    snomed_to_loinc: Dict[str, str] = {}
    for ident in identifiers:
        if ident.get('active') == '1':
            loinc_code = ident.get('alternateIdentifier', '')
            snomed_ref = ident.get('referencedComponentId', '')
            if loinc_code and snomed_ref:
                snomed_to_loinc[snomed_ref] = loinc_code

    print(f"   Mapped {len(snomed_to_loinc):,} SNOMED IDs to LOINC codes")

    # Build description lookup
    desc_lookup: Dict[str, str] = {}
    FSN_TYPE = '900000000000003001'
    for desc in descriptions:
        if desc.get('active') == '1':
            concept_id = desc.get('conceptId')
            term = desc.get('term', '')
            type_id = desc.get('typeId')
            if type_id == FSN_TYPE:
                # Remove semantic tag
                if ' (' in term:
                    desc_lookup[concept_id] = term.rsplit(' (', 1)[0]
                else:
                    desc_lookup[concept_id] = term
            elif concept_id not in desc_lookup:
                desc_lookup[concept_id] = term

    # Insert into database
    cursor = conn.cursor()

    # Check existing
    cursor.execute("SELECT COUNT(*) FROM concepts_loinc")
    existing = cursor.fetchone()[0]
    if existing > 0:
        print(f"✅ concepts_loinc already has {existing:,} records - skipping")
        cursor.close()
        return existing

    insert_sql = """
        INSERT INTO concepts_loinc
        (system, code, version, preferred_term, fully_specified_name, active, properties)
        VALUES %s
        ON CONFLICT (system, code, version) DO UPDATE SET
            preferred_term = EXCLUDED.preferred_term,
            fully_specified_name = EXCLUDED.fully_specified_name,
            active = EXCLUDED.active,
            updated_at = NOW()
    """

    batch = []
    loaded = 0

    print(f"🔄 Loading LOINC concepts...")

    for concept in concepts:
        snomed_id = concept.get('id')
        active = concept.get('active') == '1'

        # Get LOINC code from identifier mapping
        loinc_code = snomed_to_loinc.get(snomed_id, snomed_id)

        # Get description
        preferred_term = desc_lookup.get(snomed_id, f'LOINC {loinc_code}')

        # Truncate to avoid phonetics trigger 255 byte limit
        record = (
            'LOINC',
            loinc_code,
            version,
            preferred_term[:200],  # Keep under 255 for metaphone
            preferred_term[:500],
            active,
            '{}'
        )
        batch.append(record)

        if len(batch) >= batch_size:
            execute_values(cursor, insert_sql, batch, template="(%s, %s, %s, %s, %s, %s, %s::jsonb)")
            conn.commit()
            loaded += len(batch)
            print(f"   Loaded {loaded:,}")
            batch = []

    if batch:
        execute_values(cursor, insert_sql, batch, template="(%s, %s, %s, %s, %s, %s, %s::jsonb)")
        conn.commit()
        loaded += len(batch)

    print(f"✅ LOINC loading complete: {loaded:,} concepts")

    # Update search vectors
    cursor.execute("""
        UPDATE concepts_loinc
        SET search_vector = to_tsvector('english',
            coalesce(preferred_term, '') || ' ' || coalesce(code, '')
        )
        WHERE search_vector IS NULL
    """)
    conn.commit()

    cursor.close()
    return loaded


def main():
    parser = argparse.ArgumentParser(description='Load SNOMED/LOINC into KB-7')
    parser.add_argument('--data', required=True, help='Data directory')
    parser.add_argument('--system', default='snomed', choices=['snomed', 'loinc', 'all'], help='System to load')
    parser.add_argument('--batch', type=int, default=5000, help='Batch size')
    parser.add_argument('--version', default=None, help='Version string')
    args = parser.parse_args()

    print("=" * 60)
    print("KB-7 Direct Terminology Loader")
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
        if args.system in ['snomed', 'all']:
            version = args.version or '20250701'
            load_snomed_concepts(conn, args.data, args.batch, version)
            print()

        if args.system in ['loinc', 'all']:
            # For LOINC, look in loinc/snapshot subdirectory
            loinc_dir = args.data
            if not os.path.exists(os.path.join(loinc_dir, 'sct2_Concept_Snapshot_LO1010000_20250321.txt')):
                loinc_dir = os.path.join(os.path.dirname(args.data), 'loinc', 'snapshot')
            version = args.version or '20250321'
            load_loinc_concepts(conn, loinc_dir, args.batch, version)

    finally:
        conn.close()

    print()
    print("=" * 60)
    print("✅ Loading complete!")
    print("=" * 60)


if __name__ == '__main__':
    main()
