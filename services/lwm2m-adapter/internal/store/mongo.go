package store

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Mongo struct {
	DB *mongo.Database
}

func NewMongo(db *mongo.Database) *Mongo { return &Mongo{DB: db} }

type MongoConfigVersionDoc struct {
	Payload  map[string]any `bson:"payload"`
	Checksum string         `bson:"checksum"`
}

func (m *Mongo) GetConfigPayloadByMongoID(ctx context.Context, id string) (MongoConfigVersionDoc, error) {
	var out MongoConfigVersionDoc
	err := m.DB.Collection("config_versions").
		FindOne(ctx, bson.M{"_id": id}).
		Decode(&out)
	return out, err
}
