from typing import Dict, List, Any, Optional
import logging
from app.utils.hl7_to_lab import process_hl7_message
from app.services.lab_service import get_lab_service

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
    """Service for processing HL7 messages."""
    
    def __init__(self):
        self.lab_service = get_lab_service()
    
    async def process_message(self, message_str: str) -> Dict[str, Any]:
        """
        Process an HL7 message, convert it to lab data, and store it.
        
        Args:
            message_str: The raw HL7 message string
            
        Returns:
            Dictionary containing the results of processing the message
        """
        try:
            logger.info("Processing HL7 message")
            
            # Process the message and convert to lab data
            lab_data = process_hl7_message(message_str)
            
            # Store lab tests
            results = {
                "lab_tests": [],
                "lab_panel": None
            }
            
            # Store lab tests
            for lab_test in lab_data.get("lab_tests", []):
                created_test = await self.lab_service.create_lab_test(lab_test)
                results["lab_tests"].append({
                    "id": created_test.get("id") or created_test.get("_id"),
                    "test_code": created_test.get("test_code"),
                    "test_name": created_test.get("test_name"),
                    "status": "created"
                })
            
            # Store lab panel if available
            if lab_data.get("lab_panel"):
                created_panel = await self.lab_service.create_lab_panel(lab_data["lab_panel"])
                results["lab_panel"] = {
                    "id": created_panel.get("id") or created_panel.get("_id"),
                    "panel_code": created_panel.get("panel_code"),
                    "panel_name": created_panel.get("panel_name"),
                    "status": "created"
                }
            
            return {
                "message": "HL7 message processed successfully",
                "resources": results
            }
        except Exception as e:
            logger.error(f"Error processing HL7 message: {str(e)}")
            raise
