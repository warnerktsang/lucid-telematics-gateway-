package auth

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/warnerktsang/lucid-telematics-gateway/internal/test"
)

// The tests in this file build tokens with the standard library only, so they
// exercise the wire format the middleware accepts independently of whichever
// JWT library implements Handler.

const (
	regressionKey = "regression-test-key"
	regressionVIN = "50EA1TEA1PA004821"
	regressionFID = "fleet-sf-01"
)

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// hs256Token returns header.payload.signature signed with HMAC-SHA256 over the given key.
func hs256Token(t *testing.T, key string, header, payload map[string]interface{}) string {
	t.Helper()
	signingInput := b64(mustJSON(t, header)) + "." + b64(mustJSON(t, payload))
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(signingInput))
	return signingInput + "." + b64(mac.Sum(nil))
}

func hsHeader(alg string) map[string]interface{} {
	return map[string]interface{}{"alg": alg, "typ": "JWT"}
}

func vehiclePayload(exp time.Time) map[string]interface{} {
	return map[string]interface{}{
		"vin":      regressionVIN,
		"fleet_id": regressionFID,
		"sub":      "100",
		"exp":      exp.Unix(),
	}
}

func authorize(t *testing.T, key, token string) (http.Header, Vehicle, error) {
	t.Helper()
	req, _ := http.NewRequest("POST", "http://example.com/v1/telemetry-batches", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx, res := test.MockRoutingContext(req)
	err := Handler(key)(ctx)
	return res.Header(), CurrentVehicle(ctx.Request.Context()), err
}

func assertUnauthorized(t *testing.T, key, token string) {
	t.Helper()
	header, vehicle, err := authorize(t, key, token)
	if assert.Error(t, err) {
		assert.Equal(t, http.StatusUnauthorized, err.(interface{ StatusCode() int }).StatusCode())
	}
	assert.Equal(t, `Bearer realm="API"`, header.Get("WWW-Authenticate"))
	assert.Nil(t, vehicle, "no vehicle identity must be attached to a rejected request")
}

func TestTokenValidation_ValidRoundTrip(t *testing.T) {
	token := hs256Token(t, regressionKey, hsHeader("HS256"), vehiclePayload(time.Now().Add(time.Hour)))
	header, vehicle, err := authorize(t, regressionKey, token)
	assert.NoError(t, err)
	assert.Empty(t, header.Get("WWW-Authenticate"))
	if assert.NotNil(t, vehicle) {
		assert.Equal(t, regressionVIN, vehicle.GetVIN())
		assert.Equal(t, regressionFID, vehicle.GetFleetID())
	}
}

func TestTokenValidation_IssuedTokenWireFormat(t *testing.T) {
	// A token issued by the service must decode with the standard library into the
	// same claim names and numeric types that were issued before the library swap.
	s := service{regressionKey, 72, nil}
	token, err := s.generateJWT(testIdentity{"100"}, testVehicle{regressionVIN, regressionFID})
	if !assert.NoError(t, err) {
		return
	}
	parts := strings.Split(token, ".")
	if !assert.Len(t, parts, 3) {
		return
	}
	var header map[string]interface{}
	hb, _ := base64.RawURLEncoding.DecodeString(parts[0])
	assert.NoError(t, json.Unmarshal(hb, &header))
	assert.Equal(t, "HS256", header["alg"])

	var payload map[string]json.RawMessage
	pb, _ := base64.RawURLEncoding.DecodeString(parts[1])
	assert.NoError(t, json.Unmarshal(pb, &payload))
	assert.JSONEq(t, `"`+regressionVIN+`"`, string(payload["vin"]))
	assert.JSONEq(t, `"`+regressionFID+`"`, string(payload["fleet_id"]))
	assert.JSONEq(t, `"100"`, string(payload["sub"]))
	for _, claim := range []string{"iat", "exp"} {
		var seconds int64
		assert.NoError(t, json.Unmarshal(payload[claim], &seconds), "%s must be an integer number of seconds", claim)
		assert.InDelta(t, time.Now().Unix(), seconds, float64(73*3600), "%s must be close to now", claim)
	}
	assert.Equal(t, 72*3600, int(mustInt64(t, payload["exp"])-mustInt64(t, payload["iat"])))

	// And the middleware must accept the token it issued, with the same key.
	_, vehicle, err := authorize(t, regressionKey, token)
	assert.NoError(t, err)
	if assert.NotNil(t, vehicle) {
		assert.Equal(t, regressionFID, vehicle.GetFleetID())
	}
}

func TestTokenValidation_Expired(t *testing.T) {
	token := hs256Token(t, regressionKey, hsHeader("HS256"), vehiclePayload(time.Now().Add(-time.Minute)))
	assertUnauthorized(t, regressionKey, token)
}

func TestTokenValidation_TamperedSignature(t *testing.T) {
	token := hs256Token(t, regressionKey, hsHeader("HS256"), vehiclePayload(time.Now().Add(time.Hour)))
	parts := strings.Split(token, ".")
	sig := []byte(parts[2])
	if sig[0] == 'A' {
		sig[0] = 'B'
	} else {
		sig[0] = 'A'
	}
	assertUnauthorized(t, regressionKey, parts[0]+"."+parts[1]+"."+string(sig))
	assertUnauthorized(t, regressionKey, parts[0]+"."+parts[1]+".")
	assertUnauthorized(t, regressionKey, parts[0]+"."+parts[1])
}

func TestTokenValidation_TamperedPayload(t *testing.T) {
	good := hs256Token(t, regressionKey, hsHeader("HS256"), vehiclePayload(time.Now().Add(time.Hour)))
	evilPayload := vehiclePayload(time.Now().Add(time.Hour))
	evilPayload["fleet_id"] = "fleet-attacker"
	evil := hs256Token(t, "attacker-key", hsHeader("HS256"), evilPayload)
	goodParts, evilParts := strings.Split(good, "."), strings.Split(evil, ".")
	// valid signature from the real key, but payload swapped for the attacker's claims
	assertUnauthorized(t, regressionKey, goodParts[0]+"."+evilParts[1]+"."+goodParts[2])
}

func TestTokenValidation_WrongKey(t *testing.T) {
	token := hs256Token(t, "not-the-configured-key", hsHeader("HS256"), vehiclePayload(time.Now().Add(time.Hour)))
	assertUnauthorized(t, regressionKey, token)
	// a verifier configured with an empty key must not accept a token signed with the real key
	assertUnauthorized(t, "", hs256Token(t, regressionKey, hsHeader("HS256"), vehiclePayload(time.Now().Add(time.Hour))))
}

func TestTokenValidation_AlgNone(t *testing.T) {
	payload := b64(mustJSON(t, vehiclePayload(time.Now().Add(time.Hour))))
	for _, alg := range []string{"none", "None", "NONE", "nOnE"} {
		header := b64(mustJSON(t, hsHeader(alg)))
		t.Run(alg+" empty signature", func(t *testing.T) {
			assertUnauthorized(t, regressionKey, header+"."+payload+".")
		})
		t.Run(alg+" no signature segment", func(t *testing.T) {
			assertUnauthorized(t, regressionKey, header+"."+payload)
		})
		t.Run(alg+" HMAC signature present", func(t *testing.T) {
			signed := hs256Token(t, regressionKey, hsHeader(alg), vehiclePayload(time.Now().Add(time.Hour)))
			assertUnauthorized(t, regressionKey, signed)
		})
	}
	t.Run("header without alg", func(t *testing.T) {
		header := b64(mustJSON(t, map[string]interface{}{"typ": "JWT"}))
		assertUnauthorized(t, regressionKey, header+"."+payload+".")
	})
}

func TestTokenValidation_Malformed(t *testing.T) {
	valid := hs256Token(t, regressionKey, hsHeader("HS256"), vehiclePayload(time.Now().Add(time.Hour)))
	parts := strings.Split(valid, ".")
	cases := map[string]string{
		"empty":                 "",
		"single segment":        "abc",
		"two segments":          "abc.def",
		"four segments":         valid + ".extra",
		"not base64 header":     "!!!." + parts[1] + "." + parts[2],
		"not base64 payload":    parts[0] + ".!!!." + parts[2],
		"not base64 signature":  parts[0] + "." + parts[1] + ".!!!",
		"header not json":       b64([]byte("not json")) + "." + parts[1] + "." + parts[2],
		"payload not json":      parts[0] + "." + b64([]byte("not json")) + "." + parts[2],
		"payload is json array": hs256Token(t, regressionKey, hsHeader("HS256"), nil) + "",
		"whitespace inside":     parts[0] + ". " + parts[1] + "." + parts[2],
	}
	// payload is not an object: sign a JSON array so the signature is valid but claims cannot be decoded
	arrPayload := b64([]byte(`["vin","fleet_id"]`))
	signingInput := parts[0] + "." + arrPayload
	mac := hmac.New(sha256.New, []byte(regressionKey))
	mac.Write([]byte(signingInput))
	cases["payload is json array"] = signingInput + "." + b64(mac.Sum(nil))
	// padded standard base64 segments are not valid JWS encoding even when the signature covers them
	paddedHeader := base64.StdEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT","kid":"k1"}`))
	if !strings.HasSuffix(paddedHeader, "=") {
		t.Fatal("test header must produce base64 padding")
	}
	signingInput = paddedHeader + "." + parts[1]
	mac = hmac.New(sha256.New, []byte(regressionKey))
	mac.Write([]byte(signingInput))
	cases["standard base64 padded"] = signingInput + "." + b64(mac.Sum(nil))

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			assertUnauthorized(t, regressionKey, token)
		})
	}
}

