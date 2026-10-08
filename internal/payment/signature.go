package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func Verify(body []byte, secret, signature string) bool {
	if signature == "" {
		return false
	}
	expected := Sign(body, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}
