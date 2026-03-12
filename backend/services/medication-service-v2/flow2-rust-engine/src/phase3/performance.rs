//! Phase 3 Performance Monitoring
//! 
//! Tracks Phase 3 performance metrics including SLA compliance,
//! sub-phase timing, and throughput statistics.

use std::collections::HashMap;
use std::sync::atomic::{AtomicU64, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use dashmap::DashMap;

use super::models::{Phase3Output, CandidateSet};

/// Phase 3 performance metrics collector
pub struct Phase3Metrics {
    // Counters
    total_requests: AtomicUsize,
    successful_requests: AtomicUsize,
    failed_requests: AtomicUsize,
    sla_violations: AtomicUsize,
    
    // Timing metrics (in microseconds for precision)
    total_phase3_time_us: AtomicU64,
    total_candidates_time_us: AtomicU64,
    total_dosing_time_us: AtomicU64,
    total_scoring_time_us: AtomicU64,
    
    // Candidate metrics
    total_candidates_generated: AtomicUsize,
    total_candidates_vetted: AtomicUsize,
    total_candidates_safe: AtomicUsize,
    
    // Recent request tracking for percentile calculations
    recent_requests: Arc<DashMap<String, RequestMetrics>>,
    
    // Start time for uptime calculation
    start_time: Instant,
}

impl Phase3Metrics {
    /// Create new Phase 3 metrics collector
    pub fn new() -> Self {
        Self {
            total_requests: AtomicUsize::new(0),
            successful_requests: AtomicUsize::new(0),
            failed_requests: AtomicUsize::new(0),
            sla_violations: AtomicUsize::new(0),
            total_phase3_time_us: AtomicU64::new(0),
            total_candidates_time_us: AtomicU64::new(0),
            total_dosing_time_us: AtomicU64::new(0),
            total_scoring_time_us: AtomicU64::new(0),
            total_candidates_generated: AtomicUsize::new(0),
            total_candidates_vetted: AtomicUsize::new(0),
            total_candidates_safe: AtomicUsize::new(0),
            recent_requests: Arc::new(DashMap::new()),
            start_time: Instant::now(),
        }
    }
    
    /// Record Phase 3 completion metrics
    pub fn record_phase3_completion(&self, output: &Phase3Output) {
        // Update counters
        self.total_requests.fetch_add(1, Ordering::Relaxed);
        self.successful_requests.fetch_add(1, Ordering::Relaxed);
        
        // Check SLA compliance (75ms target)
        if output.phase3_duration.as_millis() > 75 {
            self.sla_violations.fetch_add(1, Ordering::Relaxed);
        }
        
        // Update timing metrics
        self.total_phase3_time_us.fetch_add(
            output.phase3_duration.as_micros() as u64,
            Ordering::Relaxed,
        );
        
        // Update sub-phase timings
        if let Some(candidates_time) = output.sub_phase_timing.get("3a_candidates") {
            self.total_candidates_time_us.fetch_add(
                candidates_time.as_micros() as u64,
                Ordering::Relaxed,
            );
        }
        
        if let Some(dosing_time) = output.sub_phase_timing.get("3b_dosing") {
            self.total_dosing_time_us.fetch_add(
                dosing_time.as_micros() as u64,
                Ordering::Relaxed,
            );
        }
        
        if let Some(scoring_time) = output.sub_phase_timing.get("3c_scoring") {
            self.total_scoring_time_us.fetch_add(
                scoring_time.as_micros() as u64,
                Ordering::Relaxed,
            );
        }
        
        // Update candidate metrics
        self.total_candidates_generated.fetch_add(
            output.candidate_count,
            Ordering::Relaxed,
        );
        self.total_candidates_vetted.fetch_add(
            output.safety_vetted,
            Ordering::Relaxed,
        );
        
        // Store recent request for percentile calculations
        let request_metrics = RequestMetrics {
            request_id: output.request_id.clone(),
            timestamp: Utc::now(),
            phase3_duration: output.phase3_duration,
            sub_phase_timing: output.sub_phase_timing.clone(),
            candidate_count: output.candidate_count,
            safety_vetted: output.safety_vetted,
            proposals_count: output.ranked_proposals.len(),
            sla_compliant: output.phase3_duration.as_millis() <= 75,
        };
        
        self.recent_requests.insert(output.request_id.clone(), request_metrics);
        
        // Cleanup old entries (keep last 1000 requests)
        if self.recent_requests.len() > 1000 {
            self.cleanup_old_requests();
        }
    }
    
    /// Record candidate generation metrics
    pub fn record_candidate_generation(&self, candidate_set: &CandidateSet) {
        self.total_candidates_generated.fetch_add(
            candidate_set.initial_count,
            Ordering::Relaxed,
        );
        self.total_candidates_vetted.fetch_add(
            candidate_set.vetted_count,
            Ordering::Relaxed,
        );
        self.total_candidates_safe.fetch_add(
            candidate_set.safe_count,
            Ordering::Relaxed,
        );
    }
    
    /// Record failed request
    pub fn record_failure(&self, request_id: &str, error: &str) {
        self.total_requests.fetch_add(1, Ordering::Relaxed);
        self.failed_requests.fetch_add(1, Ordering::Relaxed);
        
        // Store failure information
        let failure_metrics = RequestMetrics {
            request_id: request_id.to_string(),
            timestamp: Utc::now(),
            phase3_duration: Duration::from_millis(0),
            sub_phase_timing: HashMap::new(),
            candidate_count: 0,
            safety_vetted: 0,
            proposals_count: 0,
            sla_compliant: false,
        };
        
        self.recent_requests.insert(request_id.to_string(), failure_metrics);
    }
    
    /// Get current metrics snapshot
    pub fn get_snapshot(&self) -> Phase3MetricsSnapshot {
        let total_requests = self.total_requests.load(Ordering::Relaxed);
        let successful_requests = self.successful_requests.load(Ordering::Relaxed);
        let failed_requests = self.failed_requests.load(Ordering::Relaxed);
        let sla_violations = self.sla_violations.load(Ordering::Relaxed);
        
        let total_phase3_time_us = self.total_phase3_time_us.load(Ordering::Relaxed);
        let total_candidates_time_us = self.total_candidates_time_us.load(Ordering::Relaxed);
        let total_dosing_time_us = self.total_dosing_time_us.load(Ordering::Relaxed);
        let total_scoring_time_us = self.total_scoring_time_us.load(Ordering::Relaxed);
        
        let total_candidates_generated = self.total_candidates_generated.load(Ordering::Relaxed);
        let total_candidates_vetted = self.total_candidates_vetted.load(Ordering::Relaxed);
        let total_candidates_safe = self.total_candidates_safe.load(Ordering::Relaxed);
        
        // Calculate averages
        let avg_phase3_time_ms = if successful_requests > 0 {
            (total_phase3_time_us as f64 / successful_requests as f64) / 1000.0
        } else {
            0.0
        };
        
        let avg_candidates_time_ms = if successful_requests > 0 {
            (total_candidates_time_us as f64 / successful_requests as f64) / 1000.0
        } else {
            0.0
        };
        
        let avg_dosing_time_ms = if successful_requests > 0 {
            (total_dosing_time_us as f64 / successful_requests as f64) / 1000.0
        } else {
            0.0
        };
        
        let avg_scoring_time_ms = if successful_requests > 0 {
            (total_scoring_time_us as f64 / successful_requests as f64) / 1000.0
        } else {
            0.0
        };
        
        // Calculate rates
        let success_rate = if total_requests > 0 {
            (successful_requests as f64 / total_requests as f64) * 100.0
        } else {
            0.0
        };
        
        let sla_compliance_rate = if successful_requests > 0 {
            ((successful_requests - sla_violations) as f64 / successful_requests as f64) * 100.0
        } else {
            0.0
        };
        
        let safety_vetting_rate = if total_candidates_generated > 0 {
            (total_candidates_vetted as f64 / total_candidates_generated as f64) * 100.0
        } else {
            0.0
        };
        
        let safety_pass_rate = if total_candidates_vetted > 0 {
            (total_candidates_safe as f64 / total_candidates_vetted as f64) * 100.0
        } else {
            0.0
        };
        
        // Calculate throughput (requests per second)
        let uptime_seconds = self.start_time.elapsed().as_secs_f64();
        let throughput_rps = if uptime_seconds > 0.0 {
            total_requests as f64 / uptime_seconds
        } else {
            0.0
        };
        
        // Get percentile data from recent requests
        let percentiles = self.calculate_percentiles();
        
        Phase3MetricsSnapshot {
            timestamp: Utc::now(),
            uptime_seconds,
            
            // Request metrics
            total_requests,
            successful_requests,
            failed_requests,
            success_rate,
            throughput_rps,
            
            // SLA metrics
            sla_violations,
            sla_compliance_rate,
            
            // Timing metrics
            avg_phase3_time_ms,
            avg_candidates_time_ms,
            avg_dosing_time_ms,
            avg_scoring_time_ms,
            
            // Candidate metrics
            total_candidates_generated,
            total_candidates_vetted,
            total_candidates_safe,
            safety_vetting_rate,
            safety_pass_rate,
            
            // Percentile data
            phase3_time_percentiles: percentiles.phase3_time,
            candidates_time_percentiles: percentiles.candidates_time,
            dosing_time_percentiles: percentiles.dosing_time,
            scoring_time_percentiles: percentiles.scoring_time,
        }
    }
    
    /// Calculate timing percentiles from recent requests
    fn calculate_percentiles(&self) -> PercentileData {
        let mut phase3_times: Vec<f64> = Vec::new();
        let mut candidates_times: Vec<f64> = Vec::new();
        let mut dosing_times: Vec<f64> = Vec::new();
        let mut scoring_times: Vec<f64> = Vec::new();
        
        for entry in self.recent_requests.iter() {
            let metrics = entry.value();
            
            phase3_times.push(metrics.phase3_duration.as_millis() as f64);
            
            if let Some(candidates_time) = metrics.sub_phase_timing.get("3a_candidates") {
                candidates_times.push(candidates_time.as_millis() as f64);
            }
            
            if let Some(dosing_time) = metrics.sub_phase_timing.get("3b_dosing") {
                dosing_times.push(dosing_time.as_millis() as f64);
            }
            
            if let Some(scoring_time) = metrics.sub_phase_timing.get("3c_scoring") {
                scoring_times.push(scoring_time.as_millis() as f64);
            }
        }
        
        PercentileData {
            phase3_time: calculate_percentile_stats(&mut phase3_times),
            candidates_time: calculate_percentile_stats(&mut candidates_times),
            dosing_time: calculate_percentile_stats(&mut dosing_times),
            scoring_time: calculate_percentile_stats(&mut scoring_times),
        }
    }
    
    /// Cleanup old request entries to prevent memory growth
    fn cleanup_old_requests(&self) {
        let cutoff_time = Utc::now() - chrono::Duration::minutes(10);
        
        let keys_to_remove: Vec<String> = self.recent_requests
            .iter()
            .filter(|entry| entry.value().timestamp < cutoff_time)
            .map(|entry| entry.key().clone())
            .collect();
        
        for key in keys_to_remove {
            self.recent_requests.remove(&key);
        }
    }
    
    /// Get current request count
    pub fn get_request_count(&self) -> usize {
        self.total_requests.load(Ordering::Relaxed)
    }
    
    /// Get SLA compliance rate
    pub fn get_sla_compliance_rate(&self) -> f64 {
        let successful_requests = self.successful_requests.load(Ordering::Relaxed);
        let sla_violations = self.sla_violations.load(Ordering::Relaxed);
        
        if successful_requests > 0 {
            ((successful_requests - sla_violations) as f64 / successful_requests as f64) * 100.0
        } else {
            0.0
        }
    }
}

/// Individual request metrics for percentile calculations
#[derive(Debug, Clone)]
struct RequestMetrics {
    request_id: String,
    timestamp: DateTime<Utc>,
    phase3_duration: Duration,
    sub_phase_timing: HashMap<String, Duration>,
    candidate_count: usize,
    safety_vetted: usize,
    proposals_count: usize,
    sla_compliant: bool,
}

/// Complete metrics snapshot
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Phase3MetricsSnapshot {
    pub timestamp: DateTime<Utc>,
    pub uptime_seconds: f64,
    
    // Request metrics
    pub total_requests: usize,
    pub successful_requests: usize,
    pub failed_requests: usize,
    pub success_rate: f64,
    pub throughput_rps: f64,
    
    // SLA metrics
    pub sla_violations: usize,
    pub sla_compliance_rate: f64,
    
    // Timing metrics (averages in milliseconds)
    pub avg_phase3_time_ms: f64,
    pub avg_candidates_time_ms: f64,
    pub avg_dosing_time_ms: f64,
    pub avg_scoring_time_ms: f64,
    
    // Candidate metrics
    pub total_candidates_generated: usize,
    pub total_candidates_vetted: usize,
    pub total_candidates_safe: usize,
    pub safety_vetting_rate: f64,
    pub safety_pass_rate: f64,
    
    // Percentile data
    pub phase3_time_percentiles: PercentileStats,
    pub candidates_time_percentiles: PercentileStats,
    pub dosing_time_percentiles: PercentileStats,
    pub scoring_time_percentiles: PercentileStats,
}

/// Percentile statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PercentileStats {
    pub p50: f64,
    pub p90: f64,
    pub p95: f64,
    pub p99: f64,
    pub min: f64,
    pub max: f64,
    pub count: usize,
}

