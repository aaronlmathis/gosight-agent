# Makefile for GoSight builds

#---------------------------------------
# Configuration
#---------------------------------------
# Semantic version (override on CLI: make VERSION=0.1.0-alpha.1)
VERSION ?= dev

# Build metadata
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD)

# ldflags for injecting version info
LDFLAGS := -X 'main.Version=$(VERSION)' \
           -X 'main.BuildTime=$(BUILD_TIME)' \
           -X 'main.GitCommit=$(GIT_COMMIT)'

# Output directories
BIN_DIR := bin
SERVER_OUT := $(BIN_DIR)/gosight-agent

#---------------------------------------
# Environment
#---------------------------------------
# default config path (relative to this Makefile)
GOSIGHT_AGENT_CONFIG ?= ../configs/agent.yaml
export GOSIGHT_AGENT_CONFIG

#---------------------------------------
# Phony targets
#---------------------------------------
.PHONY: all agent fmt test clean

# Default target builds both binaries
all: agent


#---------------------------------------
# Run
#---------------------------------------
.PHONY: run
run: agent
	@echo "Running GoSight agent with GOSIGHT_AGENT_CONFIG=$(GOSIGHT_AGENT_CONFIG)"
	sudo GOSIGHT_AGENT_CONFIG=$(GOSIGHT_AGENT_CONFIG) $(SERVER_OUT)

# Build the GoSight agent
agent	:
	@mkdir -p $(BIN_DIR)
	@echo "Building agent $(VERSION)"
	go build \
		-ldflags "$(LDFLAGS)" \
		-o $(SERVER_OUT) \
		./cmd/

# Format code
fmt:
	go fmt ./...

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR)