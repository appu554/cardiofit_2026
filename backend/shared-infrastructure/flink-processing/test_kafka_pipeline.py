#!/usr/bin/env python3
"""
Kafka Pipeline Testing Tool
Sends events to Kafka and displays enriched output from Flink processing
"""

import json
import time
import subprocess
import sys
from datetime import datetime

# ANSI color codes for pretty output
GREEN = '\033[92m'
BLUE = '\033[94m'
YELLOW = '\033[93m'
RED = '\033[91m'
BOLD = '\033[1m'
RESET = '\033[0m'

def print_header(text):
    """Print a colored header"""
    print(f"\n{BOLD}{BLUE}{'='*70}{RESET}")
    print(f"{BOLD}{BLUE}{text}{RESET}")
    print(f"{BOLD}{BLUE}{'='*70}{RESET}\n")

def print_success(text):
    """Print success message"""
    print(f"{GREEN}✅ {text}{RESET}")

def print_info(text):
    """Print info message"""
    print(f"{BLUE}ℹ️  {text}{RESET}")

def print_warning(text):
    """Print warning message"""
    print(f"{YELLOW}⚠️  {text}{RESET}")

def print_error(text):
    """Print error message"""
    print(f"{RED}❌ {text}{RESET}")

def get_current_timestamp():
    """Get current timestamp in milliseconds"""
    return int(time.time() * 1000)

def send_to_kafka(topic, event_data):
    """Send event to Kafka topic"""
    try:
        # Convert event to single-line JSON
        json_str = json.dumps(event_data, separators=(',', ':'))

        # Send to Kafka using docker exec
        cmd = [
            'docker', 'exec', '-i', 'kafka',
            'kafka-console-producer',
            '--bootstrap-server', 'localhost:9092',
            '--topic', topic
        ]

        process = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )

        stdout, stderr = process.communicate(input=json_str)

        if process.returncode == 0:
            return True, "Event sent successfully"
        else:
            return False, stderr

    except Exception as e:
        return False, str(e)

def get_topic_message_count(topic):
    """Get total message count in a Kafka topic"""
    try:
        cmd = [
            'docker', 'exec', 'kafka',
            'kafka-run-class', 'kafka.tools.GetOffsetShell',
            '--broker-list', 'localhost:9092',
            '--topic', topic,
            '--time', '-1'
        ]

        result = subprocess.run(cmd, capture_output=True, text=True)

        if result.returncode == 0:
            # Parse output like "topic:partition:offset"
            total = 0
            for line in result.stdout.strip().split('\n'):
                if line:
                    parts = line.split(':')
                    if len(parts) >= 3:
                        total += int(parts[2])
            return total
        else:
            return -1

    except Exception as e:
        print_error(f"Error getting message count: {e}")
        return -1

def read_enriched_messages(max_messages=5):
    """Read enriched messages from output topic"""
    try:
        cmd = [
            'docker', 'exec', 'kafka',
            'kafka-console-consumer',
            '--bootstrap-server', 'localhost:9092',
            '--topic', 'enriched-patient-events-v1',
            '--from-beginning',
            '--max-messages', str(max_messages),
            '--timeout-ms', '5000'
        ]

        result = subprocess.run(cmd, capture_output=True, text=True, timeout=10)

        messages = []
        for line in result.stdout.strip().split('\n'):
            if line and not line.startswith('Processed'):
                try:
                    messages.append(json.loads(line))
                except:
                    pass

        return messages

    except subprocess.TimeoutExpired:
        print_warning("Consumer timed out (this is normal)")
        return []
    except Exception as e:
        print_error(f"Error reading messages: {e}")
        return []

def display_event(event, title="Event"):
    """Display event in pretty format"""
    print(f"\n{BOLD}{title}:{RESET}")
    print(json.dumps(event, indent=2))

# ============================================================================
# SAMPLE EVENTS - You can modify these or add your own!
# ============================================================================

