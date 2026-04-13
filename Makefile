# Scope tests to our packages only (avoid accidental packages under web/node_modules).
.PHONY: test run fmt vet up down up-api up-web logs-api
test:
	go test ./cmd/... ./internal/...

run:
	go run ./cmd/server

fmt:
	gofmt -w cmd internal

vet:
	go vet ./cmd/... ./internal/...

# Start full stack.
up:
	docker compose up -d

# Stop stack.
down:
	docker compose down

# Start/restart only API (coupon load happens only when API process restarts).
up-api:
	docker compose up -d api

# Rebuild/restart only web without recreating API.
up-web:
	docker compose up -d --no-deps --build web

logs-api:
	docker compose logs -f api
