import asyncio
import sys
import os
import json
from pathlib import Path

# Add the parent directory to the Python path
current_dir = Path(__file__).parent
parent_dir = current_dir.parent
sys.path.insert(0, str(parent_dir.parent.parent))

from app.utils.hl7_to_fhir import process_hl7_message, parse_hl7_message, extract_adt_data
from app.models.hl7 import ADTMessage

async def test_hl7_processing():
    """Test HL7 message processing"""
    # Load sample ADT message
    sample_file = current_dir / "sample_adt_message.hl7"
    with open(sample_file, "r") as f:
        message_str = f.read()
    
    # Parse the message
    parsed_message, message_type = parse_hl7_message(message_str)
    print(f"Parsed message type: {message_type}")
    
    # Extract ADT data
    adt_message = extract_adt_data(parsed_message)
    print(f"Extracted ADT data: {adt_message}")
    
    # Convert to FHIR resources
    fhir_resources = process_hl7_message(message_str)
    print(f"Generated FHIR resources:")
    print(json.dumps(fhir_resources, indent=2))
    
    return fhir_resources

if __name__ == "__main__":
    # Run the test
    fhir_resources = asyncio.run(test_hl7_processing())
    
    # Print summary
    print("\nTest Summary:")
    for resource_type, resource in fhir_resources.items():
        print(f"- {resource_type} resource created with ID: {resource['id']}")
    
    print("\nTest completed successfully!")
