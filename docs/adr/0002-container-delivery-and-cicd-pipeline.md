# ADR 0002: Container Delivery, CI/CD Pipeline & GitHub Container Registry Migration

* **Status**: Accepted
* **Date**: 2026-08-23

---

## 1. Context and Problem Statement

Hurion Bot runs as a production service on an Azure B-Series Virtual Machine (`Standard_B1s`: 1 vCPU, 1 GB RAM) orchestrated via Docker Compose and Litestream replication.

Previously, deployment and verification faced two primary operational pain points:
1. **Manual Compilation on Developer Workstations**: Testing and publishing Docker containers required a developer to manually run `docker build` and `docker push` on their local laptop, consuming local CPU/RAM and residential upload bandwidth.
2. **Pull Request Quality Verification Gap**: Pull Requests lacked automated, reproducible continuous integration (CI) test runs in isolated environments across the Go backend, Web UI frontend, and container build configurations.

### Constraints & Goals
1. **Resource Guardrails on Azure VM**: The production VM operates with 1 GB of RAM and burstable CPU credits. The deployment process must impose near-zero CPU and memory overhead on the VM to prevent Out-Of-Memory (OOM) killer terminations or WebSocket heartbeat drops.
2. **Zero-Inbound Attack Surface**: The VM operates without a dedicated public IPv4 address and has no open inbound SSH port (port 22). Deployments must occur strictly via the encrypted Azure Control Plane (`az vm run-command invoke`) using outbound HTTPS connectivity.
3. **Secret Hygiene**: CI workflows for public Pull Requests must run with zero credentials, and production deployment must avoid long-lived static passwords.
4. **Parity Between Local Development and CI**: Local developer commands (`make test`) must be the exact same entrypoint executed by automated CI runners.

---

## 2. Decision Drivers

* **Reliability & Uptime**: Prevent build-time memory exhaustion on the host VM during updates.
* **Security Posture**: Maintain zero inbound open ports on the VM and eliminate long-lived cloud credentials.
* **Fast, Atomic Deployments**: Minimize container restart downtime to <3 seconds while preserving the SQLite database on host-mounted storage.
* **Developer Experience**: Provide single-command local testing and automatic verification on every Git Pull Request.

---

## 3. Considered Alternatives & Why Rejected

### Alternative A: Compiling Source Code on the Production VM (`git pull` + Build)
* **Concept**: Have the Azure VM clone the Git repository, pull changes, and execute `go build` or `docker build` locally on the host.
* **Why Rejected**:
  * Compiling Go with CGO (SQLite bindings) and multi-stage Docker builds consumes >800 MB to 1.2 GB of RAM during compilation. On a 1 GB RAM VM, this triggers the Linux OOM Killer, terminating the bot process or freezing the server.
  * Heavy compilation exhausts Azure B-Series CPU burst credits, throttling CPU speed down to 10–20% and causing Discord gateway heartbeat disconnects.
  * Requires installing developer tooling (`gcc`, `git`, Go SDK) on the production server, expanding the security attack surface.

### Alternative B: Direct Tarball Streaming (`docker save` -> SCP/Pipe -> `docker load`)
* **Concept**: Export the entire Docker image as a raw `.tar` archive in CI and stream it directly into the VM over an SSH pipe.
* **Why Rejected**:
  * The Azure VM has no public IP and no open SSH port; incoming network connections from GitHub Actions runners are blocked by design.
  * Exporting uncompressed tarballs transfers the entire ~300–400 MB payload on every deployment, whereas container registries transfer only the modified ~25 MB delta layer.
  * Decompressing large tarballs directly in memory via `docker load` thrashes RAM on 1 GB VMs.

### Alternative C: Docker Hub (`docker.io`)
* **Concept**: Continue pushing images to Docker Hub using personal username and access token secrets in GitHub Actions.
* **Why Rejected**:
  * Requires creating and maintaining personal API tokens and third-party secrets.
  * Docker Hub enforces rate limits on image pulls.
  * GitHub Container Registry (`ghcr.io`) provides native integration with GitHub Actions using ephemeral, auto-rotating `GITHUB_TOKEN` credentials at zero cost.

---

## 4. Decision Outcome & Architecture

We decided to adopt a **Two-Tier Automated CI/CD Pipeline** using **GitHub Actions**, **GitHub Container Registry (`ghcr.io`)**, and **Azure Control Plane Remote Invocation**:

### 1. Continuous Integration (`.github/workflows/ci.yml`)
Runs on every Pull Request and branch push to `main` with zero credentials:
* **Backend Job**: Sets up Go 1.25 and executes `make test` (invoking `go generate`, unit tests, integration tests, workflow tests, and web server tests with race detection).
* **Frontend Job**: Sets up Node 22 + pnpm, executes strict TypeScript type checking (`tsc -b`), and compiles the React 19 Vite bundle (`pnpm --dir web check`).
* **Container Job**: Performs a dry-run build of the root `Dockerfile` to guarantee multi-stage build validity.

### 2. Immutable Container Delivery (`ghcr.io`)
Images are compiled in GitHub Actions on high-spec runners (16 GB RAM) and published to GitHub Container Registry:
* `ghcr.io/jorgeiba/hurion:latest`
* `ghcr.io/jorgeiba/hurion:sha-<commit>`

### 3. Continuous Deployment (`.github/workflows/deploy.yml`)
Triggered manually on demand via `workflow_dispatch` with maintainer access:
1. Accepts an optional `image_tag` parameter (default: `latest`, or `sha-<commit>` for instant rollbacks).
2. For new releases (`latest`), builds and publishes multi-tag container images to `ghcr.io`. For rollback tags (`sha-*`), skips rebuilding and deploys the existing image directly.
3. Authenticates to Azure via `azure/login` using service principal credentials (`AZURE_VM_DEPLOY_CREDENTIALS` / `AZURE_CREDENTIALS`) or federated OIDC (`AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`).
4. Invokes remote update on the VM via Azure Control Plane, injecting the ephemeral `GITHUB_TOKEN` in flight for zero-disk-credential security:
   ```bash
   VM_SCRIPT=$(cat <<EOF
   set -euo pipefail
   cleanup() {
     sudo docker logout ghcr.io || true
   }
   trap cleanup EXIT

   cd /hurion/app
   echo "$GH_TOKEN" | sudo docker login ghcr.io -u "$GH_ACTOR" --password-stdin
   sudo IMAGE_NAME="$IMAGE_URI" docker compose pull
   sudo IMAGE_NAME="$IMAGE_URI" docker compose up -d
   EOF
   )
   ```
5. Docker on the VM downloads only the changed layer from `ghcr.io` in ~1.5 seconds, immediately purges registry credentials from memory with `docker logout`, and recreates the container without data loss.

---

## 5. Security & Maintenance Rules

1. **Local Parity**: Developers can run `make test` locally to execute the exact same test matrix as CI.
2. **Local Registry Access**: Developers wishing to push manually to `ghcr.io` can authenticate via GitHub CLI:
   ```bash
   gh auth token | docker login ghcr.io -u <username> --password-stdin
   ```
3. **Database Volume Protection**: The production database path `/hurion/data` is mounted from the VM host and is never modified or erased during container image swaps.
