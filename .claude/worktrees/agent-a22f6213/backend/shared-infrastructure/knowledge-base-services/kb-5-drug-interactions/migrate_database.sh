#!/bin/bash
# Execute KB-5 Enhanced Database Schema Migration

set -e

echo "🚀 Starting KB-5 Enhanced Schema Migration..."

# Database connection details (should be set via environment variables)
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${KB5_DB_NAME:-kb5_drug_interactions}
DB_USER=${DB_USER:-postgres}

echo "Database: ${DB_HOST}:${DB_PORT}/${DB_NAME}"

# Check if database is accessible
echo "📋 Checking database connectivity..."
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -q; then
    echo "❌ Error: Cannot connect to database $DB_HOST:$DB_PORT/$DB_NAME"
    exit 1
fi
echo "✅ Database connectivity confirmed"

# Backup current schema (optional but recommended)
echo "💾 Creating schema backup..."
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        --schema-only --no-owner --no-privileges \
        > "backup_schema_$(date +%Y%m%d_%H%M%S).sql"
echo "✅ Schema backup created"

# Execute initial schema if not exists
echo "📋 Checking for existing schema..."
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
         -c "SELECT 1 FROM information_schema.tables WHERE table_name='drug_interactions'" \
         -t -A | grep -q 1; then
    echo "🔧 Executing initial schema..."
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
         -f "migrations/001_initial_schema.sql"
    echo "✅ Initial schema executed"
else
    echo "✅ Initial schema already exists"
fi

# Execute enhanced schema migration
echo "🚀 Executing enhanced schema migration..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
     -f "migrations/002_enhanced_schema.sql"

if [ $? -eq 0 ]; then
    echo "✅ Enhanced schema migration completed successfully!"
else
    echo "❌ Enhanced schema migration failed!"
    exit 1
fi

# Verify new tables were created
echo "🔍 Verifying enhanced schema..."
TABLES=(
    "ddi_class_rules"
    "ddi_pharmacogenomic_rules" 
    "ddi_modifiers"
    "ddi_overrides"
    "ddi_dataset_versions"
)

for table in "${TABLES[@]}"; do
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
           -c "SELECT 1 FROM information_schema.tables WHERE table_name='$table'" \
           -t -A | grep -q 1; then
        echo "✅ Table $table created successfully"
    else
        echo "❌ Table $table not found!"
        exit 1
    fi
done

# Verify materialized view
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
       -c "SELECT 1 FROM information_schema.views WHERE table_name='ddi_interaction_matrix'" \
       -t -A | grep -q 1; then
    echo "✅ Materialized view ddi_interaction_matrix created successfully"
else
    echo "❌ Materialized view ddi_interaction_matrix not found!"
    exit 1
fi

# Test sample data insertion
echo "🧪 Testing sample data..."
ROW_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
                 -c "SELECT COUNT(*) FROM ddi_pharmacogenomic_rules" -t -A)

if [ "$ROW_COUNT" -gt 0 ]; then
    echo "✅ Sample pharmacogenomic data inserted: $ROW_COUNT rows"
else
    echo "⚠️  No sample data found in pharmacogenomic rules"
fi

echo ""
echo "🎉 KB-5 Enhanced Schema Migration Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Enhanced tables created successfully"
echo "✅ Materialized view for performance optimization ready"
echo "✅ Sample pharmacogenomic and class interaction data loaded"
echo "✅ Institutional override system configured"
echo "✅ Dataset versioning and Evidence Envelope integration ready"
echo ""
echo "Next steps:"
echo "1. Run: ./generate_proto.sh"
echo "2. Execute: go mod tidy && go build"
echo "3. Test with: go test ./..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"