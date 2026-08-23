# Infrastructure as Code: Azure VM Hosting & Cost-Optimized Operations

This directory contains the Bicep templates, provisioning scripts, and operational runbooks to host the Hurion Bot on an **Azure B-Series Virtual Machine** using a **cost-optimized and secure networking architecture**.

---

## Architecture & Cost Optimization

### 1. Reducing Networking Costs (No Public IPv4 Required)
Across major cloud providers (Microsoft Azure, AWS, GCP), dedicated Public IPv4 addresses incur a persistent hourly surcharge (approximately $0.005 per hour, or ~3.65 to 4.00 USD per month per IP) regardless of whether the VM is actively running or idle.

Because the Hurion Bot architecture only relies on outbound connections, we can avoid public IPv4 address charges entirely:

- **Discord Bot Gateway (Outbound WebSockets):** The bot initiates outbound TLS WebSocket connections (`wss://gateway.discord.gg`). Discord never sends inbound traffic to the bot's IP.
- **Litestream & Scoreboard Exporter (Outbound HTTPS):** Database backups and scoreboard JSON compilations are pushed outbound to Azure Blob Storage over HTTPS (port 443).
- **Scoreboard Web UI (Azure Static Web Apps):** The React 19 web frontend is delivered via Azure Static Web Apps CDN and fetches static JSONs directly from Azure Blob Storage. Web users never connect to the Azure VM.

### 2. Architecture Layers
1. **Hardware & Networking (`infra/main.bicep`):** Provisions the VM (`Standard_B1s`), Virtual Network, and Network Security Group. By default, `enablePublicIP: false` is configured to avoid public IPv4 surcharges.
2. **Encrypted Provisioning (`protectedSettings`):** Secrets (`.env`) and configuration files (`config.yaml`) are passed through ARM's encrypted `protectedSettings` channel directly into the VM guest agent.
3. **Application Layer (Docker Compose):** The bot and Litestream execute as supervised containers orchestrated by Docker Compose.

### 3. Outbound Internet Connectivity (Egress)
Virtual machines deployed in standard Azure Virtual Networks utilize Azure's default outbound routing (Default SNAT) to reach internet endpoints (Discord Gateway, Azure Storage, Docker Hub) over HTTPS without requiring a dedicated public IP address or an expensive Azure NAT Gateway (~$32.40 USD/month).

If your environment or cloud policy requires an explicit outbound IP address, deploy the template with `--parameters enablePublicIP=true`.

---

## Setup & Deployment

### 1. Prerequisites
- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) installed and authenticated (`az login`).
- An SSH Public Key (usually at `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`).
- Local `.env` and `config.yaml` configured with required tokens and role IDs.

### 2. Deployment Workflow
Run the following command from the repository root to deploy or update the VM without a Public IP:

```bash
az deployment group create \
  --resource-group Hurion-RG-MX \
  --template-file infra/main.bicep \
  --parameters \
    adminPasswordOrKey=@~/.ssh/id_ed25519.pub \
    appEnvContent=@.env \
    dockerImage="jorgeibanez/hurion-bot:latest"
```

*(Optional: Pass `enablePublicIP=true` if you explicitly require a public IPv4 address for direct SSH)*.

### 3. Updating Existing Deployments (Removing Orphaned Public IPs)
In Azure Resource Manager (ARM), default deployments run in `Incremental` mode. If you previously deployed the infrastructure with a Public IP and re-deploy with `enablePublicIP=false`, ARM will detach the public IP from the VM network interface, but the orphaned public IP resource will remain in your resource group.

To permanently delete the unassociated public IP and avoid idle IPv4 charges, run:
```bash
az network public-ip delete --resource-group Hurion-RG-MX --name hurion-botPublicIP
```

---

## Remote VM Management via Azure CLI (No SSH Required)

Because the VM operates without a public IP, management operations are executed securely via the Azure Control Plane using `az vm run-command invoke`. This eliminates the need for open inbound firewall ports.

### 1. Inspect Running Containers & Status
```bash
az vm run-command invoke \
  --resource-group Hurion-RG-MX \
  --name hurion-bot \
  --command-id RunShellScript \
  --scripts "docker ps && uptime"
```

### 2. View Live Bot Logs
```bash
az vm run-command invoke \
  --resource-group Hurion-RG-MX \
  --name hurion-bot \
  --command-id RunShellScript \
  --scripts "cd /hurion/app && sudo docker compose logs --tail=50"
```

### 3. Pull Latest Code & Restart Containers
```bash
az vm run-command invoke \
  --resource-group Hurion-RG-MX \
  --name hurion-bot \
  --command-id RunShellScript \
  --scripts "cd /hurion/app && sudo docker compose pull && sudo docker compose up -d"
```

### 4. Check Disk & Memory Usage
```bash
az vm run-command invoke \
  --resource-group Hurion-RG-MX \
  --name hurion-bot \
  --command-id RunShellScript \
  --scripts "df -h / && free -m"
```

---

## Security & Cost Summary

- **Zero Inbound Attack Surface:** Without a public IP, internet port scanners cannot reach port 22 or any internal port.
- **Encrypted in Transit:** Sensitive variables in `.env` and `config.yaml` are encrypted at the Azure ARM layer before provisioning.
- **Reduced Overhead:** Disabling unused public IPs avoids unnecessary recurring IPv4 address fees.

---

## Emergency Interactive Access (On-Demand)

If an interactive terminal session is ever required for deep troubleshooting:

1. **Attach a Temporary Public IP:**
   ```bash
   # Create a temporary public IP
   az network public-ip create -g Hurion-RG-MX -n temp-ssh-ip --sku Standard

   # Associate to the VM NIC
   az network nic ip-config update -g Hurion-RG-MX --nic-name hurion-botNetInt --name ipconfig1 --public-ip-address temp-ssh-ip
   ```

2. **SSH into the VM:**
   ```bash
   ssh -i ~/.ssh/id_ed25519 azureuser@<TEMP-PUBLIC-IP>
   ```

3. **Delete the IP When Done (Stop Recurring Billing):**
   ```bash
   az network nic ip-config update -g Hurion-RG-MX --nic-name hurion-botNetInt --name ipconfig1 --remove publicIpAddress
   az network public-ip delete -g Hurion-RG-MX -n temp-ssh-ip
   ```
