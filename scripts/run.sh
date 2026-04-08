#!/bin/bash
set -e

DB_PATH="/huron-bot/data/db.sqlite"

# 1. Restore the database from Litestream (Blob Storage)
if [ ! -f "$DB_PATH" ]; then
    echo "[run.sh] Database missing locally. Checking for Litestream replicas..."
    # -if-replica-exists: prevents errors if this is a brand new bot with no backups yet.
    litestream restore -if-replica-exists -config /etc/litestream.yml "$DB_PATH"
else
    echo "[run.sh] Database already exists locally. Skipping restore."
fi

# 2. Final check and initialization
if [ ! -f "$DB_PATH" ]; then
    echo "[run.sh] No local database or Litestream replica found. Bot will initialize a fresh database on startup."
fi

# 3. Ensure WAL mode is enabled (required for Litestream)
# Ensure the data directory exists.
mkdir -p "$(dirname "$DB_PATH")"
# If the file doesn't exist yet, sqlite3 will create it when we set WAL mode.
echo "[run.sh] Ensuring WAL mode..."
sqlite3 "$DB_PATH" "PRAGMA journal_mode=WAL;"

# 4. Run Litestream to replicate changes and supervise the bot process.
echo "[run.sh] Starting Litestream replication..."
exec litestream replicate -config /etc/litestream.yml -exec "/huron-bot/bin/huron-bot"
