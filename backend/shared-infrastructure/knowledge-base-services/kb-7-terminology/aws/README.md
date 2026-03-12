# KB-7 Knowledge Factory - AWS Infrastructure

Serverless pipeline for automated SNOMED CT, RxNorm, and LOINC terminology downloads and transformation.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                  KB-7 Knowledge Factory                      │
│               Serverless Terminology Pipeline                 │
└─────────────────────────────────────────────────────────────┘

External APIs (NHS TRUD, UMLS, LOINC.org)
         ↓
  ┌──────────────────────────────────────────┐
  │   AWS Lambda Functions (Python 3.11)     │
  │   - SNOMED downloader (15min, 10GB RAM)  │
  │   - RxNorm downloader (10min, 3GB RAM)   │
  │   - LOINC downloader (5min, 2GB RAM)     │
  └──────────────────────────────────────────┘
         ↓ (streaming upload)
  ┌──────────────────────────────────────────┐
  │   S3 Buckets                             │
  │   - cardiofit-kb-sources (raw files)     │
  │   - cardiofit-kb-artifacts (kernels)     │
  └──────────────────────────────────────────┘
         ↓
  ┌──────────────────────────────────────────┐
  │   Lambda: GitHub Dispatcher              │
  │   (triggers repository_dispatch event)   │
  └──────────────────────────────────────────┘
         ↓
  ┌──────────────────────────────────────────┐
  │   GitHub Actions (knowledge-factory repo)│
  │   - SNOMED-OWL-Toolkit transformation    │
  │   - ROBOT merge + reasoning              │
  │   - SPARQL validation (5 quality gates)  │
  │   - Upload kernel to S3 artifacts        │
  └──────────────────────────────────────────┘
         ↓
  GraphDB kernel deployment (manual review)

Orchestration:
- AWS Step Functions (parallel downloads)
- CloudWatch Events (monthly cron: 1st at 2 AM UTC)
- CloudWatch Alarms (duration, failures)

Security:
- AWS Secrets Manager (NHS API key, UMLS key, LOINC creds, GitHub PAT)
- IAM roles with least-privilege policies
- S3 server-side encryption (AES256)
- No secrets in GitHub Actions payload
```

## Directory Structure

```
aws/
├── cloudformation/              # Infrastructure as Code
│   ├── s3-buckets.yaml         # S3 storage (source + artifacts)
│   ├── lambda-functions.yaml    # 4 Lambda functions
│   ├── step-functions.yaml      # Orchestration workflow
│   └── secrets-manager.yaml     # API credentials
│
├── lambda/                      # Lambda function code
│   ├── snomed-downloader/
│   │   ├── handler.py          # Streaming S3 upload (1.2GB files)
│   │   └── requirements.txt
│   ├── rxnorm-downloader/
│   │   ├── handler.py          # UMLS API integration
│   │   └── requirements.txt
│   ├── loinc-downloader/
│   │   ├── handler.py          # LOINC.org authenticated download
│   │   └── requirements.txt
│   └── github-dispatcher/
│       ├── handler.py          # repository_dispatch trigger
│       └── requirements.txt
│
├── scripts/                     # Deployment automation
│   ├── setup-infrastructure.sh  # Deploy all stacks
│   ├── teardown-infrastructure.sh # Delete all stacks
│   └── test-lambda-functions.sh  # Individual Lambda tests
│
└── README.md                    # This file
```

## Prerequisites

1. **AWS CLI v2** installed and configured
   ```bash
   aws --version  # Should be 2.x
   aws configure  # Set up credentials
   ```

2. **IAM Permissions** required:
   - CloudFormation: CreateStack, UpdateStack, DeleteStack
   - Lambda: CreateFunction, InvokeFunction, UpdateFunctionCode
   - S3: CreateBucket, PutObject, GetObject
   - Secrets Manager: CreateSecret, GetSecretValue
   - Step Functions: CreateStateMachine, StartExecution
   - IAM: CreateRole, AttachRolePolicy
   - CloudWatch: PutMetricAlarm, PutLogEvents
   - SSM Parameter Store: PutParameter, GetParameter

3. **External API Credentials** (obtain before deployment):
   - NHS TRUD API key: https://isd.digital.nhs.uk/trud/user/guest/group/0/home
   - UMLS API key: https://uts.nlm.nih.gov/uts/signup-login
   - LOINC credentials: https://loinc.org/downloads/
   - GitHub Personal Access Token: https://github.com/settings/tokens (scopes: `repo`, `workflow`)

4. **Tools** (for scripts):
   - bash 4.0+
   - jq (JSON processor)

## Deployment Steps

### 1. Deploy Infrastructure

Run the automated setup script:

```bash
cd aws/scripts
./setup-infrastructure.sh production
```

This deploys 4 CloudFormation stacks:
- `kb7-knowledge-factory-s3-production`
- `kb7-knowledge-factory-secrets-production`
- `kb7-knowledge-factory-lambda-production`
- `kb7-knowledge-factory-stepfunctions-production`

Deployment time: ~10 minutes

### 2. Update Secrets

After deployment, update Secrets Manager with actual credentials:

```bash
# NHS TRUD API key (SNOMED)
aws secretsmanager update-secret \
  --secret-id kb7/production/nhs-trud-api-key \
  --secret-string '{"api_key":"YOUR_NHS_TRUD_KEY","api_endpoint":"https://isd.digital.nhs.uk/trud/api/v1","product_id":"101","rotation_days":90}'

