# Deployment Orchestrator

Сервис безопасного развертывания конфигураций.

## Возможности

* precheck
* canary rollout
* smart rollout
* rollback
* deployment tracking

## Pipeline

```text
Telemetry → ML → Twin → Decision → Deployment
```

## API

```http
POST /v1/deployments
GET /v1/deployments/{id}
```
