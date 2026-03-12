# Secret Manager Configuration for API Keys and Credentials

# NHS TRUD API Key (SNOMED CT)
resource "google_secret_manager_secret" "nhs_trud_api_key" {
  secret_id = "kb7-nhs-trud-api-key-${var.environment}"

  replication {
    auto {}
  }

  labels = {
    service     = "kb7-knowledge-factory"
    rotation    = "90days"
    environment = var.environment
    terminology = "snomed-ct"
  }
}

resource "google_secret_manager_secret_version" "nhs_trud_api_key" {
  secret      = google_secret_manager_secret.nhs_trud_api_key.id
  secret_data = var.nhs_trud_api_key
}

# UMLS API Key (RxNorm)
resource "google_secret_manager_secret" "umls_api_key" {
  secret_id = "kb7-umls-api-key-${var.environment}"

  replication {
    auto {}
  }

  labels = {
    service     = "kb7-knowledge-factory"
    rotation    = "90days"
    environment = var.environment
    terminology = "rxnorm"
  }
}

resource "google_secret_manager_secret_version" "umls_api_key" {
  secret      = google_secret_manager_secret.umls_api_key.id
  secret_data = var.umls_api_key
}

# LOINC Credentials
resource "google_secret_manager_secret" "loinc_credentials" {
  secret_id = "kb7-loinc-credentials-${var.environment}"

  replication {
    auto {}
  }

  labels = {
    service     = "kb7-knowledge-factory"
    rotation    = "90days"
    environment = var.environment
    terminology = "loinc"
  }
}

resource "google_secret_manager_secret_version" "loinc_credentials" {
  secret = google_secret_manager_secret.loinc_credentials.id
  secret_data = jsonencode({
    username = var.loinc_username
    password = var.loinc_password
  })
}

# GitHub Token (for repository dispatch)
resource "google_secret_manager_secret" "github_token" {
  secret_id = "kb7-github-token-${var.environment}"

  replication {
    auto {}
  }

  labels = {
    service     = "kb7-knowledge-factory"
    rotation    = "90days"
    environment = var.environment
    purpose     = "github-dispatch"
  }
}

resource "google_secret_manager_secret_version" "github_token" {
  secret      = google_secret_manager_secret.github_token.id
  secret_data = var.github_token
}

# Grant Cloud Functions access to secrets (least privilege)
resource "google_secret_manager_secret_iam_member" "nhs_trud_access" {
  secret_id = google_secret_manager_secret.nhs_trud_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.kb_functions.email}"
}

resource "google_secret_manager_secret_iam_member" "umls_access" {
  secret_id = google_secret_manager_secret.umls_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.kb_functions.email}"
}

resource "google_secret_manager_secret_iam_member" "loinc_access" {
  secret_id = google_secret_manager_secret.loinc_credentials.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.kb_functions.email}"
}

resource "google_secret_manager_secret_iam_member" "github_access" {
  secret_id = google_secret_manager_secret.github_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.kb_functions.email}"
}

# Output secret resource names
output "secret_ids" {
  description = "Secret Manager secret IDs"
  value = {
    nhs_trud        = google_secret_manager_secret.nhs_trud_api_key.secret_id
    umls            = google_secret_manager_secret.umls_api_key.secret_id
    loinc           = google_secret_manager_secret.loinc_credentials.secret_id
    github_token    = google_secret_manager_secret.github_token.secret_id
  }
}
