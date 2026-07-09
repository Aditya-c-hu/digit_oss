# ws-services (Go)

Go port of the DIGIT `ws-services` (Water Connection) Spring Boot module. Drop-in
replacement for the Java service: same routes, request/response shapes, DB schema,
and Kafka topics.

## Build & run

```bash
go build ./...
go test ./...                      # unit tests (repo integration test skips without a DB)
go vet ./...
go run ./cmd/ws-services           # starts on :8090, context path /ws-services
```

> This module is not a git checkout on its own; if `go build` complains about
> VCS stamping, pass `-buildvcs=false` (the Dockerfile already does).

Docker (build context is the service root, Dockerfile lives under `deployments/`):

```bash
docker build -f deployments/Dockerfile -t ws-services .
```

Schema: apply `migrations/ddl/V001__ws_schema.sql` to the target Postgres (the
local-deploy bundle does this via `db/init.sql`).

## Configuration

Resolution order: **environment variable → `application.properties` → built-in default.**
Env keys are the property key upper-cased with `.`/`-` → `_`
(e.g. `egov.idgen.host` → `EGOV_IDGEN_HOST`, `server.context-path` → `SERVER_CONTEXT_PATH`).

### Core
| Property | Env | Default |
|---|---|---|
| server.port | SERVER_PORT | 8090 |
| server.context-path | SERVER_CONTEXT_PATH | /ws-services |
| db.host / db.port | DB_HOST / DB_PORT | postgres / 5432 |
| db.user / db.password | DB_USER / DB_PASSWORD | postgres / postgres |
| db.name | DB_NAME | rainmaker |
| kafka.brokers | KAFKA_BROKERS | kafka:9092 |
| kafka.group.id | KAFKA_GROUP_ID | egov-ws-services |

### Spring Boot env compatibility (drop-in for the Java pod)

To support the "swap and test" mandate (deploy with the **exact same env** the Java
pod used), the loader also accepts the original Spring Boot env keys. Precedence is
**Go-native env → Spring env → properties file → default**. Defaults match the Java
`application.properties` (`server.port=8090`, group `egov-ws-services`, db `rainmaker_new`).

| Java / Spring env | Go-native env | Maps to |
|---|---|---|
| `SPRING_DATASOURCE_URL` (`jdbc:postgresql://host:port/db?sslmode=`) | `DB_HOST`,`DB_PORT`,`DB_NAME`,`DB_SSLMODE` | DB host/port/name/sslmode (parsed from the JDBC URL) |
| `SPRING_DATASOURCE_USERNAME` | `DB_USER` | DB user |
| `SPRING_DATASOURCE_PASSWORD` | `DB_PASSWORD` | DB password |
| `KAFKA_CONFIG_BOOTSTRAP_SERVER_CONFIG` / `SPRING_KAFKA_BOOTSTRAP_SERVERS` | `KAFKA_BROKERS` | Kafka brokers |
| `SPRING_KAFKA_CONSUMER_GROUP_ID` | `KAFKA_GROUP_ID` | Kafka consumer group |
| `SERVER_PORT` | `SERVER_PORT` | HTTP port (same key) |

### Integration toggles (all default `false` → local/offline mode)
| Toggle | Env | When `true` |
|---|---|---|
| idgen | IS_IDGEN_ENABLED | mint app/connection no via egov-idgen (else local synth) |
| mdms | IS_MDMS_ENABLED | validate coded attributes against MDMS masters |
| property | IS_PROPERTY_ENABLED | validate propertyId via property-services |
| user | IS_USER_ENABLED | resolve/create connection-holder users via egov-user |
| workflow | IS_EXTERNAL_WORKFLOW_ENABLED | drive egov-workflow-v2 (else synthesized state) |
| persister | IS_PERSISTER_ENABLED | egov-persister owns the DB write (else direct SQL) |
| encryption | IS_ENCRYPTION_ENABLED | encrypt holder/plumber PII via egov-enc-service on create, decrypt on search |
| sms / email | NOTIFICATION_SMS_ENABLED / NOTIFICATION_EMAIL_ENABLED | publish notification events |

Hosts for the above: `EGOV_IDGEN_HOST`, `EGOV_MDMS_HOST`, `EGOV_PROPERTY_HOST`,
`EGOV_USER_HOST`, `WORKFLOW_CONTEXT_PATH`, `EGOV_ENC_HOST`.

> **Encryption caveat:** in Java the encrypted attribute set + ABAC visibility
> come from the MDMS `DataSecurity` SecurityPolicy for the `WnSConnection` model
> (not present in this repo). The Go port encrypts the standard WS PII fields
> (holder/plumber name, mobile, correspondence address) via egov-enc-service.
> Validate the attribute set / key type against the tenant SecurityPolicy so
> ciphertext round-trips with the Java service before enabling in production.

> **Production note:** set `IS_PERSISTER_ENABLED=true` only when egov-persister is
> configured to consume `save-ws-connection`/`update-ws-connection`; otherwise the
> service writes the DB directly. Never enable both paths or rows insert twice.

## API

Base path `/ws-services`. All POST, body is the eGov envelope (`RequestInfo` + payload).

