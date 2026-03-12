"""
ClickHouse Runtime Manager for KB7 Terminology Service
Manages high-performance analytics, scoring, and time-series data
"""

from clickhouse_driver import Client
import pandas as pd
from typing import Dict, List, Any, Optional
import hashlib
from datetime import datetime, timedelta
import json
from loguru import logger


class ClickHouseRuntimeManager:
    """
    Manages ClickHouse for analytics, scoring, and performance metrics
    Provides high-speed columnar storage for medication scoring and safety analytics
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize ClickHouse Runtime Manager

        Args:
            config: Configuration dictionary with ClickHouse connection details
        """
        self.client = Client(
            host=config.get('host', 'localhost'),
            port=config.get('port', 9000),
            database=config.get('database', 'kb7_analytics'),
            user=config.get('user', 'default'),
            password=config.get('password', ''),
            secure=config.get('secure', False),
            verify=config.get('verify', True),
            compression=config.get('compression', True)
        )

        self.config = config
        self.database = config.get('database', 'kb7_analytics')

        logger.info(f"ClickHouse Runtime Manager initialized for database: {self.database}")
        self.initialize_database()
        self.initialize_tables()

    def initialize_database(self) -> None:
        """Create database if it doesn't exist"""
        try:
            self.client.execute(f"CREATE DATABASE IF NOT EXISTS {self.database}")
            logger.info(f"Database {self.database} initialized")
        except Exception as e:
            logger.error(f"Failed to create database: {e}")
            raise

    def initialize_tables(self) -> None:
        """Create tables for service-specific analytics"""

        try:
            # Medication scoring table
            self.client.execute(f"""
                CREATE TABLE IF NOT EXISTS {self.database}.medication_scores (
                    drug_rxnorm String,
                    drug_name String,
                    indication_code String,
                    indication_name String,
                    guideline_score Float32,
                    formulary_tier UInt8,
                    safety_score Float32,
                    efficacy_score Float32,
                    cost_score Float32,
                    patient_preference_score Float32,
                    composite_score Float32,

                    -- Metadata
                    snapshot_id String,
                    kb_version String,
                    calculated_at DateTime DEFAULT now(),
                    calculation_metadata String,

                    INDEX idx_indication (indication_code) TYPE minmax GRANULARITY 4,
                    INDEX idx_snapshot (snapshot_id) TYPE bloom_filter GRANULARITY 1,
                    INDEX idx_composite (composite_score) TYPE minmax GRANULARITY 8
                ) ENGINE = ReplacingMergeTree(calculated_at)
                PARTITION BY toYYYYMM(calculated_at)
                ORDER BY (indication_code, drug_rxnorm, snapshot_id)
                TTL calculated_at + INTERVAL 90 DAY
            """)
            logger.info("Medication scores table created")

            # Safety analytics table
            self.client.execute(f"""
                CREATE TABLE IF NOT EXISTS {self.database}.safety_analytics (
                    patient_id String,
                    encounter_id String,
                    drug_combination Array(String),
                    drug_names Array(String),
                    risk_score Float32,
                    interaction_count UInt32,
                    interaction_details String,  -- JSON
                    contraindication_flags Array(String),
                    allergy_flags Array(String),
                    renal_adjustment_needed Bool,
                    hepatic_adjustment_needed Bool,

                    -- Risk breakdown
                    interaction_risk Float32,
                    contraindication_risk Float32,
                    allergy_risk Float32,
                    organ_function_risk Float32,
                    polypharmacy_risk Float32,

                    snapshot_id String,
                    evaluated_at DateTime DEFAULT now(),

                    INDEX idx_patient (patient_id) TYPE bloom_filter GRANULARITY 1,
                    INDEX idx_risk (risk_score) TYPE minmax GRANULARITY 8
                ) ENGINE = MergeTree()
                PARTITION BY toYYYYMM(evaluated_at)
                ORDER BY (patient_id, evaluated_at)
                TTL evaluated_at + INTERVAL 180 DAY
            """)
            logger.info("Safety analytics table created")

            # Clinical guideline compliance table
            self.client.execute(f"""
                CREATE TABLE IF NOT EXISTS {self.database}.guideline_compliance (
                    guideline_id String,
                    guideline_name String,
                    indication_code String,
                    drug_rxnorm String,
                    compliance_score Float32,
                    recommendation_strength String,  -- 'strong', 'moderate', 'weak'
                    evidence_level String,  -- 'A', 'B', 'C', 'D'
                    contraindications_checked Bool,
                    dosing_appropriate Bool,
                    monitoring_required Bool,
                    monitoring_parameters Array(String),

                    updated_at DateTime DEFAULT now(),
                    source_kb String,

                    INDEX idx_guideline (guideline_id) TYPE bloom_filter GRANULARITY 1,
                    INDEX idx_drug (drug_rxnorm) TYPE bloom_filter GRANULARITY 1
                ) ENGINE = ReplacingMergeTree(updated_at)
                ORDER BY (guideline_id, indication_code, drug_rxnorm)
                TTL updated_at + INTERVAL 365 DAY
            """)
            logger.info("Guideline compliance table created")

            # Performance metrics table
            self.client.execute(f"""
                CREATE TABLE IF NOT EXISTS {self.database}.performance_metrics (
                    query_type String,
                    data_source String,  -- 'postgres', 'elasticsearch', 'neo4j', 'clickhouse'
                    query_pattern String,
                    response_time_ms Float32,
                    rows_returned UInt32,
                    cache_hit Bool,
                    snapshot_id String,
                    service_id String,
                    timestamp DateTime DEFAULT now(),

                    INDEX idx_query_type (query_type) TYPE set(100) GRANULARITY 4,
                    INDEX idx_source (data_source) TYPE set(10) GRANULARITY 4
                ) ENGINE = MergeTree()
                PARTITION BY toYYYYMMDD(timestamp)
                ORDER BY (timestamp, query_type, data_source)
                TTL timestamp + INTERVAL 30 DAY
            """)
            logger.info("Performance metrics table created")

            # Terminology usage statistics
            self.client.execute(f"""
                CREATE TABLE IF NOT EXISTS {self.database}.terminology_usage (
                    terminology_system String,  -- 'SNOMED', 'RxNorm', 'LOINC', 'ICD10'
                    concept_code String,
                    concept_name String,
                    usage_count UInt64,
                    last_accessed DateTime,
                    service_contexts Array(String),
                    query_contexts Array(String),

                    INDEX idx_system (terminology_system) TYPE set(10) GRANULARITY 1,
                    INDEX idx_usage (usage_count) TYPE minmax GRANULARITY 8
                ) ENGINE = SummingMergeTree()
                PARTITION BY terminology_system
                ORDER BY (terminology_system, concept_code)
            """)
            logger.info("Terminology usage statistics table created")

        except Exception as e:
            logger.error(f"Failed to create tables: {e}")
            raise

    async def calculate_medication_scores(self, drugs: List[str], indication: str,
                                         patient_context: Optional[Dict] = None,
                                         snapshot_id: Optional[str] = None) -> pd.DataFrame:
        """
        Calculate composite scores for medications based on multiple factors

        Args:
            drugs: List of RxNorm drug codes
            indication: Indication code (ICD10/SNOMED)
            patient_context: Optional patient-specific factors
            snapshot_id: Optional snapshot ID for consistency

        Returns:
            DataFrame with scored medications
        """
        snapshot_id = snapshot_id or self._generate_snapshot_id()

        # Calculate composite scores with weighted factors
        weights = {
            'guideline': 0.3,
            'safety': 0.25,
            'efficacy': 0.2,
            'cost': 0.15,
            'patient_preference': 0.1
        }

        # Insert scoring data (in real implementation, scores would come from various sources)
        for drug in drugs:
            scoring_data = self._calculate_drug_scores(drug, indication, patient_context)

            composite_score = sum(
                scoring_data.get(f"{factor}_score", 0) * weight
                for factor, weight in weights.items()
            )

            self.client.execute(f"""
                INSERT INTO {self.database}.medication_scores
                (drug_rxnorm, indication_code, guideline_score, safety_score,
                 efficacy_score, cost_score, patient_preference_score,
                 composite_score, snapshot_id, kb_version)
                VALUES
            """, [{
                'drug_rxnorm': drug,
                'indication_code': indication,
                'guideline_score': scoring_data.get('guideline_score', 0),
                'safety_score': scoring_data.get('safety_score', 0),
                'efficacy_score': scoring_data.get('efficacy_score', 0),
                'cost_score': scoring_data.get('cost_score', 0),
                'patient_preference_score': scoring_data.get('patient_preference_score', 0),
                'composite_score': composite_score,
                'snapshot_id': snapshot_id,
                'kb_version': 'v1.0.0'
            }])

        # Query and return sorted results
        query = f"""
        SELECT
            drug_rxnorm,
            drug_name,
            composite_score,
            guideline_score,
            safety_score,
            efficacy_score,
            cost_score,
            patient_preference_score,
            formulary_tier
        FROM {self.database}.medication_scores
        WHERE
            drug_rxnorm IN %(drugs)s
            AND indication_code = %(indication)s
            AND snapshot_id = %(snapshot_id)s
        ORDER BY composite_score DESC
        """

        result = self.client.query_dataframe(
            query,
            parameters={
                'drugs': drugs,
                'indication': indication,
                'snapshot_id': snapshot_id
            }
        )

        return result

    async def calculate_safety_analytics(self, patient_id: str,
                                        medications: List[str],
                                        conditions: List[str]) -> Dict[str, Any]:
        """
        Calculate comprehensive safety analytics for a patient's medication regimen

        Args:
            patient_id: Patient identifier
            medications: List of medication RxNorm codes
            conditions: List of condition codes

        Returns:
            Safety analysis results
        """
        timestamp = datetime.utcnow()

        # Calculate various risk scores
        interaction_risk = await self._calculate_interaction_risk(medications)
        contraindication_risk = await self._calculate_contraindication_risk(
            medications, conditions
        )
        polypharmacy_risk = self._calculate_polypharmacy_risk(medications)

        # Calculate composite risk score
        risk_score = (
            interaction_risk * 0.4 +
            contraindication_risk * 0.3 +
            polypharmacy_risk * 0.3
        )

        # Store analytics
        self.client.execute(f"""
            INSERT INTO {self.database}.safety_analytics
            (patient_id, drug_combination, risk_score, interaction_risk,
             contraindication_risk, polypharmacy_risk, evaluated_at)
            VALUES
        """, [{
            'patient_id': patient_id,
            'drug_combination': medications,
            'risk_score': risk_score,
            'interaction_risk': interaction_risk,
            'contraindication_risk': contraindication_risk,
            'polypharmacy_risk': polypharmacy_risk,
            'evaluated_at': timestamp
        }])

        return {
            'patient_id': patient_id,
            'risk_score': risk_score,
            'risk_level': self._categorize_risk(risk_score),
            'components': {
                'interaction_risk': interaction_risk,
                'contraindication_risk': contraindication_risk,
                'polypharmacy_risk': polypharmacy_risk
            },
            'recommendations': self._generate_safety_recommendations(risk_score),
            'evaluated_at': timestamp.isoformat()
        }

    def record_performance_metric(self, query_type: str, data_source: str,
                                 response_time_ms: float, rows_returned: int,
                                 cache_hit: bool = False) -> None:
        """
        Record performance metrics for query routing optimization

        Args:
            query_type: Type of query executed
            data_source: Data source used
            response_time_ms: Response time in milliseconds
            rows_returned: Number of rows returned
            cache_hit: Whether cache was hit
        """
        try:
            self.client.execute(f"""
                INSERT INTO {self.database}.performance_metrics
                (query_type, data_source, response_time_ms, rows_returned, cache_hit)
                VALUES
            """, [{
                'query_type': query_type,
                'data_source': data_source,
                'response_time_ms': response_time_ms,
                'rows_returned': rows_returned,
                'cache_hit': cache_hit
            }])
        except Exception as e:
            logger.warning(f"Failed to record performance metric: {e}")

    def get_query_performance_stats(self, hours: int = 24) -> pd.DataFrame:
        """
        Get query performance statistics for optimization

        Args:
            hours: Number of hours to look back

        Returns:
            DataFrame with performance statistics
        """
        query = f"""
        SELECT
            query_type,
            data_source,
            count() as query_count,
            avg(response_time_ms) as avg_response_time,
            percentile(0.95)(response_time_ms) as p95_response_time,
            percentile(0.99)(response_time_ms) as p99_response_time,
            sum(cache_hit) / count() as cache_hit_rate
        FROM {self.database}.performance_metrics
        WHERE timestamp > now() - INTERVAL {hours} HOUR
        GROUP BY query_type, data_source
        ORDER BY query_count DESC
        """

        return self.client.query_dataframe(query)

    def _calculate_drug_scores(self, drug: str, indication: str,
                              patient_context: Optional[Dict]) -> Dict[str, float]:
        """
        Calculate individual scoring components for a drug
        In production, these would come from various knowledge bases
        """
        # Simplified scoring logic for demonstration
        base_scores = {
            'guideline_score': 0.8,
            'safety_score': 0.75,
            'efficacy_score': 0.85,
            'cost_score': 0.6,
            'patient_preference_score': 0.7
        }

        # Adjust based on patient context if provided
        if patient_context:
            if patient_context.get('elderly'):
                base_scores['safety_score'] *= 0.9
            if patient_context.get('renal_impairment'):
                base_scores['safety_score'] *= 0.85

        return base_scores

    async def _calculate_interaction_risk(self, medications: List[str]) -> float:
        """Calculate drug-drug interaction risk score"""
        # Simplified calculation - in production would query interaction database
        if len(medications) <= 1:
            return 0.0
        elif len(medications) <= 3:
            return 0.2
        elif len(medications) <= 5:
            return 0.4
        else:
            return 0.6

    async def _calculate_contraindication_risk(self, medications: List[str],
                                              conditions: List[str]) -> float:
        """Calculate contraindication risk score"""
        # Simplified calculation - in production would check actual contraindications
        if not conditions:
            return 0.0
        return min(0.1 * len(conditions), 0.8)

    def _calculate_polypharmacy_risk(self, medications: List[str]) -> float:
        """Calculate polypharmacy risk based on medication count"""
        med_count = len(medications)
        if med_count < 5:
            return 0.0
        elif med_count < 10:
            return 0.3
        else:
            return 0.6

    def _categorize_risk(self, risk_score: float) -> str:
        """Categorize risk score into levels"""
        if risk_score < 0.3:
            return 'low'
        elif risk_score < 0.6:
            return 'moderate'
        else:
            return 'high'

    def _generate_safety_recommendations(self, risk_score: float) -> List[str]:
        """Generate safety recommendations based on risk score"""
        recommendations = []

        if risk_score >= 0.6:
            recommendations.append("Consider medication review by clinical pharmacist")
            recommendations.append("Increase monitoring frequency")
        elif risk_score >= 0.3:
            recommendations.append("Review for potential drug interactions")
            recommendations.append("Standard monitoring recommended")

        return recommendations

    def _generate_snapshot_id(self) -> str:
        """Generate unique snapshot ID"""
        timestamp = datetime.utcnow().isoformat()
        return hashlib.md5(timestamp.encode()).hexdigest()[:16]

    def health_check(self) -> Dict[str, Any]:
        """
        Check health of ClickHouse connection

        Returns:
            Health status dictionary
        """
        try:
            result = self.client.execute("SELECT 1")
            return {
                'status': 'healthy',
                'database': self.database,
                'timestamp': datetime.utcnow().isoformat()
            }
        except Exception as e:
            return {
                'status': 'unhealthy',
                'error': str(e),
                'timestamp': datetime.utcnow().isoformat()
            }

    def close(self) -> None:
        """Close ClickHouse connection"""
        if hasattr(self.client, 'disconnect'):
            self.client.disconnect()
        logger.info("ClickHouse connection closed")