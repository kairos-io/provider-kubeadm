package utils

import (
	"github.com/kairos-io/kairos/provider-kubeadm/domain"
)

const (
	k8sNoProxy = ".svc,.svc.cluster,.svc.cluster.local"
)

func GetNoProxyConfig(clusterCtx *domain.ClusterContext) string {
	defaultNoProxy := GetDefaultNoProxy(clusterCtx)
	userNoProxy := clusterCtx.EnvConfig["NO_PROXY"]
	if len(userNoProxy) > 0 {
		return defaultNoProxy + "," + userNoProxy
	}
	return defaultNoProxy
}

func IsProxyConfigured(proxyMap map[string]string) bool {
	return len(proxyMap["HTTP_PROXY"]) > 0 || len(proxyMap["HTTPS_PROXY"]) > 0
}

func GetDefaultNoProxy(clusterCtx *domain.ClusterContext) string {
	var noProxy string

	clusterCidr := clusterCtx.ClusterCidr
	serviceCidr := clusterCtx.ServiceCidr

	if len(clusterCidr) > 0 {
		noProxy = clusterCidr
	}

	if len(serviceCidr) > 0 {
		noProxy = noProxy + "," + serviceCidr
	}
	return noProxy + "," + k8sNoProxy
}