def create_vital_signs_event(patient_id="P12345"):
    """Create a vital signs event"""
    return {
        "patient_id": patient_id,
        "event_time": get_current_timestamp(),
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

def create_medication_event(patient_id="P12345"):
    """Create a medication administration event"""
    return {
        "patient_id": patient_id,
        "event_time": get_current_timestamp(),
        "type": "medication",
        "payload": {
            "medication_name": "Aspirin",
            "dosage": "100mg",
            "route": "oral",
            "frequency": "once daily"
        },
        "metadata": {
            "source": "Python Test Script",
            "administered_by": "Nurse Smith",
            "location": "Room 302"
        }
    }

def create_lab_result_event(patient_id="P12345"):
    """Create a lab result event"""
    return {
        "patient_id": patient_id,
        "event_time": get_current_timestamp(),
        "type": "lab_result",
        "payload": {
            "test_name": "Complete Blood Count",
            "hemoglobin": 14.5,
            "white_blood_cells": 7200,
            "platelets": 250000,
            "hematocrit": 42.5
        },
        "metadata": {
            "source": "Python Test Script",
            "lab_name": "Central Lab",
            "technician": "Tech-789"
        }
    }

def create_observation_event(patient_id="P12345"):
    """Create a clinical observation event"""
    return {
        "patient_id": patient_id,
        "event_time": get_current_timestamp(),
        "type": "observation",
        "payload": {
            "observation_type": "Blood Glucose",
            "value": 105,
            "unit": "mg/dL",
            "status": "normal"
        },
        "metadata": {
            "source": "Python Test Script",
            "observer": "Dr. Johnson"
        }
    }

# ============================================================================
# MAIN FUNCTIONS
# ============================================================================

def send_single_event(event_type="vital_signs", patient_id="P12345", topic="patient-events-v1"):
    """Send a single event to Kafka"""

    # Create event based on type
    event_creators = {
        "vital_signs": create_vital_signs_event,
        "medication": create_medication_event,
        "lab_result": create_lab_result_event,
        "observation": create_observation_event
    }

    if event_type not in event_creators:
        print_error(f"Unknown event type: {event_type}")
        print_info(f"Available types: {', '.join(event_creators.keys())}")
        return False

    event = event_creators[event_type](patient_id)

    print_header(f"Sending {event_type.upper()} Event")
    display_event(event, "Event Data")

    print(f"\n{BLUE}📤 Sending to topic: {topic}{RESET}")
    success, message = send_to_kafka(topic, event)

    if success:
        print_success(message)
        return True
    else:
        print_error(f"Failed to send: {message}")
        return False

def send_batch_events(patient_id="P12345"):
    """Send a batch of different event types"""

    print_header("Sending Batch of Events")

    events_to_send = [
        ("vital_signs", "patient-events-v1"),
        ("medication", "medication-events-v1"),
        ("lab_result", "observation-events-v1"),
        ("observation", "observation-events-v1")
    ]

    sent_count = 0

    for event_type, topic in events_to_send:
        print(f"\n{BLUE}📨 Sending {event_type} event...{RESET}")

        event_creators = {
            "vital_signs": create_vital_signs_event,
            "medication": create_medication_event,
            "lab_result": create_lab_result_event,
            "observation": create_observation_event
        }

        event = event_creators[event_type](patient_id)
        success, message = send_to_kafka(topic, event)

        if success:
            print_success(f"{event_type} sent to {topic}")
            sent_count += 1
        else:
            print_error(f"Failed to send {event_type}: {message}")

        time.sleep(0.5)  # Small delay between events

    print(f"\n{GREEN}✅ Sent {sent_count}/{len(events_to_send)} events{RESET}")
    return sent_count > 0

def check_processing(wait_seconds=5):
    """Check if events were processed by Flink"""

    print_header("Checking Flink Processing")

    print_info(f"Waiting {wait_seconds} seconds for Flink to process...")
    time.sleep(wait_seconds)

    # Check input topics
    print(f"\n{BOLD}Input Topics:{RESET}")
    input_topics = [
        "patient-events-v1",
        "medication-events-v1",
        "observation-events-v1"
    ]

    for topic in input_topics:
        count = get_topic_message_count(topic)
        if count >= 0:
            print(f"  {topic}: {count} messages")

    # Check output topic
    print(f"\n{BOLD}Output Topic:{RESET}")
    output_count = get_topic_message_count("enriched-patient-events-v1")
    if output_count >= 0:
        print(f"  enriched-patient-events-v1: {GREEN}{output_count} messages{RESET}")

    # Check DLQ
    print(f"\n{BOLD}Dead Letter Queue (Errors):{RESET}")
    dlq_count = get_topic_message_count("dlq.processing-errors.v1")
    if dlq_count >= 0:
        if dlq_count == 0:
            print_success("No errors - all events validated successfully!")
        else:
            print_warning(f"{dlq_count} failed events in DLQ")

def view_enriched_output(max_messages=3):
    """View enriched messages from output topic"""

    print_header("Viewing Enriched Output")

    print_info(f"Reading last {max_messages} enriched messages...")
    print_warning("Note: Console consumer may hang - use Kafka UI for better experience")

    print(f"\n{BOLD}Recommended: View in Kafka UI{RESET}")
    print(f"  1. Open: {BLUE}http://localhost:8080{RESET}")
    print(f"  2. Click: Topics → enriched-patient-events-v1")
    print(f"  3. Click: Messages tab")
    print()

def show_monitoring_info():
    """Show monitoring URLs and commands"""

    print_header("Monitoring & Viewing Tools")

    print(f"{BOLD}Kafka UI (Recommended):{RESET}")
    print(f"  URL: {BLUE}http://localhost:8080{RESET}")
    print(f"  - Browse topics")
    print(f"  - View messages in JSON format")
    print(f"  - See consumer groups")

    print(f"\n{BOLD}Flink Web UI:{RESET}")
    print(f"  URL: {BLUE}http://localhost:8081{RESET}")
    print(f"  - View job metrics")
    print(f"  - Check processing stats")
    print(f"  - Monitor exceptions")

    print(f"\n{BOLD}Useful Commands:{RESET}")
    print(f"  Check Flink job:")
    print(f"    docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r")

    print(f"\n  Check topic messages:")
    print(f"    docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \\")
    print(f"      --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1")

# ============================================================================
# INTERACTIVE MENU
# ============================================================================

def show_menu():
    """Show interactive menu"""
    print(f"\n{BOLD}{BLUE}╔════════════════════════════════════════════════════════════╗{RESET}")
    print(f"{BOLD}{BLUE}║     Kafka Pipeline Testing Tool - Interactive Menu        ║{RESET}")
    print(f"{BOLD}{BLUE}╚════════════════════════════════════════════════════════════╝{RESET}\n")

    print(f"{BOLD}Options:{RESET}")
    print(f"  {GREEN}1{RESET}. Send single vital signs event")
    print(f"  {GREEN}2{RESET}. Send single medication event")
    print(f"  {GREEN}3{RESET}. Send single lab result event")
    print(f"  {GREEN}4{RESET}. Send batch of all event types")
    print(f"  {GREEN}5{RESET}. Check processing status")
    print(f"  {GREEN}6{RESET}. View enriched output (in Kafka UI)")
    print(f"  {GREEN}7{RESET}. Show monitoring info")
    print(f"  {GREEN}8{RESET}. Send custom event (you provide JSON)")
    print(f"  {GREEN}0{RESET}. Exit")
    print()

def send_custom_event():
    """Allow user to send custom event"""
    print_header("Send Custom Event")

    print("Available topics:")
    topics = [
        "patient-events-v1",
        "medication-events-v1",
        "observation-events-v1",
        "vital-signs-events-v1",
        "lab-result-events-v1"
    ]

    for i, topic in enumerate(topics, 1):
        print(f"  {i}. {topic}")

    topic_choice = input(f"\nSelect topic (1-{len(topics)}): ").strip()

    try:
        topic_idx = int(topic_choice) - 1
        if topic_idx < 0 or topic_idx >= len(topics):
            print_error("Invalid topic selection")
            return

        topic = topics[topic_idx]

        print("\nEnter your event JSON (or press Ctrl+C to cancel):")
        print("Example:")
        print(json.dumps(create_vital_signs_event(), indent=2))
        print("\nYour JSON:")

        lines = []
        try:
            while True:
                line = input()
                lines.append(line)
        except EOFError:
            pass

        json_str = '\n'.join(lines)
        event = json.loads(json_str)

        # Validate required fields
        if 'patient_id' not in event:
            print_error("Missing required field: patient_id")
            return

        if 'event_time' not in event:
            print_warning("No event_time provided, using current timestamp")
            event['event_time'] = get_current_timestamp()

        success, message = send_to_kafka(topic, event)

        if success:
            print_success("Custom event sent successfully!")
        else:
            print_error(f"Failed: {message}")

    except ValueError as e:
        print_error(f"Invalid JSON: {e}")
    except Exception as e:
        print_error(f"Error: {e}")

def interactive_mode():
    """Run interactive menu"""

    while True:
        show_menu()

        choice = input(f"{BOLD}Enter choice (0-8): {RESET}").strip()

        if choice == '0':
            print_info("Exiting...")
            break

        elif choice == '1':
            patient_id = input("Enter patient ID (default: P12345): ").strip() or "P12345"
            send_single_event("vital_signs", patient_id, "patient-events-v1")

        elif choice == '2':
            patient_id = input("Enter patient ID (default: P12345): ").strip() or "P12345"
            send_single_event("medication", patient_id, "medication-events-v1")

        elif choice == '3':
            patient_id = input("Enter patient ID (default: P12345): ").strip() or "P12345"
            send_single_event("lab_result", patient_id, "observation-events-v1")

        elif choice == '4':
            patient_id = input("Enter patient ID (default: P12345): ").strip() or "P12345"
            send_batch_events(patient_id)

        elif choice == '5':
            check_processing()

        elif choice == '6':
            view_enriched_output()

        elif choice == '7':
            show_monitoring_info()

        elif choice == '8':
            send_custom_event()

        else:
            print_error("Invalid choice. Please try again.")

        input(f"\n{BLUE}Press Enter to continue...{RESET}")

# ============================================================================
# COMMAND LINE MODE
# ============================================================================

def main():
    """Main entry point"""

    if len(sys.argv) == 1:
        # No arguments - run interactive mode
        interactive_mode()
    else:
        # Command line arguments
        command = sys.argv[1].lower()

        if command == "send":
            event_type = sys.argv[2] if len(sys.argv) > 2 else "vital_signs"
            patient_id = sys.argv[3] if len(sys.argv) > 3 else "P12345"

            topics = {
                "vital_signs": "patient-events-v1",
                "medication": "medication-events-v1",
                "lab_result": "observation-events-v1",
                "observation": "observation-events-v1"
            }

            topic = topics.get(event_type, "patient-events-v1")
            send_single_event(event_type, patient_id, topic)

        elif command == "batch":
            patient_id = sys.argv[2] if len(sys.argv) > 2 else "P12345"
            send_batch_events(patient_id)

        elif command == "check":
            check_processing()

        elif command == "monitor":
            show_monitoring_info()

        else:
            print(f"Usage:")
            print(f"  {sys.argv[0]}                          # Interactive mode")
            print(f"  {sys.argv[0]} send [type] [patient_id] # Send single event")
            print(f"  {sys.argv[0]} batch [patient_id]       # Send batch")
            print(f"  {sys.argv[0]} check                    # Check processing")
            print(f"  {sys.argv[0]} monitor                  # Show monitoring info")

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print(f"\n\n{YELLOW}Interrupted by user{RESET}")
        sys.exit(0)
    except Exception as e:
        print_error(f"Unexpected error: {e}")
        sys.exit(1)
