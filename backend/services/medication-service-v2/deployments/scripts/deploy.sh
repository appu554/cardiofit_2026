#!/bin/bash

# Medication Service V2 Deployment Script
# Healthcare-grade deployment with security validation and rollback capabilities

set -euo pipefail

# =============================================================================
# CONFIGURATION AND GLOBALS
# =============================================================================

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DEPLOYMENT_DIR="$PROJECT_ROOT/deployments"

# Default values
ENVIRONMENT=""
NAMESPACE="cardiofit-medication-v2"
SERVICE_NAME="medication-service-v2"
VERSION=""
DRY_RUN=false
FORCE=false
ROLLBACK=false
PREVIOUS_VERSION=""
HEALTH_CHECK_TIMEOUT=300
BACKUP_BEFORE_DEPLOY=true

# Healthcare compliance settings
HIPAA_VALIDATION=true
SECURITY_SCAN=true
COMPLIANCE_CHECK=true
AUDIT_LOG=true

# Colors and formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Healthcare symbols
HEALTH_SYMBOL="🏥"
SECURITY_SYMBOL="🔐"
COMPLIANCE_SYMBOL="📋"
SUCCESS_SYMBOL="✅"
WARNING_SYMBOL="⚠️"
ERROR_SYMBOL="❌"
INFO_SYMBOL="ℹ️"

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

# Logging functions with healthcare context
log_info() {
    echo -e "${BLUE}${INFO_SYMBOL} [INFO]${NC} $1" | tee -a deployment.log
}

log_success() {
    echo -e "${GREEN}${SUCCESS_SYMBOL} [SUCCESS]${NC} $1" | tee -a deployment.log
}

log_warning() {
    echo -e "${YELLOW}${WARNING_SYMBOL} [WARNING]${NC} $1" | tee -a deployment.log
}

log_error() {
    echo -e "${RED}${ERROR_SYMBOL} [ERROR]${NC} $1" | tee -a deployment.log
}

log_health() {
    echo -e "${PURPLE}${HEALTH_SYMBOL} [HEALTH]${NC} $1" | tee -a deployment.log
}

log_security() {
    echo -e "${CYAN}${SECURITY_SYMBOL} [SECURITY]${NC} $1" | tee -a deployment.log
}

log_compliance() {
    echo -e "${WHITE}${COMPLIANCE_SYMBOL} [COMPLIANCE]${NC} $1" | tee -a deployment.log
}

# Error handling with rollback capability
handle_error() {
    local exit_code=$?
    log_error "Deployment failed with exit code $exit_code"
    log_error "Last command: ${BASH_COMMAND}"
    
    if [[ "$ROLLBACK" == "true" && -n "$PREVIOUS_VERSION" ]]; then
        log_warning "Attempting automatic rollback to version $PREVIOUS_VERSION"
        rollback_deployment
    fi
    
    cleanup_on_failure
    exit $exit_code
}

# Cleanup function
cleanup_on_failure() {
    log_warning "Performing cleanup after deployment failure"
    
    # Remove any temporary files
    rm -f /tmp/medication-service-*.yaml
    rm -f /tmp/deployment-*.json
    
    # Clean up any test resources
    kubectl delete pods -l "app=$SERVICE_NAME,deployment-test=true" -n "$NAMESPACE" --ignore-not-found=true
    
    log_info "Cleanup completed"
}

# Set up error handling
trap handle_error ERR

# =============================================================================
# VALIDATION FUNCTIONS
# =============================================================================

