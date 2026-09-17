-- ============================================================
-- RINCO ClickHouse Comprehensive Demo Seed
-- Loop 202 — Analytics Data
-- ============================================================
-- Applies to: rinco_analytics database
-- Run: clickhouse-client --queries-file seed/clickhouse/seed.sql
-- ============================================================

-- ============================================================
-- SECTION 1: ANALYTICS EVENTS (1000 events per tenant)
-- ============================================================
-- Uses ClickHouse's generateRandom for synthetic event generation

INSERT INTO rinco_analytics.events_raw
(
    event_id,
    event_time,
    tenant_id,
    user_id,
    session_id,
    anonymous_id,
    event_name,
    source,
    utm_source,
    utm_medium,
    utm_campaign,
    utm_content,
    utm_term,
    fbclid,
    gclid,
    country,
    city,
    device_type,
    browser,
    os,
    ip_address,
    user_agent,
    referrer,
    page_url,
    properties,
    event_date
)
SELECT
    generateUUIDv4() AS event_id,
    now() - (rand() % (30 * 86400)) AS event_time,
    tenant_id,
    user_id,
    session_id,
    anonymous_id,
    event_name,
    source,
    utm_source,
    utm_medium,
    utm_campaign,
    utm_content,
    utm_term,
    fbclid,
    gclid,
    country,
    city,
    device_type,
    browser,
    os,
    ip_address,
    user_agent,
    referrer,
    page_url,
    properties,
    toDate(now() - (rand() % (30 * 86400))) AS event_date
FROM (
    -- Apex Fintech events (tenant 1)
    SELECT
        'aaaaaaaa-0000-0000-0000-000000000001' AS tenant_id,
        user_id,
        session_id,
        anonymous_id,
        event_name,
        source,
        utm_source,
        utm_medium,
        utm_campaign,
        utm_content,
        utm_term,
        fbclid,
        gclid,
        country,
        city,
        device_type,
        browser,
        os,
        ip_address,
        user_agent,
        referrer,
        page_url,
        properties
    FROM generateRandom(
        'tenant_id String, user_id UUID, session_id UUID, anonymous_id String,
         event_name String, source String, utm_source String, utm_medium String,
         utm_campaign String, utm_content String, utm_term String,
         fbclid String, gclid String, country String, city String,
         device_type String, browser String, os String,
         ip_address IPv4, user_agent String, referrer String,
         page_url String, properties JSON',
        1000
    )
    LIMIT 1000
)
ARRAY JOIN
    -- Expand event names for realistic distribution
    arrayMap(x -> (
        tenant_id,
        -- user_id: random from Apex user pool
        toUUID(concat('a0000001-0000-0000-0000-000000000', toString(1 + (rand() % 15)))) AS user_id,
        -- session_id: new session per event
        generateUUIDv4() AS session_id,
        -- anonymous_id
        concat('anon_', toString(rand())) AS anonymous_id,
        -- event_name distribution
        arrayJoin(['page_view', 'page_view', 'page_view', 'lead.created',
                   'lead.contacted', 'lead.qualified', 'conversion', 'button_click',
                   'form_submit', 'video_play']) AS event_name,
        -- source distribution
        arrayJoin(['facebook', 'facebook', 'google', 'tiktok', 'zalo', 'organic',
                   'referral', 'direct']) AS source,
        -- utm_source
        arrayJoin(['facebook', 'google', 'tiktok', 'zalo', 'direct']) AS utm_source,
        -- utm_medium
        arrayJoin(['cpc', 'cpc', 'social', 'organic', 'referral']) AS utm_medium,
        -- utm_campaign
        arrayJoin(['spring_loan_v1', 'q3_personal', 'cashback_50', 'instant_v2',
                   'tiktok_creative_a', 'zalo_broadcast']) AS utm_campaign,
        -- utm_content
        md5(tostring(rand())) AS utm_content,
        -- utm_term
        arrayJoin(['vay nhanh', 'vay tieu dung', 'vay kinh doanh', '']) AS utm_term,
        -- fbclid
        CASE WHEN rand() % 3 = 0 THEN concat('fb.', md5(tostring(rand()))) ELSE '' END AS fbclid,
        -- gclid
        CASE WHEN rand() % 4 = 0 THEN concat('g.', md5(tostring(rand()))) ELSE '' END AS gclid,
        -- country
        'Vietnam' AS country,
        -- city
        arrayJoin(['Ho Chi Minh', 'Ha Noi', 'Da Nang', 'Can Tho', 'Hai Phong']) AS city,
        -- device_type
        arrayJoin(['mobile', 'mobile', 'desktop', 'tablet']) AS device_type,
        -- browser
        arrayJoin(['Chrome', 'Safari', 'Firefox', 'Edge']) AS browser,
        -- os
        arrayJoin(['Windows', 'Android', 'iOS', 'macOS']) AS os,
        -- ip_address
        toIPv4(concat('203.0.113.', toString(10 + rand() % 200))) AS ip_address,
        -- user_agent
        'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/127.0' AS user_agent,
        -- referrer
        arrayJoin(['https://facebook.com', 'https://google.com', 'https://zalo.me', '']) AS referrer,
        -- page_url
        arrayJoin(['/landing/home', '/landing/vay-mua-xe', '/landing/vay-mua-nha',
                    '/dashboard', '/leads', '/deals']) AS page_url,
        -- properties JSON
        toJSONString(map(
            'lead_id', toString(generateUUIDv4()),
            'form_id', toString(generateUUIDv4()),
            'value', toString(rand() % 500000000)
        )) AS properties
    )
);

