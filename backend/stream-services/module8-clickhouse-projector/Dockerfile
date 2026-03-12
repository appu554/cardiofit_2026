FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Copy shared module first
COPY module8-shared /app/module8-shared

# Install module8_shared dependencies first
RUN pip install --no-cache-dir -r /app/module8-shared/requirements.txt

# Copy requirements and install projector-specific dependencies
COPY module8-clickhouse-projector/requirements.txt .

# Install projector dependencies
RUN pip install --no-cache-dir \
    fastapi==0.104.1 \
    uvicorn[standard]==0.24.0 \
    clickhouse-driver==0.2.6 \
    pyyaml==6.0.1 \
    python-dateutil==2.8.2 \
    && pip install -e /app/module8-shared

# Copy application code
COPY module8-clickhouse-projector/app/ /app/
COPY module8-clickhouse-projector/config.yaml /app/
COPY module8-clickhouse-projector/schema/ /app/schema/

# Expose service port
EXPOSE 8053

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8053/health')"

# Run service
CMD ["python", "main.py"]
