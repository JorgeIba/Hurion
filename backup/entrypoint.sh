#!/bin/bash
set -e

echo "=== Starting entrypoint 2==="

# 1. Restore DB
/huron-bot/backup/restore.sh

# 1.5 Enable WAL mode
echo "[entrypoint] Enabling WAL mode..."
sqlite3 "${DB_PATH:-/huron-bot/data/db.sqlite}" "PRAGMA journal_mode=WAL;"

# 2. Setup cron
echo "0 5 * * * root /app/backup/backup.sh >> /var/log/backup.log 2>&1" > /etc/cron.d/db_backup
chmod 0644 /etc/cron.d/db_backup
crontab /etc/cron.d/db_backup
cron

# 3. Start bot in background
echo "Starting bot..."
/huron-bot/bin/huron-bot &
BOT_PID=$!

shutdown_handler() {
    echo "[entrypoint] Received shutdown signal."

    echo "[entrypoint] Stopping bot..."
    kill -TERM "$BOT_PID"

    echo "[entrypoint] Waiting bot to stop (DB will close)..."
    wait "$BOT_PID"
    echo "[entrypoint] Bot fully stopped."

    echo "[entrypoint] Running final backup..."
    /huron-bot/backup/backup.sh || echo "[entrypoint] Final backup failed!"

    echo "[entrypoint] Shutdown complete."
    exit 0
}

trap shutdown_handler SIGTERM SIGINT

wait "$BOT_PID"
