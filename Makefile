PYTHON := python3
VENV := python/.venv
PIP := $(VENV)/bin/pip
PY := $(VENV)/bin/python

.PHONY: help venv install clean train predict shell

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
# ℹ️ Help
# ────────────────────────────────────────────────────────────────
help:
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:' Makefile | awk -F':' '{print "  - " $$1}'