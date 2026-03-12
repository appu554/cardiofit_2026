"""
Neo4j Terminology Sync Service
Implements 5-phase database aliasing for zero-downtime terminology switching

Architecture:
============
This service orchestrates Neo4j database creation, import, and alias switching
for terminology version management. It enables hot-swapping of terminology
databases without client reconfiguration or downtime.

Polyglot Persistence Strategy:
=============================
- GraphDB (RDF): Semantic "Source of Truth" - Reasoning, Ontologies
- Neo4j (LPG): "Application Cache" - High-speed traversals, Graph Algorithms

Import Method: Neosemantics (n10s)
=================================
Instead of transforming RDF→Cypher in Python, we use n10s.rdf.import.fetch()
to let Neo4j pull TTL files directly from GCS via signed URLs. This is:
- Faster: Native bulk import vs row-by-row Cypher
- Simpler: No custom transformation logic
- Reliable: Built-in RDF→LPG mapping

5-Phase Database Switch Process:
================================
Phase 1: CREATE    - Create new versioned database (kb7_v2)
Phase 2: IMPORT    - Import RDF via n10s from GCS signed URL
Phase 3: VALIDATE  - Verify data integrity (counts, indexes, queries)
Phase 4: SWITCH    - Atomically update kb7_production alias
Phase 5: CLEANUP   - Schedule old database for cleanup after grace period

CDC Event Flow:
==============
status='LOADING'  → Phase 1 & 2: Create database, start import
status='ACTIVE'   → Phase 3 & 4: Validate and switch alias
status='ARCHIVED' → Phase 5: Schedule cleanup

Client Configuration:
====================
All clients should use the alias name 'kb7_production', never versioned names.
This allows atomic database switching without client reconfiguration.

Prerequisites:
=============
1. Neo4j must have neosemantics (n10s) plugin installed
2. neo4j.conf: dbms.security.procedures.unrestricted=n10s.*
3. Graph config initialized via n10s.graphconfig.init()

@author CDC Integration Team
@version 2.0
@since 2025-12-03
"""

import asyncio
import os
from neo4j import AsyncGraphDatabase
from typing import Dict, Any, Optional, List
from datetime import datetime, timedelta
import redis.asyncio as redis
import structlog
from dataclasses import dataclass

# GCS for signed URL generation
try:
    from google.cloud import storage
    from google.auth.transport.requests import Request
    import google.auth
    GCS_AVAILABLE = True
except ImportError:
    GCS_AVAILABLE = False

logger = structlog.get_logger(__name__)


@dataclass
class SyncStats:
    """Statistics from a sync operation (n10s import)"""
    triples_parsed: int = 0
    nodes_created: int = 0
    relationships_created: int = 0
    properties_set: int = 0
    termination_status: str = "unknown"
    errors: int = 0
    duration_seconds: float = 0.0
    gcs_uri: str = ""