func TestTokenValidation_WrongClaimTypes(t *testing.T) {
	// A validly signed token whose claims have the wrong JSON type must be a clean 401, not a 500.
	cases := map[string]map[string]interface{}{
		"exp is a string":      {"vin": regressionVIN, "fleet_id": regressionFID, "exp": "tomorrow"},
		"fleet_id is a number": {"vin": regressionVIN, "fleet_id": 42, "exp": time.Now().Add(time.Hour).Unix()},
		"vin is an object":     {"vin": map[string]string{"x": "y"}, "fleet_id": regressionFID, "exp": time.Now().Add(time.Hour).Unix()},
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			assertUnauthorized(t, regressionKey, hs256Token(t, regressionKey, hsHeader("HS256"), payload))
		})
	}
}

func TestTokenValidation_SignedTokenMissingFleetID(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	cases := map[string]map[string]interface{}{
		"fleet_id absent":      {"vin": regressionVIN, "sub": "100", "exp": exp},
		"fleet_id empty":       {"vin": regressionVIN, "fleet_id": "", "sub": "100", "exp": exp},
		"fleet_id null":        {"vin": regressionVIN, "fleet_id": nil, "sub": "100", "exp": exp},
		"only standard claims": {"sub": "100", "exp": exp},
		"empty payload":        {},
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			assertUnauthorized(t, regressionKey, hs256Token(t, regressionKey, hsHeader("HS256"), payload))
		})
	}
}

