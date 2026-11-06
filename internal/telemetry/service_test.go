package telemetry

import (
	"context"
	"database/sql"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/entity"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/log"
	"testing"
)

var errCRUD = errors.New("error crud")

const validVIN = "50EA1TEA1PA004821"

func TestCreateTelemetryBatchRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		model     CreateTelemetryBatchRequest
		wantError bool
	}{
		{"success", CreateTelemetryBatchRequest{VIN: validVIN, EventCount: 1}, false},
		{"required", CreateTelemetryBatchRequest{VIN: "", EventCount: 1}, true},
		{"vin too short", CreateTelemetryBatchRequest{VIN: "50EA1TEA1PA00482", EventCount: 1}, true},
		{"vin too long", CreateTelemetryBatchRequest{VIN: "50EA1TEA1PA0048210", EventCount: 1}, true},
		{"event count zero", CreateTelemetryBatchRequest{VIN: validVIN, EventCount: 0}, true},
		{"event count negative", CreateTelemetryBatchRequest{VIN: validVIN, EventCount: -1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.Validate()
			assert.Equal(t, tt.wantError, err != nil)
		})
	}
}

func TestUpdateTelemetryBatchRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		model     UpdateTelemetryBatchRequest
		wantError bool
	}{
		{"success", UpdateTelemetryBatchRequest{VIN: validVIN, EventCount: 1}, false},
		{"required", UpdateTelemetryBatchRequest{VIN: "", EventCount: 1}, true},
		{"vin too short", UpdateTelemetryBatchRequest{VIN: "50EA1TEA1PA00482", EventCount: 1}, true},
		{"event count zero", UpdateTelemetryBatchRequest{VIN: validVIN, EventCount: 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.Validate()
			assert.Equal(t, tt.wantError, err != nil)
		})
	}
}

func Test_service_CRUD(t *testing.T) {
	logger, _ := log.NewForTest()
	s := NewService(&mockRepository{}, logger)

	ctx := context.Background()

	// initial count
	count, _ := s.Count(ctx)
	assert.Equal(t, 0, count)

	// successful creation
	batch, err := s.Create(ctx, CreateTelemetryBatchRequest{VIN: validVIN, EventCount: 10})
	assert.Nil(t, err)
	assert.NotEmpty(t, batch.ID)
	id := batch.ID
	assert.Equal(t, validVIN, batch.VIN)
	assert.Equal(t, 10, batch.EventCount)
	assert.NotEmpty(t, batch.CreatedAt)
	assert.NotEmpty(t, batch.UpdatedAt)
	count, _ = s.Count(ctx)
	assert.Equal(t, 1, count)

	// validation error in creation
	_, err = s.Create(ctx, CreateTelemetryBatchRequest{VIN: "", EventCount: 10})
	assert.NotNil(t, err)
	count, _ = s.Count(ctx)
	assert.Equal(t, 1, count)

	// unexpected error in creation
	_, err = s.Create(ctx, CreateTelemetryBatchRequest{VIN: errorVIN, EventCount: 10})
	assert.Equal(t, errCRUD, err)
	count, _ = s.Count(ctx)
	assert.Equal(t, 1, count)

	_, _ = s.Create(ctx, CreateTelemetryBatchRequest{VIN: "50EA1TEA3PA004822", EventCount: 1})

	// update
	batch, err = s.Update(ctx, id, UpdateTelemetryBatchRequest{VIN: validVIN, EventCount: 42})
	assert.Nil(t, err)
	assert.Equal(t, 42, batch.EventCount)
	_, err = s.Update(ctx, "none", UpdateTelemetryBatchRequest{VIN: validVIN, EventCount: 42})
	assert.NotNil(t, err)

	// validation error in update
	_, err = s.Update(ctx, id, UpdateTelemetryBatchRequest{VIN: "", EventCount: 42})
	assert.NotNil(t, err)
	count, _ = s.Count(ctx)
	assert.Equal(t, 2, count)

	// unexpected error in update
	_, err = s.Update(ctx, id, UpdateTelemetryBatchRequest{VIN: errorVIN, EventCount: 42})
	assert.Equal(t, errCRUD, err)
	count, _ = s.Count(ctx)
	assert.Equal(t, 2, count)

	// get
	_, err = s.Get(ctx, "none")
	assert.NotNil(t, err)
	batch, err = s.Get(ctx, id)
	assert.Nil(t, err)
	assert.Equal(t, 42, batch.EventCount)
	assert.Equal(t, id, batch.ID)

	// query
	batches, _ := s.Query(ctx, 0, 0)
	assert.Equal(t, 2, len(batches))

	// delete
	_, err = s.Delete(ctx, "none")
	assert.NotNil(t, err)
	batch, err = s.Delete(ctx, id)
	assert.Nil(t, err)
	assert.Equal(t, id, batch.ID)
	count, _ = s.Count(ctx)
	assert.Equal(t, 1, count)
}

// errorVIN is a 17-character VIN that passes validation but makes the mock repository fail.
const errorVIN = "ERRORERRORERROR00"

type mockRepository struct {
	items []entity.TelemetryBatch
}

func (m mockRepository) Get(ctx context.Context, id string) (entity.TelemetryBatch, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return entity.TelemetryBatch{}, sql.ErrNoRows
}

func (m mockRepository) Count(ctx context.Context) (int, error) {
	return len(m.items), nil
}

func (m mockRepository) Query(ctx context.Context, offset, limit int) ([]entity.TelemetryBatch, error) {
	return m.items, nil
}

func (m *mockRepository) Create(ctx context.Context, batch entity.TelemetryBatch) error {
	if batch.VIN == errorVIN {
		return errCRUD
	}
	m.items = append(m.items, batch)
	return nil
}

func (m *mockRepository) Update(ctx context.Context, batch entity.TelemetryBatch) error {
	if batch.VIN == errorVIN {
		return errCRUD
	}
	for i, item := range m.items {
		if item.ID == batch.ID {
			m.items[i] = batch
			break
		}
	}
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	for i, item := range m.items {
		if item.ID == id {
			m.items[i] = m.items[len(m.items)-1]
			m.items = m.items[:len(m.items)-1]
			break
		}
	}
	return nil
}
