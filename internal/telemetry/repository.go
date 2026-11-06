package telemetry

import (
	"context"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/entity"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/dbcontext"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/log"
)

// Repository encapsulates the logic to access telemetry batches from the data source.
type Repository interface {
	// Get returns the telemetry batch with the specified batch ID.
	Get(ctx context.Context, id string) (entity.TelemetryBatch, error)
	// Count returns the number of telemetry batches.
	Count(ctx context.Context) (int, error)
	// Query returns the list of telemetry batches with the given offset and limit.
	Query(ctx context.Context, offset, limit int) ([]entity.TelemetryBatch, error)
	// Create saves a new telemetry batch in the storage.
	Create(ctx context.Context, batch entity.TelemetryBatch) error
	// Update updates the telemetry batch with given ID in the storage.
	Update(ctx context.Context, batch entity.TelemetryBatch) error
	// Delete removes the telemetry batch with given ID from the storage.
	Delete(ctx context.Context, id string) error
}

// repository persists telemetry batches in database
type repository struct {
	db     *dbcontext.DB
	logger log.Logger
}

// NewRepository creates a new telemetry batch repository
func NewRepository(db *dbcontext.DB, logger log.Logger) Repository {
	return repository{db, logger}
}

// Get reads the telemetry batch with the specified ID from the database.
func (r repository) Get(ctx context.Context, id string) (entity.TelemetryBatch, error) {
	var batch entity.TelemetryBatch
	err := r.db.With(ctx).Select().Model(id, &batch)
	return batch, err
}

// Create saves a new telemetry batch record in the database.
// It returns the ID of the newly inserted telemetry batch record.
func (r repository) Create(ctx context.Context, batch entity.TelemetryBatch) error {
	return r.db.With(ctx).Model(&batch).Insert()
}

// Update saves the changes to a telemetry batch in the database.
func (r repository) Update(ctx context.Context, batch entity.TelemetryBatch) error {
	return r.db.With(ctx).Model(&batch).Update()
}

// Delete deletes a telemetry batch with the specified ID from the database.
func (r repository) Delete(ctx context.Context, id string) error {
	batch, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	return r.db.With(ctx).Model(&batch).Delete()
}

// Count returns the number of the telemetry batch records in the database.
func (r repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.With(ctx).Select("COUNT(*)").From("telemetry_batch").Row(&count)
	return count, err
}

// Query retrieves the telemetry batch records with the specified offset and limit from the database.
func (r repository) Query(ctx context.Context, offset, limit int) ([]entity.TelemetryBatch, error) {
	var batches []entity.TelemetryBatch
	err := r.db.With(ctx).
		Select().
		OrderBy("id").
		Offset(int64(offset)).
		Limit(int64(limit)).
		All(&batches)
	return batches, err
}
