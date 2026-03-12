// Utility functions for the safety engines
//
// This module provides common utilities used across the safety engines
// including hashing, caching, memory management, and performance monitoring.

use std::collections::HashMap;
use std::time::{Instant, Duration};
use sha2::{Sha256, Digest};
use chrono::{DateTime, Utc};

/// Memory-safe string utilities
pub mod strings {
    use std::ffi::{CStr, CString};
    use std::os::raw::c_char;
    
    /// Safely convert a C string to Rust String
    pub fn c_str_to_string(c_str: *const c_char) -> Option<String> {
        if c_str.is_null() {
            return None;
        }
        
        unsafe {
            CStr::from_ptr(c_str)
                .to_str()
                .ok()
                .map(|s| s.to_owned())
        }
    }
    
    /// Convert Rust String to C string (caller must free)
    pub fn string_to_c_str(s: &str) -> Result<*mut c_char, std::ffi::NulError> {
        let c_string = CString::new(s)?;
        Ok(c_string.into_raw())
    }
}

/// Performance monitoring utilities
pub mod performance {
    use super::*;
    
    /// Performance timer for measuring execution time
    pub struct Timer {
        start: Instant,
        label: String,
    }
    
    impl Timer {
        pub fn new(label: &str) -> Self {
            Self {
                start: Instant::now(),
                label: label.to_string(),
            }
        }
        
        pub fn elapsed(&self) -> Duration {
            self.start.elapsed()
        }
        
        pub fn elapsed_ms(&self) -> u64 {
            self.start.elapsed().as_millis() as u64
        }
        
        pub fn finish(self) -> (String, Duration) {
            (self.label, self.elapsed())
        }
        
        pub fn finish_ms(self) -> (String, u64) {
            (self.label, self.elapsed_ms())
        }
    }
    
    /// Performance statistics tracker
    #[derive(Debug, Clone)]
    pub struct PerformanceStats {
        pub total_evaluations: u64,
        pub total_time_ms: u64,
        pub average_time_ms: f64,
        pub min_time_ms: u64,
        pub max_time_ms: u64,
        pub cache_hit_rate: f64,
    }
    
    impl Default for PerformanceStats {
        fn default() -> Self {
            Self {
                total_evaluations: 0,
                total_time_ms: 0,
                average_time_ms: 0.0,
                min_time_ms: u64::MAX,
                max_time_ms: 0,
                cache_hit_rate: 0.0,
            }
        }
    }
    
    impl PerformanceStats {
        pub fn record_evaluation(&mut self, duration_ms: u64, cache_hit: bool) {
            self.total_evaluations += 1;
            self.total_time_ms += duration_ms;
            self.average_time_ms = self.total_time_ms as f64 / self.total_evaluations as f64;
            self.min_time_ms = self.min_time_ms.min(duration_ms);
            self.max_time_ms = self.max_time_ms.max(duration_ms);
            
            // Update cache hit rate (simple exponential moving average)
            let alpha = 0.1; // Smoothing factor
            if cache_hit {
                self.cache_hit_rate = alpha + (1.0 - alpha) * self.cache_hit_rate;
            } else {
                self.cache_hit_rate = (1.0 - alpha) * self.cache_hit_rate;
            }
        }
    }
}

/// Hashing utilities for cache keys and patient ID anonymization
pub mod hashing {
    use super::*;
    
    /// Generate a SHA-256 hash of the input string
    pub fn sha256_hash(input: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(input.as_bytes());
        format!("{:x}", hasher.finalize())
    }
    
    /// Generate a HIPAA-compliant hash of a patient ID
    pub fn hash_patient_id(patient_id: &str) -> String {
        // Add salt to prevent rainbow table attacks
        let salted = format!("{}:clinical_safety_salt", patient_id);
        sha256_hash(&salted)
    }
    
    /// Generate a cache key from multiple components
    pub fn generate_cache_key(components: &[&str]) -> String {
        let combined = components.join(":");
        sha256_hash(&combined)
    }
}

/// Memory management utilities
pub mod memory {
    /// Memory pool for frequent allocations
    pub struct StringPool {
        pool: Vec<String>,
        capacity: usize,
    }
    
    impl StringPool {
        pub fn new(capacity: usize) -> Self {
            Self {
                pool: Vec::with_capacity(capacity),
                capacity,
            }
        }
        
        pub fn get_string(&mut self) -> String {
            self.pool.pop().unwrap_or_else(String::new)
        }
        
        pub fn return_string(&mut self, mut s: String) {
            if self.pool.len() < self.capacity {
                s.clear();
                self.pool.push(s);
            }
        }
    }
}

/// Clinical data validation utilities
pub mod validation {
    use super::*;
    use regex::Regex;
    use once_cell::sync::Lazy;
    
    // Common medical code patterns
    static ICD10_REGEX: Lazy<Regex> = Lazy::new(|| {
        Regex::new(r"^[A-Z]\d{2}(\.\d{1,3})?$").unwrap()
    });
    
    static MEDICATION_ID_REGEX: Lazy<Regex> = Lazy::new(|| {
        Regex::new(r"^[a-zA-Z0-9_-]{1,50}$").unwrap()
    });
    
    /// Validate an ICD-10 code format
    pub fn validate_icd10_code(code: &str) -> bool {
        ICD10_REGEX.is_match(code)
    }
    
    /// Validate a medication ID format
    pub fn validate_medication_id(id: &str) -> bool {
        MEDICATION_ID_REGEX.is_match(id)
    }
    
    /// Validate patient age (reasonable bounds)
    pub fn validate_age(age: u32) -> bool {
        age <= 150
    }
    
    /// Validate weight in kilograms (reasonable bounds)
    pub fn validate_weight_kg(weight: f64) -> bool {
        weight > 0.0 && weight <= 1000.0
    }
    
