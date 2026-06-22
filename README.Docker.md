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
The Docker image is built without any default arguments to keep it agnostic. You control the bot's behavior by passing flags directly in your `compose.yaml` under the `command` property:

```yaml
services:
  service:
    image: hurion:local
    command: ["--seed"] # Optional
```

* **`--seed`**: Performs an **additive bootstrap** of the database using your `config.yaml` file every time the container boots up. It safely creates any missing classes, rooms, or roles defined in your config. Because it is strictly additive, it will *not* delete or overwrite existing data in the database.

### Data Persistence (Litestream)

The bot uses **Litestream** for real-time replication to **Azure Blob Storage**.

**Production (Azure VM):**
- On startup, the bot attempts to restore the database from the Azure Blob Storage replica.
- Once running, every database change is continuously streamed to Azure.
- On a VM, Litestream communicates directly with the Azure Blob API over the network; no external file shares need to be mounted.

### Deployment (Infrastructure as Code)

Hosting and provisioning are managed via Bicep and Cloud-Init. See the **[infra/README.md](infra/README.md)** for instructions on how to provision the Azure VM and deploy the application.

### References
* [Litestream Documentation](https://litestream.io/)
* [Docker's Go guide](https://docs.docker.com/language/golang/)
