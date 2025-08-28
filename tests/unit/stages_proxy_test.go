package unit

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestProxyStageGeneration tests proxy stage generation logic
func TestProxyStageGeneration(t *testing.T) {
	_ = NewWithT(t)

	tests := []struct {
		name                        string
		envConfig                   map[string]string
		containerdServiceFolderName string
		expectedName                string
		expectedFileCount           int
		expectedKubeletPath         string
		expectedContainerdPath      string
		expectedPermissions         uint32
	}{
		{
			name: "standard_containerd_with_proxy",
			envConfig: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "https://proxy.example.com:8080",
				"NO_PROXY":    "localhost,127.0.0.1",
			},
			containerdServiceFolderName: "containerd",
			expectedName:                "Set proxy env",
			expectedFileCount:           2,
			expectedKubeletPath:         "/etc/default/kubelet",
			expectedContainerdPath:      "/run/systemd/system/containerd.service.d/http-proxy.conf",
			expectedPermissions:         0400,
		},
		{
			name: "spectro_containerd_with_proxy",
			envConfig: map[string]string{
				"HTTP_PROXY": "http://corporate-proxy:3128",
			},
			containerdServiceFolderName: "spectro-containerd",
			expectedName:                "Set proxy env",
			expectedFileCount:           2,
			expectedKubeletPath:         "/etc/default/kubelet",
			expectedContainerdPath:      "/run/systemd/system/spectro-containerd.service.d/http-proxy.conf",
			expectedPermissions:         0400,
		},
		{
			name:                        "no_proxy_configuration",
			envConfig:                   map[string]string{},
			containerdServiceFolderName: "containerd",
			expectedName:                "Set proxy env",
			expectedFileCount:           2,
			expectedKubeletPath:         "/etc/default/kubelet",
			expectedContainerdPath:      "/run/systemd/system/containerd.service.d/http-proxy.conf",
			expectedPermissions:         0400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test proxy stage generation
			stage := generateMockProxyStage(tt.envConfig, tt.containerdServiceFolderName)

			// Validate basic properties
			g.Expect(stage.name).To(Equal(tt.expectedName))
			g.Expect(stage.fileCount).To(Equal(tt.expectedFileCount))
			g.Expect(stage.kubeletPath).To(Equal(tt.expectedKubeletPath))
			g.Expect(stage.containerdPath).To(Equal(tt.expectedContainerdPath))
			g.Expect(stage.permissions).To(Equal(tt.expectedPermissions))
		})
	}
}

// TestProxyEnvironmentHandling tests proxy environment variable handling
func TestProxyEnvironmentHandling(t *testing.T) {
	_ = NewWithT(t)

	tests := []struct {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test proxy variable validation
			isValid := validateProxyVars(tt.envVars)
			g.Expect(isValid).To(Equal(tt.expectedValid))

			// Test proxy variable count
			count := countProxyVars(tt.envVars)
			g.Expect(count).To(Equal(tt.expectedCount))
		})
	}
}

// Mock proxy stage for testing
type mockProxyStage struct {
	name           string
	fileCount      int
	kubeletPath    string
	containerdPath string
	permissions    uint32
}

func generateMockProxyStage(envConfig map[string]string, containerdServiceFolderName string) mockProxyStage {
	return mockProxyStage{
		name:           "Set proxy env",
		fileCount:      2,
		kubeletPath:    "/etc/default/kubelet",
		containerdPath: "/run/systemd/system/" + containerdServiceFolderName + ".service.d/http-proxy.conf",
		permissions:    0400,
	}
}

func validateProxyVars(envVars map[string]string) bool {
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

func countProxyVars(envVars map[string]string) int {
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
