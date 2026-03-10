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
