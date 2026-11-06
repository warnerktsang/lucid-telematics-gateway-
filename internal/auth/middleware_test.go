package auth

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/test"
	"net/http"
	"testing"
	"time"
)

const testKey = "test"

func signedToken(t *testing.T, key string, method jwt.SigningMethod, claims jwt.Claims) string {
	var signingKey interface{} = []byte(key)
	if method == jwt.SigningMethodNone {
		signingKey = jwt.UnsafeAllowNoneSignatureType
	}
	token, err := jwt.NewWithClaims(method, claims).SignedString(signingKey)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func vehicleClaims(vin, fleetID string, exp time.Time) VehicleClaims {
	return VehicleClaims{
		VIN:     vin,
		FleetID: fleetID,
		StandardClaims: jwt.StandardClaims{
			Subject:   "100",
			ExpiresAt: exp.Unix(),
		},
	}
}

func TestCurrentVehicle(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, CurrentVehicle(ctx))
	ctx = WithVehicle(ctx, "50EA1TEA1PA004821", "fleet-sf-01")
	vehicle := CurrentVehicle(ctx)
	if assert.NotNil(t, vehicle) {
		assert.Equal(t, "50EA1TEA1PA004821", vehicle.GetVIN())
		assert.Equal(t, "fleet-sf-01", vehicle.GetFleetID())
	}
}

func TestHandler(t *testing.T) {
	assert.NotNil(t, Handler(testKey))
}

func TestHandler_ValidToken(t *testing.T) {
	token := signedToken(t, testKey, jwt.SigningMethodHS256, vehicleClaims("50EA1TEA1PA004821", "fleet-sf-01", time.Now().Add(time.Hour)))
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx, _ := test.MockRoutingContext(req)
	assert.Nil(t, CurrentVehicle(ctx.Request.Context()))

	err := Handler(testKey)(ctx)
	assert.Nil(t, err)
	vehicle := CurrentVehicle(ctx.Request.Context())
	if assert.NotNil(t, vehicle) {
		assert.Equal(t, "50EA1TEA1PA004821", vehicle.GetVIN())
		assert.Equal(t, "fleet-sf-01", vehicle.GetFleetID())
	}
}

func TestHandler_Rejects(t *testing.T) {
	future := time.Now().Add(time.Hour)
	tests := []struct {
		name   string
		header string
	}{
		{"missing header", ""},
		{"not bearer", "Basic abc"},
		{"empty bearer", "Bearer "},
		{"malformed token", "Bearer not.a.jwt"},
		{"wrong key", "Bearer " + signedToken(t, "other-key", jwt.SigningMethodHS256, vehicleClaims("50EA1TEA1PA004821", "fleet-sf-01", future))},
		{"expired", "Bearer " + signedToken(t, testKey, jwt.SigningMethodHS256, vehicleClaims("50EA1TEA1PA004821", "fleet-sf-01", time.Now().Add(-time.Hour)))},
		{"empty fleet_id", "Bearer " + signedToken(t, testKey, jwt.SigningMethodHS256, vehicleClaims("50EA1TEA1PA004821", "", future))},
		{"missing vehicle claims", "Bearer " + signedToken(t, testKey, jwt.SigningMethodHS256, jwt.StandardClaims{Subject: "100", ExpiresAt: future.Unix()})},
		{"none algorithm", "Bearer " + signedToken(t, testKey, jwt.SigningMethodNone, vehicleClaims("50EA1TEA1PA004821", "fleet-sf-01", future))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			ctx, res := test.MockRoutingContext(req)
			err := Handler(testKey)(ctx)
			if assert.NotNil(t, err) {
				assert.Equal(t, http.StatusUnauthorized, err.(interface{ StatusCode() int }).StatusCode())
			}
			assert.Equal(t, `Bearer realm="API"`, res.Header().Get("WWW-Authenticate"))
			assert.Nil(t, CurrentVehicle(ctx.Request.Context()))
		})
	}
}

func TestMocks(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	ctx, _ := test.MockRoutingContext(req)
	assert.NotNil(t, MockAuthHandler(ctx))
	req.Header = MockAuthHeader()
	ctx, _ = test.MockRoutingContext(req)
	assert.Nil(t, MockAuthHandler(ctx))
	vehicle := CurrentVehicle(ctx.Request.Context())
	if assert.NotNil(t, vehicle) {
		assert.Equal(t, "50EA1TEA1PA004821", vehicle.GetVIN())
		assert.Equal(t, "fleet-test-01", vehicle.GetFleetID())
	}
}
