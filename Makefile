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
AGENT_OUT := $(BIN_DIR)/gosight-agent


# Docker / Kubernetes settings
IMAGE_NAME    := gosight-agent
IMAGE_TAG     := $(VERSION)
DOCKERFILE    := Dockerfile

# Note: this Makefile lives inside gosight-agent/, so the build context is the parent directory (..)
BUILD_CTX     := ..

# Paths to k8s manifests (relative to repo root)
K8S_DEPLOYMENT_MANIFEST := k8s/agent-deployment.yaml
K8S_DAEMONSET_MANIFEST  := k8s/agent-daemonset.yaml


#───────────────────────────────────────────────────────────────────────────────
# Variables (if not already defined)
#───────────────────────────────────────────────────────────────────────────────
K8S_NAMESPACE     := default
CONFIGMAP_NAME    := gosight-agent-config
LOCAL_CONFIG_PATH := config/config.yaml

# Containerd namespace for k3s (default)
CONTAINERD_NS := k8s.io

#---------------------------------------
# Phony targets
#---------------------------------------
.PHONY: all agent fmt test clean \
        run \
        image docker-build \
        deploy-image docker-deploy

# Default target builds the agent binary
all: agent

#---------------------------------------
# Run (local binary)
#---------------------------------------
.PHONY: run
run: agent
	@echo "▶ Running GoSight agent $(VERSION)"
	sudo $(AGENT_OUT)

# Build the GoSight agent binary
agent:
	@mkdir -p $(BIN_DIR)
	@echo "▶ Building gosight-agent binary (version=$(VERSION))"
	go build \
		-ldflags="$(LDFLAGS)" \
		-o $(AGENT_OUT) \
		./cmd/

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Run tests
.PHONY: test
test:
	go test ./...


#---------------------------------------
# Docker image build
#---------------------------------------
# Build and tag a Docker image for the agent
.PHONY: image docker-build
image docker-build:
	@echo "▶ Building Docker image $(IMAGE_NAME):$(IMAGE_TAG)"
	@docker build \
		-f $(DOCKERFILE) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		$(BUILD_CTX)



#───────────────────────────────────────────────────────────────────────────────
# Create or update the ConfigMap from config/config.yaml
#───────────────────────────────────────────────────────────────────────────────
.PHONY: configmap
configmap:
	@echo "▶ Creating/updating ConfigMap $(CONFIGMAP_NAME) in namespace $(K8S_NAMESPACE)"
	kubectl create configmap $(CONFIGMAP_NAME) \
	--from-file=config.yaml=$(LOCAL_CONFIG_PATH) \
	--namespace=$(K8S_NAMESPACE) \
	--dry-run=client -o yaml | \
	kubectl apply -f -


#---------------------------------------
# Kubernetes deploy (k3s)
#---------------------------------------
# Load the Docker image into k3s containerd
.PHONY: load
load:
	@echo "▶ Loading image into k3s containerd namespace ($(CONTAINERD_NS))"
	@docker save $(IMAGE_NAME):$(IMAGE_TAG) | \
		k3s ctr -n $(CONTAINERD_NS) images import -

# Deploy the new image to k3s by applying the manifest
.PHONY: deploy-image docker-deploy
deploy-image docker-deploy: image load
	@echo "▶ Applying Kubernetes manifest: $(K8S_MANIFEST)"
	@kubectl apply -f $(K8S_MANIFEST)

# Deploy the new image to k3s using DaemonSet manifest
.PHONY: deploy-daemonset docker-deploy-daemonset
deploy-daemonset docker-deploy-daemonset: image load
	@echo "▶ Applying DaemonSet manifest: $(K8S_DAEMONSET_MANIFEST)"
	@kubectl apply -f $(K8S_DAEMONSET_MANIFEST)

#---------------------------------------
# Comprehensive clean
#---------------------------------------
.PHONY: clean clean-all
clean:
	@echo "▶ Cleaning local build artifacts..."
	@rm -rf $(BIN_DIR)

clean-all: clean
	@echo "▶ Deleting Docker image $(IMAGE_NAME):$(IMAGE_TAG)..."
	-@docker rmi $(IMAGE_NAME):$(IMAGE_TAG) || true

	@echo "▶ Deleting ConfigMap $(CONFIGMAP_NAME) in namespace $(K8S_NAMESPACE)..."
	-@kubectl delete configmap $(CONFIGMAP_NAME) --namespace=$(K8S_NAMESPACE) --ignore-not-found

	@echo "▶ Deleting Deployment and DaemonSet in namespace $(K8S_NAMESPACE)..."
	-@kubectl delete -f $(K8S_DEPLOYMENT_MANIFEST) --namespace=$(K8S_NAMESPACE) --ignore-not-found
	-@kubectl delete -f $(K8S_DAEMONSET_MANIFEST)  --namespace=$(K8S_NAMESPACE) --ignore-not-found

	@echo "▶ Cleaning k3s containerd image store..."
	-@k3s ctr -n $(CONTAINERD_NS) images rm $(IMAGE_NAME):$(IMAGE_TAG) || true

	@echo "▶ Clean finished."
