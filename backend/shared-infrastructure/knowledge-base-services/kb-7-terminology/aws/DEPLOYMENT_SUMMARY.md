# KB-7 Knowledge Factory - AWS Infrastructure Deployment Summary

**Phase**: 1.3.2 - AWS Infrastructure Setup
**Date**: November 24, 2025
**Status**: Implementation Complete

## Files Created

### CloudFormation Templates (4)
```
aws/cloudformation/
├── s3-buckets.yaml           # S3 storage infrastructure (2 buckets)
├── lambda-functions.yaml      # 4 Lambda functions + IAM roles
├── step-functions.yaml        # Orchestration workflow + CloudWatch Events
└── secrets-manager.yaml       # 4 API credential secrets
```

### Lambda Function Code (4)
```
aws/lambda/
├── snomed-downloader/
│   ├── handler.py            # 200 lines - streaming S3 upload
│   └── requirements.txt
├── rxnorm-downloader/
│   ├── handler.py            # 150 lines - UMLS API integration
│   └── requirements.txt
├── loinc-downloader/
│   ├── handler.py            # 160 lines - authenticated download
│   └── requirements.txt
└── github-dispatcher/
    ├── handler.py            # 120 lines - repository_dispatch
    └── requirements.txt
```

### Deployment Scripts (3)
```
aws/scripts/
├── setup-infrastructure.sh    # 180 lines - deploy all stacks
├── teardown-infrastructure.sh # 140 lines - delete all resources
└── test-lambda-functions.sh   # 200 lines - individual function tests
```

### Documentation (2)
```
aws/
├── README.md                  # 500 lines - comprehensive guide
└── DEPLOYMENT_SUMMARY.md      # This file
```

## Key Security Considerations

### 1. IAM Least-Privilege Policies
- **Lambda Execution Role**: Only S3 upload, Secrets read, CloudWatch write
- **Step Functions Role**: Only Lambda invoke permissions
- **EventBridge Role**: Only Step Functions start-execution
- **No wildcard permissions**: All resources explicitly scoped

### 2. Secrets Management
- **AWS Secrets Manager**: All API credentials encrypted at rest
- **Rotation Policies**:
  - NHS TRUD: 90 days
  - UMLS: 365 days
  - LOINC: 180 days
  - GitHub PAT: 90 days
- **No Secrets in Code**: Environment variables reference secret ARNs only
- **Separation**: AWS Secrets for Lambda, GitHub Secrets for Actions (no sharing)

### 3. S3 Bucket Security
- **Server-Side Encryption**: AES256 on all objects
- **Versioning**: Enabled for rollback capability
- **Public Access**: Blocked at bucket level
- **Lifecycle Policies**:
  - Source files: 180-day retention
  - Artifacts: 365-day version retention
  - Transition to Glacier after 90 days

### 4. Network Security
- **VPC**: Lambda functions can optionally run in VPC (not configured by default)
- **API Endpoints**: HTTPS-only for all external API calls
- **GitHub Dispatch**: Bearer token authentication (PAT)

### 5. Logging and Auditing
- **CloudWatch Logs**: All Lambda invocations logged (30-day retention)
- **CloudTrail**: All API calls auditable (if enabled)
- **Step Functions Logs**: Full execution history with input/output

## Deployment Steps

### Quick Start
```bash
cd aws/scripts
./setup-infrastructure.sh production
```

### Full Deployment (Step-by-Step)

#### 1. Deploy Infrastructure (10 minutes)
```bash
./setup-infrastructure.sh production
```

**Creates**:
- 2 S3 buckets with lifecycle policies
- 4 Secrets Manager secrets (placeholder values)
- 4 Lambda functions (placeholder code)
- 1 Step Functions state machine
- 1 CloudWatch Events rule (monthly cron)
- 3 CloudWatch alarms (duration, failures)

#### 2. Update Secrets (5 minutes)
```bash
# NHS TRUD API key
aws secretsmanager update-secret \
  --secret-id kb7/production/nhs-trud-api-key \
  --secret-string '{"api_key":"YOUR_KEY",...}'

# Repeat for UMLS, LOINC, GitHub
```

#### 3. Deploy Lambda Code (15 minutes)
```bash
# For each Lambda function:
cd aws/lambda/snomed-downloader
pip install -r requirements.txt -t package/
cd package && zip -r ../function.zip . && cd ..
zip -g function.zip handler.py

aws lambda update-function-code \
  --function-name kb7-snomed-downloader-production \
  --zip-file fileb://function.zip
```

#### 4. Test Functions (5 minutes)
```bash
./test-lambda-functions.sh production
```

