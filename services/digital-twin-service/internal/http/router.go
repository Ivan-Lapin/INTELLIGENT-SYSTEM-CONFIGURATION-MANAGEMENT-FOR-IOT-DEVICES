package http

import (
	"digital-twin-service/internal/store"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type Deps struct {
	PG    *store.PG
	Mongo *store.Mongo
}

func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	h := NewHandlers(d)
	r.GET("/health", h.Health)
	r.POST("/v1/validate", h.Validate)
	r.POST("/v1/simulate", h.Simulate)

	_ = mongo.ErrNoDocuments
	return r
}
