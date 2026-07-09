-- =============================================================================
-- ws-calculator schema (Go port)
-- Owns the meter-reading table the calculator reads/writes. Demand/bill data
-- lives in billing-service (external) and is not created here.
-- Idempotent: safe to re-run.
-- =============================================================================

CREATE TABLE IF NOT EXISTS eg_ws_meterreading (
  id character varying(64) PRIMARY KEY,
  connectionno character varying(64),
  billingperiod character varying(64) NOT NULL,
  meterstatus character varying(64) NOT NULL,
  lastreading decimal NOT NULL,
  lastreadingdate bigint NOT NULL,
  currentreading decimal NOT NULL,
  currentreadingdate bigint NOT NULL,
  consumption decimal,
  createdby character varying(64),
  lastmodifiedby character varying(64),
  createdtime bigint,
  lastmodifiedtime bigint,
  tenantid character varying(64)
);
CREATE INDEX IF NOT EXISTS idx_mr_tenant ON eg_ws_meterreading (tenantid);
CREATE INDEX IF NOT EXISTS idx_mr_conn ON eg_ws_meterreading (connectionno);
