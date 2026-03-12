# Terraform Configuration for Medication Service V2
# Healthcare-grade infrastructure with HIPAA compliance and high availability

terraform {
  required_version = ">= 1.5.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
  
  # Remote state backend with encryption
  backend "s3" {
    bucket         = "cardiofit-terraform-state-prod"
    key            = "medication-service-v2/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    kms_key_id     = "arn:aws:kms:us-east-1:123456789012:key/terraform-state-key"
    dynamodb_table = "cardiofit-terraform-locks"
  }
}

# ============================================================================
# PROVIDER CONFIGURATION
# ============================================================================

provider "aws" {
  region = var.aws_region
  
  default_tags {
    tags = {
      Project             = "CardioFit"
      Service             = "medication-service-v2"
      Environment         = var.environment
      ManagedBy          = "terraform"
      Compliance         = "HIPAA"
      SecurityLevel      = "high"
      Owner              = "clinical-platform-team"
      CostCenter         = "clinical-services"
      BackupRequired     = "true"
      MonitoringLevel    = "critical"
      DataClassification = "PHI"
    }
  }
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.cluster.token
}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.cluster.endpoint
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority[0].data)
    token                  = data.aws_eks_cluster_auth.cluster.token
  }
}

# ============================================================================
# DATA SOURCES
# ============================================================================

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

data "aws_eks_cluster" "cluster" {
  name = var.eks_cluster_name
}

data "aws_eks_cluster_auth" "cluster" {
  name = var.eks_cluster_name
}

data "aws_vpc" "main" {
  tags = {
    Name = "${var.project_name}-vpc-${var.environment}"
  }
}

data "aws_subnets" "private" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.main.id]
  }
  
  tags = {
    Type = "private"
  }
}

data "aws_subnets" "database" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.main.id]
  }
  
  tags = {
    Type = "database"
  }
}

# ============================================================================
# LOCALS
# ============================================================================

locals {
  name_prefix = "${var.project_name}-${var.service_name}-${var.environment}"
  
  common_tags = {
    Project             = var.project_name
    Service             = var.service_name
    Environment         = var.environment
    ManagedBy          = "terraform"
    Compliance         = "HIPAA"
    SecurityLevel      = "high"
    DataClassification = "PHI"
  }
  
  # Healthcare-specific configurations
  healthcare_config = {
    encryption_required     = true
    audit_logging_required = true
    backup_required        = true
    monitoring_level       = "critical"
    compliance_standards   = ["HIPAA", "SOC2", "ISO27001"]
  }
}

# ============================================================================
# KMS KEYS FOR ENCRYPTION
# ============================================================================

# Main encryption key for the service
resource "aws_kms_key" "main" {
  description             = "KMS key for ${local.name_prefix}"
  deletion_window_in_days = 7
  enable_key_rotation     = true
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow service access"
        Effect = "Allow"
        Principal = {
          AWS = aws_iam_role.service_role.arn
        }
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:DescribeKey"
        ]
        Resource = "*"
      }
    ]
  })
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-kms-key"
    Type = "encryption"
  })
}

resource "aws_kms_alias" "main" {
  name          = "alias/${local.name_prefix}"
  target_key_id = aws_kms_key.main.key_id
}

# Database encryption key
resource "aws_kms_key" "database" {
  description             = "KMS key for ${local.name_prefix} database"
  deletion_window_in_days = 7
  enable_key_rotation     = true
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-database-kms-key"
    Type = "database-encryption"
  })
}

resource "aws_kms_alias" "database" {
  name          = "alias/${local.name_prefix}-database"
  target_key_id = aws_kms_key.database.key_id
}

# ============================================================================
# IAM ROLES AND POLICIES
# ============================================================================

# Service Account IAM Role
resource "aws_iam_role" "service_role" {
  name = "${local.name_prefix}-service-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRoleWithWebIdentity"
        Effect = "Allow"
        Principal = {
          Federated = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider/${replace(data.aws_eks_cluster.cluster.identity[0].oidc[0].issuer, "https://", "")}"
        }
        Condition = {
          StringEquals = {
            "${replace(data.aws_eks_cluster.cluster.identity[0].oidc[0].issuer, "https://", "")}:sub" = "system:serviceaccount:${var.k8s_namespace}:${var.service_name}"
            "${replace(data.aws_eks_cluster.cluster.identity[0].oidc[0].issuer, "https://", "")}:aud" = "sts.amazonaws.com"
          }
        }
      }
    ]
  })
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-service-role"
    Type = "iam-role"
  })
}

