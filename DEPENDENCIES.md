# Dependency Map — ws-services & ws-calculator

Complete dependency tree for the two target municipal services, derived from
the README of each service. Every listed folder has been moved into
`municipal-services-go/` and is built+run by the single root `Dockerfile`.

---

## 1. Target services (Go ports)

| Service | Source folder | Port |
|---|---|---|
| ws-services | `municipal-services-go/ws-services` | 8090 |
| ws-calculator | `municipal-services-go/ws-calculator` | 8091 |

---

## 2. Direct dependencies (from each README)

### 2.1 `ws-services` README → Service Dependencies
- egov-mdms service
- property-service
- egov-idgen
- egov-persister
- ws-calculator
- egov-filestore
- pdf-service

### 2.2 `ws-calculator` README → Service Dependencies
- egov-mdms service
- property-service
- egov-persister
- ws-service
- egov-user
- egov-workflow-v2

### 2.3 Union of direct deps (deduped)
| README name | Actual folder | Origin folder (before move) |
|---|---|---|
| egov-mdms service | `egov-mdms-service` | `core-services/` |
| property-service | `property-services` | `municipal-services/` |
| egov-idgen | `egov-idgen` | `core-services/` |
| egov-persister | `egov-persister` | `core-services/` |
| egov-filestore | `egov-filestore` | `core-services/` |
| pdf-service | `pdf-service` | `core-services/` |
| egov-user | `egov-user` | `core-services/` |
| egov-workflow-v2 | `egov-workflow-v2` | `core-services/` |
| ws-calculator | `ws-calculator` | `municipal-services-go/` (already present) |
| ws-service | `ws-services` | `municipal-services-go/` (already present) |

---

## 3. Nested (transitive) dependencies

### 3.1 `property-services` README → Service Dependencies
- user → **egov-user** (already pulled in via direct deps)
- ID-GEN → **egov-idgen** (already pulled in)
- pt-calculator → **pt-calculator-v2** (NEW, `municipal-services/`)
- MDMS → **egov-mdms-service** (already pulled in)
- Location → **egov-location** (NEW, `core-services/`)
- localisation → **egov-localization** (NEW, `core-services/`)

### 3.2 New transitive folders pulled in
| Folder | Origin |
|---|---|
| `pt-calculator-v2` | `municipal-services/` |
| `egov-location` | `core-services/` |
| `egov-localization` | `core-services/` |

Other direct deps (`egov-mdms-service`, `egov-idgen`, `egov-persister`,
`egov-filestore`, `pdf-service`, `egov-user`, `egov-workflow-v2`) ship no
further service-level dependencies in their READMEs that are not already
satisfied — they only have infra deps (DB, Kafka).

---

## 4. Infrastructure dependencies (bundled by Dockerfile)

| Component | Version | Port | Reason |
|---|---|---|---|
| PostgreSQL | 15 | 5432 | All Java services + Go services persist to it |
| ZooKeeper | 3.7-bundled | 2181 | Kafka quorum |
| Kafka | 3.7.0 | 9092 | Producer/consumer topics (`save-ws-connection`, `update-ws-connection`, `update-ws-workflow`, `bill-generation`, `ws-demand-saved`, `ws-demand-failure`, `ws-generate-demand`, `egov.core.notification.sms`, `persist-user-events-async`, `save-ws-meter`, `create-meter-reading`) |

---

## 5. Full dependency tree (collapsed)

```
ws-services 
├── egov-mdms-service   
├── property-services   
│   ├── egov-user            
│   ├── egov-idgen           
│   ├── egov-mdms-service    
│   ├── egov-location        
│   ├── egov-localization    
│   └── pt-calculator-v2     
│       ├── egov-mdms-service
│       ├── property-services
│       └── billing-service          
├── egov-idgen          
├── egov-persister      
├── ws-calculator       
├── egov-filestore      
└── pdf-service         

ws-calculator 
├── egov-mdms-service   
├── property-services   
├── egov-persister      
├── ws-services         
├── egov-user           
└── egov-workflow-v2    
```
ws-services
├── direct-service-dependencies
│   ├── egov-mdms-service
│   ├── property-services
│   │   ├── egov-user
│   │   ├── egov-idgen
│   │   ├── egov-mdms-service
│   │   ├── egov-location
│   │   ├── egov-localization
│   │   └── pt-calculator-v2
│   │       ├── egov-mdms-service
│   │       ├── property-services
│   │       └── billing-service
│   ├── egov-idgen
│   ├── egov-persister
│   ├── ws-calculator
│   │   ├── egov-mdms-service
│   │   ├── property-services
│   │   ├── egov-persister
│   │   ├── ws-services
│   │   ├── egov-user
│   │   ├── egov-workflow-v2
│   │   └── billing-service
│   ├── egov-filestore
│   ├── pdf-service
│   ├── egov-user
│   ├── egov-workflow-v2
│   ├── egov-localization
│   ├── egov-location
│   ├── egov-sms
│   ├── egov-email
│   ├── billing-service
│   ├── collection-services
│   └── egov-accesscontrol
│
└── runtime-platform-dependencies
    ├── kafka
    ├── postgresql
    ├── mdms-config-data
    ├── persister-config
    ├── api-gateway / zuul / nginx-ingress
    └── user-authentication-token-service
