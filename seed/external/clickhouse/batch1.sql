-- ClickHouse seed BATCH 1/3: Tenant 1 Apex Fintech (1000 events)
INSERT INTO rinco_analytics.events_raw
(event_id, event_time, tenant_id, user_id, session_id, anonymous_id, event_name, source, utm_source, utm_medium, utm_campaign, utm_content, utm_term, fbclid, country, city, device_type, browser, os, ip_address, user_agent, referrer, page_url, properties)
SELECT
    generateUUIDv4(),
    now() - (rand() % (30 * 86400)),
    'aaaaaaaa-0000-0000-0000-000000000001',
    toUUID(concat('a0000001-0000-0000-0000-0000000000', lpad(toString(1 + (rand() % 9)), 2, '0'))),
    generateUUIDv4(),
    concat('anon_', toString(rand())),
    arrayJoin(['page_view', 'page_view', 'page_view', 'lead.created', 'lead.contacted', 'lead.qualified', 'conversion', 'button_click', 'form_submit', 'video_play']) AS event_name,
    arrayJoin(['facebook', 'facebook', 'google', 'tiktok', 'zalo', 'organic']),
    arrayJoin(['facebook', 'google', 'tiktok', 'zalo', 'direct']) AS utm_source,
    arrayJoin(['cpc', 'cpc', 'social', 'organic']) AS utm_medium,
    arrayJoin(['spring_loan_v1', 'q3_personal', 'cashback_50', 'instant_v2']) AS utm_campaign,
    MD5(toString(rand())),
    arrayJoin(['vay nhanh', 'vay tieu dung', '']),
    CASE WHEN rand() % 3 = 0 THEN concat('fb.', MD5(toString(rand()))) ELSE '' END,
    'Vietnam',
    arrayJoin(['Ho Chi Minh', 'Ha Noi', 'Da Nang', 'Can Tho']),
    arrayJoin(['mobile', 'mobile', 'desktop', 'tablet']),
    arrayJoin(['Chrome', 'Safari', 'Firefox']),
    arrayJoin(['Windows', 'Android', 'iOS']),
    toIPv4(concat('203.0.113.', toString(10 + rand() % 200))),
    'Mozilla/5.0',
    arrayJoin(['https://facebook.com', 'https://google.com', '']),
    arrayJoin(['/landing/home', '/landing/vay-mua-xe', '/dashboard', '/leads']),
    '{"source":"seed"}'
FROM numbers(1000);
SELECT 'Batch 1 done' AS status, count() AS rows FROM rinco_analytics.events_raw WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001';
