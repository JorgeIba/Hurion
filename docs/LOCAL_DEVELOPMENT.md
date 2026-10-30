# Local Development

This guide covers running the Discord bot by itself and running it with the local scoreboard. For the meaning of YAML fields and environment variables, see the [Configuration Reference](CONFIGURATION.md).

## Prerequisites

- Go 1.27 or newer.
- GCC or another C compiler for SQLite.
- A Discord bot token and a configured `config.yaml` for live bot operation.
- Node.js and pnpm when building or developing the web UI.

Create the config file once and fill in the IDs for your server, roles, classes, and channels:

```bash
cp config.example.yaml config.yaml
```

## Run only the bot

Export the Discord token in the shell that will run Hurion:

```bash
export DISCORD_BOT_TOKEN="<your bot token>"
make run
```

`make run` builds the bot and starts it without seeding classes or enabling scoreboard exports. To create any missing classes and link the channels and student roles from `config.yaml` on startup, use:

```bash
make run FLAGS="--seed"
```

The database defaults to `./data/db.sqlite`; Hurion creates the database schema on startup. The live bot needs `guild_id` to register slash commands. See [Configuration Reference](CONFIGURATION.md) for the full `config.yaml` and environment variable details.

## Run the bot with the exporter

To publish scoreboard JSON without starting the web server, enable the exporter flag:

```bash
export DISCORD_BOT_TOKEN="<your bot token>"
make run FLAGS="--exporter"
```

The exporter writes to local disk or Azure, depending on the environment. For a local-only run, leave Azure storage variables unset. See [Scoreboard Exporter](SCOREBOARD_EXPORTER.md) for provider behavior and exported files.

## Run the bot with the web scoreboard

The local scoreboard uses two processes. In terminal 1, start the bot with seeding and exports enabled:

```bash
export DISCORD_BOT_TOKEN="<your bot token>"
make run-dev
```

`make run-dev` is equivalent to running the bot with `--seed --exporter`. For a local-only run, leave Azure storage variables unset so the exporter writes JSON files to `./data/exports`.

In terminal 2, install the web dependencies once and start the local Go server:

```bash
pnpm --dir web install
make run-server PORT=8080
```

`make run-server` builds the frontend and serves it with the exported JSON files. Open `http://localhost:8080/?class=<class_code>`, using a `code` from `config.yaml` (for the example config, use `basics`). Stop the server with Ctrl+C or `make stop-server PORT=8080`.

When changing the React UI, keep the Go server running for `/exports`, then start Vite in another terminal:

```bash
pnpm --dir web dev
```

Vite serves the UI on port `3000` and proxies `/exports` to the Go server on port `8080`.

## Common Make targets

| Target | What it does |
|---|---|
| `make` or `make build` | Formats Go packages and builds `bin/hurion`. |
| `make run FLAGS="..."` | Builds and runs the bot with the supplied flags. |
| `make run-dev` | Builds and runs the bot with `--seed --exporter`. |
| `make run-mock` | Runs a JSONL script against mock Discord providers; override the script with `SCRIPT_PATH=...`. |
| `make run-server PORT=8080` | Builds the frontend and serves it with local exports through the Go server. |
| `make stop-server PORT=8080` | Stops the local web server. |

The Makefile's `##` comments are the source of truth for target descriptions; this guide lists only common workflows. They can later feed a `make help` target. See the [Makefile](../Makefile) for all targets and their current recipes.
