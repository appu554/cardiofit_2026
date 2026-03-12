#!/bin/bash
# ============================================================================
# Setup Cron Job for Automated NCTS Updates
# ============================================================================
# Creates a cron job that runs on the 15th of each month at 2 AM
# to check for new NCTS releases and import them automatically.
# ============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CRON_SCRIPT="${SCRIPT_DIR}/ncts_cron_update.sh"
LOG_FILE="/var/log/ncts-import.log"

echo "Setting up NCTS automatic update cron job..."

# Create the cron update script if it doesn't exist
if [ ! -f "$CRON_SCRIPT" ]; then
    cat > "$CRON_SCRIPT" << 'CRONSCRIPT'
#!/bin/bash
# ============================================================================
# NCTS Automatic Update Script (Cron)
# ============================================================================
# Checks for new NCTS releases and imports them automatically.
# Run by cron on the 15th of each month.
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load configuration
if [ -f "${SCRIPT_DIR}/.env" ]; then
    source "${SCRIPT_DIR}/.env"
fi

# Default download directory
NCTS_DOWNLOAD_DIR="${NCTS_DOWNLOAD_DIR:-/path/to/ncts/downloads}"

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Checking for new NCTS releases..."

# Find the newest NCTS ZIP file
NEWEST_ZIP=$(ls -t "$NCTS_DOWNLOAD_DIR"/SnomedCT_AU_*.zip 2>/dev/null | head -1)

if [ -z "$NEWEST_ZIP" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] No NCTS ZIP files found in $NCTS_DOWNLOAD_DIR"
    exit 0
fi

# Extract version from filename
NEW_VERSION=$(basename "$NEWEST_ZIP" | grep -oP '\d{8}')

if [ -z "$NEW_VERSION" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Could not extract version from: $NEWEST_ZIP"
    exit 1
fi

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Found NCTS release: $NEW_VERSION"

# Check current version in Neo4j
CURRENT_VERSION=$(cypher-shell -a "${NEO4J_URI:-bolt://localhost:7687}" \
    -u "${NEO4J_USER:-neo4j}" -p "${NEO4J_PASSWORD}" \
    -d "${NEO4J_DATABASE:-neo4j}" --format plain \
    "MATCH (m:ImportMetadata {type: 'NCTS_REFSET'}) \
     RETURN m.version ORDER BY m.importedAt DESC LIMIT 1" 2>/dev/null | tail -1 | tr -d '"')

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Current version: ${CURRENT_VERSION:-none}"

if [ "$NEW_VERSION" == "$CURRENT_VERSION" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Version $NEW_VERSION already imported. Skipping."
    exit 0
fi

echo "[$(date '+%Y-%m-%d %H:%M:%S')] New version detected! Importing $NEW_VERSION..."

# Run import
"${SCRIPT_DIR}/ncts_rf2_import.sh" "$NEWEST_ZIP"

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Import complete."
CRONSCRIPT
    chmod +x "$CRON_SCRIPT"
    echo "Created cron update script: $CRON_SCRIPT"
fi

# Setup cron job (15th of each month at 2 AM)
CRON_CMD="0 2 15 * * $CRON_SCRIPT >> $LOG_FILE 2>&1"

# Check if cron job already exists
if crontab -l 2>/dev/null | grep -q "ncts_cron_update.sh"; then
    echo "Cron job already exists. Updating..."
    crontab -l | grep -v "ncts_cron_update.sh" | crontab -
fi

# Add new cron job
(crontab -l 2>/dev/null; echo "$CRON_CMD") | crontab -

echo ""
echo "Cron job configured successfully!"
echo "========================================"
echo "Schedule: 15th of each month at 2:00 AM"
echo "Script: $CRON_SCRIPT"
echo "Log: $LOG_FILE"
echo "========================================"
echo ""
echo "To view cron jobs: crontab -l"
echo "To remove: crontab -l | grep -v ncts_cron_update | crontab -"
