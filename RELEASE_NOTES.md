# v0.1.0 - Loglite

Initial release of the lightweight Go log aggregator.

## Added

- HTTP ingestion for JSON, JSON arrays, and NDJSON.
- UDP syslog ingestion with RFC3164-like parsing.
- Secret redaction before storage.
- Append-only JSONL storage with retention pruning.
- In-memory full-text index and label filters.
- Query API and built-in web UI.
- Prometheus text metrics.
- Dockerfile and docker-compose demo.
- Unit tests for redaction, store/search, syslog parsing, and HTTP API.
