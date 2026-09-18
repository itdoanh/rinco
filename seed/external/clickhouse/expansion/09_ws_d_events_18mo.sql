-- ============================================================
-- WS-D: CLICKHOUSE EVENTS EXPANSION (18 months / 540 days)
-- For each of 5 tenants: 2500 events spread across 540 days
-- Realistic distribution: 60% page_view, 20% click, 10% form_submit, 5% conversion, 5% other
-- Seasonal spikes: Tết (Feb), mid-year (Jun), Black Friday (Nov), year-end (Dec)
-- Vietnamese city distribution + realistic event flows
-- ============================================================================

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
    -- Spread over 540 days with seasonal spikes
    CASE
        -- Tết spikes (Jan-Feb)
        WHEN rand() < 0.15 THEN now() - (toUInt32(rand() % 60) * 86400)
        -- Mid year (Jun-Jul)
        WHEN rand() < 0.15 AND rand() < 0.4 THEN now() - (toUInt32(180 + (rand() % 60)) * 86400)
        -- Black Friday (Nov)
        WHEN rand() < 0.15 AND rand() < 0.3 THEN now() - (toUInt32(300 + (rand() % 30)) * 86400)
        -- Year-end (Dec)
        WHEN rand() < 0.15 AND rand() < 0.4 THEN now() - (toUInt32(335 + (rand() % 30)) * 86400)
        ELSE now() - (toUInt32(rand() % 540) * 86400)
    END AS event_time,
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
    CASE WHEN utm_source = 'facebook' THEN concat('fb.', generateUUIDv4()) ELSE '' END AS fbclid,
    CASE WHEN utm_source = 'google' THEN concat('g.', generateUUIDv4()) ELSE '' END AS gclid,
    'VN' AS country,
    -- 70% HCM/HN, 15% DN, 15% others
    CASE
        WHEN rand() < 0.35 THEN 'Ho Chi Minh City'
        WHEN rand() < 0.35 THEN 'Ha Noi'
        WHEN rand() < 0.5 THEN 'Da Nang'
        WHEN rand() < 0.5 THEN 'Can Tho'
        WHEN rand() < 0.5 THEN 'Hai Phong'
        ELSE 'Binh Duong'
    END AS city,
    -- Device mix
    CASE
        WHEN rand() < 0.6 THEN 'mobile'
        WHEN rand() < 0.85 THEN 'desktop'
        ELSE 'tablet'
    END AS device_type,
    CASE
        WHEN rand() < 0.4 THEN 'Chrome'
        WHEN rand() < 0.65 THEN 'Safari'
        WHEN rand() < 0.85 THEN 'Edge'
        ELSE 'Firefox'
    END AS browser,
    CASE
        WHEN rand() < 0.45 THEN 'Android'
        WHEN rand() < 0.75 THEN 'iOS'
        WHEN rand() < 0.9 THEN 'Windows'
        ELSE 'macOS'
    END AS os,
    concat(toString(toUInt8(1 + (rand() % 254))), '.', toString(toUInt8(rand() % 256)), '.', toString(toUInt8(rand() % 256)), '.', toString(toUInt8(rand() % 256))) AS ip_address,
    'Mozilla/5.0 (compatible; RINCO-Analytics/1.0)' AS user_agent,
    referrer,
    -- page_url per event type
    CASE
        WHEN event_name = 'page_view' THEN concat('/', page_pool.path)
        WHEN event_name = 'form_submit' THEN '/landing/submit'
        WHEN event_name = 'conversion' THEN '/thank-you'
        ELSE concat('/', page_pool.path)
    END AS page_url,
    '{}' AS properties,
    toDate(event_time) AS event_date