-- ============================================================
-- SECTION 2: CONVERSIONS (200 records per tenant)
-- ============================================================
INSERT INTO rinco_analytics.conversions
(
    event_time,
    tenant_id,
    user_id,
    campaign_id,
    funnel_id,
    step,
    order_id,
    order_value,
    currency,
    properties
)
SELECT
    now() - (rand() % (30 * 86400)) AS event_time,
    tenant_id,
    toUUID(concat('a0000001-0000-0000-0000-000000000', toString(1 + (rand() % 15)))) AS user_id,
    generateUUIDv4() AS campaign_id,
    'lead_funnel' AS funnel_id,
    arrayJoin(['view', 'add_to_cart', 'checkout', 'purchase', 'purchase']) AS step,
    generateUUIDv4() AS order_id,
    (rand() % 100 + 1) * 1000000 AS order_value,
    'VND' AS currency,
    '{"source":"facebook","ad_id":"ad_123"}' AS properties
FROM numbers(200);

-- ============================================================
-- SECTION 3: USER SESSIONS (500 sessions per tenant)
-- ============================================================
INSERT INTO rinco_analytics.user_sessions
(
    session_id,
    tenant_id,
    user_id,
    anonymous_id,
    started_at,
    ended_at,
    duration_sec,
    page_view_count,
    event_count,
    entry_url,
    exit_url,
    country,
    device_type,
    browser,
    os,
    utm_source,
    utm_campaign
)
SELECT
    generateUUIDv4() AS session_id,
    'aaaaaaaa-0000-0000-0000-000000000001' AS tenant_id,
    toUUID(concat('a0000001-0000-0000-0000-000000000', toString(1 + (rand() % 15)))) AS user_id,
    concat('anon_', toString(rand())) AS anonymous_id,
    now() - (rand() % (30 * 86400)) AS started_at,
    now() - (rand() % (30 * 86400)) + (rand() % 3600) AS ended_at,
    rand() % 1800 + 30 AS duration_sec,
    rand() % 15 + 1 AS page_view_count,
    rand() % 50 + 5 AS event_count,
    arrayJoin(['/landing/home', '/landing/vay-mua-xe', '/landing/vay-mua-nha']) AS entry_url,
    arrayJoin(['/dashboard', '/leads', '/deals', '/landing/home']) AS exit_url,
    'Vietnam' AS country,
    arrayJoin(['mobile', 'mobile', 'desktop', 'tablet']) AS device_type,
    arrayJoin(['Chrome', 'Safari', 'Firefox']) AS browser,
    arrayJoin(['Windows', 'Android', 'iOS']) AS os,
    arrayJoin(['facebook', 'google', 'tiktok', 'organic']) AS utm_source,
    arrayJoin(['spring_loan_v1', 'q3_personal', 'cashback_50']) AS utm_campaign
FROM numbers(500);

