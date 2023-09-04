package utils

import kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"

const (
	k8sNoProxy = ".svc,.svc.cluster,.svc.cluster.local"
)

func GetNoProxyConfig(clusterCfg kubeadmapiv3.ClusterConfiguration, proxyMap map[string]string) string {
	defaultNoProxy := GetDefaultNoProxy(clusterCfg)
	userNoProxy := proxyMap["NO_PROXY"]
	if len(userNoProxy) > 0 {
		return defaultNoProxy + "," + userNoProxy
	}
	return defaultNoProxy
}

func IsProxyConfigured(proxyMap map[string]string) bool {
	return len(proxyMap["HTTP_PROXY"]) > 0 || len(proxyMap["HTTPS_PROXY"]) > 0
}

func GetDefaultNoProxy(clusterCfg kubeadmapiv3.ClusterConfiguration) string {
	var noProxy string

	clusterCidr := clusterCfg.Networking.PodSubnet
	serviceCidr := clusterCfg.Networking.ServiceSubnet

	if len(clusterCidr) > 0 {
		noProxy = clusterCidr
	}
	if len(serviceCidr) > 0 {
		noProxy = noProxy + "," + serviceCidr
	}
	return noProxy + "," + k8sNoProxy
}
