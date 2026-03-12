# Testing HL7v2 Integration

This document provides instructions for testing the HL7v2 integration in the Clinical Synthesis Hub.

## Prerequisites

1. The FHIR service is running
2. You have a valid authentication token (if required)
3. The required Python packages are installed:
   ```
   pip install requests tabulate
   ```

## Testing Tools

We've provided several tools to help you test the HL7v2 integration:

### 1. Verify HL7 Integration

The `verify_hl7_integration.py` script performs a comprehensive test of the HL7 integration:

```bash
python verify_hl7_integration.py [TOKEN]
```

This script:
1. Sends a sample HL7 ADT message to the API
2. Verifies the API response
3. Checks that the corresponding FHIR resources were created
4. Verifies that the resources contain the correct data

### 2. View HL7 Resources

The `view_hl7_resources.py` script displays all resources in the system that were created from HL7 messages:

```bash
python view_hl7_resources.py [TOKEN]
```

This script:
1. Searches for patients with HL7 tags
2. Displays a table of patients created from HL7 messages
3. For each patient, displays associated encounters

### 3. HL7 Client

The `hl7_client.py` script allows you to send custom HL7 messages to the API:

```bash
python hl7_client.py <file> [--url URL] [--token TOKEN]
```

For example:
```bash
python hl7_client.py app/tests/sample_adt_message.hl7 --token YOUR_TOKEN_HERE
```

## How to Verify HL7 Integration is Working

To verify that the HL7 integration is working correctly, follow these steps:

1. **Start the FHIR service**:
   ```bash
   cd services/fhir-service
   uvicorn app.main:app --host 0.0.0.0 --port 8004
   ```

2. **Run the verification script**:
   ```bash
   python verify_hl7_integration.py [TOKEN]
   ```

3. **Check the results**:
   - If the script completes with "✅ HL7 INTEGRATION IS WORKING CORRECTLY!", the integration is working
   - If the script fails, check the error messages for troubleshooting

4. **View HL7 resources**:
   ```bash
   python view_hl7_resources.py [TOKEN]
   ```
   - This will show you all resources created from HL7 messages

## How to Identify Resources Created from HL7 Messages

Resources created from HL7 messages have special meta tags that identify them:

```json
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
}
```

You can use these tags to:
1. Identify resources created from HL7 messages
2. Determine the type of HL7 message that created each resource
3. Filter resources by their source

## Sample HL7 Messages

We've provided several sample HL7 messages for testing:

- `app/tests/sample_adt_message.hl7`: ADT-A01 (Admission)
- `app/tests/sample_adt_a02_message.hl7`: ADT-A02 (Transfer)
- `app/tests/sample_adt_a03_message.hl7`: ADT-A03 (Discharge)

You can use these messages with the HL7 client to test different scenarios:

```bash
python hl7_client.py app/tests/sample_adt_a02_message.hl7 --token YOUR_TOKEN_HERE
```

## Troubleshooting

If you encounter issues with the HL7 integration, check the following:

1. **FHIR Service**: Make sure the FHIR service is running and accessible
2. **Authentication**: Ensure you're using a valid authentication token
3. **Dependencies**: Verify that all required packages are installed
4. **Logs**: Check the FHIR service logs for error messages
5. **Message Format**: Ensure your HL7 messages are properly formatted

If you're still having issues, try running the test script with more detailed logging:

```bash
python -m app.tests.test_hl7_integration_local
```

This will process a sample HL7 message locally and show detailed logs of the processing steps.
