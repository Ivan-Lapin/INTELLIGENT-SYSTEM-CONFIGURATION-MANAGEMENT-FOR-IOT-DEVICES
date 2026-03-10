package store

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Mongo struct {
	DB *mongo.Database
}

func NewMongo(db *mongo.Database) *Mongo {
	return &Mongo{DB: db}
}

type ConfigVersionDoc struct {
	ID         string         `bson:"_id"`
	TemplateID string         `bson:"templateId"`
	Version    int            `bson:"version"`
	Payload    map[string]any `bson:"payload"`
	Checksum   string         `bson:"checksum"`
}

func (m *Mongo) GetConfigVersion(ctx context.Context, docID string) (*ConfigVersionDoc, error) {
	var out ConfigVersionDoc
	err := m.DB.Collection("config_versions").
		FindOne(ctx, bson.M{"_id": docID}).
		Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
