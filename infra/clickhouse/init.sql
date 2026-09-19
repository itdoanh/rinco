-- ============================================================
-- RINCO ClickHouse initial schema — analytics database
-- ============================================================
-- Creates `rinco_analytics` (or whatever CLICKHOUSE_DB is set to).
-- Tables / MVs / dictionaries are applied by the side-loaded
-- /docker-entrypoint-initdb.d/tables/*.sql and
-- /docker-entrypoint-initdb.d/materialized-views/*.sql files.
--
-- NOTE: User provisioning (`CREATE USER`, `GRANT`) is skipped because:
--   1. `CLICKHOUSE_USER` + `CLICKHOUSE_PASSWORD` env vars create the
--      `rinco` user automatically with full access via the entrypoint.
--   2. The bundled `users.xml` storage is read-only on ClickHouse 24.x,
--      so any GRANT here fails with ACCESS_STORAGE_READONLY.
--   3. The rinco user already has the privileges required by the
--      application: ALL on rinco_analytics.* and CREATE TEMPORARY TABLE.
-- ============================================================

CREATE DATABASE IF NOT EXISTS rinco_analytics;
