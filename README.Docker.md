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

### Database Seeding (Docker)
Because the `Dockerfile` is configured with the `--seed` flag by default (`CMD ["/huron-bot/run.sh", "--seed"]`), the database will automatically synchronize with your `config.yaml` every time the container boots up. 

Since the seeder is idempotent, this is completely safe and ensures your Azure (or local Docker) deployments always have the latest classes and roles applied automatically before the bot connects to Discord.

> **Important**: `config.yaml` is baked into the Docker image during the build process (`COPY config.yaml ...`). If you modify `config.yaml` (e.g., to add a new class), you **must** rebuild your Docker image and push the update to Azure for the changes to take effect.

### Data Persistence (Litestream)

The bot uses **Litestream** for real-time replication to **Azure Blob Storage**. This allows the container to run on fast, ephemeral storage while ensuring no data loss.

**Local Development:**
- In local dev mode (using `docker compose`), the bot automatically switches to **local file replication** (using `litestream.local.yml`).
- Backups are stored in the `./local-backups` folder on your host machine.
- This allows you to test restoration and persistence locally without an Azure connection.

**Production (Azure):**
- On startup, the bot attempts to restore the database from the Azure Blob Storage replica.
- Once running, every database change is continuously streamed to Azure.

### Deployment

To deploy to Azure, you must provide the following environment variables:

- `AZURE_STORAGE_ACCOUNT`: Your storage account name.
- `AZURE_STORAGE_CONTAINER`: The blob container for backups (e.g., `litestream-backups`).
- `LITESTREAM_AZURE_ACCOUNT_KEY`: Your storage account access key.

**1. Build and Push:**
```bash
docker build -t <your-username>/huron-problem-checker-bot:latest .
docker push <your-username>/huron-problem-checker-bot:latest
```

### References
* [Litestream Documentation](https://litestream.io/)
* [Docker's Go guide](https://docs.docker.com/language/golang/)
