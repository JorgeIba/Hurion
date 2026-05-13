# Huron Problem Checker Bot

A Discord bot built in **Go** for managing algorithms courses. It handles class management, problem assignments, solution tracking, and score visualization — all orchestrated through Discord commands with **stateless, role-based access control**.

Built with [`discordgo`](https://github.com/bwmarrin/discordgo) and backed by **SQLite**.

---

## Documentation

- [Architecture & Design](docs/ARCHITECTURE.md) — Layered architecture, diagram, DB schema, and confirmation flow.
- [Commands Reference](docs/COMMANDS.md) — Full list of commands by role (General, Enrolled, Mentor, Admin).
- [AI Hub](.ai/CONTEXT.md) — Central source of truth for AI agents and contribution rules.
- [Testing Guidelines](.ai/TESTING.md) — Mandatory testing patterns and parallelism.

---

## Setup & Installation

### Prerequisites

| Requirement | Version |
|---|---|
| **Go** | 1.24.1+ |
| **SQLite3** | Bundled via `go-sqlite3` (requires CGo) |
| **GCC / C Compiler** | Required for CGo (e.g., `gcc` on Linux) |

### Configuration

The bot uses a combination of environment variables and a `config.yaml` file.

1. **Environment Variables** (`.env`):
   ```env
   DISCORD_BOT_TOKEN=your_token_here
   DB_PATH=./data/db.sqlite
   ```

2. **Application Config** (`config.yaml`):
   ```yaml
   admin_role_id: "1234567890"
   mentor_role_id: "0987654321"

   classes:
     - code: "basicos_1"
       name: "Basicos 2025"
       role_ids: ["111111", "222222"]
       room_ids: ["333333", "444444"]
   ```

### Database Seeding

To bootstrap classes, roles, and channels from your `config.yaml`, use the provided make command:


```bash
make run-seed
```

---

## Running the Bot

```bash
# Standard run
make run
```

### Running with Docker

The bot is container-ready and includes **Litestream** for real-time database replication.

```bash
docker compose up --build -d
```

For detailed instructions on Docker deployment and persistence, see [README.Docker.md](README.Docker.md).

### Discord Bot Requirements

Enable these **Privileged Gateway Intents** in the Discord Developer Portal:
1. **Server Members Intent**: To resolve nicknames.
2. **Message Content Intent**: To process commands.

