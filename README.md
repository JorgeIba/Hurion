<p align="center">
  <img src="docs/logo_readme.png" alt="Hurion Logo" width="200">
</p>

<h1 align="center">Hurion</h1>

<p align="center">
  <strong>The course assistant for algorithms clubs.</strong><br>
  Manage classes, verify solutions, and track progress — all without leaving Discord.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Database-SQLite-003B57?style=flat-square&logo=sqlite" alt="SQLite">
  <img src="https://img.shields.io/badge/Deployment-Docker-2496ED?style=flat-square&logo=docker" alt="Docker">
  <img src="https://img.shields.io/badge/Platform-Discord-5865F2?style=flat-square&logo=discord" alt="Discord">
</p>

---

## 🎯 What does it do?

Hurion transforms a Discord server into a full-scale classroom management system. It's built on a **stateless, role-based architecture** that maps a server's structure directly to its courses.

### 👤 For Students
*   **Submit with Ease:** Use `!ac <id>` to propose a solved problem.
*   **Real-time Leaderboards:** Check standings and class scores with beautiful image-based tables.
*   **Personal Tracking:** Keep a history of solved problems across different classes.

### 👨‍🏫 For Mentors
*   **Verification Loop:** Approve solutions with a single ✅ reaction. No more manual spreadsheets.
*   **Content Management:** Add, update, or remove problems in seconds.
*   **Class Oversight:** See who is active and who needs support with `!users`.

### ⚙️ For Admins
*   **Infrastructure Linking:** Connect Discord channels to classes and roles to access-levels.
*   **Data Portability:** Export entire class results to Excel with `!export`.
*   **System Control:** Manage classes and users with high-integrity confirmation flows.

---

## 💬 See it in action

> **Student:** `!ac 101`
>
> **Hurion:** *Reacts with ✅ and ❌*
> 
> **Mentor:** *Reacts with ✅ on the same message*
>
> **Hurion:** 🎈 **Accepted!** @Student has solved **"Two Sum"**. Their score has been updated.

---

## 📚 Documentation

Detailed guides for every part of the system:

- 🏗️ **[Architecture & Design](docs/ARCHITECTURE.md)** — Layered structure and DB schema.
- 📜 **[Commands Reference](docs/COMMANDS.md)** — Full list of commands by role.
- 🤖 **[AI Hub](.ai/CONTEXT.md)** — Rules for AI-assisted contributions.
- 🧪 **[Testing Guidelines](.ai/TESTING.md)** — Patterns for verified changes.
- 🏃‍♂️ **[Script Runner Guide](docs/SCRIPT_RUNNER.md)** — Writing, running, and testing command scripts offline or online.

---

## 🚀 Quick Start

### Prerequisites
*   **Go 1.25+**
*   **GCC / C Compiler** (For SQLite CGo)
*   **Discord Bot Token** with *Member* and *Message Content* intents enabled.

### Local Setup
1.  **Configure:** Create a `.env` with your `DISCORD_BOT_TOKEN`.
2.  **Initialize:** Define classes in `config.yaml`.
3.  **Launch (Standard):** `make run`
4.  **Launch (Full Dev Mode):** `make run-dev` (enables `--seed` and `--exporter`)

### Runtime Flags & Makefile Composition
The bot's behavior is controlled by CLI flags composed via `make run FLAGS="..."`:

*   **Standard Mode (`make run`):** Boots up and connects to Discord using existing SQLite data.
*   **Dev Mode (`make run-dev`):** Shortcut that passes `--seed --exporter`.
*   **Seed Mode (`make run FLAGS="--seed"`):** Performs an **additive bootstrap** against the database before connecting to Discord.
*   **Exporter Mode (`make run FLAGS="--exporter"`):** Enables automatic background export of `<class_code>_score.json` files to Azure Blob Storage or local disk for the Discord Activity web UI. See **[docs/SCOREBOARD_EXPORTER.md](docs/SCOREBOARD_EXPORTER.md)**.
*   **Mock Discord Mode (`make run-mock`):** Runs the bot offline without connecting to Discord APIs (`--mock-discord --script <path>`).

### 📚 Data Persistence, Web UI & Replication
*   **Database Disaster Recovery**: For Litestream WAL replication and restoration guides, see **[docs/LITESTREAM.md](docs/LITESTREAM.md)**.
*   **Discord Activity JSON Exporter**: For background export architecture, opt-in `--exporter` flag, and schema details, see **[docs/SCOREBOARD_EXPORTER.md](docs/SCOREBOARD_EXPORTER.md)**.
*   **Scoreboard Web UI**: For React 19 frontend setup, local dev server, and Azure Static Web Apps deployment, see **[docs/WEB_UI.md](docs/WEB_UI.md)**.
*   **Docker Setup**: For container configuration, see **[README.Docker.md](README.Docker.md)**.

---

## 🌟 Why Hurion?

The name **Hurion** is the fusion of two identities:

*   **The Huron (Ferret):** The agile mascot of the algorithms club at **ESCOM** (Escuela Superior de Cómputo). Known for its curiosity and its ability to "ferret out" solutions in complex environments.
*   **Orion:** The legendary Greek hunter and celestial guide. Just as the constellation guides travelers, Hurion guides students through their journey in competitive programming.

**Hurion** is the "Hunter of Solutions"—a centralized assistant designed to handle the heavy administrative lifting so mentors can focus on teaching and students can focus on coding.

> **🎨 Note on the Logo:** The current logo was generated with the help of AI based on our club's original mascot. If you're a designer and would like to help us create a proper, handcrafted vector version, we would love your contribution!
