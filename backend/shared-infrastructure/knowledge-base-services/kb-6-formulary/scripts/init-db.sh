#!/bin/bash
# KB-6 Formulary Database Initialization Script

set -e

# Create database if it doesn't exist
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create extensions
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE EXTENSION IF NOT EXISTS "pg_trgm";
    CREATE EXTENSION IF NOT EXISTS "btree_gin";
    
    -- Set search path
    ALTER DATABASE kb_formulary SET search_path TO public;
    
    -- Log initialization
    SELECT 'KB-6 Formulary database initialized successfully' AS status;
EOSQL

echo "KB-6 Formulary database initialization completed."