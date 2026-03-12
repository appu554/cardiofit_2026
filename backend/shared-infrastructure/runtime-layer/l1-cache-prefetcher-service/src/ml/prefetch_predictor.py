"""
ML-based Prefetch Predictor
Uses machine learning to predict and preload clinical data
"""

import asyncio
import numpy as np
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple, Set
from collections import defaultdict
import structlog
from dataclasses import dataclass
from sklearn.ensemble import RandomForestRegressor
from sklearn.preprocessing import StandardScaler
from sklearn.model_selection import train_test_split
import joblib
import threading

from ..models.cache_models import (
    AccessPattern,
    SessionContext,
    PrefetchPrediction,
    CacheKeyType
)

logger = structlog.get_logger()


@dataclass
class FeatureVector:
    """Feature vector for ML prediction"""
    # Temporal features
    hour_of_day: int
    day_of_week: int
    time_since_last_access: float  # hours

    # Access pattern features
    total_access_count: int
    access_frequency: float
    access_recency_score: float

    # Session features
    session_age_hours: float
    session_activity_level: float
    recent_access_count: int

    # User pattern features
    user_access_diversity: float
    user_session_count: int

    # Key type features (one-hot encoded)
    key_type_patient: int
    key_type_clinical: int
    key_type_medication: int
    key_type_guideline: int
    key_type_semantic: int
    key_type_workflow: int

    # Correlation features
    correlated_keys_accessed: int
    session_pattern_strength: float

    def to_array(self) -> np.ndarray:
        """Convert to numpy array for ML model"""
        return np.array([
            self.hour_of_day,
            self.day_of_week,
            self.time_since_last_access,
            self.total_access_count,
            self.access_frequency,
            self.access_recency_score,
            self.session_age_hours,
            self.session_activity_level,
            self.recent_access_count,
            self.user_access_diversity,
            self.user_session_count,
            self.key_type_patient,
            self.key_type_clinical,
            self.key_type_medication,
            self.key_type_guideline,
            self.key_type_semantic,
            self.key_type_workflow,
            self.correlated_keys_accessed,
            self.session_pattern_strength
        ])


