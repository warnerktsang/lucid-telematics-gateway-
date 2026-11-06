package auth

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/entity"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/errors"
	"github.com/warnerktsang/lucid-telematics-gateway/pkg/log"
	"time"
)

// Service encapsulates the authentication logic.
type Service interface {
	// authenticate authenticates a user using username and password.
	// It returns a JWT token if authentication succeeds. Otherwise, an error is returned.
	Login(ctx context.Context, username, password string) (string, error)
}

// Identity represents an authenticated user identity.
type Identity interface {
	// GetID returns the user ID.
	GetID() string
	// GetName returns the user name.
	GetName() string
}

// Vehicle represents the authenticated vehicle identity carried by a JWT.
type Vehicle interface {
	// GetVIN returns the vehicle identification number.
	GetVIN() string
	// GetFleetID returns the ID of the fleet the vehicle belongs to.
	GetFleetID() string
}

// VehicleClaims are the custom JWT claims issued to a vehicle.
type VehicleClaims struct {
	VIN     string `json:"vin"`
	FleetID string `json:"fleet_id"`
	jwt.RegisteredClaims
}

type service struct {
	signingKey      string
	tokenExpiration int
	logger          log.Logger
}

// NewService creates a new authentication service.
func NewService(signingKey string, tokenExpiration int, logger log.Logger) Service {
	return service{signingKey, tokenExpiration, logger}
}

// Login authenticates a user and generates a JWT token if authentication succeeds.
// Otherwise, an error is returned.
func (s service) Login(ctx context.Context, username, password string) (string, error) {
	if identity := s.authenticate(ctx, username, password); identity != nil {
		return s.generateJWT(identity, s.vehicleFor(identity))
	}
	return "", errors.Unauthorized("")
}

// authenticate authenticates a user using username and password.
// If username and password are correct, an identity is returned. Otherwise, nil is returned.
func (s service) authenticate(ctx context.Context, username, password string) Identity {
	logger := s.logger.With(ctx, "user", username)

	// TODO: the following authentication logic is only for demo purpose
	if username == "demo" && password == "pass" {
		logger.Infof("authentication successful")
		return entity.User{ID: "100", Name: "demo"}
	}

	logger.Infof("authentication failed")
	return nil
}

// vehicleFor resolves the vehicle that the authenticated identity is allowed to report telemetry for.
func (s service) vehicleFor(identity Identity) Vehicle {
	// TODO: the following vehicle lookup is only for demo purpose
	return entity.Vehicle{VIN: "50EA1TEA1PA004821", FleetID: "fleet-sf-01"}
}

// generateJWT generates a JWT that encodes an identity and its vehicle claims.
func (s service) generateJWT(identity Identity, vehicle Vehicle) (string, error) {
	now := time.Now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, VehicleClaims{
		VIN:     vehicle.GetVIN(),
		FleetID: vehicle.GetFleetID(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   identity.GetID(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.tokenExpiration) * time.Hour)),
		},
	}).SignedString([]byte(s.signingKey))
}
