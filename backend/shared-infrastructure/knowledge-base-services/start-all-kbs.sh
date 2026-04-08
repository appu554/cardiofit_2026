#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
# start-all-kbs.sh — Start all KB services (KB-1 → KB-26),
#                    Auth Service (8001), and API Gateway (8000)
#
# Each KB uses its own docker-compose.yml with isolated DB/Redis.
# All services join the shared kb-network for inter-service comms.
#
# Usage:
#   ./start-all-kbs.sh              # Start everything
#   ./start-all-kbs.sh --vaidshala  # Only Vaidshala KBs (19-26) + Auth + Gateway
#   ./start-all-kbs.sh --stop       # Stop everything
#   ./start-all-kbs.sh --health     # Check health of all services
# ─────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# KB service directories that have docker-compose.yml
CORE_KBS=(
  kb-1-drug-rules
  kb-2-clinical-context
  kb-3-guidelines
  kb-4-patient-safety
  kb-5-drug-interactions
  kb-6-formulary
  kb-8-calculator-service
  kb-10-rules-engine
  kb-11-population-health
  kb-12-ordersets-careplans
  kb-13-quality-measures
  kb-14-care-navigator
  kb-16-lab-interpretation
  kb-17-population-registry
  kb-18-governance-engine
)

VAIDSHALA_KBS=(
  kb-19-protocol-orchestrator
  kb-20-patient-profile
  kb-21-behavioral-intelligence
  kb-22-hpi-engine
  kb-23-decision-cards
  kb-24-safety-constraint-engine
  kb-25-lifestyle-knowledge-graph
  kb-26-metabolic-digital-twin
)

# Health check endpoints (service_name:port)
declare -A HEALTH_PORTS=(
  [kb-1-drug-rules]=8081
  [kb-2-clinical-context]=8086
  [kb-3-guidelines]=8083
  [kb-4-patient-safety]=8088
  [kb-5-drug-interactions]=8085
  [kb-6-formulary]=8091
  [kb-8-calculator-service]=8098
  [kb-10-rules-engine]=8100
  [kb-11-population-health]=8111
  [kb-12-ordersets-careplans]=8112
  [kb-13-quality-measures]=8113
  [kb-14-care-navigator]=8114
  [kb-16-lab-interpretation]=8095
  [kb-17-population-registry]=8017
  [kb-18-governance-engine]=8018
  [kb-19-protocol-orchestrator]=8103
  [kb-20-patient-profile]=8131
  [kb-21-behavioral-intelligence]=8133
  [kb-22-hpi-engine]=8132
  [kb-23-decision-cards]=8134
  [kb-24-safety-constraint-engine]=8201
  [kb-25-lifestyle-knowledge-graph]=8136
  [kb-26-metabolic-digital-twin]=8137
)

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${CYAN}[$(date +%H:%M:%S)]${NC} $*"; }
ok()   { echo -e "${GREEN}  ✓${NC} $*"; }
fail() { echo -e "${RED}  ✗${NC} $*"; }
warn() { echo -e "${YELLOW}  ⚠${NC} $*"; }

# ─── Ensure kb-network exists ────────────────────────────────
ensure_network() {
  if ! docker network inspect kb-network >/dev/null 2>&1; then
    log "Creating kb-network..."
    docker network create kb-network
  fi
}

# ─── Start a single KB from its own compose ──────────────────
start_kb() {
  local kb_dir="$1"
  if [ ! -f "$kb_dir/docker-compose.yml" ] && [ ! -f "$kb_dir/docker-compose.yaml" ]; then
    warn "$kb_dir — no docker-compose found, skipping"
    return 0
  fi

  local compose_file="docker-compose.yml"
  [ -f "$kb_dir/docker-compose.yaml" ] && compose_file="docker-compose.yaml"

  log "Starting $kb_dir..."
  if docker compose -f "$kb_dir/$compose_file" -p "$(basename "$kb_dir")" up -d --build 2>&1 | tail -3; then
    ok "$kb_dir started"
  else
    fail "$kb_dir FAILED to start"
  fi
}

