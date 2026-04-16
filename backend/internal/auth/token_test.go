package auth

import (
	"testing"

	"github.com/stelgkio/affilflow/backend/internal/config"
)

func TestIssueAndParseJWT(t *testing.T) {
	cfg := &config.Config{
		AuthJWTSecret:   "test-secret-at-least-32-bytes-long-ok",
		AuthJWTIssuer:   "affilflow",
		AuthJWTAudience: "affilflow-api",
	}
	raw, err := IssueAccessToken(cfg, "user-123", []string{"affiliate"})
	if err != nil {
		t.Fatal(err)
	}
	sub, roles, err := ParseAndValidate(cfg, raw)
	if err != nil {
		t.Fatal(err)
	}
	if sub != "user-123" {
		t.Fatalf("sub: %q", sub)
	}
	if len(roles) != 1 || roles[0] != "affiliate" {
		t.Fatalf("roles: %v", roles)
	}
}

func TestParseInvalidSecret(t *testing.T) {
	cfg := &config.Config{
		AuthJWTSecret:   "test-secret-at-least-32-bytes-long-ok",
		AuthJWTIssuer:   "affilflow",
		AuthJWTAudience: "affilflow-api",
	}
	raw, err := IssueAccessToken(cfg, "u1", []string{"merchant"})
	if err != nil {
		t.Fatal(err)
	}
	bad := &config.Config{
		AuthJWTSecret:   "other-secret-also-32-bytes-minimum-x",
		AuthJWTIssuer:   "affilflow",
		AuthJWTAudience: "affilflow-api",
	}
	_, _, err = ParseAndValidate(bad, raw)
	if err == nil {
		t.Fatal("expected error")
	}
}
