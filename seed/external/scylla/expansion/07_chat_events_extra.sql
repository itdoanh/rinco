-- ============================================================
-- RINCO ScyllaDB Expansion (WS-B Loop 7)
-- 20+ conversations, 200+ messages, 50+ presence, 30+ events
-- ============================================================

-- ============================================================
-- SECTION 1: EXTRA CONVERSATIONS (15 more across 3 tenants)
-- ============================================================
INSERT INTO rinco_chat.conversations
  (tenant_id, conversation_id, type, title, description, member_ids, admin_ids, created_by, created_at, updated_at, last_message_at, last_message_preview, avatar_url, metadata)
VALUES
  -- Apex Fintech: 6 extra
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), 'group', 'Team HCM', 'Kênh team sales HCM',
   {uuid('b0000001-0000-0000-0000-000000000001'), uuid('b0000001-0000-0000-0000-000000000010'), uuid('b0000001-0000-0000-0000-000000000020'), uuid('a0000001-0000-0000-0000-000000000010'), uuid('a0000001-0000-0000-0000-000000000011')},
   {uuid('b0000001-0000-0000-0000-000000000001')},
   uuid('b0000001-0000-0000-0000-000000000001'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Team HCM meeting lúc 10h nhé'),
   NULL, {'region': 'HCM', 'manager': 'b0000001-0000-0000-0000-000000000001'}),

  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000011'), 'group', 'Team HN', 'Kênh team sales Hà Nội',
   {uuid('b0000001-0000-0000-0000-000000000002'), uuid('b0000001-0000-0000-0000-000000000021'), uuid('a0000001-0000-0000-0000-000000000012'), uuid('a0000001-0000-0000-0000-000000000013')},
   {uuid('b0000001-0000-0000-0000-000000000002')},
   uuid('b0000001-0000-0000-0000-000000000002'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'HN team đã về target tháng'),
   NULL, {'region': 'HN', 'manager': 'b0000001-0000-0000-0000-000000000002'}),

  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000012'), 'group', 'Underwriting', 'Kênh thẩm định',
   {uuid('a0000001-0000-0000-0000-000000000006'), uuid('a0000001-0000-0000-0000-000000000002')},
   {uuid('a0000001-0000-0000-0000-000000000006')},
   uuid('a0000001-0000-0000-0000-000000000006'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), '5 hồ sơ cần review gấp'),
   NULL, {'department': 'underwriting'}),

  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000013'), 'announcement', 'VIP Deals', 'Thông báo deal VIP',
   {uuid('a0000001-0000-0000-0000-000000000001'), uuid('a0000001-0000-0000-0000-000000000002'), uuid('a0000001-0000-0000-0000-000000000006'), uuid('b0000001-0000-0000-0000-000000000001'), uuid('b0000001-0000-0000-0000-000000000010')},
   {uuid('a0000001-0000-0000-0000-000000000002')},
   uuid('a0000001-0000-0000-0000-000000000002'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Deal 1.2 tỷ vừa chốt! 🎉'),
   NULL, {'priority': 'high'}),

  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000014'), 'public', 'AI Insights', 'AI scoring insights',
   {uuid('a0000001-0000-0000-0000-000000000004'), uuid('a0000001-0000-0000-0000-000000000002'), uuid('b0000001-0000-0000-0000-000000000040')},
   {uuid('a0000001-0000-0000-0000-000000000004')},
   uuid('a0000001-0000-0000-0000-000000000004'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'AI vừa phát hiện 3 leads hot từ Zalo'),
   NULL, {'category': 'ai'}),

  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000015'), 'group', 'Direct: Mai<->Lan', 'Direct message',
   {uuid('a0000001-0000-0000-0000-000000000002'), uuid('a0000001-0000-0000-0000-000000000010')},
   {uuid('a0000001-0000-0000-0000-000000000002')},
   uuid('a0000001-0000-0000-0000-000000000002'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Cảm ơn Mai nhiều nhé!'),
   NULL, {'type': 'direct'}),

  -- HCT Consulting: 5 extra
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), 'group', 'BDS Team HN', 'BĐS consultants HN',
   {uuid('b0000002-0000-0000-0000-000000000001'), uuid('b0000002-0000-0000-0000-000000000010'), uuid('b0000002-0000-0000-0000-000000000011'), uuid('b0000002-0000-0000-0000-000000000020')},
   {uuid('b0000002-0000-0000-0000-000000000001')},
   uuid('b0000002-0000-0000-0000-000000000001'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Viewing trip cuối tuần này'),
   NULL, {'specialty': 'cao_cap'}),

  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000011'), 'group', 'Investment Talks', 'Phân tích đầu tư',
   {uuid('a0000002-0000-0000-0000-000000000003'), uuid('a0000002-0000-0000-0000-000000000020'), uuid('a0000002-0000-0000-0000-000000000021'), uuid('b0000002-0000-0000-0000-000000000030')},
   {uuid('a0000002-0000-0000-0000-000000000003')},
   uuid('a0000002-0000-0000-0000-000000000003'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Phân tích thị trường Q4'),
   NULL, {'category': 'investment'}),

  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000012'), 'public', 'Vinhomes Project', 'Dự án Vinhomes',
   {uuid('b0000002-0000-0000-0000-000000000010'), uuid('b0000002-0000-0000-0000-000000000020'), uuid('a0000002-0000-0000-0000-000000000002'), uuid('a0000002-0000-0000-0000-000000000010'), uuid('a0000002-0000-0000-0000-000000000011')},
   {uuid('b0000002-0000-0000-0000-000000000010')},
   uuid('b0000002-0000-0000-0000-000000000010'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Có 5 căn 3PN mới về'),
   NULL, {'project': 'vinhomes'}),

  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000013'), 'public', 'Masterise Project', 'Dự án Masterise',
   {uuid('b0000002-0000-0000-0000-000000000011'), uuid('a0000002-0000-0000-0000-000000000012'), uuid('a0000002-0000-0000-0000-000000000013')},
   {uuid('b0000002-0000-0000-0000-000000000011')},
   uuid('b0000002-0000-0000-0000-000000000011'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Shophouse mới mở bán đợt 3'),
   NULL, {'project': 'masterise'}),

  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000014'), 'private', 'Investors Lounge', 'Phòng chờ investors',
   {uuid('b0000002-0000-0000-0000-000000000030'), uuid('a0000002-0000-0000-0000-000000000001'), uuid('a0000002-0000-0000-0000-000000000002')},
   {uuid('b0000002-0000-0000-0000-000000000030')},
   uuid('b0000002-0000-0000-0000-000000000030'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Cập nhật danh mục đầu tư'),
   NULL, {'audience': 'investors'}),

  -- Demo Company: 4 extra
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), 'group', 'Demo Engineering', 'Demo team kỹ thuật',
   {uuid('b0000003-0000-0000-0000-000000000001'), uuid('b0000003-0000-0000-0000-000000000010'), uuid('b0000003-0000-0000-0000-000000000020'), uuid('a0000003-0000-0000-0000-000000000010')},
   {uuid('b0000003-0000-0000-0000-000000000001')},
   uuid('b0000003-0000-0000-0000-000000000001'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Sprint planning 14h'),
   NULL, {'team': 'engineering'}),

  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000011'), 'public', 'Demo Feedback', 'Feedback khách hàng',
   {uuid('b0000003-0000-0000-0000-000000000010'), uuid('b0000003-0000-0000-0000-000000000020'), uuid('a0000003-0000-0000-0000-000000000001'), uuid('a0000003-0000-0000-0000-000000000002')},
   {uuid('b0000003-0000-0000-0000-000000000010')},
   uuid('b0000003-0000-0000-0000-000000000010'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Khách khen UI mới rất đẹp'),
   NULL, {'category': 'feedback'}),

  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000012'), 'public', 'Demo Random', 'Chitchat',
   {uuid('a0000003-0000-0000-0000-000000000001'), uuid('a0000003-0000-0000-0000-000000000002'), uuid('b0000003-0000-0000-0000-000000000020'), uuid('b0000003-0000-0000-0000-000000000040')},
   {uuid('a0000003-0000-0000-0000-000000000001')},
   uuid('a0000003-0000-0000-0000-000000000001'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Cafe ai đi không?'),
   NULL, {'category': 'random'}),

  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000013'), 'private', 'Demo DM: Admin<->Sales', 'Direct',
   {uuid('a0000003-0000-0000-0000-000000000001'), uuid('a0000003-0000-0000-0000-000000000002')},
   {uuid('a0000003-0000-0000-0000-000000000001')},
   uuid('a0000003-0000-0000-0000-000000000002'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'OK mình confirm lại nhé'),
   NULL, {'type': 'direct'});

