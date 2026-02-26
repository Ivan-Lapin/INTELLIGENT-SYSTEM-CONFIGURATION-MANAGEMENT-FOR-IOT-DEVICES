package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"config-service/internal/db"
	httpx "config-service/internal/http"
)

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func main() {
	port := getenv("APP_PORT", "8082")
	pgDsn := getenv("POSTGRES_DSN", "postgres://iot:iot@localhost:5432/iot?sslmode=disable")

	mongoURI := getenv("MONGO_URI", "mongodb://root:root@localhost:27017")
	mongoDB := getenv("MONGO_DB", "iot_configs")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := db.NewPG(ctx, pgDsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pg.Close()

	m, err := db.NewMongo(ctx, mongoURI, mongoDB)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	defer func() { _ = m.Client.Disconnect(context.Background()) }()

	r := httpx.NewRouter(pg, m.DB)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("config-service listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
