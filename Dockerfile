# =============================================================================
# DIGIT — single-file deployment of ws-services + ws-calculator (Go ports)
# bundled with every core-service dep (Java/Node) AND the runtime deps
# (PostgreSQL, ZooKeeper, Kafka) needed to exercise the full stack.
#
# Build : docker build -t digit-ws-bundle .
# Run   : docker run -d --name digit-ws \
#            -p 8080-8094:8080-8094 -p 8280:8280 -p 8281:8281 -p 8290:8290 \
#            -p 5432:5432 -p 9092:9092 -p 2181:2181 digit-ws-bundle
#
# Services bundled (28 total):
#   Java core:  egov-mdms-service, egov-idgen, egov-persister, egov-filestore,
#               egov-user, egov-workflow-v2, egov-location, egov-localization,
#               egov-accesscontrol, egov-common-masters, egov-enc-service,
#               egov-indexer, egov-notification-mail, egov-notification-sms,
#               egov-otp, egov-pg-service, egov-searcher, egov-url-shortening,
#               tenant, user-otp
#   Java muni:  property-services, pt-calculator-v2
#   Java biz:   billing-service, collection-services, egov-apportion-service
#   Node:       pdf-service
#   Go:         ws-services, ws-calculator
#   Infra:      PostgreSQL 15, ZooKeeper, Kafka
#
# NOTE: one image bundling 28 services + Postgres + Kafka is intentionally
# monolithic — kept for "single Dockerfile" requirement. Production: split.
# =============================================================================


# -----------------------------------------------------------------------------
# Stage 1 — Build all Java services with Maven (JDK 8)
# -----------------------------------------------------------------------------
FROM maven:3.8.6-openjdk-8-slim AS java-build
WORKDIR /build

# Maven settings: ensures Maven Central is tried first (eGov Nexus is down)
COPY maven-settings.xml /build/maven-settings.xml

# Each Java service: copy pom + src, package skipping tests.
COPY egov-mdms-service/pom.xml egov-mdms-service/pom.xml
COPY egov-mdms-service/src     egov-mdms-service/src
RUN mvn -U -s /build/maven-settings.xml -f egov-mdms-service/pom.xml clean package -DskipTests -B

COPY egov-idgen/pom.xml egov-idgen/pom.xml
COPY egov-idgen/src     egov-idgen/src
RUN mvn -U -s /build/maven-settings.xml -f egov-idgen/pom.xml clean package -DskipTests -B

COPY egov-persister/pom.xml egov-persister/pom.xml
COPY egov-persister/src     egov-persister/src
RUN mvn -U -s /build/maven-settings.xml -f egov-persister/pom.xml clean package -DskipTests -B

COPY egov-filestore/pom.xml egov-filestore/pom.xml
COPY egov-filestore/src     egov-filestore/src
RUN mvn -U -s /build/maven-settings.xml -f egov-filestore/pom.xml clean package -DskipTests -B

COPY egov-user/pom.xml egov-user/pom.xml
COPY egov-user/src     egov-user/src
RUN mvn -U -s /build/maven-settings.xml -f egov-user/pom.xml clean package -DskipTests -B

COPY egov-workflow-v2/pom.xml egov-workflow-v2/pom.xml
COPY egov-workflow-v2/src     egov-workflow-v2/src
RUN mvn -U -s /build/maven-settings.xml -f egov-workflow-v2/pom.xml clean package -DskipTests -B

COPY egov-location/pom.xml egov-location/pom.xml
COPY egov-location/src     egov-location/src
RUN mvn -U -s /build/maven-settings.xml -f egov-location/pom.xml clean package -DskipTests -B

COPY egov-localization/pom.xml egov-localization/pom.xml
COPY egov-localization/src     egov-localization/src
RUN mvn -U -s /build/maven-settings.xml -f egov-localization/pom.xml clean package -DskipTests -B

COPY property-services/pom.xml property-services/pom.xml
COPY property-services/src     property-services/src
RUN mvn -U -s /build/maven-settings.xml -f property-services/pom.xml clean package -DskipTests -B

