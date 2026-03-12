# Cloud Scheduler - Monthly Terminology Update Trigger

resource "google_cloud_scheduler_job" "monthly_terminology_update" {
  name             = "kb7-monthly-terminology-update-${var.environment}"
  description      = "Triggers KB-7 Knowledge Factory workflow monthly"
  schedule         = var.schedule_cron
  time_zone        = var.schedule_timezone
  attempt_deadline = "1800s" # 30 minutes max for trigger request

  retry_config {
    retry_count          = 3
    min_backoff_duration = "300s"  # 5 minutes
    max_backoff_duration = "3600s" # 1 hour
    max_retry_duration   = "7200s" # 2 hours total
    max_doublings        = 3
  }

  http_target {
    http_method = "POST"
    uri         = "https://workflowexecutions.googleapis.com/v1/projects/${var.project_id}/locations/${var.region}/workflows/${google_workflows_workflow.kb_factory.name}/executions"

    body = base64encode(jsonencode({
      argument = jsonencode({
        trigger    = "scheduled"
        timestamp  = "$${CURRENT_TIMESTAMP}"
        cron       = var.schedule_cron
        environment = var.environment
      })
    }))

    headers = {
      "Content-Type" = "application/json"
    }

    oauth_token {
      service_account_email = google_service_account.scheduler.email
      scope                 = "https://www.googleapis.com/auth/cloud-platform"
    }
  }

  depends_on = [
    google_workflows_workflow.kb_factory,
    google_service_account.scheduler,
  ]
}

# Output scheduler information
output "scheduler_job_name" {
  description = "Name of the Cloud Scheduler job"
  value       = google_cloud_scheduler_job.monthly_terminology_update.name
}

output "scheduler_schedule" {
  description = "Cron schedule for the scheduler job"
  value       = google_cloud_scheduler_job.monthly_terminology_update.schedule
}