| Method | Path | Purpose |
|---|---|---|
| POST | /wc/_create | create water connection application |
| POST | /wc/_update | update / advance workflow |
| POST | /wc/_search | search (criteria in query, RequestInfo in body) |
| POST | /wc/_plainsearch | search without tenant scoping (internal) |
| POST | /wc/_encryptOldData | privacy migration (disabled → 400) |
| GET  | /health | liveness |

Search query params: `tenantId, ids, applicationNumber, applicationStatus,
connectionNumber, oldConnectionNumber, propertyId, status, fromDate, toDate,
offset, limit, applicationType, locality, isPropertyDetailsRequired`. List params
accept repeated (`?ids=a&ids=b`) or comma (`?ids=a,b`) form.

Errors use the eGov envelope: `{ResponseInfo, Errors:[{code,message}]}` —
HTTP 400 for validation/business (`EG_WS_*`), 500 for infra.

Sample payloads: `../local-deploy/samples/`.

## Architecture (Java → Go)

| Java (Spring) | Go |
|---|---|
| `@RestController` | `internal/transport/http` (Gin) |
| `@Service` | `internal/service` |
| `@Repository` / JPA | `internal/repository/postgres` (pgx) |
| prepared-statement / query building | `internal/repository/query` |
| `RowMapper` / `ResultSetExtractor` | `internal/repository/rowmapper` |
| POJOs / models | `internal/domain` |
| `@KafkaListener` / producer | `internal/transport/kafka` |
| validators | `internal/validator` (+ field checks) |
| idgen/mdms/user/property/workflow clients | `internal/{idgen,mdms,user,property,workflow}` |
| shared error type | `pkg/apperr` |
| `application.properties` | `config` (stdlib loader) |
| Spring DI | explicit wiring in `cmd/ws-services/main.go` |

## Project layout

The service follows the reference DIGIT Go structure:

```
cmd/ws-services/main.go              entry point: config load, DI wiring, server start
configs/application.properties.sample
deployments/Dockerfile
docs/ws-services-go.postman_collection.json
internal/
  domain/                            entities, DTOs, request/response structs (json tags)
  repository/
    postgres/                        DB access (transactions, Exec/Query)
    query/                           SQL builders (parameterized, no driver dep)
    rowmapper/                       explicit row → domain mapping
  service/                           business logic, orchestration, integration calls
  transport/
    http/                            Gin routes + handlers (package httptransport)
    kafka/                           consumers + producer
  validator/                         request payload validation
  {idgen,mdms,user,property,workflow,encryption}/   external DIGIT integration clients
migrations/ddl/                      schema SQL
pkg/apperr/                          shared pure utility (typed error)
```

Two intentional, documented naming notes:

- **`internal/transport/http` is package `httptransport`, not `http`.** A package
  literally named `http` in that directory would shadow the standard library
  `net/http` (imported for status constants), so the package name is
  `httptransport` while the directory matches the standard `transport/http`.
- **`config/` (code) vs `configs/` (files).** `config/` is the Go config-loader
  package; `configs/` holds `application.properties.sample` per the structure.

Layer isolation is enforced: handlers parse→bind→call service→respond with no SQL
or business logic; services orchestrate and call integration clients; the
`postgres` repository is DB-only and depends on `query`/`rowmapper`; no
repository imports HTTP, no circular imports.

## Kafka topics

Produces: `save-ws-connection`, `update-ws-connection`,
`egov.core.notification.sms`, `egov.core.notification.email`.
Consumes: `update-ws-workflow`, `editnotification`, `ws-filestoreids-process`,
receipt business topic.

## Assumptions & known gaps

- Integrations are flag-gated; defaults run a self-contained local mode (idgen
  synth, no MDMS/property/user, synthesized workflow state, direct DB write).
- Search returns connection + service + documents + plumbers + holders +
  road-cutting. Owner name/mobile come from egov-user and are not stored locally.
- `_encryptOldData` is intentionally disabled (matches Java when privacy off).
- Field-level encryption of holder/plumber PII via egov-enc-service is wired
  (create encrypts, search decrypts) behind `IS_ENCRYPTION_ENABLED`; see the
  encryption caveat above re: SecurityPolicy parity.

## Postman

`docs/ws-services-go.postman_collection.json` — import and set `{{host}}`.

## Dependencies (Black Duck)

Direct: `gin-gonic/gin` (HTTP routing), `jackc/pgx/v5` (Postgres),
`segmentio/kafka-go` (Kafka), `google/uuid`. Config loading uses the standard
library (no config framework). Gin pulls a transitive set (sonic, validator,
quic-go, etc.); `go mod tidy` is clean and `govulncheck ./...` reports **0
vulnerabilities** (code-reachable and module-level) with the pinned toolchain
`go1.26.3` and the dependency floors set in `go.mod` (pgx ≥ v5.9.0,
golang.org/x/crypto ≥ v0.52.0, golang.org/x/net ≥ v0.54.0).

## Contributing

- Branches: `main` (release), `dev` (integration), `feat/*`, `fix/*`. PRs target `dev`.
- Commit format: `[ws-services] <type>: <Description>` where `<type>` ∈
  `feat | fix | refactor | chore | docs`.
