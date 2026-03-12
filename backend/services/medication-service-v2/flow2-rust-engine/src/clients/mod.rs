//! HTTP clients for external service integration
//! 
//! This module provides HTTP clients for integrating with external services
//! such as the Context Gateway for snapshot-based processing.

pub mod snapshot_client;

// Re-export public types
pub use snapshot_client::*;