# Service IAM Policy
resource "aws_iam_policy" "service_policy" {
  name        = "${local.name_prefix}-service-policy"
  description = "IAM policy for ${local.name_prefix}"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:DescribeKey"
        ]
        Resource = [
          aws_kms_key.main.arn,
          aws_kms_key.database.arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = [
          "${aws_s3_bucket.backups.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.backups.arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = [
          aws_secretsmanager_secret.db_password.arn,
          aws_secretsmanager_secret.redis_password.arn,
          aws_secretsmanager_secret.jwt_secret.arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "cloudwatch:PutMetricData",
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "service_policy_attachment" {
  role       = aws_iam_role.service_role.name
  policy_arn = aws_iam_policy.service_policy.arn
}

# ============================================================================
# SECRETS MANAGER
# ============================================================================

# Generate secure passwords
resource "random_password" "db_password" {
  length  = 32
  special = true
}

resource "random_password" "redis_password" {
  length  = 32
  special = true
}

resource "random_password" "jwt_secret" {
  length = 64
}

resource "random_password" "encryption_key" {
  length = 32
}

# Database password secret
resource "aws_secretsmanager_secret" "db_password" {
  name        = "${local.name_prefix}/database/password"
  description = "Database password for ${local.name_prefix}"
  kms_key_id  = aws_kms_key.main.arn
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-db-password"
    Type = "database-secret"
  })
}

resource "aws_secretsmanager_secret_version" "db_password" {
  secret_id     = aws_secretsmanager_secret.db_password.id
  secret_string = random_password.db_password.result
}

# Redis password secret
resource "aws_secretsmanager_secret" "redis_password" {
  name        = "${local.name_prefix}/redis/password"
  description = "Redis password for ${local.name_prefix}"
  kms_key_id  = aws_kms_key.main.arn
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-redis-password"
    Type = "cache-secret"
  })
}

resource "aws_secretsmanager_secret_version" "redis_password" {
  secret_id     = aws_secretsmanager_secret.redis_password.id
  secret_string = random_password.redis_password.result
}

# JWT secret
resource "aws_secretsmanager_secret" "jwt_secret" {
  name        = "${local.name_prefix}/auth/jwt-secret"
  description = "JWT secret for ${local.name_prefix}"
  kms_key_id  = aws_kms_key.main.arn
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-jwt-secret"
    Type = "auth-secret"
  })
}

resource "aws_secretsmanager_secret_version" "jwt_secret" {
  secret_id     = aws_secretsmanager_secret.jwt_secret.id
  secret_string = random_password.jwt_secret.result
}

# Encryption key secret
resource "aws_secretsmanager_secret" "encryption_key" {
  name        = "${local.name_prefix}/crypto/encryption-key"
  description = "Encryption key for ${local.name_prefix}"
  kms_key_id  = aws_kms_key.main.arn
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-encryption-key"
    Type = "crypto-secret"
  })
}

resource "aws_secretsmanager_secret_version" "encryption_key" {
  secret_id     = aws_secretsmanager_secret.encryption_key.id
  secret_string = random_password.encryption_key.result
}

# ============================================================================
# RDS DATABASE - PRODUCTION CLUSTER
# ============================================================================

# RDS Subnet Group
resource "aws_db_subnet_group" "main" {
  name       = "${local.name_prefix}-db-subnet-group"
  subnet_ids = data.aws_subnets.database.ids
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-db-subnet-group"
    Type = "database-networking"
  })
}

# Security Group for RDS
resource "aws_security_group" "rds" {
  name_prefix = "${local.name_prefix}-rds-"
  vpc_id      = data.aws_vpc.main.id
  description = "Security group for ${local.name_prefix} RDS cluster"
  
  # Allow PostgreSQL access from EKS nodes
  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_node.id]
    description     = "PostgreSQL access from EKS nodes"
  }
  
  # Allow monitoring access
  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.main.cidr_block]
    description = "Internal monitoring access"
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-rds-sg"
    Type = "database-security"
  })
}

