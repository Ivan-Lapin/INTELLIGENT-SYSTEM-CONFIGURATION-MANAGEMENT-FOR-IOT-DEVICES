# Intelligent IoT Config Management Prototype

## Requirements
- Docker + Docker Compose

## Quick start


В корне проекта:
touch .env

Заполнить данным:
POSTGRES_USER=iot
POSTGRES_PASSWORD=iot
POSTGRES_DB=iot

MONGO_ROOT_USER=root
MONGO_ROOT_PASSWORD=root
MONGO_DB=iot_configs

```bash
cp .env.example .env
make up
make health
```

Services

Postgres: localhost:5432

Mongo: localhost:27017

Redis: localhost:6379

MQTT (Mosquitto): localhost:1883

NATS: localhost:4222 (monitoring: http://localhost:8222
)