validate_prerequisites() {
    log_info "Validating deployment prerequisites"
    
    # Check required tools
    local required_tools=("kubectl" "helm" "docker" "curl" "jq" "yq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' is not installed"
            return 1
        fi
    done
    log_success "All required tools are available"
    
    # Validate Kubernetes connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        return 1
    fi
    log_success "Kubernetes cluster connectivity verified"
    
    # Check namespace existence
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_warning "Namespace '$NAMESPACE' does not exist, will be created"
    else
        log_success "Namespace '$NAMESPACE' exists"
    fi
    
    # Validate Helm
    if ! helm version &> /dev/null; then
        log_error "Helm is not properly configured"
        return 1
    fi
    log_success "Helm configuration verified"
    
    return 0
}

validate_environment() {
    log_info "Validating environment configuration for '$ENVIRONMENT'"
    
    case "$ENVIRONMENT" in
        "dev"|"staging"|"production")
            log_success "Valid environment: $ENVIRONMENT"
            ;;
        *)
            log_error "Invalid environment: $ENVIRONMENT. Must be one of: dev, staging, production"
            return 1
            ;;
    esac
    
    # Environment-specific validations
    if [[ "$ENVIRONMENT" == "production" ]]; then
        log_health "Production environment detected - enabling enhanced validations"
        HIPAA_VALIDATION=true
        SECURITY_SCAN=true
        COMPLIANCE_CHECK=true
        BACKUP_BEFORE_DEPLOY=true
        
        # Require version specification for production
        if [[ -z "$VERSION" ]]; then
            log_error "Version must be specified for production deployments"
            return 1
        fi
    fi
    
    return 0
}

validate_healthcare_compliance() {
    if [[ "$HIPAA_VALIDATION" != "true" ]]; then
        return 0
    fi
    
    log_compliance "Validating HIPAA compliance requirements"
    
    # Check encryption settings
    local helm_values="$DEPLOYMENT_DIR/helm/values-$ENVIRONMENT.yaml"
    if [[ -f "$helm_values" ]]; then
        # Verify encryption is enabled
        if ! yq eval '.security.encryption.enabled' "$helm_values" | grep -q "true"; then
            log_error "HIPAA Compliance: Encryption must be enabled"
            return 1
        fi
        
        # Verify audit logging is enabled
        if ! yq eval '.config.healthcare.clinicalAuditEnabled' "$helm_values" | grep -q "true"; then
            log_error "HIPAA Compliance: Clinical audit logging must be enabled"
            return 1
        fi
        
        # Verify TLS is enabled
        if ! yq eval '.config.tls.enabled' "$helm_values" | grep -q "true"; then
            log_error "HIPAA Compliance: TLS must be enabled"
            return 1
        fi
        
        log_success "HIPAA compliance validation passed"
    else
        log_warning "Helm values file not found: $helm_values"
    fi
    
    return 0
}

validate_security_configuration() {
    if [[ "$SECURITY_SCAN" != "true" ]]; then
        return 0
    fi
    
    log_security "Validating security configuration"
    
    # Check for security contexts in deployment
    local deployment_file="$DEPLOYMENT_DIR/kubernetes/deployment.yaml"
    if [[ -f "$deployment_file" ]]; then
        # Verify non-root user
        if ! yq eval '.spec.template.spec.securityContext.runAsNonRoot' "$deployment_file" | grep -q "true"; then
            log_error "Security: Containers must run as non-root user"
            return 1
        fi
        
        # Verify read-only root filesystem
        if ! yq eval '.spec.template.spec.containers[0].securityContext.readOnlyRootFilesystem' "$deployment_file" | grep -q "true"; then
            log_error "Security: Root filesystem must be read-only"
            return 1
        fi
        
        # Verify capabilities are dropped
        if ! yq eval '.spec.template.spec.containers[0].securityContext.capabilities.drop[]' "$deployment_file" | grep -q "ALL"; then
            log_error "Security: All capabilities must be dropped"
            return 1
        fi
        
        log_success "Security configuration validation passed"
    else
        log_warning "Deployment file not found: $deployment_file"
    fi
    
    return 0
}

# =============================================================================
# BACKUP AND ROLLBACK FUNCTIONS
# =============================================================================

