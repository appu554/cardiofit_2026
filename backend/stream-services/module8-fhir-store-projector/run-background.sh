#!/bin/bash
cd "$(dirname "$0")"
mkdir -p logs
exec python3 run.py > logs/service-$(date +%Y%m%d-%H%M%S).log 2>&1
