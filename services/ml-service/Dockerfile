FROM python:3.11-slim

WORKDIR /app

COPY services/ml-service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY services/ml-service/model.py .
COPY services/ml-service/data_utils.py .
COPY services/ml-service/train.py .
COPY services/ml-service/app.py .

RUN mkdir -p /app/model_artifacts

EXPOSE 8086

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8086"]