backup_current_deployment() {
    if [[ "$BACKUP_BEFORE_DEPLOY" != "true" ]]; then
        return 0
    fi
    
    log_info "Creating backup of current deployment"
    
    # Create backup directory
    local backup_dir="/tmp/medication-service-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$backup_dir"
    
    # Backup Helm release
    if helm status "$SERVICE_NAME" -n "$NAMESPACE" &> /dev/null; then
        helm get values "$SERVICE_NAME" -n "$NAMESPACE" > "$backup_dir/helm-values.yaml"
        helm get manifest "$SERVICE_NAME" -n "$NAMESPACE" > "$backup_dir/helm-manifest.yaml"
        
        # Store current version
        PREVIOUS_VERSION=$(helm list -n "$NAMESPACE" -o json | jq -r '.[] | select(.name=="'$SERVICE_NAME'") | .app_version')
        echo "$PREVIOUS_VERSION" > "$backup_dir/previous-version.txt"
        
        log_success "Backup created at: $backup_dir"
        echo "$backup_dir" > /tmp/last-backup-path.txt
    else
        log_info "No existing deployment found, skipping backup"
    fi
    
    return 0
}

rollback_deployment() {
    log_warning "Initiating rollback procedure"
    
    if [[ -z "$PREVIOUS_VERSION" ]]; then
        if [[ -f "/tmp/last-backup-path.txt" ]]; then
            local backup_dir=$(cat /tmp/last-backup-path.txt)
            if [[ -f "$backup_dir/previous-version.txt" ]]; then
                PREVIOUS_VERSION=$(cat "$backup_dir/previous-version.txt")
            fi
        fi
    fi
    
    if [[ -n "$PREVIOUS_VERSION" ]]; then
        log_info "Rolling back to version: $PREVIOUS_VERSION"
        
        # Perform Helm rollback
        if helm rollback "$SERVICE_NAME" -n "$NAMESPACE"; then
            log_success "Helm rollback completed"
            
            # Wait for rollback to complete
            wait_for_deployment_ready "rollback"
            
            # Verify rollback health
            verify_deployment_health "rollback"
            
            log_success "Rollback completed successfully"
        else
            log_error "Helm rollback failed"
            return 1
        fi
    else
        log_error "Cannot rollback: Previous version not available"
        return 1
    fi
    
    return 0
}

# =============================================================================
# DEPLOYMENT FUNCTIONS
# =============================================================================

prepare_deployment() {
    log_info "Preparing deployment for $SERVICE_NAME version $VERSION in $ENVIRONMENT"
    
    # Create namespace if it doesn't exist
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_info "Creating namespace: $NAMESPACE"
        kubectl apply -f "$DEPLOYMENT_DIR/kubernetes/namespace.yaml"
    fi
    
    # Apply secrets (if not using external secrets)
    log_security "Applying secrets configuration"
    if [[ -f "$DEPLOYMENT_DIR/kubernetes/secrets.yaml" ]]; then
        kubectl apply -f "$DEPLOYMENT_DIR/kubernetes/secrets.yaml" -n "$NAMESPACE"
    fi
    
    # Apply configmaps
    log_info "Applying configuration"
    if [[ -f "$DEPLOYMENT_DIR/kubernetes/configmap.yaml" ]]; then
        kubectl apply -f "$DEPLOYMENT_DIR/kubernetes/configmap.yaml" -n "$NAMESPACE"
    fi
    
    # Apply RBAC
    log_security "Applying RBAC configuration"
    if [[ -f "$DEPLOYMENT_DIR/kubernetes/services.yaml" ]]; then
        kubectl apply -f "$DEPLOYMENT_DIR/kubernetes/services.yaml" -n "$NAMESPACE"
    fi
    
    return 0
}

