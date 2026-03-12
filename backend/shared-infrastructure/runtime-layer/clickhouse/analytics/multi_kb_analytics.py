"""
Multi-KB ClickHouse Analytics Manager
Manages high-performance analytics across ALL CardioFit Knowledge Bases

Supports separate databases for each KB:
- kb1_patient_analytics: Patient data analytics
- kb2_guideline_analytics: Guideline effectiveness metrics
- kb3_drug_calculations: Drug calculation performance
- kb6_evidence_scores: Evidence-based scoring
- kb7_terminology_analytics: Terminology usage patterns

Provides unified interface for cross-KB analytics queries.
"""

from clickhouse_driver import Client
import pandas as pd
from typing import Dict, List, Any, Optional, Union
import hashlib
from datetime import datetime, timedelta
import json
from loguru import logger
from enum import Enum


class KBAnalyticsType(Enum):
    """Analytics types for different KBs"""
    PATIENT_ANALYTICS = "patient_analytics"
    GUIDELINE_ANALYTICS = "guideline_analytics"
    DRUG_CALCULATIONS = "drug_calculations"
    SAFETY_ANALYTICS = "safety_analytics"
    INTERACTION_ANALYTICS = "interaction_analytics"
    EVIDENCE_ANALYTICS = "evidence_analytics"
    TERMINOLOGY_ANALYTICS = "terminology_analytics"
    WORKFLOW_ANALYTICS = "workflow_analytics"


