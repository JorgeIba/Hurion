### Building and running your application

**Before starting, you need to configure your environment variables:**

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit the `.env` file and set the required values:
   - `DISCORD_BOT_TOKEN`: Your Discord bot token
   - `DISCORD_ADMIN_ROLE_ID`: The Discord role ID for administrators
   - `DISCORD_MENTOR_ROLE_ID`: The Discord role ID for mentors

When you're ready, start your application by running:
`docker compose up --build`.

### Data Persistence (Litestream)

The bot uses **Litestream** for real-time replication to **Azure Blob Storage**. This allows the container to run on fast, ephemeral storage while ensuring no data loss.

**Local Development:**
- In local dev mode (using `docker compose`), Litestream replication is automatically disabled if Azure credentials are not found.
- The database is stored in the container's internal filesystem.
- To persist data locally during dev, you can mount a volume to `/huron-bot/data` in your `compose.yaml`.

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
