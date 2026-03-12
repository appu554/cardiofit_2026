# Project Configuration
variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for resources"
  type        = string
  default     = "us-central1"
}

variable "environment" {
  description = "Environment name (production, staging, development)"
  type        = string
  default     = "production"
}

# Secrets (provided via terraform.tfvars or environment variables)
variable "nhs_trud_api_key" {
  description = "NHS TRUD API key for SNOMED CT downloads"
  type        = string
  sensitive   = true
}

variable "umls_api_key" {
  description = "UMLS API key for RxNorm downloads"
  type        = string
  sensitive   = true
}

variable "loinc_username" {
  description = "LOINC.org username"
  type        = string
  sensitive   = true
}

variable "loinc_password" {
  description = "LOINC.org password"
  type        = string
  sensitive   = true
}

variable "github_token" {
  description = "GitHub personal access token for repository dispatch"
  type        = string
  sensitive   = true
}

variable "github_repository" {
  description = "GitHub repository in format owner/repo"
  type        = string
  default     = "cardiofit/kb7-terminology"
}

# Notification Configuration
variable "slack_webhook_url" {
  description = "Slack webhook URL for notifications"
  type        = string
  sensitive   = true
  default     = ""
}

variable "notification_email" {
  description = "Email address for failure notifications"
  type        = string
  default     = "kb7-team@cardiofit.ai"
}

# Scheduler Configuration
variable "schedule_cron" {
  description = "Cron schedule for monthly terminology updates"
  type        = string
  default     = "0 2 1 * *" # 1st of month, 2 AM UTC
}

variable "schedule_timezone" {
  description = "Timezone for scheduler"
  type        = string
  default     = "UTC"
}

# Cost Control
variable "max_instance_count" {
  description = "Maximum concurrent function instances (cost control)"
  type        = number
  default     = 1
}

# Bucket Configuration
variable "source_bucket_retention_days" {
  description = "Retention period for source files in days"
  type        = number
  default     = 180
}

variable "artifact_bucket_retention_days" {
  description = "Retention period for artifacts in days"
  type        = number
  default     = 365
}

variable "nearline_transition_days" {
  description = "Days before transitioning to Nearline storage"
  type        = number
  default     = 30
}
