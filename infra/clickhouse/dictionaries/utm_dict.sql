-- ============================================================
-- UTM Dictionary — for human-readable UTM mapping
-- Reference: https://clickhouse.com/docs/en/sql-reference/dictionaries/external-dictionaries/external_dicts_dict_sources
-- ============================================================

CREATE DICTIONARY IF NOT EXISTS rinco_analytics.utm_dict
(
    name        String,
    full_name   String,
    parent_id   String,
    level       UInt8
)
PRIMARY KEY name
SOURCE(CLICKHOUSE(DB 'rinco_analytics' TABLE 'utm_source'))
LIFETIME(MIN 300 MAX 600)
LAYOUT(HASHED());
