# Cloud Functions 2nd Generation for Knowledge Factory Downloads

locals {
  function_location = var.region
  common_env_vars = {
    PROJECT_ID      = var.project_id
    SOURCE_BUCKET   = google_storage_bucket.kb_sources.name
    ENVIRONMENT     = var.environment
    GITHUB_REPO     = var.github_repository
  }
}

# SNOMED CT Downloader Function
resource "google_cloudfunctions2_function" "snomed_downloader" {
  name     = "kb7-snomed-downloader-${var.environment}"
  location = local.function_location

  build_config {
    runtime     = "python311"
    entry_point = "download_snomed"

    source {
      storage_source {
        bucket = google_storage_bucket.function_source.name
        object = google_storage_bucket_object.snomed_downloader_source.name
      }
    }
  }

  service_config {
    max_instance_count    = var.max_instance_count
    min_instance_count    = 0
    available_memory      = "10Gi"  # 10GB for large SNOMED files
    timeout_seconds       = 3600    # 60 minutes - no Lambda timeout!
    available_cpu         = "4"     # 4 vCPUs for faster processing

    environment_variables = merge(local.common_env_vars, {
      SECRET_NAME = google_secret_manager_secret.nhs_trud_api_key.secret_id
    })

    service_account_email = google_service_account.kb_functions.email

    # Allow unauthenticated access for Cloud Workflows (internal only)
    ingress_settings = "ALLOW_INTERNAL_ONLY"
  }

  labels = {
    service     = "kb7-knowledge-factory"
    function    = "snomed-downloader"
    environment = var.environment
  }

  depends_on = [
    google_project_service.required_apis,
    google_storage_bucket_object.snomed_downloader_source
  ]
}

# RxNorm Downloader Function
resource "google_cloudfunctions2_function" "rxnorm_downloader" {
  name     = "kb7-rxnorm-downloader-${var.environment}"
  location = local.function_location

  build_config {
    runtime     = "python311"
    entry_point = "download_rxnorm"

    source {
      storage_source {
        bucket = google_storage_bucket.function_source.name
        object = google_storage_bucket_object.rxnorm_downloader_source.name
      }
    }
  }

  service_config {
    max_instance_count    = var.max_instance_count
    min_instance_count    = 0
    available_memory      = "3Gi"   # 3GB for RxNorm files
    timeout_seconds       = 3600    # 60 minutes
    available_cpu         = "2"

    environment_variables = merge(local.common_env_vars, {
      SECRET_NAME = google_secret_manager_secret.umls_api_key.secret_id
    })

    service_account_email = google_service_account.kb_functions.email
    ingress_settings      = "ALLOW_INTERNAL_ONLY"
  }

  labels = {
    service     = "kb7-knowledge-factory"
    function    = "rxnorm-downloader"
    environment = var.environment
  }

  depends_on = [
    google_project_service.required_apis,
    google_storage_bucket_object.rxnorm_downloader_source
  ]
}

# LOINC Downloader Function
resource "google_cloudfunctions2_function" "loinc_downloader" {
  name     = "kb7-loinc-downloader-${var.environment}"
  location = local.function_location

  build_config {
    runtime     = "python311"
    entry_point = "download_loinc"

    source {
      storage_source {
        bucket = google_storage_bucket.function_source.name
        object = google_storage_bucket_object.loinc_downloader_source.name
      }
    }
  }

  service_config {
    max_instance_count    = var.max_instance_count
    min_instance_count    = 0
    available_memory      = "2Gi"   # 2GB for LOINC files
    timeout_seconds       = 1800    # 30 minutes
    available_cpu         = "1"

    environment_variables = merge(local.common_env_vars, {
      SECRET_NAME = google_secret_manager_secret.loinc_credentials.secret_id
    })

    service_account_email = google_service_account.kb_functions.email
    ingress_settings      = "ALLOW_INTERNAL_ONLY"
  }

  labels = {
    service     = "kb7-knowledge-factory"
    function    = "loinc-downloader"
    environment = var.environment
  }

  depends_on = [
    google_project_service.required_apis,
    google_storage_bucket_object.loinc_downloader_source
  ]
}

# GitHub Dispatcher Function
resource "google_cloudfunctions2_function" "github_dispatcher" {
  name     = "kb7-github-dispatcher-${var.environment}"
  location = local.function_location

  build_config {
    runtime     = "python311"
    entry_point = "dispatch_github_workflow"

    source {
      storage_source {
        bucket = google_storage_bucket.function_source.name
        object = google_storage_bucket_object.github_dispatcher_source.name
      }
    }
  }

  service_config {
    max_instance_count    = var.max_instance_count
    min_instance_count    = 0
    available_memory      = "1Gi"   # 1GB sufficient for API calls
    timeout_seconds       = 300     # 5 minutes
    available_cpu         = "1"

    environment_variables = merge(local.common_env_vars, {
      SECRET_NAME = google_secret_manager_secret.github_token.secret_id
      SLACK_WEBHOOK_SECRET = var.slack_webhook_url != "" ? "kb7-slack-webhook-${var.environment}" : ""
    })

    service_account_email = google_service_account.kb_functions.email
    ingress_settings      = "ALLOW_INTERNAL_ONLY"
  }

  labels = {
    service     = "kb7-knowledge-factory"
    function    = "github-dispatcher"
    environment = var.environment
  }

  depends_on = [
    google_project_service.required_apis,
    google_storage_bucket_object.github_dispatcher_source
  ]
}

# Upload function source code to GCS (placeholder - actual upload in deployment script)
resource "google_storage_bucket_object" "snomed_downloader_source" {
  name   = "functions/snomed-downloader-${filemd5("${path.module}/../functions/snomed-downloader/main.py")}.zip"
  bucket = google_storage_bucket.function_source.name
  source = "${path.module}/../functions/snomed-downloader/function.zip"
}

resource "google_storage_bucket_object" "rxnorm_downloader_source" {
  name   = "functions/rxnorm-downloader-${filemd5("${path.module}/../functions/rxnorm-downloader/main.py")}.zip"
  bucket = google_storage_bucket.function_source.name
  source = "${path.module}/../functions/rxnorm-downloader/function.zip"
}

resource "google_storage_bucket_object" "loinc_downloader_source" {
  name   = "functions/loinc-downloader-${filemd5("${path.module}/../functions/loinc-downloader/main.py")}.zip"
  bucket = google_storage_bucket.function_source.name
  source = "${path.module}/../functions/loinc-downloader/function.zip"
}

resource "google_storage_bucket_object" "github_dispatcher_source" {
  name   = "functions/github-dispatcher-${filemd5("${path.module}/../functions/github-dispatcher/main.py")}.zip"
  bucket = google_storage_bucket.function_source.name
  source = "${path.module}/../functions/github-dispatcher/function.zip"
}

# Outputs
output "function_urls" {
  description = "URLs of deployed Cloud Functions"
  value = {
    snomed_downloader   = google_cloudfunctions2_function.snomed_downloader.service_config[0].uri
    rxnorm_downloader   = google_cloudfunctions2_function.rxnorm_downloader.service_config[0].uri
    loinc_downloader    = google_cloudfunctions2_function.loinc_downloader.service_config[0].uri
    github_dispatcher   = google_cloudfunctions2_function.github_dispatcher.service_config[0].uri
  }
}