FROM
(
    SELECT
        -- Tenant pool (5 tenants, weight VNG/Apex higher)
        CASE
            WHEN tenant_n % 5 = 0 THEN 'aaaaaaaa-0000-0000-0000-000000000001'
            WHEN tenant_n % 5 = 1 THEN 'aaaaaaaa-0000-0000-0000-000000000002'
            WHEN tenant_n % 5 = 2 THEN 'bbbbbbbb-0000-0000-0000-000000000003'
            WHEN tenant_n % 5 = 3 THEN 'cccccccc-0000-0000-0000-000000000004'
            ELSE 'dddddddd-0000-0000-0000-000000000005'
        END AS tenant_id,
        generateUUIDv4() AS user_id,
        generateUUIDv4() AS session_id,
        generateUUIDv4() AS anonymous_id,
        -- Event distribution: 60% page_view, 20% click, 10% form_submit, 5% conversion, 5% other
        CASE
            WHEN e_n % 20 < 12 THEN 'page_view'
            WHEN e_n % 20 < 16 THEN 'click'
            WHEN e_n % 20 < 18 THEN 'form_submit'
            WHEN e_n % 20 = 18 THEN 'purchase'
        END AS event_name,
        -- Source distribution
        CASE
            WHEN e_n % 10 < 4 THEN 'facebook'
            WHEN e_n % 10 < 6 THEN 'google'
            WHEN e_n % 10 < 8 THEN 'direct'
            WHEN e_n % 10 = 8 THEN 'zalo'
            ELSE 'tiktok'
        END AS source,
        utm_source,
        utm_medium,
        utm_campaign,
        utm_content,
        utm_term,
        referrer,
        page_pool.path AS path
    FROM
    (
        SELECT
            number AS tenant_n
        FROM system.numbers
        LIMIT 5
    ) tenants
    CROSS JOIN
    (
        SELECT number AS e_n
        FROM system.numbers
        LIMIT 2500
    ) events_n
    CROSS JOIN
    (
        SELECT
            ['home','about','pricing','features','contact','blog','docs','case-studies','demo-request','pricing/enterprise','solutions/finance','solutions/retail','solutions/education','solutions/healthcare','integrations','blog/crm-best-practices','blog/lead-scoring-tips','blog/sme-vietnam','blog/ai-marketing','blog/sales-automation'] AS paths,
            paths[toUInt8(1 + (rand() % toUInt8(length(paths))))] AS path
    ) page_pool
    CROSS JOIN
    (
        SELECT
            ['facebook','google','zalo','tiktok','direct','organic','email','referral'] AS utm_sources_arr,
            utm_sources_arr[toUInt8(1 + (rand() % toUInt8(length(utm_sources_arr))))] AS utm_source,
            ['cpc','cpc','cpc','referral','social','email','organic'] AS utm_mediums,
            utm_mediums[toUInt8(1 + (rand() % toUInt8(length(utm_mediums))))] AS utm_medium,
            ['tet_2025','tet_2026','summer_sale','back_to_school','flash_sale','demo_free_trial','vay_q1','vay_q2','vay_q3','vay_q4','vinhomes_2025','masteri_2025','year_end','spring_open_house','cyber_monday_2025','black_friday_2025','black_friday_2026'] AS utm_campaigns,
            utm_campaigns[toUInt8(1 + (rand() % toUInt8(length(utm_campaigns))))] AS utm_campaign,
            ['ad_a','ad_b','ad_c','ad_d','ad_e'] AS utm_contents,
            utm_contents[toUInt8(1 + (rand() % toUInt8(length(utm_contents))))] AS utm_content,
            ['vay tin chap','mua nha','mua xe','vay kinh doanh','thiet ke','cloud','zalo','crm'] AS utm_terms,
            utm_terms[toUInt8(1 + (rand() % toUInt8(length(utm_terms))))] AS utm_term,
            ['https://google.com','https://facebook.com','https://zalo.me','https://tiktok.com','direct','https://news.google.com'] AS referrers,
            referrers[toUInt8(1 + (rand() % toUInt8(length(referrers))))] AS referrer
    ) utm_pool
)
SETTINGS join_use_nulls = 1;

