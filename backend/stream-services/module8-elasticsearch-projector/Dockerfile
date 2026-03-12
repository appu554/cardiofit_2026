FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Copy shared module first
COPY module8-shared /app/module8-shared

# Copy service requirements and install
COPY module8-elasticsearch-projector/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy service code
COPY module8-elasticsearch-projector/src /app/src

# Set Python path
ENV PYTHONPATH=/app/src

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8052/health')" || exit 1

# Run service
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8052"]
