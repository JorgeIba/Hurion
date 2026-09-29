# 📦 Litestream Database Replication & Disaster Recovery

This document explains how **Litestream** is configured in Hurion for real-time SQLite database replication and disaster recovery.

---

## 1. Overview & Purpose

Hurion uses SQLite as its primary database. To run safely on a single Azure Virtual Machine without complex SQL database clusters, Hurion uses [Litestream](https://litestream.io/) for continuous Write-Ahead Log (WAL) replication.

* **Continuous WAL Streaming**: Litestream intercepts SQLite WAL frame writes and streams them to storage over the network.
* **Auto-Restoration on Boot**: If the VM restarts or a new container spins up without a local database file, Litestream automatically restores the latest database snapshot before launching the bot.

---

## 2. Configuration Files

Litestream uses two configuration files depending on your runtime environment:

Production replica paths follow `server-<id>/db.sqlite`. The ID started at `1` for the disposable test database; the production classroom database uses `server-2/db.sqlite`. Advance to a new, unused ID only when intentionally starting another replication history. A normal restart or restore keeps the same ID. The path below is the **current value**, not a permanent path requirement.

* **Production (`/etc/litestream.yml`)**:
  Streams database backups to Azure Blob Storage:
  ```yaml
  dbs:
    - path: ${DB_PATH}
      replicas:
        - url: abs://${AZURE_STORAGE_ACCOUNT}@${AZURE_STORAGE_CONTAINER}/server-2/db.sqlite
  ```
* **Local Development (`/etc/litestream.local.yml`)**:
  Streams local backups to `/hurion/local-backups/db.sqlite` on disk when Azure credentials are missing.

Each new `server-<id>` prefix starts a separate replica history and leaves previous prefixes untouched. Changing this URL does **not** replace the local `/hurion/data/db.sqlite`: install the intended database while the service is stopped before starting an image with the new config, or Litestream will replicate the old local database to the new prefix.

---

## 3. Environment Variables

To enable production Azure replication, specify these variables in your `.env` file:

| Environment Variable | Description | Example |
| :--- | :--- | :--- |
| `DB_PATH` | Path to the SQLite database file | `/hurion/data/db.sqlite` |
| `AZURE_STORAGE_ACCOUNT` | Azure Storage Account Name | `hurionstorage` |
| `AZURE_STORAGE_KEY` *(or `LITESTREAM_AZURE_ACCOUNT_KEY`)* | Azure Shared Access Key | `w8x...==` |
| `AZURE_STORAGE_CONTAINER` | Azure Container Name | `exports` |

---

## 4. Container Startup Lifecycle

Container startup is managed by `scripts/run.sh`:

1. **Directories**: Ensures `/hurion/data` and `/hurion/local-backups` directories exist.
2. **Config Detection**: Detects if Azure credentials exist to pick `/etc/litestream.yml` or `/etc/litestream.local.yml`.
3. **Database Restore**: Runs `litestream restore -if-db-not-exists -if-replica-exists` to download existing database backups.
4. **Pragma Check**: Enforces `PRAGMA journal_mode=WAL;`.
5. **Process Supervision**: Executes `litestream replicate` to supervise the compiled binary `/hurion/bin/hurion $*` directly.

---

## 5. Emergency Disaster Recovery (Manual Restore)

If you ever need to manually restore the database from Azure Blob Storage:

```bash
litestream restore -config /etc/litestream.yml -o /hurion/data/db.sqlite /hurion/data/db.sqlite
```

---

## 6. Official References

* [Litestream Official Website](https://litestream.io/)
* [Litestream Azure Blob Storage Replica Documentation](https://litestream.io/reference/replica/#azure-blob-storage)
* [Litestream GitHub Repository](https://github.com/benbjohnson/litestream)
