# Terraform Variables for Medication Service V2
# Healthcare-grade infrastructure configuration variables

# ============================================================================
# PROJECT AND SERVICE CONFIGURATION
# ============================================================================

variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "cardiofit"
  
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.project_name))
    error_message = "Project name must contain only lowercase letters, numbers, and hyphens."
  }
}

variable "service_name" {
  description = "Name of the service"
  type        = string
  default     = "medication-service-v2"
  
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.service_name))
    error_message = "Service name must contain only lowercase letters, numbers, and hyphens."
  }
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
  
  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "service_version" {
  description = "Version of the service"
  type        = string
  default     = "1.0.0"
  
  validation {
    condition     = can(regex("^[0-9]+\\.[0-9]+\\.[0-9]+$", var.service_version))
    error_message = "Service version must follow semantic versioning (e.g., 1.0.0)."
  }
}

# ============================================================================
# AWS CONFIGURATION
# ============================================================================

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
  
  validation {
    condition = can(regex("^[a-z0-9-]+$", var.aws_region))
    error_message = "AWS region must be a valid region identifier."
  }
}

variable "aws_profile" {
  description = "AWS profile to use"
  type        = string
  default     = "default"
}

variable "availability_zones" {
  description = "List of availability zones"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b", "us-east-1c"]
  
  validation {
    condition     = length(var.availability_zones) >= 2
    error_message = "At least 2 availability zones must be specified for high availability."
  }
}

# ============================================================================
# KUBERNETES CONFIGURATION
# ============================================================================

variable "eks_cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  
  validation {
    condition     = can(regex("^[a-zA-Z0-9-_]+$", var.eks_cluster_name))
    error_message = "EKS cluster name must contain only letters, numbers, hyphens, and underscores."
  }
}

variable "k8s_namespace" {
  description = "Kubernetes namespace for the service"
  type        = string
  default     = "cardiofit-medication-v2"
  
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.k8s_namespace))
    error_message = "Kubernetes namespace must contain only lowercase letters, numbers, and hyphens."
  }
}

variable "k8s_service_account" {
  description = "Kubernetes service account name"
  type        = string
  default     = "medication-service-v2"
  
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.k8s_service_account))
    error_message = "Kubernetes service account must contain only lowercase letters, numbers, and hyphens."
  }
}

# ============================================================================
# DATABASE CONFIGURATION
# ============================================================================

variable "database_name" {
  description = "Name of the database"
  type        = string
  default     = "medication_v2"
  
  validation {
    condition     = can(regex("^[a-z0-9_]+$", var.database_name))
    error_message = "Database name must contain only lowercase letters, numbers, and underscores."
  }
}

variable "database_user" {
  description = "Database master username"
  type        = string
  default     = "medication_user"
  
  validation {
    condition     = can(regex("^[a-z0-9_]+$", var.database_user))
    error_message = "Database user must contain only lowercase letters, numbers, and underscores."
  }
}

variable "postgres_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "15.4"
  
  validation {
    condition     = can(regex("^[0-9]+\\.[0-9]+$", var.postgres_version))
    error_message = "PostgreSQL version must be in format X.Y (e.g., 15.4)."
  }
}

variable "database_instance_class" {
  description = "Database instance class"
  type        = string
  default     = "db.r6g.large"
  
  validation {
    condition = contains([
      "db.t4g.medium", "db.t4g.large", "db.t4g.xlarge", "db.t4g.2xlarge",
      "db.r6g.large", "db.r6g.xlarge", "db.r6g.2xlarge", "db.r6g.4xlarge",
      "db.r6g.8xlarge", "db.r6g.12xlarge", "db.r6g.16xlarge",
      "db.r6i.large", "db.r6i.xlarge", "db.r6i.2xlarge", "db.r6i.4xlarge"
    ], var.database_instance_class)
    error_message = "Database instance class must be a valid RDS instance type."
  }
}

