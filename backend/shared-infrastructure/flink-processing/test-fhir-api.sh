#!/bin/bash

# Test FHIR API connection and patient data fetching
# Usage: ./test-fhir-api.sh [patient_id]

PATIENT_ID="${1:-905a60cb-8241-418f-b29b-5b020e851392}"
PROJECT_ID="cardiofit-905a8"
LOCATION="asia-south1"
DATASET="clinical-synthesis-hub"
FHIR_STORE="fhir-store"

BASE_URL="https://healthcare.googleapis.com/v1/projects/${PROJECT_ID}/locations/${LOCATION}/datasets/${DATASET}/fhirStores/${FHIR_STORE}/fhir"

echo "===================================="
echo "FHIR API Connection Test"
echo "===================================="
echo "Base URL: $BASE_URL"
echo "Patient ID: $PATIENT_ID"
echo ""

# Get access token
echo "🔑 Getting access token..."
ACCESS_TOKEN=$(gcloud auth print-access-token 2>/dev/null)
if [ -z "$ACCESS_TOKEN" ]; then
    echo "❌ Failed to get access token. Run: gcloud auth login"
    exit 1
fi
echo "✅ Access token obtained"
echo ""

# Test 1: Fetch Patient resource
echo "📋 Test 1: Fetching Patient/${PATIENT_ID}..."
PATIENT_URL="${BASE_URL}/Patient/${PATIENT_ID}"
echo "URL: $PATIENT_URL"
echo ""

PATIENT_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/fhir+json" \
  "${PATIENT_URL}")

HTTP_STATUS=$(echo "$PATIENT_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
PATIENT_DATA=$(echo "$PATIENT_RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$HTTP_STATUS" = "200" ]; then
    echo "✅ Patient found (HTTP 200)"
    echo "$PATIENT_DATA" | jq '{
      resourceType,
      id,
      name: .name[0],
      gender,
      birthDate,
      identifier: .identifier[0]
    }' 2>/dev/null || echo "$PATIENT_DATA"
elif [ "$HTTP_STATUS" = "404" ]; then
    echo "❌ Patient not found (HTTP 404)"
    echo "Response: $PATIENT_DATA"
else
    echo "⚠️  Unexpected status: HTTP $HTTP_STATUS"
    echo "Response: $PATIENT_DATA"
fi
echo ""

# Test 2: Fetch Medications for patient
echo "📋 Test 2: Fetching MedicationStatement for patient..."
MEDS_URL="${BASE_URL}/MedicationStatement?subject=Patient/${PATIENT_ID}"
echo "URL: $MEDS_URL"
echo ""

MEDS_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/fhir+json" \
  "${MEDS_URL}")

MEDS_HTTP_STATUS=$(echo "$MEDS_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
MEDS_DATA=$(echo "$MEDS_RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$MEDS_HTTP_STATUS" = "200" ]; then
    MEDS_COUNT=$(echo "$MEDS_DATA" | jq '.total // 0' 2>/dev/null)
    echo "✅ Medications found: $MEDS_COUNT"
    if [ "$MEDS_COUNT" -gt 0 ]; then
        echo "$MEDS_DATA" | jq '.entry[0:3] | .[] | {
          medication: .resource.medicationCodeableConcept.text,
          status: .resource.status,
          dosage: .resource.dosage[0].text
        }' 2>/dev/null || echo "Could not parse medications"
    fi
else
    echo "⚠️  Medications query status: HTTP $MEDS_HTTP_STATUS"
fi
echo ""

# Test 3: Fetch Conditions for patient
echo "📋 Test 3: Fetching Condition for patient..."
CONDITIONS_URL="${BASE_URL}/Condition?subject=Patient/${PATIENT_ID}"
echo "URL: $CONDITIONS_URL"
echo ""

CONDITIONS_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/fhir+json" \
  "${CONDITIONS_URL}")

COND_HTTP_STATUS=$(echo "$CONDITIONS_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
COND_DATA=$(echo "$CONDITIONS_RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$COND_HTTP_STATUS" = "200" ]; then
    COND_COUNT=$(echo "$COND_DATA" | jq '.total // 0' 2>/dev/null)
    echo "✅ Conditions found: $COND_COUNT"
    if [ "$COND_COUNT" -gt 0 ]; then
        echo "$COND_DATA" | jq '.entry[0:3] | .[] | {
          condition: .resource.code.text,
          clinicalStatus: .resource.clinicalStatus.coding[0].code,
          verificationStatus: .resource.verificationStatus.coding[0].code
        }' 2>/dev/null || echo "Could not parse conditions"
    fi
else
    echo "⚠️  Conditions query status: HTTP $COND_HTTP_STATUS"
fi
echo ""

# Summary
echo "===================================="
echo "Summary"
echo "===================================="
echo "Patient Status: HTTP $HTTP_STATUS"
echo "Medications: $MEDS_COUNT"
echo "Conditions: $COND_COUNT"
echo "===================================="
