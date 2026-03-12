#!/usr/bin/env python3
"""
ValueSet Materialization Job - BUILD TIME ONLY

This script runs Neo4j traversal for intensional ValueSets and stores
the precomputed codes in PostgreSQL for runtime $expand operations.

CRITICAL ARCHITECTURE (CTO/CMO Directive):
"CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."

This job runs:
- At deploy/build time
- On SNOMED release update
- On Ontoserver ValueSet refresh
- Manually for cache refresh

After running this script:
- The precomputed_valueset_codes table contains all expanded codes
- The $expand endpoint returns precomputed codes from PostgreSQL only
- Neo4j is NOT used at runtime

Usage:
    python materialize_expansions.py                    # Materialize all
    python materialize_expansions.py --limit 100       # Test with limited set
    python materialize_expansions.py --valueset clinical-condition-1  # Single ValueSet
    python materialize_expansions.py --dry-run         # Preview without changes
"""

import os
import sys
import argparse
import psycopg2
from datetime import datetime
from typing import Optional, List, Tuple
from tqdm import tqdm

# Neo4j driver
try:
    from neo4j import GraphDatabase
    HAS_NEO4J = True
except ImportError:
    HAS_NEO4J = False
    print("WARNING: neo4j driver not installed. Install with: pip install neo4j")

# ============================================================================
# Configuration
# ============================================================================

# SNOMED Release Version - Update this when SNOMED is updated
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

# Neo4j config
NEO4J_CONFIG = {
    "uri": os.getenv("NEO4J_URI", "bolt://localhost:7688"),
    "user": os.getenv("NEO4J_USER", "neo4j"),
    "password": os.getenv("NEO4J_PASSWORD", "password")
}

# Neo4j Cypher query for subClassOf* traversal
# Gets root concept and all descendants
# Note: Proper aggregation grouping - collect root first, then combine
NEO4J_EXPANSION_QUERY = """
MATCH (root:Resource)
WHERE root.uri ENDS WITH '/' + $rootCode
WITH root
OPTIONAL MATCH (child:Resource)-[:subClassOf*]->(root)
WITH root, collect(DISTINCT child) AS children
WITH [root] + [c IN children WHERE c IS NOT NULL] AS concepts
UNWIND concepts AS c
RETURN
    replace(c.uri, 'http://snomed.info/id/', '') AS code,
    c.prefLabel AS display
LIMIT 50000
"""

# ============================================================================
# Materialization Engine
# ============================================================================

