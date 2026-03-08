import os
import pickle
from typing import Optional

import pandas as pd
import psycopg2
import torch
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from model import QoSLSTM
from data_utils import FEATURES, apply_scaler

POSTGRES_DSN = os.getenv(
    "POSTGRES_DSN",
    "dbname=iot_core user=pinchik password=pass_iot_configs host=postgres port=5432"
)

MODEL_DIR = os.getenv("MODEL_DIR", "/app/model_artifacts")
MODEL_PATH = os.path.join(MODEL_DIR, "qos_lstm.pt")
SCALER_PATH = os.path.join(MODEL_DIR, "scaler.pkl")
WINDOW_SIZE = int(os.getenv("WINDOW_SIZE", "12"))

app = FastAPI(title="ML Service")

model = None
scaler = None


class PredictRequest(BaseModel):
    deviceId: str
    windowSize: Optional[int] = WINDOW_SIZE


def load_artifacts():
    global model, scaler

    if not os.path.exists(MODEL_PATH):
        print(f"[ml-service] model file not found: {MODEL_PATH}")
        model = None
        scaler = None
        return

    if not os.path.exists(SCALER_PATH):
        print(f"[ml-service] scaler file not found: {SCALER_PATH}")
        model = None
        scaler = None
        return

    loaded_model = QoSLSTM(input_size=len(FEATURES), hidden_size=32, num_layers=1)
    loaded_model.load_state_dict(torch.load(MODEL_PATH, map_location="cpu"))
    loaded_model.eval()

    with open(SCALER_PATH, "rb") as f:
        loaded_scaler = pickle.load(f)

    model = loaded_model
    scaler = loaded_scaler
    print("[ml-service] model and scaler loaded successfully")


def fetch_recent_window(device_id: str, window_size: int):
    conn = psycopg2.connect(POSTGRES_DSN)
    query = """
        SELECT
            ts,
            latency_ms,
            loss,
            jitter_ms,
            rssi,
            battery
        FROM telemetry.metrics_raw
        WHERE device_id = %s
        ORDER BY ts DESC
        LIMIT %s
    """
    df = pd.read_sql(query, conn, params=(device_id, window_size))
    conn.close()

    if df.empty or len(df) < window_size:
        raise ValueError(f"not enough telemetry for device {device_id}, need {window_size}")

    df = df.sort_values("ts").reset_index(drop=True)
    return df


@app.on_event("startup")
def startup_event():
    load_artifacts()


@app.get("/health")
def health():
    return {
        "ok": True,
        "modelLoaded": model is not None,
        "modelPath": MODEL_PATH,
        "scalerPath": SCALER_PATH,
    }


@app.post("/reload")
def reload_model():
    load_artifacts()
    return {
        "ok": True,
        "modelLoaded": model is not None,
    }


@app.post("/predict")
def predict(req: PredictRequest):
    if model is None or scaler is None:
        raise HTTPException(
            status_code=503,
            detail="model is not loaded yet; train the model first and call /reload"
        )

    try:
        df = fetch_recent_window(req.deviceId, req.windowSize)
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))

    df_scaled = apply_scaler(df, scaler)
    x = torch.tensor(df_scaled[FEATURES].values, dtype=torch.float32).unsqueeze(0)

    with torch.no_grad():
        risk = float(model(x).item())

    return {
        "deviceId": req.deviceId,
        "windowSize": req.windowSize,
        "riskScore": round(risk, 6),
        "riskLevel": (
            "HIGH" if risk >= 0.6 else
            "MEDIUM" if risk >= 0.3 else
            "LOW"
        )
    }