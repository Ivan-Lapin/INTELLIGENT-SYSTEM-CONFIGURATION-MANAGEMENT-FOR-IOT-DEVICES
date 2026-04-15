import numpy as np
import pandas as pd
from sklearn.preprocessing import StandardScaler


RAW_FEATURES = ["latency_ms", "loss", "jitter_ms", "rssi", "battery"]

TABULAR_FEATURES = [
    "latency_avg",
    "latency_std",
    "latency_p95",
    "loss_avg",
    "loss_max",
    "jitter_avg",
    "rssi_avg",
    "battery_avg",
    "battery_delta",
    "latency_slope",
    "loss_slope",
]

SEQUENCE_FEATURES = RAW_FEATURES


def label_qos_degradation(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()

    df["target"] = (
        (df["latency_ms"] > 24.0) |
        (df["loss"] > 0.017) |
        (df["rssi"] < -69.0) |
        (df["battery"] < 25.0)
    ).astype(int)

    return df


def fit_scaler(df: pd.DataFrame, features):
    scaler = StandardScaler()
    scaler.fit(df[features])
    return scaler


def apply_scaler(df: pd.DataFrame, scaler, features):
    df = df.copy()
    df[features] = scaler.transform(df[features])
    return df


def make_sequence_windows(df: pd.DataFrame, window_size: int = 12):
    values = df[SEQUENCE_FEATURES].values
    targets = df["target"].values

    X, y = [], []
    for i in range(len(df) - window_size):
        X.append(values[i:i + window_size])
        y.append(targets[i + window_size])

    return np.array(X, dtype=np.float32), np.array(y, dtype=np.float32)


def _slope(x):
    if len(x) < 2:
        return 0.0
    return float(x.iloc[-1] - x.iloc[0]) / max(1, len(x) - 1)


def build_tabular_dataset(df: pd.DataFrame, window_size: int = 12):
    rows = []

    for device_id, g in df.groupby("device_id"):
        g = g.sort_values("ts").reset_index(drop=True)
        if len(g) <= window_size:
            continue

        for i in range(len(g) - window_size):
            window = g.iloc[i:i + window_size]
            target = int(g.iloc[i + window_size]["target"])

            row = {
                "device_id": device_id,
                "target": target,
                "latency_avg": window["latency_ms"].mean(),
                "latency_std": window["latency_ms"].std(ddof=0),
                "latency_p95": window["latency_ms"].quantile(0.95),
                "loss_avg": window["loss"].mean(),
                "loss_max": window["loss"].max(),
                "jitter_avg": window["jitter_ms"].mean(),
                "rssi_avg": window["rssi"].mean(),
                "battery_avg": window["battery"].mean(),
                "battery_delta": float(window["battery"].iloc[-1] - window["battery"].iloc[0]),
                "latency_slope": _slope(window["latency_ms"]),
                "loss_slope": _slope(window["loss"]),
            }
            rows.append(row)

    if not rows:
        raise RuntimeError("Not enough telemetry to build tabular dataset")

    return pd.DataFrame(rows)

def build_single_tabular_features(df: pd.DataFrame) -> pd.DataFrame:
    df = df.sort_values("ts").reset_index(drop=True)

    def _safe_std(series):
        v = series.std(ddof=0)
        return 0.0 if pd.isna(v) else float(v)

    def _safe_quantile(series, q):
        v = series.quantile(q)
        return 0.0 if pd.isna(v) else float(v)

    def _slope(series):
        if len(series) < 2:
            return 0.0
        return float(series.iloc[-1] - series.iloc[0]) / max(1, len(series) - 1)

    row = {
        "latency_avg": float(df["latency_ms"].mean()),
        "latency_std": _safe_std(df["latency_ms"]),
        "latency_p95": _safe_quantile(df["latency_ms"], 0.95),
        "loss_avg": float(df["loss"].mean()),
        "loss_max": float(df["loss"].max()),
        "jitter_avg": float(df["jitter_ms"].mean()),
        "rssi_avg": float(df["rssi"].mean()),
        "battery_avg": float(df["battery"].mean()),
        "battery_delta": float(df["battery"].iloc[-1] - df["battery"].iloc[0]),
        "latency_slope": _slope(df["latency_ms"]),
        "loss_slope": _slope(df["loss"]),
    }

    return pd.DataFrame([row])