// ============================================================
// RINCO MongoDB Comprehensive Demo Seed (Loop 202)
// ============================================================
// Run: mongosh --host localhost:27017 -u rinco -p rinco_dev_password
//      --authenticationDatabase admin --quiet rinco < seed/external/mongo/seed.js
// ============================================================

const db = db.getSiblingDB('rinco');

const tenants = [
  { id: 'aaaaaaaa-0000-0000-0000-000000000001', slug: 'apexfintech', name: 'Apex Fintech' },
  { id: 'aaaaaaaa-0000-0000-0000-000000000002', slug: 'hct-consulting', name: 'HCT Consulting' },
  { id: 'bbbbbbbb-0000-0000-0000-000000000003', slug: 'demo-company', name: 'Demo Company' },
];

const now = new Date();

// ============================================================
// SECTION 1: Dynamic Definitions
// ============================================================
print('[1/10] dynamic_definitions...');
tenants.forEach(t => {
  const definitions = [
    {
      tenantId: t.id,
      entityCode: 'real_estate',
      displayName: 'Bất động sản',
      icon: 'home',
      description: 'Danh sách bất động sản',
      fields: [
        { name: 'title', type: 'string', required: true },
        { name: 'price', type: 'float', required: true },
        { name: 'location', type: 'string', required: true },
        { name: 'bedrooms', type: 'int' },
        { name: 'area_sqm', type: 'float' },
        { name: 'property_type', type: 'enum', options: ['can_ho', 'shophouse', 'dat_nen', 'biet_thu'] },
        { name: 'status', type: 'enum', options: ['available', 'sold', 'reserved'] },
        { name: 'images', type: 'file', multiple: true }
      ],
      version: 1,
      isActive: true,
      createdAt: new Date(now - 60 * 24 * 3600 * 1000),
      updatedAt: now
    },
    {
      tenantId: t.id,
      entityCode: 'vehicle',
      displayName: 'Phương tiện',
      icon: 'car',
      description: 'Xe cộ cho vay',
      fields: [
        { name: 'make', type: 'string', required: true },
        { name: 'model', type: 'string', required: true },
        { name: 'year', type: 'int' },
        { name: 'price', type: 'float', required: true },
        { name: 'mileage', type: 'int' },
        { name: 'condition', type: 'enum', options: ['new', 'used', 'certified'] }
      ],
      version: 1,
      isActive: true,
      createdAt: new Date(now - 45 * 24 * 3600 * 1000),
      updatedAt: now
    }
  ];

  definitions.forEach(d => {
    db.dynamic_definitions.updateOne(
      { tenantId: d.tenantId, entityCode: d.entityCode },
      { $set: d },
      { upsert: true }
    );
  });
});
print('  dynamic_definitions: ' + tenants.length * 2 + ' records');

// ============================================================
// SECTION 2: Dynamic Records (50 per tenant = 150 total)
// ============================================================
print('[2/10] dynamic_records...');

