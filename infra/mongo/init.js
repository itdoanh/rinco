// ============================================================
// RINCO MongoDB initialization
// ============================================================
// Triggered once by the official mongo:7 image (entrypoint-initdb.d).
// Connects via the primary of the replica set; if the container is
// single-node, will attempt rs.initiate() so writes work as PRIMARY.
// ============================================================

// Make sure we are operating on the right DB. The image has already
// authenticated as MONGO_INITDB_ROOT_USERNAME so we have full rights.
db = db.getSiblingDB('rinco');

print('RINCO mongo init starting on db=rinco');

// ---------- Replica-set bootstrap (idempotent) ----------
try {
  const status = rs.status();
  print('Replica set already initiated: ' + status.set);
} catch (e) {
  print('Replica set not initiated yet, attempting rs.initiate() ...');
  try {
    rs.initiate({
      _id: 'rs0',
      members: [{ _id: 0, host: 'mongodb:27017' }]
    });
  } catch (initErr) {
    print('rs.initiate error (usually fine if already initiated): ' + initErr);
  }
}

// ---------- Application role ----------
try {
  db.createUser({
    user: 'rinco_app',
    pwd: 'rinco_dev_password',
    roles: [{ role: 'readWrite', db: 'rinco' }]
  });
} catch (err) {
  print('app role may already exist: ' + err.message);
}

// ---------- Collections + indexes ----------

// 1) Dynamic model — generic record store; one collection per entity_code
db.createCollection('dynamic_definitions');
db.dynamic_definitions.createIndex(
  { tenantId: 1, entityCode: 1 },
  { unique: true, name: 'uq_tenant_entity' }
);
db.dynamic_definitions.createIndex({ tenantId: 1, isActive: 1 });
db.dynamic_definitions.createIndex({ updatedAt: -1 });

db.createCollection('dynamic_records');
db.dynamic_records.createIndex(
  { tenantId: 1, entityCode: 1, 'recordId': 1 },
  { unique: true, name: 'uq_tenant_entity_record' }
);
db.dynamic_records.createIndex({ tenantId: 1, entityCode: 1, updatedAt: -1 });
db.dynamic_records.createIndex({ tenantId: 1, ownerUserId: 1 });
db.dynamic_records.createIndex({ 'data.email': 1 });
db.dynamic_records.createIndex({ 'data.phone': 1 });
db.dynamic_records.createIndex({ createdAt: 1 }, { expireAfterSeconds: 365 * 24 * 3600 });

// 2) Landing CMS — pages, blocks, sections
db.createCollection('landing_pages');
db.landing_pages.createIndex(
  { tenantId: 1, slug: 1 },
  { unique: true, name: 'uq_tenant_slug' }
);
db.landing_pages.createIndex({ tenantId: 1, status: 1, publishedAt: -1 });
db.landing_pages.createIndex({ domain: 1 });

// 3) Form submissions (landing)
db.createCollection('form_submissions');
db.form_submissions.createIndex({ tenantId: 1, formId: 1, submittedAt: -1 });
db.form_submissions.createIndex({ tenantId: 1, status: 1 });
db.form_submissions.createIndex({ 'data.email': 1 });
db.form_submissions.createIndex({ 'data.phone': 1 });
db.form_submissions.createIndex({ submittedAt: 1 }, { expireAfterSeconds: 365 * 86400 });

// 4) Lead enrichment cache
db.createCollection('enrichment_cache');
db.enrichment_cache.createIndex(
  { tenantId: 1, email: 1 },
  { unique: true, name: 'uq_tenant_email' }
);
db.enrichment_cache.createIndex({ tenantId: 1, phone: 1 });
db.enrichment_cache.createIndex(
  { cachedAt: 1 },
  { expireAfterSeconds: 30 * 86400 }
);

// 5) Notification preferences
db.createCollection('notification_prefs');
db.notification_prefs.createIndex(
  { tenantId: 1, userId: 1, channel: 1 },
  { unique: true, name: 'uq_user_channel' }
);

// 6) Webhook payloads (debug)
db.createCollection('webhook_payloads');
db.webhook_payloads.createIndex({ tenantId: 1, source: 1 });
db.webhook_payloads.createIndex(
  { receivedAt: 1 },
  { expireAfterSeconds: 7 * 86400 }
);

// 7) Audit mirror (Postgres is primary, Mongo caches recent)
db.createCollection('audit_recent');
db.audit_recent.createIndex({ tenantId: 1, createdAt: -1 });
db.audit_recent.createIndex({ actorId: 1, createdAt: -1 });
db.audit_recent.createIndex({ createdAt: 1 }, { expireAfterSeconds: 90 * 86400 });

print('RINCO mongo init complete');
