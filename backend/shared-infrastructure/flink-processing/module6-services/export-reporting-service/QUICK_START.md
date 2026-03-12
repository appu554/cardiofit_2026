# Export & Reporting Service - Quick Start Guide

## 🚀 Get Started in 5 Minutes

### Prerequisites
- Java 17+
- Maven 3.9+
- PostgreSQL 15+ running on port 5433
- SendGrid API key (for email functionality)

### 1. Quick Start (Easiest)

```bash
cd export-reporting-service/
./start.sh
```

This will:
- Check dependencies
- Build the project
- Start the service on port 8050

### 2. Manual Start

```bash
# Build
mvn clean install -DskipTests

# Run
mvn spring-boot:run
```

### 3. Docker Start

```bash
# Build image
docker build -t cardiofit/export-reporting-service:1.0.0 .

# Run
docker run -d -p 8050:8050 \
  -e SENDGRID_API_KEY=your-key \
  cardiofit/export-reporting-service:1.0.0
```

### 4. Docker Compose

```bash
docker-compose up -d
```

## ✅ Verify Service is Running

```bash
# Check health
curl http://localhost:8050/api/export/health

# Expected response: "Export & Reporting Service is running"
```

## 📋 Quick API Tests

### Export Patients CSV
```bash
curl -o patients.csv "http://localhost:8050/api/export/patients/csv?departmentId=ICU&startTime=1699000000000&endTime=1699999999999"
```

### Export Alerts CSV
```bash
curl -o alerts.csv "http://localhost:8050/api/export/alerts/csv?departmentId=ALL&startTime=1699000000000&endTime=1699999999999"
```

### Export Patient FHIR
```bash
curl -o patient.json "http://localhost:8050/api/export/patients/fhir?patientId=PT-12345"
```

### Generate PDF Report
```bash
curl -o report.pdf "http://localhost:8050/api/export/reports/quality-metrics?departmentId=ALL&period=MONTHLY"
```

## 📧 Email Configuration

Edit `src/main/resources/application.yml`:

```yaml
spring:
  mail:
    password: your-sendgrid-api-key

reporting:
  email:
    daily-recipients: team@example.com
    weekly-recipients: executives@example.com
    monthly-recipients: compliance@example.com
```

Or use environment variables:

```bash
export SENDGRID_API_KEY=your-key
export DAILY_RECIPIENTS=team@example.com
```

## 🗄️ Database Configuration

Default connection (edit if needed):

```yaml
spring:
  datasource:
    url: jdbc:postgresql://localhost:5433/cardiofit_analytics
    username: cardiofit
    password: cardiofit_analytics_pass
```

## 📅 Scheduled Reports

| Report | Schedule | Recipients |
|--------|----------|------------|
| Daily Quality | 6:00 AM daily | Quality team |
| Weekly Executive | 7:00 AM Monday | Executives |
| Monthly Compliance | 8:00 AM 1st of month | Compliance |

To disable scheduling:
```yaml
reporting:
  enabled: false
```

## 🧪 Testing with Postman

1. Import `postman_collection.json` into Postman
2. Set environment variables:
   - `baseUrl`: http://localhost:8050
   - `departmentId`: ICU
   - `patientId`: PT-12345
3. Run the requests

## 📊 Monitoring

```bash
# Health check
curl http://localhost:8050/actuator/health

# Metrics
curl http://localhost:8050/actuator/metrics

# View logs
docker-compose logs -f export-reporting-service
```

## 🛠️ Troubleshooting

### Service won't start
- Check PostgreSQL is running: `nc -z localhost 5433`
- Check Java version: `java -version` (need 17+)
- Check port 8050 is free: `lsof -i :8050`

### Email not sending
- Verify `SENDGRID_API_KEY` is set
- Check SendGrid account status
- Review logs for email errors

### Database connection fails
- Verify PostgreSQL credentials
- Check database exists: `psql -h localhost -p 5433 -U cardiofit -d cardiofit_analytics`
- Review connection logs

## 📖 Full Documentation

See `README.md` for complete documentation including:
- Detailed API reference
- All configuration options
- Security considerations
- Production deployment guide

## 🎯 Service Endpoints

| Endpoint | Description |
|----------|-------------|
| `/api/export/patients/csv` | Export patients CSV |
| `/api/export/alerts/csv` | Export alerts CSV |
| `/api/export/predictions/json` | Export predictions JSON |
| `/api/export/patients/fhir` | Export FHIR bundle |
| `/api/export/reports/quality-metrics` | Generate PDF report |
| `/api/export/health` | Health check |

**Base URL**: http://localhost:8050

## 💡 Common Use Cases

### Export last 24 hours of patient data
```bash
END_TIME=$(date +%s)000
START_TIME=$((END_TIME - 86400000))
curl "http://localhost:8050/api/export/patients/csv?departmentId=ALL&startTime=$START_TIME&endTime=$END_TIME"
```

### Export all ICU alerts from last week
```bash
END_TIME=$(date +%s)000
START_TIME=$((END_TIME - 604800000))
curl "http://localhost:8050/api/export/alerts/csv?departmentId=ICU&startTime=$START_TIME&endTime=$END_TIME"
```

### Generate quality report for all departments
```bash
curl -o quality_report.pdf "http://localhost:8050/api/export/reports/quality-metrics?departmentId=ALL&period=MONTHLY"
```

## 🔧 Development Mode

For local development with auto-reload:

```bash
mvn spring-boot:run -Dspring-boot.run.profiles=dev
```

## 📦 Building for Production

```bash
# Build JAR
mvn clean package

# Run JAR
java -jar target/export-reporting-service-1.0.0.jar

# Build Docker image
docker build -t cardiofit/export-reporting-service:1.0.0 .
```

## 🆘 Need Help?

1. Check logs: `docker-compose logs -f`
2. Review README.md
3. Check IMPLEMENTATION_SUMMARY.md
4. Test with Postman collection

---

**Service Port**: 8050
**Status**: ✅ Ready for Use
**Version**: 1.0.0
