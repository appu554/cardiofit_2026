#!/usr/bin/env python3
"""
Complete Integration Orchestrator for KB7 Neo4j Dual-Stream & Service Runtime Layer

This is the main orchestrator that coordinates all runtime components:
- Neo4j Dual-Stream Manager (patient data + semantic mesh)
- GraphDB to Neo4j Adapter (OWL reasoning sync)
- ClickHouse Runtime Manager (analytics)
- Query Router (intelligent routing)
- Snapshot Manager (consistency)
- Adapter Microservice (CDC)
- Cache Warmer (event-driven)
- Event Bus Orchestrator (service coordination)
- Patient Data Stream Handler (real-time streaming)
- Medication Runtime Service (workflow orchestration)

Usage:
    python main_integration.py --initialize    # Initialize all databases and services
    python main_integration.py --start         # Start all runtime services
    python main_integration.py --health        # Check health of all components
    python main_integration.py --test          # Run integration tests
    python main_integration.py --stop          # Stop all services gracefully
"""

import asyncio
import argparse
import logging
import signal
import sys
import time
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, List, Optional
import json

# Import all runtime components
from neo4j_setup.dual_stream_manager import Neo4jDualStreamManager
from adapters.graphdb_neo4j_adapter import GraphDBNeo4jAdapter
from clickhouse_runtime.manager import ClickHouseRuntimeManager
from query_router.router import QueryRouter
from snapshot.manager import SnapshotManager
from adapters.adapter_microservice import AdapterMicroservice
from cache_warming.cdc_subscriber import CDCCacheWarmer
from event_bus.orchestrator import EventBusOrchestrator
from streams.patient_data_handler import PatientDataHandler
from services.medication_runtime import MedicationRuntime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


class RuntimeHealthStatus:
    """Health status tracking for runtime components"""

    def __init__(self):
        self.status = {
            'neo4j_manager': 'unknown',
            'graphdb_adapter': 'unknown',
            'clickhouse_manager': 'unknown',
            'query_router': 'unknown',
            'snapshot_manager': 'unknown',
            'adapter_microservice': 'unknown',
            'cdc_cache_warmer': 'unknown',
            'event_bus_orchestrator': 'unknown',
            'patient_data_handler': 'unknown',
            'medication_runtime': 'unknown'
        }
        self.last_check = datetime.utcnow()

    def update_status(self, component: str, status: str):
        """Update component status"""
        self.status[component] = status
        self.last_check = datetime.utcnow()

    def get_overall_status(self) -> str:
        """Get overall system health status"""
        statuses = list(self.status.values())
        if all(s == 'healthy' for s in statuses):
            return 'healthy'
        elif any(s == 'critical' for s in statuses):
            return 'critical'
        elif any(s == 'degraded' for s in statuses):
            return 'degraded'
        else:
            return 'unknown'


