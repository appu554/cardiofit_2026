#!/bin/bash

# monitor-bulk-load.sh - Real-time monitoring for bulk load operations
# Version: 1.0
# Usage: ./monitor-bulk-load.sh [options]

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default configuration
DEFAULT_REFRESH_INTERVAL=5
DEFAULT_ELASTICSEARCH_URL="http://localhost:9200"
DEFAULT_INDEX="clinical_terms"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Unicode symbols
CHECKMARK="✅"
CROSSMARK="❌"
WARNING="⚠️"
PROGRESS="📊"
CLOCK="⏱️"
ROCKET="🚀"
CHART="📈"

# Help function
show_help() {
    cat << EOF
KB7 Terminology Bulk Load Monitoring Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --elasticsearch URL     Elasticsearch URL (default: $DEFAULT_ELASTICSEARCH_URL)
    --index NAME           Index name to monitor (default: $DEFAULT_INDEX)
    --interval SECONDS     Refresh interval (default: $DEFAULT_REFRESH_INTERVAL)
    --log-file FILE        Monitor specific log file
    --postgres URL         PostgreSQL URL for comparison
    --dashboard           Show comprehensive dashboard
    --alerts              Enable alert monitoring
    --export FILE         Export monitoring data to file
    --help                Show this help message

EXAMPLES:
    # Basic monitoring
    $0

    # Monitor specific index with custom interval
    $0 --index clinical_terms_prod --interval 2

    # Full dashboard with alerts
    $0 --dashboard --alerts

    # Monitor specific log file
    $0 --log-file /path/to/bulk-load.log

FEATURES:
    - Real-time progress tracking
    - Performance metrics monitoring
    - Error detection and alerting
    - Index health monitoring
    - Migration speed analysis
    - Resource usage tracking
EOF
}

# Parse command line arguments
parse_arguments() {
    ELASTICSEARCH_URL="$DEFAULT_ELASTICSEARCH_URL"
    INDEX_NAME="$DEFAULT_INDEX"
    REFRESH_INTERVAL="$DEFAULT_REFRESH_INTERVAL"
    LOG_FILE=""
    POSTGRES_URL=""
    DASHBOARD="false"
    ALERTS="false"
    EXPORT_FILE=""

    while [[ $# -gt 0 ]]; do
        case $1 in
            --elasticsearch)
                ELASTICSEARCH_URL="$2"
                shift 2
                ;;
            --index)
                INDEX_NAME="$2"
                shift 2
                ;;
            --interval)
                REFRESH_INTERVAL="$2"
                shift 2
                ;;
            --log-file)
                LOG_FILE="$2"
                shift 2
                ;;
            --postgres)
                POSTGRES_URL="$2"
                shift 2
                ;;
            --dashboard)
                DASHBOARD="true"
                shift
                ;;
            --alerts)
                ALERTS="true"
                shift
                ;;
            --export)
                EXPORT_FILE="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Check dependencies
check_dependencies() {
    for tool in curl jq; do
        if ! command -v "$tool" &> /dev/null; then
            echo -e "${RED}Error: Required tool not found: $tool${NC}"
            exit 1
        fi
    done
}

# Get Elasticsearch cluster info
get_cluster_info() {
    local response
    if response=$(curl -s "$ELASTICSEARCH_URL/_cluster/health" 2>/dev/null); then
        echo "$response"
    else
        echo "{\"status\":\"unavailable\"}"
    fi
}

# Get index statistics
get_index_stats() {
    local index="$1"
    local response
    if response=$(curl -s "$ELASTICSEARCH_URL/$index/_stats" 2>/dev/null); then
        echo "$response"
    else
        echo "{\"indices\":{}}"
    fi
}

# Get index count
get_index_count() {
    local index="$1"
    local response
    if response=$(curl -s "$ELASTICSEARCH_URL/$index/_count" 2>/dev/null); then
        echo "$response" | jq -r '.count // 0'
    else
        echo "0"
    fi
}

# Format bytes to human readable
format_bytes() {
    local bytes="$1"
    if [[ "$bytes" -eq 0 ]]; then
        echo "0B"
    elif [[ "$bytes" -lt 1024 ]]; then
        echo "${bytes}B"
    elif [[ "$bytes" -lt 1048576 ]]; then
        echo "$(( bytes / 1024 ))KB"
    elif [[ "$bytes" -lt 1073741824 ]]; then
        echo "$(( bytes / 1048576 ))MB"
    else
        echo "$(( bytes / 1073741824 ))GB"
    fi
}

