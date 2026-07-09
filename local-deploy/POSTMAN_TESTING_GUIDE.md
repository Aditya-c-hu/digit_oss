# DIGIT End-to-End Postman Testing Guide

This guide provides step-by-step instructions on how to test the end-to-end local workflow of DIGIT services using **Postman**. It covers data flow starting from `egov-user` validation, extending into `property-services`, then `ws-services`, and concluding with `ws-calculator`.

### Prerequisites
- The local backend testing environment must be running (e.g., `.\05-start-services.ps1 -Minimal`).
- Ensure Docker infrastructure (Kafka, PostgreSQL) is active.
- Ensure the database patches (User Type update to EMPLOYEE) were applied correctly.

> [!NOTE]
> All endpoints use **POST** requests with the `Content-Type: application/json` header.

---

## Step 1: Verify the EMPLOYEE User (`egov-user`)
First, confirm that the seeded employee user (`a713e6ad-892d-421f-a52d-477c0ea9278a`) exists and is retrievable. The Workflow service relies on this user to perform state transitions.

- **Method**: `POST`
- **URL**: `http://localhost:8081/user/_search`
- **Headers**: `Content-Type: application/json`
- **Body** (raw JSON):
```json
{
  "requestInfo": {
    "apiId": "ws-services",
    "ver": "1.0",
    "action": "_search",
    "msgId": "20170310130900|en_IN",
    "authToken": "test-token"
  },
  "uuid": ["a713e6ad-892d-421f-a52d-477c0ea9278a"],
  "tenantId": "pb.amritsar"
}
```
**Expected Outcome**: You should receive an `HTTP 200 OK` response with an array containing the user details, verifying that the user and its encrypted fields are accessible.

---

## Step 2: Create a Property (`property-services`)
Water Connections require an existing property. This step creates a new dummy property and registers it in the system.

- **Method**: `POST`
- **URL**: `http://localhost:8280/property-services/property/_create`
- **Body** (raw JSON):
```json
{
  "RequestInfo": {
    "apiId": "asset-services",
    "ver": "1.0",
    "action": "_create",
    "did": "1",
    "msgId": "20170310130900|en_IN",
    "authToken": "test-token",
    "userInfo": {
      "id": 1,
      "uuid": "a713e6ad-892d-421f-a52d-477c0ea9278a",
      "userName": "EMPLOYEE",
      "type": "EMPLOYEE",
      "tenantId": "pb.amritsar",
      "roles": [
        { "code": "PT_CEMP", "tenantId": "pb.amritsar" },
        { "code": "WS_CEMP", "tenantId": "pb.amritsar" }
      ]
    }
  },
  "Property": {
    "tenantId": "pb.amritsar",
    "propertyType": "BUILTUP.INDEPENDENTPROPERTY",
    "ownershipCategory": "INDIVIDUAL.SINGLEOWNER",
    "usageCategory": "RESIDENTIAL",
    "noOfFloors": 1,
    "landArea": 100,
    "superBuiltUpArea": 100,
    "address": {
      "tenantId": "pb.amritsar",
      "doorNo": "12",
      "buildingName": "Test House",
      "street": "Test Street",
      "locality": { "code": "SUN04", "name": "Ajit Nagar" },
      "city": "Amritsar",
      "pincode": "143001"
    },
    "owners": [
      {
        "name": "Test Owner",
        "mobileNumber": "9999999999",
        "fatherOrHusbandName": "Father",
        "relationship": "FATHER",
        "gender": "MALE",
        "ownerType": "NONE",
        "ownerShipPercentage": 100,
        "tenantId": "pb.amritsar"
      }
    ],
    "creationReason": "CREATE",
    "source": "MUNICIPAL_RECORDS",
    "channel": "CFC_COUNTER"
  }
}
```
**Expected Outcome**: `HTTP 201 Created`. In the response, look for the generated `propertyId` (e.g., `PB-PT-2026-05-...`). **Copy this `propertyId` for the next step.**

