# Scoreboard Web UI & Dev Server Guide

This document covers the setup, local development workflow, and cloud deployment configuration for the **Hurion Scoreboard Web UI** (`web/`).

---

## 1. Overview

The Scoreboard Web UI is a standalone React 19 + Vite web application that renders live competitive programming leaderboards, difficulty statistics, and student problem solves.

* **Decoupled Static Pipeline**: Reads pre-rendered `<class_code>_score.json` files generated asynchronously by the `ScoreboardExporterWorker`.
* **AI Agent Guide**: The Static Web App serves `llms.txt`; batch schemas, classes, and scoreboards are published together by the bot exporter to the configured exports storage.
* **Zero Database Load**: Web clients fetch static JSON exports directly without triggering live database queries on SQLite.
* **Agnostic Frontend**: Runs in any standard web browser, mobile overlay, or embedded window (such as a Discord Activity).

---

## 2. Local Development Workflow

To build and test the Web UI locally on your machine:

1. **Install Frontend Dependencies & Build Assets**:
   ```bash
   pnpm --prefix web install
   pnpm --prefix web build
   ```
   *(Compiles React 19 production bundle into `web/dist/`)*

2. **Generate Local Scoreboard JSON Exports**:
   ```bash
   make run-dev
   ```
   *(Runs the bot with `--seed --exporter`, exporting JSON files into `./data/exports/`)*

3. **Launch Local Web UI Dev Server**:
   ```bash
   make run-server PORT=8080
   ```
   *(If `web/dist/` is not yet compiled, the server automatically returns a 503 developer notice page with build instructions)*

4. **Open in Web Browser**:
   Navigate to:
   [http://localhost:8080/?class=basicos_1](http://localhost:8080/?class=basicos_1)

5. **Stop the Web UI Server**:
   ```bash
   make stop-server
   ```

---

## 3. Production Deployment (Azure Static Web Apps)

In production, the frontend is deployed to **Azure Static Web Apps (SWA)** via GitHub Actions (`.github/workflows/azure-static-web-apps-*.yml`).

### Single Page Application (SPA) Routing (`web/public/staticwebapp.config.json`)
Azure Static Web Apps uses `web/public/staticwebapp.config.json` (which Vite copies to `web/dist/`) to handle client-side HTML5 history navigation and prevent 404 errors on deep linking:

```json
{
  "$schema": "https://json.schemastore.org/staticwebapp.config.json",
  "navigationFallback": {
    "rewrite": "/index.html",
    "exclude": ["/assets/*", "/llms.txt"]
  },
  "responseOverrides": {
    "404": {
      "rewrite": "/index.html",
      "statusCode": 200
    }
  }
}
```

> [!NOTE]
> When fetching directly from Azure Blob Storage in production (`VITE_EXPORTS_BASE_URL`), ensure the `exports` container has public anonymous blob read access and CORS enabled (`GET, HEAD, OPTIONS`) for the Static Web App domain.
> The public agent guide is available at `${WEB_UI_BASE_URL}/llms.txt`; its schemas and state links point to the exports storage published by the bot.

---

## 4. Environment Variables

| Variable | Scope | Description | Example / Default |
| :--- | :--- | :--- | :--- |
| `PORT` | Local Dev (Go) | Listening TCP port for the local Go Web UI server | `8080` |
| `EXPORTS_DIR` | Local Dev (Go) | Path to local directory containing exported JSON score files | `./data/exports` |
| `VITE_EXPORTS_BASE_URL` | Frontend (Vite) | Base URL for fetching scoreboards (uses `/exports` if unset) | `https://<your-storage-account>.blob.core.windows.net/exports` |
| `WEB_UI_BASE_URL` | Discord Bot (VM) | Base URL used by Discord embeds to link to the Web UI (`!score`) | `https://<your-app-name>.azurestaticapps.net` |

## 5. Class Priority Presentation

Priority presentation supports general defaults and class-specific overrides,
configured in `web/src/presentation/priorityConfig.ts`. Overrides affect web display
only, not stored codes or sorting. Configuration changes require rebuilding the
web app.
