# ══════════════════════════════════════════════════════════════════
# KB-7 Knowledge Factory - GCP Configuration
# ══════════════════════════════════════════════════════════════════

# GCP Project Configuration
project_id  = "sincere-hybrid-477206-h2"
region      = "us-central1"
environment = "production"

# External API Credentials (PLACEHOLDER - Update when you get them)
nhs_trud_api_key = "PLACEHOLDER-NHS-TRUD-API-KEY"
umls_api_key     = "PLACEHOLDER-UMLS-API-KEY"
loinc_username   = "placeholder-loinc-user"
loinc_password   = "placeholder-loinc-pass"

# GitHub Configuration
github_token      = "ghp_placeholder-github-token"
github_repository = "your-org/knowledge-factory"

# Notifications
slack_webhook_url  = ""
notification_email = "onkarshahi@vaidshala.com"

# Schedule
schedule_cron     = "0 2 1 * *"
schedule_timezone = "UTC"
