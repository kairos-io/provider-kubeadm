package stages

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"

	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/utils"
	yip "github.com/mudler/yip/pkg/schema"
	"k8s.io/cli-runtime/pkg/printers"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

func GetJoinYipStages(cluster clusterplugin.Cluster, clusterCfg kubeadmapiv3.ClusterConfiguration, joinCfg kubeadmapiv3.JoinConfiguration, kubeletCfg kubeletv1beta1.KubeletConfiguration) []yip.Stage {
	utils.MutateClusterConfigDefaults(cluster, &clusterCfg)
	utils.MutateKubeletDefaults(&clusterCfg, &kubeletCfg)

	joinStg := []yip.Stage{
		getKubeadmJoinConfigStage(getJoinNodeConfiguration(cluster, joinCfg)),
		getKubeadmJoinStage(cluster, clusterCfg),
		getKubeadmJoinUpgradeStage(cluster, clusterCfg),
	}

	if cluster.Role != clusterplugin.RoleWorker {
		joinStg = append(joinStg, getKubeadmJoinCreateClusterConfigStage(clusterCfg), getKubeadmJoinCreateKubeletConfigStage(kubeletCfg))
	}

	return append(joinStg, getKubeadmJoinReconfigureStage(cluster, clusterCfg, joinCfg))
}

func getJoinNodeConfiguration(cluster clusterplugin.Cluster, joinCfg kubeadmapiv3.JoinConfiguration) string {
	if joinCfg.Discovery.BootstrapToken == nil {
		joinCfg.Discovery.BootstrapToken = &kubeadmapiv3.BootstrapTokenDiscovery{
			Token:                    cluster.ClusterToken,
			APIServerEndpoint:        fmt.Sprintf("%s:6443", cluster.ControlPlaneHost),
			UnsafeSkipCAVerification: true,
		}
	}

	if cluster.Role == clusterplugin.RoleControlPlane {
		joinCfg.ControlPlane = &kubeadmapiv3.JoinControlPlane{
			CertificateKey: utils.GetCertificateKey(cluster.ClusterToken),
		}

		var apiEndpoint kubeadmapiv3.APIEndpoint

		if joinCfg.ControlPlane.LocalAPIEndpoint.AdvertiseAddress == "" {
			apiEndpoint.AdvertiseAddress = domain.DefaultAPIAdvertiseAddress
		} else {
			apiEndpoint.AdvertiseAddress = joinCfg.ControlPlane.LocalAPIEndpoint.AdvertiseAddress
		}

		if joinCfg.ControlPlane.LocalAPIEndpoint.BindPort != 0 {
			apiEndpoint.BindPort = joinCfg.ControlPlane.LocalAPIEndpoint.BindPort
		}

		joinCfg.ControlPlane.LocalAPIEndpoint = apiEndpoint
	}

	joinPrinter := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})

	out := bytes.NewBuffer([]byte{})
	_ = joinPrinter.PrintObj(&joinCfg, out)

	return out.String()
}

func getKubeadmJoinConfigStage(kubeadmCfg string) yip.Stage {
	return yip.Stage{
		Name: "Generate Kubeadm Join Config File",
		Files: []yip.File{
			{
				Path:        filepath.Join(configurationPath, "kubeadm.yaml"),
				Permissions: 0640,
				Content:     kubeadmCfg,
			},
		},
	}
}

func getKubeadmJoinStage(cluster clusterplugin.Cluster, clusterCfg kubeadmapiv3.ClusterConfiguration) yip.Stage {
	joinStage := yip.Stage{
		Name: "Run Kubeadm Join",
		If:   "[ ! -f /opt/kubeadm.join ]",
	}

	if utils.IsProxyConfigured(cluster.Env) {
		proxy := cluster.Env
		joinStage.Commands = []string{
			fmt.Sprintf("bash %s %s %t %s %s %s", filepath.Join(helperScriptPath, "kube-join.sh"), cluster.Role, true, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCfg, cluster.Env)),
			"touch /opt/kubeadm.join",
		}
	} else {
		joinStage.Commands = []string{
			fmt.Sprintf("bash %s %s", filepath.Join(helperScriptPath, "kube-join.sh"), cluster.Role),
			"touch /opt/kubeadm.join",
		}
	}
	return joinStage
}

func getKubeadmJoinUpgradeStage(cluster clusterplugin.Cluster, clusterCfg kubeadmapiv3.ClusterConfiguration) yip.Stage {
	upgradeStage := yip.Stage{
		Name: "Run Kubeadm Join Upgrade",
	}

	if utils.IsProxyConfigured(cluster.Env) {
		proxy := cluster.Env
		upgradeStage.Commands = []string{
			fmt.Sprintf("bash %s %s %t %s %s %s", filepath.Join(helperScriptPath, "kube-upgrade.sh"), cluster.Role, true, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCfg, cluster.Env)),
		}
	} else {
		upgradeStage.Commands = []string{
			fmt.Sprintf("bash %s %s", filepath.Join(helperScriptPath, "kube-upgrade.sh"), cluster.Role),
		}
	}
	return upgradeStage
}

func getKubeadmJoinCreateClusterConfigStage(clusterCfg kubeadmapiv3.ClusterConfiguration) yip.Stage {
	return yip.Stage{
		Name: "Generate Cluster Config File",
		Files: []yip.File{
			{
				Path:        filepath.Join(configurationPath, "cluster-config.yaml"),
				Permissions: 0640,
				Content:     getUpdatedClusterConfig(clusterCfg),
			},
		},
	}
}

func getKubeadmJoinCreateKubeletConfigStage(kubeletCfg kubeletv1beta1.KubeletConfiguration) yip.Stage {
	return yip.Stage{
		Name: "Generate Kubelet Config File",
		Files: []yip.File{
			{
				Path:        filepath.Join(configurationPath, "kubelet-config.yaml"),
				Permissions: 0640,
				Content:     getUpdatedKubeletConfig(kubeletCfg),
			},
		},
	}
}

func getKubeadmJoinReconfigureStage(cluster clusterplugin.Cluster, clusterCfg kubeadmapiv3.ClusterConfiguration, joinCfg kubeadmapiv3.JoinConfiguration) yip.Stage {
	reconfigureStage := yip.Stage{
		Name: "Run Kubeadm Join Reconfiguration",
	}

	kubeletArgs := utils.RegenerateKubeletKubeadmArgsFile(&clusterCfg, &joinCfg.NodeRegistration, string(cluster.Role))
	sansRevision := utils.GetCertSansRevision(clusterCfg.APIServer.CertSANs)

	if utils.IsProxyConfigured(cluster.Env) {
		proxy := cluster.Env
		reconfigureStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %s %s %s %s", filepath.Join(helperScriptPath, "kube-reconfigure.sh"), cluster.Role, sansRevision, kubeletArgs, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCfg, cluster.Env)),
		}
	} else {
		reconfigureStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %s", filepath.Join(helperScriptPath, "kube-reconfigure.sh"), cluster.Role, sansRevision, kubeletArgs),
		}
	}
	return reconfigureStage
}