class MultiKBAnalyticsManager:
    """
    Manages ClickHouse analytics across all CardioFit Knowledge Bases
    Provides high-speed columnar storage and analytics for all KB types
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Multi-KB Analytics Manager

        Args:
            config: Configuration dictionary with ClickHouse connection details
        """
        self.config = config
        self.kb_databases = {
            'kb1': 'kb1_patient_analytics',
            'kb2': 'kb2_guideline_analytics',
            'kb3': 'kb3_drug_calculations',
            'kb4': 'kb4_safety_analytics',
            'kb5': 'kb5_interaction_analytics',
            'kb6': 'kb6_evidence_analytics',
            'kb7': 'kb7_terminology_analytics',
            'kb8': 'kb8_workflow_analytics'
        }

        # ClickHouse clients for each KB database
        self.clients = {}
        for kb_id, db_name in self.kb_databases.items():
            self.clients[kb_id] = Client(
                host=config.get('host', 'localhost'),
                port=config.get('port', 9000),
                database=db_name,
                user=config.get('user', 'default'),
                password=config.get('password', ''),
                secure=config.get('secure', False),
                verify=config.get('verify', True),
                compression=config.get('compression', True)
            )

        # Main client for cross-KB queries
        self.main_client = Client(
            host=config.get('host', 'localhost'),
            port=config.get('port', 9000),
            user=config.get('user', 'default'),
            password=config.get('password', ''),
            secure=config.get('secure', False),
            verify=config.get('verify', True),
            compression=config.get('compression', True)
        )

        logger.info(f"Multi-KB Analytics Manager initialized for {len(self.kb_databases)} KBs")
        self.initialize_all_kb_databases()

    def initialize_all_kb_databases(self) -> None:
        """Create all KB databases and tables"""
        for kb_id, db_name in self.kb_databases.items():
            try:
                # Create database
                self.main_client.execute(f"CREATE DATABASE IF NOT EXISTS {db_name}")
                logger.info(f"Database {db_name} initialized for {kb_id}")

                # Initialize KB-specific tables
                self._initialize_kb_tables(kb_id)

            except Exception as e:
                logger.error(f"Failed to create database {db_name}: {e}")
                raise

    def _initialize_kb_tables(self, kb_id: str) -> None:
        """Initialize tables for specific KB"""
        client = self.clients[kb_id]

        if kb_id == 'kb1':  # Patient Analytics
            self._create_patient_analytics_tables(client)
        elif kb_id == 'kb2':  # Guideline Analytics
            self._create_guideline_analytics_tables(client)
        elif kb_id == 'kb3':  # Drug Calculations
            self._create_drug_calculation_tables(client)
        elif kb_id == 'kb5':  # Drug Interactions
            self._create_interaction_analytics_tables(client)
        elif kb_id == 'kb6':  # Evidence Analytics
            self._create_evidence_analytics_tables(client)
        elif kb_id == 'kb7':  # Terminology Analytics
            self._create_terminology_analytics_tables(client)
        elif kb_id == 'kb8':  # Workflow Analytics
            self._create_workflow_analytics_tables(client)

        logger.info(f"Tables initialized for {kb_id}")

    def _create_patient_analytics_tables(self, client: Client) -> None:
        """Create KB1 patient analytics tables"""
        tables = [
            """
            CREATE TABLE IF NOT EXISTS patient_metrics (
                patient_id String,
                encounter_id String,
                encounter_date Date,
                encounter_type String,

                -- Demographics
                age UInt8,
                gender String,

                -- Clinical metrics
                diagnosis_count UInt16,
                medication_count UInt16,
                lab_result_count UInt32,

                -- Outcomes
                readmission_30_day UInt8,
                length_of_stay UInt16,

                -- System metrics
                kb_version String,
                calculated_at DateTime DEFAULT now()
            ) ENGINE = MergeTree()
            PARTITION BY toYYYYMM(encounter_date)
            ORDER BY (patient_id, encounter_date)
            """
        ]

        for table_sql in tables:
            client.execute(table_sql)

    def _create_guideline_analytics_tables(self, client: Client) -> None:
        """Create KB2 guideline analytics tables"""
        tables = [
            """
            CREATE TABLE IF NOT EXISTS guideline_adherence (
                guideline_id String,
                guideline_name String,
                patient_population String,

                -- Adherence metrics
                total_eligible_patients UInt32,
                adherent_patients UInt32,
                adherence_rate Float32,

                -- Outcomes
                improved_outcomes UInt32,
                adverse_events UInt32,

                -- Time period
                measurement_period_start Date,
                measurement_period_end Date,

                calculated_at DateTime DEFAULT now()
            ) ENGINE = MergeTree()
            PARTITION BY toYYYYMM(measurement_period_start)
            ORDER BY (guideline_id, measurement_period_start)
            """
        ]

        for table_sql in tables:
            client.execute(table_sql)

    def _create_drug_calculation_tables(self, client: Client) -> None:
        """Create KB3 drug calculation tables"""
        tables = [
            """
            CREATE TABLE IF NOT EXISTS drug_calculation_metrics (
                drug_rxnorm String,
                drug_name String,
                calculation_type String,

                -- Patient factors
                patient_age_group String,
                patient_weight_range String,
                kidney_function_category String,

                -- Calculation results
                recommended_dose Float32,
                dose_unit String,
                frequency String,

                -- Safety metrics
                min_dose Float32,
                max_dose Float32,
                safety_margin Float32,

                -- Performance
                calculation_time_ms Float32,

                calculated_at DateTime DEFAULT now()
            ) ENGINE = MergeTree()
            PARTITION BY toYYYYMM(calculated_at)
            ORDER BY (drug_rxnorm, calculation_type, calculated_at)
            """
        ]

        for table_sql in tables:
            client.execute(table_sql)

    def _create_interaction_analytics_tables(self, client: Client) -> None:
        """Create KB5 interaction analytics tables"""
        tables = [
            """
            CREATE TABLE IF NOT EXISTS interaction_analytics (
                drug1_rxnorm String,
                drug1_name String,
                drug2_rxnorm String,
                drug2_name String,

                -- Interaction details
                interaction_type String,
                severity_level String,
                mechanism String,

                -- Clinical impact
                frequency_reported UInt32,
                clinical_significance Float32,
                evidence_level String,

                -- Patient outcomes
                adverse_events_reported UInt32,
                hospitalizations UInt32,

                -- System metrics
                query_count UInt32,
                last_accessed DateTime,

                calculated_at DateTime DEFAULT now()
            ) ENGINE = MergeTree()
            PARTITION BY toYYYYMM(calculated_at)
            ORDER BY (drug1_rxnorm, drug2_rxnorm, calculated_at)
            """
        ]

        for table_sql in tables:
            client.execute(table_sql)

    def _create_evidence_analytics_tables(self, client: Client) -> None:
        """Create KB6 evidence analytics tables"""
        tables = [
            """
            CREATE TABLE IF NOT EXISTS evidence_scores (
                intervention_id String,
                intervention_name String,
                condition_code String,
                condition_name String,

                -- Evidence strength
                study_count UInt16,
                total_participants UInt32,
                evidence_quality String,

                -- Effectiveness scores
                efficacy_score Float32,
                safety_score Float32,
                patient_preference_score Float32,
                cost_effectiveness_score Float32,
                composite_evidence_score Float32,

                -- Meta-analysis results
                effect_size Float32,
                confidence_interval_lower Float32,
                confidence_interval_upper Float32,
                p_value Float32,

                -- System metadata
                evidence_version String,
                last_updated DateTime,

                calculated_at DateTime DEFAULT now()
            ) ENGINE = MergeTree()
            PARTITION BY toYYYYMM(calculated_at)
            ORDER BY (intervention_id, condition_code, calculated_at)
            """
        ]

        for table_sql in tables:
            client.execute(table_sql)

    def _create_terminology_analytics_tables(self, client: Client) -> None:
        """Create KB7 terminology analytics tables"""
        tables = [
            """
            CREATE TABLE IF NOT EXISTS terminology_usage (
                concept_code String,
                concept_name String,
                coding_system String,

                -- Usage metrics
                lookup_count UInt32,
                search_count UInt32,
                validation_count UInt32,

                -- System performance
                avg_lookup_time_ms Float32,
                cache_hit_rate Float32,

                -- Clinical usage
                used_in_diagnoses UInt32,
                used_in_medications UInt32,
                used_in_procedures UInt32,

                -- Quality metrics
                mapping_accuracy Float32,
                synonym_coverage Float32,

                -- Time period
                usage_date Date,

                calculated_at DateTime DEFAULT now()
            ) ENGINE = MergeTree()
            PARTITION BY toYYYYMM(usage_date)
            ORDER BY (concept_code, coding_system, usage_date)
            """
        ]

        for table_sql in tables:
            client.execute(table_sql)

    def _create_workflow_analytics_tables(self, client: Client) -> None:
        """Create KB8 workflow analytics tables"""
        tables = [
            """
            CREATE TABLE IF NOT EXISTS workflow_performance (
                workflow_id String,
                workflow_name String,
                workflow_type String,

                -- Execution metrics
                total_executions UInt32,
                successful_executions UInt32,
                failed_executions UInt32,
                success_rate Float32,

                -- Performance metrics
                avg_execution_time_ms Float32,
                min_execution_time_ms Float32,
                max_execution_time_ms Float32,

                -- Resource usage
                avg_memory_usage_mb Float32,
                avg_cpu_usage_percent Float32,

                -- Clinical impact
                patients_affected UInt32,
                clinical_decisions_supported UInt32,

                -- Time period
                measurement_date Date,

                calculated_at DateTime DEFAULT now()
            ) ENGINE = MergeTree()
            PARTITION BY toYYYYMM(measurement_date)
            ORDER BY (workflow_id, measurement_date)
            """
        ]

        for table_sql in tables:
            client.execute(table_sql)

    async def execute_kb_query(self, kb_id: str, query: str, params: Optional[Dict[str, Any]] = None) -> pd.DataFrame:
        """
        Execute analytics query on specific KB

        Args:
            kb_id: Knowledge Base identifier
            query: SQL query
            params: Query parameters

        Returns:
            Query results as DataFrame
        """
        if kb_id not in self.clients:
            raise ValueError(f"Unknown KB ID: {kb_id}")

        try:
            client = self.clients[kb_id]

            # Execute query
            if params:
                result = client.execute(query, params)
            else:
                result = client.execute(query)

            # Convert to DataFrame
            if result:
                columns = [col[0] for col in client.execute("DESCRIBE TABLE " + query.split()[3])][:len(result[0])]
                df = pd.DataFrame(result, columns=columns)
            else:
                df = pd.DataFrame()

            logger.info(f"Executed KB{kb_id} analytics query, returned {len(df)} rows")
            return df

        except Exception as e:
            logger.error(f"Failed to execute KB{kb_id} query: {e}")
            return pd.DataFrame()

    async def execute_cross_kb_query(self, query: str, params: Optional[Dict[str, Any]] = None) -> pd.DataFrame:
        """
        Execute analytics query across multiple KBs

        Args:
            query: SQL query with database references (e.g., kb1_patient_analytics.table)
            params: Query parameters

        Returns:
            Combined query results as DataFrame
        """
        try:
            # Execute cross-KB query
            if params:
                result = self.main_client.execute(query, params)
            else:
                result = self.main_client.execute(query)

            # Convert to DataFrame
            if result:
                # Extract column names from first row or use generic names
                columns = [f"col_{i}" for i in range(len(result[0]))] if result else []
                df = pd.DataFrame(result, columns=columns)
            else:
                df = pd.DataFrame()

            logger.info(f"Executed cross-KB analytics query, returned {len(df)} rows")
            return df

        except Exception as e:
            logger.error(f"Failed to execute cross-KB query: {e}")
            return pd.DataFrame()

    async def get_kb_performance_metrics(self, kb_id: str, days: int = 7) -> Dict[str, Any]:
        """
        Get performance metrics for specific KB

        Args:
            kb_id: Knowledge Base identifier
            days: Number of days to analyze

        Returns:
            Performance metrics
        """
        if kb_id not in self.clients:
            return {'error': f'Unknown KB ID: {kb_id}'}

        try:
            client = self.clients[kb_id]
            start_date = (datetime.now() - timedelta(days=days)).strftime('%Y-%m-%d')

            # Get table stats
            tables_query = f"SHOW TABLES FROM {self.kb_databases[kb_id]}"
            tables = client.execute(tables_query)

            metrics = {
                'kb_id': kb_id,
                'database': self.kb_databases[kb_id],
                'analysis_period_days': days,
                'tables': [],
                'total_rows': 0,
                'total_size_bytes': 0
            }

            for table in tables:
                table_name = table[0]
                # Get table statistics
                count_query = f"SELECT count(*) FROM {table_name}"
                row_count = client.execute(count_query)[0][0]

                size_query = f"""
                    SELECT sum(bytes_on_disk) as size_bytes
                    FROM system.parts
                    WHERE database = '{self.kb_databases[kb_id]}'
                    AND table = '{table_name}'
                """
                size_result = client.execute(size_query)
                size_bytes = size_result[0][0] if size_result[0][0] else 0

                metrics['tables'].append({
                    'name': table_name,
                    'rows': row_count,
                    'size_bytes': size_bytes
                })

                metrics['total_rows'] += row_count
                metrics['total_size_bytes'] += size_bytes

            logger.info(f"Generated performance metrics for KB{kb_id}")
            return metrics

        except Exception as e:
            logger.error(f"Failed to get KB{kb_id} performance metrics: {e}")
            return {'error': str(e)}

    async def health_check_all_kbs(self) -> Dict[str, bool]:
        """
        Check health of all KB analytics databases

        Returns:
            Health status for each KB
        """
        health_status = {}

        for kb_id, client in self.clients.items():
            try:
                # Simple connectivity test
                client.execute("SELECT 1")
                health_status[kb_id] = True
                logger.debug(f"KB{kb_id} analytics: healthy")
            except Exception as e:
                health_status[kb_id] = False
                logger.error(f"KB{kb_id} analytics health check failed: {e}")

        overall_health = all(health_status.values())
        logger.info(f"Multi-KB analytics health check: {sum(health_status.values())}/{len(health_status)} healthy")

        return {
            'overall_healthy': overall_health,
            'kb_status': health_status,
            'timestamp': datetime.utcnow().isoformat()
        }

    def close_all_connections(self) -> None:
        """Close all ClickHouse connections"""
        for kb_id, client in self.clients.items():
            try:
                client.disconnect()
                logger.debug(f"Closed KB{kb_id} analytics connection")
            except Exception as e:
                logger.warning(f"Error closing KB{kb_id} connection: {e}")

        try:
            self.main_client.disconnect()
            logger.info("Closed main ClickHouse connection")
        except Exception as e:
            logger.warning(f"Error closing main connection: {e}")


