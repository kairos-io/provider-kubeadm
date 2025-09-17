package stages

import (
	"testing"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	. "github.com/onsi/gomega"
)

// TestGetPreKubeadmCommandStages tests the GetPreKubeadmCommandStages function
func TestGetPreKubeadmCommandStages(t *testing.T) {
	tests := []struct {
		name            string
		rootPath        string
		expectedName    string
		expectedCommand string
	}{
		{
			name:            "standard_root_path",
			rootPath:        "/",
			expectedName:    "Run Pre Kubeadm Commands",
			expectedCommand: "/bin/bash /opt/kubeadm/scripts/kube-pre-init.sh /",
		},
		{
			name:            "custom_root_path",
			rootPath:        "/persistent/spectro",
			expectedName:    "Run Pre Kubeadm Commands",
			expectedCommand: "/bin/bash /persistent/spectro/opt/kubeadm/scripts/kube-pre-init.sh /persistent/spectro",
		},
		{
			name:            "agent_mode_path",
			rootPath:        "/mnt/custom",
			expectedName:    "Run Pre Kubeadm Commands",
			expectedCommand: "/bin/bash /mnt/custom/opt/kubeadm/scripts/kube-pre-init.sh /mnt/custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetPreKubeadmCommandStages(tt.rootPath)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(HaveLen(1))
			g.Expect(result.Commands[0]).To(Equal(tt.expectedCommand))
		})
	}
}

// TestGetPreKubeadmSwapOffDisableStage tests the GetPreKubeadmSwapOffDisableStage function
func TestGetPreKubeadmSwapOffDisableStage(t *testing.T) {
	t.Run("swap_off_disable_stage", func(t *testing.T) {
		g := NewWithT(t)

		result := GetPreKubeadmSwapOffDisableStage()

		// Validate stage structure
		g.Expect(result.Name).To(Equal("Run Pre Kubeadm Disable SwapOff"))
		g.Expect(result.Commands).To(HaveLen(2))
		g.Expect(result.Commands[0]).To(Equal("sed -i '/ swap / s/^\\(.*\\)$/#\\1/g' /etc/fstab"))
		g.Expect(result.Commands[1]).To(Equal("swapoff -a"))
	})
}

// TestGetPreKubeadmImportLocalImageStage tests the GetPreKubeadmImportLocalImageStage function
func TestGetPreKubeadmImportLocalImageStage(t *testing.T) {
	tests := []struct {
		name                 string
		clusterCtx           *domain.ClusterContext
		expectedName         string
		expectedCommandCount int
		expectedCondition    string
		validateCommands     func(*testing.T, []string)
	}{
		{
			name: "standard_local_images_path",
			clusterCtx: &domain.ClusterContext{
				RootPath:        "/",
				LocalImagesPath: "/opt/content/images",
			},
			expectedName:         "Run Import Local Images",
			expectedCommandCount: 2,
			expectedCondition:    "[ -d /opt/content/images ]",
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("chmod +x /opt/kubeadm/scripts/import.sh"))
				g.Expect(commands[1]).To(Equal("/bin/bash /opt/kubeadm/scripts/import.sh /opt/content/images /var/log/import.log"))
			},
		},
		{
			name: "agent_mode_local_images_path",
			clusterCtx: &domain.ClusterContext{
				RootPath:        "/persistent/spectro",
				LocalImagesPath: "/persistent/spectro/opt/content/images",
			},
			expectedName:         "Run Import Local Images",
			expectedCommandCount: 2,
			expectedCondition:    "[ -d /persistent/spectro/opt/content/images ]",
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("chmod +x /persistent/spectro/opt/kubeadm/scripts/import.sh"))
				g.Expect(commands[1]).To(Equal("/bin/bash /persistent/spectro/opt/kubeadm/scripts/import.sh /persistent/spectro/opt/content/images /var/log/import.log"))
			},
		},
		{
			name: "custom_local_images_path",
			clusterCtx: &domain.ClusterContext{
				RootPath:        "/mnt/custom",
				LocalImagesPath: "/custom/images/path",
			},
			expectedName:         "Run Import Local Images",
			expectedCommandCount: 2,
			expectedCondition:    "[ -d /custom/images/path ]",
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("chmod +x /mnt/custom/opt/kubeadm/scripts/import.sh"))
				g.Expect(commands[1]).To(Equal("/bin/bash /mnt/custom/opt/kubeadm/scripts/import.sh /custom/images/path /var/log/import.log"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetPreKubeadmImportLocalImageStage(tt.clusterCtx)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(HaveLen(tt.expectedCommandCount))
			g.Expect(result.If).To(Equal(tt.expectedCondition))
			tt.validateCommands(t, result.Commands)
		})
	}
}

// TestGetPreKubeadmImportCoreK8sImageStage tests the GetPreKubeadmImportCoreK8sImageStage function
func TestGetPreKubeadmImportCoreK8sImageStage(t *testing.T) {
	tests := []struct {
		name                 string
		rootPath             string
		expectedName         string
		expectedCommandCount int
		validateCommands     func(*testing.T, []string)
	}{
		{
			name:                 "standard_root_path",
			rootPath:             "/",
			expectedName:         "Run Load Kube Images",
			expectedCommandCount: 2,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("chmod +x /opt/kubeadm/scripts/import.sh"))
				g.Expect(commands[1]).To(Equal("/bin/bash /opt/kubeadm/scripts/import.sh /opt/kube-images /var/log/import-kube-images.log"))
			},
		},
		{
			name:                 "agent_mode_root_path",
			rootPath:             "/persistent/spectro",
			expectedName:         "Run Load Kube Images",
			expectedCommandCount: 2,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("chmod +x /persistent/spectro/opt/kubeadm/scripts/import.sh"))
				g.Expect(commands[1]).To(Equal("/bin/bash /persistent/spectro/opt/kubeadm/scripts/import.sh /persistent/spectro/opt/kube-images /var/log/import-kube-images.log"))
			},
		},
		{
			name:                 "custom_root_path",
			rootPath:             "/mnt/custom",
			expectedName:         "Run Load Kube Images",
			expectedCommandCount: 2,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("chmod +x /mnt/custom/opt/kubeadm/scripts/import.sh"))
				g.Expect(commands[1]).To(Equal("/bin/bash /mnt/custom/opt/kubeadm/scripts/import.sh /mnt/custom/opt/kube-images /var/log/import-kube-images.log"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetPreKubeadmImportCoreK8sImageStage(tt.rootPath)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(HaveLen(tt.expectedCommandCount))
			tt.validateCommands(t, result.Commands)
		})
	}
}
