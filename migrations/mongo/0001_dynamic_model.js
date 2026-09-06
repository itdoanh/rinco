// ============================================================
// Mongo migration 0001 — dynamic model
// Applied via mongosh:
//   mongosh --host mongodb:27017 -u rinco -p <pwd> --authenticationDatabase admin \
//           mongodb://mongodb:27017/rinco < migrations/mongo/0001_dynamic_model.js
// ============================================================

const db = db.getSiblingDB('rinco');

print('Applying mongo migration 0001_dynamic_model.js');

// 1) Dynamic-record definitions (one document per (tenant, entity_code))
db.dynamic_definitions.createIndex(
  { tenantId: 1, entityCode: 1 },
  { unique: true, name: 'uq_tenant_entity' }
);
db.dynamic_definitions.createIndex({ tenantId: 1, isActive: 1 });

// 2) Dynamic records (one document per row)
db.dynamic_records.createIndex(
  { tenantId: 1, entityCode: 1, recordId: 1 },
  { unique: true, name: 'uq_tenant_entity_record' }
);
db.dynamic_records.createIndex({ tenantId: 1, entityCode: 1, updatedAt: -1 });
db.dynamic_records.createIndex({ tenantId: 1, ownerUserId: 1 });
db.dynamic_records.createIndex({ 'data.email': 1 });
db.dynamic_records.createIndex({ 'data.phone': 1 });
db.dynamic_records.createIndex(
  { createdAt: 1 },
  { expireAfterSeconds: 365 * 24 * 3600, name: 'ttl_created' }
);

// 3) Landing pages
db.landing_pages.createIndex(
  { tenantId: 1, slug: 1 },
  { unique: true, name: 'uq_tenant_slug' }
);
db.landing_pages.createIndex({ tenantId: 1, status: 1, publishedAt: -1 });
db.landing_pages.createIndex({ domain: 1 });

print('Mongo migration 0001_dynamic_model.js complete');
