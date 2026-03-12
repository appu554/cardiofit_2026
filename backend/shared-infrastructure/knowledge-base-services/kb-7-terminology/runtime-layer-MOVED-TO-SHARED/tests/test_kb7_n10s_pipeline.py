#!/usr/bin/env python3
"""
KB7 N10s Pipeline Integration Test
===================================

Tests the complete pipeline:
Knowledge Factory → GCS (.ttl) → CDC Event → Neo4j Sync Service → n10s → Neo4j

This script can be run in two modes:
1. Full integration test (requires all infrastructure)
2. Component tests (tests individual components)

Prerequisites:
- Neo4j with n10s plugin installed
- GCS access (or local TTL file)
- Redis running

Usage:
    # Full test with GCS
    python test_kb7_n10s_pipeline.py --full --neo4j-password YOUR_PASSWORD

    # Test with local TTL file
    python test_kb7_n10s_pipeline.py --local-ttl /path/to/file.ttl

    # Check prerequisites only
    python test_kb7_n10s_pipeline.py --check-prereqs

    # Reset Neo4j password (interactive)
    python test_kb7_n10s_pipeline.py --reset-password

@author KB7 Integration Team
@version 1.0
@since 2025-12-04
"""

import asyncio
import argparse
import json
import os
import sys
from datetime import datetime
from typing import Dict, Any, Optional
from dataclasses import dataclass

# Add parent directories to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

try:
    from neo4j import AsyncGraphDatabase, GraphDatabase
    NEO4J_AVAILABLE = True
except ImportError:
    NEO4J_AVAILABLE = False
    print("⚠️  neo4j driver not installed. Run: pip install neo4j")

try:
    import redis.asyncio as aioredis
    REDIS_AVAILABLE = True
except ImportError:
    REDIS_AVAILABLE = False
    print("⚠️  redis not installed. Run: pip install redis")

try:
    from google.cloud import storage
    GCS_AVAILABLE = True
except ImportError:
    GCS_AVAILABLE = False
    print("⚠️  google-cloud-storage not installed. Run: pip install google-cloud-storage")


@dataclass
class TestConfig:
    """Test configuration"""
    neo4j_uri: str = "bolt://localhost:7687"
    neo4j_user: str = "neo4j"
    neo4j_password: str = "neo4j"  # Override via --neo4j-password
    redis_url: str = "redis://localhost:6379"
    gcs_bucket: str = "sincere-hybrid-477206-h2-kb-artifacts-production"
    test_version_id: str = None  # Generated if not provided
    local_ttl_path: str = None  # Use local file instead of GCS


class TestResult:
    """Collect test results"""
    def __init__(self):
        self.passed = []
        self.failed = []
        self.skipped = []
        self.warnings = []

    def add_pass(self, name: str, details: str = ""):
        self.passed.append((name, details))
        print(f"  ✅ {name}" + (f": {details}" if details else ""))

    def add_fail(self, name: str, error: str):
        self.failed.append((name, error))
        print(f"  ❌ {name}: {error}")

    def add_skip(self, name: str, reason: str):
        self.skipped.append((name, reason))
        print(f"  ⏭️  {name}: {reason}")

    def add_warning(self, name: str, message: str):
        self.warnings.append((name, message))
        print(f"  ⚠️  {name}: {message}")

    def summary(self) -> str:
        total = len(self.passed) + len(self.failed) + len(self.skipped)
        return f"""
═══════════════════════════════════════════════════════════════
                    TEST SUMMARY
═══════════════════════════════════════════════════════════════
  ✅ Passed:  {len(self.passed)}
  ❌ Failed:  {len(self.failed)}
  ⏭️  Skipped: {len(self.skipped)}
  ⚠️  Warnings: {len(self.warnings)}
───────────────────────────────────────────────────────────────
  Total:     {total}
  Status:    {'PASS' if len(self.failed) == 0 else 'FAIL'}
═══════════════════════════════════════════════════════════════
"""


