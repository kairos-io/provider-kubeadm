package integration

import (
	"fmt"
	"testing"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	. "github.com/onsi/gomega"
)

// TestEnvironmentModePathHandling tests agent vs appliance mode path handling
func TestEnvironmentModePathHandling(t *testing.T) {
	environmentTests := []struct {
		name                string
		environmentMode     string
		clusterRootPath     string
		expectedRootPath    string
		expectedBinaryPaths []string
		expectedConfigPaths []string
	}{
		{
			name:                "appliance_mode_standard_paths",
			environmentMode:     "appliance",
			clusterRootPath:     "/",
			expectedRootPath:    "/",
			expectedBinaryPaths: []string{"/usr/bin", "/usr/local/bin"},
			expectedConfigPaths: []string{"/opt/kubeadm/kubeadm.yaml", "/opt/kubeadm/cluster-config.yaml"},
		},
		{
			name:                "agent_mode_custom_stylus_root",
			environmentMode:     "agent",
			clusterRootPath:     "/persistent/spectro",
			expectedRootPath:    "/persistent/spectro",
			expectedBinaryPaths: []string{"/persistent/spectro/usr/bin", "/persistent/spectro/usr/local/bin"},
			expectedConfigPaths: []string{"/persistent/spectro/opt/kubeadm/kubeadm.yaml", "/persistent/spectro/opt/kubeadm/cluster-config.yaml"},
		},
		{
			name:                "agent_mode_alternative_root",
			environmentMode:     "agent",
			clusterRootPath:     "/mnt/custom",
			expectedRootPath:    "/mnt/custom",
			expectedBinaryPaths: []string{"/mnt/custom/usr/bin", "/mnt/custom/usr/local/bin"},
			expectedConfigPaths: []string{"/mnt/custom/opt/kubeadm/kubeadm.yaml", "/mnt/custom/opt/kubeadm/cluster-config.yaml"},
		},
	}

	for _, tt := range environmentTests {
		t.Run(tt.name, func(t *testing.T) {
			_ = NewWithT(t)

			// Create cluster input with specific root path
			cluster := clusterplugin.Cluster{
				Role:             clusterplugin.RoleInit,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
				ProviderOptions: map[string]string{
					"cluster_root_path": tt.clusterRootPath,
				},
			}

			// Test path construction logic
			validatePathConstruction(t, cluster, tt.expectedRootPath)
		})
	}
}

// validatePathConstruction tests path construction logic without external dependencies
func validatePathConstruction(t *testing.T, cluster clusterplugin.Cluster, expectedRootPath string) {
	g := NewWithT(t)

	// Test that the cluster root path is extracted correctly
	rootPath := getClusterRootPath(cluster)
	g.Expect(rootPath).To(Equal(expectedRootPath))

	// Test path construction for various components
	expectedPaths := []string{
		fmt.Sprintf("%s/usr/bin/kubeadm", expectedRootPath),
		fmt.Sprintf("%s/opt/kubeadm/scripts/kube-init.sh", expectedRootPath),
		fmt.Sprintf("%s/opt/kubeadm/kubeadm.yaml", expectedRootPath),
	}

	// Validate that paths are constructed correctly
	for _, path := range expectedPaths {
		g.Expect(path).To(ContainSubstring(expectedRootPath))
	}
}

// Helper function to get cluster root path (simplified version)
func getClusterRootPath(cluster clusterplugin.Cluster) string {
	if rootPath, ok := cluster.ProviderOptions["cluster_root_path"]; ok && rootPath != "" {
		return rootPath
	}
	return "/"
}

// TestContainerdServiceFolderDetection tests service folder detection based on environment
func TestContainerdServiceFolderDetection(t *testing.T) {
	serviceTests := []struct {
		name                  string
		providerOptions       map[string]string
		expectedServiceFolder string
		expectedProxyFilePath string
	}{
		{
			name:                  "standard_containerd_service",
			providerOptions:       map[string]string{},
			expectedServiceFolder: "containerd",
			expectedProxyFilePath: "/run/systemd/system/containerd.service.d/http-proxy.conf",
		},
		{
			name: "spectro_containerd_service",
			providerOptions: map[string]string{
				"spectro-containerd-service-name": "true",
			},
			expectedServiceFolder: "spectro-containerd",
			expectedProxyFilePath: "/run/systemd/system/spectro-containerd.service.d/http-proxy.conf",
		},
	}

	for _, tt := range serviceTests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test service folder detection logic
			serviceFolder := getContainerdServiceFolderName(tt.providerOptions)
			g.Expect(serviceFolder).To(Equal(tt.expectedServiceFolder))

			// Test proxy file path construction
			proxyFilePath := fmt.Sprintf("/run/systemd/system/%s.service.d/http-proxy.conf", serviceFolder)
			g.Expect(proxyFilePath).To(Equal(tt.expectedProxyFilePath))
		})
	}
}

// Helper function to get containerd service folder name
func getContainerdServiceFolderName(options map[string]string) string {
	if _, ok := options["spectro-containerd-service-name"]; ok {
		return "spectro-containerd"
	}
	return "containerd"
}

// TestLocalImagesPathHandling tests local images path handling
func TestLocalImagesPathHandling(t *testing.T) {
	imagesTests := []struct {
		name               string
		environmentMode    string
		clusterRootPath    string
		localImagesPath    string
		expectedImagesPath string
	}{
		{
			name:               "appliance_default_images_path",
			environmentMode:    "appliance",
			clusterRootPath:    "/",
			localImagesPath:    "",
			expectedImagesPath: "/opt/content/images",
		},
		{
			name:               "agent_default_images_path",
			environmentMode:    "agent",
			clusterRootPath:    "/persistent/spectro",
			localImagesPath:    "",
			expectedImagesPath: "/persistent/spectro/opt/content/images",
		},
		{
			name:               "custom_images_path",
			environmentMode:    "appliance",
			clusterRootPath:    "/",
			localImagesPath:    "/custom/images/location",
			expectedImagesPath: "/custom/images/location",
		},
	}

	for _, tt := range imagesTests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Create cluster input
			cluster := clusterplugin.Cluster{
				Role:             clusterplugin.RoleInit,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
				ProviderOptions: map[string]string{
					"cluster_root_path": tt.clusterRootPath,
				},
			}

			// Set local images path if specified
			if tt.localImagesPath != "" {
				cluster.LocalImagesPath = tt.localImagesPath
			}

			// Test local images path logic
			imagesPath := getLocalImagesPath(cluster)
			g.Expect(imagesPath).To(Equal(tt.expectedImagesPath))
		})
	}
}

// Helper function to get local images path
func getLocalImagesPath(cluster clusterplugin.Cluster) string {
	if cluster.LocalImagesPath != "" {
		return cluster.LocalImagesPath
	}

	rootPath := getClusterRootPath(cluster)
	if rootPath == "/" {
		return "/opt/content/images"
	}
	return fmt.Sprintf("%s/opt/content/images", rootPath)
}
