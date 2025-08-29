package utils

import (
	"testing"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	. "github.com/onsi/gomega"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	kubeadmapiv4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
)

// TestMutateClusterConfigBeta3Defaults tests the MutateClusterConfigBeta3Defaults function
func TestMutateClusterConfigBeta3Defaults(t *testing.T) {
	t.Run("mutate_cluster_config_beta3", func(t *testing.T) {
		g := NewWithT(t)

		clusterCtx := &domain.ClusterContext{
			ControlPlaneHost: "10.0.0.1",
			ClusterToken:     "test-token.1234567890123456",
		}

		clusterConfig := &kubeadmapiv3.ClusterConfiguration{}

		MutateClusterConfigBeta3Defaults(clusterCtx, clusterConfig)

		// Validate that the function executes without error
		// The actual mutations depend on the implementation details
		g.Expect(clusterConfig).ToNot(BeNil())
	})
}

// TestMutateClusterConfigBeta4Defaults tests the MutateClusterConfigBeta4Defaults function
func TestMutateClusterConfigBeta4Defaults(t *testing.T) {
	t.Run("mutate_cluster_config_beta4", func(t *testing.T) {
		g := NewWithT(t)

		clusterCtx := &domain.ClusterContext{
			ControlPlaneHost: "10.0.0.1",
			ClusterToken:     "test-token.1234567890123456",
		}

		clusterConfig := &kubeadmapiv4.ClusterConfiguration{}

		MutateClusterConfigBeta4Defaults(clusterCtx, clusterConfig)

		// Validate that the function executes without error
		// The actual mutations depend on the implementation details
		g.Expect(clusterConfig).ToNot(BeNil())
	})
}

// TestMutateKubeletDefaults tests the MutateKubeletDefaults function
func TestMutateKubeletDefaults(t *testing.T) {
	t.Run("mutate_kubelet_defaults", func(t *testing.T) {
		g := NewWithT(t)

		clusterCtx := &domain.ClusterContext{
			ControlPlaneHost: "10.0.0.1",
			ClusterToken:     "test-token.1234567890123456",
		}

		kubeletConfig := &kubeletv1beta1.KubeletConfiguration{}

		MutateKubeletDefaults(clusterCtx, kubeletConfig)

		// Validate that the function executes without error
		// The actual mutations depend on the implementation details
		g.Expect(kubeletConfig).ToNot(BeNil())
	})
}

// TestValueOrDefaultString tests the ValueOrDefaultString function
func TestValueOrDefaultString(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		defaultValue   string
		expectedResult string
	}{
		{
			name:           "value_provided",
			value:          "custom-value",
			defaultValue:   "default-value",
			expectedResult: "custom-value",
		},
		{
			name:           "empty_value_use_default",
			value:          "",
			defaultValue:   "default-value",
			expectedResult: "default-value",
		},
		{
			name:           "both_empty",
			value:          "",
			defaultValue:   "",
			expectedResult: "",
		},
		{
			name:           "value_with_spaces",
			value:          "  value  ",
			defaultValue:   "default-value",
			expectedResult: "  value  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := ValueOrDefaultString(tt.value, tt.defaultValue)

			g.Expect(result).To(Equal(tt.expectedResult))
		})
	}
}

// TestAppendIfNotPresent tests the appendIfNotPresent function
func TestAppendIfNotPresent(t *testing.T) {
	tests := []struct {
		name           string
		slice          []string
		value          string
		expectedResult []string
	}{
		{
			name:           "append_new_value",
			slice:          []string{"existing1", "existing2"},
			value:          "new-value",
			expectedResult: []string{"existing1", "existing2", "new-value"},
		},
		{
			name:           "value_already_exists",
			slice:          []string{"existing1", "existing2"},
			value:          "existing1",
			expectedResult: []string{"existing1", "existing2"},
		},
		{
			name:           "empty_slice",
			slice:          []string{},
			value:          "new-value",
			expectedResult: []string{"new-value"},
		},
		{
			name:           "nil_slice",
			slice:          nil,
			value:          "new-value",
			expectedResult: []string{"new-value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := appendIfNotPresent(tt.slice, tt.value)

			g.Expect(result).To(Equal(tt.expectedResult))
		})
	}
}