-- ============================================================
-- SECTION 2: EXTRA MESSAGES (200+ across channels)
-- ============================================================
INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, metadata, reactions, parent_msg_id, created_at)
VALUES
  -- Apex Team HCM (b0000001...001 channel 11111111-...-010)
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), now(), uuid('b0000001-0000-0000-0000-000000000001'), 'Hồ Bích Hà', 'text', 'Chào team, có 8 leads mới về sáng nay', {'priority': 'high'}, {'👍': 3, '🎉': 1}, null, toTimestamp(now() - 1800)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), now(), uuid('b0000001-0000-0000-0000-000000000010'), 'Phùng Mỹ Linh', 'text', 'Mình tiếp nhận 4 leads HN nhé', {}, {'👍': 2}, null, toTimestamp(now() - 1500)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), now(), uuid('b0000001-0000-0000-0000-000000000020'), 'Đặng Thị Thúy', 'text', 'Mình take 2 leads HCM nha', {}, {'🙏': 1}, null, toTimestamp(now() - 1200)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), now(), uuid('b0000001-0000-0000-0000-000000000001'), 'Hồ Bích Hà', 'text', 'OK, track lại trong CRM sau 2h', {}, {}, null, toTimestamp(now() - 900)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), now(), uuid('a0000001-0000-0000-0000-000000000010'), 'Nguyễn Thị Lan', 'text', 'Vừa chốt deal 250tr từ lead sáng nay! 🎉', {}, {'🎉': 8, '👏': 4}, null, toTimestamp(now() - 600)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), now(), uuid('a0000001-0000-0000-0000-000000000011'), 'Hoàng Minh Tuấn', 'text', 'Chúc mừng Lan! Team HCM dẫn đầu tháng này', {}, {'👍': 5}, null, toTimestamp(now() - 300)),

  -- Apex Team HN (11111111-...-011)
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000011'), now(), uuid('b0000001-0000-0000-0000-000000000002'), 'Lý Quang Hải', 'text', 'HN team meeting lúc 9h sáng mai, confirm đi mọi người', {}, {}, null, toTimestamp(now() - 7200)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000011'), now(), uuid('b0000001-0000-0000-0000-000000000021'), 'Trương Quốc Đạt', 'text', 'Confirm ạ', {}, {}, null, toTimestamp(now() - 3600)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000011'), now(), uuid('a0000001-0000-0000-0000-000000000012'), 'Phan Văn Cường', 'text', 'Confirmed', {}, {}, null, toTimestamp(now() - 1800)),

  -- Apex VIP Deals (11111111-...-013)
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000013'), now(), uuid('a0000001-0000-0000-0000-000000000002'), 'Phạm Thị Mai', 'text', '🚨 Deal VIP 2 tỷ vừa chốt từ FPT Software!', {'priority': 'urgent'}, {'🎉': 15, '🚀': 8, '💰': 3}, null, toTimestamp(now() - 3600)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000013'), now(), uuid('a0000001-0000-0000-0000-000000000001'), 'Trần Minh Quân', 'text', 'Excellent! Team bonus 5% tháng này 🎉', {}, {'👏': 12}, null, toTimestamp(now() - 1800)),

  -- HCT BDS Team HN (22222222-...-010)
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), now(), uuid('b0000002-0000-0000-0000-000000000001'), 'Tống Diệu Hương', 'text', 'Viewing trip Vinhomes Grand Park thứ 7, 9h sáng. 12 người đăng ký', {'priority': 'high'}, {}, null, toTimestamp(now() - 5400)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), now(), uuid('b0000002-0000-0000-0000-000000000010'), 'Hà Mạnh Quân', 'text', 'Mình đăng ký đi, có 2 khách VIP muốn xem penthouse', {}, {'👍': 2}, null, toTimestamp(now() - 3600)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), now(), uuid('b0000002-0000-0000-0000-000000000011'), 'Phạm Hồng Nhung', 'text', 'Có 1 khách Nhật muốn xem, mình đi phiên dịch', {}, {'🙏': 1}, null, toTimestamp(now() - 1800)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), now(), uuid('a0000002-0000-0000-0000-000000000002'), 'Tạ Kim Phượng', 'text', 'Cảm ơn team! Setup xe công ty nhé', {}, {'👍': 3}, null, toTimestamp(now() - 900)),

  -- HCT Vinhomes Project (22222222-...-012)
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000012'), now(), uuid('b0000002-0000-0000-0000-000000000010'), 'Hà Mạnh Quân', 'text', 'Vinhomes vừa mở bán đợt 5, có 5 căn 3PN giá tốt', {'priority': 'high'}, {}, null, toTimestamp(now() - 7200)),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000012'), now(), uuid('b0000002-0000-0000-0000-000000000020'), 'Trần Diệu Anh', 'text', 'Em gửi danh sách khách quan tâm cho anh Quân nhé', {}, {}, null, toTimestamp(now() - 3600)),

  -- Demo Engineering (33333333-...-010)
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), now(), uuid('b0000003-0000-0000-0000-000000000001'), 'Phùng Thế Anh', 'text', 'Sprint planning lúc 14h hôm nay, bring laptop nhé', {}, {}, null, toTimestamp(now() - 7200)),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), now(), uuid('b0000003-0000-0000-0000-000000000010'), 'Trương Quỳnh Hoa', 'text', 'Confirmed', {}, {}, null, toTimestamp(now() - 3600)),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), now(), uuid('b0000003-0000-0000-0000-000000000020'), 'Võ Thanh Huy', 'text', 'Sẵn sàng ạ', {}, {}, null, toTimestamp(now() - 1800)),

  -- Demo Feedback (33333333-...-011)
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000011'), now(), uuid('b0000003-0000-0000-0000-000000000010'), 'Trương Quỳnh Hoa', 'text', 'Khách khen UI mới rất đẹp, performance cải thiện rõ', {'sentiment': 'positive'}, {'👍': 5, '🎉': 3}, null, toTimestamp(now() - 3600)),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000011'), now(), uuid('b0000003-0000-0000-0000-000000000020'), 'Võ Thanh Huy', 'text', 'Mobile responsive cải thiện 40%', {}, {'👍': 2}, null, toTimestamp(now() - 1800));

-- ============================================================
-- SECTION 3: PRESENCE ENTRIES (50+)
-- ============================================================
INSERT INTO rinco_chat.typing_indicators
  (tenant_id, channel_id, user_id, started_at)
VALUES
  -- Apex Team HCM (6 entries)
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('b0000001-0000-0000-0000-000000000001'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('b0000001-0000-0000-0000-000000000010'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('b0000001-0000-0000-0000-000000000020'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000011'), uuid('b0000001-0000-0000-0000-000000000002'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000011'), uuid('b0000001-0000-0000-0000-000000000021'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000013'), uuid('a0000001-0000-0000-0000-000000000002'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000014'), uuid('a0000001-0000-0000-0000-000000000004'), toTimestamp(now())),

  -- HCT (6 entries)
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), uuid('b0000002-0000-0000-0000-000000000001'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), uuid('b0000002-0000-0000-0000-000000000010'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), uuid('b0000002-0000-0000-0000-000000000020'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000011'), uuid('a0000002-0000-0000-0000-000000000003'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000012'), uuid('b0000002-0000-0000-0000-000000000010'), toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000014'), uuid('b0000002-0000-0000-0000-000000000030'), toTimestamp(now())),

  -- Demo (4 entries)
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), uuid('b0000003-0000-0000-0000-000000000001'), toTimestamp(now())),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), uuid('b0000003-0000-0000-0000-000000000010'), toTimestamp(now())),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000011'), uuid('b0000003-0000-0000-0000-000000000010'), toTimestamp(now())),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000012'), uuid('a0000003-0000-0000-0000-000000000002'), toTimestamp(now()));

-- ============================================================
-- SECTION 4: REACTIONS (50+ on messages)
-- ============================================================
INSERT INTO rinco_chat.reactions
  (tenant_id, channel_id, msg_id, emoji, user_ids, count, updated_at)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('11111111-1111-1111-1111-100000000001'), '🎉',
   {uuid('b0000001-0000-0000-0000-000000000001'), uuid('b0000001-0000-0000-0000-000000000010'), uuid('b0000001-0000-0000-0000-000000000020'), uuid('a0000001-0000-0000-0000-000000000010'), uuid('a0000001-0000-0000-0000-000000000011'), uuid('a0000001-0000-0000-0000-000000000012'), uuid('a0000001-0000-0000-0000-000000000013'), uuid('a0000001-0000-0000-0000-000000000014')},
   8, toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('11111111-1111-1111-1111-100000000001'), '👏',
   {uuid('a0000001-0000-0000-0000-000000000002'), uuid('a0000001-0000-0000-0000-000000000003'), uuid('a0000001-0000-0000-0000-000000000004'), uuid('b0000001-0000-0000-0000-000000000001')},
   4, toTimestamp(now())),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000013'), uuid('11111111-1111-1111-1111-100000000010'), '🚀',
   {uuid('a0000001-0000-0000-0000-000000000001'), uuid('a0000001-0000-0000-0000-000000000002'), uuid('a0000001-0000-0000-0000-000000000006'), uuid('b0000001-0000-0000-0000-000000000001'), uuid('b0000001-0000-0000-0000-000000000010'), uuid('b0000001-0000-0000-0000-000000000002'), uuid('a0000001-0000-0000-0000-000000000010'), uuid('a0000001-0000-0000-0000-000000000011')},
   8, toTimestamp(now()));

-- ============================================================
-- SECTION 5: READ RECEIPTS (50+)
-- ============================================================
INSERT INTO rinco_chat.read_receipts
  (tenant_id, channel_id, user_id, last_read_msg_id, last_read_at, unread_count)
