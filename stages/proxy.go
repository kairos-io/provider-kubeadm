package stages

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/domain"
	"github.com/kairos-io/kairos/provider-kubeadm/utils"
	yip "github.com/mudler/yip/pkg/schema"
)

const (
	envPrefix = "Environment="
)

func GetPreKubeadmProxyStage(clusterCtx *domain.ClusterContext, cluster clusterplugin.Cluster) yip.Stage {
	return yip.Stage{
		Name: "Set proxy env",
		Files: []yip.File{
			{
				Path:        filepath.Join("/etc/default", "kubelet"),
				Permissions: 0400,
				Content:     kubeletProxyEnv(clusterCtx),
			},
			{
				Path:        filepath.Join(fmt.Sprintf("/run/systemd/system/%s.service.d", getContainerdServiceFolderName(cluster.ProviderOptions)), "http-proxy.conf"),
				Permissions: 0400,
				Content:     containerdProxyEnv(clusterCtx),
			},
		},
	}
}

func kubeletProxyEnv(clusterCtx *domain.ClusterContext) string {
	var proxy []string

	proxyMap := clusterCtx.EnvConfig

	httpProxy := proxyMap["HTTP_PROXY"]
	httpsProxy := proxyMap["HTTPS_PROXY"]
	userNoProxy := proxyMap["NO_PROXY"]

	if utils.IsProxyConfigured(proxyMap) {
		noProxy := utils.GetDefaultNoProxy(clusterCtx)
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

func containerdProxyEnv(clusterCtx *domain.ClusterContext) string {
	var proxy []string

	proxyMap := clusterCtx.EnvConfig

	httpProxy := proxyMap["HTTP_PROXY"]
	httpsProxy := proxyMap["HTTPS_PROXY"]
	userNoProxy := proxyMap["NO_PROXY"]

	if utils.IsProxyConfigured(proxyMap) {
		proxy = append(proxy, "[Service]")
		noProxy := utils.GetDefaultNoProxy(clusterCtx)

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

func getContainerdServiceFolderName(options map[string]string) string {
	if _, ok := options["spectro-containerd-service-name"]; ok {
		return "spectro-containerd"
	}
	return "containerd"
}
