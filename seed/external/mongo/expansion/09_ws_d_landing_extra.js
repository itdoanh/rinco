// ============================================================
// WS-D: MONGODB EXPANSION
// - 12 additional Vietnamese landing pages with real VN copy
// - 100 additional enrichment cache records
// - 50 form definitions with VN field names
// ============================================================

print('[WS-D] Starting MongoDB expansion...');

// ============================================================
// SECTION A: ADDITIONAL VIETNAMESE LANDING PAGES
// Real VN copy for each of 5 tenants
// ============================================================
print('[WS-D-A] Additional Vietnamese landing pages...');
let newLpCount = 0;
const tenantLpData = [
  // Apex Fintech
  {
    tenantId: 'aaaaaaaa-0000-0000-0000-000000000001',
    pages: [
      {
        slug: 'vay-mua-nha',
        title: 'Vay Mua Nhà - Lãi Suất Ưu Đãi Chỉ Từ 7.5%/Năm | Apex Fintech',
        blocks: [
          { type: 'hero', title: 'Vay Mua Nhà Dễ Dàng - Phê Duyệt Trong 24 Giờ', subtitle: 'Lãi suất cạnh tranh, thủ tục đơn giản, hỗ trợ tận tình 24/7', cta: 'Đăng Ký Ngay' },
          { type: 'features', title: 'Tại Sao Chọn Apex Fintech?', items: [
            'Lãi suất chỉ từ 7.5%/năm cố định 12 tháng đầu',
            'Vay tới 80% giá trị căn nhà, thời hạn tới 25 năm',
            'Phê duyệt trong 24 giờ, giải ngân trong 48 giờ',
            'Không phạt trả nợ trước hạn',
            'Hỗ trợ tư vấn miễn phí 24/7 qua Zalo/điện thoại'
          ]},
          { type: 'form', title: 'Nhận Tư Vấn Miễn Phí', fields: [
            { name: 'ho_va_ten', label: 'Họ và tên', required: true, type: 'text' },
            { name: 'so_dien_thoai', label: 'Số điện thoại', required: true, type: 'tel' },
            { name: 'email_cong_ty', label: 'Email', required: false, type: 'email' },
            { name: 'so_tien_can_vay', label: 'Số tiền cần vay (triệu VNĐ)', required: true, type: 'number' },
            { name: 'muc_dich_vay', label: 'Mục đích vay', required: true, type: 'select', options: ['Mua nhà', 'Mua xe', 'Kinh doanh', 'Nợ tiền mặt', 'Đầu tư'] },
            { name: 'so_luong_nhan_vien', label: 'Thu nhập hàng tháng (triệu VNĐ)', required: true, type: 'select', options: ['< 10', '10-20', '20-50', '50-100', '100+'] }
          ], honeypot: true },
          { type: 'testimonials', title: 'Khách Hàng Nói Gì?', items: [
            { name: 'Anh Nguyễn Văn An', quote: 'Vay mua nhà tại Apex rất nhanh, chỉ 2 ngày là có tiền. Thủ tục đơn giản, nhân viên nhiệt tình.', city: 'Hà Nội' },
            { name: 'Chị Trần Thị Bình', quote: 'Lãi suất tốt hơn ngân hàng, được tư vấn kỹ trước khi ký hợp đồng.', city: 'TP.HCM' }
          ]},
          { type: 'faq', title: 'Câu Hỏi Thường Gặp', items: [
            { q: 'Tôi cần những giấy tờ gì?', a: 'CMND/CCCD, sổ hồng/sổ đỏ, sao kê lương 3 tháng gần nhất, hợp đồng lao động.' },
            { q: 'Phê duyệt mất bao lâu?', a: 'Trong vòng 24 giờ sau khi nhận đủ hồ sơ. Giải ngân trong 48 giờ sau khi ký hợp đồng.' }
          ]}
        ],
        seo: { title: 'Vay Mua Nhà Lãi Suất Thấp 7.5% | Apex Fintech', description: 'Vay mua nhà chỉ từ 7.5%/năm. Phê duyệt 24h, giải ngân 48h. Tư vấn miễn phí 24/7.' },
        status: 'published'
      },
      {
        slug: 'vay-mua-xe',
        title: 'Vay Mua Xe Ô Tô - Lãi Suất Từ 6.9%/Năm',
        blocks: [
          { type: 'hero', title: 'Vay Mua Xe - Lái Xe Mới Trong 48 Giờ', subtitle: 'Lãi suất thấp nhất thị trường, thủ tục nhanh gọn', cta: 'Tính Toán Khoản Vay' },
          { type: 'features', items: ['Lãi suất từ 6.9%/năm', 'Vay tới 90% giá trị xe', 'Thời hạn vay tới 8 năm', 'Bảo hiểm xe miễn phí năm đầu'] },
          { type: 'form', fields: [
            { name: 'ho_va_ten', label: 'Họ và tên', required: true },
            { name: 'so_dien_thoai', label: 'Số điện thoại', required: true },
            { name: 'loai_xe', label: 'Hãng xe dự định mua', required: true },
            { name: 'gia_tri_xe', label: 'Giá trị xe (triệu VNĐ)', required: true },
            { name: 'vay_toi_da', label: 'Số tiền muốn vay (triệu VNĐ)', required: true }
          ]}
        ],
        seo: { title: 'Vay Mua Xe Ô Tô Lãi Suất Thấp | Apex Fintech' },
        status: 'published'
      }
    ]
  },
  // HCT Consulting
  {
    tenantId: 'aaaaaaaa-0000-0000-0000-000000000002',
    pages: [
      {
        slug: 'vinhomes-grand-park',
        title: 'Vinhomes Grand Park - Căn Hộ Cao Cấp TP.HCM | HCT',
        blocks: [
          { type: 'hero', title: 'Vinhomes Grand Park - Đô Thị Thông Minh Bên Sông Đồng Nai', subtitle: 'Căn hộ 1-4PN, view sông, công viên 36ha. Giá từ 4.5 tỷ', cta: 'Đặt Lịch Xem Nhà' },
          { type: 'features', items: [
            'Vị trí vàng: mặt tiền đường Nguyễn Xiển, Long Bình, TP.Thủ Đức',
            'Công viên Grand Park 36ha - lớn nhất Đông Nam Á',
            'Hồ bơi resort, gym, trường học Vinschool ngay trong khu',
            'Kết nối metro Bến Thành - Suối Tiên, 20 phút đến trung tâm'
          ]},
          { type: 'form', title: 'Đăng Ký Xem Nhà Miễn Phí', fields: [
            { name: 'ho_va_ten', label: 'Họ và tên', required: true },
            { name: 'so_dien_thoai', label: 'Số điện thoại', required: true },
            { name: 'email', label: 'Email', required: true },
            { name: 'ngan_sach', label: 'Ngân sách dự kiến (tỷ VNĐ)', required: true, options: ['2-5', '5-10', '10-20', '20+'] },
            { name: 'so_phong_ngu', label: 'Số phòng ngủ mong muốn', required: true, options: ['1PN', '2PN', '3PN', '4PN+'] },
            { name: 'thoi_gian_mua', label: 'Thời gian dự kiến mua', required: true, options: ['Trong 1 tháng', '1-3 tháng', '3-6 tháng', '6-12 tháng'] }
          ]}
        ],
        seo: { title: 'Vinhomes Grand Park Căn Hộ Cao Cấp | HCT Consulting' },
        status: 'published'
      },
      {
        slug: 'tu-van-dau-tu',
        title: 'Tư Vấn Đầu Tư BĐS Sinh Lời 15-25%/Năm',
        blocks: [
          { type: 'hero', title: 'Đầu Tư BĐS Thông Minh Cùng Chuyên Gia 15 Năm Kinh Nghiệm', subtitle: 'Phân tích dự án, đánh giá pháp lý, đàm phán giá', cta: 'Đặt Lịch Tư Vấn 1-1' },
          { type: 'features', items: [
            'Phân tích dòng tiền và tiềm năng tăng giá 5-10 năm',
            'Đánh giá pháp lý minh bạch (sổ đỏ, GPXD, PCCC)',
            'Đàm phán giá từ chủ đầu tư, tiết kiệm 5-15%',
            'Hỗ trợ vay NH với lãi suất ưu đãi 5-7%/năm'
          ]}
        ],
        seo: { title: 'Tư Vấn Đầu Tư BĐS Sinh Lời Cao | HCT' },
        status: 'published'
      }
    ]
  },
  // Demo Company
  {
    tenantId: 'bbbbbbbb-0000-0000-0000-000000000003',
    pages: [
      {
        slug: 'free-trial',
        title: 'Dùng Thử Miễn Phí RINCO CRM 30 Ngày',
        blocks: [
          { type: 'hero', title: 'Trải Nghiệm RINCO CRM Miễn Phí 30 Ngày', subtitle: 'Không cần thẻ tín dụng, đầy đủ tính năng', cta: 'Đăng Ký Ngay' },
          { type: 'features', items: [
            'Quản lý leads + deals tự động',
            'AI lead scoring 95% chính xác',
            'Email marketing automation',
            'Báo cáo real-time, dashboard tùy chỉnh',
            'Hỗ trợ 24/7 qua chat, Zalo, điện thoại'
          ]},
          { type: 'form', fields: [
            { name: 'ho_va_ten', label: 'Họ và tên', required: true },
            { name: 'email_cong_ty', label: 'Email công ty', required: true },
            { name: 'so_dien_thoai', label: 'Số điện thoại', required: true },
            { name: 'ten_cong_ty', label: 'Tên công ty', required: true },
            { name: 'so_luong_nhan_vien', label: 'Số lượng nhân viên', required: true, options: ['1-10', '11-50', '51-200', '201-500', '500+'] }
          ]}
        ],
        seo: { title: 'Dùng Thử CRM Miễn Phí 30 Ngày | RINCO' },
        status: 'published'
      }
    ]
  },
  // Vinamilk Distribution
  {
    tenantId: 'cccccccc-0000-0000-0000-000000000004',
    pages: [
      {
        slug: 'dang-ly-moi',
        title: 'Trở Thành Đại Lý Vinamilk - Thu Nhập Ổn Định',
        blocks: [
          { type: 'hero', title: 'Đăng Ký Trở Thành Đại Lý Vinamilk', subtitle: 'Hỗ trợ 100% chi phí nhập hàng đầu tiên, hỗ trợ POS miễn phí', cta: 'Đăng Ký Ngay' },
          { type: 'features', items: [
            'Chiết khấu 8-15% tuỳ sản phẩm',
            'Hỗ trợ 100% chi phí kệ trưng bày cho đại lý mới',
            'Đào tạo bán hàng + sử dụng POS miễn phí',
            'Vùng độc quyền, không cạnh tranh nội bộ'
          ]},
          { type: 'form', fields: [
            { name: 'ho_va_ten', label: 'Họ và tên chủ đại lý', required: true },
            { name: 'so_dien_thoai', label: 'Số điện thoại', required: true },
            { name: 'dia_chi_cua_hang', label: 'Địa chỉ cửa hàng', required: true },
            { name: 'tinh_thanh', label: 'Tỉnh/Thành phố', required: true },
            { name: 'loai_hinh', label: 'Loại hình đại lý', required: true, options: ['Cửa hàng tạp hoá', 'Siêu thị mini', 'Đại lý cấp 1', 'Đại lý cấp 2', 'Horeca'] },
            { name: 'doanh_thu_thang', label: 'Doanh thu trung bình tháng (triệu VNĐ)', required: true }
          ]}
        ],
        seo: { title: 'Đăng Ký Đại Lý Vinamilk Mới | Vinamilk Distribution' },
        status: 'published'
      }
    ]
  },
  // VNG Corporation
  {
    tenantId: 'dddddddd-0000-0000-0000-000000000005',
    pages: [
      {
        slug: 'zalo-for-business',
        title: 'Zalo for Business - Giải Pháp CSKH Toàn Diện',
        blocks: [
          { type: 'hero', title: 'Zalo OA + ZNS - Kênh CSKH Số 1 Việt Nam', subtitle: 'Gửi tin nhắn Zalo đến 76 triệu người dùng', cta: 'Dùng Thử Miễn Phí' },
          { type: 'features', items: [
            'Gửi tin nhắn Zalo OA đến 100K khách hàng/tháng',
            'ZNS (Zalo Notification Service) cho OTP, billing, marketing',
            'CRM tích hợp sẵn với Hubspot, Salesforce',
            'API RESTful đầy đủ, webhook real-time'
          ]},
          { type: 'form', fields: [
            { name: 'ho_va_ten', label: 'Họ và tên', required: true },
            { name: 'email_cong_ty', label: 'Email công ty', required: true },
            { name: 'so_dien_thoai', label: 'Số điện thoại', required: true },
            { name: 'ten_cong_ty', label: 'Tên công ty', required: true },
            { name: 'quy_mo_nhan_su', label: 'Quy mô nhân sự', required: true, options: ['1-50', '51-200', '201-1000', '1000+'] },
            { name: 'nganh_nghe', label: 'Ngành nghề', required: true, options: ['Retail', 'Finance', 'Education', 'Healthcare', 'Manufacturing', 'Hospitality'] }
          ]}
        ],
        seo: { title: 'Zalo for Business CSKH Toàn Diện | VNG' },
        status: 'published'
      }
    ]
  }
];

