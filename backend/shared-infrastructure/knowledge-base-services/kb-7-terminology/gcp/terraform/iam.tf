# IAM Configuration - Least Privilege Service Accounts

# Service Account for Cloud Functions
resource "google_service_account" "kb_functions" {
  account_id   = "kb7-functions-${var.environment}"
  display_name = "KB-7 Cloud Functions Service Account"
  description  = "Service account for knowledge factory downloader functions"
}

# Service Account for Cloud Workflows
resource "google_service_account" "kb_workflows" {
  account_id   = "kb7-workflows-${var.environment}"
  display_name = "KB-7 Cloud Workflows Service Account"
  description  = "Service account for knowledge factory orchestration workflow"
}

# Service Account for Cloud Scheduler
resource "google_service_account" "scheduler" {
  account_id   = "kb7-scheduler-${var.environment}"
  display_name = "KB-7 Cloud Scheduler Service Account"
  description  = "Service account for knowledge factory scheduler triggers"
}

# Service Account for GitHub Actions
resource "google_service_account" "github_actions" {
  account_id   = "kb7-github-actions-${var.environment}"
  display_name = "KB-7 GitHub Actions Service Account"
  description  = "Service account for GitHub Actions pipeline access to GCS"
}

# IAM Bindings - Cloud Functions Service Account

# Allow functions to access Secret Manager
resource "google_project_iam_member" "functions_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.kb_functions.email}"
}

# Allow functions to write logs
resource "google_project_iam_member" "functions_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.kb_functions.email}"
}

# Allow functions to write metrics
resource "google_project_iam_member" "functions_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.kb_functions.email}"
}

# IAM Bindings - Workflows Service Account

# Allow workflows to invoke Cloud Functions
resource "google_project_iam_member" "workflows_function_invoker" {
  project = var.project_id
  role    = "roles/cloudfunctions.invoker"
  member  = "serviceAccount:${google_service_account.kb_workflows.email}"
}

# Allow workflows to write logs
resource "google_project_iam_member" "workflows_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.kb_workflows.email}"
}

# IAM Bindings - Scheduler Service Account

# Allow scheduler to invoke workflows (handled by project-level permission below)
resource "google_project_iam_member" "scheduler_workflow_invoker" {
  project = var.project_id
  role    = "roles/workflows.invoker"
  member  = "serviceAccount:${google_service_account.scheduler.email}"
}

# IAM Bindings - GitHub Actions Service Account

# Allow GitHub Actions to read sources bucket (for manual testing)
resource "google_storage_bucket_iam_member" "github_sources_viewer" {
  bucket = google_storage_bucket.kb_sources.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.github_actions.email}"
}

# Storage bucket IAM is defined in storage.tf (objectAdmin for artifacts)

# Output service account emails for reference
output "functions_service_account_email" {
  description = "Email of the Cloud Functions service account"
  value       = google_service_account.kb_functions.email
}

output "workflows_service_account_email" {
  description = "Email of the Cloud Workflows service account"
  value       = google_service_account.kb_workflows.email
}

output "scheduler_service_account_email" {
  description = "Email of the Cloud Scheduler service account"
  value       = google_service_account.scheduler.email
}

output "github_actions_service_account_email" {
  description = "Email of the GitHub Actions service account"
  value       = google_service_account.github_actions.email
}
