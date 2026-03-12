# Cloud Monitoring - Alerts and Log-Based Metrics

# Notification Channel - Email
resource "google_monitoring_notification_channel" "email" {
  display_name = "KB-7 Email Notifications"
  type         = "email"

  labels = {
    email_address = var.notification_email
  }

  enabled = true
}

# Notification Channel - Slack (optional)
resource "google_monitoring_notification_channel" "slack" {
  count        = var.slack_webhook_url != "" ? 1 : 0
  display_name = "KB-7 Slack Notifications"
  type         = "slack"

  labels = {
    channel_name = "#kb7-alerts"
  }

  sensitive_labels {
    auth_token = var.slack_webhook_url
  }

  enabled = true
}

# Alert Policy - Function Duration Warning
resource "google_monitoring_alert_policy" "function_duration" {
  display_name = "KB-7 Function Duration Alert - ${var.environment}"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Function Duration > 50 minutes"

    condition_threshold {
      filter          = "resource.type=\"cloud_function\" AND resource.labels.function_name:\"kb7\" AND metric.type=\"cloudfunctions.googleapis.com/function/execution_times\""
      duration        = "60s"
      comparison      = "COMPARISON_GT"
      threshold_value = 3000000 # 50 minutes in milliseconds

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_PERCENTILE_99"
      }
    }
  }

  notification_channels = concat(
    [google_monitoring_notification_channel.email.id],
    var.slack_webhook_url != "" ? [google_monitoring_notification_channel.slack[0].id] : []
  )

  alert_strategy {
    auto_close = "3600s" # Auto-close after 1 hour
  }

  documentation {
    content   = "Cloud Function execution time exceeded 50 minutes. Function may timeout at 60 minutes. Check function logs for performance issues."
    mime_type = "text/markdown"
  }
}

# Alert Policy - Function Failures
resource "google_monitoring_alert_policy" "function_errors" {
  display_name = "KB-7 Function Error Rate - ${var.environment}"
  combiner     = "OR"

  conditions {
    display_name = "High function error rate"

    condition_threshold {
      filter          = "resource.type=\"cloud_function\" AND resource.labels.function_name:\"kb7\" AND metric.type=\"cloudfunctions.googleapis.com/function/execution_count\" AND metric.labels.status=\"error\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 1 # Any error triggers alert

      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
        group_by_fields      = ["resource.function_name"]
      }
    }
  }

  notification_channels = concat(
    [google_monitoring_notification_channel.email.id],
    var.slack_webhook_url != "" ? [google_monitoring_notification_channel.slack[0].id] : []
  )

  alert_strategy {
    auto_close = "1800s"
  }

  documentation {
    content   = "Cloud Function errors detected. Check function logs for error details and stack traces."
    mime_type = "text/markdown"
  }
}

# Alert Policy - Workflow Failures
resource "google_monitoring_alert_policy" "workflow_failures" {
  display_name = "KB-7 Workflow Failure - ${var.environment}"
  combiner     = "OR"

  conditions {
    display_name = "Workflow execution failed"

    condition_threshold {
      filter          = "resource.type=\"workflows.googleapis.com/Workflow\" AND resource.labels.workflow_id=\"${google_workflows_workflow.kb_factory.name}\" AND metric.type=\"workflows.googleapis.com/finished_execution_count\" AND metric.labels.status=\"FAILED\""
      duration        = "60s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_RATE"
      }
    }
  }

  notification_channels = concat(
    [google_monitoring_notification_channel.email.id],
    var.slack_webhook_url != "" ? [google_monitoring_notification_channel.slack[0].id] : []
  )

  alert_strategy {
    auto_close = "3600s"
  }

  documentation {
    content   = "KB-7 Knowledge Factory workflow failed. Check workflow execution logs and function status."
    mime_type = "text/markdown"
  }
}

# Log-Based Metric - Download Success
resource "google_logging_metric" "download_success" {
  name   = "kb7_download_success_count_${var.environment}"
  filter = "resource.type=\"cloud_function\" AND jsonPayload.message=\"Download complete\" AND jsonPayload.status=\"success\""

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"

    labels {
      key         = "terminology"
      value_type  = "STRING"
      description = "Terminology type (snomed/rxnorm/loinc)"
    }

    labels {
      key         = "version"
      value_type  = "STRING"
      description = "Downloaded version"
    }
  }

  label_extractors = {
    "terminology" = "EXTRACT(jsonPayload.terminology)"
    "version"     = "EXTRACT(jsonPayload.version)"
  }
}

# Log-Based Metric - Download Failures
resource "google_logging_metric" "download_failures" {
  name   = "kb7_download_failure_count_${var.environment}"
  filter = "resource.type=\"cloud_function\" AND (severity=ERROR OR jsonPayload.status=\"failed\")"

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"

    labels {
      key         = "function_name"
      value_type  = "STRING"
      description = "Name of the function that failed"
    }

    labels {
      key         = "error_type"
      value_type  = "STRING"
      description = "Type of error encountered"
    }
  }

  label_extractors = {
    "function_name" = "EXTRACT(resource.labels.function_name)"
    "error_type"    = "EXTRACT(jsonPayload.error_type)"
  }
}

# Log-Based Metric - File Size Tracking
resource "google_logging_metric" "file_size" {
  name   = "kb7_download_file_size_${var.environment}"
  filter = "resource.type=\"cloud_function\" AND jsonPayload.file_size_bytes>0"

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"

    labels {
      key         = "terminology"
      value_type  = "STRING"
      description = "Terminology type"
    }
  }

  label_extractors = {
    "terminology" = "EXTRACT(jsonPayload.terminology)"
  }
}

# Outputs
output "notification_channels" {
  description = "Notification channel IDs"
  sensitive   = true
  value = {
    email = google_monitoring_notification_channel.email.id
    slack = var.slack_webhook_url != "" ? google_monitoring_notification_channel.slack[0].id : null
  }
}

output "alert_policies" {
  description = "Alert policy names"
  value = {
    function_duration = google_monitoring_alert_policy.function_duration.name
    function_errors   = google_monitoring_alert_policy.function_errors.name
    workflow_failures = google_monitoring_alert_policy.workflow_failures.name
  }
}