# RDS Cluster Parameter Group
resource "aws_rds_cluster_parameter_group" "main" {
  family      = "aurora-postgresql15"
  name        = "${local.name_prefix}-cluster-params"
  description = "Cluster parameter group for ${local.name_prefix}"
  
  # HIPAA compliance parameters
  parameter {
    name  = "log_statement"
    value = "all"
  }
  
  parameter {
    name  = "log_min_duration_statement"
    value = "1000"
  }
  
  parameter {
    name  = "log_connections"
    value = "1"
  }
  
  parameter {
    name  = "log_disconnections"
    value = "1"
  }
  
  parameter {
    name  = "ssl"
    value = "1"
  }
  
  # Performance parameters
  parameter {
    name  = "shared_preload_libraries"
    value = "pg_stat_statements"
  }
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-cluster-params"
    Type = "database-configuration"
  })
}

# RDS DB Parameter Group
resource "aws_db_parameter_group" "main" {
  family      = "aurora-postgresql15"
  name        = "${local.name_prefix}-db-params"
  description = "DB parameter group for ${local.name_prefix}"
  
  # Additional instance-level parameters
  parameter {
    name  = "log_lock_waits"
    value = "1"
  }
  
  parameter {
    name  = "log_temp_files"
    value = "0"
  }
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-db-params"
    Type = "database-configuration"
  })
}

# Aurora PostgreSQL Cluster
resource "aws_rds_cluster" "main" {
  cluster_identifier     = "${local.name_prefix}-cluster"
  engine                = "aurora-postgresql"
  engine_version        = var.postgres_version
  database_name         = var.database_name
  master_username       = var.database_user
  manage_master_user_password = true
  master_user_secret_kms_key_id = aws_kms_key.database.arn
  
  # Network configuration
  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  
  # Backup configuration
  backup_retention_period = 30  # 30 days for healthcare compliance
  preferred_backup_window = "03:00-04:00"
  backup_target          = "region"
  copy_tags_to_snapshot  = true
  
  # Maintenance
  preferred_maintenance_window = "sun:04:00-sun:05:00"
  
  # Security
  storage_encrypted               = true
  kms_key_id                     = aws_kms_key.database.arn
  iam_database_authentication_enabled = true
  
  # Monitoring
  enabled_cloudwatch_logs_exports = ["postgresql"]
  monitoring_interval            = 60
  monitoring_role_arn           = aws_iam_role.rds_monitoring.arn
  performance_insights_enabled   = true
  performance_insights_kms_key_id = aws_kms_key.database.arn
  
  # Parameter groups
  db_cluster_parameter_group_name = aws_rds_cluster_parameter_group.main.name
  
  # Deletion protection for production
  deletion_protection = var.environment == "production" ? true : false
  skip_final_snapshot = var.environment == "production" ? false : true
  final_snapshot_identifier = var.environment == "production" ? "${local.name_prefix}-final-snapshot-${formatdate("YYYY-MM-DD-hhmm", timestamp())}" : null
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-cluster"
    Type = "database-cluster"
  })
}

# RDS Cluster Instances
resource "aws_rds_cluster_instance" "cluster_instances" {
  count              = var.database_instance_count
  identifier         = "${local.name_prefix}-instance-${count.index + 1}"
  cluster_identifier = aws_rds_cluster.main.id
  instance_class     = var.database_instance_class
  engine             = aws_rds_cluster.main.engine
  engine_version     = aws_rds_cluster.main.engine_version
  
  # Parameter group
  db_parameter_group_name = aws_db_parameter_group.main.name
  
  # Monitoring
  monitoring_interval = 60
  monitoring_role_arn = aws_iam_role.rds_monitoring.arn
  
  # Performance Insights
  performance_insights_enabled = true
  performance_insights_kms_key_id = aws_kms_key.database.arn
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-instance-${count.index + 1}"
    Type = "database-instance"
  })
}

# RDS Monitoring IAM Role
resource "aws_iam_role" "rds_monitoring" {
  name = "${local.name_prefix}-rds-monitoring-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "monitoring.rds.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "rds_monitoring" {
  role       = aws_iam_role.rds_monitoring.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole"
}

# ============================================================================
# ELASTICACHE REDIS - PRODUCTION CLUSTER
# ============================================================================

# Security Group for ElastiCache
resource "aws_security_group" "redis" {
  name_prefix = "${local.name_prefix}-redis-"
  vpc_id      = data.aws_vpc.main.id
  description = "Security group for ${local.name_prefix} Redis cluster"
  
  # Allow Redis access from EKS nodes
  ingress {
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_node.id]
    description     = "Redis access from EKS nodes"
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-redis-sg"
    Type = "cache-security"
  })
}

# ElastiCache Subnet Group
resource "aws_elasticache_subnet_group" "main" {
  name       = "${local.name_prefix}-redis-subnet-group"
  subnet_ids = data.aws_subnets.private.ids
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-redis-subnet-group"
    Type = "cache-networking"
  })
}

