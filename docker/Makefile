# =============================================================================
# flow-cli Docker Makefile
# Simplifies common Docker operations
# =============================================================================

# Configuration
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
IMAGE_NAME ?= flow-cli
REGISTRY ?=

# Docker compose file
COMPOSE_FILE := docker-compose.yml

# Export for docker compose
export VERSION COMMIT BUILD_DATE

.PHONY: help build build-gpu build-minimal up up-gpu up-sidecar down \
        logs shell pull-model list-models clean prune test health

# Default target
help:
	@echo "flow-cli Docker Commands"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build         Build CPU production image"
	@echo "  make build-gpu     Build GPU-enabled image"
	@echo "  make build-minimal Build minimal CLI-only image"
	@echo "  make build-all     Build all image variants"
	@echo ""
	@echo "Run Commands:"
	@echo "  make up            Start CPU container (all-in-one)"
	@echo "  make up-gpu        Start GPU container (all-in-one)"
	@echo "  make up-sidecar    Start sidecar mode (separate Ollama)"
	@echo "  make up-search     Start with SearXNG search"
	@echo "  make down          Stop all containers"
	@echo "  make restart       Restart containers"
	@echo ""
	@echo "Interactive Commands:"
	@echo "  make chat          Start interactive chat session"
	@echo "  make shell         Open shell in container"
	@echo "  make attach        Attach to running container"
	@echo ""
	@echo "Model Commands:"
	@echo "  make list-models   List available models"
	@echo "  make pull-model MODEL=llama3.2  Pull a model"
	@echo "  make preload       Pull recommended models"
	@echo ""
	@echo "Maintenance Commands:"
	@echo "  make logs          View container logs"
	@echo "  make health        Check container health"
	@echo "  make stats         Show resource usage"
	@echo "  make clean         Remove containers and images"
	@echo "  make prune         Remove unused Docker resources"
	@echo "  make backup        Backup models volume"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  MODEL=$(MODEL)"
	@echo "  PROJECT_DIR=$(PROJECT_DIR)"

# -----------------------------------------------------------------------------
# Build Commands
# -----------------------------------------------------------------------------

build:
	@echo "Building CPU production image..."
	docker compose -f $(COMPOSE_FILE) build flow \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE)

build-gpu:
	@echo "Building GPU-enabled image..."
	docker compose -f $(COMPOSE_FILE) build flow-gpu \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE)

build-minimal:
	@echo "Building minimal CLI-only image..."
	docker compose -f $(COMPOSE_FILE) build flow-sidecar \
		--build-arg VERSION=$(VERSION)

build-all: build build-gpu build-minimal
	@echo "All images built successfully"

# -----------------------------------------------------------------------------
# Run Commands
# -----------------------------------------------------------------------------

up:
	@echo "Starting flow-cli (CPU mode)..."
	docker compose -f $(COMPOSE_FILE) up -d flow
	@echo ""
	@echo "Container started. Attach with: make attach"
	@echo "Or run commands with: make chat"

up-gpu:
	@echo "Starting flow-cli (GPU mode)..."
	docker compose -f $(COMPOSE_FILE) --profile gpu up -d flow-gpu
	@echo ""
	@echo "GPU container started. Attach with: docker attach flow-cli-gpu"

up-sidecar:
	@echo "Starting flow-cli in sidecar mode..."
	docker compose -f $(COMPOSE_FILE) --profile sidecar up -d
	@echo ""
	@echo "Sidecar mode started."

up-sidecar-gpu:
	@echo "Starting flow-cli in GPU sidecar mode..."
	docker compose -f $(COMPOSE_FILE) --profile sidecar-gpu up -d

up-search:
	@echo "Starting flow-cli with SearXNG search..."
	docker compose -f $(COMPOSE_FILE) --profile search up -d flow searxng

down:
	@echo "Stopping all containers..."
	docker compose -f $(COMPOSE_FILE) --profile gpu --profile sidecar --profile sidecar-gpu --profile search down

restart: down up

# -----------------------------------------------------------------------------
# Interactive Commands
# -----------------------------------------------------------------------------

chat:
	@echo "Starting interactive chat..."
	docker compose -f $(COMPOSE_FILE) run --rm flow chat

run:
	@if [ -z "$(PROMPT)" ]; then \
		echo "Usage: make run PROMPT=\"your prompt here\""; \
		exit 1; \
	fi
	docker compose -f $(COMPOSE_FILE) run --rm flow run "$(PROMPT)"

