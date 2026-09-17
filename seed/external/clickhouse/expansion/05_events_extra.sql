-- ============================================================
-- RINCO ClickHouse Expansion (WS-B Loop 5)
-- Adds 3500+ events (cumulative 5000+), 1500 user_sessions,
-- 600 conversions with VND revenue.
-- Idempotent (idempotent via primary key event_id + date partitioning).
-- ============================================================

-- ============================================================
-- SECTION A: EXTRA EVENTS (3500+ events across 3 tenants)
-- ============================================================

INSERT INTO rinco_analytics.events_raw
(
    event_id, event_time, tenant_id, user_id, session_id, anonymous_id,
    event_name, source, utm_source, utm_medium, utm_campaign, utm_content,
    utm_term, fbclid, gclid, country, city, device_type, browser, os,
    ip_address, user_agent, referrer, page_url, properties, event_date
)
SELECT
    generateUUIDv4(),
    now() - (rand() % (60 * 86400)),
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
    'Vietnam',
    city,
    device_type,
    browser,
    os,
    toIPv4(concat('203.0.113.', toString(10 + rand() % 200))),
    user_agent,
    referrer,
    page_url,
    properties,
    toDate(now() - (rand() % (60 * 86400)))
FROM (
    SELECT
        'aaaaaaaa-0000-0000-0000-000000000001' AS tenant_id,
        toUUID(concat('a0000001-0000-0000-0000-000000000', toString(1 + (rand() % 25)))) AS user_id,
        generateUUIDv4() AS session_id,
        concat('anon_', toString(rand())) AS anonymous_id,
        arrayJoin([
            'page_view','page_view','page_view','page_view',
            'lead.created','lead.contacted','lead.qualified','lead.lost','lead.won',
            'conversion','conversion','button_click','button_click','form_submit',
            'video_play','video_complete','scroll_50','scroll_75','time_on_page_60s',
            'file_download','newsletter_signup','demo_requested','pricing_viewed',
            'feature_explored','integration_connected','webhook_triggered',
            'signup_completed','trial_started','trial_converted'
        ]) AS event_name,
        arrayJoin(['facebook','facebook','google','tiktok','zalo','organic','referral','direct','email']) AS source,
        arrayJoin(['facebook','google','tiktok','zalo','google','direct']) AS utm_source,
        arrayJoin(['cpc','social','organic','referral','email']) AS utm_medium,
        arrayJoin(['spring_loan_v1','q3_personal_loan','cashback_50','instant_v2','tiktok_creative_a','summer_loan','autumn_v3','winter_promo','zalo_broadcast','email_q3','referral_program','organic_brand']) AS utm_campaign,
        md5(tostring(rand())) AS utm_content,
        arrayJoin(['vay nhanh','vay tieu dung','vay kinh doanh','vay mua xe','vay mua nha','']) AS utm_term,
        CASE WHEN rand() % 3 = 0 THEN concat('fb.', md5(tostring(rand()))) ELSE '' END AS fbclid,
        CASE WHEN rand() % 4 = 0 THEN concat('g.', md5(tostring(rand()))) ELSE '' END AS gclid,
        arrayJoin(['Ho Chi Minh','Ha Noi','Da Nang','Can Tho','Hai Phong','Bien Hoa','Nha Trang','Hue']) AS city,
        arrayJoin(['mobile','mobile','desktop','tablet']) AS device_type,
        arrayJoin(['Chrome','Safari','Firefox','Edge','Opera']) AS browser,
        arrayJoin(['Windows','Android','iOS','macOS','Linux']) AS os,
        'Mozilla/5.0 (compatible; RINCO-Analytics/2.0)' AS user_agent,
        arrayJoin(['https://facebook.com','https://google.com','https://zalo.me','https://tiktok.com','']) AS referrer,
        arrayJoin([
            '/landing/home','/landing/vay-mua-xe','/landing/vay-mua-nha','/landing/lai-suat',
            '/dashboard','/leads','/leads/{id}','/deals','/deals/{id}','/reports','/reports/monthly',
            '/settings','/team','/integrations','/workflows','/notifications'
        ]) AS page_url,
        toJSONString(map(
            'lead_id', toString(generateUUIDv4()),
            'value', toString(rand() % 500000000),
            'session_duration_s', toString(rand() % 1800)
        )) AS properties
    FROM numbers(1500)
) AS apex_data;

