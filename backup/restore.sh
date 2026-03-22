#!/bin/bash
set -e

DB="${DB_PATH:-/huron-bot/data/db.sqlite}"
BACKUP="/mnt/backup/db-latest.sqlite"

mkdir -p "$(dirname "$DB")"
mkdir -p /mnt/backup

if [ -f "$DB" ]; then
    echo "[restore] Found db.sqlite, using it."
    exit 0
fi

if [ -f "$BACKUP" ]; then
    echo "[restore] Restoring from backup."
    cp "$BACKUP" "$DB"
    exit 0
fi

echo "[restore] No DB or backup found, bot will initialize on startup."
