package api

import (
	"testing"
)

func TestEncodeDecodeJSON(t *testing.T) {
	data := map[string]string{"key": "value"}

	encoded, err := EncodeJSON(data)
	if err != nil {
		t.Fatalf("EncodeJSON error: %v", err)
	}

	var decoded map[string]string
	if err := DecodeJSON(encoded, &decoded); err != nil {
		t.Fatalf("DecodeJSON error: %v", err)
	}

	if decoded["key"] != "value" {
		t.Error("Decoded value mismatch")
	}
}

func TestValidateRequired(t *testing.T) {
	rv := &RequestValidator{}

	data := map[string]interface{}{
		"address": "alice@example.com",
		"token":   "abc123",
	}

	required := []string{"address", "token"}
	if !rv.ValidateRequired(data, required) {
		t.Error("ValidateRequired should pass with all fields")
	}

	missing := []string{"address", "missing"}
	if rv.ValidateRequired(data, missing) {
		t.Error("ValidateRequired should fail with missing fields")
	}
}

func TestIdentityResponse(t *testing.T) {
	resp := &IdentityResponse{
		Address:      "alice@example.com",
		IdentityKey:  "base64_key",
		RevocationKey: "base64_revkey",
		Capabilities: []string{"ucp/1.0"},
	}

	data, _ := EncodeJSON(resp)

	var decoded IdentityResponse
	DecodeJSON(data, &decoded)

	if decoded.Address != "alice@example.com" {
		t.Error("Address mismatch in round-trip")
	}
}
