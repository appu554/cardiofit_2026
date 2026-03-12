use tonic::{Request, Response, Status};
use std::sync::Arc;

use crate::services::ClinicalDataHubService;

/// gRPC service implementation
pub struct GrpcService {
    service: Arc<ClinicalDataHubService>,
}

impl GrpcService {
    pub fn new(service: Arc<ClinicalDataHubService>) -> Self {
        Self { service }
    }
}

// TODO: Implement the actual gRPC methods once proto compilation is fixed