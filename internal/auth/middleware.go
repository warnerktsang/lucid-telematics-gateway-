package auth

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	routing "github.com/go-ozzo/ozzo-routing/v2"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/entity"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/errors"
	"net/http"
	"strings"
)

// Handler returns a JWT-based authentication middleware that validates per-vehicle tokens.
// The token is expected in the "Authorization: Bearer <token>" header and must carry
// VehicleClaims with a non-empty fleet_id.
func Handler(verificationKey string) routing.Handler {
	return func(c *routing.Context) error {
		tokenString, ok := bearerToken(c.Request)
		if !ok {
			return unauthorized(c)
		}

		claims := &VehicleClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(verificationKey), nil
		})
		if err != nil || !token.Valid {
			return unauthorized(c)
		}
		if claims.FleetID == "" {
			return unauthorized(c)
		}

		ctx := WithVehicle(c.Request.Context(), claims.VIN, claims.FleetID)
		c.Request = c.Request.WithContext(ctx)
		return nil
	}
}

// bearerToken extracts the bearer token from the Authorization header of the request.
func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if len(header) < 7 || !strings.EqualFold(header[:7], "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(header[7:])
	return token, token != ""
}

func unauthorized(c *routing.Context) error {
	c.Response.Header().Set("WWW-Authenticate", `Bearer realm="API"`)
	return errors.Unauthorized("")
}

type contextKey int

const (
	vehicleKey contextKey = iota
)

// WithVehicle returns a context that contains the vehicle identity from the given JWT.
func WithVehicle(ctx context.Context, vin, fleetID string) context.Context {
	return context.WithValue(ctx, vehicleKey, entity.Vehicle{VIN: vin, FleetID: fleetID})
}

// CurrentVehicle returns the vehicle identity from the given context.
// Nil is returned if no vehicle identity is found in the context.
func CurrentVehicle(ctx context.Context) Vehicle {
	if vehicle, ok := ctx.Value(vehicleKey).(entity.Vehicle); ok {
		return vehicle
	}
	return nil
}

// MockAuthHandler creates a mock authentication middleware for testing purpose.
// If the request contains an Authorization header whose value is "TEST", then
// it considers the request is authenticated as vehicle "50EA1TEA1PA004821" of fleet "fleet-test-01".
// It fails the authentication otherwise.
func MockAuthHandler(c *routing.Context) error {
	if c.Request.Header.Get("Authorization") != "TEST" {
		return errors.Unauthorized("")
	}
	ctx := WithVehicle(c.Request.Context(), "50EA1TEA1PA004821", "fleet-test-01")
	c.Request = c.Request.WithContext(ctx)
	return nil
}

// MockAuthHeader returns an HTTP header that can pass the authentication check by MockAuthHandler.
func MockAuthHeader() http.Header {
	header := http.Header{}
	header.Add("Authorization", "TEST")
	return header
}
