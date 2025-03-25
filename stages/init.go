package stages

import (
	"bytes"
	"fmt"
	"path/filepath"

	kubeadmapiv4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"

	"github.com/kairos-io/kairos/provider-kubeadm/domain"

	"github.com/kairos-io/kairos/provider-kubeadm/utils"
	yip "github.com/mudler/yip/pkg/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
	bootstraptokenv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/bootstraptoken/v1"
	kubeadmapiv3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = kubeadmapiv3.AddToScheme(scheme)
	_ = kubeadmapiv4.AddToScheme(scheme)
	_ = kubeletv1beta1.AddToScheme(scheme)
}

const (
	configurationPath = "opt/kubeadm"
)

func GetInitYipStagesV1Beta3(clusterCtx *domain.ClusterContext, kubeadmConfig domain.KubeadmConfigBeta3) []yip.Stage {
	utils.MutateClusterConfigBeta3Defaults(clusterCtx, &kubeadmConfig.ClusterConfiguration)
	utils.MutateKubeletDefaults(clusterCtx, &kubeadmConfig.KubeletConfiguration)

	clusterCtx.KubeletArgs = utils.RegenerateKubeletKubeadmArgsUsingBeta3Config(&kubeadmConfig.InitConfiguration.NodeRegistration, clusterCtx.NodeRole)
	clusterCtx.CertSansRevision = utils.GetCertSansRevision(kubeadmConfig.ClusterConfiguration.APIServer.CertSANs)
	clusterCtx.CustomNodeIp = kubeadmConfig.InitConfiguration.NodeRegistration.KubeletExtraArgs["node-ip"]

	return []yip.Stage{
		getKubeadmInitConfigStage(getInitNodeConfigurationBeta3(clusterCtx, kubeadmConfig.InitConfiguration, kubeadmConfig.ClusterConfiguration, kubeadmConfig.KubeletConfiguration), clusterCtx.RootPath),
		getKubeadmInitStage(clusterCtx),
		getKubeadmPostInitStage(clusterCtx.RootPath),
		getKubeadmInitCreateClusterConfigStage(&kubeadmConfig.ClusterConfiguration, &kubeadmConfig.InitConfiguration, clusterCtx.RootPath),
		getKubeadmInitUpgradeStage(clusterCtx),
		getKubeadmInitCreateKubeletConfigStage(&kubeadmConfig.ClusterConfiguration, &kubeadmConfig.InitConfiguration, &kubeadmConfig.KubeletConfiguration, clusterCtx.RootPath),
		getKubeadmInitReconfigureStage(clusterCtx),
	}
}

func GetInitYipStagesV1Beta4(clusterCtx *domain.ClusterContext, kubeadmConfig domain.KubeadmConfigBeta4) []yip.Stage {
	clusterCtx.ServiceCidr = kubeadmConfig.ClusterConfiguration.Networking.ServiceSubnet
	clusterCtx.ClusterCidr = kubeadmConfig.ClusterConfiguration.Networking.PodSubnet

	utils.MutateClusterConfigBeta4Defaults(clusterCtx, &kubeadmConfig.ClusterConfiguration)
	utils.MutateKubeletDefaults(clusterCtx, &kubeadmConfig.KubeletConfiguration)

	clusterCtx.KubeletArgs = utils.RegenerateKubeletKubeadmArgsUsingBeta4Config(&kubeadmConfig.InitConfiguration.NodeRegistration, clusterCtx.NodeRole)
	clusterCtx.CertSansRevision = utils.GetCertSansRevision(kubeadmConfig.ClusterConfiguration.APIServer.CertSANs)
	clusterCtx.CustomNodeIp = getArgValue(kubeadmConfig.InitConfiguration.NodeRegistration.KubeletExtraArgs, "node-ip")

	return []yip.Stage{
		getKubeadmInitConfigStage(getInitNodeConfigurationBeta4(clusterCtx, kubeadmConfig.InitConfiguration, kubeadmConfig.ClusterConfiguration, kubeadmConfig.KubeletConfiguration), clusterCtx.RootPath),
		getKubeadmInitStage(clusterCtx),
		getKubeadmPostInitStage(clusterCtx.RootPath),
		getKubeadmInitCreateClusterConfigStage(&kubeadmConfig.ClusterConfiguration, &kubeadmConfig.InitConfiguration, clusterCtx.RootPath),
		getKubeadmInitUpgradeStage(clusterCtx),
		getKubeadmInitCreateKubeletConfigStage(&kubeadmConfig.ClusterConfiguration, &kubeadmConfig.InitConfiguration, &kubeadmConfig.KubeletConfiguration, clusterCtx.RootPath),
		getKubeadmInitReconfigureStage(clusterCtx),
	}
}

