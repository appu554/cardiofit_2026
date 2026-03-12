#!/bin/bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services/module8-elasticsearch-projector
python3 run_service.py > elasticsearch-projector.log 2>&1 &
echo $! > elasticsearch-projector.pid
echo "Elasticsearch projector started with PID $(cat elasticsearch-projector.pid)"
sleep 5
echo "Checking logs..."
tail -20 elasticsearch-projector.log
