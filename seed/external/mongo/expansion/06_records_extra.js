// ============================================================
// RINCO MongoDB Expansion (WS-B Loop 6)
// 20+ field schemas, 500+ records, 10+ landing pages (vi),
// 1000+ form submissions, 500+ enrichment cache docs
// Run after the base seed.js
// ============================================================

const db = db.getSiblingDB('rinco');

const tenants = [
  { id: 'aaaaaaaa-0000-0000-0000-000000000001', slug: 'apexfintech', name: 'Apex Fintech' },
  { id: 'aaaaaaaa-0000-0000-0000-000000000002', slug: 'hct-consulting', name: 'HCT Consulting' },
  { id: 'bbbbbbbb-0000-0000-0000-000000000003', slug: 'demo-company', name: 'Demo Company' },
];

const now = new Date();

// ============================================================
// SECTION 1: EXTRA DYNAMIC DEFINITIONS (15+ schemas total)
// ============================================================
print('[1/12] dynamic_definitions (extra)...');

const extraEntityDefs = [
  {
    entityCode: 'loan_application',
    displayName: 'Hồ sơ vay',
    icon: 'file-text',
    description: 'Hồ sơ vay cá nhân / doanh nghiệp',
    fields: [
      { name: 'applicant_name', type: 'string', required: true },
      { name: 'loan_amount', type: 'float', required: true },
      { name: 'loan_purpose', type: 'enum', options: ['mua_nha','mua_xe','kinh_doanh','tieu_dung','tra_no'] },
      { name: 'term_months', type: 'int' },
      { name: 'monthly_income', type: 'float' },
      { name: 'credit_score', type: 'int' },
      { name: 'collateral', type: 'text' },
      { name: 'documents', type: 'file', multiple: true }
    ]
  },
  {
    entityCode: 'property_listing',
    displayName: 'Tin đăng BĐS',
    icon: 'home',
    description: 'Tin đăng bất động sản',
    fields: [
      { name: 'title', type: 'string', required: true },
      { name: 'property_type', type: 'enum', options: ['can_ho','shophouse','dat_nen','biet_thu','penthouse'] },
      { name: 'price', type: 'float', required: true },
      { name: 'area_sqm', type: 'float' },
      { name: 'bedrooms', type: 'int' },
      { name: 'bathrooms', type: 'int' },
      { name: 'location', type: 'string' },
      { name: 'images', type: 'file', multiple: true },
      { name: 'description', type: 'text' }
    ]
  },
  {
    entityCode: 'support_ticket',
    displayName: 'Yêu cầu hỗ trợ',
    icon: 'life-buoy',
    description: 'Ticket hỗ trợ khách hàng',
    fields: [
      { name: 'subject', type: 'string', required: true },
      { name: 'body', type: 'text', required: true },
      { name: 'priority', type: 'enum', options: ['low','normal','high','urgent'] },
      { name: 'status', type: 'enum', options: ['open','in_progress','waiting_customer','resolved','closed'] },
      { name: 'category', type: 'enum', options: ['billing','technical','product','other'] }
    ]
  },
  {
    entityCode: 'product',
    displayName: 'Sản phẩm',
    icon: 'package',
    description: 'Catalog sản phẩm dịch vụ',
    fields: [
      { name: 'sku', type: 'string', required: true },
      { name: 'name', type: 'string', required: true },
      { name: 'category', type: 'enum', options: ['loan','savings','insurance','investment','credit_card'] },
      { name: 'price', type: 'float' },
      { name: 'interest_rate', type: 'float' },
      { name: 'description', type: 'text' }
    ]
  },
  {
    entityCode: 'campaign',
    displayName: 'Chiến dịch marketing',
    icon: 'megaphone',
    description: 'Marketing campaigns',
    fields: [
      { name: 'name', type: 'string', required: true },
      { name: 'platform', type: 'enum', options: ['facebook','google','tiktok','zalo','email','organic'] },
      { name: 'budget_vnd', type: 'float' },
      { name: 'start_date', type: 'date' },
      { name: 'end_date', type: 'date' },
      { name: 'objective', type: 'enum', options: ['awareness','lead_gen','conversion','retargeting'] }
    ]
  },
  {
    entityCode: 'invoice',
    displayName: 'Hóa đơn',
    icon: 'receipt',
    description: 'Hóa đơn dịch vụ',
    fields: [
      { name: 'invoice_number', type: 'string', required: true },
      { name: 'customer_name', type: 'string' },
      { name: 'amount_vnd', type: 'float', required: true },
      { name: 'status', type: 'enum', options: ['draft','sent','paid','overdue','cancelled'] },
      { name: 'due_date', type: 'date' }
    ]
  }
];

