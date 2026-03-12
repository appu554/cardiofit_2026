# KB7 Terminology Migration Container
FROM python:3.11-slim

# Set working directory
WORKDIR /app/migration

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    libpq-dev \
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy migration scripts
COPY scripts/ ./scripts/
COPY data/ ./data/
COPY logs/ ./logs/

# Create necessary directories
RUN mkdir -p /app/migration/output \
    /app/migration/backup \
    /app/migration/temp

# Set environment variables
ENV PYTHONPATH=/app/migration
ENV PYTHONUNBUFFERED=1
ENV LOG_LEVEL=INFO

# Health check script
COPY --chmod=755 <<EOF /healthcheck.sh
#!/bin/bash
python -c "
import sys
try:
    import asyncpg, aiohttp, pandas, SPARQLWrapper
    print('All dependencies available')
    sys.exit(0)
except ImportError as e:
    print(f'Missing dependency: {e}')
    sys.exit(1)
"
EOF

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD /healthcheck.sh

# Default command
CMD ["python", "scripts/migrate_to_hybrid.py", "--help"]

# Migration execution examples:
# docker run --rm -v $(pwd)/config:/app/migration/config kb7-migration:latest \
#   python scripts/migrate_to_hybrid.py --config /app/migration/config/migration.yaml
#
# docker run --rm -v $(pwd)/output:/app/migration/output kb7-migration:latest \
#   python scripts/migrate_to_hybrid.py --dry-run --validate-only

LABEL maintainer="KB7 Team"
LABEL description="KB7 Terminology Migration from GraphDB to Hybrid Architecture"
LABEL version="3.5.1"