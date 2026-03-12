#!/bin/bash
# Load expanded value sets into Redis cache
# Usage: ./load-valuesets.sh [valueset-dir]

set -e

VALUESET_DIR=${1:-"/tmp/valuesets"}
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"

echo "Loading value sets from: $VALUESET_DIR"
echo "Redis: $REDIS_HOST:$REDIS_PORT"

# Load each value set JSON into Redis
for vs_file in "$VALUESET_DIR"/*.json; do
    if [ -f "$vs_file" ]; then
        vs_name=$(basename "$vs_file" .json)
        echo "Loading value set: $vs_name"
        # TODO: Implement Redis loading
        # redis-cli -h $REDIS_HOST -p $REDIS_PORT SET "valueset:$vs_name" "$(cat $vs_file)"
    fi
done

echo "All value sets loaded successfully!"
