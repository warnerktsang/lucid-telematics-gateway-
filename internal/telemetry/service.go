package telemetry

import (
	"context"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/entity"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/log"
	"time"
)

// Service encapsulates usecase logic for telemetry batches.
type Service interface {
	Get(ctx context.Context, id string) (Batch, error)
	Query(ctx context.Context, offset, limit int) ([]Batch, error)
	Count(ctx context.Context) (int, error)
	Create(ctx context.Context, input CreateTelemetryBatchRequest) (Batch, error)
	Update(ctx context.Context, id string, input UpdateTelemetryBatchRequest) (Batch, error)
	Delete(ctx context.Context, id string) (Batch, error)
}

// Batch represents the data about a telemetry batch.
type Batch struct {
	entity.TelemetryBatch
}

// CreateTelemetryBatchRequest represents a telemetry batch creation request.
type CreateTelemetryBatchRequest struct {
	VIN        string `json:"vin"`
	EventCount int    `json:"event_count"`
}

// Validate validates the CreateTelemetryBatchRequest fields.
func (m CreateTelemetryBatchRequest) Validate() error {
	return validation.ValidateStruct(&m,
		validation.Field(&m.VIN, validation.Required, validation.Length(17, 17)),
		validation.Field(&m.EventCount, validation.Required, validation.Min(1)),
	)
}

// UpdateTelemetryBatchRequest represents a telemetry batch update request.
type UpdateTelemetryBatchRequest struct {
	VIN        string `json:"vin"`
	EventCount int    `json:"event_count"`
}

// Validate validates the UpdateTelemetryBatchRequest fields.
func (m UpdateTelemetryBatchRequest) Validate() error {
	return validation.ValidateStruct(&m,
		validation.Field(&m.VIN, validation.Required, validation.Length(17, 17)),
		validation.Field(&m.EventCount, validation.Required, validation.Min(1)),
	)
}

type service struct {
	repo   Repository
	logger log.Logger
}

// NewService creates a new telemetry batch service.
func NewService(repo Repository, logger log.Logger) Service {
	return service{repo, logger}
}

// Get returns the telemetry batch with the specified the batch ID.
func (s service) Get(ctx context.Context, id string) (Batch, error) {
	batch, err := s.repo.Get(ctx, id)
	if err != nil {
		return Batch{}, err
	}
	return Batch{batch}, nil
}

// Create creates a new telemetry batch.
func (s service) Create(ctx context.Context, req CreateTelemetryBatchRequest) (Batch, error) {
	if err := req.Validate(); err != nil {
		return Batch{}, err
	}
	id := entity.GenerateID()
	now := time.Now()
	err := s.repo.Create(ctx, entity.TelemetryBatch{
		ID:         id,
		VIN:        req.VIN,
		EventCount: req.EventCount,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return Batch{}, err
	}
	return s.Get(ctx, id)
}

// Update updates the telemetry batch with the specified ID.
func (s service) Update(ctx context.Context, id string, req UpdateTelemetryBatchRequest) (Batch, error) {
	if err := req.Validate(); err != nil {
		return Batch{}, err
	}

	batch, err := s.Get(ctx, id)
	if err != nil {
		return batch, err
	}
	batch.VIN = req.VIN
	batch.EventCount = req.EventCount
	batch.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, batch.TelemetryBatch); err != nil {
		return batch, err
	}
	return batch, nil
}

// Delete deletes the telemetry batch with the specified ID.
func (s service) Delete(ctx context.Context, id string) (Batch, error) {
	batch, err := s.Get(ctx, id)
	if err != nil {
		return Batch{}, err
	}
	if err = s.repo.Delete(ctx, id); err != nil {
		return Batch{}, err
	}
	return batch, nil
}

// Count returns the number of telemetry batches.
func (s service) Count(ctx context.Context) (int, error) {
	return s.repo.Count(ctx)
}

// Query returns the telemetry batches with the specified offset and limit.
func (s service) Query(ctx context.Context, offset, limit int) ([]Batch, error) {
	items, err := s.repo.Query(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	result := []Batch{}
	for _, item := range items {
		result = append(result, Batch{item})
	}
	return result, nil
}
