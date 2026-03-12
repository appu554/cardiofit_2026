# Cloud Workflows Orchestration

resource "google_workflows_workflow" "kb_factory" {
  name            = "kb7-factory-workflow-${var.environment}"
  region          = var.region
  description     = "Orchestrates parallel terminology downloads and GitHub workflow dispatch"
  service_account = google_service_account.kb_workflows.email

  source_contents = templatefile("${path.module}/../workflows/kb-factory-workflow.yaml", {
    project_id        = var.project_id
    region            = var.region
    environment       = var.environment
    snomed_function   = google_cloudfunctions2_function.snomed_downloader.name
    rxnorm_function   = google_cloudfunctions2_function.rxnorm_downloader.name
    loinc_function    = google_cloudfunctions2_function.loinc_downloader.name
    github_function   = google_cloudfunctions2_function.github_dispatcher.name
  })

  labels = {
    service     = "kb7-knowledge-factory"
    environment = var.environment
    managed-by  = "terraform"
  }

  depends_on = [
    google_cloudfunctions2_function.snomed_downloader,
    google_cloudfunctions2_function.rxnorm_downloader,
    google_cloudfunctions2_function.loinc_downloader,
    google_cloudfunctions2_function.github_dispatcher,
  ]
}

# Output workflow information
output "workflow_id" {
  description = "ID of the Cloud Workflows workflow"
  value       = google_workflows_workflow.kb_factory.id
}

output "workflow_name" {
  description = "Name of the Cloud Workflows workflow"
  value       = google_workflows_workflow.kb_factory.name
}
