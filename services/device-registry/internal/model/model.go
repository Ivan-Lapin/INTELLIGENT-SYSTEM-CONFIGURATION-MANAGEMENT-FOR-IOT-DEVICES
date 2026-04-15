package model

import "time"

type Device struct {
	ID                    string         `json:"id"`
	ExternalID            string         `json:"external_id"`
	DeviceType            string         `json:"device_type"`
	Protocol              string         `json:"protocol"`
	Status                string         `json:"status"`
	NetworkProfile        string         `json:"network_profile"`
	LocationZone          string         `json:"location_zone"`
	HealthScore           *float64       `json:"health_score,omitempty"`
	BatteryLevel          *int           `json:"battery_level,omitempty"`
	CurrentConfigVersion  *int           `json:"current_config_version,omitempty"`
	LastSuccessfulVersion *int           `json:"last_successful_version,omitempty"`
	LastRolloutStatus     *string        `json:"last_rollout_status,omitempty"`
	Tags                  map[string]any `json:"tags"`
	LastSeenAt            *time.Time     `json:"last_seen_at,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

type CreateDeviceRequest struct {
	ExternalID            string         `json:"external_id" binding:"required"`
	DeviceType            string         `json:"device_type" binding:"required"`
	Protocol              string         `json:"protocol" binding:"required"`
	NetworkProfile        string         `json:"network_profile"`
	LocationZone          string         `json:"location_zone"`
	HealthScore           *float64       `json:"health_score"`
	BatteryLevel          *int           `json:"battery_level"`
	CurrentConfigVersion  *int           `json:"current_config_version"`
	LastSuccessfulVersion *int           `json:"last_successful_version"`
	LastRolloutStatus     *string        `json:"last_rollout_status"`
	Tags                  map[string]any `json:"tags"`
}

type UpdateDeviceStateRequest struct {
	Status                *string    `json:"status,omitempty"`
	HealthScore           *float64   `json:"health_score,omitempty"`
	BatteryLevel          *int       `json:"battery_level,omitempty"`
	CurrentConfigVersion  *int       `json:"current_config_version,omitempty"`
	LastSuccessfulVersion *int       `json:"last_successful_version,omitempty"`
	LastRolloutStatus     *string    `json:"last_rollout_status,omitempty"`
	LastSeenAt            *time.Time `json:"last_seen_at,omitempty"`
}
