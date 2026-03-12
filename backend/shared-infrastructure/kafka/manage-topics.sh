#!/bin/bash

# CardioFit Kafka Topic Management Script
# Provides utilities for managing Kafka topics

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Kafka connection settings
KAFKA_CONTAINER="cardiofit-kafka1"
KAFKA_BROKERS="kafka1:29092"

# Function to show usage
show_usage() {
    echo "==========================================="
    echo "CardioFit Kafka Topic Management"
    echo "==========================================="
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  list                    List all topics"
    echo "  describe <topic>        Describe a specific topic"
    echo "  create <topic>          Create a new topic"
    echo "  delete <topic>          Delete a topic"
    echo "  produce <topic>         Send test messages to a topic"
    echo "  consume <topic>         Read messages from a topic"
    echo "  stats                   Show topic statistics"
    echo "  validate                Validate all topics against reference"
    echo "  reset                   Reset all topics (WARNING: deletes all data)"
    echo ""
    echo "Examples:"
    echo "  $0 list"
    echo "  $0 describe patient-events.v1"
    echo "  $0 produce patient-events.v1"
    echo "  $0 stats"
    echo ""
}

# List all topics
list_topics() {
    echo "📋 Listing all topics..."
    echo ""

    # Group topics by category
    echo "CLINICAL EVENTS:"
    docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '^(patient|medication|observation|safety|vital|lab|encounter|diagnostic|procedure)-' | sort || true

    echo ""
    echo "DEVICE DATA:"
    docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '^(raw-device|validated-device|waveform|device-telemetry)' | sort || true

    echo ""
    echo "RUNTIME LAYER:"
    docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '^(enriched|clinical-patterns|pathway|semantic-mesh|patient-context)' | sort || true

    echo ""
    echo "KNOWLEDGE BASE CDC:"
    docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '^kb[0-9]\.|^semantic-mesh\.changes' | sort || true

    echo ""
    echo "EVIDENCE MANAGEMENT:"
    docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '^(audit|envelope|evidence|clinical-reasoning|inference)' | sort || true

    echo ""
    echo "DEAD LETTER QUEUES:"
    docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '(dlq|poison|failures|errors)' | sort || true

    echo ""
    TOTAL=$(docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '\.(v1|changes)$' | wc -l)
    print_info "Total CardioFit topics: $TOTAL"
}

# Describe a topic
describe_topic() {
    local TOPIC=$1

    if [ -z "$TOPIC" ]; then
        print_error "Please specify a topic name"
        exit 1
    fi

    echo "📊 Describing topic: $TOPIC"
    echo ""

    docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --describe --topic $TOPIC
}

# Create a topic
create_topic() {
    local TOPIC=$1

    if [ -z "$TOPIC" ]; then
        print_error "Please specify a topic name"
        exit 1
    fi

    echo "Creating topic: $TOPIC"

    # Default configuration
    PARTITIONS=${2:-12}
    REPLICATION=${3:-3}

    docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS \
        --create \
        --topic $TOPIC \
        --partitions $PARTITIONS \
        --replication-factor $REPLICATION \
        --config retention.ms=604800000 \
        --config compression.type=snappy

    print_status "Topic $TOPIC created"
}

# Delete a topic
delete_topic() {
    local TOPIC=$1

    if [ -z "$TOPIC" ]; then
        print_error "Please specify a topic name"
        exit 1
    fi

    print_warning "Are you sure you want to delete topic: $TOPIC?"
    read -p "Type 'yes' to confirm: " CONFIRM

    if [ "$CONFIRM" != "yes" ]; then
        print_info "Deletion cancelled"
        exit 0
    fi

    docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --delete --topic $TOPIC
    print_status "Topic $TOPIC deleted"
}

# Produce test messages to a topic
produce_messages() {
    local TOPIC=$1

    if [ -z "$TOPIC" ]; then
        print_error "Please specify a topic name"
        exit 1
    fi

    echo "📤 Producing test messages to: $TOPIC"
    echo "Type messages and press Enter. Press Ctrl+D to exit."
    echo ""

    # Generate sample message based on topic type
    if [[ $TOPIC == "patient-events.v1" ]]; then
        cat <<EOF | docker exec -i $KAFKA_CONTAINER kafka-console-producer --bootstrap-server $KAFKA_BROKERS --topic $TOPIC
{"event_id":"test-$(date +%s)","event_type":"patient.admission","timestamp":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")","patient_id":"P12345","data":{"department":"cardiology","reason":"chest pain"}}
{"event_id":"test-$(date +%s)","event_type":"patient.discharge","timestamp":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")","patient_id":"P12345","data":{"department":"cardiology","outcome":"recovered"}}
EOF
        print_status "Sent 2 test messages"
    else
        # Interactive mode for other topics
        docker exec -it $KAFKA_CONTAINER kafka-console-producer --bootstrap-server $KAFKA_BROKERS --topic $TOPIC
    fi
}

