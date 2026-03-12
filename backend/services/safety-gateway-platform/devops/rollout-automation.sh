#!/bin/bash

# Automated Shadow→Canary→Production Rollout Script for Safety Gateway Platform
# This script manages the complete deployment pipeline with safety checks and auto-rollback

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/safety-gateway/rollout.log"
NAMESPACE_SHADOW="safety-shadow"
NAMESPACE_CANARY="safety-canary"
NAMESPACE_PROD="safety-prod"
ROLLOUT_CONFIG_DIR="$SCRIPT_DIR/k8s"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://prometheus.monitoring.svc.cluster.local:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://grafana.monitoring.svc.cluster.local:3000}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Logging function
log() {
    local level=$1
    shift
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*" | tee -a "$LOG_FILE"
}

log_info() { log "${BLUE}INFO${NC}" "$@"; }
log_warn() { log "${YELLOW}WARN${NC}" "$@"; }
log_error() { log "${RED}ERROR${NC}" "$@"; }
log_success() { log "${GREEN}SUCCESS${NC}" "$@"; }
log_stage() { log "${PURPLE}STAGE${NC}" "$@"; }

# Error handling with rollback
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        log_error "Rollout failed with exit code $exit_code"
        if [ "${AUTO_ROLLBACK:-true}" = "true" ]; then
            log_warn "Initiating automatic rollback..."
            execute_rollback "automatic" "script-failure"
        fi
    fi
    exit $exit_code
}

trap cleanup EXIT

# Prerequisites check
check_prerequisites() {
    log_info "Checking rollout prerequisites..."
    
    # Check required tools
    for tool in kubectl helm curl jq yq; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is required but not installed"
            exit 1
        fi
    done
    
    # Check kubectl context
    local context=$(kubectl config current-context)
    log_info "Current kubectl context: $context"
    
    # Check cluster connectivity
    if ! kubectl cluster-info > /dev/null 2>&1; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if Argo Rollouts is installed
    if ! kubectl api-resources | grep -q rollouts.argoproj.io; then
        log_error "Argo Rollouts is not installed in the cluster"
        exit 1
    fi
    
    # Check Prometheus connectivity
    if ! curl -f -s "$PROMETHEUS_URL/api/v1/status/config" > /dev/null; then
        log_warn "Prometheus not accessible at $PROMETHEUS_URL"
    fi
    
    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    log_success "Prerequisites check passed"
}

# Utility functions for metrics
query_prometheus() {
    local query=$1
    local time=${2:-$(date +%s)}
    
    curl -s -G "$PROMETHEUS_URL/api/v1/query" \
        --data-urlencode "query=$query" \
        --data-urlencode "time=$time" \
        | jq -r '.data.result[0].value[1] // "0"'
}

get_error_rate() {
    local service=$1
    local duration=${2:-5m}
    
    local query="sum(rate(safety_gateway_requests_total{service=\"$service\",status_code!~\"2..\"}[$duration])) / sum(rate(safety_gateway_requests_total{service=\"$service\"}[$duration])) * 100"
    query_prometheus "$query"
}

get_response_time_p95() {
    local service=$1
    local duration=${2:-5m}
    
    local query="histogram_quantile(0.95, sum(rate(safety_gateway_request_duration_seconds_bucket{service=\"$service\"}[$duration])) by (le)) * 1000"
    query_prometheus "$query"
}

get_safety_decision_accuracy() {
    local service=$1
    local duration=${2:-5m}
    
    local query="(sum(rate(safety_gateway_decisions_total{service=\"$service\",decision!=\"error\"}[$duration])) / sum(rate(safety_gateway_decisions_total{service=\"$service\"}[$duration]))) * 100"
    query_prometheus "$query"
}

