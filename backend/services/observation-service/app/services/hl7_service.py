from typing import Dict, List, Any, Optional
import hl7
from datetime import datetime
import uuid
from app.models.hl7 import ORUMessage, HL7Message
from app.services.observation_service import get_observation_service

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
        self.observation_service = get_observation_service()
    
    async def process_message(self, message_str: str) -> Dict[str, Any]:
        """Process an HL7 message."""
        # Parse the message
        try:
            parsed_message = hl7.parse(message_str)
        except Exception as e:
            raise ValueError(f"Error parsing HL7 message: {str(e)}")
        
        # Get the message type
        message_type = str(parsed_message.segment('MSH')[9][0])
        
        # Process based on message type
        if message_type == 'ORU':
            return await self.process_oru_message(parsed_message, message_str)
        else:
            raise ValueError(f"Unsupported message type: {message_type}")
    
    async def process_oru_message(self, parsed_message, raw_message: str) -> Dict[str, Any]:
        """Process an ORU message."""
        # Extract message details
        message_control_id = str(parsed_message.segment('MSH')[10])
        message_datetime_str = str(parsed_message.segment('MSH')[7])
        message_datetime = self._parse_hl7_datetime(message_datetime_str)
        event_type = str(parsed_message.segment('MSH')[9][1])
        
        # Extract patient information
        patient_id = str(parsed_message.segment('PID')[3][0])
        patient_id_type = str(parsed_message.segment('PID')[3][4]) if len(parsed_message.segment('PID')[3]) > 4 else None
        patient_id_authority = str(parsed_message.segment('PID')[3][4]) if len(parsed_message.segment('PID')[3]) > 4 else None
        
        # Extract observation information from OBX segments
        observations = []
        for segment in parsed_message:
            if segment[0][0] == 'OBX':
                observation = self._extract_observation_from_obx(segment, patient_id)
                observations.append(observation)
        
        # Create ORU message model
        oru_message = {
            "message_type": "ORU",
            "message_control_id": message_control_id,
            "message_datetime": message_datetime,
            "event_type": event_type,
            "patient_id": patient_id,
            "patient_id_type": patient_id_type,
            "patient_id_authority": patient_id_authority,
            "raw_message": raw_message,
            "observations": observations
        }
        
        # Create observations in the database
        created_observations = []
        for observation in observations:
            created_observation = await self.observation_service.create_observation(observation)
            created_observations.append(created_observation)
        
        # Return the processed message and created resources
        return {
            "message": oru_message,
            "resources": {
                "observations": created_observations
            }
        }
    
    def _extract_observation_from_obx(self, obx_segment, patient_id: str) -> Dict[str, Any]:
        """Extract observation information from an OBX segment."""
        # Get observation details
        observation_id = str(obx_segment[3][0]) if len(obx_segment[3]) > 0 else str(uuid.uuid4())
        observation_name = str(obx_segment[3][1]) if len(obx_segment[3]) > 1 else ""
        observation_type = str(obx_segment[2])
        observation_value = self._get_observation_value(obx_segment[5], observation_type)
        observation_unit = str(obx_segment[6]) if len(obx_segment) > 6 else None
        observation_range = str(obx_segment[7]) if len(obx_segment) > 7 else None
        observation_status = str(obx_segment[11]) if len(obx_segment) > 11 else "F"  # Default to Final
        observation_datetime_str = str(obx_segment[14]) if len(obx_segment) > 14 else None
        
        # Parse observation datetime
        if observation_datetime_str:
            observation_datetime = self._parse_hl7_datetime(observation_datetime_str)
        else:
            observation_datetime = datetime.now()
        
        # Determine observation category based on observation name or ID
        # This is a simplified approach - in a real system, you would use a more sophisticated mapping
        category = self._determine_observation_category(observation_id, observation_name)
        
        # Create CodeableConcept for the observation code
        code = {
            "coding": [
                {
                    "system": "http://loinc.org",  # Assuming LOINC coding system
                    "code": observation_id,
                    "display": observation_name
                }
            ],
            "text": observation_name
        }
        
        # Create Reference for the patient
        subject = {
            "reference": f"Patient/{patient_id}"
        }
        
        # Create the observation
        observation = {
            "id": str(uuid.uuid4()),
            "status": self._map_hl7_status_to_fhir(observation_status),
            "category": category,
            "code": code,
            "subject": subject,
            "effective_datetime": observation_datetime
        }
        
        # Add value based on type
        if observation_type in ["NM", "SN"]:  # Numeric
            observation["value_quantity"] = {
                "value": float(observation_value) if observation_value is not None else None,
                "unit": observation_unit,
                "system": "http://unitsofmeasure.org",
                "code": observation_unit
            }
        elif observation_type == "ST":  # String
            observation["value_string"] = observation_value
        elif observation_type == "TX":  # Text
            observation["value_string"] = observation_value
        elif observation_type == "CE":  # Coded Entry
            observation["value_string"] = observation_value
        elif observation_type == "DT":  # Date
            observation["value_datetime"] = observation_value
        elif observation_type == "TM":  # Time
            observation["value_time"] = observation_value
        elif observation_type == "TS":  # Timestamp
            observation["value_datetime"] = observation_value
        else:
            observation["value_string"] = str(observation_value) if observation_value is not None else None
        
        # Add reference range if available
        if observation_range:
            # Parse reference range (e.g., "1-10" or ">5")
            reference_range = self._parse_reference_range(observation_range, observation_unit)
            if reference_range:
                observation["reference_range"] = [reference_range]
        
        return observation
    
    def _get_observation_value(self, value_field, value_type: str) -> Any:
        """Get the observation value based on the value type."""
        value_str = str(value_field)
        
        if value_type == "NM":  # Numeric
            try:
                return float(value_str)
            except ValueError:
                return None
        elif value_type == "SN":  # Structured Numeric
            # Handle structured numeric (e.g., ">10", "<5", "1-10")
            return value_str
        elif value_type == "ST":  # String
            return value_str
        elif value_type == "TX":  # Text
            return value_str
        elif value_type == "CE":  # Coded Entry
            return value_str
        elif value_type == "DT":  # Date
            return value_str
        elif value_type == "TM":  # Time
            return value_str
        elif value_type == "TS":  # Timestamp
            return value_str
        else:
            return value_str
    
    def _parse_hl7_datetime(self, datetime_str: str) -> datetime:
        """Parse an HL7 datetime string."""
        # Handle different HL7 datetime formats
        if len(datetime_str) >= 14:
            # Format: YYYYMMDDHHMMSS
            return datetime(
                year=int(datetime_str[0:4]),
                month=int(datetime_str[4:6]),
                day=int(datetime_str[6:8]),
                hour=int(datetime_str[8:10]),
                minute=int(datetime_str[10:12]),
                second=int(datetime_str[12:14])
            )
        elif len(datetime_str) >= 12:
            # Format: YYYYMMDDHHMM
            return datetime(
                year=int(datetime_str[0:4]),
                month=int(datetime_str[4:6]),
                day=int(datetime_str[6:8]),
                hour=int(datetime_str[8:10]),
                minute=int(datetime_str[10:12])
            )
        elif len(datetime_str) >= 8:
            # Format: YYYYMMDD
            return datetime(
                year=int(datetime_str[0:4]),
                month=int(datetime_str[4:6]),
                day=int(datetime_str[6:8])
            )
        else:
            # Default to current datetime
            return datetime.now()
    
    def _determine_observation_category(self, observation_id: str, observation_name: str) -> str:
        """Determine the observation category based on the observation ID or name."""
        # This is a simplified approach - in a real system, you would use a more sophisticated mapping
        observation_name_lower = observation_name.lower()
        
        # Check for laboratory observations
        lab_keywords = ["lab", "test", "blood", "urine", "serum", "plasma", "wbc", "rbc", "hgb", "hct", "plt", "glucose", "sodium", "potassium", "chloride", "bicarbonate", "bun", "creatinine", "calcium", "magnesium", "phosphorus", "protein", "albumin", "bilirubin", "ast", "alt", "alp", "ggt", "ldh", "amylase", "lipase", "cholesterol", "triglycerides", "hdl", "ldl", "a1c", "inr", "ptt", "pt"]
        for keyword in lab_keywords:
            if keyword in observation_name_lower:
                return "laboratory"
        
        # Check for vital signs
        vital_signs_keywords = ["vital", "bp", "blood pressure", "heart rate", "pulse", "temperature", "respiratory rate", "oxygen", "spo2", "height", "weight", "bmi", "body mass index"]
        for keyword in vital_signs_keywords:
            if keyword in observation_name_lower:
                return "vital-signs"
        
        # Check for imaging observations
        imaging_keywords = ["xray", "x-ray", "ct", "mri", "ultrasound", "echo", "echocardiogram", "pet", "nuclear", "radiology", "imaging"]
        for keyword in imaging_keywords:
            if keyword in observation_name_lower:
                return "imaging"
        
        # Check for social history observations
        social_history_keywords = ["smoking", "alcohol", "drug", "tobacco", "social", "occupation", "education", "marital", "sexual", "diet", "exercise", "travel"]
        for keyword in social_history_keywords:
            if keyword in observation_name_lower:
                return "social-history"
        
        # Check for survey observations
        survey_keywords = ["survey", "questionnaire", "assessment", "score", "scale", "phq", "gad", "mmse", "moca", "pain"]
        for keyword in survey_keywords:
            if keyword in observation_name_lower:
                return "survey"
        
        # Default to laboratory if we can't determine the category
        return "laboratory"
    
    def _map_hl7_status_to_fhir(self, hl7_status: str) -> str:
        """Map HL7 observation status to FHIR observation status."""
        # HL7 OBX-11 values: P (Preliminary), F (Final), C (Corrected), X (Cancelled)
        status_map = {
            "P": "preliminary",
            "F": "final",
            "C": "corrected",
            "X": "cancelled",
            "R": "registered",
            "I": "entered-in-error",
            "U": "unknown"
        }
        
        return status_map.get(hl7_status, "unknown")
    
    def _parse_reference_range(self, range_str: str, unit: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Parse a reference range string."""
        # Handle different reference range formats
        if "-" in range_str:
            # Format: "1-10"
            parts = range_str.split("-")
            if len(parts) == 2:
                try:
                    low = float(parts[0])
                    high = float(parts[1])
                    
                    return {
                        "low": {
                            "value": low,
                            "unit": unit,
                            "system": "http://unitsofmeasure.org",
                            "code": unit
                        },
                        "high": {
                            "value": high,
                            "unit": unit,
                            "system": "http://unitsofmeasure.org",
                            "code": unit
                        },
                        "text": range_str
                    }
                except ValueError:
                    pass
        elif range_str.startswith(">"):
            # Format: ">10"
            try:
                value = float(range_str[1:])
                
                return {
                    "low": {
                        "value": value,
                        "unit": unit,
                        "system": "http://unitsofmeasure.org",
                        "code": unit
                    },
                    "text": range_str
                }
            except ValueError:
                pass
        elif range_str.startswith("<"):
            # Format: "<10"
            try:
                value = float(range_str[1:])
                
                return {
                    "high": {
                        "value": value,
                        "unit": unit,
                        "system": "http://unitsofmeasure.org",
                        "code": unit
                    },
                    "text": range_str
                }
            except ValueError:
                pass
        
        # If we can't parse the range, just return the text
        return {
            "text": range_str
        }
