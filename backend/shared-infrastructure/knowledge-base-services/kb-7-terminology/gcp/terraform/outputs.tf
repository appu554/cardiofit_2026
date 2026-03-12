# Terraform Outputs - KB-7 Knowledge Factory GCP Implementation

# Project Information
output "project_id" {
  description = "GCP project ID"
  value       = var.project_id
}

output "region" {
  description = "GCP region"
  value       = var.region
}

output "environment" {
  description = "Environment name"
  value       = var.environment
}

# Storage Buckets (from storage.tf)
# These are re-exported here for convenience

# Cloud Functions (from functions.tf)
# These are re-exported here for convenience

# Cloud Workflows (from workflows.tf)
# These are re-exported here for convenience

# IAM Service Accounts (from iam.tf)
# These are re-exported here for convenience

# Secrets (from secrets.tf)
# These are re-exported here for convenience

# Cloud Scheduler (from scheduler.tf)
# These are re-exported here for convenience

# Monitoring (from monitoring.tf)
# These are re-exported here for convenience

# Summary Output
output "deployment_summary" {
  description = "Summary of deployed resources"
  value = {
    buckets = {
      sources   = google_storage_bucket.kb_sources.name
      artifacts = google_storage_bucket.kb_artifacts.name
    }
    functions = {
      snomed_downloader = google_cloudfunctions2_function.snomed_downloader.name
      rxnorm_downloader = google_cloudfunctions2_function.rxnorm_downloader.name
      loinc_downloader  = google_cloudfunctions2_function.loinc_downloader.name
      github_dispatcher = google_cloudfunctions2_function.github_dispatcher.name
    }
    workflow = {
      name = google_workflows_workflow.kb_factory.name
      id   = google_workflows_workflow.kb_factory.id
    }
    scheduler = {
      name     = google_cloud_scheduler_job.monthly_terminology_update.name
      schedule = google_cloud_scheduler_job.monthly_terminology_update.schedule
    }
    service_accounts = {
      functions = google_service_account.kb_functions.email
      workflows = google_service_account.kb_workflows.email
      scheduler = google_service_account.scheduler.email
      github    = google_service_account.github_actions.email
    }
  }
}

# Quick Access URLs
output "quick_access" {
  description = "Quick access URLs for management"
  value = {
    console_workflows = "https://console.cloud.google.com/workflows?project=${var.project_id}"
    console_functions = "https://console.cloud.google.com/functions?project=${var.project_id}"
    console_storage   = "https://console.cloud.google.com/storage/browser?project=${var.project_id}"
    console_scheduler = "https://console.cloud.google.com/cloudscheduler?project=${var.project_id}"
    console_logs      = "https://console.cloud.google.com/logs?project=${var.project_id}"
    console_monitoring = "https://console.cloud.google.com/monitoring?project=${var.project_id}"
  }
}

# Manual Testing Commands
output "testing_commands" {
  description = "Commands for manual testing"
  value = {
    test_workflow = "gcloud workflows execute ${google_workflows_workflow.kb_factory.name} --location=${var.region} --data='{\"trigger\":\"manual-test\"}'"
    view_logs     = "gcloud logging read 'resource.type=\"cloud_function\" AND resource.labels.function_name=~\"kb7-.*\"' --limit=50 --format=json"
    test_snomed   = "gcloud functions call ${google_cloudfunctions2_function.snomed_downloader.name} --region=${var.region} --gen2 --data='{\"test\":true}'"
  }
}
