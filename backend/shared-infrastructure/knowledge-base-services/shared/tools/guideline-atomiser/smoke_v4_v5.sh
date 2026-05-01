#!/usr/bin/env bash
# V4 + V5 side-by-side smoke — runs locally using .venv13
# Usage: bash smoke_v4_v5.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PYTHON=".venv13/bin/python3"
PIPELINE="data/run_pipeline_targeted.py"
METRICS="data/v5_metrics.py"
export PYTHONPATH="$SCRIPT_DIR/../../..:$SCRIPT_DIR"
export GLINER_MODEL="urchade/gliner_mediumv2.1"
export GLINER_DEVICE="cpu"
export OLLAMA_URL="http://localhost:11434"
export NUEXTRACT_MODEL="nuextract"
export MONKEYOCR_DEVICE="cpu"

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  V4 + V5 Bbox Provenance Smoke — KDIGO 2022 QRG (local)     ║"
echo "║  L1: Docling (cached models) |  Ollama NuExtract (local)    ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

GUIDELINE="kdigo_2022_diabetes_ckd.yaml"
OUTDIR="data/output/v4"

# ── RUN 1: V4 baseline ────────────────────────────────────────────
echo "━━━ RUN 1: V4 baseline (V5_BBOX_PROVENANCE unset) ━━━"
"$PYTHON" "$PIPELINE" \
  --pipeline 1 --source quick-reference --l1 docling \
  --guideline "$GUIDELINE"
V4_JOB=$(ls -td "$OUTDIR"/job_docling_* 2>/dev/null | head -1)
echo "→ V4 job: $V4_JOB"
echo ""

# ── RUN 2: V5 ─────────────────────────────────────────────────────
echo "━━━ RUN 2: V5 (V5_BBOX_PROVENANCE=1) ━━━"
V5_BBOX_PROVENANCE=1 "$PYTHON" "$PIPELINE" \
  --pipeline 1 --source quick-reference --l1 docling \
  --guideline "$GUIDELINE"
V5_JOB=$(ls -td "$OUTDIR"/job_docling_* 2>/dev/null | head -1)
echo "→ V5 job: $V5_JOB"
echo ""

# ── METRICS ───────────────────────────────────────────────────────
echo "━━━ V5 METRICS (V4 → V5 comparison) ━━━"
"$PYTHON" "$METRICS" "$V4_JOB" "$V5_JOB"

echo ""
echo "━━━ v5_features_enabled ━━━"
"$PYTHON" - <<EOF
import json
for label, path in [("V4", "$V4_JOB"), ("V5", "$V5_JOB")]:
    m = json.load(open(f"{path}/job_metadata.json"))
    print(f"  {label}: v5_features_enabled = {m.get('v5_features_enabled', 'KEY MISSING')}")
EOF

echo ""
echo "━━━ First V5 span with channel_provenance ━━━"
"$PYTHON" - <<EOF
import json, sys
spans = json.load(open("$V5_JOB/merged_spans.json"))
hit = next((s for s in spans if s.get("channel_provenance")), None)
if not hit:
    print("ERROR: no span with channel_provenance in V5 output", file=sys.stderr)
    sys.exit(1)
print("span id   :", hit.get("id", hit.get("span_id", "?")))
print("channels  :", [e["channel_id"] for e in hit["channel_provenance"]])
print(json.dumps(hit["channel_provenance"][0], indent=2))
EOF

echo ""
echo "━━━ metrics.json — V4 job ━━━"
cat "$V4_JOB/metrics.json"

echo ""
echo "━━━ metrics.json — V5 job ━━━"
cat "$V5_JOB/metrics.json"

echo ""
echo "✅  Smoke complete"
