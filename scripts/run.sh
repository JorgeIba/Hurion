#!/bin/sh
set -e

# Configuration selection
if [ -n "$AZURE_STORAGE_ACCOUNT" ] && [ -n "$AZURE_STORAGE_CONTAINER" ] && { [ -n "$LITESTREAM_AZURE_ACCOUNT_KEY" ] || [ -n "$AZURE_STORAGE_KEY" ]; }; then
    echo "Using Azure Blob Storage configuration for Litestream..."
    CONFIG="/etc/litestream.yml"
else
    echo "Using local disk storage configuration for Litestream..."
    CONFIG="/etc/litestream.local.yml"
fi

# Set default database path if not specified
DB_PATH="${DB_PATH:-/hurion/data/hurion.db}"
mkdir -p "$(dirname "$DB_PATH")"

# Attempt to restore database if it doesn't exist locally
echo "Checking for database restore..."
litestream restore -config "$CONFIG" -if-db-not-exists -if-replica-exists "$DB_PATH" || true

# Enforce WAL mode once database exists or has been restored
echo "Enforcing WAL journal mode on database..."
sqlite3 "$DB_PATH" "PRAGMA journal_mode=WAL;" || true

echo "Starting Litestream replication and Hurion bot..."
# Supervise compiled binary directly, forwarding container CMD flags ($*)
exec litestream replicate -config "$CONFIG" -exec "/hurion/bin/hurion $*"
