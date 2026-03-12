#!/bin/bash
################################################################################
# KB-7 Slack Notification Script
# Purpose: Send formatted notifications to Slack webhook
# Usage: ./notify-slack.sh <status> <message> [--details "additional info"]
################################################################################

set -euo pipefail

SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"
SLACK_CHANNEL="${SLACK_CHANNEL:-#kb7-automation}"
SLACK_USERNAME="${SLACK_USERNAME:-KB-7 Bot}"

# Status emoji mapping
get_emoji() {
    case $1 in
        success) echo "✅" ;;
        failure) echo "❌" ;;
        warning) echo "⚠️" ;;
        info) echo "ℹ️" ;;
        deployment) echo "🚀" ;;
        rollback) echo "🔄" ;;
        validation) echo "🔍" ;;
        *) echo "•" ;;
    esac
}

# Color mapping for Slack attachments
get_color() {
    case $1 in
        success) echo "good" ;;
        failure) echo "danger" ;;
        warning) echo "warning" ;;
        *) echo "#808080" ;;
    esac
}

send_simple_notification() {
    local status=$1
    local message=$2
    local emoji=$(get_emoji "$status")

    if [ -z "$SLACK_WEBHOOK" ]; then
        echo "ERROR: SLACK_WEBHOOK environment variable not set"
        exit 1
    fi

    curl -X POST "$SLACK_WEBHOOK" \
        -H 'Content-Type: application/json' \
        -d "{
            \"channel\": \"$SLACK_CHANNEL\",
            \"username\": \"$SLACK_USERNAME\",
            \"text\": \"$emoji $message\"
        }" \
        --silent --show-error

    echo "Notification sent: $emoji $message"
}

send_detailed_notification() {
    local status=$1
    local title=$2
    local details=$3
    local emoji=$(get_emoji "$status")
    local color=$(get_color "$status")

    if [ -z "$SLACK_WEBHOOK" ]; then
        echo "ERROR: SLACK_WEBHOOK environment variable not set"
        exit 1
    fi

    # Get hostname for footer
    local hostname=$(hostname)
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S %Z')

    curl -X POST "$SLACK_WEBHOOK" \
        -H 'Content-Type: application/json' \
        -d "{
            \"channel\": \"$SLACK_CHANNEL\",
            \"username\": \"$SLACK_USERNAME\",
            \"attachments\": [
                {
                    \"color\": \"$color\",
                    \"title\": \"$emoji $title\",
                    \"text\": \"$details\",
                    \"footer\": \"KB-7 Automation | $hostname\",
                    \"ts\": $(date +%s)
                }
            ]
        }" \
        --silent --show-error

    echo "Detailed notification sent: $title"
}

send_deployment_notification() {
    local status=$1
    local version=$2
    local concept_count=$3
    local duration=$4

    local emoji=$(get_emoji "$status")
    local color=$(get_color "$status")

    local title
    if [ "$status" = "success" ]; then
        title="Kernel Deployment Successful"
    else
        title="Kernel Deployment Failed"
    fi

    curl -X POST "$SLACK_WEBHOOK" \
        -H 'Content-Type: application/json' \
        -d "{
            \"channel\": \"$SLACK_CHANNEL\",
            \"username\": \"$SLACK_USERNAME\",
            \"attachments\": [
                {
                    \"color\": \"$color\",
                    \"title\": \"$emoji $title\",
                    \"fields\": [
                        {
                            \"title\": \"Version\",
                            \"value\": \"$version\",
                            \"short\": true
                        },
                        {
                            \"title\": \"Concept Count\",
                            \"value\": \"$(printf "%'d" $concept_count)\",
                            \"short\": true
                        },
                        {
                            \"title\": \"Duration\",
                            \"value\": \"${duration}s\",
                            \"short\": true
                        },
                        {
                            \"title\": \"Environment\",
                            \"value\": \"${ENVIRONMENT:-production}\",
                            \"short\": true
                        }
                    ],
                    \"footer\": \"KB-7 Kernel Deployment\",
                    \"ts\": $(date +%s)
                }
            ]
        }" \
        --silent --show-error

    echo "Deployment notification sent: $title (v$version)"
}

