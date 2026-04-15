# Device Registry Service

Сервис регистрации и управления IoT-устройствами.

## Функции

* регистрация устройств
* хранение device metadata
* управление группами
* protocol mapping

## API

### Create device

```http
POST /v1/devices
```

### Get device

```http
GET /v1/devices/{id}
```

### Health

```http
GET /health
```

## Запуск

```bash
go run ./cmd/server
```
