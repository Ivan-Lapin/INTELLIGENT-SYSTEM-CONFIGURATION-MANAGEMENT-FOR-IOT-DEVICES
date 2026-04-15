import json
import os
import pickle
import joblib


def load_best_model(model_dir, feature_count=None):
    meta_path = os.path.join(model_dir, "best_model_meta.json")
    scaler_path = os.path.join(model_dir, "best_scaler.pkl")
    sklearn_path = os.path.join(model_dir, "best_sklearn.pkl")

    if not os.path.exists(meta_path) or not os.path.exists(scaler_path) or not os.path.exists(sklearn_path):
        return None, None, None

    with open(meta_path, "r") as f:
        meta = json.load(f)

    with open(scaler_path, "rb") as f:
        scaler = pickle.load(f)

    model = joblib.load(sklearn_path)
    return model, scaler, meta


def top_feature_payload(meta):
    return {
        "top_features": meta.get("top_features", [])
    }