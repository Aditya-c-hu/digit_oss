-- =============================================================================
-- ws-services schema (Go port)
-- Canonical DDL for the tables the repository layer reads/writes. Mirrors the
-- DIGIT ws-services Flyway schema. `active` columns are character varying to
-- match the established DIGIT column types (the repo writes 'ACTIVE'/'INACTIVE').
-- Idempotent: safe to re-run.
-- =============================================================================

CREATE TABLE IF NOT EXISTS eg_ws_connection (
  id character varying(64) NOT NULL,
  tenantid character varying(250) NOT NULL,
  property_id character varying(64) NOT NULL,
  applicationno character varying(64),
  applicationstatus character varying(256),
  status character varying(64) NOT NULL DEFAULT 'Active',
  connectionno character varying(256),
  oldconnectionno character varying(64),
  roadcuttingarea numeric(12,2),
  action character varying(64),
  roadtype character varying(32),
  adhocrebate numeric(12,2),
  adhocpenalty numeric(12,2),
  adhocpenaltyreason character varying(1024),
  adhocpenaltycomment character varying(1024),
  adhocrebatereason character varying(1024),
  adhocrebatecomment character varying(1024),
  applicationtype character varying(64),
  channel character varying(64),
  dateeffectivefrom bigint,
  isoldapplication boolean DEFAULT false,
  locality character varying(256),
  disconnectionreason character varying(256),
  isdisconnectiontemporary boolean DEFAULT false,
  createdby character varying(64),
  lastmodifiedby character varying(64),
  createdtime bigint,
  lastmodifiedtime bigint,
  CONSTRAINT eg_ws_connection_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_eg_ws_connection_tenantid ON eg_ws_connection (tenantid);
CREATE INDEX IF NOT EXISTS idx_eg_ws_connection_applicationno ON eg_ws_connection (applicationno);
CREATE INDEX IF NOT EXISTS idx_eg_ws_connection_connectionno ON eg_ws_connection (connectionno);
CREATE INDEX IF NOT EXISTS idx_eg_ws_connection_property ON eg_ws_connection (property_id);

CREATE TABLE IF NOT EXISTS eg_ws_service (
  connection_id character varying(64) NOT NULL,
  connectioncategory character varying(32),
  rainwaterharvesting boolean,
  connectiontype character varying(32),
  watersource character varying(64),
  meterid character varying(64),
  meterinstallationdate bigint,
  pipesize decimal,
  nooftaps integer,
  connectionexecutiondate bigint,
  proposedpipesize decimal,
  proposedtaps integer,
  initialmeterreading numeric(12,3),
  appcreateddate bigint,
  detailsprovidedby character varying(256),
  estimationfilestoreid character varying(256),
  sanctionfilestoreid character varying(256),
  estimationletterdate bigint,
  disconnectionexecutiondate bigint,
  CONSTRAINT eg_ws_service_fk FOREIGN KEY (connection_id)
    REFERENCES eg_ws_connection(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_eg_ws_service_appcreated ON eg_ws_service (appcreateddate);

CREATE TABLE IF NOT EXISTS eg_ws_applicationdocument (
  id character varying(64) NOT NULL PRIMARY KEY,
  tenantid character varying(64),
  documenttype character varying(64),
  filestoreid character varying(64),
  wsid character varying(64),
  active character varying(16),
  documentuid character varying(64),
  createdby character varying(64),
  lastmodifiedby character varying(64),
  createdtime bigint,
  lastmodifiedtime bigint,
  CONSTRAINT fk_doc_conn FOREIGN KEY (wsid) REFERENCES eg_ws_connection(id)
);

CREATE TABLE IF NOT EXISTS eg_ws_plumberinfo (
  id character varying(64) NOT NULL PRIMARY KEY,
  name character varying(256),
  licenseno character varying(256),
  mobilenumber character varying(64),
  gender character varying(32),
  fatherorhusbandname character varying(256),
  correspondenceaddress character varying(1024),
  relationship character varying(64),
  wsid character varying(64),
  CONSTRAINT fk_plumber_conn FOREIGN KEY (wsid) REFERENCES eg_ws_connection(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS eg_ws_connectionholder (
  tenantid character varying(64),
  connectionid character varying(64),
  userid character varying(64),
  status character varying(32),
  isprimaryholder boolean,
  connectionholdertype character varying(64),
  holdershippercentage numeric(5,2),
  relationship character varying(64),
  createdby character varying(64),
  lastmodifiedby character varying(64),
  createdtime bigint,
  lastmodifiedtime bigint
);
CREATE INDEX IF NOT EXISTS idx_holder_conn ON eg_ws_connectionholder (connectionid);

CREATE TABLE IF NOT EXISTS eg_ws_roadcuttinginfo (
  id character varying(64) NOT NULL PRIMARY KEY,
  wsid character varying(64),
  roadtype character varying(64),
  roadcuttingarea numeric(12,2),
  active character varying(16) DEFAULT 'ACTIVE'
);
