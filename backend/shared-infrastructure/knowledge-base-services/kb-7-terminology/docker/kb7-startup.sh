#!/bin/bash

set -e

echo "🚀 Starting KB-7 Semantic Infrastructure Container"
echo "================================================="

# Create log directory
mkdir -p /var/log

# Wait for system to be ready
sleep 5

# Start supervisor to manage all services
echo "Starting supervisor with all KB-7 services..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/kb7.conf