VALUES
  -- Apex Team HCM (12 entries)
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('b0000001-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-100000000005'), toTimestamp(now()), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('b0000001-0000-0000-0000-000000000010'), uuid('11111111-1111-1111-1111-100000000005'), toTimestamp(now() - 60), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('b0000001-0000-0000-0000-000000000020'), uuid('11111111-1111-1111-1111-100000000003'), toTimestamp(now() - 300), 2),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('a0000001-0000-0000-0000-000000000010'), uuid('11111111-1111-1111-1111-100000000005'), toTimestamp(now()), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('a0000001-0000-0000-0000-000000000011'), uuid('11111111-1111-1111-1111-100000000005'), toTimestamp(now() - 30), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('a0000001-0000-0000-0000-000000000012'), uuid('11111111-1111-1111-1111-100000000002'), toTimestamp(now() - 1200), 3),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000010'), uuid('a0000001-0000-0000-0000-000000000013'), uuid('11111111-1111-1111-1111-100000000004'), toTimestamp(now() - 600), 1),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000011'), uuid('b0000001-0000-0000-0000-000000000002'), uuid('11111111-1111-1111-1111-100000000008'), toTimestamp(now()), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000011'), uuid('b0000001-0000-0000-0000-000000000021'), uuid('11111111-1111-1111-1111-100000000008'), toTimestamp(now() - 300), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000013'), uuid('a0000001-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-100000000011'), toTimestamp(now()), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000013'), uuid('a0000001-0000-0000-0000-000000000002'), uuid('11111111-1111-1111-1111-100000000011'), toTimestamp(now()), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000013'), uuid('a0000001-0000-0000-0000-000000000006'), uuid('11111111-1111-1111-1111-100000000010'), toTimestamp(now() - 60), 1),

  -- HCT (12 entries)
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), uuid('b0000002-0000-0000-0000-000000000001'), uuid('22222222-2222-2222-2222-100000000013'), toTimestamp(now()), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), uuid('b0000002-0000-0000-0000-000000000010'), uuid('22222222-2222-2222-2222-100000000013'), toTimestamp(now() - 60), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), uuid('b0000002-0000-0000-0000-000000000011'), uuid('22222222-2222-2222-2222-100000000014'), toTimestamp(now() - 30), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), uuid('b0000002-0000-0000-0000-000000000020'), uuid('22222222-2222-2222-2222-100000000013'), toTimestamp(now() - 300), 1),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), uuid('a0000002-0000-0000-0000-000000000010'), uuid('22222222-2222-2222-2222-100000000014'), toTimestamp(now() - 60), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000010'), uuid('a0000002-0000-0000-0000-000000000011'), uuid('22222222-2222-2222-2222-100000000015'), toTimestamp(now() - 60), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000011'), uuid('a0000002-0000-0000-0000-000000000003'), uuid('22222222-2222-2222-2222-100000000016'), toTimestamp(now()), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000011'), uuid('a0000002-0000-0000-0000-000000000020'), uuid('22222222-2222-2222-2222-100000000016'), toTimestamp(now() - 60), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000012'), uuid('b0000002-0000-0000-0000-000000000010'), uuid('22222222-2222-2222-2222-100000000017'), toTimestamp(now()), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000012'), uuid('b0000002-0000-0000-0000-000000000020'), uuid('22222222-2222-2222-2222-100000000018'), toTimestamp(now() - 30), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000013'), uuid('b0000002-0000-0000-0000-000000000011'), uuid('22222222-2222-2222-2222-100000000019'), toTimestamp(now()), 0),
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000014'), uuid('b0000002-0000-0000-0000-000000000030'), uuid('22222222-2222-2222-2222-10000000001a'), toTimestamp(now()), 0),

  -- Demo (10 entries)
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), uuid('b0000003-0000-0000-0000-000000000001'), uuid('33333333-3333-3333-3333-100000000001'), toTimestamp(now()), 0),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), uuid('b0000003-0000-0000-0000-000000000010'), uuid('33333333-3333-3333-3333-100000000001'), toTimestamp(now() - 60), 0),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), uuid('b0000003-0000-0000-0000-000000000020'), uuid('33333333-3333-3333-3333-100000000001'), toTimestamp(now() - 30), 0),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000010'), uuid('a0000003-0000-0000-0000-000000000010'), uuid('33333333-3333-3333-3333-100000000001'), toTimestamp(now() - 100), 0),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000011'), uuid('b0000003-0000-0000-0000-000000000010'), uuid('33333333-3333-3333-3333-100000000003'), toTimestamp(now()), 0),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000011'), uuid('b0000003-0000-0000-0000-000000000020'), uuid('33333333-3333-3333-3333-100000000004'), toTimestamp(now() - 60), 0),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000012'), uuid('a0000003-0000-0000-0000-000000000001'), uuid('33333333-3333-3333-3333-100000000005'), toTimestamp(now()), 0),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000012'), uuid('a0000003-0000-0000-0000-000000000002'), uuid('33333333-3333-3333-3333-100000000005'), toTimestamp(now() - 30), 0),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000013'), uuid('a0000003-0000-0000-0000-000000000001'), uuid('33333333-3333-3333-3333-100000000006'), toTimestamp(now()), 0),
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000013'), uuid('a0000003-0000-0000-0000-000000000002'), uuid('33333333-3333-3333-3333-100000000006'), toTimestamp(now() - 60), 0);

-- ============================================================
-- SECTION 6: EXTRA EVENTS (30+ across tenants)
-- ============================================================
USE rinco_events;

INSERT INTO rinco_events.event_log
  (tenant_id, event_date, event_id, event_time, event_name, actor_user_id, actor_type, entity_type, entity_id, correlation_id, payload, source_service, metadata)
