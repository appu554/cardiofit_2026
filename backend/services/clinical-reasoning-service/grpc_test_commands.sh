#!/bin/bash
# gRPC Test Commands for Clinical Assertion Engine (CAE)
# This script contains commands to test CAE gRPC endpoints using grpcurl

# Set the server address
SERVER="localhost:8027"

echo "======================================================================================"
echo "Clinical Assertion Engine (CAE) gRPC Test Suite"
echo "======================================================================================"
echo "This file contains gRPCurl commands to test standard cases and edge cases"
echo "======================================================================================"

# Helper function to format JSON output
format_output() {
  # On Windows, you might need to install jq
  # Output is piped to jq for pretty formatting if available
  if command -v jq &> /dev/null; then
    jq .
  else
    cat
  fi
}

echo -e "\n\n======================================================================================"
echo "Health Check"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/HealthCheck"
grpcurl -plaintext -d '{"service": "clinical-reasoning-service"}' $SERVER clinical_reasoning.ClinicalReasoningService/HealthCheck | format_output

echo -e "\n\n======================================================================================"
echo "1. STANDARD CASES"
echo "======================================================================================"

echo -e "\n\n======================================================================================"
echo "1.1 Generate Clinical Assertions - Standard Patient"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
grpcurl -plaintext -d '{
  "request_id": "standard-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "905a60cb-8241-418f-b29b-5b020e851392"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  },
  "priority": "PRIORITY_NORMAL",
  "reasoner_types": ["interaction", "dosing", "contraindication", "duplicate_therapy", "clinical_context"]
}' $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | format_output

echo -e "\n\n======================================================================================"
echo "1.2 Medication Interactions Check - Standard Patient"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckMedicationInteractions"
grpcurl -plaintext -d '{
  "request_id": "interaction-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "905a60cb-8241-418f-b29b-5b020e851392"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}' $SERVER clinical_reasoning.ClinicalReasoningService/CheckMedicationInteractions | format_output

echo -e "\n\n======================================================================================"
echo "1.3 Dose Adjustment Check - Patient with Renal Impairment"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckDoseAdjustments"
grpcurl -plaintext -d '{
  "request_id": "dose-adjustment-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_009"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  },
  "medication_names": ["metformin", "lisinopril"]
}' $SERVER clinical_reasoning.ClinicalReasoningService/CheckDoseAdjustments | format_output

echo -e "\n\n======================================================================================"
echo "1.4 Contraindications Check - Patient with Allergies"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckContraindications"
grpcurl -plaintext -d '{
  "request_id": "contraindication-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_008"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  },
  "medication_names": ["amoxicillin", "ibuprofen"]
}' $SERVER clinical_reasoning.ClinicalReasoningService/CheckContraindications | format_output

echo -e "\n\n======================================================================================"
echo "1.5 Duplicate Therapy Check - Cardiovascular Patient"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckDuplicateTherapy"
grpcurl -plaintext -d '{
  "request_id": "duplicate-therapy-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_001"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  },
  "new_medication_name": "enalapril"
}' $SERVER clinical_reasoning.ClinicalReasoningService/CheckDuplicateTherapy | format_output

echo -e "\n\n======================================================================================"
echo "2. EDGE CASES"
echo "======================================================================================"

echo -e "\n\n======================================================================================"
echo "2.1 Partial Patient Data Test - Missing Allergies and Conditions"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
grpcurl -plaintext -d '{
  "request_id": "partial-data-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_015"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": false},
      "include_allergies": {"bool_value": false}
    }
  }
}' $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | format_output

echo -e "\n\n======================================================================================"
echo "2.2 Orphan Disease Test - Patient with Rare Condition"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
grpcurl -plaintext -d '{
  "request_id": "orphan-disease-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_010"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}' $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | format_output

echo -e "\n\n======================================================================================"
echo "2.3 Extreme Polypharmacy Test - Patient on 20+ Medications"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckMedicationInteractions"
grpcurl -plaintext -d '{
  "request_id": "polypharmacy-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_018"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}' $SERVER clinical_reasoning.ClinicalReasoningService/CheckMedicationInteractions | format_output

echo -e "\n\n======================================================================================"
echo "2.4 Learning Override Test - Submit Clinical Feedback"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/SubmitLearningFeedback"
echo "First, get an assertion ID from a previous test"
grpcurl -plaintext -d '{
  "request_id": "learning-feedback-test-'$(date +%s)'",
  "assertion_id": "REPLACE_WITH_REAL_ASSERTION_ID",
  "feedback_type": "OVERRIDE",
  "clinician_comment": "Clinical judgment indicates this interaction is not relevant for this specific patient due to temporary therapy duration.",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_015"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}' $SERVER clinical_reasoning.ClinicalReasoningService/SubmitLearningFeedback | format_output

echo -e "\n\n======================================================================================"
echo "2.5 Error Handling - Invalid Patient ID"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
grpcurl -plaintext -d '{
  "request_id": "invalid-patient-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "non_existent_patient_id"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}' $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | format_output

echo -e "\n\n======================================================================================"
echo "2.6 Error Handling - Malformed Context (Missing Required Fields)"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
grpcurl -plaintext -d '{
  "request_id": "malformed-context-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_015"}
      // Missing source and other fields
    }
  }
}' $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | format_output

echo -e "\n\n======================================================================================"
echo "2.7 Error Handling - Empty Request"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
grpcurl -plaintext -d '{}' $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | format_output

echo -e "\n\n======================================================================================"
echo "2.8 Special Populations - Pediatric Patient"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
grpcurl -plaintext -d '{
  "request_id": "pediatric-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_004"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}' $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | format_output

echo -e "\n\n======================================================================================"
echo "2.9 Special Populations - Geriatric Patient"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckDoseAdjustments"
grpcurl -plaintext -d '{
  "request_id": "geriatric-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_005"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  },
  "medication_names": ["rivaroxaban", "digoxin"]
}' $SERVER clinical_reasoning.ClinicalReasoningService/CheckDoseAdjustments | format_output

echo -e "\n\n======================================================================================"
echo "2.10 Special Populations - Pregnant Patient"
echo "======================================================================================"
echo "grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckContraindications"
grpcurl -plaintext -d '{
  "request_id": "pregnancy-test-'$(date +%s)'",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_006"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  },
  "medication_names": ["warfarin", "lisinopril"]
}' $SERVER clinical_reasoning.ClinicalReasoningService/CheckContraindications | format_output
