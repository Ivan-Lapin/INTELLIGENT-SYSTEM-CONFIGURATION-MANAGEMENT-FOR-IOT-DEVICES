package ingest

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"telemetry-ingestor/internal/model"
	"telemetry-ingestor/internal/mqtt"
	"telemetry-ingestor/internal/store"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Consumer struct {
	pg *store.PG
}

func NewConsumer(pg *store.PG) *Consumer {
	return &Consumer{pg: pg}
}

func (c *Consumer) SubscribeTelemetry(m *mqtt.Client, topic string) error {
	return m.Subscribe(topic, 1, func(_ paho.Client, msg paho.Message) {
		c.handleTelemetryMessage(msg.Payload())
	})
}

func (c *Consumer) handleTelemetryMessage(payload []byte) {
	var ev model.TelemetryEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		log.Printf("invalid telemetry payload: %v", err)
		return
	}

	if ev.DeviceID == "" {
		log.Printf("invalid telemetry payload: empty deviceId")
		return
	}

	if ev.TS.IsZero() {
		ev.TS = time.Now().UTC()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.pg.InsertTelemetry(ctx, ev); err != nil {
		log.Printf("insert telemetry failed: %v", err)
		return
	}

	anomalies := DetectAnomalies(ev)
	for _, anomaly := range anomalies {
		if anomaly.TS.IsZero() {
			anomaly.TS = time.Now().UTC()
		}
		if err := c.pg.InsertAnomalyEvent(ctx, anomaly); err != nil {
			log.Printf("insert anomaly failed: %v", err)
		}
	}
}
