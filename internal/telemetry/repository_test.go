package telemetry

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/assert"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/entity"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/test"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/log"
	"testing"
	"time"
)

func TestRepository(t *testing.T) {
	logger, _ := log.NewForTest()
	db := test.DB(t)
	test.ResetTables(t, db, "telemetry_batch")
	repo := NewRepository(db, logger)

	ctx := context.Background()

	// initial count
	count, err := repo.Count(ctx)
	assert.Nil(t, err)

	// create
	err = repo.Create(ctx, entity.TelemetryBatch{
		ID:         "test1",
		VIN:        "50EA1TEA1PA004821",
		EventCount: 10,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	assert.Nil(t, err)
	count2, _ := repo.Count(ctx)
	assert.Equal(t, 1, count2-count)

	// get
	batch, err := repo.Get(ctx, "test1")
	assert.Nil(t, err)
	assert.Equal(t, "50EA1TEA1PA004821", batch.VIN)
	assert.Equal(t, 10, batch.EventCount)
	_, err = repo.Get(ctx, "test0")
	assert.Equal(t, sql.ErrNoRows, err)

	// update
	err = repo.Update(ctx, entity.TelemetryBatch{
		ID:         "test1",
		VIN:        "50EA1TEA1PA004821",
		EventCount: 20,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	assert.Nil(t, err)
	batch, _ = repo.Get(ctx, "test1")
	assert.Equal(t, 20, batch.EventCount)

	// query
	batches, err := repo.Query(ctx, 0, count2)
	assert.Nil(t, err)
	assert.Equal(t, count2, len(batches))

	// delete
	err = repo.Delete(ctx, "test1")
	assert.Nil(t, err)
	_, err = repo.Get(ctx, "test1")
	assert.Equal(t, sql.ErrNoRows, err)
	err = repo.Delete(ctx, "test1")
	assert.Equal(t, sql.ErrNoRows, err)
}
