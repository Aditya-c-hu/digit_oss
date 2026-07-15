-- =============================================================================
-- Local-deploy schema bootstrap for the migrated Go services.
-- Applied by local-deploy/04-init-db.ps1 against the `rainmaker` database.
-- Uses \ir (relative include) so each service's migrations/ directory stays the
-- single source of truth for its schema. Do not inline DDL here.
-- =============================================================================

\ir ../ws-services/migrations/ddl/V001__ws_schema.sql
\ir ../ws-calculator/migrations/ddl/V001__ws_calculator_schema.sql
