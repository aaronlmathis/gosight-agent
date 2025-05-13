![Go](https://img.shields.io/badge/built%20with-Go-blue) ![License](https://img.shields.io/github/license/aaronlmathis/gosight-agent) ![Status](https://img.shields.io/badge/status-active-brightgreen)

# GoSight Agent

GoSight Agent is a secure, modular telemetry collector written in Go. It gathers system metrics, container statistics, and structured logs from Linux, Windows, and macOS machines — then streams them to the GoSight Server over TLS/mTLS-secured gRPC.

## Features

- Host + container metrics (CPU, memory, disk, network, uptime)
- Log collectors (journald, syslog, flat files)
- Runtime-safe command execution (optional)
- Tag-based metadata for endpoints and containers
- Streaming metrics + logs via gRPC with auto-reconnect
- Modular collector architecture
- Lightweight, cross-platform binary

## Build

\`\`\`bash
go build -o gosight-agent ./cmd
\`\`\`

## Configuration

Configure using \`config.yaml\`, environment variables, or command-line flags. See example config in \`./agent/config\`.

## Running

\`\`\`bash
./gosight-agent --config ./config.yaml
\`\`\`

## Security

- TLS/mTLS with auto-generated certs
- Agent ID & endpoint metadata
- Secure identity and telemetry labeling

## Directory Overview

- \`internal/\` – core agent logic and collectors
- \`metrics/\` – metric collectors and sender
- \`logs/\` – log runners and batching
- \`proto/\` – gRPC models (imported from gosight-shared)
- \`cmd/\` – agent entrypoint

## License

GPL-3.0-or-later