-- ============================================================
-- SECTION 4: CROSS-TENANT EVENTS (HCT Consulting + Demo)
-- ============================================================
INSERT INTO rinco_analytics.events_raw
(
    event_id,
    event_time,
    tenant_id,
    user_id,
    session_id,
    anonymous_id,
    event_name,
    source,
    utm_source,
    utm_medium,
    utm_campaign,
    utm_content,
    fbclid,
    gclid,
    country,
    city,
    device_type,
    browser,
    os,
    ip_address,
    user_agent,
    referrer,
    page_url,
    properties,
    event_date
)
-- HCT Consulting events (500 events)
SELECT
    generateUUIDv4(),
    now() - (rand() % (30 * 86400)),
    'aaaaaaaa-0000-0000-0000-000000000002',
    toUUID(concat('a0000002-0000-0000-0000-000000000', toString(1 + (rand() % 15)))),
    generateUUIDv4(),
    concat('anon_', toString(rand())),
    arrayJoin(['page_view', 'page_view', 'lead.created', 'lead.qualified',
               'property.view', 'property.inquiry', 'conversion']) AS event_name,
    arrayJoin(['facebook', 'zalo', 'linkedin', 'organic']) AS source,
    arrayJoin(['facebook', 'zalo', 'linkedin', 'google']) AS utm_source,
    arrayJoin(['cpc', 'social', 'organic']) AS utm_medium,
    arrayJoin(['vinhomes_grand_park', 'masterise_eco', 'sun_group_30ha']) AS utm_campaign,
    md5(tostring(rand())),
    CASE WHEN rand() % 3 = 0 THEN concat('fb.', md5(tostring(rand()))) ELSE '' END,
    '',
    'Vietnam',
    arrayJoin(['Ha Noi', 'Ho Chi Minh', 'Hai Phong']),
    arrayJoin(['mobile', 'desktop']),
    arrayJoin(['Chrome', 'Safari']),
    arrayJoin(['Android', 'iOS', 'Windows']),
    toIPv4(concat('203.0.113.', toString(10 + rand() % 200))),
    'Mozilla/5.0',
    arrayJoin(['https://facebook.com', 'https://zalo.me', '']),
    arrayJoin(['/landing/can-ho-cao-cap', '/landing/dat-nen', '/projects']),
    '{"property_id":"prop_123","price":2500000000}',
    toDate(now() - (rand() % (30 * 86400)))
FROM numbers(500)
ARRAY JOIN [
    'page_view', 'page_view', 'lead.created', 'lead.qualified',
    'property.view', 'property.inquiry', 'conversion'
] AS event_name;

INSERT INTO rinco_analytics.events_raw
(
    event_id,
    event_time,
    tenant_id,
    user_id,
    session_id,
    anonymous_id,
    event_name,
    source,
    utm_source,
    utm_medium,
    utm_campaign,
    country,
    city,
    device_type,
    browser,
    os,
    ip_address,
    user_agent,
    page_url,
    properties,
    event_date
)
-- Demo Company events (200 events)
SELECT
    generateUUIDv4(),
    now() - (rand() % (10 * 86400)),
    'bbbbbbbb-0000-0000-0000-000000000003',
    toUUID(concat('a0000003-0000-0000-0000-000000000', toString(1 + (rand() % 10)))),
    generateUUIDv4(),
    concat('anon_', toString(rand())),
    arrayJoin(['page_view', 'demo.started', 'demo.completed', 'signup', 'lead.created']) AS event_name,
    arrayJoin(['facebook', 'google', 'direct']) AS source,
    arrayJoin(['facebook', 'google', 'direct']) AS utm_source,
    arrayJoin(['cpc', 'organic', 'none']) AS utm_medium,
    arrayJoin(['demo_spring', 'demo_free_trial', 'demo_youtube']) AS utm_campaign,
    'Vietnam',
    arrayJoin(['Ho Chi Minh', 'Ha Noi']),
    arrayJoin(['mobile', 'desktop']),
    arrayJoin(['Chrome', 'Safari']),
    arrayJoin(['Windows', 'Android']),
    toIPv4(concat('203.0.113.', toString(10 + rand() % 200))),
    'Mozilla/5.0',
    arrayJoin(['/demo', '/features', '/pricing', '/signup']),
    '{"demo_id":"demo_123"}',
    toDate(now() - (rand() % (10 * 86400)))
FROM numbers(200)
ARRAY JOIN [
    'page_view', 'demo.started', 'demo.completed', 'signup', 'lead.created'
] AS event_name;

-- ============================================================
-- VERIFICATION QUERIES
-- ============================================================
SELECT
    'rinco_analytics.events_raw' AS table_name,
    count() AS total_rows,
    uniq(tenant_id) AS unique_tenants,
    min(event_time) AS oldest_event,
    max(event_time) AS newest_event
FROM rinco_analytics.events_raw;

SELECT
    'rinco_analytics.conversions' AS table_name,
    count() AS total_rows,
    uniq(tenant_id) AS unique_tenants
FROM rinco_analytics.conversions;

SELECT
    'rinco_analytics.user_sessions' AS table_name,
    count() AS total_rows
FROM rinco_analytics.user_sessions;

-- ============================================================
-- MATERIALIZED VIEW REFRESH (auto by ClickHouse, but manual trigger)
-- ============================================================
SYSTEM MERGE MATERIALIZED VIEW rinco_analytics.events_hourly_mv;
SYSTEM MERGE MATERIALIZED VIEW rinco_analytics.funnel_hourly_mv;

SELECT 'ClickHouse analytics seed complete!' AS status;
