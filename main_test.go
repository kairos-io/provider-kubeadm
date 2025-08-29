package main

import (
	"testing"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	yip "github.com/mudler/yip/pkg/schema"
	. "github.com/onsi/gomega"
)

// TestCreateClusterContext tests the CreateClusterContext function
func TestCreateClusterContext(t *testing.T) {
	g := NewWithT(t)

	cluster := clusterplugin.Cluster{
		Role:             clusterplugin.RoleInit,
		ControlPlaneHost: "10.0.0.1",
		ClusterToken:     "abcdef.1234567890123456",
		Options:          `clusterConfiguration: {}`,
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	result := CreateClusterContext(cluster)

	// Basic validations
	g.Expect(result.RootPath).To(Equal("/"))
	g.Expect(result.NodeRole).To(Equal("init"))
	g.Expect(result.ControlPlaneHost).To(Equal("10.0.0.1"))
	// The token is transformed using SHA256 hash, so we just check it's not empty and has the expected format
	g.Expect(result.ClusterToken).ToNot(BeEmpty())
	g.Expect(result.ClusterToken).To(MatchRegexp(`^[a-f0-9]{6}\.[a-f0-9]{16}$`))
	g.Expect(result.ContainerdServiceFolderName).To(Equal("containerd"))
	g.Expect(result.LocalImagesPath).To(Equal("/opt/content/images"))
	g.Expect(result.EnvConfig).To(HaveKeyWithValue("TEST_VAR", "test_value"))
}

// TestCreateClusterContextComprehensive tests multiple scenarios
func TestCreateClusterContextComprehensive(t *testing.T) {

	tests := []struct {
		name                            string
		cluster                         clusterplugin.Cluster
		expectedRootPath                string
		expectedNodeRole                string
		expectedControlPlaneHost        string
		expectedContainerdServiceFolder string
		expectedLocalImagesPath         string
		validateEnvConfig               func(*testing.T, map[string]string)
	}{
		{
			name: "controlplane_mode_with_custom_root",
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleControlPlane,
				ControlPlaneHost: "192.168.1.100",
				ClusterToken:     "token.with.dots.1234567890123456",
				ProviderOptions: map[string]string{
					"cluster_root_path": "/persistent/spectro",
				},
			},
			expectedRootPath:                "/persistent/spectro",
			expectedNodeRole:                "controlplane",
			expectedControlPlaneHost:        "192.168.1.100",
			expectedContainerdServiceFolder: "containerd",
			expectedLocalImagesPath:         "/persistent/spectro/opt/content/images",
		},
		{
			name: "spectro_containerd_with_custom_images",
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleWorker,
				ControlPlaneHost: "master.k8s.local",
				ClusterToken:     "special-token.1234567890123456",
				LocalImagesPath:  "/custom/images/path",
				ProviderOptions: map[string]string{
					"spectro-containerd-service-name": "true",
					"cluster_root_path":               "/mnt/custom",
				},
				Env: map[string]string{
					"HTTP_PROXY":  "http://proxy.corp.com:8080",
					"HTTPS_PROXY": "https://proxy.corp.com:8080",
					"NO_PROXY":    "localhost,127.0.0.1",
				},
			},
			expectedRootPath:                "/mnt/custom",
			expectedNodeRole:                "worker",
			expectedControlPlaneHost:        "master.k8s.local",
			expectedContainerdServiceFolder: "spectro-containerd",
			expectedLocalImagesPath:         "/custom/images/path",
			validateEnvConfig: func(t *testing.T, env map[string]string) {
				g := NewWithT(t)
				g.Expect(env).To(HaveKeyWithValue("HTTP_PROXY", "http://proxy.corp.com:8080"))
				g.Expect(env).To(HaveKeyWithValue("HTTPS_PROXY", "https://proxy.corp.com:8080"))
				g.Expect(env).To(HaveKeyWithValue("NO_PROXY", "localhost,127.0.0.1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Execute function under test
			result := CreateClusterContext(tt.cluster)

			// Basic validations
			g.Expect(result.RootPath).To(Equal(tt.expectedRootPath))
			g.Expect(result.NodeRole).To(Equal(tt.expectedNodeRole))
			g.Expect(result.ControlPlaneHost).To(Equal(tt.expectedControlPlaneHost))
			g.Expect(result.ClusterToken).ToNot(BeEmpty())
			g.Expect(result.ClusterToken).To(MatchRegexp(`^[a-f0-9]{6}\.[a-f0-9]{16}$`))
			g.Expect(result.ContainerdServiceFolderName).To(Equal(tt.expectedContainerdServiceFolder))
			g.Expect(result.LocalImagesPath).To(Equal(tt.expectedLocalImagesPath))

			// Environment validation
			if tt.validateEnvConfig != nil {
				tt.validateEnvConfig(t, result.EnvConfig)
			}
		})
	}
}

// TestGetContainerdServiceFolderName tests the getContainerdServiceFolderName function
func TestGetContainerdServiceFolderName(t *testing.T) {
	g := NewWithT(t)

	// Test default behavior
	options := map[string]string{}
	result := getContainerdServiceFolderName(options)
	g.Expect(result).To(Equal("containerd"))

	// Test spectro-containerd
	options = map[string]string{
		"spectro-containerd-service-name": "true",
	}
	result = getContainerdServiceFolderName(options)
	g.Expect(result).To(Equal("spectro-containerd"))

	// Test other options don't affect
	options = map[string]string{
		"other_option":      "value",
		"cluster_root_path": "/custom/path",
	}
	result = getContainerdServiceFolderName(options)
	g.Expect(result).To(Equal("containerd"))
}

// TestSetClusterSubnetCtx tests the setClusterSubnetCtx function
func TestSetClusterSubnetCtx(t *testing.T) {
	g := NewWithT(t)

	clusterCtx := &domain.ClusterContext{}
	serviceSubnet := "10.96.0.0/12"
	podSubnet := "10.244.0.0/16"

	setClusterSubnetCtx(clusterCtx, serviceSubnet, podSubnet)

	g.Expect(clusterCtx.ServiceCidr).To(Equal(serviceSubnet))
	g.Expect(clusterCtx.ClusterCidr).To(Equal(podSubnet))
}

// TestClusterProvider tests the main clusterProvider function
func TestClusterProvider(t *testing.T) {
	t.Skip("Skipping clusterProvider test due to external kubeadm dependency")

	tests := []struct {
		name              string
		cluster           clusterplugin.Cluster
		expectedName      string
		expectedStageKey  string
		minExpectedStages int
		validateFunc      func(*testing.T, yip.YipConfig)
	}{
		{
			name: "basic_init_cluster",
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleInit,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
				Options: `
clusterConfiguration:
  kubernetesVersion: v1.30.11
initConfiguration:
  nodeRegistration:
    kubeletExtraArgs:
      node-ip: "10.0.0.1"`,
			},
			expectedName:      "Kubeadm Kairos Cluster Provider",
			expectedStageKey:  "boot.before",
			minExpectedStages: 1, // At least one stage should be generated
			validateFunc: func(t *testing.T, config yip.YipConfig) {
				g := NewWithT(t)
				g.Expect(config.Name).To(Equal("Kubeadm Kairos Cluster Provider"))
				g.Expect(config.Stages).To(HaveKey("boot.before"))
			},
		},
		{
			name: "controlplane_with_proxy",
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleControlPlane,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
				Options: `
clusterConfiguration:
  kubernetesVersion: v1.30.11
joinConfiguration:
  nodeRegistration:
    kubeletExtraArgs:
      node-ip: "10.0.0.2"`,
				Env: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
				},
			},
			expectedName:      "Kubeadm Kairos Cluster Provider",
			expectedStageKey:  "boot.before",
			minExpectedStages: 1, // At least one stage should be generated
			validateFunc: func(t *testing.T, config yip.YipConfig) {
				g := NewWithT(t)
				g.Expect(config.Name).To(Equal("Kubeadm Kairos Cluster Provider"))
				g.Expect(config.Stages).To(HaveKey("boot.before"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Execute function under test
			result := clusterProvider(tt.cluster)

			// Basic validations
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Stages).To(HaveKey(tt.expectedStageKey))
			g.Expect(len(result.Stages[tt.expectedStageKey])).To(BeNumerically(">=", tt.minExpectedStages))

			// Custom validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}