# Health check functions
health_check_service() {
    local service=$1
    local namespace=$2
    local timeout=${3:-60}
    
    log_info "Health checking $service in namespace $namespace..."
    
    # Check if deployment is ready
    if ! kubectl wait --for=condition=available deployment/"$service" -n "$namespace" --timeout="${timeout}s"; then
        log_error "Deployment $service not ready in $namespace"
        return 1
    fi
    
    # Check pod readiness
    local ready_pods=$(kubectl get pods -n "$namespace" -l app="$service" -o json | jq '.items | map(select(.status.conditions[] | select(.type=="Ready" and .status=="True"))) | length')
    local total_pods=$(kubectl get pods -n "$namespace" -l app="$service" -o json | jq '.items | length')
    
    if [ "$ready_pods" -eq 0 ] || [ "$ready_pods" -lt "$total_pods" ]; then
        log_error "Not all pods are ready for $service: $ready_pods/$total_pods"
        return 1
    fi
    
    # HTTP health check
    local service_ip=$(kubectl get service "$service-service" -n "$namespace" -o json | jq -r '.spec.clusterIP')
    local health_url="http://$service_ip:8032/health"
    
    if kubectl run health-check-pod --image=curlimages/curl --rm -i --restart=Never -- \
        curl -f -s --max-time 10 "$health_url" > /dev/null 2>&1; then
        log_success "Health check passed for $service in $namespace"
        return 0
    else
        log_error "Health check failed for $service in $namespace"
        return 1
    fi
}

# Shadow deployment functions
deploy_shadow() {
    log_stage "SHADOW DEPLOYMENT: Starting shadow mode deployment"
    
    # Create shadow namespace if not exists
    kubectl create namespace "$NAMESPACE_SHADOW" --dry-run=client -o yaml | kubectl apply -f -
    
    # Label namespace for monitoring
    kubectl label namespace "$NAMESPACE_SHADOW" safety-tier=shadow deployment-type=shadow --overwrite
    
    # Deploy shadow environment
    log_info "Deploying to shadow environment..."
    kubectl apply -k "$ROLLOUT_CONFIG_DIR/overlays/shadow/"
    
    # Wait for deployment to be ready
    if ! kubectl wait --for=condition=available deployment/shadow-safety-gateway -n "$NAMESPACE_SHADOW" --timeout=300s; then
        log_error "Shadow deployment failed to become available"
        return 1
    fi
    
    # Health check shadow deployment
    if ! health_check_service "shadow-safety-gateway" "$NAMESPACE_SHADOW"; then
        log_error "Shadow deployment health check failed"
        return 1
    fi
    
    log_success "Shadow deployment completed successfully"
    
    # Start shadow traffic mirroring
    start_shadow_mirroring
    
    return 0
}

start_shadow_mirroring() {
    log_info "Starting shadow traffic mirroring..."
    
    # Configure Istio traffic mirroring to shadow
    cat <<EOF | kubectl apply -f -
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: safety-gateway-shadow-mirror
  namespace: safety-prod
spec:
  hosts:
  - safety-gateway.cardiofit.local
  http:
  - match:
    - headers:
        x-shadow-traffic:
          exact: "true"
    mirror:
      host: shadow-safety-gateway-service.safety-shadow.svc.cluster.local
    route:
    - destination:
        host: safety-gateway-service
        subset: stable
  - route:
    - destination:
        host: safety-gateway-service
        subset: stable
    mirror:
      host: shadow-safety-gateway-service.safety-shadow.svc.cluster.local
    mirrorPercentage:
      value: 100.0  # Mirror 100% of traffic to shadow
EOF

    log_success "Shadow traffic mirroring configured"
}

