# Database Setup Guide for Phase 4

## Issue: Supabase Connection Failed

The setup script encountered a database connection error:
```
FATAL: password authentication failed for user "postgres"
```

## Solution Options

### Option 1: Update Supabase Credentials (Recommended)

1. **Check your Supabase project settings**:
   - Go to your Supabase dashboard
   - Navigate to Settings → Database
   - Copy the correct connection string

2. **Update your `.env` file**:
   ```env
   DATABASE_URL=postgresql://postgres:[YOUR_PASSWORD]@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres
   SUPABASE_URL=https://auugxeqzgrnknklgwqrh.supabase.co
   SUPABASE_KEY=[YOUR_ANON_KEY]
   ```

3. **Verify the password**:
   - The password should be the one you set when creating the Supabase project
   - If you forgot it, you can reset it in the Supabase dashboard

### Option 2: Run Migration Manually Later

If you want to proceed without database setup for now:

1. **Phase 4 services will work** without the database migration
2. **Run the migration later** when database is accessible:
   ```bash
   python run_migration.py
   ```

### Option 3: Use Local PostgreSQL

If you prefer local development:

1. **Install PostgreSQL locally**
2. **Create a database**:
   ```sql
   CREATE DATABASE workflow_engine;
   ```
3. **Update `.env`**:
   ```env
   DATABASE_URL=postgresql://username:password@localhost:5432/workflow_engine
   ```

## Phase 4 Migration Tables

The migration creates these tables:
- `service_task_logs` - Service task execution logs
- `event_store` - Central event store for inter-service communication
- `event_processing_logs` - Event processing activity logs
- `fhir_resource_monitor_state` - FHIR resource monitoring state
- `service_integration_config` - Service integration configuration
- `workflow_event_triggers` - Event-triggered workflow configuration

## Manual Migration

If you need to run the migration manually:

```sql
-- Copy the contents of migrations/003_phase4_integration_tables.sql
-- and run it in your Supabase SQL editor or psql
```

## Testing Without Database

You can test Phase 4 functionality without database:

```bash
python test_phase4_simple.py
```

This will test:
- Service imports
- Configuration
- Basic functionality
- Integration readiness

## Next Steps

1. **Fix database connection** (recommended)
2. **Run setup again**: `python setup_phase4.py`
3. **Start the service**: `python run_service.py`
4. **Test federation**: Test with other services in the federation

## Support

If you continue having database issues:
1. Check Supabase project status
2. Verify network connectivity
3. Confirm credentials are correct
4. Check if IP is whitelisted (if applicable)
