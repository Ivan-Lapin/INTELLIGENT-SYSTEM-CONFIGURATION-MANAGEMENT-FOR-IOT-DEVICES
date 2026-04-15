import os
import pandas as pd
import psycopg2
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Optional, List, Dict, Any

from data_utils import TABULAR_FEATURES, build_single_tabular_features
from inference_utils import load_best_model, top_feature_payload

POSTGRES_DSN = os.getenv(
    "POSTGRES_DSN",
    "dbname=iot_core user=pinchik password=pass_iot_configs host=postgres port=5432"
)

MODEL_DIR = os.getenv("MODEL_DIR", "/app/model_artifacts")
WINDOW_SIZE = int(os.getenv("WINDOW_SIZE", "12"))

app = FastAPI(title="ML Service")

loaded_model = None
loaded_scaler = None
loaded_meta = None


class PredictRiskRequest(BaseModel):
    deviceId: Optional[str] = None
    windowSize: Optional[int] = WINDOW_SIZE
    telemetryWindow: Optional[List[Dict[str, Any]]] = None


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

    return df.sort_values("ts").reset_index(drop=True)


def risk_class(prob: float) -> str:
    if prob >= 0.6:
        return "HIGH"
    if prob >= 0.3:
        return "MEDIUM"
    return "LOW"


def load_artifacts():
    global loaded_model, loaded_scaler, loaded_meta
    model, scaler, meta = load_best_model(MODEL_DIR)
    loaded_model = model
    loaded_scaler = scaler
    loaded_meta = meta


@app.on_event("startup")
def startup_event():
    load_artifacts()


@app.get("/health")
def health():
    return {
        "ok": True,
        "modelLoaded": loaded_model is not None,
        "bestModelType": loaded_meta["best_model_type"] if loaded_meta else None,
    }


@app.post("/reload")
def reload_model():
    load_artifacts()
    return {
        "ok": True,
        "modelLoaded": loaded_model is not None,
        "bestModelType": loaded_meta["best_model_type"] if loaded_meta else None,
    }


@app.get("/feature-importance")
def feature_importance():
    if loaded_meta is None:
        raise HTTPException(status_code=503, detail="model metadata not loaded")
    return top_feature_payload(loaded_meta)


@app.post("/predict-risk")
def predict_risk(req: PredictRiskRequest):
    if loaded_model is None or loaded_scaler is None or loaded_meta is None:
        raise HTTPException(status_code=503, detail="model is not loaded")

    if req.telemetryWindow is not None:
        df = pd.DataFrame(req.telemetryWindow)
    elif req.deviceId:
        try:
            df = fetch_recent_window(req.deviceId, req.windowSize)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))
        except Exception as e:
            raise HTTPException(status_code=400, detail=f"telemetry fetch error: {e}")
    else:
        raise HTTPException(status_code=400, detail="either deviceId or telemetryWindow is required")

    if len(df) < req.windowSize:
        raise HTTPException(status_code=400, detail="not enough rows for inference")

    if "ts" not in df.columns:
        df["ts"] = pd.RangeIndex(start=0, stop=len(df), step=1)

    X = build_single_tabular_features(df)[TABULAR_FEATURES]
    X_scaled = loaded_scaler.transform(X)
    risk = float(loaded_model.predict_proba(X_scaled)[:, 1][0])

    return {
        "deviceId": req.deviceId,
        "risk_probability": round(risk, 6),
        "risk_class": risk_class(risk),
        "model_type": loaded_meta["best_model_type"],
        "top_features": loaded_meta.get("top_features", []),
    }