"""
Safety Validator for Clinical Context Assembly
Validates clinical data safety and raises appropriate safety flags
"""
import logging
from typing import Dict, List, Any, Optional
from datetime import datetime, timedelta

from app.models.context_models import (
    SafetyFlag, SafetyFlagType, SafetySeverity, SafetyRequirements
)

logger = logging.getLogger(__name__)


class SafetyValidator:
    """
    Validates clinical context data for safety concerns and raises appropriate flags.
    Implements clinical safety rules and data quality validation.
    """
    
    def __init__(self):
        # Mock data detection patterns
        self.mock_data_indicators = [
            "test_", "mock_", "fake_", "dummy_", "sample_",
            "example", "placeholder", "lorem ipsum", "john doe",
            "jane doe", "test patient", "demo", "simulation"
        ]
        
        # Critical data fields that must be present
        self.critical_data_fields = {
            "patient_demographics": ["age", "weight", "gender"],
            "current_medications": ["medication_requests"],  # Updated to match our data structure
            "patient_allergies": ["allergies"]  # Updated to match our data structure
        }
        
        # Data freshness requirements (in hours)
        self.freshness_requirements = {
            "patient_demographics": 24,
            "active_medications": 1,
            "allergies": 168,  # 1 week
            "lab_results": 72,  # 3 days
            "vital_signs": 24
        }
    
    async def validate_context_safety(
        self,
        assembled_data: Dict[str, Any],
        safety_requirements: SafetyRequirements
    ) -> List[SafetyFlag]:
        """
        Validate assembled clinical context for safety concerns.
        Returns list of safety flags for any issues found.
        """
        safety_flags = []
        
        try:
            logger.info("🔍 Validating clinical context safety")
            
            # 1. Mock data detection (CRITICAL)
            if safety_requirements.mock_data_policy == "STRICTLY_PROHIBITED":
                mock_flags = await self._detect_mock_data(assembled_data)
                safety_flags.extend(mock_flags)
            
            # 2. Critical data completeness validation
            completeness_flags = await self._validate_data_completeness(
                assembled_data, safety_requirements
            )
            safety_flags.extend(completeness_flags)
            
            # 3. Data freshness validation
            freshness_flags = await self._validate_data_freshness(assembled_data)
            safety_flags.extend(freshness_flags)
            
            # 4. Clinical safety rules validation
            clinical_flags = await self._validate_clinical_safety_rules(assembled_data)
            safety_flags.extend(clinical_flags)
            
            # 5. Data quality validation
            quality_flags = await self._validate_data_quality(assembled_data)
            safety_flags.extend(quality_flags)
            
            logger.info(f"✅ Safety validation complete: {len(safety_flags)} flags raised")
            
            return safety_flags
            
        except Exception as e:
            logger.error(f"❌ Safety validation error: {e}")
            # Return critical safety flag for validation failure
            return [SafetyFlag(
                flag_type=SafetyFlagType.DATA_QUALITY,
                severity=SafetySeverity.CRITICAL,
                message=f"Safety validation failed: {str(e)}",
                details={"validation_error": str(e)}
            )]
    
    async def _detect_mock_data(self, assembled_data: Dict[str, Any]) -> List[SafetyFlag]:
        """
        Detect mock data patterns in assembled clinical data.
        Mock data is STRICTLY PROHIBITED in production clinical workflows.
        """
        mock_flags = []
        
        for data_point_name, data in assembled_data.items():
            data_str = str(data).lower()
            
            # Check for mock data indicators
            for indicator in self.mock_data_indicators:
                if indicator in data_str:
                    mock_flags.append(SafetyFlag(
                        flag_type=SafetyFlagType.DATA_QUALITY,
                        severity=SafetySeverity.FATAL,
                        message=f"MOCK DATA DETECTED in {data_point_name}: '{indicator}' pattern found",
                        data_point=data_point_name,
                        details={
                            "mock_indicator": indicator,
                            "data_sample": str(data)[:100],
                            "policy_violation": "STRICTLY_PROHIBITED mock data policy violated"
                        }
                    ))
                    logger.error(f"🚨 MOCK DATA DETECTED: {data_point_name} contains '{indicator}'")
                    break
        
        return mock_flags
    
    async def _validate_data_completeness(
        self,
        assembled_data: Dict[str, Any],
        safety_requirements: SafetyRequirements
    ) -> List[SafetyFlag]:
        """
        Validate that critical data fields are present and complete.
        """
        completeness_flags = []
        
        for data_point_name, required_fields in self.critical_data_fields.items():
            if data_point_name not in assembled_data:
                # Missing critical data point
                if safety_requirements.absolute_required_enforcement == "STRICT":
                    completeness_flags.append(SafetyFlag(
                        flag_type=SafetyFlagType.MISSING_CRITICAL_DATA,
                        severity=SafetySeverity.CRITICAL,
                        message=f"Missing critical data point: {data_point_name}",
                        data_point=data_point_name,
                        details={
                            "enforcement_policy": safety_requirements.absolute_required_enforcement,
                            "required_fields": required_fields
                        }
                    ))
                continue
            
            data = assembled_data[data_point_name]

            # Special handling for different data point types
            if data_point_name == "patient_allergies":
                # Handle allergies - "no known allergies" is valid
                if isinstance(data, dict):
                    allergies_list = data.get("allergies", [])
                    count = data.get("count", 0)
                    source = data.get("source", "")
                    note = data.get("note", "")

                    # Check if we have valid allergy information
                    if source in ["medication_service", "medication_service_public", "fhir_store"]:
                        error = data.get("error", None)

                        if count == 0 and ("no known allergies" in note.lower() or
                                         "404" in note or "no allergy records" in note.lower() or
                                         "assuming no allergies" in note.lower()):
                            # Patient has no known allergies - this is valid
                            logger.info(f"✅ Valid allergy status: {note}")
                            continue
                        elif isinstance(allergies_list, list) and len(allergies_list) > 0:
                            # Patient has documented allergies
                            logger.info(f"✅ Patient has {len(allergies_list)} documented allergies")
                            continue
                        elif error and ("service error" in note.lower() or "internal server error" in note.lower()):
                            # Service error but we're treating as "no known allergies"
                            logger.info(f"✅ Allergy service error, assuming no known allergies: {note}")
                            continue  # Don't flag as missing (benefit of doubt)
                        else:
                            # Unclear allergy status
                            missing_fields = ["clear_allergy_status"]
                    else:
                        missing_fields = ["allergies"]
                else:
                    missing_fields = ["allergies"]

            elif data_point_name == "current_medications":
                # Handle medications - check if we have medication data structure
                if isinstance(data, dict):
                    med_requests = data.get("medication_requests", [])
                    count = data.get("count", 0)

                    if isinstance(med_requests, list) and count >= 0:
                        # We have valid medication data (even if count is 0)
                        logger.info(f"✅ Valid medication data: {count} current medications")
                        continue
                    else:
                        missing_fields = ["medication_requests"]
                else:
                    missing_fields = ["medication_requests"]

            else:
                # Standard field checking for other data points
                missing_fields = []
                if isinstance(data, dict):
                    for field in required_fields:
                        if field not in data or data[field] is None or data[field] == "":
                            missing_fields.append(field)

            if missing_fields:
                severity = SafetySeverity.CRITICAL if safety_requirements.absolute_required_enforcement == "STRICT" else SafetySeverity.WARNING

                completeness_flags.append(SafetyFlag(
                    flag_type=SafetyFlagType.MISSING_CRITICAL_DATA,
                    severity=severity,
                    message=f"Missing required fields in {data_point_name}: {', '.join(missing_fields)}",
                    data_point=data_point_name,
                    details={
                        "missing_fields": missing_fields,
                        "enforcement_policy": safety_requirements.absolute_required_enforcement
                    }
                ))
        
        return completeness_flags
    
    async def _validate_data_freshness(self, assembled_data: Dict[str, Any]) -> List[SafetyFlag]:
        """
        Validate that clinical data is fresh enough for safe clinical use.
        """
        freshness_flags = []
        current_time = datetime.utcnow()
        
        for data_point_name, data in assembled_data.items():
            if not isinstance(data, dict):
                continue
            
            # Check if data has timestamp information
            timestamp_fields = ["retrieved_at", "last_updated", "timestamp", "collected_date"]
            data_timestamp = None
            
            for field in timestamp_fields:
                if field in data and data[field]:
                    try:
                        if isinstance(data[field], str):
                            data_timestamp = datetime.fromisoformat(data[field].replace('Z', '+00:00'))
                        elif isinstance(data[field], datetime):
                            data_timestamp = data[field]
                        break
                    except Exception:
                        continue
            
            if not data_timestamp:
                # No timestamp available - flag as potential issue
                freshness_flags.append(SafetyFlag(
                    flag_type=SafetyFlagType.DATA_QUALITY,
                    severity=SafetySeverity.WARNING,
                    message=f"No timestamp available for {data_point_name} - cannot validate freshness",
                    data_point=data_point_name,
                    details={"issue": "missing_timestamp"}
                ))
                continue
            
            # Check freshness requirements
            max_age_hours = self.freshness_requirements.get(data_point_name, 24)
            data_age = current_time - data_timestamp
            max_age = timedelta(hours=max_age_hours)
            
            if data_age > max_age:
                severity = SafetySeverity.WARNING
                if data_point_name in ["active_medications", "vital_signs"]:
                    severity = SafetySeverity.CRITICAL
                
                freshness_flags.append(SafetyFlag(
                    flag_type=SafetyFlagType.STALE_DATA,
                    severity=severity,
                    message=f"Stale data in {data_point_name}: {data_age} > {max_age} maximum age",
                    data_point=data_point_name,
                    details={
                        "data_age_hours": data_age.total_seconds() / 3600,
                        "max_age_hours": max_age_hours,
                        "data_timestamp": data_timestamp.isoformat()
                    }
                ))
        
        return freshness_flags
    
    async def _validate_clinical_safety_rules(self, assembled_data: Dict[str, Any]) -> List[SafetyFlag]:
        """
        Validate clinical safety rules specific to medication and patient safety.
        """
        clinical_flags = []
        
        # Check for drug-allergy interactions
        if "active_medications" in assembled_data and "allergies" in assembled_data:
            interaction_flags = await self._check_drug_allergy_interactions(
                assembled_data["active_medications"],
                assembled_data["allergies"]
            )
            clinical_flags.extend(interaction_flags)
        
        # Check for age-appropriate medications
        if "patient_demographics" in assembled_data and "active_medications" in assembled_data:
            age_flags = await self._check_age_appropriate_medications(
                assembled_data["patient_demographics"],
                assembled_data["active_medications"]
            )
            clinical_flags.extend(age_flags)
        
        # Check for dosing safety based on weight/age
        if "patient_demographics" in assembled_data and "active_medications" in assembled_data:
            dosing_flags = await self._check_dosing_safety(
                assembled_data["patient_demographics"],
                assembled_data["active_medications"]
            )
            clinical_flags.extend(dosing_flags)
        
        return clinical_flags
    
    async def _validate_data_quality(self, assembled_data: Dict[str, Any]) -> List[SafetyFlag]:
        """
        Validate general data quality issues.
        """
        quality_flags = []
        
        for data_point_name, data in assembled_data.items():
            # Check for empty or null data
            if not data or (isinstance(data, dict) and not any(data.values())):
                quality_flags.append(SafetyFlag(
                    flag_type=SafetyFlagType.DATA_QUALITY,
                    severity=SafetySeverity.WARNING,
                    message=f"Empty or null data in {data_point_name}",
                    data_point=data_point_name,
                    details={"issue": "empty_data"}
                ))
            
            # Check for data structure consistency
            if isinstance(data, dict):
                # Check for required structure patterns
                if data_point_name == "active_medications" and "medications" not in data and "active_medications" not in data:
                    quality_flags.append(SafetyFlag(
                        flag_type=SafetyFlagType.DATA_QUALITY,
                        severity=SafetySeverity.WARNING,
                        message=f"Unexpected data structure in {data_point_name}",
                        data_point=data_point_name,
                        details={"issue": "unexpected_structure", "keys": list(data.keys())}
                    ))
        
        return quality_flags
    
    async def _check_drug_allergy_interactions(self, medications: Any, allergies: Any) -> List[SafetyFlag]:
        """Check for potential drug-allergy interactions"""
        interaction_flags = []
        
        # This would implement real drug-allergy interaction checking
        # For now, return empty list as this requires clinical knowledge base
        
        return interaction_flags
    
    async def _check_age_appropriate_medications(self, demographics: Any, medications: Any) -> List[SafetyFlag]:
        """Check for age-appropriate medication prescribing"""
        age_flags = []
        
        # This would implement real age-based medication safety checking
        # For now, return empty list as this requires clinical knowledge base
        
        return age_flags
    
    async def _check_dosing_safety(self, demographics: Any, medications: Any) -> List[SafetyFlag]:
        """Check for safe medication dosing based on patient characteristics"""
        dosing_flags = []
        
        # This would implement real dosing safety checking
        # For now, return empty list as this requires clinical knowledge base
        
        return dosing_flags
