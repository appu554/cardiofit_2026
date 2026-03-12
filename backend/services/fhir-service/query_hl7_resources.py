import requests
import json
import sys
from tabulate import tabulate

def query_hl7_resources(message_type=None, resource_type=None, base_url="http://localhost:8004", token=None):
    """
    Query resources created from HL7 messages.
    
    Args:
        message_type: HL7 message type (e.g., A01, A02, A03)
        resource_type: Resource type (Patient, Encounter)
        base_url: Base URL of the FHIR service
        token: Authentication token (optional)
    """
    # Set up headers
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    
    # Build the query URL
    url = f"{base_url}/api/hl7/resources"
    params = {}
    
    if message_type:
        params["message_type"] = message_type
    
    if resource_type:
        params["resource_type"] = resource_type
    
    print(f"Querying HL7 resources: {url}")
    if message_type:
        print(f"Message Type: {message_type}")
    if resource_type:
        print(f"Resource Type: {resource_type}")
    
    try:
        # Make the API request
        response = requests.get(url, params=params, headers=headers)
        
        if response.status_code == 200:
            result = response.json()
            
            # Display patients
            if result.get("patients") and len(result["patients"]) > 0:
                patients = result["patients"]
                patient_data = []
                
                for patient in patients:
                    # Extract HL7 message type from tags
                    hl7_type = "Unknown"
                    meta = patient.get("meta", {})
                    tags = meta.get("tag", [])
                    
                    for tag in tags:
                        if tag.get("system") == "http://clinicalsynthesishub.com/hl7/message_type":
                            hl7_type = tag.get("code", "Unknown")
                    
                    # Extract patient name
                    name = "Unknown"
                    if patient.get("name") and len(patient["name"]) > 0:
                        name_obj = patient["name"][0]
                        family = name_obj.get("family", "")
                        given = name_obj.get("given", [])
                        name = f"{family}, {' '.join(given)}"
                    
                    # Extract patient identifier
                    identifier = "Unknown"
                    if patient.get("identifier") and len(patient["identifier"]) > 0:
                        identifier = patient["identifier"][0].get("value", "Unknown")
                    
                    patient_data.append({
                        "ID": patient.get("id", "Unknown"),
                        "Name": name,
                        "Identifier": identifier,
                        "Gender": patient.get("gender", "Unknown"),
                        "HL7 Type": hl7_type
                    })
                
                print("\n=== Patients ===\n")
                print(tabulate(patient_data, headers="keys", tablefmt="grid"))
            
            # Display encounters
            if result.get("encounters") and len(result["encounters"]) > 0:
                encounters = result["encounters"]
                encounter_data = []
                
                for encounter in encounters:
                    # Extract HL7 message type from tags
                    hl7_type = "Unknown"
                    event_meaning = "Unknown"
                    meta = encounter.get("meta", {})
                    tags = meta.get("tag", [])
                    
                    for tag in tags:
                        if tag.get("system") == "http://clinicalsynthesishub.com/hl7/message_type":
                            hl7_type = tag.get("code", "Unknown")
                        if tag.get("system") == "http://clinicalsynthesishub.com/hl7/event_meaning":
                            event_meaning = tag.get("display", "Unknown")
                    
                    # Extract patient reference
                    patient_ref = "Unknown"
                    if encounter.get("subject") and encounter["subject"].get("reference"):
                        patient_ref = encounter["subject"]["reference"]
                    
                    encounter_data.append({
                        "ID": encounter.get("id", "Unknown"),
                        "Status": encounter.get("status", "Unknown"),
                        "Class": encounter.get("class", {}).get("code", "Unknown"),
                        "Patient": patient_ref,
                        "HL7 Type": hl7_type,
                        "Event": event_meaning
                    })
                
                print("\n=== Encounters ===\n")
                print(tabulate(encounter_data, headers="keys", tablefmt="grid"))
            
            print(f"\nTotal resources: {result.get('total_count', 0)}")
            
            if result.get("total_count", 0) == 0:
                print("\nNo HL7 resources found. Try sending some HL7 messages first.")
            
            return result
        else:
            print(f"Error: {response.status_code}")
            print(response.text)
            return None
    
    except Exception as e:
        print(f"Error: {str(e)}")
        return None

if __name__ == "__main__":
    # Parse command line arguments
    message_type = None
    resource_type = None
    token = None
    
    # Simple argument parsing
    for i, arg in enumerate(sys.argv[1:]):
        if arg == "--message-type" and i+1 < len(sys.argv)-1:
            message_type = sys.argv[i+2]
        elif arg == "--resource-type" and i+1 < len(sys.argv)-1:
            resource_type = sys.argv[i+2]
        elif arg == "--token" and i+1 < len(sys.argv)-1:
            token = sys.argv[i+2]
    
    # Query HL7 resources
    query_hl7_resources(message_type, resource_type, token=token)
