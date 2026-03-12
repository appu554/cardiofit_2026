import requests
import json
import time
import sys
from pathlib import Path

def verify_hl7_integration(base_url="http://localhost:8004", token=None):
    """
    Verify that the HL7 integration is working correctly by:
    1. Sending an HL7 message
    2. Verifying the response
    3. Checking that the FHIR resources were created
    
    Args:
        base_url: Base URL of the FHIR service
        token: Authentication token (optional)
    """
    # Set up headers
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    
    # Step 1: Load and send an HL7 message
    current_dir = Path(__file__).parent
    sample_file = current_dir / "app" / "tests" / "sample_adt_message.hl7"
    
    with open(sample_file, "r") as f:
        message_str = f.read()
    
    # Create a unique patient identifier for this test
    test_id = int(time.time())
    message_str = message_str.replace("MRN12345", f"MRN{test_id}")
    
    print(f"Sending HL7 message with test ID: MRN{test_id}")
    
    # Send the HL7 message
    hl7_url = f"{base_url}/api/hl7/process"
    hl7_payload = {"message": message_str}
    
    try:
        hl7_response = requests.post(hl7_url, json=hl7_payload, headers=headers)
        
        # Step 2: Verify the HL7 response
        if hl7_response.status_code == 200:
            hl7_result = hl7_response.json()
            print("HL7 API Response:")
            print(json.dumps(hl7_result, indent=2))
            
            # Check if the response contains the expected fields
            if "message" in hl7_result and "resources" in hl7_result:
                print("✅ HL7 API response contains expected fields")
                
                # Check if Patient resource was created
                if "Patient" in hl7_result["resources"]:
                    patient_id = hl7_result["resources"]["Patient"]["id"]
                    print(f"✅ Patient resource created with ID: {patient_id}")
                    
                    # Step 3: Verify the Patient resource was created in FHIR
                    time.sleep(1)  # Give the system time to process
                    
                    fhir_url = f"{base_url}/api/fhir/Patient/{patient_id}"
                    fhir_response = requests.get(fhir_url, headers=headers)
                    
                    if fhir_response.status_code == 200:
                        patient = fhir_response.json()
                        print("FHIR Patient Resource:")
                        print(json.dumps(patient, indent=2))
                        
                        # Verify the patient has the correct identifier
                        identifiers = patient.get("identifier", [])
                        found_test_id = False
                        
                        for identifier in identifiers:
                            if identifier.get("value") == f"MRN{test_id}":
                                found_test_id = True
                                break
                        
                        if found_test_id:
                            print(f"✅ Patient resource contains the correct identifier: MRN{test_id}")
                            print("\n✅ HL7 INTEGRATION IS WORKING CORRECTLY!")
                            return True
                        else:
                            print(f"❌ Patient resource does not contain the expected identifier: MRN{test_id}")
                    else:
                        print(f"❌ Failed to retrieve Patient resource: {fhir_response.status_code}")
                        print(fhir_response.text)
                else:
                    print("❌ No Patient resource was created")
            else:
                print("❌ HL7 API response does not contain expected fields")
        else:
            print(f"❌ HL7 API request failed: {hl7_response.status_code}")
            print(hl7_response.text)
    
    except Exception as e:
        print(f"❌ Error during verification: {str(e)}")
        return False
    
    print("\n❌ HL7 INTEGRATION VERIFICATION FAILED")
    return False

if __name__ == "__main__":
    # Get token from command line if provided
    token = None
    if len(sys.argv) > 1:
        token = sys.argv[1]
    
    # Run the verification
    verify_hl7_integration(token=token)