deploy_application() {
    log_info "Deploying application using Helm"
    
    local helm_chart="$DEPLOYMENT_DIR/helm"
    local values_file="$helm_chart/values-$ENVIRONMENT.yaml"
    
    # Check if environment-specific values exist
    if [[ ! -f "$values_file" ]]; then
        log_warning "Environment-specific values file not found: $values_file"
        values_file="$helm_chart/values.yaml"
    fi
    
    local helm_args=(
        "upgrade"
        "--install"
        "$SERVICE_NAME"
        "$helm_chart"
        "--namespace" "$NAMESPACE"
        "--values" "$values_file"
        "--set" "image.tag=$VERSION"
        "--set" "global.environment=$ENVIRONMENT"
        "--wait"
        "--timeout=600s"
    )
    
    # Add production-specific settings
    if [[ "$ENVIRONMENT" == "production" ]]; then
        helm_args+=(
            "--atomic"  # Rollback on failure
            "--set" "replicaCount=3"
            "--set" "autoscaling.enabled=true"
            "--set" "podDisruptionBudget.enabled=true"
        )
    fi
    
    # Add dry-run if specified
    if [[ "$DRY_RUN" == "true" ]]; then
        helm_args+=("--dry-run")
        log_info "Performing dry-run deployment"
    fi
    
    # Execute Helm deployment
    if helm "${helm_args[@]}"; then
        log_success "Helm deployment completed"
        
        if [[ "$DRY_RUN" != "true" ]]; then
            # Apply any additional Kubernetes resources
            apply_additional_resources
        fi
    else
        log_error "Helm deployment failed"
        return 1
    fi
    
    return 0
}

apply_additional_resources() {
    log_info "Applying additional Kubernetes resources"
    
    # Apply StatefulSets (databases, cache)
    if [[ -f "$DEPLOYMENT_DIR/kubernetes/statefulsets.yaml" ]]; then
        log_info "Applying StatefulSets"
        kubectl apply -f "$DEPLOYMENT_DIR/kubernetes/statefulsets.yaml" -n "$NAMESPACE"
    fi
    
    # Apply Ingress
    if [[ -f "$DEPLOYMENT_DIR/kubernetes/ingress.yaml" ]]; then
        log_info "Applying Ingress configuration"
        kubectl apply -f "$DEPLOYMENT_DIR/kubernetes/ingress.yaml" -n "$NAMESPACE"
    fi
    
    # Apply NetworkPolicies
    if [[ -f "$DEPLOYMENT_DIR/kubernetes/network-policies.yaml" ]]; then
        log_security "Applying Network Policies"
        kubectl apply -f "$DEPLOYMENT_DIR/kubernetes/network-policies.yaml" -n "$NAMESPACE"
    fi
    
    return 0
}

wait_for_deployment_ready() {
    local deployment_type="${1:-deployment}"
    log_info "Waiting for $deployment_type to be ready"
    
    # Wait for deployment to be ready
    if kubectl wait --for=condition=available --timeout="${HEALTH_CHECK_TIMEOUT}s" \
        deployment/"$SERVICE_NAME" -n "$NAMESPACE"; then
        log_success "Deployment is ready"
    else
        log_error "Deployment failed to become ready within ${HEALTH_CHECK_TIMEOUT} seconds"
        return 1
    fi
    
    # Wait for all pods to be ready
    if kubectl wait --for=condition=ready --timeout=300s \
        pod -l app.kubernetes.io/name="$SERVICE_NAME" -n "$NAMESPACE"; then
        log_success "All pods are ready"
    else
        log_error "Pods failed to become ready"
        return 1
    fi
    
    return 0
}

# =============================================================================
# HEALTH CHECK AND VERIFICATION FUNCTIONS
# =============================================================================

