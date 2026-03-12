# Multi-stage build for MongoDB Projector service
FROM python:3.11-slim as base

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY app/ ./app/

# Copy shared module (will be installed via requirements.txt -e flag)
COPY ../module8-shared /app/module8-shared

# Create non-root user
RUN useradd -m -u 1000 projector && \
    chown -R projector:projector /app

USER projector

# Expose port
EXPOSE 8051

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8051/health')"

# Run the service
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8051"]
