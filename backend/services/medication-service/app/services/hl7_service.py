from typing import Dict, List, Optional, Any, Union
import logging
import traceback
from app.utils.hl7_to_fhir import process_hl7_message
from app.services.medication_service import get_medication_service
from app.core.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Singleton instance
_hl7_service_instance = None

def get_hl7_service():
    """Get or create a singleton instance of the HL7 service."""
    global _hl7_service_instance
    if _hl7_service_instance is None:
        _hl7_service_instance = HL7Service()
    return _hl7_service_instance

class HL7Service:
    """Service for processing HL7 messages and converting them to FHIR resources."""
    
    def __init__(self):
        self.medication_service = get_medication_service()
    
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
                
                if resource_type == "Medication":
                    results[resource_type] = await self.medication_service.create_medication(resource, auth_header)
                elif resource_type == "MedicationRequest":
                    results[resource_type] = await self.medication_service.create_medication_request(resource, auth_header)
                elif resource_type == "MedicationAdministration":
                    results[resource_type] = await self.medication_service.create_medication_administration(resource, auth_header)
                elif resource_type == "MedicationStatement":
                    results[resource_type] = await self.medication_service.create_medication_statement(resource, auth_header)
                else:
                    logger.warning(f"Unsupported resource type: {resource_type}")
            
            return {
                "message_type": fhir_resources.get("message_type", "Unknown"),
                "message_control_id": fhir_resources.get("message_control_id", "Unknown"),
                "resources_created": results,
                "status": "success",
                "message": "HL7 message processed successfully"
            }
        except Exception as e:
            logger.error(f"Error processing HL7 message: {str(e)}")
            logger.error(traceback.format_exc())
            return {
                "message_type": "Unknown",
                "message_control_id": "Unknown",
                "resources_created": {},
                "status": "error",
                "message": f"Error processing HL7 message: {str(e)}"
            }
    
    async def process_rde_message(self, message_str: str, auth_header: str) -> Dict[str, Any]:
        """
        Process an RDE (Pharmacy/Treatment Encoded Order) message.
        
        Args:
            message_str: The raw HL7 message string
            auth_header: The authorization header for API calls
            
        Returns:
            Dictionary containing the results of processing the message
        """
        try:
            logger.info("Processing RDE message")
            return await self.process_message(message_str, auth_header)
        except Exception as e:
            logger.error(f"Error processing RDE message: {str(e)}")
            logger.error(traceback.format_exc())
            return {
                "message_type": "RDE",
                "message_control_id": "Unknown",
                "resources_created": {},
                "status": "error",
                "message": f"Error processing RDE message: {str(e)}"
            }
    
    async def process_ras_message(self, message_str: str, auth_header: str) -> Dict[str, Any]:
        """
        Process an RAS (Pharmacy/Treatment Administration) message.
        
        Args:
            message_str: The raw HL7 message string
            auth_header: The authorization header for API calls
            
        Returns:
            Dictionary containing the results of processing the message
        """
        try:
            logger.info("Processing RAS message")
            return await self.process_message(message_str, auth_header)
        except Exception as e:
            logger.error(f"Error processing RAS message: {str(e)}")
            logger.error(traceback.format_exc())
            return {
                "message_type": "RAS",
                "message_control_id": "Unknown",
                "resources_created": {},
                "status": "error",
                "message": f"Error processing RAS message: {str(e)}"
            }
