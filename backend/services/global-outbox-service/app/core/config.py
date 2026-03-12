import os
from typing import List, Optional
from pydantic_settings import BaseSettings
from pydantic import Field

class Settings(BaseSettings):
    """Global Outbox Service Configuration"""
    
    # Service Configuration
    PROJECT_NAME: str = "Global Outbox Service"
    API_PREFIX: str = "/api"
    VERSION: str = "1.0.0"
    
    # Server Configuration
    HOST: str = Field(default="0.0.0.0", description="Server host")
    PORT: int = Field(default=8040, description="HTTP server port")
    GRPC_PORT: int = Field(default=50051, description="gRPC server port")
    
    # Database Configuration (Supabase PostgreSQL)
    DATABASE_URL: str = Field(
        default="postgresql://postgres.auugxeqzgrnknklgwqrh:9FTqQnA4LRCsu8sw@aws-0-ap-south-1.pooler.supabase.com:5432/postgres",
        description="PostgreSQL connection string"
    )
    DATABASE_POOL_SIZE: int = Field(default=20, description="Database connection pool size")
    DATABASE_MAX_OVERFLOW: int = Field(default=30, description="Database connection pool overflow")
    DATABASE_POOL_TIMEOUT: int = Field(default=30, description="Database connection timeout")
    
    # Kafka Configuration (Confluent Cloud)
    KAFKA_BOOTSTRAP_SERVERS: str = Field(
        default="pkc-619z3.us-east1.gcp.confluent.cloud:9092",
        description="Kafka bootstrap servers"
    )
    KAFKA_API_KEY: str = Field(default="LGJ3AQ2L6VRPW4S2", description="Kafka API key")
    KAFKA_API_SECRET: str = Field(default="2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl", description="Kafka API secret")
    KAFKA_SECURITY_PROTOCOL: str = Field(default="SASL_SSL", description="Kafka security protocol")
    KAFKA_SASL_MECHANISM: str = Field(default="PLAIN", description="Kafka SASL mechanism")
    
    # Publisher Configuration
    PUBLISHER_ENABLED: bool = Field(default=True, description="Enable background publisher")
    PUBLISHER_POLL_INTERVAL: int = Field(default=2, description="Publisher polling interval in seconds")
    PUBLISHER_BATCH_SIZE: int = Field(default=100, description="Publisher batch size")
    PUBLISHER_MAX_WORKERS: int = Field(default=4, description="Publisher worker threads")
    
    # Retry Configuration
    MAX_RETRY_ATTEMPTS: int = Field(default=5, description="Maximum retry attempts")
    RETRY_BASE_DELAY: float = Field(default=1.0, description="Base retry delay in seconds")
    RETRY_MAX_DELAY: float = Field(default=60.0, description="Maximum retry delay in seconds")
    RETRY_EXPONENTIAL_BASE: float = Field(default=2.0, description="Exponential backoff base")
    RETRY_JITTER: bool = Field(default=True, description="Add jitter to retry delays")
    
    # Dead Letter Queue Configuration
    DLQ_ENABLED: bool = Field(default=True, description="Enable dead letter queue")
    DLQ_MAX_RETRIES: int = Field(default=10, description="Max retries before DLQ")

    # Medical Circuit Breaker Configuration
    MEDICAL_CIRCUIT_BREAKER_ENABLED: bool = Field(default=True, description="Enable medical-aware circuit breaker")
    MEDICAL_CIRCUIT_BREAKER_MAX_QUEUE_DEPTH: int = Field(default=1000, description="Max queue depth before load shedding")
    MEDICAL_CIRCUIT_BREAKER_CRITICAL_THRESHOLD: float = Field(default=0.8, description="Critical load threshold (0.0-1.0)")
    MEDICAL_CIRCUIT_BREAKER_RECOVERY_TIMEOUT: int = Field(default=30, description="Recovery timeout in seconds")
    
    # Security Configuration
    GRPC_API_KEY: str = Field(default="global-outbox-service-key", description="gRPC API key")
    ENABLE_AUTH: bool = Field(default=False, description="Enable authentication")
    
    # Monitoring Configuration
    ENABLE_METRICS: bool = Field(default=True, description="Enable Prometheus metrics")
    METRICS_PORT: int = Field(default=8041, description="Metrics server port")
    LOG_LEVEL: str = Field(default="INFO", description="Logging level")
    
    # Environment Configuration
    ENVIRONMENT: str = Field(default="development", description="Environment name")
    DEBUG: bool = Field(default=False, description="Debug mode")
    
    # Supported Services (for partition creation)
    SUPPORTED_SERVICES: List[str] = Field(
        default=[
            "patient-service",
            "observation-service", 
            "condition-service",
            "medication-service",
            "encounter-service",
            "timeline-service",
            "workflow-engine-service",
            "order-management-service",
            "scheduling-service",
            "organization-service",
            "device-data-ingestion-service",
            "lab-service",
            "fhir-service",
            "generic-service"  # Fallback for new services
        ],
        description="List of supported microservices"
    )
    
    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"
        case_sensitive = True
        
    def get_database_url(self) -> str:
        """Get the database URL with proper formatting"""
        return self.DATABASE_URL
    
    def get_kafka_config(self) -> dict:
        """Get Kafka configuration dictionary"""
        return {
            "bootstrap.servers": self.KAFKA_BOOTSTRAP_SERVERS,
            "security.protocol": self.KAFKA_SECURITY_PROTOCOL,
            "sasl.mechanism": self.KAFKA_SASL_MECHANISM,
            "sasl.username": self.KAFKA_API_KEY,
            "sasl.password": self.KAFKA_API_SECRET,
            "client.id": f"{self.PROJECT_NAME.lower().replace(' ', '-')}-producer",
            "acks": "all",
            "retries": 3,
            "retry.backoff.ms": 1000,
            "request.timeout.ms": 30000,
            "delivery.timeout.ms": 120000,
        }
    
    def is_production(self) -> bool:
        """Check if running in production environment"""
        return self.ENVIRONMENT.lower() == "production"
    
    def is_development(self) -> bool:
        """Check if running in development environment"""
        return self.ENVIRONMENT.lower() == "development"

# Global settings instance
settings = Settings()

# Environment-specific overrides
if settings.is_production():
    settings.DEBUG = False
    settings.LOG_LEVEL = "INFO"
elif settings.is_development():
    settings.DEBUG = True
    settings.LOG_LEVEL = "DEBUG"
