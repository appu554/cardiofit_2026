import hl7
from typing import Dict, List, Any, Optional, Tuple, Union
from datetime import datetime
import uuid
import logging
from app.models.hl7 import ORUMessage
from app.models.lab import LabTest, LabPanel

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

def parse_hl7_message(message_str: str) -> Tuple[Any, str]:
    """
    Parse an HL7 message string into a parsed message object.

    Args:
        message_str: The raw HL7 message string

    Returns:
        Tuple containing the parsed message and the message type
    """
    try:
        # Replace any Windows line endings with proper HL7 segment separators
        message_str = message_str.replace('\r\n', '\r').replace('\n', '\r')
        if not message_str.endswith('\r'):
            message_str += '\r'

        message = hl7.parse(message_str)

        # Extract message type from MSH segment
        msh = message.segment('MSH')
        if len(msh) > 9 and len(msh[9]) > 1:
            message_type = str(msh[9][0])
            event_type = str(msh[9][1])
            logger.info(f"Parsed HL7 message of type {message_type}^{event_type}")
            return message, f"{message_type}^{event_type}"
        else:
            raise ValueError("MSH segment does not contain message type information")
    except Exception as e:
        logger.error(f"Error parsing HL7 message: {str(e)}")
        raise ValueError(f"Could not parse HL7 message: {str(e)}")

def extract_oru_data(message: Any) -> ORUMessage:
    """
    Extract data from an ORU message into our ORUMessage model.

    Args:
        message: The parsed HL7 message

    Returns:
        ORUMessage object with extracted data
    """
    try:
        # Initialize variables with default values
        message_control_id = "UNKNOWN"
        message_datetime = datetime.now()
        event_type = "UNKNOWN"
        patient_id = "UNKNOWN"
        patient_id_type = None
        patient_id_authority = None
        observation_datetime = datetime.now()
        observation_value = None
        observation_type = "UNKNOWN"
        observation_unit = None
        observation_range = None
        observation_status = "final"
        observation_method = None
        order_number = None
        ordering_provider = None
        raw_message = str(message)
        observations = []

        # Extract message metadata
        msh = message.segment('MSH')
        if len(msh) > 10:
            message_control_id = str(msh[10][0])

        if len(msh) > 7 and len(msh[7]) > 0:
            message_datetime_str = str(msh[7][0])
            try:
                message_datetime = datetime.strptime(message_datetime_str, '%Y%m%d%H%M%S')
            except ValueError:
                # Try alternative formats
                try:
                    message_datetime = datetime.strptime(message_datetime_str, '%Y%m%d')
                except ValueError:
                    logger.warning(f"Could not parse message datetime: {message_datetime_str}")

        if len(msh) > 9 and len(msh[9]) > 1:
            event_type = str(msh[9][1])

        # Extract patient information
        try:
            pid = message.segment('PID')

            if len(pid) > 3 and len(pid[3]) > 0:
                patient_id = str(pid[3][0])
                patient_id_type = str(pid[3][4]) if len(pid[3]) > 4 else None
                patient_id_authority = str(pid[3][3]) if len(pid[3]) > 3 else None
        except Exception as pid_error:
            logger.warning(f"Error extracting PID data: {str(pid_error)}")

        # Extract OBR (Observation Request) segments
        try:
            obr_segments = message.segments('OBR')
            for obr in obr_segments:
                if len(obr) > 3 and len(obr[3]) > 0:
                    order_number = str(obr[3][0])

                # Extract ordering provider
                if len(obr) > 16 and len(obr[16]) > 0:
                    provider_parts = str(obr[16][0]).split('^')
                    ordering_provider = {
                        "id": provider_parts[0] if len(provider_parts) > 0 else "",
                        "family": provider_parts[1] if len(provider_parts) > 1 else "",
                        "given": provider_parts[2] if len(provider_parts) > 2 else ""
                    }

                # Extract observation datetime
                if len(obr) > 7 and len(obr[7]) > 0:
                    obs_datetime_str = str(obr[7][0])
                    try:
                        observation_datetime = datetime.strptime(obs_datetime_str, '%Y%m%d%H%M%S')
                    except ValueError:
                        try:
                            observation_datetime = datetime.strptime(obs_datetime_str, '%Y%m%d')
                        except ValueError:
                            logger.warning(f"Could not parse observation datetime: {obs_datetime_str}")

                # Find related OBX segments
                obx_segments = message.segments('OBX')
                for obx in obx_segments:
                    # Extract observation data
                    if len(obx) < 5:
                        continue

                    # Get observation type
                    obs_type = ""
                    obs_type_display = ""
                    if len(obx) > 3 and len(obx[3]) > 0:
                        obs_type_parts = str(obx[3][0]).split('^')
                        obs_type = obs_type_parts[0] if len(obs_type_parts) > 0 else ""
                        obs_type_display = obs_type_parts[1] if len(obs_type_parts) > 1 else ""

                    # Get observation value
                    obs_value = str(obx[5][0]) if len(obx) > 5 and len(obx[5]) > 0 else ""

                    # Get observation unit
                    obs_unit = str(obx[6][0]) if len(obx) > 6 and len(obx[6]) > 0 else ""

                    # Get reference range
                    obs_range = str(obx[7][0]) if len(obx) > 7 and len(obx[7]) > 0 else ""

                    # Get observation status
                    obs_status = str(obx[11][0]) if len(obx) > 11 and len(obx[11]) > 0 else "final"

                    # Get value type
                    value_type = str(obx[2][0]) if len(obx) > 2 and len(obx[2]) > 0 else "ST"

                    # Convert value based on type
                    converted_value = obs_value
                    if value_type == "NM" and obs_value:
                        try:
                            converted_value = float(obs_value)
                        except ValueError:
                            logger.warning(f"Could not convert numeric value: {obs_value}")
                    elif value_type == "DT" and obs_value:
                        try:
                            converted_value = datetime.strptime(obs_value, '%Y%m%d').strftime('%Y-%m-%d')
                        except ValueError:
                            logger.warning(f"Could not convert date value: {obs_value}")

                    # Determine interpretation
                    interpretation = None
                    if "N" in obs_status:
                        interpretation = "normal"
                    elif "H" in obs_status:
                        interpretation = "high"
                    elif "L" in obs_status:
                        interpretation = "low"
                    elif "A" in obs_status:
                        interpretation = "abnormal"
                    elif "C" in obs_status:
                        interpretation = "critical"

                    # Add to observations list
                    observations.append({
                        "test_code": obs_type,
                        "test_name": obs_type_display,
                        "value": converted_value,
                        "unit": obs_unit,
                        "reference_range": obs_range,
                        "status": "final",
                        "interpretation": interpretation,
                        "value_type": value_type
                    })

        except Exception as obr_error:
            logger.warning(f"Error extracting OBR/OBX data: {str(obr_error)}")

        # Create and return ORUMessage object
        return ORUMessage(
            message_type="ORU",
            event_type=event_type,
            message_control_id=message_control_id,
            message_datetime=message_datetime,
            raw_message=raw_message,
            patient_id=patient_id,
            patient_id_type=patient_id_type,
            patient_id_authority=patient_id_authority,
            observation_datetime=observation_datetime,
            observation_value=observations,  # Use the list of observations
            observation_type=observation_type,
            observation_unit=observation_unit,
            observation_range=observation_range,
            observation_status=observation_status,
            observation_method=observation_method,
            order_number=order_number,
            ordering_provider=ordering_provider
        )
    except Exception as e:
        logger.error(f"Error extracting ORU data: {str(e)}")
        logger.error(f"Message: {message}")
        raise ValueError(f"Could not extract ORU data: {str(e)}")

