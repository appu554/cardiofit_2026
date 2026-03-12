#!/bin/bash

set -e

echo "KB-7 ROBOT Tool Service Starting..."
echo "Command: $1"
echo "Working Directory: $(pwd)"

case "$1" in
    "validate")
        echo "Running ontology validation..."
        exec python3 scripts/validate_ontologies.py
        ;;
    "convert")
        echo "Running RDF conversion..."
        exec python3 scripts/convert_rdf.py "${@:2}"
        ;;
    "reason")
        echo "Running reasoning pipeline..."
        exec python3 scripts/reasoning_pipeline.py
        ;;
    "merge")
        echo "Running ontology merge..."
        exec python3 scripts/merge_ontologies.py "${@:2}"
        ;;
    "extract")
        echo "Running terminology extraction..."
        exec python3 scripts/extract_terminology.py "${@:2}"
        ;;
    "shell")
        echo "Starting interactive shell..."
        exec /bin/bash
        ;;
    *)
        echo "Available commands: validate, convert, reason, merge, extract, shell"
        echo "Usage: docker run kb7-robot [command] [args...]"
        exit 1
        ;;
esac