send_validation_notification() {
    local status=$1
    local validation_results=$2

    local emoji=$(get_emoji "validation")
    local color=$(get_color "$status")

    local title
    if [ "$status" = "success" ]; then
        title="✓ All Validation Checks Passed"
    else
        title="✗ Validation Checks Failed"
    fi

    curl -X POST "$SLACK_WEBHOOK" \
        -H 'Content-Type: application/json' \
        -d "{
            \"channel\": \"$SLACK_CHANNEL\",
            \"username\": \"$SLACK_USERNAME\",
            \"attachments\": [
                {
                    \"color\": \"$color\",
                    \"title\": \"$emoji $title\",
                    \"text\": \"$validation_results\",
                    \"footer\": \"KB-7 Quality Gates\",
                    \"ts\": $(date +%s)
                }
            ]
        }" \
        --silent --show-error

    echo "Validation notification sent: $title"
}

send_retry_notification() {
    local attempt=$1
    local max_attempts=$2
    local error_message=$3

    local emoji=$(get_emoji "warning")

    curl -X POST "$SLACK_WEBHOOK" \
        -H 'Content-Type: application/json' \
        -d "{
            \"channel\": \"$SLACK_CHANNEL\",
            \"username\": \"$SLACK_USERNAME\",
            \"attachments\": [
                {
                    \"color\": \"warning\",
                    \"title\": \"$emoji Deployment Retry Attempt $attempt/$max_attempts\",
                    \"text\": \"Previous attempt failed. Retrying...\n\nError: $error_message\",
                    \"footer\": \"KB-7 Automation\",
                    \"ts\": $(date +%s)
                }
            ]
        }" \
        --silent --show-error

    echo "Retry notification sent: Attempt $attempt/$max_attempts"
}

# Example notification templates
show_examples() {
    cat <<EOF
Usage: $0 <command> [arguments]

Commands:
  simple <status> <message>
      Send a simple text notification
      Example: $0 simple success "Deployment completed"

  detailed <status> <title> <details>
      Send a detailed notification with formatting
      Example: $0 detailed success "Build Complete" "All tests passed"

  deployment <status> <version> <concept_count> <duration>
      Send a deployment notification with metrics
      Example: $0 deployment success 20250124 523451 180

  validation <status> <results>
      Send validation results
      Example: $0 validation success "Concept count: PASS\nOrphans: PASS"

  retry <attempt> <max_attempts> <error_message>
      Send retry attempt notification
      Example: $0 retry 2 3 "GraphDB connection timeout"

Status types: success, failure, warning, info, deployment, rollback, validation

Examples of complete messages:

  # Successful deployment
  $0 deployment success 20250124 523451 180

  # Failed validation
  $0 validation failure "Concept count: FAIL (expected: >500k, got: 450k)"

  # Warning about retry
  $0 retry 2 3 "S3 download timeout"

Environment Variables:
  SLACK_WEBHOOK     - Slack webhook URL (required)
  SLACK_CHANNEL     - Channel to post to (default: #kb7-automation)
  SLACK_USERNAME    - Bot username (default: KB-7 Bot)
  ENVIRONMENT       - Deployment environment (default: production)
EOF
}

# Main script logic
main() {
    if [ $# -lt 1 ]; then
        show_examples
        exit 1
    fi

    local command=$1
    shift

    case $command in
        simple)
            if [ $# -lt 2 ]; then
                echo "Usage: $0 simple <status> <message>"
                exit 1
            fi
            send_simple_notification "$1" "$2"
            ;;
        detailed)
            if [ $# -lt 3 ]; then
                echo "Usage: $0 detailed <status> <title> <details>"
                exit 1
            fi
            send_detailed_notification "$1" "$2" "$3"
            ;;
        deployment)
            if [ $# -lt 4 ]; then
                echo "Usage: $0 deployment <status> <version> <concept_count> <duration>"
                exit 1
            fi
            send_deployment_notification "$1" "$2" "$3" "$4"
            ;;
        validation)
            if [ $# -lt 2 ]; then
                echo "Usage: $0 validation <status> <results>"
                exit 1
            fi
            send_validation_notification "$1" "$2"
            ;;
        retry)
            if [ $# -lt 3 ]; then
                echo "Usage: $0 retry <attempt> <max_attempts> <error_message>"
                exit 1
            fi
            send_retry_notification "$1" "$2" "$3"
            ;;
        examples|help|--help|-h)
            show_examples
            ;;
        *)
            echo "Unknown command: $command"
            show_examples
            exit 1
            ;;
    esac
}

main "$@"
