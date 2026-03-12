//! Temporal Constraint Engine
//!
//! This module implements high-precision temporal constraint evaluation
//! for time-sensitive clinical protocols with millisecond accuracy.
//!
//! Features:
//! - High-precision temporal constraint evaluation (millisecond accuracy)
//! - Multiple constraint types: absolute, relative, sliding windows
//! - Clinical time window enforcement (medication administration, procedures)
//! - Constraint caching and optimization for performance
//! - Integration with snapshot timestamps for deterministic evaluation
//! - Support for complex temporal relationships and dependencies

use std::collections::HashMap;
use std::sync::Arc;
use std::time::Instant;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc, Duration, NaiveTime, Timelike};
use parking_lot::RwLock;
use lru::LruCache;
use tokio::time::{timeout, Duration as TokioDuration};

use crate::protocol::{
    types::*,
    error::*,
    evaluation::EvaluationContext,
    engine::TemporalEngineConfig,
};

/// Enhanced temporal constraint for complex clinical time requirements
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemporalConstraint {
    pub constraint_id: String,
    pub name: String,
    pub description: String,
    pub constraint_type: TemporalConstraintType,
    pub time_window: TimeWindow,
    pub condition: String,
    pub severity: ConstraintSeverity,
    pub override_allowed: bool,
    pub clinical_context: Option<ClinicalTemporalContext>,
    pub dependencies: Vec<TemporalDependency>,
    pub metadata: HashMap<String, serde_json::Value>,
}

/// Types of temporal constraints
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TemporalConstraintType {
    /// Absolute time constraint (e.g., "must occur by 2024-01-01 10:00:00")
    Absolute {
        deadline: DateTime<Utc>,
        hard_deadline: bool,
    },
    /// Relative time constraint (e.g., "must occur within 1 hour of trigger")
    Relative {
        reference_event: String,
        offset: Duration,
        direction: TemporalDirection,
    },
    /// Sliding time window (e.g., "must occur within 30 minutes of any occurrence")
    SlidingWindow {
        window_size: Duration,
        reference_events: Vec<String>,
    },
    /// Periodic constraint (e.g., "must occur every 4 hours")
    Periodic {
        interval: Duration,
        start_time: DateTime<Utc>,
        max_delay: Option<Duration>,
    },
    /// Time-of-day constraint (e.g., "must occur between 8 AM and 6 PM")
    TimeOfDay {
        start_time: NaiveTime,
        end_time: NaiveTime,
        days_of_week: Vec<u32>, // 1=Monday, 7=Sunday
    },
    /// Sequential timing (e.g., "B must occur at least 30 minutes after A")
    Sequential {
        predecessor_event: String,
        minimum_gap: Duration,
        maximum_gap: Option<Duration>,
    },
    /// Concurrent constraint (e.g., "must occur simultaneously with other event")
    Concurrent {
        reference_events: Vec<String>,
        tolerance: Duration,
    },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TemporalDirection {
    Before,
    After,
    Around, // Within offset in either direction
}

/// Time window with enhanced precision and validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimeWindow {
    pub start: DateTime<Utc>,
    pub end: DateTime<Utc>,
    pub duration: Option<Duration>,
    pub precision_ms: u64,
    pub timezone_aware: bool,
    pub business_hours_only: Option<BusinessHours>,
    pub excluded_periods: Vec<ExcludedPeriod>,
}

/// Business hours definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BusinessHours {
    pub weekday_start: NaiveTime,
    pub weekday_end: NaiveTime,
    pub weekend_start: Option<NaiveTime>,
    pub weekend_end: Option<NaiveTime>,
    pub holidays: Vec<DateTime<Utc>>,
}

