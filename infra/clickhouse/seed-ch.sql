-- ClickHouse seed fixtures (dev only)

INSERT INTO rinco_analytics.events_raw
  (event_time, event_id, tenant_id, event_name, source, country, device_type)
VALUES
  (now() - INTERVAL 5 MINUTE, generateUUIDv4(), '11111111-1111-1111-1111-111111111111', 'lead.created', 'facebook', 'VN', 'mobile'),
  (now() - INTERVAL 4 MINUTE, generateUUIDv4(), '11111111-1111-1111-1111-111111111111', 'lead.qualified', 'tiktok', 'VN', 'desktop'),
  (now() - INTERVAL 3 MINUTE, generateUUIDv4(), '11111111-1111-1111-1111-111111111111', 'deal.won',        'organic',  'VN', 'desktop'),
  (now() - INTERVAL 2 MINUTE, generateUUIDv4(), '11111111-1111-1111-1111-111111111111', 'lead.created', 'zalo',     'VN', 'tablet');

INSERT INTO rinco_analytics.conversions
  (event_time, tenant_id, user_id, funnel_id, step, order_value)
VALUES
  (now() - INTERVAL 6 MINUTE, '11111111-1111-1111-1111-111111111111', generateUUIDv4(), 'funnel_1', 'view',         0.00),
  (now() - INTERVAL 5 MINUTE, '11111111-1111-1111-1111-111111111111', generateUUIDv4(), 'funnel_1', 'add_to_cart',  0.00),
  (now() - INTERVAL 3 MINUTE, '11111111-1111-1111-1111-111111111111', generateUUIDv4(), 'funnel_1', 'checkout',     0.00),
  (now() - INTERVAL 2 MINUTE, '11111111-1111-1111-1111-111111111111', generateUUIDv4(), 'funnel_1', 'purchase',    1500000.00);
