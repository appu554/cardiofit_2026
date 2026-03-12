from typing import Dict, List, Any, Optional
import logging
import traceback
from app.utils.hl7_to_fhir import process_hl7_message
from app.core.integration import FHIRIntegrationLayer
from app.services.fhir_service import get_fhir_service

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

class HL7Service:
    """Service for processing HL7 messages and converting them to FHIR resources."""
    
    def __init__(self):
        self.fhir_integration = FHIRIntegrationLayer()
        self.fhir_service = get_fhir_service()
    
    async def process_message(self, message_str: str, auth_header: str) -> Dict[str, Any]:
        """
        Process an HL7 message, convert it to FHIR resources, and store them.
        
        Args:
            message_str: The raw HL7 message string
            auth_header: The authorization header for API calls
            
        Returns:
            Dictionary containing the results of processing the message
        """
        try:
            logger.info("Processing HL7 message")
            
            # Process the message and convert to FHIR resources
            fhir_resources = process_hl7_message(message_str)
            
            # Store each resource using the appropriate service
            results = {}
            for resource_type, resource in fhir_resources.items():
                logger.info(f"Storing {resource_type} resource")
                
                # Check if the resource already exists by identifier
                existing_resource = None
                if resource_type == "Patient" and "identifier" in resource:
                    # Search for existing patient with the same identifier
                    identifier_value = resource["identifier"][0]["value"]
                    identifier_system = resource["identifier"][0]["system"]
                    search_params = {
                        "identifier": f"{identifier_system}|{identifier_value}"
                    }
                    search_results = await self.fhir_integration.search_resources(resource_type, search_params, auth_header)
                    if search_results and len(search_results) > 0:
                        existing_resource = search_results[0]
                        logger.info(f"Found existing {resource_type} with ID {existing_resource['id']}")
                
                if existing_resource:
                    # Update the existing resource
                    resource["id"] = existing_resource["id"]
                    updated_resource = await self.fhir_integration.update_resource(
                        resource_type, 
                        existing_resource["id"], 
                        resource, 
                        auth_header
                    )
                    results[resource_type] = {
                        "id": updated_resource["id"],
                        "status": "updated"
                    }
                else:
                    # Create a new resource
                    created_resource = await self.fhir_integration.create_resource(
                        resource_type,
                        resource,
                        auth_header
                    )
                    results[resource_type] = {
                        "id": created_resource["id"],
                        "status": "created"
                    }
            
            # For Encounter resources, ensure the subject reference is updated
            if "Encounter" in fhir_resources and "Patient" in results:
                encounter = fhir_resources["Encounter"]
                patient_id = results["Patient"]["id"]
                
                # Update the subject reference to use the actual patient ID
                encounter["subject"]["reference"] = f"Patient/{patient_id}"
                
                # Store the updated encounter
                if "Encounter" in results and results["Encounter"]["status"] == "created":
                    # Update the encounter we just created
                    encounter_id = results["Encounter"]["id"]
                    updated_encounter = await self.fhir_integration.update_resource(
                        "Encounter",
                        encounter_id,
                        encounter,
                        auth_header
                    )
                    results["Encounter"]["status"] = "updated"
            
            return {
                "message": "HL7 message processed successfully",
                "resources": results
            }
        
        except Exception as e:
            logger.error(f"Error processing HL7 message: {str(e)}")
            logger.error(traceback.format_exc())
            raise

# Singleton instance
_hl7_service_instance = None

def get_hl7_service():
    """Get or create a singleton instance of the HL7 service."""
    global _hl7_service_instance
    if _hl7_service_instance is None:
        _hl7_service_instance = HL7Service()
    return _hl7_service_instance
