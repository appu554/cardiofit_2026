# HL7v2 Integration for Clinical Synthesis Hub

This document describes the HL7v2 message integration capabilities of the Clinical Synthesis Hub.

## Overview

The HL7v2 integration allows the system to receive and process HL7v2 messages, particularly ADT (Admission, Discharge, Transfer) messages, and convert them to FHIR resources. This enables interoperability with legacy healthcare systems that use HL7v2 messaging.

## Supported Message Types

Currently, the following HL7v2 message types are supported:

- **ADT (Admission, Discharge, Transfer)**: Messages related to patient registration, admission, discharge, and transfer.
  - A01: Admission of an inpatient
  - A02: Transfer a patient
  - A03: Discharge a patient
  - A04: Register an outpatient
  - A05: Pre-admit a patient
  - A08: Update patient information

## API Endpoints

The HL7v2 integration provides the following API endpoints:

### Process HL7 Message

```
POST /api/hl7/process
```

This endpoint accepts any supported HL7v2 message, processes it, and converts it to FHIR resources.

**Request Body:**
```json
{
  "message": "MSH|^~\\&|SENDING_APPLICATION|..."
}
```

**Response:**
```json
{
  "message": "HL7 message processed successfully",
  "resources": {
    "Patient": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "status": "created"
    },
    "Encounter": {
      "id": "123e4567-e89b-12d3-a456-426614174001",
      "status": "created"
    }
  }
}
```

### Process ADT Message

```
POST /api/hl7/adt
```

This endpoint is specifically for ADT messages. It processes the message and converts it to Patient and Encounter resources.

**Request Body:**
```json
{
  "message": "MSH|^~\\&|SENDING_APPLICATION|..."
}
```

**Response:**
```json
{
  "message": "HL7 message processed successfully",
  "resources": {
    "Patient": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "status": "created"
    },
    "Encounter": {
      "id": "123e4567-e89b-12d3-a456-426614174001",
      "status": "created"
    }
  }
}
```

### Query HL7 Resources

```
GET /api/hl7/resources
```

This endpoint allows you to query for resources that were created from HL7 messages, with optional filtering by message type and resource type.

**Query Parameters:**
- `message_type` (optional): Filter by HL7 message type (e.g., A01, A02, A03)
- `resource_type` (optional): Filter by resource type (Patient, Encounter)
- `limit` (optional): Maximum number of resources to return (default: 100)

**Example:**
```
GET /api/hl7/resources?message_type=A01&resource_type=Encounter
```

**Response:**
```json
{
  "patients": [],
  "encounters": [
    {
      "resourceType": "Encounter",
      "id": "123e4567-e89b-12d3-a456-426614174001",
      "meta": {
        "tag": [
          {
            "system": "http://clinicalsynthesishub.com/source",
            "code": "hl7v2",
            "display": "Created from HL7v2 message"
          },
          {
            "system": "http://clinicalsynthesishub.com/hl7/message_type",
            "code": "A01",
            "display": "HL7 ADT-A01"
          },
          {
            "system": "http://clinicalsynthesishub.com/hl7/event_meaning",
            "code": "in-progress",
            "display": "Admission"
          }
        ]
      },
      "status": "in-progress",
      "class": {
        "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
        "code": "inpatient",
        "display": "ambulatory"
      },
      "subject": {
        "reference": "Patient/ffdc1367-ae4f-42f3-867b-fc95c36202ab",
        "display": "SMITH, JOHN A"
      }
    }
  ],
  "total_count": 1
}
```

## HL7v2 to FHIR Mapping

The integration maps HL7v2 message segments to FHIR resources as follows:

### ADT Message Mapping

| HL7v2 Segment | FHIR Resource | Notes |
|---------------|---------------|-------|
| MSH | N/A | Message header information |
| EVN | N/A | Event information |
| PID | Patient | Patient demographics |
| PV1 | Encounter | Visit information |
| NK1 | Patient.contact | Next of kin information |

### Specific Field Mappings

#### Patient Resource

| HL7v2 Field | FHIR Path | Notes |
|-------------|-----------|-------|
| PID-3 | Patient.identifier | Patient identifiers |
| PID-5 | Patient.name | Patient name |
| PID-7 | Patient.birthDate | Date of birth |
| PID-8 | Patient.gender | Gender |
| PID-11 | Patient.address | Address |
| PID-13 | Patient.telecom | Phone number |

#### Encounter Resource

| HL7v2 Field | FHIR Path | Notes |
|-------------|-----------|-------|
| PV1-2 | Encounter.class | Patient class (inpatient, outpatient, etc.) |
| PV1-3 | Encounter.location | Assigned location |
| PV1-19 | Encounter.identifier | Visit number |
| PV1-44 | Encounter.period.start | Admit date/time |
| PV1-45 | Encounter.period.end | Discharge date/time |

## Testing

To test the HL7v2 integration, you can use the provided test script:

```bash
python -m app.tests.test_hl7_integration
```

This script processes a sample ADT message and prints the resulting FHIR resources.

## Sample HL7v2 Messages

### Sample ADT-A01 Message (Admission)

```
MSH|^~\&|SENDING_APPLICATION|SENDING_FACILITY|RECEIVING_APPLICATION|RECEIVING_FACILITY|20230615120000||ADT^A01|MSG00001|P|2.5
EVN|A01|20230615120000|||
PID|1||MRN12345^^^HOSPITAL^MR||SMITH^JOHN^A||19800101|M|||123 MAIN ST^^ANYTOWN^NY^12345^USA||(555)555-1234|||S||MRN12345001|123-45-6789
NK1|1|SMITH^JANE^|SPOUSE|(555)555-2345||EC
PV1|1|I|2000^2012^01||||004777^ATTEND^AARON^A|||SUR||||ADM|A0|
```

## Dependencies

The HL7v2 integration uses the following libraries:

- `hl7`: A simple HL7 parser
- `hl7apy`: A more comprehensive HL7 parser and generator

## Configuration

The HL7v2 integration is configured in `app/core/config.py` with the following settings:

- `HL7_VERSION`: The default HL7 version (default: "2.5")
- `HL7_PROCESSING_ID`: Processing ID (P for Production, T for Testing, D for Development)
- `HL7_RECEIVING_APPLICATION`: The name of the receiving application
- `HL7_RECEIVING_FACILITY`: The name of the receiving facility