# UMLS API key (RxNorm)
aws secretsmanager update-secret \
  --secret-id kb7/production/umls-api-key \
  --secret-string '{"api_key":"YOUR_UMLS_KEY","api_endpoint":"https://uts-ws.nlm.nih.gov/rest","rxnorm_endpoint":"https://rxnav.nlm.nih.gov/REST/rxnorm","rotation_days":365}'

# LOINC credentials
aws secretsmanager update-secret \
  --secret-id kb7/production/loinc-credentials \
  --secret-string '{"username":"YOUR_LOINC_USERNAME","password":"YOUR_LOINC_PASSWORD","api_endpoint":"https://loinc.org/downloads","rotation_days":180}'

# GitHub Personal Access Token
aws secretsmanager update-secret \
  --secret-id kb7/production/github-pat \
  --secret-string '{"token":"YOUR_GITHUB_PAT","repo_owner":"cardiofit","repo_name":"knowledge-factory","api_endpoint":"https://api.github.com","scopes":["repo","workflow"],"rotation_days":90}'
```

### 3. Deploy Lambda Code

Package and deploy Lambda function code:

```bash
# SNOMED downloader
cd ../lambda/snomed-downloader
pip install -r requirements.txt -t package/
cd package && zip -r ../snomed-function.zip . && cd ..
zip -g snomed-function.zip handler.py

aws lambda update-function-code \
  --function-name kb7-snomed-downloader-production \
  --zip-file fileb://snomed-function.zip

# Repeat for RxNorm, LOINC, and GitHub dispatcher
# (or create automated deploy script)
```

### 4. Test Lambda Functions

Test each function individually:

```bash
cd ../../scripts
./test-lambda-functions.sh production
```

Expected output: All functions return HTTP 200 status codes.

### 5. Test Step Functions Workflow

Trigger the complete workflow manually:

```bash
# Get state machine ARN
STATE_MACHINE_ARN=$(aws cloudformation describe-stacks \
  --stack-name kb7-knowledge-factory-stepfunctions-production \
  --query 'Stacks[0].Outputs[?OutputKey==`StateMachineArn`].OutputValue' \
  --output text)

# Start execution
aws stepfunctions start-execution \
  --state-machine-arn "$STATE_MACHINE_ARN" \
  --input '{
    "trigger": "manual-test",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
  }'
```

Monitor execution:
```bash
# Get execution ARN from output above, then:
aws stepfunctions describe-execution \
  --execution-arn <execution-arn>
