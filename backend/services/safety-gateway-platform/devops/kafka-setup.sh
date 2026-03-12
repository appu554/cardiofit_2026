#!/bin/bash

# Kafka Integration Setup Script for Safety Gateway Platform
# This script sets up Kafka topics, schemas, and streaming infrastructure
# Updated for Phase 4: Enhanced Features with Learning Gateway Integration

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/safety-gateway/kafka-setup.log"
KAFKA_BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"
ENVIRONMENT="${ENVIRONMENT:-development}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging function
log() {
    local level=$1
    shift
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*" | tee -a "$LOG_FILE"
}

log_info() { log "${BLUE}INFO${NC}" "$@"; }
log_warn() { log "${YELLOW}WARN${NC}" "$@"; }
log_error() { log "${RED}ERROR${NC}" "$@"; }
log_success() { log "${GREEN}SUCCESS${NC}" "$@"; }

# Error handling
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        log_error "Kafka setup failed with exit code $exit_code"
    fi
    exit $exit_code
}

trap cleanup EXIT

# Prerequisites check
check_prerequisites() {
    log_info "Checking Kafka setup prerequisites..."
    
    # Check if Kafka CLI tools are available
    for tool in kafka-topics kafka-console-producer kafka-console-consumer kafka-configs; do
        if ! command -v "$tool" &> /dev/null; then
            # Try kafka installation directory
            if [ -d "/opt/kafka" ] && [ -f "/opt/kafka/bin/$tool.sh" ]; then
                alias "$tool"="/opt/kafka/bin/$tool.sh"
            else
                log_error "$tool is required but not found. Install Kafka CLI tools."
                exit 1
            fi
        fi
    done
    
    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    log_success "Prerequisites check passed"
}

# Test Kafka connectivity
test_kafka_connection() {
    log_info "Testing Kafka connection..."
    
    # Test bootstrap servers connectivity
    if ! kafka-topics --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" --list > /dev/null 2>&1; then
        log_error "Cannot connect to Kafka bootstrap servers: $KAFKA_BOOTSTRAP_SERVERS"
        exit 1
    fi
    
    log_success "Kafka connection established"
}

# Topic configuration based on environment and safety requirements
get_topic_config() {
    local topic_name=$1
    local base_config="--config cleanup.policy=delete --config compression.type=snappy"
    
    case "$ENVIRONMENT" in
        "production")
            echo "$base_config --config retention.ms=2208988800000 --config min.insync.replicas=2 --config unclean.leader.election.enable=false"
            ;;
        "staging"|"canary")
            echo "$base_config --config retention.ms=604800000 --config min.insync.replicas=1"
            ;;
        *)
            echo "$base_config --config retention.ms=86400000"
            ;;
    esac
}

# Create Kafka topics for safety system
create_safety_topics() {
    log_info "Creating safety system Kafka topics..."
    
    # Topic definitions with partitions and replication factor
    declare -A topics=(
        # Existing topics
        ["safety-events-raw"]="6:3"
        ["safety-events-processed"]="6:3"
        ["safety-alerts-critical"]="3:3"
        ["safety-override-tokens"]="3:3"
        ["safety-metrics-stream"]="12:2"
        ["safety-audit-events"]="3:3"
        ["safety-engine-heartbeats"]="3:2"
        ["safety-decision-feedback"]="3:2"
        
        # Phase 4: Learning Gateway Topics
        ["clinical-learning-safety-decisions"]="6:3"
        ["clinical-learning-clinical-overrides"]="3:3"
        ["clinical-learning-clinical-outcomes"]="3:3"
        ["clinical-learning-performance-analysis"]="6:2"
        ["clinical-learning-override-patterns"]="3:2"
        ["clinical-learning-reproducibility-events"]="3:3"
        ["clinical-learning-correlation-analysis"]="6:2"
        ["clinical-learning-risk-predictions"]="3:2"
    )
    
    for topic in "${!topics[@]}"; do
        IFS=':' read -r partitions replication <<< "${topics[$topic]}"
        
        # Adjust replication factor for single-node development
        if [ "$ENVIRONMENT" = "development" ]; then
            replication=1
        fi
        
        local topic_config=$(get_topic_config "$topic")
        
        log_info "Creating topic: $topic (partitions=$partitions, replication=$replication)"
        
        if kafka-topics --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
                       --create \
                       --topic "$topic" \
                       --partitions "$partitions" \
                       --replication-factor "$replication" \
                       $topic_config \
                       --if-not-exists; then
            log_success "Topic created: $topic"
        else
            log_error "Failed to create topic: $topic"
            exit 1
        fi
    done
    
    # Special configuration for critical topics
    log_info "Applying special configuration for critical topics..."
    
    # Safety alerts - immediate processing with no delay
    kafka-configs --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
                  --entity-type topics \
                  --entity-name "safety-alerts-critical" \
                  --alter \
                  --add-config "min.insync.replicas=2,unclean.leader.election.enable=false,max.message.bytes=1048576"
    
    # Audit events - long retention for compliance
    kafka-configs --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
                  --entity-type topics \
                  --entity-name "safety-audit-events" \
                  --alter \
                  --add-config "retention.ms=220898880000,segment.ms=604800000"  # 7 years retention
    
    # Learning outcomes - extended retention for analysis
    kafka-configs --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
                  --entity-type topics \
                  --entity-name "clinical-learning-clinical-outcomes" \
                  --alter \
                  --add-config "retention.ms=15768000000,segment.ms=604800000"  # 6 months retention
    
    # Reproducibility events - long retention for compliance
    kafka-configs --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
                  --entity-type topics \
                  --entity-name "clinical-learning-reproducibility-events" \
                  --alter \
                  --add-config "retention.ms=94608000000,segment.ms=604800000"  # 3 years retention
    
    log_success "Safety topics created and configured"
}