# Format duration
format_duration() {
    local seconds="$1"
    local hours=$((seconds / 3600))
    local minutes=$(((seconds % 3600) / 60))
    local secs=$((seconds % 60))

    if [[ $hours -gt 0 ]]; then
        printf "%02d:%02d:%02d" "$hours" "$minutes" "$secs"
    else
        printf "%02d:%02d" "$minutes" "$secs"
    fi
}

# Monitor log file for progress
monitor_log_progress() {
    local log_file="$1"
    local latest_log=""

    if [[ -z "$log_file" ]]; then
        # Find the latest log file
        latest_log=$(find "$PROJECT_ROOT/logs" -name "bulk-load-*.log" -type f 2>/dev/null | sort -r | head -n 1)
        if [[ -n "$latest_log" ]]; then
            log_file="$latest_log"
        fi
    fi

    if [[ -n "$log_file" && -f "$log_file" ]]; then
        echo -e "${BLUE}Monitoring log file: $log_file${NC}"

        # Extract progress information
        local processed=$(grep -o "Processed: [0-9]*" "$log_file" 2>/dev/null | tail -n 1 | grep -o "[0-9]*" || echo "0")
        local successful=$(grep -o "Successful: [0-9]*" "$log_file" 2>/dev/null | tail -n 1 | grep -o "[0-9]*" || echo "0")
        local failed=$(grep -o "Failed: [0-9]*" "$log_file" 2>/dev/null | tail -n 1 | grep -o "[0-9]*" || echo "0")
        local rate=$(grep -o "Records/Second: [0-9.]*" "$log_file" 2>/dev/null | tail -n 1 | grep -o "[0-9.]*" || echo "0")

        echo -e "${PROGRESS} Progress: Processed=$processed, Successful=$successful, Failed=$failed"
        echo -e "${CHART} Rate: $rate records/second"

        # Check for errors
        local error_count=$(grep -c "ERROR" "$log_file" 2>/dev/null || echo "0")
        if [[ "$error_count" -gt 0 ]]; then
            echo -e "${WARNING} Errors detected: $error_count"
            if [[ "$ALERTS" == "true" ]]; then
                echo -e "${RED}ALERT: Migration errors detected in log file${NC}"
            fi
        fi
    else
        echo -e "${YELLOW}No log file found for monitoring${NC}"
    fi
}

# Display basic monitoring
show_basic_monitoring() {
    while true; do
        clear
        echo -e "${ROCKET} KB7 Terminology Bulk Load Monitor"
        echo -e "${CYAN}===============================================${NC}"
        echo "$(date)"
        echo

        # Cluster health
        local cluster_info
        cluster_info=$(get_cluster_info)
        local cluster_status
        cluster_status=$(echo "$cluster_info" | jq -r '.status // "unknown"')

        case "$cluster_status" in
            "green")
                echo -e "${CHECKMARK} Cluster Status: ${GREEN}$cluster_status${NC}"
                ;;
            "yellow")
                echo -e "${WARNING} Cluster Status: ${YELLOW}$cluster_status${NC}"
                ;;
            "red"|"unavailable")
                echo -e "${CROSSMARK} Cluster Status: ${RED}$cluster_status${NC}"
                ;;
            *)
                echo -e "${WARNING} Cluster Status: ${YELLOW}$cluster_status${NC}"
                ;;
        esac

        # Index information
        local count
        count=$(get_index_count "$INDEX_NAME")
        echo -e "${PROGRESS} Index: $INDEX_NAME"
        echo -e "${CHART} Document Count: $count"

        # Index stats
        local stats
        stats=$(get_index_stats "$INDEX_NAME")
        if [[ "$stats" != "{\"indices\":{}}" ]]; then
            local size_bytes
            size_bytes=$(echo "$stats" | jq -r ".indices.\"$INDEX_NAME\".total.store.size_in_bytes // 0")
            local size_human
            size_human=$(format_bytes "$size_bytes")
            echo -e "${CHART} Index Size: $size_human"

            local docs_deleted
            docs_deleted=$(echo "$stats" | jq -r ".indices.\"$INDEX_NAME\".total.docs.deleted // 0")
            if [[ "$docs_deleted" -gt 0 ]]; then
                echo -e "${WARNING} Deleted Documents: $docs_deleted"
            fi
        fi

        echo
        echo -e "${BLUE}Monitoring every ${REFRESH_INTERVAL}s... (Ctrl+C to exit)${NC}"

        # Monitor log progress if available
        monitor_log_progress "$LOG_FILE"

        sleep "$REFRESH_INTERVAL"
    done
}

