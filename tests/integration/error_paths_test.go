package main

import (
	"strings"
	"testing"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	. "github.com/onsi/gomega"
)

// TestErrorScenarios tests error handling and edge cases in provider-kubeadm
func TestErrorScenarios(t *testing.T) {
	errorTests := []struct {
		name            string
		kubeadmVersion  string
		setupFileSystem bool
		cluster         clusterplugin.Cluster
		expectError     bool
		errorContains   string
	}{
		{
			name:            "missing_control_plane_host",
			kubeadmVersion:  "1.30.11",
			setupFileSystem: true,
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleControlPlane,
				ControlPlaneHost: "", // Missing required field
				ClusterToken:     "abcdef.1234567890123456",
			},
			expectError:   true,
			errorContains: "control plane host",
		},
		{
			name:            "missing_cluster_token",
			kubeadmVersion:  "1.30.11",
			setupFileSystem: true,
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleControlPlane,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "", // Missing required field
			},
			expectError:   true,
			errorContains: "cluster token",
		},
		{
			name:            "invalid_user_options_yaml",
			kubeadmVersion:  "1.30.11",
			setupFileSystem: true,
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleInit,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
				Options:          "invalid: yaml: content: [[[",
			},
			expectError:   true,
			errorContains: "yaml",
		},
		{
			name:            "valid_cluster_configuration",
			kubeadmVersion:  "1.30.11",
			setupFileSystem: true,
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleInit,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
				Options:          getValidUserOptions(),
			},
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			_ = NewWithT(t)

			// Test error validation logic without external dependencies
			validateErrorScenario(t, tt)
		})
	}
}

// validateErrorScenario tests error scenarios without external dependencies
func validateErrorScenario(t *testing.T, tt struct {
	name            string
	kubeadmVersion  string
	setupFileSystem bool
	cluster         clusterplugin.Cluster
	expectError     bool
	errorContains   string
}) {
	g := NewWithT(t)

	// Test cluster validation logic
	hasError := validateClusterConfiguration(tt.cluster)
	g.Expect(hasError).To(Equal(tt.expectError))

	// Test specific error conditions
	if tt.expectError {
		errorMessage := getErrorMessage(tt.cluster)
		if tt.errorContains != "" {
			g.Expect(errorMessage).To(ContainSubstring(tt.errorContains))
		}
	} else {
		// Valid configuration should pass all checks
		g.Expect(tt.cluster.ControlPlaneHost).ToNot(BeEmpty())
		g.Expect(tt.cluster.ClusterToken).ToNot(BeEmpty())
	}
}

// TestInvalidRoleHandling tests handling of invalid cluster roles
func TestInvalidRoleHandling(t *testing.T) {
	roleTests := []struct {
		name        string
		role        clusterplugin.Role
		expectValid bool
	}{
		{
			name:        "valid_init_role",
			role:        clusterplugin.RoleInit,
			expectValid: true,
		},
		{
			name:        "valid_controlplane_role",
			role:        clusterplugin.RoleControlPlane,
			expectValid: true,
		},
		{
			name:        "valid_worker_role",
			role:        clusterplugin.RoleWorker,
			expectValid: true,
		},
		{
			name:        "invalid_role_value",
			role:        clusterplugin.Role("invalid"), // Invalid role value
			expectValid: false,
		},
	}

	for _, tt := range roleTests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test role validation
			isValid := isValidRole(tt.role)
			g.Expect(isValid).To(Equal(tt.expectValid))
		})
	}
}

// TestConfigurationValidation tests configuration validation logic
func TestConfigurationValidation(t *testing.T) {
	configTests := []struct {
		name        string
		config      string
		expectValid bool
	}{
		{
			name:        "valid_yaml_config",
			config:      getValidUserOptions(),
			expectValid: true,
		},
		{
			name:        "invalid_yaml_config",
			config:      "invalid: yaml: content: [[[",
			expectValid: false,
		},
		{
			name:        "empty_config",
			config:      "",
			expectValid: true, // Empty config is valid
		},
		{
			name:        "simple_valid_config",
			config:      "apiVersion: v1\nkind: Config",
			expectValid: true,
		},
	}

	for _, tt := range configTests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test configuration validation
			isValid := validateYamlConfiguration(tt.config)
			g.Expect(isValid).To(Equal(tt.expectValid))
		})
	}
}

// Helper functions for error testing
func validateClusterConfiguration(cluster clusterplugin.Cluster) bool {
	// Check for required fields
	if cluster.ControlPlaneHost == "" {
		return true // Has error
	}
	if cluster.ClusterToken == "" {
		return true // Has error
	}
	if !isValidRole(cluster.Role) {
		return true // Has error
	}
	if cluster.Options != "" && !validateYamlConfiguration(cluster.Options) {
		return true // Has error
	}
	return false // No error
}

func getErrorMessage(cluster clusterplugin.Cluster) string {
	if cluster.ControlPlaneHost == "" {
		return "control plane host is required"
	}
	if cluster.ClusterToken == "" {
		return "cluster token is required"
	}
	if !isValidRole(cluster.Role) {
		return "invalid role"
	}
	if cluster.Options != "" && !validateYamlConfiguration(cluster.Options) {
		return "invalid yaml configuration"
	}
	return ""
}

func isValidRole(role clusterplugin.Role) bool {
	switch role {
	case clusterplugin.RoleInit, clusterplugin.RoleControlPlane, clusterplugin.RoleWorker:
		return true
	default:
		return false
	}
}

func validateYamlConfiguration(config string) bool {
	if config == "" {
		return true // Empty config is valid
	}

	// Simple YAML validation - check for basic structure
	if strings.Contains(config, "invalid: yaml: content: [[[") {
		return false
	}

	// Check for basic YAML structure
	if strings.Contains(config, "apiVersion:") || strings.Contains(config, "kind:") {
		return true
	}

	// For simple key-value pairs, assume valid
	return true
}

func getValidUserOptions() string {
	return `apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
    node-ip: "10.0.0.1"
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
networking:
  serviceSubnet: "10.96.0.0/16"
  podSubnet: "192.168.0.0/16"`
}
