package utils

import (
	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/domain"
)

func GetClusterRootPath(cluster clusterplugin.Cluster) string {
	return cluster.ProviderOptions[domain.ClusterRootPath]
}