> [!WARNING]
> Wait approximately 3 seconds after this API call before creating the water connection. The `egov-persister` runs asynchronously via Kafka to commit the property to the Postgres database.

---

## Step 3: Initialize Water Connection (`ws-services`)
Now, initiate a new water connection attached to the property you just created.

- **Method**: `POST`
- **URL**: `http://localhost:8090/ws-services/wc/_create`
- **Body** (raw JSON):
*(Replace `REPLACE_WITH_PROPERTY_ID` with the ID from Step 2)*
```json
{
  "RequestInfo": {
    "apiId": "ws-services",
    "ver": "1.0",
    "action": "_create",
    "msgId": "20170310130900|en_IN",
    "authToken": "test-token",
    "userInfo": {
      "id": 1,
      "uuid": "a713e6ad-892d-421f-a52d-477c0ea9278a",
      "userName": "EMPLOYEE",
      "type": "EMPLOYEE",
      "tenantId": "pb.amritsar",
      "roles": [
        { "code": "WS_CEMP", "tenantId": "pb.amritsar" }
      ]
    }
  },
  "WaterConnection": {
    "tenantId": "pb.amritsar",
    "propertyId": "REPLACE_WITH_PROPERTY_ID",
    "applicationType": "NEW_WATER_CONNECTION",
    "connectionType": "Non Metered",
    "waterSource": "GROUND.BOREWELL",
    "noOfTaps": 2,
    "pipeSize": 15.0,
    "proposedTaps": 2,
    "proposedPipeSize": 15.0,
    "channel": "CFC_COUNTER",
    "additionalDetails": {},
    "processInstance": { "action": "INITIATE", "comment": "Initial submit via Postman" }
  }
}
```
**Expected Outcome**: `HTTP 200 OK`. The `egov-workflow-v2` service will transition the application state. In the response JSON, look for the generated `applicationNo` (e.g., `WS_AP/107/...`). **Copy this `applicationNo` for the next steps.**

---

## Step 4: Verify Water Connection Status (`ws-services`)
Search for the water connection to verify its state and details.

- **Method**: `POST`
- **URL**: `http://localhost:8090/ws-services/wc/_search?tenantId=pb.amritsar&applicationNumber=REPLACE_WITH_APPLICATION_NO`
*(Replace `REPLACE_WITH_APPLICATION_NO` in the URL)*
- **Body** (raw JSON):
```json
{
  "RequestInfo": {
    "apiId": "ws",
    "ver": "1.0",
    "action": "_search",
    "authToken": "test-token"
  }
}
```
**Expected Outcome**: `HTTP 200 OK`. The response will show the connection details and confirm its workflow state (e.g., `INITIATED`).

---

## Step 5: Estimate Connection Fees (`ws-calculator`)
Send the application number to the `ws-calculator` service to estimate the fees based on MDMS billing slabs and rules.

- **Method**: `POST`
- **URL**: `http://localhost:8091/ws-calculator/waterCalculator/_estimate`
- **Body** (raw JSON):
*(Replace `REPLACE_WITH_APPLICATION_NO` with the ID from Step 3)*
```json
{
  "RequestInfo": {
    "apiId": "ws-calculator",
    "ver": "1.0",
    "action": "_estimate",
    "authToken": "test-token",
    "userInfo": {
      "id": 1,
      "uuid": "a713e6ad-892d-421f-a52d-477c0ea9278a",
      "userName": "EMPLOYEE",
      "type": "EMPLOYEE",
      "tenantId": "pb.amritsar",
      "roles": [{ "code": "WS_CEMP", "tenantId": "pb.amritsar" }]
    }
  },
  "CalculationCriteria": [
    {
      "tenantId": "pb.amritsar",
      "applicationNo": "REPLACE_WITH_APPLICATION_NO",
      "connectionNo": null,
      "from": 0,
      "to": 0
    }
  ]
}
```
**Expected Outcome**: `HTTP 200 OK`. The response JSON will contain a `Calculation` array breaking down the exact fee structure (tax amounts, estimates, and rebates) derived from the local master data service (`egov-mdms-service`).
