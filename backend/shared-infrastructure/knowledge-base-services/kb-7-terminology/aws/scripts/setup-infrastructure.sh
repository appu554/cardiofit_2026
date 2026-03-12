#!/bin/bash
#
# KB-7 Knowledge Factory - Infrastructure Setup Script
# Deploys all CloudFormation stacks for serverless terminology pipeline
#
# Usage:
#   ./setup-infrastructure.sh [environment]
#   environment: production (default), staging, development
#

set -e  # Exit on error

# Configuration
ENVIRONMENT="${1:-production}"
AWS_REGION="${AWS_REGION:-us-east-1}"
STACK_PREFIX="kb7-knowledge-factory"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validate AWS CLI is installed
if ! command -v aws &> /dev/null; then
    log_error "AWS CLI not found. Please install: https://aws.amazon.com/cli/"
    exit 1
fi

# Validate AWS credentials
if ! aws sts get-caller-identity &> /dev/null; then
    log_error "AWS credentials not configured. Run 'aws configure' first."
    exit 1
fi

log_info "Starting KB-7 Knowledge Factory infrastructure deployment"
log_info "Environment: $ENVIRONMENT"
log_info "Region: $AWS_REGION"
echo ""

# Step 1: Deploy S3 Buckets
log_info "Step 1/4: Deploying S3 buckets..."
aws cloudformation deploy \
    --template-file ../cloudformation/s3-buckets.yaml \
    --stack-name "${STACK_PREFIX}-s3-${ENVIRONMENT}" \
    --parameter-overrides \
        EnvironmentName="$ENVIRONMENT" \
        SourceBucketName="cardiofit-kb-sources" \
        ArtifactBucketName="cardiofit-kb-artifacts" \
    --region "$AWS_REGION" \
    --tags \
        Project=CardioFit-KB7 \
        Environment="$ENVIRONMENT" \
        ManagedBy=CloudFormation

if [ $? -eq 0 ]; then
    log_info "S3 buckets deployed successfully"

    # Get bucket names from stack outputs
    SOURCE_BUCKET=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_PREFIX}-s3-${ENVIRONMENT}" \
        --query 'Stacks[0].Outputs[?OutputKey==`SourceBucketName`].OutputValue' \
        --output text \
        --region "$AWS_REGION")

    log_info "Source bucket: $SOURCE_BUCKET"
else
    log_error "S3 bucket deployment failed"
    exit 1
fi
echo ""

# Step 2: Deploy Secrets Manager
log_info "Step 2/4: Deploying Secrets Manager..."
log_warn "Secrets will be created with PLACEHOLDER values - update manually after deployment"

aws cloudformation deploy \
    --template-file ../cloudformation/secrets-manager.yaml \
    --stack-name "${STACK_PREFIX}-secrets-${ENVIRONMENT}" \
    --parameter-overrides \
        EnvironmentName="$ENVIRONMENT" \
    --capabilities CAPABILITY_NAMED_IAM \
    --region "$AWS_REGION" \
    --tags \
        Project=CardioFit-KB7 \
        Environment="$ENVIRONMENT" \
        ManagedBy=CloudFormation

if [ $? -eq 0 ]; then
    log_info "Secrets Manager deployed successfully"
else
    log_error "Secrets Manager deployment failed"
    exit 1
fi
echo ""

# Step 3: Deploy Lambda Functions
log_info "Step 3/4: Deploying Lambda functions..."
log_warn "Lambda function code is placeholder - deploy actual code separately"

aws cloudformation deploy \
    --template-file ../cloudformation/lambda-functions.yaml \
    --stack-name "${STACK_PREFIX}-lambda-${ENVIRONMENT}" \
    --parameter-overrides \
        EnvironmentName="$ENVIRONMENT" \
        SourceBucketName="$SOURCE_BUCKET" \
        SecretsStackName="${STACK_PREFIX}-secrets-${ENVIRONMENT}" \
    --capabilities CAPABILITY_NAMED_IAM \
    --region "$AWS_REGION" \
    --tags \
        Project=CardioFit-KB7 \
        Environment="$ENVIRONMENT" \
        ManagedBy=CloudFormation

