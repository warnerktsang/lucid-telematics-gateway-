package entity

import (
	"time"
)

// TelemetryBatch represents a batch of telemetry events ingested from a single vehicle.
type TelemetryBatch struct {
	ID         string    `json:"id"`
	VIN        string    `json:"vin"`
	EventCount int       `json:"event_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
