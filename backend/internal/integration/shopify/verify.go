package shopify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// VerifyWebhook reports whether the HMAC header matches the raw body (Shopify order webhooks).
func VerifyWebhook(secret string, body []byte, hmacHeader string) bool {
	if secret == "" || hmacHeader == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)
	got, err := base64.StdEncoding.DecodeString(hmacHeader)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, got)
}
