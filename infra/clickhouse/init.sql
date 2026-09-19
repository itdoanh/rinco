-- ============================================================
-- RINCO ClickHouse initial schema — analytics database
-- ============================================================
-- Creates `rinco_analytics` (or whatever CLICKHOUSE_DB is set to).
-- Tables / MVs / dictionaries are applied by the side-loaded
-- /docker-entrypoint-initdb.d/tables/*.sql and
-- /docker-entrypoint-initdb.d/materialized-views/*.sql files.
--
-- NOTE: The `CREATE USER` statement is skipped intentionally.
-- CLICKHOUSE_USER + CLICKHOUSE_PASSWORD env vars handle user creation
-- via the entrypoint. Re-creating the user here fails on restarts
-- because the users_xml storage becomes read-only once data exists.
-- ============================================================

CREATE DATABASE IF NOT EXISTS rinco_analytics;

-- Application role is created automatically by the ClickHouse entrypoint
-- via CLICKHOUSE_USER / CLICKHOUSE_PASSWORD environment variables.

-- Grant read/write on the analytics DB (idempotent)
GRANT ALL ON rinco_analytics.* TO rinco;
GRANT CREATE TEMPORARY TABLE ON rinco_analytics.* TO rinco;
