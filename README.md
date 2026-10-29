<p align="center">
  <img src="docs/logo_readme.png" alt="Hurion logo" width="160">
</p>

<h1 align="center">Hurion</h1>

<p align="center">
  A Discord bot for algorithm and competitive-programming classes. Students submit solved problems, mentors review them, and Hurion tracks class progress.
</p>

## What Hurion does

- **Students** submit solves with `!ac <problem_id>` or `/ac submit id:<problem_id>`. Slash submissions can include an optional screenshot.
- **Mentors** review submissions and manage problem lists, including batch add and edit workflows.
- **Admins** connect Discord channels and roles to classes, and export class results to Excel.
- **Scoreboards** are available as static PNGs with `!score_image` or through the interactive web UI with `!score`. The web UI supports search, filters, and sorting, and adapts to mobile screens.

## Quick start

You need Go 1.27+, GCC or another C compiler for SQLite, and a Discord bot with the **Server Members** and **Message Content** intents enabled. Invite the bot to your server, then clone Hurion and create its config file:

```bash
git clone https://github.com/JorgeIba/Hurion.git
cd Hurion
cp config.example.yaml config.yaml
```

Edit `config.yaml` with your server ID, admin and mentor role IDs, and class channel and role IDs. For per-class mentor roles and every configuration field, see the [configuration reference](docs/CONFIGURATION.md). Then set your bot token and start Hurion:

```bash
export DISCORD_BOT_TOKEN="<your bot token>"
make run FLAGS="--seed"
```

The `--seed` flag creates any missing classes and channel or role links from `config.yaml`; it does not create problems. For the detailed bot-only and bot-plus-web setup, see [Local Development](docs/LOCAL_DEVELOPMENT.md).

### Try your first submission

1. Run `!classes` in Discord to confirm the configured classes.
2. In a class channel, a mentor can create a problem with `/problem add`; run `!list` there to find its ID.
3. A student with access to that class can submit it with `/ac submit id:<problem_id>`. A mentor can then review the proposal.

### Optional: interactive scoreboard

The interactive scoreboard runs separately from the bot. Build and serve it using the [web UI guide](docs/WEB_UI.md). To enable `!score`, set `WEB_UI_BASE_URL` for the bot and restart Hurion with `make run FLAGS="--exporter"` to publish score data for the web UI. See the [exporter guide](docs/SCOREBOARD_EXPORTER.md) for details. For Docker, see the [Docker guide](README.Docker.md).

## Documentation

**Using Hurion**

- [Commands reference](docs/COMMANDS.md) — commands by role.
- [Scoreboard user guide](docs/SCOREBOARD_GUIDE.md) — standings, problems, search, filters, and sorting.
- [Mentor batch guide](docs/MENTOR_BATCHES.md) — JSON workflows for adding or editing problems and recording solves.

**Configuration and running Hurion**

- [Configuration reference](docs/CONFIGURATION.md) — `config.yaml` fields and environment variables.
- [Local development](docs/LOCAL_DEVELOPMENT.md) — run the bot, exporter, and web UI; common Make targets.
- [Docker setup](README.Docker.md) — container configuration and deployment.
- [Scoreboard web UI](docs/WEB_UI.md) — frontend configuration and deployment.
- [Scoreboard exporter](docs/SCOREBOARD_EXPORTER.md) — publish data for the interactive scoreboard.
- [Database replication](docs/LITESTREAM.md) — backup and restore with Litestream.

**Development**

- [Architecture](docs/ARCHITECTURE.md) — application layers and database schema.
- [Script runner](docs/SCRIPT_RUNNER.md) — run command scripts offline or against Discord.
- [Testing guidelines](.ai/TESTING.md) — project testing conventions.

## 🌟 Why Hurion?

Hurion is named after Huron, the ferret mascot of ESCOM's algorithms club, and Orion, the Greek hunter and constellation.

## Contributing

Open a [new issue](https://github.com/JorgeIba/Hurion/issues/new) first. Use its discussion to agree on the goal, scope, and approach before implementation starts. Link every PR to that issue in its description. If the work needs multiple PRs, link each one to the same issue (for example, `Refs #<issue-number>`) and close it when all the work is complete.

The current logo was created with AI using the club mascot as a reference. Contributions toward a handcrafted vector version are welcome.
