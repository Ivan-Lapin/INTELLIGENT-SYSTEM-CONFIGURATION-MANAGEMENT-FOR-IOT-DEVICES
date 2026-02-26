package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
)

func NewRouter(pg *pgxpool.Pool, mdb *mongo.Database) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	h := NewHandlers(pg, mdb)

	r.GET("/health", h.Health)
	r.POST("/v1/templates", h.CreateTemplate)
	r.GET("/v1/templates", h.ListTemplates)

	r.POST("/v1/versions", h.CreateVersion)
	r.GET("/v1/versions", h.ListVersions)

	return r
}
