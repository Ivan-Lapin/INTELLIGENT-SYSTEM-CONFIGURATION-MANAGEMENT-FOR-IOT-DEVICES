package worker

import "deployment-orchestrator/internal/model"

func shouldRollbackByAckFailRate(failed, pending, total int, maxFailRate float64) model.TelemetryGuardResult {
	if total == 0 {
		return model.TelemetryGuardResult{}
	}

	failRate := float64(failed+pending) / float64(total)
	if failRate > maxFailRate {
		return model.TelemetryGuardResult{
			ShouldRollback: true,
			Reasons:        []string{"ack fail-rate exceeded threshold"},
			OfflineRate:    failRate,
		}
	}

	return model.TelemetryGuardResult{
		ShouldRollback: false,
		OfflineRate:    failRate,
	}
}

func shouldRollbackByTelemetry(
	strat model.Strategy,
	latencyAvg float64,
	packetLossAvg float64,
	offlineRate float64,
) model.TelemetryGuardResult {
	reasons := []string{}

	if latencyAvg > strat.MaxLatencyThreshold {
		reasons = append(reasons, "latency threshold exceeded")
	}
	if packetLossAvg > strat.MaxPacketLoss {
		reasons = append(reasons, "packet loss threshold exceeded")
	}
	if offlineRate > strat.MaxOfflineRate {
		reasons = append(reasons, "device offline rate exceeded")
	}

	return model.TelemetryGuardResult{
		ShouldRollback: len(reasons) > 0,
		Reasons:        reasons,
		LatencyAvg:     latencyAvg,
		PacketLossAvg:  packetLossAvg,
		OfflineRate:    offlineRate,
	}
}
