# AI Operations Copilot

AI Operations Copilot is a Go service that lets operations teams ask natural-language questions about orders, payments, customers, and deliveries.

The service follows a strict source-of-truth boundary:

```text
User query
    -> Intent LLM
    -> validated intent
    -> fixed repository operations
    -> SQLite operational data
    -> bounded context
    -> Response LLM
    -> JSON or SSE response
```

The LLM understands and explains. The backend retrieves. SQLite provides the facts. The LLM never generates or executes SQL.

## Features

- Natural-language operational queries.
- Payment, delivery, order status, summaries, and issue diagnosis.
- Safe allowlisted intents and data sources.
- Parameterized SQLite queries through repository interfaces.
- Deterministic seed data with realistic operational inconsistencies.
- Normal JSON responses and optional Server-Sent Events streaming.
- Structured API errors and request IDs.
- Configurable OpenRouter model, timeout, database path, and query length.

## Technology

- Go 1.23+
- Gin
- SQLite using `modernc.org/sqlite`
- OpenRouter chat completions API
- Docker and Docker Compose

## Prerequisites

Install:

- Go 1.23 or newer
- An OpenRouter API key
- `curl` for command-line API requests

Docker is optional for containerized execution.

## Local Setup

Run these commands from the repository root:

```bash
cd ai-operations-copilot
cp .env.example .env
```

Open `.env` and set your OpenRouter key:

```env
OPENROUTER_API_KEY=your-openrouter-api-key
```

The default model is:

```env
OPENROUTER_MODEL=openrouter/free
```

`openrouter/free` lets OpenRouter select an available free model. Free models can still be rate-limited or temporarily unavailable. You can replace it with any model currently available to your OpenRouter account.

Install dependencies:

```bash
go mod download
```

## Seed the Database

Before starting the API, generate the local SQLite data:

```bash
go run ./cmd/seed
```

The seed command creates:

- 100 customers
- 300 orders
- 300 payments
- 300 deliveries

The generator uses a fixed random seed, so scenario assignments, IDs, amounts, and relationships are reproducible. Timestamps are generated relative to the time the seed command runs. The data includes paid orders, pending payments, failed payments, unscheduled deliveries, delayed deliveries, cancelled orders, refunded payments, and rejected orders.

The seed command clears and recreates the existing operational records in the configured database. Do not run it against production data.

The default database path is:

```text
./data/operations.db
```

The documented example orders `4521`, `1289`, `2231`, `1045`, and `1201` are included in the generated data.

## Start the API

Start the server after seeding:

```bash
go run ./cmd/api
```

The server listens on:

```text
http://localhost:8080
```

Keep this process running while sending requests from another terminal.

## Health Check

Verify that the API is running:

```bash
curl http://localhost:8080/health
```

Response:

```json
{"status":"ok"}
```

## Query API

The primary endpoint is:

```http
POST /api/v1/query
```

The endpoint is a `POST` endpoint. The exact route includes `/api/v1`; `/api/query` and `GET /api/v1/query` are not valid routes.

### Request

```json
{
  "query": "What's the payment status for order #4521?",
  "stream": false
}
```

`query` is required and must not be empty. Queries are limited to 4000 characters by default. `stream` is optional and defaults to `false`.

### Basic curl request

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query":"What is the payment status for order #4521?"}'
```

### JSON response

```json
{
  "answer": "Order #4521 has been paid successfully.",
  "intent": "get_payment_status",
  "order_id": "4521"
}
```

The answer is generated from the records retrieved from SQLite. The response does not expose provider prompts or raw OpenRouter responses.

## Query Examples

Payment status:

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query":"What is the payment status for order #4521?"}'
```

Delivery status:

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query":"Why has order #1045 not been delivered?"}'
```

Full order summary:

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query":"Give me a full status summary for order #2231."}'
```

Issue diagnosis:

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query":"Customer says they paid for order #1289 but delivery is not scheduled. What is going on?"}'
```

Rejected or failed order:

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query":"Show me what is wrong with order #1201."}'
```

Operational list query:

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query":"Show me orders where payment succeeded but delivery has not been scheduled."}'
```

Other useful questions include:

- `Is order #4521 paid?`
- `What is the delivery status for order #2231?`
- `Why was order #1201 rejected?`
- `Which orders have delivery issues?`
- `I paid for my order but it has not been scheduled for delivery. What should operations check?`

If an order-specific question does not include an order ID, the service asks for an identifier instead of guessing.

## Streaming Responses

Set `stream` to `true`:

```bash
curl -N -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query":"Give me a full status summary for order #2231.","stream":true}'
```

The response uses Server-Sent Events:

```text
event: token
data: Order

event: token
data: #2231

event: done
data: {}
```

Because this is a streaming `POST` request, consume it with `fetch()` or another HTTP client. Browser `EventSource` only supports `GET`.

