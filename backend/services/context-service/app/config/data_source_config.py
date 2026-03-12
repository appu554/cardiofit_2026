"""
Data Source Configuration for Clinical Context Service
Allows switching between Direct Elasticsearch and Microservice connections
"""
import os
from typing import Dict, Any
from enum import Enum


class DataSourceMode(Enum):
    """Data source connection modes"""
    FHIR_STORE_DIRECT = "fhir_store_direct"  # Direct FHIR Store (best for clinical data)
    ELASTICSEARCH_DIRECT = "elasticsearch_direct"  # Direct Elasticsearch (best for device data)
    MICROSERVICES = "microservices"  # Traditional microservices
    HYBRID = "hybrid"  # Try FHIR Store first, then Elasticsearch, then microservices
    SMART_ROUTING = "smart_routing"  # Intelligent routing based on data type


class DataSourceConfig:
    """Configuration for data source connections"""
    
    def __init__(self):
        # 🏥 PRIMARY MODE: Smart Routing (BEST - chooses optimal source per data type)
        self.mode = DataSourceMode.SMART_ROUTING

        # Override from environment variable
        mode_env = os.getenv("CONTEXT_SERVICE_DATA_MODE", "smart_routing").lower()
        if mode_env == "fhir_store_direct":
            self.mode = DataSourceMode.FHIR_STORE_DIRECT
        elif mode_env == "elasticsearch_direct":
            self.mode = DataSourceMode.ELASTICSEARCH_DIRECT
        elif mode_env == "microservices":
            self.mode = DataSourceMode.MICROSERVICES
        elif mode_env == "hybrid":
            self.mode = DataSourceMode.HYBRID
        
        # FHIR Store Configuration (Your Google Cloud Healthcare API)
        self.fhir_store_config = {
            "project_id": os.getenv("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8"),
            "location": os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1"),
            "dataset_id": os.getenv("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub"),
            "fhir_store_id": os.getenv("GOOGLE_CLOUD_FHIR_STORE", "fhir-store"),
            "credentials_path": os.getenv("GOOGLE_APPLICATION_CREDENTIALS", "../services/encounter-service/credentials/google-credentials.json")
        }

        # Elasticsearch Configuration (Your Elastic Cloud)
        self.elasticsearch_config = {
            "hosts": [os.getenv(
                "ELASTICSEARCH_URL", 
                "https://my-elasticsearch-project-ba1a02.es.us-east-1.aws.elastic.cloud:443"
            )],
            "api_key": os.getenv(
                "ELASTICSEARCH_API_KEY",
                "d0gyTG5aY0JGajhWTVBOTzkzeDk6VGxoNENEd29DZEtERXBxRXpRUXBEUQ=="
            ),
            "verify_certs": True,
            "ssl_show_warn": False,
            "timeout": int(os.getenv("ELASTICSEARCH_TIMEOUT", "30")),
            "max_retries": int(os.getenv("ELASTICSEARCH_MAX_RETRIES", "3")),
            "retry_on_timeout": True
        }
        
        # Index mappings for different data types
        self.elasticsearch_indices = {
            "patient_demographics": os.getenv("ES_INDEX_PATIENTS", "patient-readings*"),
            "patient_medications": os.getenv("ES_INDEX_MEDICATIONS", "patient-readings*"),
            "patient_conditions": os.getenv("ES_INDEX_CONDITIONS", "patient-readings*"),
            "patient_allergies": os.getenv("ES_INDEX_ALLERGIES", "patient-readings*"),
            "lab_results": os.getenv("ES_INDEX_LABS", "patient-readings*"),
            "vital_signs": os.getenv("ES_INDEX_VITALS", "patient-readings*"),
            "fhir_observations": os.getenv("ES_INDEX_FHIR", "fhir-observations*"),
            "device_readings": os.getenv("ES_INDEX_DEVICES", "patient-readings*")
        }
        
        # Microservice endpoints (fallback)
        self.microservice_endpoints = {
            "patient_service": os.getenv("PATIENT_SERVICE_URL", "http://localhost:8003"),
            "medication_service": os.getenv("MEDICATION_SERVICE_URL", "http://localhost:8009"),
            "lab_service": os.getenv("LAB_SERVICE_URL", "http://localhost:8000"),
            "condition_service": os.getenv("CONDITION_SERVICE_URL", "http://localhost:8010"),
            "encounter_service": os.getenv("ENCOUNTER_SERVICE_URL", "http://localhost:8020"),
            "observation_service": os.getenv("OBSERVATION_SERVICE_URL", "http://localhost:8007"),
            "cae_service": os.getenv("CAE_SERVICE_URL", "http://localhost:8027"),
            "auth_service": os.getenv("AUTH_SERVICE_URL", "http://localhost:8001")
        }
        
        # Performance settings
        self.performance_config = {
            "elasticsearch_timeout_ms": int(os.getenv("ES_TIMEOUT_MS", "5000")),
            "microservice_timeout_ms": int(os.getenv("MS_TIMEOUT_MS", "10000")),
            "max_concurrent_requests": int(os.getenv("MAX_CONCURRENT", "10")),
            "cache_ttl_seconds": int(os.getenv("CACHE_TTL", "300")),
            "retry_attempts": int(os.getenv("RETRY_ATTEMPTS", "3"))
        }
    
    def should_use_fhir_store(self) -> bool:
        """Check if FHIR Store should be used"""
        return self.mode in [DataSourceMode.FHIR_STORE_DIRECT, DataSourceMode.HYBRID, DataSourceMode.SMART_ROUTING]

    def should_use_elasticsearch(self) -> bool:
        """Check if Elasticsearch should be used"""
        return self.mode in [DataSourceMode.ELASTICSEARCH_DIRECT, DataSourceMode.HYBRID, DataSourceMode.SMART_ROUTING]

    def should_use_microservices(self) -> bool:
        """Check if microservices should be used"""
        return self.mode in [DataSourceMode.MICROSERVICES, DataSourceMode.HYBRID, DataSourceMode.SMART_ROUTING]
    
    def get_fhir_store_config(self) -> Dict[str, Any]:
        """Get FHIR Store configuration"""
        return self.fhir_store_config.copy()

    def get_elasticsearch_config(self) -> Dict[str, Any]:
        """Get Elasticsearch configuration"""
        return self.elasticsearch_config.copy()
    
    def get_microservice_url(self, service_name: str) -> str:
        """Get microservice URL"""
        return self.microservice_endpoints.get(service_name, "")
    
    def get_elasticsearch_index(self, data_type: str) -> str:
        """Get Elasticsearch index for data type"""
        return self.elasticsearch_indices.get(data_type, "patient-readings*")
    
    def get_performance_config(self) -> Dict[str, Any]:
        """Get performance configuration"""
        return self.performance_config.copy()
    
    def print_configuration(self):
        """Print current configuration"""
        print("🔧 Clinical Context Service - Data Source Configuration")
        print("=" * 60)
        print(f"Mode: {self.mode.value}")
        print(f"Use Elasticsearch: {self.should_use_elasticsearch()}")
        print(f"Use Microservices: {self.should_use_microservices()}")
        
        if self.should_use_elasticsearch():
            print(f"\n📊 Elasticsearch Configuration:")
            print(f"   URL: {self.elasticsearch_config['hosts'][0]}")
            print(f"   Timeout: {self.elasticsearch_config['timeout']}s")
            print(f"   Max Retries: {self.elasticsearch_config['max_retries']}")
            print(f"   Indices: {len(self.elasticsearch_indices)} configured")
        
        if self.should_use_microservices():
            print(f"\n📡 Microservice Configuration:")
            active_services = [k for k, v in self.microservice_endpoints.items() if v]
            print(f"   Active services: {len(active_services)}")
            for service, url in list(self.microservice_endpoints.items())[:3]:
                print(f"   {service}: {url}")
            if len(self.microservice_endpoints) > 3:
                print(f"   ... and {len(self.microservice_endpoints) - 3} more")
        
        print(f"\n⚡ Performance Settings:")
        print(f"   ES Timeout: {self.performance_config['elasticsearch_timeout_ms']}ms")
        print(f"   MS Timeout: {self.performance_config['microservice_timeout_ms']}ms")
        print(f"   Max Concurrent: {self.performance_config['max_concurrent_requests']}")
        print(f"   Cache TTL: {self.performance_config['cache_ttl_seconds']}s")


