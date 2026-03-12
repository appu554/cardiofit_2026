# SNOMED CT-AU Downloader - Cloud Run Job Container
# Downloads from Australian National Terminology Service (NTS)

FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY main.py .

# Set environment variables
ENV PYTHONUNBUFFERED=1

# Run the downloader
CMD ["python", "main.py"]
