# Reviewer Guide

This guide is the fastest way to inspect and run the Football League Simulation API.

## Quick Start

```bash
go run . migrate up
go run . seed
go run . serve
```

Open:

```text
http://localhost:8080/swagger/index.html
```

The root `openapi.yaml` can also be imported into Postman, Insomnia, or similar API clients.

## Docker

```bash
docker compose up --build
```

Health checks:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/health
```

## Happy Path

```bash
curl -X POST http://localhost:8080/api/v1/league/reset
curl http://localhost:8080/api/v1/league/overview
curl -X POST http://localhost:8080/api/v1/league/next-week
curl -X POST http://localhost:8080/api/v1/league/next-week
curl -X POST http://localhost:8080/api/v1/league/next-week
curl -X POST http://localhost:8080/api/v1/league/next-week
curl http://localhost:8080/api/v1/simulation/championship-probabilities
```

## Useful Options

```bash
SIM_SEED=42
WEATHER_PROVIDER=local
TEAM_STRENGTH_PROVIDER=local
```

Optional external-weather mode:

```bash
WEATHER_PROVIDER=open-meteo go run . serve
```

Optional strength modes:

```bash
TEAM_STRENGTH_PROVIDER=market-value go run . serve
```

Transfermarkt mode is intentionally optional and should point to a self-hosted Transfermarkt-compatible API:

```bash
TEAM_STRENGTH_PROVIDER=transfermarkt TRANSFERMARKT_API_BASE_URL=http://localhost:8000 go run . serve
```

The app falls back to local seeded strengths if the external provider is unavailable.

## Verification

```bash
gofmt -l .
go test ./...
go vet ./...
go build ./...
go test ./services -bench=BenchmarkLeagueService_GetPredictions -benchmem -run '^$'
docker build -t insider-one-backend-project:review .
```

## What To Look For

- strict request validation and problem+json errors with stable `code` fields;
- transactional mutations for edit, rollback, reset, and week simulation;
- shared ranking logic for real standings and simulated standings;
- prediction cache invalidation after every league mutation;
- optional external adapters with timeout, cache, and fallback behavior;
- health/readiness endpoints and request IDs for operational visibility.
