# KB-1 Drug Dosing Rules - Postman Collection

Complete API collection for testing and exploring the KB-1 Drug Dosing Rules Service.

## Quick Start

1. **Import Collection**: Import `KB1-Drug-Rules-Collection.postman_collection.json` into Postman
2. **Import Environment**: Import `KB1-Local.postman_environment.json` for local development
3. **Select Environment**: Choose "KB-1 Local Development" from the environment dropdown
4. **Start Testing**: Run requests from any folder

## Collection Structure

```
KB-1 Drug Dosing Rules Service
├── Health & Readiness          # Service monitoring
├── Dose Calculation            # Clinical dosing endpoints
│   ├── Calculate Standard Dose
│   ├── Calculate Weight-Based Dose
│   ├── Calculate BSA-Based Dose
│   ├── Calculate Pediatric Dose
│   ├── Calculate Renal-Adjusted Dose
│   ├── Calculate Hepatic-Adjusted Dose
│   └── Calculate Geriatric Dose
├── Patient Parameters          # Clinical parameter utilities
│   ├── Calculate BSA
│   ├── Calculate IBW
│   ├── Calculate CrCl
│   └── Calculate eGFR
├── Dose Validation             # Safety validation
├── Drug Rules                  # Rule management
├── Dose Adjustments            # Organ-specific guidelines
├── High-Alert Medications      # ISMP checks
├── Admin - Approval Workflow   # Hospital governance ⭐ NEW
│   ├── Get Approval Statistics
│   ├── Get Pending Reviews
│   ├── Submit Pharmacist Review
│   ├── Approve Rule (CMO)
│   ├── Reject Rule
│   ├── Get Rule Audit History
│   └── Admin Get Rule (All Statuses)
└── Example Workflows           # End-to-end scenarios
    ├── Workflow: Renal Patient Dosing
    └── Workflow: Drug Rule Approval
```

## Approval Workflow

The approval workflow ensures clinical safety by requiring human review before drug rules are used for dosing:

```
DRAFT → REVIEWED → ACTIVE
          ↓
       RETIRED (rejected)
```

### Workflow Steps

1. **Check Pending Queue**
   ```
   GET /v1/admin/pending?risk_level=CRITICAL
   ```

2. **Pharmacist Review** (DRAFT → REVIEWED)
   ```
   POST /v1/admin/review/{rule_id}
   {
     "reviewed_by": "Dr. Name (Clinical Pharmacist)",
     "review_notes": "Verified against FDA label",
     "dosing_verified": true,
     "safety_verified": true
   }
   ```

3. **CMO Approval** (REVIEWED → ACTIVE)
   ```
   POST /v1/admin/approve/{rule_id}
   {
     "approved_by": "Dr. Name (CMO)",
     "review_notes": "Approved for clinical use",
     "skip_verification": true  // Required for CRITICAL/HIGH risk
   }
   ```

### Risk Levels

| Risk Level | Examples | Approval Required |
|------------|----------|-------------------|
| CRITICAL | Anticoagulants, Opioids, Insulin | CMO + Pharmacist |
| HIGH | Benzodiazepines, Antiarrhythmics | Pharmacist (CMO recommended) |
| STANDARD | Most medications | Pharmacist |
| LOW | OTC, Vitamins | Pharmacist |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `base_url` | KB-1 service URL | `http://localhost:8081` |
| `api_version` | API version | `v1` |
| `rule_id` | UUID for approval operations | (set manually) |

## Common RxNorm Codes for Testing

| Drug | RxNorm | Risk Level |
|------|--------|------------|
| Acetaminophen | 161 | STANDARD |
| Aspirin | 1191 | STANDARD |
| Ibuprofen | 4850 | STANDARD |
| Metformin | 6809 | STANDARD |
| Lisinopril | 29046 | HIGH |
| Lorazepam | 6470 | HIGH |
| Warfarin | 11289 | CRITICAL |
| Heparin | 9877 | CRITICAL |
| Digoxin | 3407 | CRITICAL |

## Running Tests

### Via Postman Runner
1. Select collection or folder
2. Click "Run"
3. Review results

### Via Newman (CLI)
```bash
newman run KB1-Drug-Rules-Collection.postman_collection.json \
  -e KB1-Local.postman_environment.json \
  --reporters cli,json
```

## Troubleshooting

### Service Not Responding
```bash
# Check if service is running
curl http://localhost:8081/health

# Start service if needed
cd ../
REDIS_PORT=6382 go run cmd/server/main.go
```

### Database Connection Failed
```bash
# Ensure Docker containers are running
docker ps | grep kb1

# Start infrastructure
docker-compose up -d kb1-postgres kb1-redis
```

### 503 Service Unavailable
- Server running in degraded mode (no database)
- Check PostgreSQL on port 5481
- Check Redis on port 6382