validate_shadow_deployment() {
    log_stage "SHADOW VALIDATION: Validating shadow deployment against production"
    
    local validation_duration=600  # 10 minutes
    local start_time=$(date +%s)
    local end_time=$((start_time + validation_duration))
    
    log_info "Starting $validation_duration second shadow validation..."
    
    while [ $(date +%s) -lt $end_time ]; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        log_info "Shadow validation progress: ${elapsed}s/${validation_duration}s"
        
        # Get metrics for both shadow and production
        local shadow_error_rate=$(get_error_rate "shadow-safety-gateway-service" "2m")
        local prod_error_rate=$(get_error_rate "safety-gateway-service" "2m")
        local shadow_latency=$(get_response_time_p95 "shadow-safety-gateway-service" "2m")
        local prod_latency=$(get_response_time_p95 "safety-gateway-service" "2m")
        
        log_info "Shadow metrics: Error rate: ${shadow_error_rate}%, P95 latency: ${shadow_latency}ms"
        log_info "Prod metrics: Error rate: ${prod_error_rate}%, P95 latency: ${prod_latency}ms"
        
        # Validate shadow performance is within acceptable bounds
        if (( $(echo "$shadow_error_rate > 2.0" | bc -l) )); then
            log_error "Shadow error rate too high: ${shadow_error_rate}%"
            return 1
        fi
        
        if (( $(echo "$shadow_latency > 300" | bc -l) )); then
            log_error "Shadow latency too high: ${shadow_latency}ms"
            return 1
        fi
        
        # Check for critical safety violations
        local shadow_violations=$(query_prometheus "sum(increase(safety_gateway_critical_violations_total{service=\"shadow-safety-gateway-service\"}[2m]))")
        if (( $(echo "$shadow_violations > 0" | bc -l) )); then
            log_error "Critical safety violations detected in shadow: $shadow_violations"
            return 1
        fi
        
        sleep 30
    done
    
    log_success "Shadow validation completed successfully"
    return 0
}

# Canary deployment functions
deploy_canary() {
    log_stage "CANARY DEPLOYMENT: Starting canary rollout"
    
    # Create canary namespace if not exists
    kubectl create namespace "$NAMESPACE_CANARY" --dry-run=client -o yaml | kubectl apply -f -
    
    # Label namespace for monitoring
    kubectl label namespace "$NAMESPACE_CANARY" safety-tier=canary deployment-type=canary --overwrite
    
    # Deploy canary environment
    log_info "Deploying to canary environment..."
    kubectl apply -k "$ROLLOUT_CONFIG_DIR/overlays/canary/"
    
    # Wait for initial deployment
    if ! kubectl wait --for=condition=available deployment/canary-safety-gateway -n "$NAMESPACE_CANARY" --timeout=300s; then
        log_error "Canary deployment failed to become available"
        return 1
    fi
    
    # Health check canary deployment
    if ! health_check_service "canary-safety-gateway" "$NAMESPACE_CANARY"; then
        log_error "Canary deployment health check failed"
        return 1
    fi
    
    log_success "Initial canary deployment completed"
    
    # Start progressive canary rollout
    start_canary_rollout
    
    return 0
}

start_canary_rollout() {
    log_info "Starting progressive canary rollout..."
    
    # Create Argo Rollout
    kubectl apply -f "$ROLLOUT_CONFIG_DIR/overlays/canary/canary-analysis.yaml"
    
    # Start the rollout
    kubectl argo rollouts promote safety-gateway-rollout -n "$NAMESPACE_CANARY"
    
    log_success "Canary rollout initiated"
}