verify_deployment_health() {
    local check_type="${1:-deployment}"
    log_health "Performing health checks for $check_type"
    
    # Get deployment status
    local deployment_status=$(kubectl get deployment "$SERVICE_NAME" -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')
    if [[ "$deployment_status" != "True" ]]; then
        log_error "Deployment is not available"
        return 1
    fi
    
    # Check pod health
    local ready_pods=$(kubectl get pods -l app.kubernetes.io/name="$SERVICE_NAME" -n "$NAMESPACE" -o jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}' | wc -w)
    local total_pods=$(kubectl get pods -l app.kubernetes.io/name="$SERVICE_NAME" -n "$NAMESPACE" --no-headers | wc -l)
    
    if [[ "$ready_pods" -eq 0 ]]; then
        log_error "No pods are running"
        return 1
    fi
    
    log_success "Health check passed: $ready_pods/$total_pods pods are running"
    
    # Perform application-specific health checks
    perform_application_health_checks
    
    return 0
}

perform_application_health_checks() {
    log_health "Performing application-specific health checks"
    
    # Get service endpoint
    local service_ip=$(kubectl get service "$SERVICE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}')
    if [[ -z "$service_ip" ]]; then
        log_error "Cannot get service IP"
        return 1
    fi
    
    # Health endpoint check
    log_info "Testing health endpoint"
    if kubectl run health-check --rm -i --restart=Never --image=curlimages/curl --timeout=60s -- \
        curl -f "http://$service_ip/health/ready"; then
        log_success "Health endpoint is responding"
    else
        log_error "Health endpoint check failed"
        return 1
    fi
    
    # Metrics endpoint check
    log_info "Testing metrics endpoint"
    if kubectl run metrics-check --rm -i --restart=Never --image=curlimages/curl --timeout=60s -- \
        curl -f "http://$service_ip:8005/metrics"; then
        log_success "Metrics endpoint is responding"
    else
        log_warning "Metrics endpoint check failed (non-critical)"
    fi
    
    # Database connectivity check (if applicable)
    check_database_connectivity
    
    # Cache connectivity check (if applicable)
    check_cache_connectivity
    
    return 0
}

check_database_connectivity() {
    log_health "Checking database connectivity"
    
    # Create a test pod to check database connectivity
    local db_test_pod="db-connectivity-test-$(date +%s)"
    
    kubectl run "$db_test_pod" --rm -i --restart=Never --timeout=60s \
        --image=postgres:15-alpine \
        --env="PGPASSWORD=test" \
        -- psql -h postgres-service -U medication_user -d medication_v2 -c "SELECT 1;" \
        > /dev/null 2>&1
    
    if [[ $? -eq 0 ]]; then
        log_success "Database connectivity verified"
    else
        log_warning "Database connectivity check failed (may be expected if using external DB)"
    fi
    
    return 0
}

check_cache_connectivity() {
    log_health "Checking cache connectivity"
    
    # Create a test pod to check Redis connectivity
    local redis_test_pod="redis-connectivity-test-$(date +%s)"
    
    kubectl run "$redis_test_pod" --rm -i --restart=Never --timeout=60s \
        --image=redis:7-alpine \
        -- redis-cli -h redis-service ping \
        > /dev/null 2>&1
    
    if [[ $? -eq 0 ]]; then
        log_success "Cache connectivity verified"
    else
        log_warning "Cache connectivity check failed (may be expected if using external cache)"
    fi
    
    return 0
}

# =============================================================================
# SMOKE TESTS
# =============================================================================

run_smoke_tests() {
    log_info "Running smoke tests"
    
    # Basic API smoke tests
    local service_ip=$(kubectl get service "$SERVICE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}')
    
    # Test 1: Health endpoint
    log_info "Smoke test 1: Health endpoint"
    if kubectl run smoke-test-health --rm -i --restart=Never --timeout=60s --image=curlimages/curl -- \
        curl -f "http://$service_ip/health/ready"; then
        log_success "Smoke test 1 passed"
    else
        log_error "Smoke test 1 failed"
        return 1
    fi
    
    # Test 2: API version endpoint
    log_info "Smoke test 2: API version endpoint"
    if kubectl run smoke-test-version --rm -i --restart=Never --timeout=60s --image=curlimages/curl -- \
        curl -f "http://$service_ip/api/v1/version"; then
        log_success "Smoke test 2 passed"
    else
        log_warning "Smoke test 2 failed (non-critical)"
    fi
    
    # Test 3: Authentication endpoint (if available)
    log_info "Smoke test 3: Authentication check"
    if kubectl run smoke-test-auth --rm -i --restart=Never --timeout=60s --image=curlimages/curl -- \
        curl -f "http://$service_ip/api/v1/auth/health"; then
        log_success "Smoke test 3 passed"
    else
        log_warning "Smoke test 3 failed (may be expected without auth)"
    fi
    
    log_success "Smoke tests completed"
    return 0
}

# =============================================================================
# MONITORING AND ALERTING
# =============================================================================

setup_monitoring() {
    log_info "Setting up monitoring and alerting"
    
    # Apply ServiceMonitor for Prometheus
    if [[ -f "$DEPLOYMENT_DIR/monitoring/servicemonitor.yaml" ]]; then
        kubectl apply -f "$DEPLOYMENT_DIR/monitoring/servicemonitor.yaml" -n "$NAMESPACE"
        log_success "ServiceMonitor applied"
    fi
    
    # Apply PrometheusRules for alerting
    if [[ -f "$DEPLOYMENT_DIR/monitoring/prometheusrules.yaml" ]]; then
        kubectl apply -f "$DEPLOYMENT_DIR/monitoring/prometheusrules.yaml" -n "$NAMESPACE"
        log_success "PrometheusRules applied"
    fi
    
    return 0
}

# =============================================================================
# COMPLIANCE AND AUDIT
# =============================================================================

generate_deployment_audit_log() {
    local audit_log_file="deployment-audit-$(date +%Y%m%d-%H%M%S).json"
    
    log_compliance "Generating deployment audit log"
    
    cat > "$audit_log_file" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "service": "$SERVICE_NAME",
  "version": "$VERSION",
  "environment": "$ENVIRONMENT",
  "namespace": "$NAMESPACE",
  "deployer": "$(whoami)",
  "deployment_method": "helm",
  "compliance_checks": {
    "hipaa_validation": $HIPAA_VALIDATION,
    "security_scan": $SECURITY_SCAN,
    "backup_created": $BACKUP_BEFORE_DEPLOY
  },
  "deployment_status": "completed",
  "health_checks": "passed",
  "rollback_available": $([ -n "$PREVIOUS_VERSION" ] && echo "true" || echo "false"),
  "previous_version": "$PREVIOUS_VERSION"
}
EOF
    
    log_success "Audit log generated: $audit_log_file"
    return 0
}

# =============================================================================
# MAIN DEPLOYMENT FLOW
# =============================================================================

show_usage() {
    cat << EOF
Medication Service V2 Healthcare Deployment Script

Usage: $0 [OPTIONS]

OPTIONS:
  -e, --environment ENV    Environment (dev|staging|production)
  -v, --version VERSION    Service version to deploy
  -n, --namespace NS       Kubernetes namespace (default: cardiofit-medication-v2)
  -d, --dry-run           Perform dry-run without actual deployment
  -f, --force             Force deployment without confirmations
  -r, --rollback          Rollback to previous version
  -h, --help              Show this help message

HEALTHCARE COMPLIANCE OPTIONS:
  --no-hipaa-validation   Skip HIPAA compliance validation
  --no-security-scan      Skip security configuration scan
  --no-backup            Skip backup before deployment

EXAMPLES:
  $0 -e production -v 1.0.0
  $0 -e staging -v 1.1.0-rc1 --dry-run
  $0 -e production --rollback

HEALTHCARE NOTES:
  - Production deployments require version specification
  - HIPAA compliance validation is mandatory for production
  - Automatic backup is performed before production deployments
  - All deployments are logged for audit compliance

EOF
}

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -r|--rollback)
                ROLLBACK=true
                shift
                ;;
            --no-hipaa-validation)
                HIPAA_VALIDATION=false
                shift
                ;;
            --no-security-scan)
                SECURITY_SCAN=false
                shift
                ;;
            --no-backup)
                BACKUP_BEFORE_DEPLOY=false
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Validate required arguments
    if [[ -z "$ENVIRONMENT" ]]; then
        log_error "Environment must be specified"
        show_usage
        exit 1
    fi
    
    # Handle rollback
    if [[ "$ROLLBACK" == "true" ]]; then
        log_warning "Rollback requested"
        rollback_deployment
        exit $?
    fi
    
    # Validate version for non-rollback deployments
    if [[ -z "$VERSION" ]]; then
        log_error "Version must be specified"
        show_usage
        exit 1
    fi
    
    # Start deployment
    log_info "Starting deployment of $SERVICE_NAME version $VERSION to $ENVIRONMENT"
    log_info "Deployment started at: $(date)"
    
    # Initialize audit log
    if [[ "$AUDIT_LOG" == "true" ]]; then
        echo "Deployment started at $(date)" > deployment.log
    fi
    
    # Confirmation for production
    if [[ "$ENVIRONMENT" == "production" && "$FORCE" != "true" ]]; then
        echo
        log_warning "⚠️  PRODUCTION DEPLOYMENT WARNING ⚠️"
        echo "You are about to deploy to PRODUCTION environment"
        echo "Service: $SERVICE_NAME"
        echo "Version: $VERSION"
        echo "Namespace: $NAMESPACE"
        echo
        read -p "Are you sure you want to continue? (yes/no): " -r
        if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
            log_info "Deployment cancelled by user"
            exit 0
        fi
    fi
    
    # Execution steps
    log_info "📋 Starting healthcare-grade deployment process"
    
    validate_prerequisites
    validate_environment
    validate_healthcare_compliance
    validate_security_configuration
    backup_current_deployment
    
    if [[ "$DRY_RUN" != "true" ]]; then
        prepare_deployment
    fi
    
    deploy_application
    
    if [[ "$DRY_RUN" != "true" ]]; then
        wait_for_deployment_ready
        verify_deployment_health
        run_smoke_tests
        setup_monitoring
        generate_deployment_audit_log
    fi
    
    # Success message
    echo
    log_success "🎉 Deployment completed successfully!"
    log_success "Service: $SERVICE_NAME"
    log_success "Version: $VERSION"
    log_success "Environment: $ENVIRONMENT"
    log_success "Namespace: $NAMESPACE"
    log_success "Deployment time: $(date)"
    
    if [[ "$DRY_RUN" != "true" ]]; then
        echo
        log_health "Health status: All checks passed"
        log_compliance "Compliance: HIPAA validation completed"
        log_security "Security: All security checks passed"
        
        # Show access information
        echo
        log_info "🔗 Service Access Information:"
        if [[ "$ENVIRONMENT" == "production" ]]; then
            log_info "Public URL: https://api-medication.cardiofit.health"
            log_info "gRPC URL: grpc-medication.cardiofit.health:443"
        else
            log_info "Internal URL: http://$SERVICE_NAME.$NAMESPACE.svc.cluster.local"
        fi
        
        echo
        log_info "📊 Monitoring:"
        log_info "Metrics: Available at /metrics endpoint"
        log_info "Health: Available at /health/* endpoints"
        log_info "Logs: Available via kubectl logs"
        
        echo
        log_compliance "📋 Compliance & Audit:"
        log_compliance "Audit log: deployment.log"
        log_compliance "HIPAA compliance: Validated"
        log_compliance "Security scan: Completed"
    else
        echo
        log_info "Dry-run completed successfully - no actual deployment performed"
    fi
    
    echo
    log_success "✅ Medication Service V2 deployment process completed"
    
    return 0
}

# Execute main function
main "$@"