const realEstateListings = [
  { title: 'Vinhomes Grand Park - 2PN', location: 'TP.HCM', price: 2500000000, bedrooms: 2, area_sqm: 68 },
  { title: 'Vinhomes Grand Park - 3PN', location: 'TP.HCM', price: 3200000000, bedrooms: 3, area_sqm: 88 },
  { title: 'Masterise Eco Smart City - 2PN', location: 'TP.HCM', price: 3800000000, bedrooms: 2, area_sqm: 75 },
  { title: 'Masterise Eco Smart City - 3PN', location: 'TP.HCM', price: 4800000000, bedrooms: 3, area_sqm: 95 },
  { title: 'Sun Group Long Biên - 2PN', location: 'Ha Noi', price: 5900000000, bedrooms: 2, area_sqm: 80 },
  { title: 'Sun Group Long Biên - 3PN', location: 'Ha Noi', price: 7200000000, bedrooms: 3, area_sqm: 105 },
  { title: 'The Matrix One - 4PN', location: 'Ha Noi', price: 12500000000, bedrooms: 4, area_sqm: 130 },
  { title: 'Imperia Smart City - 1PN', location: 'Ha Noi', price: 1900000000, bedrooms: 1, area_sqm: 45 },
  { title: 'Imperia Smart City - 2PN', location: 'Ha Noi', price: 2800000000, bedrooms: 2, area_sqm: 65 },
  { title: 'Vinhomes Ocean Park - 3PN', location: 'Ha Noi', price: 4200000000, bedrooms: 3, area_sqm: 88 },
  { title: 'Vinhomes Ocean Park - 2PN', location: 'Ha Noi', price: 3500000000, bedrooms: 2, area_sqm: 72 },
  { title: 'Vinhomes Symphony - 2PN', location: 'Ha Noi', price: 3600000000, bedrooms: 2, area_sqm: 75 },
  { title: 'Vinhomes Symphony - 3PN', location: 'Ha Noi', price: 4800000000, bedrooms: 3, area_sqm: 98 },
  { title: 'The Maris - 2PN', location: 'Da Nang', price: 2900000000, bedrooms: 2, area_sqm: 70 },
  { title: 'The Maris - 3PN', location: 'Da Nang', price: 3800000000, bedrooms: 3, area_sqm: 92 },
  { title: 'Apec Mandala - Studio', location: 'Da Nang', price: 1500000000, bedrooms: 0, area_sqm: 38 },
  { title: 'Apec Mandala - 1PN', location: 'Da Nang', price: 2200000000, bedrooms: 1, area_sqm: 52 },
  { title: 'Aqua City - 2PN', location: 'Dong Nai', price: 2700000000, bedrooms: 2, area_sqm: 72 },
  { title: 'Aqua City - 3PN', location: 'Dong Nai', price: 3600000000, bedrooms: 3, area_sqm: 95 },
  { title: 'Shophouse Vinhomes - 3 tầng', location: 'TP.HCM', price: 8500000000, bedrooms: 3, area_sqm: 150 },
  { title: 'Dat nen Long Thuan My', location: 'Dong Nai', price: 4500000000, bedrooms: 0, area_sqm: 200 },
  { title: 'Biet thu Masterise - 5PN', location: 'TP.HCM', price: 25000000000, bedrooms: 5, area_sqm: 350 },
  { title: 'Can ho ican - 2PN', location: 'TP.HCM', price: 3200000000, bedrooms: 2, area_sqm: 70 },
  { title: 'Quan 9 - Dat nen 100m2', location: 'TP.HCM', price: 3800000000, bedrooms: 0, area_sqm: 100 },
  { title: 'Sai Gon Penthouse - 4PN', location: 'TP.HCM', price: 18000000000, bedrooms: 4, area_sqm: 250 }
];

