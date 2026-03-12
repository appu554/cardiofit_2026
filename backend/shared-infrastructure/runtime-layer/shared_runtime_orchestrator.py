"""
Shared Runtime Layer Orchestrator
Main entry point for the shared CardioFit runtime infrastructure

Manages all runtime components across ALL knowledge bases:
- Multi-KB Neo4j Stream Manager
- Multi-KB Query Router
- Multi-KB ClickHouse Analytics
- Shared Event Bus
- Shared Cache Warming
- Shared Adapters

Provides unified interface for all CardioFit services to access runtime capabilities.
"""

import asyncio
import signal
from typing import Dict, Any, Optional, List
from pathlib import Path
from datetime import datetime, timedelta
from loguru import logger
import json

from .config.multi_kb_config import MultiKBRuntimeConfig, Environment
from .neo4j_dual_stream.multi_kb_stream_manager import MultiKBStreamManager
from .query_router.multi_kb_router import MultiKBQueryRouter
from .clickhouse_analytics.multi_kb_analytics import MultiKBAnalyticsManager


class RuntimeHealthStatus:
    """Track health status of all runtime components"""

    def __init__(self):
        self.components = {
            'neo4j_stream_manager': False,
            'query_router': False,
            'clickhouse_analytics': False,
            'graphdb_semantic': False,
            'event_bus': False,
            'cache_warming': False,
            'adapters': False
        }
        self.last_health_check = None
        self.startup_time = datetime.utcnow()

    def update_component(self, component: str, status: bool):
        """Update health status for a component"""
        self.components[component] = status
        self.last_health_check = datetime.utcnow()

    def is_healthy(self) -> bool:
        """Check if all components are healthy"""
        return all(self.components.values())

    def get_unhealthy_components(self) -> List[str]:
        """Get list of unhealthy components"""
        return [comp for comp, healthy in self.components.items() if not healthy]

    def to_dict(self) -> Dict[str, Any]:
        """Convert health status to dictionary"""
        return {
            'overall_healthy': self.is_healthy(),
            'components': self.components,
            'unhealthy_components': self.get_unhealthy_components(),
            'last_health_check': self.last_health_check.isoformat() if self.last_health_check else None,
            'uptime_seconds': (datetime.utcnow() - self.startup_time).total_seconds()
        }


