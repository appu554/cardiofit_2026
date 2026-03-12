# Dockerfile for Complete KB7 Runtime Integration
FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy all runtime layer source code
COPY . .

# Set Python path to include all modules
ENV PYTHONPATH=/app

# Create non-root user
RUN useradd -m -u 1000 kb7runtime && chown -R kb7runtime:kb7runtime /app
USER kb7runtime

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD python main_integration.py --health || exit 1

# Default command - start all services
CMD ["python", "main_integration.py", "--start"]