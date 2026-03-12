import requests
import json
import sys
from tabulate import tabulate

def view_hl7_resources(base_url="http://localhost:8004", token=None):
    """
    View resources in the system that were created from HL7 messages.
    
    Args:
        base_url: Base URL of the FHIR service
        token: Authentication token (optional)
    """
    # Set up headers
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    
    # Search for patients with HL7 tag
    search_url = f"{base_url}/api/fhir/Patient"
    
    try:
        # First, get all patients
        response = requests.get(search_url, headers=headers)
        
        if response.status_code == 200:
            patients = response.json()
            
            # Filter patients with HL7 tags
            hl7_patients = []
            
            for patient in patients:
                meta = patient.get("meta", {})
                tags = meta.get("tag", [])
                
                is_hl7 = False
                hl7_type = ""
                
                for tag in tags:
                    if tag.get("system") == "http://clinicalsynthesishub.com/source" and tag.get("code") == "hl7v2":
                        is_hl7 = True
                    
                    if tag.get("system") == "http://clinicalsynthesishub.com/hl7/message_type":
                        hl7_type = tag.get("code", "")
                
                if is_hl7:
                    hl7_patients.append({
                        "id": patient.get("id", ""),
                        "name": ", ".join([name.get("family", "") + ", " + " ".join(name.get("given", [])) for name in patient.get("name", [])]),
                        "identifier": ", ".join([identifier.get("value", "") for identifier in patient.get("identifier", [])]),
                        "hl7_type": hl7_type
                    })
            
            if hl7_patients:
                print("\n=== Patients Created from HL7 Messages ===\n")
                print(tabulate(hl7_patients, headers="keys", tablefmt="grid"))
                
                # For each patient, get associated encounters
                for patient in hl7_patients:
                    patient_id = patient["id"]
                    encounters_url = f"{base_url}/api/fhir/Encounter?subject=Patient/{patient_id}"
                    
                    encounters_response = requests.get(encounters_url, headers=headers)
                    
                    if encounters_response.status_code == 200:
                        encounters = encounters_response.json()
                        
                        # Filter encounters with HL7 tags
                        hl7_encounters = []
                        
                        for encounter in encounters:
                            meta = encounter.get("meta", {})
                            tags = meta.get("tag", [])
                            
                            is_hl7 = False
                            hl7_type = ""
                            event_meaning = ""
                            
                            for tag in tags:
                                if tag.get("system") == "http://clinicalsynthesishub.com/source" and tag.get("code") == "hl7v2":
                                    is_hl7 = True
                                
                                if tag.get("system") == "http://clinicalsynthesishub.com/hl7/message_type":
                                    hl7_type = tag.get("code", "")
                                
                                if tag.get("system") == "http://clinicalsynthesishub.com/hl7/event_meaning":
                                    event_meaning = tag.get("display", "")
                            
                            if is_hl7:
                                hl7_encounters.append({
                                    "id": encounter.get("id", ""),
                                    "status": encounter.get("status", ""),
                                    "class": encounter.get("class", {}).get("code", ""),
                                    "hl7_type": hl7_type,
                                    "event": event_meaning
                                })
                        
                        if hl7_encounters:
                            print(f"\n=== Encounters for Patient {patient['name']} ===\n")
                            print(tabulate(hl7_encounters, headers="keys", tablefmt="grid"))
                
                print("\nHL7 integration is working correctly!")
                return True
            else:
                print("\nNo patients created from HL7 messages were found.")
                print("This could mean either:")
                print("1. The HL7 integration is not working")
                print("2. No HL7 messages have been processed yet")
                print("3. The meta tags are not being added correctly")
                
                print("\nTry sending a test HL7 message using the verify_hl7_integration.py script.")
                return False
        else:
            print(f"Error retrieving patients: {response.status_code}")
            print(response.text)
            return False
    
    except Exception as e:
        print(f"Error: {str(e)}")
        return False

if __name__ == "__main__":
    # Get token from command line if provided
    token = None
    if len(sys.argv) > 1:
        token = sys.argv[1]
    
    # Run the viewer
    view_hl7_resources(token=token)
