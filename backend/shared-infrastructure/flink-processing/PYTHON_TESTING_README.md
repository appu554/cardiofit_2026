# Python Kafka Pipeline Testing Tool

An easy-to-use Python script for testing the Flink EHR pipeline by sending events to Kafka and viewing enriched output.

## Quick Start

### Interactive Mode (Recommended)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
python3 test_kafka_pipeline.py
```

This opens an interactive menu where you can:
- Send different types of healthcare events
- Check processing status
- View enriched output
- Monitor the pipeline

### Command Line Mode

```bash
# Send a single vital signs event
python3 test_kafka_pipeline.py send vital_signs P12345

# Send a batch of events
python3 test_kafka_pipeline.py batch P12345

# Check processing status
python3 test_kafka_pipeline.py check

# Show monitoring info
python3 test_kafka_pipeline.py monitor
```

## Interactive Menu Options

When you run in interactive mode, you'll see:

```
╔════════════════════════════════════════════════════════════╗
║     Kafka Pipeline Testing Tool - Interactive Menu        ║
╚════════════════════════════════════════════════════════════╝

Options:
  1. Send single vital signs event
  2. Send single medication event
  3. Send single lab result event
  4. Send batch of all event types
  5. Check processing status
  6. View enriched output (in Kafka UI)
  7. Show monitoring info
  8. Send custom event (you provide JSON)
  0. Exit

Enter choice (0-8):
```

## Usage Examples

### Example 1: Send Vital Signs Event

```bash
python3 test_kafka_pipeline.py
# Choose option 1
# Enter patient ID: P12345
```

**What it sends:**
```json
{
  "patient_id": "P12345",
  "event_time": 1759304715000,
  "type": "vital_signs",
  "payload": {
    "heart_rate": 78,
    "blood_pressure": "120/80",
    "temperature": 98.6,
    "respiratory_rate": 16,
    "oxygen_saturation": 98
  },
  "metadata": {
    "source": "Python Test Script",
    "location": "ICU Ward",
    "device_id": "MON-001"
  }
}
```

### Example 2: Send Batch of Events

```bash
python3 test_kafka_pipeline.py batch P99999
```

This sends 4 different event types:
- Vital signs → patient-events-v1
- Medication → medication-events-v1
- Lab result → observation-events-v1
- Clinical observation → observation-events-v1

### Example 3: Check Processing

```bash
python3 test_kafka_pipeline.py check
```

**Output:**
```
Input Topics:
  patient-events-v1: 15 messages
  medication-events-v1: 12 messages
  observation-events-v1: 20 messages

Output Topic:
  enriched-patient-events-v1: 18 messages ✅

Dead Letter Queue (Errors):
✅ No errors - all events validated successfully!
```

### Example 4: Send Custom Event

Choose option 8 in interactive mode, then paste your custom JSON:

```json
{
  "patient_id": "CUSTOM-001",
  "event_time": 1759304800000,
  "type": "custom_observation",
  "payload": {
    "measurement": "Pain Scale",
    "value": 3,
    "notes": "Patient reports mild discomfort"
  },
  "metadata": {
    "source": "Manual Entry",
    "assessed_by": "Nurse Johnson"
  }
}
```

Press `Ctrl+D` (Mac/Linux) or `Ctrl+Z` then Enter (Windows) when done.

## Viewing Enriched Output

The script recommends using **Kafka UI** for the best experience:

1. **Open Kafka UI**: http://localhost:8080
2. **Navigate**: Click "Topics" in sidebar
3. **Select Topic**: Click "enriched-patient-events-v1"
4. **View Messages**: Click "Messages" tab
5. **See Enrichment**: Compare input vs output to see what Flink added

**What you'll see in enriched events:**
- Original patient data
- Added `processing_time` timestamp
- Added `ingestion_metadata` (source, time, Flink subtask)
- Normalized payload fields (e.g., `blood-pressure` → `blood_pressure`)
- Auto-generated event IDs if missing

## Modifying Event Data

You can easily customize the events by editing the functions in the script:

### Edit Vital Signs Event

Open `test_kafka_pipeline.py` and find:

```python
def create_vital_signs_event(patient_id="P12345"):
    """Create a vital signs event"""
    return {
        "patient_id": patient_id,
        "event_time": get_current_timestamp(),
        "type": "vital_signs",
        "payload": {
            "heart_rate": 78,  # Change this value
            "blood_pressure": "120/80",  # Change this
            # Add more fields here:
            "pulse_oximetry": 99,
            "respiratory_rate": 18
        },
        "metadata": {
            "source": "Python Test Script",
            "location": "ICU Ward",  # Customize location
            "device_id": "MON-001"
        }
    }