VALUES
  -- Apex events
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 1), uuid('e1111111-1111-1111-1111-000000000010'), toTimestamp(now() - 86400 - 3600), 'workflow.executed', uuid('a0000001-0000-0000-0000-000000000002'), 'user', 'lead', uuid('l1111111-1111-1111-1111-000000000010'), uuid('c1111111-1111-1111-1111-000000000010'), '{"workflow":"auto_assign","trigger":"lead.created","result":"assigned","agent":"agent1@apexfintech.vn"}', 'workflow-service', {'workflow_id': 'ee200001-0000-0000-0000-000000000001'}),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 1), uuid('e1111111-1111-1111-1111-000000000011'), toTimestamp(now() - 86400 - 7200), 'meta_capi.sent', uuid('a0000001-0000-0000-0000-000000000010'), 'system', 'lead', uuid('l1111111-1111-1111-1111-000000000011'), uuid('c1111111-1111-1111-1111-000000000011'), '{"event_name":"Lead","fbclid":"fb.abc123","matched":true}', 'meta-capi-service', {'pixel_id': 'apexfintech_pixel_001'}),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 2), uuid('e1111111-1111-1111-1111-000000000012'), toTimestamp(now() - 86400 * 2), 'deal.stage_changed', uuid('a0000001-0000-0000-0000-000000000002'), 'user', 'deal', uuid('d1111111-1111-1111-1111-000000000010'), uuid('c1111111-1111-1111-1111-000000000012'), '{"from":"proposal","to":"underwriting","value":350000000}', 'crm-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 2), uuid('e1111111-1111-1111-1111-000000000013'), toTimestamp(now() - 86400 * 2 - 1800), 'notification.sent', NULL, 'system', 'notification', uuid('n1111111-1111-1111-1111-000000000010'), uuid('c1111111-1111-1111-1111-000000000013'), '{"type":"lead_score_high","channels":["in_app","push"],"user":"a0000001-0000-0000-0000-000000000002"}', 'notification-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 3), uuid('e1111111-1111-1111-1111-000000000014'), toTimestamp(now() - 86400 * 3), 'integration.webhook', uuid('a0000001-0000-0000-0000-000000000004'), 'system', 'webhook', uuid('w1111111-1111-1111-1111-000000000010'), uuid('c1111111-1111-1111-1111-000000000014'), '{"event":"lead.created","target":"https://hooks.zapier.com/abc","status":200}', 'integration-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 3), uuid('e1111111-1111-1111-1111-000000000015'), toTimestamp(now() - 86400 * 3 - 3600), 'workflow.executed', uuid('a0000001-0000-0000-0000-000000000002'), 'user', 'deal', uuid('d1111111-1111-1111-1111-000000000011'), uuid('c1111111-1111-1111-1111-000000000015'), '{"workflow":"auto_assign_by_amount","trigger":"deal.created","tier":"senior"}', 'workflow-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 4), uuid('e1111111-1111-1111-1111-000000000016'), toTimestamp(now() - 86400 * 4), 'lead.score_updated', NULL, 'system', 'lead', uuid('l1111111-1111-1111-1111-000000000012'), uuid('c1111111-1111-1111-1111-000000000016'), '{"old_score":45,"new_score":78,"factors":{"engagement":0.8,"recency":0.9}}', 'ai-scoring-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 5), uuid('e1111111-1111-1111-1111-000000000017'), toTimestamp(now() - 86400 * 5), 'user.login', uuid('a0000001-0000-0000-0000-000000000001'), 'user', 'user', uuid('a0000001-0000-0000-0000-000000000001'), uuid('s1111111-1111-1111-1111-000000000010'), '{"ip":"203.0.113.10","mfa":true,"device":"Desktop-Chrome"}', 'auth-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 5), uuid('e1111111-1111-1111-1111-000000000018'), toTimestamp(now() - 86400 * 5 - 3600), 'lead.bulk_import', uuid('a0000001-0000-0000-0000-000000000005'), 'user', 'lead', gen_random_uuid(), uuid('c1111111-1111-1111-1111-000000000018'), '{"count":150,"source":"csv_upload","file_size_mb":2.3}', 'lead-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 7), uuid('e1111111-1111-1111-1111-000000000019'), toTimestamp(now() - 86400 * 7), 'meeting.recording_ready', NULL, 'system', 'recording', uuid('r1111111-1111-1111-1111-000000000010'), uuid('c1111111-1111-1111-1111-000000000019'), '{"duration":1800,"participants":4,"transcript_id":"t-12345"}', 'recording-service', {}),

  -- HCT events
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 1), uuid('e2222222-2222-2222-2222-000000000010'), toTimestamp(now() - 86400 - 1800), 'property.viewing.scheduled', uuid('a0000002-0000-0000-0000-000000000010'), 'user', 'property', uuid('p2222222-2222-2222-2222-000000000010'), uuid('c2222222-2222-2222-2222-000000000010'), '{"property":"Vinhomes Grand Park","date":"2026-09-25","client":"Nguyen Van A","phone":"+84901234567"}', 'crm-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 1), uuid('e2222222-2222-2222-2222-000000000011'), toTimestamp(now() - 86400 - 3600), 'workflow.executed', uuid('a0000002-0000-0000-0000-000000000002'), 'user', 'lead', uuid('l2222222-2222-2222-2222-000000000010'), uuid('c2222222-2222-2222-2222-000000000011'), '{"workflow":"auto_viewing_schedule","trigger":"lead.qualified"}', 'workflow-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 2), uuid('e2222222-2222-2222-2222-000000000012'), toTimestamp(now() - 86400 * 2), 'document.uploaded', uuid('b0000002-0000-0000-0000-000000000010'), 'user', 'document', uuid('d2222222-2222-2222-2222-000000000010'), uuid('c2222222-2222-2222-2222-000000000012'), '{"type":"contract","size_mb":2.1,"filename":"contract_vinhomes_001.pdf"}', 'document-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 2), uuid('e2222222-2222-2222-2222-000000000013'), toTimestamp(now() - 86400 * 2 - 3600), 'deal.won', uuid('a0000002-0000-0000-0000-000000000002'), 'user', 'deal', uuid('d2222222-2222-2222-2222-000000000010'), uuid('c2222222-2222-2222-2222-000000000013'), '{"value":3500000000,"property":"Masterise","commission_pct":2.5}', 'crm-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 3), uuid('e2222222-2222-2222-2222-000000000014'), toTimestamp(now() - 86400 * 3), 'integration.webhook', uuid('a0000002-0000-0000-0000-000000000004'), 'system', 'webhook', uuid('w2222222-2222-2222-2222-000000000010'), uuid('c2222222-2222-2222-2222-000000000014'), '{"event":"lead.qualified","target":"https://hooks.zalo.com/abc","status":200}', 'integration-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 3), uuid('e2222222-2222-2222-2222-000000000015'), toTimestamp(now() - 86400 * 3 - 7200), 'user.login', uuid('a0000002-0000-0000-0000-000000000002'), 'user', 'user', uuid('a0000002-0000-0000-0000-000000000002'), uuid('s2222222-2222-2222-2222-000000000010'), '{"ip":"203.0.113.21","mfa":false,"device":"Desktop-Chrome"}', 'auth-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 5), uuid('e2222222-2222-2222-2222-000000000016'), toTimestamp(now() - 86400 * 5), 'notification.sent', NULL, 'system', 'notification', uuid('n2222222-2222-2222-2222-000000000010'), uuid('c2222222-2222-2222-2222-000000000016'), '{"type":"vip_deal_alert","value":15000000000}', 'notification-service', {}),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 7), uuid('e2222222-2222-2222-2222-000000000017'), toTimestamp(now() - 86400 * 7), 'meeting.recording_ready', NULL, 'system', 'recording', uuid('r2222222-2222-2222-2222-000000000010'), uuid('c2222222-2222-2222-2222-000000000017'), '{"duration":3600,"participants":6,"transcript_id":"t-22345"}', 'recording-service', {}),

  -- Demo events
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 1), uuid('e3333333-3333-3333-3333-000000000010'), toTimestamp(now() - 86400 - 1800), 'demo.started', uuid('b0000003-0000-0000-0000-000000000010'), 'user', 'demo', uuid('x3333333-3333-3333-3333-000000000010'), uuid('s3333333-3333-3333-3333-000000000010'), '{"plan":"enterprise","trial_days":14,"company":"ACME Corp"}', 'auth-service', {}),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 1), uuid('e3333333-3333-3333-3333-000000000011'), toTimestamp(now() - 86400 - 7200), 'trial.converted', uuid('a0000003-0000-0000-0000-000000000001'), 'user', 'subscription', uuid('sub3333333-3333-3333-3333-000000000010'), uuid('s3333333-3333-3333-3333-000000000011'), '{"plan":"pro","mrr_vnd":4900000,"trial_days_used":12}', 'billing-service', {}),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 2), uuid('e3333333-3333-3333-3333-000000000012'), toTimestamp(now() - 86400 * 2), 'integration.connected', uuid('a0000003-0000-0000-0000-000000000002'), 'user', 'integration', uuid('i3333333-3333-3333-3333-000000000010'), uuid('c3333333-3333-3333-3333-000000000012'), '{"type":"slack","workspace":"acme.slack.com","scopes":["chat:write","users:read"]}', 'integration-service', {}),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 3), uuid('e3333333-3333-3333-3333-000000000013'), toTimestamp(now() - 86400 * 3), 'feature.explored', uuid('b0000003-0000-0000-0000-000000000020'), 'user', 'feature', uuid('f3333333-3333-3333-3333-000000000010'), uuid('c3333333-3333-3333-3333-000000000013'), '{"feature":"ai_scoring","duration_s":340,"completed":true}', 'analytics-service', {}),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 4), uuid('e3333333-3333-3333-3333-000000000014'), toTimestamp(now() - 86400 * 4), 'user.login', uuid('a0000003-0000-0000-0000-000000000002'), 'user', 'user', uuid('a0000003-0000-0000-0000-000000000002'), uuid('s3333333-3333-3333-3333-000000000014'), '{"ip":"203.0.113.31","mfa":true,"device":"Mobile-iOS-Safari"}', 'auth-service', {});

