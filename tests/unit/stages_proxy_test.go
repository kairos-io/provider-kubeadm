package main

import (
	"testing"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	"github.com/kairos-io/kairos/provider-kubeadm/stages"
	yip "github.com/mudler/yip/pkg/schema"
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
		{
			name: "mixed_case_proxy_variables",
			clusterCtx: &domain.ClusterContext{
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://UPPER.proxy.com:8080",
					"HTTPS_PROXY": "https://lower.proxy.com:8080",
					"NO_PROXY":    "mixed.case.local",
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
				g.Expect(content).To(ContainSubstring("HTTP_PROXY=http://UPPER.proxy.com:8080"))
				g.Expect(content).To(ContainSubstring("HTTPS_PROXY=https://lower.proxy.com:8080"))
				g.Expect(content).To(ContainSubstring("mixed.case.local"))
			},
			validateContainerdContent: func(t *testing.T, content string) {
				g := NewWithT(t)
				g.Expect(content).To(ContainSubstring(`Environment="HTTP_PROXY=http://UPPER.proxy.com:8080"`))
				g.Expect(content).To(ContainSubstring(`Environment="HTTPS_PROXY=https://lower.proxy.com:8080"`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Execute function under test
			result := stages.GetPreKubeadmProxyStage(tt.clusterCtx)

			// Validate basic properties
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Files).To(HaveLen(tt.expectedFileCount))

			// Find and validate kubelet file
			var kubeletFile *yip.File
			var containerdFile *yip.File
			for i := range result.Files {
				if result.Files[i].Path == tt.expectedKubeletPath {
					kubeletFile = &result.Files[i]
				} else if result.Files[i].Path == tt.expectedContainerdPath {
					containerdFile = &result.Files[i]
				}
			}

			// Validate kubelet file
			g.Expect(kubeletFile).ToNot(BeNil())
			g.Expect(kubeletFile.Path).To(Equal(tt.expectedKubeletPath))
			g.Expect(kubeletFile.Permissions).To(Equal(tt.expectedPermissions))
			if tt.validateKubeletContent != nil {
				tt.validateKubeletContent(t, kubeletFile.Content)
			}

			// Validate containerd file
			g.Expect(containerdFile).ToNot(BeNil())
			g.Expect(containerdFile.Path).To(Equal(tt.expectedContainerdPath))
			g.Expect(containerdFile.Permissions).To(Equal(tt.expectedPermissions))
			if tt.validateContainerdContent != nil {
				tt.validateContainerdContent(t, containerdFile.Content)
			}
		})
	}
}