COPY pt-calculator-v2/pom.xml pt-calculator-v2/pom.xml
COPY pt-calculator-v2/src     pt-calculator-v2/src
RUN mvn -U -s /build/maven-settings.xml -f pt-calculator-v2/pom.xml clean package -DskipTests -B

# --- Extra core services ----------------------------------------------------
COPY egov-accesscontrol/pom.xml egov-accesscontrol/pom.xml
COPY egov-accesscontrol/src     egov-accesscontrol/src
RUN mvn -U -s /build/maven-settings.xml -f egov-accesscontrol/pom.xml clean package -DskipTests -B

COPY egov-common-masters/pom.xml egov-common-masters/pom.xml
COPY egov-common-masters/src     egov-common-masters/src
RUN mvn -U -s /build/maven-settings.xml -f egov-common-masters/pom.xml clean package -DskipTests -B

COPY egov-enc-service/pom.xml egov-enc-service/pom.xml
COPY egov-enc-service/src     egov-enc-service/src
RUN mvn -U -s /build/maven-settings.xml -f egov-enc-service/pom.xml clean package -DskipTests -B

COPY egov-indexer/pom.xml egov-indexer/pom.xml
COPY egov-indexer/src     egov-indexer/src
RUN mvn -U -s /build/maven-settings.xml -f egov-indexer/pom.xml clean package -DskipTests -B

COPY egov-notification-mail/pom.xml egov-notification-mail/pom.xml
COPY egov-notification-mail/src     egov-notification-mail/src
RUN mvn -U -s /build/maven-settings.xml -f egov-notification-mail/pom.xml clean package -DskipTests -B

COPY egov-notification-sms/pom.xml egov-notification-sms/pom.xml
COPY egov-notification-sms/src     egov-notification-sms/src
RUN mvn -U -s /build/maven-settings.xml -f egov-notification-sms/pom.xml clean package -DskipTests -B

COPY egov-otp/pom.xml egov-otp/pom.xml
COPY egov-otp/src     egov-otp/src
RUN mvn -U -s /build/maven-settings.xml -f egov-otp/pom.xml clean package -DskipTests -B

COPY egov-pg-service/pom.xml egov-pg-service/pom.xml
COPY egov-pg-service/src     egov-pg-service/src
RUN mvn -U -s /build/maven-settings.xml -f egov-pg-service/pom.xml clean package -DskipTests -B

COPY egov-searcher/pom.xml egov-searcher/pom.xml
COPY egov-searcher/src     egov-searcher/src
RUN mvn -U -s /build/maven-settings.xml -f egov-searcher/pom.xml clean package -DskipTests -B

COPY egov-url-shortening/pom.xml egov-url-shortening/pom.xml
COPY egov-url-shortening/src     egov-url-shortening/src
RUN mvn -U -s /build/maven-settings.xml -f egov-url-shortening/pom.xml clean package -DskipTests -B

COPY tenant/pom.xml tenant/pom.xml
COPY tenant/src     tenant/src
RUN mvn -U -s /build/maven-settings.xml -f tenant/pom.xml clean package -DskipTests -B

COPY user-otp/pom.xml user-otp/pom.xml
COPY user-otp/src     user-otp/src
RUN mvn -U -s /build/maven-settings.xml -f user-otp/pom.xml clean package -DskipTests -B

# --- Business services ------------------------------------------------------
COPY billing-service/pom.xml billing-service/pom.xml
COPY billing-service/src     billing-service/src
RUN mvn -U -s /build/maven-settings.xml -f billing-service/pom.xml clean package -DskipTests -B

COPY collection-services/pom.xml collection-services/pom.xml
COPY collection-services/src     collection-services/src
RUN mvn -U -s /build/maven-settings.xml -f collection-services/pom.xml clean package -DskipTests -B

COPY egov-apportion-service/pom.xml egov-apportion-service/pom.xml
COPY egov-apportion-service/src     egov-apportion-service/src
RUN mvn -U -s /build/maven-settings.xml -f egov-apportion-service/pom.xml clean package -DskipTests -B


# -----------------------------------------------------------------------------
# Stage 2 — Build Go services (ws-services + ws-calculator)
# -----------------------------------------------------------------------------
FROM golang:1.26-alpine AS go-build
RUN apk add --no-cache git ca-certificates
WORKDIR /src

