FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1

WORKDIR /app

COPY services/lwm2m-device-simulator/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY services/lwm2m-device-simulator/sim.py .

CMD ["python", "sim.py"]