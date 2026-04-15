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
	r.GET("/v1/telemetry/:deviceId/aggregations", h.GetTelemetryAggregations)

	r.POST("/v1/anomalies", h.CreateAnomalyEvent)
	r.GET("/v1/anomalies/:deviceId", h.GetRecentAnomalies)

	return r
}