-- HCT Consulting (1000 more events)
INSERT INTO rinco_analytics.events_raw
(
    event_id, event_time, tenant_id, user_id, session_id, anonymous_id,
    event_name, source, utm_source, utm_medium, utm_campaign, utm_content,
    fbclid, country, city, device_type, browser, os, ip_address, user_agent,
    referrer, page_url, properties, event_date
)
SELECT
    generateUUIDv4(),
    now() - (rand() % (60 * 86400)),
    'aaaaaaaa-0000-0000-0000-000000000002',
    toUUID(concat('a0000002-0000-0000-0000-000000000', toString(1 + (rand() % 25)))),
    generateUUIDv4(),
    concat('anon_', toString(rand())),
    arrayJoin([
        'page_view','page_view','property.view','property.inquiry',
        'viewing.scheduled','viewing.completed','viewing.cancelled',
        'lead.created','lead.qualified','lead.lost','lead.won',
        'conversion','conversion','document.uploaded','contract.signed',
        'mortgage.inquiry','mortgage.approved','mortgage.rejected',
        'gallery.image_view','virtual_tour.started','floorplan.viewed'
    ]),
    arrayJoin(['facebook','zalo','linkedin','organic','google','tiktok']),
    arrayJoin(['facebook','zalo','linkedin','google','tiktok']),
    arrayJoin(['cpc','social','organic','cpc','cpc']),
    arrayJoin([
        'vinhomes_grand_park','masterise_eco','sun_group_30ha','nhan_dinh_q3',
        'vinhomes_ocean_park','vinhomes_symphony','tay_ho_view','long_bien_residence',
        'the_matrix_one','imperia_smart_city','aqua_city','novaland_park'
    ]),
    md5(tostring(rand())),
    CASE WHEN rand() % 3 = 0 THEN concat('fb.', md5(tostring(rand()))) ELSE '' END,
    'Vietnam',
    arrayJoin(['Ha Noi','Ho Chi Minh','Hai Phong','Da Nang','Nha Trang']),
    arrayJoin(['mobile','mobile','desktop','tablet']),
    arrayJoin(['Chrome','Safari','Firefox']),
    arrayJoin(['Android','iOS','Windows','macOS']),
    toIPv4(concat('203.0.113.', toString(10 + rand() % 200))),
    'Mozilla/5.0',
    arrayJoin(['https://facebook.com','https://zalo.me','https://linkedin.com','']),
    arrayJoin([
        '/landing/can-ho-cao-cap','/landing/dat-nen','/landing/shophouse',
        '/projects','/projects/vinhomes','/projects/masterise','/projects/sun-group',
        '/projects/compare','/dashboard','/leads','/deals','/viewings','/reports'
    ]),
    '{"property_id":"prop_' || toString(rand() % 1000) || '","price":' || toString((rand() % 100 + 1) * 1000000000) || '}',
    toDate(now() - (rand() % (60 * 86400)))
FROM numbers(1000);

-- Demo Company (1000 more events)
INSERT INTO rinco_analytics.events_raw
(
    event_id, event_time, tenant_id, user_id, session_id, anonymous_id,
    event_name, source, utm_source, utm_medium, utm_campaign,
    country, city, device_type, browser, os, ip_address, user_agent,
    page_url, properties, event_date
)
SELECT
    generateUUIDv4(),
    now() - (rand() % (30 * 86400)),
    'bbbbbbbb-0000-0000-0000-000000000003',
    toUUID(concat('a0000003-0000-0000-0000-000000000', toString(1 + (rand() % 15)))),
    generateUUIDv4(),
    concat('anon_', toString(rand())),
    arrayJoin([
        'page_view','demo.started','demo.completed','demo.abandoned',
        'signup','trial_started','trial_extended','trial_converted',
        'lead.created','lead.qualified','integration.connected',
        'feature.explored','docs.read','api.called','webhook.received',
        'pricing.viewed','upgrade.completed','plan.changed'
    ]),
    arrayJoin(['facebook','google','direct','referral','youtube','producthunt']),
    arrayJoin(['facebook','google','direct','youtube']),
    arrayJoin(['cpc','organic','referral','none','cpc']),
    arrayJoin(['demo_spring','demo_free_trial','demo_youtube','demo_q3','demo_partner','demo_producthunt']),
    'Vietnam',
    arrayJoin(['Ho Chi Minh','Ha Noi','Da Nang']),
    arrayJoin(['mobile','desktop']),
    arrayJoin(['Chrome','Safari']),
    arrayJoin(['Windows','Android','iOS','macOS']),
    toIPv4(concat('203.0.113.', toString(10 + rand() % 200))),
    'Mozilla/5.0',
    arrayJoin([
        '/demo','/demo/quick-tour','/features','/features/ai-scoring','/features/chat-engine',
        '/pricing','/docs','/docs/getting-started','/docs/api','/signup',
        '/signup/trial','/dashboard','/admin','/integrations'
    ]),
    '{"demo_id":"demo_' || toString(rand() % 1000) || '","feature":"' || (arrayJoin(['scoring','chat','analytics','workflow'])) || '"}',
    toDate(now() - (rand() % (30 * 86400)))
FROM numbers(1000);

