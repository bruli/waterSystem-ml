SHELL := /bin/bash

# ⚙️ Configuration
APP             ?= watersystem-ml
DOCKER_COMPOSE  := COMPOSE_BAKE=true docker compose

DOCKERFILE ?= Dockerfile

PYTHON := python3
VENV := python/.venv
PIP := $(VENV)/bin/pip
PY := $(VENV)/bin/python

.PHONY: help venv install clean train predict shell docker-down docker-exec docker-logs docker-ps docker-up docker-influxdb-seed

.DEFAULT_GOAL := help

# Crear entorn virtual
venv:
	$(PYTHON) -m venv $(VENV)

# Instal·lar dependències
install: venv
	$(PIP) install --upgrade pip
	$(PIP) install -r python/requirements.txt

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

docker-logs:
	@set -euo pipefail; \
	echo "👀 Showing logs for container $(APP) (CTRL+C to exit)..."; \
	docker logs -f $(APP)

# ────────────────────────────────────────────────────────────────
# ℹ️ Help
# ────────────────────────────────────────────────────────────────
help:
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:' Makefile | awk -F':' '{print "  - " $$1}'