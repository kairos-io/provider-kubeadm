package utils

import (
	"k8s.io/apimachinery/pkg/util/version"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	"github.com/sirupsen/logrus"
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
		logrus.Fatalf("Failed to parse kubernetes version [%v]. Err: %v", kubernetesVersion, err)
	}

	return v1.Compare("v1.31.0")
}
