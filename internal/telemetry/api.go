package telemetry

import (
	"github.com/go-ozzo/ozzo-routing/v2"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/errors"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/log"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/pagination"
	"net/http"
)

// RegisterHandlers sets up the routing of the HTTP handlers.
func RegisterHandlers(r *routing.RouteGroup, service Service, authHandler routing.Handler, logger log.Logger) {
	res := resource{service, logger}

	r.Get("/telemetry-batches/<id>", res.get)
	r.Get("/telemetry-batches", res.query)

	r.Use(authHandler)

	// the following endpoints require a valid JWT
	r.Post("/telemetry-batches", res.create)
	r.Put("/telemetry-batches/<id>", res.update)
	r.Delete("/telemetry-batches/<id>", res.delete)
}

type resource struct {
	service Service
	logger  log.Logger
}

func (r resource) get(c *routing.Context) error {
	batch, err := r.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	return c.Write(batch)
}

func (r resource) query(c *routing.Context) error {
	ctx := c.Request.Context()
	count, err := r.service.Count(ctx)
	if err != nil {
		return err
	}
	pages := pagination.NewFromRequest(c.Request, count)
	batches, err := r.service.Query(ctx, pages.Offset(), pages.Limit())
	if err != nil {
		return err
	}
	pages.Items = batches
	return c.Write(pages)
}

func (r resource) create(c *routing.Context) error {
	var input CreateTelemetryBatchRequest
	if err := c.Read(&input); err != nil {
		r.logger.With(c.Request.Context()).Info(err)
		return errors.BadRequest("")
	}
	batch, err := r.service.Create(c.Request.Context(), input)
	if err != nil {
		return err
	}

	return c.WriteWithStatus(batch, http.StatusCreated)
}

func (r resource) update(c *routing.Context) error {
	var input UpdateTelemetryBatchRequest
	if err := c.Read(&input); err != nil {
		r.logger.With(c.Request.Context()).Info(err)
		return errors.BadRequest("")
	}

	batch, err := r.service.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	return c.Write(batch)
}

func (r resource) delete(c *routing.Context) error {
	batch, err := r.service.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	return c.Write(batch)
}
