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

// TestIsKubeadmVersionGreaterThan131 compares kubernetesVersion against v1.31.0 (see k8s.io/apimachinery/pkg/util/version.Version.Compare).
func TestIsKubeadmVersionGreaterThan131(t *testing.T) {
	tests := []struct {
		name            string
		k8sVersion      string
		wantCmpSign     string // "<", "=", ">" relative to v1.31.0
		wantErrContains string
	}{
		{
			name:        "less_than_1_31",
			k8sVersion:  "v1.30.0",
			wantCmpSign: "<",
		},
		{
			name:        "equal_1_31",
			k8sVersion:  "v1.31.0",
			wantCmpSign: "=",
		},
		{
			name:        "greater_than_1_31",
			k8sVersion:  "v1.32.4",
			wantCmpSign: ">",
		},
		{
			name:            "invalid_version",
			k8sVersion:      "/",
			wantErrContains: "failed to parse kubernetes version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result, err := IsKubeadmVersionGreaterThan131(tt.k8sVersion)

			if tt.wantErrContains != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring(tt.wantErrContains))
				g.Expect(result).To(Equal(0))
				return
			}

			t.Log("result", result)

			g.Expect(err).NotTo(HaveOccurred())
			switch tt.wantCmpSign {
			case "<":
				g.Expect(result).To(BeNumerically("<", 0))
			case "=":
				g.Expect(result).To(Equal(0))
			case ">":
				g.Expect(result).To(BeNumerically(">", 0))
			default:
				t.Fatalf("invalid wantCmpSign %q", tt.wantCmpSign)
			}
		})
	}
}
