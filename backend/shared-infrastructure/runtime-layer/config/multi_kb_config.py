"""
Multi-KB Runtime Configuration
Unified configuration for shared runtime layer serving all CardioFit Knowledge Bases
"""

from typing import Dict, Any, List, Optional
from dataclasses import dataclass
from enum import Enum
import os
from pathlib import Path


class Environment(Enum):
    DEVELOPMENT = "development"
    STAGING = "staging"
    PRODUCTION = "production"


@dataclass
class KnowledgeBaseConfig:
    """Configuration for individual Knowledge Base"""
    kb_id: str
    name: str
    description: str
    neo4j_partition: str
    clickhouse_db: Optional[str]
    primary_storage: str
    search_index: Optional[str]
    has_analytics: bool
    has_semantic_mesh: bool


@dataclass
class DataStoreConfig:
    """Configuration for individual data store"""
    host: str
    port: int
    username: str
    password: str
    database: Optional[str] = None
    ssl: bool = False
    connection_pool_size: int = 10


class MultiKBRuntimeConfig:
    """
    Unified configuration for multi-KB runtime layer
    Manages all data stores and knowledge base configurations
    """

    def __init__(self, environment: Environment = Environment.DEVELOPMENT):
        self.environment = environment
        self.knowledge_bases = self._initialize_knowledge_bases()
        self.data_stores = self._initialize_data_stores()
        self.runtime_settings = self._initialize_runtime_settings()

    def _initialize_knowledge_bases(self) -> Dict[str, KnowledgeBaseConfig]:
        """Initialize all knowledge base configurations"""
        return {
            'kb-1': KnowledgeBaseConfig(
                kb_id='kb-1',
                name='Patient Data',
                description='Individual patient records and clinical data',
                neo4j_partition='KB1_PatientStream',
                clickhouse_db='kb1_patient_analytics',
                primary_storage='postgresql',
                search_index=None,
                has_analytics=True,
                has_semantic_mesh=True
            ),
            'kb-2': KnowledgeBaseConfig(
                kb_id='kb-2',
                name='Clinical Guidelines',
                description='Evidence-based clinical practice guidelines',
                neo4j_partition='KB2_GuidelineStream',
                clickhouse_db='kb2_guideline_analytics',
                primary_storage='postgresql',
                search_index='elasticsearch',
                has_analytics=True,
                has_semantic_mesh=True
            ),
            'kb-3': KnowledgeBaseConfig(
                kb_id='kb-3',
                name='Drug Calculations',
                description='Pharmaceutical dosing and calculation rules',
                neo4j_partition='KB3_DrugCalculationStream',
                clickhouse_db='kb3_drug_calculations',
                primary_storage='postgresql',
                search_index=None,
                has_analytics=True,
                has_semantic_mesh=False
            ),
            'kb-4': KnowledgeBaseConfig(
                kb_id='kb-4',
                name='Safety Rules',
                description='Clinical safety checks and contraindications',
                neo4j_partition='KB4_SafetyStream',
                clickhouse_db='kb4_safety_analytics',
                primary_storage='postgresql',
                search_index=None,
                has_analytics=True,
                has_semantic_mesh=True
            ),
            'kb-5': KnowledgeBaseConfig(
                kb_id='kb-5',
                name='Drug Interactions',
                description='Pharmaceutical interaction database',
                neo4j_partition='KB5_InteractionStream',
                clickhouse_db='kb5_interaction_analytics',
                primary_storage='neo4j',  # Primary in Neo4j for graph queries
                search_index=None,
                has_analytics=True,
                has_semantic_mesh=True
            ),
            'kb-6': KnowledgeBaseConfig(
                kb_id='kb-6',
                name='Evidence Base',
                description='Clinical evidence and research outcomes',
                neo4j_partition='KB6_EvidenceStream',
                clickhouse_db='kb6_evidence_analytics',
                primary_storage='postgresql',
                search_index='elasticsearch',
                has_analytics=True,
                has_semantic_mesh=True
            ),
            'kb-7': KnowledgeBaseConfig(
                kb_id='kb-7',
                name='Medical Terminology',
                description='Medical terminology and coding standards',
                neo4j_partition='KB7_TerminologyStream',
                clickhouse_db='kb7_terminology_analytics',
                primary_storage='postgresql',
                search_index='elasticsearch',
                has_analytics=True,
                has_semantic_mesh=True
            ),
            'kb-8': KnowledgeBaseConfig(
                kb_id='kb-8',
                name='Clinical Workflows',
                description='Clinical decision support workflows',
                neo4j_partition='KB8_WorkflowStream',
                clickhouse_db='kb8_workflow_analytics',
                primary_storage='postgresql',
                search_index=None,
                has_analytics=True,
                has_semantic_mesh=False
            )
        }

    def _initialize_data_stores(self) -> Dict[str, DataStoreConfig]:
        """Initialize all data store configurations"""

        # Environment-specific configurations
        if self.environment == Environment.DEVELOPMENT:
            return {
                'neo4j': DataStoreConfig(
                    host='localhost',
                    port=7687,
                    username='neo4j',
                    password=os.getenv('NEO4J_PASSWORD', 'kb7password'),
                    database='neo4j'
                ),
                'postgresql': DataStoreConfig(
                    host='localhost',
                    port=5432,
                    username=os.getenv('POSTGRES_USER', 'kb_user'),
                    password=os.getenv('POSTGRES_PASSWORD', 'kb_password'),
                    database='clinical_governance',
                    connection_pool_size=20
                ),
                'clickhouse': DataStoreConfig(
                    host='localhost',
                    port=9000,
                    username='default',
                    password=os.getenv('CLICKHOUSE_PASSWORD', ''),
                    connection_pool_size=15
                ),
                'elasticsearch': DataStoreConfig(
                    host='localhost',
                    port=9200,
                    username=os.getenv('ES_USERNAME', 'elastic'),
                    password=os.getenv('ES_PASSWORD', 'changeme')
                ),
                'redis_l2': DataStoreConfig(
                    host='localhost',
                    port=6379,
                    username='',
                    password=os.getenv('REDIS_PASSWORD', ''),
                    database='0'
                ),
                'redis_l3': DataStoreConfig(
                    host='localhost',
                    port=6379,
                    username='',
                    password=os.getenv('REDIS_PASSWORD', ''),
                    database='1'
                ),
                'kafka': DataStoreConfig(
                    host='localhost',
                    port=9092,
                    username='',
                    password=''
                ),
                'graphdb': DataStoreConfig(
                    host='localhost',
                    port=7200,
                    username=os.getenv('GRAPHDB_USERNAME', 'admin'),
                    password=os.getenv('GRAPHDB_PASSWORD', 'admin')
                )
            }

        elif self.environment == Environment.PRODUCTION:
            return {
                'neo4j': DataStoreConfig(
                    host=os.getenv('NEO4J_HOST', 'neo4j-cluster.internal'),
                    port=7687,
                    username='neo4j',
                    password=os.getenv('NEO4J_PASSWORD'),
                    ssl=True,
                    connection_pool_size=100
                ),
                'postgresql': DataStoreConfig(
                    host=os.getenv('POSTGRES_HOST', 'postgres-cluster.internal'),
                    port=5432,
                    username=os.getenv('POSTGRES_USER'),
                    password=os.getenv('POSTGRES_PASSWORD'),
                    database='clinical_governance',
                    ssl=True,
                    connection_pool_size=50
                ),
                'clickhouse': DataStoreConfig(
                    host=os.getenv('CLICKHOUSE_HOST', 'clickhouse-cluster.internal'),
                    port=9000,
                    username=os.getenv('CLICKHOUSE_USER', 'default'),
                    password=os.getenv('CLICKHOUSE_PASSWORD'),
                    ssl=True,
                    connection_pool_size=30
                ),
                # ... other production configs
            }

        else:  # STAGING
            # Similar to development but with different hosts
            return self._get_staging_config()

    def _initialize_runtime_settings(self) -> Dict[str, Any]:
        """Initialize runtime-specific settings"""
        return {
            'query_router': {
                'default_timeout_ms': 5000,
                'max_concurrent_queries': 100,
                'enable_query_caching': True,
                'cache_ttl_seconds': 300,
                'enable_cross_kb_queries': True,
                'max_cross_kb_scope': 4  # Maximum KBs in single cross-KB query
            },
            'neo4j': {
                'max_connection_pool_size': 100 if self.environment == Environment.PRODUCTION else 50,
                'connection_acquisition_timeout': 30,
                'enable_logical_partitioning': True,
                'enable_enterprise_features': self.environment == Environment.PRODUCTION
            },
            'clickhouse': {
                'compression_enabled': True,
                'max_query_size': 1000000,  # 1MB
                'query_timeout_seconds': 30,
                'enable_parallel_queries': True
            },
            'caching': {
                'l2_cache_ttl_seconds': 3600,   # 1 hour
                'l3_cache_ttl_seconds': 86400,  # 24 hours
                'max_cache_memory_mb': 2048,
                'enable_proactive_warming': True
            },
            'event_bus': {
                'kafka_batch_size': 16384,
                'kafka_linger_ms': 10,
                'kafka_compression': 'gzip',
                'enable_dead_letter_queue': True,
                'max_retry_attempts': 3
            },
            'monitoring': {
                'enable_metrics': True,
                'metrics_interval_seconds': 60,
                'enable_health_checks': True,
                'health_check_interval_seconds': 30,
                'enable_performance_logging': True
            }
        }

    def _get_staging_config(self) -> Dict[str, DataStoreConfig]:
        """Get staging environment configuration"""
        # Staging-specific configuration
        return {
            'neo4j': DataStoreConfig(
                host=os.getenv('NEO4J_HOST', 'neo4j-staging.internal'),
                port=7687,
                username='neo4j',
                password=os.getenv('NEO4J_PASSWORD', 'staging_password'),
                connection_pool_size=25
            ),
            # ... other staging configs
        }

    def get_kb_config(self, kb_id: str) -> Optional[KnowledgeBaseConfig]:
        """Get configuration for specific knowledge base"""
        return self.knowledge_bases.get(kb_id)

    def get_data_store_config(self, store_name: str) -> Optional[DataStoreConfig]:
        """Get configuration for specific data store"""
        return self.data_stores.get(store_name)

    def get_analytics_enabled_kbs(self) -> List[str]:
        """Get list of KBs with analytics enabled"""
        return [kb_id for kb_id, config in self.knowledge_bases.items() if config.has_analytics]

    def get_semantic_enabled_kbs(self) -> List[str]:
        """Get list of KBs with semantic mesh enabled"""
        return [kb_id for kb_id, config in self.knowledge_bases.items() if config.has_semantic_mesh]

    def get_elasticsearch_enabled_kbs(self) -> List[str]:
        """Get list of KBs with Elasticsearch search enabled"""
        return [kb_id for kb_id, config in self.knowledge_bases.items() if config.search_index == 'elasticsearch']

    def get_neo4j_uri(self) -> str:
        """Get Neo4j connection URI"""
        neo4j_config = self.data_stores['neo4j']
        protocol = 'neo4j+s' if neo4j_config.ssl else 'bolt'
        return f"{protocol}://{neo4j_config.host}:{neo4j_config.port}"

    def get_clickhouse_connection_params(self) -> Dict[str, Any]:
        """Get ClickHouse connection parameters"""
        ch_config = self.data_stores['clickhouse']
        return {
            'host': ch_config.host,
            'port': ch_config.port,
            'user': ch_config.username,
            'password': ch_config.password,
            'secure': ch_config.ssl,
            'compression': True
        }

    def get_kafka_brokers(self) -> List[str]:
        """Get Kafka broker list"""
        kafka_config = self.data_stores['kafka']
        return [f"{kafka_config.host}:{kafka_config.port}"]

    def get_redis_l2_url(self) -> str:
        """Get Redis L2 cache URL"""
        redis_config = self.data_stores['redis_l2']
        auth = f":{redis_config.password}@" if redis_config.password else ""
        return f"redis://{auth}{redis_config.host}:{redis_config.port}/{redis_config.database}"

    def get_redis_l3_url(self) -> str:
        """Get Redis L3 cache URL"""
        redis_config = self.data_stores['redis_l3']
        auth = f":{redis_config.password}@" if redis_config.password else ""
        return f"redis://{auth}{redis_config.host}:{redis_config.port}/{redis_config.database}"

    def to_dict(self) -> Dict[str, Any]:
        """Convert configuration to dictionary format"""
        return {
            'environment': self.environment.value,
            'knowledge_bases': {
                kb_id: {
                    'name': config.name,
                    'description': config.description,
                    'neo4j_partition': config.neo4j_partition,
                    'clickhouse_db': config.clickhouse_db,
                    'primary_storage': config.primary_storage,
                    'search_index': config.search_index,
                    'has_analytics': config.has_analytics,
                    'has_semantic_mesh': config.has_semantic_mesh
                }
                for kb_id, config in self.knowledge_bases.items()
            },
            'data_stores': {
                store_name: {
                    'host': config.host,
                    'port': config.port,
                    'username': config.username,
                    'database': config.database,
                    'ssl': config.ssl,
                    'connection_pool_size': config.connection_pool_size
                }
                for store_name, config in self.data_stores.items()
            },
            'runtime_settings': self.runtime_settings
        }

    @classmethod
    def from_env(cls) -> 'MultiKBRuntimeConfig':
        """Create configuration from environment variables"""
        env_name = os.getenv('CARDIOFIT_ENVIRONMENT', 'development')
        environment = Environment(env_name.lower())
        return cls(environment)

    @classmethod
    def from_file(cls, config_file: Path) -> 'MultiKBRuntimeConfig':
        """Load configuration from file"""
        # This would load from YAML/JSON file
        # Implementation depends on preferred config format
        pass

    def validate_configuration(self) -> List[str]:
        """Validate configuration and return list of issues"""
        issues = []

        # Check required environment variables for production
        if self.environment == Environment.PRODUCTION:
            required_vars = [
                'NEO4J_PASSWORD',
                'POSTGRES_USER',
                'POSTGRES_PASSWORD',
                'CLICKHOUSE_PASSWORD'
            ]
            for var in required_vars:
                if not os.getenv(var):
                    issues.append(f"Missing required environment variable: {var}")

        # Validate KB configurations
        for kb_id, config in self.knowledge_bases.items():
            if config.has_analytics and not config.clickhouse_db:
                issues.append(f"KB {kb_id} has analytics enabled but no ClickHouse database configured")

        # Validate data store configurations
        for store_name, config in self.data_stores.items():
            if not config.host:
                issues.append(f"Missing host configuration for {store_name}")
            if not config.port:
                issues.append(f"Missing port configuration for {store_name}")

        return issues


# Global configuration instance
runtime_config = MultiKBRuntimeConfig.from_env()


# Convenience functions for backward compatibility
def get_neo4j_config() -> Dict[str, Any]:
    """Get Neo4j configuration (backward compatibility)"""
    return {
        'neo4j_uri': runtime_config.get_neo4j_uri(),
        'neo4j_user': runtime_config.data_stores['neo4j'].username,
        'neo4j_password': runtime_config.data_stores['neo4j'].password
    }


def get_clickhouse_config() -> Dict[str, Any]:
    """Get ClickHouse configuration (backward compatibility)"""
    return runtime_config.get_clickhouse_connection_params()


def get_kafka_config() -> Dict[str, Any]:
    """Get Kafka configuration (backward compatibility)"""
    return {
        'kafka_brokers': runtime_config.get_kafka_brokers()
    }


def get_redis_config() -> Dict[str, Any]:
    """Get Redis configuration (backward compatibility)"""
    return {
        'redis_l2_url': runtime_config.get_redis_l2_url(),
        'redis_l3_url': runtime_config.get_redis_l3_url()
    }