// FFI (Foreign Function Interface) layer for Go-Rust integration
//
// This module provides C-compatible functions that can be called from Go
// using cgo. It handles memory management, string conversions, and error
// propagation across the FFI boundary.
//
// Safety considerations:
// - All pointers passed from Go must be valid for the duration of the call
// - All strings allocated in Rust must be freed by calling the appropriate free function
// - Error codes are used for error propagation instead of exceptions

use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int, c_double, c_ulong};
use std::ptr;
use std::panic;

use crate::{initialize_global_engine, with_global_engine, shutdown_global_engine};
use crate::types::{SafetyRequest, SafetyResult, SafetyStatus};
use crate::cae::CAEConfig;

/// C-compatible safety request structure
#[repr(C)]
pub struct CSafetyRequest {
    pub patient_id: *const c_char,
    pub request_id: *const c_char,
    pub action_type: *const c_char,
    pub priority: *const c_char,
    pub medication_ids_json: *const c_char,
    pub condition_ids_json: *const c_char,
    pub allergy_ids_json: *const c_char,
}

/// C-compatible safety result structure
#[repr(C)]
pub struct CSafetyResult {
    pub status: c_int,              // 0=Safe, 1=Unsafe, 2=Warning, 3=ManualReview
    pub risk_score: c_double,
    pub confidence: c_double,
    pub violations_json: *mut c_char,
    pub warnings_json: *mut c_char,
    pub processing_time_ms: c_ulong,
}

/// Error codes for FFI functions
pub const CAE_SUCCESS: c_int = 0;
pub const CAE_ERROR_INVALID_INPUT: c_int = 1;
pub const CAE_ERROR_ENGINE_NOT_INITIALIZED: c_int = 2;
pub const CAE_ERROR_EVALUATION_FAILED: c_int = 3;
pub const CAE_ERROR_JSON_PARSING: c_int = 4;
pub const CAE_ERROR_PANIC: c_int = 5;

/// Initialize the CAE engine with JSON configuration
///
/// # Safety
/// - `config_json` must be a valid null-terminated C string
/// - Returns CAE_SUCCESS on success, error code on failure
#[no_mangle]
pub extern "C" fn cae_initialize_engine(config_json: *const c_char) -> c_int {
    let result = panic::catch_unwind(|| {
        if config_json.is_null() {
            return CAE_ERROR_INVALID_INPUT;
        }

        let config_str = match unsafe { CStr::from_ptr(config_json) }.to_str() {
            Ok(s) => s,
            Err(_) => return CAE_ERROR_INVALID_INPUT,
        };

        let config: CAEConfig = match serde_json::from_str(config_str) {
            Ok(c) => c,
            Err(_) => return CAE_ERROR_JSON_PARSING,
        };

        match initialize_global_engine(config) {
            Ok(_) => CAE_SUCCESS,
            Err(_) => CAE_ERROR_EVALUATION_FAILED,
        }
    });

    result.unwrap_or(CAE_ERROR_PANIC)
}

/// Evaluate safety using the CAE engine
///
/// # Safety
/// - `request` must be a valid pointer to CSafetyRequest
/// - All string fields in request must be valid null-terminated C strings
/// - Returns null on error, valid pointer on success
/// - Caller must free the result using cae_free_result
#[no_mangle]
pub extern "C" fn cae_evaluate_safety(request: *const CSafetyRequest) -> *mut CSafetyResult {
    let result = panic::catch_unwind(|| {
        if request.is_null() {
            return ptr::null_mut();
        }

        let request_ref = unsafe { &*request };
        
        // Convert C request to Rust SafetyRequest
        let safety_request = match convert_c_request_to_rust(request_ref) {
            Ok(req) => req,
            Err(_) => return ptr::null_mut(),
        };

        // Perform safety evaluation
        let start_time = std::time::Instant::now();
        let evaluation_result = with_global_engine(|engine| {
            engine.evaluate_safety(&safety_request)
        });
        let processing_time = start_time.elapsed().as_millis() as u64;

        let mut result = match evaluation_result {
            Ok(r) => r,
            Err(_) => return ptr::null_mut(),
        };
        
        // Update processing time
        result.processing_time_ms = processing_time;

        // Convert Rust result to C result
        match convert_rust_result_to_c(result) {
            Ok(c_result) => Box::into_raw(Box::new(c_result)),
            Err(_) => ptr::null_mut(),
        }
    });

    result.unwrap_or(ptr::null_mut())
}

/// Free a safety result allocated by cae_evaluate_safety
///
/// # Safety
/// - `result` must be a valid pointer returned by cae_evaluate_safety
/// - `result` must not be used after this call
#[no_mangle]
pub extern "C" fn cae_free_result(result: *mut CSafetyResult) {
    if result.is_null() {
        return;
    }

    let _ = panic::catch_unwind(|| {
        let result_box = unsafe { Box::from_raw(result) };
        
        // Free allocated strings
        if !result_box.violations_json.is_null() {
            unsafe {
                let _ = CString::from_raw(result_box.violations_json);
            }
        }
        
        if !result_box.warnings_json.is_null() {
            unsafe {
                let _ = CString::from_raw(result_box.warnings_json);
            }
        }
    });
}

/// Shutdown the CAE engine and free resources
#[no_mangle]
pub extern "C" fn cae_shutdown_engine() {
    let _ = panic::catch_unwind(|| {
        shutdown_global_engine();
    });
}