# Consume messages from a topic
consume_messages() {
    local TOPIC=$1

    if [ -z "$TOPIC" ]; then
        print_error "Please specify a topic name"
        exit 1
    fi

    echo "📥 Consuming messages from: $TOPIC"
    echo "Press Ctrl+C to exit"
    echo ""

    docker exec -it $KAFKA_CONTAINER kafka-console-consumer \
        --bootstrap-server $KAFKA_BROKERS \
        --topic $TOPIC \
        --from-beginning \
        --max-messages 10 \
        --formatter kafka.tools.DefaultMessageFormatter \
        --property print.timestamp=true \
        --property print.key=true \
        --property print.value=true
}

# Show topic statistics
show_stats() {
    echo "📊 Kafka Topic Statistics"
    echo "========================="
    echo ""

    # Count topics by category
    echo "Topics by Category:"
    echo "  Clinical Events: $(docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -cE '^(patient|medication|observation|safety|vital|lab|encounter|diagnostic|procedure)-' || echo 0)"
    echo "  Device Data: $(docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -cE '^(raw-device|validated-device|waveform|device-telemetry)' || echo 0)"
    echo "  Runtime Layer: $(docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -cE '^(enriched|clinical-patterns|pathway|semantic-mesh|patient-context)' || echo 0)"
    echo "  Knowledge Base CDC: $(docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -cE '^kb[0-9]\.' || echo 0)"
    echo "  Evidence Management: $(docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -cE '^(audit|envelope|evidence|clinical-reasoning|inference)' || echo 0)"
    echo "  DLQ Topics: $(docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -cE '(dlq|poison|failures|errors)' || echo 0)"
    echo ""

    # Show top topics by partition count
    echo "Topics with Most Partitions:"
    docker exec $KAFKA_CONTAINER bash -c "
        for topic in \$(kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '\.(v1|changes)\$'); do
            partitions=\$(kafka-topics --bootstrap-server $KAFKA_BROKERS --describe --topic \$topic | grep PartitionCount | awk '{print \$2}' | cut -d'=' -f2)
            echo \"\$partitions|\$topic\"
        done | sort -rn | head -5 | column -t -s'|'
    "

    echo ""

    # Check for under-replicated partitions
    UNDER_REPLICATED=$(docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --describe --under-replicated-partitions | wc -l)
    if [ "$UNDER_REPLICATED" -gt "0" ]; then
        print_warning "Under-replicated partitions: $UNDER_REPLICATED"
    else
        print_status "All partitions fully replicated"
    fi
}

# Validate topics against reference
validate_topics() {
    echo "🔍 Validating topics against reference..."
    echo ""

    # Expected topic count
    EXPECTED=68
    ACTUAL=$(docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '\.(v1|changes)$' | wc -l)

    if [ "$ACTUAL" -eq "$EXPECTED" ]; then
        print_status "Topic count matches: $ACTUAL/$EXPECTED"
    else
        print_warning "Topic count mismatch: $ACTUAL/$EXPECTED"
    fi

    # Check for missing critical topics
    CRITICAL_TOPICS=(
        "patient-events.v1"
        "medication-events.v1"
        "audit-events.v1"
        "enriched-patient-events.v1"
        "clinical-patterns.v1"
    )

    echo ""
    echo "Checking critical topics:"
    for topic in "${CRITICAL_TOPICS[@]}"; do
        if docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -q "^$topic\$"; then
            print_status "$topic exists"
        else
            print_error "$topic missing"
        fi
    done
}

# Reset all topics
reset_topics() {
    print_warning "WARNING: This will delete ALL topics and recreate them!"
    print_warning "All data will be lost!"
    echo ""
    read -p "Type 'RESET' to confirm: " CONFIRM

    if [ "$CONFIRM" != "RESET" ]; then
        print_info "Reset cancelled"
        exit 0
    fi

    echo "Deleting all topics..."

    # Delete all CardioFit topics
    docker exec $KAFKA_CONTAINER bash -c "
        for topic in \$(kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '\.(v1|changes)\$'); do
            echo \"Deleting \$topic...\"
            kafka-topics --bootstrap-server $KAFKA_BROKERS --delete --topic \$topic
        done
    "

    echo "Recreating topics..."
    docker exec $KAFKA_CONTAINER bash /usr/bin/create-topics.sh

    print_status "All topics have been reset"
}

# Main script logic
case "$1" in
    list)
        list_topics
        ;;
    describe)
        describe_topic "$2"
        ;;
    create)
        create_topic "$2" "$3" "$4"
        ;;
    delete)
        delete_topic "$2"
        ;;
    produce)
        produce_messages "$2"
        ;;
    consume)
        consume_messages "$2"
        ;;
    stats)
        show_stats
        ;;
    validate)
        validate_topics
        ;;
    reset)
        reset_topics
        ;;
    *)
        show_usage
        ;;
esac