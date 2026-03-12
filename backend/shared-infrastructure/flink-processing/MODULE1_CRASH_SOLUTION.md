# Module 1 Crash - Root Cause and Solution

## Problem

**Error**: Module 1 fails with `JsonEOFException: Unexpected end-of-input`

**Root Cause**: Malformed JSON message in one of the input Kafka topics (just `{` without closing bracket)

```
Caused by: com.fasterxml.jackson.core.io.JsonEOFException:
  Unexpected end-of-input: expected close marker for Object
  at [Source: (byte[])"{"; line: 1, column: 1]
```

**Impact**: Module 1 cannot start, blocking the entire pipeline

## Solution Options

### Option 1: Clean Kafka Topics (RECOMMENDED)

Delete all input topics and recreate them:

```bash
# Delete corrupted topics
for topic in patient-events-v1 medication-events-v1 observation-events-v1 \
             vital-signs-events-v1 lab-result-events-v1 validated-device-data-v1; do
  echo "Deleting topic: $topic"
  docker exec kafka kafka-topics --delete --topic $topic --bootstrap-server localhost:9092
done

# Recreate topics
for topic in patient-events-v1 medication-events-v1 observation-events-v1 \
             vital-signs-events-v1 lab-result-events-v1 validated-device-data-v1; do
  echo "Creating topic: $topic"
  docker exec kafka kafka-topics --create --topic $topic \
    --bootstrap-server localhost:9092 \
    --partitions 4 \
    --replication-factor 1
done
```

### Option 2: Skip Bad Messages

Modify Module1_Ingestion.java to skip malformed JSON:

```java
// In RawEventDeserializer.deserialize() - Line 410-420
public RawEvent deserialize(byte[] message) {
    try {
        return objectMapper.readValue(message, RawEvent.class);
    } catch (JsonProcessingException e) {
        LOG.error("Failed to deserialize message, skipping: {}",
            new String(message, StandardCharsets.UTF_8), e);
        return null;  // Return null for bad messages
    }
}
```

Then filter out nulls in the stream:

```java
DataStream<CanonicalEvent> canonicalEvents = rawEvents
    .filter(event -> event != null)  // Skip nulls
    .map(new ValidationFunction())
    .filter(event -> event != null);
```

### Option 3: Use Dead Letter Queue (PRODUCTION READY)

Route malformed messages to a DLQ topic for later investigation:

```java
// Add side output for DLQ
OutputTag<byte[]> dlqTag = new OutputTag<byte[]>("malformed-messages"){};

SingleOutputStreamOperator<RawEvent> rawEvents = kafkaSource
    .process(new ProcessFunction<ConsumerRecord<String, byte[]>, RawEvent>() {
        @Override
        public void processElement(
                ConsumerRecord<String, byte[]> record,
                Context ctx,
                Collector<RawEvent> out) {
            try {
                RawEvent event = objectMapper.readValue(record.value(), RawEvent.class);
                out.collect(event);
            } catch (JsonProcessingException e) {
                LOG.error("Malformed JSON, sending to DLQ: topic={}, partition={}, offset={}",
                    record.topic(), record.partition(), record.offset());
                ctx.output(dlqTag, record.value());
            }
        }
    });

// Write DLQ messages to Kafka
DataStream<byte[]> dlqStream = rawEvents.getSideOutput(dlqTag);
dlqStream.sinkTo(KafkaSink.<byte[]>builder()
    .setBootstrapServers(kafkaBootstrapServers)
    .setRecordSerializer(...)
    .setDeliverGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
    .build());
```

## Immediate Fix (Quick Start)

To get the pipeline working now:

```bash
# 1. Clean all input topics
bash clean-kafka-topics.sh

# 2. Restart Module 1
curl -s -X POST "http://localhost:8081/jars/LATEST_JAR/run" \
  -H "Content-Type: application/json" \
  -d '{
    "entryClass":"com.cardiofit.flink.operators.Module1_Ingestion",
    "parallelism":2,
    "programArgs":"--module ingestion --environment development"
  }'

# 3. Run test
./test-first-time-patient.sh
```

## Prevention

### Input Validation Script

Create a script to validate JSON before sending to Kafka:

```bash
#!/bin/bash
# validate-and-send.sh

EVENT_JSON="$1"
TOPIC="$2"

# Validate JSON
if echo "$EVENT_JSON" | jq empty 2>/dev/null; then
    echo "$EVENT_JSON" | docker exec -i kafka kafka-console-producer \
        --bootstrap-server localhost:9092 \
        --topic "$TOPIC"
    echo "✓ Sent to $TOPIC"
else
    echo "✗ Invalid JSON, not sent"
    exit 1
fi
```

Usage:
```bash
./validate-and-send.sh '{"id":"evt-001","patient_id":"P-123",...}' patient-events-v1
```

### Schema Validation

Add Avro schema validation using Confluent Schema Registry (already in dependencies):

```java
// Instead of JSON deserialization
KafkaRecordDeserializationSchema<RawEvent> deserializer =
    KafkaAvroDeserializationSchema.forSpecific(
        RawEvent.class,
        schemaRegistryUrl
    );
```

This ensures all messages are validated against a schema before entering the pipeline.

## Test After Fix

```bash
# Send valid test event
cat <<EOF | docker exec -i kafka kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic patient-events-v1
{
  "id": "test-001",
  "patient_id": "P-TEST-001",
  "event_time": $(date +%s)000,
  "type": "admission",
  "payload": {"department": "ER"},
  "metadata": {"source": "Test"}
}
EOF

# Check Module 1 processed it
docker logs cardiofit-flink-taskmanager-3 --since 10s | grep "P-TEST-001"
```

## Recommended Solution

**For Development**: Option 1 (Clean topics) - fastest recovery

**For Production**: Option 3 (DLQ) - proper error handling with observability

The DLQ approach provides:
- ✅ Resilience: Pipeline continues despite bad messages
- ✅ Observability: All malformed messages logged and saved
- ✅ Recovery: DLQ messages can be fixed and replayed
- ✅ Metrics: Track DLQ rate to detect data quality issues
