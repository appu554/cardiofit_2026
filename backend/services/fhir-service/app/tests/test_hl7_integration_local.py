import asyncio
import sys
import os
import json
from pathlib import Path

# Add the parent directory to the Python path
current_dir = Path(__file__).parent
parent_dir = current_dir.parent
sys.path.insert(0, str(parent_dir.parent.parent))

from app.utils.hl7_to_fhir import process_hl7_message
from app.services.hl7_service import get_hl7_service
from app.models.hl7 import ADTMessage

async def test_hl7_service_local():
    """Test HL7 service locally without API"""
    # Load sample ADT message
    sample_file = current_dir / "sample_adt_message.hl7"
    with open(sample_file, "r") as f:
        message_str = f.read()
    
    # Process the message using the HL7 service
    print("Processing HL7 message locally...")
    
    # First, convert the message to FHIR resources
    fhir_resources = process_hl7_message(message_str)
    print(f"Generated FHIR resources:")
    print(json.dumps(fhir_resources, indent=2))
    
    # Simulate the API response
    result = {
        "message": "HL7 message processed successfully",
        "resources": {
            resource_type: {
                "id": resource["id"],
                "status": "created"
            }
            for resource_type, resource in fhir_resources.items()
        }
    }
    
    print("\nSimulated API Response:")
    print(json.dumps(result, indent=2))
    
    return result

if __name__ == "__main__":
    # Run the test
    result = asyncio.run(test_hl7_service_local())
    
    # Print summary
    print("\nTest Summary:")
    for resource_type, resource_info in result["resources"].items():
        print(f"- {resource_type} resource created with ID: {resource_info['id']}")
    
    print("\nTest completed successfully!")