variable "database_instance_count" {
  description = "Number of database instances in the cluster"
  type        = number
  default     = 3
  
  validation {
    condition     = var.database_instance_count >= 2 && var.database_instance_count <= 15
    error_message = "Database instance count must be between 2 and 15."
  }
}

variable "database_backup_retention_period" {
  description = "Database backup retention period in days"
  type        = number
  default     = 30
  
  validation {
    condition     = var.database_backup_retention_period >= 7 && var.database_backup_retention_period <= 35
    error_message = "Database backup retention period must be between 7 and 35 days."
  }
}

variable "database_preferred_backup_window" {
  description = "Preferred backup window"
  type        = string
  default     = "03:00-04:00"
  
  validation {
    condition     = can(regex("^[0-9]{2}:[0-9]{2}-[0-9]{2}:[0-9]{2}$", var.database_preferred_backup_window))
    error_message = "Database backup window must be in format HH:MM-HH:MM."
  }
}

variable "database_preferred_maintenance_window" {
  description = "Preferred maintenance window"
  type        = string
  default     = "sun:04:00-sun:05:00"
  
  validation {
    condition     = can(regex("^(mon|tue|wed|thu|fri|sat|sun):[0-9]{2}:[0-9]{2}-(mon|tue|wed|thu|fri|sat|sun):[0-9]{2}:[0-9]{2}$", var.database_preferred_maintenance_window))
    error_message = "Database maintenance window must be in format ddd:HH:MM-ddd:HH:MM."
  }
}

variable "enable_database_deletion_protection" {
  description = "Enable deletion protection for the database"
  type        = bool
  default     = true
}

variable "enable_database_performance_insights" {
  description = "Enable Performance Insights for the database"
  type        = bool
  default     = true
}

# ============================================================================
# REDIS/ELASTICACHE CONFIGURATION
# ============================================================================

variable "redis_node_type" {
  description = "Redis node type"
  type        = string
  default     = "cache.r6g.large"
  
  validation {
    condition = contains([
      "cache.t3.micro", "cache.t3.small", "cache.t3.medium",
      "cache.t4g.micro", "cache.t4g.small", "cache.t4g.medium",
      "cache.r6g.large", "cache.r6g.xlarge", "cache.r6g.2xlarge",
      "cache.r6g.4xlarge", "cache.r6g.8xlarge", "cache.r6g.12xlarge"
    ], var.redis_node_type)
    error_message = "Redis node type must be a valid ElastiCache instance type."
  }
}

variable "redis_num_replicas" {
  description = "Number of Redis replica nodes"
  type        = number
  default     = 3
  
  validation {
    condition     = var.redis_num_replicas >= 1 && var.redis_num_replicas <= 5
    error_message = "Redis replica count must be between 1 and 5."
  }
}

variable "redis_snapshot_retention_limit" {
  description = "Number of days to retain Redis snapshots"
  type        = number
  default     = 7
  
  validation {
    condition     = var.redis_snapshot_retention_limit >= 1 && var.redis_snapshot_retention_limit <= 35
    error_message = "Redis snapshot retention must be between 1 and 35 days."
  }
}

variable "redis_snapshot_window" {
  description = "Redis snapshot window"
  type        = string
  default     = "03:00-05:00"
  
  validation {
    condition     = can(regex("^[0-9]{2}:[0-9]{2}-[0-9]{2}:[0-9]{2}$", var.redis_snapshot_window))
    error_message = "Redis snapshot window must be in format HH:MM-HH:MM."
  }
}

variable "redis_maintenance_window" {
  description = "Redis maintenance window"
  type        = string
  default     = "sun:05:00-sun:06:00"
  
  validation {
    condition     = can(regex("^(mon|tue|wed|thu|fri|sat|sun):[0-9]{2}:[0-9]{2}-(mon|tue|wed|thu|fri|sat|sun):[0-9]{2}:[0-9]{2}$", var.redis_maintenance_window))
    error_message = "Redis maintenance window must be in format ddd:HH:MM-ddd:HH:MM."
  }
}

