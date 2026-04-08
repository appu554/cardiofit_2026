#!/bin/bash
# E2E 14-Day Daily Injector
# Usage: ./inject_day.sh <day_number> <kafka_bootstrap_servers>
#
# Example: ./inject_day.sh 1 localhost:9092

DAY=$1
BOOTSTRAP=${2:-localhost:9092}
DIR="./e2e_14day_dataset"

if [ -z "$DAY" ]; then
    echo "Usage: $0 <day_number> [kafka_bootstrap_servers]"
    exit 1
fi

FILE="$DIR/day$(printf '%02d' $DAY).json"
if [ ! -f "$FILE" ]; then
    echo "ERROR: $FILE not found"
    exit 1
fi

echo "═══════════════════════════════════════════════════════"
echo "  Injecting Day $DAY events from $FILE"
echo "  Kafka: $BOOTSTRAP"
echo "═══════════════════════════════════════════════════════"

# Extract and inject vitals
echo "[1/3] Injecting BP readings to ingestion.vitals..."
python3 -c "
import json, subprocess, sys
with open('$FILE') as f:
    data = json.load(f)
for event in data['inject']['ingestion.vitals']:
    key = event['patient_id']
    value = json.dumps(event)
    print(f'{key}|{value}')
" | kafka-console-producer.sh \
    --bootstrap-server $BOOTSTRAP \
    --topic ingestion.vitals \
    --property parse.key=true \
    --property key.separator='|'

# Extract and inject enriched events
echo "[2/3] Injecting enriched events to enriched-patient-events-v1..."
python3 -c "
import json
with open('$FILE') as f:
    data = json.load(f)
for event in data['inject']['enriched-patient-events-v1']:
    key = event['patientId']
    value = json.dumps(event)
    print(f'{key}|{value}')
" | kafka-console-producer.sh \
    --bootstrap-server $BOOTSTRAP \
    --topic enriched-patient-events-v1 \
    --property parse.key=true \
    --property key.separator='|'

# Extract and inject interventions
echo "[3/3] Injecting interventions to clinical.intervention-events..."
python3 -c "
import json
with open('$FILE') as f:
    data = json.load(f)
for event in data['inject']['clinical.intervention-events']:
    key = event['patientId']
    value = json.dumps(event)
    print(f'{key}|{value}')
" | kafka-console-producer.sh \
    --bootstrap-server $BOOTSTRAP \
    --topic clinical.intervention-events \
    --property parse.key=true \
    --property key.separator='|'

EVENT_COUNT=$(python3 -c "import json; d=json.load(open('$FILE')); print(sum(len(v) for v in d['inject'].values()))")
echo ""
echo "✓ Injected $EVENT_COUNT events for Day $DAY"
echo ""
echo "═══════════════════════════════════════════════════════"
echo "  ASSERTIONS TO CHECK (see playbook for full list):"
echo "═══════════════════════════════════════════════════════"
python3 -c "
import json
with open('$FILE') as f:
    data = json.load(f)
for module, info in data['assertions']['modules'].items():
    checks = info.get('checks', [])
    if checks:
        check_at = info.get('check_at', 'immediately')
        print(f'  {module} (check: {check_at}):')
        for c in checks:
            print(f'    □ {c}')
        print()
"
