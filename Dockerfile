FROM golang:1.26.1 AS go-builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/scheduler ./cmd/scheduler

FROM python:3.12-slim

WORKDIR /app

RUN python -m venv /app/venv
ENV PATH="/app/venv/bin:$PATH"

COPY python/requirements.txt /app/python/requirements.txt
RUN pip install --no-cache-dir -r /app/python/requirements.txt

COPY python /app/python
COPY --from=go-builder /out/scheduler /app/scheduler

ENV PYTHONUNBUFFERED=1

CMD ["/app/scheduler"]