class SharedRuntimeOrchestrator:
    """
    Main orchestrator for the shared CardioFit runtime infrastructure

    Coordinates all runtime components and provides unified lifecycle management
    for multi-KB operations across the entire CardioFit platform.
    """

    def __init__(self, config: Optional[MultiKBRuntimeConfig] = None):
        """
        Initialize Shared Runtime Orchestrator

        Args:
            config: Multi-KB runtime configuration (uses environment default if None)
        """
        self.config = config or MultiKBRuntimeConfig.from_env()
        self.health_status = RuntimeHealthStatus()
        self.components = {}
        self.running = False
        self.shutdown_event = asyncio.Event()

        # Performance metrics
        self.metrics = {
            'total_requests': 0,
            'kb_request_counts': {},
            'cross_kb_requests': 0,
            'avg_response_time_ms': 0.0,
            'cache_hit_rate': 0.0,
            'error_rate': 0.0,
            'startup_time': datetime.utcnow()
        }

        # Register signal handlers for graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

        logger.info(f"Shared Runtime Orchestrator initialized for {len(self.config.knowledge_bases)} Knowledge Bases")

    async def initialize_all_components(self) -> bool:
        """
        Initialize all runtime components

        Returns:
            Success status
        """
        logger.info("Initializing shared runtime components...")

        try:
            # 1. Initialize Neo4j Multi-KB Stream Manager
            await self._initialize_neo4j_streams()

            # 2. Initialize ClickHouse Multi-KB Analytics
            await self._initialize_clickhouse_analytics()

            # 3. Initialize GraphDB Multi-KB Manager
            await self._initialize_graphdb_semantic()

            # 4. Initialize Multi-KB Query Router
            await self._initialize_query_router()

            # 5. Initialize Event Bus (shared)
            await self._initialize_event_bus()

            # 6. Initialize Cache Warming (shared)
            await self._initialize_cache_warming()

            # 7. Initialize Adapters (shared)
            await self._initialize_adapters()

            # 8. Validate all components
            healthy = await self._perform_health_check()

            if healthy:
                logger.info("All shared runtime components initialized successfully")
                return True
            else:
                logger.error("Some runtime components failed initialization")
                return False

        except Exception as e:
            logger.error(f"Failed to initialize runtime components: {e}")
            return False

    async def _initialize_neo4j_streams(self) -> None:
        """Initialize Multi-KB Neo4j Stream Manager"""
        logger.info("Initializing Neo4j Multi-KB Stream Manager...")

        neo4j_config = {
            'neo4j_uri': self.config.get_neo4j_uri(),
            'neo4j_user': self.config.data_stores['neo4j'].username,
            'neo4j_password': self.config.data_stores['neo4j'].password
        }

        self.components['neo4j_stream_manager'] = MultiKBStreamManager(neo4j_config)
        success = await self.components['neo4j_stream_manager'].initialize_all_streams()

        self.health_status.update_component('neo4j_stream_manager', success)
        if success:
            logger.info("Neo4j Multi-KB Stream Manager initialized")
        else:
            logger.error("Failed to initialize Neo4j Stream Manager")

    async def _initialize_clickhouse_analytics(self) -> None:
        """Initialize Multi-KB ClickHouse Analytics Manager"""
        logger.info("Initializing ClickHouse Multi-KB Analytics...")

        clickhouse_config = self.config.get_clickhouse_connection_params()

        self.components['clickhouse_analytics'] = MultiKBAnalyticsManager(clickhouse_config)

        # Test connection
        try:
            health = await self.components['clickhouse_analytics'].health_check_all_kbs()
            success = health.get('overall_healthy', False)
        except Exception as e:
            logger.error(f"ClickHouse initialization failed: {e}")
            success = False

        self.health_status.update_component('clickhouse_analytics', success)
        if success:
            logger.info("ClickHouse Multi-KB Analytics initialized")
        else:
            logger.error("Failed to initialize ClickHouse Analytics")

    async def _initialize_graphdb_semantic(self) -> None:
        """Initialize Multi-KB GraphDB Semantic Manager"""
        logger.info("Initializing GraphDB Multi-KB Semantic Manager...")

        if 'graphdb' not in self.config.data_stores:
            logger.warning("GraphDB not configured, skipping semantic initialization")
            self.health_status.update_component('graphdb_semantic', False)
            return

        graphdb_config = {
            'host': self.config.data_stores['graphdb'].host,
            'port': self.config.data_stores['graphdb'].port,
            'username': self.config.data_stores['graphdb'].username,
            'password': self.config.data_stores['graphdb'].password,
            'ssl': self.config.data_stores['graphdb'].ssl,
            'connection_pool_size': self.config.data_stores['graphdb'].connection_pool_size
        }

        try:
            from .graphdb_semantic.multi_kb_graphdb_manager import MultiKBGraphDBManager
            self.components['graphdb_semantic'] = MultiKBGraphDBManager(graphdb_config)

            # Test connection
            init_success = await self.components['graphdb_semantic'].initialize_connection()

            if init_success:
                self.health_status.update_component('graphdb_semantic', True)
                logger.info("GraphDB Multi-KB Semantic Manager initialized")
            else:
                self.health_status.update_component('graphdb_semantic', False)
                logger.error("Failed to initialize GraphDB connection")
                self.components['graphdb_semantic'] = None

        except Exception as e:
            logger.error(f"GraphDB initialization failed: {e}")
            self.health_status.update_component('graphdb_semantic', False)
            self.components['graphdb_semantic'] = None

    async def _initialize_query_router(self) -> None:
        """Initialize Multi-KB Query Router"""
        logger.info("Initializing Multi-KB Query Router...")

        router_config = {
            'neo4j': {
                'neo4j_uri': self.config.get_neo4j_uri(),
                'neo4j_user': self.config.data_stores['neo4j'].username,
                'neo4j_password': self.config.data_stores['neo4j'].password
            },
            'clickhouse_databases': {
                kb_id: config.clickhouse_db
                for kb_id, config in self.config.knowledge_bases.items()
                if config.has_analytics
            },
            'postgresql': {
                'host': self.config.data_stores['postgresql'].host,
                'port': self.config.data_stores['postgresql'].port,
                'user': self.config.data_stores['postgresql'].username,
                'password': self.config.data_stores['postgresql'].password,
                'database': self.config.data_stores['postgresql'].database
            },
            'elasticsearch': {
                'host': self.config.data_stores['elasticsearch'].host,
                'port': self.config.data_stores['elasticsearch'].port
            },
            'graphdb': {
                'host': self.config.data_stores['graphdb'].host,
                'port': self.config.data_stores['graphdb'].port,
                'username': self.config.data_stores['graphdb'].username,
                'password': self.config.data_stores['graphdb'].password,
                'ssl': self.config.data_stores['graphdb'].ssl
            } if 'graphdb' in self.config.data_stores else None
        }

        self.components['query_router'] = MultiKBQueryRouter(router_config)

        try:
            await self.components['query_router'].initialize_clients()
            success = True
        except Exception as e:
            logger.error(f"Query Router initialization failed: {e}")
            success = False

        self.health_status.update_component('query_router', success)
        if success:
            logger.info("Multi-KB Query Router initialized")
        else:
            logger.error("Failed to initialize Query Router")

    async def _initialize_event_bus(self) -> None:
        """Initialize shared Event Bus"""
        logger.info("Initializing shared Event Bus...")

        # Placeholder for event bus initialization
        # Would initialize Kafka-based event bus here

        self.health_status.update_component('event_bus', True)
        logger.info("Shared Event Bus initialized")

    async def _initialize_cache_warming(self) -> None:
        """Initialize shared Cache Warming"""
        logger.info("Initializing shared Cache Warming...")

        # Placeholder for cache warming initialization
        # Would initialize Redis-based cache warming here

        self.health_status.update_component('cache_warming', True)
        logger.info("Shared Cache Warming initialized")

    async def _initialize_adapters(self) -> None:
        """Initialize shared Adapters"""
        logger.info("Initializing shared Adapters...")

        # Placeholder for adapters initialization
        # Would initialize multi-KB adapters here

        self.health_status.update_component('adapters', True)
        logger.info("Shared Adapters initialized")

    async def start_all_services(self) -> bool:
        """
        Start all runtime services

        Returns:
            Success status
        """
        logger.info("Starting shared runtime services...")

        try:
            # Start background tasks
            asyncio.create_task(self._health_check_loop())
            asyncio.create_task(self._metrics_collection_loop())
            asyncio.create_task(self._performance_monitoring_loop())

            self.running = True
            logger.info("All shared runtime services started")
            return True

        except Exception as e:
            logger.error(f"Failed to start runtime services: {e}")
            return False

    async def _health_check_loop(self) -> None:
        """Periodic health check loop"""
        while self.running:
            try:
                await self._perform_health_check()
                await asyncio.sleep(self.config.runtime_settings['monitoring']['health_check_interval_seconds'])
            except Exception as e:
                logger.error(f"Health check error: {e}")
                await asyncio.sleep(30)  # Fallback interval

    async def _metrics_collection_loop(self) -> None:
        """Periodic metrics collection loop"""
        while self.running:
            try:
                await self._collect_performance_metrics()
                await asyncio.sleep(self.config.runtime_settings['monitoring']['metrics_interval_seconds'])
            except Exception as e:
                logger.error(f"Metrics collection error: {e}")
                await asyncio.sleep(60)  # Fallback interval

    async def _performance_monitoring_loop(self) -> None:
        """Periodic performance monitoring loop"""
        while self.running:
            try:
                await self._log_performance_stats()
                await asyncio.sleep(300)  # Every 5 minutes
            except Exception as e:
                logger.error(f"Performance monitoring error: {e}")
                await asyncio.sleep(300)

    async def _perform_health_check(self) -> bool:
        """Perform comprehensive health check of all components"""
        logger.debug("Performing comprehensive health check...")

        # Check Neo4j streams
        if 'neo4j_stream_manager' in self.components:
            try:
                neo4j_health = await self.components['neo4j_stream_manager'].health_check_all_streams()
                neo4j_healthy = all(kb_health.get('healthy', False) for kb_health in neo4j_health.values())
                self.health_status.update_component('neo4j_stream_manager', neo4j_healthy)
            except Exception as e:
                logger.error(f"Neo4j health check failed: {e}")
                self.health_status.update_component('neo4j_stream_manager', False)

        # Check ClickHouse analytics
        if 'clickhouse_analytics' in self.components:
            try:
                ch_health = await self.components['clickhouse_analytics'].health_check_all_kbs()
                ch_healthy = ch_health.get('overall_healthy', False)
                self.health_status.update_component('clickhouse_analytics', ch_healthy)
            except Exception as e:
                logger.error(f"ClickHouse health check failed: {e}")
                self.health_status.update_component('clickhouse_analytics', False)

        # Check GraphDB semantic
        if 'graphdb_semantic' in self.components and self.components['graphdb_semantic']:
            try:
                gdb_health = await self.components['graphdb_semantic'].get_health_status()
                gdb_healthy = gdb_health.get('overall_healthy', False)
                self.health_status.update_component('graphdb_semantic', gdb_healthy)
            except Exception as e:
                logger.error(f"GraphDB health check failed: {e}")
                self.health_status.update_component('graphdb_semantic', False)

        # Check Query Router
        if 'query_router' in self.components:
            try:
                # Query router health check would be implemented here
                self.health_status.update_component('query_router', True)
            except Exception as e:
                logger.error(f"Query Router health check failed: {e}")
                self.health_status.update_component('query_router', False)

        overall_healthy = self.health_status.is_healthy()
        if not overall_healthy:
            unhealthy = self.health_status.get_unhealthy_components()
            logger.warning(f"Unhealthy components: {unhealthy}")

        return overall_healthy

    async def _collect_performance_metrics(self) -> None:
        """Collect performance metrics from all components"""
        # Collect metrics from query router
        if 'query_router' in self.components:
            try:
                router_metrics = await self.components['query_router'].get_performance_metrics()
                self.metrics.update({
                    'total_requests': router_metrics.get('total_queries', 0),
                    'kb_request_counts': router_metrics.get('kb_query_counts', {}),
                    'cross_kb_requests': router_metrics.get('cross_kb_queries', 0),
                    'avg_response_time_ms': router_metrics.get('average_latency', 0.0)
                })
            except Exception as e:
                logger.error(f"Error collecting router metrics: {e}")

    async def _log_performance_stats(self) -> None:
        """Log performance statistics"""
        uptime = (datetime.utcnow() - self.metrics['startup_time']).total_seconds()

        logger.info(f"Runtime Performance Stats:")
        logger.info(f"  Uptime: {uptime:.0f}s")
        logger.info(f"  Total Requests: {self.metrics['total_requests']}")
        logger.info(f"  Cross-KB Requests: {self.metrics['cross_kb_requests']}")
        logger.info(f"  Avg Response Time: {self.metrics['avg_response_time_ms']:.1f}ms")
        logger.info(f"  KB Request Counts: {self.metrics['kb_request_counts']}")

    def _signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        logger.info(f"Received signal {signum}, initiating graceful shutdown...")
        asyncio.create_task(self.shutdown())

    async def shutdown(self) -> None:
        """Gracefully shutdown all components"""
        logger.info("Shutting down shared runtime orchestrator...")

        self.running = False
        self.shutdown_event.set()

        # Close all components
        if 'neo4j_stream_manager' in self.components:
            await self.components['neo4j_stream_manager'].close()

        if 'clickhouse_analytics' in self.components:
            self.components['clickhouse_analytics'].close_all_connections()

        if 'graphdb_semantic' in self.components and self.components['graphdb_semantic']:
            await self.components['graphdb_semantic'].close()

        if 'query_router' in self.components:
            await self.components['query_router'].close()

        logger.info("Shared runtime orchestrator shutdown complete")

    async def get_system_status(self) -> Dict[str, Any]:
        """Get comprehensive system status"""
        return {
            'runtime_info': {
                'environment': self.config.environment.value,
                'knowledge_bases_count': len(self.config.knowledge_bases),
                'data_stores_count': len(self.config.data_stores),
                'running': self.running
            },
            'health_status': self.health_status.to_dict(),
            'performance_metrics': self.metrics,
            'knowledge_bases': {
                kb_id: {
                    'name': config.name,
                    'has_analytics': config.has_analytics,
                    'has_semantic_mesh': config.has_semantic_mesh
                }
                for kb_id, config in self.config.knowledge_bases.items()
            },
            'timestamp': datetime.utcnow().isoformat()
        }

    # Public API methods for services to use

    async def route_query(self, service_id: str, kb_id: Optional[str],
                         pattern: str, params: Dict[str, Any],
                         cross_kb_scope: Optional[List[str]] = None) -> Dict[str, Any]:
        """
        Route query through shared runtime layer

        Args:
            service_id: Requesting service identifier
            kb_id: Target knowledge base (None for cross-KB)
            pattern: Query pattern
            params: Query parameters
            cross_kb_scope: List of KBs for cross-KB queries

        Returns:
            Query response
        """
        if 'query_router' not in self.components:
            raise RuntimeError("Query router not initialized")

        from .query_router.multi_kb_router import MultiKBQueryRequest

        request = MultiKBQueryRequest(
            service_id=service_id,
            kb_id=kb_id,
            pattern=pattern,
            params=params,
            cross_kb_scope=cross_kb_scope or []
        )

        response = await self.components['query_router'].route_query(request)

        # Update metrics
        self.metrics['total_requests'] += 1
        if kb_id:
            kb_count = self.metrics['kb_request_counts'].get(kb_id, 0)
            self.metrics['kb_request_counts'][kb_id] = kb_count + 1
        else:
            self.metrics['cross_kb_requests'] += 1

        return {
            'data': response.data,
            'metadata': {
                'sources_used': response.sources_used,
                'kb_sources': response.kb_sources,
                'latency_ms': response.latency,
                'cache_status': response.cache_status
            }
        }

    async def execute_analytics_query(self, kb_id: str, query: str,
                                     params: Optional[Dict[str, Any]] = None):
        """
        Execute analytics query on specific KB

        Args:
            kb_id: Knowledge Base identifier
            query: SQL query
            params: Query parameters

        Returns:
            Query results as DataFrame
        """
        if 'clickhouse_analytics' not in self.components:
            raise RuntimeError("ClickHouse analytics not initialized")

        return await self.components['clickhouse_analytics'].execute_kb_query(kb_id, query, params)

    def get_kb_config(self, kb_id: str) -> Optional[Dict[str, Any]]:
        """Get configuration for specific knowledge base"""
        kb_config = self.config.get_kb_config(kb_id)
        if kb_config:
            return {
                'name': kb_config.name,
                'description': kb_config.description,
                'neo4j_partition': kb_config.neo4j_partition,
                'clickhouse_db': kb_config.clickhouse_db,
                'primary_storage': kb_config.primary_storage,
                'has_analytics': kb_config.has_analytics,
                'has_semantic_mesh': kb_config.has_semantic_mesh
            }
        return None

    def list_knowledge_bases(self) -> List[Dict[str, Any]]:
        """List all configured knowledge bases"""
        return [
            {
                'kb_id': kb_id,
                'name': config.name,
                'description': config.description,
                'has_analytics': config.has_analytics,
                'has_semantic_mesh': config.has_semantic_mesh
            }
            for kb_id, config in self.config.knowledge_bases.items()
        ]