variable "enable_redis_auth_token" {
  description = "Enable auth token for Redis"
  type        = bool
  default     = true
}

variable "enable_redis_encryption_at_rest" {
  description = "Enable encryption at rest for Redis"
  type        = bool
  default     = true
}

variable "enable_redis_encryption_in_transit" {
  description = "Enable encryption in transit for Redis"
  type        = bool
  default     = true
}

# ============================================================================
# STORAGE CONFIGURATION
# ============================================================================

variable "backup_bucket_name" {
  description = "Name of the S3 bucket for backups"
  type        = string
  default     = ""
  
  validation {
    condition = var.backup_bucket_name == "" || can(regex("^[a-z0-9.-]+$", var.backup_bucket_name))
    error_message = "Backup bucket name must contain only lowercase letters, numbers, dots, and hyphens."
  }
}

variable "backup_retention_days" {
  description = "Number of days to retain backups"
  type        = number
  default     = 2555  # 7 years for HIPAA compliance
  
  validation {
    condition     = var.backup_retention_days >= 30 && var.backup_retention_days <= 3650
    error_message = "Backup retention must be between 30 days and 10 years."
  }
}

variable "enable_s3_versioning" {
  description = "Enable S3 versioning for backup bucket"
  type        = bool
  default     = true
}

variable "s3_storage_class_transition_days" {
  description = "Number of days after which objects transition to IA storage class"
  type        = number
  default     = 30
  
  validation {
    condition     = var.s3_storage_class_transition_days >= 1 && var.s3_storage_class_transition_days <= 365
    error_message = "S3 storage class transition days must be between 1 and 365."
  }
}

variable "s3_glacier_transition_days" {
  description = "Number of days after which objects transition to Glacier"
  type        = number
  default     = 90
  
  validation {
    condition     = var.s3_glacier_transition_days >= 30 && var.s3_glacier_transition_days <= 365
    error_message = "S3 Glacier transition days must be between 30 and 365."
  }
}

# ============================================================================
# SECURITY CONFIGURATION
# ============================================================================

variable "enable_encryption_at_rest" {
  description = "Enable encryption at rest for all services"
  type        = bool
  default     = true
}

variable "enable_encryption_in_transit" {
  description = "Enable encryption in transit for all services"
  type        = bool
  default     = true
}

variable "kms_key_deletion_window" {
  description = "Number of days after which KMS keys can be deleted"
  type        = number
  default     = 7
  
  validation {
    condition     = var.kms_key_deletion_window >= 7 && var.kms_key_deletion_window <= 30
    error_message = "KMS key deletion window must be between 7 and 30 days."
  }
}

variable "enable_kms_key_rotation" {
  description = "Enable automatic rotation of KMS keys"
  type        = bool
  default     = true
}

variable "allowed_cidr_blocks" {
  description = "List of CIDR blocks allowed to access the service"
  type        = list(string)
  default     = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  
  validation {
    condition     = length(var.allowed_cidr_blocks) > 0
    error_message = "At least one CIDR block must be specified."
  }
}

variable "enable_waf" {
  description = "Enable AWS WAF for the service"
  type        = bool
  default     = true
}

variable "enable_vpc_flow_logs" {
  description = "Enable VPC Flow Logs"
  type        = bool
  default     = true
}

# ============================================================================
# MONITORING AND LOGGING CONFIGURATION
# ============================================================================

variable "cloudwatch_log_retention_days" {
  description = "CloudWatch log retention period in days"
  type        = number
  default     = 90
  
  validation {
    condition     = contains([1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1827, 3653], var.cloudwatch_log_retention_days)
    error_message = "CloudWatch log retention days must be a valid retention period."
  }
}