# ElastiCache Parameter Group
resource "aws_elasticache_parameter_group" "redis" {
  family = "redis7.x"
  name   = "${local.name_prefix}-redis-params"
  
  # Security parameters
  parameter {
    name  = "requirepass"
    value = "yes"
  }
  
  # Performance parameters
  parameter {
    name  = "maxmemory-policy"
    value = "allkeys-lru"
  }
  
  parameter {
    name  = "timeout"
    value = "300"
  }
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-redis-params"
    Type = "cache-configuration"
  })
}

# ElastiCache Replication Group
resource "aws_elasticache_replication_group" "main" {
  replication_group_id         = "${local.name_prefix}-redis"
  description                  = "Redis cluster for ${local.name_prefix}"
  port                         = 6379
  parameter_group_name         = aws_elasticache_parameter_group.redis.name
  node_type                    = var.redis_node_type
  num_cache_clusters          = var.redis_num_replicas
  
  # Security
  at_rest_encryption_enabled   = true
  transit_encryption_enabled   = true
  auth_token                   = random_password.redis_password.result
  kms_key_id                   = aws_kms_key.main.arn
  
  # Network
  subnet_group_name            = aws_elasticache_subnet_group.main.name
  security_group_ids           = [aws_security_group.redis.id]
  
  # Backup
  snapshot_retention_limit     = 7
  snapshot_window             = "03:00-05:00"
  maintenance_window          = "sun:05:00-sun:06:00"
  
  # Monitoring
  notification_topic_arn       = aws_sns_topic.alerts.arn
  
  # Multi-AZ
  multi_az_enabled            = true
  automatic_failover_enabled   = true
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-redis-cluster"
    Type = "cache-cluster"
  })
}

# ============================================================================
# S3 BUCKETS FOR BACKUPS
# ============================================================================

# S3 Bucket for backups
resource "aws_s3_bucket" "backups" {
  bucket = "${local.name_prefix}-backups-${random_id.bucket_suffix.hex}"
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-backups"
    Type = "backup-storage"
  })
}

resource "random_id" "bucket_suffix" {
  byte_length = 4
}

# S3 bucket versioning
resource "aws_s3_bucket_versioning" "backups" {
  bucket = aws_s3_bucket.backups.id
  versioning_configuration {
    status = "Enabled"
  }
}

# S3 bucket encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "backups" {
  bucket = aws_s3_bucket.backups.id
  
  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.main.arn
      sse_algorithm     = "aws:kms"
    }
    bucket_key_enabled = true
  }
}

# S3 bucket public access block
resource "aws_s3_bucket_public_access_block" "backups" {
  bucket = aws_s3_bucket.backups.id
  
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# S3 bucket lifecycle configuration
resource "aws_s3_bucket_lifecycle_configuration" "backups" {
  bucket = aws_s3_bucket.backups.id
  
  rule {
    id     = "backup_lifecycle"
    status = "Enabled"
    
    # Transition to IA after 30 days
    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }
    
    # Transition to Glacier after 90 days
    transition {
      days          = 90
      storage_class = "GLACIER"
    }
    
    # Delete after 7 years (HIPAA requirement)
    expiration {
      days = 2555
    }
    
    # Handle incomplete multipart uploads
    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
}

# ============================================================================
# CLOUDWATCH AND MONITORING
# ============================================================================

# SNS Topic for alerts
resource "aws_sns_topic" "alerts" {
  name         = "${local.name_prefix}-alerts"
  display_name = "Alerts for ${local.name_prefix}"
  kms_master_key_id = aws_kms_key.main.arn
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-alerts"
    Type = "monitoring-notifications"
  })
}

# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "main" {
  name              = "/aws/eks/${local.name_prefix}"
  retention_in_days = 90  # Healthcare compliance
  kms_key_id        = aws_kms_key.main.arn
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-logs"
    Type = "logging"
  })
}

# CloudWatch Metric Filters for healthcare compliance
resource "aws_cloudwatch_log_metric_filter" "error_rate" {
  name           = "${local.name_prefix}-error-rate"
  log_group_name = aws_cloudwatch_log_group.main.name
  
  pattern = "[timestamp, request_id, level=\"ERROR\", ...]"
  
  metric_transformation {
    name      = "${local.name_prefix}_error_count"
    namespace = "CardioFit/MedicationService"
    value     = "1"
  }
}

