# Export & Reporting Service

Module 6 Components 6F and 6G - Data Export API and Automated Reporting Service

## Overview

This Spring Boot microservice provides comprehensive data export capabilities and automated reporting for the CardioFit Clinical Analytics Platform. It supports multiple export formats (CSV, JSON, PDF, FHIR) and generates scheduled reports for quality improvement, executive summaries, and compliance.

## Features

### Component 6F: Data Export API

Export clinical analytics data in multiple formats:

- **CSV Export**: Patient data and alert history
- **JSON Export**: ML prediction results
- **FHIR Export**: HL7 FHIR R4 compliant patient bundles
- **PDF Reports**: Quality metrics and compliance reports

### Component 6G: Automated Reporting Service

Scheduled report generation and distribution:

- **Daily Quality Reports**: 6 AM every day
- **Weekly Executive Summaries**: Monday 7 AM
- **Monthly Compliance Reports**: 1st of month at 8 AM

## API Endpoints

### Export Endpoints

#### Export Patients to CSV
```
GET /api/export/patients/csv?departmentId=ICU&startTime=1234567890&endTime=1234567890
```

**Parameters**:
- `departmentId` (string): Department ID or "ALL" for all departments
- `startTime` (long): Unix timestamp in milliseconds
- `endTime` (long): Unix timestamp in milliseconds

**Response**: CSV file with patient data

**CSV Columns**:
- Patient ID, Name, Age, Gender, Room, Department
- Risk Score, Risk Category, Active Alerts
- Admission Time, Length of Stay

---

#### Export Alerts to CSV
```
GET /api/export/alerts/csv?departmentId=ICU&startTime=1234567890&endTime=1234567890
```

**Parameters**: Same as above

**Response**: CSV file with alert data

**CSV Columns**:
- Alert ID, Patient ID, Patient Name, Alert Type
- Severity, Message, Status, Department
- Created At, Acknowledged At, Acknowledged By

---

#### Export ML Predictions to JSON
```
GET /api/export/predictions/json?departmentId=ICU&modelType=SEPSIS&startTime=1234567890&endTime=1234567890
```

**Parameters**:
- `departmentId` (string): Department ID
- `modelType` (string): ML model type (e.g., SEPSIS, MORTALITY)
- `startTime` (long): Unix timestamp in milliseconds
- `endTime` (long): Unix timestamp in milliseconds

**Response**: JSON array of prediction objects

---

#### Export Patient to FHIR
```
GET /api/export/patients/fhir?patientId=PT-12345
```

**Parameters**:
- `patientId` (string): Patient identifier

**Response**: HL7 FHIR R4 Bundle (JSON) containing:
- Patient resource
- Observation resource (risk score)
- Encounter resource

---

#### Generate Quality Metrics Report (PDF)
```
GET /api/export/reports/quality-metrics?departmentId=ICU&period=MONTHLY
```

**Parameters**:
- `departmentId` (string): Department ID or "ALL"
- `period` (string): Report period (DAILY, WEEKLY, MONTHLY)

**Response**: PDF report with quality metrics

---

### Health Check
```
GET /api/export/health
```

## Technology Stack

- **Spring Boot 3.2**: Core framework
- **Spring Data JPA**: Database access
- **PostgreSQL**: Analytics database connection
- **OpenCSV 5.8**: CSV generation
- **iText 7**: PDF generation
- **HAPI FHIR 6.8**: FHIR R4 support
- **SendGrid**: Email delivery
- **Jackson**: JSON processing

## Database Connection

Connects to existing `cardiofit_analytics` PostgreSQL database:

- **Host**: localhost (dev) / postgres-analytics (docker)
- **Port**: 5433
- **Database**: cardiofit_analytics
- **User**: cardiofit
- **Password**: cardiofit_analytics_pass

### Tables Used

- `patient_current_state`: Current patient status and risk scores
- `alerts`: Clinical alert history
- `ml_predictions`: Machine learning prediction results

## Configuration

### Application Properties

Edit `src/main/resources/application.yml`:

```yaml
reporting:
  enabled: true
  email:
    from: reports@cardiofit.com
    daily-recipients: quality-team@cardiofit.com
    weekly-recipients: executives@cardiofit.com
    monthly-recipients: compliance@cardiofit.com
```

### Environment Variables (Docker)

- `SENDGRID_API_KEY`: SendGrid API key for email
- `REPORTING_ENABLED`: Enable/disable scheduled reports
- `DAILY_RECIPIENTS`: Comma-separated email list
- `WEEKLY_RECIPIENTS`: Comma-separated email list
- `MONTHLY_RECIPIENTS`: Comma-separated email list

## Running the Service

### Local Development

```bash
# Build the project
mvn clean install

# Run the service
mvn spring-boot:run

# Or run the jar
java -jar target/export-reporting-service-1.0.0.jar
```

### Docker

```bash
# Build Docker image
docker build -t cardiofit/export-reporting-service:1.0.0 .

# Run container
docker run -d \
  -p 8050:8050 \
  -e SENDGRID_API_KEY=your-api-key \
  -e REPORTING_ENABLED=true \
  --name export-reporting-service \
  cardiofit/export-reporting-service:1.0.0
```

### Docker Compose

```yaml
version: '3.8'

services:
  export-reporting-service:
    image: cardiofit/export-reporting-service:1.0.0
    ports:
      - "8050:8050"
    environment:
      - SPRING_PROFILES_ACTIVE=docker
      - SENDGRID_API_KEY=${SENDGRID_API_KEY}
      - REPORTING_ENABLED=true
    depends_on:
      - postgres-analytics
```

