package model

import "time"

type Device struct {
	ID         string         `json:"id"`
	ExternalID *string        `json:"externalId,omitempty"`
	DeviceType string         `json:"deviceType"`
	Protocol   string         `json:"protocol"`
	Status     string         `json:"status"`
	Tags       map[string]any `json:"tags"`
	LastSeenAt *time.Time     `json:"lastSeenAt,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type CreateDeviceRequest struct {
	ExternalID *string        `json:"externalId,omitempty"`
	DeviceType string         `json:"deviceType" binding:"required"`
	Protocol   string         `json:"protocol" binding:"required"` // mqtt|lwm2m|gateway
	Tags       map[string]any `json:"tags,omitempty"`
}