let drCount = 0;
tenants.forEach(t => {
  realEstateListings.forEach((listing, idx) => {
    db.dynamic_records.updateOne(
      { tenantId: t.id, entityCode: 'real_estate', recordId: t.slug + '_prop_' + (idx + 1) },
      {
        $set: {
          tenantId: t.id,
          entityCode: 'real_estate',
          recordId: t.slug + '_prop_' + (idx + 1),
          ownerUserId: null,
          data: {
            ...listing,
            description: 'Căn hộ ' + listing.title + ' — đầy đủ nội thất, view đẹp, gần trường học và bệnh viện',
            status: idx % 5 === 0 ? 'reserved' : 'available',
            images: [
              'https://placeholder.com/img/' + t.slug + '/prop_' + (idx + 1) + '_1.jpg',
              'https://placeholder.com/img/' + t.slug + '/prop_' + (idx + 1) + '_2.jpg'
            ],
            publishedAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000),
            agentContact: '+8490' + (8000000 + Math.floor(Math.random() * 999999)).toString().padStart(7, '0')
          },
          createdAt: new Date(now - Math.random() * 60 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    drCount++;
  });
});
print('  dynamic_records: ' + drCount + ' records');

// ============================================================
// SECTION 3: Landing Pages (Mongo mirror)
// ============================================================
print('[3/10] landing_pages...');
let lpCount = 0;
tenants.forEach(t => {
  const pages = [
    { slug: 'home', title: 'Trang chủ ' + t.name, status: 'published' },
    { slug: 'loan-application', title: 'Đăng ký khoản vay', status: 'published' },
    { slug: 'contact', title: 'Liên hệ', status: 'published' },
    { slug: 'promo-spring', title: 'Khuyến mãi mùa xuân', status: 'draft' }
  ];

  pages.forEach((page, idx) => {
    db.landing_pages.updateOne(
      { tenantId: t.id, slug: page.slug },
      {
        $set: {
          tenantId: t.id,
          slug: page.slug,
          title: page.title,
          blocks: [
            { type: 'hero', title: 'Welcome to ' + t.name, cta: 'Get started' },
            { type: 'features', items: 3 },
            { type: 'form', fields: ['Name', 'Phone', 'Email'] }
          ],
          status: page.status,
          publishedAt: page.status === 'published' ? new Date(now - 30 * 24 * 3600 * 1000) : null,
          version: 1,
          createdAt: new Date(now - 60 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    lpCount++;
  });
});
print('  landing_pages: ' + lpCount + ' records');

// ============================================================
// SECTION 4: Enrichment Cache (100 records)
// ============================================================
print('[4/10] enrichment_cache...');
const sources = ['facebook', 'linkedin', 'clearbit', 'pipl', 'hunter'];
const firstNames = ['Nguyen', 'Tran', 'Le', 'Pham', 'Hoang', 'Vu', 'Dang', 'Bui', 'Do', 'Ngo'];
const lastNames = ['Van An', 'Thi Binh', 'Hoang Cuong', 'Thi Dung', 'Van Em', 'Thi Phuong', 'Van Giang', 'Thi Hoa', 'Van Ich', 'Thi Khanh'];
const companies = ['FPT', 'Viettel', 'VNG', 'VinGroup', 'Sun Group', 'Masterise', 'VPBank', 'Techcombank'];
const titles = ['CEO', 'CTO', 'CFO', 'Marketing Manager', 'Sales Lead', 'Product Manager'];

let ecCount = 0;
for (let i = 0; i < 100; i++) {
  const fn = firstNames[Math.floor(Math.random() * firstNames.length)];
  const ln = lastNames[Math.floor(Math.random() * lastNames.length)];
  const t = tenants[Math.floor(Math.random() * tenants.length)];
  const email = fn.toLowerCase() + '.' + ln.toLowerCase().replace(/\s/g, '') + '@example.com';

  db.enrichment_cache.updateOne(
    { tenantId: t.id, email: email },
    {
      $set: {
        tenantId: t.id,
        email: email,
        phone: '+8490' + (8000000 + Math.floor(Math.random() * 999999)).toString().padStart(7, '0'),
        fullName: fn + ' ' + ln,
        company: companies[Math.floor(Math.random() * companies.length)],
        jobTitle: titles[Math.floor(Math.random() * titles.length)],
        linkedinUrl: 'https://linkedin.com/in/' + fn.toLowerCase() + '-' + ln.toLowerCase().replace(/\s/g, ''),
        facebookUrl: 'https://facebook.com/' + fn.toLowerCase() + '.' + ln.toLowerCase().replace(/\s/g, ''),
        location: ['Ha Noi', 'Ho Chi Minh', 'Da Nang', 'Can Tho'][Math.floor(Math.random() * 4)],
        source: sources[Math.floor(Math.random() * sources.length)],
        confidence: 0.7 + Math.random() * 0.3,
        cachedAt: now,
        expiresAt: new Date(now.getTime() + 30 * 24 * 3600 * 1000),
        createdAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000)
      }
    },
    { upsert: true }
  );
  ecCount++;
}
print('  enrichment_cache: ' + ecCount + ' records');

// ============================================================
// SECTION 5: Dynamic Forms (5 per tenant)
// ============================================================
print('[5/10] dynamic_forms...');
const formTemplates = [
  {
    name: 'Hero Contact Form',
    fields: [
      { name: 'full_name', type: 'text', required: true },
      { name: 'phone', type: 'phone', required: true },
      { name: 'email', type: 'email' },
      { name: 'loan_amount', type: 'number', label: 'Số tiền muốn vay' }
    ]
  },
  {
    name: 'Quick Apply Form',
    fields: [
      { name: 'phone', type: 'phone', required: true, label: 'Số điện thoại' }
    ]
  },
  {
    name: 'Newsletter Form',
    fields: [
      { name: 'email', type: 'email', required: true }
    ]
  },
  {
    name: 'Property Inquiry Form',
    fields: [
      { name: 'full_name', type: 'text', required: true },
      { name: 'phone', type: 'phone', required: true },
      { name: 'budget', type: 'number', label: 'Ngân sách (VNĐ)' },
      { name: 'preferred_location', type: 'select', options: ['HCM', 'HN', 'DN'] }
    ]
  },
  {
    name: 'Demo Booking Form',
    fields: [
      { name: 'full_name', type: 'text', required: true },
      { name: 'email', type: 'email', required: true },
      { name: 'company', type: 'text' },
      { name: 'preferred_date', type: 'date' }
    ]
  }
];

let dfCount = 0;
tenants.forEach(t => {
  formTemplates.forEach(ft => {
    db.dynamic_forms.updateOne(
      { tenantId: t.id, name: ft.name },
      {
        $set: {
          tenantId: t.id,
          name: ft.name,
          fields: ft.fields,
          isActive: true,
          submitAction: 'webhook',
          webhookUrl: 'https://api.' + t.slug + '.app/leads/intake',
          successMessage: 'Cảm ơn bạn đã để lại thông tin!',
          createdAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    dfCount++;
  });
});
print('  dynamic_forms: ' + dfCount + ' records');

// ============================================================
// SECTION 6: FB Pixel Events (500 records)
// ============================================================
print('[6/10] fb_pixel_events...');
const fbEvents = ['PageView', 'Lead', 'CompleteRegistration', 'Purchase', 'InitiateCheckout', 'ViewContent'];
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
    fbclid: Math.random() < 0.6 ? 'fb.' + Math.random().toString(36).substring(2, 18) : null,
    value: ev === 'Purchase' ? Math.floor(1000000 + Math.random() * 100000000) : null,
    currency: 'VND',
    sourceUrl: 'https://' + t.slug + '.rinco.app/landing/' + (Math.random() < 0.5 ? 'home' : 'loan'),
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/127.0',
    ip: '203.0.113.' + Math.floor(10 + Math.random() * 200),
    sentToMeta: Math.random() < 0.7,
    sentAt: Math.random() < 0.7 ? now : null,
    receivedAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000)
  });

  if (fbBatch.length >= 100) {
    db.fb_pixel_events.insertMany(fbBatch);
    fbBatch = [];
  }
}
if (fbBatch.length > 0) {
  db.fb_pixel_events.insertMany(fbBatch);
}
print('  fb_pixel_events: 500 records');

// ============================================================
// SECTION 7: Conversation Meta (9 records = 3 per tenant)
// ============================================================
print('[7/10] conversation_meta...');
let cmCount = 0;
tenants.forEach(t => {
  ['General', 'Sales', 'Support'].forEach((title, i) => {
    db.conversation_meta.updateOne(
      { tenantId: t.id, title: title },
      {
        $set: {
          tenantId: t.id,
          conversationId: UUID().toString(),
          title: title,
          type: i === 0 ? 'public' : 'private',
          memberCount: Math.floor(5 + Math.random() * 15),
          lastMessageAt: new Date(now - Math.random() * 4 * 3600 * 1000),
          lastMessagePreview: 'Sample message for ' + title,
          createdAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000),
          updatedAt: now
        }
      },
      { upsert: true }
    );
    cmCount++;
  });
});
print('  conversation_meta: ' + cmCount + ' records');

// ============================================================
// SECTION 8: Upload Temp (10 records)
// ============================================================
print('[8/10] upload_temp...');
let tempBatch = [];
for (let i = 0; i < 10; i++) {
  const t = tenants[Math.floor(Math.random() * tenants.length)];
  tempBatch.push({
    tempId: UUID().toString(),
    tenantId: t.id,
    filename: 'upload_' + i + (i % 2 === 0 ? '.pdf' : '.jpg'),
    mimeType: i % 2 === 0 ? 'application/pdf' : 'image/jpeg',
    sizeBytes: Math.floor(100000 + Math.random() * 5000000),
    s3Key: 'temp/' + i + '/' + Math.random().toString(36).substring(2, 10),
    bucket: 'rinco-temp',
    uploadedBy: 'a000000' + (1 + Math.floor(Math.random() * 3)) + '-0000-0000-0000-00000000000' + (10 + Math.floor(Math.random() * 15)).toString().padStart(2, '0'),
    status: 'pending',
    expiresAt: new Date(now.getTime() + 24 * 3600 * 1000),
    createdAt: now
  });
}
db.upload_temp.insertMany(tempBatch);
print('  upload_temp: 10 records');

// ============================================================
// SECTION 9: Form Submissions (50 records per tenant)
// ============================================================
print('[9/10] form_submissions...');
const formIds = ['hero-contact', 'quick-apply', 'newsletter', 'property-inquiry', 'demo-booking'];
let fsBatch = [];

for (let i = 0; i < 150; i++) {
  const t = tenants[Math.floor(Math.random() * tenants.length)];
  fsBatch.push({
    tenantId: t.id,
    formId: formIds[Math.floor(Math.random() * formIds.length)],
    submittedAt: new Date(now - Math.random() * 30 * 24 * 3600 * 1000),
    ipAddress: '203.0.113.' + Math.floor(10 + Math.random() * 200),
    userAgent: 'Mozilla/5.0',
    data: {
      full_name: 'Người dùng ' + (i + 1),
      phone: '+8490' + (8000000 + Math.floor(Math.random() * 999999)).toString().padStart(7, '0'),
      email: 'user' + (i + 1) + '@example.com'
    },
    meta: {},
    status: 'processed'
  });

  if (fsBatch.length >= 50) {
    db.form_submissions.insertMany(fsBatch);
    fsBatch = [];
  }
}
if (fsBatch.length > 0) {
  db.form_submissions.insertMany(fsBatch);
}
print('  form_submissions: 150 records');

// ============================================================
// SECTION 10: Create Indexes
// ============================================================
print('[10/10] Creating indexes...');
try {
  db.dynamic_records.createIndex({ tenantId: 1, entityCode: 1, recordId: 1 }, { unique: true, name: 'uq_tenant_entity_record' });
  db.dynamic_records.createIndex({ tenantId: 1, entityCode: 1, updatedAt: -1 });
  db.dynamic_records.createIndex({ tenantId: 1, ownerUserId: 1 });
  db.dynamic_records.createIndex({ 'data.email': 1 });
  db.dynamic_records.createIndex({ 'data.phone': 1 });
} catch (e) {
  print('  Index creation warning: ' + e.message);
}

try {
  db.enrichment_cache.createIndex({ tenantId: 1, email: 1 }, { unique: true, name: 'uq_tenant_email' });
} catch (e) {
  print('  Index warning: ' + e.message);
}

try {
  db.fb_pixel_events.createIndex({ tenantId: 1, eventTime: -1 });
  db.fb_pixel_events.createIndex({ tenantId: 1, eventName: 1, eventTime: -1 });
} catch (e) {
  print('  Index warning: ' + e.message);
}

print('');

// ============================================================
// SUMMARY
// ============================================================
print('=== MongoDB comprehensive seed complete! ===');
print('');
print('Summary:');
print('  - dynamic_definitions: ' + tenants.length * 2 + ' records');
print('  - dynamic_records: ' + drCount + ' records (real estate listings)');
print('  - landing_pages: ' + lpCount + ' records');
print('  - enrichment_cache: ' + ecCount + ' records');
print('  - dynamic_forms: ' + dfCount + ' records');
print('  - fb_pixel_events: 500 records');
print('  - conversation_meta: ' + cmCount + ' records');
print('  - upload_temp: 10 records');
print('  - form_submissions: 150 records');
print('');
print('Total: ~' + (tenants.length * 2 + drCount + lpCount + ecCount + dfCount + 500 + cmCount + 10 + 150) + ' documents');
