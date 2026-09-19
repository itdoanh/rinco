-- ============================================================
-- RINCO ClickHouse Demo Seed (simplified)
-- 1500 events across 3 tenants
-- ============================================================

-- Tenant 1: Apex Fintech (1000 events)
INSERT INTO rinco_analytics.events_raw
(event_id, event_time, tenant_id, user_id, session_id, anonymous_id, event_name, source, utm_source, utm_medium, utm_campaign, utm_content, utm_term, fbclid, country, city, device_type, browser, os, ip_address, user_agent, referrer, page_url, properties)
SELECT
    generateUUIDv4()                                                    AS event_id,
    now() - (rand() % (30 * 86400))                                     AS event_time,
    'aaaaaaaa-0000-0000-0000-000000000001'                              AS tenant_id,
    toUUID(concat('a0000001-0000-0000-0000-0000000000', lpad(toString(1 + (rand() % 9)), 2, '0'))) AS user_id,
    generateUUIDv4()                                                    AS session_id,
    concat('anon_', toString(rand()))                                   AS anonymous_id,
    arrayJoin(['page_view', 'page_view', 'page_view', 'lead.created', 'lead.contacted', 'lead.qualified', 'conversion', 'button_click', 'form_submit', 'video_play']) AS event_name,
    arrayJoin(['facebook', 'facebook', 'google', 'tiktok', 'zalo', 'organic']) AS source,
    arrayJoin(['facebook', 'google', 'tiktok', 'zalo', 'direct'])       AS utm_source,
    arrayJoin(['cpc', 'cpc', 'social', 'organic'])                      AS utm_medium,
    arrayJoin(['spring_loan_v1', 'q3_personal', 'cashback_50', 'instant_v2']) AS utm_campaign,
    MD5(toString(rand()))                                                AS utm_content,
    arrayJoin(['vay nhanh', 'vay tieu dung', ''])                        AS utm_term,
    CASE WHEN rand() % 3 = 0 THEN concat('fb.', MD5(toString(rand()))) ELSE '' END AS fbclid,
    'Vietnam'                                                            AS country,
    arrayJoin(['Ho Chi Minh', 'Ha Noi', 'Da Nang', 'Can Tho'])           AS city,
    arrayJoin(['mobile', 'mobile', 'desktop', 'tablet'])                 AS device_type,
    arrayJoin(['Chrome', 'Safari', 'Firefox'])                           AS browser,
    arrayJoin(['Windows', 'Android', 'iOS'])                             AS os,
    toIPv4(concat('203.0.113.', toString(10 + rand() % 200)))            AS ip_address,
    'Mozilla/5.0'                                                        AS user_agent,
    arrayJoin(['https://facebook.com', 'https://google.com', ''])        AS referrer,
    arrayJoin(['/landing/home', '/landing/vay-mua-xe', '/dashboard', '/leads']) AS page_url,
    '{"source":"seed"}'                                                  AS properties
FROM numbers(1000);

-- Tenant 2: HCT Consulting (200 events)
INSERT INTO rinco_analytics.events_raw
(event_id, event_time, tenant_id, user_id, session_id, anonymous_id, event_name, source, utm_source, utm_medium, utm_campaign, utm_content, fbclid, country, city, device_type, browser, os, ip_address, user_agent, referrer, page_url, properties)
SELECT
    generateUUIDv4(),
    now() - (rand() % (30 * 86400)),
    'aaaaaaaa-0000-0000-0000-000000000002',
    toUUID(concat('a0000002-0000-0000-0000-0000000000', lpad(toString(1 + (rand() % 9)), 2, '0'))),
    generateUUIDv4(),
    concat('anon_', toString(rand())),
    arrayJoin(['page_view', 'lead.created', 'property.view', 'conversion']) AS event_name,
    arrayJoin(['facebook', 'zalo', 'linkedin'])                          AS source,
    arrayJoin(['facebook', 'zalo', 'google'])                            AS utm_source,
    arrayJoin(['cpc', 'social'])                                         AS utm_medium,
    arrayJoin(['vinhomes_grand_park', 'masterise_eco'])                  AS utm_campaign,
    MD5(toString(rand())),
    CASE WHEN rand() % 3 = 0 THEN concat('fb.', MD5(toString(rand()))) ELSE '' END,
    'Vietnam',
    arrayJoin(['Ha Noi', 'Ho Chi Minh']),
    arrayJoin(['mobile', 'desktop']),
    arrayJoin(['Chrome', 'Safari']),
    arrayJoin(['Android', 'iOS', 'Windows']),
    toIPv4(concat('203.0.113.', toString(10 + rand() % 200))),
    'Mozilla/5.0',
    arrayJoin(['https://facebook.com', 'https://zalo.me', '']),
    arrayJoin(['/landing/can-ho-cao-cap', '/projects']),
    '{"property_id":"prop_123"}'
