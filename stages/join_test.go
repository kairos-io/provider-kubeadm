package stages

import (
	"testing"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	yip "github.com/mudler/yip/pkg/schema"
	. "github.com/onsi/gomega"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	kubeadmapiv4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
)

// TestGetJoinYipStagesV1Beta3 tests the GetJoinYipStagesV1Beta3 function
func TestGetJoinYipStagesV1Beta3(t *testing.T) {
	tests := []struct {
		name               string
		clusterCtx         *domain.ClusterContext
		kubeadmConfig      domain.KubeadmConfigBeta3
		expectedStageCount int
		validateStages     func(*testing.T, []yip.Stage)
	}{
		{
			name: "worker_node_join",
			clusterCtx: &domain.ClusterContext{
				RootPath:         "/",
				NodeRole:         "worker",
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "test-token.1234567890123456",
				EnvConfig: map[string]string{
					"HTTP_PROXY": "http://proxy.example.com:8080",
				},
			},
			kubeadmConfig: domain.KubeadmConfigBeta3{
				ClusterConfiguration: kubeadmapiv3.ClusterConfiguration{
					APIServer: kubeadmapiv3.APIServer{
						CertSANs: []string{"cluster-test.example.com"},
					},
				},
				JoinConfiguration: kubeadmapiv3.JoinConfiguration{
					NodeRegistration: kubeadmapiv3.NodeRegistrationOptions{
						KubeletExtraArgs: map[string]string{
							"node-ip": "10.0.0.2",
						},
					},
				},
				KubeletConfiguration: kubeletv1beta1.KubeletConfiguration{},
			},
			expectedStageCount: 4, // 2 base + 2 additional stages
			validateStages: func(t *testing.T, stages []yip.Stage) {
				g := NewWithT(t)
				expectedNames := []string{
					"Generate Kubeadm Join Config File",
					"Run Kubeadm Join",
					"Run Kubeadm Join Upgrade",
					"Run Kubeadm Join Reconfiguration",
				}
				for i, expectedName := range expectedNames {
					g.Expect(stages[i].Name).To(Equal(expectedName))
				}
			},
		},
		{
			name: "controlplane_node_join",
			clusterCtx: &domain.ClusterContext{
				RootPath:         "/persistent/spectro",
				NodeRole:         "controlplane",
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "test-token.1234567890123456",
				EnvConfig: map[string]string{
					"HTTP_PROXY": "http://proxy.example.com:8080",
				},
			},
			kubeadmConfig: domain.KubeadmConfigBeta3{
				ClusterConfiguration: kubeadmapiv3.ClusterConfiguration{
					APIServer: kubeadmapiv3.APIServer{
						CertSANs: []string{"cluster-test.example.com"},
					},
				},
				JoinConfiguration: kubeadmapiv3.JoinConfiguration{
					NodeRegistration: kubeadmapiv3.NodeRegistrationOptions{
						KubeletExtraArgs: map[string]string{
							"node-ip": "10.0.0.3",
						},
					},
				},
				KubeletConfiguration: kubeletv1beta1.KubeletConfiguration{},
			},
			expectedStageCount: 6, // 2 base + 2 additional + 2 controlplane stages
			validateStages: func(t *testing.T, stages []yip.Stage) {
				g := NewWithT(t)
				expectedNames := []string{
					"Generate Kubeadm Join Config File",
					"Run Kubeadm Join",
					"Generate Cluster Config File",
					"Generate Kubelet Config File",
					"Run Kubeadm Join Upgrade",
					"Run Kubeadm Join Reconfiguration",
				}
				for i, expectedName := range expectedNames {
					g.Expect(stages[i].Name).To(Equal(expectedName))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetJoinYipStagesV1Beta3(tt.clusterCtx, tt.kubeadmConfig)

			// Validate stage count
			g.Expect(result).To(HaveLen(tt.expectedStageCount))
			tt.validateStages(t, result)
		})
	}
}

// TestGetJoinYipStagesV1Beta4 tests the GetJoinYipStagesV1Beta4 function
func TestGetJoinYipStagesV1Beta4(t *testing.T) {
	tests := []struct {
		name               string
		clusterCtx         *domain.ClusterContext
		kubeadmConfig      domain.KubeadmConfigBeta4
		expectedStageCount int
		validateStages     func(*testing.T, []yip.Stage)
	}{
		{
			name: "worker_node_join_v1beta4",
			clusterCtx: &domain.ClusterContext{
				RootPath:         "/",
				NodeRole:         "worker",
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "test-token.1234567890123456",
				EnvConfig: map[string]string{
					"HTTP_PROXY": "http://proxy.example.com:8080",
				},
			},
			kubeadmConfig: domain.KubeadmConfigBeta4{
				ClusterConfiguration: kubeadmapiv4.ClusterConfiguration{
					APIServer: kubeadmapiv4.APIServer{
						CertSANs: []string{"cluster-test.example.com"},
					},
				},
				JoinConfiguration: kubeadmapiv4.JoinConfiguration{
					NodeRegistration: kubeadmapiv4.NodeRegistrationOptions{
						KubeletExtraArgs: []kubeadmapiv4.Arg{
							{Name: "node-ip", Value: "10.0.0.2"},
						},
					},
				},
				KubeletConfiguration: kubeletv1beta1.KubeletConfiguration{},
			},
			expectedStageCount: 4, // 2 base + 2 additional stages
			validateStages: func(t *testing.T, stages []yip.Stage) {
				g := NewWithT(t)
				expectedNames := []string{
					"Generate Kubeadm Join Config File",
					"Run Kubeadm Join",
					"Run Kubeadm Join Upgrade",
					"Run Kubeadm Join Reconfiguration",
				}
				for i, expectedName := range expectedNames {
					g.Expect(stages[i].Name).To(Equal(expectedName))
				}
			},
		},
		{
			name: "controlplane_node_join_v1beta4",
			clusterCtx: &domain.ClusterContext{
				RootPath:         "/persistent/spectro",
				NodeRole:         "controlplane",
				ControlPlaneHost: "10.0.0.1",
				ClusterToken:     "test-token.1234567890123456",
				EnvConfig: map[string]string{
					"HTTP_PROXY": "http://proxy.example.com:8080",
				},
			},
			kubeadmConfig: domain.KubeadmConfigBeta4{
				ClusterConfiguration: kubeadmapiv4.ClusterConfiguration{
					APIServer: kubeadmapiv4.APIServer{
						CertSANs: []string{"cluster-test.example.com"},
					},
				},
				JoinConfiguration: kubeadmapiv4.JoinConfiguration{
					NodeRegistration: kubeadmapiv4.NodeRegistrationOptions{
						KubeletExtraArgs: []kubeadmapiv4.Arg{
							{Name: "node-ip", Value: "10.0.0.3"},
						},
					},
				},
				KubeletConfiguration: kubeletv1beta1.KubeletConfiguration{},
			},
			expectedStageCount: 6, // 2 base + 2 additional + 2 controlplane stages
			validateStages: func(t *testing.T, stages []yip.Stage) {
				g := NewWithT(t)
				expectedNames := []string{
					"Generate Kubeadm Join Config File",
					"Run Kubeadm Join",
					"Generate Cluster Config File",
					"Generate Kubelet Config File",
					"Run Kubeadm Join Upgrade",
					"Run Kubeadm Join Reconfiguration",
				}
				for i, expectedName := range expectedNames {
					g.Expect(stages[i].Name).To(Equal(expectedName))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetJoinYipStagesV1Beta4(tt.clusterCtx, tt.kubeadmConfig)

			// Validate stage count
			g.Expect(result).To(HaveLen(tt.expectedStageCount))
			tt.validateStages(t, result)
		})
	}
}

// TestGetKubeadmJoinConfigStage tests the getKubeadmJoinConfigStage function
func TestGetKubeadmJoinConfigStage(t *testing.T) {
	tests := []struct {
		name            string
		joinCfg         string
		rootPath        string
		expectedName    string
		expectedPath    string
		expectedContent string
	}{
		{
			name:            "standard_join_config",
			joinCfg:         "apiVersion: v1beta3\nkind: JoinConfiguration",
			rootPath:        "/",
			expectedName:    "Generate Kubeadm Join Config File",
			expectedPath:    "/opt/kubeadm/kubeadm.yaml",
			expectedContent: "apiVersion: v1beta3\nkind: JoinConfiguration",
		},
		{
			name:            "agent_mode_join_config",
			joinCfg:         "apiVersion: v1beta4\nkind: JoinConfiguration",
			rootPath:        "/persistent/spectro",
			expectedName:    "Generate Kubeadm Join Config File",
			expectedPath:    "/persistent/spectro/opt/kubeadm/kubeadm.yaml",
			expectedContent: "apiVersion: v1beta4\nkind: JoinConfiguration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getKubeadmJoinConfigStage(tt.joinCfg, tt.rootPath)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Files).To(HaveLen(1))
			g.Expect(result.Files[0].Path).To(Equal(tt.expectedPath))
			g.Expect(result.Files[0].Content).To(Equal(tt.expectedContent))
		})
	}
}

// TestGetKubeadmJoinStage tests the getKubeadmJoinStage function
func TestGetKubeadmJoinStage(t *testing.T) {
	tests := []struct {
		name                 string
		clusterCtx           *domain.ClusterContext
		expectedName         string
		expectedCommandCount int
		validateCommands     func(*testing.T, []string)
	}{
		{
			name: "with_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath: "/",
				NodeRole: "worker",
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
				},
			},
			expectedName:         "Run Kubeadm Join",
			expectedCommandCount: 2,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(ContainSubstring("bash /opt/kubeadm/scripts/kube-join.sh worker"))
				g.Expect(commands[0]).To(ContainSubstring("http://proxy.example.com:8080"))
			},
		},
		{
			name: "without_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath:  "/persistent/spectro",
				NodeRole:  "controlplane",
				EnvConfig: map[string]string{},
			},
			expectedName:         "Run Kubeadm Join",
			expectedCommandCount: 2,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("bash /persistent/spectro/opt/kubeadm/scripts/kube-join.sh controlplane /persistent/spectro"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getKubeadmJoinStage(tt.clusterCtx)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(HaveLen(tt.expectedCommandCount))
			tt.validateCommands(t, result.Commands)
		})
	}
}

