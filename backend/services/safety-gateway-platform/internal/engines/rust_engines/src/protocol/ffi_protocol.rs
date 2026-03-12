//! FFI Bridge for Protocol Engine
//!
//! This module provides C-compatible FFI interface for Go integration
//! with the Protocol Engine.

use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int};
use std::ptr;
use serde_json;

use crate::protocol::{
    types::*,
    error::*,
    initialize_protocol_engine,
    with_protocol_engine,
    shutdown_protocol_engine,
};

/// FFI result codes
pub const PROTOCOL_ENGINE_SUCCESS: c_int = 0;
pub const PROTOCOL_ENGINE_ERROR: c_int = -1;
pub const PROTOCOL_ENGINE_TIMEOUT: c_int = -2;
pub const PROTOCOL_ENGINE_NOT_FOUND: c_int = -3;

/// Initialize Protocol Engine for a tenant
#[no_mangle]
pub extern "C" fn rust_protocol_engine_init(
    tenant_id: *const c_char,
    config_json: *const c_char,
) -> c_int {
    if tenant_id.is_null() || config_json.is_null() {
        return PROTOCOL_ENGINE_ERROR;
    }
    
    let tenant_id = match unsafe { CStr::from_ptr(tenant_id) }.to_str() {
        Ok(s) => s.to_string(),
        Err(_) => return PROTOCOL_ENGINE_ERROR,
    };
    
    let config_str = match unsafe { CStr::from_ptr(config_json) }.to_str() {
        Ok(s) => s,
        Err(_) => return PROTOCOL_ENGINE_ERROR,
    };
    
    let config: ProtocolEngineConfig = match serde_json::from_str(config_str) {
        Ok(c) => c,
        Err(_) => return PROTOCOL_ENGINE_ERROR,
    };
    
    match initialize_protocol_engine(tenant_id, config) {
        Ok(_) => PROTOCOL_ENGINE_SUCCESS,
        Err(_) => PROTOCOL_ENGINE_ERROR,
    }
}

/// Evaluate protocol (async - returns immediately)
#[no_mangle]
pub extern "C" fn rust_protocol_engine_evaluate(
    tenant_id: *const c_char,
    request_json: *const c_char,
    result_json: *mut *mut c_char,
) -> c_int {
    if tenant_id.is_null() || request_json.is_null() || result_json.is_null() {
        return PROTOCOL_ENGINE_ERROR;
    }
    
    let tenant_id = match unsafe { CStr::from_ptr(tenant_id) }.to_str() {
        Ok(s) => s,
        Err(_) => return PROTOCOL_ENGINE_ERROR,
    };
    
    let request_str = match unsafe { CStr::from_ptr(request_json) }.to_str() {
        Ok(s) => s,
        Err(_) => return PROTOCOL_ENGINE_ERROR,
    };
    
    let request: ProtocolEvaluationRequest = match serde_json::from_str(request_str) {
        Ok(r) => r,
        Err(_) => return PROTOCOL_ENGINE_ERROR,
    };
    
    // For now, return a stub result
    // In full implementation, this would use async evaluation
    let stub_result = ProtocolEvaluationResult {
        result_id: uuid::Uuid::new_v4(),
        request_id: request.metadata.as_ref().map(|m| m.request_id),
        protocol_id: request.protocol_id.clone(),
        decision: ProtocolDecision {
            decision_type: ProtocolDecisionType::Allow,
            confidence: 1.0,
            reasoning: "Stub implementation".to_string(),
            requires_approval: false,
            override_available: false,
        },
        evaluation_details: EvaluationDetails {
            rules_evaluated: vec![],
            constraints_checked: vec![],
            conditions_met: vec![],
            conditions_failed: vec![],
            warnings: vec![],
            information: vec![],
        },
        state_changes: vec![],
        temporal_constraints: vec![],
        recommendations: vec![],
        performance_metrics: EvaluationMetrics {
            total_execution_time_ms: 1,
            rules_execution_time_ms: 0,
            constraints_execution_time_ms: 0,
            state_machine_time_ms: 0,
            snapshot_resolution_time_ms: 0,
            memory_usage_bytes: None,
        },
        evaluated_at: chrono::Utc::now(),
        snapshot_context: None,
    };
    
    let result_json_str = match serde_json::to_string(&stub_result) {
        Ok(s) => s,
        Err(_) => return PROTOCOL_ENGINE_ERROR,
    };
    
    let result_cstring = match CString::new(result_json_str) {
        Ok(s) => s,
        Err(_) => return PROTOCOL_ENGINE_ERROR,
    };
    
    unsafe {
        *result_json = result_cstring.into_raw();
    }
    
    PROTOCOL_ENGINE_SUCCESS
}

/// Shutdown Protocol Engine for a tenant
#[no_mangle]
pub extern "C" fn rust_protocol_engine_shutdown(tenant_id: *const c_char) -> c_int {
    if tenant_id.is_null() {
        return PROTOCOL_ENGINE_ERROR;
    }
    
    let tenant_id = match unsafe { CStr::from_ptr(tenant_id) }.to_str() {
        Ok(s) => s,
        Err(_) => return PROTOCOL_ENGINE_ERROR,
    };
    
    match shutdown_protocol_engine(tenant_id) {
        Ok(_) => PROTOCOL_ENGINE_SUCCESS,
        Err(_) => PROTOCOL_ENGINE_NOT_FOUND,
    }
}

/// Free memory allocated by Rust
#[no_mangle]
pub extern "C" fn rust_protocol_engine_free_string(s: *mut c_char) {
    if s.is_null() {
        return;
    }
    
    unsafe {
        let _ = CString::from_raw(s);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ptr;

    #[test]
    fn test_ffi_interface() {
        // Test basic FFI functions compile
        let tenant_id = CString::new("test-tenant").unwrap();
        let config = ProtocolEngineConfig::test_config();
        let config_json = serde_json::to_string(&config).unwrap();
        let config_cstring = CString::new(config_json).unwrap();
        
        let result = rust_protocol_engine_init(
            tenant_id.as_ptr(),
            config_cstring.as_ptr(),
        );
        
        assert_eq!(result, PROTOCOL_ENGINE_SUCCESS);
        
        // Cleanup
        let shutdown_result = rust_protocol_engine_shutdown(tenant_id.as_ptr());
        assert_eq!(shutdown_result, PROTOCOL_ENGINE_SUCCESS);
    }
}