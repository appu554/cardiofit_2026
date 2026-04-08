#!/bin/bash
# Quick checker — run anytime to see which modules have produced output
# Usage: bash scripts/check_module_outputs.sh
#   --watch   : auto-refresh every 60s
#   --detail  : show message content for modules with output

WATCH=false
DETAIL=false
for arg in "$@"; do
  case $arg in
    --watch) WATCH=true ;;
    --detail) DETAIL=true ;;
  esac
done

KAFKA_CONTAINER="cardiofit-kafka-lite"

check_once() {
  echo "══════════════════════════════════════════════════════════"
  echo "  MODULE OUTPUT CHECK — $(date '+%Y-%m-%d %H:%M:%S %Z')"
  echo "  UTC: $(date -u '+%H:%M:%S')  |  IST: $(TZ=Asia/Kolkata date '+%H:%M:%S')"
  echo "══════════════════════════════════════════════════════════"
  echo ""

  # Define modules and their topics
  declare -a MODULES=(
    "M7 |flink.bp-variability-metrics|immediate"
    "M8 |alerts.comorbidity-interactions|immediate"
    "M9 |flink.engagement-signals|23:59 UTC daily"
    "M10|flink.meal-response|meal + 3h05m"
    "M10b|flink.meal-patterns|Monday 00:00 UTC"
    "M11|flink.activity-response|activity_end + 2h05m"
    "M11b|flink.fitness-patterns|Monday 00:00 UTC"
    "M12|clinical.intervention-window-signals|immediate"
    "M12b|flink.intervention-deltas|on WINDOW_CLOSED"
    "M13|clinical.state-change-events|immediate"
  )

  for entry in "${MODULES[@]}"; do
    IFS='|' read -r name topic timer <<< "$entry"
    COUNT=$(docker exec $KAFKA_CONTAINER bash -c \
      "kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:29092 --topic $topic 2>/dev/null" \
      | awk -F: '{sum+=$3} END {print sum}')

    if [ "$COUNT" -gt 0 ]; then
      printf "  ✅ %-5s %3d messages  %-40s\n" "$name" "$COUNT" "$topic"

      # Show detail if requested
      if [ "$DETAIL" = true ]; then
        echo "    ── Latest message ──"
        docker exec $KAFKA_CONTAINER bash -c \
          "kafka-console-consumer --bootstrap-server localhost:29092 --topic $topic \
           --from-beginning --max-messages 1 --timeout-ms 3000 \
           --consumer-property group.id=check-$(date +%s) 2>/dev/null" \
          | python3 -c "
import sys, json
try:
    msg = json.loads(sys.stdin.read())
    # Print key fields based on module
    keys = list(msg.keys())[:6]
    for k in keys:
        v = msg[k]
        if isinstance(v, dict):
            v = '{...}'
        elif isinstance(v, list):
            v = f'[{len(v)} items]'
        elif isinstance(v, str) and len(v) > 60:
            v = v[:60] + '...'
        print(f'      {k}: {v}')
except: pass
" 2>/dev/null
        echo ""
      fi
    else
      printf "  ○  %-5s %3d messages  %-40s  (timer: %s)\n" "$name" "$COUNT" "$topic" "$timer"
    fi
  done

  echo ""

  # Show expected fire times
  echo "  ── Expected Timer Fire Times ──"
  echo "  M11 activity response : ~18:52 UTC  (00:22 IST)"
  echo "  M10 meal response     : ~19:27 UTC  (00:57 IST)"
  echo "  M9  engagement score  : 23:59 UTC   (05:29 IST)"
  echo "  M10b/M11b weekly      : Mon 00:00 UTC (next week)"
  echo "  M12 midpoint          : ~7 days from E2E run"
  echo ""

  # Check job health
  echo "  ── Job Health ──"
  RUNNING=$(docker exec cardiofit-flink-jobmanager bash -c "flink list -r 2>/dev/null" 2>&1 | grep -c "RUNNING")
  RESTARTING=$(docker exec cardiofit-flink-jobmanager bash -c "flink list -r 2>/dev/null" 2>&1 | grep -c "RESTARTING")
  echo "  Jobs RUNNING: $RUNNING/10   RESTARTING: $RESTARTING"

  if [ "$RESTARTING" -gt 0 ]; then
    echo "  ⚠️  Unhealthy jobs:"
    docker exec cardiofit-flink-jobmanager bash -c "flink list -r 2>/dev/null" 2>&1 | grep "RESTARTING"
  fi
  echo ""
}

if [ "$WATCH" = true ]; then
  while true; do
    clear
    check_once
    echo "  [Auto-refreshing every 60s — Ctrl+C to stop]"
    sleep 60
  done
else
  check_once
fi
