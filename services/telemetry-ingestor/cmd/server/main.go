package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpx "telemetry-ingestor/internal/http"
	"telemetry-ingestor/internal/ingest"
	"telemetry-ingestor/internal/mqtt"
	"telemetry-ingestor/internal/store"
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

	consumer := ingest.NewConsumer(pg)

	if err := consumer.SubscribeTelemetry(mc, "telemetry/+/metrics"); err != nil {
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

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