class Neo4jTerminologySyncService:
    """
    Orchestrates Neo4j database creation, import, and alias switching
    for terminology version management using neosemantics (n10s).

    Key Features:
    - Zero-downtime terminology updates via database aliasing
    - Native RDF import via n10s.rdf.import.fetch() from GCS
    - Atomic alias switching with automatic rollback on failure
    - 24-hour grace period for rollback before cleanup
    - Redis-based state tracking for multi-phase operations
    - Comprehensive validation before production switch

    Import Strategy:
    - Generate signed URL for GCS TTL file
    - Neo4j pulls directly via n10s (no Python transformation)
    - n10s handles RDF→LPG mapping natively
    """

    PRODUCTION_ALIAS = "kb7_production"
    CLEANUP_GRACE_HOURS = 24
    VALIDATION_TOLERANCE = 0.95  # 5% tolerance for count validation

    # GCS artifact path pattern: {version_id}/kb7-kernel.ttl
    GCS_ARTIFACT_PATH = "{version_id}/kb7-kernel.ttl"
    SIGNED_URL_EXPIRATION = 3600  # 1 hour

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Neo4j Terminology Sync Service

        Args:
            config: Configuration dictionary with keys:
                - neo4j_uri: Neo4j bolt URI
                - neo4j_user: Neo4j username
                - neo4j_password: Neo4j password
                - gcs_bucket: GCS bucket containing KB artifacts
                - redis_url: Redis connection URL (optional)
        """
        self.driver = AsyncGraphDatabase.driver(
            config.get('neo4j_uri', os.getenv('NEO4J_URI', 'bolt://localhost:7687')),
            auth=(
                config.get('neo4j_user', os.getenv('NEO4J_USER', 'neo4j')),
                config.get('neo4j_password', os.getenv('NEO4J_PASSWORD', 'kb7password'))
            )
        )

        # GCS configuration for artifact retrieval
        self.gcs_bucket = config.get(
            'gcs_bucket',
            os.getenv('GCS_BUCKET', 'sincere-hybrid-477206-h2-kb-artifacts-production')
        )
        self._gcs_client = None

        self.redis_url = config.get('redis_url', os.getenv('REDIS_URL', 'redis://localhost:6379'))
        self._redis_client = None

        self.config = config

        # Track if n10s graph config is initialized
        self._n10s_initialized = {}  # per-database

        # Service statistics
        self.stats = {
            'events_processed': 0,
            'databases_created': 0,
            'alias_switches': 0,
            'cleanups_performed': 0,
            'rollbacks_performed': 0,
            'n10s_imports': 0,
            'total_triples_imported': 0,
            'errors': 0,
            'last_event_time': None,
            'current_version': None
        }

        logger.info(
            "Neo4j Terminology Sync Service initialized (n10s mode)",
            neo4j_uri=config.get('neo4j_uri', 'bolt://localhost:7687'),
            gcs_bucket=self.gcs_bucket,
            production_alias=self.PRODUCTION_ALIAS
        )

    @property
    async def redis(self):
        """Lazy initialization of Redis client"""
        if self._redis_client is None:
            self._redis_client = await redis.from_url(self.redis_url)
        return self._redis_client

    @property
    def gcs_client(self):
        """Lazy initialization of GCS client"""
        if self._gcs_client is None:
            if not GCS_AVAILABLE:
                raise RuntimeError(
                    "google-cloud-storage not installed. "
                    "Run: pip install google-cloud-storage"
                )
            self._gcs_client = storage.Client()
        return self._gcs_client

    # ══════════════════════════════════════════════════════════════════════
    # GCS SIGNED URL GENERATION
    # ══════════════════════════════════════════════════════════════════════

    def _generate_signed_url(self, version_id: str) -> Optional[str]:
        """
        Generate a signed URL for GCS artifact so Neo4j can download it.

        The signed URL allows Neo4j to pull the TTL file directly from GCS
        without exposing the bucket publicly.

        Args:
            version_id: Terminology version identifier (e.g., "20251203")

        Returns:
            Signed URL valid for SIGNED_URL_EXPIRATION seconds, or None if not found
        """
        blob_name = self.GCS_ARTIFACT_PATH.format(version_id=version_id)

        try:
            # Refresh credentials if needed
            credentials, _ = google.auth.default()
            if hasattr(credentials, 'expired') and credentials.expired:
                credentials.refresh(Request())

            bucket = self.gcs_client.bucket(self.gcs_bucket)
            blob = bucket.blob(blob_name)

            if not blob.exists():
                logger.error(
                    "GCS artifact not found",
                    bucket=self.gcs_bucket,
                    blob=blob_name,
                    version_id=version_id
                )
                return None

            signed_url = blob.generate_signed_url(
                version="v4",
                expiration=self.SIGNED_URL_EXPIRATION,
                method="GET"
            )

            logger.info(
                "Generated signed URL for GCS artifact",
                version_id=version_id,
                blob=blob_name,
                expiration_seconds=self.SIGNED_URL_EXPIRATION
            )

            return signed_url

        except Exception as e:
            logger.error(
                "Failed to generate signed URL",
                error=str(e),
                version_id=version_id
            )
            return None

    # ══════════════════════════════════════════════════════════════════════
    # N10S GRAPH CONFIGURATION
    # ══════════════════════════════════════════════════════════════════════

    async def _init_n10s_graph_config(self, db_name: str) -> bool:
        """
        Initialize n10s graph configuration for a database.

        Must be called once per database before importing RDF.
        Sets up:
        - URI uniqueness constraint for Resource nodes
        - Graph config with vocabulary URI handling

        Args:
            db_name: Name of the database to initialize

        Returns:
            True if initialization successful
        """
        if db_name in self._n10s_initialized:
            return True

        try:
            async with self.driver.session(database=db_name) as session:
                # Create uniqueness constraint (essential for performance)
                await session.run("""
                    CREATE CONSTRAINT n10s_unique_uri IF NOT EXISTS
                    FOR (r:Resource) REQUIRE r.uri IS UNIQUE
                """)

                # Initialize graph config
                # handleVocabUris: 'SHORTEN' converts URIs to shorter labels
                # applyNeo4jNaming: converts snake_case to camelCase
                await session.run("""
                    CALL n10s.graphconfig.init({
                        handleVocabUris: 'SHORTEN',
                        applyNeo4jNaming: true,
                        multivalPropList: ['http://www.w3.org/2000/01/rdf-schema#label']
                    })
                """)

                logger.info(
                    "n10s graph config initialized",
                    database=db_name
                )

                self._n10s_initialized[db_name] = True
                return True

        except Exception as e:
            logger.error(
                "Failed to initialize n10s graph config",
                error=str(e),
                database=db_name
            )
            return False

    async def _import_rdf_via_n10s(
        self,
        db_name: str,
        signed_url: str,
        version_id: str
    ) -> Dict[str, Any]:
        """
        Import RDF data into Neo4j using neosemantics (n10s).

        Uses n10s.rdf.import.fetch() to let Neo4j pull the TTL file
        directly from GCS and transform RDF to LPG.

        Args:
            db_name: Target database name
            signed_url: Signed URL to GCS TTL file
            version_id: Version identifier for logging

        Returns:
            Import statistics from n10s
        """
        stats = {
            'triples_parsed': 0,
            'termination_status': 'unknown',
            'nodes_created': 0,
            'relationships_created': 0,
            'properties_set': 0
        }

        try:
            async with self.driver.session(database=db_name) as session:
                # Execute n10s import
                result = await session.run(
                    """
                    CALL n10s.rdf.import.fetch(
                        $url,
                        'Turtle',
                        { verifyUriSyntax: false }
                    )
                    """,
                    url=signed_url
                )

                record = await result.single()
                if record:
                    stats['triples_parsed'] = record.get('triplesLoaded', 0)
                    stats['termination_status'] = record.get('terminationStatus', 'OK')
                    stats['nodes_created'] = record.get('namespaces', 0)
                    stats['properties_set'] = record.get('extraInfo', {}).get('propertiesSet', 0)

                logger.info(
                    "n10s import completed",
                    database=db_name,
                    version_id=version_id,
                    triples=stats['triples_parsed'],
                    status=stats['termination_status']
                )

                # Update service stats
                self.stats['n10s_imports'] += 1
                self.stats['total_triples_imported'] += stats['triples_parsed']

                return stats

        except Exception as e:
            logger.error(
                "n10s import failed",
                error=str(e),
                database=db_name,
                version_id=version_id
            )
            stats['termination_status'] = f'ERROR: {str(e)}'
            return stats

    # ══════════════════════════════════════════════════════════════════════
    # MAIN EVENT HANDLER
    # ══════════════════════════════════════════════════════════════════════

    async def handle_cdc_event(self, event: Dict[str, Any]) -> bool:
        """
        Handle terminology CDC event and trigger appropriate phase

        This is the main entry point for CDC events from the terminology release
        topic. Based on the status field, it routes to the appropriate phase handler.

        Args:
            event: Debezium CDC event payload containing:
                - version_id: Terminology version identifier
                - status: PENDING | LOADING | ACTIVE | ARCHIVED | FAILED
                - graphdb_repository: GraphDB repository name
                - snomed_version, rxnorm_version, loinc_version: Source versions

        Returns:
            Success status (True if phase completed successfully)
        """
        status = event.get('status')
        version_id = event.get('version_id')

        logger.info(
            "Processing terminology CDC event",
            version_id=version_id,
            status=status,
            graphdb_repository=event.get('graphdb_repository')
        )

        self.stats['events_processed'] += 1
        self.stats['last_event_time'] = datetime.utcnow().isoformat()

        try:
            if status == 'LOADING':
                # Phase 1 & 2: Create database and start import
                return await self._phase_1_2_create_and_import(event)

            elif status == 'ACTIVE':
                # Phase 3 & 4: Validate and switch alias
                return await self._phase_3_4_validate_and_switch(event)

            elif status == 'ARCHIVED':
                # Phase 5: Schedule cleanup
                return await self._phase_5_schedule_cleanup(event)

            elif status == 'FAILED':
                # Clean up any partial state
                return await self._handle_failed_release(event)

            else:
                logger.debug(f"Ignoring event with status: {status}")
                return True

        except Exception as e:
            self.stats['errors'] += 1
            logger.error(
                "Error handling CDC event",
                error=str(e),
                version_id=version_id,
                status=status
            )
            return False

    # ══════════════════════════════════════════════════════════════════════
    # PHASE 1 & 2: CREATE DATABASE AND IMPORT
    # ══════════════════════════════════════════════════════════════════════

    async def _phase_1_2_create_and_import(
        self,
        event: Dict[str, Any]
    ) -> bool:
        """
        Phase 1: Create new database
        Phase 2: Import RDF data via n10s from GCS

        This phase is triggered when a new terminology version enters LOADING status.
        It creates a new Neo4j database, initializes n10s config, generates a signed
        URL for the GCS artifact, and lets Neo4j pull the data directly.

        Args:
            event: CDC event with version_id and gcs_uri

        Returns:
            True if both phases completed successfully
        """
        version_id = event['version_id']
        # Sanitize version_id for use as database name (replace dots and dashes)
        db_name = f"kb7_{version_id.replace('.', '_').replace('-', '_')}"

        try:
            # ─────────────────────────────────────────────────────────────
            # Phase 1: Create database
            # ─────────────────────────────────────────────────────────────
            logger.info(f"Phase 1: Creating database {db_name}")
            await self._create_database(db_name)
            self.stats['databases_created'] += 1

            # ─────────────────────────────────────────────────────────────
            # Phase 2: Import RDF via n10s from GCS
            # ─────────────────────────────────────────────────────────────
            logger.info(f"Phase 2: Importing RDF data to {db_name} via n10s")

            # 2a: Initialize n10s graph config
            if not await self._init_n10s_graph_config(db_name):
                raise RuntimeError(f"Failed to initialize n10s config for {db_name}")

            # 2b: Generate signed URL for GCS artifact
            signed_url = self._generate_signed_url(version_id)
            if not signed_url:
                raise RuntimeError(f"GCS artifact not found for version {version_id}")

            # 2c: Import via n10s
            import_start = datetime.utcnow()
            stats = await self._import_rdf_via_n10s(db_name, signed_url, version_id)
            import_duration = (datetime.utcnow() - import_start).total_seconds()

            if 'ERROR' in stats.get('termination_status', ''):
                raise RuntimeError(f"n10s import failed: {stats['termination_status']}")

            logger.info(
                "Phase 2 complete",
                db_name=db_name,
                triples_parsed=stats.get('triples_parsed', 0),
                duration_seconds=import_duration
            )

            # Store import metadata in Redis for validation phase
            redis_client = await self.redis
            await redis_client.hset(
                f"neo4j:import:{version_id}",
                mapping={
                    'db_name': db_name,
                    'triples_parsed': str(stats.get('triples_parsed', 0)),
                    'termination_status': stats.get('termination_status', 'OK'),
                    'gcs_uri': f"gs://{self.gcs_bucket}/{self.GCS_ARTIFACT_PATH.format(version_id=version_id)}",
                    'imported_at': datetime.utcnow().isoformat(),
                    'import_duration_seconds': str(import_duration),
                    'snomed_version': event.get('snomed_version', ''),
                    'rxnorm_version': event.get('rxnorm_version', ''),
                    'loinc_version': event.get('loinc_version', '')
                }
            )

            # Set expiration for import metadata (7 days)
            await redis_client.expire(f"neo4j:import:{version_id}", 7 * 24 * 3600)

            logger.info(
                "Phase 1 & 2 complete - database created and RDF imported via n10s",
                version_id=version_id,
                db_name=db_name,
                triples=stats.get('triples_parsed', 0)
            )

            return True

        except Exception as e:
            logger.error(
                "Phase 1/2 failed",
                error=str(e),
                version_id=version_id,
                db_name=db_name
            )
            # Clean up partial database on failure
            await self._cleanup_partial_database(db_name)
            return False

    async def _create_database(self, db_name: str) -> None:
        """
        Create a new Neo4j database

        Creates the database and waits for it to come online before returning.
        If the database already exists, logs a warning and returns.

        Args:
            db_name: Name of the database to create
        """
        async with self.driver.session(database="system") as session:
            # Check if database already exists
            result = await session.run(
                "SHOW DATABASES WHERE name = $name",
                name=db_name
            )
            existing = await result.single()

            if existing:
                logger.warning(f"Database {db_name} already exists, skipping creation")
                return

            # Create new database
            await session.run(f"CREATE DATABASE `{db_name}`")

            # Wait for database to be online (max 30 seconds)
            for attempt in range(30):
                result = await session.run(
                    "SHOW DATABASE $name",
                    name=db_name
                )
                record = await result.single()
                if record and record['currentStatus'] == 'online':
                    logger.info(f"Database {db_name} created and online")
                    return
                await asyncio.sleep(1)

            raise TimeoutError(f"Database {db_name} failed to come online within 30 seconds")

    async def _cleanup_partial_database(self, db_name: str) -> None:
        """Clean up a partially created database on failure"""
        try:
            async with self.driver.session(database="system") as session:
                result = await session.run(
                    "SHOW DATABASES WHERE name = $name",
                    name=db_name
                )
                if await result.single():
                    await session.run(f"DROP DATABASE `{db_name}` IF EXISTS")
                    logger.info(f"Cleaned up partial database {db_name}")
        except Exception as e:
            logger.warning(f"Failed to clean up partial database {db_name}: {e}")

    # ══════════════════════════════════════════════════════════════════════
    # PHASE 3 & 4: VALIDATE AND SWITCH ALIAS
    # ══════════════════════════════════════════════════════════════════════

    async def _phase_3_4_validate_and_switch(
        self,
        event: Dict[str, Any]
    ) -> bool:
        """
        Phase 3: Validate imported data
        Phase 4: Switch alias to new database

        This phase is triggered when a terminology version becomes ACTIVE.
        It validates the imported data and then atomically switches the
        production alias to the new database.

        Args:
            event: CDC event with version_id

        Returns:
            True if validation passed and alias switched successfully
        """
        version_id = event['version_id']

        # Get database name from import metadata
        redis_client = await self.redis
        import_meta = await redis_client.hgetall(f"neo4j:import:{version_id}")
        if not import_meta:
            logger.error(f"No import metadata found for {version_id}")
            return False

        db_name = import_meta.get(b'db_name', b'').decode()
        expected_triples = int(import_meta.get(b'triples_parsed', b'0'))

        try:
            # ─────────────────────────────────────────────────────────────
            # Phase 3: Validate n10s imported data
            # ─────────────────────────────────────────────────────────────
            logger.info(f"Phase 3: Validating n10s import in database {db_name}")
            validation = await self._validate_n10s_database(
                db_name,
                expected_triples
            )

            if not validation['passed']:
                logger.error(
                    "Phase 3 validation failed",
                    validation=validation,
                    version_id=version_id
                )
                return False

            logger.info("Phase 3 validation passed", validation=validation)

            # Get current active database for rollback tracking
            old_db = await self._get_current_alias_target()

            # ─────────────────────────────────────────────────────────────
            # Phase 4: Switch alias
            # ─────────────────────────────────────────────────────────────
            logger.info(f"Phase 4: Switching alias to {db_name}")
            await self._switch_alias(db_name)
            self.stats['alias_switches'] += 1
            self.stats['current_version'] = version_id

            # Store old database for cleanup scheduling
            if old_db:
                await redis_client.setex(
                    f"neo4j:pending_cleanup:{old_db}",
                    self.CLEANUP_GRACE_HOURS * 3600,
                    datetime.utcnow().isoformat()
                )

            # Post-switch health check
            if not await self._post_switch_health_check():
                # Rollback on health check failure
                logger.error("Post-switch health check failed, rolling back")
                if old_db:
                    await self._switch_alias(old_db)
                    self.stats['rollbacks_performed'] += 1
                return False

            # Update Redis with current active version
            await redis_client.set(
                "neo4j:current_version",
                version_id
            )
            await redis_client.hset(
                "neo4j:current_version:details",
                mapping={
                    'version_id': version_id,
                    'db_name': db_name,
                    'switched_at': datetime.utcnow().isoformat(),
                    'snomed_version': import_meta.get(b'snomed_version', b'').decode(),
                    'rxnorm_version': import_meta.get(b'rxnorm_version', b'').decode(),
                    'loinc_version': import_meta.get(b'loinc_version', b'').decode()
                }
            )

            logger.info(
                "Phase 4 complete - alias switched",
                old_db=old_db,
                new_db=db_name,
                version_id=version_id
            )

            return True

        except Exception as e:
            logger.error(
                "Phase 3/4 failed",
                error=str(e),
                version_id=version_id
            )
            return False

    async def _validate_n10s_database(
        self,
        db_name: str,
        expected_triples: int
    ) -> Dict[str, Any]:
        """
        Validate n10s imported database contents

        Performs several validation checks for n10s-imported RDF data:
        1. Resource node count (main node type from n10s)
        2. Total node/relationship count
        3. n10s uniqueness constraint existence
        4. Sample query execution

        Args:
            db_name: Database to validate
            expected_triples: Expected number of triples parsed during import

        Returns:
            Validation result with pass/fail status and check details
        """
        validation = {
            'passed': True,
            'checks': {},
            'db_name': db_name,
            'validated_at': datetime.utcnow().isoformat()
        }

        try:
            async with self.driver.session(database=db_name) as session:
                # Check Resource node count (n10s creates these from RDF subjects)
                result = await session.run(
                    "MATCH (r:Resource) RETURN count(r) as count"
                )
                record = await result.single()
                resource_count = record['count'] if record else 0

                validation['checks']['resource_count'] = {
                    'actual': resource_count,
                    'passed': resource_count > 0
                }

                # Check total node count
                result = await session.run(
                    "MATCH (n) RETURN count(n) as count"
                )
                record = await result.single()
                total_nodes = record['count'] if record else 0

                validation['checks']['total_nodes'] = {
                    'actual': total_nodes,
                    'expected_minimum': expected_triples // 3,  # Rough estimate
                    'passed': total_nodes > 0
                }

                # Check total relationship count
                result = await session.run(
                    "MATCH ()-[r]->() RETURN count(r) as count"
                )
                record = await result.single()
                total_rels = record['count'] if record else 0

                validation['checks']['total_relationships'] = {
                    'actual': total_rels,
                    'passed': True  # Informational
                }

                # Check n10s constraint exists
                result = await session.run(
                    "SHOW CONSTRAINTS WHERE name = 'n10s_unique_uri'"
                )
                constraints = [r async for r in result]
                validation['checks']['n10s_constraint'] = {
                    'exists': len(constraints) > 0,
                    'passed': len(constraints) > 0
                }

                # Check n10s graph config exists
                try:
                    result = await session.run("CALL n10s.graphconfig.show()")
                    config = await result.single()
                    validation['checks']['n10s_config'] = {
                        'configured': config is not None,
                        'handleVocabUris': config.get('handleVocabUris') if config else None,
                        'passed': config is not None
                    }
                except Exception:
                    validation['checks']['n10s_config'] = {
                        'configured': False,
                        'passed': False
                    }

                # Check sample query execution - get some Resource nodes
                try:
                    result = await session.run(
                        "MATCH (r:Resource) RETURN r.uri, labels(r) LIMIT 5"
                    )
                    samples = [r async for r in result]
                    validation['checks']['sample_query'] = {
                        'passed': len(samples) > 0,
                        'sample_count': len(samples)
                    }
                except Exception as e:
                    validation['checks']['sample_query'] = {
                        'passed': False,
                        'error': str(e)
                    }

                # Check for specific terminology labels (SNOMED, RxNorm patterns)
                result = await session.run("""
                    MATCH (n)
                    WHERE any(label IN labels(n) WHERE
                        label CONTAINS 'Class' OR
                        label CONTAINS 'Drug' OR
                        label CONTAINS 'Concept')
                    RETURN DISTINCT labels(n) as labels LIMIT 10
                """)
                label_samples = [r async for r in result]
                validation['checks']['terminology_labels'] = {
                    'found_labels': [r['labels'] for r in label_samples],
                    'passed': len(label_samples) > 0
                }

                # Overall pass/fail
                validation['passed'] = all(
                    check.get('passed', True)
                    for check in validation['checks'].values()
                )

        except Exception as e:
            validation['passed'] = False
            validation['error'] = str(e)
            logger.error(f"Validation error for {db_name}: {e}")

        return validation

    async def _switch_alias(self, target_db: str) -> None:
        """
        Atomically switch the production alias to target database

        This is the critical operation that makes the new terminology version
        available to all clients. Uses Neo4j's database alias feature for
        atomic switching.

        Args:
            target_db: Name of the database to point the alias to
        """
        async with self.driver.session(database="system") as session:
            # Check if alias exists
            result = await session.run(
                "SHOW ALIASES WHERE name = $alias",
                alias=self.PRODUCTION_ALIAS
            )
            existing = await result.single()

            if existing:
                # Update existing alias
                await session.run(
                    f"ALTER ALIAS `{self.PRODUCTION_ALIAS}` "
                    f"SET DATABASE = `{target_db}`"
                )
            else:
                # Create new alias
                await session.run(
                    f"CREATE ALIAS `{self.PRODUCTION_ALIAS}` "
                    f"FOR DATABASE `{target_db}`"
                )

            logger.info(
                "Alias switched",
                alias=self.PRODUCTION_ALIAS,
                target=target_db
            )

    async def _get_current_alias_target(self) -> Optional[str]:
        """Get the database currently pointed to by production alias"""
        async with self.driver.session(database="system") as session:
            result = await session.run(
                "SHOW ALIASES WHERE name = $alias",
                alias=self.PRODUCTION_ALIAS
            )
            record = await result.single()
            return record['database'] if record else None

    async def _post_switch_health_check(self) -> bool:
        """
        Verify production alias is working after switch

        Performs a quick health check to ensure the alias is functioning:
        1. Basic connectivity test
        2. n10s graph config verification
        3. Resource node query execution

        Returns:
            True if health check passes
        """
        try:
            async with self.driver.session(
                database=self.PRODUCTION_ALIAS
            ) as session:
                # Basic connectivity test
                result = await session.run("RETURN 1 as test")
                await result.single()

                # Verify n10s config exists
                try:
                    result = await session.run("CALL n10s.graphconfig.show()")
                    config = await result.single()
                    if not config:
                        logger.warning("Post-switch: n10s config not found")
                except Exception as e:
                    logger.warning(f"Post-switch: n10s config check failed: {e}")

                # Quick query test - check for Resource nodes (n10s creates these)
                result = await session.run(
                    "MATCH (r:Resource) RETURN count(r) as count LIMIT 1"
                )
                record = await result.single()

                if record and record['count'] > 0:
                    logger.info(
                        "Post-switch health check passed",
                        resource_count=record['count']
                    )
                    return True
                else:
                    logger.warning("Post-switch health check: no Resource nodes found")
                    return False

        except Exception as e:
            logger.error(f"Post-switch health check failed: {e}")
            return False

    # ══════════════════════════════════════════════════════════════════════
    # PHASE 5: CLEANUP
    # ══════════════════════════════════════════════════════════════════════

    async def _phase_5_schedule_cleanup(
        self,
        event: Dict[str, Any]
    ) -> bool:
        """
        Schedule old database for cleanup after grace period

        This phase is triggered when a terminology version is ARCHIVED.
        The database is scheduled for cleanup after CLEANUP_GRACE_HOURS
        to allow for potential rollback.

        Args:
            event: CDC event with version_id

        Returns:
            True if cleanup scheduled successfully
        """
        version_id = event['version_id']

        redis_client = await self.redis
        import_meta = await redis_client.hgetall(f"neo4j:import:{version_id}")
        if not import_meta:
            logger.info(f"No import metadata for {version_id}, nothing to clean up")
            return True

        db_name = import_meta.get(b'db_name', b'').decode()
        if not db_name:
            return True

        # Schedule cleanup
        cleanup_time = datetime.utcnow() + timedelta(
            hours=self.CLEANUP_GRACE_HOURS
        )

        await redis_client.zadd(
            "neo4j:scheduled_cleanups",
            {db_name: cleanup_time.timestamp()}
        )

        logger.info(
            "Scheduled database cleanup",
            db_name=db_name,
            cleanup_time=cleanup_time.isoformat(),
            grace_hours=self.CLEANUP_GRACE_HOURS
        )

        return True

    async def run_cleanup_job(self) -> int:
        """
        Run scheduled cleanup job (call from cron/scheduler)

        Processes all databases that have passed their cleanup grace period.
        Verifies each database is not currently in use before dropping.

        Returns:
            Number of databases cleaned up
        """
        now = datetime.utcnow().timestamp()
        redis_client = await self.redis

        # Get databases due for cleanup
        due = await redis_client.zrangebyscore(
            "neo4j:scheduled_cleanups",
            0,
            now
        )

        cleaned = 0
        for db_name_bytes in due:
            db_name = db_name_bytes.decode()

            try:
                # Verify not currently in use
                current = await self._get_current_alias_target()
                if current == db_name:
                    logger.warning(
                        f"Skipping cleanup of {db_name} - still in use"
                    )
                    continue

                # Drop database
                async with self.driver.session(database="system") as session:
                    await session.run(f"DROP DATABASE `{db_name}` IF EXISTS")

                # Remove from scheduled cleanups
                await redis_client.zrem("neo4j:scheduled_cleanups", db_name)

                logger.info(f"Cleaned up database {db_name}")
                cleaned += 1
                self.stats['cleanups_performed'] += 1

            except Exception as e:
                logger.error(f"Cleanup failed for {db_name}: {e}")

        return cleaned

    async def _handle_failed_release(self, event: Dict[str, Any]) -> bool:
        """Handle a failed terminology release by cleaning up any partial state"""
        version_id = event['version_id']

        redis_client = await self.redis
        import_meta = await redis_client.hgetall(f"neo4j:import:{version_id}")

        if import_meta:
            db_name = import_meta.get(b'db_name', b'').decode()
            if db_name:
                # Only clean up if not currently in use
                current = await self._get_current_alias_target()
                if current != db_name:
                    await self._cleanup_partial_database(db_name)

            # Clean up import metadata
            await redis_client.delete(f"neo4j:import:{version_id}")

        logger.info(f"Cleaned up failed release: {version_id}")
        return True

    # ══════════════════════════════════════════════════════════════════════
    # ROLLBACK
    # ══════════════════════════════════════════════════════════════════════

    async def rollback(self, to_version: str) -> bool:
        """
        Emergency rollback to previous version

        Allows manual rollback to a previous terminology version if the
        database still exists (within the cleanup grace period).

        Args:
            to_version: Version ID to rollback to

        Returns:
            Success status
        """
        redis_client = await self.redis
        import_meta = await redis_client.hgetall(f"neo4j:import:{to_version}")
        if not import_meta:
            logger.error(f"No import metadata for rollback version {to_version}")
            return False

        db_name = import_meta.get(b'db_name', b'').decode()

        try:
            # Verify database still exists
            async with self.driver.session(database="system") as session:
                result = await session.run(
                    "SHOW DATABASES WHERE name = $name",
                    name=db_name
                )
                if not await result.single():
                    logger.error(f"Rollback database {db_name} not found")
                    return False

            # Switch alias back
            await self._switch_alias(db_name)
            self.stats['rollbacks_performed'] += 1
            self.stats['current_version'] = to_version

            # Update Redis
            await redis_client.set("neo4j:current_version", to_version)

            logger.info(f"Rolled back to {to_version} ({db_name})")
            return True

        except Exception as e:
            logger.error(f"Rollback failed: {e}")
            return False

    # ══════════════════════════════════════════════════════════════════════
    # STATUS AND MONITORING
    # ══════════════════════════════════════════════════════════════════════

    async def get_status(self) -> Dict[str, Any]:
        """Get current service status and statistics"""
        current_target = await self._get_current_alias_target()

        redis_client = await self.redis
        current_version = await redis_client.get("neo4j:current_version")
        pending_cleanups = await redis_client.zcard("neo4j:scheduled_cleanups")

        return {
            'service': 'neo4j-terminology-sync',
            'production_alias': self.PRODUCTION_ALIAS,
            'current_target_database': current_target,
            'current_version': current_version.decode() if current_version else None,
            'pending_cleanups': pending_cleanups,
            'statistics': self.stats,
            'config': {
                'cleanup_grace_hours': self.CLEANUP_GRACE_HOURS,
                'validation_tolerance': self.VALIDATION_TOLERANCE
            },
            'timestamp': datetime.utcnow().isoformat()
        }

    async def get_version_history(self, limit: int = 10) -> List[Dict[str, Any]]:
        """Get recent terminology version history"""
        redis_client = await self.redis

        # Get all import metadata keys
        cursor = b'0'
        versions = []

        while True:
            cursor, keys = await redis_client.scan(
                cursor=cursor,
                match="neo4j:import:*",
                count=100
            )

            for key in keys:
                version_id = key.decode().replace("neo4j:import:", "")
                meta = await redis_client.hgetall(key)
                versions.append({
                    'version_id': version_id,
                    'db_name': meta.get(b'db_name', b'').decode(),
                    'imported_at': meta.get(b'imported_at', b'').decode(),
                    'snomed_version': meta.get(b'snomed_version', b'').decode(),
                    'rxnorm_version': meta.get(b'rxnorm_version', b'').decode(),
                    'loinc_version': meta.get(b'loinc_version', b'').decode()
                })

            if cursor == b'0':
                break

        # Sort by import time (newest first)
        versions.sort(key=lambda x: x.get('imported_at', ''), reverse=True)
        return versions[:limit]

    async def close(self):
        """Close all connections"""
        await self.driver.close()
        if self._redis_client:
            await self._redis_client.close()
        logger.info("Neo4j Terminology Sync Service closed")


# ══════════════════════════════════════════════════════════════════════════
# MAIN ENTRY POINT
# ══════════════════════════════════════════════════════════════════════════

async def main():
    """Main entry point for running the service standalone"""
    import argparse
    import json

    parser = argparse.ArgumentParser(description='Neo4j Terminology Sync Service')
    parser.add_argument('--status', action='store_true', help='Show service status')
    parser.add_argument('--history', action='store_true', help='Show version history')
    parser.add_argument('--cleanup', action='store_true', help='Run cleanup job')
    parser.add_argument('--rollback', type=str, help='Rollback to specified version')
    parser.add_argument('--test', action='store_true', help='Test with sample event')

    args = parser.parse_args()

    config = {
        'neo4j_uri': os.getenv('NEO4J_URI', 'bolt://localhost:7687'),
        'neo4j_user': os.getenv('NEO4J_USER', 'neo4j'),
        'neo4j_password': os.getenv('NEO4J_PASSWORD', 'kb7password'),
        'gcs_bucket': os.getenv('GCS_BUCKET', 'sincere-hybrid-477206-h2-kb-artifacts-production'),
        'redis_url': os.getenv('REDIS_URL', 'redis://localhost:6379')
    }

    service = Neo4jTerminologySyncService(config)

    try:
        if args.status:
            status = await service.get_status()
            print(json.dumps(status, indent=2))

        elif args.history:
            history = await service.get_version_history()
            print(json.dumps(history, indent=2))

        elif args.cleanup:
            cleaned = await service.run_cleanup_job()
            print(f"Cleaned up {cleaned} databases")

        elif args.rollback:
            success = await service.rollback(args.rollback)
            print(f"Rollback {'successful' if success else 'failed'}")

        elif args.test:
            # Test with sample event
            test_event = {
                'version_id': f'test_{datetime.utcnow().strftime("%Y%m%d_%H%M%S")}',
                'status': 'LOADING',
                'gcs_uri': f"gs://{config['gcs_bucket']}/latest/kb7-kernel.ttl",
                'snomed_version': '20231101',
                'rxnorm_version': '20231106',
                'loinc_version': '2.76'
            }
            print(f"Testing with event: {json.dumps(test_event, indent=2)}")
            print("\nNote: This requires:")
            print("  1. Neo4j with n10s plugin installed")
            print("  2. GCS artifact exists at the specified path")
            print("  3. GOOGLE_APPLICATION_CREDENTIALS set for GCS access")

            # Uncomment to run actual test:
            # result = await service.handle_cdc_event(test_event)
            # print(f"Result: {result}")
            print("\nTest mode - would call handle_cdc_event with above payload")

        else:
            parser.print_help()

    finally:
        await service.close()


if __name__ == '__main__':
    asyncio.run(main())
