# Cloud Storage Buckets for KB-7 Knowledge Factory

# Source Files Bucket (raw downloads from terminology APIs)
resource "google_storage_bucket" "kb_sources" {
  name          = "${var.project_id}-kb-sources-${var.environment}"
  location      = "US" # Multi-region for availability
  storage_class = "STANDARD"
  force_destroy = false

  # Lifecycle policies for cost optimization
  lifecycle_rule {
    condition {
      age = var.source_bucket_retention_days
    }
    action {
      type = "Delete"
    }
  }

  lifecycle_rule {
    condition {
      age = var.nearline_transition_days
    }
    action {
      type          = "SetStorageClass"
      storage_class = "NEARLINE" # Cheaper storage for older files
    }
  }

  # Enable versioning for safety
  versioning {
    enabled = true
  }

  # Uniform bucket-level access (recommended)
  uniform_bucket_level_access = true

  # Labels for organization
  labels = {
    service     = "kb7-knowledge-factory"
    environment = var.environment
    managed-by  = "terraform"
    purpose     = "source-files"
  }
}

# Artifacts Bucket (processed ontologies for GraphDB)
resource "google_storage_bucket" "kb_artifacts" {
  name          = "${var.project_id}-kb-artifacts-${var.environment}"
  location      = "US"
  storage_class = "STANDARD"
  force_destroy = false

  # Longer retention for production artifacts
  lifecycle_rule {
    condition {
      age = var.artifact_bucket_retention_days
    }
    action {
      type = "Delete"
    }
  }

  lifecycle_rule {
    condition {
      age = 90 # Transition after 3 months
    }
    action {
      type          = "SetStorageClass"
      storage_class = "NEARLINE"
    }
  }

  versioning {
    enabled = true
  }

  uniform_bucket_level_access = true

  labels = {
    service     = "kb7-knowledge-factory"
    environment = var.environment
    managed-by  = "terraform"
    purpose     = "processed-artifacts"
  }
}

# Grant Cloud Functions read/write access to sources bucket
resource "google_storage_bucket_iam_member" "sources_function_object_admin" {
  bucket = google_storage_bucket.kb_sources.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.kb_functions.email}"
}

# Grant Cloud Functions read access to artifacts bucket (write via GitHub Actions)
resource "google_storage_bucket_iam_member" "artifacts_function_object_viewer" {
  bucket = google_storage_bucket.kb_artifacts.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.kb_functions.email}"
}

# Grant GitHub Actions service account write access to artifacts
resource "google_storage_bucket_iam_member" "artifacts_github_object_admin" {
  bucket = google_storage_bucket.kb_artifacts.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.github_actions.email}"
}

# Outputs for bucket information
output "sources_bucket_name" {
  description = "Name of the sources bucket"
  value       = google_storage_bucket.kb_sources.name
}

output "sources_bucket_url" {
  description = "URL of the sources bucket"
  value       = google_storage_bucket.kb_sources.url
}

output "artifacts_bucket_name" {
  description = "Name of the artifacts bucket"
  value       = google_storage_bucket.kb_artifacts.name
}

output "artifacts_bucket_url" {
  description = "URL of the artifacts bucket"
  value       = google_storage_bucket.kb_artifacts.url
}