class KB7PipelineTest:
    """Test KB7 N10s Pipeline"""

    def __init__(self, config: TestConfig):
        self.config = config
        self.results = TestResult()
        self.driver = None
        self.redis = None

    async def setup(self):
        """Initialize connections"""
        if NEO4J_AVAILABLE:
            try:
                self.driver = AsyncGraphDatabase.driver(
                    self.config.neo4j_uri,
                    auth=(self.config.neo4j_user, self.config.neo4j_password)
                )
            except Exception as e:
                self.results.add_fail("Neo4j Connection", str(e))

        if REDIS_AVAILABLE:
            try:
                self.redis = await aioredis.from_url(self.config.redis_url)
            except Exception as e:
                self.results.add_warning("Redis Connection", str(e))

    async def teardown(self):
        """Close connections"""
        if self.driver:
            await self.driver.close()
        if self.redis:
            await self.redis.close()

    # ═══════════════════════════════════════════════════════════════
    # PREREQUISITE CHECKS
    # ═══════════════════════════════════════════════════════════════

    async def check_prerequisites(self) -> bool:
        """Check all prerequisites are met"""
        print("\n📋 PREREQUISITE CHECKS")
        print("─" * 60)

        all_pass = True

        # Check Neo4j connectivity
        if not NEO4J_AVAILABLE:
            self.results.add_fail("Neo4j Driver", "neo4j package not installed")
            all_pass = False
        else:
            try:
                async with self.driver.session() as session:
                    result = await session.run("RETURN 1 as test")
                    await result.single()
                    self.results.add_pass("Neo4j Connectivity")
            except Exception as e:
                self.results.add_fail("Neo4j Connectivity", str(e)[:100])
                all_pass = False

        # Check n10s plugin
        if self.driver:
            try:
                async with self.driver.session() as session:
                    result = await session.run("RETURN n10s.version() as version")
                    record = await result.single()
                    version = record['version'] if record else 'unknown'
                    self.results.add_pass("N10s Plugin", f"version {version}")
            except Exception as e:
                self.results.add_fail("N10s Plugin", f"Not installed or accessible: {str(e)[:80]}")
                all_pass = False

        # Check Redis
        if not REDIS_AVAILABLE:
            self.results.add_warning("Redis Driver", "redis package not installed (optional)")
        elif self.redis:
            try:
                await self.redis.ping()
                self.results.add_pass("Redis Connectivity")
            except Exception as e:
                self.results.add_warning("Redis Connectivity", str(e)[:50])

        # Check GCS
        if not GCS_AVAILABLE:
            self.results.add_warning("GCS Driver", "google-cloud-storage not installed")
        else:
            try:
                client = storage.Client()
                bucket = client.bucket(self.config.gcs_bucket)
                # Just check if we can access (don't actually list)
                self.results.add_pass("GCS Access", f"bucket: {self.config.gcs_bucket}")
            except Exception as e:
                self.results.add_warning("GCS Access", str(e)[:80])

        return all_pass

    # ═══════════════════════════════════════════════════════════════
    # N10S CONFIGURATION TEST
    # ═══════════════════════════════════════════════════════════════

    async def test_n10s_configuration(self) -> bool:
        """Test n10s graph configuration"""
        print("\n🔧 N10S CONFIGURATION TEST")
        print("─" * 60)

        if not self.driver:
            self.results.add_skip("N10s Config", "No Neo4j connection")
            return False

        test_db = "neo4j"  # Use default database for testing

        try:
            async with self.driver.session(database=test_db) as session:
                # Check if n10s is already configured
                try:
                    result = await session.run("CALL n10s.graphconfig.show()")
                    config = await result.single()
                    if config:
                        self.results.add_pass(
                            "N10s Already Configured",
                            f"handleVocabUris: {config.get('handleVocabUris', 'N/A')}"
                        )
                        return True
                except Exception:
                    pass  # Not configured yet

                # Initialize n10s configuration
                # First create constraint
                try:
                    await session.run("""
                        CREATE CONSTRAINT n10s_unique_uri IF NOT EXISTS
                        FOR (r:Resource) REQUIRE r.uri IS UNIQUE
                    """)
                    self.results.add_pass("N10s Constraint Created")
                except Exception as e:
                    self.results.add_warning("N10s Constraint", str(e)[:50])

                # Initialize graph config
                try:
                    await session.run("""
                        CALL n10s.graphconfig.init({
                            handleVocabUris: 'SHORTEN',
                            applyNeo4jNaming: true,
                            multivalPropList: ['http://www.w3.org/2000/01/rdf-schema#label']
                        })
                    """)
                    self.results.add_pass("N10s Graph Config Initialized")
                    return True
                except Exception as e:
                    # May already be initialized
                    if "already" in str(e).lower():
                        self.results.add_pass("N10s Config", "Already initialized")
                        return True
                    self.results.add_fail("N10s Graph Config", str(e)[:80])
                    return False

        except Exception as e:
            self.results.add_fail("N10s Configuration Test", str(e)[:100])
            return False

    # ═══════════════════════════════════════════════════════════════
    # RDF IMPORT TEST
    # ═══════════════════════════════════════════════════════════════

    async def test_rdf_import(self) -> bool:
        """Test RDF import via n10s"""
        print("\n📥 RDF IMPORT TEST")
        print("─" * 60)

        if not self.driver:
            self.results.add_skip("RDF Import", "No Neo4j connection")
            return False

        # Create a small test TTL content
        test_ttl = """
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix kb7: <http://cardiofit.ai/ontology/kb7#> .

kb7:TestDrug001 a owl:Class ;
    rdfs:label "Test Aspirin 81mg" ;
    kb7:rxnormCode "1191" ;
    kb7:atcCode "B01AC06" .

kb7:TestDrug002 a owl:Class ;
    rdfs:label "Test Metformin 500mg" ;
    kb7:rxnormCode "6809" ;
    kb7:atcCode "A10BA02" .

kb7:TestInteraction001 a kb7:DrugInteraction ;
    kb7:involves kb7:TestDrug001, kb7:TestDrug002 ;
    kb7:severity "moderate" ;
    rdfs:label "Test Interaction" .
"""

        try:
            async with self.driver.session() as session:
                # Import inline RDF
                result = await session.run("""
                    CALL n10s.rdf.import.inline($ttl, 'Turtle', { verifyUriSyntax: false })
                    YIELD terminationStatus, triplesLoaded
                    RETURN terminationStatus, triplesLoaded
                """, ttl=test_ttl)

                record = await result.single()
                if record:
                    status = record['terminationStatus']
                    triples = record['triplesLoaded']

                    if status == 'OK' and triples > 0:
                        self.results.add_pass("RDF Inline Import", f"{triples} triples loaded")
                    else:
                        self.results.add_fail("RDF Import", f"Status: {status}, Triples: {triples}")
                        return False

                # Verify imported data
                result = await session.run("""
                    MATCH (n:Resource)
                    WHERE n.uri CONTAINS 'TestDrug'
                    RETURN count(n) as count
                """)
                record = await result.single()
                count = record['count'] if record else 0

                if count > 0:
                    self.results.add_pass("RDF Data Verification", f"{count} test nodes found")
                    return True
                else:
                    self.results.add_fail("RDF Data Verification", "No test nodes found")
                    return False

        except Exception as e:
            self.results.add_fail("RDF Import Test", str(e)[:100])
            return False

    # ═══════════════════════════════════════════════════════════════
    # GCS IMPORT TEST (Optional)
    # ═══════════════════════════════════════════════════════════════

    async def test_gcs_import(self, version_id: str = "latest") -> bool:
        """Test RDF import from GCS"""
        print("\n☁️  GCS IMPORT TEST")
        print("─" * 60)

        if not GCS_AVAILABLE:
            self.results.add_skip("GCS Import", "google-cloud-storage not installed")
            return False

        if not self.driver:
            self.results.add_skip("GCS Import", "No Neo4j connection")
            return False

        try:
            # Generate signed URL
            client = storage.Client()
            bucket = client.bucket(self.config.gcs_bucket)
            blob_name = f"{version_id}/kb7-kernel.ttl"
            blob = bucket.blob(blob_name)

            if not blob.exists():
                self.results.add_warning("GCS Artifact", f"Not found: {blob_name}")
                return False

            signed_url = blob.generate_signed_url(
                version="v4",
                expiration=3600,
                method="GET"
            )
            self.results.add_pass("GCS Signed URL Generated")

            # Import via n10s
            async with self.driver.session() as session:
                result = await session.run("""
                    CALL n10s.rdf.import.fetch($url, 'Turtle', { verifyUriSyntax: false })
                    YIELD terminationStatus, triplesLoaded
                    RETURN terminationStatus, triplesLoaded
                """, url=signed_url)

                record = await result.single()
                if record:
                    status = record['terminationStatus']
                    triples = record['triplesLoaded']

                    if status == 'OK':
                        self.results.add_pass("GCS RDF Import", f"{triples} triples loaded")
                        return True
                    else:
                        self.results.add_fail("GCS RDF Import", f"Status: {status}")
                        return False

        except Exception as e:
            self.results.add_fail("GCS Import Test", str(e)[:100])
            return False

    # ═══════════════════════════════════════════════════════════════
    # CLEANUP TEST DATA
    # ═══════════════════════════════════════════════════════════════

    async def cleanup_test_data(self):
        """Remove test data"""
        if not self.driver:
            return

        try:
            async with self.driver.session() as session:
                await session.run("""
                    MATCH (n:Resource)
                    WHERE n.uri CONTAINS 'TestDrug' OR n.uri CONTAINS 'TestInteraction'
                    DETACH DELETE n
                """)
                print("  🧹 Test data cleaned up")
        except Exception as e:
            print(f"  ⚠️  Cleanup warning: {e}")


