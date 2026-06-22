# Infrastructure as Code: Azure VM Hosting

This directory contains the Bicep templates and provisioning scripts to host the application on an **Azure B-Series Virtual Machine**.

## Architecture Layers
1.  **Hardware (Bicep):** `main.bicep` defines the VM (Standard_B1s), Virtual Network, and Network Security Group (Firewall).
2.  **Secret Store (Azure Key Vault):** Secrets are stored in a Key Vault and accessed via the VM's **Managed Identity**.
3.  **Provisioning (Custom Script Extension):** Automatically handles Docker installation, config file creation, and bot updates.
4.  **Application (Docker):** The bot runs as a container pulled from a registry (Docker Hub / GitHub Packages).

---

## Setup & Deployment

### 1. Prerequisites
- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) installed and logged in (`az login`).
- An SSH Public Key (usually at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`).

### 2. Initial Infrastructure Creation
Run the following command to create the VM, the Key Vault, and upload your secrets in one step. This command "pushes" your local configuration files and environment variables to the VM.

```bash
# Create a resource group
az group create --name Hurion-RG-MX --location mexicocentral

# Deploy the Bicep template
az deployment group create \
  --resource-group Hurion-RG-MX \
  --template-file infra/main.bicep \
  --parameters \
    adminPasswordOrKey=@~/.ssh/id_ed25519.pub \
    appEnvContent=@.env \
    dockerEnvContent=@.docker.env \
    dockerImage="your-username/hurion-bot:latest"
```
*(Note: If you do not use a private Docker registry, you can omit the `dockerEnvContent=@.docker.env` line).*

### 3. Deploying Updates (The "One Command" Workflow)
Whenever you push a new Docker Image or update your local `compose.yaml`/`config.yaml`/`.env`, just re-run the deployment command above. 

This will automatically update the files and secrets on the VM, pull the new image, and restart the bot.

```bash
az deployment group create \
  --resource-group Hurion-RG-MX \
  --template-file infra/main.bicep \
  --parameters \
    adminPasswordOrKey=@~/.ssh/id_ed25519.pub \
    appEnvContent=@.env \
    dockerEnvContent=@.docker.env \
    dockerImage="your-username/hurion-bot:latest"
```

---

## Security
- **Managed Identity:** The VM uses a system-assigned identity to authenticate to the Key Vault.
- **SSH Keys:** Password authentication is disabled.
- **Config Injection:** `compose.yaml` and `config.yaml` are injected securely via the Azure Control Plane.

---

## Useful Commands

### SSH into the VM
You can connect to the VM to view logs or debug directly using the IP address from the deployment output:
```bash
ssh -i ~/.ssh/id_ed25519 azureuser@<VM-PUBLIC-IP>
```
*(Once inside, the bot files are located at `/hurion/app`)*


### Troubleshooting: Docker Logs (`IMAGE_NAME` warning)
The `compose.yaml` file relies on the `IMAGE_NAME` variable. Our automated Bicep deployment automatically injects this variable into your `/hurion/app/.env` file. 

However, if you ever manually clear your `.env` file or hit a warning saying `"IMAGE_NAME" variable is not set` when running `sudo docker compose logs -f`, you can fix it by adding it back to the `.env` file:
```bash
sudo bash -c 'echo "IMAGE_NAME=jorgeibanez/hurion-bot:latest" >> /hurion/app/.env'
```
