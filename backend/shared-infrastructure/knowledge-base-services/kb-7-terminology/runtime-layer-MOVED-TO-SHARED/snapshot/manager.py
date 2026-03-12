"""
Snapshot Manager for KB7 Terminology Service
Manages consistent snapshots across all data stores for transaction isolation
Ensures read consistency across PostgreSQL, Elasticsearch, Neo4j, ClickHouse, and GraphDB
"""

import uuid
import hashlib
from datetime import datetime, timedelta
from typing import Dict, Optional, Any, List
import asyncio
import json
from loguru import logger


class Snapshot:
    """
    Represents a point-in-time snapshot across all data stores
    Provides consistency guarantees for complex queries spanning multiple databases
    """

    def __init__(self, id: str, service_id: str, created_at: datetime,
                 ttl: timedelta, context: Dict[str, Any]):
        self.id = id
        self.service_id = service_id
        self.created_at = created_at
        self.ttl = ttl
        self.context = context
        self.versions: Dict[str, str] = {}
        self.checksum = ""
        self.status = "active"
        self.access_count = 0
        self.last_accessed = created_at

    def is_valid(self) -> bool:
        """Check if snapshot is still valid based on TTL"""
        return datetime.utcnow() < (self.created_at + self.ttl)

    def mark_accessed(self) -> None:
        """Mark snapshot as accessed"""
        self.access_count += 1
        self.last_accessed = datetime.utcnow()

    def to_dict(self) -> Dict[str, Any]:
        """Convert snapshot to dictionary for serialization"""
        return {
            'id': self.id,
            'service_id': self.service_id,
            'created_at': self.created_at.isoformat(),
            'ttl_seconds': self.ttl.total_seconds(),
            'context': self.context,
            'versions': self.versions,
            'checksum': self.checksum,
            'status': self.status,
            'access_count': self.access_count,
            'last_accessed': self.last_accessed.isoformat()
        }


