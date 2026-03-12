#!/bin/bash
#
# KB-7 Knowledge Factory - Infrastructure Teardown Script
# Deletes all CloudFormation stacks and associated resources
#
# Usage:
#   ./teardown-infrastructure.sh [environment]
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

# Confirm deletion
log_warn "========================================="
log_warn "WARNING: This will DELETE all KB-7 Knowledge Factory infrastructure"
log_warn "Environment: $ENVIRONMENT"
log_warn "Region: $AWS_REGION"
log_warn "========================================="
echo ""
read -p "Are you sure you want to continue? (yes/no): " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    log_info "Teardown cancelled"
    exit 0
fi

# Step 1: Delete Step Functions
log_info "Step 1/5: Deleting Step Functions..."
aws cloudformation delete-stack \
    --stack-name "${STACK_PREFIX}-stepfunctions-${ENVIRONMENT}" \
    --region "$AWS_REGION"

aws cloudformation wait stack-delete-complete \
    --stack-name "${STACK_PREFIX}-stepfunctions-${ENVIRONMENT}" \
    --region "$AWS_REGION" 2>/dev/null || log_warn "Stack may not exist"

log_info "Step Functions deleted"
echo ""

# Step 2: Delete Lambda Functions
log_info "Step 2/5: Deleting Lambda functions..."
aws cloudformation delete-stack \
    --stack-name "${STACK_PREFIX}-lambda-${ENVIRONMENT}" \
    --region "$AWS_REGION"

aws cloudformation wait stack-delete-complete \
    --stack-name "${STACK_PREFIX}-lambda-${ENVIRONMENT}" \
    --region "$AWS_REGION" 2>/dev/null || log_warn "Stack may not exist"

# Clean up Parameter Store entries
log_info "Cleaning up Parameter Store..."
aws ssm delete-parameter --name "/kb7/$ENVIRONMENT/lambda/snomed-arn" --region "$AWS_REGION" 2>/dev/null || true
aws ssm delete-parameter --name "/kb7/$ENVIRONMENT/lambda/rxnorm-arn" --region "$AWS_REGION" 2>/dev/null || true
aws ssm delete-parameter --name "/kb7/$ENVIRONMENT/lambda/loinc-arn" --region "$AWS_REGION" 2>/dev/null || true
aws ssm delete-parameter --name "/kb7/$ENVIRONMENT/lambda/github-arn" --region "$AWS_REGION" 2>/dev/null || true

log_info "Lambda functions deleted"
echo ""

# Step 3: Delete Secrets Manager
log_info "Step 3/5: Deleting Secrets Manager..."
log_warn "Secrets will be scheduled for deletion (30-day recovery period)"

aws cloudformation delete-stack \
    --stack-name "${STACK_PREFIX}-secrets-${ENVIRONMENT}" \
    --region "$AWS_REGION"

aws cloudformation wait stack-delete-complete \
    --stack-name "${STACK_PREFIX}-secrets-${ENVIRONMENT}" \
    --region "$AWS_REGION" 2>/dev/null || log_warn "Stack may not exist"

log_info "Secrets Manager deleted"
echo ""

# Step 4: Empty S3 Buckets (required before deletion)
log_info "Step 4/5: Emptying S3 buckets..."

SOURCE_BUCKET="cardiofit-kb-sources-${ENVIRONMENT}"
ARTIFACT_BUCKET="cardiofit-kb-artifacts-${ENVIRONMENT}"

# Empty source bucket
log_info "Emptying source bucket: $SOURCE_BUCKET"
aws s3 rm "s3://${SOURCE_BUCKET}" --recursive --region "$AWS_REGION" 2>/dev/null || log_warn "Bucket may not exist or already empty"

# Empty artifact bucket
log_info "Emptying artifact bucket: $ARTIFACT_BUCKET"
aws s3 rm "s3://${ARTIFACT_BUCKET}" --recursive --region "$AWS_REGION" 2>/dev/null || log_warn "Bucket may not exist or already empty"

# Delete all versions (if versioning enabled)
log_info "Deleting bucket versions..."
aws s3api list-object-versions --bucket "$SOURCE_BUCKET" --region "$AWS_REGION" 2>/dev/null | \
    jq -r '.Versions[]? | .Key + " " + .VersionId' | \
    while read key version; do
        aws s3api delete-object --bucket "$SOURCE_BUCKET" --key "$key" --version-id "$version" --region "$AWS_REGION" 2>/dev/null || true
    done

aws s3api list-object-versions --bucket "$ARTIFACT_BUCKET" --region "$AWS_REGION" 2>/dev/null | \
    jq -r '.Versions[]? | .Key + " " + .VersionId' | \
    while read key version; do
        aws s3api delete-object --bucket "$ARTIFACT_BUCKET" --key "$key" --version-id "$version" --region "$AWS_REGION" 2>/dev/null || true
    done

log_info "S3 buckets emptied"
echo ""

# Step 5: Delete S3 Buckets
log_info "Step 5/5: Deleting S3 buckets..."
aws cloudformation delete-stack \
    --stack-name "${STACK_PREFIX}-s3-${ENVIRONMENT}" \
    --region "$AWS_REGION"

aws cloudformation wait stack-delete-complete \
    --stack-name "${STACK_PREFIX}-s3-${ENVIRONMENT}" \
    --region "$AWS_REGION" 2>/dev/null || log_warn "Stack may not exist"

log_info "S3 buckets deleted"
echo ""

# Summary
log_info "========================================="
log_info "KB-7 Knowledge Factory teardown complete!"
log_info "========================================="
echo ""
log_info "Deleted stacks:"
log_info "  - ${STACK_PREFIX}-s3-${ENVIRONMENT}"
log_info "  - ${STACK_PREFIX}-secrets-${ENVIRONMENT}"
log_info "  - ${STACK_PREFIX}-lambda-${ENVIRONMENT}"
log_info "  - ${STACK_PREFIX}-stepfunctions-${ENVIRONMENT}"
echo ""
log_warn "Note: Secrets are scheduled for deletion (30-day recovery period)"
log_warn "To immediately delete: aws secretsmanager delete-secret --secret-id <secret-name> --force-delete-without-recovery"
echo ""
log_info "All resources have been removed from AWS"
