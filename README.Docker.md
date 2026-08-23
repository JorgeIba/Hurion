### Building and running your application

**Before starting, you need to configure your application:**

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit the `.env` file and set the `DISCORD_BOT_TOKEN`.

3. Configure your roles and classes in `config.yaml`:
   ```bash
   cp config.example.yaml config.yaml
   ```
   Edit `config.yaml` to set your `admin_role_id`, `mentor_role_id`, and class definitions. Role IDs are now managed here instead of `.env`.

When you're ready, start your application by running:
```bash
docker compose up --build -d
```

### Runtime Commands & Flags (Docker)
The Docker image runs via `make run FLAGS="..."`. You control the bot's behavior by passing flags directly in your `compose.yaml` under the `command` property:

```yaml
services:
  service:
    image: hurion:local
    command: ["--seed", "--exporter"] # Optional composable flags
```

* **`--seed`**: Performs an **additive bootstrap** of the database using your `config.yaml` file every time the container boots up.
* **`--exporter`**: Enables automatic background scoreboard JSON exports to Azure Blob Storage or local disk for the Discord Activity web UI. See **[docs/SCOREBOARD_EXPORTER.md](docs/SCOREBOARD_EXPORTER.md)**.

### Data Persistence & Replication
* **Litestream Database Replication**: For disaster recovery configuration and restore steps, see **[docs/LITESTREAM.md](docs/LITESTREAM.md)**.
* **Discord Activity JSON Exporter**: For background export architecture and schema details, see **[docs/SCOREBOARD_EXPORTER.md](docs/SCOREBOARD_EXPORTER.md)**.

The bot uses **Litestream** for real-time replication to **Azure Blob Storage**.

**Production (Azure VM):**
- On startup, the bot attempts to restore the database from the Azure Blob Storage replica.
- Once running, every database change is continuously streamed to Azure.
- On a VM, Litestream communicates directly with the Azure Blob API over the network; no external file shares need to be mounted.

### Deployment & Container Registry (`ghcr.io`)

Production container images are published to **GitHub Container Registry (`ghcr.io`)**.

#### 1. Local Testing
To build and run the container locally without pushing to any remote registry:
```bash
docker compose up --build -d
```

#### 2. Manual Build & Push to `ghcr.io`
If you wish to manually compile and push an image to GitHub Container Registry from your terminal:

```bash
# Authenticate using the GitHub CLI (zero password prompts):
gh auth token | docker login ghcr.io -u <YOUR_GITHUB_USERNAME> --password-stdin

# Build and tag image:
docker build -t ghcr.io/jorgeiba/hurion-bot:latest .

# Push image to GitHub Packages:
docker push ghcr.io/jorgeiba/hurion-bot:latest
```

Hosting and provisioning on Azure VM are managed via Bicep and GitHub Actions CD. See **[infra/README.md](infra/README.md)** and **[docs/adr/0002-container-delivery-and-cicd-pipeline.md](docs/adr/0002-container-delivery-and-cicd-pipeline.md)** for architecture details.

### References
* [Litestream Documentation](https://litestream.io/)
* [Docker's Go guide](https://docs.docker.com/language/golang/)
* [GitHub Packages Docker Guide](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
