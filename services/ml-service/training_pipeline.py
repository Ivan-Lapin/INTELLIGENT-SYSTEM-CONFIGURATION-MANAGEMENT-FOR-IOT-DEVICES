import json
import os
import pickle
import joblib
import numpy as np
import pandas as pd
import psycopg2
# import torch
# import torch.nn as nn

from sklearn.metrics import accuracy_score, precision_score, recall_score, f1_score, roc_auc_score
from sklearn.model_selection import train_test_split
# from torch.utils.data import TensorDataset, DataLoader

from data_utils import (
    RAW_FEATURES,
    TABULAR_FEATURES,
    SEQUENCE_FEATURES,
    label_qos_degradation,
    fit_scaler,
    apply_scaler,
    make_sequence_windows,
    build_tabular_dataset,
)
from model import (
    build_logistic_regression,
    build_random_forest,
    build_gradient_boosting,
)

POSTGRES_DSN = os.getenv(
    "POSTGRES_DSN",
    "postgres://pinchik:pass_iot_core@postgres:5432/iot_core?sslmode=disable"
)

MODEL_DIR = os.getenv("MODEL_DIR", "/app/model_artifacts")
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


def calc_metrics(y_true, y_pred, y_proba):
    return {
        "accuracy": float(accuracy_score(y_true, y_pred)),
        "precision": float(precision_score(y_true, y_pred, zero_division=0)),
        "recall": float(recall_score(y_true, y_pred, zero_division=0)),
        "f1": float(f1_score(y_true, y_pred, zero_division=0)),
        "roc_auc": float(roc_auc_score(y_true, y_proba)) if len(set(y_true)) > 1 else 0.0,
    }


def train_lstm(df):
    scaler = fit_scaler(df, SEQUENCE_FEATURES)
    df_scaled = apply_scaler(df, scaler, SEQUENCE_FEATURES)

    X_all, y_all = [], []
    for _, g in df_scaled.groupby("device_id"):
        if len(g) <= WINDOW_SIZE:
            continue
        X, y = make_sequence_windows(g, window_size=WINDOW_SIZE)
        if len(X) > 0:
            X_all.append(X)
            y_all.append(y)

    if not X_all:
        raise RuntimeError("Not enough telemetry to build LSTM dataset")

    X = np.concatenate(X_all, axis=0)
    y = np.concatenate(y_all, axis=0)
    
    y_series = pd.Series(y)
    class_counts = y_series.value_counts().to_dict()
    print(f"[lstm] target distribution: {class_counts}")

    if y_series.nunique() < 2:
        raise RuntimeError(
            f"Training dataset for lstm contains only one class: {class_counts}. "
            "Need both normal and degradation samples."
        )

    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )

    X_train_tensor = torch.tensor(X_train, dtype=torch.float32)
    y_train_tensor = torch.tensor(y_train, dtype=torch.float32).unsqueeze(1)
    X_test_tensor = torch.tensor(X_test, dtype=torch.float32)

    loader = DataLoader(TensorDataset(X_train_tensor, y_train_tensor), batch_size=32, shuffle=True)

    model = QoSLSTM(input_size=len(SEQUENCE_FEATURES), hidden_size=32, num_layers=1)
    criterion = nn.BCELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=1e-3)

    for _ in range(10):
        model.train()
        for xb, yb in loader:
            optimizer.zero_grad()
            pred = model(xb)
            loss = criterion(pred, yb)
            loss.backward()
            optimizer.step()

    model.eval()
    with torch.no_grad():
        y_proba = model(X_test_tensor).squeeze(1).numpy()

    y_pred = (y_proba >= 0.5).astype(int)
    metrics = calc_metrics(y_test, y_pred, y_proba)

    return {
        "model_type": "lstm",
        "model": model,
        "scaler": scaler,
        "metrics": metrics,
        "feature_names": SEQUENCE_FEATURES,
        "feature_importance": None,
    }


def train_tabular_model(df, model_type, builder):
    tab_df = build_tabular_dataset(df, window_size=WINDOW_SIZE)

    X = tab_df[TABULAR_FEATURES]
    y = tab_df["target"]
    
    class_counts = y.value_counts().to_dict()
    print(f"[{model_type}] target distribution: {class_counts}")

    if y.nunique() < 2:
        raise RuntimeError(
            f"Training dataset for {model_type} contains only one class: {class_counts}. "
            "Need both normal and degradation samples."
        )

    scaler = fit_scaler(tab_df, TABULAR_FEATURES)
    X_scaled = scaler.transform(X)

    X_train, X_test, y_train, y_test = train_test_split(
        X_scaled, y, test_size=0.2, random_state=42, stratify=y
    )

    clf = builder()
    clf.fit(X_train, y_train)

    y_proba = clf.predict_proba(X_test)[:, 1]
    y_pred = (y_proba >= 0.5).astype(int)

    metrics = calc_metrics(y_test, y_pred, y_proba)

    feature_importance = None
    if hasattr(clf, "feature_importances_"):
        feature_importance = list(clf.feature_importances_)
    elif hasattr(clf, "coef_"):
        feature_importance = list(np.abs(clf.coef_[0]))

    return {
        "model_type": model_type,
        "model": clf,
        "scaler": scaler,
        "metrics": metrics,
        "feature_names": TABULAR_FEATURES,
        "feature_importance": feature_importance,
    }


def save_artifacts(results):
    os.makedirs(MODEL_DIR, exist_ok=True)

    leaderboard = []
    best = None

    for res in results:
        leaderboard.append({
            "model_type": res["model_type"],
            **res["metrics"],
        })
        if best is None or res["metrics"]["f1"] > best["metrics"]["f1"]:
            best = res

    leaderboard = sorted(leaderboard, key=lambda x: x["f1"], reverse=True)

    with open(os.path.join(MODEL_DIR, "leaderboard.json"), "w") as f:
        json.dump(leaderboard, f, indent=2)

    metadata = {
        "best_model_type": best["model_type"],
        "metrics": best["metrics"],
        "feature_names": best["feature_names"],
    }

    if best["feature_importance"] is not None:
        pairs = list(zip(best["feature_names"], best["feature_importance"]))
        pairs.sort(key=lambda x: x[1], reverse=True)
        metadata["top_features"] = [name for name, _ in pairs[:10]]
    else:
        metadata["top_features"] = []

    with open(os.path.join(MODEL_DIR, "best_model_meta.json"), "w") as f:
        json.dump(metadata, f, indent=2)

    if best["model_type"] == "lstm":
        torch.save(best["model"].state_dict(), os.path.join(MODEL_DIR, "best_lstm.pt"))
        with open(os.path.join(MODEL_DIR, "best_scaler.pkl"), "wb") as f:
            pickle.dump(best["scaler"], f)
    else:
        joblib.dump(best["model"], os.path.join(MODEL_DIR, "best_sklearn.pkl"))
        with open(os.path.join(MODEL_DIR, "best_scaler.pkl"), "wb") as f:
            pickle.dump(best["scaler"], f)

    return leaderboard, metadata


def run_training_pipeline():
    df = load_data()
    df = label_qos_degradation(df)
    
    

    results = []

    results.append(train_tabular_model(df, "logistic_regression", build_logistic_regression))
    results.append(train_tabular_model(df, "random_forest", build_random_forest))
    results.append(train_tabular_model(df, "gradient_boosting", build_gradient_boosting))
    # LSTM temporarily disabled due to runtime instability under container emulation
    # results.append(train_lstm(df))

    leaderboard, metadata = save_artifacts(results)
    return leaderboard, metadata