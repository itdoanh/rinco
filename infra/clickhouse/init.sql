-- ============================================================
-- RINCO ClickHouse initial schema — analytics database
-- ============================================================
-- Creates `rinco_analytics` (or whatever CLICKHOUSE_DB is set to).
-- Tables / MVs / dictionaries are applied by the side-loaded
-- /docker-entrypoint-initdb.d/tables/*.sql and
-- /docker-entrypoint-initdb.d/materialized-views/*.sql files.
-- ============================================================

CREATE DATABASE IF NOT EXISTS rinco_analytics;

-- Application role (created if it does not exist)
CREATE USER IF NOT EXISTS rinco IDENTIFIED WITH plaintext_password BY 'rinco_dev_password';

-- Grant read/write on the analytics DB
GRANT ALL ON rinco_analytics.* TO rinco;
GRANT CREATE TEMPORARY TABLE ON rinco_analytics.* TO rinco;
