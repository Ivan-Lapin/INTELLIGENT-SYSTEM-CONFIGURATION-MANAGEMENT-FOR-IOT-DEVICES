package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpx "deployment-orchestrator/internal/http"
	"deployment-orchestrator/internal/store"
	"deployment-orchestrator/internal/worker"
)

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func main() {
	port := getenv("APP_PORT", "8084")
	pgDsn := getenv("POSTGRES_DSN", "postgres://pinchik:pass@localhost:5432/iot_core?sslmode=disable")
	mqttAdapterURL := getenv("MQTT_ADAPTER_URL", "http://mqtt-adapter:8083")
	mlServiceURL := getenv("ML_SERVICE_URL", "http://ml-service:8086")
	twinServiceURL := getenv("TWIN_SERVICE_URL", "http://digital-twin-service:8087")
	lwm2mAdapterURL := getenv("LWM2M_ADAPTER_URL", "http://lwm2m-adapter:8088")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := store.NewPG(ctx, pgDsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pg.Pool.Close()

	runner := worker.NewRunner(pg, mqttAdapterURL, lwm2mAdapterURL, mlServiceURL, twinServiceURL)
	router := httpx.NewRouter(httpx.Deps{PG: pg, R: runner})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("deployment-orchestrator listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
