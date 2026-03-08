import torch
import torch.nn as nn


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
        # берём последний timestep
        last = out[:, -1, :]
        out = self.fc(last)
        return self.sigmoid(out)