FROM golang:1.26.5 AS builder

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build -o /app/bin/watersystem-ml ./cmd/scheduler


FROM python:3.14-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV HOME=/home/watersystem

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        tzdata \
    && rm -rf /var/lib/apt/lists/*

COPY python/requirements.txt /app/python/requirements.txt

RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir -r /app/python/requirements.txt

COPY python /app/python

COPY --from=builder \
    /app/bin/watersystem-ml \
    /app/watersystem-ml

RUN groupadd --gid 10001 watersystem \
    && useradd \
        --uid 10001 \
        --gid 10001 \
        --create-home \
        --home-dir /home/watersystem \
        --shell /usr/sbin/nologin \
        watersystem \
    && chmod 0755 /app/watersystem-ml \
    && chown -R watersystem:watersystem \
        /app \
        /home/watersystem

USER 10001:10001

CMD ["/app/watersystem-ml"]