    /// Validate height in centimeters (reasonable bounds)
    pub fn validate_height_cm(height: f64) -> bool {
        height > 0.0 && height <= 300.0
    }
    
    /// Validate vital signs
    pub fn validate_vital_signs(
        systolic_bp: Option<u32>,
        diastolic_bp: Option<u32>,
        heart_rate: Option<u32>,
        temperature_c: Option<f64>,
    ) -> bool {
        if let Some(sbp) = systolic_bp {
            if sbp < 50 || sbp > 300 {
                return false;
            }
        }
        
        if let Some(dbp) = diastolic_bp {
            if dbp < 30 || dbp > 200 {
                return false;
            }
        }
        
        if let Some(hr) = heart_rate {
            if hr < 20 || hr > 300 {
                return false;
            }
        }
        
        if let Some(temp) = temperature_c {
            if temp < 25.0 || temp > 45.0 {
                return false;
            }
        }
        
        true
    }
}

/// Logging utilities compatible with Go's zap logger
pub mod logging {
    use super::*;
    use serde_json::{json, Value};
    
    #[derive(Debug, Clone)]
    pub enum LogLevel {
        Debug,
        Info,
        Warn,
        Error,
    }
    
    impl LogLevel {
        pub fn as_str(&self) -> &'static str {
            match self {
                LogLevel::Debug => "debug",
                LogLevel::Info => "info",
                LogLevel::Warn => "warn",
                LogLevel::Error => "error",
            }
        }
    }
    
    /// Log a structured message compatible with zap logger format
    pub fn log_structured(level: LogLevel, message: &str, fields: HashMap<&str, Value>) {
        let mut log_entry = json!({
            "level": level.as_str(),
            "msg": message,
            "timestamp": Utc::now().to_rfc3339(),
            "logger": "rust_safety_engines"
        });
        
        // Add custom fields
        if let Some(obj) = log_entry.as_object_mut() {
            for (key, value) in fields {
                obj.insert(key.to_string(), value);
            }
        }
        
        println!("{}", log_entry);
    }
    
    /// Log a safety evaluation event
    pub fn log_safety_evaluation(
        request_id: &str,
        patient_id_hash: &str,
        status: &str,
        risk_score: f64,
        duration_ms: u64,
    ) {
        let fields = [
            ("request_id", json!(request_id)),
            ("patient_id_hash", json!(patient_id_hash)),
            ("status", json!(status)),
            ("risk_score", json!(risk_score)),
            ("duration_ms", json!(duration_ms)),
            ("event_type", json!("safety_evaluation")),
        ].iter().cloned().collect();
        
        log_structured(LogLevel::Info, "Safety evaluation completed", fields);
    }
    
    /// Log a performance metric
    pub fn log_performance_metric(metric_name: &str, value: f64, unit: &str) {
        let fields = [
            ("metric_name", json!(metric_name)),
            ("value", json!(value)),
            ("unit", json!(unit)),
            ("event_type", json!("performance_metric")),
        ].iter().cloned().collect();
        
        log_structured(LogLevel::Info, "Performance metric", fields);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_performance_timer() {
        let timer = performance::Timer::new("test_operation");
        std::thread::sleep(std::time::Duration::from_millis(10));
        let elapsed = timer.elapsed_ms();
        assert!(elapsed >= 10);
    }
    
    #[test]
    fn test_performance_stats() {
        let mut stats = performance::PerformanceStats::default();
        stats.record_evaluation(100, true);
        stats.record_evaluation(200, false);
        
        assert_eq!(stats.total_evaluations, 2);
        assert_eq!(stats.total_time_ms, 300);
        assert_eq!(stats.average_time_ms, 150.0);
        assert_eq!(stats.min_time_ms, 100);
        assert_eq!(stats.max_time_ms, 200);
    }
    
    #[test]
    fn test_hashing() {
        let patient_id = "patient-123";
        let hash1 = hashing::hash_patient_id(patient_id);
        let hash2 = hashing::hash_patient_id(patient_id);
        
        // Same input should produce same hash
        assert_eq!(hash1, hash2);
        
        // Hash should be different from input
        assert_ne!(hash1, patient_id);
        
        // Hash should have expected length (SHA-256 = 64 hex chars)
        assert_eq!(hash1.len(), 64);
    }
    
    #[test]
    fn test_validation() {
        // Test ICD-10 validation
        assert!(validation::validate_icd10_code("A00"));
        assert!(validation::validate_icd10_code("Z99.9"));
        assert!(!validation::validate_icd10_code("invalid"));
        
        // Test medication ID validation
        assert!(validation::validate_medication_id("aspirin"));
        assert!(validation::validate_medication_id("med_123"));
        assert!(!validation::validate_medication_id(""));
        assert!(!validation::validate_medication_id("invalid medication id with spaces"));
        
        // Test vital signs validation
        assert!(validation::validate_vital_signs(Some(120), Some(80), Some(72), Some(37.0)));
        assert!(!validation::validate_vital_signs(Some(400), Some(80), Some(72), Some(37.0))); // Invalid BP
        assert!(!validation::validate_vital_signs(Some(120), Some(80), Some(400), Some(37.0))); // Invalid HR
    }
    
    #[test]
    fn test_cache_key_generation() {
        let key1 = hashing::generate_cache_key(&["patient123", "medication", "aspirin"]);
        let key2 = hashing::generate_cache_key(&["patient123", "medication", "aspirin"]);
        let key3 = hashing::generate_cache_key(&["patient456", "medication", "aspirin"]);
        
        // Same input should produce same key
        assert_eq!(key1, key2);
        
        // Different input should produce different key
        assert_ne!(key1, key3);
    }
}