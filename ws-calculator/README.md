# ws-calculator (Go)

Go port of the DIGIT `ws-calculator` (Water Connection Calculator) Spring Boot
module. Drop-in replacement: same routes, calculation tax heads, DB schema, and
Kafka topics.

## Build & run

```bash
go build ./...
go test ./...                      # unit + parity tests
go vet ./...
go run ./cmd/ws-calculator         # starts on :8091, context path /ws-calculator
```

> If `go build` complains about VCS stamping (this module is not a standalone git
> checkout), pass `-buildvcs=false` (the Dockerfile already does).

Docker (context = service root, Dockerfile under `deployments/`):

```bash
docker build -f deployments/Dockerfile -t ws-calculator .
```

Schema: apply `migrations/ddl/V001__ws_calculator_schema.sql` (meter readings).
Demand and bill data live in billing-service (external).

## Configuration

Resolution order: **environment variable → `application.properties` → default.**
Env keys = property key upper-cased with `.`/`-` → `_`.

### Core
| Property | Env | Default |
|---|---|---|
| server.port | SERVER_PORT | 8083 (Java default; compose/bundle set 8091 via env) |
| server.context-path | SERVER_CONTEXT_PATH | /ws-calculator |
| db.* | DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME | postgres/5432/postgres/postgres/rainmaker |
| kafka.brokers | KAFKA_BROKERS | kafka:9092 |
| kafka.group.id | KAFKA_GROUP_ID | egov-ws-calc-services |

### Spring Boot env compatibility (drop-in for the Java pod)

The loader accepts the original Spring Boot env keys so the service deploys with the
**exact same env** as the Java pod. Precedence: **Go-native env → Spring env → properties
file → default**. Defaults match the Java `application.properties` (`server.port=8083`,
group `egov-ws-calc-services`, db `rainmaker_new`).

| Java / Spring env | Go-native env | Maps to |
|---|---|---|
| `SPRING_DATASOURCE_URL` (`jdbc:postgresql://host:port/db?sslmode=`) | `DB_HOST`,`DB_PORT`,`DB_NAME`,`DB_SSLMODE` | DB host/port/name/sslmode (parsed from the JDBC URL) |
| `SPRING_DATASOURCE_USERNAME` | `DB_USER` | DB user |
| `SPRING_DATASOURCE_PASSWORD` | `DB_PASSWORD` | DB password |
| `KAFKA_CONFIG_BOOTSTRAP_SERVER_CONFIG` / `SPRING_KAFKA_BOOTSTRAP_SERVERS` | `KAFKA_BROKERS` | Kafka brokers |
| `SPRING_KAFKA_CONSUMER_GROUP_ID` | `KAFKA_GROUP_ID` | Kafka consumer group |
| `SERVER_PORT` | `SERVER_PORT` | HTTP port (same key) |

### One-time fees (used by `_estimate`; stand in for MDMS fee masters)
| Property | Env | Default |
|---|---|---|
| egov.ws.fee.form | EGOV_WS_FEE_FORM | 100 |
| egov.ws.fee.scrutiny | EGOV_WS_FEE_SCRUTINY | 50 |
| egov.ws.fee.security | EGOV_WS_FEE_SECURITY | 500 |
| egov.ws.fee.roadcutting.rate | EGOV_WS_FEE_ROADCUTTING_RATE | 200 (per unit area) |

### Penalty / interest (PayService parity; MDMS Penalty/Interest masters override)
| Property | Env | Default |
|---|---|---|
| egov.ws.penalty.rate | EGOV_WS_PENALTY_RATE | 10 (% of charge) |
| egov.ws.penalty.applicableafterdays | EGOV_WS_PENALTY_APPLICABLEAFTERDAYS | 30 |
| egov.ws.interest.rate | EGOV_WS_INTEREST_RATE | 12 (% of charge) |
| egov.ws.interest.applicableafterdays | EGOV_WS_INTEREST_APPLICABLEAFTERDAYS | 30 |

Penalty/interest are a flat rate% of the water charge applied once overdue beyond
`applicableAfterDays` (no day-proration) — matches Java `PayService`.

### Integration toggles (default `false`)
| Toggle | Env | When `true` |
|---|---|---|
| mdms | IS_MDMS_ENABLED | load `WCBillingSlab` from MDMS (else seeded slabs) |
| billing | IS_BILLING_ENABLED | search/create/update demands via billing-service (else local synth) |

Hosts: `EGOV_MDMS_HOST`, `EGOV_BILLING_SERVICE_HOST`. Slab master is configurable
(`egov.ws.billingslab.module` / `.master`, default `ws-services-calculation` /
`WCBillingSlab`).

## API

Base path `/ws-calculator`. All POST; eGov envelope.

| Method | Path | Purpose |
|---|---|---|
| POST | /waterCalculator/_estimate | one-time fees for a draft application |
| POST | /waterCalculator/_calculate | periodic water-charge calculation |
| POST | /waterCalculator/_updateDemand | (re)build demands for consumer codes (query) |
| POST | /waterCalculator/_jobscheduler | bulk demand generation (worker-pool fan-out) |
| POST | /waterCalculator/_applyAdhocTax | ad-hoc penalty / rebate |
| POST | /meterConnection/_create | create meter reading |
| POST | /meterConnection/_search | search meter readings (query) |
| GET  | /health | liveness |

