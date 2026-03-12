#!/usr/bin/env python3
"""
Module Integration Testing Script
Tests individual modules and their integration
"""

import subprocess
import json
import time
import sys

# Color codes
GREEN = '\033[92m'
BLUE = '\033[94m'
YELLOW = '\033[93m'
RED = '\033[91m'
BOLD = '\033[1m'
RESET = '\033[0m'

def print_header(text):
    print(f"\n{BOLD}{BLUE}{'='*70}{RESET}")
    print(f"{BOLD}{BLUE}{text}{RESET}")
    print(f"{BOLD}{BLUE}{'='*70}{RESET}\n")

def print_success(text):
    print(f"{GREEN}✅ {text}{RESET}")

def print_error(text):
    print(f"{RED}❌ {text}{RESET}")

def print_info(text):
    print(f"{BLUE}ℹ️  {text}{RESET}")

def get_topic_message_count(topic):
    """Get message count in a Kafka topic"""
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
            total = 0
            for line in result.stdout.strip().split('\n'):
                if line:
                    parts = line.split(':')
                    if len(parts) >= 3:
                        total += int(parts[2])
            return total
        return -1
    except Exception as e:
        print_error(f"Error getting count for {topic}: {e}")
        return -1

def get_running_jobs():
    """Get list of running Flink jobs"""
    try:
        cmd = [
            'docker', 'exec', 'cardiofit-flink-jobmanager',
            '/opt/flink/bin/flink', 'list', '-r'
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        return result.stdout
    except Exception as e:
        print_error(f"Error getting jobs: {e}")
        return ""

def test_module1():
    """Test Module 1 independently"""
    print_header("Testing Module 1: Ingestion")

    # Check if Module 1 is running
    jobs = get_running_jobs()
    if "ingestion-only" in jobs.lower():
        print_success("Module 1 is running")
    else:
        print_error("Module 1 is NOT running")
        print_info("Start with: bash submit-job.sh ingestion-only development")
        return False

    # Check input topics
    print("\n📥 Input Topics:")
    input_topics = [
        "patient-events-v1",
        "medication-events-v1",
        "observation-events-v1"
    ]

    for topic in input_topics:
        count = get_topic_message_count(topic)
        if count >= 0:
            print(f"  {topic}: {count} messages")

    # Check Module 1 output
    print("\n📤 Module 1 Output:")
    m1_output_count = get_topic_message_count("enriched-patient-events-v1")
    if m1_output_count > 0:
        print_success(f"enriched-patient-events-v1: {m1_output_count} messages")
        return True
    else:
        print_error("No messages in enriched-patient-events-v1")
        print_info("Send test events with: python3 test_kafka_pipeline.py")
        return False

def test_module2():
    """Test Module 2 independently"""
    print_header("Testing Module 2: Context Assembly")

    # Check if Module 1 output exists (Module 2's input)
    m1_output = get_topic_message_count("enriched-patient-events-v1")
    if m1_output <= 0:
        print_error("Module 2 needs Module 1 output to process")
        print_info("Run Module 1 first to create events in enriched-patient-events-v1")
        return False

    print_success(f"Module 1 output available: {m1_output} events")

    # Check if Module 2 is running
    jobs = get_running_jobs()
    if "context-assembly" in jobs.lower():
        print_success("Module 2 is running")
    else:
        print_error("Module 2 is NOT running")
        print_info("Start with:")
        print(f"  docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run \\")
        print(f"    --detached \\")
        print(f"    --class com.cardiofit.flink.FlinkJobOrchestrator \\")
        print(f"    /opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \\")
        print(f"    context-assembly development")
        return False

    # Check Module 2 output
    print("\n📤 Module 2 Output:")
    m2_output_count = get_topic_message_count("context-enriched-events-v1")
    if m2_output_count > 0:
        print_success(f"context-enriched-events-v1: {m2_output_count} messages")
        return True
    else:
        print_error("No messages in context-enriched-events-v1")
        print_info("Wait for Module 2 to process events, or check for errors")
        return False

def test_integration_m1_m2():
    """Test Module 1 + Module 2 integration"""
    print_header("Testing Module 1 + Module 2 Integration")

    print("📊 Data Flow Check:")
    print("\nExpected flow:")
    print("  Input → Module 1 → enriched-patient-events-v1 → Module 2 → context-enriched-events-v1")

    # Check each stage
    stages = {
        "Module 1 Input": "patient-events-v1",
        "Module 1 Output": "enriched-patient-events-v1",
        "Module 2 Output": "context-enriched-events-v1"
    }

    print("\n📈 Message Counts:")
    all_ok = True
    for stage, topic in stages.items():
        count = get_topic_message_count(topic)
        if count > 0:
            print_success(f"{stage} ({topic}): {count} messages")
        else:
            print_error(f"{stage} ({topic}): 0 messages")
            all_ok = False

    if all_ok:
        print("\n" + "="*70)
        print_success("Integration Test PASSED: Data flowing through both modules!")
        print("="*70)
    else:
        print("\n" + "="*70)
        print_error("Integration Test FAILED: Check missing stages above")
        print("="*70)

    return all_ok

def test_full_pipeline():
    """Test full pipeline (all 6 modules)"""
    print_header("Testing Full Pipeline (All 6 Modules)")

    jobs = get_running_jobs()
    if "full-pipeline" in jobs.lower():
        print_success("Full pipeline is running")
    else:
        print_error("Full pipeline is NOT running")
        print_info("Start with: bash submit-job.sh full-pipeline production")
        return False

    # Check all intermediate topics
    print("\n📊 All Pipeline Topics:")
    all_topics = {
        "M1 Output": "enriched-patient-events-v1",
        "M2 Output": "context-enriched-events-v1",
        "M3 Output": "semantic-enriched-events-v1",
        "M4 Output": "pattern-detected-events-v1",
        "M5 Output": "ml-inference-events-v1",
        "M6 Output": "final-output-events-v1"
    }

    for stage, topic in all_topics.items():
        count = get_topic_message_count(topic)
        if count >= 0:
            if count > 0:
                print_success(f"{stage} ({topic}): {count} messages")
            else:
                print_info(f"{stage} ({topic}): 0 messages (may not exist yet)")

    return True

def show_module_status():
    """Show status of all modules"""
    print_header("Module Status Overview")

    print("🔍 Running Flink Jobs:")
    jobs = get_running_jobs()
    if jobs.strip():
        print(jobs)
    else:
        print_error("No Flink jobs running")

    print("\n📦 Kafka Topics Status:")
    topics = [
        ("Input", "patient-events-v1"),
        ("M1 Output", "enriched-patient-events-v1"),
        ("M2 Output", "context-enriched-events-v1"),
        ("DLQ", "dlq.processing-errors.v1")
    ]

    for name, topic in topics:
        count = get_topic_message_count(topic)
        if count >= 0:
            print(f"  {name:15} ({topic:35}): {count:4} messages")

def run_progressive_test():
    """Run progressive testing: M1 → M2 → Integration"""
    print_header("Progressive Module Testing")

    print("This will test modules progressively:")
    print("  1. Module 1 alone")
    print("  2. Module 2 alone")
    print("  3. Modules 1+2 integration")

    input("\nPress Enter to start testing...")

    # Test Module 1
    m1_ok = test_module1()
    if not m1_ok:
        print_error("\nModule 1 test failed. Fix Module 1 before proceeding.")
        return

    input("\nModule 1 test complete. Press Enter to test Module 2...")

    # Test Module 2
    m2_ok = test_module2()
    if not m2_ok:
        print_error("\nModule 2 test failed. Check Module 2 setup.")
        return

    input("\nModule 2 test complete. Press Enter to test integration...")

    # Test integration
    integration_ok = test_integration_m1_m2()

    if integration_ok:
        print("\n" + "="*70)
        print_success("ALL TESTS PASSED! Modules 1 and 2 are working correctly.")
        print("="*70)
        print("\nNext steps:")
        print("  1. Test Module 3 (Semantic Mesh)")
        print("  2. Test Module 4 (Pattern Detection)")
        print("  3. Test Module 5 (ML Inference)")
        print("  4. Test Module 6 (Egress Routing)")
        print("  5. Test full pipeline (all 6 modules)")
    else:
        print_error("\nIntegration test failed. Review the flow above.")

def show_menu():
    """Show interactive menu"""
    print(f"\n{BOLD}{BLUE}╔════════════════════════════════════════════════════════╗{RESET}")
    print(f"{BOLD}{BLUE}║     Module Integration Testing - Menu                 ║{RESET}")
    print(f"{BOLD}{BLUE}╚════════════════════════════════════════════════════════╝{RESET}\n")

    print(f"{BOLD}Test Individual Modules:{RESET}")
    print(f"  {GREEN}1{RESET}. Test Module 1 (Ingestion)")
    print(f"  {GREEN}2{RESET}. Test Module 2 (Context Assembly)")

    print(f"\n{BOLD}Test Integration:{RESET}")
    print(f"  {GREEN}3{RESET}. Test Module 1 + 2 Integration")
    print(f"  {GREEN}4{RESET}. Test Full Pipeline (All 6 modules)")

    print(f"\n{BOLD}Utilities:{RESET}")
    print(f"  {GREEN}5{RESET}. Show module status")
    print(f"  {GREEN}6{RESET}. Run progressive test (M1 → M2 → Integration)")
    print(f"  {GREEN}0{RESET}. Exit")
    print()

def main():
    """Main entry point"""

    if len(sys.argv) > 1:
        # Command line mode
        command = sys.argv[1].lower()

        if command == "m1" or command == "module1":
            test_module1()
        elif command == "m2" or command == "module2":
            test_module2()
        elif command == "integration":
            test_integration_m1_m2()
        elif command == "full":
            test_full_pipeline()
        elif command == "status":
            show_module_status()
        elif command == "progressive":
            run_progressive_test()
        else:
            print(f"Usage:")
            print(f"  {sys.argv[0]}                # Interactive mode")
            print(f"  {sys.argv[0]} m1             # Test Module 1")
            print(f"  {sys.argv[0]} m2             # Test Module 2")
            print(f"  {sys.argv[0]} integration    # Test M1+M2")
            print(f"  {sys.argv[0]} full           # Test full pipeline")
            print(f"  {sys.argv[0]} status         # Show status")
            print(f"  {sys.argv[0]} progressive    # Progressive testing")
    else:
        # Interactive mode
        while True:
            show_menu()
            choice = input(f"{BOLD}Enter choice (0-6): {RESET}").strip()

            if choice == '0':
                print_info("Exiting...")
                break
            elif choice == '1':
                test_module1()
            elif choice == '2':
                test_module2()
            elif choice == '3':
                test_integration_m1_m2()
            elif choice == '4':
                test_full_pipeline()
            elif choice == '5':
                show_module_status()
            elif choice == '6':
                run_progressive_test()
            else:
                print_error("Invalid choice")

            input(f"\n{BLUE}Press Enter to continue...{RESET}")

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print(f"\n\n{YELLOW}Interrupted by user{RESET}")
        sys.exit(0)
    except Exception as e:
        print_error(f"Error: {e}")
        sys.exit(1)