FROM numbers(200);

-- Tenant 3: Demo Company (300 events)
INSERT INTO rinco_analytics.events_raw
(event_id, event_time, tenant_id, user_id, session_id, anonymous_id, event_name, source, utm_source, utm_medium, utm_campaign, utm_content, country, device_type, browser, os, ip_address, user_agent, page_url, properties)
SELECT
    generateUUIDv4(),
    now() - (rand() % (10 * 86400)),
    'bbbbbbbb-0000-0000-0000-000000000003',
    toUUID(concat('a0000003-0000-0000-0000-0000000000', lpad(toString(1 + (rand() % 9)), 2, '0'))),
    generateUUIDv4(),
    concat('anon_', toString(rand())),
    arrayJoin(['page_view', 'demo.started', 'demo.completed', 'signup', 'lead.created']) AS event_name,
    arrayJoin(['facebook', 'google', 'direct'])                          AS source,
    arrayJoin(['facebook', 'google', 'direct'])                          AS utm_source,
    arrayJoin(['cpc', 'organic', 'none'])                                AS utm_medium,
    arrayJoin(['demo_spring', 'demo_free_trial'])                        AS utm_campaign,
    'developer.mozilla.org',
    'Vietnam',
    arrayJoin(['mobile', 'desktop']),
    arrayJoin(['Chrome', 'Safari']),
    arrayJoin(['Windows', 'Android']),
    toIPv4(concat('203.0.113.', toString(10 + rand() % 200))),
    'Mozilla/5.0',
    arrayJoin(['/demo', '/features', '/pricing']),
    '{"demo_id":"demo_123"}'
FROM numbers(300);

-- ============================================================
-- SESSIONS (500 rows for tenant 1)
-- ============================================================
INSERT INTO rinco_analytics.user_sessions
(session_id, tenant_id, user_id, anonymous_id, started_at, ended_at, duration_sec, page_view_count, event_count, entry_url, exit_url, country, device_type, browser, os, utm_source, utm_campaign)
SELECT
    generateUUIDv4(),
    'aaaaaaaa-0000-0000-0000-000000000001',
    toUUID(concat('a0000001-0000-0000-0000-00000000000', toString(1 + (rand() % 15)))),
    concat('anon_', toString(rand())),
    now() - (rand() % (30 * 86400)),
    now() - (rand() % (30 * 86400)) + (rand() % 3600),
    60 + (rand() % 1800),
    1 + (rand() % 15),
    5 + (rand() % 50),
    arrayJoin(['/landing/home', '/landing/vay-mua-xe', '/landing/vay-mua-nha']),
    arrayJoin(['/dashboard', '/leads', '/landing/home']),
    'Vietnam',
    arrayJoin(['mobile', 'mobile', 'desktop', 'tablet']),
    arrayJoin(['Chrome', 'Safari', 'Firefox']),
    arrayJoin(['Windows', 'Android', 'iOS']),
    arrayJoin(['facebook', 'google', 'tiktok', 'organic']),
    arrayJoin(['spring_loan_v1', 'q3_personal', 'cashback_50'])
FROM numbers(500);

-- ============================================================
-- CONVERSIONS (200 funnels)
-- ============================================================
INSERT INTO rinco_analytics.conversions
(event_time, tenant_id, user_id, campaign_id, funnel_id, step, order_id, order_value, currency, properties)
SELECT
    now() - (rand() % (30 * 86400)),
    'aaaaaaaa-0000-0000-0000-000000000001',
    toUUID(concat('a0000001-0000-0000-0000-00000000000', toString(1 + (rand() % 15)))),
    generateUUIDv4(),
    arrayJoin(['funnel_vay', 'funnel_register', 'funnel_demo']),
    arrayJoin(['view', 'add_to_cart', 'checkout', 'purchase']),
    generateUUIDv4(),
    rand() % 50000000 + 1000000,
    'VND',
    '{"source":"seed"}'
FROM numbers(200);

-- ============================================================
-- VERIFICATION
-- ============================================================
SELECT
    'events_raw'    AS table_name,
    count()         AS rows,
    uniq(tenant_id) AS tenants
FROM rinco_analytics.events_raw
UNION ALL
SELECT
    'user_sessions', count(), uniq(tenant_id)
FROM rinco_analytics.user_sessions
UNION ALL
SELECT
    'conversions', count(), uniq(tenant_id)
FROM rinco_analytics.conversions
UNION ALL
SELECT
    'events_hourly_mv', count(), uniq(tenant_id)
FROM rinco_analytics.events_hourly_mv
UNION ALL
SELECT
    'funnel_hourly_mv', count(), uniq(tenant_id)
FROM rinco_analytics.funnel_hourly_mv;

SELECT 'ClickHouse analytics seed complete!' AS status;
