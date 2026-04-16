package woocommerce

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// VerifySignature checks X-WC-Webhook-Signature (base64(HMAC-SHA256(body, secret))).
func VerifySignature(secret string, body []byte, signatureB64 string) bool {
	if secret == "" || signatureB64 == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)
	got, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, got)
}