COPY ws-services/ ./ws-services/
WORKDIR /src/ws-services
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w" \
        -o /out/ws-services ./cmd/ws-services

WORKDIR /src
COPY ws-calculator/ ./ws-calculator/
WORKDIR /src/ws-calculator
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w" \
        -o /out/ws-calculator ./cmd/ws-calculator


# -----------------------------------------------------------------------------
# Stage 3 — Install pdf-service (Node 10) deps + babel build
# -----------------------------------------------------------------------------
FROM node:10-buster-slim AS node-build
WORKDIR /app
COPY pdf-service/package.json ./
RUN npm install --no-audit --no-fund || true
COPY pdf-service/ ./
RUN npm run build || true


# -----------------------------------------------------------------------------
# Stage 4 — Runtime: JRE 8 + Node 10 + Postgres 15 + ZooKeeper + Kafka
#                   + supervisord orchestrating every process
# -----------------------------------------------------------------------------
FROM eclipse-temurin:8-jre-jammy

ENV DEBIAN_FRONTEND=noninteractive \
    POSTGRES_USER=postgres \
    POSTGRES_PASSWORD=postgres \
    POSTGRES_DB=rainmaker \
    PGDATA=/var/lib/postgresql/data \
    KAFKA_VERSION=3.7.0 \
    SCALA_VERSION=2.13 \
    KAFKA_HOME=/opt/kafka

# ---- OS packages: postgres, node, supervisor, curl, gosu, git ----------------
# 1) Install base utilities first
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates curl gnupg lsb-release git supervisor gosu netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*

