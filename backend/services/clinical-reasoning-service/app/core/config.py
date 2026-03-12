import os
from typing import List, Optional
from pydantic_settings import BaseSettings
from pydantic import Field

class Settings(BaseSettings):
    """Clinical Reasoning Service Configuration"""
    
    # Service Configuration
    PROJECT_NAME: str = "Clinical Reasoning Service"
    API_PREFIX: str = "/api"
    VERSION: str = "1.0.0"
    
    # Server Configuration
    HOST: str = Field(default="0.0.0.0", description="Server host")
    PORT: int = Field(default=8027, description="HTTP server port")
    GRPC_PORT: int = Field(default=8027, description="gRPC server port (same as HTTP)")
    
    # Database Configuration (Google Healthcare API)
    USE_GOOGLE_HEALTHCARE_API: bool = True
    GOOGLE_CLOUD_PROJECT: str = os.getenv("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8")
    GOOGLE_CLOUD_LOCATION: str = os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1")
    GOOGLE_CLOUD_DATASET: str = os.getenv("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub")
    GOOGLE_CLOUD_FHIR_STORE: str = os.getenv("GOOGLE_CLOUD_FHIR_STORE", "fhir-store")
    GOOGLE_APPLICATION_CREDENTIALS: str = os.getenv("GOOGLE_APPLICATION_CREDENTIALS", "credentials/google-credentials.json")
    
    # Authentication Configuration (Supabase)
    SUPABASE_URL: str = os.getenv("SUPABASE_URL", "https://auugxeqzgrnknklgwqrh.supabase.co")
    SUPABASE_KEY: str = os.getenv("SUPABASE_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3MzE0MTI4NzEsImV4cCI6MjA0Njk4ODg3MX0.t_TzqOOKWnqKdYQjKhNjNzNJOGJdNzNJOGJdNzNJOGJk")
    
    # External Service URLs
    API_GATEWAY_URL: str = os.getenv("API_GATEWAY_URL", "http://localhost:8005")
    APOLLO_FEDERATION_URL: str = os.getenv("APOLLO_FEDERATION_URL", "http://localhost:4000")
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001")
    FHIR_SERVICE_URL: str = os.getenv("FHIR_SERVICE_URL", "http://localhost:8014")
    PATIENT_SERVICE_URL: str = os.getenv("PATIENT_SERVICE_URL", "http://localhost:8003")
    MEDICATION_SERVICE_URL: str = os.getenv("MEDICATION_SERVICE_URL", "http://localhost:8009")
    OBSERVATION_SERVICE_URL: str = os.getenv("OBSERVATION_SERVICE_URL", "http://localhost:8007")
    CONDITION_SERVICE_URL: str = os.getenv("CONDITION_SERVICE_URL", "http://localhost:8010")
    
    # Global Outbox Service Configuration
    GLOBAL_OUTBOX_SERVICE_URL: str = os.getenv("GLOBAL_OUTBOX_SERVICE_URL", "localhost:50051")
    USE_GLOBAL_OUTBOX: bool = os.getenv("USE_GLOBAL_OUTBOX", "true").lower() == "true"

    # Neo4j Configuration for CAE Engine
    NEO4J_URI: Optional[str] = Field(default=None, description="Neo4j connection URI")
    NEO4J_USERNAME: Optional[str] = Field(default=None, description="Neo4j username")
    NEO4J_PASSWORD: Optional[str] = Field(default=None, description="Neo4j password")
    NEO4J_DATABASE: Optional[str] = Field(default="neo4j", description="Neo4j database name")
    AURA_INSTANCEID: Optional[str] = Field(default=None, description="Neo4j Aura instance ID")
    AURA_INSTANCENAME: Optional[str] = Field(default=None, description="Neo4j Aura instance name")
    NEO4J_MAX_CONNECTION_POOL_SIZE: Optional[int] = Field(default=50, description="Neo4j max connection pool size")
    NEO4J_CONNECTION_ACQUISITION_TIMEOUT: Optional[int] = Field(default=60, description="Neo4j connection timeout")
    NEO4J_MAX_CONNECTION_LIFETIME: Optional[int] = Field(default=3600, description="Neo4j max connection lifetime")
    
    # Clinical Reasoning Configuration
    DEFAULT_REASONER_TIMEOUT: int = Field(default=30, description="Default timeout for reasoners in seconds")
    MAX_CONCURRENT_REQUESTS: int = Field(default=100, description="Maximum concurrent reasoning requests")
    ENABLE_STREAMING: bool = Field(default=True, description="Enable streaming assertions")
    
    # Caching Configuration
    REDIS_URL: str = os.getenv("REDIS_URL", "redis://localhost:6379")
    CACHE_TTL_SECONDS: int = Field(default=300, description="Default cache TTL in seconds")
    ENABLE_L1_CACHE: bool = Field(default=True, description="Enable in-memory L1 cache")
    ENABLE_L2_CACHE: bool = Field(default=True, description="Enable Redis L2 cache")
    
    # Knowledge Base Configuration
    KNOWLEDGE_UPDATE_INTERVAL: int = Field(default=3600, description="Knowledge update interval in seconds")
    DRUG_INTERACTION_DB_URL: str = os.getenv("DRUG_INTERACTION_DB_URL", "")
    TERMINOLOGY_SERVICE_URL: str = os.getenv("TERMINOLOGY_SERVICE_URL", "")

    # GraphDB Configuration (Real Integration)
    GRAPHDB_ENDPOINT: str = os.getenv("GRAPHDB_ENDPOINT", "http://localhost:7200")
    GRAPHDB_REPOSITORY: str = os.getenv("GRAPHDB_REPOSITORY", "cae-clinical-intelligence")
    GRAPHDB_USERNAME: str = os.getenv("GRAPHDB_USERNAME", "")
    GRAPHDB_PASSWORD: str = os.getenv("GRAPHDB_PASSWORD", "")
    GRAPHDB_TIMEOUT: int = Field(default=30, description="GraphDB query timeout in seconds")
    GRAPHDB_MAX_RETRIES: int = Field(default=3, description="Maximum GraphDB retry attempts")

    # Learning Foundation Configuration
    LEARNING_ENABLED: bool = Field(default=True, description="Enable learning foundation")
    OUTCOME_TRACKING_ENABLED: bool = Field(default=True, description="Enable outcome tracking")
    OVERRIDE_TRACKING_ENABLED: bool = Field(default=True, description="Enable override tracking")
    LEARNING_UPDATE_INTERVAL: int = Field(default=300, description="Learning update interval in seconds")

    # Test Configuration
    PRIMARY_TEST_PATIENT_ID: str = os.getenv("PRIMARY_TEST_PATIENT_ID", "905a60cb-8241-418f-b29b-5b020e851392")
    
    # Safety Configuration
    ENABLE_SAFETY_NET: bool = Field(default=True, description="Enable safety net wrapper")
    MIN_CONFIDENCE_THRESHOLD: float = Field(default=0.7, description="Minimum confidence threshold for assertions")
    CONSERVATIVE_MODE: bool = Field(default=True, description="Enable conservative fallback mode")
    
    # Monitoring Configuration
    ENABLE_METRICS: bool = Field(default=True, description="Enable Prometheus metrics")
    METRICS_PORT: int = Field(default=9090, description="Metrics server port")
    LOG_LEVEL: str = Field(default="INFO", description="Logging level")
    
    # Performance Configuration
    GRPC_MAX_WORKERS: int = Field(default=10, description="Maximum gRPC worker threads")
    HTTP_MAX_WORKERS: int = Field(default=4, description="Maximum HTTP worker processes")
    REQUEST_TIMEOUT: int = Field(default=30, description="Request timeout in seconds")
    
    # Clinical Validation Configuration
    ENABLE_CLINICAL_VALIDATION: bool = Field(default=True, description="Enable clinical validation")
    VALIDATION_SAMPLE_RATE: float = Field(default=0.1, description="Fraction of requests to validate")
    EXPERT_REVIEW_THRESHOLD: float = Field(default=0.8, description="Confidence threshold for expert review")
    
    class Config:
        env_file = ".env"
        case_sensitive = True

# Global settings instance
settings = Settings()
