package stages

import (
	"testing"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	. "github.com/onsi/gomega"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	kubeadmapiv4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
)

// TestGetInitYipStagesV1Beta3 tests the GetInitYipStagesV1Beta3 function
func TestGetInitYipStagesV1Beta3(t *testing.T) {
	t.Run("init_stages_v1beta3", func(t *testing.T) {
		g := NewWithT(t)

		clusterCtx := &domain.ClusterContext{
			RootPath:         "/",
			NodeRole:         "init",
			ControlPlaneHost: "10.0.0.1",
			ClusterToken:     "abcdef.1234567890123456",
			EnvConfig: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
			},
		}

		kubeadmConfig := domain.KubeadmConfigBeta3{
			ClusterConfiguration: kubeadmapiv3.ClusterConfiguration{
				APIServer: kubeadmapiv3.APIServer{
					CertSANs: []string{"cluster-test.example.com"},
				},
			},
			InitConfiguration: kubeadmapiv3.InitConfiguration{
				NodeRegistration: kubeadmapiv3.NodeRegistrationOptions{
					KubeletExtraArgs: map[string]string{
						"node-ip": "10.0.0.1",
					},
				},
			},
			KubeletConfiguration: kubeletv1beta1.KubeletConfiguration{},
		}

		result := GetInitYipStagesV1Beta3(clusterCtx, kubeadmConfig)

		// Validate that we get the expected number of stages
		g.Expect(result).To(HaveLen(7))

		// Validate stage names
		expectedStageNames := []string{
			"Generate Kubeadm Init Config File",
			"Run Kubeadm Init",
			"Run Post Kubeadm Init",
			"Generate Cluster Config File",
			"Generate Kubelet Config File",
			"Run Kubeadm Init Upgrade",
			"Run Kubeadm Reconfiguration",
		}

		for i, expectedName := range expectedStageNames {
			g.Expect(result[i].Name).To(Equal(expectedName))
		}
	})
}

// TestGetInitYipStagesV1Beta4 tests the GetInitYipStagesV1Beta4 function
func TestGetInitYipStagesV1Beta4(t *testing.T) {
	t.Run("init_stages_v1beta4", func(t *testing.T) {
		g := NewWithT(t)

		clusterCtx := &domain.ClusterContext{
			RootPath:         "/",
			NodeRole:         "init",
			ControlPlaneHost: "10.0.0.1",
			ClusterToken:     "abcdef.1234567890123456",
			EnvConfig: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
			},
		}

		kubeadmConfig := domain.KubeadmConfigBeta4{
			ClusterConfiguration: kubeadmapiv4.ClusterConfiguration{
				Networking: kubeadmapiv4.Networking{
					ServiceSubnet: "10.96.0.0/12",
					PodSubnet:     "192.168.0.0/16",
				},
				APIServer: kubeadmapiv4.APIServer{
					CertSANs: []string{"cluster-test.example.com"},
				},
			},
			InitConfiguration: kubeadmapiv4.InitConfiguration{
				NodeRegistration: kubeadmapiv4.NodeRegistrationOptions{
					KubeletExtraArgs: []kubeadmapiv4.Arg{
						{Name: "node-ip", Value: "10.0.0.1"},
					},
				},
			},
			KubeletConfiguration: kubeletv1beta1.KubeletConfiguration{},
		}

		result := GetInitYipStagesV1Beta4(clusterCtx, kubeadmConfig)

		// Validate that we get the expected number of stages
		g.Expect(result).To(HaveLen(7))

		// Validate stage names
		expectedStageNames := []string{
			"Generate Kubeadm Init Config File",
			"Run Kubeadm Init",
			"Run Post Kubeadm Init",
			"Generate Cluster Config File",
			"Generate Kubelet Config File",
			"Run Kubeadm Init Upgrade",
			"Run Kubeadm Reconfiguration",
		}

		for i, expectedName := range expectedStageNames {
			g.Expect(result[i].Name).To(Equal(expectedName))
		}

		// Validate that ServiceCidr and ClusterCidr are set
		g.Expect(clusterCtx.ServiceCidr).To(Equal("10.96.0.0/12"))
		g.Expect(clusterCtx.ClusterCidr).To(Equal("192.168.0.0/16"))
	})
}

