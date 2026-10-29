package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Sign(payload interface{}, key int64) string {
	b, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, []byte(itoa(key)))
	mac.Write(b)
	return hex.EncodeToString(mac.Sum(nil))
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
