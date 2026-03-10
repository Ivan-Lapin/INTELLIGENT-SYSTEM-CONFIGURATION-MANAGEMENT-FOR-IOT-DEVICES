package twin

import (
	"digital-twin-service/internal/model"
	"digital-twin-service/internal/store"
)

func asFloat(v any, def float64) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	default:
		return def
	}
}

func Evaluate(latest *store.LatestTelemetry, cfg map[string]any) model.ValidateConfigResponse {
	// Базовая логика: параметр rate влияет на нагрузку.
	// Чем меньше интервал/выше частота, тем выше риск роста latency/loss и разряда батареи.
	// В нашей модели payload["rate"] трактуем как логическую интенсивность отправки.

	rate := asFloat(cfg["rate"], 10)

	latDelta := 0.0
	lossDelta := 0.0
	jitterDelta := 0.0
	batteryDelta := 0.0
	risk := 0.0
	reason := "configuration is acceptable"

	// Эмпирические правила MVP
	if rate >= 20 {
		latDelta += 4.0
		lossDelta += 0.004
		jitterDelta += 0.8
		batteryDelta -= 0.010
		risk += 0.20
	}
	if rate >= 40 {
		latDelta += 7.0
		lossDelta += 0.008
		jitterDelta += 1.2
		batteryDelta -= 0.020
		risk += 0.25
	}

	// Если текущее состояние уже плохое — twin усиливает риск
	if latest.LatencyMs > 22 {
		risk += 0.20
		latDelta += 2.0
	}
	if latest.Loss > 0.02 {
		risk += 0.20
		lossDelta += 0.003
	}
	if latest.RSSI < -75 {
		risk += 0.20
		lossDelta += 0.004
	}
	if latest.Battery < 0.20 {
		risk += 0.20
		batteryDelta -= 0.020
	}

	// Ограничиваем
	if risk > 1.0 {
		risk = 1.0
	}

	valid := true
	if risk >= 0.65 {
		valid = false
		reason = "predicted QoS degradation risk is too high"
	} else if risk >= 0.40 {
		reason = "configuration is risky but still acceptable"
	}

	return model.ValidateConfigResponse{
		Valid:     valid,
		RiskScore: risk,
		Reason:    reason,
		ExpectedDelta: model.ExpectedDelta{
			LatencyMs: latDelta,
			Loss:      lossDelta,
			JitterMs:  jitterDelta,
			Battery:   batteryDelta,
		},
	}
}
