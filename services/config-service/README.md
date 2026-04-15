# Config Service

Сервис управления конфигурациями устройств.

## Функции

* шаблоны конфигураций
* versioning
* checksum
* diff
* validation

## API

### Create template

```http
POST /v1/templates
```

### Create version

```http
POST /v1/config-versions
```

### Get version

```http
GET /v1/config-versions/{id}
```