arch:
	@echo "Starting architecture mode..."
	docker compose -f $(COMPOSE_FILE) run --rm flow arch

shell:
	@echo "Opening shell in container..."
	docker compose -f $(COMPOSE_FILE) exec flow /bin/bash || \
		docker compose -f $(COMPOSE_FILE) run --rm flow shell

attach:
	@echo "Attaching to flow-cli container..."
	@echo "(Press Ctrl+P, Ctrl+Q to detach without stopping)"
	docker attach flow-cli

# -----------------------------------------------------------------------------
# Model Commands
# -----------------------------------------------------------------------------

list-models:
	@docker compose -f $(COMPOSE_FILE) exec flow ollama list 2>/dev/null || \
		echo "Container not running. Start with: make up"

pull-model:
	@if [ -z "$(MODEL)" ]; then \
		echo "Usage: make pull-model MODEL=llama3.2"; \
		exit 1; \
	fi
	@echo "Pulling model: $(MODEL)"
	docker compose -f $(COMPOSE_FILE) exec flow ollama pull $(MODEL)

preload:
	@echo "Pulling recommended models..."
	docker compose -f $(COMPOSE_FILE) exec flow ollama pull llama3.2
	docker compose -f $(COMPOSE_FILE) exec flow ollama pull codellama:7b
	@echo "Models preloaded successfully"

# -----------------------------------------------------------------------------
# Maintenance Commands
# -----------------------------------------------------------------------------

logs:
	docker compose -f $(COMPOSE_FILE) logs -f

logs-ollama:
	docker compose -f $(COMPOSE_FILE) logs -f | grep -i ollama

health:
	@echo "=== Container Status ==="
	@docker compose -f $(COMPOSE_FILE) ps
	@echo ""
	@echo "=== Health Check ==="
	@curl -s http://localhost:11434/api/tags >/dev/null 2>&1 && \
		echo "Ollama API: HEALTHY" || echo "Ollama API: UNHEALTHY"
	@echo ""
	@echo "=== Resource Usage ==="
	@docker stats --no-stream flow-cli 2>/dev/null || echo "Container not running"

stats:
	docker stats flow-cli

version:
	docker compose -f $(COMPOSE_FILE) exec flow flow version

clean:
	@echo "Removing containers..."
	docker compose -f $(COMPOSE_FILE) --profile gpu --profile sidecar --profile sidecar-gpu --profile search down -v --remove-orphans
	@echo "Removing images..."
	docker rmi $(IMAGE_NAME):latest $(IMAGE_NAME):gpu $(IMAGE_NAME):minimal 2>/dev/null || true
	@echo "Clean complete"

prune:
	@echo "Pruning unused Docker resources..."
	docker system prune -f
	@echo "Prune complete"

# Backup models volume
backup:
	@echo "Backing up Ollama models..."
	@mkdir -p backups
	docker run --rm \
		-v flow-ollama-models:/data \
		-v $(PWD)/backups:/backup \
		alpine tar czf /backup/ollama-models-$(shell date +%Y%m%d).tar.gz -C /data .
	@echo "Backup saved to backups/ollama-models-$(shell date +%Y%m%d).tar.gz"

# Restore models volume
restore:
	@if [ -z "$(BACKUP)" ]; then \
		echo "Usage: make restore BACKUP=backups/ollama-models-YYYYMMDD.tar.gz"; \
		exit 1; \
	fi
	@echo "Restoring Ollama models from $(BACKUP)..."
	docker run --rm \
		-v flow-ollama-models:/data \
		-v $(PWD):/backup \
		alpine sh -c "rm -rf /data/* && tar xzf /backup/$(BACKUP) -C /data"
	@echo "Restore complete"

# -----------------------------------------------------------------------------
# Testing
# -----------------------------------------------------------------------------

test:
	@echo "Running container tests..."
	@echo "1. Testing build..."
	docker compose -f $(COMPOSE_FILE) build flow
	@echo ""
	@echo "2. Testing startup..."
	docker compose -f $(COMPOSE_FILE) up -d flow
	@sleep 10
	@echo ""
	@echo "3. Testing health..."
	@curl -sf http://localhost:11434/api/tags >/dev/null && echo "Health check: PASS" || echo "Health check: FAIL"
	@echo ""
	@echo "4. Testing CLI..."
	docker compose -f $(COMPOSE_FILE) exec flow flow version
	@echo ""
	@echo "5. Cleanup..."
	docker compose -f $(COMPOSE_FILE) down
	@echo ""
	@echo "All tests completed"