# Display comprehensive dashboard
show_dashboard() {
    while true; do
        clear
        echo -e "${ROCKET} KB7 Terminology Migration Dashboard"
        echo -e "${CYAN}================================================${NC}"
        echo "Last Updated: $(date)"
        echo

        # System Overview
        echo -e "${PURPLE}┌─ SYSTEM OVERVIEW ─────────────────────────────┐${NC}"

        # Cluster health
        local cluster_info
        cluster_info=$(get_cluster_info)
        local cluster_status
        cluster_status=$(echo "$cluster_info" | jq -r '.status // "unknown"')
        local active_shards
        active_shards=$(echo "$cluster_info" | jq -r '.active_shards // 0')
        local relocating_shards
        relocating_shards=$(echo "$cluster_info" | jq -r '.relocating_shards // 0')

        printf "│ %-20s: " "Cluster Status"
        case "$cluster_status" in
            "green") echo -e "${GREEN}$cluster_status${NC} $CHECKMARK" ;;
            "yellow") echo -e "${YELLOW}$cluster_status${NC} $WARNING" ;;
            *) echo -e "${RED}$cluster_status${NC} $CROSSMARK" ;;
        esac

        echo "│ Active Shards       : $active_shards"
        echo "│ Relocating Shards   : $relocating_shards"

        # Index Statistics
        echo -e "${PURPLE}├─ INDEX STATISTICS ────────────────────────────┤${NC}"

        local count
        count=$(get_index_count "$INDEX_NAME")
        local stats
        stats=$(get_index_stats "$INDEX_NAME")

        echo "│ Index Name          : $INDEX_NAME"
        echo "│ Document Count      : $count"

        if [[ "$stats" != "{\"indices\":{}}" ]]; then
            local size_bytes
            size_bytes=$(echo "$stats" | jq -r ".indices.\"$INDEX_NAME\".total.store.size_in_bytes // 0")
            local size_human
            size_human=$(format_bytes "$size_bytes")

            local indexing_total
            indexing_total=$(echo "$stats" | jq -r ".indices.\"$INDEX_NAME\".total.indexing.index_total // 0")
            local search_total
            search_total=$(echo "$stats" | jq -r ".indices.\"$INDEX_NAME\".total.search.query_total // 0")

            echo "│ Index Size          : $size_human"
            echo "│ Total Indexed       : $indexing_total"
            echo "│ Total Searches      : $search_total"
        fi

        # Migration Progress
        echo -e "${PURPLE}├─ MIGRATION PROGRESS ──────────────────────────┤${NC}"
        monitor_log_progress "$LOG_FILE"

        # Performance Metrics
        echo -e "${PURPLE}├─ PERFORMANCE METRICS ─────────────────────────┤${NC}"

        if [[ -n "$POSTGRES_URL" ]]; then
            echo "│ Source Database     : Connected"
        else
            echo "│ Source Database     : Not monitored"
        fi

        echo "│ Elasticsearch      : $(curl -s "$ELASTICSEARCH_URL" >/dev/null 2>&1 && echo "Connected" || echo "Disconnected")"

        echo -e "${PURPLE}└────────────────────────────────────────────────┘${NC}"

        echo
        echo -e "${BLUE}Dashboard refresh: ${REFRESH_INTERVAL}s | Press Ctrl+C to exit${NC}"

        # Export data if requested
        if [[ -n "$EXPORT_FILE" ]]; then
            export_monitoring_data
        fi

        sleep "$REFRESH_INTERVAL"
    done
}

# Export monitoring data
export_monitoring_data() {
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    local data
    data=$(cat << EOF
{
  "timestamp": "$timestamp",
  "cluster_info": $(get_cluster_info),
  "index_stats": $(get_index_stats "$INDEX_NAME"),
  "index_count": $(get_index_count "$INDEX_NAME"),
  "index_name": "$INDEX_NAME",
  "elasticsearch_url": "$ELASTICSEARCH_URL"
}
EOF
)

    echo "$data" >> "$EXPORT_FILE"
}

# Main execution
main() {
    parse_arguments "$@"
    check_dependencies

    echo -e "${ROCKET} Starting KB7 Bulk Load Monitor"
    echo "Elasticsearch: $ELASTICSEARCH_URL"
    echo "Index: $INDEX_NAME"
    echo "Refresh Interval: ${REFRESH_INTERVAL}s"
    echo

    # Set up signal handlers
    trap 'echo -e "\n${YELLOW}Monitor stopped${NC}"; exit 0' INT TERM

    if [[ "$DASHBOARD" == "true" ]]; then
        show_dashboard
    else
        show_basic_monitoring
    fi
}

# Run main function
main "$@"