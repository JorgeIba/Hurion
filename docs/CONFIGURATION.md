# Configuration Reference

Hurion uses `config.yaml` for Discord server, role, and class IDs. Environment variables configure the bot, web server, storage providers, and deployment. This page documents the YAML fields and indexes the repository's environment variables; detailed provider and frontend behavior stays in the linked guides. Start by copying the YAML example:

```bash
cp config.example.yaml config.yaml
```

`config.yaml` is read at startup and validated strictly. Environment variables do not override its YAML fields. Copy `.env.example` to `.env` only when using Docker Compose; local `make` commands do not load `.env` automatically.

## `config.yaml`

| Field | Purpose |
|---|---|
| `guild_id` | Discord server ID. Required when the live bot registers its guild slash commands. |
| `admin_role_id` | Required Discord role ID. Administrators can manage every class. |
| `mentor_role_id` | Required global mentor role. Classes inherit it unless they define an override. |
| `test_mode` | Optional; defaults to `false`. When enabled, at least one whitelist must be configured. |
| `whitelist_user_ids` | User IDs allowed by test mode. |
| `whitelist_role_ids` | Role IDs allowed by test mode. |
| `classes` | Class definitions used by Hurion. |

Each entry under `classes` supports:

| Field | Purpose |
|---|---|
| `code` | Required class code used in commands, URLs, and exported scoreboard filenames. |
| `name` | Required display name. |
| `room_ids` | Discord channel IDs associated with the class. |
| `role_ids` | Discord roles that give students access to the class. This is separate from mentor permissions. |
| `mentor_role_id` | Optional class-specific mentor role. It replaces the global `mentor_role_id` for this class; an omitted or blank value inherits the global role. |

Use quoted strings for Discord IDs. To create classes and link the configured channels and student roles during startup, run the bot with `--seed`; this does not create Discord roles, channels, or problems. Configuration changes take effect after restarting Hurion. Mentor-role overrides load at startup without `--seed` and do not require a database migration. To limit the global mentor role's access to a class, define an override for that class; classes without an override continue to use the global role.

## Environment variable index

| Variable | Used by | Purpose and details |
|---|---|---|
| `DISCORD_BOT_TOKEN` | Live bot | Discord bot credentials. See [Local Development](LOCAL_DEVELOPMENT.md). |
| `DB_PATH` | Bot and containers | SQLite database location. See [Litestream](LITESTREAM.md). |
| `WEB_UI_BASE_URL` | Bot | Base URL for the interactive `!score` link. See the [Web UI guide](WEB_UI.md). |
| `PORT` | Local Go web server | HTTP listening port. See [Local Development](LOCAL_DEVELOPMENT.md). |
| `EXPORTS_DIR` | Local Go web server | Directory for local scoreboard exports. See [Scoreboard Exporter](SCOREBOARD_EXPORTER.md). |
| `VITE_EXPORTS_BASE_URL` | Web UI build | Build-time base URL for scoreboard JSON. See the [Web UI guide](WEB_UI.md). |
| `AZURE_STORAGE_ACCOUNT` | Exporter and Litestream | Azure Storage account name. See [Scoreboard Exporter](SCOREBOARD_EXPORTER.md) and [Litestream](LITESTREAM.md). |
| `AZURE_STORAGE_KEY` | Exporter and Litestream | Azure Storage shared key; provider-specific use is covered in the linked guides. |
| `AZURE_STORAGE_CONNECTION_STRING` | Exporter | Azure connection string. See [Scoreboard Exporter](SCOREBOARD_EXPORTER.md). |
| `AZURE_STORAGE_CONTAINER` | Exporter and Litestream | Azure Blob container name. See the provider guides above. |
| `LITESTREAM_AZURE_ACCOUNT_KEY` | Litestream | Litestream-specific Azure key. See [Litestream](LITESTREAM.md). |
| `IMAGE_NAME` | Azure VM deployment | Container image reference used by `infra/compose.yaml`. See the [infrastructure guide](../infra/README.md). |
| `GOMEMLIMIT` | Azure VM container | Go runtime memory limit set by `infra/compose.yaml`. See the [infrastructure guide](../infra/README.md). |

## Where values are loaded

- Docker Compose reads `.env` through `compose.yaml`'s `env_file` setting.
- `make run` and `make run-dev` inherit variables from the shell. Export `DISCORD_BOT_TOKEN` before running them; creating `.env` alone is not enough.
- Vite reads its own `.env` files when building the frontend. Put frontend values in `web/.env.local`, or export them in the shell before building; the root `.env` used by Docker Compose is not loaded by Vite.

For local startup recipes and the Make targets, see [Local Development](LOCAL_DEVELOPMENT.md). For Docker setup, see [README.Docker.md](../README.Docker.md).
