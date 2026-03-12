#!/usr/bin/env python3
"""
materialize_explicit_expansions.py - Materialize Explicit ValueSet Expansions

PURPOSE (single responsibility):
    Copy `value_sets.expansion.contains[]` → into `precomputed_valueset_codes`

This script handles EXPLICIT ValueSets (those with pre-populated expansion JSONB).
It does NOT use Neo4j - that's handled by materialize_expansions.py for INTENSIONAL ValueSets.

ARCHITECTURE CONSTRAINT (CTO/CMO Directive):
    "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."

This is a BUILD-TIME job that:
    - Reads expansion data from value_sets.expansion JSONB
    - Inserts into precomputed_valueset_codes for O(1) runtime lookups
    - Is idempotent (safe to re-run)
    - Has no Neo4j dependency
    - Has no FHIR dependency

SAFETY GUARANTEES:
    ✅ Idempotent (uses ON CONFLICT DO NOTHING)
    ✅ No Neo4j dependency
    ✅ No FHIR dependency
    ✅ Deterministic
    ✅ Can be re-run anytime
    ✅ Does not mutate value_sets

Usage:
    POSTGRES_HOST=localhost POSTGRES_PORT=5432 POSTGRES_DB=kb_terminology \
    POSTGRES_USER=postgres POSTGRES_PASSWORD=password \
    python3 materialize_explicit_expansions.py
"""

import os
import sys
import json
import logging
from datetime import datetime

try:
    import psycopg2
    from psycopg2.extras import execute_values
except ImportError:
    print("ERROR: psycopg2 not installed. Run: pip install psycopg2-binary")
    sys.exit(1)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def get_db_connection():
    """Create database connection from environment variables."""
    return psycopg2.connect(
        host=os.environ.get('POSTGRES_HOST', 'localhost'),
        port=os.environ.get('POSTGRES_PORT', '5432'),
        dbname=os.environ.get('POSTGRES_DB', 'kb_terminology'),
        user=os.environ.get('POSTGRES_USER', 'postgres'),
        password=os.environ.get('POSTGRES_PASSWORD', 'password')
    )


def get_unmaterialized_valuesets(cur):
    """
    Get ValueSets that have expansion data but are not yet materialized.

    Selection criteria:
    - Have expansion IS NOT NULL
    - expansion.contains has at least one element
    - Do not already have rows in precomputed_valueset_codes
    """
    cur.execute("""
        SELECT vs.id, vs.url, vs.name, vs.expansion, vs.snomed_version
        FROM value_sets vs
        WHERE vs.expansion IS NOT NULL
          AND vs.expansion != '{}'::jsonb
          AND jsonb_array_length(COALESCE(vs.expansion->'contains', '[]'::jsonb)) > 0
          AND NOT EXISTS (
              SELECT 1
              FROM precomputed_valueset_codes p
              WHERE p.valueset_id = vs.id
          )
        ORDER BY vs.name
    """)
    return cur.fetchall()


def materialize_valueset(cur, vs_id, vs_url, vs_name, expansion, snomed_version):
    """
    Materialize a single ValueSet's expansion into precomputed_valueset_codes.

    Returns the number of codes inserted.
    """
    contains = expansion.get("contains", [])
    if not contains:
        return 0

    # Default SNOMED version if not specified
    version = snomed_version or "20240901"

    # Prepare batch insert data
    insert_data = []
    for concept in contains:
        code = concept.get("code")
        system = concept.get("system")
        display = concept.get("display", "")

        if not code or not system:
            logger.warning(f"  Skipping concept with missing code/system in {vs_name}")
            continue

        insert_data.append((
            vs_url,           # valueset_url
            vs_id,            # valueset_id
            version,          # snomed_version
            system,           # code_system
            code,             # code
            display           # display
        ))

    if not insert_data:
        return 0

    # Batch insert with ON CONFLICT DO NOTHING for idempotency
    execute_values(
        cur,
        """
        INSERT INTO precomputed_valueset_codes
            (valueset_url, valueset_id, snomed_version, code_system, code, display, materialized_at)
        VALUES %s
        ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING
        """,
        insert_data,
        template="(%s, %s, %s, %s, %s, %s, NOW())"
    )

    return len(insert_data)


def main():
    """Main materialization process."""
    logger.info("=" * 70)
    logger.info("KB-7 EXPLICIT VALUESET MATERIALIZATION")
    logger.info("=" * 70)
    logger.info("Purpose: Copy expansion.contains[] → precomputed_valueset_codes")
    logger.info("This is a BUILD-TIME job for O(1) runtime lookups")
    logger.info("=" * 70)

    try:
        conn = get_db_connection()
        cur = conn.cursor()

        # Get unmaterialized ValueSets
        logger.info("\n📊 Finding unmaterialized ValueSets...")
        valuesets = get_unmaterialized_valuesets(cur)

        if not valuesets:
            logger.info("✅ All ValueSets with expansion data are already materialized!")
            return

        logger.info(f"📋 Found {len(valuesets)} ValueSets to materialize")

        # Process each ValueSet
        total_codes = 0
        successful = 0
        failed = 0

        for vs_id, vs_url, vs_name, expansion, snomed_version in valuesets:
            try:
                # Parse expansion JSON if it's a string
                if isinstance(expansion, str):
                    expansion = json.loads(expansion)

                code_count = materialize_valueset(cur, vs_id, vs_url, vs_name, expansion, snomed_version)

                if code_count > 0:
                    logger.info(f"  ✅ {vs_name}: {code_count} codes")
                    total_codes += code_count
                    successful += 1
                else:
                    logger.info(f"  ⚠️  {vs_name}: No valid codes found")

            except Exception as e:
                logger.error(f"  ❌ {vs_name}: {str(e)}")
                failed += 1

        # Commit all changes
        conn.commit()

        # Summary
        logger.info("\n" + "=" * 70)
        logger.info("📊 MATERIALIZATION SUMMARY")
        logger.info("=" * 70)
        logger.info(f"  ValueSets processed: {len(valuesets)}")
        logger.info(f"  Successful: {successful}")
        logger.info(f"  Failed: {failed}")
        logger.info(f"  Total codes materialized: {total_codes}")
        logger.info("=" * 70)

        # Verification query
        cur.execute("""
            SELECT
                valueset_url,
                COUNT(*) as code_count
            FROM precomputed_valueset_codes
            WHERE valueset_url LIKE '%kb7.health%'
            GROUP BY valueset_url
            ORDER BY valueset_url
        """)
        kb7_results = cur.fetchall()

        if kb7_results:
            logger.info("\n📋 KB7.HEALTH VALUESETS NOW MATERIALIZED:")
            for url, count in kb7_results:
                name = url.split("/")[-1]
                logger.info(f"  • {name}: {count} codes")

        cur.close()
        conn.close()

        logger.info("\n✅ Materialization complete!")
        logger.info("   $validate-code will now work for these ValueSets")

    except psycopg2.Error as e:
        logger.error(f"Database error: {e}")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