# Create Avro schemas for type safety
create_avro_schemas() {
    log_info "Creating Avro schemas for safety events..."
    
    # Create schemas directory
    mkdir -p "$SCRIPT_DIR/schemas"
    
    # Safety Event Schema
    cat > "$SCRIPT_DIR/schemas/safety-event.avsc" << 'EOF'
{
    "type": "record",
    "name": "SafetyEvent",
    "namespace": "com.cardiofit.safety",
    "fields": [
        {"name": "eventId", "type": "string"},
        {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
        {"name": "requestId", "type": "string"},
        {"name": "patientId", "type": "string"},
        {"name": "userId", "type": ["null", "string"], "default": null},
        {"name": "safetyTier", "type": {"type": "enum", "name": "SafetyTier", "symbols": ["VETO_CRITICAL", "ADVISORY", "INFORMATIONAL"]}},
        {"name": "decision", "type": {"type": "enum", "name": "Decision", "symbols": ["SAFE", "UNSAFE", "WARNING", "BLOCKED"]}},
        {"name": "enginesEvaluated", "type": {"type": "array", "items": "string"}},
        {"name": "responseTimeMs", "type": "int"},
        {"name": "overrideUsed", "type": "boolean", "default": false},
        {"name": "overrideLevel", "type": ["null", "string"], "default": null},
        {"name": "contextData", "type": ["null", {"type": "map", "values": "string"}], "default": null},
        {"name": "errors", "type": {"type": "array", "items": "string"}, "default": []},
        {"name": "metadata", "type": ["null", {"type": "map", "values": "string"}], "default": null}
    ]
}
EOF

    # Phase 4: Enhanced Override Token Schema with Snapshot Integration
    cat > "$SCRIPT_DIR/schemas/enhanced-override-token.avsc" << 'EOF'
{
    "type": "record",
    "name": "EnhancedOverrideToken",
    "namespace": "com.cardiofit.safety.learning",
    "fields": [
        {"name": "tokenId", "type": "string"},
        {"name": "requestId", "type": "string"},
        {"name": "patientId", "type": "string"},
        {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
        {"name": "requiredLevel", "type": {"type": "enum", "name": "OverrideLevel", "symbols": ["RESIDENT", "ATTENDING", "PHARMACIST", "CHIEF"]}},
        {"name": "expiresAt", "type": {"type": "long", "logicalType": "timestamp-millis"}},
        {"name": "contextHash", "type": "string"},
        {"name": "signature", "type": "string"},
        {"name": "snapshotReference", "type": {
            "type": "record",
            "name": "SnapshotReference",
            "fields": [
                {"name": "snapshotId", "type": "string"},
                {"name": "checksum", "type": "string"},
                {"name": "createdAt", "type": {"type": "long", "logicalType": "timestamp-millis"}},
                {"name": "dataCompleteness", "type": "double"}
            ]
        }},
        {"name": "reproducibilityPackage", "type": {
            "type": "record",
            "name": "ReproducibilityPackage",
            "fields": [
                {"name": "proposalId", "type": "string"},
                {"name": "engineVersions", "type": {"type": "map", "values": "string"}},
                {"name": "ruleVersions", "type": {"type": "map", "values": "string"}},
                {"name": "dataSources", "type": {"type": "array", "items": "string"}},
                {"name": "snapshotCreationTime", "type": {"type": "long", "logicalType": "timestamp-millis"}},
                {"name": "validationTime", "type": {"type": "long", "logicalType": "timestamp-millis"}},
                {"name": "metadata", "type": ["null", {"type": "map", "values": "string"}], "default": null}
            ]
        }},
        {"name": "decisionSummary", "type": {
            "type": "record",
            "name": "DecisionSummary",
            "fields": [
                {"name": "status", "type": "string"},
                {"name": "riskScore", "type": "double"},
                {"name": "criticalViolations", "type": {"type": "array", "items": "string"}},
                {"name": "enginesFailed", "type": {"type": "array", "items": "string"}},
                {"name": "explanation", "type": ["null", "string"], "default": null}
            ]
        }}
    ]
}
EOF

    # Phase 4: Clinical Outcome Event Schema
    cat > "$SCRIPT_DIR/schemas/clinical-outcome-event.avsc" << 'EOF'
{
    "type": "record",
    "name": "ClinicalOutcomeEvent",
    "namespace": "com.cardiofit.safety.learning",
    "fields": [
        {"name": "eventId", "type": "string"},
        {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
        {"name": "patientId", "type": "string"},
        {"name": "outcomeType", "type": {"type": "enum", "name": "OutcomeType", "symbols": ["ADVERSE_EVENT", "MEDICATION_RESPONSE", "CLINICAL_IMPROVEMENT", "DETERIORATION", "READMISSION", "LENGTH_OF_STAY"]}},
        {"name": "outcomeValue", "type": "string"},
        {"name": "outcomeSeverity", "type": {"type": "enum", "name": "Severity", "symbols": ["MILD", "MODERATE", "SEVERE", "CRITICAL", "FATAL"]}},
        {"name": "relatedRequestId", "type": ["null", "string"], "default": null},
        {"name": "relatedTokenId", "type": ["null", "string"], "default": null},
        {"name": "timeToOutcome", "type": ["null", "long"], "default": null},
        {"name": "metadata", "type": ["null", {"type": "map", "values": "string"}], "default": null}
    ]
}
EOF

    # Phase 4: Performance Analysis Event Schema
    cat > "$SCRIPT_DIR/schemas/performance-analysis-event.avsc" << 'EOF'
{
    "type": "record",
    "name": "PerformanceAnalysisEvent",
    "namespace": "com.cardiofit.safety.learning",
    "fields": [
        {"name": "eventId", "type": "string"},
        {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
        {"name": "analysisType", "type": {"type": "enum", "name": "AnalysisType", "symbols": ["OVERRIDE_PATTERN", "OUTCOME_CORRELATION", "PERFORMANCE_TREND", "RISK_PREDICTION", "SYSTEM_HEALTH"]}},
        {"name": "timeWindow", "type": "string"},
        {"name": "metrics", "type": {"type": "map", "values": "double"}},
        {"name": "recommendations", "type": {"type": "array", "items": "string"}},
        {"name": "patientId", "type": ["null", "string"], "default": null},
        {"name": "clinicianId", "type": ["null", "string"], "default": null},
        {"name": "metadata", "type": ["null", {"type": "map", "values": "string"}], "default": null}
    ]
}
EOF

    # Phase 4: Decision Reproducibility Event Schema
    cat > "$SCRIPT_DIR/schemas/reproducibility-event.avsc" << 'EOF'
{
    "type": "record",
    "name": "ReproducibilityEvent",
    "namespace": "com.cardiofit.safety.learning",
    "fields": [
        {"name": "eventId", "type": "string"},
        {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
        {"name": "replayId", "type": "string"},
        {"name": "tokenId", "type": "string"},
        {"name": "proposalId", "type": "string"},
        {"name": "success", "type": "boolean"},
        {"name": "reproducibilityScore", "type": "double"},
        {"name": "snapshotValid", "type": "boolean"},
        {"name": "engineComparisons", "type": {
            "type": "array",
            "items": {
                "type": "record",
                "name": "EngineComparison",
                "fields": [
                    {"name": "engineId", "type": "string"},
                    {"name": "originalVersion", "type": "string"},
                    {"name": "currentVersion", "type": "string"},
                    {"name": "versionMatch", "type": "boolean"},
                    {"name": "successful", "type": "boolean"},
                    {"name": "partialReplay", "type": "boolean", "default": false},
                    {"name": "issues", "type": {"type": "array", "items": "string"}}
                ]
            }
        }},
        {"name": "issues", "type": {
            "type": "array",
            "items": {
                "type": "record",
                "name": "ReproducibilityIssue",
                "fields": [
                    {"name": "type", "type": "string"},
                    {"name": "description", "type": "string"},
                    {"name": "severity", "type": {"type": "enum", "name": "IssueSeverity", "symbols": ["LOW", "MEDIUM", "HIGH", "CRITICAL"]}}
                ]
            }
        }},
        {"name": "metadata", "type": ["null", {"type": "map", "values": "string"}], "default": null}
    ]
}
EOF

    log_success "Avro schemas created"
    
    # Register schemas if Schema Registry is available
    if curl -f -s "$SCHEMA_REGISTRY_URL" > /dev/null 2>&1; then
        log_info "Registering schemas with Schema Registry..."
        
        for schema_file in "$SCRIPT_DIR/schemas"/*.avsc; do
            local subject=$(basename "$schema_file" .avsc)
            local schema_content=$(cat "$schema_file" | jq -c '.')
            
            curl -X POST \
                -H "Content-Type: application/vnd.schemaregistry.v1+json" \
                --data "{\"schema\":\"$schema_content\"}" \
                "$SCHEMA_REGISTRY_URL/subjects/$subject-value/versions" \
                || log_warn "Failed to register schema: $subject"
        done
        
        log_success "Schemas registered with Schema Registry"
    else
        log_warn "Schema Registry not available, schemas created locally only"
    fi
}

# Create Kafka consumer groups
create_consumer_groups() {
    log_info "Setting up consumer groups..."
    
    # Consumer group configurations
    declare -A consumer_groups=(
        # Existing groups
        ["safety-real-time-processor"]="safety-events-raw"
        ["safety-alert-handler"]="safety-alerts-critical"
        ["safety-metrics-aggregator"]="safety-metrics-stream"
        ["safety-audit-logger"]="safety-audit-events"
        ["safety-dashboard-updates"]="safety-events-processed"
        
        # Phase 4: Learning Gateway Groups
        ["learning-decision-analyzer"]="clinical-learning-safety-decisions"
        ["learning-override-analyzer"]="clinical-learning-clinical-overrides"
        ["learning-outcome-correlator"]="clinical-learning-clinical-outcomes"
        ["learning-performance-analyzer"]="clinical-learning-performance-analysis"
        ["learning-pattern-detector"]="clinical-learning-override-patterns"
        ["learning-reproducibility-tracker"]="clinical-learning-reproducibility-events"
        ["learning-risk-predictor"]="clinical-learning-risk-predictions"
    )
    
    for group in "${!consumer_groups[@]}"; do
        local topic="${consumer_groups[$group]}"
        
        log_info "Setting up consumer group: $group for topic: $topic"
        
        # Create a temporary consumer to initialize the group
        timeout 5 kafka-console-consumer \
            --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
            --topic "$topic" \
            --group "$group" \
            --from-beginning \
            --max-messages 1 \
            > /dev/null 2>&1 || true
        
        # Configure consumer group settings
        kafka-configs --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
                      --entity-type groups \
                      --entity-name "$group" \
                      --alter \
                      --add-config "consumer.session.timeout.ms=30000,consumer.heartbeat.interval.ms=3000" \
                      || log_warn "Failed to configure consumer group: $group"
    done
    
    log_success "Consumer groups configured"
}

# Create Kafka Streams topology configuration for Phase 4
create_streams_topology() {
    log_info "Creating Kafka Streams topology configuration..."
    
    cat > "$SCRIPT_DIR/streams-topology.yaml" << 'EOF'
# Kafka Streams Topology Configuration for Safety Gateway Phase 4
safety_streams:
  application_id: "safety-gateway-streams-v4"
  bootstrap_servers: "${KAFKA_BOOTSTRAP_SERVERS}"
  
  # Processing guarantee for safety-critical data
  processing_guarantee: "exactly_once_v2"
  
  # Topology definitions
  topologies:
    # Real-time safety event processing
    - name: "safety-event-processor"
      source_topics: ["safety-events-raw"]
      sink_topics: ["safety-events-processed", "safety-alerts-critical", "clinical-learning-safety-decisions"]
      processors:
        - type: "filter"
          config:
            condition: "decision != 'SAFE'"
        - type: "enrich"
          config:
            lookup_table: "patient_context"
        - type: "branch"
          config:
            branches:
              - condition: "decision == 'UNSAFE'"
                target: "safety-alerts-critical"
              - condition: "true"
                target: "safety-events-processed"
        - type: "transform"
          config:
            target: "clinical-learning-safety-decisions"
            add_learning_metadata: true
    
    # Phase 4: Override pattern analysis
    - name: "override-pattern-analyzer"
      source_topics: ["clinical-learning-clinical-overrides"]
      sink_topics: ["clinical-learning-override-patterns"]
      window_type: "session"
      session_timeout: "4_hours"
      processors:
        - type: "sessionize"
          config:
            group_by: ["patientId", "clinicianId"]
        - type: "aggregate"
          config:
            aggregations:
              - type: "count"
                alias: "override_count"
              - type: "avg"
                field: "originalDecision.riskScore"
                alias: "avg_risk_score"
              - type: "collect_list"
                field: "requiredLevel"
                alias: "override_levels"
        - type: "filter"
          config:
            condition: "override_count > 2"  # Pattern detection threshold
    
    # Phase 4: Outcome correlation stream
    - name: "outcome-correlator"
      source_topics: ["clinical-learning-clinical-outcomes", "clinical-learning-clinical-overrides"]
      sink_topics: ["clinical-learning-correlation-analysis"]
      window_type: "tumbling"
      window_size: "24_hours"
      processors:
        - type: "join"
          config:
            join_type: "left_outer"
            join_key: "patientId"
            time_difference: "72_hours"  # Correlation window
        - type: "filter"
          config:
            condition: "correlation_strength > 0.3"
        - type: "enrich"
          config:
            add_correlation_metrics: true
    
    # Phase 4: Risk prediction stream
    - name: "risk-predictor"
      source_topics: ["clinical-learning-override-patterns", "clinical-learning-correlation-analysis"]
      sink_topics: ["clinical-learning-risk-predictions"]
      window_type: "hopping"
      window_size: "7_days"
      advance_by: "1_day"
      processors:
        - type: "ml_predict"
          config:
            model_type: "risk_assessment"
            features: ["override_frequency", "correlation_strength", "patient_demographics"]
            threshold: 0.7
        - type: "filter"
          config:
            condition: "risk_score > threshold"
    
    # Phase 4: Reproducibility tracking
    - name: "reproducibility-tracker"
      source_topics: ["clinical-learning-reproducibility-events"]
      sink_topics: ["clinical-learning-performance-analysis"]
      window_type: "tumbling"
      window_size: "1_hour"
      processors:
        - type: "aggregate"
          config:
            group_by: ["engineId"]
            aggregations:
              - type: "avg"
                field: "reproducibilityScore"
                alias: "avg_reproducibility"
              - type: "count"
                condition: "success == true"
                alias: "successful_replays"
              - type: "count"
                alias: "total_replays"
        - type: "transform"
          config:
            add_success_rate: true
            alert_threshold: 0.85  # Alert if reproducibility < 85%

# State stores for enrichment and caching
state_stores:
  - name: "patient_context"
    type: "key_value"
    config:
      retention: "24_hours"
      caching_enabled: true
      
  - name: "safety_rules_cache"
    type: "key_value"
    config:
      retention: "1_hour"
      caching_enabled: true
      
  # Phase 4: Learning state stores
  - name: "override_patterns"
    type: "session_store"
    config:
      retention: "7_days"
      caching_enabled: true
      
  - name: "outcome_correlations"
    type: "window_store"
    config:
      window_size: "30_days"
      retention: "90_days"
      
  - name: "risk_models"
    type: "key_value"
    config:
      retention: "30_days"
      caching_enabled: true

# Error handling
error_handling:
  deserialization_exception_handler: "log_and_continue"
  production_exception_handler: "log_and_fail"
  
# Monitoring
monitoring:
  metrics_reporters: ["prometheus"]
  metrics_recording_level: "info"
  
# Phase 4: Learning-specific monitoring
learning_monitoring:
  pattern_detection_alerts: true
  correlation_threshold_alerts: true
  reproducibility_alerts: true
  ml_model_performance_tracking: true
EOF

    log_success "Kafka Streams topology configuration created"
}

# Setup monitoring for Kafka topics
setup_kafka_monitoring() {
    log_info "Setting up Kafka monitoring..."
    
    # Create JMX metrics configuration
    cat > "$SCRIPT_DIR/kafka-jmx-config.yaml" << 'EOF'
# Kafka JMX Metrics Configuration - Phase 4
kafka_metrics:
  # Topic-level metrics
  topic_metrics:
    - kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=safety-events-raw
    - kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec,topic=safety-events-raw
    - kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec,topic=safety-events-raw
    - kafka.server:type=BrokerTopicMetrics,name=TotalProduceRequestsPerSec,topic=safety-alerts-critical
    - kafka.server:type=BrokerTopicMetrics,name=TotalFetchRequestsPerSec,topic=safety-alerts-critical
    
    # Phase 4: Learning gateway topic metrics
    - kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=clinical-learning-safety-decisions
    - kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=clinical-learning-clinical-overrides
    - kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=clinical-learning-clinical-outcomes
    - kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=clinical-learning-performance-analysis
    
  # Consumer group metrics
  consumer_metrics:
    - kafka.consumer:type=consumer-fetch-manager-metrics,client-id=safety-real-time-processor
    - kafka.consumer:type=consumer-fetch-manager-metrics,client-id=safety-alert-handler
    - kafka.consumer:type=consumer-fetch-manager-metrics,client-id=learning-decision-analyzer
    - kafka.consumer:type=consumer-fetch-manager-metrics,client-id=learning-override-analyzer
    - kafka.consumer:type=consumer-fetch-manager-metrics,client-id=learning-outcome-correlator
    
  # Producer metrics
  producer_metrics:
    - kafka.producer:type=producer-metrics,client-id=safety-gateway-producer
    - kafka.producer:type=producer-metrics,client-id=learning-gateway-producer
    - kafka.producer:type=producer-topic-metrics,client-id=safety-gateway-producer,topic=safety-events-raw

# Alerting rules
alerting_rules:
  - alert: KafkaTopicHighLatency
    expression: kafka_topic_partition_leader_lag > 1000
    duration: 2m
    severity: warning
    
  - alert: KafkaConsumerLag
    expression: kafka_consumer_lag_sum > 5000
    duration: 5m
    severity: critical
    
  - alert: SafetyAlertsBacklog
    expression: kafka_topic_partition_messages{topic="safety-alerts-critical"} > 100
    duration: 1m
    severity: critical
    
  # Phase 4: Learning-specific alerts
  - alert: LearningEventProcessingLag
    expression: kafka_consumer_lag_sum{group="learning-decision-analyzer"} > 2000
    duration: 3m
    severity: warning
    
  - alert: OutcomeCorrelationBacklog
    expression: kafka_topic_partition_messages{topic="clinical-learning-clinical-outcomes"} > 500
    duration: 5m
    severity: warning
    
  - alert: ReproducibilityEventLag
    expression: kafka_consumer_lag_sum{group="learning-reproducibility-tracker"} > 100
    duration: 2m
    severity: high
EOF

    # Create Prometheus scraping configuration
    cat > "$SCRIPT_DIR/prometheus-kafka.yaml" << 'EOF'
# Add to prometheus.yml - Phase 4
scrape_configs:
  - job_name: 'kafka-jmx'
    static_configs:
      - targets: ['localhost:9999']  # Kafka JMX port
    scrape_interval: 15s
    metrics_path: /metrics
    
  - job_name: 'safety-kafka-streams'
    static_configs:
      - targets: ['localhost:8080']  # Streams application metrics
    scrape_interval: 10s
    metrics_path: /metrics
    
  # Phase 4: Learning gateway metrics
  - job_name: 'learning-kafka-streams'
    static_configs:
      - targets: ['localhost:8081']  # Learning streams application metrics
    scrape_interval: 10s
    metrics_path: /metrics
    
  - job_name: 'override-analyzer-metrics'
    static_configs:
      - targets: ['localhost:8082']  # Override analyzer metrics
    scrape_interval: 15s
    metrics_path: /metrics
    
  - job_name: 'reproducibility-tracker-metrics'
    static_configs:
      - targets: ['localhost:8083']  # Reproducibility tracker metrics
    scrape_interval: 30s
    metrics_path: /metrics
EOF

    log_success "Kafka monitoring configuration created"
}

# Test Kafka setup
test_kafka_setup() {
    log_info "Testing Kafka setup..."
    
    # Test topic creation and basic functionality
    local test_topic="safety-test-topic"
    local test_message='{"test": "message", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}'
    
    # Create test topic
    kafka-topics --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
                 --create \
                 --topic "$test_topic" \
                 --partitions 1 \
                 --replication-factor 1 \
                 --if-not-exists
    
    # Produce test message
    echo "$test_message" | kafka-console-producer \
        --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
        --topic "$test_topic"
    
    # Consume test message
    local consumed_message=$(kafka-console-consumer \
        --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
        --topic "$test_topic" \
        --from-beginning \
        --max-messages 1 \
        --timeout-ms 5000)
    
    # Verify message
    if [ "$consumed_message" = "$test_message" ]; then
        log_success "Kafka functionality test passed"
    else
        log_error "Kafka functionality test failed"
        log_error "Expected: $test_message"
        log_error "Got: $consumed_message"
        exit 1
    fi
    
    # Clean up test topic
    kafka-topics --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
                 --delete \
                 --topic "$test_topic"
    
    # Test all safety topics exist
    local topics_list=$(kafka-topics --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" --list)
    local required_topics=(
        "safety-events-raw" 
        "safety-events-processed" 
        "safety-alerts-critical" 
        "safety-override-tokens" 
        "safety-metrics-stream" 
        "safety-audit-events"
        "clinical-learning-safety-decisions"
        "clinical-learning-clinical-overrides"
        "clinical-learning-clinical-outcomes"
        "clinical-learning-performance-analysis"
        "clinical-learning-reproducibility-events"
    )
    
    for topic in "${required_topics[@]}"; do
        if echo "$topics_list" | grep -q "$topic"; then
            log_success "Topic exists: $topic"
        else
            log_error "Required topic missing: $topic"
            exit 1
        fi
    done
    
    log_success "All Kafka setup tests passed"
}

# Generate producer/consumer code examples for Phase 4
generate_code_examples() {
    log_info "Generating Phase 4 code examples..."
    
    mkdir -p "$SCRIPT_DIR/examples"
    
    # Phase 4: Enhanced Override Token Producer
    cat > "$SCRIPT_DIR/examples/enhanced_override_producer.go" << 'EOF'
package main

import (
    "context"
    "encoding/json"
    "log"
    "time"
    
    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type EnhancedOverrideToken struct {
    TokenID    string    `json:"token_id"`
    RequestID  string    `json:"request_id"`
    PatientID  string    `json:"patient_id"`
    Timestamp  time.Time `json:"timestamp"`
    RequiredLevel string `json:"required_level"`
    SnapshotReference SnapshotReference `json:"snapshot_reference"`
    ReproducibilityPackage ReproducibilityPackage `json:"reproducibility_package"`
    DecisionSummary DecisionSummary `json:"decision_summary"`
}

type SnapshotReference struct {
    SnapshotID       string    `json:"snapshot_id"`
    Checksum         string    `json:"checksum"`
    CreatedAt        time.Time `json:"created_at"`
    DataCompleteness float64   `json:"data_completeness"`
}

type ReproducibilityPackage struct {
    ProposalID     string            `json:"proposal_id"`
    EngineVersions map[string]string `json:"engine_versions"`
    RuleVersions   map[string]string `json:"rule_versions"`
    DataSources    []string          `json:"data_sources"`
}

type DecisionSummary struct {
    Status             string   `json:"status"`
    RiskScore          float64  `json:"risk_score"`
    CriticalViolations []string `json:"critical_violations"`
}

func main() {
    producer, err := kafka.NewProducer(&kafka.ConfigMap{
        "bootstrap.servers": "localhost:9092",
        "acks":             "all",
        "retries":          3,
        "enable.idempotence": true,
        "compression.type":  "snappy",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer producer.Close()

    token := EnhancedOverrideToken{
        TokenID:   "token-12345",
        RequestID: "req-67890",
        PatientID: "patient-123",
        Timestamp: time.Now(),
        RequiredLevel: "PHARMACIST",
        SnapshotReference: SnapshotReference{
            SnapshotID:       "snap-abc123",
            Checksum:         "sha256-hash",
            CreatedAt:        time.Now().Add(-time.Hour),
            DataCompleteness: 92.5,
        },
        ReproducibilityPackage: ReproducibilityPackage{
            ProposalID: "prop-xyz789",
            EngineVersions: map[string]string{
                "cae-engine": "1.2.3",
                "drug-interaction": "2.1.0",
            },
            RuleVersions: map[string]string{
                "dosing-rules": "3.0.1",
                "allergy-rules": "1.5.2",
            },
            DataSources: []string{"fhir", "snapshot"},
        },
        DecisionSummary: DecisionSummary{
            Status:    "UNSAFE",
            RiskScore: 0.85,
            CriticalViolations: []string{"drug-allergy-interaction", "dosing-error"},
        },
    }

    data, err := json.Marshal(token)
    if err != nil {
        log.Fatal(err)
    }

    topic := "clinical-learning-clinical-overrides"
    err = producer.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
        Key:            []byte(token.PatientID),
        Value:          data,
        Headers: []kafka.Header{
            {Key: "content-type", Value: []byte("application/json")},
            {Key: "event-type", Value: []byte("enhanced-override-token")},
            {Key: "version", Value: []byte("4.0.0")},
        },
    }, nil)

    if err != nil {
        log.Fatal("Failed to produce message:", err)
    }

    producer.Flush(15 * 1000) // Wait up to 15 seconds
    log.Println("Enhanced override token published successfully")
}
EOF

    # Phase 4: Learning Event Consumer
    cat > "$SCRIPT_DIR/examples/learning_event_consumer.go" << 'EOF'
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type LearningEvent struct {
    EventID         string                 `json:"event_id"`
    EventType       string                 `json:"event_type"`
    Timestamp       string                 `json:"timestamp"`
    PatientID       string                 `json:"patient_id"`
    AnalysisData    map[string]interface{} `json:"analysis_data"`
    Recommendations []string               `json:"recommendations"`
}

func main() {
    consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
        "bootstrap.servers":  "localhost:9092",
        "group.id":          "learning-event-processor",
        "auto.offset.reset": "earliest",
        "enable.auto.commit": false,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer consumer.Close()

    topics := []string{
        "clinical-learning-safety-decisions",
        "clinical-learning-clinical-overrides", 
        "clinical-learning-clinical-outcomes",
        "clinical-learning-performance-analysis",
    }

    err = consumer.SubscribeTopics(topics, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Set up signal handling for graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    fmt.Println("Learning event consumer started. Listening for events...")

    for {
        select {
        case <-sigChan:
            fmt.Println("Received shutdown signal, closing consumer...")
            return
        default:
            msg, err := consumer.ReadMessage(1000) // 1 second timeout
            if err != nil {
                // Timeout is expected, continue
                if err.(kafka.Error).Code() == kafka.ErrTimedOut {
                    continue
                }
                log.Printf("Consumer error: %v\n", err)
                continue
            }

            var event LearningEvent
            if err := json.Unmarshal(msg.Value, &event); err != nil {
                log.Printf("Error unmarshaling event: %v\n", err)
                continue
            }

            // Process learning event
            processLearningEvent(*msg.TopicPartition.Topic, event)

            // Commit the message
            if err := consumer.CommitMessage(msg); err != nil {
                log.Printf("Error committing message: %v\n", err)
            }
        }
    }
}

func processLearningEvent(topic string, event LearningEvent) {
    fmt.Printf("Processing %s event: %s from topic: %s\n", 
        event.EventType, event.EventID, topic)

    switch topic {
    case "clinical-learning-safety-decisions":
        processDecisionEvent(event)
    case "clinical-learning-clinical-overrides":
        processOverrideEvent(event)
    case "clinical-learning-clinical-outcomes":
        processOutcomeEvent(event)
    case "clinical-learning-performance-analysis":
        processPerformanceEvent(event)
    }
}

func processDecisionEvent(event LearningEvent) {
    // Analyze decision patterns, risk factors, engine performance
    fmt.Printf("  -> Analyzing decision for patient: %s\n", event.PatientID)
}

func processOverrideEvent(event LearningEvent) {
    // Analyze override patterns, correlate with outcomes
    fmt.Printf("  -> Analyzing override pattern for patient: %s\n", event.PatientID)
}

func processOutcomeEvent(event LearningEvent) {
    // Correlate outcomes with previous decisions and overrides
    fmt.Printf("  -> Correlating outcome for patient: %s\n", event.PatientID)
}

func processPerformanceEvent(event LearningEvent) {
    // Analyze system and clinical performance metrics
    fmt.Printf("  -> Processing performance analysis\n")
}
EOF

    log_success "Phase 4 code examples generated"
}

# Main setup function
run_setup() {
    log_info "Starting Kafka setup for Safety Gateway Platform - Phase 4"
    
    check_prerequisites
    test_kafka_connection
    create_safety_topics
    create_avro_schemas
    create_consumer_groups
    create_streams_topology
    setup_kafka_monitoring
    generate_code_examples
    test_kafka_setup
    
    log_success "Phase 4 Kafka setup completed successfully!"
    log_info "Next steps:"
    log_info "1. Update safety-gateway configuration to use new Kafka topics"
    log_info "2. Deploy learning gateway Kafka Streams applications"
    log_info "3. Configure enhanced monitoring and alerting"
    log_info "4. Run Phase 4 integration tests"
    log_info "5. Enable reproducibility tracking and outcome correlation"
}

# Script execution
case "${1:-run}" in
    "run")
        run_setup
        ;;
    "test")
        test_kafka_setup
        ;;
    "topics")
        create_safety_topics
        ;;
    "schemas")
        create_avro_schemas
        ;;
    "monitor")
        setup_kafka_monitoring
        ;;
    "examples")
        generate_code_examples
        ;;
    *)
        echo "Usage: $0 [run|test|topics|schemas|monitor|examples]"
        echo "  run      - Execute full Phase 4 Kafka setup"
        echo "  test     - Test Kafka functionality only"
        echo "  topics   - Create safety and learning topics only"
        echo "  schemas  - Create Phase 4 Avro schemas only"
        echo "  monitor  - Setup enhanced monitoring only"
        echo "  examples - Generate Phase 4 code examples only"
        exit 1
        ;;
esac