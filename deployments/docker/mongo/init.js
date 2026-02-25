db = db.getSiblingDB(process.env.MONGO_INITDB_DATABASE || "iot_configs");

db.createCollection("config_versions");
db.createCollection("config_templates");

db.config_versions.createIndex({ deviceType: 1, createdAt: -1 });
db.config_versions.createIndex({ versionId: 1 }, { unique: true });

db.config_templates.createIndex({ templateId: 1 }, { unique: true });