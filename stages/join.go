package stages

import (
	"bytes"
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"

	kubeadmapiv4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"

	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-kubeadm/utils"
	yip "github.com/mudler/yip/pkg/schema"
	"k8s.io/cli-runtime/pkg/printers"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

func GetJoinYipStagesV1Beta3(clusterCtx *domain.ClusterContext, kubeadmConfig domain.KubeadmConfigBeta3) []yip.Stage {
	utils.MutateClusterConfigBeta3Defaults(clusterCtx, &kubeadmConfig.ClusterConfiguration)
	utils.MutateKubeletDefaults(clusterCtx, &kubeadmConfig.KubeletConfiguration)

	joinStg := []yip.Stage{
		getKubeadmJoinConfigStage(getJoinNodeConfigurationBeta3(clusterCtx, kubeadmConfig.JoinConfiguration), clusterCtx.RootPath),
		getKubeadmJoinStage(clusterCtx),
		getKubeadmJoinUpgradeStage(clusterCtx),
	}

	if clusterCtx.NodeRole != clusterplugin.RoleWorker {
		joinStg = append(joinStg,
			getKubeadmJoinCreateClusterConfigStage(&kubeadmConfig.ClusterConfiguration, &kubeadmConfig.InitConfiguration, &kubeadmConfig.JoinConfiguration, clusterCtx.RootPath),
			getKubeadmJoinCreateKubeletConfigStage(kubeadmConfig.KubeletConfiguration, clusterCtx.RootPath))
	}

	return append(joinStg, getKubeadmJoinReconfigureStage(clusterCtx))
}

func GetJoinYipStagesV1Beta4(clusterCtx *domain.ClusterContext, kubeadmConfig domain.KubeadmConfigBeta4) []yip.Stage {
	utils.MutateClusterConfigBeta4Defaults(clusterCtx, &kubeadmConfig.ClusterConfiguration)
	utils.MutateKubeletDefaults(clusterCtx, &kubeadmConfig.KubeletConfiguration)

	joinStg := []yip.Stage{
		getKubeadmJoinConfigStage(getJoinNodeConfigurationBeta4(clusterCtx, kubeadmConfig.JoinConfiguration), clusterCtx.RootPath),
		getKubeadmJoinStage(clusterCtx),
		getKubeadmJoinUpgradeStage(clusterCtx),
	}

	if clusterCtx.NodeRole != clusterplugin.RoleWorker {
		joinStg = append(joinStg,
			getKubeadmJoinCreateClusterConfigStage(&kubeadmConfig.ClusterConfiguration, &kubeadmConfig.InitConfiguration, &kubeadmConfig.JoinConfiguration, clusterCtx.RootPath),
			getKubeadmJoinCreateKubeletConfigStage(kubeadmConfig.KubeletConfiguration, clusterCtx.RootPath))
	}

	return append(joinStg, getKubeadmJoinReconfigureStage(clusterCtx))
}

func getJoinNodeConfigurationBeta3(clusterCtx *domain.ClusterContext, joinCfg kubeadmapiv3.JoinConfiguration) string {
	if joinCfg.Discovery.BootstrapToken == nil {
		joinCfg.Discovery.BootstrapToken = &kubeadmapiv3.BootstrapTokenDiscovery{
			Token:                    clusterCtx.ClusterToken,
			APIServerEndpoint:        fmt.Sprintf("%s:6443", clusterCtx.ControlPlaneHost),
			UnsafeSkipCAVerification: true,
		}
	}

	if clusterCtx.NodeRole == clusterplugin.RoleControlPlane {
		if joinCfg.ControlPlane == nil {
			joinCfg.ControlPlane = &kubeadmapiv3.JoinControlPlane{}
		}
		joinCfg.ControlPlane.CertificateKey = utils.GetCertificateKey(clusterCtx.ClusterToken)

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

func getJoinNodeConfigurationBeta4(clusterCtx *domain.ClusterContext, joinCfg kubeadmapiv4.JoinConfiguration) string {
	if joinCfg.Discovery.BootstrapToken == nil {
		joinCfg.Discovery.BootstrapToken = &kubeadmapiv4.BootstrapTokenDiscovery{
			Token:                    clusterCtx.ClusterToken,
			APIServerEndpoint:        fmt.Sprintf("%s:6443", clusterCtx.ControlPlaneHost),
			UnsafeSkipCAVerification: true,
		}
	}

	if clusterCtx.NodeRole == clusterplugin.RoleControlPlane {
		if joinCfg.ControlPlane == nil {
			joinCfg.ControlPlane = &kubeadmapiv4.JoinControlPlane{}
		}
		joinCfg.ControlPlane.CertificateKey = utils.GetCertificateKey(clusterCtx.ClusterToken)

		var apiEndpoint kubeadmapiv4.APIEndpoint

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

func getKubeadmJoinConfigStage(kubeadmCfg, rootPath string) yip.Stage {
	return yip.Stage{
		Name: "Generate Kubeadm Join Config File",
		Files: []yip.File{
			{
				Path:        filepath.Join(rootPath, configurationPath, "kubeadm.yaml"),
				Permissions: 0640,
				Content:     kubeadmCfg,
			},
		},
	}
}

func getKubeadmJoinStage(clusterCtx *domain.ClusterContext) yip.Stage {
	clusterRootPath := clusterCtx.RootPath

	joinStage := yip.Stage{
		Name: "Run Kubeadm Join",
		If:   fmt.Sprintf("[ ! -f %s ]", filepath.Join(clusterRootPath, "opt/kubeadm.join")),
	}

	if utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		proxy := clusterCtx.EnvConfig
		joinStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %t %s %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-join.sh"), clusterCtx.NodeRole, clusterRootPath, true, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCtx)),
			fmt.Sprintf("touch %s", filepath.Join(clusterRootPath, "opt/kubeadm.join")),
		}
	} else {
		joinStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-join.sh"), clusterCtx.NodeRole, clusterRootPath),
			fmt.Sprintf("touch %s", filepath.Join(clusterRootPath, "opt/kubeadm.join")),
		}
	}
	return joinStage
}

func getKubeadmJoinUpgradeStage(clusterCtx *domain.ClusterContext) yip.Stage {
	upgradeStage := yip.Stage{
		Name: "Run Kubeadm Join Upgrade",
	}

	if utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		proxy := clusterCtx.EnvConfig
		upgradeStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %t %s %s %s", filepath.Join(clusterCtx.RootPath, helperScriptPath, "kube-upgrade.sh"), clusterCtx.NodeRole, clusterCtx.RootPath, true, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCtx)),
		}
	} else {
		upgradeStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s", filepath.Join(clusterCtx.RootPath, helperScriptPath, "kube-upgrade.sh"), clusterCtx.NodeRole, clusterCtx.RootPath),
		}
	}
	return upgradeStage
}