```

### 6. Verify Monthly Cron

Check CloudWatch Events rule:

```bash
aws events list-rules --name-prefix kb7-monthly-terminology-update
```

Expected schedule: `cron(0 2 1 * ? *)` (1st of month, 2 AM UTC)

## Monitoring and Alerting

### CloudWatch Metrics

Custom metrics published by Lambda functions:

- `KB7/KnowledgeFactory/SNOMEDDownloadSize` (MB)
- `KB7/KnowledgeFactory/SNOMEDDownloadDuration` (seconds)
- `KB7/KnowledgeFactory/SNOMEDDownloadSuccess` (count)
- `KB7/KnowledgeFactory/SNOMEDDownloadFailures` (count)
- Similar metrics for RxNorm, LOINC, and GitHub dispatcher

### CloudWatch Alarms

Deployed alarms:

1. **SNOMED Duration High**: Warns if download >10 minutes (before 15min timeout)
2. **State Machine Failed**: Triggers on workflow execution failures
3. **State Machine Long Duration**: Warns if execution >30 minutes

### CloudWatch Logs

Lambda log groups:
```bash
aws logs tail /aws/lambda/kb7-snomed-downloader-production --follow
aws logs tail /aws/lambda/kb7-rxnorm-downloader-production --follow
aws logs tail /aws/lambda/kb7-loinc-downloader-production --follow
aws logs tail /aws/lambda/kb7-github-dispatcher-production --follow
```

Step Functions log group:
```bash
aws logs tail /aws/stepfunctions/kb7-knowledge-factory-production --follow
```

## Cost Estimation

### Monthly Operational Costs (Production)

| Component | Monthly Cost | Details |
|-----------|-------------|---------|
| **S3 Storage** | $5.00 | 200GB @ $0.023/GB (sources + kernels) |
| **Lambda Invocations** | $1.50 | 4 functions × 1 run/month × $0.20 each |
| **Lambda Duration** | $0.50 | ~30 mins total @ $0.0000166667/GB-second |
| **Step Functions** | $0.50 | 1 execution/month (parallel downloads) |
| **Secrets Manager** | $1.60 | 4 secrets @ $0.40/secret/month |
| **CloudWatch Logs** | $0.50 | Log retention and metrics |
| **Data Transfer** | $1.00 | S3 → Lambda → GitHub Actions |
| **Total AWS** | **$10.60/month** | **$127/year** |

### One-Time Setup Costs
- Infrastructure deployment: $0 (CloudFormation is free)
- Initial data downloads: ~$2 (data transfer from external APIs)

### GitHub Actions Costs
- Standard runners (free tier): $0/month (if under limits)
- Larger runners (16GB RAM for ROBOT reasoning): ~$12/month @ $0.16/min
- **Total GitHub**: ~$12/month (if OOM occurs, otherwise free)

### **Total Monthly Cost**: $10.60 - $22.60/month ($127 - $271/year)

## Key Features and Mitigations

### SNOMED Lambda: Streaming Upload
**Problem**: 1.2GB SNOMED files exceed Lambda memory limits
**Solution**: Chunked streaming upload with `boto3.s3.upload_fileobj()`
- 10MB chunks to avoid OOM
- Multipart S3 upload for large files
- SHA256 calculated during transfer (no double read)

### Timeout Fallback: ECS Fargate
**Problem**: Lambda 15-minute max timeout
**Solution**: Ready for migration to ECS Fargate (no timeout limit)
```yaml
# Future: docker/Dockerfile.snomed-downloader
FROM python:3.11-slim
# ... ECS task definition for longer-running downloads
```

### Security: Separation of Concerns
**AWS Lambda Side**:
- Uses AWS Secrets Manager for API credentials
- Downloads terminology files from external APIs
- Uploads to S3 with encryption

**GitHub Actions Side**:
- Uses GitHub Secrets (separate store)
- Reads from S3 (no credential sharing)
- ROBOT transformation and validation

**No Secrets Leakage**: GitHub dispatcher does NOT pass secrets in payload

## Troubleshooting

### Lambda Function Errors

**Error**: `AccessDenied` when writing to S3
**Fix**: Check IAM role has `s3:PutObject` permission

**Error**: `ResourceNotFoundException` for secrets
**Fix**: Verify secrets were created and names match environment variables

**Error**: `Timeout` on SNOMED downloader
**Fix**: Check NHS TRUD API responsiveness, consider ECS migration

### Step Functions Failures

**Error**: Parallel download branch fails
**Fix**: Check individual Lambda CloudWatch logs for root cause

**Error**: GitHub dispatch fails with 404
**Fix**: Verify GitHub repository exists and PAT has `repo` scope

### Secrets Rotation

Secrets are configured with rotation periods:
- NHS TRUD API key: 90 days
- UMLS API key: 365 days
- LOINC credentials: 180 days
- GitHub PAT: 90 days

Manual rotation:
```bash
aws secretsmanager update-secret \
  --secret-id kb7/production/nhs-trud-api-key \
  --secret-string '{"api_key":"NEW_KEY",...}'
```

## Teardown

To completely remove all infrastructure:

```bash
cd scripts
./teardown-infrastructure.sh production
```

**Warning**: This deletes:
- All S3 buckets (and their contents)
- All Lambda functions
- Step Functions state machine
- CloudWatch alarms and logs
- Secrets Manager secrets (30-day recovery period)

## Next Steps

After successful deployment:

1. **Verify Monthly Execution**: Wait for 1st of next month, verify cron triggers workflow
2. **GitHub Actions Setup**: Configure `cardiofit/knowledge-factory` repository with:
   - GitHub Secrets for S3 access (IAM user with read permissions)
   - Workflow file: `.github/workflows/kb-factory.yml`
   - ROBOT and SNOMED-OWL-Toolkit Docker containers
3. **GraphDB Integration**: Setup kernel deployment script in KB-7 service
4. **Monitoring Dashboard**: Create Grafana dashboard for metrics
5. **Alerting**: Configure SNS topics for CloudWatch alarms

## Support

For issues or questions:
- CloudFormation errors: Check AWS CloudFormation console → Stack events
- Lambda errors: Check CloudWatch Logs
- Architecture questions: Refer to `KB7_ARCHITECTURE_TRANSFORMATION_PLAN.md`
- Contact: kb7-architecture@cardiofit.ai
