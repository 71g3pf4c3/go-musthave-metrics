package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const HeaderHashSHA256 = "HashSHA256"

func ComputeHMAC(body []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func EqualHMAC(gotHeader string, body []byte, key string) bool {
	got, err := hex.DecodeString(gotHeader)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)

	return hmac.Equal(got, expected)
}
