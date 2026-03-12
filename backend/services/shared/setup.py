from setuptools import setup, find_packages

setup(
    name="shared",
    version="0.1.0",
    packages=find_packages(),
    install_requires=[
        # Core dependencies
        "fastapi>=0.100.0",
        "pydantic>=2.0.0",
        "python-dateutil>=2.8.2",

        # Kafka and event processing
        "confluent-kafka>=2.3.0",
        "avro-python3>=1.11.0",
        "jsonschema>=4.17.0",

        # Google Healthcare API
        "google-cloud-healthcare>=1.11.0",
        "google-auth>=2.17.0",

        # Authentication
        "python-jose[cryptography]>=3.3.0",
        "supabase>=1.0.0",

        # Monitoring
        "prometheus-client>=0.16.0",
        "structlog>=23.1.0",
    ],
    python_requires=">=3.8",
)
