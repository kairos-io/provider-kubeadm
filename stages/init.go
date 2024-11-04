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

func GetInitYipStagesV1Beta3(clusterCtx *domain.ClusterContext, initCfg kubeadmapiv3.InitConfiguration, clusterCfg kubeadmapiv3.ClusterConfiguration, kubeletCfg kubeletv1beta1.KubeletConfiguration) []yip.Stage {
	utils.MutateClusterConfigBeta3Defaults(clusterCtx, &clusterCfg)
	utils.MutateKubeletDefaults(clusterCtx, &kubeletCfg)

	clusterCtx.KubeletArgs = utils.RegenerateKubeletKubeadmArgsUsingBeta3Config(&clusterCfg, &initCfg.NodeRegistration, clusterCtx.NodeRole)
	clusterCtx.CertSansRevision = utils.GetCertSansRevision(clusterCfg.APIServer.CertSANs)

	return []yip.Stage{
		getKubeadmInitConfigStage(getInitNodeConfigurationBeta3(clusterCtx, initCfg, clusterCfg, kubeletCfg), clusterCtx.RootPath),
		getKubeadmInitStage(clusterCtx),
		getKubeadmPostInitStage(clusterCtx.RootPath),
		getKubeadmInitUpgradeStage(clusterCtx),
		getKubeadmInitCreateClusterConfigStage(&clusterCfg, &initCfg, clusterCtx.RootPath),
		getKubeadmInitCreateKubeletConfigStage(kubeletCfg, clusterCtx.RootPath),
		getKubeadmInitReconfigureStage(clusterCtx),
	}
}

func GetInitYipStagesV1Beta4(clusterCtx *domain.ClusterContext, initCfg kubeadmapiv4.InitConfiguration, clusterCfg kubeadmapiv4.ClusterConfiguration, kubeletCfg kubeletv1beta1.KubeletConfiguration) []yip.Stage {
	utils.MutateClusterConfigBeta4Defaults(clusterCtx, &clusterCfg)
	utils.MutateKubeletDefaults(clusterCtx, &kubeletCfg)

	clusterCtx.KubeletArgs = utils.RegenerateKubeletKubeadmArgsUsingBeta4Config(&clusterCfg, &initCfg.NodeRegistration, clusterCtx.NodeRole)
	clusterCtx.CertSansRevision = utils.GetCertSansRevision(clusterCfg.APIServer.CertSANs)

	return []yip.Stage{
		getKubeadmInitConfigStage(getInitNodeConfigurationBeta4(clusterCtx, initCfg, clusterCfg, kubeletCfg), clusterCtx.RootPath),
		getKubeadmInitStage(clusterCtx),
		getKubeadmPostInitStage(clusterCtx.RootPath),
		getKubeadmInitUpgradeStage(clusterCtx),
		getKubeadmInitCreateClusterConfigStage(&clusterCfg, &initCfg, clusterCtx.RootPath),
		getKubeadmInitCreateKubeletConfigStage(kubeletCfg, clusterCtx.RootPath),
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

func getKubeadmInitCreateKubeletConfigStage(kubeletCfg kubeletv1beta1.KubeletConfiguration, rootPath string) yip.Stage {
	return utils.GetFileStage("Generate Kubelet Config File", filepath.Join(rootPath, configurationPath, "kubelet-config.yaml"), getUpdatedKubeletConfig(kubeletCfg))
}

func getKubeadmInitReconfigureStage(clusterCtx *domain.ClusterContext) yip.Stage {
	reconfigureStage := yip.Stage{
		Name: "Run Kubeadm Reconfiguration",
	}

	clusterRootPath := clusterCtx.RootPath

	if utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		proxy := clusterCtx.EnvConfig
		reconfigureStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %s %s %s %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-reconfigure.sh"), clusterCtx.NodeRole, clusterCtx.CertSansRevision, clusterCtx.KubeletArgs, clusterRootPath, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCtx)),
		}
	} else {
		reconfigureStage.Commands = []string{
			fmt.Sprintf("bash %s %s %s %s %s", filepath.Join(clusterRootPath, helperScriptPath, "kube-reconfigure.sh"), clusterCtx.NodeRole, clusterCtx.CertSansRevision, clusterCtx.KubeletArgs, clusterRootPath),
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

func getUpdatedKubeletConfig(kubeletCfg kubeletv1beta1.KubeletConfiguration) string {
	return printObj([]runtime.Object{&kubeletCfg})
}

func printObj(objects []runtime.Object) string {
	initPrintr := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})
	out := bytes.NewBuffer([]byte{})

	for _, obj := range objects {
		_ = initPrintr.PrintObj(obj, out)
	}

	return out.String()
}
