# Loglite

Loglite is a compact Go log aggregator for owned services. It accepts HTTP and UDP syslog events, redacts common secrets before storage, indexes log text and labels in memory, applies retention, exposes a query API, and ships with a built-in web UI.

This implements the Notion project **"Лёгкий Log Aggregator"** without Python.

## Features

- HTTP ingestion: single JSON event, JSON array, or NDJSON.
- UDP syslog ingestion with RFC3164-like parsing.
- Labels and fields on every event.
- Full-text search over message, labels, fields, level, and source.
- Query filters for `level`, `label`, `since`, and `limit`.
- Secret redaction before persistence.
- Append-only JSONL storage with retention pruning.
- Built-in web query UI.
- Prometheus text metrics.
- Docker image and docker-compose demo with generated logs.

## Run

```powershell
go run ./cmd/loglite --listen :8080 --syslog :5514 --store data/loglite.jsonl --retention 24h
```

Open:

```text
http://localhost:8080
```

## Ingest Logs

HTTP:

```powershell
curl.exe -X POST http://localhost:8080/api/logs `
  -H "content-type: application/json" `
  -d "{\"level\":\"error\",\"message\":\"login failed password=hunter2\",\"labels\":{\"service\":\"auth\",\"env\":\"dev\"}}"
```

Syslog:

```powershell
"<13>Jun 21 10:11:12 dev-host sshd[42]: Failed password for root" | nc -u -w1 127.0.0.1 5514
```

## Search

```powershell
curl.exe "http://localhost:8080/api/search?q=failed&label=service=auth&level=error&since=1h&limit=50"
```

## Metrics

```powershell
curl.exe http://localhost:8080/api/metrics
```

## Docker Demo

```powershell
docker compose -f examples/docker-compose.yml up --build
```

The compose demo starts Loglite, an HTTP log generator, and a UDP syslog generator.

## Development

```powershell
go test ./...
go build -o bin/loglite.exe ./cmd/loglite
```

## Safety Scope

Loglite is for logs from your own systems and authorized lab services. Redaction is enabled before storage, but it is still best practice to avoid sending raw production secrets into any logging pipeline.

## Project Structure

```text
cmd/loglite       CLI entry point
internal/model    Log entry model
internal/redact   Secret redaction
internal/store    JSONL store and in-memory index
internal/syslog   UDP syslog parser
internal/server   HTTP API, metrics, and web UI
docs/             Architecture and demo notes
examples/         Docker Compose demo
```