# ─── Stop a single KB ────────────────────────────────────────
stop_kb() {
  local kb_dir="$1"
  local compose_file="docker-compose.yml"
  [ -f "$kb_dir/docker-compose.yaml" ] && compose_file="docker-compose.yaml"
  [ ! -f "$kb_dir/$compose_file" ] && return 0

  docker compose -f "$kb_dir/$compose_file" -p "$(basename "$kb_dir")" down 2>/dev/null || true
}

# ─── Start Auth Service + API Gateway ────────────────────────
start_gateway_stack() {
  log "Starting Auth Service (port 8001)..."
  docker compose -f "$SCRIPT_DIR/docker-compose.gateway-e2e.yml" \
    -p gateway-stack up -d --build auth-service api-gateway 2>&1 | tail -5
  ok "Auth Service + API Gateway started"
}

stop_gateway_stack() {
  docker compose -f "$SCRIPT_DIR/docker-compose.gateway-e2e.yml" \
    -p gateway-stack down 2>/dev/null || true
}

# ─── Health check ─────────────────────────────────────────────
check_health() {
  local passed=0 failed=0 total=0

  for kb_dir in "${@}"; do
    local port="${HEALTH_PORTS[$kb_dir]:-}"
    [ -z "$port" ] && continue
    total=$((total + 1))

    if curl -sf --max-time 3 "http://localhost:$port/health" >/dev/null 2>&1; then
      ok "$kb_dir (port $port) — healthy"
      passed=$((passed + 1))
    else
      fail "$kb_dir (port $port) — unreachable"
      failed=$((failed + 1))
    fi
  done

  # Auth Service
  total=$((total + 1))
  if curl -sf --max-time 3 "http://localhost:8001/health" >/dev/null 2>&1; then
    ok "Auth Service (port 8001) — healthy"
    passed=$((passed + 1))
  else
    fail "Auth Service (port 8001) — unreachable"
    failed=$((failed + 1))
  fi

  # API Gateway
  total=$((total + 1))
  if curl -sf --max-time 3 "http://localhost:8000/health" >/dev/null 2>&1; then
    ok "API Gateway (port 8000) — healthy"
    passed=$((passed + 1))
  else
    fail "API Gateway (port 8000) — unreachable"
    failed=$((failed + 1))
  fi

  echo ""
  log "Health: ${GREEN}$passed passed${NC} / ${RED}$failed failed${NC} / $total total"
}

# ─── Main ─────────────────────────────────────────────────────
case "${1:-}" in
  --stop)
    log "Stopping all services..."
    for kb in "${VAIDSHALA_KBS[@]}" "${CORE_KBS[@]}"; do stop_kb "$kb"; done
    stop_gateway_stack
    log "All services stopped."
    ;;

  --health)
    log "Checking health of all services..."
    check_health "${CORE_KBS[@]}" "${VAIDSHALA_KBS[@]}"
    ;;

  --vaidshala)
    ensure_network
    log "Starting Vaidshala runtime KBs (19-26) + Auth + Gateway..."
    for kb in "${VAIDSHALA_KBS[@]}"; do start_kb "$kb"; done
    start_gateway_stack
    log "Waiting 15s for services to initialize..."
    sleep 15
    check_health "${VAIDSHALA_KBS[@]}"
    ;;

  *)
    ensure_network
    log "Starting ALL KB services (1-26) + Auth + Gateway..."
    log ""
    log "Phase 1: Core KBs (1-18)..."
    for kb in "${CORE_KBS[@]}"; do start_kb "$kb"; done
    log ""
    log "Phase 2: Vaidshala Runtime KBs (19-26)..."
    for kb in "${VAIDSHALA_KBS[@]}"; do start_kb "$kb"; done
    log ""
    log "Phase 3: Auth Service + API Gateway..."
    start_gateway_stack
    log ""
    log "Waiting 20s for services to initialize..."
    sleep 20
    log "Checking health..."
    check_health "${CORE_KBS[@]}" "${VAIDSHALA_KBS[@]}"
    ;;
esac