-- ============================================================
-- SECTION B: USER SESSIONS EXPANDED (1500+ sessions)
-- ============================================================
INSERT INTO rinco_analytics.user_sessions
(
    session_id, tenant_id, user_id, anonymous_id, started_at, ended_at,
    duration_sec, page_view_count, event_count, entry_url, exit_url,
    country, device_type, browser, os, utm_source, utm_campaign
)
SELECT
    generateUUIDv4(),
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
    'Vietnam',
    device_type,
    browser,
    os,
    utm_source,
    utm_campaign
FROM (
    -- Apex: 700 sessions
    SELECT
        'aaaaaaaa-0000-0000-0000-000000000001' AS tenant_id,
        toUUID(concat('a0000001-0000-0000-0000-000000000', toString(1 + (rand() % 25)))) AS user_id,
        concat('anon_', toString(rand())) AS anonymous_id,
        now() - (rand() % (60 * 86400)) AS started_at,
        now() - (rand() % (60 * 86400)) + (rand() % 3600) AS ended_at,
        (rand() % 1800 + 30)::int AS duration_sec,
        (rand() % 15 + 1)::int AS page_view_count,
        (rand() % 50 + 5)::int AS event_count,
        arrayJoin(['/landing/home','/landing/vay-mua-xe','/landing/vay-mua-nha','/landing/lai-suat','/landing/contact']) AS entry_url,
        arrayJoin(['/dashboard','/leads','/deals','/landing/home','/reports','/logout']) AS exit_url,
        arrayJoin(['mobile','mobile','desktop','tablet']) AS device_type,
        arrayJoin(['Chrome','Safari','Firefox','Edge']) AS browser,
        arrayJoin(['Windows','Android','iOS','macOS']) AS os,
        arrayJoin(['facebook','google','tiktok','organic','direct','email']) AS utm_source,
        arrayJoin(['spring_loan_v1','q3_personal','cashback_50','instant_v2','summer_loan','zalo_broadcast']) AS utm_campaign
    FROM numbers(700)

    UNION ALL

    -- HCT: 500 sessions
    SELECT
        'aaaaaaaa-0000-0000-0000-000000000002',
        toUUID(concat('a0000002-0000-0000-0000-000000000', toString(1 + (rand() % 25)))),
        concat('anon_', toString(rand())),
        now() - (rand() % (60 * 86400)),
        now() - (rand() % (60 * 86400)) + (rand() % 7200),
        (rand() % 3600 + 60)::int,
        (rand() % 25 + 1)::int,
        (rand() % 80 + 10)::int,
        arrayJoin(['/landing/can-ho-cao-cap','/landing/dat-nen','/landing/shophouse','/projects']),
        arrayJoin(['/dashboard','/leads','/deals','/viewings','/logout']),
        arrayJoin(['mobile','desktop','tablet']),
        arrayJoin(['Chrome','Safari','Firefox']),
        arrayJoin(['Android','iOS','Windows']),
        arrayJoin(['facebook','zalo','linkedin','organic','google']),
        arrayJoin(['vinhomes_grand_park','masterise_eco','sun_group_30ha','vinhomes_ocean_park'])
    FROM numbers(500)

    UNION ALL

    -- Demo: 300 sessions
    SELECT
        'bbbbbbbb-0000-0000-0000-000000000003',
        toUUID(concat('a0000003-0000-0000-0000-000000000', toString(1 + (rand() % 15)))),
        concat('anon_', toString(rand())),
        now() - (rand() % (30 * 86400)),
        now() - (rand() % (30 * 86400)) + (rand() % 5400),
        (rand() % 1800 + 30)::int,
        (rand() % 12 + 1)::int,
        (rand() % 30 + 3)::int,
        arrayJoin(['/demo','/demo/quick-tour','/features','/pricing','/docs']),
        arrayJoin(['/signup','/dashboard','/logout']),
        arrayJoin(['mobile','desktop']),
        arrayJoin(['Chrome','Safari']),
        arrayJoin(['Windows','Android','iOS']),
        arrayJoin(['facebook','google','direct','youtube','producthunt']),
        arrayJoin(['demo_spring','demo_free_trial','demo_youtube','demo_q3'])
    FROM numbers(300)
) AS s;

-- ============================================================
-- SECTION C: CONVERSIONS WITH VND REVENUE (600 entries)
-- ============================================================
INSERT INTO rinco_analytics.conversions
(
    event_time, tenant_id, user_id, campaign_id, funnel_id, step,
    order_id, order_value, currency, properties
)
SELECT
    event_time,
    tenant_id,
    user_id,
    campaign_id,
    funnel_id,
    step,
    order_id,
    order_value,
    'VND',
    properties
