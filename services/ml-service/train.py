import os
import pickle
import numpy as np
import pandas as pd
import psycopg2
import torch
import torch.nn as nn
from torch.utils.data import TensorDataset, DataLoader

from model import QoSLSTM
from data_utils import (
    FEATURES,
    label_qos_degradation,
    fit_scaler,
    apply_scaler,
    make_windows,
)


POSTGRES_DSN = os.getenv(
    "POSTGRES_DSN",
    "postgres://pinchik:pass_iot_core@postgres:5432/iot_core?sslmode=disable"
)

MODEL_DIR = os.getenv("MODEL_DIR", "/app/model_artifacts")
MODEL_PATH = os.path.join(MODEL_DIR, "qos_lstm.pt")
SCALER_PATH = os.path.join(MODEL_DIR, "scaler.pkl")
WINDOW_SIZE = int(os.getenv("WINDOW_SIZE", "12"))


def load_data():
    conn = psycopg2.connect(POSTGRES_DSN)
    query = """
        SELECT
            device_id::text,
            ts,
            latency_ms,
            loss,
            jitter_ms,
            rssi,
            battery
        FROM telemetry.metrics_raw
        ORDER BY device_id, ts
    """
    df = pd.read_sql(query, conn)
    conn.close()
    return df


def build_dataset(df: pd.DataFrame):
    df = label_qos_degradation(df)

    scaler = fit_scaler(df)
    df = apply_scaler(df, scaler)

    X_all, y_all = [], []

    for device_id, g in df.groupby("device_id"):
        if len(g) <= WINDOW_SIZE:
            continue
        X, y = make_windows(g, window_size=WINDOW_SIZE)
        if len(X) > 0:
            X_all.append(X)
            y_all.append(y)

    if not X_all:
        raise RuntimeError("Not enough telemetry to build dataset")

    X = np.concatenate(X_all, axis=0)
    y = np.concatenate(y_all, axis=0)

    return X, y, scaler


def train():
    os.makedirs(MODEL_DIR, exist_ok=True)

    df = load_data()
    X, y, scaler = build_dataset(df)

    X_tensor = torch.tensor(X, dtype=torch.float32)
    y_tensor = torch.tensor(y, dtype=torch.float32).unsqueeze(1)

    dataset = TensorDataset(X_tensor, y_tensor)
    loader = DataLoader(dataset, batch_size=32, shuffle=True)

    model = QoSLSTM(input_size=len(FEATURES), hidden_size=32, num_layers=1)
    criterion = nn.BCELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=1e-3)

    epochs = 10
    for epoch in range(epochs):
        total_loss = 0.0
        model.train()
        for xb, yb in loader:
            optimizer.zero_grad()
            pred = model(xb)
            loss = criterion(pred, yb)
            loss.backward()
            optimizer.step()
            total_loss += loss.item()

        avg_loss = total_loss / max(1, len(loader))
        print(f"epoch={epoch+1} loss={avg_loss:.6f}")

    torch.save(model.state_dict(), MODEL_PATH)
    with open(SCALER_PATH, "wb") as f:
        pickle.dump(scaler, f)

    print(f"saved model to {MODEL_PATH}")
    print(f"saved scaler to {SCALER_PATH}")


if __name__ == "__main__":
    train()