# Global configuration instance
config = DataSourceConfig()


def get_data_source_config() -> DataSourceConfig:
    """Get the global data source configuration"""
    return config


def set_data_source_mode(mode: DataSourceMode):
    """Set the data source mode"""
    global config
    config.mode = mode
    print(f"🔄 Data source mode changed to: {mode.value}")


def enable_elasticsearch_direct():
    """Enable direct Elasticsearch mode"""
    set_data_source_mode(DataSourceMode.ELASTICSEARCH_DIRECT)
    print("🚀 Direct Elasticsearch mode enabled - bypassing microservices")


def enable_microservices_mode():
    """Enable microservices mode"""
    set_data_source_mode(DataSourceMode.MICROSERVICES)
    print("📡 Microservices mode enabled - using service-to-service calls")


def enable_hybrid_mode():
    """Enable hybrid mode (Elasticsearch first, microservices fallback)"""
    set_data_source_mode(DataSourceMode.HYBRID)
    print("🔄 Hybrid mode enabled - Elasticsearch first, microservices fallback")


# Environment-based configuration
if __name__ == "__main__":
    # Print configuration when run directly
    config.print_configuration()
    
    print("\n🔧 Environment Variables for Configuration:")
    print("   CONTEXT_SERVICE_DATA_MODE=elasticsearch_direct|microservices|hybrid")
    print("   ELASTICSEARCH_URL=https://your-elasticsearch-url")
    print("   ELASTICSEARCH_API_KEY=your-api-key")
    print("   PATIENT_SERVICE_URL=http://localhost:8003")
    print("   ... (and other service URLs)")
    
    print("\n💡 Usage Examples:")
    print("   # Use direct Elasticsearch (fastest)")
    print("   export CONTEXT_SERVICE_DATA_MODE=elasticsearch_direct")
    print("   ")
    print("   # Use microservices (traditional)")
    print("   export CONTEXT_SERVICE_DATA_MODE=microservices")
    print("   ")
    print("   # Use hybrid (best of both)")
    print("   export CONTEXT_SERVICE_DATA_MODE=hybrid")