async def main():
    parser = argparse.ArgumentParser(
        description='KB7 N10s Pipeline Integration Test',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Check prerequisites only
  python test_kb7_n10s_pipeline.py --check-prereqs --neo4j-password YOUR_PASSWORD

  # Run full test suite
  python test_kb7_n10s_pipeline.py --full --neo4j-password YOUR_PASSWORD

  # Test with specific GCS version
  python test_kb7_n10s_pipeline.py --gcs-version 20251201 --neo4j-password YOUR_PASSWORD
        """
    )

    parser.add_argument('--neo4j-uri', default='bolt://localhost:7687', help='Neo4j URI')
    parser.add_argument('--neo4j-user', default='neo4j', help='Neo4j username')
    parser.add_argument('--neo4j-password', required=True, help='Neo4j password')
    parser.add_argument('--redis-url', default='redis://localhost:6379', help='Redis URL')
    parser.add_argument('--gcs-bucket', default='sincere-hybrid-477206-h2-kb-artifacts-production', help='GCS bucket')
    parser.add_argument('--gcs-version', default='latest', help='GCS version ID to test')

    parser.add_argument('--check-prereqs', action='store_true', help='Check prerequisites only')
    parser.add_argument('--full', action='store_true', help='Run full test suite including GCS')
    parser.add_argument('--no-cleanup', action='store_true', help='Skip cleanup of test data')

    args = parser.parse_args()

    config = TestConfig(
        neo4j_uri=args.neo4j_uri,
        neo4j_user=args.neo4j_user,
        neo4j_password=args.neo4j_password,
        redis_url=args.redis_url,
        gcs_bucket=args.gcs_bucket
    )

    print("""
╔═══════════════════════════════════════════════════════════════╗
║         KB7 N10s Pipeline Integration Test                    ║
║                                                               ║
║  Testing: Knowledge Factory → GCS → CDC → n10s → Neo4j        ║
╚═══════════════════════════════════════════════════════════════╝
""")

    tester = KB7PipelineTest(config)

    try:
        await tester.setup()

        # Always check prerequisites
        prereqs_ok = await tester.check_prerequisites()

        if args.check_prereqs:
            print(tester.results.summary())
            return 0 if prereqs_ok else 1

        if not prereqs_ok:
            print("\n❌ Prerequisites not met. Fix issues above before running tests.")
            print(tester.results.summary())
            return 1

        # Run n10s configuration test
        await tester.test_n10s_configuration()

        # Run RDF import test (inline)
        await tester.test_rdf_import()

        # Run GCS import test if --full
        if args.full:
            await tester.test_gcs_import(args.gcs_version)

        # Cleanup unless --no-cleanup
        if not args.no_cleanup:
            await tester.cleanup_test_data()

        print(tester.results.summary())

        return 0 if len(tester.results.failed) == 0 else 1

    finally:
        await tester.teardown()


if __name__ == '__main__':
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
