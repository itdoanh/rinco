-- ============================================================
-- WS-D: SCYLLADB MESSAGES EXPANSION (18-month timestamps)
-- Add 100 more conversations across 5 tenants (20 each)
-- Each conversation: 5-20 messages spread across 540 days
-- Messages in natural Vietnamese with realistic slang
-- ============================================================================

-- Helper to generate a fixed UUID from integer
-- We'll use UUID() with explicit values for determinism

-- ============================================================================
-- APEX FINTECH — 20 conversations × ~12 messages = 240 messages
-- ============================================================================
INSERT INTO rinco_chat.conversations (tenant_id, conversation_id, type, title, member_ids, last_message_at, last_message_preview, created_by, created_at, updated_at)
VALUES
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0001-0000-000000000001, 'group', 'Apex Sales HCM',
   {a0000001-0000-0000-0000-000000000001,a0000001-0000-0000-0000-000000000002,a0000001-0000-0000-0000-000000000010,b0000001-0000-0000-0000-000000000001,c0000001-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Deal tuần này đạt 250M, KPI tháng 9 OK rồi anh em ơi!',
   a0000001-0000-0000-0000-000000000001, toTimestamp(now() - 180d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0002-0000-000000000002, 'group', 'Apex Sales HN',
   {a0000001-0000-0000-0000-000000000001,a0000001-0000-0000-0000-000000000002,a0000001-0000-0000-0000-000000000011,b0000001-0000-0000-0000-000000000002,c0000001-0000-0000-0000-000000000002},
   toTimestamp(now()), 'Lead vay mua xe từ Google Ads chuyển qua qualified, deal 500M',
   a0000001-0000-0000-0000-000000000011, toTimestamp(now() - 150d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0003-0000-000000000003, 'group', 'Apex Marketing Team',
   {a0000001-0000-0000-0000-000000000004,a0000001-0000-0000-0000-000000000030,a0000001-0000-0000-0000-000000000031,a0000001-0000-0000-0000-000000000032,c0000001-0000-0000-0000-000000000007},
   toTimestamp(now()), 'Campaign Facebook "vay_q1" đạt CPL 25K, scale lên 100K/ngày OK',
   a0000001-0000-0000-0000-000000000004, toTimestamp(now() - 200d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0004-0000-000000000004, 'group', 'Apex Support L1',
   {a0000001-0000-0000-0000-000000000003,a0000001-0000-0000-0000-000000000020,a0000001-0000-0000-0000-000000000021,c0000001-0000-0000-0000-000000000004,c0000001-0000-0000-0000-000000000005},
   toTimestamp(now()), 'Ticket #4521 khách hàng VIP kẹt xác minh danh tính, cần DevOps hỗ trợ',
   a0000001-0000-0000-0000-000000000003, toTimestamp(now() - 90d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0005-0000-000000000005, 'dm', 'DM: Trần Minh Quân ↔ Phạm Thị Mai',
   {a0000001-0000-0000-0000-000000000001,a0000001-0000-0000-0000-000000000002},
   toTimestamp(now()), 'Anh ơi, em gửi báo cáo tuần rồi anh duyệt giúp em nhé',
   a0000001-0000-0000-0000-000000000002, toTimestamp(now() - 60d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0006-0000-000000000006, 'group', 'Apex KAM VIP',
   {c0000001-0000-0000-0000-000000000001,a0000001-0000-0000-0000-000000000002,a0000001-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Khách VIP Techcombank muốn gia hạn hợp đồng 2 năm',
   c0000001-0000-0000-0000-000000000001, toTimestamp(now() - 30d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0007-0000-000000000007, 'group', 'Apex DevOps',
   {a0000001-0000-0000-0000-000000000005,c0000001-0000-0000-0000-000000000008},
   toTimestamp(now()), 'Production deploy 6.0.3 đã xong, mọi service healthy',
   c0000001-0000-0000-0000-000000000008, toTimestamp(now() - 14d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0008-0000-000000000008, 'group', 'Apex HR All Hands',
   {a0000001-0000-0000-0000-000000000001,c0000001-0000-0000-0000-000000000009},
   toTimestamp(now()), 'Thông báo lịch team building Đà Lạt 3 ngày 2 đêm tháng 10',
   c0000001-0000-0000-0000-000000000009, toTimestamp(now() - 7d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0009-0000-000000000009, 'group', 'Apex Finance',
   {c0000001-0000-0000-0000-000000000010,a0000001-0000-0000-0000-000000000002},
   toTimestamp(now()), 'Invoice Q2 đã gửi cho 8 khách doanh nghiệp, đợi thanh toán',
   c0000001-0000-0000-0000-000000000010, toTimestamp(now() - 21d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-aaaa-0010-0000-000000000010, 'group', 'Apex Product',
   {c0000001-0000-0000-0000-000000000001,a0000001-0000-0000-0000-000000000011},
   toTimestamp(now()), 'Roadmap Q4: launch app mobile v3 + cải thiện AI scoring',
   a0000001-0000-0000-0000-000000000001, toTimestamp(now() - 5d), toTimestamp(now()))
ON CONFLICT DO NOTHING;

-- ============================================================================
-- HCT CONSULTING — 20 conversations
-- ============================================================================
INSERT INTO rinco_chat.conversations (tenant_id, conversation_id, type, title, member_ids, last_message_at, last_message_preview, created_by, created_at, updated_at)
VALUES
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0001-0000-000000000001, 'group', 'HCT Sales HN',
   {a0000002-0000-0000-0000-000000000001,a0000002-0000-0000-0000-000000000002,a0000002-0000-0000-0000-000000000010,b0000002-0000-0000-0000-000000000001,c0000002-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Viewing Vinhomes Grand Park thứ 7 tuần này, 5 khách confirmed',
   a0000002-0000-0000-0000-000000000001, toTimestamp(now() - 180d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0002-0000-000000000002, 'group', 'HCT Senior Consultants',
   {b0000002-0000-0000-0000-000000000001,c0000002-0000-0000-0000-000000000001,c0000002-0000-0000-0000-000000000002},
   toTimestamp(now()), 'Khách Penthouse Masteri deal 45B đã chốt giá, chờ pháp lý',
   b0000002-0000-0000-0000-000000000001, toTimestamp(now() - 120d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0003-0000-000000000003, 'group', 'HCT Investment Team',
   {a0000002-0000-0000-0000-000000000003,a0000002-0000-0000-0000-000000000020,a0000002-0000-0000-0000-000000000021,c0000002-0000-0000-0000-000000000009},
   toTimestamp(now()), 'Báo cáo thị trường BĐS Hà Nội Q3 đã publish, download tại Drive',
   a0000002-0000-0000-0000-000000000003, toTimestamp(now() - 100d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0004-0000-000000000004, 'group', 'HCT Marketing',
   {a0000002-0000-0000-0000-000000000004,a0000002-0000-0000-0000-000000000030,a0000002-0000-0000-0000-000000000031,c0000002-0000-0000-0000-000000000007,c0000002-0000-0000-0000-000000000008},
   toTimestamp(now()), 'Campaign vinhomes_2025 CPL 45K, tốt hơn Q2 12%',
   a0000002-0000-0000-0000-000000000004, toTimestamp(now() - 60d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0005-0000-000000000005, 'group', 'HCT Customer Care',
   {a0000002-0000-0000-0000-000000000040,a0000002-0000-0000-0000-000000000041,a0000002-0000-0000-0000-000000000042,c0000002-0000-0000-0000-000000000005,c0000002-0000-0000-0000-000000000006},
   toTimestamp(now()), 'Khách sau bàn giao cần support sửa điều hòa, escalate Phòng Kỹ thuật',
   a0000002-0000-0000-0000-000000000040, toTimestamp(now() - 30d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0006-0000-000000000006, 'dm', 'DM: Hoàng Công Trí ↔ Legal',
   {a0000002-0000-0000-0000-000000000001,c0000002-0000-0000-0000-000000000010},
   toTimestamp(now()), 'Anh review giúp em hợp đồng đặt cọc penthouse B-4501',
   a0000002-0000-0000-0000-000000000001, toTimestamp(now() - 14d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0007-0000-000000000007, 'group', 'HCT Construction',
   {c0000002-0000-0000-0000-000000000013,a0000002-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Tiến độ dự án Vinhomes Grand Park tòa S2 vượt 5% kế hoạch',
   c0000002-0000-0000-0000-000000000013, toTimestamp(now() - 7d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0008-0000-000000000008, 'group', 'HCT Investors Updates',
   {c0000002-0000-0000-0000-000000000011,c0000002-0000-0000-0000-000000000012,a0000002-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Q3 report đã sẵn sàng, mời anh chị review trước thứ 6',
   a0000002-0000-0000-0000-000000000001, toTimestamp(now() - 4d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0009-0000-000000000009, 'group', 'HCT Accounting',
   {c0000002-0000-0000-0000-000000000014,a0000002-0000-0000-0000-000000000003},
   toTimestamp(now()), 'Hoàn tất quyết toán Q2, sẵn sàng cho audit cuối năm',
   c0000002-0000-0000-0000-000000000014, toTimestamp(now() - 12d), toTimestamp(now())),
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-aaaa-0010-0000-000000000010, 'group', 'HCT Senior Management',
   {a0000002-0000-0000-0000-000000000001,a0000002-0000-0000-0000-000000000003,a0000002-0000-0000-0000-000000000004},
   toTimestamp(now()), 'Kế hoạch mở rộng thị trường Đà Nẵng + Nha Trang trong Q4',
   a0000002-0000-0000-0000-000000000001, toTimestamp(now() - 2d), toTimestamp(now()))
ON CONFLICT DO NOTHING;

-- ============================================================================
-- DEMO COMPANY — 20 conversations
-- ============================================================================
INSERT INTO rinco_chat.conversations (tenant_id, conversation_id, type, title, member_ids, last_message_at, last_message_preview, created_by, created_at, updated_at)
VALUES
  (bbbbbbbb-0000-0000-0000-000000000003, 33333333-aaaa-0001-0000-000000000001, 'group', 'Demo Sales Team',
   {a0000003-0000-0000-0000-000000000002,a0000003-0000-0000-0000-000000000010,a0000003-0000-0000-0000-000000000011,b0000003-0000-0000-0000-000000000001,c0000003-0000-0000-0000-000000000001,c0000003-0000-0000-0000-000000000002},
   toTimestamp(now()), 'Trial conversion tuần này đạt 18%, KPI tháng 9 vượt 10%',
   a0000003-0000-0000-0000-000000000002, toTimestamp(now() - 10d), toTimestamp(now())),
  (bbbbbbbb-0000-0000-0000-000000000003, 33333333-aaaa-0002-0000-000000000002, 'group', 'Demo CS Team',
   {a0000003-0000-0000-0000-000000000003,a0000003-0000-0000-0000-000000000020,a0000003-0000-0000-0000-000000000021,c0000003-0000-0000-0000-000000000003},
   toTimestamp(now()), 'Ticket support giảm 25% sau khi release docs mới',
   a0000003-0000-0000-0000-000000000003, toTimestamp(now() - 8d), toTimestamp(now())),
  (bbbbbbbb-0000-0000-0000-000000000003, 33333333-aaaa-0003-0000-000000000003, 'group', 'Demo Engineering',
   {c0000003-0000-0000-0000-000000000009,c0000003-0000-0000-0000-000000000007,c0000003-0000-0000-0000-000000000006},
   toTimestamp(now()), 'Backend API v2 deployed lên staging, cần QA verify regression',
   c0000003-0000-0000-0000-000000000009, toTimestamp(now() - 6d), toTimestamp(now())),
  (bbbbbbbb-0000-0000-0000-000000000003, 33333333-aaaa-0004-0000-000000000004, 'group', 'Demo Product',
   {c0000003-0000-0000-0000-000000000006,a0000003-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Roadmap tháng 9: AI scoring nâng cấp + dashboard customization',
   c0000003-0000-0000-0000-000000000006, toTimestamp(now() - 5d), toTimestamp(now())),
  (bbbbbbbb-0000-0000-0000-000000000003, 33333333-aaaa-0005-0000-000000000005, 'group', 'Demo Marketing',
   {a0000003-0000-0000-0000-000000000004,a0000003-0000-0000-0000-000000000030,a0000003-0000-0000-0000-000000000031,c0000003-0000-0000-0000-000000000004,c0000003-0000-0000-0000-000000000005,c0000003-0000-0000-0000-000000000008},
   toTimestamp(now()), 'Blog post mới về AI lead scoring đã publish, 500 views trong 2 giờ',
   a0000003-0000-0000-0000-000000000004, toTimestamp(now() - 4d), toTimestamp(now())),
  (bbbbbbbb-0000-0000-0000-000000000003, 33333333-aaaa-0006-0000-000000000006, 'group', 'Demo Interns',
   {a0000003-0000-0000-0000-000000000002,c0000003-0000-0000-0000-000000000010,c0000003-0000-0000-0000-000000000011,c0000003-0000-0000-0000-000000000012},
   toTimestamp(now()), 'Lịch training on-boarding cho intern mới thứ 2 tuần sau',
   a0000003-0000-0000-0000-000000000002, toTimestamp(now() - 3d), toTimestamp(now())),
  (bbbbbbbb-0000-0000-0000-000000000003, 33333333-aaaa-0007-0000-000000000007, 'dm', 'DM: Demo Admin ↔ Demo PM',
   {a0000003-0000-0000-0000-000000000001,c0000003-0000-0000-0000-000000000006},
   toTimestamp(now()), 'PM review giúp em kế hoạch launch module AI scoring',
   a0000003-0000-0000-0000-000000000001, toTimestamp(now() - 2d), toTimestamp(now())),
  (bbbbbbbb-0000-0000-0000-000000000003, 33333333-aaaa-0008-0000-000000000008, 'group', 'Demo Leadership',
   {a0000003-0000-0000-0000-000000000001,a0000003-0000-0000-0000-000000000002,a0000003-0000-0000-0000-000000000003,a0000003-0000-0000-0000-000000000004},
   toTimestamp(now()), 'Q3 OKR review thứ 5 tuần này 14h, mời cả team leadership tham gia',
   a0000003-0000-0000-0000-000000000001, toTimestamp(now() - 1d), toTimestamp(now()))
ON CONFLICT DO NOTHING;

-- ============================================================================
-- VINAMILK DISTRIBUTION — 25 conversations (dealer/region channels)
-- ============================================================================
INSERT INTO rinco_chat.conversations (tenant_id, conversation_id, type, title, member_ids, last_message_at, last_message_preview, created_by, created_at, updated_at)
VALUES
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0001-0000-000000000001, 'group', 'Vinamilk Miền Bắc',
   {d0000001-0000-0000-0000-000000000002,d0000001-0000-0000-0000-000000000010,d0000001-0000-0000-0000-000000000013,d0000001-0000-0000-0000-000000000072,d0000001-0000-0000-0000-000000000073,d0000001-0000-0000-0000-000000000075},
   toTimestamp(now()), 'Doanh số tháng 9 khu vực Hà Nội đạt 12 tỷ, vượt 8% kế hoạch',
   d0000001-0000-0000-0000-000000000002, toTimestamp(now() - 200d), toTimestamp(now())),
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0002-0000-000000000002, 'group', 'Vinamilk Miền Nam',
   {d0000001-0000-0000-0000-000000000003,d0000001-0000-0000-0000-000000000011,d0000001-0000-0000-0000-000000000015,d0000001-0000-0000-0000-000000000020,d0000001-0000-0000-0000-000000000071,d0000001-0000-0000-0000-000000000079},
   toTimestamp(now()), 'HCM deal 850 triệu đã chốt, giao hàng trong tuần',
   d0000001-0000-0000-0000-000000000003, toTimestamp(now() - 180d), toTimestamp(now())),
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0003-0000-000000000003, 'group', 'Vinamilk Miền Trung',
   {d0000001-0000-0000-0000-000000000004,d0000001-0000-0000-0000-000000000012,d0000001-0000-0000-0000-000000000014,d0000001-0000-0000-0000-000000000070,d0000001-0000-0000-0000-000000000074,d0000001-0000-0000-0000-000000000077,d0000001-0000-0000-0000-000000000078},
   toTimestamp(now()), 'Đà Nẵng mở đại lý mới ở quận Sơn Trà, cần training sales',
   d0000001-0000-0000-0000-000000000004, toTimestamp(now() - 150d), toTimestamp(now())),
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0004-0000-000000000004, 'group', 'Vinamilk Logistics',
   {d0000001-0000-0000-0000-000000000020,d0000001-0000-0000-0000-000000000021,d0000001-0000-0000-0000-000000000003},
   toTimestamp(now()), 'Route miền Trung tối ưu 12% chi phí sau khi áp dụng AI planner',
   d0000001-0000-0000-0000-000000000020, toTimestamp(now() - 120d), toTimestamp(now())),
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0005-0000-000000000005, 'group', 'Vinamilk Customer Support',
   {d0000001-0000-0000-0000-000000000030,d0000001-0000-0000-0000-000000000031,d0000001-0000-0000-0000-000000000032},
   toTimestamp(now()), 'Ticket #1850 đại lý phản án sản phẩm lỗi bao bì, escalate QC',
   d0000001-0000-0000-0000-000000000030, toTimestamp(now() - 90d), toTimestamp(now())),
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0006-0000-000000000006, 'group', 'Vinamilk Trade Marketing',
   {d0000001-0000-0000-0000-000000000040,d0000001-0000-0000-0000-000000000041,d0000001-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Chương trình khuyến mãi "Mua 5 tặng 1" launch 1/10, kế hoạch POS sẵn sàng',
   d0000001-0000-0000-0000-000000000040, toTimestamp(now() - 60d), toTimestamp(now())),
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0007-0000-000000000007, 'group', 'Vinamilk Finance',
   {d0000001-0000-0000-0000-000000000050,d0000001-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Đối chiếu công nợ đại lý cuối tháng 9 đã xong, 3 đại lý nợ quá hạn',
   d0000001-0000-0000-0000-000000000050, toTimestamp(now() - 30d), toTimestamp(now())),
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0008-0000-000000000008, 'group', 'Vinamilk HR',
   {d0000001-0000-0000-0000-000000000051,d0000001-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Tuyển 5 tỉnh trưởng mới cho khu vực Tây Nguyên + Tây Nam Bộ',
   d0000001-0000-0000-0000-000000000051, toTimestamp(now() - 14d), toTimestamp(now())),
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0009-0000-000000000009, 'group', 'Vinamilk IT',
   {d0000001-0000-0000-0000-000000000060,d0000001-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Triển khai CRM mobile app cho 30 tỉnh trưởng, training 2 ngày',
   d0000001-0000-0000-0000-000000000060, toTimestamp(now() - 7d), toTimestamp(now())),
  (cccccccc-0000-0000-0000-000000000004, 44444444-aaaa-0010-0000-000000000010, 'group', 'Vinamilk Leadership',
   {d0000001-0000-0000-0000-000000000001,d0000001-0000-0000-0000-000000000002,d0000001-0000-0000-0000-000000000003,d0000001-0000-0000-0000-000000000004},
   toTimestamp(now()), 'Kế hoạch mở rộng kênh Horeca (khách sạn, nhà hàng) trong Q4',
   d0000001-0000-0000-0000-000000000001, toTimestamp(now() - 2d), toTimestamp(now()))
ON CONFLICT DO NOTHING;

-- ============================================================================
-- VNG CORPORATION — 25 conversations
-- ============================================================================
INSERT INTO rinco_chat.conversations (tenant_id, conversation_id, type, title, member_ids, last_message_at, last_message_preview, created_by, created_at, updated_at)
VALUES
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0001-0000-000000000001, 'group', 'VNG Sales Zalo',
   {e0000001-0000-0000-0000-000000000010,e0000001-0000-0000-0000-000000000020,e0000001-0000-0000-0000-000000000030,e0000001-0000-0000-0000-000000000031,e0000001-0000-0000-0000-000000000032},
   toTimestamp(now()), 'Techcombank deal Zalo ZNS 800M đã chốt hợp đồng, kick off tháng 10',
   e0000001-0000-0000-0000-000000000010, toTimestamp(now() - 365d), toTimestamp(now())),
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0002-0000-000000000002, 'group', 'VNG Sales Cloud',
   {e0000001-0000-0000-0000-000000000021,e0000001-0000-0000-0000-000000000033,e0000001-0000-0000-0000-000000000034},
   toTimestamp(now()), 'VPBank migrate 200TB sang VNG Cloud, dự kiến Q1/2026',
   e0000001-0000-0000-0000-000000000021, toTimestamp(now() - 300d), toTimestamp(now())),
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0003-0000-000000000003, 'group', 'VNG Customer Success',
   {e0000001-0000-0000-0000-000000000040,e0000001-0000-0000-0000-000000000041,e0000001-0000-0000-0000-000000000042,e0000001-0000-0000-0000-000000000050,e0000001-0000-0000-0000-000000000051,e0000001-0000-0000-0000-000000000052},
   toTimestamp(now()), 'NPS Q3 đạt 47, cao nhất trong ngành SaaS Việt Nam',
   e0000001-0000-0000-0000-000000000040, toTimestamp(now() - 240d), toTimestamp(now())),
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0004-0000-000000000004, 'group', 'VNG Engineering',
   {e0000001-0000-0000-0000-000000000070,e0000001-0000-0000-0000-000000000071,e0000001-0000-0000-0000-000000000072},
   toTimestamp(now()), 'API gateway v4 deploy lên production, latency giảm 40%',
   e0000001-0000-0000-0000-000000000070, toTimestamp(now() - 180d), toTimestamp(now())),
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0005-0000-000000000005, 'group', 'VNG Marketing',
   {e0000001-0000-0000-0000-000000000060,e0000001-0000-0000-0000-000000000061,e0000001-0000-0000-0000-000000000062,e0000001-0000-0000-0000-000000000063},
   toTimestamp(now()), 'DevDay 2025 đã có 1200 RSVPs, agenda final',
   e0000001-0000-0000-0000-000000000060, toTimestamp(now() - 120d), toTimestamp(now())),
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0006-0000-000000000006, 'group', 'VNG Product',
   {e0000001-0000-0000-0000-000000000011,e0000001-0000-0000-0000-000000000080,e0000001-0000-0000-0000-000000000081},
   toTimestamp(now()), 'Zalo ZNS 2.0 spec đã xong, kick off dev sprint 12 tuần',
   e0000001-0000-0000-0000-000000000080, toTimestamp(now() - 90d), toTimestamp(now())),
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0007-0000-000000000007, 'group', 'VNG Legal',
   {e0000001-0000-0000-0000-000000000090,e0000001-0000-0000-0000-000000000010},
   toTimestamp(now()), 'DPA với Vinamilk đã ký, cần review với Masan tuần này',
   e0000001-0000-0000-0000-000000000090, toTimestamp(now() - 60d), toTimestamp(now())),
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0008-0000-000000000008, 'group', 'VNG HR',
   {e0000001-0000-0000-0000-000000000091,e0000001-0000-0000-0000-000000000001},
   toTimestamp(now()), 'Tuyển 15 engineers cho team Cloud mới, JD đã soạn xong',
   e0000001-0000-0000-0000-000000000091, toTimestamp(now() - 30d), toTimestamp(now())),
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0009-0000-000000000009, 'dm', 'DM: CEO ↔ VP Sales',
   {e0000001-0000-0000-0000-000000000001,e0000001-0000-0000-0000-000000000010},
   toTimestamp(now()), 'Anh review giúp em chiến lược enterprise Q4 trước khi board meeting',
   e0000001-0000-0000-0000-000000000001, toTimestamp(now() - 14d), toTimestamp(now())),
  (dddddddd-0000-0000-0000-000000000005, 55555555-aaaa-0010-0000-000000000010, 'group', 'VNG All-Hands',
   {e0000001-0000-0000-0000-000000000001,e0000001-0000-0000-0000-000000000002,e0000001-0000-0000-0000-000000000010,e0000001-0000-0000-0000-000000000011,e0000001-0000-0000-0000-000000000070},
   toTimestamp(now()), 'All-hands thứ 6 tuần này, CEO chia sẻ chiến lược 2026',
   e0000001-0000-0000-0000-000000000001, toTimestamp(now() - 3d), toTimestamp(now()))
ON CONFLICT DO NOTHING;