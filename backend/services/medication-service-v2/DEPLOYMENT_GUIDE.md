# Medication Service V2 - Deployment Guide

## Healthcare-Grade Production Deployment Documentation

### Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Prerequisites](#prerequisites)
4. [Environment Configuration](#environment-configuration)
5. [Security & Compliance](#security--compliance)
6. [Deployment Methods](#deployment-methods)
7. [Monitoring & Observability](#monitoring--observability)
8. [Disaster Recovery](#disaster-recovery)
9. [Troubleshooting](#troubleshooting)
10. [Compliance Validation](#compliance-validation)

## Overview

The Medication Service V2 is a FHIR-compliant healthcare microservice designed for production healthcare environments with strict compliance requirements including HIPAA, SOC2, and ISO27001. This guide provides comprehensive deployment instructions for secure, scalable, and compliant deployments.

### Key Features
- 🏥 **HIPAA Compliant**: End-to-end encryption, audit logging, access controls
- 🔐 **Security Hardened**: Container security, network policies, vulnerability scanning
- ⚡ **High Performance**: <250ms latency targets, auto-scaling, caching
- 📊 **Observable**: Comprehensive monitoring, tracing, alerting
- 🔄 **Resilient**: High availability, disaster recovery, automated rollback

### Service Components
- **Recipe Resolver**: Clinical medication intelligence with rule-based processing
- **Context Gateway**: Real-time clinical decision support integration
- **4-Phase Workflow**: Validation → Resolution → Calculation → Snapshot
- **Rust Clinical Engine**: High-performance rule evaluation engine
- **Apollo Federation**: GraphQL API composition and federation
- **Multi-Level Caching**: Performance-optimized with HIPAA-compliant encryption

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Load Balancer                            │
│                      (SSL Termination)                         │
└─────────────────────────┬───────────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────────┐
│                    Ingress Controller                           │
│                 (Rate Limiting + WAF)                          │
└─────────────────────────┬───────────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────────┐
│               Medication Service V2 (3+ Replicas)              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌──────────┐  │
│  │Recipe       │ │Context      │ │Workflow     │ │Clinical  │  │
│  │Resolver     │ │Gateway      │ │Orchestration│ │Engine    │  │
│  │(Port 8005)  │ │Integration  │ │(4-Phase)    │ │(Rust)    │  │
│  └─────────────┘ └─────────────┘ └─────────────┘ └──────────┘  │
└─────────────────────────┬───────────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────────┐
│                     Data Layer                                  │
│  ┌─────────────────┐              ┌─────────────────┐           │
│  │PostgreSQL       │              │Redis Cluster    │           │
│  │Cluster (HA)     │              │(Multi-Level     │           │
│  │- Primary        │              │ Caching)        │           │
│  │- 2x Replicas    │              │- 3x Nodes       │           │
│  │- SSL/TLS        │              │- Encryption     │           │
│  │- Encrypted      │              │- Auth Required  │           │
│  └─────────────────┘              └─────────────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

### Security Architecture
```
┌─────────────────────────────────────────────────────────────────┐
│                    Security Boundaries                          │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Network Security                             │  │
│  │  • Network Policies (Strict Egress/Ingress)             │  │
│  │  • Service Mesh (mTLS)                                   │  │
│  │  • Private Subnets                                       │  │
│  │  • Security Groups (Least Privilege)                    │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │            Application Security                           │  │
│  │  • Non-root containers                                   │  │
│  │  • Read-only filesystems                                 │  │
│  │  • Dropped capabilities (ALL)                           │  │
│  │  • Security contexts (restricted)                       │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │               Data Security                               │  │
│  │  • Encryption at rest (KMS)                             │  │
│  │  • Encryption in transit (TLS 1.3)                      │  │
│  │  • Secret management (Vault/AWS Secrets)                │  │
│  │  • RBAC + Service Accounts                              │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

### Infrastructure Requirements
- **Kubernetes Cluster**: v1.25+ with Pod Security Standards
- **Node Resources**: Minimum 4 vCPU, 8GB RAM per node
- **Storage**: High-performance SSD storage class
- **Network**: Private subnets with NAT Gateway for egress

### Security Requirements
- **TLS Certificates**: Valid SSL/TLS certificates (Let's Encrypt or CA)
- **KMS Keys**: Encryption keys for data at rest
- **Secrets Management**: HashiCorp Vault, AWS Secrets Manager, or similar
- **Network Security**: WAF, DDoS protection, Network policies

### Compliance Requirements
- **HIPAA**: Business Associate Agreement, audit logging, encryption
- **SOC2**: Security controls, monitoring, incident response
- **ISO27001**: Information security management system

### Required Tools
```bash
# Core tools
kubectl >= 1.25.0
helm >= 3.13.0
docker >= 24.0.0

# Security scanning
trivy >= 0.45.0
cosign >= 2.0.0

# Healthcare utilities
openssl >= 3.0.0
jq >= 1.6
yq >= 4.0.0
```

## Environment Configuration

### Development Environment
```bash
# Resource allocation
CPU_REQUEST=100m
CPU_LIMIT=250m
MEMORY_REQUEST=128Mi
MEMORY_LIMIT=256Mi
REPLICAS=1

# Features
DEBUG_ENABLED=true
SWAGGER_ENABLED=true
SECURITY_LEVEL=basic
```

### Staging Environment  
```bash
# Resource allocation
CPU_REQUEST=250m
CPU_LIMIT=500m
MEMORY_REQUEST=256Mi
MEMORY_LIMIT=512Mi
REPLICAS=2

# Features
DEBUG_ENABLED=false
SWAGGER_ENABLED=true
SECURITY_LEVEL=high
HIPAA_COMPLIANCE=true
```

### Production Environment
```bash
# Resource allocation
CPU_REQUEST=250m
CPU_LIMIT=500m
MEMORY_REQUEST=256Mi
MEMORY_LIMIT=512Mi
REPLICAS=3

# Security (all required)
DEBUG_ENABLED=false
SWAGGER_ENABLED=false
SECURITY_LEVEL=maximum
HIPAA_COMPLIANCE=true
SOC2_COMPLIANCE=true
TLS_ENABLED=true
AUDIT_LOGGING=true
```

## Security & Compliance

### HIPAA Compliance Checklist

#### ✅ Administrative Safeguards
- [ ] Assigned security responsibility
- [ ] Workforce training and access management
- [ ] Information access management procedures
- [ ] Security awareness and training
- [ ] Security incident procedures
- [ ] Contingency plan (disaster recovery)
- [ ] Regular security evaluations

#### ✅ Physical Safeguards
- [ ] Facility access controls (cloud provider)
- [ ] Workstation use restrictions
- [ ] Device and media controls
- [ ] Secure data centers (AWS/Azure/GCP)

#### ✅ Technical Safeguards
- [ ] Access control (RBAC + IAM)
- [ ] Audit controls (comprehensive logging)
- [ ] Integrity controls (encryption + checksums)
- [ ] Person or entity authentication (mTLS + JWT)
- [ ] Transmission security (TLS 1.3)

### Security Implementation

#### Container Security
```yaml
# Pod Security Context
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  runAsGroup: 65532
  fsGroup: 65532
  seccompProfile:
    type: RuntimeDefault

# Container Security Context  
containers:
- name: medication-service
  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop: ["ALL"]
```

#### Network Security
```yaml
# Network Policy Example
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: medication-service-netpol
spec:
  podSelector:
    matchLabels:
      app: medication-service-v2
  policyTypes: ["Ingress", "Egress"]
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
  egress:
  - to: []
    ports:
    - protocol: UDP
      port: 53  # DNS only
```

#### Encryption Configuration
```yaml
# Database encryption
postgresql:
  primary:
    configuration: |
      ssl = on
      ssl_cert_file = '/etc/ssl/certs/server.crt'
      ssl_key_file = '/etc/ssl/private/server.key'
      ssl_ca_file = '/etc/ssl/certs/ca.crt'
      password_encryption = scram-sha-256

# Redis encryption
redis:
  auth:
    enabled: true
  tls:
    enabled: true
    authClients: true
```

## Deployment Methods

### Method 1: Automated Deployment Script (Recommended)

```bash
# Production deployment
./deployments/scripts/deploy.sh \
  --environment production \
  --version 1.0.0 \
  --namespace cardiofit-medication-v2

# Staging deployment with dry-run
./deployments/scripts/deploy.sh \
  --environment staging \
  --version 1.1.0-rc1 \
  --dry-run

# Emergency rollback
./deployments/scripts/deploy.sh \
  --environment production \
  --rollback
```

### Method 2: Manual Helm Deployment

```bash
# 1. Create namespace and apply base resources
kubectl apply -f deployments/kubernetes/namespace.yaml
kubectl apply -f deployments/kubernetes/configmap.yaml
kubectl apply -f deployments/kubernetes/secrets.yaml

# 2. Deploy with Helm
helm upgrade --install medication-service-v2 \
  deployments/helm \
  --namespace cardiofit-medication-v2 \
  --values deployments/helm/values-production.yaml \
  --set image.tag=1.0.0 \
  --set global.environment=production \
  --wait --timeout=600s

# 3. Apply additional resources
kubectl apply -f deployments/kubernetes/statefulsets.yaml
kubectl apply -f deployments/kubernetes/ingress.yaml
```

### Method 3: GitOps with ArgoCD

```yaml
# argocd-application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: medication-service-v2
  namespace: argocd
spec:
  project: healthcare
  source:
    repoURL: https://github.com/clinical-synthesis-hub/cardiofit
    targetRevision: main
    path: backend/services/medication-service-v2/deployments/helm
    helm:
      valueFiles:
      - values-production.yaml
      parameters:
      - name: image.tag
        value: "1.0.0"
  destination:
    server: https://kubernetes.default.svc
    namespace: cardiofit-medication-v2
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
    - PrunePropagationPolicy=foreground
```

### Method 4: Terraform + Helm

```bash
# 1. Deploy infrastructure
cd deployments/terraform
terraform init
terraform plan -var="environment=production"
terraform apply -var="environment=production"

# 2. Deploy application
helm upgrade --install medication-service-v2 \
  ../helm \
  --namespace $(terraform output -raw kubernetes_namespace) \
  --values ../helm/values-production.yaml
```

## Monitoring & Observability

### Metrics Collection
- **Prometheus**: Application and infrastructure metrics
- **Custom Metrics**: Healthcare-specific KPIs (medication calculations, clinical decision latency)
- **SLI Monitoring**: Response time, error rate, availability

### Logging
- **Structured Logging**: JSON format with correlation IDs
- **Audit Logs**: HIPAA-compliant access and modification logs
- **Log Retention**: 7 years minimum for healthcare compliance
- **Log Encryption**: All logs encrypted at rest and in transit

### Distributed Tracing
- **Jaeger**: Request flow tracing across microservices
- **Sampling**: Configurable sampling rates (production: 10%)
- **Healthcare Context**: Trace clinical workflows and decision paths

### Alerting Rules

```yaml
# Critical Healthcare Alerts
groups:
- name: medication-service-critical
  rules:
  - alert: MedicationServiceDown
    expr: up{job="medication-service-v2"} == 0
    for: 1m
    severity: critical
    
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.01
    for: 2m
    severity: critical
    
  - alert: DatabaseConnectionFailed
    expr: database_connection_active == 0
    for: 30s
    severity: critical
    
  - alert: ClinicalLatencyHigh
    expr: histogram_quantile(0.99, clinical_decision_duration_seconds) > 0.5
    for: 5m
    severity: warning
```

### Health Checks

```go
// Health check endpoints
/health/live     - Liveness probe (basic service health)
/health/ready    - Readiness probe (dependencies health)  
/health/startup  - Startup probe (initialization complete)

// Healthcare-specific health checks
/health/clinical - Clinical decision engine status
/health/fhir     - FHIR compliance validation
/health/phi      - PHI handling system status
```

## Disaster Recovery

### Backup Strategy
- **Database**: Continuous backup with 30-day retention
- **Application State**: Snapshot-based backups every 4 hours
- **Configuration**: Git-based versioning with rollback capability
- **Cross-Region**: Automated replication to DR region

### RTO/RPO Targets
- **RTO (Recovery Time Objective)**: 60 minutes
- **RPO (Recovery Point Objective)**: 15 minutes
- **Data Loss**: Zero tolerance for PHI data

### DR Procedures

```bash
# 1. Declare disaster
./scripts/disaster-response.sh --declare --region us-east-1

# 2. Activate DR site
./scripts/activate-dr.sh --target-region us-west-2

# 3. Verify DR deployment
./scripts/verify-dr.sh --environment production-dr

# 4. Update DNS/Load Balancer
./scripts/failover-traffic.sh --to-dr

# 5. Monitor and validate
./scripts/monitor-dr.sh --continuous
```

### Backup Verification
```bash
# Weekly backup testing
./scripts/test-backup-restore.sh --environment staging-dr

# Database restore test
./scripts/test-db-restore.sh --backup-timestamp "2024-01-15T02:00:00Z"

# Application state restore
./scripts/test-app-restore.sh --snapshot-id "snap-12345"
```

## Troubleshooting

### Common Issues

#### 1. Pod Startup Failures
```bash
# Check pod status
kubectl get pods -n cardiofit-medication-v2 -l app=medication-service-v2

# View pod logs
kubectl logs -n cardiofit-medication-v2 -l app=medication-service-v2 --tail=100

# Describe problematic pod
kubectl describe pod <pod-name> -n cardiofit-medication-v2

# Check events
kubectl get events -n cardiofit-medication-v2 --sort-by=.metadata.creationTimestamp
```

#### 2. Database Connection Issues
```bash
# Test database connectivity
kubectl run db-test --rm -i --restart=Never \
  --image=postgres:15-alpine \
  --env="PGPASSWORD=<password>" \
  -- psql -h postgres-service -U medication_user -d medication_v2 -c "SELECT 1;"

# Check database pod status
kubectl get pods -n cardiofit-medication-v2 -l app=postgres

# View database logs
kubectl logs -n cardiofit-medication-v2 -l app=postgres --tail=50
```

#### 3. Cache Connection Problems
```bash
# Test Redis connectivity
kubectl run redis-test --rm -i --restart=Never \
  --image=redis:7-alpine \
  -- redis-cli -h redis-service -a <password> ping

# Check Redis cluster status
kubectl exec -n cardiofit-medication-v2 redis-0 -- redis-cli -a <password> cluster info
```

#### 4. Performance Issues
```bash
# Check resource usage
kubectl top pods -n cardiofit-medication-v2

# View HPA status
kubectl get hpa -n cardiofit-medication-v2

# Check service mesh metrics
kubectl exec -n istio-system <istio-proxy-pod> -- \
  curl localhost:15000/stats | grep medication-service
```

#### 5. Security/Compliance Issues
```bash
# Verify network policies
kubectl get networkpolicies -n cardiofit-medication-v2

# Check pod security context
kubectl get pods <pod-name> -n cardiofit-medication-v2 -o yaml | grep -A 10 securityContext

# Validate TLS certificates
kubectl get secret medication-service-tls -n cardiofit-medication-v2 -o yaml
```

### Debugging Tools

#### Interactive Debugging Pod
```bash
# Deploy debug pod with healthcare tools
kubectl run debug-pod --rm -i --tty --restart=Never \
  --image=nicolaka/netshoot \
  --namespace cardiofit-medication-v2 \
  -- /bin/bash

# Inside debug pod:
# Test service connectivity
curl http://medication-service-v2:8005/health/ready

# Check DNS resolution
nslookup medication-service-v2.cardiofit-medication-v2.svc.cluster.local

# Test database from within cluster
pg_isready -h postgres-service -p 5432
```

#### Performance Profiling
```bash
# Enable profiling endpoint (development only)
kubectl port-forward svc/medication-service-v2 6060:6060 -n cardiofit-medication-v2

# Capture CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# Capture memory profile  
curl http://localhost:6060/debug/pprof/heap > mem.prof

# Analyze with go tool pprof
go tool pprof cpu.prof
```

## Compliance Validation

### Automated Compliance Checks
```bash
# Run compliance validation suite
./scripts/compliance-check.sh --environment production

# HIPAA-specific validation
./scripts/hipaa-audit.sh --comprehensive

# SOC2 controls verification
./scripts/soc2-validation.sh --quarterly-report
```

### Manual Audit Procedures

#### 1. Access Control Audit
```bash
# Review RBAC permissions
kubectl auth can-i --list --as=system:serviceaccount:cardiofit-medication-v2:medication-service-v2

# Check service account permissions
kubectl describe clusterrole medication-service-v2-role
kubectl describe rolebinding medication-service-v2-binding -n cardiofit-medication-v2
```

#### 2. Encryption Verification
```bash
# Verify database encryption
kubectl exec postgres-0 -n cardiofit-medication-v2 -- \
  psql -U medication_user -d medication_v2 -c "SHOW ssl;"

# Check TLS configuration
openssl s_client -connect api-medication.cardiofit.health:443 -servername api-medication.cardiofit.health
```

#### 3. Audit Log Verification  
```bash
# Check audit log configuration
kubectl get configmap audit-policy -n kube-system -o yaml

# Verify log retention
aws logs describe-log-groups --log-group-name-prefix "/aws/eks/cardiofit"

# Sample audit entries
kubectl logs -n kube-system kube-apiserver-* | grep medication-service | tail -10
```

### Compliance Reports
- **Daily**: Automated security scan results
- **Weekly**: Access control and authentication audit
- **Monthly**: Comprehensive compliance report
- **Quarterly**: Full security assessment and penetration testing
- **Annually**: HIPAA compliance certification review

---

## Quick Start Commands

```bash
# Complete production deployment
./deployments/scripts/deploy.sh -e production -v 1.0.0

# Health check
kubectl get pods -n cardiofit-medication-v2
curl https://api-medication.cardiofit.health/health/ready

# View logs
kubectl logs -f deployment/medication-service-v2 -n cardiofit-medication-v2

# Monitor metrics  
kubectl port-forward svc/medication-service-v2 8005:8005 -n cardiofit-medication-v2
curl http://localhost:8005/metrics

# Emergency rollback
./deployments/scripts/deploy.sh -e production --rollback
```

## Support & Contacts

- **DevOps Team**: devops@cardiofit.health
- **Security Team**: security@cardiofit.health  
- **Compliance Team**: compliance@cardiofit.health
- **24/7 On-Call**: +1-555-CARDIO-1
- **Documentation**: https://docs.cardiofit.health/medication-service-v2
- **Incident Response**: https://status.cardiofit.health

---

**Note**: This service handles Protected Health Information (PHI). All deployments must comply with HIPAA regulations and organizational security policies. When in doubt, consult with the Healthcare Compliance team before proceeding.