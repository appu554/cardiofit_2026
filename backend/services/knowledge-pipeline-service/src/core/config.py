"""
Configuration settings for Knowledge Pipeline Service
"""

import os
from typing import List, Optional
from pydantic import Field
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings"""
    
    # Service configuration
    PROJECT_NAME: str = "Knowledge Pipeline Service"
    VERSION: str = "1.0.0"
    DEBUG: bool = Field(default=False, description="Debug mode")
    HOST: str = Field(default="0.0.0.0", description="Host to bind to")
    PORT: int = Field(default=8031, description="Port to bind to")
    
    # GraphDB Configuration (using existing CAE GraphDB instance)
    GRAPHDB_ENDPOINT: str = Field(
        default="http://localhost:7200",
        description="GraphDB endpoint URL"
    )
    GRAPHDB_REPOSITORY: str = Field(
        default="cae-clinical-intelligence",
        description="GraphDB repository name"
    )
    GRAPHDB_USERNAME: str = Field(default="", description="GraphDB username")
    GRAPHDB_PASSWORD: str = Field(default="", description="GraphDB password")
    GRAPHDB_TIMEOUT: int = Field(default=30, description="GraphDB query timeout in seconds")
    GRAPHDB_MAX_RETRIES: int = Field(default=3, description="Maximum GraphDB retry attempts")

    # Database Type Selection
    DATABASE_TYPE: str = Field(
        default="neo4j",  # Options: "graphdb", "neo4j"
        description="Database type to use for knowledge graph"
    )

    # Neo4j Cloud Configuration
    NEO4J_URI: str = Field(
        default="neo4j+s://52721fa5.databases.neo4j.io",
        description="Neo4j Cloud (AuraDB) connection URI"
    )

    NEO4J_USERNAME: str = Field(
        default="neo4j",
        description="Neo4j Cloud username"
    )

    NEO4J_PASSWORD: str = Field(
        default="Wy5lxkHowS66L8rCnyGQG-XAdX1JIihsYj_vfIT8KNw",
        description="Neo4j Cloud password"
    )

    NEO4J_DATABASE: str = Field(
        default="neo4j",
        description="Neo4j Cloud database name"
    )

    # Real data source URLs - NO FALLBACKS
    RXNORM_DOWNLOAD_URL: str = Field(
        default="https://download.nlm.nih.gov/umls/kss/rxnorm/RxNorm_full_current.zip",
        description="RxNorm full release download URL (REAL DATA REQUIRED)"
    )

    CREDIBLEMEDS_QT_URL: str = Field(
        default="https://www.crediblemeds.org/pdftemp/pdf/CombinedList.pdf",
        description="CredibleMeds QT drug list URL (REAL DATA REQUIRED)"
    )

    AHRQ_CDS_CONNECT_URL: str = Field(
        default="https://cds.ahrq.gov/cdsconnect/artifacts",
        description="AHRQ CDS Connect artifacts base URL (REAL DATA REQUIRED)"
    )

    DRUGBANK_DOWNLOAD_URL: str = Field(
        default="https://go.drugbank.com/releases/latest#open-data",
        description="DrugBank Academic download URL (MANUAL DOWNLOAD REQUIRED)"
    )

    UMLS_DOWNLOAD_URL: str = Field(
        default="https://www.nlm.nih.gov/research/umls/licensedcontent/umlsknowledgesources.html",
        description="UMLS Metathesaurus download URL (UMLS LICENSE REQUIRED)"
    )

    SNOMED_DOWNLOAD_URL: str = Field(
        default="https://www.nlm.nih.gov/healthit/snomedct/international.html",
        description="SNOMED CT download URL (SNOMED LICENSE REQUIRED)"
    )

    LOINC_DOWNLOAD_URL: str = Field(
        default="https://loinc.org/downloads/",
        description="LOINC download URL (LOINC LICENSE REQUIRED)"
    )

    OPENFDA_API_URL: str = Field(
        default="https://api.fda.gov/drug/event.json",
        description="OpenFDA FAERS API URL (LIVE API ACCESS REQUIRED)"
    )

    # Optional API keys for enhanced access
    OPENFDA_API_KEY: Optional[str] = Field(
        default=None,
        description="OpenFDA API key for higher rate limits (optional but recommended)"
    )
    
    # Data processing configuration
    DATA_DIR: str = Field(
        default="./data",
        description="Directory for downloaded and processed data"
    )
    TEMP_DIR: str = Field(
        default="./temp",
        description="Temporary directory for processing"
    )
    CACHE_DIR: str = Field(
        default="./cache",
        description="Cache directory for processed data"
    )
    
    # Processing limits and performance
    MAX_CONCURRENT_DOWNLOADS: int = Field(
        default=5,
        description="Maximum concurrent downloads"
    )
    MAX_BATCH_SIZE: int = Field(
        default=1000,
        description="Maximum batch size for GraphDB inserts"
    )
    DOWNLOAD_TIMEOUT: int = Field(
        default=300,
        description="Download timeout in seconds"
    )
    
    # RxNorm specific configuration
    RXNORM_PROCESS_TABLES: List[str] = Field(
        default=[
            "RXNCONSO.RRF",  # Concept names and sources
            "RXNREL.RRF",    # Relationships
            "RXNSAT.RRF",    # Simple attributes
            "RXNCUI.RRF"     # Concept unique identifiers
        ],
        description="RxNorm RRF tables to process"
    )
    
    # CredibleMeds configuration
    CREDIBLEMEDS_CATEGORIES: List[str] = Field(
        default=[
            "Known Risk of TdP",
            "Possible Risk of TdP", 
            "Conditional Risk of TdP"
        ],
        description="CredibleMeds QT risk categories to process"
    )
    
    # AHRQ CDS Connect configuration
    AHRQ_ARTIFACT_TYPES: List[str] = Field(
        default=[
            "clinical-decision-support",
            "clinical-pathway",
            "order-set",
            "clinical-guideline"
        ],
        description="AHRQ artifact types to process"
    )
    
    # Clinical ontology configuration
    CLINICAL_ONTOLOGY_BASE_URI: str = Field(
        default="http://clinical-assertion-engine.org/ontology/",
        description="Base URI for clinical ontology"
    )
    
    # Redis configuration for caching
    REDIS_URL: str = Field(
        default="redis://localhost:6379",
        description="Redis URL for caching"
    )
    REDIS_TTL: int = Field(
        default=3600,
        description="Redis cache TTL in seconds"
    )
    
    # Logging configuration
    LOG_LEVEL: str = Field(default="INFO", description="Logging level")
    LOG_FORMAT: str = Field(default="json", description="Log format (json or text)")
    
    # Monitoring and metrics
    ENABLE_METRICS: bool = Field(default=True, description="Enable Prometheus metrics")
    METRICS_PORT: int = Field(default=8031, description="Metrics server port")
    
    # Security
    API_KEY_HEADER: str = Field(default="X-API-Key", description="API key header name")
    ALLOWED_HOSTS: List[str] = Field(default=["*"], description="Allowed hosts")
    
    class Config:
        env_file = ".env"
        case_sensitive = True


# Global settings instance
settings = Settings()