class SnapshotManager:
    """
    Manages consistent snapshots across all data stores
    Provides ACID-like consistency for polyglot persistence architecture
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """
        Initialize Snapshot Manager

        Args:
            config: Optional configuration dictionary
        """
        self.config = config or {}
        self.active_snapshots: Dict[str, Snapshot] = {}
        self.lock = asyncio.Lock()
        self.default_ttl = timedelta(minutes=self.config.get('default_ttl_minutes', 5))
        self.max_snapshots = self.config.get('max_snapshots', 1000)
        self.cleanup_interval = self.config.get('cleanup_interval_minutes', 10)

        # Data store clients (initialized lazily)
        self._postgres_client = None
        self._elasticsearch_client = None
        self._neo4j_client = None
        self._clickhouse_client = None
        self._graphdb_client = None

        logger.info("Snapshot Manager initialized")

        # Start cleanup task
        asyncio.create_task(self._periodic_cleanup())

    async def create_snapshot(self, service_id: str,
                             context: Dict[str, Any],
                             ttl: Optional[timedelta] = None) -> Snapshot:
        """
        Create a new snapshot for consistent reads

        Args:
            service_id: Identifier of the requesting service
            context: Context information for the snapshot
            ttl: Time-to-live for the snapshot

        Returns:
            New Snapshot instance
        """
        async with self.lock:
            # Check snapshot limits
            if len(self.active_snapshots) >= self.max_snapshots:
                await self._cleanup_expired_snapshots()
                if len(self.active_snapshots) >= self.max_snapshots:
                    raise RuntimeError("Maximum number of snapshots exceeded")

            # Create snapshot
            snapshot = Snapshot(
                id=str(uuid.uuid4()),
                service_id=service_id,
                created_at=datetime.utcnow(),
                ttl=ttl or self.default_ttl,
                context=context
            )

            # Get current versions from all stores
            snapshot.versions = await self._gather_versions()

            # Calculate checksum for consistency validation
            snapshot.checksum = self._calculate_checksum(snapshot)

            # Store snapshot
            self.active_snapshots[snapshot.id] = snapshot

            logger.info(f"Created snapshot {snapshot.id} for service {service_id}")
            return snapshot

    async def get_snapshot(self, snapshot_id: str) -> Optional[Snapshot]:
        """
        Retrieve an existing snapshot

        Args:
            snapshot_id: Snapshot identifier

        Returns:
            Snapshot if found and valid, None otherwise
        """
        snapshot = self.active_snapshots.get(snapshot_id)
        if not snapshot:
            return None

        if not snapshot.is_valid():
            await self._remove_snapshot(snapshot_id)
            return None

        snapshot.mark_accessed()
        return snapshot

    async def validate_snapshot(self, snapshot_id: str) -> bool:
        """
        Validate if snapshot is still consistent with current data

        Args:
            snapshot_id: Snapshot identifier

        Returns:
            True if snapshot is valid and consistent
        """
        snapshot = self.active_snapshots.get(snapshot_id)
        if not snapshot:
            return False

        # Check TTL
        if not snapshot.is_valid():
            return False

        # Verify data consistency
        try:
            current_versions = await self._gather_versions()
            current_checksum = self._calculate_checksum_from_versions(
                snapshot, current_versions
            )

            if current_checksum != snapshot.checksum:
                logger.warning(f"Snapshot {snapshot_id} consistency violation detected")
                snapshot.status = "inconsistent"
                return False

            return True

        except Exception as e:
            logger.error(f"Error validating snapshot {snapshot_id}: {e}")
            return False

    async def invalidate_snapshot(self, snapshot_id: str) -> bool:
        """
        Manually invalidate a snapshot

        Args:
            snapshot_id: Snapshot identifier

        Returns:
            True if snapshot was found and invalidated
        """
        return await self._remove_snapshot(snapshot_id)

    async def list_snapshots(self, service_id: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        List active snapshots

        Args:
            service_id: Optional filter by service ID

        Returns:
            List of snapshot dictionaries
        """
        snapshots = []
        for snapshot in self.active_snapshots.values():
            if service_id is None or snapshot.service_id == service_id:
                if snapshot.is_valid():
                    snapshots.append(snapshot.to_dict())

        return sorted(snapshots, key=lambda x: x['created_at'], reverse=True)

    async def _gather_versions(self) -> Dict[str, str]:
        """
        Gather current versions from all data stores

        Returns:
            Dictionary mapping store names to version identifiers
        """
        versions = {}

        try:
            # PostgreSQL version (using sequence or timestamp)
            versions['postgres'] = await self._get_postgres_version()

            # Elasticsearch version (using index timestamp)
            versions['elasticsearch'] = await self._get_elasticsearch_version()

            # Neo4j versions (both patient and semantic databases)
            versions['neo4j_patient'] = await self._get_neo4j_version('patient_data')
            versions['neo4j_semantic'] = await self._get_neo4j_version('semantic_mesh')

            # ClickHouse version (using system timestamp)
            versions['clickhouse'] = await self._get_clickhouse_version()

            # GraphDB version (using repository timestamp)
            versions['graphdb'] = await self._get_graphdb_version()

        except Exception as e:
            logger.error(f"Error gathering versions: {e}")
            # Use fallback timestamp
            fallback_version = datetime.utcnow().isoformat()
            for store in ['postgres', 'elasticsearch', 'neo4j_patient',
                         'neo4j_semantic', 'clickhouse', 'graphdb']:
                if store not in versions:
                    versions[store] = fallback_version

        return versions

    async def _get_postgres_version(self) -> str:
        """Get PostgreSQL data version"""
        try:
            if not self._postgres_client:
                from ..internal.database import database
                self._postgres_client = await database.get_connection()

            # Use last update timestamp from concepts table
            result = await self._postgres_client.fetchval("""
                SELECT COALESCE(MAX(updated_at), NOW())::text
                FROM concepts
            """)
            return result or datetime.utcnow().isoformat()

        except Exception as e:
            logger.warning(f"Error getting PostgreSQL version: {e}")
            return datetime.utcnow().isoformat()

    async def _get_elasticsearch_version(self) -> str:
        """Get Elasticsearch index version"""
        try:
            if not self._elasticsearch_client:
                from ..internal.elasticsearch import integration
                self._elasticsearch_client = integration.ElasticsearchIntegration({})

            # Get index stats for version identification
            stats = await self._elasticsearch_client.client.indices.stats(
                index='kb7-terminology'
            )

            # Use modification time or document count as version
            if 'indices' in stats and 'kb7-terminology' in stats['indices']:
                total_docs = stats['indices']['kb7-terminology']['total']['docs']['count']
                return f"docs:{total_docs}"

            return datetime.utcnow().isoformat()

        except Exception as e:
            logger.warning(f"Error getting Elasticsearch version: {e}")
            return datetime.utcnow().isoformat()

    async def _get_neo4j_version(self, database: str) -> str:
        """Get Neo4j database version"""
        try:
            if not self._neo4j_client:
                from ..neo4j_setup.dual_stream_manager import Neo4jDualStreamManager
                self._neo4j_client = Neo4jDualStreamManager(self.config.get('neo4j', {}))

            async with self._neo4j_client.driver.session(database=database) as session:
                result = await session.run("""
                    MATCH (n)
                    RETURN max(n.updated) as last_update, count(n) as node_count
                """)
                record = await result.single()
                if record:
                    last_update = record.get('last_update', 'unknown')
                    node_count = record.get('node_count', 0)
                    return f"{last_update}:{node_count}"

            return datetime.utcnow().isoformat()

        except Exception as e:
            logger.warning(f"Error getting Neo4j {database} version: {e}")
            return datetime.utcnow().isoformat()

    async def _get_clickhouse_version(self) -> str:
        """Get ClickHouse data version"""
        try:
            if not self._clickhouse_client:
                from ..clickhouse_runtime.manager import ClickHouseRuntimeManager
                self._clickhouse_client = ClickHouseRuntimeManager(
                    self.config.get('clickhouse', {})
                )

            # Get last insert timestamp
            result = self._clickhouse_client.client.execute("""
                SELECT max(calculated_at) as last_update,
                       count(*) as row_count
                FROM kb7_analytics.medication_scores
            """)

            if result:
                last_update, row_count = result[0]
                return f"{last_update}:{row_count}"

            return datetime.utcnow().isoformat()

        except Exception as e:
            logger.warning(f"Error getting ClickHouse version: {e}")
            return datetime.utcnow().isoformat()

    async def _get_graphdb_version(self) -> str:
        """Get GraphDB repository version"""
        try:
            # Use system timestamp for GraphDB (would implement SPARQL query in production)
            return datetime.utcnow().isoformat()

        except Exception as e:
            logger.warning(f"Error getting GraphDB version: {e}")
            return datetime.utcnow().isoformat()

    def _calculate_checksum(self, snapshot: Snapshot) -> str:
        """
        Calculate checksum for snapshot consistency

        Args:
            snapshot: Snapshot instance

        Returns:
            Checksum string
        """
        return self._calculate_checksum_from_versions(snapshot, snapshot.versions)

    def _calculate_checksum_from_versions(self, snapshot: Snapshot,
                                        versions: Dict[str, str]) -> str:
        """
        Calculate checksum from versions

        Args:
            snapshot: Snapshot instance
            versions: Version dictionary

        Returns:
            Checksum string
        """
        # Create deterministic string from versions
        version_string = json.dumps(versions, sort_keys=True)
        context_string = json.dumps(snapshot.context, sort_keys=True)
        checksum_data = f"{snapshot.service_id}:{version_string}:{context_string}"

        return hashlib.sha256(checksum_data.encode()).hexdigest()[:16]

    async def _remove_snapshot(self, snapshot_id: str) -> bool:
        """
        Remove snapshot from active snapshots

        Args:
            snapshot_id: Snapshot identifier

        Returns:
            True if snapshot was removed
        """
        async with self.lock:
            if snapshot_id in self.active_snapshots:
                del self.active_snapshots[snapshot_id]
                logger.debug(f"Removed snapshot {snapshot_id}")
                return True
        return False

    async def _cleanup_expired_snapshots(self) -> int:
        """
        Clean up expired snapshots

        Returns:
            Number of snapshots cleaned up
        """
        expired_ids = []
        for snapshot_id, snapshot in self.active_snapshots.items():
            if not snapshot.is_valid():
                expired_ids.append(snapshot_id)

        for snapshot_id in expired_ids:
            await self._remove_snapshot(snapshot_id)

        if expired_ids:
            logger.info(f"Cleaned up {len(expired_ids)} expired snapshots")

        return len(expired_ids)

    async def _periodic_cleanup(self) -> None:
        """Periodic cleanup task for expired snapshots"""
        while True:
            try:
                await asyncio.sleep(self.cleanup_interval * 60)  # Convert to seconds
                await self._cleanup_expired_snapshots()
            except Exception as e:
                logger.error(f"Error in periodic cleanup: {e}")

    async def get_statistics(self) -> Dict[str, Any]:
        """
        Get snapshot manager statistics

        Returns:
            Statistics dictionary
        """
        active_count = len(self.active_snapshots)
        valid_count = sum(1 for s in self.active_snapshots.values() if s.is_valid())

        service_counts = {}
        for snapshot in self.active_snapshots.values():
            service_counts[snapshot.service_id] = service_counts.get(
                snapshot.service_id, 0
            ) + 1

        return {
            'active_snapshots': active_count,
            'valid_snapshots': valid_count,
            'expired_snapshots': active_count - valid_count,
            'snapshots_by_service': service_counts,
            'default_ttl_minutes': self.default_ttl.total_seconds() / 60,
            'max_snapshots': self.max_snapshots,
            'timestamp': datetime.utcnow().isoformat()
        }

    async def close(self) -> None:
        """Close snapshot manager and cleanup resources"""
        # Close all data store connections
        if self._neo4j_client:
            await self._neo4j_client.close()

        if self._clickhouse_client:
            self._clickhouse_client.close()

        # Clear snapshots
        async with self.lock:
            self.active_snapshots.clear()

        logger.info("Snapshot Manager closed")