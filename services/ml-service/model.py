import torch
import torch.nn as nn
from sklearn.linear_model import LogisticRegression
from sklearn.ensemble import RandomForestClassifier, GradientBoostingClassifier


class QoSLSTM(nn.Module):
    def __init__(self, input_size: int = 5, hidden_size: int = 32, num_layers: int = 1):
        super().__init__()
        self.lstm = nn.LSTM(
            input_size=input_size,
            hidden_size=hidden_size,
            num_layers=num_layers,
            batch_first=True,
        )
        self.fc = nn.Linear(hidden_size, 1)
        self.sigmoid = nn.Sigmoid()

    def forward(self, x):
        out, _ = self.lstm(x)
        last = out[:, -1, :]
        out = self.fc(last)
        return self.sigmoid(out)


def build_logistic_regression():
    return LogisticRegression(max_iter=1000, class_weight="balanced")


def build_random_forest():
    return RandomForestClassifier(
        n_estimators=200,
        max_depth=8,
        min_samples_leaf=3,
        random_state=42,
        class_weight="balanced"
    )


def build_gradient_boosting():
    return GradientBoostingClassifier(
        n_estimators=150,
        learning_rate=0.05,
        max_depth=3,
        random_state=42
    )