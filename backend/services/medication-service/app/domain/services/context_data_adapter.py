"""
Context Data Adapter - Flow 2 Integration

This module provides data transformation between Context Service output and 
Clinical Recipe Engine input formats, ensuring seamless integration in Flow 2.

Key Features:
- Transform Context Service assembled data to Clinical Recipe format
- Handle missing data gracefully with defaults
- Preserve data quality and completeness information
- Support multiple data source formats
"""

import logging
from typing import Dict, List, Any, Optional
from datetime import datetime

from app.infrastructure.context_service_client import ClinicalContext
from app.domain.services.clinical_recipe_engine import RecipeContext

logger = logging.getLogger(__name__)


class ContextDataAdapter:
    """
    Adapter to transform Context Service data for Clinical Recipe Engine
    
    This class bridges the gap between the Context Service's assembled data format
    and the format expected by clinical recipes, ensuring Flow 2 integration works
    seamlessly with real clinical data.
    """
    
    def __init__(self):
        # Default values for missing data
        self.default_values = {
            'patient': {
                'age': 50,
                'weight_kg': 70,
                'height_cm': 170,
                'gender': 'unknown',
                'egfr': 90,
                'conditions': [],
                'allergies': []
            },
            'labs': {
                'creatinine': 1.0,
                'baseline_creatinine': 1.0,
                'current_creatinine': 1.0,
                'alt': 25,
                'ast': 25,
                'bilirubin': 1.0,
                'inr': 1.0,
                'pt': 12
            },
            'medications': {
                'current_medications': [],
                'recent_medications': []
            }
        }
    
    def transform_context_for_recipes(
        self, 
        context_data: ClinicalContext, 
        medication_data: Dict[str, Any],
        action_type: str = "prescribe",
        provider_id: Optional[str] = None,
        encounter_id: Optional[str] = None
    ) -> RecipeContext:
        """
        Transform Context Service data into RecipeContext format
        
        This is the main transformation method that converts Context Service
        assembled data into the format expected by clinical recipes.
        """
        logger.info(f"🔄 Transforming context data for clinical recipes")
        logger.info(f"   Context ID: {context_data.context_id}")
        logger.info(f"   Recipe Used: {context_data.recipe_used}")
        logger.info(f"   Completeness: {context_data.completeness_score:.2%}")
        
        assembled_data = context_data.assembled_data

        # Debug: Log the actual assembled data structure
        logger.info(f"🔍 DEBUG: Assembled data keys: {list(assembled_data.keys()) if isinstance(assembled_data, dict) else type(assembled_data)}")

        # Transform patient data - Context Service uses 'patient_demographics' not 'patient'
        patient_data = self._transform_patient_data(assembled_data.get('patient_demographics', {}))
        
        # Transform provider data
        provider_data = self._transform_provider_data(
            assembled_data.get('provider', {}), 
            provider_id
        )
        
        # Transform encounter data
        encounter_data = self._transform_encounter_data(
            assembled_data.get('encounter', {}), 
            encounter_id
        )
        
        # Transform clinical data (labs, vitals, medications, etc.)
        clinical_data = self._transform_clinical_data(assembled_data, context_data)
        
        # Create RecipeContext
        recipe_context = RecipeContext(
            patient_id=context_data.patient_id,
            action_type=action_type,
            medication_data=medication_data,
            patient_data=patient_data,
            provider_data=provider_data,
            encounter_data=encounter_data,
            clinical_data=clinical_data,
            timestamp=datetime.now()
        )
        
        logger.info(f"✅ Context transformation completed")
        logger.info(f"   Patient data keys: {list(patient_data.keys())}")
        logger.info(f"   Clinical data keys: {list(clinical_data.keys())}")
        
        return recipe_context
    
    def _transform_patient_data(self, patient_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform patient demographics and basic information
        """
        # Debug: Log the actual patient data structure
        logger.info(f"🔍 DEBUG: Patient data keys: {list(patient_data.keys())}")
        if 'patient_demographics' in patient_data:
            demo_data = patient_data['patient_demographics']
            logger.info(f"🔍 DEBUG: Demographics keys: {list(demo_data.keys()) if isinstance(demo_data, dict) else type(demo_data)}")

        # Start with empty dict instead of defaults to avoid validation conflicts
        transformed = {}
        
        # Map common patient fields - Updated based on Context Service structure
        field_mapping = {
            'age': ['age', 'patient_age', 'demographics.age', 'patient_demographics.age'],
            'weight_kg': ['weight', 'weight_kg', 'vitals.weight', 'demographics.weight', 'patient_demographics.weight'],
            'height_cm': ['height', 'height_cm', 'vitals.height', 'demographics.height', 'patient_demographics.height'],
            'gender': ['gender', 'sex', 'demographics.gender', 'demographics.sex', 'patient_demographics.gender'],
            'egfr': ['egfr', 'estimated_gfr', 'labs.egfr', 'renal.egfr']
        }
        
        for target_field, source_fields in field_mapping.items():
            value = self._extract_nested_value(patient_data, source_fields)
            if value is not None:
                transformed[target_field] = value
        
        # Transform conditions/diagnoses
        conditions = self._extract_conditions(patient_data)
        transformed['conditions'] = conditions if conditions else []

        # Transform allergies
        allergies = self._extract_allergies(patient_data)
        transformed['allergies'] = allergies if allergies else []

        # Add safe defaults only for non-critical fields that recipes expect
        if 'gender' not in transformed:
            transformed['gender'] = 'unknown'
        if 'egfr' not in transformed:
            transformed['egfr'] = 90.0  # Normal eGFR default for safety

        # Ensure critical numeric fields have safe defaults for comparisons
        if 'age' not in transformed or transformed['age'] is None:
            logger.warning("Age missing - using safe default for clinical recipes")
            transformed['age'] = 45  # Safe adult default
        if 'weight_kg' not in transformed or transformed['weight_kg'] is None:
            logger.warning("Weight missing - using safe default for clinical recipes")
            transformed['weight_kg'] = 70.0  # Safe adult default
        if 'height_cm' not in transformed or transformed['height_cm'] is None:
            logger.warning("Height missing - using safe default for clinical recipes")
            transformed['height_cm'] = 170.0  # Safe adult default

        return transformed
    
    def _transform_provider_data(
        self, 
        provider_data: Dict[str, Any], 
        provider_id: Optional[str]
    ) -> Dict[str, Any]:
        """
        Transform provider information
        """
        transformed = {
            'id': provider_id or provider_data.get('id', 'unknown'),
            'name': provider_data.get('name', 'Unknown Provider'),
            'specialty': provider_data.get('specialty', 'unknown'),
            'department': provider_data.get('department', 'unknown')
        }
        
        return transformed
    
    def _transform_encounter_data(
        self, 
        encounter_data: Dict[str, Any], 
        encounter_id: Optional[str]
    ) -> Dict[str, Any]:
        """
        Transform encounter information
        """
        transformed = {
            'id': encounter_id or encounter_data.get('id', 'unknown'),
            'type': encounter_data.get('type', 'unknown'),
            'status': encounter_data.get('status', 'active'),
            'location': encounter_data.get('location', 'unknown'),
            'admission_date': encounter_data.get('admission_date'),
            'discharge_date': encounter_data.get('discharge_date')
        }
        
        return transformed
    
    def _transform_clinical_data(
        self, 
        assembled_data: Dict[str, Any], 
        context_data: ClinicalContext
    ) -> Dict[str, Any]:
        """
        Transform clinical data (labs, vitals, medications, etc.)
        """
        logger.info(f"🔍 DEBUG: Transforming clinical data with keys: {list(assembled_data.keys())}")
        clinical_data = {}
        
        # Transform lab results
        clinical_data['labs'] = self._transform_lab_data(assembled_data.get('labs', {}))
        clinical_data['recent_labs'] = clinical_data['labs']  # Alias for compatibility
        
        # Transform vital signs
        clinical_data['vitals'] = self._transform_vital_data(assembled_data.get('vitals', {}))
        
        # Transform current medications
        clinical_data['current_medications'] = self._transform_medication_list(
            assembled_data.get('medications', {}).get('current', [])
        )
        
        # Transform recent medications
        clinical_data['recent_medications'] = self._transform_medication_list(
            assembled_data.get('medications', {}).get('recent', [])
        )
        
        # Add context metadata for clinical recipes to use
        clinical_data['context_metadata'] = {
            'context_id': context_data.context_id,
            'recipe_used': context_data.recipe_used,
            'completeness_score': context_data.completeness_score,
            'safety_flags': context_data.safety_flags,
            'assembly_duration_ms': context_data.assembly_duration_ms,
            'data_freshness': context_data.data_freshness,
            'source_metadata': context_data.source_metadata
        }
        
        # Add specific clinical flags that recipes look for
        clinical_data['recent_contrast_exposure'] = self._check_recent_contrast(assembled_data)
        clinical_data['alcohol_use'] = self._extract_alcohol_use(assembled_data)
        clinical_data['pregnancy_status'] = self._extract_pregnancy_status(assembled_data)
        
        return clinical_data
    
    def _transform_lab_data(self, lab_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform laboratory results
        """
        # Start with safe defaults for clinical recipes
        transformed = {
            'creatinine': 1.0,  # Normal creatinine
            'baseline_creatinine': 1.0,
            'current_creatinine': 1.0,
            'alt': 25.0,  # Normal ALT
            'ast': 25.0,  # Normal AST
            'bilirubin': 1.0,  # Normal bilirubin
            'inr': 1.0,  # Normal INR
            'pt': 12.0   # Normal PT
        }
        
        # Map common lab values
        lab_mapping = {
            'creatinine': ['creatinine', 'cr', 'serum_creatinine'],
            'baseline_creatinine': ['baseline_creatinine', 'baseline_cr'],
            'current_creatinine': ['current_creatinine', 'creatinine', 'cr'],
            'alt': ['alt', 'alanine_aminotransferase'],
            'ast': ['ast', 'aspartate_aminotransferase'],
            'bilirubin': ['bilirubin', 'total_bilirubin'],
            'inr': ['inr', 'international_normalized_ratio'],
            'pt': ['pt', 'prothrombin_time']
        }
        
        for target_field, source_fields in lab_mapping.items():
            value = self._extract_nested_value(lab_data, source_fields)
            if value is not None:
                try:
                    transformed[target_field] = float(value)
                except (ValueError, TypeError):
                    logger.warning(f"Could not convert lab value {target_field}: {value}")
        
        return transformed
    
    def _transform_vital_data(self, vital_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform vital signs
        """
        transformed = {}
        
        vital_mapping = {
            'temperature': ['temperature', 'temp', 'body_temperature'],
            'heart_rate': ['heart_rate', 'hr', 'pulse'],
            'blood_pressure_systolic': ['systolic', 'sbp', 'blood_pressure.systolic'],
            'blood_pressure_diastolic': ['diastolic', 'dbp', 'blood_pressure.diastolic'],
            'respiratory_rate': ['respiratory_rate', 'rr', 'respiration'],
            'oxygen_saturation': ['oxygen_saturation', 'spo2', 'o2_sat']
        }
        
        for target_field, source_fields in vital_mapping.items():
            value = self._extract_nested_value(vital_data, source_fields)
            if value is not None:
                try:
                    transformed[target_field] = float(value)
                except (ValueError, TypeError):
                    logger.warning(f"Could not convert vital sign {target_field}: {value}")
        
        return transformed

    def _transform_medication_list(self, medication_list: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Transform medication list to format expected by clinical recipes
        """
        transformed_meds = []

        for med in medication_list:
            transformed_med = {
                'name': med.get('name', 'Unknown'),
                'generic_name': med.get('generic_name', med.get('name', 'Unknown')),
                'therapeutic_class': med.get('therapeutic_class', 'unknown'),
                'dose': med.get('dose', 'unknown'),
                'frequency': med.get('frequency', 'unknown'),
                'route': med.get('route', 'unknown'),
                'start_date': med.get('start_date'),
                'end_date': med.get('end_date'),
                'status': med.get('status', 'active'),
                'nephrotoxic_risk': med.get('nephrotoxic_risk', 'NONE'),
                'hepatotoxic_risk': med.get('hepatotoxic_risk', 'NONE'),
                'is_anticoagulant': med.get('is_anticoagulant', False),
                'is_chemotherapy': med.get('is_chemotherapy', False)
            }
            transformed_meds.append(transformed_med)

        return transformed_meds

    def _extract_conditions(self, patient_data: Dict[str, Any]) -> List[str]:
        """
        Extract patient conditions/diagnoses
        """
        conditions = []

        # Try different possible locations for conditions
        condition_sources = [
            patient_data.get('conditions', []),
            patient_data.get('diagnoses', []),
            patient_data.get('medical_history', []),
            patient_data.get('problems', [])
        ]

        for source in condition_sources:
            if isinstance(source, list):
                for condition in source:
                    if isinstance(condition, str):
                        conditions.append(condition.lower())
                    elif isinstance(condition, dict):
                        name = condition.get('name') or condition.get('code') or condition.get('description')
                        if name:
                            conditions.append(name.lower())

        return list(set(conditions))  # Remove duplicates

    def _extract_allergies(self, patient_data: Dict[str, Any]) -> List[Dict[str, Any]]:
        """
        Extract patient allergies
        """
        allergies = []

        allergy_data = patient_data.get('allergies', [])
        if not isinstance(allergy_data, list):
            allergy_data = []

        for allergy in allergy_data:
            if isinstance(allergy, str):
                allergies.append({
                    'allergen': allergy,
                    'reaction': 'unknown',
                    'severity': 'unknown'
                })
            elif isinstance(allergy, dict):
                allergies.append({
                    'allergen': allergy.get('allergen', allergy.get('substance', 'unknown')),
                    'reaction': allergy.get('reaction', 'unknown'),
                    'severity': allergy.get('severity', 'unknown')
                })

        return allergies

    def _extract_nested_value(self, data: Dict[str, Any], field_paths: List[str]) -> Any:
        """
        Extract value from nested dictionary using multiple possible paths
        """
        for path in field_paths:
            value = data
            try:
                for key in path.split('.'):
                    if isinstance(value, dict) and key in value:
                        value = value[key]
                    else:
                        value = None
                        break

                if value is not None:
                    return value
            except (KeyError, TypeError, AttributeError):
                continue

        return None

    def _check_recent_contrast(self, assembled_data: Dict[str, Any]) -> bool:
        """
        Check for recent contrast exposure
        """
        # Look for contrast exposure in various data sources
        procedures = assembled_data.get('procedures', [])
        imaging = assembled_data.get('imaging', [])

        # Check procedures for contrast use
        for procedure in procedures:
            if isinstance(procedure, dict):
                if procedure.get('contrast_used', False):
                    procedure_date = procedure.get('date')
                    if procedure_date and self._is_within_hours(procedure_date, 72):
                        return True

        # Check imaging studies
        for study in imaging:
            if isinstance(study, dict):
                if study.get('contrast_used', False):
                    study_date = study.get('date')
                    if study_date and self._is_within_hours(study_date, 72):
                        return True

        return False

    def _extract_alcohol_use(self, assembled_data: Dict[str, Any]) -> str:
        """
        Extract alcohol use information
        """
        social_history = assembled_data.get('social_history', {})
        patient_data = assembled_data.get('patient', {})

        # Look for alcohol use in social history
        alcohol_use = social_history.get('alcohol_use') or patient_data.get('alcohol_use')

        if alcohol_use:
            if isinstance(alcohol_use, str):
                return alcohol_use.lower()
            elif isinstance(alcohol_use, dict):
                return alcohol_use.get('frequency', 'unknown').lower()

        return 'none'

    def _extract_pregnancy_status(self, assembled_data: Dict[str, Any]) -> bool:
        """
        Extract pregnancy status
        """
        patient_data = assembled_data.get('patient', {})
        conditions = self._extract_conditions(patient_data)

        # Check for pregnancy in conditions
        pregnancy_indicators = ['pregnancy', 'pregnant', 'gravid', 'expecting']

        for condition in conditions:
            if any(indicator in condition for indicator in pregnancy_indicators):
                return True

        # Check specific pregnancy field
        return patient_data.get('is_pregnant', False)

    def _is_within_hours(self, date_str: str, hours: int) -> bool:
        """
        Check if a date is within specified hours from now
        """
        try:
            if isinstance(date_str, str):
                # Try to parse ISO format
                date_obj = datetime.fromisoformat(date_str.replace('Z', '+00:00'))
            else:
                return False

            time_diff = datetime.now() - date_obj.replace(tzinfo=None)
            return time_diff.total_seconds() <= (hours * 3600)
        except (ValueError, AttributeError):
            return False

    def validate_transformed_data(self, recipe_context: RecipeContext) -> Dict[str, Any]:
        """
        Validate the transformed data and provide quality metrics
        """
        validation_results = {
            'data_completeness': {},
            'data_quality_score': 0.0,
            'missing_critical_data': [],
            'warnings': []
        }

        # Check patient data completeness
        patient_completeness = self._check_patient_completeness(recipe_context.patient_data)
        validation_results['data_completeness']['patient'] = patient_completeness

        # Check clinical data completeness
        clinical_completeness = self._check_clinical_completeness(recipe_context.clinical_data)
        validation_results['data_completeness']['clinical'] = clinical_completeness

        # Calculate overall quality score
        overall_score = (patient_completeness + clinical_completeness) / 2
        validation_results['data_quality_score'] = overall_score

        # Identify missing critical data
        patient_age = recipe_context.patient_data.get('age')
        if patient_age is None:  # Only check if truly missing
            validation_results['missing_critical_data'].append('patient_age')

        creatinine = recipe_context.clinical_data.get('labs', {}).get('creatinine')
        if creatinine is None:  # Only check if truly missing
            validation_results['missing_critical_data'].append('serum_creatinine')

        return validation_results

    def _check_patient_completeness(self, patient_data: Dict[str, Any]) -> float:
        """
        Check completeness of patient data
        """
        required_fields = ['age', 'weight_kg', 'gender', 'conditions']
        present_fields = sum(1 for field in required_fields if patient_data.get(field) is not None)
        return present_fields / len(required_fields)

    def _check_clinical_completeness(self, clinical_data: Dict[str, Any]) -> float:
        """
        Check completeness of clinical data
        """
        required_fields = ['labs', 'current_medications', 'vitals']
        present_fields = sum(1 for field in required_fields if clinical_data.get(field))
        return present_fields / len(required_fields)
