package model

type Strategy struct {
	CanaryPercent  int     `json:"canaryPercent"`  // default 10
	MaxFailRate    float64 `json:"maxFailRate"`    // default 0.1
	AckWaitSec     int     `json:"ackWaitSec"`     // default 5
	PollIntervalMs int     `json:"pollIntervalMs"` // default 500

	EnableTwin bool    `json:"enableTwin"` // default true
	EnableML   bool    `json:"enableML"`   // default true
	MaxMLRisk  float64 `json:"maxMlRisk"`  // default 0.6

	MaxTwinRisk          float64 `json:"maxTwinRisk"`          // default 0.8
	CanaryTwinRisk       float64 `json:"canaryTwinRisk"`       // default 0.5
	MaxLatencyThreshold  float64 `json:"maxLatencyThreshold"`  // например 50
	MaxPacketLoss        float64 `json:"maxPacketLoss"`        // например 0.05
	MaxOfflineRate       float64 `json:"maxOfflineRate"`       // например 0.2
	EnableTelemetryGuard bool    `json:"enableTelemetryGuard"` // default true
}

type CreateDeploymentRequest struct {
	ConfigVersionId string   `json:"configVersionId" binding:"required"`
	DeviceIds       []string `json:"deviceIds" binding:"required"`
	Strategy        Strategy `json:"strategy"`
}

type CreateDeploymentResponse struct {
	DeploymentId string `json:"deploymentId"`
	Status       string `json:"status"`
}

type DeploymentCounts struct {
	Total      int `json:"total"`
	Canary     int `json:"canary"`
	Full       int `json:"full"`
	Applied    int `json:"applied"`
	Failed     int `json:"failed"`
	Pending    int `json:"pending"`
	Sent       int `json:"sent"`
	RolledBack int `json:"rolledBack"`
}

type DeploymentStatusResponse struct {
	DeploymentId    string           `json:"deploymentId"`
	ConfigVersionId string           `json:"configVersionId"`
	Status          string           `json:"status"`
	StartedAt       *string          `json:"startedAt,omitempty"`
	FinishedAt      *string          `json:"finishedAt,omitempty"`
	Counts          DeploymentCounts `json:"counts"`
}

type DecisionResult struct {
	Action  string   `json:"action"` // reject / canary / full
	Reasons []string `json:"reasons"`
}

type TelemetryGuardResult struct {
	ShouldRollback bool     `json:"shouldRollback"`
	Reasons        []string `json:"reasons"`
	LatencyAvg     float64  `json:"latencyAvg"`
	PacketLossAvg  float64  `json:"packetLossAvg"`
	OfflineRate    float64  `json:"offlineRate"`
}
