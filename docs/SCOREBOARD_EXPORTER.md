# 📊 Scoreboard Exporter & Discord Activity Pipeline

This document details the **Scoreboard Exporter** pipeline, its architectural motivation, opt-in feature flag configuration, storage provider implementations, and cloud production deployment.

---

## 1. Problem & Architectural Motivation (Why the Exporter Exists)

### A. The Discord Image Legibility Issue
For algorithms club classes with 30+ students and 50+ problems, generating a single PNG table image (`!score`) resulted in massive images that Discord automatically downscaled to a 500px preview width in the chat UI. All text became completely unreadable unless users manually downloaded the file.

### B. The Interactive Web Scoreboard Solution
To solve legibility, Hurion provides an interactive React 19 web application hosted on Azure Static Web Apps and accessible directly via Discord embeds (`!score`) or as an embedded Discord Activity. The web frontend requires student score data to render responsive tables, topic filtering, and sorting.

### C. Data Push (Static JSON) vs Data Pull (Live API Queries)
We explicitly avoided building a live `GET /api/score` web server reading SQLite on demand for three key architectural reasons:
1. **SQLite Concurrency (`SQLITE_BUSY`)**: Avoids multi-process file locking conflicts when a web server reads SQLite while Litestream and the bot write to it.
2. **Zero Database Load on Web Views**: 100 users opening the scoreboard fetch static JSON files from Azure Blob Storage / CDN without executing a single query on the SQLite database.
3. **Zero Bot Command Latency**: An in-process SQLite driver update hook catches writes asynchronously, and a background worker debounces writes over 800ms to compile JSON files in the background without delaying bot commands.

---

## 2. Opt-In Feature Flag & Makefile Integration

The Scoreboard Exporter is **opt-in** and strictly disabled by default to save resources in lightweight or test environments.

### CLI Flag
* `--exporter`: Enables the background `ScoreboardExporterWorker` and SQLite driver update hook.

### Running via `make`
All execution is unified through the project `Makefile` using composable flags:

```bash
# Run bot with Exporter enabled
make run FLAGS="--exporter"

# Run bot in full local Dev Mode (enables --seed and --exporter)
make run-dev

# Combine multiple flags dynamically
make run FLAGS="--seed --exporter"
```

---

## 3. Storage Providers

The exporter uses the `storage.BlobStorage` interface (`Save(ctx context.Context, name string, data []byte) error`) to support both cloud and local workflows:

### A. Local Development (`DiskStorage`)
* Automatically selected when Azure environment variables are absent.
* Writes JSON files atomically (`writeAtomic` via temporary file and `os.Rename`) to `./data/exports/<class_code>_score.json`.

### B. Production Cloud (`AzureBlobStorage`)
* Automatically selected when `AZURE_STORAGE_ACCOUNT` and `AZURE_STORAGE_KEY` (or `LITESTREAM_AZURE_ACCOUNT_KEY`) are set via `storage.NewStorageFromEnv()`.
* Stream JSON byte buffers directly to Azure Blob Storage over HTTPS via the official Azure SDK for Go (`azblob`).

---

## 4. Production Environment Variables (Azure VM)

The following 5 environment variables are configured on the Azure VM where the Hurion Bot executes:

| Variable | Description | Example |
| :--- | :--- | :--- |
| `DISCORD_BOT_TOKEN` | Discord Bot Authentication Token | `MTA...` |
| `AZURE_STORAGE_ACCOUNT` | Azure Storage Account Name | `hurionstorage` |
| `AZURE_STORAGE_KEY` *(or `LITESTREAM_AZURE_ACCOUNT_KEY`)* | Azure Storage Shared Access Key | `w8x...==` |
| `AZURE_STORAGE_CONTAINER` | Azure Blob Container for exports | `exports` |
| `WEB_UI_BASE_URL` | Base URL of hosted Azure Static Web App | `https://<your-app-name>.azurestaticapps.net` |

---

## 5. Scoreboard JSON Schema

Exported files (`<class_code>_score.json`) follow this JSON schema:

```json
{
  "classCode": "basicos_1",
  "className": "Algoritmos Básicos I",
  "problems": [
    {
      "id": 101,
      "name": "Two Sum",
      "url": "https://leetcode.com/problems/two-sum",
      "difficulty": "Easy",
      "topic": "Arrays"
    }
  ],
  "users": [
    {
      "userId": "123456789",
      "displayName": "Alice",
      "solvedCount": 1,
      "solved": [101]
    }
  ]
}
```

---

## 6. Architecture References

* [ADR 0001: Discord Activity Scoreboard Architecture](adr/0001-discord-activity-scoreboard.md)
* [Scoreboard Web UI Guide](WEB_UI.md)
* [Litestream Backup Documentation](LITESTREAM.md)
