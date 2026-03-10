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

	httpx "lwm2m-adapter/internal/http"
	"lwm2m-adapter/internal/model"
	"lwm2m-adapter/internal/mqtt"
	"lwm2m-adapter/internal/store"

	paho "github.com/eclipse/paho.mqtt.golang"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func main() {
	appPort := getenv("APP_PORT", "8088")
	pgDsn := getenv("POSTGRES_DSN", "postgres://pinchik:pass_iot_configs@postgres:5432/iot_core?sslmode=disable")
	mongoURI := getenv("MONGO_URI", "mongodb://pinchik:pass_iot_configs@mongo:27017")
	mongoDB := getenv("MONGO_DB", "iot_configs")
	mqttURL := getenv("MQTT_BROKER_URL", "tcp://mosquitto:1883")
	mqttClientID := getenv("MQTT_CLIENT_ID", "lwm2m-adapter")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := store.NewPG(ctx, pgDsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pg.Pool.Close()

	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	defer func() { _ = mc.Disconnect(context.Background()) }()
	mdb := mc.Database(mongoDB)
	mstore := store.NewMongo(mdb)

	mq, err := mqtt.New(mqtt.Config{
		BrokerURL: mqttURL,
		ClientID:  mqttClientID,
	})
	if err != nil {
		log.Fatalf("mqtt: %v", err)
	}
	defer mq.Close()

	err = mq.Subscribe("lwm2m/ack/+", 1, func(_ paho.Client, msg paho.Message) {
		var ack model.AckMessage
		if err := json.Unmarshal(msg.Payload(), &ack); err != nil {
			log.Printf("bad ack json: %v", err)
			return
		}

		cctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		status := "failed"
		if ack.Status == "APPLIED" {
			status = "applied"
		}

		_ = pg.InsertApplyLogResult(cctx, ack.DeviceID, ack.VersionID, status, ack.Error, ack.TS)
		_ = pg.UpsertAssignment(cctx, "device", ack.DeviceID, ack.VersionID, status)
	})

	if err != nil {
		log.Fatalf("subscribe ack: %v", err)
	}

	err = mq.Subscribe("lwm2m/reported/+", 0, func(_ paho.Client, msg paho.Message) {
		log.Printf("reported topic=%s payload=%s", msg.Topic(), string(msg.Payload()))
	})
	if err != nil {
		log.Fatalf("subscribe reported: %v", err)
	}

	router := httpx.NewRouter(httpx.Deps{
		PG:    pg,
		Mongo: mstore,
		MQTT:  mq,
	})

	srv := &http.Server{
		Addr:              ":" + appPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("lwm2m-adapter listening on :%s", appPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
