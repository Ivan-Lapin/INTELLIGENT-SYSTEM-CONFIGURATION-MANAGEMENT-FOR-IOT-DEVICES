# IoT Configuration Management System

Прототип интеллектуальной системы безопасного управления конфигурациями IoT-устройств.

Проект реализует безопасное развертывание конфигураций с использованием:

* **ML-based risk assessment**
* **Digital Twin simulation**
* **Canary rollout**
* **Automatic rollback**
* **Telemetry monitoring**
* **MQTT / LwM2M adapters**

---

## Архитектура системы

Система построена по микросервисной архитектуре.

Основные сервисы:

* `device-registry` — реестр устройств
* `config-service` — управление шаблонами и версиями конфигураций
* `deployment-orchestrator` — безопасное развертывание
* `telemetry-ingestor` — сбор телеметрии
* `ml-service` — оценка риска
* `digital-twin-service` — моделирование последствий
* `mqtt-adapter` — MQTT integration
* `lwm2m-adapter` — LwM2M integration
* `device-simulator` — MQTT device simulation
* `lwm2m-device-simulator` — LwM2M device simulation

---

## Основные возможности

### Управление устройствами

* регистрация устройств
* управление группами устройств
* хранение device metadata

### Управление конфигурациями

* шаблоны конфигураций
* версионирование
* checksum validation
* diff calculation

### Интеллектуальное развертывание

* risk prediction
* twin simulation
* canary rollout
* automatic rollback
* full rollout

### Мониторинг

* telemetry ingestion
* QoS aggregation
* anomaly detection
* latency / loss analysis

---

## Технологии

### Backend

* Go
* Python
* FastAPI
* PostgreSQL
* MongoDB
* Redis
* MQTT
* Docker / Docker Compose

### ML

* scikit-learn
* pandas
* numpy

---

## Запуск проекта

```bash
make up_all
```

---

## Проверка health endpoints

```bash
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8086/health
```

---

## Запуск ML training

```bash
docker compose -f docker-compose.yml -f docker-compose.services.yml exec ml-service python train.py
```

---

## Эксперименты

```bash
cd experiments
python3 run_experiment.py baseline
python3 run_experiment.py canary
python3 run_experiment.py smart
python3 plot_results.py
```

---

## Структура проекта

```text
services/
contracts/
proto/
deployments/
experiments/
tools/scripts/tests/
```

