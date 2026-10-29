package service

import (
	"encoding/hex"
	"testing"
)

func TestCallbackSignature(t *testing.T) {
	const secret = "callback-secret"
	const timestamp = "1793311200"
	body := []byte(`{"tx_id":"tx-123","status":"SUCCESS"}`)

	signature := callbackSignature(secret, timestamp, body)
	if _, err := hex.DecodeString(signature); err != nil {
		t.Fatalf("signature is not valid hex: %v", err)
	}
	if len(signature) != 64 {
		t.Fatalf("signature length = %d, want 64 hex characters", len(signature))
	}
	if signature != "45be82877683972a0003caa0af515508ad5c362aaeeaf7ad7508a9e94f747f1c" {
		t.Fatalf("signature = %q, want known HMAC-SHA256 vector", signature)
	}
	if signature != callbackSignature(secret, timestamp, body) {
		t.Fatal("same input produced different signatures")
	}
	if signature == callbackSignature(secret, timestamp, []byte(`{"tx_id":"tx-123","status":"FAILED"}`)) {
		t.Fatal("signature did not change when body changed")
	}
	if signature == callbackSignature(secret, "1793311201", body) {
		t.Fatal("signature did not change when timestamp changed")
	}
	if signature == callbackSignature("different-secret", timestamp, body) {
		t.Fatal("signature did not change when secret changed")
	}
}