class MaterializationEngine:
    """
    Materializes intensional ValueSet expansions via Neo4j.

    RUNS AT BUILD TIME ONLY - NOT AT RUNTIME!

    Flow:
    1. Read ValueSets with root_code from PostgreSQL
    2. For each: query Neo4j for descendants
    3. Insert codes into precomputed_valueset_codes
    4. Neo4j no longer needed for runtime $expand
    """

    def __init__(self, dry_run: bool = False):
        self.dry_run = dry_run
        self.pg_conn = None
        self.neo4j_driver = None
        self.stats = {
            "valuesets_processed": 0,
            "valuesets_materialized": 0,
            "valuesets_skipped": 0,
            "codes_inserted": 0,
            "errors": []
        }

    def connect(self):
        """Connect to PostgreSQL and Neo4j."""
        if self.dry_run:
            print("DRY RUN MODE - No database changes will be made")
            return

        # PostgreSQL
        self.pg_conn = psycopg2.connect(**POSTGRES_CONFIG)
        self.pg_conn.autocommit = False
        print(f"Connected to PostgreSQL: {POSTGRES_CONFIG['host']}:{POSTGRES_CONFIG['port']}")

        # Neo4j
        if not HAS_NEO4J:
            raise RuntimeError("Neo4j driver not installed. Install with: pip install neo4j")

        self.neo4j_driver = GraphDatabase.driver(
            NEO4J_CONFIG["uri"],
            auth=(NEO4J_CONFIG["user"], NEO4J_CONFIG["password"])
        )
        print(f"Connected to Neo4j: {NEO4J_CONFIG['uri']}")

        # Verify Neo4j connectivity
        with self.neo4j_driver.session() as session:
            result = session.run("RETURN 1 AS test")
            result.single()
        print("Neo4j connection verified")

    def close(self):
        """Close database connections."""
        if self.pg_conn:
            self.pg_conn.close()
        if self.neo4j_driver:
            self.neo4j_driver.close()

    def get_intensional_valuesets(self, limit: Optional[int] = None,
                                    valueset_filter: Optional[str] = None) -> List[Tuple]:
        """Get ValueSets that need materialization (have root_code)."""
        if self.dry_run:
            return []

        cursor = self.pg_conn.cursor()

        query = """
            SELECT id, url, name, root_code, root_system, definition_type
            FROM value_sets
            WHERE root_code IS NOT NULL
              AND definition_type IN ('refset', 'concept_in', 'ecl', 'intensional')
              AND status = 'active'
        """
        params = []

        if valueset_filter:
            query += " AND (name ILIKE %s OR url ILIKE %s)"
            params.extend([f"%{valueset_filter}%", f"%{valueset_filter}%"])

        query += " ORDER BY name"

        if limit:
            query += f" LIMIT {limit}"

        cursor.execute(query, params)
        results = cursor.fetchall()
        cursor.close()

        return results

    def materialize_valueset(self, valueset_id: str, valueset_url: str,
                              root_code: str, display_name: str) -> int:
        """
        Materialize a single ValueSet via Neo4j traversal.

        This is the BUILD TIME Neo4j query - runs ONCE per ValueSet.
        Results are stored in PostgreSQL for runtime use.
        """
        if self.dry_run:
            print(f"  [DRY RUN] Would materialize: {display_name} (root: {root_code})")
            return 0

        pg_cursor = self.pg_conn.cursor()

        # Step 1: Delete existing expansion for this version (idempotent)
        pg_cursor.execute("""
            DELETE FROM precomputed_valueset_codes
            WHERE valueset_url = %s AND snomed_version = %s
        """, (valueset_url, SNOMED_RELEASE))

        # Step 2: Query Neo4j for all descendants
        codes_inserted = 0
        with self.neo4j_driver.session() as session:
            result = session.run(NEO4J_EXPANSION_QUERY, rootCode=root_code)
            rows = list(result)

            if not rows:
                print(f"  WARNING: No codes found for root {root_code} ({display_name})")
                self.stats["errors"].append({
                    "valueset": display_name,
                    "root_code": root_code,
                    "error": "No codes returned from Neo4j"
                })
                self.pg_conn.commit()
                return 0

            # Step 3: Insert precomputed codes into PostgreSQL
            for row in rows:
                code = row["code"]
                display = row["display"]

                if not code:
                    continue

                pg_cursor.execute("""
                    INSERT INTO precomputed_valueset_codes
                    (valueset_url, valueset_id, snomed_version, code_system, code, display)
                    VALUES (%s, %s, %s, %s, %s, %s)
                    ON CONFLICT (valueset_url, snomed_version, code_system, code) DO NOTHING
                """, (
                    valueset_url,
                    valueset_id,
                    SNOMED_RELEASE,
                    SNOMED_SYSTEM,
                    code,
                    display
                ))
                codes_inserted += 1

        # Commit this ValueSet's expansion
        self.pg_conn.commit()
        self.stats["codes_inserted"] += codes_inserted

        return codes_inserted

    def materialize_all(self, limit: Optional[int] = None,
                        valueset_filter: Optional[str] = None):
        """
        Materialize all intensional ValueSets.

        This is the main BUILD TIME job that:
        1. Gets all ValueSets with root codes
        2. Runs Neo4j traversal for each
        3. Stores results in precomputed_valueset_codes
        """
        print(f"\n{'='*70}")
        print(f"  VALUESET MATERIALIZATION JOB - BUILD TIME ONLY")
        print(f"{'='*70}")
        print(f"  SNOMED Release: {SNOMED_RELEASE}")
        print(f"  PostgreSQL: {POSTGRES_CONFIG['host']}:{POSTGRES_CONFIG['port']}")
        print(f"  Neo4j: {NEO4J_CONFIG['uri']}")
        print(f"  Limit: {limit or 'None'}")
        print(f"  Filter: {valueset_filter or 'None'}")
        print(f"  Dry Run: {self.dry_run}")
        print(f"{'='*70}\n")

        self.connect()

        # Get ValueSets to materialize
        valuesets = self.get_intensional_valuesets(limit, valueset_filter)
        total = len(valuesets)
        print(f"Found {total:,} intensional ValueSets to materialize\n")

        if total == 0:
            print("No ValueSets found. Run load_valuesets_with_roots.py first!")
            self.close()
            return

        # Progress bar
        progress = tqdm(valuesets, desc="Materializing", unit="vs")

        for vs_id, vs_url, vs_name, root_code, root_system, def_type in progress:
            self.stats["valuesets_processed"] += 1
            progress.set_postfix_str(f"{vs_name[:30]}...")

            try:
                codes = self.materialize_valueset(
                    str(vs_id), vs_url, root_code, vs_name
                )

                if codes > 0:
                    self.stats["valuesets_materialized"] += 1
                else:
                    self.stats["valuesets_skipped"] += 1

            except Exception as e:
                self.stats["errors"].append({
                    "valueset": vs_name,
                    "error": str(e)
                })
                # Continue with next ValueSet
                continue

        self.close()
        self._print_summary()

    def _print_summary(self):
        """Print materialization summary."""
        print(f"\n{'='*70}")
        print(f"  MATERIALIZATION COMPLETE!")
        print(f"{'='*70}")
        print(f"  SNOMED Release: {SNOMED_RELEASE}")
        print(f"  ValueSets Processed:    {self.stats['valuesets_processed']:,}")
        print(f"  ValueSets Materialized: {self.stats['valuesets_materialized']:,}")
        print(f"  ValueSets Skipped:      {self.stats['valuesets_skipped']:,}")
        print(f"  Total Codes Inserted:   {self.stats['codes_inserted']:,}")
        print(f"  Errors:                 {len(self.stats['errors'])}")
        print(f"{'='*70}\n")

        print("NEXT STEPS:")
        print("  1. Verify codes in precomputed_valueset_codes table")
        print("  2. Test $expand endpoint: curl http://localhost:8087/fhir/ValueSet/clinical-condition-1/$expand")
        print("  3. Response should be <50ms (pure PostgreSQL read)")
        print("  4. Neo4j is NO LONGER NEEDED for runtime $expand")
        print("")

        if self.stats['errors']:
            print(f"First {min(10, len(self.stats['errors']))} errors:")
            for err in self.stats['errors'][:10]:
                print(f"  - {err['valueset']}: {err['error']}")