# Backward compatibility for KB7-specific usage
class ClickHouseRuntimeManager(MultiKBAnalyticsManager):
    """
    Backward compatibility wrapper for KB7-specific usage
    Maps old single-KB interface to new multi-KB system
    """

    def __init__(self, config: Dict[str, Any]):
        # Adapt config for multi-KB format
        multi_kb_config = {
            'host': config.get('host', 'localhost'),
            'port': config.get('port', 9000),
            'user': config.get('user', 'default'),
            'password': config.get('password', ''),
            'secure': config.get('secure', False),
            'verify': config.get('verify', True),
            'compression': config.get('compression', True)
        }
        super().__init__(multi_kb_config)
        self.kb_id = 'kb7'  # Default to KB7
        self.database = 'kb7_terminology_analytics'

    def initialize_database(self) -> None:
        """Legacy method - maps to new multi-KB initialization"""
        self._initialize_kb_tables('kb7')

    def initialize_tables(self) -> None:
        """Legacy method - maps to new table initialization"""
        self._initialize_kb_tables('kb7')

    async def calculate_medication_scores(self, drugs: List[str], indication: str,
                                        snapshot_id: Optional[str] = None,
                                        patient_context: Optional[Dict[str, Any]] = None,
                                        scoring_weights: Optional[Dict[str, float]] = None) -> pd.DataFrame:
        """Legacy method for KB7 medication scoring"""
        query = """
            SELECT
                drug_rxnorm,
                drug_name,
                indication_code,
                composite_score,
                safety_score,
                efficacy_score
            FROM medication_scores
            WHERE drug_rxnorm IN %(drugs)s
            AND indication_code = %(indication)s
            ORDER BY composite_score DESC
        """

        params = {
            'drugs': drugs,
            'indication': indication
        }

        return await self.execute_kb_query('kb7', query, params)