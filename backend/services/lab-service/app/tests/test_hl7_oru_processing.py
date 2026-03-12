import asyncio
import sys
import os
import json
from pathlib import Path

# Add the parent directory to the Python path
current_dir = Path(__file__).parent
parent_dir = current_dir.parent
sys.path.insert(0, str(parent_dir.parent.parent))

from app.utils.hl7_to_lab import process_hl7_message, parse_hl7_message, extract_oru_data

async def test_oru_processing():
    """Test HL7 ORU message processing"""
    # Load sample ORU message
    sample_file = current_dir / "sample_oru_message.hl7"
    with open(sample_file, "r") as f:
        message_str = f.read()
    
    # Parse the message
    parsed_message, message_type = parse_hl7_message(message_str)
    print(f"Parsed message type: {message_type}")
    
    # Extract ORU data
    oru_message = extract_oru_data(parsed_message)
    print(f"Extracted ORU data: {oru_message}")
    
    # Convert to lab data
    lab_data = process_hl7_message(message_str)
    print(f"Generated lab data:")
    print(json.dumps(lab_data, indent=2, default=str))
    
    return lab_data

if __name__ == "__main__":
    # Run the test
    loop = asyncio.get_event_loop()
    result = loop.run_until_complete(test_oru_processing())
    
    # Print summary
    print("\nSummary:")
    print(f"Number of lab tests: {len(result.get('lab_tests', []))}")
    if result.get('lab_panel'):
        print(f"Lab panel: {result['lab_panel']['panel_name']}")
    
    # Print each lab test
    for i, test in enumerate(result.get('lab_tests', [])):
        print(f"  - Test {i+1}: {test['test_name']} = {test['value']} {test['unit']}")