if [ $? -eq 0 ]; then
    log_info "Lambda functions deployed successfully"

    # Store Lambda ARNs in SSM Parameter Store for Step Functions
    SNOMED_ARN=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_PREFIX}-lambda-${ENVIRONMENT}" \
        --query 'Stacks[0].Outputs[?OutputKey==`SNOMEDFunctionArn`].OutputValue' \
        --output text \
        --region "$AWS_REGION")

    RXNORM_ARN=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_PREFIX}-lambda-${ENVIRONMENT}" \
        --query 'Stacks[0].Outputs[?OutputKey==`RxNormFunctionArn`].OutputValue' \
        --output text \
        --region "$AWS_REGION")

    LOINC_ARN=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_PREFIX}-lambda-${ENVIRONMENT}" \
        --query 'Stacks[0].Outputs[?OutputKey==`LOINCFunctionArn`].OutputValue' \
        --output text \
        --region "$AWS_REGION")

    GITHUB_ARN=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_PREFIX}-lambda-${ENVIRONMENT}" \
        --query 'Stacks[0].Outputs[?OutputKey==`GitHubFunctionArn`].OutputValue' \
        --output text \
        --region "$AWS_REGION")

    # Store in Parameter Store
    aws ssm put-parameter \
        --name "/kb7/$ENVIRONMENT/lambda/snomed-arn" \
        --value "$SNOMED_ARN" \
        --type String \
        --overwrite \
        --region "$AWS_REGION" > /dev/null

    aws ssm put-parameter \
        --name "/kb7/$ENVIRONMENT/lambda/rxnorm-arn" \
        --value "$RXNORM_ARN" \
        --type String \
        --overwrite \
        --region "$AWS_REGION" > /dev/null

    aws ssm put-parameter \
        --name "/kb7/$ENVIRONMENT/lambda/loinc-arn" \
        --value "$LOINC_ARN" \
        --type String \
        --overwrite \
        --region "$AWS_REGION" > /dev/null

    aws ssm put-parameter \
        --name "/kb7/$ENVIRONMENT/lambda/github-arn" \
        --value "$GITHUB_ARN" \
        --type String \
        --overwrite \
        --region "$AWS_REGION" > /dev/null

    log_info "Lambda ARNs stored in Parameter Store"
else
    log_error "Lambda function deployment failed"
    exit 1
fi
echo ""

# Step 4: Deploy Step Functions
log_info "Step 4/4: Deploying Step Functions orchestration..."

aws cloudformation deploy \
    --template-file ../cloudformation/step-functions.yaml \
    --stack-name "${STACK_PREFIX}-stepfunctions-${ENVIRONMENT}" \
    --parameter-overrides \
        EnvironmentName="$ENVIRONMENT" \
        LambdaStackName="${STACK_PREFIX}-lambda-${ENVIRONMENT}" \
    --capabilities CAPABILITY_NAMED_IAM \
    --region "$AWS_REGION" \
    --tags \
        Project=CardioFit-KB7 \
        Environment="$ENVIRONMENT" \
        ManagedBy=CloudFormation

if [ $? -eq 0 ]; then
    log_info "Step Functions deployed successfully"

    STATE_MACHINE_ARN=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_PREFIX}-stepfunctions-${ENVIRONMENT}" \
        --query 'Stacks[0].Outputs[?OutputKey==`StateMachineArn`].OutputValue' \
        --output text \
        --region "$AWS_REGION")

    log_info "State machine ARN: $STATE_MACHINE_ARN"
else
    log_error "Step Functions deployment failed"
    exit 1
fi
echo ""

# Summary
log_info "========================================="
log_info "KB-7 Knowledge Factory deployment complete!"
log_info "========================================="
echo ""
log_info "Deployed stacks:"
log_info "  - ${STACK_PREFIX}-s3-${ENVIRONMENT}"
log_info "  - ${STACK_PREFIX}-secrets-${ENVIRONMENT}"
log_info "  - ${STACK_PREFIX}-lambda-${ENVIRONMENT}"
log_info "  - ${STACK_PREFIX}-stepfunctions-${ENVIRONMENT}"
echo ""
log_warn "Next steps:"
log_warn "  1. Update Secrets Manager with actual API credentials:"
log_warn "     - NHS TRUD API key"
log_warn "     - UMLS API key"
log_warn "     - LOINC credentials"
log_warn "     - GitHub PAT"
log_warn "  2. Deploy Lambda function code (see deploy-lambda-code.sh)"
log_warn "  3. Test Lambda functions individually (see test-lambda-functions.sh)"
log_warn "  4. Trigger Step Functions workflow manually for validation"
log_warn "  5. Monthly cron is already configured (1st of month, 2 AM UTC)"
echo ""
log_info "Useful commands:"
log_info "  Update secret: aws secretsmanager update-secret --secret-id kb7/$ENVIRONMENT/nhs-trud-api-key --secret-string '{\"api_key\":\"YOUR_KEY\"}'"
log_info "  Test workflow: aws stepfunctions start-execution --state-machine-arn $STATE_MACHINE_ARN"
log_info "  View logs: aws logs tail /aws/lambda/kb7-snomed-downloader-$ENVIRONMENT --follow"
