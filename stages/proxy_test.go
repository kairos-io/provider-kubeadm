package stages

import (
	"testing"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	. "github.com/onsi/gomega"
)

// TestGetPreKubeadmProxyStage tests the GetPreKubeadmProxyStage function
func TestGetPreKubeadmProxyStage(t *testing.T) {
	tests := []struct {
		name                      string
		clusterCtx                *domain.ClusterContext
		expectedName              string
		expectedFileCount         int
		expectedKubeletPath       string
		expectedContainerdPath    string
		expectedPermissions       uint32
		validateKubeletContent    func(*testing.T, string)
		validateContainerdContent func(*testing.T, string)
	}{
		{
			name: "standard_containerd_with_proxy",
			clusterCtx: &domain.ClusterContext{
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
					"NO_PROXY":    "localhost,127.0.0.1",
				},
				ContainerdServiceFolderName: "containerd",
				ControlPlaneHost:            "10.0.0.1",
				ServiceCidr:                 "10.96.0.0/12",
				ClusterCidr:                 "192.168.0.0/16",
			},
			expectedName:           "Set proxy env",
			expectedFileCount:      2,
			expectedKubeletPath:    "/etc/default/kubelet",
			expectedContainerdPath: "/run/systemd/system/containerd.service.d/http-proxy.conf",
			expectedPermissions:    0400,
			validateKubeletContent: func(t *testing.T, content string) {
				g := NewWithT(t)
				g.Expect(content).To(ContainSubstring("HTTP_PROXY=http://proxy.example.com:8080"))
				g.Expect(content).To(ContainSubstring("HTTPS_PROXY=https://proxy.example.com:8080"))
				g.Expect(content).To(ContainSubstring("NO_PROXY="))
				g.Expect(content).To(ContainSubstring("localhost,127.0.0.1"))
			},
			validateContainerdContent: func(t *testing.T, content string) {
				g := NewWithT(t)
				g.Expect(content).To(ContainSubstring("[Service]"))
				g.Expect(content).To(ContainSubstring(`Environment="HTTP_PROXY=http://proxy.example.com:8080"`))
				g.Expect(content).To(ContainSubstring(`Environment="HTTPS_PROXY=https://proxy.example.com:8080"`))
				g.Expect(content).To(ContainSubstring(`Environment="NO_PROXY=`))
			},
		},
		{
			name: "spectro_containerd_with_proxy",
			clusterCtx: &domain.ClusterContext{
				EnvConfig: map[string]string{
					"HTTP_PROXY": "http://corporate-proxy:3128",
				},
				ContainerdServiceFolderName: "spectro-containerd",
				ControlPlaneHost:            "192.168.1.100",
				ServiceCidr:                 "172.20.0.0/16",
				ClusterCidr:                 "10.244.0.0/16",
			},
			expectedName:           "Set proxy env",
			expectedFileCount:      2,
			expectedKubeletPath:    "/etc/default/kubelet",
			expectedContainerdPath: "/run/systemd/system/spectro-containerd.service.d/http-proxy.conf",
			expectedPermissions:    0400,
			validateKubeletContent: func(t *testing.T, content string) {
				g := NewWithT(t)
				g.Expect(content).To(ContainSubstring("HTTP_PROXY=http://corporate-proxy:3128"))
				g.Expect(content).To(ContainSubstring("NO_PROXY="))
			},
			validateContainerdContent: func(t *testing.T, content string) {
				g := NewWithT(t)
				g.Expect(content).To(ContainSubstring("[Service]"))
				g.Expect(content).To(ContainSubstring(`Environment="HTTP_PROXY=http://corporate-proxy:3128"`))
			},
		},
		{
			name: "no_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				EnvConfig:                   map[string]string{},
				ContainerdServiceFolderName: "containerd",
				ControlPlaneHost:            "10.0.0.1",
			},
			expectedName:           "Set proxy env",
			expectedFileCount:      2,
			expectedKubeletPath:    "/etc/default/kubelet",
			expectedContainerdPath: "/run/systemd/system/containerd.service.d/http-proxy.conf",
			expectedPermissions:    0400,
			validateKubeletContent: func(t *testing.T, content string) {
				g := NewWithT(t)
				g.Expect(content).To(BeEmpty())
			},
			validateContainerdContent: func(t *testing.T, content string) {
				g := NewWithT(t)
				g.Expect(content).To(BeEmpty())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetPreKubeadmProxyStage(tt.clusterCtx)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Files).To(HaveLen(tt.expectedFileCount))

			// Validate kubelet file
			kubeletFile := result.Files[0]
			g.Expect(kubeletFile.Path).To(Equal(tt.expectedKubeletPath))
			g.Expect(kubeletFile.Permissions).To(Equal(tt.expectedPermissions))
			tt.validateKubeletContent(t, kubeletFile.Content)

			// Validate containerd file
			containerdFile := result.Files[1]
			g.Expect(containerdFile.Path).To(Equal(tt.expectedContainerdPath))
			g.Expect(containerdFile.Permissions).To(Equal(tt.expectedPermissions))
			tt.validateContainerdContent(t, containerdFile.Content)
		})
	}
}

