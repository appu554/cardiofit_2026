# Notification Service Database - Quick Start Guide

## Prerequisites

- Docker with PostgreSQL container running on port 5433
- Go 1.21+ (for migration tool)

## 1. Verify PostgreSQL Connection

```bash
docker ps | grep postgres
# Should show: cardiofit-postgres-analytics on port 5433
```

## 2. Run Migrations

### Option A: Using Docker (Recommended)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/services/notification-service

# Apply migrations
docker exec -i cardiofit-postgres-analytics psql -U cardiofit -d cardiofit_analytics \
  < migrations/001_create_notifications_tables.up.sql

# Load seed data
docker exec -i cardiofit-postgres-analytics psql -U cardiofit -d cardiofit_analytics \
  < migrations/seed_test_data.sql
```

### Option B: Using Go Migration Tool

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/services/notification-service

# Build migration tool
go build -o bin/migrate cmd/migrate/main.go

# Run migrations
./bin/migrate -command=up \
  -host=localhost \
  -port=5433 \
  -database=cardiofit_analytics \
  -user=cardiofit \
  -password=cardiofit_analytics_pass \
  -sslmode=disable \
  -migrations=./migrations

# Load seed data
./bin/migrate -command=seed \
  -host=localhost \
  -port=5433 \
  -database=cardiofit_analytics \
  -user=cardiofit \
  -password=cardiofit_analytics_pass \
  -sslmode=disable \
  -seed=./migrations/seed_test_data.sql
```

## 3. Verify Installation

```bash
# Connect to database
docker exec -it cardiofit-postgres-analytics psql -U cardiofit -d cardiofit_analytics

# Run verification queries
SET search_path TO notification_service, public;

-- Check tables
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'notification_service'
ORDER BY table_name;

-- Check record counts
SELECT 'notifications' as table_name, COUNT(*) FROM notifications
UNION ALL
SELECT 'user_preferences', COUNT(*) FROM user_preferences
UNION ALL
SELECT 'escalation_log', COUNT(*) FROM escalation_log
UNION ALL
SELECT 'alert_fatigue_history', COUNT(*) FROM alert_fatigue_history
UNION ALL
SELECT 'delivery_metrics', COUNT(*) FROM delivery_metrics
UNION ALL
SELECT 'notification_templates', COUNT(*) FROM notification_templates;
```

Expected output:
```
table_name               | count
-------------------------+-------
notifications            |     7
user_preferences         |     5
escalation_log           |     4
alert_fatigue_history    |    30
delivery_metrics         |    19
notification_templates   |     4
```

## 4. Test Views

```sql
-- Pending notifications
SELECT * FROM v_pending_notifications;

-- Delivery success rates
SELECT * FROM v_delivery_success_rates WHERE date >= CURRENT_DATE - INTERVAL '1 day';

-- Escalation effectiveness
SELECT * FROM v_escalation_effectiveness;
```

## 5. Connection String for Go Service

```go
import (
    "database/sql"
    _ "github.com/lib/pq"
)

connStr := "host=localhost port=5433 user=cardiofit password=cardiofit_analytics_pass dbname=cardiofit_analytics sslmode=disable"
db, err := sql.Open("postgres", connStr)
if err != nil {
    log.Fatal(err)
}

// Set search path
_, err = db.Exec("SET search_path TO notification_service, public")
if err != nil {
    log.Fatal(err)
}
```

## 6. Sample Queries

### Insert Notification

```sql
INSERT INTO notifications (alert_id, user_id, channel, priority, message, status, metadata)
VALUES (
    'alert_001',
    'user_attending_001',
    'SMS',
    1,
    'CRITICAL: Patient PAT-001 Sepsis Alert',
    'PENDING',
    '{"severity": "CRITICAL", "patient_id": "PAT-001"}'::jsonb
);
```

### Update Notification Status

```sql
UPDATE notifications
SET status = 'SENT',
    sent_at = NOW(),
    external_id = 'SM123456789'
WHERE id = 'notification-id-here';
```

### Query Pending Notifications

```sql
SELECT id, alert_id, user_id, channel, priority, message
FROM notifications
WHERE status IN ('PENDING', 'SENDING')
ORDER BY priority, created_at;
```

### Check User Preferences

```sql
SELECT user_id, channel_preferences, severity_channels, max_alerts_per_hour
FROM user_preferences
WHERE user_id = 'user_attending_001';
```

### Record Escalation

```sql
INSERT INTO escalation_log (alert_id, escalation_level, escalated_to_user, escalated_to_role)
VALUES ('alert_001', 1, 'user_attending_001', 'Attending Physician');
```

## 7. Rollback (if needed)

```bash
# Using Docker
docker exec -i cardiofit-postgres-analytics psql -U cardiofit -d cardiofit_analytics \
  < migrations/001_create_notifications_tables.down.sql

# Using Go tool
./bin/migrate -command=down \
  -host=localhost \
  -port=5433 \
  -database=cardiofit_analytics \
  -user=cardiofit \
  -password=cardiofit_analytics_pass \
  -sslmode=disable \
  -migrations=./migrations
```

## 8. Troubleshooting

### Cannot connect to database

```bash
# Check container is running
docker ps | grep postgres

# Check container logs
docker logs cardiofit-postgres-analytics

# Test connection
docker exec cardiofit-postgres-analytics psql -U cardiofit -d cardiofit_analytics -c "SELECT version();"
```

### Schema already exists

```bash
# Drop and recreate (WARNING: deletes all data)
docker exec -i cardiofit-postgres-analytics psql -U cardiofit -d cardiofit_analytics <<EOF
DROP SCHEMA IF EXISTS notification_service CASCADE;
EOF

# Then re-run migrations
```

### Permission denied errors

Verify you're using the correct database user (cardiofit) and the schema is owned by that user.

## Next Steps

1. ✅ Database schema created
2. ✅ Seed data loaded
3. 🚧 Implement Go notification service
4. 🚧 Integrate with Kafka consumers
5. 🚧 Add delivery channel integrations (Twilio, SendGrid, Firebase)

## Files Location

All files are in:
```
/Users/apoorvabk/Downloads/cardiofit/backend/services/notification-service/
```

## Documentation

- **Full Documentation**: README.md (database migration guide)
- **Migration Status**: MIGRATION_STATUS.md (detailed completion report)
- **Database Schema**: migrations/001_create_notifications_tables.up.sql
- **Specification**: /Users/apoorvabk/Downloads/cardiofit/claudedocs/NOTIFICATION_SERVICE_SPECIFICATION.md