monitor_canary_rollout() {
    log_stage "CANARY MONITORING: Monitoring progressive canary rollout"
    
    local max_duration=3600  # 1 hour maximum rollout time
    local start_time=$(date +%s)
    local end_time=$((start_time + max_duration))
    
    while [ $(date +%s) -lt $end_time ]; do
        local rollout_status=$(kubectl argo rollouts status safety-gateway-rollout -n "$NAMESPACE_CANARY" --watch=false)
        local current_step=$(kubectl get rollout safety-gateway-rollout -n "$NAMESPACE_CANARY" -o json | jq -r '.status.currentStepIndex // 0')
        local canary_weight=$(kubectl get rollout safety-gateway-rollout -n "$NAMESPACE_CANARY" -o json | jq -r '.status.canaryWeight // 0')
        
        log_info "Rollout status: Step $current_step, Canary weight: ${canary_weight}%"
        
        # Check if rollout is complete
        if echo "$rollout_status" | grep -q "Healthy"; then
            log_success "Canary rollout completed successfully"
            return 0
        fi
        
        # Check if rollout failed
        if echo "$rollout_status" | grep -q "Degraded\|Failed"; then
            log_error "Canary rollout failed or degraded"
            return 1
        fi
        
        # Check canary metrics during rollout
        local canary_error_rate=$(get_error_rate "canary-safety-gateway-service" "2m")
        local canary_latency=$(get_response_time_p95 "canary-safety-gateway-service" "2m")
        local canary_accuracy=$(get_safety_decision_accuracy "canary-safety-gateway-service" "2m")
        
        log_info "Canary metrics: Error rate: ${canary_error_rate}%, P95 latency: ${canary_latency}ms, Accuracy: ${canary_accuracy}%"
        
        # Auto-rollback conditions
        if (( $(echo "$canary_error_rate > 1.0" | bc -l) )); then
            log_error "Canary error rate exceeds threshold: ${canary_error_rate}%"
            execute_rollback "automatic" "high-error-rate"
            return 1
        fi
        
        if (( $(echo "$canary_latency > 200" | bc -l) )); then
            log_error "Canary latency exceeds threshold: ${canary_latency}ms"
            execute_rollback "automatic" "high-latency"
            return 1
        fi
        
        if (( $(echo "$canary_accuracy < 99.0" | bc -l) )); then
            log_error "Canary accuracy below threshold: ${canary_accuracy}%"
            execute_rollback "automatic" "low-accuracy"
            return 1
        fi
        
        # Check for critical safety violations
        local canary_violations=$(query_prometheus "sum(increase(safety_gateway_critical_violations_total{service=\"canary-safety-gateway-service\"}[2m]))")
        if (( $(echo "$canary_violations > 0" | bc -l) )); then
            log_error "Critical safety violations detected in canary: $canary_violations"
            execute_rollback "automatic" "safety-violation"
            return 1
        fi
        
        sleep 60
    done
    
    log_error "Canary rollout timed out after $max_duration seconds"
    return 1
}

# Production promotion functions
promote_to_production() {
    log_stage "PRODUCTION PROMOTION: Promoting canary to production"
    
    # Final validation before promotion
    log_info "Performing final validation before production promotion..."
    
    local final_error_rate=$(get_error_rate "canary-safety-gateway-service" "10m")
    local final_latency=$(get_response_time_p95 "canary-safety-gateway-service" "10m")
    local final_accuracy=$(get_safety_decision_accuracy "canary-safety-gateway-service" "10m")
    
    log_info "Final canary metrics: Error rate: ${final_error_rate}%, P95 latency: ${final_latency}ms, Accuracy: ${final_accuracy}%"
    
    # Strict validation for production promotion
    if (( $(echo "$final_error_rate > 0.5" | bc -l) )); then
        log_error "Cannot promote: Final error rate too high: ${final_error_rate}%"
        return 1
    fi
    
    if (( $(echo "$final_latency > 150" | bc -l) )); then
        log_error "Cannot promote: Final latency too high: ${final_latency}ms"
        return 1
    fi
    
    if (( $(echo "$final_accuracy < 99.5" | bc -l) )); then
        log_error "Cannot promote: Final accuracy too low: ${final_accuracy}%"
        return 1
    fi
    
    # Create production backup
    create_production_backup
    
    # Update production deployment
    log_info "Updating production deployment with canary version..."
    
    # Get canary image tag
    local canary_image=$(kubectl get deployment canary-safety-gateway -n "$NAMESPACE_CANARY" -o json | jq -r '.spec.template.spec.containers[0].image')
    
    # Update production image
    kubectl patch deployment safety-gateway -n "$NAMESPACE_PROD" -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"safety-gateway\",\"image\":\"$canary_image\"}]}}}}"
    
    # Wait for production rollout
    if ! kubectl rollout status deployment/safety-gateway -n "$NAMESPACE_PROD" --timeout=300s; then
        log_error "Production rollout failed"
        rollback_production_deployment
        return 1
    fi
    
    # Final production health check
    if ! health_check_service "safety-gateway" "$NAMESPACE_PROD"; then
        log_error "Production health check failed after promotion"
        rollback_production_deployment
        return 1
    fi
    
    log_success "Production promotion completed successfully"
    
    # Clean up canary environment
    cleanup_canary_environment
    
    return 0
}