// TestKubeletProxyEnv tests the kubeletProxyEnv function
func TestKubeletProxyEnv(t *testing.T) {
	tests := []struct {
		name           string
		clusterCtx     *domain.ClusterContext
		expectedResult string
	}{
		{
			name: "with_all_proxy_variables",
			clusterCtx: &domain.ClusterContext{
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
					"NO_PROXY":    "localhost,127.0.0.1",
				},
				ControlPlaneHost: "10.0.0.1",
				ServiceCidr:      "10.96.0.0/12",
				ClusterCidr:      "192.168.0.0/16",
			},
			expectedResult: "HTTP_PROXY=http://proxy.example.com:8080\nHTTPS_PROXY=https://proxy.example.com:8080\nNO_PROXY=localhost,127.0.0.1,.svc,.svc.cluster.local,10.0.0.1,10.96.0.0/12,192.168.0.0/16",
		},
		{
			name: "with_only_http_proxy",
			clusterCtx: &domain.ClusterContext{
				EnvConfig: map[string]string{
					"HTTP_PROXY": "http://proxy.example.com:8080",
				},
				ControlPlaneHost: "10.0.0.1",
				ServiceCidr:      "10.96.0.0/12",
				ClusterCidr:      "192.168.0.0/16",
			},
			expectedResult: "HTTP_PROXY=http://proxy.example.com:8080\nNO_PROXY=.svc,.svc.cluster.local,10.0.0.1,10.96.0.0/12,192.168.0.0/16",
		},
		{
			name: "no_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				EnvConfig:        map[string]string{},
				ControlPlaneHost: "10.0.0.1",
				ServiceCidr:      "10.96.0.0/12",
				ClusterCidr:      "192.168.0.0/16",
			},
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test the private function indirectly through the stage
			stage := GetPreKubeadmProxyStage(tt.clusterCtx)
			kubeletContent := stage.Files[0].Content

			if tt.expectedResult == "" {
				g.Expect(kubeletContent).To(BeEmpty())
			} else {
				g.Expect(kubeletContent).To(ContainSubstring("HTTP_PROXY="))
				g.Expect(kubeletContent).To(ContainSubstring("NO_PROXY="))
			}
		})
	}
}

// TestContainerdProxyEnv tests the containerdProxyEnv function
func TestContainerdProxyEnv(t *testing.T) {
	tests := []struct {
		name           string
		clusterCtx     *domain.ClusterContext
		expectedResult string
	}{
		{
			name: "with_all_proxy_variables",
			clusterCtx: &domain.ClusterContext{
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
					"NO_PROXY":    "localhost,127.0.0.1",
				},
				ControlPlaneHost: "10.0.0.1",
				ServiceCidr:      "10.96.0.0/12",
				ClusterCidr:      "192.168.0.0/16",
			},
			expectedResult: "[Service]\nEnvironment=\"HTTP_PROXY=http://proxy.example.com:8080\"\nEnvironment=\"HTTPS_PROXY=https://proxy.example.com:8080\"\nEnvironment=\"NO_PROXY=localhost,127.0.0.1,.svc,.svc.cluster.local,10.0.0.1,10.96.0.0/12,192.168.0.0/16\"",
		},
		{
			name: "with_only_http_proxy",
			clusterCtx: &domain.ClusterContext{
				EnvConfig: map[string]string{
					"HTTP_PROXY": "http://proxy.example.com:8080",
				},
				ControlPlaneHost: "10.0.0.1",
				ServiceCidr:      "10.96.0.0/12",
				ClusterCidr:      "192.168.0.0/16",
			},
			expectedResult: "[Service]\nEnvironment=\"HTTP_PROXY=http://proxy.example.com:8080\"\nEnvironment=\"NO_PROXY=.svc,.svc.cluster.local,10.0.0.1,10.96.0.0/12,192.168.0.0/16\"",
		},
		{
			name: "no_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				EnvConfig:        map[string]string{},
				ControlPlaneHost: "10.0.0.1",
				ServiceCidr:      "10.96.0.0/12",
				ClusterCidr:      "192.168.0.0/16",
			},
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test the private function indirectly through the stage
			stage := GetPreKubeadmProxyStage(tt.clusterCtx)
			containerdContent := stage.Files[1].Content

			if tt.expectedResult == "" {
				g.Expect(containerdContent).To(BeEmpty())
			} else {
				g.Expect(containerdContent).To(ContainSubstring("[Service]"))
				g.Expect(containerdContent).To(ContainSubstring(`Environment="HTTP_PROXY=`))
				g.Expect(containerdContent).To(ContainSubstring(`Environment="NO_PROXY=`))
			}
		})
	}
}
