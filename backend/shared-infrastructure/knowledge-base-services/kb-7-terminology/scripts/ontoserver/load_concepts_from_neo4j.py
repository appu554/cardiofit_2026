#!/usr/bin/env python3
"""
Load SNOMED/LOINC concepts from Neo4j AU into PostgreSQL concepts table.

This script extracts concepts from the Neo4j AU graph database and loads them
into the PostgreSQL concepts table for runtime terminology lookups.

Usage:
    python load_concepts_from_neo4j.py                     # Load all concepts
    python load_concepts_from_neo4j.py --limit 1000       # Test with limited set
    python load_concepts_from_neo4j.py --system snomed    # Load only SNOMED
    python load_concepts_from_neo4j.py --dry-run          # Preview without changes
"""

import os
import sys
import argparse
import psycopg2
from psycopg2.extras import execute_batch
from datetime import datetime
from typing import Optional, List, Dict, Any
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

SNOMED_VERSION = os.getenv("SNOMED_RELEASE", "20241130")
LOINC_VERSION = os.getenv("LOINC_VERSION", "2.77")

# PostgreSQL config
POSTGRES_CONFIG = {
    "host": os.getenv("POSTGRES_HOST", "localhost"),
    "port": int(os.getenv("POSTGRES_PORT", "5437")),
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

# System code mappings (must fit varchar(20) in concepts table)
# The concepts table uses short codes like "SNOMED", not full URIs
SYSTEM_CODES = {
    "snomed": "SNOMED",
    "loinc": "LOINC"
}

# Neo4j queries
NEO4J_SNOMED_QUERY = """
MATCH (r:Resource)
WHERE r.uri STARTS WITH 'http://snomed.info/id/'
OPTIONAL MATCH (r)-[:subClassOf]->(parent:Resource)
WITH r, collect(DISTINCT parent.uri) as parents
RETURN
    replace(r.uri, 'http://snomed.info/id/', '') AS code,
    CASE
        WHEN r.prefLabel IS NULL THEN 'Unknown'
        WHEN r.prefLabel IS NOT NULL AND size(r.prefLabel) > 0 THEN r.prefLabel[0]
        ELSE 'Unknown'
    END AS preferred_term,
    CASE
        WHEN r.prefLabel IS NOT NULL AND size(r.prefLabel) > 1 THEN r.prefLabel[1..]
        ELSE []
    END AS synonyms,
    [p IN parents | replace(p, 'http://snomed.info/id/', '')] AS parent_codes
SKIP $skip
LIMIT $limit
"""

NEO4J_LOINC_QUERY = """
MATCH (r:Resource)
WHERE r.uri STARTS WITH 'http://loinc.org/'
OPTIONAL MATCH (r)-[:subClassOf]->(parent:Resource)
WITH r, collect(DISTINCT parent.uri) as parents
RETURN
    replace(r.uri, 'http://loinc.org/', '') AS code,
    CASE
        WHEN r.prefLabel IS NULL THEN 'Unknown'
        WHEN r.prefLabel IS NOT NULL AND size(r.prefLabel) > 0 THEN r.prefLabel[0]
        ELSE 'Unknown'
    END AS preferred_term,
    CASE
        WHEN r.prefLabel IS NOT NULL AND size(r.prefLabel) > 1 THEN r.prefLabel[1..]
        ELSE []
    END AS synonyms,
    [p IN parents | replace(p, 'http://loinc.org/', '')] AS parent_codes
SKIP $skip
LIMIT $limit
"""

NEO4J_COUNT_QUERY = """
MATCH (r:Resource)
WHERE r.uri STARTS WITH $prefix
RETURN count(r) as total
"""

# ============================================================================
# Concept Loader Engine
# ============================================================================

class ConceptLoaderEngine:
    """
    Loads concepts from Neo4j AU into PostgreSQL.

    This is a BUILD TIME operation that populates the concepts table
    for runtime terminology lookups.
    """

    def __init__(self, dry_run: bool = False):
        self.dry_run = dry_run
        self.pg_conn = None
        self.neo4j_driver = None
        self.stats = {
            "concepts_processed": 0,
            "concepts_inserted": 0,
            "concepts_skipped": 0,
            "errors": []
        }
        self.batch_size = 1000

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

    def get_concept_count(self, system: str) -> int:
        """Get total count of concepts for a system."""
        if self.dry_run:
            return 0

        prefix = "http://snomed.info/id/" if system == "snomed" else "http://loinc.org/"

        with self.neo4j_driver.session() as session:
            result = session.run(NEO4J_COUNT_QUERY, prefix=prefix)
            record = result.single()
            return record["total"] if record else 0

    def load_concepts(self, system: str, limit: Optional[int] = None) -> int:
        """
        Load concepts for a specific terminology system.

        Args:
            system: 'snomed' or 'loinc'
            limit: Maximum concepts to load (for testing)

        Returns:
            Number of concepts inserted
        """
        if self.dry_run:
            print(f"[DRY RUN] Would load {system.upper()} concepts")
            return 0

        system_code = SYSTEM_CODES.get(system)
        version = SNOMED_VERSION if system == "snomed" else LOINC_VERSION
        query = NEO4J_SNOMED_QUERY if system == "snomed" else NEO4J_LOINC_QUERY

        total = self.get_concept_count(system)
        if limit:
            total = min(total, limit)

        print(f"\nLoading {system.upper()} concepts: {total:,} total")

        pg_cursor = self.pg_conn.cursor()

        # Process in batches
        offset = 0
        batch = []
        progress = tqdm(total=total, desc=f"Loading {system.upper()}", unit="concepts")

        while offset < total:
            fetch_limit = min(self.batch_size, total - offset)

            with self.neo4j_driver.session() as session:
                result = session.run(query, skip=offset, limit=fetch_limit)

                for record in result:
                    code = record["code"]
                    preferred_term = record["preferred_term"] or "Unknown"
                    synonyms = record["synonyms"] or []
                    parent_codes = record["parent_codes"] or []

                    # Skip empty codes
                    if not code:
                        self.stats["concepts_skipped"] += 1
                        continue

                    # Clean up code (remove any prefixes)
                    code = code.split("/")[-1] if "/" in code else code

                    batch.append((
                        system_code,          # system (short code like "SNOMED")
                        code,                 # code
                        version,              # version
                        preferred_term[:500], # preferred_term (truncate)
                        None,                 # fully_specified_name
                        synonyms[:10],        # synonyms (limit to 10)
                        parent_codes[:20],    # parent_codes (limit to 20)
                        len(parent_codes) == 0,  # is_leaf (no parents = potential leaf)
                        True                  # active
                    ))

                    self.stats["concepts_processed"] += 1
                    progress.update(1)

                    # Insert batch when full
                    if len(batch) >= self.batch_size:
                        self._insert_batch(pg_cursor, batch)
                        batch = []

            offset += fetch_limit

        # Insert remaining batch
        if batch:
            self._insert_batch(pg_cursor, batch)

        progress.close()
        self.pg_conn.commit()

        return self.stats["concepts_inserted"]

    def _insert_batch(self, cursor, batch: List[tuple]):
        """Insert a batch of concepts into PostgreSQL."""
        insert_query = """
            INSERT INTO concepts (
                system, code, version, preferred_term, fully_specified_name,
                synonyms, parent_codes, is_leaf, active
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (system, code, version) DO UPDATE SET
                preferred_term = EXCLUDED.preferred_term,
                synonyms = EXCLUDED.synonyms,
                parent_codes = EXCLUDED.parent_codes,
                is_leaf = EXCLUDED.is_leaf,
                updated_at = NOW()
        """

        try:
            execute_batch(cursor, insert_query, batch, page_size=100)
            self.pg_conn.commit()  # Commit each batch immediately for reliability
            self.stats["concepts_inserted"] += len(batch)
        except Exception as e:
            self.pg_conn.rollback()
            self.stats["errors"].append(str(e))
            print(f"Batch insert error: {e}")

    def load_all(self, systems: List[str], limit: Optional[int] = None):
        """
        Load concepts for all specified systems.

        Args:
            systems: List of systems to load ('snomed', 'loinc')
            limit: Maximum concepts per system (for testing)
        """
        print(f"\n{'='*70}")
        print(f"  CONCEPT LOADER - NEO4J AU → POSTGRESQL")
        print(f"{'='*70}")
        print(f"  PostgreSQL: {POSTGRES_CONFIG['host']}:{POSTGRES_CONFIG['port']}")
        print(f"  Neo4j: {NEO4J_CONFIG['uri']}")
        print(f"  Systems: {', '.join(systems)}")
        print(f"  Limit: {limit or 'None'}")
        print(f"  Dry Run: {self.dry_run}")
        print(f"{'='*70}\n")

        self.connect()

        for system in systems:
            if system in SYSTEM_CODES:
                self.load_concepts(system, limit)
            else:
                print(f"Unknown system: {system}")

        self.close()
        self._print_summary()

    def _print_summary(self):
        """Print loading summary."""
        print(f"\n{'='*70}")
        print(f"  LOADING COMPLETE!")
        print(f"{'='*70}")
        print(f"  Concepts Processed:  {self.stats['concepts_processed']:,}")
        print(f"  Concepts Inserted:   {self.stats['concepts_inserted']:,}")
        print(f"  Concepts Skipped:    {self.stats['concepts_skipped']:,}")
        print(f"  Errors:              {len(self.stats['errors'])}")
        print(f"{'='*70}\n")

        print("NEXT STEPS:")
        print("  1. Verify concepts in PostgreSQL:")
        print("     SELECT system, COUNT(*) FROM concepts GROUP BY system;")
        print("  2. Test terminology lookup:")
        print("     curl http://localhost:8092/v1/concepts/snomed/10000006")
        print("")

        if self.stats['errors']:
            print(f"First {min(5, len(self.stats['errors']))} errors:")
            for err in self.stats['errors'][:5]:
                print(f"  - {err}")


# ============================================================================
# Main Entry Point
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Load concepts from Neo4j AU into PostgreSQL"
    )
    parser.add_argument(
        "--limit", "-l",
        type=int,
        help="Limit number of concepts per system (for testing)"
    )
    parser.add_argument(
        "--system", "-s",
        choices=["snomed", "loinc", "all"],
        default="all",
        help="Terminology system to load (default: all)"
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Preview without making database changes"
    )
    parser.add_argument(
        "--pg-host",
        default=os.getenv("POSTGRES_HOST", "localhost"),
        help="PostgreSQL host"
    )
    parser.add_argument(
        "--pg-port",
        type=int,
        default=int(os.getenv("POSTGRES_PORT", "5437")),
        help="PostgreSQL port"
    )
    parser.add_argument(
        "--neo4j-uri",
        default=os.getenv("NEO4J_URI", "bolt://localhost:7688"),
        help="Neo4j URI"
    )

    args = parser.parse_args()

    # Update config from args
    POSTGRES_CONFIG["host"] = args.pg_host
    POSTGRES_CONFIG["port"] = args.pg_port
    NEO4J_CONFIG["uri"] = args.neo4j_uri

    # Verify Neo4j driver
    if not HAS_NEO4J and not args.dry_run:
        print("ERROR: Neo4j driver not installed. Install with: pip install neo4j")
        sys.exit(1)

    # Determine systems to load
    systems = ["snomed", "loinc"] if args.system == "all" else [args.system]

    # Run loader
    engine = ConceptLoaderEngine(dry_run=args.dry_run)
    engine.load_all(systems=systems, limit=args.limit)


if __name__ == "__main__":
    main()
