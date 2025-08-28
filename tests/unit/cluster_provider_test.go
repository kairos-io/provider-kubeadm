package main

import (
	"testing"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
	. "github.com/onsi/gomega"
)

// TestClusterProvider tests the main clusterProvider function
func TestClusterProvider(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name              string
		cluster           clusterplugin.Cluster
		expectedName      string
		expectedStageKey  string
		minExpectedStages int
		validateFunc      func(*testing.T, yip.YipConfig)
	}{
		{
			name: "basic_init_cluster_v1beta3",
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
			minExpectedStages: 8,
			validateFunc: func(t *testing.T, config yip.YipConfig) {
				g := NewWithT(t)
				stages := config.Stages["boot.before"]

				// Should have init-specific stages
				stageNames := getStageNames(stages)
				g.Expect(stageNames).To(ContainElement("Generate Kubeadm Init Config File"))
				g.Expect(stageNames).To(ContainElement("Run Kubeadm Init"))

				// Should contain v1beta3 configuration
				configStage := findStageByName(stages, "Generate Kubeadm Init Config File")
				g.Expect(configStage).ToNot(BeNil())
				g.Expect(configStage.Files[0].Content).To(ContainSubstring("apiVersion: kubeadm.k8s.io/v1beta3"))
			},
		},
		{
			name: "basic_join_cluster_v1beta4",
			cluster: clusterplugin.Cluster{
				Role:             clusterplugin.RoleControlPlane,
				ControlPlaneHost: "192.168.1.100",
				ClusterToken:     "token.1234567890123456",
				Options: `
clusterConfiguration:
  kubernetesVersion: v1.31.0
joinConfiguration:
  nodeRegistration:
    kubeletExtraArgs:
    - name: node-ip
      value: "192.168.1.101"`,
			},
			expectedName:      "Kubeadm Kairos Cluster Provider",
			expectedStageKey:  "boot.before",
			minExpectedStages: 7,
			validateFunc: func(t *testing.T, config yip.YipConfig) {
				g := NewWithT(t)
				stages := config.Stages["boot.before"]

				// Should have join-specific stages
				stageNames := getStageNames(stages)
				g.Expect(stageNames).To(ContainElement("Generate Kubeadm Join Config File"))
				g.Expect(stageNames).To(ContainElement("Run Kubeadm Join"))

				// Should contain v1beta4 configuration
				configStage := findStageByName(stages, "Generate Kubeadm Join Config File")
				g.Expect(configStage).ToNot(BeNil())
				g.Expect(configStage.Files[0].Content).To(ContainSubstring("apiVersion: kubeadm.k8s.io/v1beta4"))
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
			minExpectedStages: 8, // Includes proxy stage
			validateFunc: func(t *testing.T, config yip.YipConfig) {
				g := NewWithT(t)
				stages := config.Stages["boot.before"]
				stageNames := getStageNames(stages)
				g.Expect(stageNames).To(ContainElement("Set proxy env"))
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

// Helper functions for unit tests
func getStageNames(stages []yip.Stage) []string {
	names := make([]string, len(stages))
	for i, stage := range stages {
		names[i] = stage.Name
	}
	return names
}

func findStageByName(stages []yip.Stage, name string) *yip.Stage {
	for i := range stages {
		if stages[i].Name == name {
			return &stages[i]
		}
	}
	return nil
}
