#!/bin/bash
set -e

DB="/app/data/db.sqlite"
BACKUP_DIR="/mnt/backup"
TIMESTAMP=$(date +"%Y%m%d-%H%M")

mkdir -p "$BACKUP_DIR"

# Local temporary backup to avoid SQLite writing directly to Azure Files
LOCAL_TMP="/app/data/db-$TIMESTAMP.tmp.sqlite"

BACKUP_FILE="$BACKUP_DIR/db-$TIMESTAMP.sqlite"
LATEST="$BACKUP_DIR/db-latest.sqlite"

echo "[backup] WAL checkpoint..."
sqlite3 "$DB" "PRAGMA wal_checkpoint(FULL);"

echo "[backup] Safe-sync mode..."
# Save old sync settings
OLD_SYNC=$(sqlite3 "$DB" "PRAGMA synchronous;")

# Safer sync settings during backup
sqlite3 "$DB" "PRAGMA synchronous=FULL;"
sqlite3 "$DB" "PRAGMA busy_timeout=5000;"

echo "[backup] Creating backup..."
# Create backup LOCALLY (Azure Files cannot safely be used with .backup)
sqlite3 "$DB" ".backup '$LOCAL_TMP'"

# Restore old sync settings
sqlite3 "$DB" "PRAGMA synchronous=$OLD_SYNC;"

echo "[backup] Writing latest backup (Azure-safe)..."
# Copy local backup to Azure Files
cp "$LOCAL_TMP" "$BACKUP_FILE"
cp "$LOCAL_TMP" "$LATEST"

echo "[backup] Backup done → $BACKUP_FILE"

# Remove local temporary backup
rm -f "$LOCAL_TMP"

# Cleanup old backups
find "$BACKUP_DIR" -name "db-*.sqlite" -mtime +7 -exec rm {} \;
echo "[backup] Cleanup done."
