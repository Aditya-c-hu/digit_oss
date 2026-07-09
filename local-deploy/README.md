# Local Native Deploy — All 28 DIGIT WS Services on Windows

Run every service from `municipal-services-go/` directly on the host (no Docker).
Tested on Windows 11 / PowerShell 5.1.

---

## What gets started

| Tier  | Count | Names |
|-------|-------|-------|
| Infra | 3 | PostgreSQL 15, ZooKeeper, Kafka 3.7 |
| Java core | 20 | egov-mdms-service, egov-user, egov-idgen, egov-persister, egov-filestore, egov-workflow-v2, egov-location, egov-localization, egov-accesscontrol, egov-common-masters, egov-enc-service, egov-indexer, egov-notification-mail, egov-notification-sms, egov-otp, egov-pg-service, egov-searcher, egov-url-shortening, tenant, user-otp |
| Java muni | 2 | property-services, pt-calculator-v2 |
| Java biz | 3 | billing-service, collection-services, egov-apportion-service |
| Node | 1 | pdf-service |
| Go | 2 | ws-services, ws-calculator |

**Total: 31 OS processes.** Steady-state RAM ~6-8 GB. Disk for Maven cache ~3 GB.

---

## 0. Prereqs (one-time)

Open **PowerShell as Administrator**, run:

```powershell
cd C:\Users\Aditya\Downloads\DIGIT-OSS-master\municipal-services-go\local-deploy
.\01-install-prereqs.ps1
```

Installs via `winget`: Eclipse Temurin JDK 8, Maven 3.9, PostgreSQL 15.
Downloads + extracts Kafka 3.7 to `C:\kafka`.

After install: **close and reopen PowerShell** so PATH refreshes. Verify:

```powershell
java -version    # 1.8.x
mvn -v           # 3.9.x
go version       # 1.22+
node -v          # 18+ (24 OK)
psql --version   # 15.x
```

---

## 1. Build everything (~20-40 min, one-time + on source change)

```powershell
.\02-build-all.ps1
```

Runs `mvn clean package -DskipTests` against all 25 Java services (with the
included `maven-settings.xml` so Maven Central is tried first), `go build` for
the two Go services, and `npm install` for pdf-service. Logs land in
`local-deploy\logs\build-*.log`.

Re-run if you change source. Stops on first failure — open the log to debug.

---

## 2. Start infrastructure (Postgres + ZooKeeper + Kafka)

```powershell
.\03-start-infra.ps1
```

- Starts the Windows `postgresql-x64-15` service
- Launches ZooKeeper + Kafka in two new windows from `C:\kafka\bin\windows\`

Waits for ports 5432, 2181, 9092 to open before returning.

---

## 3. Initialise the database (one-time)

```powershell
.\04-init-db.ps1
```

Creates user `postgres` / db `rainmaker` and applies `db\init.sql`.

---

## 4. Start every Java + Node + Go service

```powershell
.\05-start-services.ps1
```

Launches all 28 application services in dependency order with port + env
overrides (per the Dockerfile supervisord block). Each service runs in its own
hidden process and writes to `local-deploy\logs\<svc>.log`. Cold start takes
2-3 minutes for all JVMs to register on their ports.

Watch progress:

```powershell
Get-Content .\logs\ws-services.log -Wait -Tail 50
```

---

## 5. Smoke-test the WS workflow end-to-end

```powershell
.\07-smoke-test.ps1
```

Walks the full flow: create property → create WS connection → search →
estimate → workflow transitions → demand → payment → CONNECTION_ACTIVATED.
Prints `OK` per step.

Manual curl of the headline endpoints:

```powershell
curl http://localhost:8090/health
curl http://localhost:8091/health
curl http://localhost:8094/egov-mdms-service/v1/_search
curl http://localhost:8280/property-services/property/_search?tenantId=pb
```

Postman: import `municipal-services-go\postman\ws-services.postman_collection.json`
and `ws-calculator.postman_collection.json`. Set `host=localhost`. Run in order.

---

## 6. Stop everything

```powershell
.\06-stop-all.ps1
```

Kills the JVMs, Node, Go binaries, Kafka, ZooKeeper, then stops the Postgres
service.

---

## Port map

| Service | Port |   | Service | Port |
|---|---|---|---|---|
| postgres | 5432 |   | egov-pg-service | 8097 |
| zookeeper | 2181 |   | egov-searcher | 8098 |
| kafka | 9092 |   | egov-url-shortening | 8099 |
| pdf-service | 8080 |   | tenant | 8200 |
| egov-user | 8081 |   | user-otp | 8201 |
| egov-persister | 8082 |   | billing-service | 8202 |
| egov-filestore | 8083 |   | collection-services | 8203 |
| egov-location | 8084 |   | egov-apportion-service | 8204 |
| egov-accesscontrol | 8085 |   | property-services | 8280 |
| egov-common-masters | 8086 |   | pt-calculator-v2 | 8281 |
| egov-localization | 8087 |   | egov-workflow-v2 | 8290 |
| egov-idgen | 8088 |   | ws-services | 8090 |
| egov-enc-service | 8089 |   | ws-calculator | 8091 |
| egov-indexer | 8092 |   | egov-mdms-service | 8094 |
| egov-notification-mail | 8093 |   |   |   |
| egov-notification-sms | 8095 |   |   |   |
| egov-otp | 8096 |   |   |   |

---

## Troubleshooting

- **Port already in use** → another service is bound. `Get-NetTCPConnection -LocalPort 8094` then `Stop-Process -Id <pid>`.
- **JVM OOM** → bump `-Xmx` in `05-start-services.ps1`, default `128m` per service.
- **mvn fails: cannot resolve eGov dependency** → `maven-settings.xml` already routes to Maven Central; if a specific eGov artifact is missing, that one service won't compile but others will. Acceptable for WS-flow testing.
- **psql: password authentication failed** → reset with `ALTER USER postgres PASSWORD 'postgres';` from the `postgresql-x64-15` setup wizard, then re-run `04-init-db.ps1`.
- **ws-services: kafka subscribe skipped: empty topic** → expected for unset env topics; not a fault.