def oru_to_lab_data(oru_message: ORUMessage) -> Dict[str, Any]:
    """
    Convert an ORU message to lab data.

    Args:
        oru_message: The ORU message to convert

    Returns:
        Dictionary containing lab tests and panels created from the ORU message
    """
    resources = {
        "lab_tests": [],
        "lab_panel": None
    }

    # Extract panel information from OBR segment
    panel_code = ""
    panel_name = ""
    if hasattr(oru_message, "order_number") and oru_message.order_number:
        # Try to extract panel information from the order number
        panel_code = oru_message.order_number

    # Create lab tests for each observation
    lab_tests = []
    if isinstance(oru_message.observation_value, list):
        for obs in oru_message.observation_value:
            # Create a lab test
            lab_test = {
                "id": str(uuid.uuid4()),
                "test_code": obs.get("test_code", ""),
                "test_name": obs.get("test_name", ""),
                "value": obs.get("value", ""),
                "unit": obs.get("unit", ""),
                "reference_range": obs.get("reference_range", ""),
                "interpretation": obs.get("interpretation", ""),
                "status": obs.get("status", "final"),
                "category": "laboratory",
                "effective_date_time": oru_message.observation_datetime,
                "issued": oru_message.message_datetime,
                "patient_id": oru_message.patient_id,
                "order_number": oru_message.order_number,
                "performer": oru_message.ordering_provider["family"] if oru_message.ordering_provider and "family" in oru_message.ordering_provider else None
            }
            
            lab_tests.append(lab_test)
            resources["lab_tests"].append(lab_test)
            
            # Try to determine panel information from the first test
            if not panel_code and not panel_name and obs.get("test_name", ""):
                # Check if this is part of a panel
                if "CBC" in obs.get("test_name", ""):
                    panel_code = "CBC"
                    panel_name = "COMPLETE BLOOD COUNT"
                elif "CHEM" in obs.get("test_name", ""):
                    panel_code = "CHEM"
                    panel_name = "CHEMISTRY PANEL"
                elif "LFT" in obs.get("test_name", ""):
                    panel_code = "LFT"
                    panel_name = "LIVER FUNCTION TESTS"
                elif "LYTES" in obs.get("test_name", ""):
                    panel_code = "LYTES"
                    panel_name = "ELECTROLYTES"

    # Create a lab panel if we have multiple tests
    if len(lab_tests) > 1 and (panel_code or panel_name):
        lab_panel = {
            "id": str(uuid.uuid4()),
            "panel_code": panel_code or "PANEL",
            "panel_name": panel_name or "LAB PANEL",
            "tests": lab_tests,
            "effective_date_time": oru_message.observation_datetime,
            "issued": oru_message.message_datetime,
            "patient_id": oru_message.patient_id,
            "order_number": oru_message.order_number,
            "performer": oru_message.ordering_provider["family"] if oru_message.ordering_provider and "family" in oru_message.ordering_provider else None
        }
        
        resources["lab_panel"] = lab_panel

    return resources

def process_hl7_message(message_str: str) -> Dict[str, Any]:
    """
    Process an HL7 message and convert it to lab data.

    Args:
        message_str: The raw HL7 message string

    Returns:
        Dictionary containing lab data created from the HL7 message
    """
    # Parse the message
    parsed_message, message_type = parse_hl7_message(message_str)

    # Process based on message type
    if message_type.startswith('ORU'):
        # Extract ORU data
        oru_message = extract_oru_data(parsed_message)

        # Convert to lab data
        return oru_to_lab_data(oru_message)
    else:
        raise ValueError(f"Unsupported message type for lab service: {message_type}")
