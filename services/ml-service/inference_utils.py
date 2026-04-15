import json
import os
import pickle
import joblib
import torch
import numpy as np

from model import QoSLSTM


def load_best_model(model_dir, feature_count):
    meta_path = os.path.join(model_dir, "best_model_meta.json")
    scaler_path = os.path.join(model_dir, "best_scaler.pkl")

    if not os.path.exists(meta_path) or not os.path.exists(scaler_path):
        return None, None, None

    with open(meta_path, "r") as f:
        meta = json.load(f)

    with open(scaler_path, "rb") as f:
        scaler = pickle.load(f)

    model_type = meta["best_model_type"]

    if model_type == "lstm":
        model = QoSLSTM(input_size=feature_count, hidden_size=32, num_layers=1)
        model.load_state_dict(torch.load(os.path.join(model_dir, "best_lstm.pt"), map_location="cpu"))
        model.eval()
    else:
        model = joblib.load(os.path.join(model_dir, "best_sklearn.pkl"))

    return model, scaler, meta


def top_feature_payload(meta):
    return {
        "top_features": meta.get("top_features", [])
    }