func getKubeadmInitConfigStage(kubeadmCfg, rootPath string) yip.Stage {
	return utils.GetFileStage("Generate Kubeadm Init Config File", filepath.Join(rootPath, configurationPath, "kubeadm.yaml"), kubeadmCfg)
}

func getKubeadmInitStage(clusterCtx *domain.ClusterContext) yip.Stage {
	clusterRootPath := clusterCtx.RootPath

	initStage := yip.Stage{
		Name: "Run Kubeadm Init",
		If:   fmt.Sprintf("[ ! -f %s ]", filepath.Join(clusterRootPath, "opt/kubeadm.init")),
	}

	if utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		proxy := clusterCtx.EnvConfig
		initStage.Commands = []string{
			fmt.Sprintf("bash %s %s %t %s %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-init.sh"), clusterRootPath, true, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCtx)),
			fmt.Sprintf("touch %s", filepath.Join(clusterRootPath, "opt/kubeadm.init")),
		}
	} else {
		initStage.Commands = []string{
			fmt.Sprintf("bash %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-init.sh"), clusterRootPath),
			fmt.Sprintf("touch %s", filepath.Join(clusterRootPath, "opt/kubeadm.init")),
		}
	}
	return initStage
}

func getKubeadmPostInitStage(clusterRootPath string) yip.Stage {
	return yip.Stage{
		Name: "Run Post Kubeadm Init",
		If:   fmt.Sprintf("[ ! -f %s ]", filepath.Join(clusterRootPath, "opt/post-kubeadm.init")),
		Commands: []string{
			fmt.Sprintf("bash %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-post-init.sh"), clusterRootPath),
			fmt.Sprintf("touch %s", filepath.Join(clusterRootPath, "opt/post-kubeadm.init")),
		},
	}
}

func getKubeadmInitUpgradeStage(clusterCtx *domain.ClusterContext) yip.Stage {
	upgradeStage := yip.Stage{
		Name: "Run Kubeadm Init Upgrade",
	}
	clusterRootPath := clusterCtx.RootPath

	if utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		upgradeStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %t %s %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-upgrade.sh"), clusterCtx.NodeRole, clusterRootPath, true, clusterCtx.EnvConfig["HTTP_PROXY"], clusterCtx.EnvConfig["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCtx)),
		}
	} else {
		upgradeStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-upgrade.sh"), clusterCtx.NodeRole, clusterRootPath),
		}
	}
	return upgradeStage
}

func getKubeadmInitCreateClusterConfigStage(clusterCfgObj, initCfgObj runtime.Object, rootPath string) yip.Stage {
	return utils.GetFileStage("Generate Cluster Config File", filepath.Join(rootPath, configurationPath, "cluster-config.yaml"), getUpdatedInitClusterConfig(clusterCfgObj, initCfgObj))
}

func getKubeadmInitCreateKubeletConfigStage(clusterCfgObj, initCfg, kubeletCfg runtime.Object, rootPath string) yip.Stage {
	return utils.GetFileStage("Generate Kubelet Config File", filepath.Join(rootPath, configurationPath, "kubelet-config.yaml"), getUpdatedKubeletConfig(clusterCfgObj, initCfg, kubeletCfg))
}

func getKubeadmInitReconfigureStage(clusterCtx *domain.ClusterContext) yip.Stage {
	reconfigureStage := yip.Stage{
		Name: "Run Kubeadm Reconfiguration",
	}

	clusterRootPath := clusterCtx.RootPath

	if utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		proxy := clusterCtx.EnvConfig
		reconfigureStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %s %s %s %s %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-reconfigure.sh"), clusterCtx.NodeRole,
				clusterCtx.CertSansRevision, clusterCtx.KubeletArgs, clusterRootPath, clusterCtx.CustomNodeIp, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"],
				utils.GetNoProxyConfig(clusterCtx)),
		}
	} else {
		reconfigureStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %s %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-reconfigure.sh"), clusterCtx.NodeRole,
				clusterCtx.CertSansRevision, clusterCtx.KubeletArgs, clusterRootPath, clusterCtx.CustomNodeIp),
		}
	}
	return reconfigureStage
}