# CloudWatch Alarms
resource "aws_cloudwatch_metric_alarm" "high_error_rate" {
  alarm_name          = "${local.name_prefix}-high-error-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "${local.name_prefix}_error_count"
  namespace           = "CardioFit/MedicationService"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "This metric monitors error rate for ${local.name_prefix}"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-high-error-rate-alarm"
    Type = "monitoring-alarm"
  })
}

resource "aws_cloudwatch_metric_alarm" "database_cpu" {
  alarm_name          = "${local.name_prefix}-database-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/RDS"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors database CPU utilization"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  
  dimensions = {
    DBClusterIdentifier = aws_rds_cluster.main.cluster_identifier
  }
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-database-cpu-alarm"
    Type = "monitoring-alarm"
  })
}

# ============================================================================
# SECURITY GROUPS FOR EKS NODES
# ============================================================================

resource "aws_security_group" "eks_node" {
  name_prefix = "${local.name_prefix}-eks-node-"
  vpc_id      = data.aws_vpc.main.id
  description = "Security group for EKS worker nodes running ${local.name_prefix}"
  
  # Allow node to node communication
  ingress {
    from_port = 0
    to_port   = 65535
    protocol  = "tcp"
    self      = true
  }
  
  # Allow pods to communicate with cluster API Server
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.main.cidr_block]
  }
  
  # Healthcare-specific port for secure communication
  ingress {
    from_port   = 8005
    to_port     = 8005
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.main.cidr_block]
    description = "Medication service HTTPS port"
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-eks-node-sg"
    Type = "compute-security"
    "kubernetes.io/cluster/${var.eks_cluster_name}" = "owned"
  })
}

# ============================================================================
# KUBERNETES NAMESPACE
# ============================================================================

resource "kubernetes_namespace" "main" {
  metadata {
    name = var.k8s_namespace
    
    labels = {
      name                    = var.k8s_namespace
      environment            = var.environment
      compliance             = "hipaa"
      "security-policy"      = "healthcare"
      version                = "v1"
      "pod-security.kubernetes.io/enforce" = "restricted"
      "pod-security.kubernetes.io/audit"   = "restricted"
      "pod-security.kubernetes.io/warn"    = "restricted"
    }
    
    annotations = {
      description                              = "Medication Service V2 - FHIR-compliant medication management"
      contact                                 = "devops@cardiofit.health"
      "compliance.policy/hipaa"               = "enabled"
      "compliance.policy/soc2"                = "enabled"
      "security.policy/network-isolation"    = "strict"
      "security.policy/pod-security"          = "restricted"
      "cost-center"                           = "clinical-services"
    }
  }
}

# ============================================================================
# KUBERNETES SERVICE ACCOUNT
# ============================================================================

resource "kubernetes_service_account" "main" {
  metadata {
    name      = var.service_name
    namespace = kubernetes_namespace.main.metadata[0].name
    
    annotations = {
      "eks.amazonaws.com/role-arn" = aws_iam_role.service_role.arn
    }
    
    labels = {
      app       = var.service_name
      component = "serviceaccount"
      version   = "v1"
    }
  }
  
  automount_service_account_token = true
}

# ============================================================================
# OUTPUTS
# ============================================================================

output "database_endpoint" {
  description = "Aurora cluster endpoint"
  value       = aws_rds_cluster.main.endpoint
  sensitive   = true
}

output "database_reader_endpoint" {
  description = "Aurora cluster reader endpoint"
  value       = aws_rds_cluster.main.reader_endpoint
  sensitive   = true
}

output "redis_endpoint" {
  description = "Redis cluster endpoint"
  value       = aws_elasticache_replication_group.main.configuration_endpoint_address
  sensitive   = true
}

output "kms_key_id" {
  description = "KMS key ID for encryption"
  value       = aws_kms_key.main.key_id
}

output "s3_backup_bucket" {
  description = "S3 bucket for backups"
  value       = aws_s3_bucket.backups.bucket
}

output "iam_role_arn" {
  description = "IAM role ARN for the service"
  value       = aws_iam_role.service_role.arn
}

output "security_group_eks_node" {
  description = "Security group ID for EKS nodes"
  value       = aws_security_group.eks_node.id
}

output "kubernetes_namespace" {
  description = "Kubernetes namespace"
  value       = kubernetes_namespace.main.metadata[0].name
}

output "service_account_name" {
  description = "Kubernetes service account name"
  value       = kubernetes_service_account.main.metadata[0].name
}