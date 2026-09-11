# AI Operations Copilot

A small Go service for operations teams to ask natural-language questions about orders, payments, customers, and deliveries.

## Architecture

```text
Client -> Gin API -> Intent LLM -> validated intent -> SQLite repositories
       -> operational context -> Response LLM -> JSON or SSE response
```

The LLM understands and explains. The backend retrieves. SQLite is the source of truth. The LLM never generates SQL.

## Stack

Go, Gin, SQLite, OpenRouter, and optional SSE streaming.

## Setup

```bash
cp .env.example .env
go mod download
go run ./cmd/seed
go run ./cmd/api
```

`cmd/seed` creates 100 deterministic customers and 300 orders with payment and delivery scenarios. Set `OPENROUTER_API_KEY` in `.env` before querying.

## API

`GET /health` returns `{"status":"ok"}`.

`POST /api/v1/query` accepts:

```json
{"query":"What's the payment status for order #4521?","stream":false}
```

Normal responses contain `answer`, `intent`, and an optional `order_id`:

```json
{"answer":"...","intent":"get_payment_status","order_id":"4521"}
```

Example:

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H 'Content-Type: application/json' \
  -d '{"query":"Give me a full status summary for order #4231"}'
```

Set `stream` to `true` to receive `event: token` SSE events and a final `event: done`. Consume the POST response body with `fetch()`; browser `EventSource` only supports GET.

## Design decisions

The first LLM converts natural language to a strict allowlisted intent. The backend validates it and calls fixed, parameterized repository operations. A second LLM explains only the retrieved context. Repository interfaces keep storage replaceable and make the service testable. SQLite keeps local setup simple.

## Tests and Docker

```bash
go test ./...
docker build -t ai-operations-copilot .
docker run --env-file .env -p 8080:8080 ai-operations-copilot
```