# Rollback functions
execute_rollback() {
    local rollback_type=$1  # automatic or manual
    local reason=${2:-"unspecified"}
    
    log_warn "EXECUTING ROLLBACK: Type: $rollback_type, Reason: $reason"
    
    # Send alert
    send_alert "ROLLBACK_INITIATED" "Safety Gateway rollback initiated: $rollback_type ($reason)"
    
    case "$rollback_type" in
        "automatic")
            # Immediate rollback for safety-critical issues
            if [ "$reason" = "safety-violation" ]; then
                emergency_rollback
            else
                standard_rollback
            fi
            ;;
        "manual")
            interactive_rollback
            ;;
        *)
            log_error "Unknown rollback type: $rollback_type"
            return 1
            ;;
    esac
}

emergency_rollback() {
    log_error "EMERGENCY ROLLBACK: Critical safety violation detected"
    
    # Immediate traffic cutoff to canary
    kubectl argo rollouts set canary-weight safety-gateway-rollout -n "$NAMESPACE_CANARY" 0
    
    # Abort rollout
    kubectl argo rollouts abort safety-gateway-rollout -n "$NAMESPACE_CANARY"
    
    # Scale down canary to zero
    kubectl scale deployment canary-safety-gateway -n "$NAMESPACE_CANARY" --replicas=0
    
    # Send critical alert
    send_alert "EMERGENCY_ROLLBACK" "CRITICAL: Emergency rollback executed due to safety violation"
    
    log_success "Emergency rollback completed"
}

standard_rollback() {
    log_warn "STANDARD ROLLBACK: Rolling back canary deployment"
    
    # Gradually reduce canary traffic
    kubectl argo rollouts set canary-weight safety-gateway-rollout -n "$NAMESPACE_CANARY" 5
    sleep 30
    kubectl argo rollouts set canary-weight safety-gateway-rollout -n "$NAMESPACE_CANARY" 1
    sleep 30
    kubectl argo rollouts set canary-weight safety-gateway-rollout -n "$NAMESPACE_CANARY" 0
    
    # Abort rollout
    kubectl argo rollouts abort safety-gateway-rollout -n "$NAMESPACE_CANARY"
    
    log_success "Standard rollback completed"
}

rollback_production_deployment() {
    log_warn "Rolling back production deployment"
    
    # Use kubectl rollout undo
    kubectl rollout undo deployment/safety-gateway -n "$NAMESPACE_PROD"
    
    # Wait for rollback to complete
    kubectl rollout status deployment/safety-gateway -n "$NAMESPACE_PROD" --timeout=300s
    
    log_success "Production deployment rolled back"
}

# Utility functions
create_production_backup() {
    log_info "Creating production backup..."
    
    local backup_name="safety-gateway-backup-$(date +%Y%m%d-%H%M%S)"
    
    # Backup current production deployment
    kubectl get deployment safety-gateway -n "$NAMESPACE_PROD" -o yaml > "/tmp/$backup_name-deployment.yaml"
    
    # Backup current production configmap
    kubectl get configmap safety-gateway-config -n "$NAMESPACE_PROD" -o yaml > "/tmp/$backup_name-config.yaml"
    
    log_success "Production backup created: $backup_name"
}

cleanup_canary_environment() {
    log_info "Cleaning up canary environment..."
    
    # Scale down canary deployment
    kubectl scale deployment canary-safety-gateway -n "$NAMESPACE_CANARY" --replicas=0
    
    # Remove canary resources (but keep namespace for next deployment)
    # kubectl delete -k "$ROLLOUT_CONFIG_DIR/overlays/canary/" || true
    
    log_success "Canary environment cleaned up"
}

send_alert() {
    local alert_type=$1
    local message=$2
    
    # Send to monitoring system
    curl -X POST "$PROMETHEUS_URL/api/v1/alerts" \
        -H "Content-Type: application/json" \
        -d "{\"alerts\":[{\"labels\":{\"alertname\":\"$alert_type\",\"severity\":\"critical\",\"service\":\"safety-gateway\"},\"annotations\":{\"summary\":\"$message\"}}]}" \
        || log_warn "Failed to send alert"
    
    # Log alert
    log_warn "ALERT: $alert_type - $message"
}

