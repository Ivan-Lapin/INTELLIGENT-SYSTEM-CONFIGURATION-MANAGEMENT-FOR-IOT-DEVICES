package twin

import (
	"digital-twin-service/internal/model"
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

func evaluateDeviceTwin(req model.SimulateRequest) (float64, float64, []string) {
	score := 0.0
	batteryImpact := 0.0
	reasons := []string{}

	currentRate := asFloat(req.CurrentConfig["rate"], 10)
	candidateRate := asFloat(req.CandidateConfig["rate"], currentRate)

	if candidateRate > currentRate {
		delta := candidateRate - currentRate
		score += delta * 0.01
		batteryImpact += delta * 0.005
		reasons = append(reasons, "higher telemetry frequency increases device load")
	}

	if req.TelemetryWindow.BatteryLevel < 20 {
		score += 0.20
		batteryImpact += 0.05
		reasons = append(reasons, "low battery level")
	}

	if candidateRate < 5 && req.TelemetryWindow.BatteryLevel < 20 {
		score += 0.25
		reasons = append(reasons, "very aggressive reporting interval on low battery")
	}

	return clamp(score), batteryImpact, reasons
}

func evaluateNetworkTwin(req model.SimulateRequest) (float64, float64, float64, []string) {
	score := 0.0
	predLatency := req.TelemetryWindow.LatencyAvg
	predLoss := req.TelemetryWindow.PacketLossAvg
	reasons := []string{}

	currentRate := asFloat(req.CurrentConfig["rate"], 10)
	candidateRate := asFloat(req.CandidateConfig["rate"], currentRate)
	rateDelta := candidateRate - currentRate

	if rateDelta > 0 {
		predLatency += rateDelta * 0.8
		predLoss += rateDelta * 0.001
		score += rateDelta * 0.01
		reasons = append(reasons, "higher rate may increase network latency and packet loss")
	}

	if req.TelemetryWindow.RSSIAvg < -85 {
		score += 0.20
		predLoss += 0.01
		reasons = append(reasons, "low RSSI")
	}

	if req.TelemetryWindow.PacketLossAvg > 0.03 {
		score += 0.20
		predLoss += 0.005
		reasons = append(reasons, "already elevated packet loss")
	}

	if req.TelemetryWindow.LatencyP95 > 50 {
		score += 0.15
		predLatency += 3
		reasons = append(reasons, "high latency p95 baseline")
	}

	return clamp(score), predLatency, predLoss, reasons
}

func evaluateDeploymentTwin(req model.SimulateRequest) (float64, []string) {
	score := 0.0
	reasons := []string{}

	if req.Deployment.RolloutStrategy == "full" && req.Deployment.TargetGroupSize > 50 {
		score += 0.25
		reasons = append(reasons, "full rollout to large target group")
	}

	if req.Deployment.RolloutStrategy == "canary" && req.Deployment.CanarySize > 20 {
		score += 0.10
		reasons = append(reasons, "large canary size")
	}

	if req.Deployment.RolloutStrategy == "full" && req.Deployment.TargetGroupSize > 100 {
		score += 0.20
		reasons = append(reasons, "high deployment blast radius")
	}

	return clamp(score), reasons
}

func EvaluateSimulation(req model.SimulateRequest) model.SimulationResponse {
	deviceScore, batteryImpact, deviceReasons := evaluateDeviceTwin(req)
	networkScore, predictedLatency, predictedLoss, networkReasons := evaluateNetworkTwin(req)
	deploymentScore, deploymentReasons := evaluateDeploymentTwin(req)

	total := clamp(deviceScore*0.35 + networkScore*0.45 + deploymentScore*0.20)

	reasons := append([]string{}, deviceReasons...)
	reasons = append(reasons, networkReasons...)
	reasons = append(reasons, deploymentReasons...)

	recommendation := "approve"
	if total >= 0.70 {
		recommendation = "reject"
	} else if total >= 0.40 {
		recommendation = "canary"
	}

	return model.SimulationResponse{
		RiskScore:           total,
		Recommendation:      recommendation,
		PredictedLatency:    predictedLatency,
		PredictedPacketLoss: predictedLoss,
		BatteryImpact:       batteryImpact,
		Reasons:             reasons,
		LayerScores: model.LayerScores{
			DeviceScore:     deviceScore,
			NetworkScore:    networkScore,
			DeploymentScore: deploymentScore,
		},
	}
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
