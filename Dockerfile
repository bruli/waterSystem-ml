
FROM golang:1.26.1 AS builder

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/watersystem-ml ./cmd/scheduler


# =========================================================
# Runtime
# =========================================================
FROM python:3.12-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV PATH="/app/venv/bin:$PATH"

# Paquets mínims útils
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Entorn virtual de Python
RUN python -m venv /app/venv

# Instal·lar dependències Python
COPY python/requirements.txt /app/python/requirements.txt
RUN pip install --no-cache-dir -r /app/python/requirements.txt

# Copiar scripts i codi Python
COPY python /app/python

# Copiar binari de Go des de la fase builder
COPY --from=builder /app/bin/watersystem-ml /app/watersystem-ml

# Donar permisos d'execució
RUN chmod +x /app/watersystem-ml

CMD ["/app/watersystem-ml"]