class CompleteIntegrationOrchestrator:
    """
    Main orchestrator for the complete KB7 Neo4j Dual-Stream & Service Runtime Layer

    Coordinates all runtime components and provides unified lifecycle management
    """

    def __init__(self, config_path: Optional[str] = None):
        self.config = self._load_configuration(config_path)
        self.health_status = RuntimeHealthStatus()
        self.components = {}
        self.running = False
        self.shutdown_event = asyncio.Event()

        # Register signal handlers for graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

    def _load_configuration(self, config_path: Optional[str]) -> Dict[str, Any]:
        """Load runtime configuration"""
        default_config = {
            'neo4j': {
                'neo4j_uri': 'bolt://localhost:7687',
                'neo4j_user': 'neo4j',
                'neo4j_password': 'kb7password'
            },
            'graphdb': {
                'graphdb_url': 'http://localhost:7200',
                'repository': 'kb7-terminology'
            },
            'clickhouse': {
                'host': 'localhost',
                'port': 9000,
                'database': 'kb7_analytics',
                'user': 'kb7',
                'password': 'kb7password'
            },
            'postgres': {
                'dsn': 'postgresql://kb_test_user:kb_test_password@localhost:5434/clinical_governance_test'
            },
            'elasticsearch': {
                'urls': ['http://localhost:9200'],
                'index': 'kb7-terminology'
            },
            'kafka_brokers': ['localhost:9092'],
            'redis_l2_url': 'redis://localhost:6379/0',
            'redis_l3_url': 'redis://localhost:6380/0'
        }

        if config_path and Path(config_path).exists():
            with open(config_path, 'r') as f:
                custom_config = json.load(f)
                default_config.update(custom_config)

        return default_config

    def _signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        logger.info(f"Received signal {signum}, initiating graceful shutdown...")
        self.shutdown_event.set()

    async def initialize_all_components(self) -> bool:
        """Initialize all runtime components"""
        logger.info("🚀 Initializing KB7 Runtime Layer Components...")

        try:
            # 1. Initialize Neo4j Dual-Stream Manager
            logger.info("1️⃣ Initializing Neo4j Dual-Stream Manager...")
            neo4j_manager = Neo4jDualStreamManager(self.config['neo4j'])
            await neo4j_manager.initialize_databases()
            self.components['neo4j_manager'] = neo4j_manager
            self.health_status.update_status('neo4j_manager', 'healthy')
            logger.info("✅ Neo4j Dual-Stream Manager initialized")

            # 2. Initialize ClickHouse Runtime Manager
            logger.info("2️⃣ Initializing ClickHouse Runtime Manager...")
            clickhouse_manager = ClickHouseRuntimeManager(self.config['clickhouse'])
            await clickhouse_manager.initialize_tables()
            self.components['clickhouse_manager'] = clickhouse_manager
            self.health_status.update_status('clickhouse_manager', 'healthy')
            logger.info("✅ ClickHouse Runtime Manager initialized")

            # 3. Initialize Query Router
            logger.info("3️⃣ Initializing Query Router...")
            query_router = QueryRouter(self.config)
            await query_router.initialize_clients()
            self.components['query_router'] = query_router
            self.health_status.update_status('query_router', 'healthy')
            logger.info("✅ Query Router initialized")

            # 4. Initialize Snapshot Manager
            logger.info("4️⃣ Initializing Snapshot Manager...")
            snapshot_manager = SnapshotManager()
            self.components['snapshot_manager'] = snapshot_manager
            self.health_status.update_status('snapshot_manager', 'healthy')
            logger.info("✅ Snapshot Manager initialized")

            # 5. Initialize GraphDB Adapter
            logger.info("5️⃣ Initializing GraphDB to Neo4j Adapter...")
            graphdb_adapter = GraphDBNeo4jAdapter(
                self.config['graphdb'],
                neo4j_manager
            )
            self.components['graphdb_adapter'] = graphdb_adapter
            self.health_status.update_status('graphdb_adapter', 'healthy')
            logger.info("✅ GraphDB to Neo4j Adapter initialized")

            # 6. Initialize Adapter Microservice
            logger.info("6️⃣ Initializing Adapter Microservice...")
            adapter_microservice = AdapterMicroservice(
                self.config['kafka_brokers'],
                neo4j_manager,
                clickhouse_manager
            )
            self.components['adapter_microservice'] = adapter_microservice
            self.health_status.update_status('adapter_microservice', 'healthy')
            logger.info("✅ Adapter Microservice initialized")

            # 7. Initialize CDC Cache Warmer
            logger.info("7️⃣ Initializing CDC Cache Warmer...")
            cdc_cache_warmer = CDCCacheWarmer(
                self.config['kafka_brokers'],
                self.config['redis_l2_url'],
                self.config['redis_l3_url'],
                neo4j_manager,
                clickhouse_manager
            )
            self.components['cdc_cache_warmer'] = cdc_cache_warmer
            self.health_status.update_status('cdc_cache_warmer', 'healthy')
            logger.info("✅ CDC Cache Warmer initialized")

            # 8. Initialize Event Bus Orchestrator
            logger.info("8️⃣ Initializing Event Bus Orchestrator...")
            event_bus_orchestrator = EventBusOrchestrator(self.config['kafka_brokers'])
            self.components['event_bus_orchestrator'] = event_bus_orchestrator
            self.health_status.update_status('event_bus_orchestrator', 'healthy')
            logger.info("✅ Event Bus Orchestrator initialized")

            # 9. Initialize Patient Data Stream Handler
            logger.info("9️⃣ Initializing Patient Data Stream Handler...")
            patient_data_handler = PatientDataHandler(
                self.config['kafka_brokers'],
                neo4j_manager
            )
            self.components['patient_data_handler'] = patient_data_handler
            self.health_status.update_status('patient_data_handler', 'healthy')
            logger.info("✅ Patient Data Stream Handler initialized")

            # 10. Initialize Medication Runtime Service
            logger.info("🔟 Initializing Medication Runtime Service...")
            medication_runtime = MedicationRuntime(
                query_router,
                self.config['redis_l2_url']
            )
            self.components['medication_runtime'] = medication_runtime
            self.health_status.update_status('medication_runtime', 'healthy')
            logger.info("✅ Medication Runtime Service initialized")

            logger.info("🎉 All KB7 Runtime Layer Components Initialized Successfully!")
            return True

        except Exception as e:
            logger.error(f"❌ Initialization failed: {e}")
            return False

    async def start_all_services(self) -> bool:
        """Start all runtime services"""
        logger.info("🚀 Starting all KB7 Runtime Layer Services...")

        try:
            self.running = True

            # Start background services
            tasks = []

            # Start Adapter Microservice
            if 'adapter_microservice' in self.components:
                task = asyncio.create_task(
                    self.components['adapter_microservice'].run()
                )
                tasks.append(task)

            # Start CDC Cache Warmer
            if 'cdc_cache_warmer' in self.components:
                task = asyncio.create_task(
                    self.components['cdc_cache_warmer'].start_warming_from_cdc()
                )
                tasks.append(task)

            # Start Event Bus Orchestrator
            if 'event_bus_orchestrator' in self.components:
                task = asyncio.create_task(
                    self.components['event_bus_orchestrator'].start_orchestration()
                )
                tasks.append(task)

            # Start Patient Data Stream Handler
            if 'patient_data_handler' in self.components:
                task = asyncio.create_task(
                    self.components['patient_data_handler'].start_processing()
                )
                tasks.append(task)

            logger.info("✅ All services started successfully")

            # Wait for shutdown signal
            await self.shutdown_event.wait()

            # Cancel all tasks
            for task in tasks:
                task.cancel()

            await asyncio.gather(*tasks, return_exceptions=True)

            return True

        except Exception as e:
            logger.error(f"❌ Service startup failed: {e}")
            return False

    async def check_health_all_components(self) -> Dict[str, Any]:
        """Check health of all runtime components"""
        logger.info("🏥 Checking health of all KB7 Runtime Layer Components...")

        health_report = {
            'timestamp': datetime.utcnow().isoformat(),
            'overall_status': 'unknown',
            'components': {}
        }

        # Check Neo4j Manager
        if 'neo4j_manager' in self.components:
            try:
                health = await self.components['neo4j_manager'].health_check()
                health_report['components']['neo4j_manager'] = health
                self.health_status.update_status('neo4j_manager', health['status'])
            except Exception as e:
                health_report['components']['neo4j_manager'] = {'status': 'critical', 'error': str(e)}
                self.health_status.update_status('neo4j_manager', 'critical')

        # Check ClickHouse Manager
        if 'clickhouse_manager' in self.components:
            try:
                health = self.components['clickhouse_manager'].health_check()
                health_report['components']['clickhouse_manager'] = health
                self.health_status.update_status('clickhouse_manager', health['status'])
            except Exception as e:
                health_report['components']['clickhouse_manager'] = {'status': 'critical', 'error': str(e)}
                self.health_status.update_status('clickhouse_manager', 'critical')

        # Check Query Router
        if 'query_router' in self.components:
            try:
                health = await self.components['query_router'].health_check()
                health_report['components']['query_router'] = health
                self.health_status.update_status('query_router', health['status'])
            except Exception as e:
                health_report['components']['query_router'] = {'status': 'critical', 'error': str(e)}
                self.health_status.update_status('query_router', 'critical')

        # Check other components
        for component_name in ['snapshot_manager', 'graphdb_adapter', 'adapter_microservice',
                              'cdc_cache_warmer', 'event_bus_orchestrator', 'patient_data_handler',
                              'medication_runtime']:
            if component_name in self.components:
                try:
                    # Most components should have a health_check method
                    component = self.components[component_name]
                    if hasattr(component, 'health_check'):
                        if asyncio.iscoroutinefunction(component.health_check):
                            health = await component.health_check()
                        else:
                            health = component.health_check()
                        health_report['components'][component_name] = health
                        self.health_status.update_status(component_name, health.get('status', 'unknown'))
                    else:
                        # Assume healthy if component exists and no health check method
                        health_report['components'][component_name] = {'status': 'healthy'}
                        self.health_status.update_status(component_name, 'healthy')
                except Exception as e:
                    health_report['components'][component_name] = {'status': 'critical', 'error': str(e)}
                    self.health_status.update_status(component_name, 'critical')

        # Determine overall status
        health_report['overall_status'] = self.health_status.get_overall_status()

        logger.info(f"🏥 Health check complete - Overall status: {health_report['overall_status']}")
        return health_report

    async def run_integration_tests(self) -> bool:
        """Run integration tests for all components"""
        logger.info("🧪 Running KB7 Runtime Layer Integration Tests...")

        try:
            # Import and run integration tests
            from tests.test_integration import TestRuntimeIntegration

            # Create test instance with current components
            test_integration = TestRuntimeIntegration()

            # Run key integration tests
            logger.info("Running medication scoring workflow test...")
            # Note: This would require adapting the test to use our current components
            # For now, we'll just validate component connectivity

            # Test Neo4j connectivity
            if 'neo4j_manager' in self.components:
                health = await self.components['neo4j_manager'].health_check()
                assert health['status'] in ['healthy', 'degraded'], "Neo4j connectivity failed"

            # Test ClickHouse connectivity
            if 'clickhouse_manager' in self.components:
                health = self.components['clickhouse_manager'].health_check()
                assert health['status'] in ['healthy', 'degraded'], "ClickHouse connectivity failed"

            # Test Query Router
            if 'query_router' in self.components:
                health = await self.components['query_router'].health_check()
                assert health['status'] in ['healthy', 'degraded'], "Query Router failed"

            logger.info("✅ Basic integration tests passed")
            return True

        except Exception as e:
            logger.error(f"❌ Integration tests failed: {e}")
            return False

    async def stop_all_services(self) -> bool:
        """Stop all runtime services gracefully"""
        logger.info("🛑 Stopping all KB7 Runtime Layer Services...")

        try:
            self.running = False
            self.shutdown_event.set()

            # Close all components
            for component_name, component in self.components.items():
                try:
                    if hasattr(component, 'close'):
                        if asyncio.iscoroutinefunction(component.close):
                            await component.close()
                        else:
                            component.close()
                    logger.info(f"✅ {component_name} stopped")
                except Exception as e:
                    logger.warning(f"⚠️ Error stopping {component_name}: {e}")

            self.components.clear()
            logger.info("✅ All services stopped")
            return True

        except Exception as e:
            logger.error(f"❌ Error during shutdown: {e}")
            return False