// TestGetKubeadmInitConfigStage tests the getKubeadmInitConfigStage function
func TestGetKubeadmInitConfigStage(t *testing.T) {
	tests := []struct {
		name            string
		kubeadmCfg      string
		rootPath        string
		expectedName    string
		expectedPath    string
		expectedContent string
	}{
		{
			name:            "standard_config",
			kubeadmCfg:      "apiVersion: v1beta3\nkind: InitConfiguration",
			rootPath:        "/",
			expectedName:    "Generate Kubeadm Init Config File",
			expectedPath:    "/opt/kubeadm/kubeadm.yaml",
			expectedContent: "apiVersion: v1beta3\nkind: InitConfiguration",
		},
		{
			name:            "agent_mode_config",
			kubeadmCfg:      "apiVersion: v1beta4\nkind: InitConfiguration",
			rootPath:        "/persistent/spectro",
			expectedName:    "Generate Kubeadm Init Config File",
			expectedPath:    "/persistent/spectro/opt/kubeadm/kubeadm.yaml",
			expectedContent: "apiVersion: v1beta4\nkind: InitConfiguration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getKubeadmInitConfigStage(tt.kubeadmCfg, tt.rootPath)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Files).To(HaveLen(1))
			g.Expect(result.Files[0].Path).To(Equal(tt.expectedPath))
			g.Expect(result.Files[0].Content).To(Equal(tt.expectedContent))
		})
	}
}

// TestGetKubeadmInitStage tests the getKubeadmInitStage function
func TestGetKubeadmInitStage(t *testing.T) {
	tests := []struct {
		name                 string
		clusterCtx           *domain.ClusterContext
		expectedName         string
		expectedCondition    string
		expectedCommandCount int
		validateCommands     func(*testing.T, []string)
	}{
		{
			name: "with_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath: "/",
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
				},
			},
			expectedName:         "Run Kubeadm Init",
			expectedCondition:    "[ ! -f /opt/kubeadm.init ]",
			expectedCommandCount: 2,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(ContainSubstring("bash /opt/kubeadm/scripts/kube-init.sh / true"))
				g.Expect(commands[0]).To(ContainSubstring("http://proxy.example.com:8080"))
				g.Expect(commands[1]).To(Equal("touch /opt/kubeadm.init"))
			},
		},
		{
			name: "without_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath:  "/persistent/spectro",
				EnvConfig: map[string]string{},
			},
			expectedName:         "Run Kubeadm Init",
			expectedCondition:    "[ ! -f /persistent/spectro/opt/kubeadm.init ]",
			expectedCommandCount: 2,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("bash /persistent/spectro/opt/kubeadm/scripts/kube-init.sh /persistent/spectro"))
				g.Expect(commands[1]).To(Equal("touch /persistent/spectro/opt/kubeadm.init"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getKubeadmInitStage(tt.clusterCtx)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.If).To(Equal(tt.expectedCondition))
			g.Expect(result.Commands).To(HaveLen(tt.expectedCommandCount))
			tt.validateCommands(t, result.Commands)
		})
	}
}

// TestGetKubeadmPostInitStage tests the getKubeadmPostInitStage function
func TestGetKubeadmPostInitStage(t *testing.T) {
	tests := []struct {
		name                 string
		clusterRootPath      string
		expectedName         string
		expectedCondition    string
		expectedCommandCount int
		validateCommands     func(*testing.T, []string)
	}{
		{
			name:                 "standard_root_path",
			clusterRootPath:      "/",
			expectedName:         "Run Post Kubeadm Init",
			expectedCondition:    "[ ! -f /opt/post-kubeadm.init ]",
			expectedCommandCount: 2,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("bash /opt/kubeadm/scripts/kube-post-init.sh /"))
				g.Expect(commands[1]).To(Equal("touch /opt/post-kubeadm.init"))
			},
		},
		{
			name:                 "agent_mode_root_path",
			clusterRootPath:      "/persistent/spectro",
			expectedName:         "Run Post Kubeadm Init",
			expectedCondition:    "[ ! -f /persistent/spectro/opt/post-kubeadm.init ]",
			expectedCommandCount: 2,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("bash /persistent/spectro/opt/kubeadm/scripts/kube-post-init.sh /persistent/spectro"))
				g.Expect(commands[1]).To(Equal("touch /persistent/spectro/opt/post-kubeadm.init"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getKubeadmPostInitStage(tt.clusterRootPath)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.If).To(Equal(tt.expectedCondition))
			g.Expect(result.Commands).To(HaveLen(tt.expectedCommandCount))
			tt.validateCommands(t, result.Commands)
		})
	}
}

