FROM golang:1.26.1 AS builder

WORKDIR /app
ENV GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /app/bin/watersystem-ml ./cmd/scheduler


FROM python:3.12-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

COPY python/requirements.txt /app/python/requirements.txt

RUN --mount=type=cache,target=/root/.cache/pip \
    pip install -r /app/python/requirements.txt

COPY python /app/python
COPY --from=builder /app/bin/watersystem-ml /app/watersystem-ml

RUN chmod +x /app/watersystem-ml

CMD ["/app/watersystem-ml"]