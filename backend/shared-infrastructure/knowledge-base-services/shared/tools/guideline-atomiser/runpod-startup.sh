#!/usr/bin/env bash
# RunPod entrypoint for Pipeline 1 V4 extraction.
#
# Brings up Ollama, ensures the NuExtract model is pulled, pre-warms the
# MonkeyOCR / GLiNER HF cache on first run, then dispatches based on CMD.
#
# CMD modes (passed as $1):
#   bash           Drop into interactive shell after env is ready (default)
#   smoke          Run pipeline on AU-HF-ACS-HCP-Summary (2 pages, ~3 min on 4090)
#                  — proves env works; output should match local 10-span run
#   flagship       Run the 2 flagship HF PDFs sequentially (acs-guideline + cvd-risk)
#   queue <a> <b>  Run arbitrary sources from heart_foundation_au_2025 profile
#   pull-only      Stage repo + deps + models, then exit (for image pre-warm)

set -euo pipefail

REPO_DIR=/workspace/cardiofit_2026/backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
VENV=/workspace/venv

cd "$REPO_DIR"

# --- 1. Activate venv ----------------------------------------------------
# shellcheck disable=SC1091
source "$VENV/bin/activate"
export PYTHONPATH=".:..:../..:${PYTHONPATH:-}"
export OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"

# --- 2. Bring up Ollama --------------------------------------------------
echo "==> starting Ollama..."
mkdir -p /workspace/logs
ollama serve > /workspace/logs/ollama.log 2>&1 &
OLLAMA_PID=$!

# Wait up to 60 sec for Ollama health
for i in $(seq 1 30); do
  if curl -sf "$OLLAMA_URL/api/tags" >/dev/null 2>&1; then
    echo "    Ollama ready (PID $OLLAMA_PID)"
    break
  fi
  sleep 2
done
if ! curl -sf "$OLLAMA_URL/api/tags" >/dev/null 2>&1; then
  echo "!! Ollama failed to start; see /workspace/logs/ollama.log"
  exit 1
fi

# --- 3. Pull NuExtract if missing ---------------------------------------
if ! ollama list 2>/dev/null | grep -q "${NUEXTRACT_MODEL:-nuextract}"; then
  echo "==> pulling NuExtract (~2 GB, one-time)..."
  ollama pull "${NUEXTRACT_MODEL:-nuextract}"
fi

# --- 4. Pre-warm MonkeyOCR + GLiNER on first run ------------------------
HF_MARKER="/workspace/hf_cache/.warmed"
if [ ! -f "$HF_MARKER" ]; then
  echo "==> pre-warming MonkeyOCR + GLiNER (one-time, ~5 GB download)..."
  python3 -c "
import sys
sys.path.insert(0, '.')
sys.path.insert(0, '..')
sys.path.insert(0, '../..')
from monkeyocr_extractor import MonkeyOCRExtractor
e = MonkeyOCRExtractor()
print('  MonkeyOCR ready')
from gliner import GLiNER
m = GLiNER.from_pretrained('urchade/gliner_mediumv2.1')
print('  GLiNER ready')
"
  mkdir -p /workspace/hf_cache
  touch "$HF_MARKER"
fi

# --- 5. Verify GPU ------------------------------------------------------
echo "==> GPU check:"
python3 -c "
import torch
print(f'  CUDA available: {torch.cuda.is_available()}')
if torch.cuda.is_available():
    print(f'  GPU: {torch.cuda.get_device_name(0)}')
    print(f'  VRAM total: {torch.cuda.get_device_properties(0).total_memory / 1e9:.1f} GB')
"

# --- 6. Sync any new commits if branch advanced -------------------------
if [ "${SKIP_GIT_PULL:-0}" = "0" ]; then
  echo "==> git pull latest..."
  cd /workspace/cardiofit_2026
  git pull --ff-only || echo "  (git pull failed — continuing with current revision)"
  cd "$REPO_DIR"
fi

# --- 7. Dispatch based on CMD -------------------------------------------
MODE="${1:-bash}"
shift || true

run_one() {
  local src="$1"
  echo "================================================================"
  echo "==> $src — start: $(date)"
  echo "================================================================"
  local LOG="/workspace/logs/${src}.log"
  local START=$(date +%s)
  python3 data/run_pipeline_targeted.py \
    --pipeline 1 \
    --guideline heart_foundation_au_2025 \
    --source "$src" \
    --l1 monkeyocr \
    --target-kb all \
    2>&1 | tee "$LOG"
  local RC=${PIPESTATUS[0]}
  local DUR=$(( $(date +%s) - START ))
  echo "==> $src — finished exit=$RC duration=${DUR}s ($((DUR/60))m$((DUR%60))s)"
  return $RC
}

case "$MODE" in
  bash)
    echo "==> environment ready. Drop into shell."
    echo "    cd $REPO_DIR"
    echo "    python3 data/run_pipeline_targeted.py --pipeline 1 --guideline heart_foundation_au_2025 --source <src> --l1 monkeyocr"
    exec /bin/bash
    ;;
  pull-only)
    echo "==> pre-warm complete; exiting."
    exit 0
    ;;
  smoke)
    run_one acs-hcp-summary
    ;;
  flagship)
    run_one acs-guideline
    run_one cvd-risk
    ;;
  queue)
    if [ "$#" -eq 0 ]; then
      echo "!! 'queue' mode requires source names: queue acs-summary cholesterol-action ..."
      exit 1
    fi
    for src in "$@"; do
      run_one "$src" || echo "  ($src failed; continuing)"
    done
    ;;
  *)
    echo "!! unknown mode: $MODE"
    echo "   valid: bash | smoke | flagship | queue <src...> | pull-only"
    exit 1
    ;;
esac

echo ""
echo "==> all done. Job artefacts in: $REPO_DIR/data/output/v4/"
echo "    Push to KB-0:"
echo "      python3 data/push_to_kb0_gcp.py data/output/v4/job_monkeyocr_*/"
