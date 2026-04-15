package model

type ValidateConfigRequest struct {
	DeviceID        string `json:"deviceId" binding:"required"`
	ConfigVersionID string `json:"configVersionId" binding:"required"`
}

type ExpectedDelta struct {
	LatencyMs float64 `json:"latencyMs"`
	Loss      float64 `json:"loss"`
	JitterMs  float64 `json:"jitterMs"`
	Battery   float64 `json:"battery"`
}

type ValidateConfigResponse struct {
	Valid         bool          `json:"valid"`
	RiskScore     float64       `json:"riskScore"`
	Reason        string        `json:"reason"`
	ExpectedDelta ExpectedDelta `json:"expectedDelta"`
}

type SimulateRequest struct {
	DeviceID        string                 `json:"deviceId" binding:"required"`
	CurrentConfig   map[string]any         `json:"currentConfig" binding:"required"`
	CandidateConfig map[string]any         `json:"candidateConfig" binding:"required"`
	TelemetryWindow TelemetryWindowInput   `json:"telemetryWindow" binding:"required"`
	Deployment      DeploymentContextInput `json:"deployment"`
}

type TelemetryWindowInput struct {
	LatencyAvg    float64 `json:"latencyAvg"`
	LatencyP95    float64 `json:"latencyP95"`
	PacketLossAvg float64 `json:"packetLossAvg"`
	JitterAvg     float64 `json:"jitterAvg"`
	RSSIAvg       float64 `json:"rssiAvg"`
	BatteryLevel  float64 `json:"batteryLevel"`
	SampleCount   int     `json:"sampleCount"`
	Window        string  `json:"window"` // 1m / 5m / 15m
}

type DeploymentContextInput struct {
	RolloutStrategy string `json:"rolloutStrategy"` // canary / full
	CanarySize      int    `json:"canarySize"`
	TargetGroupSize int    `json:"targetGroupSize"`
}

type SimulationResponse struct {
	RiskScore           float64     `json:"riskScore"`
	Recommendation      string      `json:"recommendation"` // approve / canary / reject
	PredictedLatency    float64     `json:"predictedLatency"`
	PredictedPacketLoss float64     `json:"predictedPacketLoss"`
	BatteryImpact       float64     `json:"batteryImpact"`
	Reasons             []string    `json:"reasons"`
	LayerScores         LayerScores `json:"layerScores"`
}

type LayerScores struct {
	DeviceScore     float64 `json:"deviceScore"`
	NetworkScore    float64 `json:"networkScore"`
	DeploymentScore float64 `json:"deploymentScore"`
}
