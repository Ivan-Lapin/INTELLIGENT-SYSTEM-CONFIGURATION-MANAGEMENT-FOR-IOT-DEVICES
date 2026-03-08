
db = db.getSiblingDB(process.env.MONGO_INITDB_DATABASE || "iot_configs");

if (!db.getCollectionNames().includes("config_templates")) {
  db.createCollection("config_templates");
}
if (!db.getCollectionNames().includes("config_versions")) {
  db.createCollection("config_versions");
}

// Индексы для templates
db.config_templates.createIndex({ deviceType: 1, createdAt: -1 });
db.config_templates.createIndex({ name: 1, deviceType: 1 }, { unique: true });

// Индексы для versions
db.config_versions.createIndex({ templateId: 1, version: -1 });
db.config_versions.createIndex({ createdAt: -1 });
db.config_versions.createIndex({ checksum: 1 });