/// Get the version of the safety engines library
///
/// # Safety
/// - Returns a static string, caller should not free
#[no_mangle]
pub extern "C" fn cae_get_version() -> *const c_char {
    "0.1.0\0".as_ptr() as *const c_char
}

/// Health check function for the CAE engine
///
/// Returns CAE_SUCCESS if engine is healthy, error code otherwise
#[no_mangle]
pub extern "C" fn cae_health_check() -> c_int {
    let result = panic::catch_unwind(|| {
        // Simple health check - try to create a minimal request
        let test_request = SafetyRequest {
            patient_id: "health_check".to_string(),
            request_id: "health_check_001".to_string(),
            medication_ids: vec![],
            condition_ids: vec![],
            allergy_ids: vec![],
            action_type: "health_check".to_string(),
            priority: "low".to_string(),
        };

        match with_global_engine(|engine| engine.evaluate_safety(&test_request)) {
            Ok(_) => CAE_SUCCESS,
            Err(_) => CAE_ERROR_EVALUATION_FAILED,
        }
    });

    result.unwrap_or(CAE_ERROR_PANIC)
}

// Helper functions for type conversion

fn convert_c_request_to_rust(c_request: &CSafetyRequest) -> Result<SafetyRequest, Box<dyn std::error::Error>> {
    let patient_id = c_str_to_string(c_request.patient_id)?;
    let request_id = c_str_to_string(c_request.request_id)?;
    let action_type = c_str_to_string(c_request.action_type)?;
    let priority = c_str_to_string(c_request.priority)?;
    
    let medication_ids: Vec<String> = if c_request.medication_ids_json.is_null() {
        Vec::new()
    } else {
        let json_str = c_str_to_string(c_request.medication_ids_json)?;
        serde_json::from_str(&json_str)?
    };
    
    let condition_ids: Vec<String> = if c_request.condition_ids_json.is_null() {
        Vec::new()
    } else {
        let json_str = c_str_to_string(c_request.condition_ids_json)?;
        serde_json::from_str(&json_str)?
    };
    
    let allergy_ids: Vec<String> = if c_request.allergy_ids_json.is_null() {
        Vec::new()
    } else {
        let json_str = c_str_to_string(c_request.allergy_ids_json)?;
        serde_json::from_str(&json_str)?
    };

    Ok(SafetyRequest {
        patient_id,
        request_id,
        medication_ids,
        condition_ids,
        allergy_ids,
        action_type,
        priority,
    })
}

fn convert_rust_result_to_c(rust_result: SafetyResult) -> Result<CSafetyResult, Box<dyn std::error::Error>> {
    let status = match rust_result.status {
        SafetyStatus::Safe => 0,
        SafetyStatus::Unsafe => 1,
        SafetyStatus::Warning => 2,
        SafetyStatus::ManualReview => 3,
    };

    let violations_json = if rust_result.violations.is_empty() {
        ptr::null_mut()
    } else {
        let json_str = serde_json::to_string(&rust_result.violations)?;
        CString::new(json_str)?.into_raw()
    };

    let warnings_json = if rust_result.warnings.is_empty() {
        ptr::null_mut()
    } else {
        let json_str = serde_json::to_string(&rust_result.warnings)?;
        CString::new(json_str)?.into_raw()
    };

    Ok(CSafetyResult {
        status,
        risk_score: rust_result.risk_score,
        confidence: rust_result.confidence,
        violations_json,
        warnings_json,
        processing_time_ms: rust_result.processing_time_ms,
    })
}

fn c_str_to_string(c_str: *const c_char) -> Result<String, Box<dyn std::error::Error>> {
    if c_str.is_null() {
        return Ok(String::new());
    }
    
    let cstr = unsafe { CStr::from_ptr(c_str) };
    Ok(cstr.to_str()?.to_owned())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ffi::CString;
    
    #[test]
    fn test_ffi_initialization() {
        let config = r#"{"rules_path": "test", "knowledge_db_path": "test", "cache_size": 1000}"#;
        let config_cstr = CString::new(config).unwrap();
        
        let result = cae_initialize_engine(config_cstr.as_ptr());
        assert_eq!(result, CAE_SUCCESS);
        
        cae_shutdown_engine();
    }

    #[test]
    fn test_version_function() {
        let version_ptr = cae_get_version();
        assert!(!version_ptr.is_null());
        
        let version_str = unsafe { CStr::from_ptr(version_ptr) };
        assert_eq!(version_str.to_str().unwrap(), "0.1.0");
    }

    #[test]
    fn test_c_request_conversion() {
        let patient_id = CString::new("patient-123").unwrap();
        let request_id = CString::new("req-001").unwrap();
        let action_type = CString::new("medication_order").unwrap();
        let priority = CString::new("normal").unwrap();
        let meds_json = CString::new(r#"["aspirin", "warfarin"]"#).unwrap();
        
        let c_request = CSafetyRequest {
            patient_id: patient_id.as_ptr(),
            request_id: request_id.as_ptr(),
            action_type: action_type.as_ptr(),
            priority: priority.as_ptr(),
            medication_ids_json: meds_json.as_ptr(),
            condition_ids_json: ptr::null(),
            allergy_ids_json: ptr::null(),
        };
        
        let rust_request = convert_c_request_to_rust(&c_request).unwrap();
        assert_eq!(rust_request.patient_id, "patient-123");
        assert_eq!(rust_request.medication_ids.len(), 2);
        assert_eq!(rust_request.medication_ids[0], "aspirin");
        assert_eq!(rust_request.medication_ids[1], "warfarin");
    }
}