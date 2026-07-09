# Detailed Essential Services Workflow (Minimal Setup)

This document maps the architectural flow of data between the minimal **essential services** currently running in your local DIGIT environment. It explains exactly what happens under the hood when you execute the Postman APIs.

The essential services running are:
`egov-user`, `egov-mdms-service`, `egov-idgen`, `property-services`, `egov-workflow-v2`, `ws-services`, and `ws-calculator`.

---

## Step 0: User Context & Authentication Mock
**API**: All endpoints (via `RequestInfo`)
**Involved services**: Postman → `egov-user`

**What happens:**
In a full deployment, the UI calls `egov-user` to authenticate and retrieve a token. In our minimal local testing, Postman mocks this by passing the `RequestInfo.userInfo` object in every request. 
The minimal setup relies on the seeded employee UUID (`a713e6ad-892d-421f-a52d-477c0ea9278a`) which has the `WS_CEMP` and `PT_CEMP` roles. 

**How the next services use this:**
Every subsequent microservice (`property-services`, `ws-services`, `egov-workflow-v2`) extracts this UUID and its roles to:
1. Verify Role-Based Access Control (RBAC).
2. Populate `auditDetails.createdBy` and `lastModifiedBy` fields.
3. Allow `egov-workflow-v2` to record exactly *who* initiated the workflow transition.

---

## Step 1: Create Property
**API**: `POST /property-services/property/_create`
**Involved services**: Postman → `property-services` → `egov-idgen` → Kafka → `egov-persister` → PostgreSQL

**What happens:**
A water connection requires an active property. 
1. **Property Creation**: Postman sends the property payload to `property-services`.
2. **ID Generation**: `property-services` calls `egov-idgen` to generate a unique `propertyId` (e.g., `PB-PT-2026-05-21-000018`) and an acknowledgment number.
3. **Persistence Flow**: `property-services` does *not* write directly to PostgreSQL. Instead, it pushes a message to the Kafka topic `save-property-registry`.
4. The background `egov-persister` service reads this Kafka topic and executes the SQL `INSERT` statements into the `eg_pt_property` and related tables.

**Why it matters for WS:**
`ws-services` strictly requires an existing, `ACTIVE` property. Because of the asynchronous Kafka persister, you must wait ~3 seconds after creating a property before attempting to link a water connection to it.

---

## Step 2: Create Water Connection Application
**API**: `POST /ws-services/wc/_create`
**Involved services**: Postman → `ws-services` → `property-services` & `egov-mdms-service`

**What happens inside `ws-services`:**
1. **Property Verification**: `ws-services` extracts the `propertyId` from the payload and internally calls `property-services/_search`. 
   - It validates that the property exists.
   - It validates that the property status is `ACTIVE`.
   - It validates the property's `usageCategory` (crucial for calculation slabs later).
2. **MDMS Validation**: `ws-services` internally calls `egov-mdms-service` to verify that the `connectionType` (e.g., Non-Metered) and `waterSource` are valid configuration values for the `pb.amritsar` tenant.

---

## Step 3: Application Enrichment and ID Generation
**Involved services**: `ws-services` → `egov-idgen`

**What happens:**
Once validation passes, `ws-services` prepares the record for saving:
1. Generates an internal UUID for the water connection database record.
2. Creates `auditDetails` using the `RequestInfo.userInfo`.
3. Sets `applicationType = NEW_WATER_CONNECTION`.
4. Calls `egov-idgen` to generate the human-readable `applicationNo` (e.g., `WS_AP/107/2026-27/000006`).

---

## Step 4: Workflow Creation & Initiation
**Involved services**: `ws-services` → `egov-workflow-v2`

**What happens:**
The water connection payload explicitly contains:
`"processInstance": { "action": "INITIATE" }`

1. `ws-services` calls the `egov-workflow-v2/egov-wf/process/_transition` API.
2. It passes `businessId = applicationNo`, `businessService = NewWS1`, and `action = INITIATE`.
3. `egov-workflow-v2` checks the `NewWS1` state machine. Since it's a new application, the state is `null`. The action `INITIATE` transitions the state to `INITIATED`.
4. `egov-workflow-v2` verifies that the user (`a713e6ad-892d-421f-a52d-477c0ea9278a`) has the `WS_CEMP` role required for this transition by calling `egov-user/_search`.
5. `egov-workflow-v2` records the workflow history and returns the new state.
6. `ws-services` updates the `applicationStatus` of the water connection to `INITIATED`.

---

## Step 5: Connection Persistence Flow
**Involved services**: `ws-services` → Kafka → `egov-persister` → PostgreSQL

**What happens:**
1. `ws-services` pushes the enriched WaterConnection object (now containing the `applicationNo` and `applicationStatus`) to the Kafka topic `save-ws-connection`.
2. `ws-services` responds `HTTP 200 OK` back to Postman.
3. Asynchronously, `egov-persister` listens to `save-ws-connection` and saves the records into PostgreSQL (`eg_ws_connection`, `eg_ws_connection_holder`, etc.).

---

## Step 6: Fee Estimation / Demand Generation
**API**: `POST /ws-calculator/waterCalculator/_estimate`
**Involved services**: Postman → `ws-calculator` → `ws-services` → `egov-mdms-service`

**What happens:**
1. Postman requests an estimation by passing the `applicationNo`.
2. `ws-calculator` searches for the application details by calling `ws-services/wc/_search`.
3. `ws-calculator` retrieves master billing data slabs and tax heads from `egov-mdms-service`.
4. Based on the `propertyType`, `usageCategory` (from the property), `connectionType`, and `pipeSize`, the calculator runs its calculation engine.
5. It returns a detailed `Calculation` JSON response showing the breakdown of Scrutiny Fees, Form Fees, and initial taxes.

*(Note: Because `billing-service` is excluded in `-Minimal` mode, the actual `Demand` objects are not stored in the billing database, but the calculator successfully completes its mathematical estimation).*