/// Percentile data for all timing metrics
#[derive(Debug, Clone)]
struct PercentileData {
    phase3_time: PercentileStats,
    candidates_time: PercentileStats,
    dosing_time: PercentileStats,
    scoring_time: PercentileStats,
}

/// Calculate percentile statistics from a vector of values
fn calculate_percentile_stats(values: &mut Vec<f64>) -> PercentileStats {
    if values.is_empty() {
        return PercentileStats {
            p50: 0.0,
            p90: 0.0,
            p95: 0.0,
            p99: 0.0,
            min: 0.0,
            max: 0.0,
            count: 0,
        };
    }
    
    values.sort_by(|a, b| a.partial_cmp(b).unwrap());
    
    let len = values.len();
    let min = values[0];
    let max = values[len - 1];
    
    let p50 = percentile(values, 50.0);
    let p90 = percentile(values, 90.0);
    let p95 = percentile(values, 95.0);
    let p99 = percentile(values, 99.0);
    
    PercentileStats {
        p50,
        p90,
        p95,
        p99,
        min,
        max,
        count: len,
    }
}

/// Calculate a specific percentile from sorted values
fn percentile(sorted_values: &[f64], percentile: f64) -> f64 {
    if sorted_values.is_empty() {
        return 0.0;
    }
    
    let index = (percentile / 100.0) * (sorted_values.len() - 1) as f64;
    let lower_index = index.floor() as usize;
    let upper_index = index.ceil() as usize;
    
    if lower_index == upper_index {
        sorted_values[lower_index]
    } else {
        let weight = index - lower_index as f64;
        sorted_values[lower_index] * (1.0 - weight) + sorted_values[upper_index] * weight
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::phase3::models::*;
    use std::collections::HashMap;
    
    #[test]
    fn test_metrics_creation() {
        let metrics = Phase3Metrics::new();
        assert_eq!(metrics.get_request_count(), 0);
        assert_eq!(metrics.get_sla_compliance_rate(), 0.0);
    }
    
    #[test]
    fn test_percentile_calculation() {
        let mut values = vec![1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0];
        let stats = calculate_percentile_stats(&mut values);
        
        assert_eq!(stats.min, 1.0);
        assert_eq!(stats.max, 10.0);
        assert_eq!(stats.p50, 5.5); // median of 1-10
        assert_eq!(stats.count, 10);
    }
    
    #[test]
    fn test_empty_percentile_calculation() {
        let mut values = vec![];
        let stats = calculate_percentile_stats(&mut values);
        
        assert_eq!(stats.min, 0.0);
        assert_eq!(stats.max, 0.0);
        assert_eq!(stats.p50, 0.0);
        assert_eq!(stats.count, 0);
    }
    
    #[tokio::test]
    async fn test_metrics_recording() {
        let metrics = Phase3Metrics::new();
        
        // Create mock Phase3Output
        let output = Phase3Output {
            request_id: "test_request".to_string(),
            candidate_count: 5,
            safety_vetted: 4,
            dose_calculated: 3,
            ranked_proposals: vec![],
            candidate_evidence: vec![],
            dose_evidence: vec![],
            scoring_evidence: vec![],
            phase3_duration: Duration::from_millis(50),
            sub_phase_timing: {
                let mut timing = HashMap::new();
                timing.insert("3a_candidates".to_string(), Duration::from_millis(15));
                timing.insert("3b_dosing".to_string(), Duration::from_millis(20));
                timing.insert("3c_scoring".to_string(), Duration::from_millis(15));
                timing
            },
        };
        
        metrics.record_phase3_completion(&output);
        
        let snapshot = metrics.get_snapshot();
        assert_eq!(snapshot.total_requests, 1);
        assert_eq!(snapshot.successful_requests, 1);
        assert_eq!(snapshot.sla_violations, 0); // 50ms < 75ms SLA
        assert_eq!(snapshot.total_candidates_generated, 5);
    }
}