func getInitNodeConfigurationBeta3(clusterCtx *domain.ClusterContext, initCfg kubeadmapiv3.InitConfiguration, clusterCfg kubeadmapiv3.ClusterConfiguration, kubeletCfg kubeletv1beta1.KubeletConfiguration) string {
	certificateKey := utils.GetCertificateKey(clusterCtx.ClusterToken)
	substrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(clusterCtx.ClusterToken)

	initCfg.BootstrapTokens = []bootstraptokenv1.BootstrapToken{
		{
			Token: &bootstraptokenv1.BootstrapTokenString{
				ID:     substrs[1],
				Secret: substrs[2],
			},
			TTL: &metav1.Duration{
				Duration: 0,
			},
		},
	}
	initCfg.CertificateKey = certificateKey

	var apiEndpoint kubeadmapiv3.APIEndpoint

	if initCfg.LocalAPIEndpoint.AdvertiseAddress == "" {
		apiEndpoint.AdvertiseAddress = domain.DefaultAPIAdvertiseAddress
	} else {
		apiEndpoint.AdvertiseAddress = initCfg.LocalAPIEndpoint.AdvertiseAddress
	}

	if initCfg.LocalAPIEndpoint.BindPort != 0 {
		apiEndpoint.BindPort = initCfg.LocalAPIEndpoint.BindPort
	}

	initCfg.LocalAPIEndpoint = apiEndpoint

	return printObj([]runtime.Object{&clusterCfg, &initCfg, &kubeletCfg})
}

func getInitNodeConfigurationBeta4(clusterCtx *domain.ClusterContext, initCfg kubeadmapiv4.InitConfiguration, clusterCfg kubeadmapiv4.ClusterConfiguration, kubeletCfg kubeletv1beta1.KubeletConfiguration) string {
	certificateKey := utils.GetCertificateKey(clusterCtx.ClusterToken)
	substrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(clusterCtx.ClusterToken)

	initCfg.BootstrapTokens = []bootstraptokenv1.BootstrapToken{
		{
			Token: &bootstraptokenv1.BootstrapTokenString{
				ID:     substrs[1],
				Secret: substrs[2],
			},
			TTL: &metav1.Duration{
				Duration: 0,
			},
		},
	}
	initCfg.CertificateKey = certificateKey

	var apiEndpoint kubeadmapiv4.APIEndpoint

	if initCfg.LocalAPIEndpoint.AdvertiseAddress == "" {
		apiEndpoint.AdvertiseAddress = domain.DefaultAPIAdvertiseAddress
	} else {
		apiEndpoint.AdvertiseAddress = initCfg.LocalAPIEndpoint.AdvertiseAddress
	}

	if initCfg.LocalAPIEndpoint.BindPort != 0 {
		apiEndpoint.BindPort = initCfg.LocalAPIEndpoint.BindPort
	}

	initCfg.LocalAPIEndpoint = apiEndpoint

	return printObj([]runtime.Object{&clusterCfg, &initCfg, &kubeletCfg})
}

func getUpdatedInitClusterConfig(clusterCfgObj, initCfgObj runtime.Object) string {
	return printObj([]runtime.Object{clusterCfgObj, initCfgObj})
}

func getUpdatedKubeletConfig(clusterCfgObj, initCfg, kubeletCfg runtime.Object) string {
	return printObj([]runtime.Object{clusterCfgObj, initCfg, kubeletCfg})
}

func printObj(objects []runtime.Object) string {
	initPrintr := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})
	out := bytes.NewBuffer([]byte{})

	for _, obj := range objects {
		_ = initPrintr.PrintObj(obj, out)
	}

	return out.String()
}

func getArgValue(args []kubeadmapiv4.Arg, name string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg.Name == name {
			return arg.Value
		}
	}
	return ""
}
