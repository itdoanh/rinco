// ============================================================
// RINCO MongoDB Init
// ============================================================

// Switch to rinco database
db = db.getSiblingDB('rinco');

db.createUser({
  user: 'rinco_app',
  pwd: 'rinco_dev_password',
  roles: [{ role: 'readWrite', db: 'rinco' }],
});

// Create collections + indexes
// Lead enrichment cache
db.createCollection('enrichment_cache', { capped: false });
db.enrichment_cache.createIndex({ tenantId: 1, email: 1 }, { unique: true });
db.enrichment_cache.createIndex({ tenantId: 1, phone: 1 });
db.enrichment_cache.createIndex({ createdAt: 1 }, { expireAfterSeconds: 2592000 }); // 30 days TTL

// Lead rich data
db.createCollection('leads_rich');
db.leads_rich.createIndex({ tenantId: 1, score: -1 });
db.leads_rich.createIndex({ tenantId: 1, stage: 1, score: -1 });
db.leads_rich.createIndex({ tenantId: 1, 'enrichmentData.company': 1 });
db.leads_rich.createIndex({ tenantId: 1, createdAt: -1 });
db.leads_rich.createIndex({ email: 1 });
db.leads_rich.createIndex({ phone: 1 });

// FB pixel events (browser-side events from tracking SDK)
db.createCollection('fb_pixel_events');
db.fb_pixel_events.createIndex({ tenantId: 1, eventId: 1 }, { unique: true });
db.fb_pixel_events.createIndex({ tenantId: 1, eventName: 1, createdAt: -1 });
db.fb_pixel_events.createIndex({ tenantId: 1, fbclid: 1 });
db.fb_pixel_events.createIndex({ createdAt: 1 }, { expireAfterSeconds: 90 * 86400 });

// Form submissions (landing page forms)
db.createCollection('form_submissions');
db.form_submissions.createIndex({ tenantId: 1, formId: 1 });
db.form_submissions.createIndex({ tenantId: 1, createdAt: -1 });
db.form_submissions.createIndex({ ipAddress: 1, createdAt: 1 });

// AI prediction history
db.createCollection('ai_predictions');
db.ai_predictions.createIndex({ tenantId: 1, leadId: 1, createdAt: -1 });
db.ai_predictions.createIndex({ tenantId: 1, modelType: 1, createdAt: -1 });
db.ai_predictions.createIndex({ createdAt: 1 }, { expireAfterSeconds: 180 * 86400 });

// Sessions / browser sessions
db.createCollection('sessions');
db.sessions.createIndex({ sessionId: 1 }, { unique: true });
db.sessions.createIndex({ userId: 1 });
db.sessions.createIndex({ lastActivityAt: 1 }, { expireAfterSeconds: 7 * 86400 });

print('MongoDB initialization complete');
