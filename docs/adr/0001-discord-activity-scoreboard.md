# ADR 0001: Asynchronous Scoreboard Exporter & Discord Activity Launcher

* **Status**: Accepted
* **Date**: 2026-07-31

---

## 1. Context and Problem Statement

Hurion is a Go-based Discord bot designed for competitive programming clubs. Previously, when a user requested a class scoreboard (`!score`), the bot rendered a monolithic PNG table image in memory and attached it to a Discord message.

When a class grows to **30+ students and 50+ problems**, rendering the entire grid as a static image creates a massive canvas. Discord automatically downscales large images to a 500px preview width, rendering text completely illegible without forcing users to manually download the file.

### Constraints & Goals
1. **Legibility & Interactivity**: Users must be able to view and explore full class standings interactively inside Discord without downloading files.
2. **Zero Response Latency**: Database writes (e.g. marking a problem as solved) must return instantly without blocking on file compilation or network cloud uploads.
3. **Database Concurrency & Locking**: The application uses SQLite. Running an external OS process to read the SQLite database triggers file locking conflicts (`SQLITE_BUSY`) during Write-Ahead Logging (WAL) checkpoints.
4. **Clean Layered Architecture**: The solution must adhere to Hurion's strict layered architecture (`app/data`, `app/services`, `app/presentation`, `app/infrastructure`, `app/workers`).

---

## 2. Decision Drivers

* **User Experience (UX)**: Provide a modern, interactive web application embedded directly in Discord via a Discord Activity (iframe).
* **Performance**: Asynchronous, debounced background processing so SQLite write transactions never stall.
* **Separation of Concerns**: Keep business domain services pure; decouple presentation formatting and storage infrastructure behind clear interfaces.
* **Flexibility & Portability**: Support local disk persistence (`DiskStorage`) for offline development/testing and cloud storage (`AzureBlobStorage`) for production deployments.

---

## 3. Considered Alternatives & Why Rejected

### Alternative A: Image Pagination / Split PNGs
* **Concept**: Split large tables across multiple PNG images or paginated embeds.
* **Why Rejected**: Fails to solve the scale issue when a single class has 50+ problems—horizontal scrolling or multi-image pagination remains clunky and hard to read in Discord chat.

### Alternative B: External Microservice / Binary
* **Concept**: Run a separate Go/Node binary alongside the bot to read SQLite and compile JSON files.
* **Why Rejected**: Triggers multi-process file locking conflicts (`SQLITE_BUSY`) on SQLite database files, leading to intermittent query failures under write loads.

### Alternative C: Monolithic Service compiling files inline
* **Concept**: Put file compilation directly inside `ScoreService` or command controllers.
* **Why Rejected**: Violates Separation of Concerns. Inline file writes block Discord message handlers whenever disk or network I/O stalls.

---

## 4. Decision Outcome & Architecture

We decided to adopt an **In-Process, Event-Driven Asynchronous Pipeline** using SQLite driver update hooks, a background orchestrator worker, pure presentation formatters, and a storage abstraction interface.

```
┌─────────────────────────┐
│ SQLite Database Write   │
└────────────┬────────────┘
             │ (RegisterUpdateHook)
             ▼
┌─────────────────────────┐
│ Database Update Hook    │  ──► Async Goroutine (Panic Recovery)
└────────────┬────────────┘
             │ (updateEventChannel)
             ▼
┌─────────────────────────┐
│ Exporter Worker         │  ──► 800ms Debounce & 10s Timeout
└────────────┬────────────┘
             │
      ┌──────┴────────────────────────┐
      ▼                               ▼
┌─────────────────────────┐     ┌─────────────────────────┐
│ jsonexport Formatter    │     │ BlobStorage Interface   │
│ (Pure Domain -> JSON)   │     │ (Disk / Azure Blob)     │
└─────────────────────────┘     └─────────────────────────┘
```

### Key Components:

1. **Database Event Hook (`app/data/database.go`)**:
   * Registers a custom SQLite driver `sqlite3_with_hook` using `sql.Register`.
   * Attaches `conn.RegisterUpdateHook` to intercept write events on `solved_problems`.
   * Dispatches callbacks inside asynchronous goroutines wrapped with `defer recover()` panic handlers, ensuring database connection threads return immediately.

2. **Orchestrator Worker (`app/workers/exporter.go`)**:
   * Implements `ScoreboardExporterWorker` with a buffered event channel `updateEventChannel` (capacity 1) using a non-blocking `select/default` send.
   * Pauses for 800ms (`DefaultDebounceDuration`) to consolidate rapid write bursts.
   * Applies a 10-second timeout context (`DefaultExportTimeout = 10s`) to prevent stalled exports.
   * Manages graceful application shutdown using `sync.WaitGroup` (`workerWaitGroup`) and `shutdownSignal`, guaranteeing active file writes complete cleanly before process exit.

3. **Presentation Formatter (`app/presentation/jsonexport/scoreboard.go`)**:
   * Pure presentation package mapping `service_model.Score` domain entities into `ScoreboardJSON` byte artifacts.
   * Initializes empty slices (`make(..., 0)`) so empty classes format as valid JSON arrays (`"problems": []`, `"users": []`) rather than `null`.

4. **Storage Infrastructure (`app/infrastructure/storage`)**:
   * Declares `BlobStorage` interface (`Save(ctx context.Context, path string, data []byte) error`).
   * Implements `DiskStorage` with standard Go octal permissions (`DefaultDirPermissions = 0o755`, `DefaultFilePermissions = 0o644`).
   * Prepares seamless injection of `AzureBlobStorage` for cloud deployment.

---

## 5. Phased PR Rollout Plan

To ensure rigorous quality control and isolated code reviews, implementation is divided into **3 distinct Pull Requests**:

* **PR #1: Driver Hook & Async Exporter Pipeline (COMPLETED)**
  * SQLite update hook registration, `ScoreboardExporterWorker`, `jsonexport` formatter, `DiskStorage`, and 100% test coverage across unit and integration workflows.
* **PR #2: Azure Blob Storage Integration (NEXT)**
  * Implement `AzureBlobStorage` fulfilling `BlobStorage` interface and update worker initialization in `main.go` to support Azure credentials.
* **PR #3: Discord Activity Web UI & Embed Launcher**
  * Interactive scoreboard frontend (HTML/CSS/JS) and `!score` command embed update containing a Discord Activity Button launcher.

---

## 6. Consequences & Trade-offs

### Positive
* **Legible & Scalable UX**: Interactive web application allows smooth scrolling and sorting for 50+ problems across 30+ students.
* **Zero Command Latency**: Asynchronous driver hooks ensure Discord command execution times remain ultra-fast.
* **Concurrency Safety**: In-process event handling avoids SQLite multi-process file locking issues (`SQLITE_BUSY`).
* **High Testability**: Presentation formatting and storage logic are 100% testable in isolation without database or network mocks.

### Negative / Neutral
* **Background Worker Memory**: Worker goroutines consume a tiny background memory footprint (~few KB).
* **Storage Configuration**: Production deployments require configured Azure Blob Storage environment variables or local disk exports folder access.
