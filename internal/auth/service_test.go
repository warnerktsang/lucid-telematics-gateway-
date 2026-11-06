package auth

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/entity"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/errors"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/test"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/log"
	"net/http"
	"testing"
)

func Test_service_Authenticate(t *testing.T) {
	logger, _ := log.NewForTest()
	s := NewService("test", 100, logger)
	_, err := s.Login(context.Background(), "unknown", "bad")
	assert.Equal(t, errors.Unauthorized(""), err)
	token, err := s.Login(context.Background(), "demo", "pass")
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
}

func Test_service_authenticate(t *testing.T) {
	logger, _ := log.NewForTest()
	s := service{"test", 100, logger}
	assert.Nil(t, s.authenticate(context.Background(), "unknown", "bad"))
	assert.NotNil(t, s.authenticate(context.Background(), "demo", "pass"))
}

func Test_service_GenerateJWT(t *testing.T) {
	logger, _ := log.NewForTest()
	s := service{"test", 100, logger}
	token, err := s.generateJWT(entity.User{ID: "100", Name: "demo"}, entity.Vehicle{VIN: "50EA1TEA1PA004821", FleetID: "fleet-sf-01"})
	if assert.Nil(t, err) {
		assert.NotEmpty(t, token)
	}

	claims := &VehicleClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte("test"), nil
	})
	if assert.Nil(t, err) {
		assert.True(t, parsed.Valid)
		assert.Equal(t, jwt.SigningMethodHS256, parsed.Method)
		assert.Equal(t, "50EA1TEA1PA004821", claims.VIN)
		assert.Equal(t, "fleet-sf-01", claims.FleetID)
		assert.Equal(t, "100", claims.Subject)
		assert.NotZero(t, claims.ExpiresAt)
	}
}

func Test_service_LoginIssuesVehicleClaims(t *testing.T) {
	logger, _ := log.NewForTest()
	s := NewService("test", 100, logger)
	token, err := s.Login(context.Background(), "demo", "pass")
	assert.Nil(t, err)

	claims := &VehicleClaims{}
	_, err = jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte("test"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "50EA1TEA1PA004821", claims.VIN)
	assert.Equal(t, "fleet-sf-01", claims.FleetID)
}

func Test_service_LoginTokenPassesHandler(t *testing.T) {
	logger, _ := log.NewForTest()
	s := NewService("test", 100, logger)
	token, err := s.Login(context.Background(), "demo", "pass")
	assert.Nil(t, err)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx, _ := test.MockRoutingContext(req)
	assert.Nil(t, Handler("test")(ctx))
	vehicle := CurrentVehicle(ctx.Request.Context())
	if assert.NotNil(t, vehicle) {
		assert.Equal(t, "fleet-sf-01", vehicle.GetFleetID())
	}
}