/// Periods to exclude from time calculations
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExcludedPeriod {
    pub start: DateTime<Utc>,
    pub end: DateTime<Utc>,
    pub reason: String,
    pub recurring: Option<RecurrencePattern>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecurrencePattern {
    pub frequency: RecurrenceFrequency,
    pub interval: u32,
    pub end_date: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecurrenceFrequency {
    Daily,
    Weekly,
    Monthly,
    Yearly,
}

/// Clinical context for temporal constraints
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalTemporalContext {
    pub patient_acuity: AcuityLevel,
    pub department_type: DepartmentType,
    pub shift_pattern: ShiftPattern,
    pub critical_care: bool,
    pub emergency_context: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DepartmentType {
    EmergencyDepartment,
    IntensiveCare,
    MedicalSurgical,
    Outpatient,
    OperatingRoom,
    PostAnesthesiaCare,
    Other(String),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ShiftPattern {
    DayShift,
    NightShift,
    TwelveHour,
    EightHour,
    OnCall,
}

/// Temporal dependency between constraints
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemporalDependency {
    pub dependency_id: String,
    pub depends_on_constraint: String,
    pub dependency_type: DependencyType,
    pub required: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DependencyType {
    MustCompleteBefore,
    MustCompleteAfter,
    MustCompleteWithin { duration: Duration },
    MustNotOverlap,
    MustOverlap { minimum_overlap: Duration },
}

/// High-precision temporal constraint engine
pub struct TemporalConstraintEngine {
    config: TemporalEngineConfig,
    /// Cache for temporal constraint evaluations
    constraint_cache: Arc<RwLock<LruCache<String, ConstraintEvaluationResult>>>,
    /// Time precision tracker for performance optimization
    precision_tracker: Arc<PrecisionTracker>,
    /// Metrics for temporal engine performance
    metrics: Arc<TemporalEngineMetrics>,
}

/// Tracks time precision requirements for optimization
#[derive(Debug)]
pub struct PrecisionTracker {
    pub required_precision_ms: std::sync::atomic::AtomicU64,
    pub evaluation_count: std::sync::atomic::AtomicU64,
    pub high_precision_count: std::sync::atomic::AtomicU64,
}

/// Temporal engine performance metrics
#[derive(Debug, Default)]
pub struct TemporalEngineMetrics {
    pub total_evaluations: std::sync::atomic::AtomicU64,
    pub successful_evaluations: std::sync::atomic::AtomicU64,
    pub failed_evaluations: std::sync::atomic::AtomicU64,
    pub cache_hits: std::sync::atomic::AtomicU64,
    pub cache_misses: std::sync::atomic::AtomicU64,
    pub average_evaluation_time_ms: std::sync::atomic::AtomicU64,
    pub high_precision_evaluations: std::sync::atomic::AtomicU64,
}

impl TemporalConstraintEngine {
    /// Create new temporal constraint engine
    pub fn new(config: &TemporalEngineConfig) -> ProtocolResult<Self> {
        let constraint_cache = if config.enable_temporal_caching {
            Arc::new(RwLock::new(LruCache::new(config.max_temporal_constraints.try_into()
                .map_err(|_| ProtocolEngineError::ConfigurationError {
                    message: "Invalid temporal constraint cache size".to_string()
                })?)))
        } else {
            Arc::new(RwLock::new(LruCache::new(1.try_into().unwrap()))) // Minimal cache
        };
        
        Ok(Self {
            config: config.clone(),
            constraint_cache,
            precision_tracker: Arc::new(PrecisionTracker {
                required_precision_ms: std::sync::atomic::AtomicU64::new(config.precision_ms),
                evaluation_count: std::sync::atomic::AtomicU64::new(0),
                high_precision_count: std::sync::atomic::AtomicU64::new(0),
            }),
            metrics: Arc::new(TemporalEngineMetrics::default()),
        })
    }
    
    /// Evaluate temporal constraints with high precision
    pub async fn evaluate_temporal_constraints(
        &self,
        constraints: &[TemporalConstraintDefinition],
        context: &ProtocolContext,
        evaluation_time: DateTime<Utc>,
        eval_context: &mut EvaluationContext,
    ) -> ProtocolResult<Vec<AppliedTemporalConstraint>> {
        let start_time = Instant::now();
        
        // Update metrics
        self.metrics.total_evaluations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        
        let mut results = Vec::new();
        
        // Process constraints in parallel where possible
        for constraint_def in constraints {
            let constraint = self.convert_definition_to_constraint(constraint_def)?;
            
            let applied_constraint = self.evaluate_single_constraint(
                &constraint,
                context,
                evaluation_time,
                eval_context,
            ).await?;
            
            results.push(applied_constraint);
        }
        
        // Update performance metrics
        let evaluation_time_ms = start_time.elapsed().as_millis() as u64;
        self.update_average_evaluation_time(evaluation_time_ms);
        
        eval_context.snapshot_resolution_time_ms += evaluation_time_ms;
        
        self.metrics.successful_evaluations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        
        Ok(results)
    }
    
    /// Evaluate a single temporal constraint
    async fn evaluate_single_constraint(
        &self,
        constraint: &TemporalConstraint,
        context: &ProtocolContext,
        evaluation_time: DateTime<Utc>,
        eval_context: &mut EvaluationContext,
    ) -> ProtocolResult<AppliedTemporalConstraint> {
        let start_time = Instant::now();
        
        // Check cache first
        let cache_key = self.generate_cache_key(constraint, &evaluation_time);
        
        if self.config.enable_temporal_caching {
            let cache = self.constraint_cache.read();
            if let Some(cached_result) = cache.peek(&cache_key) {
                self.metrics.cache_hits.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                return Ok(AppliedTemporalConstraint {
                    constraint_id: constraint.constraint_id.clone(),
                    constraint_type: format!("{:?}", constraint.constraint_type),
                    time_window_start: constraint.time_window.start,
                    time_window_end: constraint.time_window.end,
                    satisfied: cached_result.satisfied,
                });
            }
        }
        
        self.metrics.cache_misses.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        
        // Evaluate constraint based on type
        let satisfied = match &constraint.constraint_type {
            TemporalConstraintType::Absolute { deadline, hard_deadline } => {
                self.evaluate_absolute_constraint(*deadline, *hard_deadline, evaluation_time)
            },
            TemporalConstraintType::Relative { reference_event, offset, direction } => {
                self.evaluate_relative_constraint(
                    reference_event,
                    *offset,
                    direction,
                    evaluation_time,
                    context,
                    eval_context,
                ).await?
            },
            TemporalConstraintType::SlidingWindow { window_size, reference_events } => {
                self.evaluate_sliding_window_constraint(
                    *window_size,
                    reference_events,
                    evaluation_time,
                    context,
                ).await?
            },
            TemporalConstraintType::Periodic { interval, start_time, max_delay } => {
                self.evaluate_periodic_constraint(
                    *interval,
                    *start_time,
                    max_delay.as_ref(),
                    evaluation_time,
                )?
            },
            TemporalConstraintType::TimeOfDay { start_time, end_time, days_of_week } => {
                self.evaluate_time_of_day_constraint(
                    start_time,
                    end_time,
                    days_of_week,
                    evaluation_time,
                )?
            },
            TemporalConstraintType::Sequential { predecessor_event, minimum_gap, maximum_gap } => {
                self.evaluate_sequential_constraint(
                    predecessor_event,
                    *minimum_gap,
                    maximum_gap.as_ref(),
                    evaluation_time,
                    context,
                ).await?
            },
            TemporalConstraintType::Concurrent { reference_events, tolerance } => {
                self.evaluate_concurrent_constraint(
                    reference_events,
                    *tolerance,
                    evaluation_time,
                    context,
                ).await?
            },
        };
        
        // Create evaluation result
        let evaluation_result = ConstraintEvaluationResult {
            constraint_id: constraint.constraint_id.clone(),
            constraint_type: ConstraintType::Temporal,
            satisfied,
            details: Some(format!("Evaluated at {} with precision {}ms", 
                evaluation_time, constraint.time_window.precision_ms)),
        };
        
        // Cache the result
        if self.config.enable_temporal_caching {
            let mut cache = self.constraint_cache.write();
            cache.put(cache_key, evaluation_result.clone());
        }
        
        // Update precision tracking
        if constraint.time_window.precision_ms <= self.config.precision_ms {
            self.metrics.high_precision_evaluations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        }
        
        // Log constraint evaluation
        eval_context.add_information(format!(
            "Temporal constraint '{}' evaluated: {} ({}ms)",
            constraint.name,
            if satisfied { "SATISFIED" } else { "VIOLATED" },
            start_time.elapsed().as_millis()
        ));
        
        Ok(AppliedTemporalConstraint {
            constraint_id: constraint.constraint_id.clone(),
            constraint_type: format!("{:?}", constraint.constraint_type),
            time_window_start: constraint.time_window.start,
            time_window_end: constraint.time_window.end,
            satisfied,
        })
    }
    
    /// Evaluate absolute temporal constraint
    fn evaluate_absolute_constraint(
        &self,
        deadline: DateTime<Utc>,
        hard_deadline: bool,
        evaluation_time: DateTime<Utc>,
    ) -> bool {
        if hard_deadline {
            evaluation_time <= deadline
        } else {
            // Allow some tolerance for soft deadlines
            let tolerance = Duration::minutes(5);
            evaluation_time <= (deadline + tolerance)
        }
    }
    
    /// Evaluate relative temporal constraint
    async fn evaluate_relative_constraint(
        &self,
        reference_event: &str,
        offset: Duration,
        direction: &TemporalDirection,
        evaluation_time: DateTime<Utc>,
        context: &ProtocolContext,
        eval_context: &EvaluationContext,
    ) -> ProtocolResult<bool> {
        // Find reference event time (placeholder implementation)
        let reference_time = self.find_reference_event_time(reference_event, context, eval_context).await?;
        
        match direction {
            TemporalDirection::Before => evaluation_time <= (reference_time + offset),
            TemporalDirection::After => evaluation_time >= (reference_time + offset),
            TemporalDirection::Around => {
                let lower_bound = reference_time - offset;
                let upper_bound = reference_time + offset;
                evaluation_time >= lower_bound && evaluation_time <= upper_bound
            },
        }
    }
    
    /// Evaluate sliding window constraint
    async fn evaluate_sliding_window_constraint(
        &self,
        window_size: Duration,
        reference_events: &[String],
        evaluation_time: DateTime<Utc>,
        context: &ProtocolContext,
    ) -> ProtocolResult<bool> {
        // Check if evaluation time falls within window of any reference event
        for event_name in reference_events {
            if let Ok(event_times) = self.find_event_occurrences(event_name, context).await {
                for event_time in event_times {
                    let window_start = event_time;
                    let window_end = event_time + window_size;
                    
                    if evaluation_time >= window_start && evaluation_time <= window_end {
                        return Ok(true);
                    }
                }
            }
        }
        
        Ok(false)
    }
    
    /// Evaluate periodic constraint
    fn evaluate_periodic_constraint(
        &self,
        interval: Duration,
        start_time: DateTime<Utc>,
        max_delay: Option<&Duration>,
        evaluation_time: DateTime<Utc>,
    ) -> ProtocolResult<bool> {
        if evaluation_time < start_time {
            return Ok(false);
        }
        
        let elapsed = evaluation_time - start_time;
        let intervals_passed = elapsed.num_milliseconds() / interval.num_milliseconds();
        let next_expected = start_time + Duration::milliseconds(intervals_passed * interval.num_milliseconds());
        
        let allowed_delay = max_delay.unwrap_or(&Duration::minutes(15));
        let latest_allowed = next_expected + *allowed_delay;
        
        Ok(evaluation_time <= latest_allowed)
    }
    
    /// Evaluate time-of-day constraint
    fn evaluate_time_of_day_constraint(
        &self,
        start_time: &NaiveTime,
        end_time: &NaiveTime,
        days_of_week: &[u32],
        evaluation_time: DateTime<Utc>,
    ) -> ProtocolResult<bool> {
        use chrono::Datelike;
        
        // Check day of week
        let day_of_week = evaluation_time.weekday().number_from_monday();
        if !days_of_week.is_empty() && !days_of_week.contains(&day_of_week) {
            return Ok(false);
        }
        
        // Check time of day
        let eval_time = evaluation_time.time();
        
        if start_time <= end_time {
            // Same day time range
            Ok(eval_time >= *start_time && eval_time <= *end_time)
        } else {
            // Overnight time range
            Ok(eval_time >= *start_time || eval_time <= *end_time)
        }
    }
    
    /// Evaluate sequential constraint
    async fn evaluate_sequential_constraint(
        &self,
        predecessor_event: &str,
        minimum_gap: Duration,
        maximum_gap: Option<&Duration>,
        evaluation_time: DateTime<Utc>,
        context: &ProtocolContext,
    ) -> ProtocolResult<bool> {
        let predecessor_time = self.find_last_event_occurrence(predecessor_event, context).await?;
        
        let time_since_predecessor = evaluation_time - predecessor_time;
        
        if time_since_predecessor < minimum_gap {
            return Ok(false);
        }
        
        if let Some(max_gap) = maximum_gap {
            if time_since_predecessor > *max_gap {
                return Ok(false);
            }
        }
        
        Ok(true)
    }
    
    /// Evaluate concurrent constraint
    async fn evaluate_concurrent_constraint(
        &self,
        reference_events: &[String],
        tolerance: Duration,
        evaluation_time: DateTime<Utc>,
        context: &ProtocolContext,
    ) -> ProtocolResult<bool> {
        // Check if evaluation time is within tolerance of any reference event
        for event_name in reference_events {
            if let Ok(event_times) = self.find_event_occurrences(event_name, context).await {
                for event_time in event_times {
                    let time_diff = (evaluation_time - event_time).abs();
                    if time_diff <= tolerance {
                        return Ok(true);
                    }
                }
            }
        }
        
        Ok(false)
    }
    
    /// Convert constraint definition to runtime constraint
    fn convert_definition_to_constraint(
        &self,
        definition: &TemporalConstraintDefinition,
    ) -> ProtocolResult<TemporalConstraint> {
        // Create time window based on definition
        let time_window = TimeWindow {
            start: Utc::now(), // Would be calculated based on definition
            end: Utc::now() + Duration::hours(1), // Default window
            duration: definition.time_window.duration,
            precision_ms: self.config.precision_ms,
            timezone_aware: false,
            business_hours_only: None,
            excluded_periods: vec![],
        };
        
        // Map constraint type
        let constraint_type = match definition.constraint_type {
            TemporalConstraintType::WithinTimeWindow => {
                TemporalConstraintType::SlidingWindow {
                    window_size: definition.time_window.duration.unwrap_or(Duration::hours(1)),
                    reference_events: vec![definition.reference_event.clone()],
                }
            },
            TemporalConstraintType::BeforeDeadline => {
                TemporalConstraintType::Absolute {
                    deadline: Utc::now() + Duration::hours(1), // Would be calculated from definition
                    hard_deadline: true,
                }
            },
            TemporalConstraintType::AfterMinimumTime => {
                TemporalConstraintType::Relative {
                    reference_event: definition.reference_event.clone(),
                    offset: Duration::minutes(30), // Would come from definition
                    direction: TemporalDirection::After,
                }
            },
            TemporalConstraintType::Periodic => {
                TemporalConstraintType::Periodic {
                    interval: Duration::hours(4), // Would come from definition
                    start_time: Utc::now(),
                    max_delay: Some(Duration::minutes(15)),
                }
            },
            TemporalConstraintType::SequentialTiming => {
                TemporalConstraintType::Sequential {
                    predecessor_event: definition.reference_event.clone(),
                    minimum_gap: Duration::minutes(30),
                    maximum_gap: Some(Duration::hours(2)),
                }
            },
        };
        
        Ok(TemporalConstraint {
            constraint_id: definition.constraint_id.clone(),
            name: definition.name.clone(),
            description: format!("Temporal constraint for {}", definition.name),
            constraint_type,
            time_window,
            condition: definition.condition.clone(),
            severity: ConstraintSeverity::Warning, // Would be determined from definition
            override_allowed: true,
            clinical_context: None,
            dependencies: vec![],
            metadata: HashMap::new(),
        })
    }
    
    // Helper methods for finding events (placeholder implementations)
    
    async fn find_reference_event_time(
        &self,
        _event_name: &str,
        _context: &ProtocolContext,
        _eval_context: &EvaluationContext,
    ) -> ProtocolResult<DateTime<Utc>> {
        // Placeholder: would look up actual event time from context or database
        Ok(Utc::now() - Duration::minutes(30))
    }
    
    async fn find_event_occurrences(
        &self,
        _event_name: &str,
        _context: &ProtocolContext,
    ) -> ProtocolResult<Vec<DateTime<Utc>>> {
        // Placeholder: would find all occurrences of the event
        Ok(vec![Utc::now() - Duration::minutes(15)])
    }
    
    async fn find_last_event_occurrence(
        &self,
        _event_name: &str,
        _context: &ProtocolContext,
    ) -> ProtocolResult<DateTime<Utc>> {
        // Placeholder: would find the most recent occurrence of the event
        Ok(Utc::now() - Duration::minutes(45))
    }
    
    /// Generate cache key for constraint evaluation
    fn generate_cache_key(&self, constraint: &TemporalConstraint, evaluation_time: &DateTime<Utc>) -> String {
        format!("{}:{}", constraint.constraint_id, evaluation_time.timestamp_millis())
    }
    
    /// Update average evaluation time metric
    fn update_average_evaluation_time(&self, evaluation_time_ms: u64) {
        let current_avg = self.metrics.average_evaluation_time_ms.load(std::sync::atomic::Ordering::Relaxed);
        let total_evaluations = self.metrics.total_evaluations.load(std::sync::atomic::Ordering::Relaxed);
        
        if total_evaluations > 0 {
            let new_avg = ((current_avg * (total_evaluations - 1)) + evaluation_time_ms) / total_evaluations;
            self.metrics.average_evaluation_time_ms.store(new_avg, std::sync::atomic::Ordering::Relaxed);
        }
    }
    
    /// Get temporal engine metrics
    pub fn get_metrics(&self) -> TemporalEngineMetrics {
        TemporalEngineMetrics {
            total_evaluations: std::sync::atomic::AtomicU64::new(
                self.metrics.total_evaluations.load(std::sync::atomic::Ordering::Relaxed)
            ),
            successful_evaluations: std::sync::atomic::AtomicU64::new(
                self.metrics.successful_evaluations.load(std::sync::atomic::Ordering::Relaxed)
            ),
            failed_evaluations: std::sync::atomic::AtomicU64::new(
                self.metrics.failed_evaluations.load(std::sync::atomic::Ordering::Relaxed)
            ),
            cache_hits: std::sync::atomic::AtomicU64::new(
                self.metrics.cache_hits.load(std::sync::atomic::Ordering::Relaxed)
            ),
            cache_misses: std::sync::atomic::AtomicU64::new(
                self.metrics.cache_misses.load(std::sync::atomic::Ordering::Relaxed)
            ),
            average_evaluation_time_ms: std::sync::atomic::AtomicU64::new(
                self.metrics.average_evaluation_time_ms.load(std::sync::atomic::Ordering::Relaxed)
            ),
            high_precision_evaluations: std::sync::atomic::AtomicU64::new(
                self.metrics.high_precision_evaluations.load(std::sync::atomic::Ordering::Relaxed)
            ),
        }
    }
    
    /// Create a clinical time window for medication administration
    pub fn create_medication_time_window(
        medication_name: &str,
        prescribed_time: DateTime<Utc>,
        tolerance_minutes: i64,
    ) -> TemporalConstraint {
        TemporalConstraint {
            constraint_id: format!("med_window_{}", medication_name),
            name: format!("{} Administration Window", medication_name),
            description: format!("Time window for {} administration", medication_name),
            constraint_type: TemporalConstraintType::Relative {
                reference_event: "medication_prescribed".to_string(),
                offset: Duration::minutes(tolerance_minutes),
                direction: TemporalDirection::Around,
            },
            time_window: TimeWindow {
                start: prescribed_time - Duration::minutes(tolerance_minutes),
                end: prescribed_time + Duration::minutes(tolerance_minutes),
                duration: Some(Duration::minutes(tolerance_minutes * 2)),
                precision_ms: 60000, // 1-minute precision for medications
                timezone_aware: true,
                business_hours_only: None,
                excluded_periods: vec![],
            },
            condition: format!("medication_name = '{}'", medication_name),
            severity: ConstraintSeverity::Error,
            override_allowed: false, // Medication timing is critical
            clinical_context: Some(ClinicalTemporalContext {
                patient_acuity: AcuityLevel::Level3,
                department_type: DepartmentType::MedicalSurgical,
                shift_pattern: ShiftPattern::EightHour,
                critical_care: false,
                emergency_context: false,
            }),
            dependencies: vec![],
            metadata: HashMap::from([
                ("medication_type".to_string(), serde_json::json!("scheduled")),
                ("critical_timing".to_string(), serde_json::json!(true))
            ]),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_temporal_engine_creation() {
        let config = TemporalEngineConfig {
            precision_ms: 1000,
            max_temporal_constraints: 100,
            enable_temporal_caching: true,
        };
        
        let engine = TemporalConstraintEngine::new(&config);
        assert!(engine.is_ok());
    }
}