# ML Service

Сервис оценки риска rollout.

## Модель

Используется:

* Logistic Regression
* Random Forest
* Gradient Boosting

Лучшая модель выбирается автоматически.

## API

### Predict risk

```http
POST /predict-risk
```

### Health

```http
GET /health
```

## Training

```bash
python train.py
```