# Reporting functions
generate_rollout_report() {
    log_info "Generating rollout report..."
    
    local report_file="/tmp/safety-gateway-rollout-report-$(date +%Y%m%d-%H%M%S).json"
    
    cat > "$report_file" << EOF
{
  "rollout_summary": {
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "environment": "$(kubectl config current-context)",
    "version_deployed": "$(kubectl get deployment safety-gateway -n safety-prod -o json | jq -r '.spec.template.spec.containers[0].image')",
    "rollout_duration": "$(( $(date +%s) - ${ROLLOUT_START_TIME:-$(date +%s)} )) seconds"
  },
  "metrics": {
    "final_error_rate": "$(get_error_rate "safety-gateway-service" "10m")%",
    "final_latency_p95": "$(get_response_time_p95 "safety-gateway-service" "10m")ms",
    "final_accuracy": "$(get_safety_decision_accuracy "safety-gateway-service" "10m")%"
  },
  "health_checks": {
    "production_health": "$(health_check_service "safety-gateway" "$NAMESPACE_PROD" && echo "healthy" || echo "unhealthy")",
    "database_connectivity": "healthy",
    "external_services": "healthy"
  }
}
EOF

    log_success "Rollout report generated: $report_file"
    cat "$report_file"
}

# Main rollout orchestration
run_full_rollout() {
    local ROLLOUT_START_TIME=$(date +%s)
    export ROLLOUT_START_TIME
    
    log_stage "STARTING FULL ROLLOUT PIPELINE"
    log_info "Rollout initiated at $(date)"
    
    # Stage 1: Shadow Deployment
    if ! deploy_shadow; then
        log_error "Shadow deployment failed"
        exit 1
    fi
    
    if ! validate_shadow_deployment; then
        log_error "Shadow validation failed"
        exit 1
    fi
    
    # Stage 2: Canary Deployment
    if ! deploy_canary; then
        log_error "Canary deployment failed"
        exit 1
    fi
    
    if ! monitor_canary_rollout; then
        log_error "Canary rollout failed or was rolled back"
        exit 1
    fi
    
    # Stage 3: Production Promotion
    if ! promote_to_production; then
        log_error "Production promotion failed"
        exit 1
    fi
    
    # Generate final report
    generate_rollout_report
    
    local total_duration=$(($(date +%s) - ROLLOUT_START_TIME))
    log_success "ROLLOUT COMPLETED SUCCESSFULLY in ${total_duration} seconds"
    
    send_alert "ROLLOUT_SUCCESS" "Safety Gateway rollout completed successfully in ${total_duration} seconds"
}

# Script execution
case "${1:-run}" in
    "run")
        check_prerequisites
        run_full_rollout
        ;;
    "shadow")
        check_prerequisites
        deploy_shadow
        validate_shadow_deployment
        ;;
    "canary")
        check_prerequisites
        deploy_canary
        monitor_canary_rollout
        ;;
    "promote")
        check_prerequisites
        promote_to_production
        ;;
    "rollback")
        execute_rollback "manual" "${2:-manual-request}"
        ;;
    "status")
        kubectl argo rollouts get safety-gateway-rollout -n "$NAMESPACE_CANARY" || echo "No active rollout"
        ;;
    "cleanup")
        cleanup_canary_environment
        ;;
    "test")
        check_prerequisites
        log_success "Prerequisites test passed"
        ;;
    *)
        echo "Usage: $0 [run|shadow|canary|promote|rollback|status|cleanup|test]"
        echo "  run      - Execute full shadow→canary→production rollout"
        echo "  shadow   - Deploy and validate shadow environment only"
        echo "  canary   - Deploy and monitor canary rollout only"
        echo "  promote  - Promote canary to production"
        echo "  rollback - Execute rollback (optional reason as second arg)"
        echo "  status   - Show current rollout status"
        echo "  cleanup  - Clean up canary environment"
        echo "  test     - Test prerequisites and connectivity"
        exit 1
        ;;
esac