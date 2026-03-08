package http

import (
	"mqtt-adapter/internal/store"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	PG    *store.PG
	Mongo *store.Mongo
	MQTT  MQTTPublisher
}

type MQTTPublisher interface {
	PublishJSON(topic string, qos byte, retain bool, payload any) error
}

func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	h := NewHandlers(d)

	r.GET("/health", h.Health)
	r.POST("/v1/publish/desired", h.PublishDesired)

	return r
}
