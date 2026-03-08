import numpy as np
import pandas as pd
from sklearn.preprocessing import StandardScaler


FEATURES = ["latency_ms", "loss", "jitter_ms", "rssi", "battery"]


def label_qos_degradation(df: pd.DataFrame) -> pd.DataFrame:
    """
    Примитивная, но честная целевая функция для MVP:
    1 = деградация QoS
    если latency > 25ms или loss > 0.03 или rssi < -80 или battery < 0.15
    """
    df = df.copy()
    df["target"] = (
        (df["latency_ms"] > 25.0) |
        (df["loss"] > 0.03) |
        (df["rssi"] < -80.0) |
        (df["battery"] < 0.15)
    ).astype(int)
    return df


def make_windows(df: pd.DataFrame, window_size: int = 12):
    """
    Из последовательности строк делаем:
    X shape = [samples, window_size, features]
    y shape = [samples]
    """
    values = df[FEATURES].values
    targets = df["target"].values

    X, y = [], []
    for i in range(len(df) - window_size):
        X.append(values[i:i + window_size])
        y.append(targets[i + window_size])

    return np.array(X, dtype=np.float32), np.array(y, dtype=np.float32)


def fit_scaler(df: pd.DataFrame):
    scaler = StandardScaler()
    scaler.fit(df[FEATURES])
    return scaler


def apply_scaler(df: pd.DataFrame, scaler):
    df = df.copy()
    df[FEATURES] = scaler.transform(df[FEATURES])
    return df