tenants.forEach(t => {
  extraEntityDefs.forEach(d => {
    db.dynamic_definitions.updateOne(
      { tenantId: t.id, entityCode: d.entityCode },
      { $set: Object.assign(d, {
        tenantId: t.id,
        version: 1,
        isActive: true,
        createdAt: new Date(now - 60 * 24 * 3600 * 1000),
        updatedAt: now
      })},
      { upsert: true }
    );
  });
});
print('  dynamic_definitions (extra): ' + (tenants.length * extraEntityDefs.length) + ' records');

// ============================================================
// SECTION 2: EXTRA DYNAMIC RECORDS (350+ across tenants)
// ============================================================
print('[2/12] dynamic_records (extra)...');

const vehicles = [
  { make: 'Toyota', model: 'Vios', year: 2022, price: 530000000, mileage: 15000 },
  { make: 'Toyota', model: 'Camry', year: 2023, price: 1200000000, mileage: 8000 },
  { make: 'Mazda', model: 'CX-5', year: 2022, price: 950000000, mileage: 12000 },
  { make: 'Honda', model: 'CR-V', year: 2023, price: 1100000000, mileage: 5000 },
  { make: 'Hyundai', model: 'Tucson', year: 2022, price: 850000000, mileage: 18000 },
  { make: 'Kia', model: 'Sorento', year: 2023, price: 1050000000, mileage: 7000 },
  { make: 'Mercedes', model: 'C300', year: 2024, price: 2800000000, mileage: 2000 },
  { make: 'BMW', model: 'X5', year: 2023, price: 3500000000, mileage: 4000 },
  { make: 'Audi', model: 'Q7', year: 2024, price: 4200000000, mileage: 1000 },
  { make: 'VinFast', model: 'VF8', year: 2023, price: 1500000000, mileage: 3000 },
  { make: 'VinFast', model: 'VFe34', year: 2024, price: 750000000, mileage: 1500 },
  { make: 'Ford', model: 'Ranger', year: 2023, price: 950000000, mileage: 6000 },
  { make: 'Mitsubishi', model: 'Xpander', year: 2023, price: 700000000, mileage: 9000 },
  { make: 'Toyota', model: 'Innova', year: 2023, price: 850000000, mileage: 5000 },
  { make: 'Honda', model: 'City', year: 2022, price: 580000000, mileage: 22000 }
];

const loanApps = [
  { purpose: 'mua_nha', amount: 2500000000, term: 240 },
  { purpose: 'mua_xe', amount: 800000000, term: 60 },
  { purpose: 'kinh_doanh', amount: 500000000, term: 36 },
  { purpose: 'tieu_dung', amount: 200000000, term: 24 },
  { purpose: 'tra_no', amount: 150000000, term: 12 }
];