-- ============================================================
-- WS-D: SESSIONS EXPANSION (18 months)
-- 200 sessions per tenant x 5 = 1000 sessions
-- ============================================================
INSERT INTO rinco_analytics.sessions
(
    session_id,
    session_start,
    session_end,
    tenant_id,
    user_id,
    anonymous_id,
    device_type,
    browser,
    os,
    country,
    city,
    ip_address,
    utm_source,
    utm_medium,
    utm_campaign,
    page_views,
    events_count,
    is_bounce,
    session_duration_seconds,
    conversions,
    revenue_vnd,
    event_date
)
SELECT
    generateUUIDv4() AS session_id,
    -- Spread over 540 days
    CASE
        WHEN rand() < 0.15 THEN now() - (toUInt32(rand() % 60) * 86400)
        WHEN rand() < 0.15 AND rand() < 0.4 THEN now() - (toUInt32(180 + (rand() % 60)) * 86400)
        WHEN rand() < 0.15 AND rand() < 0.3 THEN now() - (toUInt32(300 + (rand() % 30)) * 86400)
        ELSE now() - (toUInt32(rand() % 540) * 86400)
    END AS session_start,
    session_start + toIntervalSecond(toUInt32(60 + (rand() % 1800))) AS session_end,
    tenant_id,
    user_id,
    generateUUIDv4() AS anonymous_id,
    -- Device
    CASE WHEN rand() < 0.6 THEN 'mobile' WHEN rand() < 0.85 THEN 'desktop' ELSE 'tablet' END AS device_type,
    CASE WHEN rand() < 0.4 THEN 'Chrome' WHEN rand() < 0.65 THEN 'Safari' WHEN rand() < 0.85 THEN 'Edge' ELSE 'Firefox' END AS browser,
    CASE WHEN rand() < 0.45 THEN 'Android' WHEN rand() < 0.75 THEN 'iOS' WHEN rand() < 0.9 THEN 'Windows' ELSE 'macOS' END AS os,
    'VN' AS country,
    CASE WHEN rand() < 0.4 THEN 'Ho Chi Minh City' WHEN rand() < 0.7 THEN 'Ha Noi' WHEN rand() < 0.85 THEN 'Da Nang' ELSE 'Can Tho' END AS city,
    concat(toString(toUInt8(1 + (rand() % 254))), '.', toString(toUInt8(rand() % 256)), '.', toString(toUInt8(rand() % 256)), '.', toString(toUInt8(rand() % 256))) AS ip_address,
    CASE WHEN rand() < 0.4 THEN 'facebook' WHEN rand() < 0.6 THEN 'google' WHEN rand() < 0.75 THEN 'zalo' WHEN rand() < 0.9 THEN 'direct' ELSE 'tiktok' END AS utm_source,
    'cpc' AS utm_medium,
    ['tet_2025','summer_sale','flash_sale','vay_q1','back_to_school','vinhomes_2025','black_friday_2025','year_end'] AS utm_campaigns,
    utm_campaigns[toUInt8(1 + (rand() % toUInt8(length(utm_campaigns))))] AS utm_campaign,
    toUInt8(1 + (rand() % 15)) AS page_views,
    toUInt8(2 + (rand() % 30)) AS events_count,
    rand() < 0.35 AS is_bounce,
    toUInt32(30 + (rand() % 1800)) AS session_duration_seconds,
    -- 5% conversion rate
    IF(rand() < 0.05, 1, 0) AS conversions,
    -- Revenue: only if conversion (5M-50M VND for B2B SaaS)
    IF(rand() < 0.05, toDecimal64(5000000 + (rand() % 95000000), 2), toDecimal64(0, 2)) AS revenue_vnd,
    toDate(session_start) AS event_date
FROM
(
    SELECT number AS tenant_n,
           CASE
            WHEN number % 5 = 0 THEN 'aaaaaaaa-0000-0000-0000-000000000001'
            WHEN number % 5 = 1 THEN 'aaaaaaaa-0000-0000-0000-000000000002'
            WHEN number % 5 = 2 THEN 'bbbbbbbb-0000-0000-0000-000000000003'
            WHEN number % 5 = 3 THEN 'cccccccc-0000-0000-0000-000000000004'
            ELSE 'dddddddd-0000-0000-0000-000000000005'
           END AS tenant_id,
           generateUUIDv4() AS user_id
    FROM system.numbers
    LIMIT 1000
) src;

