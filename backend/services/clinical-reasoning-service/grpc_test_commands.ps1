# PowerShell Script for Clinical Assertion Engine (CAE) gRPC Tests
# This script contains commands to test CAE gRPC endpoints using grpcurl on Windows

# Set the server address
$SERVER = "localhost:8027"

Write-Host "======================================================================================"
Write-Host "Clinical Assertion Engine (CAE) gRPC Test Suite"
Write-Host "======================================================================================"
Write-Host "This file contains gRPCurl commands to test standard cases and edge cases"
Write-Host "======================================================================================"

# Note: You need to have grpcurl installed and in your PATH
# Download from: https://github.com/fullstorydev/grpcurl/releases

# Helper function to format JSON output
function Format-JsonOutput {
    param (
        [Parameter(ValueFromPipeline=$true)]
        [string]$InputObject
    )
    
    process {
        # Try to format with ConvertFrom-Json if it's valid JSON
        try {
            $InputObject | ConvertFrom-Json | ConvertTo-Json -Depth 10
        } catch {
            # If not valid JSON, return the original string
            $InputObject
        }
    }
}

Write-Host "`n`n======================================================================================"
Write-Host "Health Check"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/HealthCheck"
grpcurl -plaintext -d '{\"service\": \"clinical-reasoning-service\"}' $SERVER clinical_reasoning.ClinicalReasoningService/HealthCheck | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "1. STANDARD CASES"
Write-Host "======================================================================================"

Write-Host "`n`n======================================================================================"
Write-Host "1.1 Generate Clinical Assertions - Standard Patient"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "standard-test-$timestamp",
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
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "1.2 Medication Interactions Check - Standard Patient"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckMedicationInteractions"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "interaction-test-$timestamp",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "905a60cb-8241-418f-b29b-5b020e851392"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/CheckMedicationInteractions | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "1.3 Dose Adjustment Check - Patient with Renal Impairment"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckDoseAdjustments"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "dose-adjustment-test-$timestamp",
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
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/CheckDoseAdjustments | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "1.4 Contraindications Check - Patient with Allergies"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckContraindications"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "contraindication-test-$timestamp",
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
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/CheckContraindications | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "1.5 Duplicate Therapy Check - Cardiovascular Patient"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckDuplicateTherapy"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "duplicate-therapy-test-$timestamp",
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
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/CheckDuplicateTherapy | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2. EDGE CASES"
Write-Host "======================================================================================"

Write-Host "`n`n======================================================================================"
Write-Host "2.1 Partial Patient Data Test - Missing Allergies and Conditions"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "partial-data-test-$timestamp",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_015"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": false},
      "include_allergies": {"bool_value": false}
    }
  }
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2.2 Orphan Disease Test - Patient with Rare Condition"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "orphan-disease-test-$timestamp",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_010"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2.3 Extreme Polypharmacy Test - Patient on 20+ Medications"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckMedicationInteractions"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "polypharmacy-test-$timestamp",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_018"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/CheckMedicationInteractions | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2.4 Learning Override Test - Submit Clinical Feedback"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/SubmitLearningFeedback"
Write-Host "First, get an assertion ID from a previous test"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "learning-feedback-test-$timestamp",
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
}
"@
Write-Host "NOTE: You need to replace 'REPLACE_WITH_REAL_ASSERTION_ID' with an actual assertion ID first"
# Commented out until you have a real assertion ID
# grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/SubmitLearningFeedback | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2.5 Error Handling - Invalid Patient ID"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "invalid-patient-test-$timestamp",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "non_existent_patient_id"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2.6 Error Handling - Malformed Context (Missing Required Fields)"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "malformed-context-test-$timestamp",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_015"}
    }
  }
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2.7 Error Handling - Empty Request"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
grpcurl -plaintext -d '{}' $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2.8 Special Populations - Pediatric Patient"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "pediatric-test-$timestamp",
  "patient_context": {
    "fields": {
      "patient_id": {"string_value": "patient_004"},
      "source": {"string_value": "graphdb"},
      "include_medications": {"bool_value": true},
      "include_conditions": {"bool_value": true},
      "include_allergies": {"bool_value": true}
    }
  }
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/GenerateAssertions | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2.9 Special Populations - Geriatric Patient"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckDoseAdjustments"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "geriatric-test-$timestamp",
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
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/CheckDoseAdjustments | Format-JsonOutput

Write-Host "`n`n======================================================================================"
Write-Host "2.10 Special Populations - Pregnant Patient"
Write-Host "======================================================================================"
Write-Host "Executing: grpcurl -plaintext $SERVER clinical_reasoning.ClinicalReasoningService/CheckContraindications"
$timestamp = [int64](Get-Date -UFormat %s)
$jsonPayload = @"
{
  "request_id": "pregnancy-test-$timestamp",
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
}
"@
grpcurl -plaintext -d $jsonPayload $SERVER clinical_reasoning.ClinicalReasoningService/CheckContraindications | Format-JsonOutput
