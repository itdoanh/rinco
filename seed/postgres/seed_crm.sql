-- =============================================================
-- RINCO PostgreSQL Seed Data - Core CRM tables (FULLY CORRECTED)
-- Target: Tenant 1 (Apex Fintech) aaaaaaaa-0000-0000-0000-000000000001
-- =============================================================

-- Lead sources
INSERT INTO public.lead_sources (id, tenant_id, name, medium, is_active) VALUES
  ('31000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Facebook Ads', 'paid', true),
  ('31000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Google Search', 'paid', true),
  ('31000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'Zalo Ads', 'paid', true),
  ('31000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'TikTok Ads', 'paid', true),
  ('31000001-0000-0000-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'Organic Search', 'organic', true),
  ('31000001-0000-0000-0000-000000000006', 'aaaaaaaa-0000-0000-0000-000000000001', 'Direct', 'direct', true),
  ('31000001-0000-0000-0000-000000000007', 'aaaaaaaa-0000-0000-0000-000000000001', 'Referral', 'referral', true),
  ('31000001-0000-0000-0000-000000000008', 'aaaaaaaa-0000-0000-0000-000000000001', 'Landing Page', 'landing', true)
ON CONFLICT DO NOTHING;

-- Companies (size must be: startup/small/medium/large/enterprise)
INSERT INTO public.companies (id, tenant_id, name, industry, size, website, phone, email, address, city, country, custom_fields) VALUES
  ('32000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Công Ty TNHH TM&DV Việt Phát', 'Retail', 'medium', 'https://vietphat.vn', '02838123456', 'contact@vietphat.vn', '123 Lê Lợi, Q1', 'Ho Chi Minh', 'Vietnam', '{"tax_id":"0312345678"}'),
  ('32000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Tập Đoàn Bất Động Sản Phú Mỹ', 'Real Estate', 'large', 'https://phumy.vn', '02839998888', 'info@phumy.vn', '456 Nguyễn Trãi, Q5', 'Ho Chi Minh', 'Vietnam', '{"tax_id":"0312345679","license":"GD-12345"}'),
  ('32000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'NHÀ MÁY SẢN XUẤT TÂN THÀNH', 'Manufacturing', 'large', 'https://tanthanh.vn', '02742234567', 'sales@tanthanh.vn', '789 KCN Bình Dương', 'Bình Dương', 'Vietnam', '{"tax_id":"0312345680","iso":"9001:2015"}'),
  ('32000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'Công Ty Cổ Phần Logistics Express', 'Logistics', 'large', 'https://logisticsexpress.vn', '02835556677', 'hotro@logisticsexpress.vn', '101 Đại lộ Thăng Long', 'Ha Noi', 'Vietnam', '{"tax_id":"0312345681","fleet_size":"50 trucks"}'),
  ('32000001-0000-0000-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'TRƯỜNG QUỐC TẾ Á CHÂU', 'Education', 'large', 'https://aci.edu.vn', '02834567890', 'admissions@aci.edu.vn', '222 Điện Biên Phủ, Q3', 'Ho Chi Minh', 'Vietnam', '{"tax_id":"0312345682","students":"2500"}')
ON CONFLICT (id) DO NOTHING;

-- Contacts (source must be: organic/referral/social/ads/direct/other)
INSERT INTO public.contacts (id, tenant_id, company_id, first_name, last_name, email, phone, job_title, department, owner_user_id, status, source, custom_fields) VALUES
  ('33000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000001', 'Nguyễn', 'Văn A', 'nvana@vietphat.vn', '0903123456', 'Giám đốc', 'Executive', 'a0000001-0000-0000-0000-000000000002', 'active', 'referral', '{"linkedin":"linkedin.com/in/nvana"}'),
  ('33000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000001', 'Trần', 'Thị B', 'ttb@vietphat.vn', '0903234567', 'Kế toán trưởng', 'Finance', 'a0000001-0000-0000-0000-000000000002', 'active', 'organic', '{}'),
  ('33000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000002', 'Lê', 'Văn C', 'lvc@phumy.vn', '0903344567', 'Phó giám đốc kinh doanh', 'Sales', 'a0000001-0000-0000-0000-000000000003', 'active', 'social', '{"budget":"5 tỷ/month"}'),
  ('33000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000002', 'Phạm', 'Thị D', 'ptd@phumy.vn', '0903456789', 'Trợ lý giám đốc', 'Executive', 'a0000001-0000-0000-0000-000000000003', 'active', 'organic', '{}'),
  ('33000001-0000-0000-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000003', 'Hoàng', 'Văn E', 'hve@tanthanh.vn', '0903567890', 'Giám đốc sản xuất', 'Operations', 'a0000001-0000-0000-0000-000000000004', 'active', 'social', '{"production_capacity":"10,000 units/month"}'),
  ('33000001-0000-0000-0000-000000000006', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000003', 'Ngô', 'Thị F', 'ntf@tanthanh.vn', '0903678901', 'Quản lý kho vận', 'Logistics', 'a0000001-0000-0000-0000-000000000004', 'active', 'referral', '{}'),
  ('33000001-0000-0000-0000-000000000007', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000004', 'Đặng', 'Văn G', 'dvg@logisticsexpress.vn', '0903789012', 'CEO', 'Executive', 'a0000001-0000-0000-0000-000000000005', 'active', 'social', '{"routes":"50+ routes nationwide"}'),
  ('33000001-0000-0000-0000-000000000008', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000004', 'Bùi', 'Thị H', 'bth@logisticsexpress.vn', '0903890123', 'Trưởng phòng IT', 'IT', 'a0000001-0000-0000-0000-000000000005', 'active', 'organic', '{"fleet_system":"TMS v2.1"}'),
  ('33000001-0000-0000-0000-000000000009', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000005', 'Vũ', 'Văn I', 'vvi@aci.edu.vn', '0903901234', 'Hiệu trưởng', 'Executive', 'a0000001-0000-0000-0000-000000000006', 'active', 'referral', '{"students":"2500","staff":"150"}'),
  ('33000001-0000-0000-0000-000000000010', 'aaaaaaaa-0000-0000-0000-000000000001', '32000001-0000-0000-0000-000000000005', 'Đỗ', 'Thị K', 'dtk@aci.edu.vn', '0903012345', 'Trưởng phòng tài chính', 'Finance', 'a0000001-0000-0000-0000-000000000006', 'active', 'organic', '{}')
ON CONFLICT (id) DO NOTHING;

-- Leads (leads.leads - full column set)
INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, utm_source, utm_medium, utm_campaign, score, quality_score, stage, custom_fields, metadata) VALUES
  ('41000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000010', 'Trần Đình Minh', 'minh.tran@gmail.com', '0911234567', 'ads', 'facebook', 'cpc', 'spring_loan_v1', 85, 'high', 'qualified', '{"company":"Công ty TNHH Minh Thịnh","company_size":"medium","estimated_value":50000000}', '{"notes":"Quan tâm vay mua xe, ngân hàng đang xử lý hồ sơ"}'),
  ('41000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000010', 'Nguyễn Thu Hà', 'hanguyen@yahoo.com', '0912345678', 'organic', 'google', 'cpc', 'q3_personal', 70, 'high', 'contacted', '{"company":"Hà Nguyễn Corp","company_size":"small"}', '{"notes":"Cần gọi lại vào 14h chiều thứ 6"}'),
  ('41000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000011', 'Lê Hoàng Nam', 'nam.le@gmail.com', '0913456789', 'ads', 'zalo', 'cpc', 'cashback_50', 55, 'medium', 'new', '{"company":"Nam Le Trading","company_size":"small","estimated_value":20000000}', '{"notes":"Mới đăng ký, chưa tư vấn"}'),
  ('41000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000011', 'Phạm Thị Lan', 'phamlan.88@gmail.com', '0914567890', 'ads', 'facebook', 'cpc', 'spring_loan_v1', 90, 'high', 'qualified', '{"company":"PTL Fashion","company_size":"small","estimated_value":80000000}', '{"notes":"Khách hàng cũ, muốn vay mở rộng kinh doanh"}'),
  ('41000001-0000-0000-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000012', 'Hoàng Văn Tuấn', 'tuanhoang.pro@gmail.com', '0915678901', 'referral', 'referral', 'referral', 'partner_q3', 95, 'high', 'proposal', '{"company":"Tuấn Hoàng JSC","company_size":"medium","estimated_value":150000000}', '{"notes":"Đang đàm phán hợp đồng, dự kiến ký tuần sau"}'),
  ('41000001-0000-0000-0000-000000000006', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000012', 'Đặng Minh Khoa', 'khoa.dang@microsoft.com', '0916789012', 'organic', 'google', 'organic', 'brand_awareness', 65, 'medium', 'contacted', '{"company":"Khoa Đặng Consulting","company_size":"small","estimated_value":45000000}', '{"notes":"Interest in SME financing package"}'),
  ('41000001-0000-0000-0000-000000000007', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000013', 'Trương Thanh Hà', 'tt.ha@gmail.com', '0917890123', 'ads', 'tiktok', 'cpc', 'instant_v2', 40, 'low', 'new', '{"company":"Hà Thanh Store","company_size":"small","estimated_value":15000000}', '{"notes":"Cold lead, browsing products"}'),
  ('41000001-0000-0000-0000-000000000008', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000013', 'Ngô Quốc Trung', 'nqtrung.vn@gmail.com', '0918901234', 'organic', 'organic', 'landing', 'vay_tieu_dung_q3', 78, 'high', 'qualified', '{"company":"Trung Ngô Tech","company_size":"small","estimated_value":60000000}', '{"notes":"Đang xác minh thu nhập"}'),
  ('41000001-0000-0000-0000-000000000009', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000014', 'Lý Thị Mai', 'lymaifinance@gmail.com', '0919012345', 'ads', 'facebook', 'cpc', 'spring_loan_v1', 72, 'high', 'contacted', '{"company":"Mai Lý Investment","company_size":"small","estimated_value":100000000}', '{"notes":"Đang chờ phê duyệt từ phía ngân hàng"}'),
  ('41000001-0000-0000-0000-000000000010', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000014', 'Vũ Đình Phong', 'vdphong88@gmail.com', '0910123456', 'organic', 'google', 'cpc', 'q3_personal', 100, 'high', 'won', '{"company":"Phong Vũ Logistics","company_size":"medium","estimated_value":200000000}', '{"notes":"Đã ký hợp đồng thành công!"}'),
  ('41000001-0000-0000-0000-000000000011', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000015', 'Trần Thị Phương', 'ttphuong.acb@gmail.com', '0901234567', 'referral', 'referral', 'referral', 'referral_q3', 88, 'high', 'proposal', '{"company":"Phương Trần Group","company_size":"medium","estimated_value":300000000}', '{"notes":"Meeting scheduled for next Monday"}'),
  ('41000001-0000-0000-0000-000000000012', 'aaaaaaaa-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000015', 'Đỗ Văn Hùng', 'dovanhung.z@gmail.com', '0902345678', 'ads', 'zalo', 'cpc', 'cashback_50', 35, 'low', 'new', '{"company":"Hùng Đỗ Trading","company_size":"small","estimated_value":10000000}', '{"notes":"Just signed up, needs follow-up"}')
ON CONFLICT (id) DO NOTHING;

-- Deal stage history (public.lead_stage_history - correct schema)
INSERT INTO public.lead_stage_history (id, tenant_id, lead_id, from_stage, to_stage, changed_by, notes) VALUES
  ('42000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000001', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000010', 'Initial call completed, interested in loan'),
  ('42000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000001', 'contacted', 'qualified', 'a0000001-0000-0000-0000-000000000010', 'Documents verified, income confirmed'),
  ('42000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000004', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000011', 'Phone consultation done'),
  ('42000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000004', 'contacted', 'qualified', 'a0000001-0000-0000-0000-000000000011', 'Referred by existing client, high intent'),
  ('42000001-0000-0000-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000005', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000012', 'Email inquiry'),
  ('42000001-0000-0000-0000-0000-000000000006', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000005', 'contacted', 'qualified', 'a0000001-0000-0000-0000-000000000012', 'Site visit completed'),
  ('42000001-0000-0000-0000-000000000007', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000005', 'qualified', 'proposal', 'a0000001-0000-0000-0000-000000000012', 'Proposal sent, awaiting response'),
  ('42000001-0000-0000-0000-000000000008', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000010', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000014', 'Walk-in customer'),
  ('42000001-0000-0000-0000-000000000009', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000010', 'contacted', 'qualified', 'a0000001-0000-0000-0000-000000000014', 'Documents submitted'),
  ('42000001-0000-0000-0000-000000000010', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000010', 'qualified', 'proposal', 'a0000001-0000-0000-0000-000000000014', 'Custom loan package presented'),
  ('42000001-0000-0000-0000-000000000011', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000010', 'proposal', 'won', 'a0000001-0000-0000-0000-000000000014', 'Contract signed!'),
  ('42000001-0000-0000-0000-000000000012', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000003', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000011', 'Zalo contact made'),
  ('42000001-0000-0000-0000-000000000013', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000003', 'contacted', 'lost', 'a0000001-0000-0000-0000-000000000011', 'Not interested anymore, budget concerns'),
  ('42000001-0000-0000-0000-000000000014', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000007', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000013', 'TikTok ad response'),
  ('42000001-0000-0000-0000-000000000015', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000007', 'contacted', 'lost', 'a0000001-0000-0000-0000-000000000013', 'Chose competitor offering lower rate')
ON CONFLICT (id) DO NOTHING;

-- Deals (public.deals - correct schema: no lead_id, has contact_id, has name not title, stage: prospecting/qualification/proposal/negotiation/won/lost/on_hold)
INSERT INTO public.deals (id, tenant_id, contact_id, owner_user_id, name, value, currency, stage, probability, expected_close_date) VALUES
  ('51000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', '33000001-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000014', 'Vũ Đình Phong - Logistics Loan', 200000000, 'VND', 'won', 100, CURRENT_DATE + 7),
  ('51000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', '33000001-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000012', 'Hoàng Văn Tuấn - SME Package', 150000000, 'VND', 'proposal', 75, CURRENT_DATE + 14),
  ('51000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', '33000001-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000015', 'Trần Thị Phương - Group Finance', 300000000, 'VND', 'proposal', 60, CURRENT_DATE + 21),
  ('51000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', '33000001-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000010', 'Trần Đình Minh - Auto Loan', 50000000, 'VND', 'qualification', 50, CURRENT_DATE + 30),
  ('51000001-0000-0000-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', '33000001-0000-0000-0000-000000000001', 'a0000001-0000-0000-0000-000000000011', 'Phạm Thị Lan - Business Expansion', 80000000, 'VND', 'qualification', 50, CURRENT_DATE + 28)
ON CONFLICT (id) DO NOTHING;

-- Deal stage history (public.deal_stage_history - correct schema: no tenant_id)
INSERT INTO public.deal_stage_history (id, deal_id, from_stage, to_stage, changed_by, notes) VALUES
  ('52000001-0000-0000-0000-000000000001', '51000001-0000-0000-0000-000000000001', 'prospecting', 'qualification', 'a0000001-0000-0000-0000-000000000014', 'Walk-in registration'),
  ('52000001-0000-0000-0000-000000000002', '51000001-0000-0000-0000-000000000001', 'qualification', 'proposal', 'a0000001-0000-0000-0000-000000000014', 'Income verification passed'),
  ('52000001-0000-0000-0000-000000000003', '51000001-0000-0000-0000-000000000001', 'proposal', 'negotiation', 'a0000001-0000-0000-0000-000000000014', 'Custom rate proposed'),
  ('52000001-0000-0000-0000-000000000004', '51000001-0000-0000-0000-000000000001', 'negotiation', 'won', 'a0000001-0000-0000-0000-000000000014', 'Contract signed, disbursement scheduled')
ON CONFLICT (id) DO NOTHING;

-- Notes (public.notes - correct schema: no subject, has body, has author_id)
INSERT INTO public.notes (id, tenant_id, contact_id, deal_id, body, author_id) VALUES
  ('61000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', NULL, NULL, 'Trần Đình Minh là khách hàng tiềm năng cao. Đã xác minh thu nhập qua sao kê ngân hàng. Cần theo dõi tiến độ phê duyệt từ ngân hàng đối tác.', 'a0000001-0000-0000-0000-000000000010'),
  ('61000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', NULL, NULL, 'Gọi điện tư vấn lần 2 cho Hoàng Văn Tuấn. Khách hàng rất quan tâm đến gói SME với lãi suất ưu đãi. Đã gửi proposal qua email. Dự kiến phản hồi trong 48h.', 'a0000001-0000-0000-0000-000000000011'),
  ('61000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', NULL, '51000001-0000-0000-0000-000000000001', 'Hợp đồng vay 200 triệu đã ký thành công với Vũ Đình Phong. Disbursement scheduled for next Monday. CRM updated.', 'a0000001-0000-0000-0000-000000000014'),
  ('61000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', NULL, NULL, 'Facebook Ads spring_loan_v1: 150 clicks, 12 leads, 3 qualified. CTR 8%, CPL 250k VND. Need to optimize landing page for mobile.', 'a0000001-0000-0000-0000-000000000012')
ON CONFLICT (id) DO NOTHING;

-- Tags
INSERT INTO public.tags (id, tenant_id, name, color, usage_count) VALUES
  ('71000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'hot-lead', '#e74c3c', 5),
  ('71000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'follow-up', '#f39c12', 8),
  ('71000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'vip-customer', '#9b59b6', 3),
  ('71000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'sme', '#3498db', 12),
  ('71000001-0000-0000-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'retail', '#2ecc71', 7),
  ('71000001-0000-0000-0000-000000000006', 'aaaaaaaa-0000-0000-0000-000000000001', 'pending-docs', '#95a5a6', 4),
  ('71000001-0000-0000-0000-000000000007', 'aaaaaaaa-0000-0000-0000-000000000001', 'auto-loan', '#1abc9c', 6),
  ('71000001-0000-0000-0000-000000000008', 'aaaaaaaa-0000-0000-0000-000000000001', 'high-value', '#e67e22', 9)
ON CONFLICT (id) DO NOTHING;

-- Activities (public.activities - correct schema)
INSERT INTO public.activities (id, tenant_id, contact_id, deal_id, type, subject, body, status, owner_user_id) VALUES
  ('81000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', NULL, NULL, 'call', 'Outbound Call - Trần Đình Minh', 'Called Trần Đình Minh - discussed loan options', 'completed', 'a0000001-0000-0000-0000-000000000010'),
  ('81000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', NULL, NULL, 'email', 'Follow-up Email - Hoàng Văn Tuấn', 'Sent SME package proposal to Hoàng Văn Tuấn', 'completed', 'a0000001-0000-0000-0000-000000000011'),
  ('81000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', NULL, '51000001-0000-0000-0000-000000000001', 'meeting', 'Contract Signing - Vũ Đình Phong', 'Signed loan contract with Vũ Đình Phong - 200M VND', 'completed', 'a0000001-0000-0000-0000-000000000014'),
  ('81000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', NULL, NULL, 'call', 'Discovery Call - Đỗ Văn Hùng', 'Initial consultation call with Đỗ Văn Hùng', 'pending', 'a0000001-0000-0000-0000-000000000015')
ON CONFLICT (id) DO NOTHING;

-- Custom fields (public.custom_fields - entity_type: contact/company/deal/lead/activity)
INSERT INTO public.custom_fields (id, tenant_id, entity_type, name, field_key, field_type, description, options, is_required) VALUES
  ('91000002-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'lead', 'Mục đích vay', 'loan_purpose', 'select', 'Lựa chọn mục đích vay', '["Mua xe","Mua nhà","Kinh doanh","Tiêu dùng","Khác"]', false),
  ('91000002-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'lead', 'Loại xe quan tâm', 'vehicle_type', 'text', 'Loại xe khách hàng muốn mua', NULL, false),
  ('91000002-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'lead', 'Thu nhập hàng tháng', 'monthly_income', 'number', 'Thu nhập hàng tháng (VND)', NULL, true),
  ('91000002-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'deal', 'Kỳ hạn vay', 'loan_term', 'select', 'Thời hạn vay', '["6 tháng","12 tháng","24 tháng","36 tháng","48 tháng","60 tháng"]', true),
  ('91000002-0000-0000-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'deal', 'Lãi suất đề xuất', 'interest_rate', 'number', 'Lãi suất được đề xuất cho deal', NULL, false)
ON CONFLICT (id) DO NOTHING;

-- Contact tags
INSERT INTO public.contact_tags (contact_id, tag_id) VALUES
  ('33000001-0000-0000-0000-000000000001', '71000001-0000-0000-0000-000000000003'),
  ('33000001-0000-0000-0000-000000000001', '71000001-0000-0000-0000-000000000008'),
  ('33000001-0000-0000-0000-000000000003', '71000001-0000-0000-0000-000000000004'),
  ('33000001-0000-0000-0000-000000000003', '71000001-0000-0000-0000-000000000001'),
  ('33000001-0000-0000-0000-000000000005', '71000001-0000-0000-0000-000000000005'),
  ('33000001-0000-0000-0000-000000000007', '71000001-0000-0000-0000-000000000004'),
  ('33000001-0000-0000-0000-000000000009', '71000001-0000-0000-0000-000000000003')
ON CONFLICT DO NOTHING;

-- =============================================================
-- Verify counts
-- =============================================================
SELECT 'leads' AS table_name, COUNT(*) AS row_count FROM leads.leads WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'contacts', COUNT(*) FROM public.contacts WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'companies', COUNT(*) FROM public.companies WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'deals', COUNT(*) FROM public.deals WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'notes', COUNT(*) FROM public.notes WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'activities', COUNT(*) FROM public.activities WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'tags', COUNT(*) FROM public.tags WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'lead_stage_history', COUNT(*) FROM public.lead_stage_history WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'deal_stage_history', COUNT(*) FROM public.deal_stage_history
UNION ALL SELECT 'contact_tags', COUNT(*) FROM public.contact_tags
UNION ALL SELECT 'custom_fields', COUNT(*) FROM public.custom_fields WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'lead_sources', COUNT(*) FROM public.lead_sources WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001';
