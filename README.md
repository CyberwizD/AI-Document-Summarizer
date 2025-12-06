AI Document Summarizer
======================

Golang service that accepts PDF/DOCX uploads, stores the raw file in S3/MinIO, extracts text, runs an OpenRouter LLM for summary + document type + metadata, and serves combined results from SQLite (default).

Architecture / Stack
--------------------
- Go 1.24.x, Gin (HTTP), GORM (ORM)
- SQLite (default) — swappable to Postgres/MySQL by changing the DSN/driver
- MinIO Go client — S3-compatible object storage
- OpenRouter — LLM gateway for analysis
- OpenAPI spec — `docs/swagger.yaml`

Project Layout
--------------
- `cmd/server/main.go` — wiring for config, DB, storage, LLM, routes
- `internal/config` — environment-driven configuration
- `internal/db` — GORM initialization
- `internal/documents` — PDF/DOCX text extractors
- `internal/llm` — OpenRouter client
- `internal/services` — document business logic
- `internal/handlers` — Gin HTTP handlers
- `internal/storage` — S3/MinIO helper
- `docs/swagger.yaml` — API specification
- `tests/` — service-level tests

Prerequisites
-------------
- Go 1.24.11+ on PATH
- Running S3-compatible storage (MinIO recommended)
- OpenRouter API key

Configuration
-------------
Copy `.env.example` to `.env` and set your values:
```
SERVER_PORT=8080
DATABASE_URL=file:./data/app.db?_foreign_keys=on

S3_ENDPOINT=localhost:9000
S3_REGION=us-east-1
S3_BUCKET=documents
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_USE_SSL=false

OPENROUTER_API_KEY=sk-your-key
OPENROUTER_MODEL=gpt-4o-mini
REQUEST_TIMEOUT=30s
```

MinIO quickstart (local)
------------------------
```
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  quay.io/minio/minio server /data --console-address ":9001"

mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/documents   # must match S3_BUCKET
```

Install & Run
-------------
```
go mod tidy
go run ./cmd/server
```
Server listens on `:SERVER_PORT` (default 8080). SQLite file is created under `./data`.

API (requests and responses)
----------------------------
`POST /documents/upload`
- Purpose: upload `.pdf` or `.docx` (<=5MB), store in S3, extract text, persist record.
- Example:
  ```
  curl -X POST http://localhost:8080/documents/upload \
    -F "file=@/path/to/sample.pdf"
  ```
- Response (201):
  ```json
  {
    "id": "a1b2c3",
    "filename": "sample.pdf",
    "mime_type": "application/pdf",
    "size_bytes": 12345,
    "storage_key": "documents/a1b2c3.pdf",
    "extracted_text": "....",
    "summary": "",
    "document_type": "",
    "metadata": null,
    "created_at": "2025-12-06T01:23:45Z",
    "updated_at": "2025-12-06T01:23:45Z"
  }
  ```

`POST /documents/{id}/analyze`
- Purpose: send extracted text to OpenRouter, save summary/type/metadata.
- Example:
  ```
  curl -X POST http://localhost:8080/documents/a1b2c3/analyze
  ```
- Response (200):
  ```json
  {
    "id": "a1b2c3",
    "summary": "Short summary ...",
    "document_type": "invoice",
    "metadata": {"total": "1200.00", "date": "2025-12-01"},
    "extracted_text": "....",
    "filename": "sample.pdf",
    "mime_type": "application/pdf",
    "size_bytes": 12345,
    "storage_key": "documents/a1b2c3.pdf",
    "created_at": "2025-12-06T01:23:45Z",
    "updated_at": "2025-12-06T01:24:10Z"
  }
  ```
- If OpenRouter returns an error, the response body is included in the server error for debugging.

`GET /documents/{id}`
- Purpose: fetch stored record (file info, extracted text, summary, metadata).
- Example:
  ```
  curl http://localhost:8080/documents/a1b2c3
  ```
- Response (200): same shape as above.

`GET /health`
- Purpose: liveness probe.

Swagger / OpenAPI
-----------------
Import `docs/swagger.yaml` into Swagger UI/Insomnia/Postman or host it with your preferred viewer.

Testing
-------
If you have a parent `go.work`, disable it for this repo:
```
set GOWORK=off   # PowerShell: $env:GOWORK='off'
go test ./...
```

Switching Databases
-------------------
- SQLite default: `DATABASE_URL=file:./data/app.db?_foreign_keys=on`
- Postgres example: `DATABASE_URL=postgres://user:pass@localhost:5432/dbname`
- MySQL example: `DATABASE_URL=user:pass@tcp(localhost:3306)/dbname?parseTime=true`
If you switch drivers, update `internal/db` to open the matching GORM driver.

Operational Notes
-----------------
- Only `.pdf` and `.docx` up to 5MB are accepted.
- Bucket creation is attempted at startup; the configured user must have permissions.
- `/documents/{id}/analyze` requires `OPENROUTER_API_KEY`; without it, calls will fail. The text must be non-empty.*** End Patch
