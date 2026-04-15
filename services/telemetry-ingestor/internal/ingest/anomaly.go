package ingest

import "telemetry-ingestor/internal/model"

func DetectAnomalies(ev model.TelemetryEvent) []model.AnomalyEvent {
	if ev.DeviceID == "" {
		return nil
	}

	out := []model.AnomalyEvent{}

	if ev.Metrics.Loss > 0.1 {
		out = append(out, model.AnomalyEvent{
			DeviceID:     ev.DeviceID,
			TS:           ev.TS,
			RolloutID:    ev.RolloutID,
			RolloutPhase: ev.RolloutPhase,
			EventType:    "qos_degradation",
			Severity:     "high",
			Message:      "packet loss exceeded threshold",
			Metrics: map[string]any{
				"loss": ev.Metrics.Loss,
			},
		})
	}

	if ev.Metrics.LatencyMs > 100 {
		out = append(out, model.AnomalyEvent{
			DeviceID:     ev.DeviceID,
			TS:           ev.TS,
			RolloutID:    ev.RolloutID,
			RolloutPhase: ev.RolloutPhase,
			EventType:    "latency_spike",
			Severity:     "medium",
			Message:      "latency exceeded threshold",
			Metrics: map[string]any{
				"latency_ms": ev.Metrics.LatencyMs,
			},
		})
	}

	if ev.Metrics.RSSI < -90 {
		out = append(out, model.AnomalyEvent{
			DeviceID:     ev.DeviceID,
			TS:           ev.TS,
			RolloutID:    ev.RolloutID,
			RolloutPhase: ev.RolloutPhase,
			EventType:    "weak_signal",
			Severity:     "medium",
			Message:      "RSSI below threshold",
			Metrics: map[string]any{
				"rssi": ev.Metrics.RSSI,
			},
		})
	}

	if ev.Metrics.Battery < 10 {
		out = append(out, model.AnomalyEvent{
			DeviceID:     ev.DeviceID,
			TS:           ev.TS,
			RolloutID:    ev.RolloutID,
			RolloutPhase: ev.RolloutPhase,
			EventType:    "low_battery",
			Severity:     "medium",
			Message:      "battery below threshold",
			Metrics: map[string]any{
				"battery": ev.Metrics.Battery,
			},
		})
	}

	return out
}
