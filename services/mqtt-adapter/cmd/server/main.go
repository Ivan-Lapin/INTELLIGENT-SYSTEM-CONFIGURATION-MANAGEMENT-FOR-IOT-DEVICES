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

	httpx "mqtt-adapter/internal/http"
	"mqtt-adapter/internal/model"
	"mqtt-adapter/internal/mqtt"
	"mqtt-adapter/internal/store"

	mqttPaho "github.com/eclipse/paho.mqtt.golang"

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
	appPort := getenv("APP_PORT", "8083")
	pgDsn := getenv("POSTGRES_DSN", "postgres://iot:iot@localhost:5432/iot?sslmode=disable")
	mongoURI := getenv("MONGO_URI", "mongodb://root:root@localhost:27017")
	mongoDB := getenv("MONGO_DB", "iot_configs")
	mqttURL := getenv("MQTT_BROKER_URL", "tcp://mosquitto:1883")
	mqttClientID := getenv("MQTT_CLIENT_ID", "mqtt-adapter")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := store.NewPG(ctx, pgDsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pg.Pool.Close()

	// mongo
	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	defer func() { _ = mc.Disconnect(context.Background()) }()
	mdb := mc.Database(mongoDB)
	mstore := store.NewMongo(mdb)

	mcMqtt, err := mqtt.New(mqtt.Config{BrokerURL: mqttURL, ClientID: mqttClientID})
	if err != nil {
		log.Fatalf("mqtt: %v", err)
	}
	defer mcMqtt.Close()

	// subscribe ACK
	err = mcMqtt.Subscribe("config/ack/+", 1, func(_ mqttPaho.Client, msg mqttPaho.Message) {
		var ack model.AckMessage
		if err := json.Unmarshal(msg.Payload(), &ack); err != nil {
			log.Printf("bad ack json: %v", err)
			return
		}

		if ack.ConfigVersionID == "" {
			log.Printf("ack missing configVersionId for device=%s", ack.DeviceID)
			return
		}

		cctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		status := "failed"
		if ack.Status == "APPLIED" {
			status = "applied"
		}

		_ = pg.InsertApplyLogResult(cctx, ack.DeviceID, ack.ConfigVersionID, status, ack.Error, ack.TS)
		_ = pg.UpsertAssignment(cctx, "device", ack.DeviceID, ack.ConfigVersionID, status)
	})

	// subscribe REPORTED
	_ = mcMqtt.Subscribe("state/reported/+", 0, func(_ mqttPaho.Client, msg mqttPaho.Message) {
		var reported model.ReportedMessage
		if err := json.Unmarshal(msg.Payload(), &reported); err != nil {
			log.Printf("bad reported json: %v", err)
			return
		}

		log.Printf("reported device=%s version=%d state=%v", reported.DeviceID, reported.Version, reported.State)
	})

	_ = mcMqtt.Subscribe("telemetry/+", 0, func(_ mqttPaho.Client, msg mqttPaho.Message) {
		var telemetry model.TelemetryMessage
		if err := json.Unmarshal(msg.Payload(), &telemetry); err != nil {
			log.Printf("bad telemetry json topic=%s err=%v", msg.Topic(), err)
			return
		}

		log.Printf("telemetry device=%s version=%d metrics=%v", telemetry.DeviceID, telemetry.Version, telemetry.Metrics)
	})

	deps := httpx.Deps{PG: pg, Mongo: mstore, MQTT: mcMqtt}
	r := httpx.NewRouter(deps)

	srv := &http.Server{
		Addr:              ":" + appPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("mqtt-adapter listening on :%s", appPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
