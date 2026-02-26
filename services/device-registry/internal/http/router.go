package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pg *pgxpool.Pool) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	h := NewHandlers(pg)

	r.GET("/health", h.Health)
	r.POST("/v1/devices", h.CreateDevice)
	r.GET("/v1/devices", h.ListDevices)

	return r
}
