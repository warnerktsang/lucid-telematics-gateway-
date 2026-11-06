package telemetry

import (
	"github.com/warnerktsang/lucid-telematics-gateway/internal/auth"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/entity"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/test"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/log"
	"net/http"
	"testing"
	"time"
)

func TestAPI(t *testing.T) {
	logger, _ := log.NewForTest()
	router := test.MockRouter(logger)
	repo := &mockRepository{items: []entity.TelemetryBatch{
		{"123", "50EA1TEA1PA004821", 10, time.Now(), time.Now()},
	}}
	RegisterHandlers(router.Group(""), NewService(repo, logger), auth.MockAuthHandler, logger)
	header := auth.MockAuthHeader()

	tests := []test.APITestCase{
		{"get all", "GET", "/telemetry-batches", "", nil, http.StatusOK, `*"total_count":1*`},
		{"get 123", "GET", "/telemetry-batches/123", "", nil, http.StatusOK, `*50EA1TEA1PA004821*`},
		{"get unknown", "GET", "/telemetry-batches/1234", "", nil, http.StatusNotFound, ""},
		{"create ok", "POST", "/telemetry-batches", `{"vin":"50EA1TEA3PA004822","event_count":5}`, header, http.StatusCreated, "*50EA1TEA3PA004822*"},
		{"create ok count", "GET", "/telemetry-batches", "", nil, http.StatusOK, `*"total_count":2*`},
		{"create auth error", "POST", "/telemetry-batches", `{"vin":"50EA1TEA3PA004822","event_count":5}`, nil, http.StatusUnauthorized, ""},
		{"create input error", "POST", "/telemetry-batches", `"vin":"50EA1TEA3PA004822"}`, header, http.StatusBadRequest, ""},
		{"create validation error", "POST", "/telemetry-batches", `{"vin":"short","event_count":5}`, header, http.StatusBadRequest, ""},
		{"update ok", "PUT", "/telemetry-batches/123", `{"vin":"50EA1TEA5PA004823","event_count":7}`, header, http.StatusOK, "*50EA1TEA5PA004823*"},
		{"update verify", "GET", "/telemetry-batches/123", "", nil, http.StatusOK, `*"event_count":7*`},
		{"update auth error", "PUT", "/telemetry-batches/123", `{"vin":"50EA1TEA5PA004823","event_count":7}`, nil, http.StatusUnauthorized, ""},
		{"update input error", "PUT", "/telemetry-batches/123", `"vin":"50EA1TEA5PA004823"}`, header, http.StatusBadRequest, ""},
		{"delete ok", "DELETE", "/telemetry-batches/123", ``, header, http.StatusOK, "*50EA1TEA5PA004823*"},
		{"delete verify", "DELETE", "/telemetry-batches/123", ``, header, http.StatusNotFound, ""},
		{"delete auth error", "DELETE", "/telemetry-batches/123", ``, nil, http.StatusUnauthorized, ""},
	}
	for _, tc := range tests {
		test.Endpoint(t, router, tc)
	}
}
