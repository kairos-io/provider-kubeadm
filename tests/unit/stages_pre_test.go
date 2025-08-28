package main

import (
	"testing"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	"github.com/kairos-io/kairos/provider-kubeadm/stages"
	. "github.com/onsi/gomega"
)

// TestGetPreKubeadmCommandStages tests the GetPreKubeadmCommandStages function
func TestGetPreKubeadmCommandStages(t *testing.T) {

	tests := []struct {
		name             string
		rootPath         string
		expectedName     string
		expectedCommands []string
		expectedIf       string
	}{
		{
			name:         "appliance_mode_root_path",
			rootPath:     "/",
			expectedName: "Run Pre Kubeadm Commands",
			expectedCommands: []string{
				"/bin/bash /opt/kubeadm/scripts/kube-pre-init.sh /",
			},
			expectedIf: "",
		},
		{
			name:         "agent_mode_custom_root",
			rootPath:     "/persistent/spectro",
			expectedName: "Run Pre Kubeadm Commands",
			expectedCommands: []string{
				"/bin/bash /persistent/spectro/opt/kubeadm/scripts/kube-pre-init.sh /persistent/spectro",
			},
			expectedIf: "",
		},
		{
			name:         "custom_deep_root_path",
			rootPath:     "/mnt/custom/deep/path",
			expectedName: "Run Pre Kubeadm Commands",
			expectedCommands: []string{
				"/bin/bash /mnt/custom/deep/path/opt/kubeadm/scripts/kube-pre-init.sh /mnt/custom/deep/path",
			},
			expectedIf: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Execute function under test
			result := stages.GetPreKubeadmCommandStages(tt.rootPath)

			// Validate results
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(Equal(tt.expectedCommands))
			g.Expect(result.If).To(Equal(tt.expectedIf))
		})
	}
}

// TestGetPreKubeadmSwapOffDisableStage tests the GetPreKubeadmSwapOffDisableStage function
func TestGetPreKubeadmSwapOffDisableStage(t *testing.T) {
	g := NewWithT(t)

	// Execute function under test
	result := stages.GetPreKubeadmSwapOffDisableStage()

	// Validate results
	g.Expect(result.Name).To(Equal("Run Pre Kubeadm Disable SwapOff"))
	g.Expect(result.Commands).To(HaveLen(2))
	g.Expect(result.Commands[0]).To(Equal("sed -i '/ swap / s/^\\(.*\\)$/#\\1/g' /etc/fstab"))
	g.Expect(result.Commands[1]).To(Equal("swapoff -a"))
	g.Expect(result.If).To(BeEmpty())
}

// TestGetPreKubeadmImportLocalImageStage tests the GetPreKubeadmImportLocalImageStage function
func TestGetPreKubeadmImportLocalImageStage(t *testing.T) {

	tests := []struct {
		name             string
		clusterCtx       *domain.ClusterContext
		expectedName     string
		expectedCommands []string
		expectedIf       string
	}{
		{
			name: "appliance_mode_default_images_path",
			clusterCtx: &domain.ClusterContext{
				RootPath:        "/",
				LocalImagesPath: "/opt/content/images",
			},
			expectedName: "Run Import Local Images",
			expectedCommands: []string{
				"chmod +x /opt/kubeadm/scripts/import.sh",
				"/bin/sh /opt/kubeadm/scripts/import.sh /opt/content/images > /var/log/import.log",
			},
			expectedIf: "[ -d /opt/content/images ]",
		},
		{
			name: "agent_mode_custom_root_and_images",
			clusterCtx: &domain.ClusterContext{
				RootPath:        "/persistent/spectro",
				LocalImagesPath: "/persistent/spectro/opt/content/images",
			},
			expectedName: "Run Import Local Images",
			expectedCommands: []string{
				"chmod +x /persistent/spectro/opt/kubeadm/scripts/import.sh",
				"/bin/sh /persistent/spectro/opt/kubeadm/scripts/import.sh /persistent/spectro/opt/content/images > /var/log/import.log",
			},
			expectedIf: "[ -d /persistent/spectro/opt/content/images ]",
		},
		{
			name: "custom_images_path",
			clusterCtx: &domain.ClusterContext{
				RootPath:        "/",
				LocalImagesPath: "/custom/images/location",
			},
			expectedName: "Run Import Local Images",
			expectedCommands: []string{
				"chmod +x /opt/kubeadm/scripts/import.sh",
				"/bin/sh /opt/kubeadm/scripts/import.sh /custom/images/location > /var/log/import.log",
			},
			expectedIf: "[ -d /custom/images/location ]",
		},
		{
			name: "agent_mode_with_custom_images_path",
			clusterCtx: &domain.ClusterContext{
				RootPath:        "/mnt/stylus",
				LocalImagesPath: "/opt/custom/container/images",
			},
			expectedName: "Run Import Local Images",
			expectedCommands: []string{
				"chmod +x /mnt/stylus/opt/kubeadm/scripts/import.sh",
				"/bin/sh /mnt/stylus/opt/kubeadm/scripts/import.sh /opt/custom/container/images > /var/log/import.log",
			},
			expectedIf: "[ -d /opt/custom/container/images ]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Execute function under test
			result := stages.GetPreKubeadmImportLocalImageStage(tt.clusterCtx)

			// Validate results
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(Equal(tt.expectedCommands))
			g.Expect(result.If).To(Equal(tt.expectedIf))
		})
	}
}

// TestGetPreKubeadmImportCoreK8sImageStage tests the GetPreKubeadmImportCoreK8sImageStage function
func TestGetPreKubeadmImportCoreK8sImageStage(t *testing.T) {

	tests := []struct {
		name             string
		rootPath         string
		expectedName     string
		expectedCommands []string
		expectedIf       string
	}{
		{
			name:         "appliance_mode_default_path",
			rootPath:     "/",
			expectedName: "Run Load Kube Images",
			expectedCommands: []string{
				"chmod +x /opt/kubeadm/scripts/import.sh",
				"/bin/sh /opt/kubeadm/scripts/import.sh /opt/kube-images > /var/log/import-kube-images.log",
			},
			expectedIf: "",
		},
		{
			name:         "agent_mode_custom_root",
			rootPath:     "/persistent/spectro",
			expectedName: "Run Load Kube Images",
			expectedCommands: []string{
				"chmod +x /persistent/spectro/opt/kubeadm/scripts/import.sh",
				"/bin/sh /persistent/spectro/opt/kubeadm/scripts/import.sh /persistent/spectro/opt/kube-images > /var/log/import-kube-images.log",
			},
			expectedIf: "",
		},
		{
			name:         "deep_custom_path",
			rootPath:     "/mnt/custom/k8s/root",
			expectedName: "Run Load Kube Images",
			expectedCommands: []string{
				"chmod +x /mnt/custom/k8s/root/opt/kubeadm/scripts/import.sh",
				"/bin/sh /mnt/custom/k8s/root/opt/kubeadm/scripts/import.sh /mnt/custom/k8s/root/opt/kube-images > /var/log/import-kube-images.log",
			},
			expectedIf: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Execute function under test
			result := stages.GetPreKubeadmImportCoreK8sImageStage(tt.rootPath)

			// Validate results
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(Equal(tt.expectedCommands))
			g.Expect(result.If).To(Equal(tt.expectedIf))
		})
	}
}