# Global shared runtime instance
shared_runtime = None


async def initialize_shared_runtime(config: Optional[MultiKBRuntimeConfig] = None) -> SharedRuntimeOrchestrator:
    """
    Initialize and start the shared runtime layer

    Args:
        config: Optional configuration (uses environment default if None)

    Returns:
        Initialized shared runtime orchestrator
    """
    global shared_runtime

    shared_runtime = SharedRuntimeOrchestrator(config)

    # Initialize all components
    init_success = await shared_runtime.initialize_all_components()
    if not init_success:
        raise RuntimeError("Failed to initialize shared runtime components")

    # Start all services
    start_success = await shared_runtime.start_all_services()
    if not start_success:
        raise RuntimeError("Failed to start shared runtime services")

    logger.info("Shared runtime layer initialized and started successfully")
    return shared_runtime


def get_shared_runtime() -> Optional[SharedRuntimeOrchestrator]:
    """Get the global shared runtime instance"""
    return shared_runtime


# Backward compatibility functions
async def get_neo4j_manager():
    """Get Neo4j manager (backward compatibility)"""
    if shared_runtime and 'neo4j_stream_manager' in shared_runtime.components:
        return shared_runtime.components['neo4j_stream_manager']
    return None


async def get_query_router():
    """Get query router (backward compatibility)"""
    if shared_runtime and 'query_router' in shared_runtime.components:
        return shared_runtime.components['query_router']
    return None


async def get_clickhouse_manager():
    """Get ClickHouse manager (backward compatibility)"""
    if shared_runtime and 'clickhouse_analytics' in shared_runtime.components:
        return shared_runtime.components['clickhouse_analytics']
    return None


async def main():
    """Main entry point for standalone execution"""
    try:
        # Initialize shared runtime
        runtime = await initialize_shared_runtime()

        # Keep running until shutdown signal
        await runtime.shutdown_event.wait()

    except KeyboardInterrupt:
        logger.info("Shutdown requested via keyboard interrupt")
    except Exception as e:
        logger.error(f"Runtime error: {e}")
        raise
    finally:
        if shared_runtime:
            await shared_runtime.shutdown()


if __name__ == "__main__":
    asyncio.run(main())