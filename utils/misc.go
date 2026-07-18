package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/version"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/domain"
)

func GetClusterRootPath(cluster clusterplugin.Cluster) string {
	rootpath := cluster.ProviderOptions[domain.ClusterRootPath]
	if rootpath == "" {
		return domain.DefaultRootPath
	}
	return rootpath
}

func IsKubeadmVersionGreaterThan131(kubernetesVersion string) (int, error) {
	v1, err := version.ParseSemantic(kubernetesVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to parse kubernetes version %q: %w", kubernetesVersion, err)
	}

	return v1.Compare("v1.31.0")
}
