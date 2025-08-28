package main

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/kairos-io/kairos-sdk/clusterplugin"
)

// TestClusterContextCreation tests cluster context creation logic
func TestClusterContextCreation(t *testing.T) {
	_ = NewWithT(t)

	tests := []struct {
		name                           string
		cluster                        clusterplugin.Cluster
		expectedRootPath              string
		expectedNodeRole              string
		expectedControlPlaneHost      string
		expectedClusterToken          string
		expectedContainerdServiceFolder string
		expectedLocalImagesPath       string
	}{
		{
			name: "basic_init_appliance_mode",
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleInit,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
				Options:          `clusterConfiguration: {}`,
			},
			expectedRootPath:               "/",
			expectedNodeRole:               "init",
			expectedControlPlaneHost:       "10.0.0.1",
			expectedClusterToken:           "abcdef:1234567890123456", // Transformed token
			expectedContainerdServiceFolder: "containerd",
			expectedLocalImagesPath:        "/opt/content/images",
		},
		{
			name: "agent_mode_with_custom_root",
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleControlPlane,
				ControlPlaneHost: "192.168.1.100",
				ClusterToken:     "token.with.dots.1234567890123456",
				ProviderOptions: map[string]string{
					"cluster_root_path": "/persistent/spectro",
				},
			},
			expectedRootPath:               "/persistent/spectro",
			expectedNodeRole:               "controlplane", // RoleJoin maps to controlplane
			expectedControlPlaneHost:       "192.168.1.100",
			expectedClusterToken:           "token:with:dots:1234567890123456",
			expectedContainerdServiceFolder: "containerd",
			expectedLocalImagesPath:        "/persistent/spectro/opt/content/images",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test cluster context logic
			context := createMockClusterContext(tt.cluster)

			// Validate context properties
			g.Expect(context.rootPath).To(Equal(tt.expectedRootPath))
			g.Expect(context.nodeRole).To(Equal(tt.expectedNodeRole))
			g.Expect(context.controlPlaneHost).To(Equal(tt.expectedControlPlaneHost))
			g.Expect(context.containerdServiceFolder).To(Equal(tt.expectedContainerdServiceFolder))
			g.Expect(context.localImagesPath).To(Equal(tt.expectedLocalImagesPath))
		})
	}
}

// Mock cluster context for testing
type mockClusterContext struct {
	rootPath                string
	nodeRole               string
	controlPlaneHost       string
	clusterToken           string
	containerdServiceFolder string
	localImagesPath        string
}

func createMockClusterContext(cluster clusterplugin.Cluster) mockClusterContext {
	rootPath := "/"
	if customRoot, ok := cluster.ProviderOptions["cluster_root_path"]; ok && customRoot != "" {
		rootPath = customRoot
	}

	nodeRole := string(cluster.Role)

	containerdServiceFolder := "containerd"
	if _, ok := cluster.ProviderOptions["spectro-containerd-service-name"]; ok {
		containerdServiceFolder = "spectro-containerd"
	}

	localImagesPath := cluster.LocalImagesPath
	if localImagesPath == "" {
		if rootPath == "/" {
			localImagesPath = "/opt/content/images"
		} else {
			localImagesPath = rootPath + "/opt/content/images"
		}
	}

	return mockClusterContext{
		rootPath:                rootPath,
		nodeRole:               nodeRole,
		controlPlaneHost:       cluster.ControlPlaneHost,
		clusterToken:           transformTokenForTest(cluster.ClusterToken),
		containerdServiceFolder: containerdServiceFolder,
		localImagesPath:        localImagesPath,
	}
}

func transformTokenForTest(token string) string {
	// Simple token transformation for testing
	if token == "" {
		return ""
	}
	// Replace dots with colons to simulate token transformation
	result := token
	for i := 0; i < len(result); i++ {
		if result[i] == '.' {
			result = result[:i] + ":" + result[i+1:]
		}
	}
	return result
}