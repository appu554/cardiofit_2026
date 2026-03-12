# V4.3 Clinical Guideline Curation Pipeline - Hybrid Architecture Image
# L1: MonkeyOCR (default) | Marker (legacy) | Docling (alternative)
# Channel A: Granite-Docling structural oracle
# L2: V4.3 Multi-Channel (0/A/B/C/D/E/F) — Channel F via Ollama sidecar
# L3: Claude | L4: RxNav/Snow Owl | L5: CQL Validation
#
# Build from shared/ directory context:
#   docker build -f tools/guideline-atomiser/Dockerfile -t v4-pipeline .
#
# Run with V4.3 (MonkeyOCR L1 + Granite Channel A + CoverageGuard):
#   docker compose up -d
#   docker compose exec ollama ollama pull nuextract
#   docker compose run --rm v4-pipeline-cli python /app/guideline-atomiser/data/run_pipeline_targeted.py --pipeline 1 --source full-guide

FROM python:3.11-slim-bookworm AS base

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# Install system dependencies for PDF processing + Aho-Corasick C extension
RUN apt-get update && apt-get install -y --no-install-recommends \
    # PDF processing (Marker + MonkeyOCR)
    poppler-utils \
    tesseract-ocr \
    tesseract-ocr-eng \
    libgl1-mesa-glx \
    libglib2.0-0 \
    # Build tools (needed for pyahocorasick C extension)
    build-essential \
    git \
    curl \
    # Cleanup
    && rm -rf /var/lib/apt/lists/*

# ═══════════════════════════════════════════════════════════════════════════════
# Stage 1: Build dependencies
# ═══════════════════════════════════════════════════════════════════════════════
FROM base AS builder

WORKDIR /build

# Copy requirements file
COPY tools/guideline-atomiser/requirements.txt /build/requirements.txt

# Install all dependencies (includes magic-pdf for MonkeyOCR)
RUN pip install --no-cache-dir -r requirements.txt

# Pre-download GLiNER model for Channel E (~500MB)
RUN python -c "from huggingface_hub import snapshot_download; snapshot_download('urchade/gliner_mediumv2.1')"

# Pre-download Marker/Surya ML models for L1 legacy (~1.35GB)
RUN python -c "from marker.models import create_model_dict; print('Pre-loading Marker models...'); create_model_dict(); print('Marker models cached!')"

# Pre-download Docling StandardPipeline models for L1 alternative (~800MB)
RUN python -c "from docling.document_converter import DocumentConverter; print('Pre-loading Docling StandardPipeline models...'); dc = DocumentConverter(); print('Docling models cached!')"

# Pre-download Granite-Docling VlmPipeline model for Channel A structural oracle (~258MB)
RUN python -c "\
from docling.document_converter import DocumentConverter, PdfFormatOption; \
from docling.datamodel.base_models import InputFormat; \
from docling.pipeline.vlm_pipeline import VlmPipeline; \
from docling.datamodel.pipeline_options import VlmPipelineOptions; \
print('Pre-loading Granite-Docling VlmPipeline model...'); \
dc = DocumentConverter(format_options={InputFormat.PDF: PdfFormatOption(pipeline_cls=VlmPipeline, pipeline_options=VlmPipelineOptions())}); \
print('Granite-Docling VlmPipeline model cached!')"

# Pre-download MonkeyOCR models for L1 default (~3GB)
# Downloads: echo840/MonkeyOCR (Qwen2.5-VL based) + DocLayout-YOLO (.pt weights)
RUN python -c "\
from huggingface_hub import snapshot_download, hf_hub_download; \
print('Pre-loading MonkeyOCR model (echo840/MonkeyOCR)...'); \
snapshot_download('echo840/MonkeyOCR'); \
print('Pre-loading DocLayout-YOLO model...'); \
hf_hub_download(repo_id='juliozhao/DocLayout-YOLO-DocStructBench', filename='doclayout_yolo_docstructbench_imgsz1024.pt'); \
print('MonkeyOCR models cached!')"

# V4.1: Channel F uses Ollama sidecar (NOT in-process model)
# NuExtract 2.0-8B runs in a separate Ollama container with HTTP API
# No model pre-download needed here — Ollama manages its own model storage

# ═══════════════════════════════════════════════════════════════════════════════
# Stage 2: Production image
# ═══════════════════════════════════════════════════════════════════════════════
FROM base AS production

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Copy pre-downloaded models (GLiNER + Marker/Surya + MonkeyOCR)
COPY --from=builder /root/.cache/huggingface /root/.cache/huggingface
COPY --from=builder /root/.cache/datalab /root/.cache/datalab

# Copy the entire shared directory (build context is shared/)
# This includes extraction/ (with v4/ channels), cql/, tools/guideline-atomiser/
COPY extraction /app/extraction/
COPY cql /app/cql/
COPY tools/guideline-atomiser /app/guideline-atomiser/

# MonkeyOCR model config — tells MonkeyOCR where models are and which backend to use
COPY tools/guideline-atomiser/model_configs.yaml /app/guideline-atomiser/model_configs.yaml
ENV MONKEYOCR_CONFIG=/app/guideline-atomiser/model_configs.yaml

# Set Python path — both /app (for extraction.v4.*) and /app/guideline-atomiser (for marker_extractor, fact_extractor)
ENV PYTHONPATH=/app:/app/guideline-atomiser

# Create directories for data
RUN mkdir -p /data/pdfs /data/output /data/vaidshala

# Health check — verify V4 channel imports resolve (includes MonkeyOCR)
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "from extraction.v4.channel_a_docling import ChannelA; from extraction.v4.channel_b_drug_dict import ChannelB; from magic_pdf.data.data_reader_writer import FileBasedDataReader; print('OK')" || exit 1

# Default command: run V4 Pipeline 1 with MonkeyOCR on full guide
CMD ["python", "/app/guideline-atomiser/data/run_pipeline_targeted.py", "--pipeline", "1", "--source", "full-guide"]

# ═══════════════════════════════════════════════════════════════════════════════
# Labels
# ═══════════════════════════════════════════════════════════════════════════════
LABEL maintainer="Clinical Informatics Team" \
      version="4.3.1" \
      description="V4.3.1 Clinical Guideline Curation Pipeline - MonkeyOCR L1 + CoverageGuard (Multi-Channel L1-L5)" \
      org.opencontainers.image.source="https://github.com/cardiofit/clinical-synthesis-hub"
