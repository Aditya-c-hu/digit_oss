# DIGIT ws-services + ws-calculator — Go Port

This folder is a Go reimplementation of the two Java/Spring Boot DIGIT-OSS modules
`ws-services` (water-connection registry) and `ws-calculator` (water-demand
calculator). It is API-compatible with the originals: same paths, same JSON
shape, same Postgres schema, same Kafka topics.

---

## Table of Contents

- [Layout](#layout)
- [Prerequisites](#prerequisites)
- [Deployment Options](#deployment-options)
  - [Option A — Docker Compose (recommended)](#option-a--docker-compose-recommended)
  - [Option B — Single-container Bundle](#option-b--single-container-bundle-monolithic-dockerfile)
  - [Option C — Local development (no Docker)](#option-c--local-development-no-docker)
- [Configuration](#configuration)
- [API Surface](#api-surface)
- [Port Map](#port-map)
- [Troubleshooting](#troubleshooting)
- [Building from Source](#building-from-source)
- [Multithreading Parity](#multithreading-parity)
- [Migration Notes](#migration-notes)

---

## Layout

```
municipal-services-go/
├── ws-services/         Go module — water connection CRUD + workflow
│   ├── cmd/ws-services/ main.go entrypoint (DI wiring, server start)
│   ├── config/          stdlib config loader (env / properties)
│   ├── configs/         application.properties.sample
│   ├── deployments/     Dockerfile (multi-stage golang:1.26 -> alpine:3.20)
│   ├── docs/            Postman collection
│   ├── internal/
│   │   ├── transport/http/   Gin routes + handlers (package httptransport)
│   │   ├── transport/kafka/  producer + multi-goroutine consumer group
│   │   ├── service/          business logic (port of WaterServiceImpl)
│   │   ├── repository/postgres|query|rowmapper/  pgx DB access, SQL builders, row mapping
│   │   ├── validator/        request payload validation
│   │   ├── workflow/         egov-workflow-v2 client (or local stub)
│   │   └── domain/           structs mirroring Java DTOs
│   ├── migrations/ddl/  schema SQL
│   └── pkg/apperr/      shared typed-error utility
├── ws-calculator/       Go module — estimate, calculate, demand, meter reading
│   └── (mirror layout; validation is in the service layer, no validator pkg)
├── db/init.sql          Combined DDL — auto-applied to Postgres on first boot
├── docs/                OpenAPI specs (Swagger 3) for both services
├── postman/             Importable Postman collections
├── docker-compose.yml   Postgres + Kafka (KRaft) + both Go apps
├── Dockerfile           Monolithic all-in-one bundle (28 services + infra)
├── DEPENDENCIES.md      Full dependency tree documentation
└── .env.example         Environment variable template
```

---

## Prerequisites

| Tool | Minimum version | Install |
|------|----------------|---------|
| **Docker Desktop** | 4.x+ | [docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop/) |
| **Docker Compose** | v2 (bundled with Docker Desktop) | Included in Docker Desktop |
| **Go** *(local dev only)* | 1.22+ | [go.dev/dl](https://go.dev/dl/) |
| **Git** | any | [git-scm.com](https://git-scm.com/) |

### Docker Desktop DNS Configuration (important!)

Docker builds may fail with `Temporary failure in name resolution` if your
network DNS isn't reachable from inside containers. **Fix this once:**

1. Open **Docker Desktop → Settings (⚙️) → Docker Engine**
2. Add the `"dns"` key to the JSON config:

```json
{
  "builder": {
    "gc": {
      "defaultKeepStorage": "20GB",
      "enabled": true
    }
  },
  "dns": ["8.8.8.8", "8.8.4.4"],
  "experimental": false
}
```

3. Click **Apply & Restart**

---

## Deployment Options

### Option A — Docker Compose (recommended)

Starts only the Go services + their infra (Postgres, Kafka). Lightweight and
fast to iterate.

```powershell
# 1) Build & start the full stack
docker compose up -d --build

# 2) Verify all containers are running
docker compose ps

# 3) Tail logs for the Go services
docker compose logs -f ws-services ws-calculator

# 4) Health checks
curl http://localhost:8090/health
curl http://localhost:8091/health

# 5) Smoke-test: create a water connection
curl -X POST http://localhost:8090/ws-services/wc/_create `
  -H "Content-Type: application/json" `
  -d '{
    "RequestInfo": {
      "userInfo": { "uuid": "u-1", "type": "CITIZEN", "tenantId": "pb.amritsar" }
    },
    "WaterConnection": {
      "tenantId": "pb.amritsar",
      "propertyId": "PB-PT-001",
      "connectionCategory": "RESIDENTIAL",
      "connectionType": "Non_Metered",
      "channel": "CITIZEN",
      "applicationType": "NEW_WATER_CONNECTION"
    }
  }'

# 6) Tear down (removes containers + volumes)
docker compose down -v
```

**Services started by `docker compose`:**

| Container | Image | Port |
|-----------|-------|------|
| `ws-postgres` | `postgres:15-alpine` | 5432 |
| `ws-kafka` | `apache/kafka:3.7.0` (KRaft mode, no Zookeeper) | 9092 |
| `ws-services` | Built from `./ws-services/Dockerfile` | 8090 |
| `ws-calculator` | Built from `./ws-calculator/Dockerfile` | 8091 |

---

### Option B — Single-container Bundle (monolithic Dockerfile)

Builds **all 28 services** (20 Java, 1 Node, 2 Go) plus PostgreSQL, ZooKeeper,
and Kafka into a single image managed by `supervisord`. Intended for demo /
offline environments.

```powershell
# 1) Ensure any conflicting docker-compose services are stopped
docker compose down -v

# 2) Build the monolithic image (takes 10–20 min on first run)
docker build -t digit-ws-bundle .

# 3) Run the container, exposing all service ports
docker run -d --name digit-ws -p 5432:5432 -p 9092:9092 -p 2181:2181 -p 8080-8099:8080-8099 -p 8200-8204:8200-8204 -p 8280:8280 -p 8281:8281 -p 8290:8290 digit-ws-bundle

# 3) Watch supervisor logs (all 28 services + infra)
docker logs -f digit-ws

# 4) Check individual service logs inside the container
docker exec digit-ws cat /var/log/supervisor/ws-services.log
docker exec digit-ws cat /var/log/supervisor/postgres.log
docker exec digit-ws cat /var/log/supervisor/kafka.log

# 5) Verify services are healthy
curl http://localhost:8090/health    # ws-services
curl http://localhost:8091/health    # ws-calculator

# 6) Stop & remove
docker stop digit-ws && docker rm digit-ws
```

> **Note:** The monolithic image is large (~2 GB). For development, prefer
> [Option A](#option-a--docker-compose-recommended).

---

### Option C — Local development (no Docker)

You need Postgres 15 and Kafka 3.7+ running on your machine (or Docker
Compose just for infra).

```powershell
# Start only infra via compose (Postgres + Kafka)
docker compose up -d postgres kafka

# Apply the DDL
psql -h localhost -U postgres -d rainmaker -f db/init.sql

# Terminal 1 — ws-services
cd ws-services
$env:DB_HOST="localhost"; $env:DB_PORT="5432"
$env:DB_USER="postgres"; $env:DB_PASSWORD="postgres"
$env:DB_NAME="rainmaker"
$env:KAFKA_BROKERS="localhost:9092"
$env:SERVER_PORT="8090"
go run ./cmd/ws-services

# Terminal 2 — ws-calculator
cd ws-calculator
$env:DB_HOST="localhost"; $env:DB_PORT="5432"
$env:DB_USER="postgres"; $env:DB_PASSWORD="postgres"
$env:DB_NAME="rainmaker"
$env:KAFKA_BROKERS="localhost:9092"
$env:SERVER_PORT="8091"
go run ./cmd/ws-calculator
```

**Bash/Linux/macOS equivalent:**

```bash
cd ws-services
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres \
DB_NAME=rainmaker KAFKA_BROKERS=localhost:9092 SERVER_PORT=8090 \
go run ./cmd/ws-services
```

---

## Configuration

Both services use [viper](https://github.com/spf13/viper). Every property in the
original Java `application.properties` is bindable as an environment variable
with `.` mapped to `_` (e.g. `egov.idgen.host` → `EGOV_IDGEN_HOST`). Defaults
are sensible for the docker-compose stack.

Copy `.env.example` to `.env` and customise as needed.

**Important toggles:**

| Env var | Default | Effect |
|---------|---------|--------|
| `IS_EXTERNAL_WORKFLOW_ENABLED` | `false` | When false, workflow transitions resolve via a built-in state mapper instead of calling egov-workflow-v2. |
| `NOTIFICATION_SMS_ENABLED` | `false` | Wires Kafka publishes to the SMS topic. |
| `NOTIFICATION_EMAIL_ENABLED` | `false` | Same, for email. |
| `KAFKA_BROKERS` | `kafka:9092` | Comma-separated brokers. |

---

## API Surface

### ws-services (`http://localhost:8090/ws-services`)

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/wc/_create` | Create a water connection application |
| POST | `/wc/_update` | Update / advance workflow on an application |
| POST | `/wc/_search` | Search with criteria + pagination |
| POST | `/wc/_plainsearch` | Privileged search (indexer / migration) |
| POST | `/wc/_encryptOldData` | One-shot privacy-migration (returns 400 unless enabled) |
| GET  | `/health` | Liveness probe |

### ws-calculator (`http://localhost:8091/ws-calculator`)

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/waterCalculator/_estimate` | Estimate fees for a draft application |
| POST | `/waterCalculator/_calculate` | Run full demand calculation |
| POST | `/waterCalculator/_updateDemand` | Recompute demands for given consumer codes |
| POST | `/waterCalculator/_jobscheduler` | Bulk demand generation with worker pool |
| POST | `/waterCalculator/_applyAdhocTax` | Adhoc penalty / rebate |
| POST | `/meterConnection/_create` | Create a meter reading |
| POST | `/meterConnection/_search` | Search meter readings |
| GET  | `/health` | Liveness probe |

Full request/response schemas: see `docs/ws-services.swagger.yaml` and
`docs/ws-calculator.swagger.yaml`.

Postman collections are in `postman/` — import both, set `host=localhost`,
`port=8090` or `8091`.

---

## Port Map

All ports used across every deployment mode:

| Port | Service | Mode |
|------|---------|------|
| 5432 | PostgreSQL | All |
| 9092 | Kafka broker | All |
| 2181 | ZooKeeper | Bundle only |
| 8080 | pdf-service (Node) | Bundle only |
| 8081 | egov-user | Bundle only |
| 8082 | egov-persister | Bundle only |
| 8083 | egov-filestore | Bundle only |
| 8084 | egov-location | Bundle only |
| 8085 | egov-accesscontrol | Bundle only |
| 8086 | egov-common-masters | Bundle only |
| 8087 | egov-localization | Bundle only |
| 8088 | egov-idgen | Bundle only |
| 8089 | egov-enc-service | Bundle only |
| 8090 | **ws-services (Go)** | All |
| 8091 | **ws-calculator (Go)** | All |
| 8092 | egov-indexer | Bundle only |
| 8093 | egov-notification-mail | Bundle only |
| 8094 | egov-mdms-service | Bundle only |
| 8095 | egov-notification-sms | Bundle only |
| 8096 | egov-otp | Bundle only |
| 8097 | egov-pg-service | Bundle only |
| 8098 | egov-searcher | Bundle only |
| 8099 | egov-url-shortening | Bundle only |
| 8200 | tenant | Bundle only |
| 8201 | user-otp | Bundle only |
| 8202 | billing-service | Bundle only |
| 8203 | collection-services | Bundle only |
| 8204 | egov-apportion-service | Bundle only |
| 8280 | property-services | Bundle only |
| 8281 | pt-calculator-v2 | Bundle only |
| 8290 | egov-workflow-v2 | Bundle only |

---

## Troubleshooting

### DNS resolution failures during `docker build`

**Symptom:**
```
Unknown host repo.maven.apache.org: Temporary failure in name resolution
```
or
```
WARNING: fetching https://dl-cdn.alpinelinux.org/alpine/v3.21/main: temporary error
```

**Fix:** Add DNS servers to Docker Desktop (see [Prerequisites](#docker-desktop-dns-configuration-important)).

---

### `bitnami/kafka:3.7` or `bitnami/zookeeper:3.9` not found

**Symptom:**
```
failed to resolve reference "docker.io/bitnami/kafka:3.7": not found
```

**Cause:** Bitnami migrated all public images to a legacy repository (Aug 2025).

**Fix:** The `docker-compose.yml` now uses `apache/kafka:3.7.0` with **KRaft
mode** (no Zookeeper needed). Pull the latest compose file.

---

### `Unable to locate package postgresql-15`

**Symptom:**
```
E: Unable to locate package postgresql-15
```

**Cause:** The `eclipse-temurin:8-jre-jammy` base image (Ubuntu 22.04) doesn't
include PostgreSQL 15 in its default APT repositories.

**Fix:** The `Dockerfile` now adds the official PGDG APT repository before
installing. Pull the latest Dockerfile.

---

### `docker compose` warns about obsolete `version` attribute

**Symptom:**
```
the attribute `version` is obsolete, it will be ignored
```

**Fix:** Already removed from `docker-compose.yml`. This is just a warning and
does not affect functionality.

---

### Container ports already in use

**Symptom:**
```
Bind for 0.0.0.0:5432 failed: port is already allocated
```

**Fix:** Stop local PostgreSQL/Kafka, or remap ports in `.env` / `docker-compose.yml`:
```powershell
# Check what's using the port
netstat -ano | findstr :5432
```

---

### Go build fails with missing `go.sum`

**Fix:** Run `go mod tidy` before building:
```powershell
cd ws-services && go mod tidy
cd ..\ws-calculator && go mod tidy
```

---

## Building from Source

```powershell
# Build both Go services
cd ws-services
go mod tidy
go build ./...

cd ..\ws-calculator
go mod tidy
go build ./...
```

The first `go build` inside the Docker image runs `go mod tidy` to populate
`go.sum` (we ship `go.mod` only). If you build offline, run `go mod tidy` once
on a connected machine and commit the resulting `go.sum`.

---

## Multithreading Parity

The Java services use:
- `@KafkaListener`-backed consumer threads.
- `ThreadPoolExecutor` for bulk demand gen.
- `@Scheduled` master-data refresh.

The Go port keeps the same shape using goroutines:
- `internal/transport/kafka.ConsumerGroup.Subscribe` spawns one goroutine per topic.
- `service.GenerateDemandsForCycle` uses a worker pool of 8 goroutines (`jobs`/`results`
  channels + `sync.WaitGroup`).
- `cmd/ws-calculator/main.go` runs a master-data refresh ticker on its own goroutine.
- The slab table is protected by an `sync.RWMutex` so handlers do not block each other.

---

## Migration Notes

| Java module | Go counterpart |
|-------------|----------------|
| Spring Boot starter | `gin-gonic/gin` |
| spring-kafka | `segmentio/kafka-go` |
| Spring JDBC + flyway | `jackc/pgx/v5` + `db/init.sql` |
| Lombok DTOs | Plain Go structs with `json:"..."` tags |
| `@Value` properties | Viper config |
| `@Service` beans | Plain structs in `internal/service` |
| `RestTemplate` HTTP calls | `net/http` clients (workflow client is the example) |
| swagger annotations | `docs/*.swagger.yaml` (hand-authored) |

Some Java capabilities are intentionally simplified or stubbed:
- `enc-client`/encryption: privacy migration endpoint hard-fails as in Java when disabled.
- `mdms-client`: master-data is seeded in-process; the refresher hook is wired
  for plugging in a real MDMS pull.
- `idgen-client`: `applicationNo` is generated locally from tenant + UUID.
- `notification`: SMS/email producers exist but are off by default.

These are clearly marked at the call sites and are easy to swap for HTTP clients
to the real services in a production deployment.
# Updated README
