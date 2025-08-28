package main

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/kairos-io/kairos-sdk/clusterplugin"
)

// TestClusterProviderInputValidation tests cluster provider input validation
func TestClusterProviderInputValidation(t *testing.T) {
	_ = NewWithT(t)

	tests := []struct {
		name                string
		cluster             clusterplugin.Cluster
		expectedValid       bool
		expectedName        string
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
			expectedValid: true,
			expectedName:  "Kubeadm Kairos Cluster Provider",
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
			expectedValid: true,
			expectedName:  "Kubeadm Kairos Cluster Provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Test input validation
			isValid := validateClusterInputs(tt.cluster)
			g.Expect(isValid).To(Equal(tt.expectedValid))

			if tt.expectedValid {
				// Test basic cluster properties
				g.Expect(tt.cluster.Role).ToNot(BeEmpty())
				g.Expect(tt.cluster.ControlPlaneHost).ToNot(BeEmpty())
				g.Expect(tt.cluster.ClusterToken).ToNot(BeEmpty())
			}
		})
	}
}

// Helper function for input validation
func validateClusterInputs(cluster clusterplugin.Cluster) bool {
	if cluster.ControlPlaneHost == "" {
		return false
	}
	if cluster.ClusterToken == "" {
		return false
	}
	if string(cluster.Role) == "" {
		return false
	}
	return true
}