// TestGetKubeadmJoinUpgradeStage tests the getKubeadmJoinUpgradeStage function
func TestGetKubeadmJoinUpgradeStage(t *testing.T) {
	tests := []struct {
		name                 string
		clusterCtx           *domain.ClusterContext
		expectedName         string
		expectedCommandCount int
		validateCommands     func(*testing.T, []string)
	}{
		{
			name: "with_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath: "/",
				NodeRole: "worker",
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
				},
			},
			expectedName:         "Run Kubeadm Join Upgrade",
			expectedCommandCount: 1,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(ContainSubstring("bash /opt/kubeadm/scripts/kube-upgrade.sh worker / true"))
				g.Expect(commands[0]).To(ContainSubstring("http://proxy.example.com:8080"))
			},
		},
		{
			name: "without_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath:  "/persistent/spectro",
				NodeRole:  "controlplane",
				EnvConfig: map[string]string{},
			},
			expectedName:         "Run Kubeadm Join Upgrade",
			expectedCommandCount: 1,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("bash /persistent/spectro/opt/kubeadm/scripts/kube-upgrade.sh controlplane /persistent/spectro"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getKubeadmJoinUpgradeStage(tt.clusterCtx)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(HaveLen(tt.expectedCommandCount))
			tt.validateCommands(t, result.Commands)
		})
	}
}

