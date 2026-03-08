package http

import (
	"telemetry-ingestor/internal/store"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	PG *store.PG
}

func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	h := NewHandlers(d)
	r.GET("/health", h.Health)
	r.GET("/v1/telemetry/:deviceId", h.GetRecentTelemetry)

	return r
}
