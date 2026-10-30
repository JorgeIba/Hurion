# Scoreboard Web UI & Dev Server Guide

This document covers frontend-specific configuration and cloud deployment for the **Hurion Scoreboard Web UI** (`web/`). For the complete local setup, see [Local Development](LOCAL_DEVELOPMENT.md). For class users, see the [Scoreboard User Guide](SCOREBOARD_GUIDE.md).

---

## 1. Overview

The Scoreboard Web UI is a standalone React 19 + Vite web application that renders live competitive programming leaderboards, difficulty statistics, and student problem solves.

* **Decoupled Static Pipeline**: Reads pre-rendered `<class_code>_score.json` files generated asynchronously by the `ScoreboardExporterWorker`.
* **AI Agent Guide**: Vite generates `dist/llms.txt` from `web/llms.txt.template` using the web app's `VITE_EXPORTS_BASE_URL` and shared default URL. Batch schemas, classes, and scoreboards are published together by the bot exporter to the configured exports storage.
* **Zero Database Load**: Web clients fetch static JSON exports directly without triggering live database queries on SQLite.
* **Agnostic Frontend**: Runs in any standard web browser, mobile overlay, or embedded window (such as a Discord Activity).

---

## 2. Frontend development

Install dependencies with `pnpm --dir web install`. For the bot, local data, and server workflow, see [Local Development](LOCAL_DEVELOPMENT.md). When changing the React UI, run `pnpm --dir web dev`; Vite serves port `3000` and proxies `/exports` to the local Go server on port `8080`.

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
> When fetching directly from Azure Blob Storage in production, ensure the `exports` container has public anonymous blob read access and CORS enabled (`GET, HEAD, OPTIONS`) for the Static Web App domain.
> The public agent guide is available at `${WEB_UI_BASE_URL}/llms.txt`; its schemas and state links point to the exports storage published by the bot. `pnpm build` generates the guide automatically using Vite's resolved environment; `pnpm dev` serves it with `/exports`.

---

## 4. Configuration

`VITE_EXPORTS_BASE_URL` is the frontend's build-time URL override for scoreboard JSON. Set it in `web/.env.local` or the shell for local builds, or use the GitHub Actions repository variable for Azure Static Web Apps. Local browsers and Discord Activities use `/exports`. The complete variable list, including the Go server's `PORT` and `EXPORTS_DIR`, lives in the [Configuration Reference](CONFIGURATION.md).

## 5. Class Priority Presentation

Priority presentation supports general defaults and class-specific overrides,
configured in `web/src/presentation/priorityConfig.ts`. Overrides affect web display
only, not stored codes or sorting. Configuration changes require rebuilding the
web app.
