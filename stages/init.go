package stages

import (
	"bytes"
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/kubernetes/cmd/kubeadm/app/constants"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
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
	_ = kubeletv1beta1.AddToScheme(scheme)
}

const (
	configurationPath = "/opt/kubeadm"
)

func GetInitYipStages(cluster clusterplugin.Cluster, initCfg kubeadmapiv3.InitConfiguration, clusterCfg kubeadmapiv3.ClusterConfiguration, kubeletCfg kubeletv1beta1.KubeletConfiguration) []yip.Stage {
	kubeadmCfg := getInitNodeConfiguration(cluster, initCfg, clusterCfg, kubeletCfg)

	mutateClusterConfigDefaults(&clusterCfg)

	return []yip.Stage{
		getKubeadmInitConfigStage(kubeadmCfg),
		getKubeadmInitStage(cluster, clusterCfg),
		getKubeadmPostInitStage(),
		getKubeadmInitUpgradeStage(cluster, clusterCfg),
		getKubeadmInitCreateClusterConfigStage(cluster, clusterCfg),
		getKubeadmInitCreateKubeletConfigStage(cluster, clusterCfg, kubeletCfg),
		getKubeadmInitReconfigureStage(cluster, kubeletCfg, clusterCfg, initCfg),
	}
}

func getKubeadmInitConfigStage(kubeadmCfg string) yip.Stage {
	return yip.Stage{
		Name: "Generate Kubeadm Init Config File",
		Files: []yip.File{
			{
				Path:        filepath.Join(configurationPath, "kubeadm.yaml"),
				Permissions: 0640,
				Content:     kubeadmCfg,
			},
		},
	}
}

func getKubeadmInitStage(cluster clusterplugin.Cluster, clusterCfg kubeadmapiv3.ClusterConfiguration) yip.Stage {
	initStage := yip.Stage{
		Name: "Run Kubeadm Init",
		If:   "[ ! -f /opt/kubeadm.init ]",
	}

	if utils.IsProxyConfigured(cluster.Env) {
		proxy := cluster.Env
		initStage.Commands = []string{
			fmt.Sprintf("bash %s %t %s %s %s", filepath.Join(helperScriptPath, "kube-init.sh"), true, proxy["HTTP_PROXY"], proxy["HTTPS_PROXY"], utils.GetNoProxyConfig(clusterCfg, cluster.Env)),
			"touch /opt/kubeadm.init",
		}
	} else {
		initStage.Commands = []string{
			fmt.Sprintf("bash %s", filepath.Join(helperScriptPath, "kube-init.sh")),
			"touch /opt/kubeadm.init",
		}
	}
	return initStage
}

func getKubeadmPostInitStage() yip.Stage {
	return yip.Stage{
		Name: "Run Post Kubeadm Init",
		If:   "[ ! -f /opt/post-kubeadm.init ]",
		Commands: []string{
			fmt.Sprintf("bash %s", filepath.Join(helperScriptPath, "kube-post-init.sh")),
			"touch /opt/post-kubeadm.init",
		},
	}
}