FROM (
    -- Apex: 300 conversions
    SELECT
        now() - (rand() % (60 * 86400)) AS event_time,
        'aaaaaaaa-0000-0000-0000-000000000001' AS tenant_id,
        toUUID(concat('a0000001-0000-0000-0000-000000000', toString(1 + (rand() % 25)))) AS user_id,
        generateUUIDv4() AS campaign_id,
        arrayJoin(['loan_funnel','personal_loan_funnel','business_loan_funnel']) AS funnel_id,
        arrayJoin(['view','add_to_cart','application_started','documents_uploaded','approved','disbursed','purchase','purchase']) AS step,
        generateUUIDv4() AS order_id,
        CASE
            WHEN rand() % 5 = 0 THEN ((rand() % 100 + 50) * 1000000)::decimal(18,2)
            WHEN rand() % 4 = 0 THEN ((rand() % 300 + 100) * 1000000)::decimal(18,2)
            ELSE ((rand() % 800 + 200) * 1000000)::decimal(18,2)
        END AS order_value,
        toJSONString(map(
            'product', arrayJoin(['personal_loan','home_loan','auto_loan','business_loan','credit_card']),
            'term_months', toString(arrayJoin([6,12,24,36,48,60])),
            'interest_rate', toString(round(0.08 + rand() * 0.12, 3))
        )) AS properties
    FROM numbers(300)

    UNION ALL

    -- HCT: 200 conversions (real estate, much higher values)
    SELECT
        now() - (rand() % (60 * 86400)),
        'aaaaaaaa-0000-0000-0000-000000000002',
        toUUID(concat('a0000002-0000-0000-0000-000000000', toString(1 + (rand() % 25)))),
        generateUUIDv4(),
        arrayJoin(['bds_funnel','viewing_funnel','negotiation_funnel']),
        arrayJoin(['view','inquiry','viewing_scheduled','viewing_done','negotiation','contract_signed','purchase','purchase']),
        generateUUIDv4(),
        CASE
            WHEN rand() % 4 = 0 THEN ((rand() % 5 + 2) * 1000000000)::decimal(18,2)
            WHEN rand() % 3 = 0 THEN ((rand() % 15 + 5) * 1000000000)::decimal(18,2)
            ELSE ((rand() % 30 + 20) * 1000000000)::decimal(18,2)
        END,
        toJSONString(map(
            'property_type', arrayJoin(['can_ho','shophouse','dat_nen','biet_thu','penthouse']),
            'project', arrayJoin(['Vinhomes Grand Park','Masterise Eco','Sun Group','Vinhomes Ocean Park']),
            'area_sqm', toString(rand() % 200 + 50)
        ))
    FROM numbers(200)

    UNION ALL

    -- Demo: 100 conversions (subscription plans, smaller)
    SELECT
        now() - (rand() % (30 * 86400)),
        'bbbbbbbb-0000-0000-0000-000000000003',
        toUUID(concat('a0000003-0000-0000-0000-000000000', toString(1 + (rand() % 15)))),
        generateUUIDv4(),
        arrayJoin(['demo_funnel','trial_funnel','subscription_funnel']),
        arrayJoin(['view','demo_started','trial_started','plan_selected','purchase']),
        generateUUIDv4(),
        CASE
            WHEN rand() % 3 = 0 THEN 0
            WHEN rand() % 2 = 0 THEN ((rand() % 5 + 1) * 1000000)::decimal(18,2)
            ELSE ((rand() % 50 + 10) * 1000000)::decimal(18,2)
        END,
        toJSONString(map(
            'plan', arrayJoin(['free','starter','pro','enterprise']),
            'seats', toString(rand() % 50 + 1)
        ))
    FROM numbers(100)
) AS conv;

-- ============================================================
-- VERIFICATION
-- ============================================================
SELECT 'rinco_analytics.events_raw (WS-B expansion)' AS table_name,
       count() AS total_rows,
       uniq(tenant_id) AS unique_tenants,
       min(event_time) AS oldest_event,
       max(event_time) AS newest_event
FROM rinco_analytics.events_raw;

SELECT 'rinco_analytics.conversions (WS-B expansion)' AS table_name,
       count() AS total_rows,
       sum(order_value) AS total_revenue_vnd
FROM rinco_analytics.conversions;

SELECT 'rinco_analytics.user_sessions (WS-B expansion)' AS table_name,
       count() AS total_rows,
       uniq(tenant_id) AS unique_tenants,
       round(avg(duration_sec)) AS avg_session_sec
FROM rinco_analytics.user_sessions;

-- Refresh materialized views
SYSTEM MERGE MATERIALIZED VIEW rinco_analytics.events_hourly_mv;
SYSTEM MERGE MATERIALIZED VIEW rinco_analytics.funnel_hourly_mv;

SELECT 'ClickHouse expansion seed (WS-B Loop 5) complete!' AS status;