variable "enable_enhanced_monitoring" {
  description = "Enable enhanced monitoring for RDS"
  type        = bool
  default     = true
}

variable "monitoring_interval" {
  description = "Monitoring interval for RDS enhanced monitoring"
  type        = number
  default     = 60
  
  validation {
    condition     = contains([1, 5, 10, 15, 30, 60], var.monitoring_interval)
    error_message = "Monitoring interval must be one of: 1, 5, 10, 15, 30, 60."
  }
}

variable "enable_performance_insights" {
  description = "Enable Performance Insights"
  type        = bool
  default     = true
}

variable "performance_insights_retention_period" {
  description = "Performance Insights retention period in days"
  type        = number
  default     = 7
  
  validation {
    condition     = contains([7, 731], var.performance_insights_retention_period)
    error_message = "Performance Insights retention period must be 7 or 731 days."
  }
}

variable "enable_cloudtrail" {
  description = "Enable CloudTrail for auditing"
  type        = bool
  default     = true
}

variable "enable_config" {
  description = "Enable AWS Config for compliance monitoring"
  type        = bool
  default     = true
}

# ============================================================================
# HIGH AVAILABILITY AND DISASTER RECOVERY
# ============================================================================

variable "enable_multi_az" {
  description = "Enable Multi-AZ deployment"
  type        = bool
  default     = true
}

variable "enable_cross_region_backup" {
  description = "Enable cross-region backup"
  type        = bool
  default     = true
}

variable "disaster_recovery_region" {
  description = "AWS region for disaster recovery"
  type        = string
  default     = "us-west-2"
  
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.disaster_recovery_region))
    error_message = "Disaster recovery region must be a valid AWS region."
  }
}

variable "rto_minutes" {
  description = "Recovery Time Objective in minutes"
  type        = number
  default     = 60
  
  validation {
    condition     = var.rto_minutes >= 15 && var.rto_minutes <= 480
    error_message = "RTO must be between 15 minutes and 8 hours."
  }
}

variable "rpo_minutes" {
  description = "Recovery Point Objective in minutes"
  type        = number
  default     = 15
  
  validation {
    condition     = var.rpo_minutes >= 5 && var.rpo_minutes <= 240
    error_message = "RPO must be between 5 minutes and 4 hours."
  }
}

# ============================================================================
# APPLICATION CONFIGURATION
# ============================================================================

variable "application_replicas" {
  description = "Number of application replicas"
  type        = number
  default     = 3
  
  validation {
    condition     = var.application_replicas >= 2 && var.application_replicas <= 10
    error_message = "Application replicas must be between 2 and 10."
  }
}

variable "application_cpu_request" {
  description = "CPU request for application pods"
  type        = string
  default     = "250m"
  
  validation {
    condition     = can(regex("^[0-9]+m?$", var.application_cpu_request))
    error_message = "CPU request must be in format like '250m' or '1'."
  }
}

variable "application_memory_request" {
  description = "Memory request for application pods"
  type        = string
  default     = "256Mi"
  
  validation {
    condition     = can(regex("^[0-9]+[KMG]i?$", var.application_memory_request))
    error_message = "Memory request must be in format like '256Mi' or '1Gi'."
  }
}

variable "application_cpu_limit" {
  description = "CPU limit for application pods"
  type        = string
  default     = "500m"
  
  validation {
    condition     = can(regex("^[0-9]+m?$", var.application_cpu_limit))
    error_message = "CPU limit must be in format like '500m' or '1'."
  }
}

variable "application_memory_limit" {
  description = "Memory limit for application pods"
  type        = string
  default     = "512Mi"
  
  validation {
    condition     = can(regex("^[0-9]+[KMG]i?$", var.application_memory_limit))
    error_message = "Memory limit must be in format like '512Mi' or '1Gi'."
  }
}

# ============================================================================
# HEALTHCARE COMPLIANCE CONFIGURATION
# ============================================================================

