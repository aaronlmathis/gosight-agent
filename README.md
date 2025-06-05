![Go](https://img.shields.io/badge/built%20with-Go-blue) ![License](https://img.shields.io/github/license/aaronlmathis/gosight-agent) ![Status](https://img.shields.io/badge/status-active-brightgreen) [![Go Report Card](https://goreportcard.com/badge/github.com/aaronlmathis/gosight-agent)](https://goreportcard.com/report/github.com/aaronlmathis/gosight-agent) 

# GoSight Agent

GoSight Agent is a secure, modular telemetry collector written in Go. It gathers system metrics, container statistics, and structured logs from Linux, Windows, and macOS machines — then streams them to the GoSight Server over TLS/mTLS-secured gRPC.



## Documentation
[Documentation](docs/)


## Features

- Fully-featured OpenTelemetry agent for streaming metrics and logs
- Host + container metrics (CPU, memory, disk, network, uptime)
- Log collectors (journald, eventlog, flat files)
- Runtime-safe command execution (optional)
- Tag-based metadata for endpoints and containers
- Streaming metrics + logs via gRPC with auto-reconnect
- Modular collector architecture
- Lightweight, cross-platform binary

## Security

- TLS/mTLS with auto-generated certs
- Agent ID & endpoint metadata
- Secure identity and telemetry labeling

## Directory Overview

- `cmd/` – Main entrypoint for launching the agent
- `docs/` – Developer documentation and integration notes
- `internal/agent/` – Agent lifecycle management
- `internal/bootstrap/` – Startup logic and initialization
- `internal/command/` – Remote command execution handling
- `internal/config/` – Configuration parsing and validation
- `internal/grpc/` – gRPC connection management
- `internal/identity/` – Persistent agent ID generation
- `internal/logs/` – Log collection runners and batching
- `internal/meta/` – Metadata builders for metrics/logs
- `internal/metrics/` – Metric collectors and task runners
- `internal/processes/` – Process snapshot collection
- `internal/protohelper/` – Converters between model and proto
- `internal/utils/` – Logging, error handling, and helpers



## GoSight Agent: Local Build & k3s Deployment

This README section explains how to set up and run the GoSight agent both **locally** (for development or non‐Kubernetes use) and **in k3s** (as a DaemonSet or Deployment). All code blocks have escaped backticks (\`) so you can copy–paste directly.

---

### 1. Prerequisites

- **Go** ≥ 1.20 installed locally (for `make agent`, `make run`, etc.).  
- **Docker** (or Podman) installed (for building images).  
- **k3s** installed and running locally (for Kubernetes deployment).  
  - Ensure `KUBECONFIG=/etc/rancher/k3s/k3s.yaml` or equivalent.  
- **Make** (GNU Make) installed on your system.  
- **Certificates** directory (`certs/`) containing your mTLS files:  
  ```
  certs/
  ├── ca.crt
  ├── server.crt
  └── server.key
  ```
  These will be mounted/baked into your container for mTLS.  
- **Configuration** file: start from `config/example-config.yaml` and copy it to `config/config.yaml`, then fill in the fields:
  ```bash
  cp config/example-config.yaml config/config.yaml
  # Edit config/config.yaml:
  #   - Set server_url, agent.interval, TLS paths, etc.
  ```
  In `config/config.yaml`, any TLS file references should be relative to the `certs/` directory. For example:
  ```yaml
  tls:
    ca_file:    "../certs/ca.crt"
    cert_file:  "../certs/server.crt"
    key_file:   "../certs/server.key"
  ```

The repository layout should look like this:

```
.
├── certs/                           ← your mTLS certificate files
│   ├── ca.crt
│   ├── server.crt
│   └── server.key
├── gosight-agent/
│   ├── Makefile
│   ├── Dockerfile
│   ├── cmd/
│   │   └── main.go
│   ├── config/
│   │   ├── example-config.yaml
│   │   └── config.yaml         ← renamed from example-config.yaml
│   ├── internal/
│   ├── k8s/
│   │   ├── agent-deployment.yaml
│   │   └── agent-daemonset.yaml
│   └── certs/                   ← copy of ../certs/ for Docker context
│       ├── ca.crt
│       ├── server.crt
│       └── server.key
├── README.md
└── other-project-files…
```

> **Note**: We copy `certs/` into `gosight-agent/certs/` to simplify the Docker context. If you keep certificates in the parent folder, adjust your `Makefile` and `Dockerfile` accordingly.

---

### 2. Makefile Targets Overview

Within `gosight-agent/Makefile`, you have the following key targets:

- **Local build & run (non‐Kubernetes):**  
  - `make agent`  
    \> Builds the Go binary into `bin/gosight-agent`.  
  - `make run`  
    \> Runs the binary with `sudo` (mounts `/etc/gosight-agent/config.yaml` and `certs/`).  

- **Docker image & k3s (Kubernetes) deployment:**  
  - `make image` (alias: `docker-build`)  
    \> Builds and tags a Docker image `gosight-agent:$(VERSION)`.  
  - `make load`  
    \> Streams the built image into k3s’s containerd namespace (`k8s.io`).  
  - `make configmap`  
    \> Creates/updates a ConfigMap named `gosight-agent-config` from `config/config.yaml`.  
  - `make deploy-image` (alias: `docker-deploy`)  
    \> Builds image, loads into k3s, updates ConfigMap, and applies **Deployment** manifest.  
  - `make deploy-daemonset` (alias: `docker-deploy-daemonset`)  
    \> Builds image, loads into k3s, updates ConfigMap, and applies **DaemonSet** manifest.  

- **Cleanup:**  
  - `make clean`  
    \> Deletes local `bin/` directory.  
  - `make clean-all`  
    \> Deletes `bin/`, Docker image, ConfigMap, Deployment, DaemonSet, and removes the image from k3s’s containerd.

---

### 3. Local Development (Non-Kubernetes)

1. **Copy & edit the example config**  
   ```bash
   cd gosight-agent
   cp config/example-config.yaml config/config.yaml
   # Now edit config/config.yaml:
   #   - server_url, intervals, TLS paths (relative to ../certs/)
   ```

2. **Build the agent binary**  
   ```bash
   cd gosight-agent
   make agent
   ```
   - Output: `bin/gosight-agent`.

3. **Run the agent locally**  
   ```bash
   cd gosight-agent
   sudo make run
   ```
   - Runs `/bin/gosight-agent --config /etc/gosight-agent/config.yaml`.  
   - **Mount your config**: if you did not bake `config.yaml` into the image, ensure you have a local copy at `config/config.yaml` and that your binary’s default entrypoint reads from `/etc/gosight-agent/config.yaml`. For local run, you can copy:
     ```bash
     sudo mkdir -p /etc/gosight-agent
     sudo cp config/config.yaml /etc/gosight-agent/config.yaml
     sudo cp -r certs/ /etc/gosight-agent/         # so ../certs/ paths resolve
     ```

4. **Verify logs & behavior**  
   - The agent will use your provided `config/config.yaml` and reference certificates at `../certs/`.  
   - Check logs on stdout for any errors.

---

### 4. Building & Deploying to k3s

#### 4.1 Initial Setup

1. **Ensure k3s is running**  
   ```bash
   sudo systemctl start k3s
   export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
   ```
2. **Ensure your user can load images into k3s**  
   - If you haven’t already, either add yourself to the `root` or `containerd` group, or run `make load` with `sudo`.  

#### 4.2 Build & Load the Docker Image

From within `gosight-agent/`:

```bash
cd gosight-agent
make image    # builds `gosight-agent:dev` (or VERSION you set)
make load     # loads the image into k3s containerd namespace (k8s.io)
```

- **`make image`**:  
  \> Runs:
  \```
  docker build -f Dockerfile -t gosight-agent:$(VERSION) ..
  \```  
  where `..` is the parent folder containing both `gosight-agent/` and `certs/`.

- **`make load`**:  
  \> Runs:
  \```
  docker save gosight-agent:$(VERSION) | k3s ctr -n k8s.io images import -
  \```

  Make sure you have permission to write to `/run/k3s/containerd/containerd.sock`. If you get "permission denied," either run as `sudo make load` or adjust your user’s group.

#### 4.3 Create/Update ConfigMap

Create a ConfigMap from your `config/config.yaml` so that pods can mount it at `/etc/gosight-agent/config.yaml`:

```bash
cd gosight-agent
make configmap
```

This runs:

\```
kubectl create configmap gosight-agent-config \
  --from-file=config.yaml=config/config.yaml \
  --namespace=default \
  --dry-run=client -o yaml | kubectl apply -f -
\```

Now Kubernetes has a ConfigMap key `config.yaml` containing your configuration.

#### 4.4 Deploy as a DaemonSet

If you want the agent on **every node** in your k3s cluster:

```bash
cd gosight-agent
make deploy-daemonset
```

- This runs, in order:
  1. `make image`  
  2. `make load`  
  3. `make configmap`  
  4. `kubectl apply -f k8s/agent-daemonset.yaml --namespace=default`

Verify:

```bash
kubectl get daemonset gosight-agent -n default
kubectl get pods -l app=gosight-agent -n default
```

The DaemonSet will start one pod per node, each mounting:

- `/etc/gosight-agent/config.yaml` from the ConfigMap.  
- `/etc/certs/` filled with the files you baked into the image.

Logs:

```bash
kubectl logs -f <pod-name> -n default
```

#### 4.5 (Alternative) Deploy as a Deployment

If you prefer a single‐replica agent (for development):

1. Edit `k8s/agent-deployment.yaml` to set `replicas: 1` (or as desired).  
2. Then run:

   ```bash
   cd gosight-agent
   make deploy-image
   ```

   - This runs, in order:
     1. `make image`
     2. `make load`
     3. `make configmap`
     4. `kubectl apply -f k8s/agent-deployment.yaml --namespace=default`

3. Verify:

   ```bash
   kubectl get deployment gosight-agent-dev -n default
   kubectl get pods -l app=gosight-agent -n default
   ```

4. Tail logs:

   ```bash
   kubectl logs -f <pod-name> -n default
   ```

---

### 5. Comprehensive Cleanup

When you want to remove the agent and all resources:

```bash
cd gosight-agent
make clean-all
```

This will:

1. Remove the `bin/` directory (local Go binary).  
2. Delete the `gosight-agent:$(VERSION)` Docker image locally.  
3. Delete the `gosight-agent-config` ConfigMap from namespace `default`.  
4. Delete both the Deployment (`agent-deployment.yaml`) and DaemonSet (`agent-daemonset.yaml`) from namespace `default`.  
5. Remove the image from k3s’s containerd (`k3s ctr -n k8s.io images rm gosight-agent:$(VERSION)`).

---

### 6. Example Full Workflow

```bash
# 1) Prepare config & certs
cd gosight-agent
cp config/example-config.yaml config/config.yaml
# Edit config/config.yaml as needed (server_url, TLS paths → ../certs/)

# 2) Local build & run
make agent
make run

# 3) Build image & load into k3s
make image
make load

# 4) Create/update ConfigMap & deploy DaemonSet
make configmap
make deploy-daemonset

# 5) Verify in k3s
kubectl get daemonset gosight-agent -n default
kubectl get pods -l app=gosight-agent -n default
kubectl logs -f <gosight-agent-pod-name> -n default

# 6) (Optional) Tear everything down
make clean-all
```


## License

GPL-3.0-or-later
