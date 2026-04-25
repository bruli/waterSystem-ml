SHELL := /bin/bash

# ⚙️ Configuration
APP             ?= watersystem-ml
DOCKER_COMPOSE  := COMPOSE_BAKE=true docker compose

DOCKERFILE ?= Dockerfile

PYTHON := python3
VENV := python/.venv
PIP := $(VENV)/bin/pip
PY := $(VENV)/bin/python

.PHONY: help venv install clean train predict shell docker-down docker-exec docker-logs\
 	docker-ps docker-up docker-influxdb-seed fmt security install-lint lint check\
 	docker-login docker-push-image test docker-train docker-predict

.DEFAULT_GOAL := help

DOCKERFILE ?= Dockerfile

IMAGE_REG  ?= ghcr.io/bruli
IMAGE_NAME := $(IMAGE_REG)/$(APP)
VERSION    ?= 0.9.1
CURRENT_IMAGE := $(IMAGE_NAME):$(VERSION)

GOLANGCI_LINT_VERSION ?= v2.11.4

# Crear entorn virtual
venv:
	$(PYTHON) -m venv $(VENV)

# Instal·lar dependències
install: venv
	$(PIP) install --upgrade pip
	$(PIP) install -r python/requirements-dev.txt

# Executar train
train:
	$(PY) python/train.py

# Executar predict
predict:
	$(PY) python/predict.py --json

# Shell amb entorn activat
shell:
	@echo "Activant venv..."
	@bash --rcfile <(echo "source $(VENV)/bin/activate")

# Netejar entorn
clean:
	rm -rf $(VENV)

# ────────────────────────────────────────────────────────────────
# 🐳 Docker
# ────────────────────────────────────────────────────────────────
docker-up:
	@set -euo pipefail; \
	echo "🚀 Starting services with Docker Compose..."; \
	$(DOCKER_COMPOSE) up -d --build

docker-down:
	@set -euo pipefail; \
	echo "🛑 Stopping and removing Docker Compose services..."; \
	$(DOCKER_COMPOSE) down

docker-ps:
	@set -euo pipefail; \
	echo "📋 Active services:"; \
	$(DOCKER_COMPOSE) ps

docker-exec:
	@set -euo pipefail; \
	echo "🔎 Opening shell inside ..."; \
	$(DOCKER_COMPOSE) exec $(APP) sh

docker-influxdb-seed:
	@set -euo pipefail; \
	echo "🔎 Creating data in influxdb ..."; \
	$(DOCKER_COMPOSE) exec $(APP) sh /app/scripts/seed_influxdb.sh

docker-train:
	@set -euo pipefail; \
	echo "🚀 Running train ..."; \
	$(DOCKER_COMPOSE) exec $(APP) /opt/venv/bin/python /app/python/train.py

docker-predict:
	@set -euo pipefail; \
	echo "🚀 Running predict ..."; \
	$(DOCKER_COMPOSE) exec $(APP) /opt/venv/bin/python /app/python/predict.py

docker-logs:
	@set -euo pipefail; \
	echo "👀 Showing logs for container $(APP) (CTRL+C to exit)..."; \
	docker logs -f $(APP)

docker-login:
	echo "🔐 Logging into Docker registry...";
	echo "$$CR_PAT" | docker login ghcr.io -u bruli --password-stdin

docker-push-image: docker-login
	echo "🐳 Building and pushing Docker image $(CURRENT_IMAGE) ...";
	docker buildx build \
      --builder rpi-container-builder \
      --platform linux/arm64 \
      -t $(CURRENT_IMAGE) \
      -f $(DOCKERFILE) \
      --cache-to=type=registry,ref=$(IMAGE_NAME)-buildcache,mode=max \
      --cache-from=type=registry,ref=$(IMAGE_NAME)-buildcache \
      --push .
	 echo "✅ Image $(CURRENT_IMAGE) pushed successfully."

# ────────────────────────────────────────────────────────────────
# 🧹 Code quality: format, lint, tests
# ────────────────────────────────────────────────────────────────
fmt:
	@set -euo pipefail; \
	echo "👉 Formatting code with gofumpt..."; \
	go tool gofumpt -w .

security:
	@set -euo pipefail; \
	echo "👉 Check security"; \
	go tool govulncheck ./...

install-lint:
	@set -euo pipefail; \
    echo "🔧 Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
    	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: install-lint
	@set -euo pipefail; \
	echo "🚀 Executing golangci-lint..."; \
    golangci-lint run ./...

test:
	@set -euo pipefail; \
	echo "🧪 Running unit tests (race, JSON → tparse)..."; \
	go test -race ./... -json -cover -coverprofile=coverage.out| go tool tparse -all

check: fmt security lint test


# ────────────────────────────────────────────────────────────────
# ℹ️ Help
# ────────────────────────────────────────────────────────────────
help:
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:' Makefile | awk -F':' '{print "  - " $$1}'