package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpx "telemetry-ingestor/internal/http"
	"telemetry-ingestor/internal/model"
	"telemetry-ingestor/internal/mqtt"
	"telemetry-ingestor/internal/store"

	paho "github.com/eclipse/paho.mqtt.golang"
)

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func main() {
	port := getenv("APP_PORT", "8085")
	pgDsn := getenv("POSTGRES_DSN", "postgres://pinchik:pass@localhost:5432/iot_core?sslmode=disable")
	mqttURL := getenv("MQTT_BROKER_URL", "tcp://mosquitto:1883")
	mqttClientID := getenv("MQTT_CLIENT_ID", "telemetry-ingestor")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := store.NewPG(ctx, pgDsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pg.Pool.Close()

	mc, err := mqtt.New(mqtt.Config{
		BrokerURL: mqttURL,
		ClientID:  mqttClientID,
	})
	if err != nil {
		log.Fatalf("mqtt: %v", err)
	}
	defer mc.Close()

	err = mc.Subscribe("telemetry/+/metrics", 0, func(_ paho.Client, msg paho.Message) {
		var ev model.TelemetryEvent
		if err := json.Unmarshal(msg.Payload(), &ev); err != nil {
			log.Printf("bad telemetry json topic=%s err=%v", msg.Topic(), err)
			return
		}

		if ev.TS.IsZero() {
			ev.TS = time.Now().UTC()
		}

		cctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := pg.InsertTelemetry(cctx, ev); err != nil {
			log.Printf("insert telemetry failed device=%s err=%v", ev.DeviceID, err)
			return
		}
	})

	if err != nil {
		log.Fatalf("subscribe telemetry: %v", err)
	}

	router := httpx.NewRouter(httpx.Deps{PG: pg})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("telemetry-ingestor listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