func TestTokenValidation_NonDefaultSigningMethods(t *testing.T) {
	payload := vehiclePayload(time.Now().Add(time.Hour))

	t.Run("RS256 signed with a real RSA key is rejected", func(t *testing.T) {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		signingInput := b64(mustJSON(t, hsHeader("RS256"))) + "." + b64(mustJSON(t, payload))
		digest := sha256.Sum256([]byte(signingInput))
		sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest[:])
		if err != nil {
			t.Fatal(err)
		}
		assertUnauthorized(t, regressionKey, signingInput+"."+b64(sig))
	})

	t.Run("alg confusion: non-HMAC alg header with an HMAC signature over the key", func(t *testing.T) {
		for _, alg := range []string{"RS256", "ES256", "PS256", "EdDSA"} {
			assertUnauthorized(t, regressionKey, hs256Token(t, regressionKey, hsHeader(alg), payload))
		}
	})

	t.Run("unknown alg is rejected", func(t *testing.T) {
		assertUnauthorized(t, regressionKey, hs256Token(t, regressionKey, hsHeader("HS257"), payload))
	})

	t.Run("HS256 header but signature computed with a different hash is rejected", func(t *testing.T) {
		// header says HS256, signature is HMAC-SHA256 over the wrong input (payload swapped) -> invalid
		a := hs256Token(t, regressionKey, hsHeader("HS256"), payload)
		other := vehiclePayload(time.Now().Add(2 * time.Hour))
		b := hs256Token(t, regressionKey, hsHeader("HS256"), other)
		pa, pb := strings.Split(a, "."), strings.Split(b, ".")
		assertUnauthorized(t, regressionKey, pa[0]+"."+pa[1]+"."+pb[2])
	})
}

func TestTokenValidation_FutureIssuedAtAndNotBefore(t *testing.T) {
	exp := time.Now().Add(2 * time.Hour).Unix()
	t.Run("iat in the future", func(t *testing.T) {
		payload := map[string]interface{}{"vin": regressionVIN, "fleet_id": regressionFID, "exp": exp, "iat": time.Now().Add(time.Hour).Unix()}
		assertUnauthorized(t, regressionKey, hs256Token(t, regressionKey, hsHeader("HS256"), payload))
	})
	t.Run("nbf in the future", func(t *testing.T) {
		payload := map[string]interface{}{"vin": regressionVIN, "fleet_id": regressionFID, "exp": exp, "nbf": time.Now().Add(time.Hour).Unix()}
		assertUnauthorized(t, regressionKey, hs256Token(t, regressionKey, hsHeader("HS256"), payload))
	})
	t.Run("iat and nbf in the past are accepted", func(t *testing.T) {
		payload := map[string]interface{}{"vin": regressionVIN, "fleet_id": regressionFID, "exp": exp, "iat": time.Now().Add(-time.Hour).Unix(), "nbf": time.Now().Add(-time.Hour).Unix()}
		_, vehicle, err := authorize(t, regressionKey, hs256Token(t, regressionKey, hsHeader("HS256"), payload))
		assert.NoError(t, err)
		assert.NotNil(t, vehicle)
	})
}

func mustInt64(t *testing.T, raw json.RawMessage) int64 {
	t.Helper()
	var v int64
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

type testIdentity struct{ id string }

func (i testIdentity) GetID() string   { return i.id }
func (i testIdentity) GetName() string { return "demo" }

type testVehicle struct{ vin, fleetID string }

func (v testVehicle) GetVIN() string     { return v.vin }
func (v testVehicle) GetFleetID() string { return v.fleetID }
