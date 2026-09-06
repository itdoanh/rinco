// MongoDB demo data fixture for development
db = db.getSiblingDB('rinco');

const now = new Date();

db.dynamic_definitions.insertMany([
  {
    tenantId: '11111111-1111-1111-1111-111111111111',
    entityCode: 'real_estate',
    displayName: 'Bất động sản',
    icon: 'home',
    isActive: true,
    version: 1,
    createdAt: now,
    updatedAt: now
  },
  {
    tenantId: '11111111-1111-1111-1111-111111111111',
    entityCode: 'real_estate_project',
    displayName: 'Dự án bất động sản',
    icon: 'building',
    isActive: true,
    version: 1,
    createdAt: now,
    updatedAt: now
  }
]);

print('Mongo demo fixtures inserted.');