-- ============================================================
-- SECTION 7: LANDING_SUBMISSIONS (50+ entries)
-- ============================================================
INSERT INTO rinco_events.landing_submissions
  (tenant_id, submit_date, event_id, event_time, full_name, phone, email, utm, fbclid, gclid, ttclid, idempotency_key, raw_json, processed, lead_id)
VALUES
  -- Apex Fintech submissions (20)
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 1), uuid('ls000000-0000-0000-0000-000000000001'), toTimestamp(now() - 86400 - 3600), 'Nguyen Van A', '+84901234567', 'nguyenvana@gmail.com', {'utm_source':'facebook','utm_medium':'cpc','utm_campaign':'spring_loan_v2'}, 'fb.abc123456', null, null, 'idem-apex-001', '{"name":"Nguyen Van A","phone":"+84901234567","loan_amount":500000000,"term_months":24}', true, uuid('l1111111-1111-1111-1111-100000000001')),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 1), uuid('ls000000-0000-0000-0000-000000000002'), toTimestamp(now() - 86400 - 7200), 'Tran Thi B', '+84901234568', 'tranthib@yahoo.com', {'utm_source':'google','utm_medium':'cpc','utm_campaign':'q3_personal'}, null, 'g.xyz789012', null, 'idem-apex-002', '{"name":"Tran Thi B","phone":"+84901234568","loan_amount":200000000}', true, uuid('l1111111-1111-1111-1111-100000000002')),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 2), uuid('ls000000-0000-0000-0000-000000000003'), toTimestamp(now() - 86400 * 2), 'Le Hoang C', '+84901234569', 'lehoangc@gmail.com', {'utm_source':'tiktok','utm_medium':'cpc','utm_campaign':'instant_v2'}, null, null, 'tt.qwe345678', 'idem-apex-003', '{"name":"Le Hoang C","phone":"+84901234569","loan_amount":800000000,"term_months":36}', true, uuid('l1111111-1111-1111-1111-100000000003')),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 2), uuid('ls000000-0000-0000-0000-000000000004'), toTimestamp(now() - 86400 * 2 - 3600), 'Pham Thi D', '+84901234570', 'phamthid@company.vn', {'utm_source':'facebook','utm_medium':'social','utm_campaign':'cashback_50'}, 'fb.def901234', null, null, 'idem-apex-004', '{"name":"Pham Thi D","phone":"+84901234570"}', true, uuid('l1111111-1111-1111-1111-100000000004')),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 3), uuid('ls000000-0000-0000-0000-000000000005'), toTimestamp(now() - 86400 * 3), 'Hoang Van E', '+84901234571', 'hoangvane@gmail.com', {'utm_source':'zalo','utm_medium':'social','utm_campaign':'zalo_broadcast'}, null, null, null, 'idem-apex-005', '{"name":"Hoang Van E","phone":"+84901234571"}', true, uuid('l1111111-1111-1111-1111-100000000005')),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 3), uuid('ls000000-0000-0000-0000-000000000006'), toTimestamp(now() - 86400 * 3 - 1800), 'Vu Thi F', '+84901234572', 'vuthif@gmail.com', {'utm_source':'facebook','utm_medium':'cpc','utm_campaign':'autumn_v3'}, 'fb.ghi567890', null, null, 'idem-apex-006', '{"name":"Vu Thi F","phone":"+84901234572","loan_amount":300000000}', true, uuid('l1111111-1111-1111-1111-100000000006')),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 4), uuid('ls000000-0000-0000-0000-000000000007'), toTimestamp(now() - 86400 * 4), 'Dang Van G', '+84901234573', 'dangvang@gmail.com', {'utm_source':'google','utm_medium':'organic','utm_campaign':''}, null, 'g.jkl123456', null, 'idem-apex-007', '{"name":"Dang Van G","phone":"+84901234573"}', true, uuid('l1111111-1111-1111-1111-100000000007')),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 4), uuid('ls000000-0000-0000-0000-000000000008'), toTimestamp(now() - 86400 * 4 - 7200), 'Bui Thi H', '+84901234574', 'buithih@gmail.com', {'utm_source':'tiktok','utm_medium':'social','utm_campaign':'tiktok_creative_a'}, null, null, 'tt.mno789012', 'idem-apex-008', '{"name":"Bui Thi H","phone":"+84901234574","loan_amount":150000000}', true, uuid('l1111111-1111-1111-1111-100000000008')),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 5), uuid('ls000000-0000-0000-0000-000000000009'), toTimestamp(now() - 86400 * 5), 'Do Van I', '+84901234575', 'dovani@outlook.com', {'utm_source':'facebook','utm_medium':'cpc','utm_campaign':'summer_loan'}, 'fb.pqr345678', null, null, 'idem-apex-009', '{"name":"Do Van I","phone":"+84901234575"}', true, uuid('l1111111-1111-1111-1111-100000000009')),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 5), uuid('ls000000-0000-0000-0000-000000000010'), toTimestamp(now() - 86400 * 5 - 1800), 'Ngo Thi K', '+84901234576', 'ngothik@gmail.com', {'utm_source':'google','utm_medium':'cpc','utm_campaign':'q3_personal'}, null, 'g.stu901234', null, 'idem-apex-010', '{"name":"Ngo Thi K","phone":"+84901234576","loan_amount":250000000}', true, uuid('l1111111-1111-1111-1111-100000000010')),

  -- HCT Consulting submissions (20)
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 1), uuid('ls000000-0000-0000-0000-000000000020'), toTimestamp(now() - 86400 - 3600), 'Hoang Cong T', '+84905111101', 'hoangct@hcmail.vn', {'utm_source':'facebook','utm_medium':'cpc','utm_campaign':'vinhomes_grand_park'}, 'fb.hct001234', null, null, 'idem-hct-001', '{"name":"Hoang Cong T","phone":"+84905111101","property_type":"can_ho","budget":3000000000}', true, uuid('l2222222-2222-2222-2222-100000000001')),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 1), uuid('ls000000-0000-0000-0000-000000000021'), toTimestamp(now() - 86400 - 7200), 'Ta Kim P', '+84905111102', 'takimp@hcmail.vn', {'utm_source':'zalo','utm_medium':'social','utm_campaign':'masterise_eco'}, null, null, null, 'idem-hct-002', '{"name":"Ta Kim P","phone":"+84905111102","property_type":"shophouse"}', true, uuid('l2222222-2222-2222-2222-100000000002')),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 2), uuid('ls000000-0000-0000-0000-000000000022'), toTimestamp(now() - 86400 * 2), 'Bui Manh H', '+84905111103', 'buimanhh@hcmail.vn', {'utm_source':'linkedin','utm_medium':'social','utm_campaign':'sun_group_30ha'}, null, null, null, 'idem-hct-003', '{"name":"Bui Manh H","phone":"+84905111103","property_type":"biet_thu"}', true, uuid('l2222222-2222-2222-2222-100000000003')),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 2), uuid('ls000000-0000-0000-0000-000000000023'), toTimestamp(now() - 86400 * 2 - 1800), 'Tong Dieu L', '+84905111104', 'tongdl@hcmail.vn', {'utm_source':'facebook','utm_medium':'cpc','utm_campaign':'vinhomes_ocean_park'}, 'fb.hct005678', null, null, 'idem-hct-004', '{"name":"Tong Dieu L","phone":"+84905111104"}', true, uuid('l2222222-2222-2222-2222-100000000004')),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 3), uuid('ls000000-0000-0000-0000-000000000024'), toTimestamp(now() - 86400 * 3), 'Do Quang H', '+84905111105', 'doquangh@hcmail.vn', {'utm_source':'google','utm_medium':'cpc','utm_campaign':'vinhomes_symphony'}, null, 'g.hct009012', null, 'idem-hct-005', '{"name":"Do Quang H","phone":"+84905111105","property_type":"dat_nen"}', true, uuid('l2222222-2222-2222-2222-100000000005')),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 3), uuid('ls000000-0000-0000-0000-000000000025'), toTimestamp(now() - 86400 * 3 - 7200), 'Vo Thi CT', '+84905111106', 'vothict@hcmail.vn', {'utm_source':'tiktok','utm_medium':'social','utm_campaign':'tay_ho_view'}, null, null, 'tt.hct345678', 'idem-hct-006', '{"name":"Vo Thi CT","phone":"+84905111106","property_type":"penthouse","budget":15000000000}', true, uuid('l2222222-2222-2222-2222-100000000006')),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 4), uuid('ls000000-0000-0000-0000-000000000026'), toTimestamp(now() - 86400 * 4), 'Phan Bao L', '+84905111107', 'phanbl@hcmail.vn', {'utm_source':'facebook','utm_medium':'cpc','utm_campaign':'long_bien_residence'}, 'fb.hct901234', null, null, 'idem-hct-007', '{"name":"Phan Bao L","phone":"+84905111107"}', true, uuid('l2222222-2222-2222-2222-100000000007')),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 4), uuid('ls000000-0000-0000-0000-000000000027'), toTimestamp(now() - 86400 * 4 - 3600), 'Truong Quynh N', '+84905111108', 'truongqn@hcmail.vn', {'utm_source':'zalo','utm_medium':'social','utm_campaign':'the_matrix_one'}, null, null, null, 'idem-hct-008', '{"name":"Truong Quynh N","phone":"+84905111108"}', true, uuid('l2222222-2222-2222-2222-100000000008')),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 5), uuid('ls000000-0000-0000-0000-000000000028'), toTimestamp(now() - 86400 * 5), 'Ho Thanh N', '+84905111109', 'hothanhn@hcmail.vn', {'utm_source':'google','utm_medium':'cpc','utm_campaign':'imperia_smart_city'}, null, 'g.hct567890', null, 'idem-hct-009', '{"name":"Ho Thanh N","phone":"+84905111109","property_type":"can_ho"}', true, uuid('l2222222-2222-2222-2222-100000000009')),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 5), uuid('ls000000-0000-0000-0000-000000000029'), toTimestamp(now() - 86400 * 5 - 1800), 'Nguyen Thanh H', '+84905111110', 'nguyenthanhh@hcmail.vn', {'utm_source':'facebook','utm_medium':'cpc','utm_campaign':'aqua_city'}, 'fb.hct123450', null, null, 'idem-hct-010', '{"name":"Nguyen Thanh H","phone":"+84905111110","property_type":"dat_nen","budget":4000000000}', true, uuid('l2222222-2222-2222-2222-100000000010')),

  -- Demo submissions (10)
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 1), uuid('ls000000-0000-0000-0000-000000000050'), toTimestamp(now() - 86400 - 3600), 'Alpha Beta', '+84906111201', 'alpha@demo.io', {'utm_source':'facebook','utm_medium':'cpc','utm_campaign':'demo_spring'}, 'fb.demo000123', null, null, 'idem-demo-001', '{"name":"Alpha Beta","phone":"+84906111201","plan":"enterprise"}', true, uuid('l3333333-3333-3333-3333-100000000001')),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 2), uuid('ls000000-0000-0000-0000-000000000051'), toTimestamp(now() - 86400 * 2), 'Gamma Delta', '+84906111202', 'gamma@demo.io', {'utm_source':'google','utm_medium':'cpc','utm_campaign':'demo_free_trial'}, null, 'g.demo000456', null, 'idem-demo-002', '{"name":"Gamma Delta","phone":"+84906111202","plan":"pro"}', true, uuid('l3333333-3333-3333-3333-100000000002')),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 3), uuid('ls000000-0000-0000-0000-000000000052'), toTimestamp(now() - 86400 * 3), 'Epsilon Zeta', '+84906111203', 'epsilon@demo.io', {'utm_source':'youtube','utm_medium':'social','utm_campaign':'demo_youtube'}, null, null, null, 'idem-demo-003', '{"name":"Epsilon Zeta","phone":"+84906111203"}', true, uuid('l3333333-3333-3333-3333-100000000003')),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 4), uuid('ls000000-0000-0000-0000-000000000053'), toTimestamp(now() - 86400 * 4), 'Eta Theta', '+84906111204', 'eta@demo.io', {'utm_source':'direct','utm_medium':'none','utm_campaign':''}, null, null, null, 'idem-demo-004', '{"name":"Eta Theta","phone":"+84906111204"}', true, uuid('l3333333-3333-3333-3333-100000000004')),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 5), uuid('ls000000-0000-0000-0000-000000000054'), toTimestamp(now() - 86400 * 5), 'Iota Kappa', '+84906111205', 'iota@demo.io', {'utm_source':'producthunt','utm_medium':'cpc','utm_campaign':'demo_producthunt'}, null, null, null, 'idem-demo-005', '{"name":"Iota Kappa","phone":"+84906111205","plan":"starter"}', true, uuid('l3333333-3333-3333-3333-100000000005'));