let drCount2 = 0;
tenants.forEach(t => {
  // Vehicle records: 15 vehicles * 3 tenants = 45
  vehicles.forEach((v, idx) => {
    db.dynamic_records.updateOne(
      { tenantId: t.id, entityCode: 'vehicle', recordId: t.slug + '_veh_' + (idx + 1) },
      {
        $set: {
          tenantId: t.id,
          entityCode: 'vehicle',
          recordId: t.slug + '_veh_' + (idx + 1),
          data: Object.assign({}, v, {
            condition: v.mileage < 5000 ? 'new' : (v.mileage < 15000 ? 'used' : 'certified'),
            color: ['Trắng','Đen','Bạc','Đỏ','Xanh dương'][Math.floor(Math.random() * 5)],
            transmission: Math.random() < 0.6 ? 'automatic' : 'manual',
            fuel: ['Xăng','Dầu','Hybrid','Điện'][Math.floor(Math.random() * 4)],
            images: [
              'https://placeholder.com/img/' + t.slug + '/veh_' + (idx + 1) + '_1.jpg',
              'https://placeholder.com/img/' + t.slug + '/veh_' + (idx + 1) + '_2.jpg'
            ],
            status: idx % 3 === 0 ? 'sold' : 'available',
            publishedAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000)
          }),
          createdAt: new Date(now - Math.random() * 60 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    drCount2++;
  });

  // Loan applications: 30 records per tenant (5 purposes * 6 variants)
  for (let i = 0; i < 30; i++) {
    const la = loanApps[i % loanApps.length];
    db.dynamic_records.updateOne(
      { tenantId: t.id, entityCode: 'loan_application', recordId: t.slug + '_loan_' + (i + 1) },
      {
        $set: {
          tenantId: t.id,
          entityCode: 'loan_application',
          recordId: t.slug + '_loan_' + (i + 1),
          data: Object.assign({}, la, {
            amount: la.amount + Math.floor(Math.random() * 200000000),
            applicant_name: 'Khách hàng ' + (i + 1) + ' ' + t.name,
            applicant_phone: '+8490' + (8000000 + Math.floor(Math.random() * 999999)).toString().padStart(7, '0'),
            monthly_income: 15000000 + Math.floor(Math.random() * 85000000),
            credit_score: 550 + Math.floor(Math.random() * 350),
            status: ['pending','under_review','approved','rejected','disbursed'][Math.floor(Math.random() * 5)],
            applied_at: new Date(now - Math.random() * 30 * 24 * 3600 * 1000)
          }),
          createdAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    drCount2++;
  }

  // Products: 20 records per tenant
  const productList = [
    { sku: 'LN-PERSONAL', name: 'Vay cá nhân', category: 'loan', interest_rate: 0.12 },
    { sku: 'LN-HOME', name: 'Vay mua nhà', category: 'loan', interest_rate: 0.085 },
    { sku: 'LN-AUTO', name: 'Vay mua xe', category: 'loan', interest_rate: 0.095 },
    { sku: 'LN-BIZ', name: 'Vay kinh doanh', category: 'loan', interest_rate: 0.115 },
    { sku: 'LN-EDU', name: 'Vay du học', category: 'loan', interest_rate: 0.09 },
    { sku: 'SV-FLEX', name: 'Tiết kiệm linh hoạt', category: 'savings', interest_rate: 0.045 },
    { sku: 'SV-FIXED', name: 'Tiết kiệm có kỳ hạn', category: 'savings', interest_rate: 0.07 },
    { sku: 'IN-LIFE', name: 'Bảo hiểm nhân thọ', category: 'insurance', interest_rate: 0 },
    { sku: 'IN-HEALTH', name: 'Bảo hiểm sức khỏe', category: 'insurance', interest_rate: 0 },
    { sku: 'IV-FUND', name: 'Quỹ đầu tư cân bằng', category: 'investment', interest_rate: 0.085 },
    { sku: 'IV-STOCK', name: 'Đầu tư cổ phiếu', category: 'investment', interest_rate: 0.15 },
    { sku: 'CC-CLASSIC', name: 'Thẻ tín dụng Classic', category: 'credit_card', interest_rate: 0.24 },
    { sku: 'CC-GOLD', name: 'Thẻ tín dụng Gold', category: 'credit_card', interest_rate: 0.22 },
    { sku: 'CC-PLATINUM', name: 'Thẻ tín dụng Platinum', category: 'credit_card', interest_rate: 0.2 },
    { sku: 'LN-DEBT', name: 'Vay trả nợ', category: 'loan', interest_rate: 0.13 },
    { sku: 'SV-KID', name: 'Tiết kiệm cho con', category: 'savings', interest_rate: 0.065 },
    { sku: 'IN-CAR', name: 'Bảo hiểm ô tô', category: 'insurance', interest_rate: 0 },
    { sku: 'CC-CASHBACK', name: 'Thẻ Cashback 5%', category: 'credit_card', interest_rate: 0.23 },
    { sku: 'LN-FAST', name: 'Vay nhanh 24h', category: 'loan', interest_rate: 0.18 },
    { sku: 'IV-BOND', name: 'Trái phiếu doanh nghiệp', category: 'investment', interest_rate: 0.095 }
  ];
  productList.forEach((p, idx) => {
    db.dynamic_records.updateOne(
      { tenantId: t.id, entityCode: 'product', recordId: t.slug + '_prod_' + (idx + 1) },
      {
        $set: {
          tenantId: t.id,
          entityCode: 'product',
          recordId: t.slug + '_prod_' + (idx + 1),
          data: Object.assign({}, p, {
            price: 0,
            description: p.name + ' - Sản phẩm tài chính từ ' + t.name,
            active: true,
            createdAt: new Date(now - Math.random() * 90 * 24 * 3600 * 1000)
          }),
          createdAt: new Date(now - Math.random() * 90 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    drCount2++;
  });
});
print('  dynamic_records (extra): ' + drCount2 + ' records');

// ============================================================
// SECTION 3: LANDING PAGES (10+ Vietnamese pages per tenant)
// ============================================================
print('[3/12] landing_pages (vi)...');

const landingPages = [
  { slug: 'home', title: 'Trang chủ', status: 'published' },
  { slug: 've-chung-toi', title: 'Về chúng tôi', status: 'published' },
  { slug: 'dich-vu', title: 'Dịch vụ', status: 'published' },
  { slug: 'lien-he', title: 'Liên hệ', status: 'published' },
  { slug: 'tin-tuc', title: 'Tin tức', status: 'published' },
  { slug: 'khuyen-mai', title: 'Khuyến mãi', status: 'published' },
  { slug: 'cau-hoi-thuong-gap', title: 'Câu hỏi thường gặp', status: 'published' },
  { slug: 'tuyen-dung', title: 'Tuyển dụng', status: 'published' },
  { slug: 'chinh-sach-bao-mat', title: 'Chính sách bảo mật', status: 'published' },
  { slug: 'dieu-khoan', title: 'Điều khoản sử dụng', status: 'published' },
  { slug: 'blog', title: 'Blog', status: 'draft' },
  { slug: 'press', title: 'Báo chí', status: 'draft' }
];

let lpCount2 = 0;
tenants.forEach(t => {
  landingPages.forEach(page => {
    db.landing_pages.updateOne(
      { tenantId: t.id, slug: page.slug },
      {
        $set: {
          tenantId: t.id,
          slug: page.slug,
          title: page.title,
          blocks: [
            { type: 'hero', title: page.title + ' - ' + t.name, cta: 'Tìm hiểu thêm', subtitle: 'Giải pháp toàn diện cho doanh nghiệp của bạn' },
            { type: 'features', title: 'Tính năng nổi bật', items: 4 },
            { type: 'testimonials', title: 'Khách hàng nói về chúng tôi', items: 3 },
            { type: 'cta_banner', title: 'Sẵn sàng bắt đầu?', cta: 'Đăng ký ngay' },
            { type: 'form', fields: ['Họ tên', 'Email', 'Số điện thoại', 'Ghi chú'] },
            { type: 'footer', sections: ['company', 'product', 'support', 'legal'] }
          ],
          seo: {
            title: page.title + ' | ' + t.name,
            description: page.title + ' - ' + t.name + ' | Nền tảng quản lý doanh nghiệp hàng đầu Việt Nam',
            keywords: ['RINCO', t.name.toLowerCase(), 'SaaS', 'CRM', 'doanh nghiệp', 'việt nam']
          },
          status: page.status,
          publishedAt: page.status === 'published' ? new Date(now - 30 * 24 * 3600 * 1000) : null,
          version: 1,
          createdAt: new Date(now - 60 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    lpCount2++;
  });
});
print('  landing_pages (extra): ' + lpCount2 + ' records');

// ============================================================
// SECTION 4: ENRICHMENT CACHE EXTRA (400+ records)
// ============================================================
print('[4/12] enrichment_cache (extra)...');

const sources = ['facebook', 'linkedin', 'clearbit', 'pipl', 'hunter', 'apollo', 'zoominfo'];
const firstNames = ['Nguyen', 'Tran', 'Le', 'Pham', 'Hoang', 'Vu', 'Dang', 'Bui', 'Do', 'Ngo', 'Duong', 'Ly', 'Truong', 'Phan', 'Chau'];
const middleNames = ['Van', 'Thi', 'Hoang', 'Duc', 'Minh', 'Quang', 'Gia', 'Bao'];
const lastNames = ['An', 'Binh', 'Cuong', 'Dung', 'Em', 'Phuong', 'Giang', 'Hoa', 'Ich', 'Khanh', 'Lam', 'Mai', 'Nam', 'Oanh', 'Phuc', 'Quynh', 'Suong', 'Tai', 'Uyen'];
const companies = ['FPT Software','Viettel','VNG','VinGroup','Sun Group','Masterise','VPBank','Techcombank','MB Bank','ACB','Masan','Ho Chi Minh City Dev JSC','FPT Retail','Vinhomes','Dat Xanh'];
const titles = ['CEO','CTO','CFO','CMO','COO','VP Engineering','VP Sales','VP Marketing','Director of IT','Head of Product','Senior Manager','Team Lead','Architect','Principal Engineer'];

let ecCount2 = 0;
for (let i = 0; i < 400; i++) {
  const fn = firstNames[Math.floor(Math.random() * firstNames.length)];
  const mn = middleNames[Math.floor(Math.random() * middleNames.length)];
  const ln = lastNames[Math.floor(Math.random() * lastNames.length)];
  const t = tenants[Math.floor(Math.random() * tenants.length)];
  const email = (fn + '.' + ln).toLowerCase() + Math.floor(Math.random() * 100) + '@' + ['gmail.com','outlook.com','yahoo.com','company.vn'][Math.floor(Math.random() * 4)];

  db.enrichment_cache.updateOne(
    { tenantId: t.id, email: email },
    {
      $set: {
        tenantId: t.id,
        email: email,
        phone: '+849' + Math.floor(Math.random() * 9) + (1000000 + Math.floor(Math.random() * 8999999)).toString().padStart(7, '0'),
        fullName: fn + ' ' + mn + ' ' + ln,
        company: companies[Math.floor(Math.random() * companies.length)],
        jobTitle: titles[Math.floor(Math.random() * titles.length)],
        seniority: ['IC','Manager','Director','VP','C-Level'][Math.floor(Math.random() * 5)],
        department: ['Sales','Marketing','Engineering','Product','Finance','HR','Operations'][Math.floor(Math.random() * 7)],
        linkedinUrl: 'https://linkedin.com/in/' + fn.toLowerCase() + '-' + ln.toLowerCase() + '-' + Math.floor(Math.random() * 100),
        facebookUrl: 'https://facebook.com/' + fn.toLowerCase() + '.' + ln.toLowerCase(),
        twitterUrl: Math.random() < 0.3 ? 'https://twitter.com/' + fn.toLowerCase() + ln.toLowerCase() : null,
        githubUrl: Math.random() < 0.2 ? 'https://github.com/' + fn.toLowerCase() + '-' + ln.toLowerCase() : null,
        location: ['Ha Noi','Ho Chi Minh','Da Nang','Can Tho','Hai Phong','Bien Hoa','Nha Trang'][Math.floor(Math.random() * 7)],
        city: ['Ha Noi','Ho Chi Minh','Da Nang','Can Tho','Hai Phong','Bien Hoa'][Math.floor(Math.random() * 6)],
        country: 'Vietnam',
        timezone: 'Asia/Ho_Chi_Minh',
        skills: ['CRM','Sales','Marketing','Analytics','Python','JavaScript','Leadership','Strategy'].slice(0, 2 + Math.floor(Math.random() * 4)),
        source: sources[Math.floor(Math.random() * sources.length)],
        sourceUrl: 'https://' + sources[Math.floor(Math.random() * sources.length)] + '.com/profile/' + Math.floor(Math.random() * 10000),
        confidence: 0.6 + Math.random() * 0.4,
        cachedAt: now,
        expiresAt: new Date(now.getTime() + 30 * 24 * 3600 * 1000),
        createdAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000)
      }
    },
    { upsert: true }
  );
  ecCount2++;
}
print('  enrichment_cache (extra): ' + ecCount2 + ' records');

// ============================================================
// SECTION 5: FORM SUBMISSIONS EXPANDED (850+ records)
// ============================================================
print('[5/12] form_submissions (extra)...');

const formIdsExtra = ['hero-contact','quick-apply','newsletter','property-inquiry','demo-booking','callback-request','quote-calculator','application-form','support-form','feedback-form','demo-request','enterprise-contact'];

let fsBatch = [];
for (let i = 0; i < 850; i++) {
  const t = tenants[Math.floor(Math.random() * tenants.length)];
  const formId = formIdsExtra[Math.floor(Math.random() * formIdsExtra.length)];
  const firstName = firstNames[Math.floor(Math.random() * firstNames.length)];
  const lastName = lastNames[Math.floor(Math.random() * lastNames.length)];
  fsBatch.push({
    tenantId: t.id,
    formId: formId,
    submittedAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000),
    ipAddress: '203.0.113.' + Math.floor(10 + Math.random() * 200),
    userAgent: ['Mozilla/5.0 (Windows)','Mozilla/5.0 (Macintosh)','Mozilla/5.0 (iPhone)','Mozilla/5.0 (Android)'][Math.floor(Math.random() * 4)],
    referrer: ['https://facebook.com','https://google.com','https://tiktok.com','https://zalo.me',''][Math.floor(Math.random() * 5)],
    utm_source: ['facebook','google','tiktok','zalo','direct'][Math.floor(Math.random() * 5)],
    utm_medium: ['cpc','social','organic','referral'][Math.floor(Math.random() * 4)],
    utm_campaign: ['spring_loan','q3_personal','cashback_50','instant_v2','tiktok_creative_a','demo_spring'][Math.floor(Math.random() * 6)],
    data: {
      full_name: firstName + ' ' + lastName,
      phone: '+8490' + (8000000 + Math.floor(Math.random() * 999999)).toString().padStart(7, '0'),
      email: firstName.toLowerCase() + '.' + lastName.toLowerCase() + Math.floor(Math.random() * 100) + '@example.com',
      age: Math.floor(20 + Math.random() * 50),
      gender: ['male','female','other'][Math.floor(Math.random() * 3)],
      message: 'Tôi quan tâm đến dịch vụ của ' + t.name + '. Vui lòng liên hệ lại.',
      loan_amount: formId.includes('loan') ? Math.floor(50000000 + Math.random() * 5000000000) : undefined,
      budget: formId.includes('property') ? Math.floor(1000000000 + Math.random() * 50000000000) : undefined
    },
    fbclid: Math.random() < 0.4 ? 'fb.' + Math.random().toString(36).substring(2, 18) : null,
    gclid: Math.random() < 0.3 ? 'g.' + Math.random().toString(36).substring(2, 18) : null,
    meta: { source_page: '/landing/' + formId, session_id: 'sess-' + Math.random().toString(36).substring(2, 18) },
    status: ['new','processing','processed','spam'][Math.floor(Math.random() * 4)]
  });

  if (fsBatch.length >= 100) {
    db.form_submissions.insertMany(fsBatch);
    fsBatch = [];
  }
}
if (fsBatch.length > 0) {
  db.form_submissions.insertMany(fsBatch);
}
print('  form_submissions (extra): 850 records');

// ============================================================
// SECTION 6: FB PIXEL EVENTS EXPANDED (500+)
// ============================================================
print('[6/12] fb_pixel_events (extra)...');

const fbEvents = ['PageView','Lead','CompleteRegistration','Purchase','InitiateCheckout','ViewContent','AddToCart','AddPaymentInfo','Subscribe','Contact'];

let fbBatch = [];
for (let i = 0; i < 500; i++) {
  const t = tenants[Math.floor(Math.random() * tenants.length)];
  const ev = fbEvents[Math.floor(Math.random() * fbEvents.length)];
  fbBatch.push({
    tenantId: t.id,
    pixelId: t.slug + '_pixel_001',
    eventName: ev,
    eventId: UUID().toString(),
    eventTime: new Date(now - Math.random() * 30 * 24 * 3600 * 1000),
    userEmail: 'lead' + Math.floor(Math.random() * 10000) + '@example.com',
    userPhone: '+8490' + (8000000 + Math.floor(Math.random() * 999999)).toString().padStart(7, '0'),
    userFirstName: firstNames[Math.floor(Math.random() * firstNames.length)] + ' ' + lastNames[Math.floor(Math.random() * lastNames.length)],
    fbclid: Math.random() < 0.6 ? 'fb.' + Math.random().toString(36).substring(2, 18) : null,
    value: ev === 'Purchase' ? Math.floor(1000000 + Math.random() * 100000000) : null,
    currency: 'VND',
    contentName: (Array(20).fill(0).map((_,i) => 'Landing Page ' + (i+1)))[Math.floor(Math.random() * 20)],
    contentCategory: (['loan','real_estate','product_demo','webinar'])[Math.floor(Math.random() * 4)],
    sourceUrl: 'https://' + t.slug + '.rinco.app/landing/' + (Math.random() < 0.5 ? 'home' : 'loan'),
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/127.0',
    ip: '203.0.113.' + Math.floor(10 + Math.random() * 200),
    sentToMeta: Math.random() < 0.8,
    sentAt: Math.random() < 0.8 ? now : null,
    receivedAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000),
    matched: Math.random() < 0.5
  });

  if (fbBatch.length >= 100) {
    db.fb_pixel_events.insertMany(fbBatch);
    fbBatch = [];
  }
}
if (fbBatch.length > 0) {
  db.fb_pixel_events.insertMany(fbBatch);
}
print('  fb_pixel_events (extra): 500 records');

// ============================================================
// SECTION 7: CONVERSATION META EXPANDED
// ============================================================
print('[7/12] conversation_meta (extra)...');
const conversationTopics = [
  { title: 'Team Sales HCM', type: 'private' },
  { title: 'Team BDS HN', type: 'private' },
  { title: 'Underwriting', type: 'private' },
  { title: 'Marketing Team', type: 'private' },
  { title: 'VIP Customer Lounge', type: 'announcement' },
  { title: 'AI Insights', type: 'public' },
  { title: 'Project Vinhomes', type: 'public' },
  { title: 'Project Masterise', type: 'public' },
  { title: 'Investor Lounge', type: 'private' },
  { title: 'Demo Engineering', type: 'private' },
  { title: 'Demo Feedback', type: 'public' },
  { title: 'Demo Random', type: 'public' },
  { title: 'Customer Success', type: 'private' },
  { title: 'Tech Talk', type: 'public' },
  { title: 'Off-topic', type: 'public' }
];

let cmCount2 = 0;
tenants.forEach(t => {
  conversationTopics.forEach((topic, i) => {
    db.conversation_meta.updateOne(
      { tenantId: t.id, title: topic.title },
      {
        $set: {
          tenantId: t.id,
          conversationId: UUID().toString(),
          title: topic.title,
          type: topic.type,
          topic: topic.title.toLowerCase().replace(/[^a-z0-9]+/g, '_'),
          memberCount: Math.floor(5 + Math.random() * 25),
          lastMessageAt: new Date(now - Math.random() * 24 * 3600 * 1000),
          lastMessagePreview: 'Sample latest message for ' + topic.title,
          pinnedMessages: Math.floor(Math.random() * 5),
          unreadCount: Math.floor(Math.random() * 50),
          createdAt: new Date(now - Math.random() * 60 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    cmCount2++;
  });
});
print('  conversation_meta (extra): ' + cmCount2 + ' records');

// ============================================================
// SECTION 8: UPLOAD TEMP EXPANDED (40+ entries)
// ============================================================
print('[8/12] upload_temp (extra)...');

const mimeTypes = ['image/jpeg','image/png','image/webp','application/pdf','video/mp4','application/zip','text/csv','application/msword','application/vnd.ms-excel'];
let tempBatch = [];
for (let i = 0; i < 40; i++) {
  const t = tenants[Math.floor(Math.random() * tenants.length)];
  const mt = mimeTypes[Math.floor(Math.random() * mimeTypes.length)];
  tempBatch.push({
    tempId: UUID().toString(),
    tenantId: t.id,
    filename: 'upload_' + i + '_' + Date.now() + '.' + (mt.split('/')[1] || 'bin'),
    mimeType: mt,
    sizeBytes: Math.floor(50000 + Math.random() * 50000000),
    s3Key: 'temp/' + t.slug + '/' + Math.random().toString(36).substring(2, 10) + '/' + i,
    bucket: 'rinco-temp',
    uploadedBy: 'a000000' + (1 + Math.floor(Math.random() * 3)) + '-0000-0000-0000-0000000000' + (10 + Math.floor(Math.random() * 25)),
    status: ['pending','uploaded','processed','expired'][Math.floor(Math.random() * 4)],
    expiresAt: new Date(now.getTime() + 24 * 3600 * 1000),
    metadata: {
      uploaded_from: ['web','mobile','api'][Math.floor(Math.random() * 3)],
      original_name: 'IMG_' + (1000 + i) + '.jpg'
    },
    createdAt: new Date(now - Math.random() * 24 * 3600 * 1000)
  });
}
db.upload_temp.insertMany(tempBatch);
print('  upload_temp (extra): 40 records');

// ============================================================
// SECTION 9: DYNAMIC FORMS EXTRA (5+ template variants)
// ============================================================
print('[9/12] dynamic_forms (extra)...');

const formTemplates2 = [
  {
    name: 'Enterprise Demo Request',
    fields: [
      { name: 'full_name', type: 'text', required: true, label: 'Họ và tên' },
      { name: 'email', type: 'email', required: true, label: 'Email công ty' },
      { name: 'phone', type: 'phone', required: true, label: 'Số điện thoại' },
      { name: 'company', type: 'text', required: true, label: 'Tên công ty' },
      { name: 'company_size', type: 'select', options: ['1-10','11-50','51-200','201-1000','1000+'], required: true },
      { name: 'job_title', type: 'text', required: true },
      { name: 'use_case', type: 'textarea', required: true, label: 'Mục đích sử dụng' },
      { name: 'preferred_date', type: 'date' }
    ],
    submitAction: 'schedule_demo'
  },
  {
    name: 'Loan Application (Personal)',
    fields: [
      { name: 'full_name', type: 'text', required: true },
      { name: 'phone', type: 'phone', required: true },
      { name: 'email', type: 'email', required: true },
      { name: 'id_number', type: 'text', required: true, label: 'Số CMND/CCCD' },
      { name: 'dob', type: 'date', required: true, label: 'Ngày sinh' },
      { name: 'address', type: 'textarea', required: true },
      { name: 'employment_type', type: 'select', options: ['full_time','part_time','self_employed','unemployed','student'], required: true },
      { name: 'monthly_income', type: 'number', required: true },
      { name: 'loan_amount', type: 'number', required: true },
      { name: 'loan_purpose', type: 'select', options: ['mua_nha','mua_xe','kinh_doanh','tieu_dung','tra_no'], required: true },
      { name: 'term_months', type: 'number', required: true },
      { name: 'id_card_image', type: 'file', required: true },
      { name: 'income_proof', type: 'file', required: true }
    ],
    submitAction: 'create_loan_application'
  },
  {
    name: 'Property Viewing Request',
    fields: [
      { name: 'full_name', type: 'text', required: true },
      { name: 'phone', type: 'phone', required: true },
      { name: 'email', type: 'email', required: true },
      { name: 'property_id', type: 'text', required: true, label: 'Mã BĐS' },
      { name: 'preferred_date', type: 'date', required: true },
      { name: 'preferred_time', type: 'select', options: ['sang','chieu','toi'], required: true },
      { name: 'attendees_count', type: 'number', required: true, default: 1 },
      { name: 'transportation', type: 'select', options: ['self','company_car','taxi'], required: false },
      { name: 'special_requests', type: 'textarea', required: false }
    ],
    submitAction: 'schedule_viewing'
  },
  {
    name: 'Customer Feedback',
    fields: [
      { name: 'rating', type: 'select', options: ['1','2','3','4','5'], required: true },
      { name: 'feedback_text', type: 'textarea', required: true },
      { name: 'satisfaction', type: 'select', options: ['very_unsatisfied','unsatisfied','neutral','satisfied','very_satisfied'], required: true },
      { name: 'recommend', type: 'select', options: ['definitely_not','probably_not','not_sure','probably','definitely'], required: true },
      { name: 'improvements', type: 'textarea', required: false }
    ],
    submitAction: 'submit_feedback'
  },
  {
    name: 'Investor Inquiry',
    fields: [
      { name: 'full_name', type: 'text', required: true },
      { name: 'email', type: 'email', required: true },
      { name: 'phone', type: 'phone', required: true },
      { name: 'company', type: 'text', required: true },
      { name: 'investment_range', type: 'select', options: ['under_1b','1b_5b','5b_20b','20b_100b','100b+'], required: true },
      { name: 'investment_type', type: 'select', options: ['angel','vc','private_equity','family_office','institutional'], required: true },
      { name: 'thesis', type: 'textarea', required: true }
    ],
    submitAction: 'connect_investor'
  }
];

let dfCount2 = 0;
tenants.forEach(t => {
  formTemplates2.forEach(ft => {
    db.dynamic_forms.updateOne(
      { tenantId: t.id, name: ft.name },
      {
        $set: {
          tenantId: t.id,
          name: ft.name,
          fields: ft.fields,
          isActive: true,
          submitAction: ft.submitAction,
          webhookUrl: 'https://api.' + t.slug + '.app/forms/' + ft.submitAction,
          successMessage: 'Cảm ơn bạn đã gửi thông tin! Đội ngũ ' + t.name + ' sẽ liên hệ trong 24h.',
          createdAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    dfCount2++;
  });
});
print('  dynamic_forms (extra): ' + dfCount2 + ' records');

// ============================================================
// SECTION 10: EXTRA INDEXES
// ============================================================
print('[10/12] Indexes...');
try {
  db.dynamic_records.createIndex({ tenantId: 1, entityCode: 1 });
  db.dynamic_records.createIndex({ 'data.sku': 1 }, { sparse: true });
  db.dynamic_records.createIndex({ 'data.loan_amount': -1 }, { sparse: true });
  db.form_submissions.createIndex({ tenantId: 1, submittedAt: -1 });
  db.form_submissions.createIndex({ formId: 1 });
  db.form_submissions.createIndex({ 'data.email': 1 });
  db.enrichment_cache.createIndex({ tenantId: 1, cachedAt: -1 });
  db.enrichment_cache.createIndex({ 'linkedinUrl': 1 }, { sparse: true });
  db.conversation_meta.createIndex({ tenantId: 1, lastMessageAt: -1 });
  db.landing_pages.createIndex({ tenantId: 1, status: 1 });
  db.fb_pixel_events.createIndex({ pixelId: 1, eventTime: -1 });
  db.fb_pixel_events.createIndex({ matched: 1 });
} catch (e) {
  print('  Index warning: ' + e.message);
}

// ============================================================
// SECTION 11: NOTIFICATION_PREFERENCES EXTRA
// ============================================================
print('[11/12] notification_preferences (extra)...');

const notifTypes = ['lead_assigned','deal_won','deal_lost','lead_score_high','meeting_reminder','workflow_executed','payment_received','new_message','mention','document_signed','lead_bulk_import','integration_error','tenant_quota_warning','feature_release','security_alert','report_ready','lead_created','lead_updated','deal_created','deal_stage_changed'];
const channels2 = ['in_app','email','push','sms','webhook'];

tenants.forEach(t => {
  notifTypes.forEach(nt => {
    channels2.forEach(ch => {
      db.notification_preferences.updateOne(
        { tenantId: t.id, userId: 'a' + t.id.substring(0,7) + '-admin', notifType: nt, channel: ch },
        {
          $set: {
            tenantId: t.id,
            userId: 'a' + t.id.substring(0,7) + '-admin',
            notifType: nt,
            channel: ch,
            enabled: Math.random() > 0.2,
            quietStart: Math.random() > 0.7 ? 22 : null,
            quietEnd: Math.random() > 0.7 ? 7 : null,
            digestMode: ['none','daily','weekly'][Math.floor(Math.random() * 3)]
          }
        },
        { upsert: true }
      );
    });
  });
});
print('  notification_preferences (extra): ' + (tenants.length * notifTypes.length * channels2.length) + ' records');

// ============================================================
// SECTION 12: SYNC INDEXES FOR NEW COLLECTIONS
// ============================================================
print('[12/12] Final summary...');
print('');
print('=== MongoDB expansion seed (WS-B Loop 6) complete! ===');
print('  - dynamic_definitions (extra): ' + (tenants.length * extraEntityDefs.length) + ' records');
print('  - dynamic_records (extra): ' + drCount2 + ' records');
print('  - landing_pages (extra): ' + lpCount2 + ' records');
print('  - enrichment_cache (extra): ' + ecCount2 + ' records');
print('  - form_submissions (extra): 850 records');
print('  - fb_pixel_events (extra): 500 records');
print('  - conversation_meta (extra): ' + cmCount2 + ' records');
print('  - upload_temp (extra): 40 records');
print('  - dynamic_forms (extra): ' + dfCount2 + ' records');
print('  - notification_preferences (extra): ' + (tenants.length * notifTypes.length * channels2.length) + ' records');