class PrefetchPredictor:
    """
    ML-powered prefetch prediction engine

    Features:
    - Random Forest for access time prediction
    - Collaborative filtering for key correlation
    - Temporal pattern analysis
    - Session-aware predictions
    - Continuous learning from access patterns
    """

    def __init__(
        self,
        model_update_interval_hours: int = 6,
        min_training_samples: int = 1000,
        prediction_horizon_hours: int = 1
    ):
        self.model_update_interval = timedelta(hours=model_update_interval_hours)
        self.min_training_samples = min_training_samples
        self.prediction_horizon_seconds = prediction_horizon_hours * 3600

        # ML models
        self._access_time_model: Optional[RandomForestRegressor] = None
        self._feature_scaler = StandardScaler()
        self._model_lock = threading.RLock()

        # Training data storage
        self._training_data: List[Tuple[FeatureVector, float]] = []  # (features, time_to_access)
        self._last_model_update = datetime.utcnow() - self.model_update_interval

        # Key correlation matrix
        self._key_correlations: Dict[str, Dict[str, float]] = defaultdict(lambda: defaultdict(float))

        # Session pattern cache
        self._session_patterns: Dict[str, List[str]] = {}  # session_id -> access sequence

        # Background training task
        self._training_task: Optional[asyncio.Task] = None

        logger.info(
            "prefetch_predictor_initialized",
            update_interval_hours=model_update_interval_hours,
            prediction_horizon_hours=prediction_horizon_hours
        )

    async def predict_prefetch_candidates(
        self,
        access_patterns: Dict[str, AccessPattern],
        session_contexts: Dict[str, SessionContext],
        current_session_id: Optional[str] = None,
        max_candidates: int = 50
    ) -> List[PrefetchPrediction]:
        """
        Predict which keys should be prefetched based on ML models
        """
        try:
            current_time = datetime.utcnow()
            predictions = []

            # Get current session context
            current_session = None
            if current_session_id and current_session_id in session_contexts:
                current_session = session_contexts[current_session_id]

            # Generate predictions for all keys with sufficient history
            for key, pattern in access_patterns.items():
                if pattern.access_count < 3:  # Need minimum history
                    continue

                # Generate feature vector
                features = self._extract_features(
                    key,
                    pattern,
                    access_patterns,
                    session_contexts,
                    current_session
                )

                # Predict access probability and timing
                confidence, time_to_access = await self._predict_access(features)

                if confidence > 0.3 and time_to_access <= self.prediction_horizon_seconds:
                    # Determine trigger factors
                    trigger_factors = self._analyze_trigger_factors(
                        key,
                        pattern,
                        current_session,
                        features
                    )

                    prediction = PrefetchPrediction(
                        key=key,
                        key_type=pattern.key_type,
                        confidence=confidence,
                        predicted_access_time=current_time + timedelta(seconds=time_to_access),
                        time_to_access_seconds=int(time_to_access),
                        session_context=current_session.dict() if current_session else {},
                        trigger_factors=trigger_factors
                    )

                    predictions.append(prediction)

            # Sort by priority score and limit results
            predictions.sort(key=lambda x: x.priority_score, reverse=True)
            predictions = predictions[:max_candidates]

            logger.debug(
                "prefetch_predictions_generated",
                total_candidates=len(predictions),
                avg_confidence=np.mean([p.confidence for p in predictions]) if predictions else 0,
                session_id=current_session_id
            )

            return predictions

        except Exception as e:
            logger.error("prefetch_prediction_error", error=str(e))
            return []

    def record_access(
        self,
        key: str,
        access_time: datetime,
        pattern: AccessPattern,
        session_context: Optional[SessionContext] = None
    ):
        """
        Record an access event for model training
        """
        try:
            # Update session patterns
            if session_context:
                session_id = session_context.session_id
                if session_id not in self._session_patterns:
                    self._session_patterns[session_id] = []

                self._session_patterns[session_id].append(key)

                # Keep only recent accesses (last 100)
                if len(self._session_patterns[session_id]) > 100:
                    self._session_patterns[session_id] = self._session_patterns[session_id][-100:]

                # Update key correlations
                self._update_key_correlations(session_id, key)

            # If this was a predicted access, record the training sample
            prediction_time = access_time - timedelta(seconds=300)  # Look back 5 minutes
            # In a real implementation, we'd store predictions and match them with actual accesses

        except Exception as e:
            logger.error("access_recording_error", key=key, error=str(e))

    def _extract_features(
        self,
        key: str,
        pattern: AccessPattern,
        all_patterns: Dict[str, AccessPattern],
        session_contexts: Dict[str, SessionContext],
        current_session: Optional[SessionContext]
    ) -> FeatureVector:
        """Extract ML features from access pattern and context"""
        current_time = datetime.utcnow()

        # Temporal features
        hour_of_day = current_time.hour
        day_of_week = current_time.weekday()
        time_since_last_access = (current_time - pattern.last_accessed).total_seconds() / 3600

        # Access pattern features
        total_access_count = pattern.access_count
        access_frequency = pattern.access_frequency
        access_recency_score = self._calculate_recency_score(pattern.last_accessed)

        # Session features
        session_age_hours = 0.0
        session_activity_level = 0.0
        recent_access_count = 0

        if current_session:
            session_age_hours = (current_time - current_session.started_at).total_seconds() / 3600
            session_activity_level = len(current_session.access_pattern) / max(session_age_hours, 0.1)
            # Count recent accesses (last 10 minutes)
            recent_cutoff = current_time - timedelta(minutes=10)
            recent_access_count = len([
                t for t in current_session.access_pattern[-20:]  # Check last 20 accesses
                if True  # In real implementation, check timestamp
            ])

        # User pattern features (simplified)
        user_access_diversity = len(set(pattern.user_correlation.keys()))
        user_session_count = len(pattern.session_correlation)

        # Key type one-hot encoding
        key_type_features = {
            'key_type_patient': 1 if pattern.key_type == CacheKeyType.PATIENT_CONTEXT else 0,
            'key_type_clinical': 1 if pattern.key_type == CacheKeyType.CLINICAL_DATA else 0,
            'key_type_medication': 1 if pattern.key_type == CacheKeyType.MEDICATION_DATA else 0,
            'key_type_guideline': 1 if pattern.key_type == CacheKeyType.GUIDELINE_DATA else 0,
            'key_type_semantic': 1 if pattern.key_type == CacheKeyType.SEMANTIC_MESH else 0,
            'key_type_workflow': 1 if pattern.key_type == CacheKeyType.WORKFLOW_STATE else 0,
        }

        # Correlation features
        correlated_keys_accessed = 0
        session_pattern_strength = 0.0

        if current_session and key in self._key_correlations:
            correlations = self._key_correlations[key]
            recent_keys = set(current_session.access_pattern[-10:])  # Last 10 accesses
            correlated_keys_accessed = sum(
                1 for k in recent_keys
                if k in correlations and correlations[k] > 0.3
            )

            # Calculate pattern strength
            if len(current_session.access_pattern) > 5:
                session_pattern_strength = self._calculate_pattern_strength(
                    current_session.access_pattern[-20:]
                )

        return FeatureVector(
            hour_of_day=hour_of_day,
            day_of_week=day_of_week,
            time_since_last_access=time_since_last_access,
            total_access_count=total_access_count,
            access_frequency=access_frequency,
            access_recency_score=access_recency_score,
            session_age_hours=session_age_hours,
            session_activity_level=session_activity_level,
            recent_access_count=recent_access_count,
            user_access_diversity=user_access_diversity,
            user_session_count=user_session_count,
            correlated_keys_accessed=correlated_keys_accessed,
            session_pattern_strength=session_pattern_strength,
            **key_type_features
        )

    async def _predict_access(self, features: FeatureVector) -> Tuple[float, float]:
        """
        Predict access probability and time using ML model
        """
        try:
            with self._model_lock:
                if self._access_time_model is None:
                    # Use heuristic-based prediction if no model available
                    return self._heuristic_prediction(features)

                # Prepare features
                feature_array = features.to_array().reshape(1, -1)
                scaled_features = self._feature_scaler.transform(feature_array)

                # Predict time to access
                predicted_time = self._access_time_model.predict(scaled_features)[0]

                # Calculate confidence based on model uncertainty and feature strength
                confidence = self._calculate_confidence(features, predicted_time)

                return confidence, max(1.0, predicted_time)

        except Exception as e:
            logger.error("ml_prediction_error", error=str(e))
            return self._heuristic_prediction(features)

    def _heuristic_prediction(self, features: FeatureVector) -> Tuple[float, float]:
        """
        Fallback heuristic-based prediction when ML model unavailable
        """
        # Simple heuristic based on access frequency and recency
        base_confidence = min(0.8, features.access_frequency / 10.0)

        # Boost confidence for recent activity
        if features.time_since_last_access < 1.0:  # < 1 hour
            base_confidence *= 1.5

        # Boost for high session activity
        if features.session_activity_level > 5.0:
            base_confidence *= 1.2

        # Boost for correlated key access
        if features.correlated_keys_accessed > 0:
            base_confidence *= 1.3

        confidence = min(0.9, base_confidence)

        # Predict time based on historical frequency
        if features.access_frequency > 0:
            predicted_time = 3600.0 / features.access_frequency  # seconds
        else:
            predicted_time = 1800.0  # Default 30 minutes

        # Adjust for time since last access
        predicted_time = max(60.0, predicted_time - (features.time_since_last_access * 3600 * 0.1))

        return confidence, predicted_time

    def _calculate_recency_score(self, last_accessed: datetime) -> float:
        """Calculate recency score (0-1, higher is more recent)"""
        hours_since = (datetime.utcnow() - last_accessed).total_seconds() / 3600
        return max(0.0, 1.0 - (hours_since / 24.0))  # Decay over 24 hours

    def _calculate_confidence(self, features: FeatureVector, predicted_time: float) -> float:
        """Calculate prediction confidence based on features and model output"""
        base_confidence = 0.5

        # Boost for high access frequency
        if features.access_frequency > 1.0:
            base_confidence += 0.2

        # Boost for recent access
        if features.access_recency_score > 0.8:
            base_confidence += 0.2

        # Boost for session activity
        if features.session_activity_level > 3.0:
            base_confidence += 0.15

        # Boost for correlated accesses
        if features.correlated_keys_accessed > 0:
            base_confidence += 0.1

        # Reduce for very long predicted times
        if predicted_time > 1800:  # > 30 minutes
            base_confidence *= 0.8

        return min(0.95, max(0.1, base_confidence))

    def _update_key_correlations(self, session_id: str, accessed_key: str):
        """Update key correlation matrix"""
        if session_id not in self._session_patterns:
            return

        recent_keys = self._session_patterns[session_id][-10:]  # Last 10 accesses

        for other_key in recent_keys[:-1]:  # Exclude the current key
            if other_key != accessed_key:
                self._key_correlations[other_key][accessed_key] += 1
                self._key_correlations[accessed_key][other_key] += 1

    def _calculate_pattern_strength(self, access_sequence: List[str]) -> float:
        """Calculate the strength of access patterns in a sequence"""
        if len(access_sequence) < 3:
            return 0.0

        # Look for repeating patterns
        pattern_counts = defaultdict(int)

        for i in range(len(access_sequence) - 2):
            pattern = tuple(access_sequence[i:i+3])
            pattern_counts[pattern] += 1

        if not pattern_counts:
            return 0.0

        max_pattern_count = max(pattern_counts.values())
        return min(1.0, max_pattern_count / len(access_sequence))

    def _analyze_trigger_factors(
        self,
        key: str,
        pattern: AccessPattern,
        session_context: Optional[SessionContext],
        features: FeatureVector
    ) -> List[str]:
        """Analyze what factors triggered the prediction"""
        factors = []

        if features.access_frequency > 2.0:
            factors.append("high_frequency")

        if features.access_recency_score > 0.8:
            factors.append("recent_access")

        if features.session_activity_level > 5.0:
            factors.append("active_session")

        if features.correlated_keys_accessed > 0:
            factors.append("correlated_access")

        if features.session_pattern_strength > 0.5:
            factors.append("pattern_recognition")

        # Temporal factors
        if 9 <= features.hour_of_day <= 17:  # Business hours
            factors.append("business_hours")

        if pattern.temporal_pattern and len(pattern.temporal_pattern) == 24:
            current_hour_accesses = pattern.temporal_pattern[features.hour_of_day]
            avg_accesses = sum(pattern.temporal_pattern) / 24
            if current_hour_accesses > avg_accesses * 1.5:
                factors.append("temporal_pattern")

        return factors

    async def start_training_loop(self):
        """Start background model training loop"""
        if self._training_task is None:
            self._training_task = asyncio.create_task(self._training_loop())

    async def stop_training_loop(self):
        """Stop background model training"""
        if self._training_task:
            self._training_task.cancel()
            try:
                await self._training_task
            except asyncio.CancelledError:
                pass
            self._training_task = None

    async def _training_loop(self):
        """Background task for model training"""
        while True:
            try:
                # Check if model needs update
                if (datetime.utcnow() - self._last_model_update) >= self.model_update_interval:
                    if len(self._training_data) >= self.min_training_samples:
                        await self._train_model()
                        self._last_model_update = datetime.utcnow()

                # Wait before next check
                await asyncio.sleep(3600)  # Check every hour

            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("training_loop_error", error=str(e))
                await asyncio.sleep(3600)

    async def _train_model(self):
        """Train the access time prediction model"""
        try:
            logger.info("training_ml_model", samples=len(self._training_data))

            # Prepare training data
            X = np.array([sample[0].to_array() for sample in self._training_data])
            y = np.array([sample[1] for sample in self._training_data])

            # Split data
            X_train, X_test, y_train, y_test = train_test_split(
                X, y, test_size=0.2, random_state=42
            )

            # Scale features
            X_train_scaled = self._feature_scaler.fit_transform(X_train)
            X_test_scaled = self._feature_scaler.transform(X_test)

            # Train model
            with self._model_lock:
                self._access_time_model = RandomForestRegressor(
                    n_estimators=100,
                    max_depth=10,
                    random_state=42,
                    n_jobs=-1
                )
                self._access_time_model.fit(X_train_scaled, y_train)

            # Evaluate model
            train_score = self._access_time_model.score(X_train_scaled, y_train)
            test_score = self._access_time_model.score(X_test_scaled, y_test)

            logger.info(
                "ml_model_trained",
                train_score=train_score,
                test_score=test_score,
                training_samples=len(self._training_data)
            )

        except Exception as e:
            logger.error("model_training_error", error=str(e))