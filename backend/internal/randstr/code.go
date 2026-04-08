package randstr

import (
	"crypto/rand"
	"encoding/hex"
)

// Hex returns n random bytes as hex string (2n hex chars).
func Hex(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