variable "hipaa_compliance_enabled" {
  description = "Enable HIPAA compliance features"
  type        = bool
  default     = true
}

variable "soc2_compliance_enabled" {
  description = "Enable SOC2 compliance features"
  type        = bool
  default     = true
}

variable "enable_audit_logging" {
  description = "Enable comprehensive audit logging"
  type        = bool
  default     = true
}

variable "enable_data_encryption" {
  description = "Enable data encryption at all levels"
  type        = bool
  default     = true
}

variable "enable_access_logging" {
  description = "Enable access logging for all services"
  type        = bool
  default     = true
}

variable "phi_data_retention_years" {
  description = "PHI data retention period in years"
  type        = number
  default     = 7
  
  validation {
    condition     = var.phi_data_retention_years >= 6 && var.phi_data_retention_years <= 50
    error_message = "PHI data retention must be between 6 and 50 years."
  }
}

# ============================================================================
# COST OPTIMIZATION CONFIGURATION
# ============================================================================

variable "enable_cost_optimization" {
  description = "Enable cost optimization features"
  type        = bool
  default     = true
}

variable "enable_scheduled_scaling" {
  description = "Enable scheduled scaling for cost optimization"
  type        = bool
  default     = false
}

variable "weekend_scale_down" {
  description = "Scale down resources during weekends"
  type        = bool
  default     = false
}

variable "cost_center" {
  description = "Cost center for resource allocation"
  type        = string
  default     = "clinical-services"
  
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.cost_center))
    error_message = "Cost center must contain only lowercase letters, numbers, and hyphens."
  }
}

# ============================================================================
# FEATURE FLAGS
# ============================================================================

variable "enable_blue_green_deployment" {
  description = "Enable blue-green deployment strategy"
  type        = bool
  default     = true
}

variable "enable_canary_deployment" {
  description = "Enable canary deployment strategy"
  type        = bool
  default     = false
}

variable "enable_auto_scaling" {
  description = "Enable horizontal pod auto-scaling"
  type        = bool
  default     = true
}

variable "enable_vertical_pod_autoscaling" {
  description = "Enable vertical pod auto-scaling"
  type        = bool
  default     = false
}

variable "enable_pod_disruption_budget" {
  description = "Enable pod disruption budget"
  type        = bool
  default     = true
}

variable "enable_network_policies" {
  description = "Enable Kubernetes network policies"
  type        = bool
  default     = true
}

# ============================================================================
# TESTING AND DEVELOPMENT
# ============================================================================

variable "enable_chaos_engineering" {
  description = "Enable chaos engineering tools"
  type        = bool
  default     = false
}

variable "enable_load_testing" {
  description = "Enable load testing infrastructure"
  type        = bool
  default     = false
}

variable "enable_debug_mode" {
  description = "Enable debug mode (disable in production)"
  type        = bool
  default     = false
  
  validation {
    condition     = var.environment == "production" ? var.enable_debug_mode == false : true
    error_message = "Debug mode must be disabled in production environment."
  }
}

# ============================================================================
# TAGS AND METADATA
# ============================================================================

variable "additional_tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}

variable "team" {
  description = "Team responsible for the service"
  type        = string
  default     = "clinical-platform-team"
  
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.team))
    error_message = "Team name must contain only lowercase letters, numbers, and hyphens."
  }
}

variable "contact_email" {
  description = "Contact email for the service"
  type        = string
  default     = "devops@cardiofit.health"
  
  validation {
    condition     = can(regex("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.contact_email))
    error_message = "Contact email must be a valid email address."
  }
}

variable "documentation_url" {
  description = "URL to the service documentation"
  type        = string
  default     = "https://docs.cardiofit.health/medication-service-v2"
  
  validation {
    condition     = can(regex("^https?://", var.documentation_url))
    error_message = "Documentation URL must be a valid HTTP/HTTPS URL."
  }
}