# Runtime Platform Architecture

## Control Plane vs Data Plane

### Control Plane (Global, NO PHI)
```
┌─────────────────────────────────────────────────────────────┐
│                     CONTROL PLANE                            │
├─────────────────────────────────────────────────────────────┤
│  • Git repositories (clinical-knowledge-core)               │
│  • CI/CD pipelines                                          │
│  • Artifact registry (signed ELM, value sets)               │
│  • Configuration management                                  │
│  • Monitoring dashboards                                     │
└─────────────────────────────────────────────────────────────┘
```

### Data Plane (Per Region, PHI-Safe)
```
┌─────────────────────────────────────────────────────────────┐
│                  DATA PLANE (India)                          │
├─────────────────────────────────────────────────────────────┤
│  • FHIR Server (patient data)                               │
│  • Redis (expanded value sets)                              │
│  • CQL Executor                                              │
│  • Audit logs                                                │
│  • KMS (regional keys)                                       │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                  DATA PLANE (Australia)                      │
├─────────────────────────────────────────────────────────────┤
│  • FHIR Server (patient data)                               │
│  • Redis (expanded value sets)                              │
│  • CQL Executor                                              │
│  • Audit logs                                                │
│  • KMS (regional keys)                                       │
└─────────────────────────────────────────────────────────────┘
```

## Request Flow

```
Application Request
       │
       ▼
┌──────────────────┐
│  Load Balancer   │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐     ┌──────────────────┐
│   API Gateway    │────▶│  Auth Service    │
└────────┬─────────┘     └──────────────────┘
         │
         ▼
┌──────────────────┐     ┌──────────────────┐
│  CQL Executor    │────▶│  Term Cache      │
│                  │     │  (Redis)         │
└────────┬─────────┘     └──────────────────┘
         │
         ▼
┌──────────────────┐
│  FHIR Server     │
│  (Patient Data)  │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐     ┌──────────────────┐
│  Audit Service   │────▶│ Evidence Envelope│
└──────────────────┘     └──────────────────┘
```

## Artifact Verification

```
1. Pull artifact from registry
           │
           ▼
2. Verify Ed25519 signature
           │
    ┌──────┴──────┐
    │             │
   PASS         FAIL
    │             │
    ▼             ▼
3. Load       Reject &
   artifact   Alert

4. Cache in Redis
           │
           ▼
5. Ready for execution
```

## Evidence Envelope Structure

```json
{
  "envelope_id": "uuid",
  "timestamp": "2024-01-15T10:30:00Z",
  "inputs": {
    "patient_id": "hashed",
    "fhir_resources": ["Condition/1", "Observation/2"]
  },
  "logic": {
    "library": "DiabetesScreening",
    "version": "1.2.0",
    "elm_hash": "sha256:abc123"
  },
  "outputs": {
    "recommendations": [...],
    "alerts": [...]
  },
  "metadata": {
    "executor_version": "2.1.0",
    "terminology_version": "2024.01",
    "region": "AU"
  },
  "signature": "base64-ed25519-signature"
}
```

## Regional Configuration

### India (IN)
- Data residency: Mumbai, Hyderabad
- Compliance: DISHA, IT Act
- Terminology: NLEM, ICD-10-WHO
- KMS: AWS KMS ap-south-1

### Australia (AU)
- Data residency: Sydney
- Compliance: Privacy Act, My Health Record
- Terminology: AMT, PBS, ICD-10-AM
- KMS: AWS KMS ap-southeast-2

## Scaling Strategy

```
┌─────────────────────────────────────────────────────────────┐
│                    HORIZONTAL SCALING                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  CQL Executor Pods: Auto-scale based on CPU/Memory         │
│  ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐                        │
│  │ E1 │ │ E2 │ │ E3 │ │ E4 │ │ E5 │  ← HPA: 2-10 replicas │
│  └────┘ └────┘ └────┘ └────┘ └────┘                        │
│                                                              │
│  Redis Cluster: Read replicas for value sets                │
│  ┌────────┐   ┌────────┐   ┌────────┐                       │
│  │ Master │───│ Replica│───│ Replica│                       │
│  └────────┘   └────────┘   └────────┘                       │
│                                                              │
│  FHIR Server: Stateful set with persistent storage          │
│  ┌────────┐   ┌────────┐   ┌────────┐                       │
│  │ FHIR-0 │   │ FHIR-1 │   │ FHIR-2 │                       │
│  └────────┘   └────────┘   └────────┘                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Security Layers

1. **Network**: VPC isolation, security groups, WAF
2. **Transport**: TLS 1.3, mTLS for service-to-service
3. **Authentication**: OAuth 2.0, SMART on FHIR
4. **Authorization**: RBAC, attribute-based access
5. **Data**: Encryption at rest (AES-256), in transit (TLS)
6. **Audit**: Immutable logs, tamper detection
