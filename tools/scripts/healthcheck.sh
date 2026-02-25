#!/usr/bin/env bash
set -euo pipefail

echo "== Checking containers =="
docker compose ps

echo
echo "== Checking Postgres =="
docker exec -i iot_postgres pg_isready -U "${POSTGRES_USER:-iot}" -d "${POSTGRES_DB:-iot}" >/dev/null
echo "Postgres OK"

echo
echo "== Checking Mongo =="
docker exec -i iot_mongo mongosh --quiet --username "${MONGO_ROOT_USER:-root}" --password "${MONGO_ROOT_PASSWORD:-root}" --eval "db.runCommand({ ping: 1 })" admin >/dev/null
echo "Mongo OK"

echo
echo "== Checking Redis =="
docker exec -i iot_redis redis-cli ping | grep -q PONG
echo "Redis OK"

echo
echo "== Checking Mosquitto (port open) =="
docker exec -i iot_mosquitto sh -lc "nc -z localhost 1883"
echo "Mosquitto OK"

echo
echo "== Checking NATS health =="
docker exec -i iot_nats wget -qO- http://localhost:8222/healthz | grep -qi ok
echo "NATS OK"

echo
echo "ALL OK ✅"