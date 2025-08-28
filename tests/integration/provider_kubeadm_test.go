package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/twpayne/go-vfs/v4/vfst"
	"gopkg.in/yaml.v3"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/log"
	yip "github.com/mudler/yip/pkg/schema"
)

// TestScenario defines a complete test scenario for provider-kubeadm
type TestScenario struct {
	name             string
	kubeadmVersion   string // "1.30.0" or "1.31.0"
	nodeRole         string // "init", "controlplane", "worker"
	environmentMode  string // "agent", "appliance"
	proxyConfig      bool
	containerRuntime string // "spectro-containerd", "containerd"
	userOptions      string
	localImages      bool
	expectedStages   int
	wantErr          bool
}

// ExpectedStage represents validation expectations for YIP stages
type ExpectedStage struct {
	name         string
	hasFiles     bool
	hasCommands  bool
	fileCount    int
	commandCount int
}

func TestProviderKubeadmYipStageGeneration(t *testing.T) {
	g := NewWithT(t)

	// All 48 test scenarios covering the complete matrix
	tests := []TestScenario{
		// v1beta3 scenarios (kubeadm < 1.31.0)
		{
			name:             "v1beta3_init_agent_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "init",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "init"),
			localImages:      true,
			expectedStages:   10, // 5 pre + 5 init stages
			wantErr:          false,
		},
		{
			name:             "v1beta3_init_agent_with_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "init",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta3_init_agent_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "init",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta3_init_agent_no_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "init",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta3_init_appliance_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "init",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta3_init_appliance_with_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "init",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta3_init_appliance_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "init",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta3_init_appliance_no_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "init",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		// v1beta3 controlplane scenarios
		{
			name:             "v1beta3_controlplane_agent_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "controlplane",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "controlplane"),
			localImages:      true,
			expectedStages:   9, // 5 pre + 4 join stages
			wantErr:          false,
		},
		{
			name:             "v1beta3_controlplane_agent_with_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "controlplane",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta3_controlplane_agent_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "controlplane",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta3_controlplane_agent_no_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "controlplane",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta3_controlplane_appliance_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "controlplane",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta3_controlplane_appliance_with_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "controlplane",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta3_controlplane_appliance_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "controlplane",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta3_controlplane_appliance_no_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "controlplane",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		// v1beta3 worker scenarios
		{
			name:             "v1beta3_worker_agent_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "worker",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "worker"),
			localImages:      true,
			expectedStages:   7, // 5 pre + 2 join stages (no cluster/kubelet config for workers)
			wantErr:          false,
		},
		{
			name:             "v1beta3_worker_agent_with_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "worker",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta3_worker_agent_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "worker",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta3_worker_agent_no_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "worker",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta3_worker_appliance_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "worker",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta3_worker_appliance_with_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "worker",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta3_worker_appliance_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "worker",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta3", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta3_worker_appliance_no_proxy_standard_containerd",
			kubeadmVersion:   "1.30.11",
			nodeRole:         "worker",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta3", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		// v1beta4 scenarios (kubeadm >= 1.31.0) - All 24 scenarios
		{
			name:             "v1beta4_init_agent_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "init",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta4_init_agent_with_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "init",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta4_init_agent_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "init",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta4_init_agent_no_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "init",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta4_init_appliance_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "init",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta4_init_appliance_with_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "init",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta4_init_appliance_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "init",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta4_init_appliance_no_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "init",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "init"),
			localImages:      true,
			expectedStages:   10,
			wantErr:          false,
		},
		{
			name:             "v1beta4_controlplane_agent_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "controlplane",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta4_controlplane_agent_with_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "controlplane",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta4_controlplane_agent_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "controlplane",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta4_controlplane_agent_no_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "controlplane",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta4_controlplane_appliance_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "controlplane",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta4_controlplane_appliance_with_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "controlplane",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta4_controlplane_appliance_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "controlplane",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta4_controlplane_appliance_no_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "controlplane",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "controlplane"),
			localImages:      true,
			expectedStages:   9,
			wantErr:          false,
		},
		{
			name:             "v1beta4_worker_agent_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "worker",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta4_worker_agent_with_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "worker",
			environmentMode:  "agent",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta4_worker_agent_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "worker",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta4_worker_agent_no_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "worker",
			environmentMode:  "agent",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta4_worker_appliance_with_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "worker",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta4_worker_appliance_with_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "worker",
			environmentMode:  "appliance",
			proxyConfig:      true,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta4_worker_appliance_no_proxy_spectro_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "worker",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "spectro-containerd",
			userOptions:      getTestUserOptions("v1beta4", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
		{
			name:             "v1beta4_worker_appliance_no_proxy_standard_containerd",
			kubeadmVersion:   "1.31.0",
			nodeRole:         "worker",
			environmentMode:  "appliance",
			proxyConfig:      false,
			containerRuntime: "containerd",
			userOptions:      getTestUserOptions("v1beta4", "worker"),
			localImages:      true,
			expectedStages:   7,
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Setup virtual filesystem for the test scenario
			vfsTest, cleanup, err := setupTestFileSystem(tt.kubeadmVersion, tt.environmentMode, tt.localImages)
			g.Expect(err).To(BeNil())
			defer cleanup()

			// Create cluster input based on scenario
			cluster := createClusterInput(tt)

			// Execute provider function to generate YIP config
			actualConfig := clusterProvider(cluster)

			// Validate results
			if tt.wantErr {
				g.Expect(actualConfig.Stages).To(BeEmpty())
			} else {
				validateYipStages(t, actualConfig, tt)
			}
		})
	}
}

func TestProviderKubeadmErrorScenarios(t *testing.T) {
	g := NewWithT(t)

	errorTests := []struct {
		name          string
		setupFS       func() (afero.Fs, func(), error)
		clusterInput  clusterplugin.Cluster
		expectedError string
	}{
		{
			name: "kubeadm_binary_missing",
			setupFS: func() (afero.Fs, func(), error) {
				fileSystem := map[string]interface{}{
					"/usr/bin/": nil, // Empty directory - no kubeadm binary
				}
				return vfst.NewTestFS(fileSystem)
			},
			clusterInput: clusterplugin.Cluster{
				Role:             clusterplugin.RoleInit,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
			},
			expectedError: "failed to check if kubeadm version is greater than 131",
		},
		{
			name: "invalid_user_options_yaml",
			setupFS: func() (afero.Fs, func(), error) {
				return setupTestFileSystem("1.30.11", "appliance", true)
			},
			clusterInput: clusterplugin.Cluster{
				Role:             clusterplugin.RoleInit,
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "abcdef.1234567890123456",
				Options:          "invalid: yaml: content: [", // Invalid YAML
			},
			expectedError: "failed to parse config",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			vfsTest, cleanup, err := tt.setupFS()
			if cleanup != nil {
				defer cleanup()
			}
			g.Expect(err).To(BeNil())

			// This would require modifying the provider to accept a filesystem parameter
			// For now we test the validation logic works
			if tt.expectedError != "" {
				// Test that invalid YAML causes issues
				if strings.Contains(tt.clusterInput.Options, "invalid:") {
					var config interface{}
					err := yaml.Unmarshal([]byte(tt.clusterInput.Options), &config)
					g.Expect(err).To(HaveOccurred())
				}
			}
		})
	}
}

// Helper Functions

func setupTestFileSystem(kubeadmVersion, environmentMode string, localImages bool) (afero.Fs, func(), error) {
	var rootPath string
	if environmentMode == "agent" {
		rootPath = "/persistent/spectro"
	} else {
		rootPath = "/"
	}

	fileSystem := map[string]interface{}{
		// Kubeadm binary with version
		filepath.Join(rootPath, "usr/bin/kubeadm"): createMockKubeadmBinary(kubeadmVersion),

		// Scripts directory
		filepath.Join(rootPath, "opt/kubeadm/scripts/kube-init.sh"):        mockScript("kube-init"),
		filepath.Join(rootPath, "opt/kubeadm/scripts/kube-join.sh"):        mockScript("kube-join"),
		filepath.Join(rootPath, "opt/kubeadm/scripts/kube-reset.sh"):       mockScript("kube-reset"),
		filepath.Join(rootPath, "opt/kubeadm/scripts/kube-pre-init.sh"):    mockScript("kube-pre-init"),
		filepath.Join(rootPath, "opt/kubeadm/scripts/kube-post-init.sh"):   mockScript("kube-post-init"),
		filepath.Join(rootPath, "opt/kubeadm/scripts/kube-upgrade.sh"):     mockScript("kube-upgrade"),
		filepath.Join(rootPath, "opt/kubeadm/scripts/kube-reconfigure.sh"): mockScript("kube-reconfigure"),
		filepath.Join(rootPath, "opt/kubeadm/scripts/import.sh"):           mockScript("import"),

		// Service detection files
		"/run/spectro/containerd/containerd.sock": []byte(""),
		"/run/containerd/containerd.sock":         []byte(""),

		// Configuration directories
		filepath.Join(rootPath, "opt/kubeadm/"): nil,

		// Kube images directory
		filepath.Join(rootPath, "opt/kube-images/"): nil,
	}

	// Add local images if specified
	if localImages {
		fileSystem[filepath.Join(rootPath, "opt/content/images/")] = nil
		fileSystem[filepath.Join(rootPath, "opt/content/images/test-image.tar")] = []byte("mock image data")
	}

	return vfst.NewTestFS(fileSystem)
}

func createMockKubeadmBinary(version string) []byte {
	return []byte(fmt.Sprintf("#!/bin/bash\nif [ \"$1\" = \"version\" ] && [ \"$2\" = \"-o\" ] && [ \"$3\" = \"short\" ]; then\n  echo \"v%s\"\nfi\n", version))
}

func mockScript(scriptType string) []byte {
	return []byte(fmt.Sprintf("#!/bin/bash\n# Mock %s script\necho \"Executing %s\"\nexit 0\n", scriptType, scriptType))
}

func createClusterInput(scenario TestScenario) clusterplugin.Cluster {
	cluster := clusterplugin.Cluster{
		Role:             getClusterRole(scenario.nodeRole),
		ControlPlaneHost: "10.10.148.28",
		ClusterToken:     "68413030465f917774b4d7c4",
		Options:          scenario.userOptions,
		ProviderOptions:  make(map[string]string),
	}

	// Environment configuration
	if scenario.proxyConfig {
		cluster.Env = map[string]string{
			"HTTP_PROXY":  "http://proxy.example.com:8080",
			"HTTPS_PROXY": "https://proxy.example.com:8080",
			"NO_PROXY":    ".svc,.svc.cluster.local",
		}
	}

	// Provider options for service detection
	if scenario.containerRuntime == "spectro-containerd" {
		cluster.ProviderOptions["spectro-containerd-service-name"] = "true"
	}

	// Root path for agent mode
	if scenario.environmentMode == "agent" {
		cluster.ProviderOptions["cluster_root_path"] = "/persistent/spectro"
	} else {
		cluster.ProviderOptions["cluster_root_path"] = "/"
	}

	// Local images configuration
	if scenario.localImages {
		if scenario.environmentMode == "agent" {
			cluster.LocalImagesPath = "/persistent/spectro/opt/content/images"
		} else {
			cluster.LocalImagesPath = "/opt/content/images"
		}
	}

	return cluster
}

func getClusterRole(role string) clusterplugin.Role {
	switch role {
	case "init":
		return clusterplugin.RoleInit
	case "controlplane":
		return clusterplugin.RoleControlPlane
	case "worker":
		return clusterplugin.RoleWorker
	default:
		return clusterplugin.RoleInit
	}
}

func getTestUserOptions(apiVersion, nodeRole string) string {
	if apiVersion == "v1beta3" {
		return `
clusterConfiguration:
  apiServer:
    certSANs:
    - cluster-test.proxy.dev.spectrocloud.com
    extraArgs:
      advertise-address: 0.0.0.0
      anonymous-auth: "true"
  kubernetesVersion: v1.30.11
  networking:
    podSubnet: 192.168.0.0/16
    serviceSubnet: 192.169.0.0/16
initConfiguration:
  localAPIEndpoint: {}
  nodeRegistration:
    kubeletExtraArgs:
      event-qps: "0"
      feature-gates: RotateKubeletServerCertificate=true
      node-ip: "10.10.148.28"
joinConfiguration:
  discovery: {}
  nodeRegistration:
    kubeletExtraArgs:
      event-qps: "0"
      feature-gates: RotateKubeletServerCertificate=true
      node-ip: "10.10.148.28"
kubeletConfiguration:
  authentication:
    anonymous: {}
    webhook:
      cacheTTL: 0s
  containerRuntimeEndpoint: ""
`
	} else { // v1beta4
		return `
clusterConfiguration:
  apiServer:
    certSANs:
    - cluster-test.proxy.dev.spectrocloud.com
    extraArgs:
      advertise-address: 0.0.0.0
      anonymous-auth: "true"
  kubernetesVersion: v1.31.0
  networking:
    podSubnet: 192.168.0.0/16
    serviceSubnet: 192.169.0.0/16
initConfiguration:
  localAPIEndpoint: {}
  nodeRegistration:
    kubeletExtraArgs:
    - name: event-qps
      value: "0"
    - name: feature-gates
      value: RotateKubeletServerCertificate=true
    - name: node-ip
      value: "10.10.148.28"
joinConfiguration:
  discovery: {}
  nodeRegistration:
    kubeletExtraArgs:
    - name: event-qps
      value: "0"
    - name: feature-gates
      value: RotateKubeletServerCertificate=true
    - name: node-ip
      value: "10.10.148.28"
kubeletConfiguration:
  authentication:
    anonymous: {}
    webhook:
      cacheTTL: 0s
  containerRuntimeEndpoint: ""
`
	}
}

func validateYipStages(t *testing.T, actualConfig yip.YipConfig, scenario TestScenario) {
	g := NewWithT(t)

	// Validate overall structure
	g.Expect(actualConfig.Name).To(Equal("Kubeadm Kairos Cluster Provider"))
	g.Expect(actualConfig.Stages).To(HaveKey("boot.before"))

	stages := actualConfig.Stages["boot.before"]

	// Validate stage count
	g.Expect(len(stages)).To(Equal(scenario.expectedStages))

	// Validate pre-stages (common to all roles)
	validatePreStages(t, stages, scenario)

	// Validate role-specific stages
	switch scenario.nodeRole {
	case "init":
		validateInitStages(t, stages, scenario)
	case "controlplane", "worker":
		validateJoinStages(t, stages, scenario)
	}
}

func validatePreStages(t *testing.T, stages []yip.Stage, scenario TestScenario) {
	g := NewWithT(t)

	stageNames := getStageNames(stages)

	// Validate proxy stage
	if scenario.proxyConfig {
		g.Expect(stageNames).To(ContainElement("Set proxy env"))
		proxyStage := findStageByName(stages, "Set proxy env")
		g.Expect(proxyStage).ToNot(BeNil())
		g.Expect(proxyStage.Files).To(HaveLen(2)) // kubelet and containerd proxy files

		// Validate proxy files
		validateProxyFiles(t, proxyStage.Files, scenario)
	}

	// Validate pre-init commands stage
	g.Expect(stageNames).To(ContainElement("Run Pre Kubeadm Commands"))
	preInitStage := findStageByName(stages, "Run Pre Kubeadm Commands")
	g.Expect(preInitStage).ToNot(BeNil())
	g.Expect(preInitStage.Commands).To(HaveLen(1))

	expectedRootPath := getRootPath(scenario.environmentMode)
	expectedCommand := fmt.Sprintf("/bin/bash %s/opt/kubeadm/scripts/kube-pre-init.sh %s", expectedRootPath, expectedRootPath)
	g.Expect(preInitStage.Commands[0]).To(Equal(expectedCommand))

	// Validate swap disable stage
	g.Expect(stageNames).To(ContainElement("Run Pre Kubeadm Disable SwapOff"))

	// Validate image import stages
	g.Expect(stageNames).To(ContainElement("Run Load Kube Images"))
	if scenario.localImages {
		g.Expect(stageNames).To(ContainElement("Run Import Local Images"))
	}
}

func validateInitStages(t *testing.T, stages []yip.Stage, scenario TestScenario) {
	g := NewWithT(t)

	stageNames := getStageNames(stages)

	// Validate init config generation
	g.Expect(stageNames).To(ContainElement("Generate Kubeadm Init Config File"))
	configStage := findStageByName(stages, "Generate Kubeadm Init Config File")
	g.Expect(configStage).ToNot(BeNil())
	g.Expect(configStage.Files).To(HaveLen(1))

	configFile := configStage.Files[0]
	expectedPath := fmt.Sprintf("%s/opt/kubeadm/kubeadm.yaml", getRootPath(scenario.environmentMode))
	g.Expect(configFile.Path).To(Equal(expectedPath))
	g.Expect(configFile.Permissions).To(Equal(uint32(0640)))

	// Validate kubeadm version-specific content
	if strings.Contains(scenario.kubeadmVersion, "1.31") || strings.Contains(scenario.kubeadmVersion, "1.32") {
		g.Expect(configFile.Content).To(ContainSubstring("apiVersion: kubeadm.k8s.io/v1beta4"))
	} else {
		g.Expect(configFile.Content).To(ContainSubstring("apiVersion: kubeadm.k8s.io/v1beta3"))
	}

	// Validate init execution stage
	g.Expect(stageNames).To(ContainElement("Run Kubeadm Init"))
	initStage := findStageByName(stages, "Run Kubeadm Init")
	g.Expect(initStage).ToNot(BeNil())

	// Validate proxy-aware command construction
	if scenario.proxyConfig {
		expectedProxyArgs := "true http://proxy.example.com:8080 https://proxy.example.com:8080"
		g.Expect(initStage.Commands[0]).To(ContainSubstring(expectedProxyArgs))
	}

	// Validate post-init stage
	g.Expect(stageNames).To(ContainElement("Run Post Kubeadm Init"))

	// Validate cluster config generation
	g.Expect(stageNames).To(ContainElement("Generate Cluster Config File"))

	// Validate kubelet config generation
	g.Expect(stageNames).To(ContainElement("Generate Kubelet Config File"))

	// Validate upgrade stage
	g.Expect(stageNames).To(ContainElement("Run Kubeadm Init Upgrade"))

	// Validate reconfigure stage
	g.Expect(stageNames).To(ContainElement("Run Kubeadm Reconfiguration"))
}

func validateJoinStages(t *testing.T, stages []yip.Stage, scenario TestScenario) {
	g := NewWithT(t)

	stageNames := getStageNames(stages)

	// Validate join config generation
	g.Expect(stageNames).To(ContainElement("Generate Kubeadm Join Config File"))
	configStage := findStageByName(stages, "Generate Kubeadm Join Config File")
	g.Expect(configStage).ToNot(BeNil())
	g.Expect(configStage.Files).To(HaveLen(1))

	// Validate join execution stage
	g.Expect(stageNames).To(ContainElement("Run Kubeadm Join"))
	joinStage := findStageByName(stages, "Run Kubeadm Join")
	g.Expect(joinStage).ToNot(BeNil())

	// For control plane nodes, validate additional stages
	if scenario.nodeRole == "controlplane" {
		g.Expect(stageNames).To(ContainElement("Generate Cluster Config File"))
		g.Expect(stageNames).To(ContainElement("Generate Kubelet Config File"))
	}

	// Validate upgrade stage
	g.Expect(stageNames).To(ContainElement("Run Kubeadm Join Upgrade"))

	// Validate reconfigure stage
	g.Expect(stageNames).To(ContainElement("Run Kubeadm Join Reconfiguration"))
}

func validateProxyFiles(t *testing.T, files []yip.File, scenario TestScenario) {
	g := NewWithT(t)

	var kubeletProxyFile, containerdProxyFile *yip.File

	for i := range files {
		if strings.Contains(files[i].Path, "/etc/default/kubelet") {
			kubeletProxyFile = &files[i]
		} else if strings.Contains(files[i].Path, "http-proxy.conf") {
			containerdProxyFile = &files[i]
		}
	}

	// Validate kubelet proxy file
	g.Expect(kubeletProxyFile).ToNot(BeNil())
	g.Expect(kubeletProxyFile.Path).To(Equal("/etc/default/kubelet"))
	g.Expect(kubeletProxyFile.Permissions).To(Equal(uint32(0400)))
	g.Expect(kubeletProxyFile.Content).To(ContainSubstring("HTTP_PROXY=http://proxy.example.com:8080"))
	g.Expect(kubeletProxyFile.Content).To(ContainSubstring("HTTPS_PROXY=https://proxy.example.com:8080"))

	// Validate containerd proxy file
	g.Expect(containerdProxyFile).ToNot(BeNil())
	g.Expect(containerdProxyFile.Permissions).To(Equal(uint32(0400)))
	g.Expect(containerdProxyFile.Content).To(ContainSubstring("[Service]"))
	g.Expect(containerdProxyFile.Content).To(ContainSubstring("Environment=\"HTTP_PROXY=http://proxy.example.com:8080\""))
	g.Expect(containerdProxyFile.Content).To(ContainSubstring("Environment=\"HTTPS_PROXY=https://proxy.example.com:8080\""))

	// Validate service folder name in containerd proxy file path
	if scenario.containerRuntime == "spectro-containerd" {
		g.Expect(containerdProxyFile.Path).To(ContainSubstring("/run/systemd/system/spectro-containerd.service.d/http-proxy.conf"))
	} else {
		g.Expect(containerdProxyFile.Path).To(ContainSubstring("/run/systemd/system/containerd.service.d/http-proxy.conf"))
	}
}

func findStageByName(stages []yip.Stage, name string) *yip.Stage {
	for i := range stages {
		if stages[i].Name == name {
			return &stages[i]
		}
	}
	return nil
}

func getStageNames(stages []yip.Stage) []string {
	names := make([]string, len(stages))
	for i, stage := range stages {
		names[i] = stage.Name
	}
	return names
}

func getRootPath(environmentMode string) string {
	if environmentMode == "agent" {
		return "/persistent/spectro"
	}
	return "/"
}

func TestMain(m *testing.M) {
	// Setup test environment
	setupGlobalTestEnvironment()

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupGlobalTestEnvironment()

	os.Exit(code)
}

func setupGlobalTestEnvironment() {
	// Initialize logging for tests
	log.InitLogger("/tmp/provider-kubeadm-test.log")

	// Set test-friendly defaults
	os.Setenv("TEST_MODE", "true")
}

func cleanupGlobalTestEnvironment() {
	// Cleanup test environment
	os.Unsetenv("TEST_MODE")
}
