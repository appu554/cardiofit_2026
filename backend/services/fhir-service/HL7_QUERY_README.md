# Querying HL7 Resources

This document explains how to use the new `/api/hl7/resources` endpoint to query resources created from HL7 messages.

## Overview

The `/api/hl7/resources` endpoint allows you to:
- Retrieve all resources created from HL7 messages
- Filter resources by HL7 message type (e.g., A01 for admissions)
- Filter resources by resource type (Patient, Encounter)

This makes it easy to find and analyze data that originated from HL7 messages, particularly ADT (Admission, Discharge, Transfer) messages.

## Using the Endpoint

### Basic Query

To retrieve all resources created from HL7 messages:

```
GET /api/hl7/resources
```

### Filtering by Message Type

To retrieve resources created from a specific HL7 message type:

```
GET /api/hl7/resources?message_type=A01
```

This will return all resources created from ADT-A01 (Admission) messages.

Other common message types:
- `A01`: Admission
- `A02`: Transfer
- `A03`: Discharge
- `A04`: Registration
- `A05`: Pre-admission

### Filtering by Resource Type

To retrieve specific types of resources:

```
GET /api/hl7/resources?resource_type=Encounter
```

This will return only Encounter resources created from HL7 messages.

### Combined Filtering

You can combine filters to narrow down your results:

```
GET /api/hl7/resources?message_type=A01&resource_type=Encounter
```

This will return only Encounter resources created from ADT-A01 (Admission) messages.

## Response Format

The endpoint returns a JSON object with the following structure:

```json
{
  "patients": [
    {
      "resourceType": "Patient",
      "id": "...",
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
          }
        ]
      },
      // Other patient fields...
    }
  ],
  "encounters": [
    {
      "resourceType": "Encounter",
      "id": "...",
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
      // Other encounter fields...
    }
  ],
  "total_count": 2
}
```

## Using the Query Script

We've provided a script to make it easy to query HL7 resources:

```bash
python query_hl7_resources.py [--message-type TYPE] [--resource-type TYPE] [--token TOKEN]
```

Examples:

```bash
# Query all HL7 resources
python query_hl7_resources.py

# Query ADT-A01 (Admission) resources
python query_hl7_resources.py --message-type A01

# Query ADT-A01 Encounters
python query_hl7_resources.py --message-type A01 --resource-type Encounter
```

## Using Postman

We've also provided a Postman collection (`HL7_Postman_Collection.json`) that includes requests for:
- Sending HL7 messages
- Querying HL7 resources

To use the collection:
1. Import the collection into Postman
2. Set the `base_url` variable to your server URL (default: `http://localhost:8004`)
3. Set the `token` variable to your authentication token
4. Use the requests to test the HL7 integration
