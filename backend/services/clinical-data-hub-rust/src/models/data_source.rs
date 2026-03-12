use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataSource {
    pub id: String,
    pub name: String,
    pub source_type: DataSourceType,
    pub connection_string: String,
    pub metadata: HashMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DataSourceType {
    PostgreSQL,
    MongoDB,
    Redis,
    Kafka,
    FHIR,
    GraphQL,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataQuery {
    pub source_id: String,
    pub query_type: QueryType,
    pub parameters: HashMap<String, String>,
    pub timeout_ms: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum QueryType {
    SQL,
    NoSQL,
    GraphQL,
    REST,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataResult {
    pub source_id: String,
    pub data: Vec<u8>,
    pub metadata: HashMap<String, String>,
    pub timestamp: i64,
}