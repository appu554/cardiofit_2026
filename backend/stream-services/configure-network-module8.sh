#!/bin/bash

# ========================================
# Module 8 Network Configuration Helper
# ========================================

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env.module8"

# External container IDs
POSTGRES_CONTAINER="a2f55d83b1fa"
INFLUXDB_CONTAINER="8502fd5d078d"
NEO4J_CONTAINER="e8b3df4d8a02"

# ========================================
# Helper Functions
# ========================================

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

# ========================================
# Detect Container IPs
# ========================================

detect_container_ip() {
    local container_id=$1
    local container_name=$2

    if docker ps --format '{{.ID}}' | grep -q "^$container_id"; then
        # Try to get IP from multiple networks
        local ip=$(docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$container_id" | head -1)

        if [ -n "$ip" ]; then
            print_success "$container_name ($container_id): $ip"
            echo "$ip"
        else
            print_warning "$container_name ($container_id): No IP address found"
            echo ""
        fi
    else
        print_error "$container_name ($container_id): Container not running"
        echo ""
    fi
}

detect_all_ips() {
    print_header "Detecting External Container IPs"

    # Detect PostgreSQL IP
    POSTGRES_IP=$(detect_container_ip "$POSTGRES_CONTAINER" "PostgreSQL")

    # Detect InfluxDB IP
    INFLUXDB_IP=$(detect_container_ip "$INFLUXDB_CONTAINER" "InfluxDB")

    # Detect Neo4j IP
    NEO4J_IP=$(detect_container_ip "$NEO4J_CONTAINER" "Neo4j")

    echo ""
}

# ========================================
# Network Connectivity Tests
# ========================================

test_network_connectivity() {
    print_header "Testing Network Connectivity"

    # Test PostgreSQL
    if [ -n "$POSTGRES_IP" ]; then
        if nc -z "$POSTGRES_IP" 5432 2>/dev/null; then
            print_success "PostgreSQL ($POSTGRES_IP:5432) - Reachable"
        else
            print_error "PostgreSQL ($POSTGRES_IP:5432) - Not reachable"
        fi
    fi

    # Test InfluxDB
    if [ -n "$INFLUXDB_IP" ]; then
        if nc -z "$INFLUXDB_IP" 8086 2>/dev/null; then
            print_success "InfluxDB ($INFLUXDB_IP:8086) - Reachable"
        else
            print_error "InfluxDB ($INFLUXDB_IP:8086) - Not reachable"
        fi
    fi

    # Test Neo4j
    if [ -n "$NEO4J_IP" ]; then
        if nc -z "$NEO4J_IP" 7687 2>/dev/null; then
            print_success "Neo4j ($NEO4J_IP:7687) - Reachable"
        else
            print_error "Neo4j ($NEO4J_IP:7687) - Not reachable"
        fi
    fi

    echo ""
}

# ========================================
# Update Environment File
# ========================================

update_env_file() {
    print_header "Updating Environment File"

    if [ ! -f "$ENV_FILE" ]; then
        print_warning "Environment file not found: $ENV_FILE"
        print_info "Creating from example..."

        if [ -f "${ENV_FILE}.example" ]; then
            cp "${ENV_FILE}.example" "$ENV_FILE"
            print_success "Created $ENV_FILE from example"
        else
            print_error "Example file not found: ${ENV_FILE}.example"
            return 1
        fi
    fi

    # Backup original file
    cp "$ENV_FILE" "${ENV_FILE}.backup-$(date +%Y%m%d-%H%M%S)"
    print_info "Backup created: ${ENV_FILE}.backup-$(date +%Y%m%d-%H%M%S)"

    # Update PostgreSQL IP
    if [ -n "$POSTGRES_IP" ]; then
        if grep -q "^POSTGRES_HOST=" "$ENV_FILE"; then
            sed -i.tmp "s|^POSTGRES_HOST=.*|POSTGRES_HOST=$POSTGRES_IP|" "$ENV_FILE"
            print_success "Updated POSTGRES_HOST=$POSTGRES_IP"
        else
            echo "POSTGRES_HOST=$POSTGRES_IP" >> "$ENV_FILE"
            print_success "Added POSTGRES_HOST=$POSTGRES_IP"
        fi
    fi

    # Update InfluxDB URL
    if [ -n "$INFLUXDB_IP" ]; then
        if grep -q "^INFLUXDB_URL=" "$ENV_FILE"; then
            sed -i.tmp "s|^INFLUXDB_URL=.*|INFLUXDB_URL=http://$INFLUXDB_IP:8086|" "$ENV_FILE"
            print_success "Updated INFLUXDB_URL=http://$INFLUXDB_IP:8086"
        else
            echo "INFLUXDB_URL=http://$INFLUXDB_IP:8086" >> "$ENV_FILE"
            print_success "Added INFLUXDB_URL=http://$INFLUXDB_IP:8086"
        fi
    fi

    # Update Neo4j URI
    if [ -n "$NEO4J_IP" ]; then
        if grep -q "^NEO4J_URI=" "$ENV_FILE"; then
            sed -i.tmp "s|^NEO4J_URI=.*|NEO4J_URI=bolt://$NEO4J_IP:7687|" "$ENV_FILE"
            print_success "Updated NEO4J_URI=bolt://$NEO4J_IP:7687"
        else
            echo "NEO4J_URI=bolt://$NEO4J_IP:7687" >> "$ENV_FILE"
            print_success "Added NEO4J_URI=bolt://$NEO4J_IP:7687"
        fi
    fi

    # Clean up temporary files
    rm -f "${ENV_FILE}.tmp"

    echo ""
    print_success "Environment file updated successfully"
}

# ========================================
# Display Network Configuration
# ========================================

show_network_config() {
    print_header "Current Network Configuration"

    echo "External Container IPs:"
    echo "  PostgreSQL:  ${POSTGRES_IP:-Not detected}"
    echo "  InfluxDB:    ${INFLUXDB_IP:-Not detected}"
    echo "  Neo4j:       ${NEO4J_IP:-Not detected}"
    echo ""

    echo "Service Endpoints:"
    echo "  PostgreSQL:  ${POSTGRES_IP:-N/A}:5432"
    echo "  InfluxDB:    ${INFLUXDB_IP:-N/A}:8086"
    echo "  Neo4j Bolt:  ${NEO4J_IP:-N/A}:7687"
    echo "  Neo4j HTTP:  ${NEO4J_IP:-N/A}:7474"
    echo ""

    echo "Docker Networks:"
    docker network ls | grep -E "module8|cardiofit" || echo "  No Module 8 networks found"
    echo ""
}

# ========================================
# Create Network Bridge
# ========================================

create_network_bridge() {
    print_header "Creating Network Bridge"

    # Check if module8-network exists
    if docker network ls | grep -q "module8-network"; then
        print_info "module8-network already exists"
    else
        print_info "Creating module8-network..."
        docker network create module8-network --subnet 172.28.0.0/16
        print_success "Network created"
    fi

    # Connect external containers to module8-network
    for container_id in $POSTGRES_CONTAINER $INFLUXDB_CONTAINER $NEO4J_CONTAINER; do
        if docker ps --format '{{.ID}}' | grep -q "^$container_id"; then
            if docker network inspect module8-network 2>/dev/null | grep -q "$container_id"; then
                print_info "Container $container_id already connected"
            else
                print_info "Connecting container $container_id to module8-network..."
                docker network connect module8-network "$container_id" 2>/dev/null || true
                print_success "Container connected"
            fi
        fi
    done

    echo ""
}

# ========================================
# Validate Configuration
# ========================================

validate_configuration() {
    print_header "Validating Configuration"

    local all_valid=true

    # Check if containers are running
    for container_id in $POSTGRES_CONTAINER $INFLUXDB_CONTAINER $NEO4J_CONTAINER; do
        if ! docker ps --format '{{.ID}}' | grep -q "^$container_id"; then
            print_error "Container $container_id is not running"
            all_valid=false
        fi
    done

    # Check if containers are connected to module8-network
    if docker network ls | grep -q "module8-network"; then
        for container_id in $POSTGRES_CONTAINER $INFLUXDB_CONTAINER $NEO4J_CONTAINER; do
            if docker ps --format '{{.ID}}' | grep -q "^$container_id"; then
                if ! docker network inspect module8-network 2>/dev/null | grep -q "$container_id"; then
                    print_warning "Container $container_id not connected to module8-network"
                    all_valid=false
                fi
            fi
        done
    else
        print_error "module8-network does not exist"
        all_valid=false
    fi

    # Check environment file
    if [ ! -f "$ENV_FILE" ]; then
        print_error "Environment file not found: $ENV_FILE"
        all_valid=false
    fi

    echo ""
    if [ "$all_valid" = true ]; then
        print_success "All validations passed"
    else
        print_warning "Some validations failed"
    fi
}

# ========================================
# Show Manual Configuration Steps
# ========================================

show_manual_steps() {
    print_header "Manual Configuration Steps"

    cat << EOF
If automatic detection failed, you can manually configure the network:

1. Find container IPs:
   docker inspect $POSTGRES_CONTAINER | grep IPAddress
   docker inspect $INFLUXDB_CONTAINER | grep IPAddress
   docker inspect $NEO4J_CONTAINER | grep IPAddress

2. Update .env.module8:
   POSTGRES_HOST=<postgres-ip>
   INFLUXDB_URL=http://<influxdb-ip>:8086
   NEO4J_URI=bolt://<neo4j-ip>:7687

3. Connect containers to module8-network:
   docker network connect module8-network $POSTGRES_CONTAINER
   docker network connect module8-network $INFLUXDB_CONTAINER
   docker network connect module8-network $NEO4J_CONTAINER

4. Test connectivity:
   nc -z <postgres-ip> 5432
   nc -z <influxdb-ip> 8086
   nc -z <neo4j-ip> 7687

EOF
}

# ========================================
# Main Execution
# ========================================

main() {
    print_header "🌐 Module 8 Network Configuration"

    # Detect IPs
    detect_all_ips

    # Test connectivity
    test_network_connectivity

    # Create/verify network bridge
    create_network_bridge

    # Update environment file
    echo -n "Update .env.module8 with detected IPs? [Y/n]: "
    read -r response
    if [[ ! "$response" =~ ^[Nn]$ ]]; then
        update_env_file
    fi

    # Show configuration
    show_network_config

    # Validate
    validate_configuration

    # Show manual steps if needed
    if [ -z "$POSTGRES_IP" ] || [ -z "$INFLUXDB_IP" ] || [ -z "$NEO4J_IP" ]; then
        show_manual_steps
    fi

    print_header "✅ Network Configuration Complete"

    echo "Next steps:"
    echo "  1. Verify .env.module8 has correct values"
    echo "  2. Start services: ./start-module8-projectors.sh"
    echo "  3. Check health: ./health-check-module8.sh"
    echo ""
}

# Run main function
main "$@"
