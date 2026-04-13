# Food Ordering API (Backend)

Simple Go backend for a food ordering challenge.

## What this project does

- Lists products
- Gets a product by ID
- Places an order
- Validates coupon codes (must exist in at least 2 of 3 coupon files)

## Endpoints

- `GET /product`
- `GET /product/{productId}`
- `POST /order` (requires header: `api_key`)
- `GET /healthz`

## Folder structure (simple)

```text
api/
cmd/
internal/
web/
```

- `api/`  
  Keeps the OpenAPI spec file (`openapi.yaml`).

- `cmd/`  
  App entrypoint. `cmd/server/main.go` starts the API server.

- `internal/`  
  Main backend code:
  - `api/` -> HTTP handlers and routes
  - `catalog/` -> product list data
  - `coupon/` -> coupon loading and validation logic
  - `config/` -> env config reading
  - `models/` -> request/response structs

- `web/`  
  Optional frontend and API tester page.  
  (Unable to finish this ontime, this to be ignored in git.)

## Requirements

- Go 1.25+
- Docker (optional)

## Run locally (Go)

1. Make sure `.env` exists (default one is already in repo).
2. Start server:

```bash
go run ./cmd/server
```

Server runs on `http://localhost:8080`.

## Test quickly with curl

```bash
# health
curl -sS http://localhost:8080/healthz

# products
curl -sS http://localhost:8080/product

# single product
curl -sS http://localhost:8080/product/1

# place order
curl -sS -X POST http://localhost:8080/order \
  -H "Content-Type: application/json" \
  -H "api_key: apitest" \
  -d '{"items":[{"productId":"1","quantity":1}]}'
```

## Run with Docker

```bash
docker compose up -d api
```

API: `http://localhost:8080`

Docker uses local folder `coupon-data/` as `/data` in the container.
So users can copy coupon files there first:

- `coupon-data/couponbase1.gz`
- `coupon-data/couponbase2.gz`
- `coupon-data/couponbase3.gz`

## Swagger

```bash
docker compose up -d swagger
```

Swagger UI: `http://localhost:8081`

## Notes

- Coupon files are large. First startup can take time.
- If `.idx` files are missing, the app creates them automatically.
- If `.idx` files already exist in `coupon-data/`, startup is faster.