func getKubeadmJoinCreateClusterConfigStage(clusterCfgObj, initCfgObj, joinCfgObj runtime.Object, rootPath string) yip.Stage {
	return utils.GetFileStage("Generate Cluster Config File", filepath.Join(rootPath, configurationPath, "cluster-config.yaml"), getUpdatedJoinClusterConfig(clusterCfgObj, initCfgObj, joinCfgObj))
}

func getKubeadmJoinCreateKubeletConfigStage(kubeletCfg kubeletv1beta1.KubeletConfiguration, rootPath string) yip.Stage {
	return utils.GetFileStage("Generate Kubelet Config File", filepath.Join(rootPath, configurationPath, "kubelet-config.yaml"), getUpdatedKubeletConfig(kubeletCfg))
}

func getKubeadmJoinReconfigureStage(clusterCtx *domain.ClusterContext) yip.Stage {
	reconfigureStage := yip.Stage{
		Name: "Run Kubeadm Join Reconfiguration",
	}

	if utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		proxy := clusterCtx.EnvConfig
		reconfigureStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %s %s %s %s %s", filepath.Join(clusterCtx.RootPath, helperScriptPath, "kube-reconfigure.sh"), clusterCtx.NodeRole, clusterCtx.CertSansRevision, clusterCtx.KubeletArgs, clusterCtx.RootPath, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCtx)),
		}
	} else {
		reconfigureStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %s %s", filepath.Join(clusterCtx.RootPath, helperScriptPath, "kube-reconfigure.sh"), clusterCtx.NodeRole, clusterCtx.CertSansRevision, clusterCtx.KubeletArgs, clusterCtx.RootPath),
		}
	}
	return reconfigureStage
}

func getUpdatedJoinClusterConfig(clusterCfgObj, initCfgObj, joinCfgObj runtime.Object) string {
	switch initCfgObj.(type) {
	case *kubeadmapiv3.InitConfiguration:
		initCfg := initCfgObj.(*kubeadmapiv3.InitConfiguration)
		joinCfg := joinCfgObj.(*kubeadmapiv3.JoinConfiguration)
		if joinCfg.ControlPlane != nil {
			initCfg.LocalAPIEndpoint.AdvertiseAddress = joinCfg.ControlPlane.LocalAPIEndpoint.AdvertiseAddress
			initCfg.LocalAPIEndpoint.BindPort = joinCfg.ControlPlane.LocalAPIEndpoint.BindPort
		}
	case *kubeadmapiv4.InitConfiguration:
		initCfg := initCfgObj.(*kubeadmapiv4.InitConfiguration)
		joinCfg := joinCfgObj.(*kubeadmapiv4.JoinConfiguration)
		if joinCfg.ControlPlane != nil {
			initCfg.LocalAPIEndpoint.AdvertiseAddress = joinCfg.ControlPlane.LocalAPIEndpoint.AdvertiseAddress
			initCfg.LocalAPIEndpoint.BindPort = joinCfg.ControlPlane.LocalAPIEndpoint.BindPort
		}
	}

	return printObj([]runtime.Object{clusterCfgObj, initCfgObj, joinCfgObj})
}
