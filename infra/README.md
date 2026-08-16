# Infrastructure as Code: Azure VM Hosting

This directory contains the Bicep templates and provisioning scripts to host the application on an **Azure B-Series Virtual Machine**.

## Architecture Layers
1. **Hardware & Networking (Bicep):** `main.bicep` defines the VM (Standard_B1s), Virtual Network, Public IP, and Network Security Group (Firewall).
2. **Encrypted Provisioning (`protectedSettings`):** Secrets (`.env`) and configuration files are passed via ARM's encrypted `protectedSettings` channel directly into the VM guest agent.
3. **Application (Docker):** The bot and Litestream run as supervised containers orchestrated by Docker Compose.

---

## Setup & Deployment

### 1. Prerequisites
- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) installed and logged in (`az login`).
- An SSH Public Key (usually at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`).
- A local `.env` and `config.yaml` configured with your credentials.

### 2. Deployment (The "One Command" Workflow)
Run the following command from the repository root to create or update the VM, push configuration, and start the bot:

```bash
az deployment group create \
  --resource-group Hurion-RG-MX \
  --template-file infra/main.bicep \
  --parameters \
    adminPasswordOrKey=@~/.ssh/id_ed25519.pub \
    appEnvContent=@.env \
    dockerImage="jorgeibanez/hurion-bot:latest"
```

*(Note: If you use a private Docker registry, add `dockerEnvContent=@.docker.env`)*.

---

## Security
- **Encrypted in Transit:** `protectedSettings` encrypts sensitive variables before deployment, decrypting them strictly inside the VM.
- **SSH Keys Only:** Password authentication is disabled.
- **Network Isolation:** Only port 22 (SSH) is opened; all Discord and Azure Storage traffic is initiated outbound over HTTPS.

---

## Useful Commands

### SSH into the VM
```bash
ssh -i ~/.ssh/id_ed25519 azureuser@<VM-PUBLIC-IP>
```
*(Inside the VM, application files live at `/hurion/app`)*.

### View Live Bot Logs
```bash
cd /hurion/app
sudo docker compose logs -f
```