### Tax heads
- `_estimate`: `WS_FORM_FEE`, `WS_SCRUTINY_FEE`, `WS_SECURITY_CHARGE`,
  `WS_ROAD_CUTTING_CHARGE` (area × rate), `WS_ONE_TIME_FEE_ROUND_OFF`.
- `_calculate`: `WS_CHARGE` (slab-based), `WS_TIME_PENALTY`, `WS_TIME_INTEREST`,
  `WS_ROUND_OFF`.

Errors: eGov envelope, 400 for validation/business (`EG_WS_*`), 500 for infra.

## Architecture (Java → Go)

| Java | Go |
|---|---|
| `CalculatorController` / `MeterReadingController` | `internal/transport/http` (Gin) |
| `EstimationService` / `PayService` / `MasterDataService` | `internal/service/calculation.go` |
| `DemandService` (billing REST) | `internal/service/demand.go` + `internal/billing` |
| `MeterServicesImpl` | `internal/service/meter.go` |
| `@Repository` | `internal/repository/postgres` (pgx) |
| query building / row mapping | `internal/repository/query` + `internal/repository/rowmapper` |
| `@KafkaListener` / producer | `internal/transport/kafka` |
| MDMS client | `internal/mdms` |
| shared error type | `pkg/apperr` |
| `application.properties` | `config` (stdlib loader) |

## Project layout

Follows the reference DIGIT Go structure:

```
cmd/ws-calculator/main.go            entry point: config load, DI wiring, server start
configs/application.properties.sample
deployments/Dockerfile
docs/ws-calculator-go.postman_collection.json
internal/
  domain/                            entities + request/response DTOs (json tags)
  repository/
    postgres/                        DB access (meter readings)
    query/                           SQL builders
    rowmapper/                       explicit row → domain mapping
  service/                           calculation, estimation, penalty/interest, demand, meter
  transport/
    http/                            Gin routes + handlers (package httptransport)
    kafka/                           consumers + producer
  {billing,mdms}/                    external DIGIT integration clients
migrations/ddl/                      meter-reading schema SQL
pkg/apperr/                          shared pure utility (typed error)
```

Structure notes:

- **`internal/transport/http` is package `httptransport`, not `http`** — avoids
  shadowing the standard library `net/http` (imported for status constants).
- **`config/` (loader code) vs `configs/` (files).**
- **No `internal/validator` package.** Calculator input validation is light
  (request-shape + tenant/consumer checks) and lives in the service/handler
  layer, unlike ws-services which has a dedicated validator. Documented rather
  than forcing a near-empty package.

Layer isolation is enforced: handlers parse→bind→call service→respond (no SQL or
calculation logic); services hold calculation/orchestration; the `postgres`
repository is DB-only and depends on `query`/`rowmapper`; no circular imports.

## Kafka topics

Produces: demand-save (`egov.demand.save`), bill-gen (`ws-bill-gen`).
Consumes: `create-meter-reading`, payment topic (`egov.collection.payment-create`),
bill-gen.

## Calculation parity & tests

`internal/service/calculation_test.go` covers fees, road-cutting, non-metered
minimum charge, tiered metered charge, round-off boundaries, and penalty/interest.
`internal/transport/http/handler_test.go` covers the estimate endpoint (driven
through the Gin router) + query parsing.

## Assumptions & known gaps

- Billing slabs come from MDMS when enabled; otherwise three seeded representative
  slabs are used so `_estimate`/`_calculate` work offline.
- `_estimate` uses the MDMS `FeeSlab` / `RoadType` / `PlotSizeSlab` /
  `PropertyUsageType` masters (module `ws-services-calculation`) when MDMS is
  enabled — EstimationService parity (form/scrutiny/meter/other/road-cutting/
  usage-type/plot-size/tax heads). Usage-type and plot-size charges need the
  caller to pass `usageCategory` + `landArea` on the criteria's `waterConnection`
  (Java reads these from the property). Without MDMS, flat config fees are used.
  Master field assumptions: `FeeSlab.{formFee,scrutinyFee,other,taxpercentage,
  meterCost}`, `RoadType/PropertyUsageType.{code,unitCost}`,
  `PlotSizeSlab.{from,to,unitCost}` — confirm against tenant MDMS before enabling.
- Penalty / interest use the Java `PayService` formula (flat rate% of charge,
  gated by `applicableAfterDays`); MDMS `Penalty`/`Interest` masters override the
  config defaults when MDMS is enabled.
- When billing is disabled, `_updateDemand` returns locally-synthesized demands.

## Postman

`docs/ws-calculator-go.postman_collection.json` — import and set `{{host}}`.

## Dependencies (Black Duck)

Direct: `gin-gonic/gin` (HTTP routing), `jackc/pgx/v5`, `segmentio/kafka-go`,
`google/uuid`. Config loading uses the standard library. `go mod tidy` is clean
and `govulncheck ./...` reports **0 vulnerabilities** (code-reachable and
module-level) with toolchain `go1.26.3` and the `go.mod` dependency floors
(pgx ≥ v5.9.0, golang.org/x/crypto ≥ v0.52.0, golang.org/x/net ≥ v0.54.0).

## Contributing

- Branches: `main` (release), `dev` (integration), `feat/*`, `fix/*`. PRs target `dev`.
- Commit format: `[ws-calculator] <type>: <Description>` where `<type>` ∈
  `feat | fix | refactor | chore | docs`.
