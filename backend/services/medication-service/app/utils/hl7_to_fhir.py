from typing import Dict, List, Any, Optional
import logging
import hl7
from datetime import datetime
from app.models.medication import (
    MedicationCreate, MedicationRequestCreate, MedicationAdministrationCreate, MedicationStatementCreate,
    MedicationRequestStatus, MedicationRequestIntent, MedicationAdministrationStatus, MedicationStatementStatus,
    CodeableConcept, Coding, Reference, Quantity, DosageInstruction
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def process_hl7_message(message_str: str) -> Dict[str, Any]:
    """
    Process an HL7 message and convert it to FHIR resources.
    
    Args:
        message_str: The raw HL7 message string
        
    Returns:
        Dictionary containing the FHIR resources
    """
    try:
        # Parse the HL7 message
        message = hl7.parse(message_str)
        
        # Get message type
        message_type = message.segment('MSH')[9][0]
        message_event = message.segment('MSH')[9][1]
        message_control_id = message.segment('MSH')[10][0]
        
        # Process based on message type
        if message_type == 'RDE':
            return process_rde_message(message, message_str, message_control_id)
        elif message_type == 'RAS':
            return process_ras_message(message, message_str, message_control_id)
        else:
            logger.warning(f"Unsupported message type: {message_type}")
            return {
                "message_type": message_type,
                "message_control_id": message_control_id,
                "error": f"Unsupported message type: {message_type}"
            }
    except Exception as e:
        logger.error(f"Error processing HL7 message: {str(e)}")
        return {
            "message_type": "Unknown",
            "message_control_id": "Unknown",
            "error": f"Error processing HL7 message: {str(e)}"
        }

def process_rde_message(message, message_str: str, message_control_id: str) -> Dict[str, Any]:
    """
    Process an RDE (Pharmacy/Treatment Encoded Order) message.
    
    Args:
        message: The parsed HL7 message
        message_str: The raw HL7 message string
        message_control_id: The message control ID
        
    Returns:
        Dictionary containing the FHIR resources
    """
    try:
        # Extract patient information
        patient_id = message.segment('PID')[3][0]
        
        # Extract order information
        order_number = message.segment('ORC')[2][0]
        order_status = message.segment('ORC')[5][0]
        order_datetime_str = message.segment('ORC')[9][0]
        order_datetime = datetime.strptime(order_datetime_str, '%Y%m%d%H%M%S')
        
        # Extract medication information
        medication_code = message.segment('RXE')[2][0]
        medication_name = message.segment('RXE')[2][1]
        dosage = message.segment('RXE')[3][0]
        frequency = message.segment('RXE')[5][0]
        duration = message.segment('RXE')[6][0]
        quantity = message.segment('RXE')[10][0]
        
        # Extract provider information
        provider_id = message.segment('ORC')[12][0]
        provider_last_name = message.segment('ORC')[12][1]
        provider_first_name = message.segment('ORC')[12][2]
        
        # Create FHIR resources
        
        # Medication
        medication = MedicationCreate(
            code=CodeableConcept(
                coding=[
                    Coding(
                        system="http://www.nlm.nih.gov/research/umls/rxnorm",
                        code=medication_code,
                        display=medication_name
                    )
                ],
                text=medication_name
            )
        )
        
        # MedicationRequest
        medication_request = MedicationRequestCreate(
            status=MedicationRequestStatus.ACTIVE,
            intent=MedicationRequestIntent.ORDER,
            medicationCodeableConcept=CodeableConcept(
                coding=[
                    Coding(
                        system="http://www.nlm.nih.gov/research/umls/rxnorm",
                        code=medication_code,
                        display=medication_name
                    )
                ],
                text=medication_name
            ),
            subject=Reference(
                reference=f"Patient/{patient_id}"
            ),
            authoredOn=order_datetime.isoformat(),
            requester=Reference(
                reference=f"Practitioner/{provider_id}",
                display=f"{provider_first_name} {provider_last_name}"
            ),
            dosageInstruction=[
                DosageInstruction(
                    text=f"{dosage} {frequency} for {duration}",
                    timing={"code": {"text": frequency}}
                )
            ]
        )
        
        return {
            "message_type": "RDE",
            "message_control_id": message_control_id,
            "Medication": medication,
            "MedicationRequest": medication_request
        }
    except Exception as e:
        logger.error(f"Error processing RDE message: {str(e)}")
        return {
            "message_type": "RDE",
            "message_control_id": message_control_id,
            "error": f"Error processing RDE message: {str(e)}"
        }

def process_ras_message(message, message_str: str, message_control_id: str) -> Dict[str, Any]:
    """
    Process an RAS (Pharmacy/Treatment Administration) message.
    
    Args:
        message: The parsed HL7 message
        message_str: The raw HL7 message string
        message_control_id: The message control ID
        
    Returns:
        Dictionary containing the FHIR resources
    """
    try:
        # Extract patient information
        patient_id = message.segment('PID')[3][0]
        
        # Extract order information
        order_number = message.segment('ORC')[2][0]
        
        # Extract administration information
        admin_datetime_str = message.segment('RXA')[3][0]
        admin_datetime = datetime.strptime(admin_datetime_str, '%Y%m%d%H%M%S')
        medication_code = message.segment('RXA')[5][0]
        medication_name = message.segment('RXA')[5][1]
        dosage = message.segment('RXA')[6][0]
        
        # Extract provider information
        provider_id = message.segment('RXA')[10][0]
        provider_last_name = message.segment('RXA')[10][1]
        provider_first_name = message.segment('RXA')[10][2]
        
        # Create FHIR resources
        
        # Medication
        medication = MedicationCreate(
            code=CodeableConcept(
                coding=[
                    Coding(
                        system="http://www.nlm.nih.gov/research/umls/rxnorm",
                        code=medication_code,
                        display=medication_name
                    )
                ],
                text=medication_name
            )
        )
        
        # MedicationAdministration
        medication_administration = MedicationAdministrationCreate(
            status=MedicationAdministrationStatus.COMPLETED,
            medicationCodeableConcept=CodeableConcept(
                coding=[
                    Coding(
                        system="http://www.nlm.nih.gov/research/umls/rxnorm",
                        code=medication_code,
                        display=medication_name
                    )
                ],
                text=medication_name
            ),
            subject=Reference(
                reference=f"Patient/{patient_id}"
            ),
            effectiveDateTime=admin_datetime.isoformat(),
            performer=[
                {
                    "actor": {
                        "reference": f"Practitioner/{provider_id}",
                        "display": f"{provider_first_name} {provider_last_name}"
                    }
                }
            ],
            request=Reference(
                reference=f"MedicationRequest/{order_number}"
            ),
            dosage={
                "text": dosage,
                "dose": {
                    "value": float(dosage.split()[0]),
                    "unit": dosage.split()[1]
                }
            }
        )
        
        return {
            "message_type": "RAS",
            "message_control_id": message_control_id,
            "Medication": medication,
            "MedicationAdministration": medication_administration
        }
    except Exception as e:
        logger.error(f"Error processing RAS message: {str(e)}")
        return {
            "message_type": "RAS",
            "message_control_id": message_control_id,
            "error": f"Error processing RAS message: {str(e)}"
        }