```

### Add New Event Type

Add your own event creator function:

```python
def create_allergy_event(patient_id="P12345"):
    """Create an allergy documentation event"""
    return {
        "patient_id": patient_id,
        "event_time": get_current_timestamp(),
        "type": "allergy",
        "payload": {
            "allergen": "Penicillin",
            "reaction": "Rash",
            "severity": "Moderate",
            "documented_date": "2024-01-15"
        },
        "metadata": {
            "source": "Python Test Script",
            "documented_by": "Dr. Smith"
        }
    }
```

Then add it to the `event_creators` dictionary in `send_single_event()` function.

## Event Validation Rules

Your events must pass these checks:

✅ **Required Fields:**
- `patient_id` - must exist and not be empty
- `event_time` - must be > 0 (milliseconds since epoch)
- `type` - must exist and not be empty
- `payload` - must exist and not be empty

✅ **Time Validation:**
- Event time must not be > 1 hour in the future
- Event time must not be > 30 days in the past

❌ **Invalid Events:**
Events that fail validation are sent to the DLQ topic: `dlq.processing-errors.v1`

## Troubleshooting

### Script says "docker: command not found"

Make sure Docker is installed and running:
```bash
docker ps
```

### No messages appearing in enriched topic

1. Check Flink job is running:
```bash
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r
```

2. Check for Flink exceptions:
- Open http://localhost:8081
- Click on running job
- Check "Exceptions" tab

3. Check your event timestamp is current:
```python
# The script uses get_current_timestamp() which is correct
# Don't use old/hardcoded timestamps
```

### Kafka UI not accessible

Start Kafka UI container:
```bash
docker run -d --rm -p 8080:8080 \
  --network kafka_cardiofit-network \
  -e KAFKA_CLUSTERS_0_NAME=cardiofit \
  -e KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=kafka:9092 \
  --name kafka-ui \
  provectuslabs/kafka-ui:latest
```

## Available Event Types

The script includes pre-built event creators for:

| Event Type | Topic | Description |
|------------|-------|-------------|
| `vital_signs` | patient-events-v1 | Heart rate, BP, temperature, SpO2 |
| `medication` | medication-events-v1 | Medication administration |
| `lab_result` | observation-events-v1 | Laboratory test results |
| `observation` | observation-events-v1 | Clinical observations |

You can send to additional topics:
- `vital-signs-events-v1`
- `lab-result-events-v1`
- `validated-device-data-v1`

All events get enriched and output to: **enriched-patient-events-v1**

## Advanced Usage

### Custom Patient ID

```bash
# Interactive mode
python3 test_kafka_pipeline.py
# Choose option 1
# Enter patient ID: CUSTOM-12345

# Command line
python3 test_kafka_pipeline.py send vital_signs CUSTOM-12345
```

### Batch Testing with Different Patients

```bash
# Send batch for patient P001
python3 test_kafka_pipeline.py batch P001

# Send batch for patient P002
python3 test_kafka_pipeline.py batch P002

# Check processing
python3 test_kafka_pipeline.py check
```

### Automated Testing Loop

Create a bash script:
```bash
#!/bin/bash
for i in {1..10}; do
    echo "Sending batch $i..."
    python3 test_kafka_pipeline.py batch "P-TEST-$i"
    sleep 2
done

python3 test_kafka_pipeline.py check
```

## Monitoring Tools

### Kafka UI (Best for viewing messages)
- **URL**: http://localhost:8080
- **Features**: Browse topics, view JSON messages, search

### Flink Web UI (Best for metrics)
- **URL**: http://localhost:8081
- **Features**: Job metrics, processing stats, exceptions

### Command Line Tools
```bash
# List running Flink jobs
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r

# Check topic message counts
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1

# Check consumer groups
docker exec kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 --list
```

## Summary

**Quick Test Workflow:**
1. Run `python3 test_kafka_pipeline.py`
2. Choose option 4 (send batch)
3. Choose option 5 (check processing)
4. Open http://localhost:8080 to view enriched events

**Customize Events:**
- Edit event creator functions in the script
- Or use option 8 to send custom JSON

**Monitor Pipeline:**
- Kafka UI for messages: http://localhost:8080
- Flink UI for metrics: http://localhost:8081
- Script option 7 for monitoring info

Happy testing! 🚀