// TestGetKubeadmInitUpgradeStage tests the getKubeadmInitUpgradeStage function
func TestGetKubeadmInitUpgradeStage(t *testing.T) {
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
				NodeRole: "init",
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
				},
			},
			expectedName:         "Run Kubeadm Init Upgrade",
			expectedCommandCount: 1,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(ContainSubstring("bash /opt/kubeadm/scripts/kube-upgrade.sh init / true"))
				g.Expect(commands[0]).To(ContainSubstring("http://proxy.example.com:8080"))
			},
		},
		{
			name: "without_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath:  "/persistent/spectro",
				NodeRole:  "init",
				EnvConfig: map[string]string{},
			},
			expectedName:         "Run Kubeadm Init Upgrade",
			expectedCommandCount: 1,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(Equal("bash /persistent/spectro/opt/kubeadm/scripts/kube-upgrade.sh init /persistent/spectro"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getKubeadmInitUpgradeStage(tt.clusterCtx)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(HaveLen(tt.expectedCommandCount))
			tt.validateCommands(t, result.Commands)
		})
	}
}

// TestGetKubeadmInitCreateClusterConfigStage tests the getKubeadmInitCreateClusterConfigStage function
func TestGetKubeadmInitCreateClusterConfigStage(t *testing.T) {
	t.Run("create_cluster_config_stage", func(t *testing.T) {
		g := NewWithT(t)

		clusterCfg := &kubeadmapiv3.ClusterConfiguration{}
		initCfg := &kubeadmapiv3.InitConfiguration{}
		rootPath := "/"

		result := getKubeadmInitCreateClusterConfigStage(clusterCfg, initCfg, rootPath)

		// Validate stage structure
		g.Expect(result.Name).To(Equal("Generate Cluster Config File"))
		g.Expect(result.Files).To(HaveLen(1))
		g.Expect(result.Files[0].Path).To(Equal("/opt/kubeadm/cluster-config.yaml"))
		g.Expect(result.Files[0].Content).To(ContainSubstring("apiVersion:"))
	})
}

// TestGetKubeadmInitCreateKubeletConfigStage tests the getKubeadmInitCreateKubeletConfigStage function
func TestGetKubeadmInitCreateKubeletConfigStage(t *testing.T) {
	t.Run("create_kubelet_config_stage", func(t *testing.T) {
		g := NewWithT(t)

		clusterCfg := &kubeadmapiv3.ClusterConfiguration{}
		initCfg := &kubeadmapiv3.InitConfiguration{}
		kubeletCfg := &kubeletv1beta1.KubeletConfiguration{}
		rootPath := "/"

		result := getKubeadmInitCreateKubeletConfigStage(clusterCfg, initCfg, kubeletCfg, rootPath)

		// Validate stage structure
		g.Expect(result.Name).To(Equal("Generate Kubelet Config File"))
		g.Expect(result.Files).To(HaveLen(1))
		g.Expect(result.Files[0].Path).To(Equal("/opt/kubeadm/kubelet-config.yaml"))
		g.Expect(result.Files[0].Content).To(ContainSubstring("apiVersion:"))
	})
}

// TestGetKubeadmInitReconfigureStage tests the getKubeadmInitReconfigureStage function
func TestGetKubeadmInitReconfigureStage(t *testing.T) {
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
				NodeRole: "init",
				EnvConfig: map[string]string{
					"HTTP_PROXY":  "http://proxy.example.com:8080",
					"HTTPS_PROXY": "https://proxy.example.com:8080",
				},
			},
			expectedName:         "Run Kubeadm Reconfiguration",
			expectedCommandCount: 1,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(ContainSubstring("bash /opt/kubeadm/scripts/kube-reconfigure.sh init"))
				g.Expect(commands[0]).To(ContainSubstring("http://proxy.example.com:8080"))
			},
		},
		{
			name: "without_proxy_configuration",
			clusterCtx: &domain.ClusterContext{
				RootPath:  "/persistent/spectro",
				NodeRole:  "init",
				EnvConfig: map[string]string{},
			},
			expectedName:         "Run Kubeadm Reconfiguration",
			expectedCommandCount: 1,
			validateCommands: func(t *testing.T, commands []string) {
				g := NewWithT(t)
				g.Expect(commands[0]).To(ContainSubstring("bash /persistent/spectro/opt/kubeadm/scripts/kube-reconfigure.sh init"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getKubeadmInitReconfigureStage(tt.clusterCtx)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Commands).To(HaveLen(tt.expectedCommandCount))
			tt.validateCommands(t, result.Commands)
		})
	}
}
