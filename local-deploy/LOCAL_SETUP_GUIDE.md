# DIGIT OSS Local Testing Environment Setup Guide

This document outlines the step-by-step instructions and commands required to set up, run, and test the local backend environment for DIGIT OSS microservices (specifically focusing on `ws-services` and `ws-calculator`), without deploying to a cloud or Kubernetes environment.

## Prerequisites
Ensure you have the following installed on your system:
- Java 8
- Go (1.20+)
- Maven
- PostgreSQL Client (`psql`)
- Docker Desktop
- PowerShell

---

## 1. Build the Microservices
Before running the services, you must compile and build the Java and Go microservices.
Navigate to the `local-deploy` directory and run the build script:
```powershell
# Compiles all Java services via Maven and Go services.
# The -ContinueOnError flag prevents harmless npm warnings (like old lockfiles) from crashing the PowerShell script.
.\02-build-all.ps1 -ContinueOnError
```

## 2. Start Infrastructure Services (Docker)
Instead of installing PostgreSQL and Kafka natively on your machine, you can run them using Docker. This ensures a clean and isolated environment.

Run the following commands in PowerShell:
```powershell
# Start PostgreSQL container on port 5433
docker run -d --name ws-postgres -e POSTGRES_PASSWORD=postgres -p 5433:5432 postgres:15-alpine

# Start Kafka container on port 9092
docker run -d --name ws-kafka -p 9092:9092 apache/kafka:3.7.0
```

## 3. Initialize the Database
Initialize the core databases and bootstrap the schema using the provided scripts:
```powershell
# Create the necessary databases in Postgres
.\04-init-db.ps1

# Execute DDL scripts to create schemas and tables
.\04-bootstrap-all-schemas.ps1
```

## 4. Apply Database Patches (Workflow & User Setup)
To ensure the end-to-end workflow functions correctly (especially ID generation and Workflow transitions), apply the following SQL patches inside the `rainmaker` database:

```powershell
# Connect to the PostgreSQL container
docker exec -it ws-postgres psql -U postgres -d rainmaker
```
Once inside the PostgreSQL shell, run:
```sql
-- Create missing sequences for ID Generator
CREATE SEQUENCE IF NOT EXISTS seq_ws_app_pb_amritsar;
CREATE SEQUENCE IF NOT EXISTS seq_ws_con_pb_amritsar;
CREATE SEQUENCE IF NOT EXISTS SEQ_WS_APP_PB_AMRITSAR;
CREATE SEQUENCE IF NOT EXISTS SEQ_WS_CON_PB_AMRITSAR;
CREATE SEQUENCE IF NOT EXISTS seq_ws_app_pb;
CREATE SEQUENCE IF NOT EXISTS seq_ws_con_pb;

-- Update a seeded CITIZEN user to act as an EMPLOYEE with correct workflow roles
UPDATE eg_user SET type = 'EMPLOYEE' WHERE id = 101;

-- Assign necessary workflow roles for Water Services to the user
INSERT INTO eg_userrole_v1 (role_code, role_tenantid, user_id, user_tenantid) VALUES 
('WS_CEMP', 'pb.amritsar', 101, 'pb'), 
('WS_DOC_VERIFIER', 'pb.amritsar', 101, 'pb'), 
('WS_FIELD_INSPECTOR', 'pb.amritsar', 101, 'pb'), 
('WS_APPROVER', 'pb.amritsar', 101, 'pb'), 
('WS_CLERK', 'pb.amritsar', 101, 'pb'), 
('WS_CEMP', 'pb', 101, 'pb'), 
('WS_DOC_VERIFIER', 'pb', 101, 'pb'), 
('WS_FIELD_INSPECTOR', 'pb', 101, 'pb'), 
('WS_APPROVER', 'pb', 101, 'pb'), 
('WS_CLERK', 'pb', 101, 'pb');
```

*(Note: You will also need to ensure the `NewWS1` workflow definition is registered in the `egov-workflow-v2` service if it isn't seeded automatically).*

## 5. Modify Sample Test Payloads
Update the smoke test JSON payloads located in `municipal-services-go\local-deploy\samples\` to use the properly configured user and workflow transition state.

- **In `property-create.json`, `ws-create.json`, and `ws-estimate.json`**:
  - Replace the placeholder `test-uuid-1` with `a713e6ad-892d-421f-a52d-477c0ea9278a` (the UUID of user 101).
- **In `ws-create.json`**:
  - Change `"action": "SUBMIT_APPLICATION"` to `"action": "INITIATE"` inside the `processInstance` object.

## 6. Start the Microservices
Start the essential microservices. We use the `-Minimal` flag to only start the services required for Water Services (WS) workflows, saving significant RAM.
```powershell
.\05-start-services.ps1 -Minimal
```
*Wait a few moments for the Java Spring Boot applications to fully initialize on their respective ports.*

## 7. Run the End-to-End Smoke Test
Execute the smoke test script to verify that MDMS search, Property Creation, Water Connection Creation, and Calculations all function together:
```powershell
.\07-smoke-test.ps1
```

## 8. Stop the Environment
Once you are done testing, you can cleanly stop all Java/Go processes and remove the Docker infrastructure containers:

```powershell
# Stop all Java and Go microservice background processes
.\06-stop-all.ps1

# Stop and remove the Docker infrastructure containers
docker stop ws-postgres ws-kafka
docker rm ws-postgres ws-kafka
```