#### 5. Test Workflow (20 minutes)
```bash
# Trigger Step Functions manually
aws stepfunctions start-execution \
  --state-machine-arn <arn> \
  --input '{"trigger":"manual-test"}'
```

**Total Deployment Time**: ~55 minutes

## Estimated AWS Monthly Cost

### Production Environment

| Component | Configuration | Monthly Cost |
|-----------|--------------|--------------|
| **S3 Storage** | 200GB (sources + kernels) | $5.00 |
| **Lambda SNOMED** | 15min timeout, 10GB RAM, 1 run/month | $0.50 |
| **Lambda RxNorm** | 10min timeout, 3GB RAM, 1 run/month | $0.30 |
| **Lambda LOINC** | 5min timeout, 2GB RAM, 1 run/month | $0.20 |
| **Lambda GitHub** | 5min timeout, 1GB RAM, 1 run/month | $0.10 |
| **Step Functions** | 1 execution/month (parallel workflow) | $0.50 |
| **Secrets Manager** | 4 secrets @ $0.40 each | $1.60 |
| **CloudWatch Logs** | 30-day retention, 4 functions | $0.50 |
| **CloudWatch Alarms** | 3 alarms @ $0.10 each | $0.30 |
| **Data Transfer** | Downloads + S3 → Lambda | $1.00 |
| **Total** | | **$10.00/month** |

### Annual Cost: **$120/year**

### Cost Optimizations
- Use S3 Intelligent-Tiering for automatic cost optimization
- Enable S3 Glacier transition after 90 days (included in estimate)
- Lambda concurrency limits prevent runaway costs
- CloudWatch log retention limited to 30 days

### Additional Costs (GitHub)
- GitHub Actions Standard Runners: Free (if under limits)
- GitHub Actions Larger Runners (16GB): ~$12/month (if needed for ROBOT reasoning)

**Total Platform Cost**: $10-22/month ($120-264/year)

## Testing Instructions

### 1. Lambda Function Tests

#### Test SNOMED Downloader
```bash
aws lambda invoke \
  --function-name kb7-snomed-downloader-production \
  --payload '{"release_date":"20250131","edition":"international"}' \
  response.json

cat response.json | jq
```

**Expected Output**:
```json
{
  "statusCode": 200,
  "body": {
    "message": "SNOMED CT download successful",
    "s3_key": "snomed-ct/20250131/SnomedCT_international_20250131.zip",
    "file_size": 1258291200,
    "checksum": "abc123..."
  }
}
```

#### Test RxNorm Downloader
```bash
aws lambda invoke \
  --function-name kb7-rxnorm-downloader-production \
  --payload '{"version":"01042025","subset":"full"}' \
  response.json
```

#### Test LOINC Downloader
```bash
aws lambda invoke \
  --function-name kb7-loinc-downloader-production \
  --payload '{"version":"2.77","format":"csv"}' \
  response.json
```

#### Test GitHub Dispatcher
```bash
aws lambda invoke \
  --function-name kb7-github-dispatcher-production \
  --payload '{"downloads":[...]}' \
  response.json
```

### 2. Step Functions Workflow Test

#### Start Execution
```bash
STATE_MACHINE_ARN=$(aws cloudformation describe-stacks \
  --stack-name kb7-knowledge-factory-stepfunctions-production \
  --query 'Stacks[0].Outputs[?OutputKey==`StateMachineArn`].OutputValue' \
  --output text)

EXECUTION_ARN=$(aws stepfunctions start-execution \
  --state-machine-arn "$STATE_MACHINE_ARN" \
  --input '{"trigger":"manual-test","timestamp":"2025-11-24T10:00:00Z"}' \
  --query 'executionArn' \
  --output text)

echo "Execution started: $EXECUTION_ARN"
```

#### Monitor Execution
```bash
# Check status
aws stepfunctions describe-execution \
  --execution-arn "$EXECUTION_ARN" \
  --query 'status' \
  --output text

# Get execution history
aws stepfunctions get-execution-history \
  --execution-arn "$EXECUTION_ARN" \
  --max-results 100
```

#### View Logs
```bash
# Step Functions logs
aws logs tail /aws/stepfunctions/kb7-knowledge-factory-production --follow

# Individual Lambda logs
aws logs tail /aws/lambda/kb7-snomed-downloader-production --follow
```

### 3. S3 Upload Verification

```bash
# List downloaded files
aws s3 ls s3://cardiofit-kb-sources-production/snomed-ct/ --recursive
aws s3 ls s3://cardiofit-kb-sources-production/rxnorm/ --recursive
aws s3 ls s3://cardiofit-kb-sources-production/loinc/ --recursive

# Check file metadata
aws s3api head-object \
  --bucket cardiofit-kb-sources-production \
  --key snomed-ct/20250131/SnomedCT_international_20250131.zip
```

