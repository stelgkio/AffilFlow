package auth

import (
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stelgkio/affilflow/backend/internal/config"
)

// IssueAccessToken creates a signed JWT for API access (roles in custom claim "roles").
func IssueAccessToken(cfg *config.Config, userID string, roles []string) ([]byte, error) {
	if cfg.AuthJWTSecret == "" {
		return nil, fmt.Errorf("AUTH_JWT_SECRET not configured")
	}
	tok, err := jwt.NewBuilder().
		Subject(userID).
		Issuer(cfg.AuthJWTIssuer).
		Audience([]string{cfg.AuthJWTAudience}).
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(24*time.Hour)).
		Claim("roles", roles).
		Build()
	if err != nil {
		return nil, err
	}
	return jwt.Sign(tok, jwt.WithKey(jwa.HS256, []byte(cfg.AuthJWTSecret)))
}

// ParseAndValidate parses Bearer JWT and returns subject and roles.
func ParseAndValidate(cfg *config.Config, tokenBytes []byte) (sub string, roles []string, err error) {
	if cfg.AuthJWTSecret == "" {
		return "", nil, fmt.Errorf("AUTH_JWT_SECRET not configured")
	}
	tok, err := jwt.Parse(tokenBytes, jwt.WithKey(jwa.HS256, []byte(cfg.AuthJWTSecret)), jwt.WithValidate(true))
	if err != nil {
		return "", nil, err
	}
	if iss := tok.Issuer(); iss != cfg.AuthJWTIssuer {
		return "", nil, fmt.Errorf("invalid issuer")
	}
	var audOK bool
	for _, a := range tok.Audience() {
		if a == cfg.AuthJWTAudience {
			audOK = true
			break
		}
	}
	if !audOK {
		return "", nil, fmt.Errorf("invalid audience")
	}
	sub = tok.Subject()
	if sub == "" {
		return "", nil, fmt.Errorf("missing sub")
	}
	if raw, ok := tok.Get("roles"); ok && raw != nil {
		switch v := raw.(type) {
		case []interface{}:
			for _, x := range v {
				if s, ok := x.(string); ok {
					roles = append(roles, s)
				}
			}
		case []string:
			roles = v
		}
	}
	return sub, roles, nil
}

// ParseAndValidateString is a convenience for raw JWT string.
func ParseAndValidateString(cfg *config.Config, raw string) (sub string, roles []string, err error) {
	return ParseAndValidate(cfg, []byte(raw))
}
