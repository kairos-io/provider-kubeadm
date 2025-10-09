package utils

import (
	"os"
	"testing"
)

func TestGetKubeadmBinaryPath(t *testing.T) {
	tests := []struct {
		name           string
		stylusRoot     string
		expectedResult string
	}{
		{
			name:           "No STYLUS_ROOT set (appliance mode)",
			stylusRoot:     "",
			expectedResult: "kubeadm",
		},
		{
			name:           "STYLUS_ROOT set but kubeadm not found (agent mode fallback)",
			stylusRoot:     "/nonexistent/root",
			expectedResult: "kubeadm", // Falls back to system PATH when file doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.stylusRoot != "" {
				os.Setenv("STYLUS_ROOT", tt.stylusRoot)
			} else {
				os.Unsetenv("STYLUS_ROOT")
			}
			defer os.Unsetenv("STYLUS_ROOT")

			// Test the function
			result := GetKubeadmBinaryPath()

			// Verify the result
			if result != tt.expectedResult {
				t.Errorf("GetKubeadmBinaryPath() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}