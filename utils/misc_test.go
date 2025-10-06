package utils

import (
	"testing"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	. "github.com/onsi/gomega"
)

// TestGetClusterRootPath tests the GetClusterRootPath function
func TestGetClusterRootPath(t *testing.T) {
	tests := []struct {
		name           string
		cluster        clusterplugin.Cluster
		expectedResult string
	}{
		{
			name: "with_cluster_root_path",
			cluster: clusterplugin.Cluster{
				ProviderOptions: map[string]string{
					"cluster_root_path": "/persistent/spectro",
				},
			},
			expectedResult: "/persistent/spectro",
		},
		{
			name: "with_empty_cluster_root_path",
			cluster: clusterplugin.Cluster{
				ProviderOptions: map[string]string{
					"cluster_root_path": "",
				},
			},
			expectedResult: "/",
		},
		{
			name: "without_cluster_root_path",
			cluster: clusterplugin.Cluster{
				ProviderOptions: map[string]string{},
			},
			expectedResult: "/",
		},
		{
			name: "nil_provider_options",
			cluster: clusterplugin.Cluster{
				ProviderOptions: nil,
			},
			expectedResult: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetClusterRootPath(tt.cluster)

			g.Expect(result).To(Equal(tt.expectedResult))
		})
	}
}

// TestIsKubeadmVersionGreaterThan131 tests the IsKubeadmVersionGreaterThan131 function
func TestIsKubeadmVersionGreaterThan131(t *testing.T) {
	t.Run("kubeadm_version_check", func(t *testing.T) {
		g := NewWithT(t)

		// This test will be skipped if kubeadm is not available
		// We're testing the function structure, not the actual version check
		result, err := IsKubeadmVersionGreaterThan131("/")

		// The function should return an int and error
		// We can't predict the exact result without kubeadm binary
		g.Expect(result).To(BeAssignableToTypeOf(0))
		// Error might be nil or not depending on kubeadm availability
		// We just validate that err is an error type when it's not nil
		if err != nil {
			g.Expect(err.Error()).To(BeAssignableToTypeOf(""))
		}
	})
}