### 4. CloudWatch Metrics Verification

```bash
# SNOMED download metrics
aws cloudwatch get-metric-statistics \
  --namespace KB7/KnowledgeFactory \
  --metric-name SNOMEDDownloadSuccess \
  --start-time 2025-11-24T00:00:00Z \
  --end-time 2025-11-24T23:59:59Z \
  --period 3600 \
  --statistics Sum

# Execution duration
aws cloudwatch get-metric-statistics \
  --namespace AWS/Lambda \
  --metric-name Duration \
  --dimensions Name=FunctionName,Value=kb7-snomed-downloader-production \
  --start-time 2025-11-24T00:00:00Z \
  --end-time 2025-11-24T23:59:59Z \
  --period 3600 \
  --statistics Maximum
```

### 5. Monthly Cron Verification

```bash
# Check EventBridge rule
aws events list-rules --name-prefix kb7-monthly-terminology-update

# Verify schedule expression
aws events describe-rule \
  --name kb7-monthly-terminology-update-production \
  --query 'ScheduleExpression' \
  --output text

# Expected: cron(0 2 1 * ? *)
```

## Important Mitigations Implemented

### 1. SNOMED Lambda: Streaming Upload
**Problem**: 1.2GB SNOMED files exceed Lambda ephemeral storage
**Solution**:
- `boto3.s3.upload_fileobj()` with chunked streaming
- 10MB chunks to minimize memory footprint
- Multipart S3 upload API for large files
- SHA256 hash calculated during transfer (no double read)

**Code**: `aws/lambda/snomed-downloader/handler.py:stream_download_to_s3()`

### 2. CloudWatch Alarm: Lambda Duration
**Problem**: Lambda timeout (15min) causes pipeline failure
**Solution**:
- Alarm triggers at 10 minutes (before timeout)
- Provides early warning for intervention
- Fallback: ECS Fargate documented in comments

**Alarm**: `kb7-knowledge-factory-lambda-production` stack output

### 3. Secrets Separation
**Problem**: Credential leakage between AWS and GitHub
**Solution**:
- AWS Secrets Manager: Lambda downloads only
- GitHub Secrets: GitHub Actions transformation only
- No secrets passed in repository_dispatch payload
- IAM roles scoped to specific secret ARNs

### 4. S3 Lifecycle Policies
**Problem**: Storage costs grow indefinitely
**Solution**:
- Source files: 180-day expiration
- Artifacts: 365-day version retention
- Transition to Glacier after 90 days
- Non-current versions expire after 30 days

### 5. Step Functions Retry Logic
**Problem**: Transient API failures cause workflow failure
**Solution**:
- 2 retries with exponential backoff (1.5x)
- Separate error handling per download branch
- GitHub dispatcher: 3 retries (API often fails first attempt)

## Next Phase: 1.3.3 Knowledge Factory Pipeline

After AWS infrastructure is deployed:

1. **GitHub Repository**: Create `cardiofit/knowledge-factory`
2. **GitHub Actions Workflow**: `.github/workflows/kb-factory.yml`
3. **ROBOT Integration**: Docker container with SNOMED-OWL-Toolkit
4. **SPARQL Validation**: 5 quality gates for kernel verification
5. **S3 Upload**: Transformed kernel uploaded to artifacts bucket

**Timeline**: Week 3 (Phase 1.3.3)

## Support and Troubleshooting

### Common Issues

#### Issue: `AccessDenied` on S3 upload
**Fix**: Check IAM role has `s3:PutObject` permission for target bucket

#### Issue: `ResourceNotFoundException` for secrets
**Fix**: Verify secret names match environment variables in Lambda config

#### Issue: Lambda timeout on SNOMED download
**Fix**: Check NHS TRUD API response time, consider ECS migration

#### Issue: GitHub dispatch returns 404
**Fix**: Verify repository exists and PAT has `repo` scope

### Debug Commands
```bash
# Check stack status
aws cloudformation describe-stacks --stack-name kb7-knowledge-factory-lambda-production

# View Lambda configuration
aws lambda get-function --function-name kb7-snomed-downloader-production

# Check IAM role permissions
aws iam get-role --role-name KB7-Lambda-Execution-Role-production

# View recent invocations
aws lambda list-invocations --function-name kb7-snomed-downloader-production
```

### Contact
- Architecture: kb7-architecture@cardiofit.ai
- AWS Issues: Check CloudFormation stack events
- Lambda Issues: Check CloudWatch Logs