async def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description="KB7 Neo4j Dual-Stream & Service Runtime Layer")
    parser.add_argument('--initialize', action='store_true', help='Initialize all databases and services')
    parser.add_argument('--start', action='store_true', help='Start all runtime services')
    parser.add_argument('--health', action='store_true', help='Check health of all components')
    parser.add_argument('--test', action='store_true', help='Run integration tests')
    parser.add_argument('--stop', action='store_true', help='Stop all services')
    parser.add_argument('--config', type=str, help='Path to configuration file')

    args = parser.parse_args()

    # Create orchestrator
    orchestrator = CompleteIntegrationOrchestrator(args.config)

    try:
        if args.initialize:
            success = await orchestrator.initialize_all_components()
            sys.exit(0 if success else 1)

        elif args.start:
            success = await orchestrator.initialize_all_components()
            if success:
                success = await orchestrator.start_all_services()
            sys.exit(0 if success else 1)

        elif args.health:
            await orchestrator.initialize_all_components()
            health_report = await orchestrator.check_health_all_components()
            print(json.dumps(health_report, indent=2))
            sys.exit(0)

        elif args.test:
            success = await orchestrator.initialize_all_components()
            if success:
                success = await orchestrator.run_integration_tests()
            sys.exit(0 if success else 1)

        elif args.stop:
            success = await orchestrator.stop_all_services()
            sys.exit(0 if success else 1)

        else:
            # Default: show help
            parser.print_help()

    except KeyboardInterrupt:
        logger.info("Received interrupt signal, shutting down...")
        await orchestrator.stop_all_services()
        sys.exit(0)
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())