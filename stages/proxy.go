package stages

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	"github.com/kairos-io/kairos/provider-kubeadm/utils"
	yip "github.com/mudler/yip/pkg/schema"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

const (
	envPrefix = "Environment="
)

func GetPreKubeadmProxyStage(kubeadmConfig domain.KubeadmConfig, cluster clusterplugin.Cluster) yip.Stage {
	return yip.Stage{
		Name: "Set proxy env",
		Files: []yip.File{
			{
				Path:        filepath.Join("/etc/default", "kubelet"),
				Permissions: 0400,
				Content:     kubeletProxyEnv(kubeadmConfig.ClusterConfiguration, cluster.Env),
			},
			{
				Path:        filepath.Join("/etc/systemd/system/containerd.service.d", "http-proxy.conf"),
				Permissions: 0400,
				Content:     containerdProxyEnv(kubeadmConfig.ClusterConfiguration, cluster.Env),
			},
		},
	}
}

func kubeletProxyEnv(clusterCfg kubeadmapiv3.ClusterConfiguration, proxyMap map[string]string) string {
	var proxy []string

	httpProxy := proxyMap["HTTP_PROXY"]
	httpsProxy := proxyMap["HTTP_PROXY"]
	userNoProxy := proxyMap["NO_PROXY"]

	if utils.IsProxyConfigured(proxyMap) {
		noProxy := utils.GetDefaultNoProxy(clusterCfg)
		if len(httpProxy) > 0 {
			proxy = append(proxy, fmt.Sprintf("HTTP_PROXY=%s", httpProxy))
		}

		if len(httpsProxy) > 0 {
			proxy = append(proxy, fmt.Sprintf("HTTPS_PROXY=%s", httpsProxy))
		}

		if len(userNoProxy) > 0 {
			noProxy = noProxy + "," + userNoProxy
		}
		proxy = append(proxy, fmt.Sprintf("NO_PROXY=%s", noProxy))
	}
	return strings.Join(proxy, "\n")
}

func containerdProxyEnv(clusterCfg kubeadmapiv3.ClusterConfiguration, proxyMap map[string]string) string {
	var proxy []string

	httpProxy := proxyMap["HTTP_PROXY"]
	httpsProxy := proxyMap["HTTPS_PROXY"]
	userNoProxy := proxyMap["NO_PROXY"]

	if utils.IsProxyConfigured(proxyMap) {
		proxy = append(proxy, "[Service]")
		noProxy := utils.GetDefaultNoProxy(clusterCfg)

		if len(httpProxy) > 0 {
			proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"HTTP_PROXY=%s"+"\"", httpProxy))
		}

		if len(httpsProxy) > 0 {
			proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"HTTPS_PROXY=%s"+"\"", httpsProxy))
		}

		if len(userNoProxy) > 0 {
			noProxy = noProxy + "," + userNoProxy
		}
		proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"NO_PROXY=%s"+"\"", noProxy))
	}
	return strings.Join(proxy, "\n")
}