func getKubeadmInitUpgradeStage(cluster clusterplugin.Cluster, clusterCfg kubeadmapiv3.ClusterConfiguration) yip.Stage {
	upgradeStage := yip.Stage{
		Name: "Run Kubeadm Init Upgrade",
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

func getKubeadmInitCreateClusterConfigStage(cluster clusterplugin.Cluster, clusterCfg kubeadmapiv3.ClusterConfiguration) yip.Stage {
	return yip.Stage{
		Name: "Generate Cluster Config File",
		If:   fmt.Sprintf("[ \"%s\" != \"worker\" ]", cluster.Role),
		Files: []yip.File{
			{
				Path:        filepath.Join(configurationPath, "cluster-config.yaml"),
				Permissions: 0640,
				Content:     getUpdatedClusterConfig(clusterCfg, cluster),
			},
		},
	}
}

func getKubeadmInitCreateKubeletConfigStage(cluster clusterplugin.Cluster, clusterCfg kubeadmapiv3.ClusterConfiguration, kubeletCfg kubeletv1beta1.KubeletConfiguration) yip.Stage {
	return yip.Stage{
		Name: "Generate Kubelet Config File",
		If:   fmt.Sprintf("[ \"%s\" != \"worker\" ]", cluster.Role),
		Files: []yip.File{
			{
				Path:        filepath.Join(configurationPath, "kubelet-config.yaml"),
				Permissions: 0640,
				Content:     getUpdatedKubeletConfig(clusterCfg, kubeletCfg),
			},
		},
	}
}

func getKubeadmInitReconfigureStage(cluster clusterplugin.Cluster, kubeletCfg kubeletv1beta1.KubeletConfiguration, clusterCfg kubeadmapiv3.ClusterConfiguration, initCfg kubeadmapiv3.InitConfiguration) yip.Stage {
	reconfigureStage := yip.Stage{
		Name: "Run Kubeadm Reconfiguration",
	}

	kubeletArgs := utils.RegenerateKubeletKubeadmArgsFile(&clusterCfg, &initCfg.NodeRegistration, string(cluster.Role))
	sansRevision := utils.GetCertSansRevision(clusterCfg.APIServer.CertSANs)

	utils.WriteKubeletConfigToDisk(&clusterCfg, &kubeletCfg, filepath.Join("/var/lib/kubelet", constants.KubeletConfigurationFileName))

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

func getInitNodeConfiguration(cluster clusterplugin.Cluster, initCfg kubeadmapiv3.InitConfiguration, clusterCfg kubeadmapiv3.ClusterConfiguration, kubeletCfg kubeletv1beta1.KubeletConfiguration) string {
	certificateKey := utils.GetCertificateKey(cluster.ClusterToken)

	substrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(cluster.ClusterToken)

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
	initCfg.LocalAPIEndpoint = kubeadmapiv3.APIEndpoint{
		AdvertiseAddress: "0.0.0.0",
	}
	clusterCfg.APIServer.CertSANs = append(clusterCfg.APIServer.CertSANs, cluster.ControlPlaneHost)
	clusterCfg.ControlPlaneEndpoint = fmt.Sprintf("%s:6443", cluster.ControlPlaneHost)

	if kubeletCfg.ShutdownGracePeriod.Duration == 0 {
		kubeletCfg.ShutdownGracePeriod = metav1.Duration{
			Duration: 120 * time.Second,
		}
	}

	if kubeletCfg.ShutdownGracePeriodCriticalPods.Duration == 0 {
		kubeletCfg.ShutdownGracePeriodCriticalPods = metav1.Duration{
			Duration: 60 * time.Second,
		}
	}

	initPrintr := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})

	out := bytes.NewBuffer([]byte{})

	_ = initPrintr.PrintObj(&clusterCfg, out)
	_ = initPrintr.PrintObj(&initCfg, out)
	_ = initPrintr.PrintObj(&kubeletCfg, out)

	return out.String()
}

func getUpdatedClusterConfig(clusterCfg kubeadmapiv3.ClusterConfiguration, cluster clusterplugin.Cluster) string {
	clusterCfg.APIServer.CertSANs = append(clusterCfg.APIServer.CertSANs, cluster.ControlPlaneHost)
	clusterCfg.ControlPlaneEndpoint = fmt.Sprintf("%s:6443", cluster.ControlPlaneHost)

	initPrintr := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})

	out := bytes.NewBuffer([]byte{})
	_ = initPrintr.PrintObj(&clusterCfg, out)

	return out.String()
}

func getUpdatedKubeletConfig(clusterCfg kubeadmapiv3.ClusterConfiguration, kubeletCfg kubeletv1beta1.KubeletConfiguration) string {
	if kubeletCfg.ShutdownGracePeriod.Duration == 0 {
		kubeletCfg.ShutdownGracePeriod = metav1.Duration{
			Duration: 120 * time.Second,
		}
	}

	if kubeletCfg.ShutdownGracePeriodCriticalPods.Duration == 0 {
		kubeletCfg.ShutdownGracePeriodCriticalPods = metav1.Duration{
			Duration: 60 * time.Second,
		}
	}

	utils.WriteKubeletConfigToDisk(&clusterCfg, &kubeletCfg, filepath.Join(configurationPath, "kubelet-config.yaml"))

	initPrintr := printers.NewTypeSetter(scheme).ToPrinter(&printers.YAMLPrinter{})

	out := bytes.NewBuffer([]byte{})
	_ = initPrintr.PrintObj(&kubeletCfg, out)

	return out.String()
}

func mutateClusterConfigDefaults(clusterCfg *kubeadmapiv3.ClusterConfiguration) {
	if clusterCfg.ImageRepository == "" {
		clusterCfg.ImageRepository = kubeadmapiv3.DefaultImageRepository
	}
}
