// Package api implements HTTP endpoints, well-known routes, and request validation.
package api

import (
	"encoding/json"
)

// IdentityResponse is the well-known identity endpoint response.
type IdentityResponse struct {
	Address            string      `json:"address"`
	IdentityKey        string      `json:"identity_key"`
	SigningKeys        interface{} `json:"signing_keys"`
	RevocationKey      string      `json:"revocation_key"`
	Server             string      `json:"server"`
	Capabilities       []string    `json:"capabilities"`
	Preferences        interface{} `json:"preferences,omitempty"`
	ServerProcessing   interface{} `json:"server_processing,omitempty"`
}

// KeyPackagesResponse is the well-known keypackages endpoint response.
type KeyPackagesResponse struct {
	KeyPackages []interface{} `json:"keypackages"`
}

// ServerKeyResponse is the well-known server-key endpoint response.
type ServerKeyResponse struct {
	Key    string `json:"key"`
	Domain string `json:"domain"`
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RequestValidator validates HTTP requests.
type RequestValidator struct{}

// ValidateJSON validates and parses JSON request body.
func (rv *RequestValidator) ValidateJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ValidateRequired checks if required fields are present.
func (rv *RequestValidator) ValidateRequired(data map[string]interface{}, requiredFields []string) bool {
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			return false
		}
	}
	return true
}

// EncodeJSON encodes a value as JSON.
func EncodeJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// DecodeJSON decodes JSON data into a value.
func DecodeJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
