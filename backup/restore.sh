#!/bin/bash
set -e

DB="/app/data/db.sqlite"
BACKUP="/mnt/backup/db-latest.sqlite"
SCHEMA="/app/data/schema.sql"

mkdir -p /app/data
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

echo "[restore] No DB or backup found, creating new DB."
sqlite3 "$DB" < "$SCHEMA"
