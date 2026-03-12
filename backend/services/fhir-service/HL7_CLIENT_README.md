# HL7 Client for Clinical Synthesis Hub

This client allows you to send HL7v2 messages to the Clinical Synthesis Hub API.

## Usage

```bash
python hl7_client.py <file> [--url URL] [--token TOKEN]
```

### Arguments

- `file`: Path to the HL7 message file
- `--url`: URL of the HL7 API endpoint (default: http://localhost:8004/api/hl7/process)
- `--token`: Authentication token (optional)

### Example

```bash
python hl7_client.py app/tests/sample_adt_message.hl7 --token YOUR_TOKEN_HERE
```

## Sample HL7 Messages

### ADT-A01 (Admission)

```
MSH|^~\&|SENDING_APPLICATION|SENDING_FACILITY|RECEIVING_APPLICATION|RECEIVING_FACILITY|20230615120000||ADT^A01|MSG00001|P|2.3
EVN|A01|20230615120000|||
PID|1||MRN12345^^^HOSPITAL^MR||SMITH^JOHN^A||19800101|M|||123 MAIN ST^^ANYTOWN^NY^12345^USA||(555)555-1234|||S||MRN12345001|123-45-6789
NK1|1|SMITH^JANE^|SPOUSE|(555)555-2345||EC
PV1|1|I|2000^2012^01||||004777^ATTEND^AARON^A|||SUR||||ADM|A0|
```

### ADT-A02 (Transfer)

```
MSH|^~\&|SENDING_APPLICATION|SENDING_FACILITY|RECEIVING_APPLICATION|RECEIVING_FACILITY|20230615120000||ADT^A02|MSG00002|P|2.3
EVN|A02|20230615120000|||
PID|1||MRN12345^^^HOSPITAL^MR||SMITH^JOHN^A||19800101|M|||123 MAIN ST^^ANYTOWN^NY^12345^USA||(555)555-1234|||S||MRN12345001|123-45-6789
PV1|1|I|3000^3012^01||||004777^ATTEND^AARON^A|||SUR||||ADM|A0|
```

### ADT-A03 (Discharge)

```
MSH|^~\&|SENDING_APPLICATION|SENDING_FACILITY|RECEIVING_APPLICATION|RECEIVING_FACILITY|20230615120000||ADT^A03|MSG00003|P|2.3
EVN|A03|20230615120000|||
PID|1||MRN12345^^^HOSPITAL^MR||SMITH^JOHN^A||19800101|M|||123 MAIN ST^^ANYTOWN^NY^12345^USA||(555)555-1234|||S||MRN12345001|123-45-6789
PV1|1|I|3000^3012^01||||004777^ATTEND^AARON^A|||SUR||||ADM|A0|
```

## API Response

The API will respond with a JSON object containing the status of the message processing and the IDs of the created or updated FHIR resources:

```json
{
  "message": "HL7 message processed successfully",
  "resources": {
    "Patient": {
      "id": "ffdc1367-ae4f-42f3-867b-fc95c36202ab",
      "status": "created"
    },
    "Encounter": {
      "id": "a35a0303-a1b7-458a-bb54-1df7996f81b3",
      "status": "created"
    }
  }
}
```

## Creating Custom HL7 Messages

You can create custom HL7 messages by following the HL7v2 message format. The most important segments for ADT messages are:

- **MSH**: Message Header
- **EVN**: Event Type
- **PID**: Patient Identification
- **PV1**: Patient Visit

For more information on HL7v2 message formats, refer to the [HL7 International website](https://www.hl7.org/).
