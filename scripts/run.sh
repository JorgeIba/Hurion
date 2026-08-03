#!/bin/bash
set -e

# Use the environment variable or fall back to the default path
export DB_PATH="${DB_PATH:-/hurion/data/db.sqlite}"

# 1. Ensure the data directory exists before any Litestream or SQLite operations
mkdir -p "$(dirname "$DB_PATH")"
mkdir -p "/hurion/local-backups"

# 2. Pick the right Litestream configuration
# We require all three Azure variables to use the production config.
if [ -n "$AZURE_STORAGE_ACCOUNT" ] && [ -n "$AZURE_STORAGE_CONTAINER" ] && [ -n "$LITESTREAM_AZURE_ACCOUNT_KEY" ]; then
    CONFIG="/etc/litestream.yml"
    echo "[run.sh] Azure credentials found. Using production config: $CONFIG"
else
    CONFIG="/etc/litestream.local.yml"
    echo "[run.sh] Azure credentials missing or incomplete. Using local dev config: $CONFIG"
fi

# 3. Restore the database if it doesn't exist
# -if-db-not-exists: prevents overwriting an existing local database.
# -if-replica-exists: prevents errors if this is a brand new setup.
echo "[run.sh] Checking for Litestream replicas..."
litestream restore -if-db-not-exists -if-replica-exists -config "$CONFIG" "$DB_PATH"

# 4. Ensure WAL mode is enabled (required for Litestream)
echo "[run.sh] Ensuring WAL mode..."
sqlite3 "$DB_PATH" "PRAGMA journal_mode=WAL;"

# 5. Run Litestream to replicate changes and supervise the bot process.
echo "[run.sh] Starting Litestream replication..."
exec litestream replicate -config "$CONFIG" -exec "make -C /hurion run ARGS=\"$*\""