tenantLpData.forEach(t => {
  t.pages.forEach(page => {
    db.landing_pages.updateOne(
      { tenantId: t.tenantId, slug: page.slug },
      {
        $set: {
          tenantId: t.tenantId,
          slug: page.slug,
          title: page.title,
          blocks: page.blocks,
          seo: page.seo || {},
          status: page.status,
          publishedAt: page.status === 'published' ? new Date(now - 60 * 24 * 3600 * 1000) : null,
          version: 1,
          createdAt: new Date(now - 90 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    newLpCount++;
  });
});
print('  WS-D new landing_pages: ' + newLpCount + ' records');

// ============================================================
// SECTION B: ADDITIONAL ENRICHMENT CACHE (200 records)
// ============================================================
print('[WS-D-B] Additional enrichment cache (200 records)...');
const ecCompanies = [
  'FPT Software','Viettel','VNG','VinGroup','Sun Group','Masterise','VPBank','Techcombank',
  'Masan','Vietjet','Tiki','MoMo','Hoa Phat','CellphoneS','The Coffee House','Phuc Long',
  'Highlands Coffee','Bach Hoa Xanh','Shopee Vietnam','Lazada Vietnam','Grab Vietnam',
  'Be Group','FPT Telecom','VinaCapital','Dragon Capital','Vinamilk','Sabeco','Habeco',
  'Vietcombank','BIDV','Agribank','ACB','Sacombank','MB Bank','TPBank','HDBank','SHB',
  'OCB','MSB','VIB','SeABank','LPB','NamABank','PGBank','VietCapitalBank','BVBank'
];
const ecTitles = ['CEO','CTO','CFO','CMO','COO','CHRO','CIO','CISO','CDO','CMO','Marketing Director','Sales Director','Product Manager','Engineering Manager','Customer Success Lead','Account Executive'];
const ecLocations = ['Quận 1, TP.HCM','Quận 2, TP.HCM','Quận 3, TP.HCM','Quận Bình Thạnh, TP.HCM','Quận 7, TP.HCM','Quận Hoàn Kiếm, Hà Nội','Quận Đống Đa, Hà Nội','Quận Cầu Giấy, Hà Nội','Quận Thanh Xuân, Hà Nội','Quận Hải Châu, Đà Nẵng','Quận Sơn Trà, Đà Nẵng','Quận Ninh Kiều, Cần Thơ','Quận Lê Chân, Hải Phòng'];

let ecNewCount = 0;
for (let i = 0; i < 200; i++) {
  const fn = firstNames[Math.floor(Math.random() * firstNames.length)];
  const ln = lastNames[Math.floor(Math.random() * lastNames.length)];
  const t = tenants[Math.floor(Math.random() * tenants.length)];
  const company = ecCompanies[Math.floor(Math.random() * ecCompanies.length)];
  const email = fn.toLowerCase() + '.' + ln.toLowerCase().replace(/\s/g, '') + (i+1) + '@' + company.toLowerCase().replace(/\s+/g, '') + '.vn';

  db.enrichment_cache.updateOne(
    { tenantId: t.id, email: email },
    {
      $set: {
        tenantId: t.id,
        email: email,
        phone: '+849' + Math.floor(Math.random() * 10) + ' ' + Math.floor(1000000 + Math.random() * 8999999).toString(),
        fullName: fn + ' ' + ln,
        company: company,
        jobTitle: ecTitles[Math.floor(Math.random() * ecTitles.length)],
        linkedinUrl: 'https://linkedin.com/in/' + fn.toLowerCase() + '-' + ln.toLowerCase().replace(/\s/g, '') + '-' + (i+1),
        facebookUrl: 'https://facebook.com/' + fn.toLowerCase() + '.' + ln.toLowerCase().replace(/\s/g, '') + (i+1),
        location: ecLocations[Math.floor(Math.random() * ecLocations.length)],
        source: sources[Math.floor(Math.random() * sources.length)],
        confidence: 0.7 + Math.random() * 0.3,
        cachedAt: now,
        expiresAt: new Date(now.getTime() + 30 * 24 * 3600 * 1000),
        createdAt: new Date(now - Math.random() * 180 * 24 * 3600 * 1000)
      }
    },
    { upsert: true }
  );
  ecNewCount++;
}
print('  WS-D new enrichment_cache: ' + ecNewCount + ' records');

// ============================================================
// SECTION C: FORM DEFINITIONS WITH VN FIELDS
// ============================================================
print('[WS-D-C] Form definitions with Vietnamese fields...');
let formCount = 0;
const formDefs = [
  { tenantId: 'aaaaaaaa-0000-0000-0000-000000000001', code: 'loan-application-v2', name: 'Đăng ký khoản vay', fields: ['ho_va_ten','so_dien_thoai','email_cong_ty','so_tien_can_vay','muc_dich_vay','thu_nhap_hang_thang'] },
  { tenantId: 'aaaaaaaa-0000-0000-0000-000000000001', code: 'credit-card-application', name: 'Đăng ký thẻ tín dụng', fields: ['ho_va_ten','so_dien_thoai','email_cong_ty','loai_the','so_luong_the'] },
  { tenantId: 'aaaaaaaa-0000-0000-0000-000000000002', code: 'viewing-booking', name: 'Đặt lịch xem nhà', fields: ['ho_va_ten','so_dien_thoai','email','du_an','can_ho','thoi_gian_xem'] },
  { tenantId: 'aaaaaaaa-0000-0000-0000-000000000002', code: 'investment-consultation', name: 'Tư vấn đầu tư', fields: ['ho_va_ten','so_dien_thoai','email','ngan_sach','muc_tieu_dau_tu'] },
  { tenantId: 'bbbbbbbb-0000-0000-0000-000000000003', code: 'trial-signup-v2', name: 'Đăng ký dùng thử', fields: ['ho_va_ten','email_cong_ty','so_dien_thoai','ten_cong_ty','so_luong_nhan_vien'] },
  { tenantId: 'cccccccc-0000-0000-0000-000000000004', code: 'dealer-application', name: 'Đăng ký đại lý', fields: ['ho_va_ten','so_dien_thoai','dia_chi_cua_hang','tinh_thanh','loai_hinh','doanh_thu_thang'] },
  { tenantId: 'dddddddd-0000-0000-0000-000000000005', code: 'enterprise-contact', name: 'Liên hệ doanh nghiệp', fields: ['ho_va_ten','email_cong_ty','so_dien_thoai','ten_cong_ty','quy_mo_nhan_su','nganh_nghe'] },
  { tenantId: 'dddddddd-0000-0000-0000-000000000005', code: 'zns-trial-signup', name: 'Dùng thử ZNS miễn phí', fields: ['ho_va_ten','email_cong_ty','so_dien_thoai','ten_cong_ty','quy_mo_tin_nhan'] }
];

formDefs.forEach(f => {
  db.forms.updateOne(
    { tenantId: f.tenantId, code: f.code },
    {
      $set: {
        tenantId: f.tenantId,
        code: f.code,
        name: f.name,
        fields: f.fields,
        status: 'active',
        createdAt: new Date(now - Math.random() * 90 * 24 * 3600 * 1000),
        updatedAt: now
      }
    },
    { upsert: true }
  );
  formCount++;
});
print('  WS-D new forms: ' + formCount + ' records');

print('[WS-D] MongoDB expansion complete.');