## Error Responses

Errors use this format:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "query cannot be empty"
  }
}
```

Common statuses:

| Status | Code | Meaning |
| --- | --- | --- |
| `400` | `INVALID_REQUEST` | Invalid JSON, empty query, or query too long |
| `404` | `NOT_FOUND` | Requested order or related record was not found |
| `422` | `INVALID_INTENT` | The query could not be safely understood |
| `502` | `LLM_PROVIDER_ERROR` | OpenRouter failed, rejected the model, or timed out |

Every request receives an `X-Request-ID` response header. Include that ID when investigating server logs.

## Configuration

Configuration is loaded from `.env` and the process environment. Explicitly exported environment variables take precedence over `.env` values.

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP server port |
| `DATABASE_PATH` | `./data/operations.db` | SQLite database path |
| `OPENROUTER_API_KEY` | empty | OpenRouter API key |
| `OPENROUTER_MODEL` | `openrouter/free` | OpenRouter model or router slug |
| `OPENROUTER_BASE_URL` | `https://openrouter.ai/api/v1` | OpenRouter API base URL |
| `LLM_TIMEOUT_SECONDS` | `30` | Timeout applied to each LLM call |
| `MAX_QUERY_LENGTH` | `4000` | Maximum user query length |

Never commit `.env` or API keys. Rotate a key immediately if it is exposed in source control, logs, screenshots, or chat.

## Tests and Static Checks

Run the full test suite:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

The tests use mocked LLM clients where appropriate and do not require an OpenRouter key.

## Docker

Build the image:

```bash
docker build -t ai-operations-copilot .
```

Seed the host-mounted database before starting the container. Run this from the project root:

```bash
go run ./cmd/seed
```

For normal server execution:

```bash
docker run --rm \
  --env-file .env \
  -p 8080:8080 \
  -v "$PWD/data:/app/data" \
  ai-operations-copilot
```

The container starts the API but does not automatically reseed the database. This avoids silently destroying data during container restarts.

Docker Compose:

```bash
docker compose build
docker compose up api
```

Docker Compose uses the same `.env` file and mounts `./data` into the container. Rerun `go run ./cmd/seed` when you intentionally want to recreate the local data. The seed command clears existing operational records, so do not use it against production data.

## Architecture and Design

The request pipeline has two LLM stages:

1. The intent stage converts the user query into an allowlisted intent, entities, and required data sources.
2. The backend validates that result and retrieves only relevant records with fixed repository methods.
3. The response stage explains the retrieved context and cannot access the database directly.

This separation prevents arbitrary SQL generation, limits prompt-injection impact, keeps SQLite as the source of truth, and makes repository and LLM behavior independently testable.

See [DESIGN.md](DESIGN.md) for the detailed architecture, data model, failure handling, scalability considerations, and tradeoffs.

## Project Layout

```text
cmd/api/          API server entry point
cmd/seed/         deterministic database seed command
internal/config/  environment and .env configuration
internal/handler/ HTTP and SSE handlers
internal/llm/     OpenRouter client and prompts
internal/models/  domain models
internal/query/   intent validation and context building
internal/repo/    SQLite repository implementations
internal/service/ query orchestration
migrations/       SQLite schema
tests/            integration-test location
```

## Troubleshooting

### `401 Missing Authentication header`

The process did not receive `OPENROUTER_API_KEY`. Confirm that `.env` exists in the project root, that the key is non-empty, and restart the API after changing it.

### `404` for a known example order

The API reads the database at `DATABASE_PATH`. Rerun the seed command using the same `.env` and restart the API:

```bash
go run ./cmd/seed
go run ./cmd/api
```

### `404` for `/api/query`

Use the complete route:

```text
POST /api/v1/query
```

Do not use `GET`, and make sure the URL has no trailing spaces.

### `422` invalid intent

Free models may return malformed or non-JSON intent output. The service validates and safely rejects unsupported values, and includes deterministic fallback handling for common order and payment questions. Try the same request again or select a currently available OpenRouter model.

### OpenRouter model unavailable

Model availability changes. Check the current catalog:

```bash
curl -sS https://openrouter.ai/api/v1/models \
  -H "Authorization: Bearer $OPENROUTER_API_KEY" \
  | jq -r '.data[] | [.id, .pricing.prompt, .pricing.completion] | @tsv' \
  | head -30
```

Then set `OPENROUTER_MODEL` to an available model or use `openrouter/free`.

If the startup log still shows an old model after editing `.env`, an exported shell variable is overriding the file. Clear it before restarting:

```bash
unset OPENROUTER_MODEL
go run ./cmd/api
```

The startup log prints the model selected by the process. It should show `openrouter/free` unless you intentionally configured another model.

## License

This project is an assignment implementation and does not currently declare a separate open-source license.