# 2) Add official PostgreSQL APT repo (PGDG) — Jammy doesn't ship pg-15
RUN curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc \
        | gpg --dearmor -o /usr/share/keyrings/pgdg.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/pgdg.gpg] http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
        > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update && apt-get install -y --no-install-recommends \
        postgresql-15 postgresql-contrib-15 \
    && rm -rf /var/lib/apt/lists/*

# 3) Add Node.js 18.x (Node 10 is EOL; 18 LTS is the closest compatible)
RUN curl -fsSL https://deb.nodesource.com/setup_18.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# ---- Kafka + ZooKeeper (bundled in same Kafka tarball) -----------------------
RUN curl -fsSL "https://archive.apache.org/dist/kafka/${KAFKA_VERSION}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz" \
        -o /tmp/kafka.tgz \
    && mkdir -p ${KAFKA_HOME} \
    && tar -xzf /tmp/kafka.tgz -C ${KAFKA_HOME} --strip-components=1 \
    && rm /tmp/kafka.tgz

# ---- App layout --------------------------------------------------------------
WORKDIR /app
RUN mkdir -p /app/jars /app/pdf-service /app/bin /app/mdms-data \
             /var/lib/postgresql/data /var/log/supervisor

# Java jars
COPY --from=java-build /build/egov-mdms-service/target/*.jar /app/jars/egov-mdms-service.jar
COPY --from=java-build /build/egov-idgen/target/*.jar        /app/jars/egov-idgen.jar
COPY --from=java-build /build/egov-persister/target/*.jar    /app/jars/egov-persister.jar
COPY --from=java-build /build/egov-filestore/target/*.jar    /app/jars/egov-filestore.jar
COPY --from=java-build /build/egov-user/target/*.jar         /app/jars/egov-user.jar
COPY --from=java-build /build/egov-workflow-v2/target/*.jar  /app/jars/egov-workflow-v2.jar
COPY --from=java-build /build/egov-location/target/*.jar     /app/jars/egov-location.jar
COPY --from=java-build /build/egov-localization/target/*.jar /app/jars/egov-localization.jar
COPY --from=java-build /build/property-services/target/*.jar /app/jars/property-services.jar
COPY --from=java-build /build/pt-calculator-v2/target/*.jar  /app/jars/pt-calculator-v2.jar

# Extra core jars
COPY --from=java-build /build/egov-accesscontrol/target/*.jar    /app/jars/egov-accesscontrol.jar
COPY --from=java-build /build/egov-common-masters/target/*.jar   /app/jars/egov-common-masters.jar
COPY --from=java-build /build/egov-enc-service/target/*.jar      /app/jars/egov-enc-service.jar
COPY --from=java-build /build/egov-indexer/target/*.jar          /app/jars/egov-indexer.jar
COPY --from=java-build /build/egov-notification-mail/target/*.jar /app/jars/egov-notification-mail.jar
COPY --from=java-build /build/egov-notification-sms/target/*.jar  /app/jars/egov-notification-sms.jar
COPY --from=java-build /build/egov-otp/target/*.jar              /app/jars/egov-otp.jar
COPY --from=java-build /build/egov-pg-service/target/*.jar       /app/jars/egov-pg-service.jar
COPY --from=java-build /build/egov-searcher/target/*.jar         /app/jars/egov-searcher.jar
COPY --from=java-build /build/egov-url-shortening/target/*.jar   /app/jars/egov-url-shortening.jar
COPY --from=java-build /build/tenant/target/*.jar                /app/jars/tenant.jar
COPY --from=java-build /build/user-otp/target/*.jar              /app/jars/user-otp.jar

# Business jars
COPY --from=java-build /build/billing-service/target/*.jar        /app/jars/billing-service.jar
COPY --from=java-build /build/collection-services/target/*.jar    /app/jars/collection-services.jar
COPY --from=java-build /build/egov-apportion-service/target/*.jar /app/jars/egov-apportion-service.jar

# Go binaries
COPY --from=go-build /out/ws-services   /app/bin/ws-services
COPY --from=go-build /out/ws-calculator /app/bin/ws-calculator

# Node pdf-service
COPY --from=node-build /app /app/pdf-service

# MDMS data — egov-mdms-service expects this repo at /app/egov-mdms-data
RUN git clone --depth=1 https://github.com/egovernments/egov-mdms-data.git /app/mdms-data || true

# DB bootstrap SQL (init.sql shipped with municipal-services-go)
COPY db/ /docker-entrypoint-initdb.d/

# -----------------------------------------------------------------------------
# Inline configs — Postgres, ZooKeeper, Kafka, Supervisor
# -----------------------------------------------------------------------------
RUN set -eux; \
    # Postgres: prepare data dir
    chown -R postgres:postgres /var/lib/postgresql /docker-entrypoint-initdb.d; \
    # Kafka data dirs
    mkdir -p /tmp/zookeeper /tmp/kafka-logs; \
    # ZooKeeper config
    printf '%s\n' \
        'dataDir=/tmp/zookeeper' \
        'clientPort=2181' \
        'maxClientCnxns=0' \
        'admin.enableServer=false' \
        > ${KAFKA_HOME}/config/zookeeper.properties; \
    # Kafka broker config
    printf '%s\n' \
        'broker.id=0' \
        'listeners=PLAINTEXT://0.0.0.0:9092' \
        'advertised.listeners=PLAINTEXT://localhost:9092' \
        'log.dirs=/tmp/kafka-logs' \
        'num.partitions=1' \
        'offsets.topic.replication.factor=1' \
        'transaction.state.log.replication.factor=1' \
        'transaction.state.log.min.isr=1' \
        'auto.create.topics.enable=true' \
        'zookeeper.connect=localhost:2181' \
        > ${KAFKA_HOME}/config/server.properties

# ---- Postgres init script (runs once at container start) ---------------------
RUN cat > /app/bin/init-postgres.sh <<'EOF' && chmod +x /app/bin/init-postgres.sh
#!/usr/bin/env bash
set -e
if [ ! -s "$PGDATA/PG_VERSION" ]; then
  echo "$POSTGRES_PASSWORD" > /tmp/pwfile
  chown postgres:postgres /tmp/pwfile
  gosu postgres /usr/lib/postgresql/15/bin/initdb -D "$PGDATA" \
      --username="$POSTGRES_USER" --pwfile=/tmp/pwfile
  rm -f /tmp/pwfile
  echo "host all all 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"
  echo "listen_addresses='*'"        >> "$PGDATA/postgresql.conf"
  gosu postgres /usr/lib/postgresql/15/bin/pg_ctl -D "$PGDATA" -w start
  gosu postgres psql -v ON_ERROR_STOP=1 --username="$POSTGRES_USER" <<-SQL
    CREATE DATABASE $POSTGRES_DB;
SQL
  for f in /docker-entrypoint-initdb.d/*.sql; do
    [ -f "$f" ] && gosu postgres psql -v ON_ERROR_STOP=1 \
        --username="$POSTGRES_USER" --dbname="$POSTGRES_DB" -f "$f" || true
  done
  gosu postgres /usr/lib/postgresql/15/bin/pg_ctl -D "$PGDATA" -m fast -w stop
fi
EOF

# ---- wait-for-port helper ----------------------------------------------------
RUN cat > /app/bin/wait-port.sh <<'EOF' && chmod +x /app/bin/wait-port.sh
#!/usr/bin/env bash
host="$1"; port="$2"; tries="${3:-120}"
for i in $(seq 1 "$tries"); do
  nc -z "$host" "$port" && exit 0
  sleep 2
done
echo "timeout waiting for $host:$port" >&2
exit 1
EOF

# ---- Supervisor config: every process under one supervisord ------------------
RUN cat > /etc/supervisor/conf.d/digit.conf <<'EOF'
[supervisord]
nodaemon=true
user=root
logfile=/var/log/supervisor/supervisord.log
pidfile=/var/run/supervisord.pid

# --- Infra ------------------------------------------------------------------
[program:postgres-init]
command=/app/bin/init-postgres.sh
autorestart=false
startsecs=0
priority=1
stdout_logfile=/var/log/supervisor/postgres-init.log
stderr_logfile=/var/log/supervisor/postgres-init.err

[program:postgres]
command=gosu postgres /usr/lib/postgresql/15/bin/postgres -D /var/lib/postgresql/data
autorestart=true
priority=5
stdout_logfile=/var/log/supervisor/postgres.log
stderr_logfile=/var/log/supervisor/postgres.err

[program:zookeeper]
command=/opt/kafka/bin/zookeeper-server-start.sh /opt/kafka/config/zookeeper.properties
autorestart=true
priority=10
stdout_logfile=/var/log/supervisor/zookeeper.log
stderr_logfile=/var/log/supervisor/zookeeper.err

[program:kafka]
command=bash -lc "/app/bin/wait-port.sh localhost 2181 60 && /opt/kafka/bin/kafka-server-start.sh /opt/kafka/config/server.properties"
autorestart=true
priority=15
stdout_logfile=/var/log/supervisor/kafka.log
stderr_logfile=/var/log/supervisor/kafka.err

# --- Core Java services -----------------------------------------------------
[program:egov-mdms-service]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8094 -jar /app/jars/egov-mdms-service.jar"
environment=MASTERS_PATH="/app/mdms-data"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-mdms-service.log
stderr_logfile=/var/log/supervisor/egov-mdms-service.err

[program:egov-idgen]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8088 -jar /app/jars/egov-idgen.jar"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-idgen.log
stderr_logfile=/var/log/supervisor/egov-idgen.err

[program:egov-persister]
command=bash -lc "/app/bin/wait-port.sh localhost 9092 90 && exec java -Xmx128m -Xms64m -Dserver.port=8082 -jar /app/jars/egov-persister.jar"
autorestart=true
priority=35
stdout_logfile=/var/log/supervisor/egov-persister.log
stderr_logfile=/var/log/supervisor/egov-persister.err

[program:egov-filestore]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8083 -jar /app/jars/egov-filestore.jar"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-filestore.log
stderr_logfile=/var/log/supervisor/egov-filestore.err

[program:egov-user]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8081 -jar /app/jars/egov-user.jar"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-user.log
stderr_logfile=/var/log/supervisor/egov-user.err

[program:egov-workflow-v2]
command=bash -lc "/app/bin/wait-port.sh localhost 8094 60 && exec java -Xmx128m -Xms64m -Dserver.port=8290 -jar /app/jars/egov-workflow-v2.jar"
autorestart=true
priority=40
stdout_logfile=/var/log/supervisor/egov-workflow-v2.log
stderr_logfile=/var/log/supervisor/egov-workflow-v2.err

[program:egov-location]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8084 -jar /app/jars/egov-location.jar"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-location.log
stderr_logfile=/var/log/supervisor/egov-location.err

[program:egov-localization]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8087 -jar /app/jars/egov-localization.jar"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-localization.log
stderr_logfile=/var/log/supervisor/egov-localization.err

[program:property-services]
command=bash -lc "/app/bin/wait-port.sh localhost 8094 60 && /app/bin/wait-port.sh localhost 8088 60 && /app/bin/wait-port.sh localhost 8081 60 && exec java -Xmx128m -Xms64m -Dserver.port=8280 -jar /app/jars/property-services.jar"
autorestart=true
priority=45
stdout_logfile=/var/log/supervisor/property-services.log
stderr_logfile=/var/log/supervisor/property-services.err

[program:pt-calculator-v2]
command=bash -lc "/app/bin/wait-port.sh localhost 8094 60 && /app/bin/wait-port.sh localhost 8280 60 && exec java -Xmx128m -Xms64m -Dserver.port=8281 -jar /app/jars/pt-calculator-v2.jar"
autorestart=true
priority=50
stdout_logfile=/var/log/supervisor/pt-calculator-v2.log
stderr_logfile=/var/log/supervisor/pt-calculator-v2.err

# --- Extra core Java services -----------------------------------------------
[program:egov-enc-service]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8089 -jar /app/jars/egov-enc-service.jar"
autorestart=true
priority=25
stdout_logfile=/var/log/supervisor/egov-enc-service.log
stderr_logfile=/var/log/supervisor/egov-enc-service.err

[program:tenant]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8200 -jar /app/jars/tenant.jar"
autorestart=true
priority=25
stdout_logfile=/var/log/supervisor/tenant.log
stderr_logfile=/var/log/supervisor/tenant.err

[program:egov-common-masters]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8086 -jar /app/jars/egov-common-masters.jar"
autorestart=true
priority=25
stdout_logfile=/var/log/supervisor/egov-common-masters.log
stderr_logfile=/var/log/supervisor/egov-common-masters.err

[program:egov-accesscontrol]
command=bash -lc "/app/bin/wait-port.sh localhost 8094 60 && exec java -Xmx128m -Xms64m -Dserver.port=8085 -jar /app/jars/egov-accesscontrol.jar"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-accesscontrol.log
stderr_logfile=/var/log/supervisor/egov-accesscontrol.err

[program:egov-otp]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8096 -jar /app/jars/egov-otp.jar"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-otp.log
stderr_logfile=/var/log/supervisor/egov-otp.err

[program:user-otp]
command=bash -lc "/app/bin/wait-port.sh localhost 8081 60 && /app/bin/wait-port.sh localhost 8096 60 && exec java -Xmx128m -Xms64m -Dserver.port=8201 -jar /app/jars/user-otp.jar"
autorestart=true
priority=35
stdout_logfile=/var/log/supervisor/user-otp.log
stderr_logfile=/var/log/supervisor/user-otp.err

[program:egov-url-shortening]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8099 -jar /app/jars/egov-url-shortening.jar"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-url-shortening.log
stderr_logfile=/var/log/supervisor/egov-url-shortening.err

[program:egov-searcher]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8098 -jar /app/jars/egov-searcher.jar"
autorestart=true
priority=30
stdout_logfile=/var/log/supervisor/egov-searcher.log
stderr_logfile=/var/log/supervisor/egov-searcher.err

[program:egov-notification-sms]
command=bash -lc "/app/bin/wait-port.sh localhost 9092 90 && exec java -Xmx128m -Xms64m -Dserver.port=8095 -jar /app/jars/egov-notification-sms.jar"
autorestart=true
priority=35
stdout_logfile=/var/log/supervisor/egov-notification-sms.log
stderr_logfile=/var/log/supervisor/egov-notification-sms.err

[program:egov-notification-mail]
command=bash -lc "/app/bin/wait-port.sh localhost 9092 90 && exec java -Xmx128m -Xms64m -Dserver.port=8093 -jar /app/jars/egov-notification-mail.jar"
autorestart=true
priority=35
stdout_logfile=/var/log/supervisor/egov-notification-mail.log
stderr_logfile=/var/log/supervisor/egov-notification-mail.err

[program:egov-indexer]
command=bash -lc "/app/bin/wait-port.sh localhost 9092 90 && exec java -Xmx128m -Xms64m -Dserver.port=8092 -jar /app/jars/egov-indexer.jar"
autorestart=true
priority=35
stdout_logfile=/var/log/supervisor/egov-indexer.log
stderr_logfile=/var/log/supervisor/egov-indexer.err

[program:egov-pg-service]
command=bash -lc "/app/bin/wait-port.sh localhost 5432 60 && exec java -Xmx128m -Xms64m -Dserver.port=8097 -jar /app/jars/egov-pg-service.jar"
autorestart=true
priority=40
stdout_logfile=/var/log/supervisor/egov-pg-service.log
stderr_logfile=/var/log/supervisor/egov-pg-service.err

# --- Business Java services -------------------------------------------------
[program:billing-service]
command=bash -lc "/app/bin/wait-port.sh localhost 8094 60 && /app/bin/wait-port.sh localhost 8081 60 && exec java -Xmx128m -Xms64m -Dserver.port=8202 -jar /app/jars/billing-service.jar"
autorestart=true
priority=42
stdout_logfile=/var/log/supervisor/billing-service.log
stderr_logfile=/var/log/supervisor/billing-service.err

[program:egov-apportion-service]
command=bash -lc "/app/bin/wait-port.sh localhost 8094 60 && exec java -Xmx128m -Xms64m -Dserver.port=8204 -jar /app/jars/egov-apportion-service.jar"
autorestart=true
priority=42
stdout_logfile=/var/log/supervisor/egov-apportion-service.log
stderr_logfile=/var/log/supervisor/egov-apportion-service.err

[program:collection-services]
command=bash -lc "/app/bin/wait-port.sh localhost 8202 90 && /app/bin/wait-port.sh localhost 8097 60 && exec java -Xmx128m -Xms64m -Dserver.port=8203 -jar /app/jars/collection-services.jar"
autorestart=true
priority=45
stdout_logfile=/var/log/supervisor/collection-services.log
stderr_logfile=/var/log/supervisor/collection-services.err

# --- Node pdf-service -------------------------------------------------------
[program:pdf-service]
directory=/app/pdf-service
command=bash -lc "/app/bin/wait-port.sh localhost 9092 90 && exec node --max-old-space-size=2048 dist"
environment=PORT="8080",DATA_CONFIG_URLS="https://raw.githubusercontent.com/egovernments/egov-pdf/master/data-config/receipt.json",FORMAT_CONFIG_URLS="https://raw.githubusercontent.com/egovernments/egov-pdf/master/format-config/receipt.json"
autorestart=true
priority=50
stdout_logfile=/var/log/supervisor/pdf-service.log
stderr_logfile=/var/log/supervisor/pdf-service.err

# --- Go municipal services --------------------------------------------------
[program:ws-services]
command=/app/bin/ws-services
environment=SERVER_PORT="8090",DB_HOST="localhost",DB_PORT="5432",DB_USER="postgres",DB_PASSWORD="postgres",DB_NAME="rainmaker",KAFKA_BROKERS="localhost:9092",KAFKA_GROUP_ID="egov-ws-services",IS_EXTERNAL_WORKFLOW_ENABLED="false"
autorestart=true
priority=60
stdout_logfile=/var/log/supervisor/ws-services.log
stderr_logfile=/var/log/supervisor/ws-services.err

[program:ws-calculator]
command=/app/bin/ws-calculator
environment=SERVER_PORT="8091",DB_HOST="localhost",DB_PORT="5432",DB_USER="postgres",DB_PASSWORD="postgres",DB_NAME="rainmaker",KAFKA_BROKERS="localhost:9092",KAFKA_GROUP_ID="egov-ws-calculator"
autorestart=true
priority=60
stdout_logfile=/var/log/supervisor/ws-calculator.log
stderr_logfile=/var/log/supervisor/ws-calculator.err
EOF

EXPOSE 5432 9092 2181 \
       8081 8082 8083 8084 8085 8086 8087 8088 8089 \
       8092 8093 8094 8095 8096 8097 8098 8099 \
       8200 8201 8202 8203 8204 \
       8280 8281 8290 \
       8080 8090 8091

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/supervisord.conf"]
