package model

type Strategy struct {
	CanaryPercent  int     `json:"canaryPercent"`  // default 10
	MaxFailRate    float64 `json:"maxFailRate"`    // default 0.1
	AckWaitSec     int     `json:"ackWaitSec"`     // default 5
	PollIntervalMs int     `json:"pollIntervalMs"` // default 500
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

type DeploymentStatusResponse struct {
	DeploymentId    string  `json:"deploymentId"`
	ConfigVersionId string  `json:"configVersionId"`
	Status          string  `json:"status"`
	StartedAt       *string `json:"startedAt,omitempty"`
	FinishedAt      *string `json:"finishedAt,omitempty"`
	Counts          struct {
		Total      int `json:"total"`
		Canary     int `json:"canary"`
		Full       int `json:"full"`
		Applied    int `json:"applied"`
		Failed     int `json:"failed"`
		Pending    int `json:"pending"`
		Sent       int `json:"sent"`
		RolledBack int `json:"rolledBack"`
	} `json:"counts"`
}