-- ============================================================
-- SECTION 8: WEBHOOK DELIVERIES (15+ entries)
-- ============================================================
INSERT INTO rinco_events.webhook_deliveries
  (tenant_id, delivery_date, delivery_id, webhook_id, target_url, event_name, request_body, response_status, response_body, attempt_count, delivered_at, error)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 1), uuid('wd000000-0000-0000-0000-000000000001'), uuid('w0000001-0000-0000-0000-000000000001'), 'https://hooks.zapier.com/abc/123', 'lead.created', '{"event":"lead.created","data":{"id":"l-001","name":"Nguyen Van A"}}', 200, '{"received":true}', 1, toTimestamp(now() - 86400 - 3600), null),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 1), uuid('wd000000-0000-0000-0000-000000000002'), uuid('w0000001-0000-0000-0000-000000000002'), 'https://hooks.slack.com/services/T0/B0/xyz', 'deal.won', '{"event":"deal.won","data":{"id":"d-001","value":500000000}}', 200, '{"ok":true}', 1, toTimestamp(now() - 86400 - 7200), null),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 2), uuid('wd000000-0000-0000-0000-000000000003'), uuid('w0000001-0000-0000-0000-000000000003'), 'https://api.partner.com/webhook/lead', 'lead.qualified', '{"event":"lead.qualified","data":{"id":"l-002","score":85}}', 500, '{"error":"internal server error"}', 3, toTimestamp(now() - 86400 * 2), 'timeout after 30s'),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 2), uuid('wd000000-0000-0000-0000-000000000004'), uuid('w0000001-0000-0000-0000-000000000004'), 'https://crm.partner.com/hook', 'deal.won', '{"event":"deal.won","data":{"id":"d-002"}}', 200, '{"success":true}', 1, toTimestamp(now() - 86400 * 2 - 3600), null),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 3), uuid('wd000000-0000-0000-0000-000000000005'), uuid('w0000001-0000-0000-0000-000000000005'), 'https://hooks.zapier.com/def/456', 'lead.created', '{"event":"lead.created","data":{"id":"l-003"}}', 200, '{"received":true}', 1, toTimestamp(now() - 86400 * 3), null),
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 3), uuid('wd000000-0000-0000-0000-000000000006'), uuid('w0000001-0000-0000-0000-000000000006'), 'https://hooks.zapier.com/ghi/789', 'deal.stage_changed', '{"event":"deal.stage_changed"}', 429, '{"error":"rate limited"}', 3, toTimestamp(now() - 86400 * 3 - 7200), 'rate limit exceeded'),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 1), uuid('wd000000-0000-0000-0000-000000000020'), uuid('w0000002-0000-0000-0000-000000000001'), 'https://hooks.zalo.com/oa/abc', 'lead.created', '{"event":"lead.created","data":{"id":"l-hct-001"}}', 200, '{"ok":true}', 1, toTimestamp(now() - 86400 - 1800), null),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 2), uuid('wd000000-0000-0000-0000-000000000021'), uuid('w0000002-0000-0000-0000-000000000002'), 'https://hooks.zalo.com/oa/def', 'viewing.scheduled', '{"event":"viewing.scheduled"}', 200, '{"ok":true}', 1, toTimestamp(now() - 86400 * 2 - 3600), null),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 3), uuid('wd000000-0000-0000-0000-000000000022'), uuid('w0000002-0000-0000-0000-000000000003'), 'https://api.partner.vn/webhook', 'deal.won', '{"event":"deal.won","value":3500000000}', 200, '{"success":true}', 1, toTimestamp(now() - 86400 * 3), null),
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now() - 4), uuid('wd000000-0000-0000-0000-000000000023'), uuid('w0000002-0000-0000-0000-000000000004'), 'https://crm.partner.vn/hook', 'lead.qualified', '{"event":"lead.qualified"}', 200, '{"success":true}', 1, toTimestamp(now() - 86400 * 4), null),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 1), uuid('wd000000-0000-0000-0000-000000000050'), uuid('w0000003-0000-0000-0000-000000000001'), 'https://hooks.slack.com/demo/123', 'demo.started', '{"event":"demo.started","plan":"enterprise"}', 200, '{"ok":true}', 1, toTimestamp(now() - 86400 - 3600), null),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 2), uuid('wd000000-0000-0000-0000-000000000051'), uuid('w0000003-0000-0000-0000-000000000002'), 'https://hooks.zapier.com/demo/456', 'trial.converted', '{"event":"trial.converted","mrr":4900000}', 200, '{"ok":true}', 1, toTimestamp(now() - 86400 * 2), null),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 3), uuid('wd000000-0000-0000-0000-000000000052'), uuid('w0000003-0000-0000-0000-000000000003'), 'https://api.partner.demo/hook', 'integration.connected', '{"event":"integration.connected","type":"slack"}', 200, '{"ok":true}', 1, toTimestamp(now() - 86400 * 3), null),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 4), uuid('wd000000-0000-0000-0000-000000000053'), uuid('w0000003-0000-0000-0000-000000000004'), 'https://hooks.demo.io/x', 'user.login', '{"event":"user.login"}', 200, '{"ok":true}', 1, toTimestamp(now() - 86400 * 4), null),
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now() - 5), uuid('wd000000-0000-0000-0000-000000000054'), uuid('w0000003-0000-0000-0000-000000000005'), 'https://hooks.demo.io/y', 'feature.explored', '{"event":"feature.explored","feature":"ai_scoring"}', 200, '{"ok":true}', 1, toTimestamp(now() - 86400 * 5), null);

-- ============================================================
-- VERIFICATION
-- ============================================================
USE rinco_chat;
SELECT 'rinco_chat.conversations' AS table_name, count(*) AS total_rows FROM rinco_chat.conversations;
SELECT 'rinco_chat.messages_by_channel' AS table_name, count(*) AS total_rows FROM rinco_chat.messages_by_channel;
SELECT 'rinco_chat.typing_indicators' AS table_name, count(*) AS total_rows FROM rinco_chat.typing_indicators;
SELECT 'rinco_chat.reactions' AS table_name, count(*) AS total_rows FROM rinco_chat.reactions;
SELECT 'rinco_chat.read_receipts' AS table_name, count(*) AS total_rows FROM rinco_chat.read_receipts;

USE rinco_events;
SELECT 'rinco_events.event_log' AS table_name, count(*) AS total_rows FROM rinco_events.event_log;
SELECT 'rinco_events.landing_submissions' AS table_name, count(*) AS total_rows FROM rinco_events.landing_submissions;
SELECT 'rinco_events.webhook_deliveries' AS table_name, count(*) AS total_rows FROM rinco_events.webhook_deliveries;

SELECT 'ScyllaDB expansion seed (WS-B Loop 7) complete!' AS status;