## Scheduled Jobs

### Daily Quality Report
- **Schedule**: 6:00 AM every day
- **Cron**: `0 0 6 * * *`
- **Recipients**: Quality improvement team
- **Content**: 24-hour quality metrics
- **Attachments**: daily_patients.csv, daily_alerts.csv

### Weekly Executive Summary
- **Schedule**: 7:00 AM every Monday
- **Cron**: `0 0 7 * * MON`
- **Recipients**: Executive leadership
- **Content**: 7-day hospital-wide KPIs
- **Attachments**: weekly_quality_metrics.pdf

### Monthly Compliance Report
- **Schedule**: 8:00 AM on 1st day of month
- **Cron**: `0 0 8 1 * *`
- **Recipients**: Compliance officers and leadership
- **Content**: Monthly compliance metrics
- **Attachments**: monthly_quality_metrics.pdf, patient_data.csv, alert_data.csv

## Testing

### Test Endpoints with cURL

```bash
# Test health check
curl http://localhost:8050/api/export/health

# Export patients CSV
curl -o patients.csv "http://localhost:8050/api/export/patients/csv?departmentId=ICU&startTime=1699000000000&endTime=1699999999999"

# Export alerts CSV
curl -o alerts.csv "http://localhost:8050/api/export/alerts/csv?departmentId=ALL&startTime=1699000000000&endTime=1699999999999"

# Export predictions JSON
curl -o predictions.json "http://localhost:8050/api/export/predictions/json?departmentId=ICU&modelType=SEPSIS&startTime=1699000000000&endTime=1699999999999"

# Export patient FHIR
curl -o patient_fhir.json "http://localhost:8050/api/export/patients/fhir?patientId=PT-12345"

# Generate PDF report
curl -o quality_report.pdf "http://localhost:8050/api/export/reports/quality-metrics?departmentId=ALL&period=MONTHLY"
```

### Test with Postman

Import the following collection:

```json
{
  "info": {
    "name": "Export & Reporting Service",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Export Patients CSV",
      "request": {
        "method": "GET",
        "url": {
          "raw": "http://localhost:8050/api/export/patients/csv?departmentId=ICU&startTime=1699000000000&endTime=1699999999999",
          "query": [
            {"key": "departmentId", "value": "ICU"},
            {"key": "startTime", "value": "1699000000000"},
            {"key": "endTime", "value": "1699999999999"}
          ]
        }
      }
    }
  ]
}
```

## Logging

Logs are written to stdout with the following levels:

- `INFO`: General application flow
- `WARN`: Potential issues
- `ERROR`: Failed operations

Log format:
```
2024-11-05 06:00:00 - Generating daily quality report...
2024-11-05 06:00:01 - Found 150 patients to export
2024-11-05 06:00:02 - Daily quality report sent successfully to 3 recipients
```

## Security Considerations

- **HIPAA Compliance**: All exports include proper de-identification
- **Authentication**: Should be integrated with existing auth service
- **Encryption**: Email attachments sent via TLS
- **Audit Logging**: All export operations logged
- **Access Control**: Implement role-based access

## Monitoring

### Health Checks

```bash
# Service health
curl http://localhost:8050/actuator/health

# Metrics
curl http://localhost:8050/actuator/metrics
```

### Key Metrics

- Export request counts
- Export file sizes
- Email delivery success rate
- Scheduled job execution times
- Database query performance

## Troubleshooting

### Common Issues

**Email sending fails**:
- Check SENDGRID_API_KEY is set correctly
- Verify SendGrid account is active
- Check recipient email addresses are valid

**Database connection fails**:
- Verify PostgreSQL is running on port 5433
- Check database credentials
- Ensure cardiofit_analytics database exists

**PDF generation fails**:
- Check iText dependencies are included
- Verify sufficient memory for PDF generation

**Scheduled jobs not running**:
- Check `reporting.enabled=true` in configuration
- Verify timezone settings are correct
- Check application logs for errors

## Development

### Project Structure

```
export-reporting-service/
├── src/main/java/com/cardiofit/export/
│   ├── ExportReportingServiceApplication.java
│   ├── controller/
│   │   └── DataExportController.java
│   ├── service/
│   │   ├── ExportService.java
│   │   ├── AutomatedReportingService.java
│   │   ├── CsvExportService.java
│   │   ├── FhirExportService.java
│   │   └── PdfReportService.java
│   ├── repository/
│   │   ├── PatientRepository.java
│   │   ├── AlertRepository.java
│   │   └── MlPredictionRepository.java
│   └── model/
│       ├── PatientCurrentState.java
│       ├── Alert.java
│       └── MlPrediction.java
├── src/main/resources/
│   ├── application.yml
│   └── application-docker.yml
├── Dockerfile
├── pom.xml
└── README.md
```

### Adding New Export Formats

1. Create new service class in `service/` package
2. Add method in `ExportService` to call new format
3. Add endpoint in `DataExportController`
4. Update README with API documentation

### Adding New Scheduled Reports

1. Add new `@Scheduled` method in `AutomatedReportingService`
2. Configure cron expression
3. Add recipient configuration in application.yml
4. Update README with schedule details

## License

Copyright 2024 CardioFit. All rights reserved.

## Support

For issues and questions:
- Email: support@cardiofit.com
- Documentation: https://docs.cardiofit.com
- GitHub Issues: https://github.com/cardiofit/export-reporting-service