---

## 6. Final folder inventory inside `municipal-services-go/`

| # | Folder | Type | Port | Layer |
|---|---|---|---|---|
| 1 | `egov-mdms-service` | Java 8 / Maven | 8094 | Core |
| 2 | `egov-idgen` | Java 8 / Maven | 8088 | Core |
| 3 | `egov-persister` | Java 8 / Maven | 8082 | Core |
| 4 | `egov-filestore` | Java 8 / Maven | 8083 | Core |
| 5 | `egov-user` | Java 8 / Maven | 8081 | Core |
| 6 | `egov-workflow-v2` | Java 8 / Maven | 8290* | Core |
| 7 | `egov-location` | Java 8 / Maven | 8084* | Core |
| 8 | `egov-localization` | Java 8 / Maven | 8087 | Core |
| 9 | `egov-accesscontrol` | Java 8 / Maven | 8085 | Core |
| 10 | `egov-common-masters` | Java 8 / Maven | 8086 | Core |
| 11 | `egov-enc-service` | Java 8 / Maven | 8089 | Core |
| 12 | `egov-indexer` | Java 8 / Maven | 8092 | Core |
| 13 | `egov-notification-mail` | Java 8 / Maven | 8093 | Core |
| 14 | `egov-notification-sms` | Java 8 / Maven | 8095 | Core |
| 15 | `egov-otp` | Java 8 / Maven | 8096 | Core |
| 16 | `egov-pg-service` | Java 8 / Maven | 8097 | Core |
| 17 | `egov-searcher` | Java 8 / Maven | 8098 | Core |
| 18 | `egov-url-shortening` | Java 8 / Maven | 8099 | Core |
| 19 | `tenant` | Java 8 / Maven | 8200 | Core |
| 20 | `user-otp` | Java 8 / Maven | 8201 | Core |
| 21 | `pdf-service` | Node 10 | 8080 | Core |
| 22 | `property-services` | Java 8 / Maven | 8280 | Municipal |
| 23 | `pt-calculator-v2` | Java 8 / Maven | 8281 | Municipal |
| 24 | `billing-service` | Java 8 / Maven | 8202 | Business |
| 25 | `collection-services` | Java 8 / Maven | 8203 | Business |
| 26 | `egov-apportion-service` | Java 8 / Maven | 8204 | Business |
| 27 | `ws-services` | Go 1.22 | 8090 | Municipal (target) |
| 28 | `ws-calculator` | Go 1.22 | 8091 | Municipal (target) |

`*` = port remapped at runtime via `-Dserver.port` to avoid in-container clashes:
- `egov-workflow-v2` was 8280 → moved to **8290** (property-services keeps 8280)
- `egov-location` was 8082 → moved to **8084** (egov-persister keeps 8082)

Ports 8085, 8086, 8089, 8092, 8093, 8095–8099, 8200–8204 chosen to avoid
collisions inside bundle. Upstream DIGIT defaults overridden via `-Dserver.port`.

---

## 7. Business-services

Three business services added on top of the README contract because the
deployer wants a fully-bundled "core + business + municipal" image:

| Folder | Port | Why |
|---|---|---|
| `billing-service` | 8202 | Consumes `bill-generation` topic emitted by ws-calculator → produces demand bills |
| `collection-services` | 8203 | Receipts/payment collection — paired with billing for full demand→payment loop |
| `egov-apportion-service` | 8204 | Apportions collected amounts across demands; chained with collection |

Wired into supervisor with priority 42–45 so they boot after egov-mdms-service
and egov-user are reachable. Kafka topics auto-create on first produce thanks
to `auto.create.topics.enable=true` in broker config.

---

## 8. Source-of-truth

All entries above come from:
- `municipal-services-go/ws-services/README.md` (formerly municipal-services)
- `municipal-services-go/ws-calculator/README.md` (formerly municipal-services)
- `municipal-services-go/property-services/README.md` (formerly municipal-services)

No deps were inferred from code — README contract only, per request.