-- ============================================================
-- WS-D: CONVERSIONS EXPANSION (18 months)
-- 100 conversions per tenant x 5 = 500 conversions
-- ============================================================
INSERT INTO rinco_analytics.conversions
(
    conversion_id,
    conversion_time,
    tenant_id,
    user_id,
    session_id,
    conversion_type,
    value_vnd,
    currency,
    attribution_source,
    attribution_campaign,
    event_date
)
SELECT
    generateUUIDv4() AS conversion_id,
    -- Spread across 540 days with Tết spike
    CASE
        WHEN rand() < 0.2 THEN now() - (toUInt32(rand() % 60) * 86400)
        WHEN rand() < 0.2 AND rand() < 0.4 THEN now() - (toUInt32(180 + (rand() % 60)) * 86400)
        WHEN rand() < 0.2 AND rand() < 0.3 THEN now() - (toUInt32(300 + (rand() % 30)) * 86400)
        ELSE now() - (toUInt32(rand() % 540) * 86400)
    END AS conversion_time,
    tenant_id,
    user_id,
    generateUUIDv4() AS session_id,
    -- Conversion type
    conv_pool.ctype AS conv_type,
    -- Value (realistic AOV)
    CASE
        WHEN conv_pool.ctype = 'subscription' THEN toDecimal64(5000000 + (rand() % 25000000), 2)        -- 5-30M
        WHEN conv_pool.ctype = 'lead_form_submit' THEN toDecimal64(0, 2)
        WHEN conv_pool.ctype = 'trial_signup' THEN toDecimal64(0, 2)
        WHEN conv_pool.ctype = 'purchase' THEN toDecimal64(100000000 + (rand() % 500000000), 2)       -- 100-600M
        WHEN conv_pool.ctype = 'appointment_booked' THEN toDecimal64(0, 2)
        ELSE toDecimal64(2000000 + (rand() % 15000000), 2)
    END AS value_vnd,
    'VND' AS currency,
    -- Attribution
    CASE
        WHEN rand() < 0.35 THEN 'facebook'
        WHEN rand() < 0.55 THEN 'google'
        WHEN rand() < 0.75 THEN 'zalo'
        WHEN rand() < 0.9 THEN 'direct'
        ELSE 'tiktok'
    END AS attribution_source,
    ['tet_2025','summer_sale','flash_sale','vay_q1','back_to_school','vinhomes_2025','black_friday_2025','year_end'] AS utm_campaigns,
    utm_campaigns[toUInt8(1 + (rand() % toUInt8(length(utm_campaigns))))] AS attribution_campaign,
    toDate(conversion_time) AS event_date
FROM
(
    SELECT number AS tenant_n,
           CASE
            WHEN number % 5 = 0 THEN 'aaaaaaaa-0000-0000-0000-000000000001'
            WHEN number % 5 = 1 THEN 'aaaaaaaa-0000-0000-0000-000000000002'
            WHEN number % 5 = 2 THEN 'bbbbbbbb-0000-0000-0000-000000000003'
            WHEN number % 5 = 3 THEN 'cccccccc-0000-0000-0000-000000000004'
            ELSE 'dddddddd-0000-0000-0000-000000000005'
           END AS tenant_id,
           generateUUIDv4() AS user_id
    FROM system.numbers
    LIMIT 500
) src
CROSS JOIN
(
    SELECT
        ['subscription','lead_form_submit','trial_signup','purchase','appointment_booked','demo_request','download_whitepaper'] AS ctypes,
        ctypes[toUInt8(1 + (rand() % toUInt8(length(ctypes))))] AS ctype
) conv_pool
CROSS JOIN
(
    SELECT
        ['tet_2025','summer_sale','flash_sale','vay_q1','back_to_school','vinhomes_2025','black_friday_2025','year_end'] AS utm_campaigns,
        utm_campaigns[toUInt8(1 + (rand() % toUInt8(length(utm_campaigns))))] AS uc
) uc_pool;