# ============================================================================
# Main Entry Point
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Materialize ValueSet expansions via Neo4j (BUILD TIME ONLY)"
    )
    parser.add_argument(
        "--limit", "-l",
        type=int,
        help="Limit number of ValueSets to materialize (for testing)"
    )
    parser.add_argument(
        "--valueset", "-v",
        help="Materialize specific ValueSet (by name or URL substring)"
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
    parser.add_argument(
        "--neo4j-uri",
        default=os.getenv("NEO4J_URI", "bolt://localhost:7688"),
        help="Neo4j URI"
    )

    args = parser.parse_args()

    # Update config from args
    global SNOMED_RELEASE
    SNOMED_RELEASE = args.snomed_version
    POSTGRES_CONFIG["host"] = args.pg_host
    POSTGRES_CONFIG["port"] = args.pg_port
    NEO4J_CONFIG["uri"] = args.neo4j_uri

    # Verify Neo4j driver
    if not HAS_NEO4J and not args.dry_run:
        print("ERROR: Neo4j driver not installed. Install with: pip install neo4j")
        sys.exit(1)

    # Run materialization
    engine = MaterializationEngine(dry_run=args.dry_run)
    engine.materialize_all(limit=args.limit, valueset_filter=args.valueset)


if __name__ == "__main__":
    main()
