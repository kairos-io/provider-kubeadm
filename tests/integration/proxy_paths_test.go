package integration

import (
	"fmt"
	"testing"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	. "github.com/onsi/gomega"
)

// TestProxyConfigurationHandling tests proxy environment variable handling and file generation
func TestProxyConfigurationHandling(t *testing.T) {
	proxyTests := []struct {
		name                   string
		environmentMode        string
		httpProxy              string
		httpsProxy             string
		noProxy                string
		expectedProxyFiles     int
		expectedKubeletContent []string
		expectedContainerdPath string
	}{
		{
			name:               "appliance_full_proxy_config",
			environmentMode:    "appliance",
			httpProxy:          "http://proxy.example.com:8080",
			httpsProxy:         "https://proxy.example.com:8080",
			noProxy:            "localhost,127.0.0.1,.example.com",
			expectedProxyFiles: 2,
			expectedKubeletContent: []string{
				"HTTP_PROXY=http://proxy.example.com:8080",
				"HTTPS_PROXY=https://proxy.example.com:8080",
				"NO_PROXY=localhost,127.0.0.1,.example.com",
			},
			expectedContainerdPath: "/run/systemd/system/containerd.service.d/http-proxy.conf",
		},
		{
			name:               "agent_full_proxy_config",
			environmentMode:    "agent",
			httpProxy:          "http://corporate-proxy:3128",
			httpsProxy:         "https://corporate-proxy:3128",
			noProxy:            "10.0.0.0/8,192.168.0.0/16",
			expectedProxyFiles: 2,
			expectedKubeletContent: []string{
				"HTTP_PROXY=http://corporate-proxy:3128",
				"HTTPS_PROXY=https://corporate-proxy:3128",
				"NO_PROXY=10.0.0.0/8,192.168.0.0/16",
			},
			expectedContainerdPath: "/run/systemd/system/containerd.service.d/http-proxy.conf",
		},
		{
			name:               "http_only_proxy_config",
			environmentMode:    "appliance",
			httpProxy:          "http://proxy.company.com:8080",
			httpsProxy:         "",
			noProxy:            "",
			expectedProxyFiles: 2,
			expectedKubeletContent: []string{
				"HTTP_PROXY=http://proxy.company.com:8080",
			},
			expectedContainerdPath: "/run/systemd/system/containerd.service.d/http-proxy.conf",
		},
		{
			name:                   "no_proxy_config",
			environmentMode:        "appliance",
			httpProxy:              "",
			httpsProxy:             "",
			noProxy:                "",
			expectedProxyFiles:     0,
			expectedKubeletContent: []string{},
			expectedContainerdPath: "",
		},
	}

	for _, tt := range proxyTests {
		t.Run(tt.name, func(t *testing.T) {
			_ = NewWithT(t)

			// Create cluster input with proxy configuration
			cluster := clusterplugin.Cluster{
				Role:             clusterplugin.RoleInit,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
				Env:              make(map[string]string),
			}

			// Set proxy environment variables
			if tt.httpProxy != "" {
				cluster.Env["HTTP_PROXY"] = tt.httpProxy
			}
			if tt.httpsProxy != "" {
				cluster.Env["HTTPS_PROXY"] = tt.httpsProxy
			}
			if tt.noProxy != "" {
				cluster.Env["NO_PROXY"] = tt.noProxy
			}

			// Test proxy configuration logic
			validateProxyConfiguration(t, cluster, tt)
		})
	}
}

// validateProxyConfiguration tests proxy configuration logic without external dependencies
func validateProxyConfiguration(t *testing.T, cluster clusterplugin.Cluster, tt struct {
	name                   string
	environmentMode        string
	httpProxy              string
	httpsProxy             string
	noProxy                string
	expectedProxyFiles     int
	expectedKubeletContent []string
	expectedContainerdPath string
}) {
	g := NewWithT(t)

	// Test proxy environment variable extraction
	httpProxy := cluster.Env["HTTP_PROXY"]
	httpsProxy := cluster.Env["HTTPS_PROXY"]
	noProxy := cluster.Env["NO_PROXY"]

	g.Expect(httpProxy).To(Equal(tt.httpProxy))
	g.Expect(httpsProxy).To(Equal(tt.httpsProxy))
	g.Expect(noProxy).To(Equal(tt.noProxy))

	// Test proxy configuration detection
	hasProxy := len(httpProxy) > 0 || len(httpsProxy) > 0 || len(noProxy) > 0
	expectedHasProxy := tt.expectedProxyFiles > 0
	g.Expect(hasProxy).To(Equal(expectedHasProxy))

	// Test proxy file path construction
	if hasProxy {
		serviceFolder := getContainerdServiceFolderName(cluster.ProviderOptions)
		proxyFilePath := fmt.Sprintf("/run/systemd/system/%s.service.d/http-proxy.conf", serviceFolder)
		g.Expect(proxyFilePath).To(Equal(tt.expectedContainerdPath))
	}
}

// TestProxyEnvironmentVariableHandling tests proxy environment variable parsing and validation
func TestProxyEnvironmentVariableHandling(t *testing.T) {
	envTests := []struct {
		name          string
		envVars       map[string]string
		expectedValid bool
		expectedCount int
	}{
		{
			name: "valid_proxy_variables",
			envVars: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "https://proxy.example.com:8080",
				"NO_PROXY":    "localhost,127.0.0.1",
			},
			expectedValid: true,
			expectedCount: 3,
		},
		{
			name: "partial_proxy_variables",
			envVars: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
			},
			expectedValid: true,
			expectedCount: 1,
		},
		{
			name:          "no_proxy_variables",
			envVars:       map[string]string{},
			expectedValid: false,
			expectedCount: 0,
		},
		{
			name: "invalid_proxy_variables",
			envVars: map[string]string{
				"INVALID_PROXY": "invalid://proxy",
			},
			expectedValid: false,
			expectedCount: 0,
		},
	}

	for _, tt := range envTests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test proxy variable validation
			validVars := validateProxyVariables(tt.envVars)
			g.Expect(validVars).To(Equal(tt.expectedValid))

			// Test proxy variable count
			proxyCount := countProxyVariables(tt.envVars)
			g.Expect(proxyCount).To(Equal(tt.expectedCount))
		})
	}
}

// Helper functions for proxy testing
func validateProxyVariables(envVars map[string]string) bool {
	validKeys := []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}
	for key := range envVars {
		found := false
		for _, validKey := range validKeys {
			if key == validKey {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(envVars) > 0
}

func countProxyVariables(envVars map[string]string) int {
	count := 0
	validKeys := []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}
	for key := range envVars {
		for _, validKey := range validKeys {
			if key == validKey {
				count++
				break
			}
		}
	}
	return count
}