// TestGetKubeadmJoinCreateClusterConfigStage tests the getKubeadmJoinCreateClusterConfigStage function
func TestGetKubeadmJoinCreateClusterConfigStage(t *testing.T) {
	t.Run("create_cluster_config_stage", func(t *testing.T) {
		g := NewWithT(t)

		clusterCfg := &kubeadmapiv3.ClusterConfiguration{}
		initCfg := &kubeadmapiv3.InitConfiguration{}
		joinCfg := &kubeadmapiv3.JoinConfiguration{}
		rootPath := "/"

		result := getKubeadmJoinCreateClusterConfigStage(clusterCfg, initCfg, joinCfg, rootPath)

		// Validate stage structure
		g.Expect(result.Name).To(Equal("Generate Cluster Config File"))
		g.Expect(result.Files).To(HaveLen(1))
		g.Expect(result.Files[0].Path).To(Equal("/opt/kubeadm/cluster-config.yaml"))
		g.Expect(result.Files[0].Content).To(ContainSubstring("apiVersion:"))
	})
}

// TestGetKubeadmJoinCreateKubeletConfigStage tests the getKubeadmJoinCreateKubeletConfigStage function
func TestGetKubeadmJoinCreateKubeletConfigStage(t *testing.T) {
	t.Run("create_kubelet_config_stage", func(t *testing.T) {
		g := NewWithT(t)

		clusterCfg := &kubeadmapiv3.ClusterConfiguration{}
		initCfg := &kubeadmapiv3.InitConfiguration{}
		kubeletCfg := &kubeletv1beta1.KubeletConfiguration{}
		rootPath := "/"

		result := getKubeadmJoinCreateKubeletConfigStage(clusterCfg, initCfg, kubeletCfg, rootPath)

		// Validate stage structure
		g.Expect(result.Name).To(Equal("Generate Kubelet Config File"))
		g.Expect(result.Files).To(HaveLen(1))
		g.Expect(result.Files[0].Path).To(Equal("/opt/kubeadm/kubelet-config.yaml"))
		g.Expect(result.Files[0].Content).To(ContainSubstring("apiVersion:"))
	})
}

// TestGetKubeadmJoinReconfigureStage tests the getKubeadmJoinReconfigureStage function
func TestGetKubeadmJoinReconfigureStage(t *testing.T) {
	tests := []struct {
		name                 string
		clusterCtx           *domain.ClusterContext
		expectedName         string
		expectedCommandCount int
		validateCommands     func(*testing.T, []string)
	}{
		{
			name: "with_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath: "/",
				NodeRole: "worker",
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
				},
			},
			expectedName:         "Run Kubeadm Join Reconfiguration",
			expectedCommandCount: 1,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(ContainSubstring("bash /opt/kubeadm/scripts/kube-reconfigure.sh worker"))
				g.Expect(commands[0]).To(ContainSubstring("http://proxy.example.com:8080"))
			},
		},
		{
			name: "without_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath:  "/persistent/spectro",
				NodeRole:  "controlplane",
				EnvConfig: map[string]string{},
			},
			expectedName:         "Run Kubeadm Join Reconfiguration",
			expectedCommandCount: 1,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(ContainSubstring("bash /persistent/spectro/opt/kubeadm/scripts/kube-reconfigure.sh controlplane"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getKubeadmJoinReconfigureStage(tt.clusterCtx)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(HaveLen(tt.expectedCommandCount))
			tt.validateCommands(t, result.Commands)
		})
	}
}
