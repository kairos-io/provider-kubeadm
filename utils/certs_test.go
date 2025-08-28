package utils

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestGetCertificateKey tests the GetCertificateKey function
func TestGetCertificateKey(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		expectedLength int
	}{
		{
			name:           "standard_token",
			token:          "test-token.1234567890123456",
			expectedLength: 64, // SHA256 hash length
		},
		{
			name:           "token_with_dots",
			token:          "token.with.dots.1234567890123456",
			expectedLength: 64,
		},
		{
			name:           "token_without_dots",
			token:          "tokenwithoutdots1234567890123456",
			expectedLength: 64,
		},
		{
			name:           "empty_token",
			token:          "",
			expectedLength: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetCertificateKey(tt.token)

			g.Expect(result).To(HaveLen(tt.expectedLength))
			g.Expect(result).To(MatchRegexp(`^[a-f0-9]{64}$`)) // SHA256 hex format
		})
	}
}

// TestTransformToken tests the TransformToken function
func TestTransformToken(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		expectedFormat string
	}{
		{
			name:           "token_with_dots",
			token:          "test-token.1234567890123456",
			expectedFormat: `^[a-f0-9]{6}\.[a-f0-9]{16}$`,
		},
		{
			name:           "token_with_multiple_dots",
			token:          "token.with.dots.1234567890123456",
			expectedFormat: `^[a-f0-9]{6}\.[a-f0-9]{16}$`,
		},
		{
			name:           "token_without_dots",
			token:          "tokenwithoutdots1234567890123456",
			expectedFormat: `^[a-f0-9]{6}\.[a-f0-9]{16}$`,
		},
		{
			name:           "empty_token",
			token:          "",
			expectedFormat: `^[a-f0-9]{6}\.[a-f0-9]{16}$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := TransformToken(tt.token)

			g.Expect(result).To(MatchRegexp(tt.expectedFormat))
		})
	}
}

// TestGetCertSansRevision tests the GetCertSansRevision function
func TestGetCertSansRevision(t *testing.T) {
	tests := []struct {
		name           string
		certSANs       []string
		expectedFormat string
	}{
		{
			name:           "single_cert_san",
			certSANs:       []string{"cluster.example.com"},
			expectedFormat: `^[a-f0-9]+$`,
		},
		{
			name:           "multiple_cert_sans",
			certSANs:       []string{"cluster.example.com", "api.cluster.example.com", "10.0.0.1"},
			expectedFormat: `^[a-f0-9]+$`,
		},
		{
			name:           "empty_cert_sans",
			certSANs:       []string{},
			expectedFormat: `^[a-f0-9]+$`,
		},
		{
			name:           "nil_cert_sans",
			certSANs:       nil,
			expectedFormat: `^[a-f0-9]+$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetCertSansRevision(tt.certSANs)

			g.Expect(result).To(MatchRegexp(tt.expectedFormat))
		})
	}
}
