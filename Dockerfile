# ┌─────────────────────────────────────────────────────────────────────────────┐
# │  gosight-agent/Dockerfile                                                  │
# └─────────────────────────────────────────────────────────────────────────────┘
# gosight-agent/Dockerfile

# ---- Stage 1: Compile gosight-agent binary (Debian‐based with CGO enabled) ----
FROM golang:1.23.7-bookworm AS builder

# Install pkg-config and libsystemd-dev for go-systemd
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      pkg-config \
      libsystemd-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /src/gosight-agent

# Copy only go.mod & go.sum, then strip toolchain and fix go directive
COPY gosight-agent/go.mod gosight-agent/go.sum ./
RUN grep -v '^toolchain ' go.mod > go.mod.fixed && mv go.mod.fixed go.mod
RUN go mod edit -go=1.23

# Download dependencies
RUN go mod download

# Copy entire gosight-agent source (including certs/) 
COPY gosight-agent/ ./

# Build the agent binary with CGO (systemd bindings require it)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /out/gosight-agent ./cmd


# ---- Stage 2: Minimal runtime image ----
FROM debian:bullseye-slim AS runtime

# Install ca-certificates and libsystemd0 if binary links dynamically
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      libsystemd0 && \
    rm -rf /var/lib/apt/lists/*

# Create directories for config and certs
RUN mkdir -p /etc/gosight-agent /etc/certs

# Copy compiled binary
COPY --from=builder /out/gosight-agent /usr/local/bin/gosight-agent

# Copy certs from builder (/src/gosight-agent/certs) into /etc/certs
COPY --from=builder /src/gosight-agent/certs/ /etc/certs/

# (Optional) To bake in config.yaml at build time, uncomment:
COPY --from=builder /src/gosight-agent/config/config.yaml /etc/gosight-agent/config.yaml

ENTRYPOINT ["/usr/local/bin/gosight-agent"]
CMD ["--config", "/etc/gosight-agent/config.yaml"]
