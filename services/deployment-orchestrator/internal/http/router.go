package http

import (
	"deployment-orchestrator/internal/store"
	"deployment-orchestrator/internal/worker"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	PG *store.PG
	R  *worker.Runner
}

func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	h := NewHandlers(d)
	r.GET("/health", h.Health)
	r.POST("/v1/deployments", h.CreateDeployment)
	r.GET("/v1/deployments/:id", h.GetDeployment)

	return r
}
