FROM python:3.11-slim

LABEL maintainer="CardioFit Platform Team"
LABEL description="PostgreSQL Projector Service - Projects enriched clinical events to PostgreSQL"

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy shared module first (for layer caching)
COPY ../module8-shared /app/module8-shared

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY app/ ./app/
COPY schema/ ./schema/

# Create non-root user for security
RUN useradd -m -u 1000 projector && \
    chown -R projector:projector /app

USER projector

# Expose service and metrics ports
EXPOSE 8050 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8050/health').raise_for_status()" || exit 1

# Run FastAPI application
CMD ["python", "-m", "app.main"]
